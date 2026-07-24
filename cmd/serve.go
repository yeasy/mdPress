// serve.go implements the local live preview server.
// It watches files, rebuilds HTML on change, and pushes reload events over WebSocket.
package cmd

import (
	"context"
	"fmt"
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
	// serveStagingDirPattern names the per-rebuild staging directory created
	// next to the output. The file watcher skips directories matching it by
	// name (internal/server.isIgnoredDirName), so changing it here without
	// changing it there makes every rebuild retrigger the watcher.
	serveStagingDirPattern = "mdpress-serve-*.tmp"
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
  mdpress serve https://github.com/yeasy/agentic_ai_guide
  mdpress serve https://github.com/yeasy/agentic_ai_guide --branch main`,
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
	serveCmd.Flags().StringVar(&serveHost, "host", defaultServeHost, "HTTP listen address")
	serveCmd.Flags().StringVarP(&serveDir, "output", "o", "", "Directory to write the generated site (default _book)")
	serveCmd.Flags().BoolVar(&serveOpen, "open", false, "Open the browser automatically (default false)")
	serveCmd.Flags().StringVar(&buildSummary, "summary", "", "Path to SUMMARY.md file")
	serveCmd.Flags().StringVar(&buildBranch, "branch", "", "Git branch name (GitHub sources only)")
	serveCmd.Flags().StringVar(&buildSubDir, "subdir", "", "Subdirectory inside the source")
	serveCmd.Flags().BoolVar(&allowPlugins, "allow-plugins", false, "Execute plugins declared by a remote project's book.yaml (arbitrary code; local sources always run plugins)")
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
		// Record whether the resolved source is remote so the orchestrator can
		// gate plugin execution from untrusted remote projects.
		buildSourceIsRemote = src.Type() != "local"
		srcCleanup = src.Cleanup

		workDir, err = src.Prepare()
		if err != nil {
			return fmt.Errorf("failed to prepare source directory: %w", err)
		}
		logger.Info("Source directory is ready", slog.String("type", src.Type()), slog.String("dir", workDir))
	} else {
		// Default to the current directory (always a local source).
		buildSourceIsRemote = false
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
		// An explicit --config wins over the source directory's own book.yaml.
		sourceDir := ""
		if inputSource != "" {
			sourceDir = workDir
		}
		configPath, allowDiscovery := resolveConfigPath(sourceDir)
		if !allowDiscovery && !utils.FileExists(configPath) {
			return nil, errExplicitConfigMissing(configPath)
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

	// Every rebuild replaces outputDir wholesale via an atomic swap, so a
	// directory holding anything other than a previously generated site would
	// lose its contents on the first save. Refuse it up front, exactly like
	// `mdpress build --format site` does.
	//
	// This runs before any cleanup: a refused run must leave the filesystem
	// exactly as it found it. The sweep below used to run first, so `serve`
	// deleted scratch next to the output and only then bailed out.
	if err := ensureReplaceableSiteDir(outputDir); err != nil {
		return err
	}

	// A serve process killed mid-rebuild (Ctrl-C at the wrong moment, or
	// SIGKILL) leaves the atomic-swap scratch behind. Sweep it now so the
	// project does not accumulate mdpress-serve-*.tmp copies.
	cleanupServeLeftovers(outputDir, logger)

	// Warn when binding to a non-loopback address as this exposes the
	// preview server (including WebSocket and all generated content) to
	// other machines on the network.
	if opts.Host != "" && opts.Host != "127.0.0.1" && opts.Host != "::1" && opts.Host != "localhost" {
		logger.Warn("Server is exposed to the network; use only on trusted networks", slog.String("host", opts.Host))
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
		staging, err := newSiteStaging(filepath.Dir(outputDir), serveStagingDirPattern)
		if err != nil {
			return fmt.Errorf("create temp output dir: %w", err)
		}
		if buildErr := buildSiteForServe(ctx, newCfg, staging.Site, logger); buildErr != nil {
			// Clean up the failed temp build, keep the previous good output.
			staging.Discard(logger)
			return buildErr
		}
		// Swap the temp build into the final output directory. Shared with
		// `build --format site` so both park the previous output inside the
		// staging area rather than in the user's "<outputDir>.old" sibling.
		if err := swapSiteDir(staging, outputDir, logger); err != nil {
			// Rename can fail across devices; fall back to building in place.
			logger.Debug("Atomic site swap failed, rebuilding in place", slog.Any("error", err))
			return buildSiteForServe(ctx, newCfg, outputDir, logger)
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

// cleanupServeLeftovers removes the mdpress-serve-*.tmp staging directories
// the rebuild swap creates next to the output directory. They are only ever
// meaningful inside a single rebuild, so any that survive belong to a previous
// run that did not exit cleanly.
//
// It deliberately never touches "<outputDir>.old". mdpress no longer parks the
// previous site there, and sweeping that name destroyed hand-made backups of
// the last release that happened to sit next to the output directory.
func cleanupServeLeftovers(outputDir string, logger *slog.Logger) {
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(outputDir), serveStagingDirPattern))
	if err != nil {
		logger.Debug("Failed to scan for leftover staging directories", slog.Any("error", err))
		return
	}
	for _, dir := range matches {
		if err := os.RemoveAll(dir); err != nil {
			logger.Debug("Failed to remove leftover staging directory", slog.String("dir", dir), slog.Any("error", err))
		}
	}
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
	// Ensure plugin cleanup runs when the build finishes, matching the build command.
	if orchestrator.PluginManager != nil {
		defer func() {
			if cleanupErr := orchestrator.PluginManager.CleanupAll(); cleanupErr != nil {
				logger.Warn("Plugin cleanup failed", slog.Any("error", cleanupErr))
			}
		}()
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
	if err := generateSiteOutput(cfg, orchestrator.Theme, customCSS, outputDir, siteChapters, chapterFiles, sitePages, chapterMarkdown); err != nil {
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
