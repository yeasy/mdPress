package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/output"
	"github.com/yeasy/mdpress/internal/pdf"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/typst"
	"github.com/yeasy/mdpress/pkg/utils"
)

const (
	// Default PDF generation timeout when not configured.
	defaultPDFTimeout = 2 * time.Minute
	// Default Typst PDF generation timeout when not configured.
	defaultTypstTimeout = 2 * time.Minute
)

// BuildContext carries all data needed by format builders.
type BuildContext struct {
	Config             *config.BookConfig
	Theme              *theme.Theme
	SinglePageParts    *renderer.RenderParts
	PDFSinglePageParts *renderer.RenderParts
	ChaptersHTML       []renderer.ChapterHTML
	ChapterFiles       []string
	CustomCSS          string
	Logger             *slog.Logger
}

// FormatBuilder generates output in a specific format.
type FormatBuilder interface {
	// Name returns the format name (e.g. "pdf", "html", "site", "epub").
	Name() string
	// Build generates the output file(s) at the given base path.
	Build(ctx *BuildContext, baseName string) error
}

// FormatBuilderRegistry manages registered format builders.
type FormatBuilderRegistry struct {
	builders map[string]FormatBuilder
}

// NewFormatBuilderRegistry creates a registry pre-populated with all built-in formats.
func NewFormatBuilderRegistry() *FormatBuilderRegistry {
	r := &FormatBuilderRegistry{
		builders: make(map[string]FormatBuilder),
	}
	r.Register(&PDFBuilder{})
	r.Register(&TypstBuilder{})
	r.Register(&HTMLBuilder{})
	r.Register(&SiteBuilder{})
	r.Register(&EpubBuilder{})
	return r
}

// Register adds a format builder.
func (r *FormatBuilderRegistry) Register(b FormatBuilder) {
	r.builders[b.Name()] = b
}

// Get returns a builder by format name, or nil if not found.
func (r *FormatBuilderRegistry) Get(name string) FormatBuilder {
	return r.builders[name]
}

// ---- PDF ----

// PDFBuilder generates PDF output via Chromium.
type PDFBuilder struct{}

func (b *PDFBuilder) Name() string { return "pdf" }

func (b *PDFBuilder) Build(ctx *BuildContext, baseName string) error {
	htmlRenderer, err := renderer.NewHTMLRenderer(ctx.Config, ctx.Theme)
	if err != nil {
		return fmt.Errorf("failed to create HTML renderer: %w", err)
	}
	parts := ctx.SinglePageParts
	if ctx.PDFSinglePageParts != nil {
		parts = ctx.PDFSinglePageParts
	}
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		return fmt.Errorf("failed to assemble HTML: %w", err)
	}
	outputPath := baseName + ".pdf"
	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	ctx.Logger.Info("Generating PDF", slog.String("output", outputPath))

	// Warn early if the content contains CJK characters but the system lacks CJK fonts.
	pdf.WarnIfCJKFontsMissing(fullHTML, ctx.Logger)

	pageWidth, pageHeight := getPageDimensions(ctx.Config.Style.PageSize)
	pdfTimeout := time.Duration(ctx.Config.Output.PDFTimeout) * time.Second
	if pdfTimeout <= 0 {
		pdfTimeout = defaultPDFTimeout
	}

	// Prepare margin options from config, with fallback defaults
	marginOpts := []pdf.GeneratorOption{
		pdf.WithTimeout(pdfTimeout),
		pdf.WithPageSize(pageWidth, pageHeight),
		pdf.WithPrintBackground(true),
		pdf.WithFooterTemplate(`<div style="width:100%;text-align:center;font-size:8px;color:#c0c0c0;font-family:Arial,sans-serif;">Build with <a href="https://github.com/yeasy/mdpress" style="color:#8ab4f8;text-decoration:none;">md<span style="color:#8ab4f8;">Press</span></a></div>`),
	}

	// Add custom margins if provided in config, otherwise use defaults
	if ctx.Config.Output.MarginLeft != "" || ctx.Config.Output.MarginRight != "" ||
		ctx.Config.Output.MarginTop != "" || ctx.Config.Output.MarginBottom != "" {
		marginOpts = append(marginOpts, pdf.WithMarginStrings(
			ctx.Config.Output.MarginLeft,
			ctx.Config.Output.MarginRight,
			ctx.Config.Output.MarginTop,
			ctx.Config.Output.MarginBottom,
		))
	} else {
		// Default margins: 0 on sides, 10mm on bottom for footer
		marginOpts = append(marginOpts, pdf.WithMargins(0, 0, 0, 10))
	}

	// Add document outline option if enabled
	if ctx.Config.Output.GenerateBookmarks {
		marginOpts = append(marginOpts, pdf.WithDocumentOutline(true))
	}

	pdfGen := pdf.NewGenerator(marginOpts...)
	if err := pdfGen.Generate(fullHTML, outputPath); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "PDF"), slog.String("path", outputPath))
	return nil
}

// ---- HTML ----

// HTMLBuilder generates a self-contained single-page HTML document.
type HTMLBuilder struct{}

func (b *HTMLBuilder) Name() string { return "html" }

func (b *HTMLBuilder) Build(ctx *BuildContext, baseName string) error {
	outputPath := baseName + ".html"
	ctx.Logger.Info("Generating standalone HTML", slog.String("output", outputPath))

	standaloneRenderer, err := renderer.NewStandaloneHTMLRenderer(ctx.Config, ctx.Theme)
	if err != nil {
		return fmt.Errorf("failed to create standalone HTML renderer: %w", err)
	}
	standaloneHTML, err := standaloneRenderer.Render(ctx.SinglePageParts)
	if err != nil {
		return fmt.Errorf("failed to generate standalone HTML: %w", err)
	}
	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(standaloneHTML), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "HTML"), slog.String("path", outputPath))
	return nil
}

