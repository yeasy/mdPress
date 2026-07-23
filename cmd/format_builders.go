package cmd

import (
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
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

// buildContext carries all data needed by format builders.
type buildContext struct {
	Config             *config.BookConfig
	Theme              *theme.Theme
	SinglePageParts    *renderer.RenderParts
	PDFSinglePageParts *renderer.RenderParts
	ChaptersHTML       []renderer.ChapterHTML
	ChapterFiles       []string
	ChapterMarkdown    []string
	CustomCSS          string
	// SiteDir is the directory the "site" format writes into. When empty the
	// site builder falls back to "<baseName>_site" (used for multi-language
	// builds, where each language needs a distinct directory). Single-language
	// builds set this to "_book" (matching `mdpress serve`) or to an explicit
	// --output directory.
	SiteDir string
	Logger  *slog.Logger
}

// formatBuilder generates output in a specific format.
type formatBuilder interface {
	// Name returns the format name (e.g. "pdf", "html", "site", "epub").
	Name() string
	// Build generates the output file(s) at the given base path.
	Build(ctx *buildContext, baseName string) error
}

// formatBuilderRegistry manages registered format builders.
type formatBuilderRegistry struct {
	builders map[string]formatBuilder
}

// newFormatBuilderRegistry creates a registry pre-populated with all built-in formats.
func newFormatBuilderRegistry() *formatBuilderRegistry {
	r := &formatBuilderRegistry{
		builders: make(map[string]formatBuilder),
	}
	r.Register(&pdfBuilder{})
	r.Register(&typstBuilder{})
	r.Register(&htmlBuilder{})
	r.Register(&siteBuilder{})
	r.Register(&epubBuilder{})
	return r
}

// Register adds a format builder.
func (r *formatBuilderRegistry) Register(b formatBuilder) {
	r.builders[b.Name()] = b
}

// Get returns a builder by format name, or nil if not found.
func (r *formatBuilderRegistry) Get(name string) formatBuilder {
	return r.builders[name]
}

// ---- PDF ----

// pdfBuilder generates PDF output via Chromium.
type pdfBuilder struct{}

func (b *pdfBuilder) Name() string { return "pdf" }

func (b *pdfBuilder) Build(ctx *buildContext, baseName string) error {
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
	// Chrome's PrintToPDF does not trigger lazy loading for off-screen images,
	// so strip loading="lazy" to ensure all images render in the PDF.
	fullHTML = strings.ReplaceAll(fullHTML, " loading=\"lazy\"", "")
	outputPath := baseName + ".pdf"
	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	ctx.Logger.Info("Generating PDF", slog.String("output", outputPath))

	// Warn early if the content contains CJK characters but the system lacks CJK fonts.
	pdf.WarnIfCJKFontsMissing(fullHTML, ctx.Logger)

	pageDims := utils.GetPageDimensions(ctx.Config.Style.PageSize)
	pageWidth, pageHeight := pageDims.Width, pageDims.Height
	pdfTimeout := time.Duration(ctx.Config.Output.PDFTimeout) * time.Second
	if pdfTimeout <= 0 {
		pdfTimeout = defaultPDFTimeout
	}

	headerTmpl, footerTmpl := pdfHeaderFooterTemplates(ctx.Config)

	// Prepare margin options from config, with fallback defaults
	marginOpts := []pdf.GeneratorOption{
		pdf.WithTimeout(pdfTimeout),
		pdf.WithPageSize(pageWidth, pageHeight),
		pdf.WithPrintBackground(true),
		pdf.WithMetadata(pdfDocumentMetadata(ctx.Config)),
	}
	if headerTmpl != "" {
		marginOpts = append(marginOpts, pdf.WithHeaderTemplate(headerTmpl))
	}
	if footerTmpl != "" {
		marginOpts = append(marginOpts, pdf.WithFooterTemplate(footerTmpl))
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
		// Default margins: 0 on the sides, with space reserved at the top and
		// bottom only where a header or footer is actually rendered.
		const headerFooterMarginMM = 10
		topMargin, bottomMargin := 0.0, 0.0
		if headerTmpl != "" {
			topMargin = headerFooterMarginMM
		}
		if footerTmpl != "" {
			bottomMargin = headerFooterMarginMM
		}
		marginOpts = append(marginOpts, pdf.WithMargins(0, 0, topMargin, bottomMargin))
	}

	// Always pass the document outline option so that generate_bookmarks: false
	// actually disables bookmarks (the generator defaults to true).
	marginOpts = append(marginOpts, pdf.WithDocumentOutline(ctx.Config.Output.GenerateBookmarks))

	// tagged_pdf: false trades accessibility metadata for noticeably smaller
	// files; unset keeps the accessible default.
	taggedPDF := ctx.Config.Output.TaggedPDF == nil || *ctx.Config.Output.TaggedPDF
	marginOpts = append(marginOpts, pdf.WithTaggedPDF(taggedPDF))

	pdfGen := pdf.NewGenerator(marginOpts...)
	if err := pdfGen.Generate(fullHTML, outputPath); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "PDF"), slog.String("path", outputPath))
	return nil
}

