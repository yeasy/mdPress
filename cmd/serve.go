// serve.go implements the local live preview server.
// It watches files, rebuilds HTML on change, and pushes reload events over WebSocket.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/server"
	"github.com/yeasy/mdpress/internal/source"
	"github.com/yeasy/mdpress/pkg/utils"
)

const (
	defaultServePort = 9000
	defaultServeHost = "127.0.0.1"
)

var (
	servePort int
	serveHost string
	serveDir  string
	serveOpen bool
)

// serveOptions encapsulates configuration for the serve command.
type serveOptions struct {
	Port        int
	Host        string
	OutputDir   string
	AutoOpen    bool
	PortChanged bool
}

var serveCmd = &cobra.Command{
	Use:           "serve [source]",
	Short:         "Start the live preview server",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `Build an HTML site and start a local HTTP server with live reload.

Supports both local directories and GitHub repository URLs as input.
Also supports zero-config mode by auto-discovering .md files.

Examples:
  mdpress serve
  mdpress serve --port 9000
  mdpress serve --host 0.0.0.0
  mdpress serve --open
  mdpress serve --config path/to/book.yaml
  mdpress serve https://github.com/yeasy/agentic_ai_guide`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var inputSource string
		if len(args) > 0 {
			inputSource = args[0]
		}
		opts := serveOptions{
			Port:        servePort,
			Host:        serveHost,
			OutputDir:   serveDir,
			AutoOpen:    serveOpen,
			PortChanged: cmd.Flags().Changed("port"),
		}
		return executeServe(cmd.Context(), inputSource, opts)
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", defaultServePort, "HTTP server port")
	serveCmd.Flags().StringVar(&serveHost, "host", defaultServeHost, "HTTP listen address (default 127.0.0.1)")
	serveCmd.Flags().StringVar(&serveDir, "output", "", "Output directory (defaults to _book)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open the browser automatically (default false)")
	serveCmd.Flags().StringVar(&buildSummary, "summary", "", "Path to SUMMARY.md file")
}

func executeServe(ctx context.Context, inputSource string, opts serveOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := initLogger()

	// ========== 1. Resolve the input source ==========
	var workDir string
	var srcCleanup func() error

	if inputSource != "" {
		sourceOpts := source.Options{
			Branch: buildBranch,
			SubDir: buildSubDir,
		}
		src, err := source.Detect(inputSource, sourceOpts)
		if err != nil {
			return fmt.Errorf("failed to parse input source: %w", err)
		}
		srcCleanup = src.Cleanup

		workDir, err = src.Prepare()
		if err != nil {
			return fmt.Errorf("failed to prepare source directory: %w", err)
		}
		logger.Info("Source directory is ready", slog.String("type", src.Type()), slog.String("dir", workDir))
	} else {
		var absErr error
		workDir, absErr = filepath.Abs(".")
		if absErr != nil {
			return fmt.Errorf("failed to resolve current directory: %w", absErr)
		}
	}

	defer func() {
		if srcCleanup != nil {
			if err := srcCleanup(); err != nil {
				logger.Debug("Failed to clean up source", slog.Any("error", err))
			}
		}
	}()

	// ========== 2. Load config (supports zero-config mode) ==========
	loadConfig := func() (*config.BookConfig, error) {
		configPath := filepath.Join(workDir, "book.yaml")
		if inputSource == "" {
			configPath = cfgFile
		}

		var cfg *config.BookConfig
		var err error
		if utils.FileExists(configPath) {
			cfg, err = config.Load(configPath)
		} else {
			cfg, err = config.Discover(ctx, workDir)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load or discover book config: %w", err)
		}

		// Handle explicit --summary flag.
		if buildSummary != "" {
			summaryPath, err := filepath.Abs(buildSummary)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve summary path: %w", err)
			}
			chapters, err := config.ParseSummary(summaryPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SUMMARY.md: %w", err)
			}
			cfg.Chapters = chapters
			slog.Default().Info("Loaded chapters from SUMMARY.md", slog.String("path", summaryPath), slog.Int("chapters", len(chapters)))
		}

		return cfg, nil
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve the output directory.
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(cfg.BaseDir(), "_book")
	}

	srv := server.NewServer(opts.Host, opts.Port, workDir, outputDir, logger)
	srv.AutoOpen = opts.AutoOpen

	var ln net.Listener
	if opts.PortChanged {
		ln, err = srv.Listen()
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	} else {
		ln, err = srv.ListenFrom(defaultServePort)
		if err != nil {
			return fmt.Errorf("failed to listen on default port: %w", err)
		}
	}
	listenerOwned := true
	defer func() {
		if listenerOwned {
			ln.Close() //nolint:errcheck
		}
	}()

	// Register the rebuild callback with atomic directory swap for safety.
	srv.BuildFunc = func() error {
		// Reload the config on every rebuild.
		newCfg, err := loadConfig()
		if err != nil {
			return fmt.Errorf("reload config: %w", err)
		}
		// Build to a temporary directory first, then swap on success.
		tempOutput, err := os.MkdirTemp(filepath.Dir(outputDir), "mdpress-serve-*.tmp")
		if err != nil {
			return fmt.Errorf("create temp output dir: %w", err)
		}
		if buildErr := buildSiteForServe(ctx, newCfg, tempOutput, logger); buildErr != nil {
			// Clean up the failed temp build, keep the previous good output.
			if err := os.RemoveAll(tempOutput); err != nil {
				logger.Debug("failed to remove temp output directory", slog.String("path", tempOutput), slog.Any("error", err))
			}
			return buildErr
		}
		// Swap the temp build into the final output directory.
		// Rename the old dir out of the way first, then rename the new dir in.
		// This minimizes the window where no content is available.
		backupDir := outputDir + ".old"
		if err := os.RemoveAll(backupDir); err != nil {
			logger.Debug("Failed to remove previous backup directory", slog.String("dir", backupDir), slog.Any("error", err))
		}
		if err := os.Rename(outputDir, backupDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
			logger.Debug("Failed to move previous output aside", slog.Any("error", err))
		}
		if renameErr := os.Rename(tempOutput, outputDir); renameErr != nil {
			// Restore the previous build if the swap failed.
			if err := os.Rename(backupDir, outputDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
				logger.Debug("Failed to restore previous output", slog.Any("error", err))
			}
			if err := os.RemoveAll(tempOutput); err != nil {
				logger.Debug("Failed to remove temp output directory", slog.String("dir", tempOutput), slog.Any("error", err))
			}
			// Fallback: if rename fails (cross-device), try the direct build.
			return buildSiteForServe(ctx, newCfg, outputDir, logger)
		}
		if err := os.RemoveAll(backupDir); err != nil {
			logger.Debug("Failed to remove backup directory after swap", slog.String("dir", backupDir), slog.Any("error", err))
		}
		return nil
	}

	// Initial build.
	logger.Info("Running initial site build", slog.String("title", cfg.Book.Title))
	if err := buildSiteForServe(ctx, cfg, outputDir, logger); err != nil {
		return fmt.Errorf("failed to build site: %w", err)
	}

	listenerOwned = false
	return srv.StartWithListener(ctx, ln)
}

