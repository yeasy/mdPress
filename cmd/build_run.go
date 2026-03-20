package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/sync/errgroup"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/i18n"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/output"
	"github.com/yeasy/mdpress/internal/plugin"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"github.com/yeasy/mdpress/pkg/utils"
)

type languageBuildSummary struct {
	Name    string
	Dir     string
	Title   string
	Outputs map[string]string
}

func loadCustomCSS(cfg *config.BookConfig, logger *slog.Logger) string {
	if cfg.Style.CustomCSS == "" {
		return ""
	}
	cssPath := cfg.ResolvePath(cfg.Style.CustomCSS)
	cssData, err := utils.ReadFile(cssPath)
	if err != nil {
		logger.Warn("Failed to load custom CSS", slog.String("error", err.Error()))
		return ""
	}
	return string(cssData)
}

func executeMultilingualBuild(ctx context.Context, rootDir string, langs []i18n.LangDef, formats []string, outputOverride string, logger *slog.Logger) error {
	logger.Info("Detected multi-language project", slog.Int("languages", len(langs)))
	for _, lang := range langs {
		logger.Info("  Language", slog.String("name", lang.Name), slog.String("dir", lang.Dir))
	}

	summaries := make([]languageBuildSummary, 0, len(langs))
	for _, lang := range langs {
		langDir := filepath.Join(rootDir, lang.Dir)
		logger.Info("Building language variant", slog.String("name", lang.Name), slog.String("dir", langDir))

		langCfg, err := config.Discover(langDir)
		if err != nil {
			return fmt.Errorf("failed to load language directory %s: %w", langDir, err)
		}

		// Only override the language code when none was explicitly set by the
		// user. The previous logic also overwrote an explicit "zh-CN" when
		// the directory name implied a different language, which was wrong.
		if guessed := guessLanguageCode(lang.Dir); guessed != "" {
			if langCfg.Book.Language == "" {
				langCfg.Book.Language = guessed
			}
		}

		langOutputOverride := deriveLanguageOutputOverride(outputOverride, lang.Dir)
		baseOutput, err := resolveBuildBaseOutput(langCfg, langOutputOverride)
		if err != nil {
			return err
		}
		if err := executeBuildForConfig(ctx, langCfg, formats, langOutputOverride, logger); err != nil {
			return fmt.Errorf("failed to build language %s: %w", lang.Dir, err)
		}
		summaries = append(summaries, languageBuildSummary{
			Name:    lang.Name,
			Dir:     lang.Dir,
			Title:   langCfg.Book.Title,
			Outputs: predictedOutputLinks(baseOutput, formats),
		})
	}

	if err := writeMultilingualLandingPage(rootDir, outputOverride, summaries); err != nil {
		return fmt.Errorf("failed to write multilingual landing page: %w", err)
	}
	if err := injectMultilingualSwitchers(rootDir, outputOverride, summaries); err != nil {
		return fmt.Errorf("failed to inject multilingual switchers: %w", err)
	}

	return nil
}

