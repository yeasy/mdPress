// epub.go generates EPUB 3 ebooks.
// The resulting .epub file is a ZIP archive containing XHTML, CSS, OPF metadata,
// and both EPUB 3 navigation and NCX files for wider reader compatibility.
package output

import (
	"archive/zip"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// epubKaTeXWarningOnce ensures the CDN dependency warning is logged only once.
var epubKaTeXWarningOnce sync.Once

// EpubMeta contains EPUB metadata.
type EpubMeta struct {
	Title          string
	Subtitle       string
	Author         string
	Language       string
	Version        string
	Description    string
	IncludeCover   bool
	CoverImagePath string
	// CoverBackground is the configured cover background color (may be empty).
	CoverBackground string
}

// EpubChapter stores one EPUB chapter.
type EpubChapter struct {
	Title     string
	ID        string
	Filename  string
	HTML      string // XHTML body content.
	SourceDir string // Source directory used to resolve relative asset paths.
	// Depth is the chapter's nesting level (0 = top level). Reading systems
	// render the navigation document as a tree, so a book using `sections:`
	// needs this to keep its hierarchy instead of showing one flat list.
	Depth int
}

// EpubGenerator builds an EPUB file.
type EpubGenerator struct {
	meta     EpubMeta
	chapters []EpubChapter
	css      string
	// bookRoot is the containment base used when resolving relative image
	// paths. Images resolving inside this directory are packaged, even when
	// they live above an individual chapter's directory (e.g. a shared
	// ../images referenced from chapters in docs/). When empty, the common
	// ancestor of all chapter source directories is used instead.
	bookRoot string
	// thm is the active document theme; when set the generator can derive
	// EPUB-appropriate CSS from it (reader-friendly, literal values).
	thm *theme.Theme
}

type epubAsset struct {
	ID        string
	Filename  string
	MediaType string
	Data      []byte
}

type epubAssetCollector struct {
	nextIndex int
	cache     map[string]*epubAsset
}

// NewEpubGenerator creates an ePub generator.
func NewEpubGenerator(meta EpubMeta) *EpubGenerator {
	return &EpubGenerator{
		meta: meta,
	}
}

// SetCSS sets the user's custom CSS. It is appended after the generator's own
// theme-derived stylesheet when writing OEBPS/style.css so custom rules win.
func (g *EpubGenerator) SetCSS(css string) {
	g.css = css
}

// SetTheme sets the active document theme used to derive EPUB-appropriate
// styling. A nil theme leaves the explicitly set CSS as-is.
func (g *EpubGenerator) SetTheme(thm *theme.Theme) {
	g.thm = thm
}

// SetBookRoot sets the containment base directory used to resolve relative
// image paths. Relative images (including those above a chapter's own
// directory, such as a shared ../images) are packaged as long as they resolve
// inside this root. When unset, the common ancestor of all chapter source
// directories is used as the containment base.
func (g *EpubGenerator) SetBookRoot(root string) {
	g.bookRoot = root
}

// AddChapter appends a chapter.
func (g *EpubGenerator) AddChapter(ch EpubChapter) {
	g.chapters = append(g.chapters, ch)
}

// Generate writes the EPUB file to disk.
func (g *EpubGenerator) Generate(outputPath string) error {
	coverAsset, err := g.loadCoverImageAsset()
	if err != nil {
		return fmt.Errorf("load cover image: %w", err)
	}
	chapters, chapterAssets, err := g.collectChapterAssets()
	if err != nil {
		return fmt.Errorf("collect chapter assets: %w", err)
	}

	// Every other backend creates its output directory; the EPUB writer used
	// to be the only one that did not, so `-o release/book.epub` failed with a
	// bare "no such file or directory" on a fresh checkout or CI runner.
	if err := utils.EnsureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("failed to create EPUB output directory: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create EPUB file: %w", err)
	}
	// Safety-net cleanup: on error paths, close the file and remove the
	// partial output so the caller never sees a truncated/corrupt .epub.
	success := false
	fileClosed := false
	defer func() {
		if !success {
			if !fileClosed {
				f.Close() //nolint:errcheck
			}
			if removeErr := os.Remove(outputPath); removeErr != nil {
				slog.Warn("Failed to remove partial EPUB", slog.String("path", outputPath), slog.Any("error", removeErr))
			}
		}
	}()

	w := zip.NewWriter(f)
	writerClosed := false
	defer func() {
		if !writerClosed {
			w.Close() //nolint:errcheck // best-effort on error paths
		}
	}()

	// 1. mimetype must be the first file and must not be compressed.
	mimeWriter, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // Uncompressed
	})
	if err != nil {
		return fmt.Errorf("failed to create mimetype header: %w", err)
	}
	if _, err := mimeWriter.Write([]byte("application/epub+zip")); err != nil {
		return fmt.Errorf("failed to write mimetype entry: %w", err)
	}

	// 2. META-INF/container.xml
	if err := writeZipFile(w, "META-INF/container.xml", containerXML); err != nil {
		return fmt.Errorf("failed to write container.xml: %w", err)
	}

	// 3. OEBPS/content.opf
	opf := g.generateOPF(chapters, coverAsset, chapterAssets)
	if err := writeZipFile(w, "OEBPS/content.opf", opf); err != nil {
		return fmt.Errorf("failed to write content.opf: %w", err)
	}

	// 4. EPUB 3 nav document.
	nav := g.generateNavDocument(chapters)
	if err := writeZipFile(w, "OEBPS/nav.xhtml", nav); err != nil {
		return fmt.Errorf("failed to write nav.xhtml: %w", err)
	}

	// 5. NCX kept for broader reader compatibility.
	ncx := g.generateNCX(chapters)
	if err := writeZipFile(w, "OEBPS/toc.ncx", ncx); err != nil {
		return fmt.Errorf("failed to write toc.ncx: %w", err)
	}

	// 6. Optional generated title page.
	if g.meta.IncludeCover {
		if err := writeZipFile(w, "OEBPS/cover.xhtml", g.generateCoverPage(coverAsset)); err != nil {
			return fmt.Errorf("failed to write cover.xhtml: %w", err)
		}
	}

	// 7. Optional cover image asset.
	if coverAsset != nil {
		if strings.Contains(coverAsset.Filename, "..") || filepath.IsAbs(coverAsset.Filename) {
			return fmt.Errorf("invalid cover asset filename: %s", coverAsset.Filename)
		}
		if err := writeZipBinaryFile(w, "OEBPS/"+coverAsset.Filename, coverAsset.Data); err != nil {
			return fmt.Errorf("failed to write cover image asset: %w", err)
		}
	}
	for _, asset := range chapterAssets {
		// Reject asset filenames that could escape the OEBPS directory.
		if strings.Contains(asset.Filename, "..") || filepath.IsAbs(asset.Filename) {
			return fmt.Errorf("invalid asset filename: %s", asset.Filename)
		}
		if err := writeZipBinaryFile(w, "OEBPS/"+asset.Filename, asset.Data); err != nil {
			return fmt.Errorf("failed to write asset %s: %w", asset.Filename, err)
		}
	}

	// 8. OEBPS/style.css — the theme-derived reader stylesheet followed by the
	// user's custom CSS. Always written: even without a theme or custom CSS a
	// minimal reader-friendly stylesheet is shipped.
	if err := writeZipFile(w, "OEBPS/style.css", g.stylesheet()); err != nil {
		return fmt.Errorf("failed to write style.css: %w", err)
	}

	// 9. Chapter XHTML documents.
	for _, ch := range chapters {
		// Reject chapter filenames that could escape the OEBPS directory.
		if strings.Contains(ch.Filename, "..") || filepath.IsAbs(ch.Filename) {
			return fmt.Errorf("invalid chapter filename: %s", ch.Filename)
		}
		xhtml := g.wrapXHTML(ch.Title, ch.HTML)
		if err := writeZipFile(w, "OEBPS/"+ch.Filename, xhtml); err != nil {
			return fmt.Errorf("failed to write chapter %s: %w", ch.Filename, err)
		}
	}

	// Close the zip.Writer explicitly so we can check the error — the close
	// operation writes the central directory, and if it fails the .epub is
	// corrupt. On error paths the safety-net defer above removes the file.
	writerClosed = true // prevent double-close in deferred cleanup
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finalize epub archive: %w", err)
	}
	closeErr := f.Close()
	fileClosed = true // prevent double-close in deferred cleanup
	if closeErr != nil {
		return fmt.Errorf("failed to close epub file: %w", closeErr)
	}

	success = true
	return nil
}

