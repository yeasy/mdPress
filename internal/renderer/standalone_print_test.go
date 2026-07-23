package renderer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// renderStandalone renders a one-chapter document with the given content.
func renderStandalone(tb testing.TB, content string) string {
	tb.Helper()
	r, err := NewStandaloneHTMLRenderer(newTestConfig(), newTestTheme(tb))
	if err != nil {
		tb.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: content}},
	})
	if err != nil {
		tb.Fatalf("Render failed: %v", err)
	}
	return html
}

// printCSSBlock returns the body of the @media print block of a rendered
// document, so print-only assertions cannot be satisfied by screen rules.
func printCSSBlock(tb testing.TB, html string) string {
	tb.Helper()
	start := strings.Index(html, "@media print {")
	if start < 0 {
		tb.Fatal("rendered document has no @media print block")
	}
	depth := 0
	for i := start; i < len(html); i++ {
		switch html[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return html[start : i+1]
			}
		}
	}
	tb.Fatal("@media print block is unterminated")
	return ""
}

// TestStandalonePrintCSSPaginatesChapters guards printing a standalone
// document: each chapter must start on a fresh sheet, but a forced break in
// front of the first one would print a blank opening page.
func TestStandalonePrintCSSPaginatesChapters(t *testing.T) {
	printCSS := printCSSBlock(t, renderStandalone(t, "<p>a</p>"))

	if !regexp.MustCompile(`\.chapter\s*\{[^}]*page-break-before:\s*always`).MatchString(printCSS) {
		t.Error("print CSS does not start each chapter on a new page")
	}
	if !regexp.MustCompile(`\.chapter:first-of-type\s*\{[^}]*page-break-before:\s*auto`).MatchString(printCSS) {
		t.Error("print CSS forces a break before the first chapter, which prints a blank opening page")
	}
}

// TestStandalonePrintCSSWrapsCode guards printing long code lines: paper has
// no horizontal scrollbar, so a `white-space: pre` code block loses everything
// past the right margin.
func TestStandalonePrintCSSWrapsCode(t *testing.T) {
	printCSS := printCSSBlock(t, renderStandalone(t, "<pre><code>x</code></pre>"))

	rule := regexp.MustCompile(`(?s)pre,\s*pre code,[^{]*\{(.*?)\}`).FindStringSubmatch(printCSS)
	if rule == nil {
		t.Fatal("print CSS has no rule covering pre / pre code")
	}
	// The web layout re-assertions are appended after the print block, so a
	// plain declaration here would lose the cascade.
	for _, want := range []string{"white-space: pre-wrap !important", "overflow: visible !important"} {
		if !strings.Contains(rule[1], want) {
			t.Errorf("print CSS code rule is missing %q", want)
		}
	}
}

// TestStandalonePrintCSSOnlyAnnotatesExternalLinks guards against printing
// "(#fn:1)" after every footnote marker and cross-reference: only links that
// leave the document are worth spelling out on paper.
func TestStandalonePrintCSSOnlyAnnotatesExternalLinks(t *testing.T) {
	printCSS := printCSSBlock(t, renderStandalone(t, `<p><a href="#fn:1">1</a></p>`))

	if regexp.MustCompile(`a\[href\]::after`).MatchString(printCSS) {
		t.Error(`print CSS still appends the href of every link, so in-page anchors print as "(#fn:1)"`)
	}
	if !strings.Contains(printCSS, `a[href^="http"]::after`) {
		t.Error("print CSS no longer spells out external link targets")
	}
	if !regexp.MustCompile(`a\[href\^="#"\]::after\s*\{[^}]*content:\s*none`).MatchString(printCSS) {
		t.Error("print CSS does not suppress the href suffix for in-page anchors")
	}
}

// TestStandaloneThemeIconsAreStroked guards the toolbar theme toggle: the sun
// and monitor glyphs are line drawings, and filling them paints a solid black
// lozenge where the icon should be.
func TestStandaloneThemeIconsAreStroked(t *testing.T) {
	html := renderStandalone(t, "<p>a</p>")

	for _, glyph := range []struct{ name, marker string }{
		{"light", `<circle cx="12" cy="12" r="5"/>`},
		{"system", `<rect x="2" y="3" width="20" height="14" rx="2"/>`},
	} {
		idx := strings.Index(html, glyph.marker)
		if idx < 0 {
			t.Fatalf("%s theme icon not found in the rendered document", glyph.name)
		}
		svg := html[strings.LastIndex(html[:idx], "<svg"):idx]
		if !strings.Contains(svg, `fill="none"`) || !strings.Contains(svg, `stroke="currentColor"`) {
			t.Errorf("%s theme icon is filled instead of stroked: %s", glyph.name, svg)
		}
	}
}

// TestStandaloneCoverHonorsBookCover guards book.cover.image and
// book.cover.background, which reach the PDF and EPUB covers but used to be
// dropped on the floor by --format html.
func TestStandaloneCoverHonorsBookCover(t *testing.T) {
	dir := t.TempDir()
	// 1x1 transparent GIF: small enough to inline, real enough to sniff.
	gif := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff!\xf9\x04\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	if err := os.WriteFile(filepath.Join(dir, "cover.gif"), gif, 0o644); err != nil {
		t.Fatalf("failed to write cover image: %v", err)
	}

	cfg := newTestConfig()
	cfg.SetBaseDir(dir)
	cfg.Book.Cover.Image = "cover.gif"
	cfg.Book.Cover.Background = "#1a1a2e"

	r, err := NewStandaloneHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		CoverHTML:    "<html>cover</html>",
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<p>a</p>"}},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, `<img class="cover-hero-image" src="data:image/gif;base64,`) {
		t.Error("book.cover.image was not embedded into the cover hero")
	}
	if !strings.Contains(html, "--color-cover-bg: #1a1a2e") {
		t.Error("book.cover.background did not reach the cover hero background")
	}
}

// TestStandaloneCoverBackgroundRejectsInjection: the background value is
// inlined into the document stylesheet, so it must not be able to close the
// declaration and add rules of its own.
func TestStandaloneCoverBackgroundRejectsInjection(t *testing.T) {
	cfg := newTestConfig()
	cfg.Book.Cover.Background = "red; } body { display: none"

	r, err := NewStandaloneHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		CoverHTML:    "<html>cover</html>",
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<p>a</p>"}},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(html, "body { display: none") {
		t.Error("a malformed cover background was inlined verbatim into the stylesheet")
	}
}

// TestStandaloneMissingCoverImageStillRenders: an unreadable cover image must
// not break the build or leave a dangling reference in a self-contained file.
func TestStandaloneMissingCoverImageStillRenders(t *testing.T) {
	cfg := newTestConfig()
	cfg.SetBaseDir(t.TempDir())
	cfg.Book.Cover.Image = "nope.png"

	r, err := NewStandaloneHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		CoverHTML:    "<html>cover</html>",
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<p>a</p>"}},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(html, `<img class="cover-hero-image"`) {
		t.Error("a missing cover image produced an <img> with no source")
	}
	if !strings.Contains(html, "cover-hero-title") {
		t.Error("the cover hero disappeared along with the missing image")
	}
}