func executeBuildForConfig(ctx context.Context, cfg *config.BookConfig, formats []string, outputOverride string, logger *slog.Logger) error {
	logger.Info("Configuration loaded",
		slog.String("title", cfg.Book.Title),
		slog.String("author", cfg.Book.Author),
		slog.String("base_dir", cfg.BaseDir()),
		slog.Int("chapters", len(cfg.Chapters)))

	progress := utils.NewProgressTracker(5)
	needsPDF := containsBuildFormat(formats, "pdf")
	needsNonPDF := containsAnyNonPDFFormat(formats)

	// Initialize incremental build system
	cacheDir := filepath.Join(utils.CacheRootDir(), "build")
	manifest, _ := LoadManifest(cacheDir)
	if manifest == nil {
		manifest = NewBuildManifest(Version)
	}
	stats := NewCacheStatistics()

	// Compute hashes for cache invalidation
	configPath := filepath.Join(cfg.BaseDir(), "book.yaml")
	configHash := ""
	if hash, err := ComputeConfigHash(configPath); err == nil {
		configHash = hash
	}

	progress.Start("Initializing theme system")
	orchestrator, err := NewBuildOrchestrator(cfg, logger)
	if err != nil {
		return err
	}
	progress.DoneWithDetail(orchestrator.Theme.Name)

	// Compute CSS hash for cache invalidation
	customCSS := orchestrator.LoadCustomCSS()
	cssHash := ComputeCSSHash(customCSS)

	// Check if manifest is stale and invalidate if needed
	if manifest.IsStale(Version, configHash, cssHash) {
		logger.Debug("Build manifest is stale, invalidating cache")
		manifest = NewBuildManifest(Version)
		manifest.ConfigSH = configHash
		manifest.CSSHash = cssHash
	}

	// Invoke the BeforeBuild hook so plugins can do pre-build setup work.
	runPluginHook(orchestrator.PluginManager, &plugin.HookContext{
		Context:  ctx,
		Config:   cfg,
		Phase:    plugin.PhaseBeforeBuild,
		Metadata: make(map[string]interface{}),
	}, logger)

	progress.Start(fmt.Sprintf("Parsing chapters (%d top-level)", len(cfg.Chapters)))
	primaryPipelineOptions := ChapterPipelineOptions{ImageOptions: func() *utils.ImageProcessingOptions {
		if needsPDF && !needsNonPDF {
			opts := pdfChapterImageOptions()
			return &opts
		}
		opts := defaultEmbeddedChapterImageOptions()
		return &opts
	}()}
	result, err := orchestrator.ProcessChaptersWithOptions(ctx, primaryPipelineOptions)
	if err != nil {
		progress.Fail()
		return err
	}
	progress.DoneWithDetail(fmt.Sprintf("%d chapters", len(result.Chapters)))
	if len(result.Issues) > 0 {
		reportBuildIssues(logger, result.Issues)
	}

	chaptersHTML := result.Chapters
	chapterFiles := result.ChapterFiles
	allHeadings := result.AllHeadings
	var pdfChaptersHTML []renderer.ChapterHTML
	var pdfChapterFiles []string

	if needsPDF && needsNonPDF {
		pdfOpts := pdfChapterImageOptions()
		pdfResult, pdfErr := orchestrator.ProcessChaptersWithOptions(ctx, ChapterPipelineOptions{ImageOptions: &pdfOpts})
		if pdfErr != nil {
			progress.Fail()
			return pdfErr
		}
		pdfChaptersHTML = pdfResult.Chapters
		pdfChapterFiles = pdfResult.ChapterFiles
	}

	progress.Start("Generating cover and TOC")
	var coverHTML string
	if cfg.Output.Cover {
		coverHTML = cover.NewCoverGenerator(cfg.Book).RenderHTML()
	}

	var tocHTML string
	if cfg.Output.TOC {
		tocHeadings := allHeadings
		if maxDepth := cfg.Output.TOCMaxDepth; maxDepth > 0 && maxDepth < 6 {
			filtered := make([]toc.HeadingInfo, 0, len(allHeadings))
			for _, h := range allHeadings {
				if h.Level <= maxDepth {
					filtered = append(filtered, h)
				}
			}
			tocHeadings = filtered
		}
		entries := toc.NewGenerator().Generate(tocHeadings)
		tocHTML = toc.NewGenerator().RenderHTML(entries)
		logger.Debug("TOC generated", slog.Int("entries", toc.CountEntries(entries)),
			slog.Int("maxDepth", cfg.Output.TOCMaxDepth))
	}
	progress.Done()

	var glossaryHTML string
	if orchestrator.Gloss != nil && len(orchestrator.Gloss.Terms) > 0 {
		glossaryHTML = orchestrator.Gloss.RenderHTML()
	}

	singlePageChapters := rewriteChapterLinks(chaptersHTML, chapterFiles)

	progress.Start("Assembling HTML")
	if glossaryHTML != "" {
		glossaryChapter := renderer.ChapterHTML{
			Title:   "Glossary",
			ID:      "glossary",
			Content: glossaryHTML,
		}
		chaptersHTML = append(chaptersHTML, glossaryChapter)
		singlePageChapters = append(singlePageChapters, glossaryChapter)
		chapterFiles = append(chapterFiles, "")
		if needsPDF && needsNonPDF {
			pdfChaptersHTML = append(pdfChaptersHTML, glossaryChapter)
			pdfChapterFiles = append(pdfChapterFiles, "")
		}
	}

	pdfSinglePageChapters := singlePageChapters
	if needsPDF && needsNonPDF {
		pdfSinglePageChapters = rewriteChapterLinks(pdfChaptersHTML, pdfChapterFiles)
	}

	// Invoke the BeforeRender hook before the final HTML document is assembled.
	// The cover HTML is passed as the content payload so plugins can inspect it.
	runPluginHook(orchestrator.PluginManager, &plugin.HookContext{
		Context:  ctx,
		Config:   cfg,
		Phase:    plugin.PhaseBeforeRender,
		Content:  coverHTML,
		Metadata: make(map[string]interface{}),
	}, logger)

	singlePageParts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: singlePageChapters,
		CustomCSS:    customCSS,
	}
	var pdfSinglePageParts *renderer.RenderParts
	if needsPDF && needsNonPDF {
		pdfSinglePageParts = &renderer.RenderParts{
			CoverHTML:    coverHTML,
			TOCHTML:      tocHTML,
			ChaptersHTML: pdfSinglePageChapters,
			CustomCSS:    customCSS,
		}
	}

	// Invoke the AfterRender hook after HTML assembly is complete.
	// The TOC HTML is passed as the content payload.
	runPluginHook(orchestrator.PluginManager, &plugin.HookContext{
		Context:  ctx,
		Config:   cfg,
		Phase:    plugin.PhaseAfterRender,
		Content:  tocHTML,
		Metadata: make(map[string]interface{}),
	}, logger)

	progress.Done()

	progress.Start(fmt.Sprintf("Generating output (%s)", strings.Join(formats, ", ")))
	baseOutput, err := resolveBuildBaseOutput(cfg, outputOverride)
	if err != nil {
		return err
	}
	baseName := strings.TrimSuffix(baseOutput, filepath.Ext(baseOutput))

	buildCtx := &BuildContext{
		Config:             cfg,
		Theme:              orchestrator.Theme,
		SinglePageParts:    singlePageParts,
		PDFSinglePageParts: pdfSinglePageParts,
		ChaptersHTML:       chaptersHTML,
		ChapterFiles:       chapterFiles,
		CustomCSS:          customCSS,
		Logger:             logger,
	}
	registry := NewFormatBuilderRegistry()

	// Build formats in parallel (but not PDF with others, as PDF generation is memory-intensive)
	if err := buildFormatsInParallel(ctx, registry, buildCtx, baseName, formats, logger); err != nil {
		return err
	}

	// Invoke the AfterBuild hook after all output formats have been written.
	runPluginHook(orchestrator.PluginManager, &plugin.HookContext{
		Context:  ctx,
		Config:   cfg,
		Phase:    plugin.PhaseAfterBuild,
		Metadata: make(map[string]interface{}),
	}, logger)

	// Release plugin resources.
	if err := orchestrator.PluginManager.CleanupAll(); err != nil {
		logger.Warn("plugin cleanup failed", slog.String("error", err.Error()))
	}

	// Save the build manifest for incremental builds
	manifest.ConfigSH = configHash
	manifest.CSSHash = cssHash
	if err := SaveManifest(cacheDir, manifest); err != nil {
		logger.Warn("failed to save build manifest", slog.String("error", err.Error()))
	}

	// Log cache statistics if we tracked any
	if stats.Total > 0 {
		logger.Info(stats.String())
	}

	progress.Done()
	progress.Finish()
	return nil
}