// generateOPF builds the OPF package file.
func (g *EpubGenerator) generateOPF(chapters []EpubChapter, coverAsset *epubAsset, chapterAssets []*epubAsset) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
`)
	fmt.Fprintf(&b, "    <dc:title id=\"title\">%s</dc:title>\n", utils.EscapeXML(g.meta.Title))
	if g.meta.Subtitle != "" {
		fmt.Fprintf(&b, "    <dc:title id=\"subtitle\">%s</dc:title>\n", utils.EscapeXML(g.meta.Subtitle))
		b.WriteString("    <meta property=\"title-type\" refines=\"#subtitle\">subtitle</meta>\n")
	}
	fmt.Fprintf(&b, "    <dc:creator>%s</dc:creator>\n", utils.EscapeXML(g.meta.Author))
	fmt.Fprintf(&b, "    <dc:language>%s</dc:language>\n", utils.EscapeXML(g.meta.Language))
	fmt.Fprintf(&b, "    <dc:identifier id=\"bookid\">%s</dc:identifier>\n", utils.EscapeXML(g.uniqueIdentifier()))
	if g.meta.Version != "" {
		fmt.Fprintf(&b, "    <meta name=\"mdpress:version\" content=\"%s\"/>\n", utils.EscapeXML(g.meta.Version))
	}
	if g.meta.Description != "" {
		fmt.Fprintf(&b, "    <dc:description>%s</dc:description>\n", utils.EscapeXML(g.meta.Description))
	}
	fmt.Fprintf(&b, "    <meta property=\"dcterms:modified\">%s</meta>\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	if coverAsset != nil {
		b.WriteString("    <meta name=\"cover\" content=\"cover-image\"/>\n")
	}
	b.WriteString("  </metadata>\n  <manifest>\n")
	b.WriteString("    <item id=\"nav\" href=\"nav.xhtml\" media-type=\"application/xhtml+xml\" properties=\"nav\"/>\n")
	b.WriteString("    <item id=\"ncx\" href=\"toc.ncx\" media-type=\"application/x-dtbncx+xml\"/>\n")

	if g.meta.IncludeCover {
		b.WriteString("    <item id=\"cover\" href=\"cover.xhtml\" media-type=\"application/xhtml+xml\"/>\n")
	}
	if coverAsset != nil {
		fmt.Fprintf(&b, "    <item id=\"cover-image\" href=\"%s\" media-type=\"%s\" properties=\"cover-image\"/>\n",
			epubHref(coverAsset.Filename), utils.EscapeXML(coverAsset.MediaType))
	}

	b.WriteString("    <item id=\"css\" href=\"style.css\" media-type=\"text/css\"/>\n")

	for _, asset := range chapterAssets {
		fmt.Fprintf(&b, "    <item id=\"%s\" href=\"%s\" media-type=\"%s\"/>\n",
			utils.EscapeXML(asset.ID), epubHref(asset.Filename), utils.EscapeXML(asset.MediaType))
	}

	for i, ch := range chapters {
		// Chapters containing math embed KaTeX <script> tags and remote CDN
		// resources; EPUB 3 requires those manifest items to declare the
		// "scripted" and "remote-resources" properties to validate.
		props := ""
		if epubChapterHasMath(ch.HTML) {
			props = ` properties="scripted remote-resources"`
		}
		fmt.Fprintf(&b, "    <item id=\"ch%d\" href=\"%s\" media-type=\"application/xhtml+xml\"%s/>\n",
			i, epubHref(ch.Filename), props)
	}

	b.WriteString("  </manifest>\n  <spine toc=\"ncx\">\n")
	if g.meta.IncludeCover {
		b.WriteString("    <itemref idref=\"cover\"/>\n")
	}
	for i := range chapters {
		fmt.Fprintf(&b, "    <itemref idref=\"ch%d\"/>\n", i)
	}
	b.WriteString("  </spine>\n</package>\n")

	return b.String()
}

// generateNCX builds the NCX table of contents.
func (g *EpubGenerator) generateNCX(chapters []EpubChapter) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="` + utils.EscapeXML(g.uniqueIdentifier()) + `"/>
  </head>
  <docTitle><text>`)
	b.WriteString(utils.EscapeXML(g.meta.Title))
	b.WriteString(`</text></docTitle>
  <navMap>
`)
	playOrder := 1
	if g.meta.IncludeCover {
		fmt.Fprintf(&b, "    <navPoint id=\"nav-cover\" playOrder=\"%d\">\n", playOrder)
		b.WriteString("      <navLabel><text>Cover</text></navLabel>\n")
		b.WriteString("      <content src=\"cover.xhtml\"/>\n")
		b.WriteString("    </navPoint>\n")
		playOrder++
	}
	writeNavPoints(&b, chapters, 0, &playOrder, "    ")
	b.WriteString("  </navMap>\n</ncx>\n")
	return b.String()
}

