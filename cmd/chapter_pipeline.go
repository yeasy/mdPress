package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/crossref"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"github.com/yeasy/mdpress/internal/variables"
	"github.com/yeasy/mdpress/pkg/utils"
)

// ChapterPipelineResult encapsulates the output of chapter processing.
type ChapterPipelineResult struct {
	Chapters      []renderer.ChapterHTML
	ChapterFiles  []string
	Issues        []projectIssue
	AllHeadings   []toc.HeadingInfo
	Resolver      *crossref.Resolver
	HeadingRecords []chapterHeadingRecord
}

// ChapterPipeline orchestrates the complete chapter processing workflow.
type ChapterPipeline struct {
	Config   *config.BookConfig
	Theme    *theme.Theme
	Parser   *markdown.Parser
	Glossary *glossary.Glossary
	Logger   *slog.Logger
}

// NewChapterPipeline creates a new chapter pipeline with the given configuration.
func NewChapterPipeline(cfg *config.BookConfig, thm *theme.Theme, parser *markdown.Parser, gloss *glossary.Glossary, logger *slog.Logger) *ChapterPipeline {
	return &ChapterPipeline{
		Config:   cfg,
		Theme:    thm,
		Parser:   parser,
		Glossary: gloss,
		Logger:   logger,
	}
}

// Process executes the complete chapter processing pipeline.
// It returns processed chapters, chapter file paths, validation issues, and any error encountered.
// Always uses ParseWithDiagnostics regardless of caller preference.
func (p *ChapterPipeline) Process() (*ChapterPipelineResult, error) {
	resolver := crossref.NewResolver()

	var allHeadings []toc.HeadingInfo
	chaptersHTML := make([]renderer.ChapterHTML, 0, len(p.Config.Chapters))
	chapterFiles := make([]string, 0, len(p.Config.Chapters))
	issues := make([]projectIssue, 0)
	chapterHeadingRecords := make([]chapterHeadingRecord, 0, len(p.Config.Chapters))

	flatChapters := flattenChaptersWithDepth(p.Config.Chapters)
	for i, flatChapter := range flatChapters {
		chDef := flatChapter.Def
		chapterPath := p.Config.ResolvePath(chDef.File)
		p.Logger.Debug("Processing chapter", slog.Int("index", i+1), slog.String("file", chDef.File))

		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			p.Logger.Warn("Failed to read chapter, skipping", slog.String("file", chDef.File), slog.String("error", err.Error()))
			continue
		}

		content = variables.Expand(content, p.Config)

		htmlContent, headings, diagnostics, err := p.Parser.ParseWithDiagnostics(content)
		if err != nil {
			p.Logger.Warn("Failed to parse Markdown", slog.String("file", chDef.File), slog.String("error", err.Error()))
			continue
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
		htmlContent, err = utils.ProcessImages(htmlContent, chapterDir, true, p.Logger)
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

		// Process cross-references and glossary.
		htmlContent = resolver.ProcessHTML(htmlContent)
		htmlContent = resolver.AddCaptions(htmlContent)
		if p.Glossary != nil {
			htmlContent = p.Glossary.ProcessHTML(htmlContent)
		}

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