// buildFormatsInParallel builds multiple output formats in parallel.
// PDF formats are built sequentially (they're memory-intensive), while other formats build in parallel.
func buildFormatsInParallel(ctx context.Context, registry *FormatBuilderRegistry, buildCtx *BuildContext, baseName string, formats []string, logger *slog.Logger) error {
	// Separate PDF from other formats
	var pdfFormats []string
	var otherFormats []string

	for _, format := range formats {
		lower := strings.ToLower(strings.TrimSpace(format))
		if lower == "pdf" {
			pdfFormats = append(pdfFormats, format)
		} else {
			otherFormats = append(otherFormats, format)
		}
	}

	// Build non-PDF formats in parallel
	if len(otherFormats) > 0 {
		eg, _ := errgroup.WithContext(ctx)
		var mu sync.Mutex

		for _, format := range otherFormats {
			format := format // capture for closure
			eg.Go(func() error {
				builder := registry.Get(strings.ToLower(format))
				if builder == nil {
					mu.Lock()
					logger.Warn("Unsupported output format, skipping", slog.String("format", format))
					mu.Unlock()
					return nil
				}
				return builder.Build(buildCtx, baseName)
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	// Build PDF formats sequentially (they're resource-intensive)
	for _, format := range pdfFormats {
		builder := registry.Get(strings.ToLower(format))
		if builder == nil {
			logger.Warn("Unsupported output format, skipping", slog.String("format", format))
			continue
		}
		if err := builder.Build(buildCtx, baseName); err != nil {
			return err
		}
	}

	return nil
}

func containsBuildFormat(formats []string, target string) bool {
	for _, format := range formats {
		if strings.EqualFold(strings.TrimSpace(format), target) {
			return true
		}
	}
	return false
}

func containsAnyNonPDFFormat(formats []string) bool {
	for _, format := range formats {
		if !strings.EqualFold(strings.TrimSpace(format), "pdf") {
			return true
		}
	}
	return false
}

// runPluginHook dispatches a hook to the plugin manager.
// Errors are logged as warnings; they never abort the build.
func runPluginHook(mgr *plugin.Manager, hookCtx *plugin.HookContext, logger *slog.Logger) {
	if mgr == nil {
		return
	}
	if err := mgr.RunHook(hookCtx); err != nil {
		logger.Warn("plugin hook failed",
			slog.String("phase", string(hookCtx.Phase)),
			slog.String("error", err.Error()))
	}
}

var (
	decimalTitleSequencePattern = regexp.MustCompile(`^\s*(\d+(?:\.\d+)*)(?:\s*[.)、．:：）-]\s*|\s+|$)`)
	chineseTitleSequencePattern = regexp.MustCompile(`^\s*第\s*([一二三四五六七八九十百零〇两\d]+)\s*([章节篇部卷])`)
	englishTitleSequencePattern = regexp.MustCompile(`^\s*(?:Chapter|CHAPTER)\s+(\d+(?:\.\d+)*)\b`)
)

type chapterHeadingRecord struct {
	File         string
	SummaryTitle string
	Heading      markdown.HeadingInfo
}

type chapterHeadingWarning struct {
	File       string
	Diagnostic markdown.Diagnostic
}

func validateChapterTitleSequence(summaryTitle string, headings []markdown.HeadingInfo) *markdown.Diagnostic {
	if summaryTitle == "" || len(headings) == 0 {
		return nil
	}

	actual := headings[0]
	expectedSeq, expectedHas := extractTitleSequence(summaryTitle)
	actualSeq, actualHas := extractTitleSequence(actual.Text)
	if !expectedHas || !actualHas {
		return nil
	}
	if expectedSeq == actualSeq {
		return nil
	}

	line, column := actual.Line, actual.Column
	if line <= 0 {
		line = 1
	}
	if column <= 0 {
		column = 1
	}

	return &markdown.Diagnostic{
		Rule:   "chapter-title-sequence",
		Line:   line,
		Column: column,
		Message: fmt.Sprintf("summary title numbering does not match the chapter heading: summary=%q, heading=%q",
			summaryTitle, actual.Text),
	}
}

// compatibleTitleStyles reports whether two numbering styles can coexist
// without triggering a style-mismatch warning.  Chinese technical books
// commonly use Chinese ordinals for top-level chapters (第一章, 第二章) and
// Arabic decimals for sections (1.1, 2.3), so "chinese" + "arabic" is a
// natural pairing that should not be flagged.
func compatibleTitleStyles(a, b string) bool {
	if a == b {
		return true
	}
	if a == "none" || b == "none" {
		return true
	}
	// Chinese chapter-level + Arabic section-level is standard practice.
	if (a == "chinese" && b == "arabic") || (a == "arabic" && b == "chinese") {
		return true
	}
	return false
}

// commonRecurringSectionTitles lists short generic headings that naturally
// appear in every chapter of a book (e.g. "本章小结", "简介", "总结").
// These are excluded from the duplicate-title check because flagging them
// produces only false positives in multi-chapter books.
var commonRecurringSectionTitles = map[string]bool{
	// Chinese
	"本章小结": true, "小结": true, "总结": true, "简介": true, "介绍": true,
	"概述": true, "参考资料": true, "参考文献": true, "习题": true, "练习": true,
	"思考题": true, "延伸阅读": true, "本章总结": true, "章节总结": true,
	// English
	"summary": true, "introduction": true, "overview": true,
	"conclusion": true, "references": true, "exercises": true,
	"further reading": true, "chapter summary": true,
}

func validateBookTitleConsistency(records []chapterHeadingRecord) []chapterHeadingWarning {
	if len(records) < 2 {
		return nil
	}

	warnings := make([]chapterHeadingWarning, 0)

	// Collect all styles present in the book.
	primaryStyle := ""
	for _, record := range records {
		style := titleSequenceStyle(record.Heading.Text)
		if style != "none" {
			primaryStyle = style
			break
		}
	}

	if primaryStyle != "" {
		for _, record := range records {
			style := titleSequenceStyle(record.Heading.Text)
			if compatibleTitleStyles(primaryStyle, style) {
				continue
			}
			warnings = append(warnings, chapterHeadingWarning{
				File: record.File,
				Diagnostic: markdown.Diagnostic{
					Rule:    "book-title-style",
					Line:    max(record.Heading.Line, 1),
					Column:  max(record.Heading.Column, 1),
					Message: describeTitleStyleMismatch(primaryStyle, record.Heading.Text),
				},
			})
		}
	}

	// Duplicate title detection: use directory-scoped keys so that
	// "chapter01/本章小结" and "chapter02/本章小结" are treated as distinct.
	// Also skip common recurring section titles entirely.
	seenTitles := make(map[string]chapterHeadingRecord)
	for _, record := range records {
		normalized := normalizeChapterTitle(record.Heading.Text)
		if normalized == "" {
			continue
		}

		// Skip common section headings that naturally repeat across chapters.
		if commonRecurringSectionTitles[strings.ToLower(normalized)] {
			continue
		}

		// Scope by the top-level directory so that "ch01/简介" and "ch02/简介"
		// don't collide.  Files at the root level use "" as their scope.
		scope := ""
		if idx := strings.IndexAny(record.File, "/\\"); idx >= 0 {
			scope = record.File[:idx]
		}
		key := scope + "\x00" + normalized

		if prev, ok := seenTitles[key]; ok {
			warnings = append(warnings, chapterHeadingWarning{
				File: record.File,
				Diagnostic: markdown.Diagnostic{
					Rule:   "book-title-duplicate",
					Line:   max(record.Heading.Line, 1),
					Column: max(record.Heading.Column, 1),
					Message: fmt.Sprintf("possible duplicate chapter title: current title %q normalizes to the same value as %q in %s",
						record.Heading.Text, prev.File, prev.Heading.Text),
				},
			})
			continue
		}
		seenTitles[key] = record
	}

	return warnings
}

func extractTitleSequence(title string) (string, bool) {
	if matches := decimalTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return matches[1], true
	}
	if matches := englishTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return matches[1], true
	}
	if matches := chineseTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 3 {
		value := parseChineseOrdinal(matches[1])
		if value > 0 || strings.ContainsAny(matches[1], "零〇0") {
			return strconv.Itoa(value), true
		}
	}
	return "", false
}

func titleSequenceStyle(title string) string {
	if matches := decimalTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return "arabic"
	}
	if matches := englishTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return "english"
	}
	if matches := chineseTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 3 {
		return "chinese"
	}
	return "none"
}