// writeNavPoints emits NCX navPoints for the chapters at the current depth,
// recursing so that a book using `sections:` keeps its hierarchy. The flat
// list it replaced collapsed every level into one, which is what the reader's
// table of contents shows.
func writeNavPoints(b *strings.Builder, chapters []EpubChapter, depth int, playOrder *int, indent string) {
	for i := 0; i < len(chapters); i++ {
		ch := chapters[i]
		if ch.Depth != depth {
			continue
		}
		// Children are the following entries that sit deeper than this one,
		// up to the next sibling.
		end := i + 1
		for end < len(chapters) && chapters[end].Depth > depth {
			end++
		}
		children := chapters[i+1 : end]

		fmt.Fprintf(b, "%s<navPoint id=\"nav-%s\" playOrder=\"%d\">\n", indent, utils.EscapeXML(ch.ID), *playOrder)
		fmt.Fprintf(b, "%s  <navLabel><text>%s</text></navLabel>\n", indent, utils.EscapeXML(ch.Title))
		fmt.Fprintf(b, "%s  <content src=\"%s\"/>\n", indent, epubHref(ch.Filename))
		*playOrder++
		writeNavPoints(b, children, depth+1, playOrder, indent+"  ")
		fmt.Fprintf(b, "%s</navPoint>\n", indent)

		i = end - 1
	}
}

// writeNavItems emits the EPUB 3 navigation document's list items, nesting a
// child <ol> for chapters that have sub-sections.
func writeNavItems(b *strings.Builder, chapters []EpubChapter, indent string) {
	writeNavItemsAtDepth(b, chapters, 0, indent)
}

func writeNavItemsAtDepth(b *strings.Builder, chapters []EpubChapter, depth int, indent string) {
	for i := 0; i < len(chapters); i++ {
		ch := chapters[i]
		if ch.Depth != depth {
			continue
		}
		end := i + 1
		for end < len(chapters) && chapters[end].Depth > depth {
			end++
		}
		children := chapters[i+1 : end]

		fmt.Fprintf(b, "%s<li><a href=\"%s\">%s</a>", indent, epubHref(ch.Filename), utils.EscapeXML(ch.Title))
		if len(children) > 0 {
			b.WriteString("\n" + indent + "  <ol>\n")
			writeNavItemsAtDepth(b, children, depth+1, indent+"    ")
			b.WriteString(indent + "  </ol>\n" + indent)
		}
		b.WriteString("</li>\n")

		i = end - 1
	}
}

// generateNavDocument builds the EPUB 3 navigation document.
func (g *EpubGenerator) generateNavDocument(chapters []EpubChapter) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="`)
	b.WriteString(utils.EscapeXML(languageOrDefault(g.meta.Language)))
	b.WriteString(`">
<head>
  <title>Table of Contents</title>
  <meta charset="UTF-8" />
</head>
<body>
  <nav epub:type="toc" id="toc">
    <h1>Contents</h1>
    <ol>
`)
	if g.meta.IncludeCover {
		b.WriteString(`      <li><a href="cover.xhtml">Cover</a></li>` + "\n")
	}
	writeNavItems(&b, chapters, "      ")
	b.WriteString(`    </ol>
  </nav>
</body>
</html>
`)
	return b.String()
}

// wrapXHTML wraps HTML body content into a complete XHTML document.
// When the body contains math elements (class="math …"), KaTeX is injected so
// that EPUB readers with JavaScript support (e.g. Apple Books) can render the
// formulas. Readers without JS support will display the raw LaTeX source.
func (g *EpubGenerator) wrapXHTML(title, body string) string {
	var b strings.Builder
	body = normalizeHTMLForXHTML(body)
	hasMath := epubChapterHasMath(body)

	// The chapter pipeline strips the leading <h1> because the PDF/HTML/site
	// templates re-render the chapter title themselves. EPUB has no such
	// template layer, so re-emit the title when the body does not already
	// start with a top-level heading.
	if strings.TrimSpace(title) != "" && !epubBodyStartsWithH1(body) {
		body = fmt.Sprintf("<h1>%s</h1>\n", utils.EscapeXML(title)) + body
	}

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="`)
	b.WriteString(utils.EscapeXML(languageOrDefault(g.meta.Language)))
	b.WriteString(`">
<head>
  <meta charset="UTF-8" />
`)
	fmt.Fprintf(&b, "  <title>%s</title>\n", utils.EscapeXML(title))
	// Base styles for images and captions. These act as fallback when no
	// external stylesheet is provided; the user's style.css can override them.
	b.WriteString("  <style>\n")
	b.WriteString("    img { max-width: 100%; height: auto; }\n")
	b.WriteString("    figure { margin: 1rem 0; text-align: center; }\n")
	b.WriteString("    figcaption { text-align: center; font-size: 0.9em; color: #666; margin-top: 0.5rem; font-style: italic; }\n")
	b.WriteString("  </style>\n")
	// style.css is always packaged (theme-derived or minimal fallback).
	b.WriteString("  <link rel=\"stylesheet\" type=\"text/css\" href=\"style.css\"/>\n")
	// Include KaTeX CSS when math is present (works even without JS for visual
	// structure, e.g. in readers that support CSS but not JS).
	// NOTE: This relies on external CDN access. Readers without internet access
	// will not render math formulas. A future version should bundle KaTeX locally.
	if hasMath {
		epubKaTeXWarningOnce.Do(func() {
			slog.Warn("ePub math rendering depends on an external CDN (KaTeX). Readers without internet access will see raw LaTeX source.")
		})
		fmt.Fprintf(&b, "  <link rel=\"stylesheet\" href=\"%s\"/>\n", utils.KaTeXCSSURL)
	}
	b.WriteString("</head>\n<body>\n")
	b.WriteString(body)
	// Inject KaTeX JS at the end of body for readers that support JavaScript.
	if hasMath {
		b.WriteString("\n")
		fmt.Fprintf(&b, "<script src=\"%s\"></script>\n", utils.KaTeXJSURL)
		fmt.Fprintf(&b, "<script src=\"%s\"></script>\n", utils.KaTeXAutoRenderURL)
		b.WriteString("<script>\n")
		b.WriteString("if(typeof renderMathInElement==='function'){\n")
		b.WriteString("  renderMathInElement(document.querySelector('body>section')||document.body,{\n")
		b.WriteString("    delimiters:[{left:'$$',right:'$$',display:true},{left:'$',right:'$',display:false}],\n")
		b.WriteString("    throwOnError:false\n")
		b.WriteString("  });\n")
		b.WriteString("}\n")
		b.WriteString("</script>")
	}
	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}

