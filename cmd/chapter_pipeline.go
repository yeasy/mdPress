package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

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
}

// ChapterPipelineResult encapsulates the output of chapter processing.
type ChapterPipelineResult struct {
	Chapters       []renderer.ChapterHTML
	ChapterFiles   []string
	Issues         []projectIssue
	AllHeadings    []toc.HeadingInfo
	Resolver       *crossref.Resolver
	HeadingRecords []chapterHeadingRecord
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
	issues := make([]projectIssue, 0)
	chapterHeadingRecords := make([]chapterHeadingRecord, 0, len(p.Config.Chapters))

	flatChapters := flattenChaptersWithDepth(p.Config.Chapters)
	for i, flatChapter := range flatChapters {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		chDef := flatChapter.Def
		chapterPath := p.Config.ResolvePath(chDef.File)
		p.Logger.Debug("Processing chapter", slog.Int("index", i+1), slog.String("file", chDef.File))

		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			p.Logger.Warn("Failed to read chapter, skipping", slog.String("file", chDef.File), slog.String("error", err.Error()))
			continue
		}

		content = variables.Expand(content, p.Config)
		expandedContent := string(content)

		codeTheme := p.Config.Style.CodeTheme
		if codeTheme == "" && p.Theme != nil {
			codeTheme = p.Theme.CodeTheme
		}
		cached, cacheHit, cacheErr := loadParsedChapterCache(chapterPath, expandedContent, codeTheme)
		var htmlContent string
		var headings []markdown.HeadingInfo
		var diagnostics []markdown.Diagnostic
		switch {
		case cacheErr != nil:
			p.Logger.Debug("Parsed chapter cache read failed", slog.String("file", chDef.File), slog.String("error", cacheErr.Error()))
		case cacheHit:
			htmlContent = cached.HTML
			headings = cached.Headings
			diagnostics = cached.Diagnostics
		default:
			var parseErr error
			htmlContent, headings, diagnostics, parseErr = p.Parser.ParseWithDiagnostics(content)
			if parseErr != nil {
				p.Logger.Warn("Failed to parse Markdown", slog.String("file", chDef.File), slog.String("error", parseErr.Error()))
				continue
			}
			if storeErr := storeParsedChapterCache(chapterPath, expandedContent, codeTheme, &cachedParsedChapter{
				HTML:        htmlContent,
				Headings:    headings,
				Diagnostics: diagnostics,
			}); storeErr != nil {
				p.Logger.Debug("Parsed chapter cache write failed", slog.String("file", chDef.File), slog.String("error", storeErr.Error()))
			}
		}

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

		// Process images in the chapter.
		chapterDir := filepath.Dir(chapterPath)
		imageOptions.Logger = p.Logger
		htmlContent, err = utils.ProcessImagesWithOptions(htmlContent, chapterDir, imageOptions)
		if err != nil {
			p.Logger.Warn("Failed to process images", slog.String("file", chDef.File), slog.String("error", err.Error()))
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
			Metadata:     make(map[string]interface{}),
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
	}

	// Validate that at least some chapters were processed.
	if len(chaptersHTML) == 0 {
		return nil, fmt.Errorf("no chapters were processed successfully (check chapter paths in book.yaml and run mdpress validate)")
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
		Chapters:       chaptersHTML,
		ChapterFiles:   chapterFiles,
		Issues:         issues,
		AllHeadings:    allHeadings,
		Resolver:       resolver,
		HeadingRecords: chapterHeadingRecords,
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

func pdfChapterImageOptions() utils.ImageProcessingOptions {
	return utils.ImageProcessingOptions{
		RewriteLocalToFileURL:  true,
		RewriteRemoteToFileURL: true,
		DownloadRemote:         true,
		CacheDir:               filepath.Join(utils.CacheRootDir(), "images"),
		MaxConcurrentDownloads: 4,
	}
}

// leadingH1Pattern matches the first <h1...>...</h1> at the start of content
// (allowing only whitespace before it).
var leadingH1Pattern = regexp.MustCompile(`(?is)^\s*<h1[^>]*>(.*?)</h1>`)

// stripDuplicateLeadingH1 removes the first <h1> from htmlContent if its
// text matches the summaryTitle. This prevents duplicate headings when the
// template already renders the SUMMARY.md title as <h1 class="chapter-title">.
func stripDuplicateLeadingH1(htmlContent, summaryTitle string) string {
	if summaryTitle == "" {
		return htmlContent
	}

	m := leadingH1Pattern.FindStringSubmatchIndex(htmlContent)
	if m == nil {
		return htmlContent
	}

	// Extract inner text of the <h1>, strip HTML tags for comparison.
	innerHTML := htmlContent[m[2]:m[3]]
	innerText := strings.TrimSpace(stripHTMLTags(innerHTML))
	summaryText := strings.TrimSpace(summaryTitle)

	if innerText == summaryText {
		return strings.TrimSpace(htmlContent[:m[0]] + htmlContent[m[1]:])
	}
	return htmlContent
}

// htmlTagPattern strips HTML tags for plain-text comparison.
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(s string) string {
	return htmlTagPattern.ReplaceAllString(s, "")
}