func describeTitleStyleMismatch(primaryStyle, title string) string {
	currentStyle := titleSequenceStyle(title)
	switch {
	case currentStyle == "none":
		return fmt.Sprintf("inconsistent chapter numbering style: earlier chapters use %s numbering, but %q has no numbering", titleStyleLabel(primaryStyle), title)
	case currentStyle != primaryStyle:
		return fmt.Sprintf("inconsistent chapter numbering style: earlier chapters use %s numbering, but %q uses %s numbering",
			titleStyleLabel(primaryStyle), title, titleStyleLabel(currentStyle))
	default:
		return ""
	}
}

func titleStyleLabel(style string) string {
	switch style {
	case "arabic":
		return "Arabic"
	case "english":
		return "English chapter"
	case "chinese":
		return "Chinese chapter"
	default:
		return "unnumbered"
	}
}

func normalizeChapterTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	if matches := decimalTitleSequencePattern.FindStringSubmatchIndex(title); matches != nil {
		title = strings.TrimSpace(title[matches[1]:])
	}
	if matches := englishTitleSequencePattern.FindStringSubmatchIndex(title); matches != nil {
		title = strings.TrimSpace(title[matches[1]:])
	}
	if matches := chineseTitleSequencePattern.FindStringSubmatchIndex(title); matches != nil {
		title = strings.TrimSpace(title[matches[1]:])
	}
	title = strings.TrimLeft(title, ":-：.、)） \t")
	return strings.Join(strings.Fields(title), " ")
}