// generateCoverPage emits a generated title page for EPUB readers, styled to
// match the premium default book cover: a deep navy background (or the
// configured book.cover.background) with text colors adapted to it.
func (g *EpubGenerator) generateCoverPage(coverAsset *epubAsset) string {
	bg := epubCoverBackground(g.meta.CoverBackground)
	// Light text on dark backgrounds, deep navy ink on light ones — the same
	// adaptive logic internal/cover applies to the PDF/HTML cover.
	titleColor := "#f6f8fc"
	subtitleColor := "rgba(255, 255, 255, 0.85)"
	metaColor := "rgba(255, 255, 255, 0.9)"
	versionColor := "rgba(255, 255, 255, 0.7)"
	imageShadow := "0 18px 50px rgba(0, 0, 0, 0.45)"
	if epubIsLightColor(bg) {
		titleColor = "#14304a"
		subtitleColor = "#475569"
		metaColor = "#334155"
		versionColor = "#64748b"
		imageShadow = "0 18px 50px rgba(15, 23, 42, 0.22)"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="`)
	b.WriteString(utils.EscapeXML(languageOrDefault(g.meta.Language)))
	b.WriteString(`">
<head>
  <title>Cover</title>
  <meta charset="UTF-8" />
  <style>
    html, body { height: 100%; margin: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans CJK SC", "Source Han Sans SC", sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
`)
	fmt.Fprintf(&b, "      background-color: %s;\n", bg)
	fmt.Fprintf(&b, "      color: %s;\n", titleColor)
	b.WriteString(`      text-align: center;
      padding: 8vh 8vw;
      box-sizing: border-box;
    }
    .cover { max-width: 42rem; }
    .cover-image-wrap { margin-bottom: 2rem; }
    .cover-image {
      display: block;
      max-width: min(100%, 24rem);
      max-height: 70vh;
      margin: 0 auto;
`)
	fmt.Fprintf(&b, "      box-shadow: %s;\n", imageShadow)
	b.WriteString(`      border-radius: 0.4rem;
      background: #fff;
    }
    .title { font-size: 2.2rem; font-weight: 700; line-height: 1.2; margin: 0; letter-spacing: 0.02em; }
`)
	fmt.Fprintf(&b, "    .subtitle { font-size: 1.1rem; color: %s; margin: 1rem 0 0; }\n", subtitleColor)
	fmt.Fprintf(&b, "    .meta { margin-top: 2.5rem; color: %s; }\n", metaColor)
	b.WriteString("    .meta div + div { margin-top: 0.45rem; }\n")
	fmt.Fprintf(&b, "    .version { color: %s; }\n", versionColor)
	b.WriteString(`  </style>
</head>
<body>
  <section class="cover" epub:type="cover">
`)
	if coverAsset != nil {
		b.WriteString("    <div class=\"cover-image-wrap\">\n")
		fmt.Fprintf(&b, "      <img class=\"cover-image\" src=\"%s\" alt=\"%s\" />\n",
			utils.EscapeXML(coverAsset.Filename), utils.EscapeXML(g.meta.Title))
		b.WriteString("    </div>\n")
	}
	fmt.Fprintf(&b, "    <h1 class=\"title\">%s</h1>\n", utils.EscapeXML(g.meta.Title))
	if g.meta.Subtitle != "" {
		fmt.Fprintf(&b, "    <p class=\"subtitle\">%s</p>\n", utils.EscapeXML(g.meta.Subtitle))
	}
	b.WriteString("    <div class=\"meta\">\n")
	if g.meta.Author != "" {
		fmt.Fprintf(&b, "      <div>%s</div>\n", utils.EscapeXML(g.meta.Author))
	}
	if g.meta.Version != "" {
		fmt.Fprintf(&b, "      <div class=\"version\">Version %s</div>\n", utils.EscapeXML(g.meta.Version))
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n</body>\n</html>\n")
	return b.String()
}

// epubChapterHasMath reports whether chapter HTML contains math markup
// (Goldmark math extension output), which is rendered via KaTeX scripts.
func epubChapterHasMath(html string) bool {
	return strings.Contains(html, `class="math `)
}

// epubBodyStartsWithH1 reports whether the body content begins with a
// top-level <h1> heading (ignoring leading whitespace).
func epubBodyStartsWithH1(body string) bool {
	trimmed := strings.TrimSpace(body)
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "<h1") {
		return false
	}
	if len(lower) == 3 {
		return false // "<h1" alone is not a complete tag
	}
	switch lower[3] {
	case '>', ' ', '\t', '\n', '/':
		return true
	}
	return false
}

// epubDefaultCoverBg mirrors internal/cover's premium deep-navy default cover
// background.
const epubDefaultCoverBg = "#102a43"

// epubCSSColorPattern matches safe CSS color values (hex, rgb[a], hsl[a],
// named colors) — the same validation internal/cover applies to
// book.cover.background.
var epubCSSColorPattern = regexp.MustCompile(`^(?i)(?:#[0-9a-f]{3,8}|(?:rgb|rgba|hsl|hsla)\([\d\s,%.]+\)|[a-z]{1,30})$`)

// epubCoverBackground returns the configured cover background when it is a
// safe CSS color value, otherwise the default navy.
func epubCoverBackground(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" && epubCSSColorPattern.MatchString(configured) {
		return configured
	}
	return epubDefaultCoverBg
}

// epubIsLightColor reports whether a CSS color is perceptually light. Only hex
// colors (#rgb, #rgba, #rrggbb, #rrggbbaa) are analyzed; all other formats are
// assumed dark so light text is the safer default. Same heuristic as
// internal/cover (ITU-R BT.601 luminance, cutoff 186).
func epubIsLightColor(color string) bool {
	color = strings.TrimSpace(color)
	if !strings.HasPrefix(color, "#") {
		return false
	}
	hex := color[1:]
	// Expand shorthand (#rgb -> #rrggbb, #rgba -> #rrggbb).
	if len(hex) == 3 || len(hex) == 4 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	// Strip alpha channel from #rrggbbaa.
	if len(hex) == 8 {
		hex = hex[:6]
	}
	if len(hex) < 6 {
		return false
	}
	r := epubHexVal(hex[0])*16 + epubHexVal(hex[1])
	g := epubHexVal(hex[2])*16 + epubHexVal(hex[3])
	b := epubHexVal(hex[4])*16 + epubHexVal(hex[5])
	luminance := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return luminance > 186
}

func epubHexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return 0
	}
}

// epubMonoFontFamily is the monospace stack used for code in EPUB output.
const epubMonoFontFamily = "ui-monospace, 'SF Mono', Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace"

