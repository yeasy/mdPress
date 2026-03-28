package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/crossref"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/plugin"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"github.com/yeasy/mdpress/internal/variables"
	"github.com/yeasy/mdpress/pkg/utils"
)

// ChapterPipelineOptions controls expensive per-chapter processing behavior.
type ChapterPipelineOptions struct {
	ImageOptions *utils.ImageProcessingOptions
	// MaxConcurrency controls how many chapters are parsed in parallel.
	// If 0, defaults to runtime.NumCPU() (capped at 8).
	// If negative, sequential processing (concurrency = 1).
	MaxConcurrency int
}

// parsedChapterData holds the parsed output of a single chapter.
type parsedChapterData struct {
	index            int
	chDef            config.ChapterDef
	chapterPath      string
	htmlContent      string
	headings         []markdown.HeadingInfo
	diagnostics      []markdown.Diagnostic
	expandedContent  string
	depth            int
	flatChapterIndex int
	err              error
}

// ChapterPipelineResult encapsulates the output of chapter processing.
type ChapterPipelineResult struct {
	Chapters        []renderer.ChapterHTML
	ChapterFiles    []string
	ChapterMarkdown []string
	Issues          []projectIssue
	AllHeadings     []toc.HeadingInfo
	Resolver        *crossref.Resolver
	HeadingRecords  []chapterHeadingRecord
}

// ChapterPipeline orchestrates the complete chapter processing workflow.
type ChapterPipeline struct {
	Config   *config.BookConfig
	Theme    *theme.Theme
	Parser   *markdown.Parser
	Glossary *glossary.Glossary
	Logger   *slog.Logger
	// PluginManager is invoked at the AfterParse hook, allowing plugins to
	// transform the HTML of each chapter after Markdown parsing.
	PluginManager *plugin.Manager
}

// NewChapterPipeline creates a new chapter pipeline with the given configuration.
func NewChapterPipeline(cfg *config.BookConfig, thm *theme.Theme, parser *markdown.Parser, gloss *glossary.Glossary, logger *slog.Logger, mgr *plugin.Manager) *ChapterPipeline {
	if mgr == nil {
		mgr = plugin.NewManager()
	}
	return &ChapterPipeline{
		Config:        cfg,
		Theme:         thm,
		Parser:        parser,
		Glossary:      gloss,
		Logger:        logger,
		PluginManager: mgr,
	}
}

// maxConcurrency caps parallel workers to avoid memory issues from
// multiple Chrome/typst instances.
const maxConcurrency = 8

// computeMaxConcurrency determines the number of worker goroutines to use.
// Returns at least 1 (sequential), at most maxConcurrency.
func computeMaxConcurrency(requested int) int {
	if requested < 0 {
		return 1 // Sequential processing
	}
	if requested > 0 {
		if requested > maxConcurrency {
			return maxConcurrency
		}
		return requested
	}
	// Default: use number of CPUs, capped at maxConcurrency
	numCPU := runtime.NumCPU()
	if numCPU <= 0 {
		numCPU = 1
	}
	if numCPU > maxConcurrency {
		numCPU = maxConcurrency
	}
	return numCPU
}

// parseChaptersParallel parses chapters in parallel using a worker pool.
// It maintains chapter order by accepting results indexed by their position.
// If any chapter fails, it returns the first error immediately.
func (p *ChapterPipeline) parseChaptersParallel(
	ctx context.Context,
	flatChapters []flattenedChapter,
	imageOptions utils.ImageProcessingOptions,
	maxConcurrency int,
) ([]parsedChapterData, error) {
	maxConcurrency = computeMaxConcurrency(maxConcurrency)

	results := make([]parsedChapterData, len(flatChapters))
	resultsChan := make(chan *parsedChapterData, maxConcurrency)
	jobsChan := make(chan *parsedChapterData, maxConcurrency)

	// Shared state
	var mu sync.Mutex
	var firstErr error

	// Launch workers.
	// Each worker gets its own Parser instance because goldmark.Parser (and
	// our wrapper's heading collector) carries mutable per-parse state that is
	// not safe for concurrent use.
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		workerParser := markdown.NewParser(markdown.WithCodeTheme(p.parserCodeTheme()))
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("chapter parsing worker panicked", slog.Any("panic", r))
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("worker panic: %v", r)
					}
					mu.Unlock()
				}
			}()
			for job := range jobsChan {
				// Check context cancellation
				if err := ctx.Err(); err != nil {
					job.err = err
					resultsChan <- job
					continue
				}

				// Check if an earlier job failed
				mu.Lock()
				if firstErr != nil {
					mu.Unlock()
					job.err = firstErr
					resultsChan <- job
					continue
				}
				mu.Unlock()

				// Parse this chapter with the worker-local parser
				p.parseChapterWorker(ctx, job, imageOptions, workerParser)
				resultsChan <- job
			}
		}()
	}

	// Send jobs to workers
	go func() {
		defer close(jobsChan)
		for i, flatChapter := range flatChapters {
			chDef := flatChapter.Def
			chapterPath := p.Config.ResolvePath(chDef.File)

			job := &parsedChapterData{
				flatChapterIndex: i,
				index:            i,
				chDef:            chDef,
				chapterPath:      chapterPath,
				depth:            flatChapter.Depth,
			}
			jobsChan <- job
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		if result.err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = result.err
			}
			mu.Unlock()
		}
		results[result.index] = *result
	}

	// Check for context cancellation or errors
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// parseChapterWorker performs the parsing for a single chapter.
// It's designed to be run in a worker goroutine.
// Returns with job.err != nil if the chapter could not be read or parsed.
// Returns with job.err == nil on success.
// parserCodeTheme returns the code highlighting theme from config or theme.
func (p *ChapterPipeline) parserCodeTheme() string {
	codeTheme := p.Config.Style.CodeTheme
	if codeTheme == "" && p.Theme != nil {
		codeTheme = p.Theme.CodeTheme
	}
	if codeTheme == "" {
		codeTheme = "github"
	}
	return codeTheme
}