// pdfDocumentMetadata maps book metadata onto the PDF /Info dictionary.
// Chrome fills that dictionary in with the headless browser's user agent and
// nothing else, so a PDF opened in a library or a document manager shows no
// author and claims to have been written by HeadlessChrome.
func pdfDocumentMetadata(cfg *config.BookConfig) pdf.DocumentMetadata {
	title := cfg.Book.Title
	if cfg.Book.Subtitle != "" && title != "" {
		title += ": " + cfg.Book.Subtitle
	}
	return pdf.DocumentMetadata{
		Title:   title,
		Author:  cfg.Book.Author,
		Subject: cfg.Book.Description,
		Creator: "mdPress " + Version,
	}
}

// defaultPDFFooterTemplate is the out-of-the-box PDF footer: a centered page
// number in subtle small print.
const defaultPDFFooterTemplate = `<div style='width:100%;text-align:center;font-size:9px;color:#9aa5b1;font-family:-apple-system,Arial,sans-serif;'><span class='pageNumber'></span></div>`

// pdfHeaderFooterTemplates derives the Chrome print header/footer templates
// from the book configuration. The output.header/output.footer booleans act
// as on/off switches; style.header/style.footer customize the content (with
// {page}/{title} token expansion). By default PDFs get a centered page-number
// footer and no header. An empty return value means "no header/footer".
func pdfHeaderFooterTemplates(cfg *config.BookConfig) (headerTmpl, footerTmpl string) {
	if cfg.Output.Footer {
		if isCustomHeaderFooterStyle(cfg.Style.Footer) {
			footerTmpl = renderPDFHeaderFooter(cfg.Style.Footer, cfg.Book)
		}
		if footerTmpl == "" {
			footerTmpl = defaultPDFFooterTemplate
		}
	}
	if cfg.Output.Header {
		// Default: no header. Only a configured style.header renders one.
		if isCustomHeaderFooterStyle(cfg.Style.Header) {
			headerTmpl = renderPDFHeaderFooter(cfg.Style.Header, cfg.Book)
		}
	}
	return headerTmpl, footerTmpl
}

// isCustomHeaderFooterStyle reports whether the user configured a
// header/footer style.
//
// This used to also require the value to differ from the built-in default —
// and the manual's example was that default, so a reader who copied it got no
// header and nothing to explain why.
func isCustomHeaderFooterStyle(s config.HeaderFooterStyle) bool {
	return s != (config.HeaderFooterStyle{})
}

// renderPDFHeaderFooter builds a Chrome print template with left/center/right
// cells from a configured header/footer style. Returns "" when every cell
// expands to nothing.
func renderPDFHeaderFooter(parts config.HeaderFooterStyle, book config.BookMeta) string {
	left := expandPDFTemplateTokens(parts.Left, book)
	center := expandPDFTemplateTokens(parts.Center, book)
	right := expandPDFTemplateTokens(parts.Right, book)
	if left == "" && center == "" && right == "" {
		return ""
	}
	return "<div style='width:100%;font-size:9px;color:#9aa5b1;font-family:-apple-system,Arial,sans-serif;display:flex;justify-content:space-between;padding:0 10mm;'>" +
		"<span style='flex:1;text-align:left;'>" + left + "</span>" +
		"<span style='flex:1;text-align:center;'>" + center + "</span>" +
		"<span style='flex:1;text-align:right;'>" + right + "</span>" +
		"</div>"
}