func parseChineseOrdinal(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if value, err := strconv.Atoi(raw); err == nil {
		return value
	}

	digits := map[rune]int{
		'零': 0, '〇': 0,
		'一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9,
	}
	units := map[rune]int{'十': 10, '百': 100}

	total := 0
	current := 0
	for _, r := range raw {
		if value, ok := digits[r]; ok {
			current = value
			continue
		}
		unit, ok := units[r]
		if !ok {
			return 0
		}
		if current == 0 {
			current = 1
		}
		total += current * unit
		current = 0
	}
	total += current
	return total
}

func resolveRequestedBuildOutput(requested string) (string, error) {
	if requested == "" {
		return "", nil
	}
	absPath, err := filepath.Abs(requested)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output path: %w", err)
	}
	return absPath, nil
}

// deriveOutputFilename returns the base output filename for the book.
// Priority: explicit config filename > title-based name > directory name > "output".
func deriveOutputFilename(cfg *config.BookConfig) string {
	if cfg.Output.Filename != "" && cfg.Output.Filename != "output.pdf" {
		return cfg.Output.Filename
	}
	title := cfg.Book.Title
	if title == "" || title == "Untitled Book" {
		title = filepath.Base(cfg.BaseDir())
	}
	return sanitizeBookFilename(title) + ".pdf"
}

// filenameReplacer strips characters that are invalid in file system names.
var filenameReplacer = strings.NewReplacer(
	"/", "_", "\\", "_", ":", "_", "*", "_",
	"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
)

// sanitizeBookFilename strips characters that are invalid in file system names.
func sanitizeBookFilename(s string) string {
	result := strings.TrimSpace(filenameReplacer.Replace(s))
	if !strings.ContainsFunc(result, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsNumber(r)
	}) {
		return "output"
	}
	if result == "" {
		return "output"
	}
	return result
}