// epubMinimalCSS is the stylesheet shipped when no theme is available. Only
// structural, reader-friendly rules: no colors forced on body text, no
// absolute sizes, links underlined so they stay distinguishable without color.
const epubMinimalCSS = `/* mdpress EPUB stylesheet (minimal fallback). */
a { text-decoration: underline; }
img { max-width: 100%; height: auto; }
pre { white-space: pre-wrap; overflow-wrap: anywhere; }
table { border-collapse: collapse; width: 100%; }
table th, table td { border: 1px solid #cccccc; padding: 0.55em 0.85em; text-align: left; }
blockquote { border-left: 3px solid #cccccc; margin: 1.2em 0; padding: 0.2em 0 0.2em 1.1em; }
`

// stylesheet returns the full content of OEBPS/style.css: the theme-derived
// reader stylesheet followed by the user's custom CSS so custom rules win.
func (g *EpubGenerator) stylesheet() string {
	css := g.epubThemeCSS()
	// Chapters carry chroma class markup, so without these rules every code
	// block in the book renders as undifferentiated plain text. Only the light
	// palette is packaged: reading systems apply their own night mode over it.
	if highlight := markdown.HighlightCSSLight(g.codeTheme()); strings.TrimSpace(highlight) != "" {
		css += "\n/* Syntax highlighting */\n" + highlight
	}
	if strings.TrimSpace(g.css) != "" {
		css += "\n/* Custom user CSS */\n" + g.css
	}
	return css
}

// codeTheme returns the chroma theme the chapters were highlighted with,
// falling back to the generator's default.
func (g *EpubGenerator) codeTheme() string {
	if g.thm != nil && g.thm.CodeTheme != "" {
		return g.thm.CodeTheme
	}
	return "github"
}

// epubColorOrDefault returns the trimmed color value, or def when empty.
func epubColorOrDefault(v, def string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

// epubThemeCSS derives a reader-friendly stylesheet from the active theme.
// Unlike theme.ToCSS (which targets paged/print output), it:
//   - sets no body margins and no forced body background — reading systems own
//     page geometry and night/dark modes;
//   - uses only relative font sizes (em/%) so the reader's font-size
//     preference is honored;
//   - emits literal color values instead of CSS custom properties so older
//     EPUB engines (RMSDK, Kindle KF8) don't silently drop every themed rule;
//   - underlines links so they remain distinguishable without color
//     (WCAG 1.4.1), e.g. on grayscale e-ink screens.
func (g *EpubGenerator) epubThemeCSS() string {
	if g.thm == nil {
		return epubMinimalCSS
	}

	text := epubColorOrDefault(g.thm.Colors.Text, "#1F2933")
	heading := epubColorOrDefault(g.thm.Colors.Heading, "#12344D")
	link := epubColorOrDefault(g.thm.Colors.Link, "#1C5A9E")
	codeBg := epubColorOrDefault(g.thm.Colors.CodeBg, "#F5F7F9")
	codeText := epubColorOrDefault(g.thm.Colors.CodeText, "#1F2933")
	accent := epubColorOrDefault(g.thm.Colors.Accent, "#1C5A9E")
	border := epubColorOrDefault(g.thm.Colors.Border, "#E4E7EB")
	lineHeight := g.thm.LineHeight
	if lineHeight <= 0 {
		lineHeight = 1.6
	}

	var b strings.Builder
	b.WriteString("/* mdpress EPUB stylesheet — derived from the document theme. */\n")
	b.WriteString("/* Reader-friendly: no page margins, no forced background, relative sizes, literal colors. */\n\n")

	b.WriteString("body {\n")
	if ff := strings.TrimSpace(g.thm.FontFamily); ff != "" {
		fmt.Fprintf(&b, "  font-family: %s;\n", ff)
	}
	fmt.Fprintf(&b, "  line-height: %.2f;\n", lineHeight)
	fmt.Fprintf(&b, "  color: %s;\n", text)
	b.WriteString("}\n\n")

	// Headings with a modest relative scale — no renderer-level CSS layer
	// exists for EPUB, so the scale lives here.
	fmt.Fprintf(&b, "h1, h2, h3, h4, h5, h6 {\n  color: %s;\n  font-weight: 600;\n  line-height: 1.35;\n}\n\n", heading)
	b.WriteString("h1 { font-size: 1.8em; }\n")
	b.WriteString("h2 { font-size: 1.45em; }\n")
	b.WriteString("h3 { font-size: 1.2em; }\n")
	b.WriteString("h4, h5, h6 { font-size: 1em; }\n\n")

	fmt.Fprintf(&b, "a {\n  color: %s;\n  text-decoration: underline;\n}\n\n", link)

	fmt.Fprintf(&b, "code, pre {\n  font-family: %s;\n  color: %s;\n}\n\n", epubMonoFontFamily, codeText)

	// Inline code chip; reset inside pre (avoids :not(), which some older
	// EPUB engines do not support).
	fmt.Fprintf(&b, "code {\n  background-color: %s;\n  padding: 0.12em 0.36em;\n  border-radius: 4px;\n  font-size: 0.88em;\n}\n\n", codeBg)
	fmt.Fprintf(&b, "pre {\n  padding: 0.9em 1.1em;\n  font-size: 0.82em;\n  line-height: 1.55;\n  border: 1px solid %s;\n  border-radius: 6px;\n  overflow-x: auto;\n  white-space: pre-wrap;\n  overflow-wrap: anywhere;\n  word-break: break-all;\n}\n\n", border)
	b.WriteString("pre code {\n  background: none;\n  padding: 0;\n  border-radius: 0;\n  font-size: 1em;\n}\n\n")

	fmt.Fprintf(&b, "blockquote {\n  border-left: 3px solid %s;\n  margin: 1.2em 0;\n  padding: 0.2em 0 0.2em 1.1em;\n  color: %s;\n  opacity: 0.78;\n}\n\n", accent, text)

	b.WriteString("table {\n  border-collapse: collapse;\n  width: 100%;\n  margin: 1.2em 0;\n  font-size: 0.96em;\n}\n\n")
	fmt.Fprintf(&b, "table th, table td {\n  border: 1px solid %s;\n  padding: 0.55em 0.85em;\n  text-align: left;\n  overflow-wrap: anywhere;\n  word-break: break-word;\n}\n\n", border)
	fmt.Fprintf(&b, "table th {\n  background-color: %s;\n  color: %s;\n  font-weight: 600;\n  border-bottom: 2px solid %s;\n}\n\n", codeBg, heading, accent)
	fmt.Fprintf(&b, "table tbody tr:nth-child(even) td {\n  background-color: %s;\n}\n\n", codeBg)

	b.WriteString("img {\n  max-width: 100%;\n  height: auto;\n}\n")

	return b.String()
}

const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

func writeZipFile(w *zip.Writer, name, content string) error {
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", name, err)
	}
	if _, err = fw.Write([]byte(content)); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}

