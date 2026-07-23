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
