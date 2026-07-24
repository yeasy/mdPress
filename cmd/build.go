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
	Short: "Build documents (PDF/HTML/site/ePub/Typst)",
	Long: `Build high-quality documents from a local directory or GitHub repository.

Supported input sources:
  Local directory (current directory by default)
  GitHub repository URL

Output formats:
  pdf   - PDF document (default)
  html  - Self-contained single-page HTML
  site  - Multi-page static site
  epub  - ePub ebook
  typst - PDF via Typst CLI (Chromium-free)

Output paths (--output):
  pdf/html/epub/typst treat --output as a file path or base name; an existing
  directory (or a path ending with a separator) receives "<name>.<ext>" files.
  site writes to "_book/" by default. With --format site alone, --output is
  the site directory. Combined with other formats it is a shared base, so the
  site goes to a "<base>_site/" sibling and the build says so.

Zero-config mode:
  If neither book.yaml nor SUMMARY.md exists, mdpress auto-discovers .md files.

Examples:
  mdpress build
  mdpress build --format html
  mdpress build --format site --output ./dist
  mdpress build --format pdf,html,epub
  mdpress build --format all
  mdpress build --config path/to/book.yaml
  mdpress build https://github.com/yeasy/agentic_ai_guide
  mdpress build github.com/yeasy/agentic_ai_guide --branch main
  mdpress build https://github.com/yeasy/agentic_ai_guide --subdir docs/
  mdpress build https://github.com/yeasy/agentic_ai_guide --allow-plugins
  mdpress build --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var inputSource string
		if len(args) > 0 {
			inputSource = args[0]
		}
		// An empty --format is indistinguishable from an unset one by the time
		// executeBuild runs, so `--format ""` used to quietly build a PDF
		// instead of reporting the typo (usually an unset shell variable).
		if cmd.Flags().Changed("format") && strings.TrimSpace(buildFormat) == "" {
			return errEmptyFormatFlag()
		}
		return executeBuild(cmd.Context(), inputSource)
	},
}

// buildFormatNames lists every value --format accepts, with the descriptions
// shells display next to a candidate. It is the single source of truth for
// what is valid: the validation below and the shell completions both read it,
// so a new format cannot be completable but rejected, or vice versa.
var buildFormatNames = []string{
	"pdf\tPDF document",
	"html\tSelf-contained single-page HTML",
	"site\tMulti-page static site",
	"epub\tePub ebook",
	"typst\tPDF via the Typst CLI (Chromium-free)",
	"all\tpdf, html, site and epub",
}

// supportedBuildFormats returns the accepted --format values without their
// completion descriptions.
func supportedBuildFormats() []string {
	names := make([]string, 0, len(buildFormatNames))
	for _, entry := range buildFormatNames {
		names = append(names, completionValue(entry))
	}
	return names
}

// errEmptyFormatFlag is returned when --format is present but names nothing.
func errEmptyFormatFlag() error {
	return fmt.Errorf("--format was given but names no output format (supported: %s)",
		strings.Join(supportedBuildFormats(), ", "))
}

// parseFormatFlag splits a --format value into normalized format names.
//
// Values are lower-cased and trimmed because "PDF" and " html " are what
// people type, and empty elements are dropped: a stray comma ("pdf,,html", or
// a trailing one produced by a shell loop) used to abort the build with
// `unsupported format ""`, which named nothing the user had written.
func parseFormatFlag(raw string) ([]string, error) {
	requested := strings.Split(raw, ",")
	formats := make([]string, 0, len(requested))
	for _, f := range requested {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == "" {
			continue
		}
		formats = append(formats, f)
	}
	if len(formats) == 0 {
		return nil, errEmptyFormatFlag()
	}
	return formats, nil
}

func init() {
	buildCmd.Flags().StringVarP(&buildFormat, "format", "f", "", "Output formats, comma-separated (pdf,html,site,epub,typst) or 'all'")
	buildCmd.Flags().StringVar(&buildBranch, "branch", "", "Git branch name (GitHub sources only)")
	buildCmd.Flags().StringVar(&buildSubDir, "subdir", "", "Subdirectory inside the source")
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", "", "Output file path, directory, or base name for pdf/html/epub/typst; with --format site alone this is the site directory; shared with other formats the site goes to \"<base>_site/\" (default: _book/)")
	buildCmd.Flags().StringVar(&buildSummary, "summary", "", "Path to SUMMARY.md file")
	buildCmd.Flags().BoolVar(&allowPlugins, "allow-plugins", false, "Execute plugins declared by a remote project's book.yaml (arbitrary code; local sources always run plugins)")

	// --format takes a comma-separated list, so complete element by element.
	_ = buildCmd.RegisterFlagCompletionFunc("format", func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeCommaSeparated(buildFormatNames, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
	// --summary names a Markdown file; --output can be either a file or a
	// directory depending on the format, so it keeps plain file completion.
	_ = buildCmd.MarkFlagFilename("summary", "md")
}

// executeBuild runs the full build flow.
func executeBuild(ctx context.Context, inputSource string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := initLogger()

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
		// Record whether the resolved source is remote so the orchestrator can
		// gate plugin execution from untrusted remote projects.
		buildSourceIsRemote = src.Type() != "local"
		defer func() {
			if err := src.Cleanup(); err != nil {
				logger.Debug("Failed to clean up source", slog.Any("error", err))
			}
		}()

		logger.Info("preparing build source", slog.String("type", src.Type()), slog.String("source", inputSource))

		workDir, err = src.Prepare()
		if err != nil {
			return fmt.Errorf("failed to prepare source directory: %w", err)
		}
		logger.Info("source directory is ready", slog.String("dir", workDir))
	} else {
		// Default to the current directory (always a local source).
		buildSourceIsRemote = false
		workDir = "."
		if buildSubDir != "" {
			workDir = buildSubDir
		}
		if buildBranch != "" {
			// --branch only means something for a cloned source. Accepting it
			// silently made a mistyped invocation look like it had built the
			// requested branch.
			return fmt.Errorf("--branch applies to Git sources only; no source URL was given")
		}
	}

	// ========== 2. Load config (supports zero-config mode) ==========
	var cfg *config.BookConfig
	var err error

	// An explicit --config wins over the source directory's own book.yaml.
	// workDir is also the right base for a local `--subdir`: it used to be
	// dropped here, so `mdpress build --subdir mybook` ignored mybook/book.yaml,
	// fell into zero-config discovery of the parent, and wrote its output
	// beside the wrong directory while still exiting 0.
	sourceDir := ""
	if inputSource != "" || buildSubDir != "" {
		sourceDir = workDir
	}
	configPath, allowDiscovery := resolveConfigPath(sourceDir)
	if !allowDiscovery && !utils.FileExists(configPath) {
		return errExplicitConfigMissing(configPath)
	}

	if utils.FileExists(configPath) {
		// Load from book.yaml.
		cfg, err = config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		logger.Info("loaded configuration from file", slog.String("config", configPath))
	} else {
		// Zero-config mode: auto-discover Markdown files. Discovery always
		// starts at the directory the build was pointed at — "." by default,
		// but the --subdir when one was given.
		targetDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve source directory: %w", err)
		}
		logger.Info("zero-config mode: auto-discovering Markdown files", slog.String("dir", targetDir))
		cfg, err = config.Discover(ctx, targetDir)
		if err != nil {
			return fmt.Errorf("auto-discovery failed: %w (try running 'mdpress init' to create a config)", err)
		}
	}

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
		logger.Info("loaded chapters from SUMMARY.md", slog.String("path", summaryPath), slog.Int("chapters", len(chapters)))
	}

	// ========== 3. Resolve output formats ==========
	// CLI --format overrides the config file.
	// Special value "all" expands to all supported formats.
	formats := cfg.Output.Formats
	if buildFormat != "" {
		formats, err = parseFormatFlag(buildFormat)
		if err != nil {
			return err
		}
	}
	// Expand the "all" alias to the full set of supported formats.
	expandedFormats := make([]string, 0, len(formats))
	for _, f := range formats {
		if f == "all" {
			// "typst" is deliberately excluded: it is an opt-in alternative
			// PDF backend that needs the optional Typst CLI and produces the
			// same artifact as "pdf". Including it made `--format all` fail on
			// any machine without Typst, including the project's own CI recipe.
			expandedFormats = append(expandedFormats, "pdf", "html", "site", "epub")
		} else {
			expandedFormats = append(expandedFormats, f)
		}
	}
	formats = expandedFormats
	if len(formats) == 0 {
		formats = []string{"pdf"}
	}

	// Validate format names when specified via CLI flag.
	if buildFormat != "" {
		// "all" was expanded above, so it is not a valid element here.
		validFormats := make(map[string]bool, len(buildFormatNames))
		concrete := make([]string, 0, len(buildFormatNames))
		for _, name := range supportedBuildFormats() {
			if name == "all" {
				continue
			}
			validFormats[name] = true
			concrete = append(concrete, name)
		}
		for _, f := range formats {
			if !validFormats[f] {
				return fmt.Errorf("unsupported format %q (supported: %s)", f, strings.Join(concrete, ", "))
			}
		}
	}

	logger.Info("starting build", slog.Any("formats", formats))

	outputOverride, err := resolveRequestedBuildOutput(buildOutput)
	if err != nil {
		return fmt.Errorf("failed to resolve build output: %w", err)
	}

	// Remote sources are cloned into a temporary directory that is deleted
	// when the build finishes, so default outputs must land in the user's
	// working directory to survive cleanup.
	remoteOutputDir := ""
	if buildSourceIsRemote && outputOverride == "" {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			return fmt.Errorf("failed to resolve current directory for remote build output: %w", cwdErr)
		}
		remoteOutputDir = cwd
		// Trailing separator marks explicit directory intent for file formats.
		outputOverride = cwd + string(os.PathSeparator)
		logger.Info("remote source: writing outputs to the current directory", slog.String("dir", cwd))
	}

	if cfg.LangsFile != "" {
		langs, langsErr := i18n.ParseLangsFile(cfg.LangsFile)
		if langsErr != nil {
			logger.Warn("failed to parse LANGS.md, continuing as a single-language project", slog.Any("error", langsErr))
		} else if len(langs) > 0 {
			return executeMultilingualBuild(ctx, cfg.BaseDir(), langs, formats, outputOverride, logger)
		}
	}

	siteDir := resolveSiteOutputDir(cfg.BaseDir(), outputOverride, formats, logger)
	if remoteOutputDir != "" {
		// Keep the site in "<cwd>/_book" rather than spilling its pages into
		// the working directory itself.
		siteDir = filepath.Join(remoteOutputDir, "_book")
	}
	return executeBuildForConfig(ctx, cfg, formats, outputOverride, siteDir, logger)
}

// resolveSiteOutputDir picks the directory the "site" format writes into for a
// single-language build:
//   - no --output: the GitBook-style "_book" directory under the project,
//     matching `mdpress serve` and the CI deploy examples;
//   - --output with only "site" requested: used verbatim. The site is the
//     whole output, so the path the user named is the directory they want.
//     This used to depend on whether the path already existed — a clean CI
//     checkout got "dist_site" while the developer's second local run got
//     "dist", which is the worst possible way for a path to be decided;
//   - --output alongside other formats: the site goes to a "<base>_site"
//     sibling so pdf/html/epub can keep using the base path for their files,
//     and a warning names the directory actually used. The warning is only
//     printed when "site" is actually one of the requested formats: a plain
//     `build --format pdf -o book.pdf` used to warn about a "book_site"
//     directory it was never going to create, which sent users looking for
//     output that does not exist and tripped every CI job that greps stderr.
func resolveSiteOutputDir(baseDir, outputOverride string, formats []string, logger *slog.Logger) string {
	if outputOverride == "" {
		return filepath.Join(baseDir, "_book")
	}
	if hasTrailingPathSeparator(outputOverride) {
		return filepath.Clean(outputOverride)
	}
	if onlySiteFormat(formats) {
		return filepath.Clean(outputOverride)
	}
	if info, err := os.Stat(outputOverride); err == nil && info.IsDir() {
		return outputOverride
	}
	siteDir := strings.TrimSuffix(outputOverride, filepath.Ext(outputOverride)) + "_site"
	if logger != nil && containsBuildFormat(formats, "site") {
		logger.Warn("--output is shared with other formats, so the site goes to a sibling directory",
			slog.String("site_dir", siteDir),
			slog.String("hint", "build the site on its own to write straight into --output"))
	}
	return siteDir
}

// onlySiteFormat reports whether "site" is the only requested format.
func onlySiteFormat(formats []string) bool {
	seen := false
	for _, f := range formats {
		if !strings.EqualFold(strings.TrimSpace(f), "site") {
			return false
		}
		seen = true
	}
	return seen
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

// rewriteChapterLinksForEpub points cross-chapter Markdown links at the flat
// <chapterID>.xhtml documents that make up an ePub, mirroring what the single
// page and site paths already do for their own layouts.
func rewriteChapterLinksForEpub(chapters []renderer.ChapterHTML, chapterFiles []string) []renderer.ChapterHTML {
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
		rewritten[i].Content = linkrewrite.RewriteLinks(ch.Content, chapterFiles[i], targets, linkrewrite.ModeEpub)
	}

	return rewritten
}

// rewriteEpubGlossaryLinks re-points the glossary term links Glossary.ProcessHTML
// injects into every chapter at the glossary's own packaged document. Those links
// are same-document "#glossary-<term>" anchors — correct for the single-file HTML
// and PDF where the glossary shares one document — but in an ePub each chapter is
// its own flat OEBPS/<id>.xhtml file and the "glossary-<term>" anchors live only
// in glossary.xhtml, so left alone every highlighted term jumped nowhere. This
// mirrors resolveSiteGlossaryPage, which does the same for the site layout.
func rewriteEpubGlossaryLinks(chapters []renderer.ChapterHTML, chapterFiles []string) []renderer.ChapterHTML {
	glossaryIdx := -1
	for i, ch := range chapters {
		// The synthesized glossary appendix is the one chapter with no source
		// file and the reserved glossary ID.
		if i < len(chapterFiles) && chapterFiles[i] == "" && ch.ID == glossaryChapterID {
			glossaryIdx = i
			break
		}
	}
	if glossaryIdx < 0 {
		return chapters
	}

	// ePub documents are flat under OEBPS and the glossary is packaged as
	// <id>.xhtml (see epubBuilder.Build), so a bare filename reaches it from any
	// chapter.
	href := `href="` + chapters[glossaryIdx].ID + `.xhtml#glossary-`
	rewritten := make([]renderer.ChapterHTML, len(chapters))
	for i, ch := range chapters {
		rewritten[i] = ch
		// Leave the glossary chapter's own cross-term links alone: their
		// same-document anchors already resolve within glossary.xhtml.
		if i != glossaryIdx {
			rewritten[i].Content = glossarySelfLinkPattern.ReplaceAllLiteralString(ch.Content, href)
		}
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