func writeZipBinaryFile(w *zip.Writer, name string, data []byte) error {
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", name, err)
	}
	if _, err = fw.Write(data); err != nil {
		return fmt.Errorf("failed to write %s: %w", name, err)
	}
	return nil
}

func languageOrDefault(lang string) string {
	if strings.TrimSpace(lang) == "" {
		return "en"
	}
	return lang
}

func (g *EpubGenerator) loadCoverImageAsset() (*epubAsset, error) {
	if !g.meta.IncludeCover || strings.TrimSpace(g.meta.CoverImagePath) == "" {
		return nil, nil
	}

	coverPath := g.meta.CoverImagePath
	data, err := utils.ReadFile(coverPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read EPUB cover image %q: %w", coverPath, err)
	}
	if int64(len(data)) > utils.MaxImageSize {
		return nil, fmt.Errorf("EPUB cover image %q exceeds maximum size (%d bytes)", coverPath, utils.MaxImageSize)
	}

	mediaType := utils.DetectImageMIME(coverPath, data)
	ext := strings.ToLower(filepath.Ext(coverPath))
	if ext == "" {
		ext = extensionForMediaType(mediaType)
	}
	if ext == "" {
		ext = ".png"
	}

	return &epubAsset{
		ID:        "cover-image",
		Filename:  "assets/cover" + ext,
		MediaType: mediaType,
		Data:      data,
	}, nil
}

func (g *EpubGenerator) uniqueIdentifier() string {
	parts := []string{"urn:mdpress"}
	if title := slugify(g.meta.Title); title != "" {
		parts = append(parts, title)
	}
	if author := slugify(g.meta.Author); author != "" {
		parts = append(parts, author)
	}
	if g.meta.Version != "" {
		parts = append(parts, slugify(g.meta.Version))
	}
	if len(parts) == 1 {
		parts = append(parts, "book")
	}
	return strings.Join(parts, ":")
}

var epubVoidTagPattern = regexp.MustCompile(`(?i)<(img|br|hr|input|meta|link|col|area|base|embed|source|track|wbr)(\s[^<>]*?)?>`)

// ampAndEntityPattern matches an & optionally followed by a valid HTML entity
// reference body (name + semicolon). Go's RE2 does not support negative
// lookaheads, so we match both cases and disambiguate in the replacement
// function: a match of length 1 is a bare & that must be escaped; a longer
// match is an existing entity reference that must be preserved.
var ampAndEntityPattern = regexp.MustCompile(`&([a-zA-Z0-9#][a-zA-Z0-9#]{0,31};)?`)

// booleanAttrPattern matches HTML boolean attributes like checked, disabled,
// selected, etc., which XHTML requires to have an explicit value.
var booleanAttrPattern = regexp.MustCompile(`(?i)\s(checked|disabled|selected|readonly|multiple|autofocus|autoplay|controls|loop|muted|open|required|reversed|hidden|defer|async|novalidate)(\s|/?>)`)

// epubStartTagPattern matches a start tag, so attribute rewriting can be
// confined to markup instead of running across the document's text. Goldmark
// escapes ">" inside attribute values, so a tag never contains a bare one.
var epubStartTagPattern = regexp.MustCompile(`<[a-zA-Z][^<>]*>`)

// normalizeHTMLForXHTML converts HTML produced by Goldmark into valid XHTML.
//
// It handles the following transformations:
//  1. Self-closes void elements (e.g. <br> → <br />)
//  2. Escapes bare ampersands (e.g. A&B → A&amp;B)
//  3. Expands boolean attributes (e.g. checked → checked="checked")
func normalizeHTMLForXHTML(html string) string {
	// 1. Self-close void elements.
	html = epubVoidTagPattern.ReplaceAllStringFunc(html, func(match string) string {
		if strings.HasSuffix(match, "/>") {
			return match
		}
		return strings.TrimSuffix(match, ">") + " />"
	})

	// 2. Escape bare ampersands that are not part of a valid entity reference.
	// A match of length 1 is a bare &; longer matches are existing entity refs.
	html = ampAndEntityPattern.ReplaceAllStringFunc(html, func(m string) string {
		if len(m) > 1 {
			return m // already a valid entity reference — keep it
		}
		return "&amp;"
	})

	// 3. Expand boolean attributes to attribute="attribute" form.
	//
	// This must run inside start tags only. The attribute names are ordinary
	// English words ("multiple", "open", "required", "hidden", "controls"…),
	// so applying the pattern to the whole document rewrote prose and code
	// samples: "supports multiple output formats" became
	// `supports multiple="multiple" output formats` in the reader.
	expandBoolAttr := func(match string) string {
		sub := booleanAttrPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		attr := strings.ToLower(sub[1])
		trailing := sub[2]
		return " " + attr + `="` + attr + `"` + trailing
	}
	html = epubStartTagPattern.ReplaceAllStringFunc(html, func(tag string) string {
		// The trailing group (\s|/?>) consumes the whitespace separating
		// adjacent boolean attributes, so one pass misses the second of a
		// pair like `disabled multiple`; run it twice.
		tag = booleanAttrPattern.ReplaceAllStringFunc(tag, expandBoolAttr)
		return booleanAttrPattern.ReplaceAllStringFunc(tag, expandBoolAttr)
	})

	return html
}

// epubHref renders a packaged file name as an XML-safe, percent-encoded URI
// reference. OCF requires every path in the package document, NCX and nav to
// be a valid URI, so a chapter file named after a CJK heading must be
// percent-encoded here even though the ZIP entry keeps its readable UTF-8
// name (Go sets the archive's UTF-8 flag).
func epubHref(filename string) string {
	return utils.EscapeXML((&url.URL{Path: filename}).EscapedPath())
}

// epubImageSrcPattern reuses the shared img-src regex from pkg/utils.
var epubImageSrcPattern = utils.ImgSrcRegex
var dataURIImagePattern = regexp.MustCompile(`^data:([^;,]+);base64,(.+)$`)