func resolveBuildBaseOutput(cfg *config.BookConfig, outputOverride string) (string, error) {
	filename := deriveOutputFilename(cfg)

	if outputOverride == "" {
		return cfg.ResolvePath(filename), nil
	}

	if info, err := os.Stat(outputOverride); err == nil && info.IsDir() {
		return filepath.Join(outputOverride, filepath.Base(filename)), nil
	}

	return outputOverride, nil
}

func deriveLanguageOutputOverride(outputOverride string, langDir string) string {
	if outputOverride == "" {
		return ""
	}

	if info, err := os.Stat(outputOverride); err == nil && info.IsDir() {
		return filepath.Join(outputOverride, langDir, "output")
	}

	ext := filepath.Ext(outputOverride)
	if ext == "" {
		return outputOverride + "-" + langDir
	}

	return strings.TrimSuffix(outputOverride, ext) + "-" + langDir + ext
}

func predictedOutputLinks(baseOutput string, formats []string) map[string]string {
	links := make(map[string]string, len(formats))
	baseName := strings.TrimSuffix(baseOutput, filepath.Ext(baseOutput))
	for _, format := range formats {
		switch strings.ToLower(format) {
		case "pdf":
			links["pdf"] = baseName + ".pdf"
		case "html":
			links["html"] = baseName + ".html"
		case "site":
			links["site"] = filepath.Join(baseName+"_site", "index.html")
		case "epub":
			links["epub"] = baseName + ".epub"
		}
	}
	return links
}

func multilingualLandingPath(rootDir string, outputOverride string) string {
	if outputOverride == "" {
		return filepath.Join(rootDir, "_mdpress_langs.html")
	}
	if info, err := os.Stat(outputOverride); err == nil && info.IsDir() {
		return filepath.Join(outputOverride, "index.html")
	}
	dir := filepath.Dir(outputOverride)
	base := strings.TrimSuffix(filepath.Base(outputOverride), filepath.Ext(outputOverride))
	if base == "" || base == "." {
		base = "index"
	}
	return filepath.Join(dir, base+"-index.html")
}

func writeMultilingualLandingPage(rootDir string, outputOverride string, summaries []languageBuildSummary) error {
	if len(summaries) == 0 {
		return nil
	}

	landingPath := multilingualLandingPath(rootDir, outputOverride)
	landingDir := filepath.Dir(landingPath)
	if err := utils.EnsureDir(landingDir); err != nil {
		return err
	}
	defaultTarget := defaultLanguageTarget(landingDir, summaries)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString("<title>Language Variants</title>\n<style>\n")
	b.WriteString("body{font-family:-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;background:#f6f7fb;color:#1f2937;margin:0;padding:40px;line-height:1.6;} ")
	b.WriteString(".wrap{max-width:920px;margin:0 auto;} h1{margin:0 0 8px;font-size:2rem;} p{color:#4b5563;} ")
	b.WriteString(".notice{margin-top:8px;padding:10px 14px;background:#eef2ff;border:1px solid #c7d2fe;border-radius:10px;color:#3730a3;} ")
	b.WriteString(".grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:16px;margin-top:24px;} ")
	b.WriteString(".card{background:#fff;border:1px solid #e5e7eb;border-radius:14px;padding:18px 20px;box-shadow:0 6px 24px rgba(15,23,42,.06);} ")
	b.WriteString(".card h2{margin:0 0 4px;font-size:1.1rem;} .meta{color:#6b7280;font-size:.9rem;margin-bottom:12px;} ")
	b.WriteString("ul{list-style:none;padding:0;margin:0;} li+li{margin-top:8px;} a{color:#2563eb;text-decoration:none;} a:hover{text-decoration:underline;}\n")
	b.WriteString("</style>\n</head>\n<body>\n<div class=\"wrap\">\n<h1>Language Variants</h1>\n")
	b.WriteString("<p>Select a language output generated from this multi-language project.</p>\n<div class=\"grid\">\n")
	if defaultTarget != "" {
		fmt.Fprintf(&b, "<div class=\"notice\">Redirecting to the default language in a moment. If you prefer, choose a language below. <a href=\"%s\">Open default now</a>.</div>\n", utils.EscapeHTML(defaultTarget))
	}

	for _, summary := range summaries {
		b.WriteString("<section class=\"card\">\n")
		fmt.Fprintf(&b, "<h2>%s</h2>\n", utils.EscapeHTML(summary.Name))
		if summary.Title != "" {
			fmt.Fprintf(&b, "<div class=\"meta\">%s</div>\n", utils.EscapeHTML(summary.Title))
		} else {
			fmt.Fprintf(&b, "<div class=\"meta\">%s</div>\n", utils.EscapeHTML(summary.Dir))
		}
		b.WriteString("<ul>\n")
		for _, key := range []string{"html", "site", "pdf", "epub"} {
			target, ok := summary.Outputs[key]
			if !ok {
				continue
			}
			rel, err := filepath.Rel(landingDir, target)
			if err != nil {
				rel = target
			}
			fmt.Fprintf(&b, "<li><a href=\"%s\">%s</a></li>\n", utils.EscapeHTML(filepath.ToSlash(rel)), strings.ToUpper(key))
		}
		b.WriteString("</ul>\n</section>\n")
	}

	if defaultTarget != "" {
		fmt.Fprintf(&b, "<script>setTimeout(function(){ window.location.href = %q; }, 1200);</script>\n", defaultTarget)
	}
	b.WriteString("</div>\n</div>\n</body>\n</html>\n")
	return os.WriteFile(landingPath, []byte(b.String()), 0644)
}