func (p *ChapterPipeline) parseChapterWorker(
	ctx context.Context,
	job *parsedChapterData,
	imageOptions utils.ImageProcessingOptions,
	workerParser *markdown.Parser,
) {
	chDef := job.chDef
	chapterPath := job.chapterPath

	p.Logger.Debug("Processing chapter", slog.Int("index", job.flatChapterIndex+1), slog.String("file", chDef.File))

	// Read file
	content, err := utils.ReadFile(chapterPath)
	if err != nil {
		p.Logger.Warn("Failed to read chapter", slog.String("file", chDef.File), slog.String("error", err.Error()))
		job.err = fmt.Errorf("failed to read chapter %q: %w", chDef.File, err)
		return
	}

	// Expand variables
	content = variables.Expand(content, p.Config)
	job.expandedContent = string(content)

	// Check cache
	codeTheme := p.parserCodeTheme()
	cached, cacheHit, cacheErr := loadParsedChapterCache(chapterPath, job.expandedContent, codeTheme)

	var htmlContent string
	var headings []markdown.HeadingInfo
	var diagnostics []markdown.Diagnostic

	switch {
	case cacheErr != nil:
		p.Logger.Debug("Parsed chapter cache read failed", slog.String("file", chDef.File), slog.String("error", cacheErr.Error()))
		fallthrough
	case !cacheHit:
		// Parse markdown
		var parseErr error
		htmlContent, headings, diagnostics, parseErr = workerParser.ParseWithDiagnostics(content)
		if parseErr != nil {
			p.Logger.Warn("Failed to parse Markdown", slog.String("file", chDef.File), slog.String("error", parseErr.Error()))
			job.err = fmt.Errorf("failed to parse chapter %q: %w", chDef.File, parseErr)
			return
		}
		if storeErr := storeParsedChapterCache(chapterPath, job.expandedContent, codeTheme, &cachedParsedChapter{
			HTML:        htmlContent,
			Headings:    headings,
			Diagnostics: diagnostics,
		}); storeErr != nil {
			p.Logger.Debug("Parsed chapter cache write failed", slog.String("file", chDef.File), slog.String("error", storeErr.Error()))
		}
	default:
		// Cache hit
		htmlContent = cached.HTML
		headings = cached.Headings
		diagnostics = cached.Diagnostics
	}

	// Process images
	chapterDir := filepath.Dir(chapterPath)
	imageOptions.Logger = p.Logger
	processedHTML, err := utils.ProcessImagesWithOptions(htmlContent, chapterDir, imageOptions)
	if err != nil {
		p.Logger.Warn("Failed to process images", slog.String("file", chDef.File), slog.String("error", err.Error()))
		p.Logger.Warn("Using original HTML without image processing", slog.String("file", chDef.File))
	} else {
		htmlContent = processedHTML
	}

	job.htmlContent = htmlContent
	job.headings = headings
	job.diagnostics = diagnostics
}

// Process executes the complete chapter processing pipeline.
// It returns processed chapters, chapter file paths, validation issues, and any error encountered.
// Always uses ParseWithDiagnostics regardless of caller preference.
func (p *ChapterPipeline) Process(ctx context.Context) (*ChapterPipelineResult, error) {
	return p.ProcessWithOptions(ctx, ChapterPipelineOptions{})
}

