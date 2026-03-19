// epub.go generates EPUB 3 ebooks.
// The resulting .epub file is a ZIP archive containing XHTML, CSS, OPF metadata,
// and both EPUB 3 navigation and NCX files for wider reader compatibility.
package output

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yeasy/mdpress/pkg/utils"
)

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
}

// EpubChapter stores one EPUB chapter.
type EpubChapter struct {
	Title     string
	ID        string
	Filename  string
	HTML      string // XHTML body content.
	SourceDir string // Source directory used to resolve relative asset paths.
}

// EpubGenerator builds an EPUB file.
type EpubGenerator struct {
	meta     EpubMeta
	chapters []EpubChapter
	css      string
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

// SetCSS sets the global CSS.
func (g *EpubGenerator) SetCSS(css string) {
	g.css = css
}

// AddChapter appends a chapter.
func (g *EpubGenerator) AddChapter(ch EpubChapter) {
	g.chapters = append(g.chapters, ch)
}

// Generate writes the EPUB file to disk.
func (g *EpubGenerator) Generate(outputPath string) error {
	coverAsset, err := g.loadCoverImageAsset()
	if err != nil {
		return err
	}
	chapters, chapterAssets, err := g.collectChapterAssets()
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create EPUB file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	// 1. mimetype must be the first file and must not be compressed.
	mimeWriter, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // Uncompressed
	})
	if err != nil {
		return err
	}
	mimeWriter.Write([]byte("application/epub+zip"))

	// 2. META-INF/container.xml
	if err := writeZipFile(w, "META-INF/container.xml", containerXML); err != nil {
		return err
	}

	// 3. OEBPS/content.opf
	opf := g.generateOPF(chapters, coverAsset, chapterAssets)
	if err := writeZipFile(w, "OEBPS/content.opf", opf); err != nil {
		return err
	}

	// 4. EPUB 3 nav document.
	nav := g.generateNavDocument(chapters)
	if err := writeZipFile(w, "OEBPS/nav.xhtml", nav); err != nil {
		return err
	}

	// 5. NCX kept for broader reader compatibility.
	ncx := g.generateNCX(chapters)
	if err := writeZipFile(w, "OEBPS/toc.ncx", ncx); err != nil {
		return err
	}

	// 6. Optional generated title page.
	if g.meta.IncludeCover {
		if err := writeZipFile(w, "OEBPS/cover.xhtml", g.generateCoverPage(coverAsset)); err != nil {
			return err
		}
	}

	// 7. Optional cover image asset.
	if coverAsset != nil {
		if err := writeZipBinaryFile(w, "OEBPS/"+coverAsset.Filename, coverAsset.Data); err != nil {
			return err
		}
	}
	for _, asset := range chapterAssets {
		if err := writeZipBinaryFile(w, "OEBPS/"+asset.Filename, asset.Data); err != nil {
			return err
		}
	}

	// 8. OEBPS/style.css
	if g.css != "" {
		if err := writeZipFile(w, "OEBPS/style.css", g.css); err != nil {
			return err
		}
	}

	// 9. Chapter XHTML documents.
	for _, ch := range chapters {
		xhtml := g.wrapXHTML(ch.Title, ch.HTML)
		if err := writeZipFile(w, "OEBPS/"+ch.Filename, xhtml); err != nil {
			return err
		}
	}

	return nil
}

