package renderer

import (
	"math"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestParseLengthMM(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"20mm", 20, true},
		{"2.5cm", 25, true},
		{"1in", 25.4, true},
		{"72pt", 25.4, true},
		{"96px", 25.4, true},
		{"  30mm  ", 30, true},
		{"30MM", 30, true},
		{"18", 18, true}, // bare number matches style.margin's unit
		{"", 0, false},
		{"wide", 0, false},
		{"-5mm", 0, false},
		{"2em", 0, false}, // relative units have no fixed page meaning
		{"50%", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseLengthMM(tt.in)
		// Unit conversion is floating point (72pt -> 25.400000000000002), so
		// compare within a tolerance far below a printable difference.
		if ok != tt.ok || (ok && math.Abs(got-tt.want) > 1e-9) {
			t.Errorf("parseLengthMM(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

// TestResolveMarginsLayersOutputOverStyle covers a setting that was parsed,
// validated and documented in four places as the way to fix cramped output, and
// did nothing: the PDF's @page rule was built from style.margin alone.
func TestResolveMarginsLayersOutputOverStyle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Style.Margin = config.MarginConfig{Top: 25, Right: 20, Bottom: 25, Left: 20}

	t.Run("style.margin is the base", func(t *testing.T) {
		if got := resolveMargins(cfg); got != (pageMargins{Top: 25, Right: 20, Bottom: 25, Left: 20}) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("output.margin_* overrides per edge", func(t *testing.T) {
		cfg.Output.MarginLeft = "60mm"
		cfg.Output.MarginTop = "3cm"
		got := resolveMargins(cfg)
		if got.Left != 60 || got.Top != 30 {
			t.Errorf("overrides not applied: %+v", got)
		}
		if got.Right != 20 || got.Bottom != 25 {
			t.Errorf("unset edges should keep style.margin: %+v", got)
		}
	})

	t.Run("an unparseable value keeps style.margin", func(t *testing.T) {
		cfg.Output.MarginRight = "wide"
		if got := resolveMargins(cfg); got.Right != 20 {
			t.Errorf("a typo collapsed the margin to %v; a borderless page must not be the failure mode", got.Right)
		}
	})
}

// TestPageCSSUsesResolvedMargins pins the rule the browser actually reads.
func TestPageCSSUsesResolvedMargins(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.Margin = config.MarginConfig{Top: 25, Right: 20, Bottom: 25, Left: 20}
	cfg.Output.MarginLeft = "60mm"

	r, err := NewHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(&RenderParts{ChaptersHTML: []ChapterHTML{{Title: "A", ID: "a", Content: "<p>x</p>"}}})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if !strings.Contains(html, "margin: 25mm 20mm 25mm 60mm;") {
		t.Error("the generated @page rule does not carry the resolved margins")
	}
}

// TestPageBoxIsTheOnlyMargin covers a book whose margins came out double the
// configured value: theme.ToCSS() emits the page margin as a body margin, and
// @page emits it again, so the two stacked. Books with a cover were spared by
// accident — the cover's embedded stylesheet resets the body margin — so the
// symptom only appeared with output.cover: false.
func TestPageBoxIsTheOnlyMargin(t *testing.T) {
	cfg := newTestConfig()
	cfg.Output.Cover = false
	cfg.Output.MarginLeft = "30mm"
	cfg.Output.MarginRight = "30mm"

	r, err := NewHTMLRenderer(cfg, newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	css := r.buildFullCSS("")

	themeMargin := strings.Index(css, "margin: var(--margin-top")
	reset := strings.Index(css, "body {\n  margin: 0;\n}")
	if themeMargin < 0 {
		t.Fatalf("expected the theme's own body margin in the stylesheet:\n%s", css)
	}
	if reset < 0 || reset < themeMargin {
		t.Error("the body margin must be zeroed after the theme CSS, so @page alone defines the page box")
	}
	// Custom CSS still gets the final say.
	custom := r.buildFullCSS("body { margin: 5mm; }")
	if strings.Index(custom, "body { margin: 5mm; }") < strings.Index(custom, "body {\n  margin: 0;\n}") {
		t.Error("custom CSS should come after the reset so users keep the final say")
	}
}

// TestFirstPageKeepsMarginsWithoutACover pins the other half: @page :first is
// zeroed so cover artwork can bleed to the sheet edge, but with no cover page
// one is the table of contents or the first chapter and was printed edge to
// edge.
func TestFirstPageKeepsMarginsWithoutACover(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.Margin = config.MarginConfig{Top: 25, Right: 20, Bottom: 25, Left: 20}

	t.Run("with a cover the first page bleeds", func(t *testing.T) {
		cfg.Output.Cover = true
		r, err := NewHTMLRenderer(cfg, newTestTheme(t))
		if err != nil {
			t.Fatalf("NewHTMLRenderer failed: %v", err)
		}
		if !strings.Contains(r.buildPrintCSS(), "@page :first {\n  margin: 0;\n}") {
			t.Error("a cover page must have no page box")
		}
	})

	t.Run("without a cover the first page keeps the book's margins", func(t *testing.T) {
		cfg.Output.Cover = false
		r, err := NewHTMLRenderer(cfg, newTestTheme(t))
		if err != nil {
			t.Fatalf("NewHTMLRenderer failed: %v", err)
		}
		if !strings.Contains(r.buildPrintCSS(), "@page :first {\n  margin: 25mm 20mm 25mm 20mm;\n}") {
			t.Errorf("the first page lost its margins:\n%s", r.buildPrintCSS())
		}
	})
}