// expandPDFTemplateTokens HTML-escapes user text and expands the supported
// placeholder tokens into Chrome print-template markup. Both the documented
// {page}/{pages}/{title} tokens and the legacy Go-template-style tokens used
// by scaffolded configs ({{.PageNum}}, {{.Book.Title}}, ...) are supported.
// {{.Chapter.Title}} has no Chrome equivalent and expands to nothing.
func expandPDFTemplateTokens(text string, book config.BookMeta) string {
	if text == "" {
		return ""
	}
	const pageSpan = "<span class='pageNumber'></span>"
	const totalPagesSpan = "<span class='totalPages'></span>"
	escapedTitle := html.EscapeString(book.Title)
	escapedAuthor := html.EscapeString(book.Author)
	replacer := strings.NewReplacer(
		"{page}", pageSpan,
		"{pages}", totalPagesSpan,
		"{title}", escapedTitle,
		"{author}", escapedAuthor,
		"{{.PageNum}}", pageSpan,
		"{{.TotalPages}}", totalPagesSpan,
		"{{.Book.Title}}", escapedTitle,
		"{{.Book.Author}}", escapedAuthor,
		"{{.Chapter.Title}}", "",
	)
	expanded := replacer.Replace(html.EscapeString(text))
	// A token with no Chrome equivalent must not be printed onto the paper.
	// Chrome escapes nothing here, so an unexpanded "{{.Whatever}}" would
	// appear verbatim in the running head of every page.
	return stripUnexpandedTokens(expanded)
}

// unexpandedTokenPattern matches a leftover Go-template-style token.
var unexpandedTokenPattern = regexp.MustCompile(`\{\{[^}]*\}\}`)

// stripUnexpandedTokens removes tokens mdpress does not understand, warning
// once per token so the author learns which one was dropped.
func stripUnexpandedTokens(s string) string {
	return unexpandedTokenPattern.ReplaceAllStringFunc(s, func(token string) string {
		slog.Warn("unsupported header/footer token removed",
			slog.String("token", token),
			slog.String("supported", "{page} {pages} {title} {author}"))
		return ""
	})
}

// ---- HTML ----

// htmlBuilder generates a self-contained single-page HTML document.
type htmlBuilder struct{}

func (b *htmlBuilder) Name() string { return "html" }

func (b *htmlBuilder) Build(ctx *buildContext, baseName string) error {
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
	if err := os.WriteFile(outputPath, []byte(standaloneHTML), 0o644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}
	ctx.Logger.Info("Output ready", slog.String("format", "HTML"), slog.String("path", outputPath))
	return nil
}

// ---- Site ----

// siteBuilder generates a multi-page HTML site.
type siteBuilder struct{}

func (b *siteBuilder) Name() string { return "site" }