// generateOPF builds the OPF package file.
func (g *EpubGenerator) generateOPF(chapters []EpubChapter, coverAsset *epubAsset, chapterAssets []*epubAsset) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
`)
	b.WriteString(fmt.Sprintf("    <dc:title id=\"title\">%s</dc:title>\n", utils.EscapeXML(g.meta.Title)))
	if g.meta.Subtitle != "" {
		b.WriteString(fmt.Sprintf("    <dc:title id=\"subtitle\">%s</dc:title>\n", utils.EscapeXML(g.meta.Subtitle)))
		b.WriteString("    <meta property=\"title-type\" refines=\"#subtitle\">subtitle</meta>\n")
	}
	b.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", utils.EscapeXML(g.meta.Author)))
	b.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", utils.EscapeXML(g.meta.Language)))
	b.WriteString(fmt.Sprintf("    <dc:identifier id=\"bookid\">%s</dc:identifier>\n", utils.EscapeXML(g.uniqueIdentifier())))
	if g.meta.Version != "" {
		b.WriteString(fmt.Sprintf("    <meta name=\"mdpress:version\" content=\"%s\"/>\n", utils.EscapeXML(g.meta.Version)))
	}
	if g.meta.Description != "" {
		b.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", utils.EscapeXML(g.meta.Description)))
	}
	b.WriteString(fmt.Sprintf("    <meta property=\"dcterms:modified\">%s</meta>\n", time.Now().UTC().Format("2006-01-02T15:04:05Z")))
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
		b.WriteString(fmt.Sprintf("    <item id=\"cover-image\" href=\"%s\" media-type=\"%s\" properties=\"cover-image\"/>\n",
			utils.EscapeXML(coverAsset.Filename), utils.EscapeXML(coverAsset.MediaType)))
	}

	if g.css != "" {
		b.WriteString("    <item id=\"css\" href=\"style.css\" media-type=\"text/css\"/>\n")
	}

	for _, asset := range chapterAssets {
		b.WriteString(fmt.Sprintf("    <item id=\"%s\" href=\"%s\" media-type=\"%s\"/>\n",
			utils.EscapeXML(asset.ID), utils.EscapeXML(asset.Filename), utils.EscapeXML(asset.MediaType)))
	}

	for i, ch := range chapters {
		b.WriteString(fmt.Sprintf("    <item id=\"ch%d\" href=\"%s\" media-type=\"application/xhtml+xml\"/>\n",
			i, ch.Filename))
	}

	b.WriteString("  </manifest>\n  <spine>\n")
	if g.meta.IncludeCover {
		b.WriteString("    <itemref idref=\"cover\"/>\n")
	}
	for i := range chapters {
		b.WriteString(fmt.Sprintf("    <itemref idref=\"ch%d\"/>\n", i))
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
    <meta name="dtb:uid" content="urn:mdpress"/>
  </head>
  <docTitle><text>`)
	b.WriteString(utils.EscapeXML(g.meta.Title))
	b.WriteString(`</text></docTitle>
  <navMap>
`)
	playOrder := 1
	if g.meta.IncludeCover {
		b.WriteString(fmt.Sprintf("    <navPoint id=\"nav-cover\" playOrder=\"%d\">\n", playOrder))
		b.WriteString("      <navLabel><text>Cover</text></navLabel>\n")
		b.WriteString("      <content src=\"cover.xhtml\"/>\n")
		b.WriteString("    </navPoint>\n")
		playOrder++
	}
	for i, ch := range chapters {
		b.WriteString(fmt.Sprintf("    <navPoint id=\"nav%d\" playOrder=\"%d\">\n", i, playOrder))
		b.WriteString(fmt.Sprintf("      <navLabel><text>%s</text></navLabel>\n", utils.EscapeXML(ch.Title)))
		b.WriteString(fmt.Sprintf("      <content src=\"%s\"/>\n", ch.Filename))
		b.WriteString("    </navPoint>\n")
		playOrder++
	}
	b.WriteString("  </navMap>\n</ncx>\n")
	return b.String()
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
	for _, ch := range chapters {
		b.WriteString(fmt.Sprintf("      <li><a href=\"%s\">%s</a></li>\n", utils.EscapeXML(ch.Filename), utils.EscapeXML(ch.Title)))
	}
	b.WriteString(`    </ol>
  </nav>
</body>
</html>
`)
	return b.String()
}