func (g *EpubGenerator) collectChapterAssets() ([]EpubChapter, []*epubAsset, error) {
	chapters := make([]EpubChapter, len(g.chapters))
	copy(chapters, g.chapters)

	assets := make([]*epubAsset, 0)
	remoteTempDir, err := os.MkdirTemp("", "mdpress-epub-assets-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temporary EPUB asset directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(remoteTempDir) }()
	collector := &epubAssetCollector{
		cache: make(map[string]*epubAsset),
	}

	// Determine the containment base: the configured book root when available,
	// otherwise the common ancestor of all chapter source directories. This
	// lets shared images referenced above a chapter's own directory (e.g.
	// ../images from chapters in docs/) resolve inside the book and be packaged.
	containBase := g.containmentBase(chapters)

	for i := range chapters {
		updated, chapterAssets, err := collectImageAssetsFromHTML(chapters[i].HTML, chapters[i].SourceDir, containBase, remoteTempDir, collector)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to collect EPUB assets for chapter %q: %w", chapters[i].Title, err)
		}
		chapters[i].HTML = updated
		assets = append(assets, chapterAssets...)
	}

	return chapters, assets, nil
}

// containmentBase returns the directory that resolved relative image paths must
// stay within. It prefers the explicitly configured book root; otherwise it
// falls back to the common ancestor directory of all chapter source dirs.
func (g *EpubGenerator) containmentBase(chapters []EpubChapter) string {
	if strings.TrimSpace(g.bookRoot) != "" {
		if abs, err := filepath.Abs(g.bookRoot); err == nil {
			return abs
		}
		return g.bookRoot
	}
	return commonAncestorDir(chapters)
}

// commonAncestorDir computes the deepest directory that is an ancestor of (or
// equal to) every non-empty chapter source directory. It returns "" when there
// are no usable source directories, in which case containment falls back to
// each chapter's own source directory.
func commonAncestorDir(chapters []EpubChapter) string {
	var common string
	for _, ch := range chapters {
		dir := strings.TrimSpace(ch.SourceDir)
		if dir == "" {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if common == "" {
			common = abs
			continue
		}
		common = commonPrefixDir(common, abs)
	}
	return common
}

// commonPrefixDir returns the longest shared ancestor directory of two absolute
// paths.
func commonPrefixDir(a, b string) string {
	aParts := strings.Split(a, string(filepath.Separator))
	bParts := strings.Split(b, string(filepath.Separator))
	n := len(aParts)
	if len(bParts) < n {
		n = len(bParts)
	}
	i := 0
	for i < n && aParts[i] == bParts[i] {
		i++
	}
	prefix := strings.Join(aParts[:i], string(filepath.Separator))
	if prefix == "" {
		// Preserve the leading separator for absolute POSIX paths.
		return string(filepath.Separator)
	}
	return prefix
}

func collectImageAssetsFromHTML(html string, sourceDir string, containBase string, remoteTempDir string, collector *epubAssetCollector) (string, []*epubAsset, error) {
	var collectErr error
	assets := make([]*epubAsset, 0)

	updated := epubImageSrcPattern.ReplaceAllStringFunc(html, func(match string) string {
		if collectErr != nil {
			return match
		}

		matches := epubImageSrcPattern.FindStringSubmatch(match)
		if len(matches) < 4 {
			return match
		}

		src := matches[2]
		prefix := matches[1]
		suffix := matches[3]

		asset, created, err := collector.assetForSource(src, sourceDir, containBase, remoteTempDir)
		if err != nil {
			collectErr = err
			return match
		}
		if asset == nil {
			return match
		}

		if created {
			assets = append(assets, asset)
		}
		return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, asset.Filename, suffix)
	})

	if collectErr != nil {
		return "", nil, collectErr
	}
	return updated, assets, nil
}

func (c *epubAssetCollector) assetForSource(src string, sourceDir string, containBase string, remoteTempDir string) (*epubAsset, bool, error) {
	key, asset, err := buildImageAssetFromSource(src, sourceDir, containBase, remoteTempDir, c.nextIndex)
	if err != nil || asset == nil {
		return asset, false, err
	}
	if existing, ok := c.cache[key]; ok {
		return existing, false, nil
	}
	c.cache[key] = asset
	c.nextIndex++
	return asset, true, nil
}

func buildImageAssetFromSource(src string, sourceDir string, containBase string, remoteTempDir string, index int) (string, *epubAsset, error) {
	if strings.HasPrefix(src, "data:") {
		asset, err := buildDataURIImageAsset(src, index)
		return "data:" + src, asset, err
	}
	if utils.IsRemoteURL(src) {
		asset, err := buildRemoteImageAsset(src, remoteTempDir, index)
		return "remote:" + src, asset, err
	}
	if filepath.IsAbs(src) {
		// Reject absolute paths to prevent reading arbitrary files.
		slog.Warn("Skipping EPUB image with absolute path; keeping original src", slog.String("src", src))
		return "", nil, nil
	}
	if sourceDir != "" && src != "" && !strings.HasPrefix(src, "#") {
		// Resolve the image relative to the chapter's own directory, but use
		// the book root (containBase) as the containment boundary so that a
		// shared image referenced above the chapter directory (e.g.
		// ../images/pic.png from a chapter in docs/) is still packaged.
		resolved := filepath.Clean(filepath.Join(sourceDir, filepath.FromSlash(src)))
		// Fall back to the chapter directory when no wider containment base is
		// available (preserves the previous, stricter behavior).
		base := containBase
		if strings.TrimSpace(base) == "" {
			base = sourceDir
		}
		// Ensure the resolved path stays within the containment base.
		// Use EvalSymlinks to prevent symlink-based containment bypass.
		absBase, err1 := filepath.Abs(base)
		absResolved, err2 := filepath.Abs(resolved)
		if err1 != nil || err2 != nil {
			slog.Warn("Skipping EPUB image; cannot resolve path", slog.String("src", src))
			return "", nil, nil
		}
		// Resolve symlinks to prevent containment bypass. Only apply when
		// both paths can be resolved to keep them comparable.
		if evaledR, errR := filepath.EvalSymlinks(absResolved); errR == nil {
			if evaledB, errB := filepath.EvalSymlinks(absBase); errB == nil {
				absResolved = evaledR
				absBase = evaledB
			}
		}
		if !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) && absResolved != absBase {
			slog.Warn("Skipping EPUB image outside book root; keeping original src",
				slog.String("src", src), slog.String("resolved", absResolved), slog.String("root", absBase))
			return "", nil, nil
		}
		asset, err := buildFileImageAsset(resolved, index)
		if err != nil {
			// Missing/unreadable file: warn and keep the original src rather
			// than failing the whole build.
			slog.Warn("Skipping EPUB image that could not be read; keeping original src",
				slog.String("src", src), slog.Any("error", err))
			return "file:" + resolved, nil, nil
		}
		return "file:" + resolved, asset, nil
	}
	return "", nil, nil
}