func (b *siteBuilder) Build(ctx *buildContext, baseName string) error {
	outputDir := ctx.SiteDir
	if outputDir == "" {
		outputDir = baseName + "_site"
	}
	ctx.Logger.Info("Generating HTML site", slog.String("output", outputDir))

	// Migration hint: the default site directory moved from "<name>_site" to
	// "_book" in v0.7.13. Surface the old location so stale deploys pointing
	// at it do not go unnoticed.
	if filepath.Base(outputDir) == "_book" {
		if info, err := os.Stat(baseName + "_site"); err == nil && info.IsDir() {
			ctx.Logger.Info("site output moved to _book/ in v0.7.13; the legacy directory is no longer updated and can be removed",
				slog.String("legacy", baseName+"_site"),
				slog.String("current", outputDir))
		}
	}

	pageNames := sitePageFilenames(ctx.ChapterFiles)
	siteChapters := rewriteChapterLinksForSite(ctx.ChaptersHTML, ctx.ChapterFiles, pageNames)
	generate := func(dir string) error {
		return generateSiteOutput(ctx.Config, ctx.Theme, ctx.CustomCSS, dir, siteChapters, ctx.ChapterFiles, pageNames, ctx.ChapterMarkdown)
	}

	// When the site shares its directory with the other output files (an
	// explicit "--output <dir>"), other format builders may be writing into
	// the same directory concurrently, so generate in place instead of
	// swapping the whole directory out.
	if filepath.Clean(outputDir) == filepath.Dir(baseName) {
		if err := generate(outputDir); err != nil {
			return fmt.Errorf("failed to generate HTML site: %w", err)
		}
		ctx.Logger.Info("Output ready", slog.String("format", "site"), slog.String("path", outputDir))
		return nil
	}

	if err := ensureReplaceableSiteDir(outputDir); err != nil {
		return err
	}

	// Build into a temp dir next to the target, then atomically swap it in
	// (mirroring `mdpress serve`), so the target never holds a half-written
	// site and stale pages from previous builds are removed.
	if err := utils.EnsureDir(filepath.Dir(outputDir)); err != nil {
		return fmt.Errorf("failed to create site output parent directory: %w", err)
	}
	tempDir, err := newSiteStagingDir(filepath.Dir(outputDir), "mdpress-site-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary site directory: %w", err)
	}
	if err := generate(tempDir); err != nil {
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			ctx.Logger.Debug("failed to remove temporary site directory", slog.String("dir", tempDir), slog.Any("error", rmErr))
		}
		return fmt.Errorf("failed to generate HTML site: %w", err)
	}
	if err := swapSiteDir(tempDir, outputDir, ctx.Logger); err != nil {
		// Rename can fail across devices; fall back to building in place.
		ctx.Logger.Debug("atomic site swap failed, rebuilding in place", slog.Any("error", err))
		if genErr := generate(outputDir); genErr != nil {
			return fmt.Errorf("failed to generate HTML site: %w", genErr)
		}
	}
	ctx.Logger.Info("Output ready", slog.String("format", "site"), slog.String("path", outputDir))
	return nil
}

// ensureReplaceableSiteDir refuses to replace a non-empty directory that does
// not look like a previously generated site, so the atomic swap can never
// wipe user data. A generated site always contains index.html and
// search-index.json.
func ensureReplaceableSiteDir(dir string) error {
	info, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat site output directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("site output path %s already exists as a file; remove it or choose another --output", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read site output directory: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if utils.FileExists(filepath.Join(dir, "index.html")) || utils.FileExists(filepath.Join(dir, "search-index.json")) {
		return nil
	}
	return fmt.Errorf("refusing to replace %s: the directory is not empty and does not look like a generated site (no index.html or search-index.json); remove it or choose another --output", dir)
}

// newSiteStagingDir creates the staging directory that the atomic swap renames
// into place as the published site root.
//
// os.MkdirTemp always creates 0700 directories. That is right for a scratch
// directory but wrong here: after the rename it *is* the site root, and a
// 0700 site root is unreadable to the web server user (nginx/httpd → 403) and
// is preserved by rsync -a, docker COPY and CI artifact upload. Everything
// generated inside it is already world-readable, so make the root match.
func newSiteStagingDir(parent, pattern string) (string, error) {
	dir, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o755); err != nil { //nolint:gosec // G302: a published site root must be world-readable
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			return "", fmt.Errorf("chmod staging dir: %w (cleanup also failed: %v)", err, rmErr)
		}
		return "", fmt.Errorf("chmod staging dir: %w", err)
	}
	return dir, nil
}

// swapSiteDir atomically replaces outputDir with tempDir: the previous output
// is renamed aside, the fresh build renamed in, and the backup removed. On
// failure the previous output is restored and an error is returned.
func swapSiteDir(tempDir, outputDir string, logger *slog.Logger) error {
	backupDir := outputDir + ".old"
	if err := os.RemoveAll(backupDir); err != nil {
		logger.Debug("failed to remove previous backup directory", slog.String("dir", backupDir), slog.Any("error", err))
	}
	if err := os.Rename(outputDir, backupDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		logger.Debug("failed to move previous site output aside", slog.Any("error", err))
	}
	if err := os.Rename(tempDir, outputDir); err != nil {
		if restoreErr := os.Rename(backupDir, outputDir); restoreErr != nil && !errors.Is(restoreErr, fs.ErrNotExist) {
			logger.Debug("failed to restore previous site output", slog.Any("error", restoreErr))
		}
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			logger.Debug("failed to remove temporary site directory", slog.String("dir", tempDir), slog.Any("error", rmErr))
		}
		return fmt.Errorf("swap site directory: %w", err)
	}
	if err := os.RemoveAll(backupDir); err != nil {
		logger.Debug("failed to remove backup directory after swap", slog.String("dir", backupDir), slog.Any("error", err))
	}
	return nil
}