// wrapXHTML wraps HTML body content into a complete XHTML document.
func (g *EpubGenerator) wrapXHTML(title, body string) string {
	var b strings.Builder
	body = normalizeHTMLForXHTML(body)
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="`)
	b.WriteString(utils.EscapeXML(languageOrDefault(g.meta.Language)))
	b.WriteString(`">
<head>
  <meta charset="UTF-8" />
`)
	b.WriteString(fmt.Sprintf("  <title>%s</title>\n", utils.EscapeXML(title)))
	if g.css != "" {
		b.WriteString("  <link rel=\"stylesheet\" type=\"text/css\" href=\"style.css\"/>\n")
	}
	b.WriteString("</head>\n<body>\n")
	b.WriteString(body)
	b.WriteString("\n</body>\n</html>\n")
	return b.String()
}

// generateCoverPage emits a lightweight generated title page for EPUB readers.
func (g *EpubGenerator) generateCoverPage(coverAsset *epubAsset) string {
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
      font-family: serif;
      display: flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(160deg, #f8fafc, #e2e8f0);
      color: #0f172a;
      text-align: center;
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
      box-shadow: 0 18px 50px rgba(15, 23, 42, 0.22);
      border-radius: 0.4rem;
      background: #fff;
    }
    .title { font-size: 2.2rem; font-weight: 700; line-height: 1.2; margin: 0; }
    .subtitle { font-size: 1.1rem; color: #475569; margin: 1rem 0 0; }
    .meta { margin-top: 2.5rem; color: #334155; }
    .meta div + div { margin-top: 0.45rem; }
    .version { color: #64748b; }
  </style>
</head>
<body>
  <section class="cover" epub:type="cover">
`)
	if coverAsset != nil {
		b.WriteString("    <div class=\"cover-image-wrap\">\n")
		b.WriteString(fmt.Sprintf("      <img class=\"cover-image\" src=\"%s\" alt=\"%s\" />\n",
			utils.EscapeXML(coverAsset.Filename), utils.EscapeXML(g.meta.Title)))
		b.WriteString("    </div>\n")
	}
	b.WriteString(fmt.Sprintf("    <h1 class=\"title\">%s</h1>\n", utils.EscapeXML(g.meta.Title)))
	if g.meta.Subtitle != "" {
		b.WriteString(fmt.Sprintf("    <p class=\"subtitle\">%s</p>\n", utils.EscapeXML(g.meta.Subtitle)))
	}
	b.WriteString("    <div class=\"meta\">\n")
	if g.meta.Author != "" {
		b.WriteString(fmt.Sprintf("      <div>%s</div>\n", utils.EscapeXML(g.meta.Author)))
	}
	if g.meta.Version != "" {
		b.WriteString(fmt.Sprintf("      <div class=\"version\">Version %s</div>\n", utils.EscapeXML(g.meta.Version)))
	}
	b.WriteString("    </div>\n")
	b.WriteString("  </section>\n</body>\n</html>\n")
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
	_, err = fw.Write([]byte(content))
	return err
}

func writeZipBinaryFile(w *zip.Writer, name string, data []byte) error {
	fw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", name, err)
	}
	_, err = fw.Write(data)
	return err
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
	data, err := os.ReadFile(coverPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read EPUB cover image %q: %w", coverPath, err)
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
		Filename:  filepath.ToSlash(filepath.Join("assets", "cover"+ext)),
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
	// The trailing group (\s|/?>) in booleanAttrPattern consumes the whitespace
	// that separates adjacent boolean attributes, so a single pass misses the
	// second attribute in sequences like `disabled multiple`. Running the
	// replacement twice ensures all consecutive boolean attributes are expanded.
	expandBoolAttr := func(match string) string {
		sub := booleanAttrPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		attr := strings.ToLower(sub[1])
		trailing := sub[2]
		return " " + attr + `="` + attr + `"` + trailing
	}
	html = booleanAttrPattern.ReplaceAllStringFunc(html, expandBoolAttr)
	html = booleanAttrPattern.ReplaceAllStringFunc(html, expandBoolAttr)

	return html
}

var epubImageSrcPattern = regexp.MustCompile(`<img\s+([^>]*\s+)?src=["']([^"']+)["']([^>]*)>`)
var dataURIImagePattern = regexp.MustCompile(`^data:([^;,]+);base64,(.+)$`)

func (g *EpubGenerator) collectChapterAssets() ([]EpubChapter, []*epubAsset, error) {
	chapters := make([]EpubChapter, len(g.chapters))
	copy(chapters, g.chapters)

	assets := make([]*epubAsset, 0)
	remoteTempDir, err := os.MkdirTemp("", "mdpress-epub-assets-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create temporary EPUB asset directory: %w", err)
	}
	defer os.RemoveAll(remoteTempDir)
	collector := &epubAssetCollector{
		cache: make(map[string]*epubAsset),
	}

	for i := range chapters {
		updated, chapterAssets, err := collectImageAssetsFromHTML(chapters[i].HTML, chapters[i].SourceDir, remoteTempDir, collector)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to collect EPUB assets for chapter %q: %w", chapters[i].Title, err)
		}
		chapters[i].HTML = updated
		assets = append(assets, chapterAssets...)
	}

	return chapters, assets, nil
}

func collectImageAssetsFromHTML(html string, sourceDir string, remoteTempDir string, collector *epubAssetCollector) (string, []*epubAsset, error) {
	var collectErr error
	assets := make([]*epubAsset, 0)

	updated := epubImageSrcPattern.ReplaceAllStringFunc(html, func(match string) string {
		if collectErr != nil {
			return match
		}

		matches := epubImageSrcPattern.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		src := matches[2]
		prefix := matches[1]
		suffix := matches[3]

		asset, created, err := collector.assetForSource(src, sourceDir, remoteTempDir)
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

func (c *epubAssetCollector) assetForSource(src string, sourceDir string, remoteTempDir string) (*epubAsset, bool, error) {
	key, asset, err := buildImageAssetFromSource(src, sourceDir, remoteTempDir, c.nextIndex)
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

func buildImageAssetFromSource(src string, sourceDir string, remoteTempDir string, index int) (string, *epubAsset, error) {
	if strings.HasPrefix(src, "data:") {
		asset, err := buildDataURIImageAsset(src, index)
		return "data:" + src, asset, err
	}
	if utils.IsRemoteURL(src) {
		asset, err := buildRemoteImageAsset(src, remoteTempDir, index)
		return "remote:" + src, asset, err
	}
	if filepath.IsAbs(src) {
		asset, err := buildFileImageAsset(src, index)
		return "file:" + filepath.Clean(src), asset, err
	}
	if sourceDir != "" && src != "" && !strings.HasPrefix(src, "#") {
		resolved := filepath.Clean(filepath.Join(sourceDir, filepath.FromSlash(src)))
		asset, err := buildFileImageAsset(resolved, index)
		return "file:" + resolved, asset, err
	}
	return "", nil, nil
}

func buildDataURIImageAsset(src string, index int) (*epubAsset, error) {
	matches := dataURIImagePattern.FindStringSubmatch(src)
	if len(matches) != 3 {
		return nil, fmt.Errorf("unsupported data URI image format")
	}

	mediaType := strings.TrimSpace(matches[1])
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

func buildFileImageAsset(src string, index int) (*epubAsset, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("read image file %q: %w", src, err)
	}

	mediaType := utils.DetectImageMIME(src, data)
	return buildImageAssetFromBytes(data, src, mediaType, index), nil
}

func buildRemoteImageAsset(src string, remoteTempDir string, index int) (*epubAsset, error) {
	if remoteTempDir == "" {
		return nil, fmt.Errorf("temporary directory for remote EPUB assets is not available")
	}

	localPath, err := utils.DownloadImage(src, remoteTempDir)
	if err != nil {
		return nil, fmt.Errorf("download remote image %q: %w", src, err)
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read downloaded remote image %q: %w", src, err)
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
	switch strings.ToLower(mediaType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	default:
		return ""
	}
}
