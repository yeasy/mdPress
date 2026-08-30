package renderer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/katex"
)

// mathSpanFixture is the markup internal/markdown emits for one inline and one
// display formula, HTML-escaped exactly the way the pipeline escapes it.
const mathSpanFixture = `<p>Inline <span class="math math-inline">$E = mc^2$</span> energy.</p>` +
	`<p><span class="math math-display">$$\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}$$</span></p>`

var fontFaceBlock = regexp.MustCompile(`(?s)@font-face\{.*?\}`)

// TestInlineKaTeXFontsLeavesNoRelativeReference is the property that makes the
// single-file HTML honest: after the rewrite there is nothing left for a
// browser to fetch. A surviving url(fonts/…) is a glyph that silently falls
// back to the system serif in every copy of the document.
func TestInlineKaTeXFontsLeavesNoRelativeReference(t *testing.T) {
	css, err := katexInlineStylesheet()
	if err != nil {
		t.Fatalf("katexInlineStylesheet: %v", err)
	}

	if strings.Contains(css, "url(fonts/") {
		t.Error("stylesheet still references fonts by relative path")
	}
	if strings.Contains(css, ".woff2)") || strings.Contains(css, ".ttf)") {
		t.Error("stylesheet still references a font file rather than a data URI")
	}

	// Every @font-face must resolve, and the count must match the fonts that
	// actually ship — a rewrite that quietly dropped half the faces would pass
	// the two checks above.
	assets, err := katex.Assets()
	if err != nil {
		t.Fatalf("katex.Assets: %v", err)
	}
	wantFonts := 0
	for _, a := range assets {
		if strings.HasPrefix(a.Path, "fonts/") {
			wantFonts++
		}
	}
	if wantFonts == 0 {
		t.Fatal("the vendored KaTeX distribution packages no fonts")
	}

	faces := fontFaceBlock.FindAllString(css, -1)
	if len(faces) != wantFonts {
		t.Errorf("got %d @font-face rules for %d packaged fonts", len(faces), wantFonts)
	}
	for _, face := range faces {
		if !strings.Contains(face, "url(data:font/woff2;base64,") {
			t.Errorf("@font-face does not resolve to a data URI: %.120s", face)
		}
	}
	if got := strings.Count(css, "url(data:font/woff2;base64,"); got != wantFonts {
		t.Errorf("got %d embedded fonts, want %d", got, wantFonts)
	}
}

// TestInlineKaTeXFontsRejectsAnUnresolvableReference pins the build error. A
// font the packager forgot must stop the build; a warning would ship a
// document that looks fine to whoever built it and wrong to every reader.
func TestInlineKaTeXFontsRejectsAnUnresolvableReference(t *testing.T) {
	_, err := inlineKaTeXFonts(
		"@font-face{font-family:KaTeX_AMS;src:url(fonts/KaTeX_Missing.woff2) format(\"woff2\")}",
		[]katex.Asset{{Path: "fonts/KaTeX_AMS-Regular.woff2", MediaType: "font/woff2", Data: []byte("x")}},
	)
	if err == nil {
		t.Fatal("expected an unresolvable font reference to be a build error")
	}
	if !strings.Contains(err.Error(), "fonts/KaTeX_Missing.woff2") {
		t.Errorf("error should name the missing font, got: %v", err)
	}
}

// TestRenderMathSpansProducesFinishedKaTeXMarkup checks the build-time render
// itself: no delimiters survive, and KaTeX's MathML twin is present, which is
// only produced when the library really ran.
func TestRenderMathSpansProducesFinishedKaTeXMarkup(t *testing.T) {
	out, errs := renderMathSpans(mathSpanFixture)
	if len(errs) != 0 {
		t.Fatalf("unexpected render errors: %v", errs)
	}
	if strings.Contains(out, `class="math math-`) {
		t.Error("a math span survived the build-time render")
	}
	if got := strings.Count(out, `class="katex`); got == 0 {
		t.Fatal("no KaTeX markup was produced")
	}
	if got := strings.Count(out, "katex-mathml"); got != 2 {
		t.Errorf("got %d MathML twins, want one per formula", got)
	}
	if !strings.Contains(out, `class="katex-display"`) {
		t.Error("the display formula was not rendered in display mode")
	}
	if strings.Contains(out, `\frac`) && !strings.Contains(out, "annotation") {
		t.Error("raw LaTeX survived outside the MathML annotation")
	}
}

// TestRenderMathSpansReportsRejectedFormulas keeps a broken formula from
// failing silently: the build has to say which one KaTeX could not parse.
func TestRenderMathSpansReportsRejectedFormulas(t *testing.T) {
	out, errs := renderMathSpans(`<span class="math math-inline">$\frac{1}{$</span>`)
	if len(errs) == 0 {
		t.Fatal("expected KaTeX to reject the malformed formula")
	}
	// The reader still gets KaTeX's own error rendering rather than nothing.
	if !strings.Contains(out, "katex") {
		t.Errorf("a rejected formula should still produce KaTeX error markup, got: %s", out)
	}
}

// TestRenderMathSpansIgnoresFragmentsWithoutMath guards the fast path that
// keeps the JavaScript interpreter out of every ordinary build.
func TestRenderMathSpansIgnoresFragmentsWithoutMath(t *testing.T) {
	in := `<p>No math here, just a $ sign.</p>`
	out, errs := renderMathSpans(in)
	if out != in || errs != nil {
		t.Errorf("fragment without math was modified: %q", out)
	}
}

// TestStandaloneEmbedsRenderedMathAndFonts is the end-to-end shape of the
// portable file: rendered formulas, an inlined stylesheet, and no KaTeX URL
// anywhere for a browser to try to reach.
func TestStandaloneEmbedsRenderedMathAndFonts(t *testing.T) {
	html := renderStandalone(t, mathSpanFixture)

	if strings.Contains(html, `class="math math-`) {
		t.Error("the document still carries unrendered math spans")
	}
	if !strings.Contains(html, "katex-mathml") {
		t.Error("the document carries no rendered KaTeX markup")
	}
	if !strings.Contains(html, "url(data:font/woff2;base64,") {
		t.Error("the KaTeX fonts were not embedded")
	}
	for _, forbidden := range []string{"katex@", "renderMathInElement", "auto-render.min.js", "url(fonts/"} {
		if strings.Contains(html, forbidden) {
			t.Errorf("the document still depends on external KaTeX: found %q", forbidden)
		}
	}
}

// TestStandaloneWithoutMathCarriesNoKaTeX is the cost check. The KaTeX
// stylesheet is ~390 KB of base64 fonts; a book that never writes a formula
// must not pay for it just because the feature exists.
func TestStandaloneWithoutMathCarriesNoKaTeX(t *testing.T) {
	withoutMath := renderStandalone(t, `<p>Plain prose, no formulas at all.</p>`)

	for _, forbidden := range []string{"KaTeX_", "url(data:font/woff2;base64,", "katex"} {
		if strings.Contains(withoutMath, forbidden) {
			t.Errorf("a book without math should not carry %q", forbidden)
		}
	}

	withMath := renderStandalone(t, mathSpanFixture)
	if len(withMath) <= len(withoutMath) {
		t.Fatalf("sanity: the math document (%d bytes) should be larger than the plain one (%d bytes)",
			len(withMath), len(withoutMath))
	}
	// A few KB of slack would mean the fonts did not actually make it in.
	if grew := len(withMath) - len(withoutMath); grew < 300*1024 {
		t.Errorf("math added only %d bytes; the embedded fonts are missing", grew)
	}
}