// ProcessWithOptions executes the complete chapter processing pipeline with
// caller-controlled image processing behavior.
func (p *ChapterPipeline) ProcessWithOptions(ctx context.Context, options ChapterPipelineOptions) (*ChapterPipelineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	imageOptions := defaultEmbeddedChapterImageOptions()
	if options.ImageOptions != nil {
		imageOptions = *options.ImageOptions
	}

	resolver := crossref.NewResolver()

	var allHeadings []toc.HeadingInfo
	chaptersHTML := make([]renderer.ChapterHTML, 0, len(p.Config.Chapters))
	chapterFiles := make([]string, 0, len(p.Config.Chapters))
	chapterMarkdown := make([]string, 0, len(p.Config.Chapters))
	issues := make([]projectIssue, 0)
	chapterHeadingRecords := make([]chapterHeadingRecord, 0, len(p.Config.Chapters))

	flatChapters := flattenChaptersWithDepth(p.Config.Chapters)

	// Parse chapters in parallel
	parsedChapters, err := p.parseChaptersParallel(ctx, flatChapters, imageOptions, options.MaxConcurrency)
	if err != nil {
		return nil, err
	}

	// Process results in order
	for i, parsed := range parsedChapters {
		// Skip chapters with no content (they were skipped during parsing)
		if parsed.htmlContent == "" {
			continue
		}

		// Bounds check: ensure flatChapters index is valid
		if i >= len(flatChapters) {
			p.Logger.Error("Internal error: chapter index out of bounds", slog.Int("index", i), slog.Int("flatChapters_len", len(flatChapters)))
			continue
		}

		chDef := parsed.chDef
		htmlContent := parsed.htmlContent
		headings := parsed.headings
		diagnostics := parsed.diagnostics
		flatChapter := flatChapters[i]

		// Collect diagnostic issues.
		for _, diag := range diagnostics {
			issues = append(issues, projectIssue{
				Rule:    diag.Rule,
				File:    chDef.File,
				Line:    diag.Line,
				Column:  diag.Column,
				Message: diag.Message,
			})
		}

		// Validate chapter title sequence.
		if headingWarning := validateChapterTitleSequence(chDef.Title, headings); headingWarning != nil {
			issues = append(issues, projectIssue{
				Rule:    headingWarning.Rule,
				File:    chDef.File,
				Line:    headingWarning.Line,
				Column:  headingWarning.Column,
				Message: headingWarning.Message,
			})
		}

		// Record chapter headings for later consistency validation.
		if len(headings) > 0 {
			chapterHeadingRecords = append(chapterHeadingRecords, chapterHeadingRecord{
				File:         chDef.File,
				SummaryTitle: chDef.Title,
				Heading:      headings[0],
			})
		}

		// Register headings with the cross-reference resolver.
		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
			resolver.RegisterSection(h.ID, h.Text, h.Level)
		}

		// Determine chapter ID (prefer first heading ID, fallback to 1-based index).
		chapterID := fmt.Sprintf("chapter-%d", i+1)
		if len(headings) > 0 {
			chapterID = headings[0].ID
		}

		// Invoke the AfterParse hook so plugins can modify this chapter's HTML.
		hookCtx := &plugin.HookContext{
			Context:      ctx,
			Config:       p.Config,
			Phase:        plugin.PhaseAfterParse,
			Content:      htmlContent,
			ChapterIndex: i,
			ChapterFile:  chDef.File,
			Metadata:     make(map[string]any),
		}
		if err := p.PluginManager.RunHook(hookCtx); err != nil {
			p.Logger.Warn("AfterParse plugin hook failed", slog.String("file", chDef.File), slog.String("error", err.Error()))
		} else if hookCtx.Content != "" {
			htmlContent = hookCtx.Content
		}

		// Process cross-references and glossary.
		htmlContent = resolver.ProcessHTML(htmlContent)
		htmlContent = resolver.AddCaptions(htmlContent)
		if p.Glossary != nil {
			htmlContent = p.Glossary.ProcessHTML(htmlContent)
		}

		// Remove duplicate leading h1 if it matches the SUMMARY title.
		// The template already renders chDef.Title as <h1 class="chapter-title">,
		// so having the same h1 in the content creates a duplicate.
		htmlContent = stripDuplicateLeadingH1(htmlContent, chDef.Title)

		// Build heading tree for navigation.
		headingTree := buildHeadingTree(headings, chapterID)
		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:    chDef.Title,
			ID:       chapterID,
			Content:  htmlContent,
			Depth:    flatChapter.Depth,
			Headings: toRendererNavHeadings(headingTree),
		})
		chapterFiles = append(chapterFiles, linkrewrite.NormalizePath(chDef.File))
		chapterMarkdown = append(chapterMarkdown, parsed.expandedContent)
	}

	// Validate that at least some chapters were processed.
	if len(chaptersHTML) == 0 {
		return nil, errors.New("no chapters were processed successfully (check chapter paths in book.yaml and run mdpress validate)")
	}

	// Validate book title consistency across chapters.
	for _, consistencyWarning := range validateBookTitleConsistency(chapterHeadingRecords) {
		issues = append(issues, projectIssue{
			Rule:    consistencyWarning.Diagnostic.Rule,
			File:    consistencyWarning.File,
			Line:    consistencyWarning.Diagnostic.Line,
			Column:  consistencyWarning.Diagnostic.Column,
			Message: consistencyWarning.Diagnostic.Message,
		})
	}

	// Check for unresolved markdown links.
	if unresolvedLinks, unresolvedErr := findUnresolvedMarkdownLinks(p.Config); unresolvedErr == nil {
		for _, item := range unresolvedLinks {
			issues = append(issues, projectIssue{
				Rule:    "unresolved-markdown-link",
				File:    item.Source,
				Message: fmt.Sprintf("Markdown link target is outside the build graph: %s", item.Target),
			})
		}
	}

	return &ChapterPipelineResult{
		Chapters:        chaptersHTML,
		ChapterFiles:    chapterFiles,
		ChapterMarkdown: chapterMarkdown,
		Issues:          issues,
		AllHeadings:     allHeadings,
		Resolver:        resolver,
		HeadingRecords:  chapterHeadingRecords,
	}, nil
}

