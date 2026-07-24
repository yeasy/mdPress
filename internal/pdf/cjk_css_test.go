package pdf

import (
	"regexp"
	"strings"
	"testing"
)

// declarationFor returns the single-line font-family declaration of the first
// rule whose selector list starts with selector.
func declarationFor(t *testing.T, css, selector string) string {
	t.Helper()
	idx := strings.Index(css, selector+" {")
	if idx == -1 {
		t.Fatalf("no %q rule in generated CSS:\n%s", selector, css)
	}
	body := css[idx:]
	end := strings.Index(body, "}")
	if end == -1 {
		t.Fatalf("unterminated %q rule in generated CSS:\n%s", selector, css)
	}
	return strings.Join(strings.Fields(body[:end]), " ")
}

// TestCJKFontFaceCSSKeepsThemeFontFamily guards style.font_family (and every
// theme's fonts) in PDF output. The injected CJK block is spliced in after the
// document stylesheet, so when it named a literal font stack it replaced the
// requested typography outright: PDFs came out in the browser default sans while
// the site and ePub honored the setting. Deferring to var(--font-family) — the
// property theme.ToCSS() writes — keeps "CJK-Embedded" first for CJK coverage
// without discarding the book's own font.
func TestCJKFontFaceCSSKeepsThemeFontFamily(t *testing.T) {
	css := cjkFontFaceCSS(cjkFontSource{path: "/System/Library/Fonts/PingFang.ttc"})

	tests := []struct {
		selector string
		property string
	}{
		{selector: "body", property: "--font-family"},
		{selector: "code, pre, kbd, samp, .hljs", property: "--font-family-mono"},
	}
	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			decl := declarationFor(t, css, tt.selector)
			want := `font-family: "CJK-Embedded", var(` + tt.property + `,`
			if !strings.Contains(decl, want) {
				t.Errorf("rule %q must defer to the theme font, got:\n%s", tt.selector, decl)
			}
			// The var() fallback still has to close before !important, or the
			// whole declaration is dropped as invalid.
			if !strings.Contains(decl, ") !important;") {
				t.Errorf("rule %q must close var() before !important, got:\n%s", tt.selector, decl)
			}
			// Nothing may sit between "CJK-Embedded" and the var(): a literal
			// family there wins over the theme for every non-CJK character,
			// which is exactly the bug.
			if regexp.MustCompile(`font-family: "CJK-Embedded", [^v]`).MatchString(decl) {
				t.Errorf("rule %q pins a literal family ahead of the theme font, got:\n%s", tt.selector, decl)
			}
		})
	}
}

// TestCJKFontFaceCSSStillCoversCJK pins the parts the font-family change must
// not disturb: the embeddable alias, its CJK-only unicode-range, and the /cjk-font
// URL the local font server answers.
func TestCJKFontFaceCSSStillCoversCJK(t *testing.T) {
	css := cjkFontFaceCSS(cjkFontSource{path: "/System/Library/Fonts/PingFang.ttc"})

	for _, want := range []string{
		`font-family: "CJK-Embedded";`,
		`url("/cjk-font") format("collection")`,
		"unicode-range: U+2E80-2EFF",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("generated CSS missing %q:\n%s", want, css)
		}
	}
	if !strings.HasPrefix(declarationFor(t, css, "body"), `body { font-family: "CJK-Embedded",`) {
		t.Error("CJK-Embedded must stay first in the body stack so CJK glyphs are embeddable")
	}
}