// ---- Site ----

// SiteBuilder generates a multi-page HTML site.
type SiteBuilder struct{}

func (b *SiteBuilder) Name() string { return "site" }

func (b *SiteBuilder) Build(ctx *BuildContext, baseName string) error {
	outputDir := baseName + "_site"
	ctx.Logger.Info("Generating HTML site", slog.String("output", outputDir))

	pageNames := sitePageFilenames(ctx.ChapterFiles)
	siteChapters := rewriteChapterLinksForSite(ctx.ChaptersHTML, ctx.ChapterFiles, pageNames)
	if err := generateSiteOutput(ctx.Config, ctx.Theme, ctx.CustomCSS, outputDir, siteChapters, pageNames); err != nil {
		return fmt.Errorf("failed to generate HTML site: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "site"), slog.String("path", outputDir))
	return nil
}

// ---- ePub ----

// EpubBuilder generates an EPUB 3 ebook.
type EpubBuilder struct{}

func (b *EpubBuilder) Name() string { return "epub" }

func (b *EpubBuilder) Build(ctx *BuildContext, baseName string) error {
	outputPath := baseName + ".epub"
	ctx.Logger.Info("Generating ePub", slog.String("output", outputPath))

	coverImagePath := ""
	if ctx.Config.Book.Cover.Image != "" {
		coverImagePath = ctx.Config.ResolvePath(ctx.Config.Book.Cover.Image)
	}
	epubGen := output.NewEpubGenerator(output.EpubMeta{
		Title:          ctx.Config.Book.Title,
		Subtitle:       ctx.Config.Book.Subtitle,
		Author:         ctx.Config.Book.Author,
		Language:       ctx.Config.Book.Language,
		Version:        ctx.Config.Book.Version,
		Description:    ctx.Config.Book.Description,
		IncludeCover:   ctx.Config.Output.Cover,
		CoverImagePath: coverImagePath,
	})
	epubGen.SetCSS(ctx.Theme.ToCSS() + "\n" + ctx.CustomCSS)
	for i, ch := range ctx.ChaptersHTML {
		sourceDir := ""
		if i < len(ctx.ChapterFiles) && ctx.ChapterFiles[i] != "" {
			sourceDir = filepath.Dir(ctx.Config.ResolvePath(ctx.ChapterFiles[i]))
		}
		epubGen.AddChapter(output.EpubChapter{
			Title:     ch.Title,
			ID:        ch.ID,
			Filename:  ch.ID + ".xhtml",
			HTML:      ch.Content,
			SourceDir: sourceDir,
		})
	}
	if err := epubGen.Generate(outputPath); err != nil {
		return fmt.Errorf("failed to generate ePub: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "ePub"), slog.String("path", outputPath))
	return nil
}

// ---- Typst PDF ----

// TypstBuilder generates PDF output via Typst (proof-of-concept).
type TypstBuilder struct{}

func (b *TypstBuilder) Name() string { return "typst" }

func (b *TypstBuilder) Build(ctx *BuildContext, baseName string) error {
	// For Typst, we work directly with Markdown content rather than HTML.
	// This is a proof-of-concept that demonstrates the core use case.

	outputPath := baseName + ".pdf"
	ctx.Logger.Info("Generating PDF via Typst", slog.String("output", outputPath))

	// Extract text from chapters for now (proof of concept)
	var markdownContent strings.Builder

	// Add title
	if ctx.Config.Book.Title != "" {
		markdownContent.WriteString("# ")
		markdownContent.WriteString(ctx.Config.Book.Title)
		markdownContent.WriteString("\n\n")
	}

	// Add chapters
	for _, ch := range ctx.ChaptersHTML {
		if ch.Title != "" {
			markdownContent.WriteString("## ")
			markdownContent.WriteString(ch.Title)
			markdownContent.WriteString("\n\n")
		}
		// For PoC, use the HTML content as-is. In production, convert HTML to Markdown.
		markdownContent.WriteString(ch.Content)
		markdownContent.WriteString("\n\n")
	}

	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	_, _ = getPageDimensions(ctx.Config.Style.PageSize) // For future use
	typstTimeout := time.Duration(ctx.Config.Output.PDFTimeout) * time.Second
	if typstTimeout <= 0 {
		typstTimeout = defaultTypstTimeout
	}

	typstGen := typst.NewGenerator(
		typst.WithTimeout(typstTimeout),
		typst.WithPageSize(ctx.Config.Style.PageSize),
		typst.WithTitle(ctx.Config.Book.Title),
		typst.WithAuthor(ctx.Config.Book.Author),
		typst.WithVersion(ctx.Config.Book.Version),
		typst.WithLanguage(ctx.Config.Book.Language),
		typst.WithFontFamily(ctx.Config.Style.FontFamily),
		typst.WithFontSize(ctx.Config.Style.FontSize),
		typst.WithLineHeight(ctx.Config.Style.LineHeight),
		typst.WithMargins(
			typst.ConvertMarginToTypst(ctx.Config.Output.MarginLeft, "20mm"),
			typst.ConvertMarginToTypst(ctx.Config.Output.MarginRight, "20mm"),
			typst.ConvertMarginToTypst(ctx.Config.Output.MarginTop, "20mm"),
			typst.ConvertMarginToTypst(ctx.Config.Output.MarginBottom, "20mm"),
		),
	)

	if err := typstGen.Generate(markdownContent.String(), outputPath); err != nil {
		return fmt.Errorf("failed to generate PDF via Typst: %w", err)
	}

	ctx.Logger.Info("Output ready", slog.String("format", "Typst PDF"), slog.String("path", outputPath))
	return nil
}