func defaultLanguageTarget(landingDir string, summaries []languageBuildSummary) string {
	if len(summaries) == 0 {
		return ""
	}
	for _, summary := range summaries {
		for _, key := range []string{"html", "site"} {
			if target, ok := summary.Outputs[key]; ok {
				rel, err := filepath.Rel(landingDir, target)
				if err == nil {
					return filepath.ToSlash(rel)
				}
			}
		}
	}
	return ""
}

func injectMultilingualSwitchers(rootDir string, outputOverride string, summaries []languageBuildSummary) error {
	if len(summaries) < 2 {
		return nil
	}

	landingPath := multilingualLandingPath(rootDir, outputOverride)
	for _, summary := range summaries {
		currentTarget := preferredLanguageFile(summary)
		if currentTarget == "" {
			continue
		}
		switcherHTML, err := buildLanguageSwitcherHTML(filepath.Dir(currentTarget), landingPath, summaries, summary.Dir)
		if err != nil {
			return err
		}
		if err := injectBannerIntoOutput(currentTarget, switcherHTML); err != nil {
			return err
		}
		if siteIndex, ok := summary.Outputs["site"]; ok {
			siteDir := filepath.Dir(siteIndex)
			if err := injectBannerIntoSite(siteDir, switcherHTML); err != nil {
				return err
			}
		}
	}
	return nil
}

func preferredLanguageFile(summary languageBuildSummary) string {
	for _, key := range []string{"html", "site"} {
		if target, ok := summary.Outputs[key]; ok {
			return target
		}
	}
	return ""
}

func buildLanguageSwitcherHTML(currentDir, landingPath string, summaries []languageBuildSummary, currentLangDir string) (string, error) {
	landingRel, err := filepath.Rel(currentDir, landingPath)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<style>.mdpress-lang-switcher{position:sticky;top:0;z-index:9999;display:flex;flex-wrap:wrap;gap:10px;align-items:center;padding:10px 16px;background:#111827;color:#f9fafb;font:14px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;box-shadow:0 6px 16px rgba(0,0,0,.15)}.mdpress-lang-switcher a{color:#93c5fd;text-decoration:none}.mdpress-lang-switcher a:hover{text-decoration:underline}.mdpress-lang-switcher .current{font-weight:700;color:#fff}</style>`)
	b.WriteString(`<nav class="mdpress-lang-switcher" aria-label="Language switcher"><span>Languages:</span>`)
	for _, summary := range summaries {
		target := preferredLanguageFile(summary)
		if target == "" {
			continue
		}
		rel, err := filepath.Rel(currentDir, target)
		if err != nil {
			return "", err
		}
		if summary.Dir == currentLangDir {
			fmt.Fprintf(&b, `<span class="current">%s</span>`, utils.EscapeHTML(summary.Name))
		} else {
			fmt.Fprintf(&b, `<a href="%s">%s</a>`, utils.EscapeHTML(filepath.ToSlash(rel)), utils.EscapeHTML(summary.Name))
		}
	}
	fmt.Fprintf(&b, `<a href="%s">All languages</a>`, utils.EscapeHTML(filepath.ToSlash(landingRel)))
	b.WriteString(`</nav>`)
	return b.String(), nil
}

func injectBannerIntoOutput(targetPath string, bannerHTML string) error {
	content, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File does not exist yet; skip silently.
		}
		return fmt.Errorf("failed to read %s for language switcher injection: %w", targetPath, err)
	}
	updated := injectBannerIntoHTML(string(content), bannerHTML)
	if updated == string(content) {
		return nil
	}
	return os.WriteFile(targetPath, []byte(updated), 0644)
}