// ---- ePub ----

// epubBuilder generates an EPUB 3 ebook.
type epubBuilder struct{}

func (b *epubBuilder) Name() string { return "epub" }

func (b *epubBuilder) Build(ctx *buildContext, baseName string) error {
	outputPath := baseName + ".epub"
	ctx.Logger.Info("Generating ePub", slog.String("output", outputPath))

	coverImagePath := ""
	if ctx.Config.Book.Cover.Image != "" {
		coverImagePath = ctx.Config.ResolvePath(ctx.Config.Book.Cover.Image)
	}
	epubGen := output.NewEpubGenerator(output.EpubMeta{
		Title:           ctx.Config.Book.Title,
		Subtitle:        ctx.Config.Book.Subtitle,
		Author:          ctx.Config.Book.Author,
		Language:        ctx.Config.Book.Language,
		Version:         ctx.Config.Book.Version,
		Description:     ctx.Config.Book.Description,
		IncludeCover:    ctx.Config.Output.Cover,
		CoverImagePath:  coverImagePath,
		CoverBackground: ctx.Config.Book.Cover.Background,
	})
	// Use the book root as the image-containment base so chapters can reference
	// shared assets above their own directory (e.g. ../images/pic.png).
	epubGen.SetBookRoot(ctx.Config.BaseDir())
	// EPUB styling contract: SetCSS carries only the user's custom CSS; the
	// generator derives its own reader-friendly stylesheet from the theme.
	epubGen.SetCSS(ctx.CustomCSS)
	epubGen.SetTheme(ctx.Theme)
	// Cross-chapter .md links must point at the packaged .xhtml documents;
	// the raw chapter HTML still carries the Markdown hrefs.
	epubChapters := rewriteChapterLinksForEpub(ctx.ChaptersHTML, ctx.ChapterFiles)
	for i, ch := range epubChapters {
		sourceDir := ""
		if i < len(ctx.ChapterFiles) && ctx.ChapterFiles[i] != "" {
			sourceDir = filepath.Dir(ctx.Config.ResolvePath(ctx.ChapterFiles[i]))
		}
		epubGen.AddChapter(output.EpubChapter{
			Depth:     ch.Depth,
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

// typstBuilder generates PDF output via Typst.
type typstBuilder struct{}

func (b *typstBuilder) Name() string { return "typst" }

func (b *typstBuilder) Build(ctx *buildContext, baseName string) error {
	// For Typst, we work directly with Markdown content rather than HTML.

	outputPath := baseName + "-typst.pdf"
	ctx.Logger.Info("Generating PDF via Typst", slog.String("output", outputPath))

	// Determine the Typst project root: the book source directory, made
	// absolute. Image paths are rewritten to be root-relative against this
	// directory so that "typst compile --root <root>" can resolve them.
	typstRoot, err := filepath.Abs(ctx.Config.BaseDir())
	if err != nil {
		return fmt.Errorf("failed to resolve book root directory: %w", err)
	}

	// Extract text from chapters
	var markdownContent strings.Builder

	// Add title
	if ctx.Config.Book.Title != "" {
		markdownContent.WriteString("# ")
		markdownContent.WriteString(ctx.Config.Book.Title)
		markdownContent.WriteString("\n\n")
	}

	// Add chapters using original Markdown content.
	// Skip injecting the chapter title heading when the raw Markdown
	// already starts with a level-1 or level-2 heading, to avoid
	// duplicate headings in the output.  Lower-level headings (###, etc.)
	// do not conflict with the injected ## title and are kept as-is.
	for i, ch := range ctx.ChaptersHTML {
		md := ""
		if i < len(ctx.ChapterMarkdown) {
			md = ctx.ChapterMarkdown[i]
		}
		// Rewrite chapter-relative image paths to root-relative paths that
		// resolve under typstRoot, skipping images that do not exist.
		chapterDir := ""
		if i < len(ctx.ChapterFiles) && ctx.ChapterFiles[i] != "" {
			chapterDir = filepath.Dir(ctx.Config.ResolvePath(ctx.ChapterFiles[i]))
		}
		md = rewriteTypstImagePaths(md, chapterDir, typstRoot, ctx.Logger)

		mdTrimmed := strings.TrimSpace(md)
		startsWithH1orH2 := (strings.HasPrefix(mdTrimmed, "# ") || strings.HasPrefix(mdTrimmed, "#\t") ||
			strings.HasPrefix(mdTrimmed, "## ") || strings.HasPrefix(mdTrimmed, "##\t"))
		if ch.Title != "" && !startsWithH1orH2 {
			markdownContent.WriteString("## ")
			markdownContent.WriteString(ch.Title)
			markdownContent.WriteString("\n\n")
		}
		markdownContent.WriteString(md)
		markdownContent.WriteString("\n\n")
	}

	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	typstTimeout := time.Duration(ctx.Config.Output.PDFTimeout) * time.Second
	if typstTimeout <= 0 {
		typstTimeout = defaultTypstTimeout
	}

	typstGen := typst.NewGenerator(
		typst.WithTimeout(typstTimeout),
		typst.WithPageSize(ctx.Config.Style.PageSize),
		typst.WithTitle(ctx.Config.Book.Title),
		typst.WithAuthor(ctx.Config.Book.Author),
		typst.WithDescription(ctx.Config.Book.Description),
		typst.WithVersion(ctx.Config.Book.Version),
		typst.WithLanguage(ctx.Config.Book.Language),
		// Read the resolved theme, not the raw config: the theme already
		// carries the built-in values with the user's style overlaid, so all
		// backends render the same typography.
		typst.WithFontFamily(ctx.Theme.FontFamily),
		typst.WithFontSize(ctx.Theme.ResolvedFontSize()),
		typst.WithLineHeight(ctx.Theme.LineHeight),
		typst.WithRootDir(typstRoot),
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

// typstImagePattern matches Markdown image syntax ![alt](path).
var typstImagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// rewriteTypstImagePaths rewrites chapter-relative Markdown image paths so they
// resolve when the document is compiled with "typst compile --root <root>".
//
// For each local image reference it:
//   - resolves the path relative to chapterDir,
//   - verifies the file exists (missing images are replaced with their alt text
//     so a single broken image does not abort the whole PDF build), and
//   - rewrites the path to be root-relative ("/sub/dir/img.png") against root,
//     which Typst resolves against the --root directory.
//
// Remote references (http://, https://, data:) and images that already resolve
// outside the root are left unchanged.
func rewriteTypstImagePaths(md, chapterDir, root string, logger *slog.Logger) string {
	if md == "" {
		return md
	}
	return typstImagePattern.ReplaceAllStringFunc(md, func(match string) string {
		m := typstImagePattern.FindStringSubmatch(match)
		if m == nil {
			return match
		}
		alt, src := m[1], strings.TrimSpace(m[2])

		// Leave remote/data URIs untouched.
		lower := strings.ToLower(src)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
			strings.HasPrefix(lower, "data:") {
			return match
		}

		// Resolve the image to an absolute path.
		absImg := src
		if !filepath.IsAbs(absImg) {
			base := chapterDir
			if base == "" {
				base = root
			}
			absImg = filepath.Join(base, src)
		}

		// Warn and drop images that do not exist, replacing them with their
		// alt text so the build does not abort.
		if info, err := os.Stat(absImg); err != nil || info.IsDir() {
			if logger != nil {
				logger.Warn("Typst: skipping missing image",
					slog.String("path", src), slog.String("resolved", absImg))
			}
			return imageFallbackText(alt)
		}

		// Compute a path relative to the Typst root. If the image lives
		// outside the root, Typst cannot reference it, so drop it too.
		rel, err := filepath.Rel(root, absImg)
		if err != nil || strings.HasPrefix(rel, "..") {
			if logger != nil {
				logger.Warn("Typst: image outside project root, skipping",
					slog.String("path", src), slog.String("resolved", absImg))
			}
			return imageFallbackText(alt)
		}

		// Root-relative path with forward slashes (Typst path separator).
		rootRel := "/" + filepath.ToSlash(rel)
		return "![" + alt + "](" + rootRel + ")"
	})
}

// imageFallbackText returns replacement text for an image that cannot be
// rendered: its alt text if present, otherwise an empty string.
func imageFallbackText(alt string) string {
	return strings.TrimSpace(alt)
}