// buildSiteForServe builds the preview HTML site.
func buildSiteForServe(ctx context.Context, cfg *config.BookConfig, outputDir string, logger *slog.Logger) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Initialize the orchestrator.
	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create build orchestrator: %w", err)
	}

	// Use the chapter pipeline for consistent processing.
	// Note: Pipeline always uses ParseWithDiagnostics and performs full validation.
	// For serve, we ignore the issues but benefit from consistent processing.
	result, err := orchestrator.ProcessChapters(ctx)
	if err != nil {
		return fmt.Errorf("failed to process chapters: %w", err)
	}

	chaptersHTML := result.Chapters
	chapterFiles := result.ChapterFiles
	chapterMarkdown := result.ChapterMarkdown

	customCSS := orchestrator.LoadCustomCSS()

	sitePages := sitePageFilenames(chapterFiles)
	siteChapters := rewriteChapterLinksForSite(chaptersHTML, chapterFiles, sitePages)
	if err := generateSiteOutput(cfg, orchestrator.Theme, customCSS, outputDir, siteChapters, sitePages, chapterMarkdown); err != nil {
		return fmt.Errorf("failed to generate site output: %w", err)
	}

	// Also generate a standalone HTML file for convenient reading.
	standaloneRenderer, err := renderer.NewStandaloneHTMLRenderer(cfg, orchestrator.Theme)
	if err != nil {
		return fmt.Errorf("failed to create standalone HTML renderer: %w", err)
	}
	standaloneHTML, err := standaloneRenderer.Render(&renderer.RenderParts{
		ChaptersHTML: rewriteChapterLinks(chaptersHTML, chapterFiles),
		CustomCSS:    customCSS,
	})
	if err != nil {
		logger.Warn("Failed to generate standalone HTML", slog.Any("error", err))
	} else {
		standalonePath := filepath.Join(outputDir, "standalone.html")
		if writeErr := os.WriteFile(standalonePath, []byte(standaloneHTML), 0o644); writeErr != nil {
			logger.Warn("Failed to write standalone HTML", slog.Any("error", writeErr))
		}
	}

	logger.Info("Site build completed", slog.String("output", outputDir))

	return nil
}