func injectBannerIntoSite(siteDir string, bannerHTML string) error {
	return filepath.Walk(siteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.ToLower(filepath.Ext(path)) != ".html" {
			return err
		}
		return injectBannerIntoOutput(path, bannerHTML)
	})
}

func injectBannerIntoHTML(htmlContent string, bannerHTML string) string {
	if strings.Contains(htmlContent, `class="mdpress-lang-switcher"`) {
		return htmlContent
	}
	if idx := strings.Index(strings.ToLower(htmlContent), "<body>"); idx >= 0 {
		insertAt := idx + len("<body>")
		return htmlContent[:insertAt] + bannerHTML + htmlContent[insertAt:]
	}
	return bannerHTML + htmlContent
}

func guessLanguageCode(langDir string) string {
	switch strings.ToLower(langDir) {
	case "en", "en-us":
		return "en-US"
	case "cn", "zh", "zh-cn":
		return "zh-CN"
	case "zh-tw":
		return "zh-TW"
	case "ja", "ja-jp":
		return "ja-JP"
	case "ko", "ko-kr":
		return "ko-KR"
	default:
		return ""
	}
}

func sitePageFilenames(count int) []string {
	files := make([]string, 0, count)
	for i := 0; i < count; i++ {
		files = append(files, fmt.Sprintf("ch_%03d.html", i))
	}
	return files
}

func rewriteChapterLinksForSite(chapters []renderer.ChapterHTML, chapterFiles []string, pageFilenames []string) []renderer.ChapterHTML {
	if len(chapters) == 0 || len(chapters) != len(chapterFiles) || len(chapters) != len(pageFilenames) {
		return chapters
	}

	targets := make(map[string]linkrewrite.Target, len(chapters))
	for i, ch := range chapters {
		if chapterFiles[i] == "" || ch.ID == "" {
			continue
		}
		targets[linkrewrite.NormalizePath(chapterFiles[i])] = linkrewrite.Target{
			ChapterID:    ch.ID,
			PageFilename: pageFilenames[i],
		}
	}

	rewritten := make([]renderer.ChapterHTML, len(chapters))
	for i, ch := range chapters {
		rewritten[i] = ch
		rewritten[i].Content = linkrewrite.RewriteLinks(ch.Content, chapterFiles[i], targets, linkrewrite.ModeSite)
	}
	return rewritten
}

func generateSiteOutput(cfg *config.BookConfig, thm *theme.Theme, customCSS, outputDir string, chapters []renderer.ChapterHTML, pageFilenames []string) error {
	siteGen := output.NewSiteGenerator(output.SiteMeta{
		Title:    cfg.Book.Title,
		Author:   cfg.Book.Author,
		Language: cfg.Book.Language,
	})
	siteGen.SetCSS(thm.ToCSS() + "\n" + customCSS)

	for _, ch := range buildSiteChapterTree(cfg.Chapters, chapters, pageFilenames) {
		siteGen.AddChapter(ch)
	}

	return siteGen.Generate(outputDir)
}

func buildSiteChapterTree(defs []config.ChapterDef, chapters []renderer.ChapterHTML, pageFilenames []string) []output.SiteChapter {
	flatDefs := flattenChaptersWithDepth(defs)
	type siteChapterData struct {
		html     renderer.ChapterHTML
		filename string
	}
	byFile := make(map[string]siteChapterData, len(flatDefs))
	for i, flat := range flatDefs {
		if i >= len(chapters) {
			break
		}
		filename := ""
		if i < len(pageFilenames) {
			filename = pageFilenames[i]
		}
		byFile[linkrewrite.NormalizePath(flat.Def.File)] = siteChapterData{
			html:     chapters[i],
			filename: filename,
		}
	}

	var build func([]config.ChapterDef) []output.SiteChapter
	build = func(items []config.ChapterDef) []output.SiteChapter {
		result := make([]output.SiteChapter, 0, len(items))
		for _, def := range items {
			data, ok := byFile[linkrewrite.NormalizePath(def.File)]
			if !ok {
				continue
			}
			result = append(result, output.SiteChapter{
				Title:    data.html.Title,
				ID:       data.html.ID,
				Filename: data.filename,
				Content:  data.html.Content,
				Depth:    data.html.Depth,
				Headings: rendererHeadingsToSiteHeadings(data.html.Headings),
				Children: build(def.Sections),
			})
		}
		return result
	}

	return build(defs)
}

func rendererHeadingsToSiteHeadings(items []renderer.NavHeading) []output.SiteNavHeading {
	result := make([]output.SiteNavHeading, 0, len(items))
	for _, item := range items {
		result = append(result, output.SiteNavHeading{
			Title:    item.Title,
			ID:       item.ID,
			Children: rendererHeadingsToSiteHeadings(item.Children),
		})
	}
	return result
}
