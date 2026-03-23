// build.go implements the core mdpress build command.
// It loads config, resolves sources, and dispatches document generation.
// Both local directories and GitHub repositories are supported, including zero-config discovery mode.
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/i18n"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/source"
	"github.com/yeasy/mdpress/pkg/utils"
)

var (
	// buildFormat stores the --format override for output formats.
	buildFormat string
	// buildBranch stores the Git branch override for GitHub sources.
	buildBranch string
	// buildSubDir stores the source subdirectory override.
	buildSubDir string
	// buildOutput stores the requested output path, directory, or prefix.
	buildOutput string
	// buildSummary stores the path to a SUMMARY.md file.
	buildSummary string
)

// buildCmd is the main build subcommand.
var buildCmd = &cobra.Command{
	Use:   "build [source]",
	Short: "Build documents (PDF/HTML/ePub)",
	Long: `Build high-quality documents from a local directory or GitHub repository.

Supported input sources:
  Local directory (current directory by default)
  GitHub repository URL

Output formats:
  pdf   - PDF document (default)
  html  - Self-contained single-page HTML
  site  - Multi-page static site
  epub  - ePub ebook

Zero-config mode:
  If neither book.yaml nor SUMMARY.md exists, mdpress auto-discovers .md files.

Examples:
  mdpress build
  mdpress build --format html
  mdpress build --format site --output ./dist/book
  mdpress build --format pdf,html,epub
  mdpress build --format all
  mdpress build --config path/to/book.yaml
  mdpress build https://github.com/yeasy/agentic_ai_guide
  mdpress build github.com/yeasy/agentic_ai_guide --branch main
  mdpress build https://github.com/yeasy/agentic_ai_guide --subdir docs/
  mdpress build --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var inputSource string
		if len(args) > 0 {
			inputSource = args[0]
		}
		return executeBuild(cmd.Context(), inputSource)
	},
}

func init() {
	buildCmd.Flags().StringVar(&buildFormat, "format", "", "Output formats, comma-separated (pdf,html,site,epub) or 'all'")
	buildCmd.Flags().StringVar(&buildBranch, "branch", "", "Git branch name (GitHub sources only)")
	buildCmd.Flags().StringVar(&buildSubDir, "subdir", "", "Subdirectory inside the source")
	buildCmd.Flags().StringVar(&buildOutput, "output", "", "Output file path, directory, or filename prefix")
	buildCmd.Flags().StringVar(&buildSummary, "summary", "", "Path to SUMMARY.md file")
}

// executeBuild runs the full build flow.
func executeBuild(ctx context.Context, inputSource string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Set log level based on quiet/verbose flags:
	//   --quiet: errors only
	//   --verbose: debug output
	//   default: info and above
	logLevel := slog.LevelInfo
	switch {
	case quiet:
		logLevel = slog.LevelError
	case verbose:
		logLevel = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// ========== 1. Resolve the input source ==========
	var workDir string
	if inputSource != "" {
		// Resolve external sources through the source module.
		opts := source.Options{
			Branch: buildBranch,
			SubDir: buildSubDir,
		}
		src, err := source.Detect(inputSource, opts)
		if err != nil {
			return fmt.Errorf("failed to parse input source: %w", err)
		}
		defer func() {
			if err := src.Cleanup(); err != nil {
				logger.Debug("Failed to clean up source", slog.String("error", err.Error()))
			}
		}()

		logger.Info("Preparing build source", slog.String("type", src.Type()), slog.String("source", inputSource))

		workDir, err = src.Prepare()
		if err != nil {
			return fmt.Errorf("failed to prepare source directory: %w", err)
		}
		logger.Info("Source directory is ready", slog.String("dir", workDir))
	} else {
		// Default to the current directory.
		workDir = "."
		if buildSubDir != "" {
			workDir = buildSubDir
		}
	}

	// ========== 2. Load config (supports zero-config mode) ==========
	var cfg *config.BookConfig
	var err error

	// Prefer an explicit config path when one is available.
	configPath := cfgFile
	if inputSource != "" {
		// External sources first look for book.yaml in the prepared working directory.
		configPath = filepath.Join(workDir, "book.yaml")
	}

	if utils.FileExists(configPath) {
		// Load from book.yaml.
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		logger.Info("Loaded configuration from file", slog.String("config", configPath))
	} else {
		// Zero-config mode: auto-discover Markdown files.
		targetDir := workDir
		if inputSource == "" && configPath == "book.yaml" {
			// In the default case, discover from the current directory.
			targetDir, err = filepath.Abs(".")
			if err != nil {
				return fmt.Errorf("failed to resolve current directory: %w", err)
			}
		}
		logger.Info("Zero-config mode: auto-discovering Markdown files", slog.String("dir", targetDir))
		cfg, err = config.Discover(ctx, targetDir)
		if err != nil {
			return fmt.Errorf("auto-discovery failed: %w (try running 'mdpress init' to create a config)", err)
		}
	}

	logger.Info("Configuration loaded",
		slog.String("title", cfg.Book.Title),
		slog.String("author", cfg.Book.Author),
		slog.Int("chapters", len(cfg.Chapters)))

	// Handle explicit --summary flag.
	if buildSummary != "" {
		summaryPath, err := filepath.Abs(buildSummary)
		if err != nil {
			return fmt.Errorf("failed to resolve summary path: %w", err)
		}
		chapters, err := config.ParseSummary(summaryPath)
		if err != nil {
			return fmt.Errorf("failed to parse SUMMARY.md: %w", err)
		}
		cfg.Chapters = chapters
		logger.Info("Loaded chapters from SUMMARY.md", slog.String("path", summaryPath), slog.Int("chapters", len(chapters)))
	}

	// ========== 3. Resolve output formats ==========
	// CLI --format overrides the config file.
	// Special value "all" expands to all supported formats.
	formats := cfg.Output.Formats
	if buildFormat != "" {
		formats = strings.Split(buildFormat, ",")
		for i := range formats {
			formats[i] = strings.TrimSpace(formats[i])
		}
	}
	// Expand the "all" alias to the full set of supported formats.
	expandedFormats := make([]string, 0, len(formats))
	for _, f := range formats {
		if f == "all" {
			expandedFormats = append(expandedFormats, "pdf", "html", "site", "epub")
		} else {
			expandedFormats = append(expandedFormats, f)
		}
	}
	formats = expandedFormats
	if len(formats) == 0 {
		formats = []string{"pdf"}
	}

	logger.Info("Starting build", slog.Any("formats", formats))

	outputOverride, err := resolveRequestedBuildOutput(buildOutput)
	if err != nil {
		return err
	}

	if cfg.LangsFile != "" {
		langs, langsErr := i18n.ParseLangsFile(cfg.LangsFile)
		if langsErr != nil {
			logger.Warn("Failed to parse LANGS.md, continuing as a single-language project", slog.String("error", langsErr.Error()))
		} else if len(langs) > 0 {
			return executeMultilingualBuild(ctx, workDir, langs, formats, outputOverride, logger)
		}
	}

	return executeBuildForConfig(ctx, cfg, formats, outputOverride, logger)
}

// flattenChapters delegates to the canonical config.FlattenChapters.
func flattenChapters(chapters []config.ChapterDef) []config.ChapterDef {
	return config.FlattenChapters(chapters)
}

// getPageDimensions returns page dimensions in millimeters from a size name.
func getPageDimensions(size string) (width, height float64) {
	switch strings.ToUpper(size) {
	case "A5":
		return 148, 210
	case "LETTER":
		return 216, 279
	case "LEGAL":
		return 216, 356
	case "B5":
		return 176, 250
	default: // A4
		return 210, 297
	}
}

func rewriteChapterLinks(chapters []renderer.ChapterHTML, chapterFiles []string) []renderer.ChapterHTML {
	if len(chapters) == 0 || len(chapters) != len(chapterFiles) {
		return chapters
	}

	targets := make(map[string]linkrewrite.Target, len(chapters))
	for i, ch := range chapters {
		if chapterFiles[i] == "" || ch.ID == "" {
			continue
		}
		targets[linkrewrite.NormalizePath(chapterFiles[i])] = linkrewrite.Target{ChapterID: ch.ID}
	}

	rewritten := make([]renderer.ChapterHTML, len(chapters))
	for i, ch := range chapters {
		rewritten[i] = ch
		rewritten[i].Content = linkrewrite.RewriteLinks(ch.Content, chapterFiles[i], targets, linkrewrite.ModeSingle)
	}

	return rewritten
}

// rewriteMarkdownLinksInHTML is a backward-compatible wrapper around linkrewrite.RewriteLinks
// that accepts the legacy map[string]string target format. Used by build_links_test.go.
func rewriteMarkdownLinksInHTML(htmlContent string, currentFile string, targets map[string]string) string {
	typedTargets := make(map[string]linkrewrite.Target, len(targets))
	for path, targetID := range targets {
		typedTargets[path] = linkrewrite.Target{ChapterID: targetID}
	}
	return linkrewrite.RewriteLinks(htmlContent, currentFile, typedTargets, linkrewrite.ModeSingle)
}
