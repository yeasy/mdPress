package renderer

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// imagePrintCapMM extracts the printed image height cap, in millimeters.
func imagePrintCapMM(tb testing.TB, css string) string {
	tb.Helper()
	m := regexp.MustCompile(`(?s)@media print \{\s*img \{(.*?)\}`).FindStringSubmatch(css)
	if m == nil {
		tb.Fatal("print CSS has no image height cap")
	}
	capMM := regexp.MustCompile(`max-height:\s*([0-9.]+)mm`).FindStringSubmatch(m[1])
	if capMM == nil {
		tb.Fatalf("image height cap is not expressed in millimeters: %s", m[1])
	}
	if !strings.Contains(m[1], "object-fit: contain") {
		tb.Error("image height cap does not preserve the aspect ratio")
	}
	return capMM[1]
}

// TestPrintImageCapFitsContentBox: a tall image used to be capped at 85vh,
// which is 85% of the whole sheet and therefore taller than the content box
// on any book with generous margins — the bottom of the image was clipped by
// the page edge. The cap must be derived from the real content box.
func TestPrintImageCapFitsContentBox(t *testing.T) {
	tests := []struct {
		name     string
		pageSize string
		margin   config.MarginConfig
		heightMM float64
	}{
		{"a4 default margins", "A4", config.MarginConfig{Top: 20, Right: 20, Bottom: 20, Left: 20}, 297},
		{"a4 wide margins", "A4", config.MarginConfig{Top: 40, Right: 20, Bottom: 40, Left: 20}, 297},
		{"a5", "A5", config.MarginConfig{Top: 15, Right: 15, Bottom: 15, Left: 15}, 210},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Style.PageSize = tc.pageSize
			cfg.Style.Margin = tc.margin

			r, err := NewHTMLRenderer(cfg, newTestTheme(t))
			if err != nil {
				t.Fatalf("NewHTMLRenderer failed: %v", err)
			}

			got := r.printableImageHeightMM(tc.pageSize)
			contentBox := tc.heightMM - tc.margin.Top - tc.margin.Bottom
			if got > contentBox {
				t.Errorf("image cap %.0fmm exceeds the %.0fmm content box; tall images will be clipped", got, contentBox)
			}
			if got <= 0 {
				t.Errorf("image cap %.0fmm would hide every image", got)
			}
			imagePrintCapMM(t, r.buildPrintCSS())
		})
	}
}

// TestPrintImageCapSurvivesAbsurdMargins: margins wider than the page must
// not produce a zero or negative cap, which would make images disappear.
func TestPrintImageCapSurvivesAbsurdMargins(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "A4"
	cfg.Style.Margin = config.MarginConfig{Top: 200, Right: 20, Bottom: 200, Left: 20}

	r, err := NewHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	if got := r.printableImageHeightMM("A4"); got <= 0 {
		t.Errorf("image cap is %.0fmm; every image would vanish", got)
	}
}

// TestPrintProseHyphenates: justified text without hyphenation opens rivers
// of white space on a book-width measure.
func TestPrintProseHyphenates(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<p>text</p>"}},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	rule := regexp.MustCompile(`(?s)\n    p \{(.*?)\}`).FindStringSubmatch(html)
	if rule == nil {
		t.Fatal("paragraph rule not found in the rendered document")
	}
	if !strings.Contains(rule[1], "text-align: justify") {
		t.Skip("paragraphs are no longer justified; hyphenation is moot")
	}
	if !strings.Contains(rule[1], "hyphens: auto") {
		t.Error("justified paragraphs do not hyphenate")
	}
}

// TestPrintTableCellsDoNotChopWords: table cells set overflow-wrap: anywhere,
// which already breaks a word too wide for its column. Adding word-break also
// broke words that fit, mid-word and without a hyphen.
func TestPrintTableCellsDoNotChopWords(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{
		ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<table><tr><td>x</td></tr></table>"}},
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	for _, sel := range []string{"table th", "table td"} {
		rule := regexp.MustCompile(`(?s)\n    ` + sel + ` \{(.*?)\}`).FindStringSubmatch(html)
		if rule == nil {
			t.Fatalf("%s rule not found in the rendered document", sel)
		}
		if strings.Contains(rule[1], "word-break: break-word") {
			t.Errorf("%s still chops words that fit the column", sel)
		}
		if !strings.Contains(rule[1], "overflow-wrap: anywhere") {
			t.Errorf("%s lost the overflow guard for words wider than the column", sel)
		}
	}
}