func defaultEmbeddedChapterImageOptions() utils.ImageProcessingOptions {
	return utils.ImageProcessingOptions{
		EmbedLocalAsBase64:     true,
		EmbedRemoteAsBase64:    true,
		DownloadRemote:         true,
		CacheDir:               filepath.Join(utils.CacheRootDir(), "images"),
		MaxConcurrentDownloads: 4,
	}
}

// pdfChapterImageOptions returns image options for PDF output.
// PDF HTML is served via a local HTTP server for font loading.
// Chrome blocks file:// URLs from HTTP pages, so images must be
// embedded as base64 data URIs instead of rewritten to file:// URLs.
// Currently identical to defaultEmbeddedChapterImageOptions.
var pdfChapterImageOptions = defaultEmbeddedChapterImageOptions

// firstHeadingPattern matches the first heading (<h1>–<h6>) anywhere in
// the content. Many README files start with non-heading HTML (e.g.
// <div align="center">, badge images) before the title heading, so we
// cannot require the heading to be at the very start.
var firstHeadingPattern = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)

// stripDuplicateLeadingH1 removes the first heading (h1–h6) from htmlContent
// if its text matches the summaryTitle. This prevents duplicate headings when
// the template already renders the SUMMARY.md title as <h1 class="chapter-title">.
// Sub-chapters often use h2 or lower in their Markdown, so matching only h1 is
// not sufficient.
//
// The heading is considered "leading" if only non-heading HTML elements appear
// before it (e.g. <div>, <p>, <img>, badge links). This handles the common
// pattern where README files wrap the title in a centered div or precede it
// with shield badges.
func stripDuplicateLeadingH1(htmlContent, summaryTitle string) string {
	if summaryTitle == "" {
		return htmlContent
	}

	m := firstHeadingPattern.FindStringSubmatchIndex(htmlContent)
	if m == nil {
		return htmlContent
	}

	// Verify that no other heading appears before this one.
	// The content before the match must not contain any <h1>–<h6> tags.
	prefix := htmlContent[:m[0]]
	if firstHeadingPattern.MatchString(prefix) {
		return htmlContent // another heading precedes this one; not leading
	}

	// Extract inner text of the heading, strip HTML tags for comparison.
	// m[4]:m[5] is the second capture group (heading content).
	innerHTML := htmlContent[m[4]:m[5]]
	innerText := strings.TrimSpace(stripHTMLTags(innerHTML))
	summaryText := strings.TrimSpace(summaryTitle)

	if innerText == summaryText {
		return strings.TrimSpace(htmlContent[:m[0]] + htmlContent[m[1]:])
	}
	return htmlContent
}

// stripHTMLTags removes HTML tags for plain-text comparison.
// Delegates to the shared pattern in pkg/utils to avoid duplication.
func stripHTMLTags(s string) string {
	return utils.StripHTMLTags(s)
}