// dataURINonBase64Pattern matches non-base64 data URIs of the form
// data:<mediatype>,<data> where <data> is either plain (e.g. utf8) or
// URL/percent-encoded text (common for inline SVG). The payload is decoded
// via url.PathUnescape below.
var dataURINonBase64Pattern = regexp.MustCompile(`^data:([^;,]+)(;[^,]*)?,(.*)$`)

func buildDataURIImageAsset(src string, index int) (*epubAsset, error) {
	matches := dataURIImagePattern.FindStringSubmatch(src)
	if len(matches) != 3 {
		// Not a base64 data URI. Try to salvage non-base64 variants (e.g.
		// data:image/svg+xml;utf8,<svg...> or URL-encoded SVG payloads) so a
		// single unusual inline image never aborts the whole EPUB build.
		if asset, ok := buildNonBase64DataURIImageAsset(src, index); ok {
			return asset, nil
		}
		// Genuinely unsupported: log a warning and keep the original src by
		// returning (nil, nil) so the build degrades gracefully instead of
		// failing the entire .epub.
		slog.Warn("Skipping unsupported data URI image; keeping original src", slog.String("preview", dataURIPreview(src)))
		return nil, nil
	}

	mediaType := strings.TrimSpace(matches[1])

	// Estimate decoded size before allocating to prevent OOM from huge data URIs.
	// Base64 encodes 3 bytes into 4 characters, so decoded ≈ len * 3/4.
	if estimatedSize := len(matches[2]) * 3 / 4; estimatedSize > int(utils.MaxImageSize) {
		return nil, fmt.Errorf("data URI image exceeds maximum size (%d bytes)", utils.MaxImageSize)
	}

	data, err := base64.StdEncoding.DecodeString(matches[2])
	if err != nil {
		return nil, fmt.Errorf("decode data URI image: %w", err)
	}

	ext := extensionForMediaType(mediaType)
	if ext == "" {
		ext = ".bin"
	}

	return &epubAsset{
		ID:        fmt.Sprintf("asset-img-%03d", index),
		Filename:  filepath.ToSlash(filepath.Join("assets", fmt.Sprintf("img-%03d%s", index, ext))),
		MediaType: mediaType,
		Data:      data,
	}, nil
}

// dataURIPreview returns a short, log-safe preview of a data URI.
func dataURIPreview(src string) string {
	preview := src
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	return preview
}

// buildNonBase64DataURIImageAsset handles non-base64 data URIs whose payload is
// stored inline as plain or URL/percent-encoded text (commonly used for inline
// SVG, e.g. data:image/svg+xml;utf8,<svg...> or data:image/svg+xml,%3Csvg...).
// It returns (asset, true) when the URI is a recognized non-base64 form, or
// (nil, false) when it is not so the caller can decide how to degrade.
func buildNonBase64DataURIImageAsset(src string, index int) (*epubAsset, bool) {
	matches := dataURINonBase64Pattern.FindStringSubmatch(src)
	if len(matches) != 4 {
		return nil, false
	}
	// Defensively skip base64 URIs; those are handled by the base64 path.
	if strings.Contains(strings.ToLower(matches[2]), "base64") {
		return nil, false
	}
	mediaType := strings.TrimSpace(matches[1])
	payload := matches[3]

	// Reject oversized payloads before decoding.
	if len(payload) > int(utils.MaxImageSize) {
		return nil, false
	}

	// Payloads may be percent-encoded (e.g. %3Csvg%3E). PathUnescape decodes
	// those; if there is nothing to decode it returns the input unchanged.
	decoded, err := url.PathUnescape(payload)
	if err != nil {
		// Fall back to the raw payload when unescaping fails (e.g. a stray %).
		decoded = payload
	}

	data := []byte(decoded)
	if int64(len(data)) > utils.MaxImageSize {
		return nil, false
	}

	ext := extensionForMediaType(mediaType)
	if ext == "" {
		ext = ".bin"
	}

	return &epubAsset{
		ID:        fmt.Sprintf("asset-img-%03d", index),
		Filename:  filepath.ToSlash(filepath.Join("assets", fmt.Sprintf("img-%03d%s", index, ext))),
		MediaType: mediaType,
		Data:      data,
	}, true
}

func buildFileImageAsset(src string, index int) (*epubAsset, error) {
	data, err := utils.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read image file %q: %w", src, err)
	}
	if int64(len(data)) > utils.MaxImageSize {
		return nil, fmt.Errorf("image file %q exceeds maximum size (%d bytes)", src, utils.MaxImageSize)
	}

	mediaType := utils.DetectImageMIME(src, data)
	return buildImageAssetFromBytes(data, src, mediaType, index), nil
}

func buildRemoteImageAsset(src string, remoteTempDir string, index int) (*epubAsset, error) {
	if remoteTempDir == "" {
		return nil, errors.New("temporary directory for remote EPUB assets is not available")
	}

	localPath, err := utils.DownloadImage(src, remoteTempDir)
	if err != nil {
		return nil, fmt.Errorf("download remote image %q: %w", src, err)
	}
	data, err := utils.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read downloaded remote image %q: %w", src, err)
	}
	if int64(len(data)) > utils.MaxImageSize {
		return nil, fmt.Errorf("remote image %q exceeds maximum size (%d bytes)", src, utils.MaxImageSize)
	}
	sourceName := src
	if parsed, parseErr := url.Parse(src); parseErr == nil && parsed.Path != "" {
		sourceName = parsed.Path
	}
	mediaType := utils.DetectImageMIME(sourceName, data)
	return buildImageAssetFromBytes(data, sourceName, mediaType, index), nil
}

func buildImageAssetFromBytes(data []byte, sourceName string, mediaType string, index int) *epubAsset {
	ext := strings.ToLower(filepath.Ext(sourceName))
	if ext == "" {
		ext = extensionForMediaType(mediaType)
	}
	if ext == "" {
		ext = ".bin"
	}

	filename := filepath.Base(sourceName)
	filename = sanitizeAssetFilename(strings.TrimSuffix(filename, filepath.Ext(filename)))
	if filename == "" {
		filename = fmt.Sprintf("img-%03d", index)
	}

	return &epubAsset{
		ID:        fmt.Sprintf("asset-img-%03d", index),
		Filename:  filepath.ToSlash(filepath.Join("assets", fmt.Sprintf("%s-%03d%s", filename, index, ext))),
		MediaType: mediaType,
		Data:      data,
	}
}

var nonAssetFilenamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeAssetFilename(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = nonAssetFilenamePattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-.")
	return name
}

func extensionForMediaType(mediaType string) string {
	if ext, ok := utils.ImageExtForMIME(strings.ToLower(mediaType)); ok {
		return ext
	}
	return ""
}
