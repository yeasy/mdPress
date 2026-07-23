// margins.go resolves the two places a book can set page margins.
//
// book.yaml has both `style.margin` (millimeters, as numbers) and
// `output.margin_top/bottom/left/right` (CSS lengths, as strings). Only the
// former reached the PDF: the generator asks Chrome to honor the document's
// own @page rule, and @page was built from style.margin alone. The output.*
// keys were parsed, validated, and documented in four places as the way to fix
// cramped output — and did nothing at all.
package renderer

import (
	"strconv"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
)

// pageMargins are the effective page margins in millimeters.
type pageMargins struct {
	Top    float64
	Right  float64
	Bottom float64
	Left   float64
}

// resolveMargins layers output.margin_* over style.margin. A value that cannot
// be parsed leaves that edge on the style.margin value rather than collapsing
// it to zero, so a typo cannot silently produce a borderless page.
func resolveMargins(cfg *config.BookConfig) pageMargins {
	m := pageMargins{
		Top:    cfg.Style.Margin.Top,
		Right:  cfg.Style.Margin.Right,
		Bottom: cfg.Style.Margin.Bottom,
		Left:   cfg.Style.Margin.Left,
	}
	for _, edge := range []struct {
		raw string
		dst *float64
	}{
		{cfg.Output.MarginTop, &m.Top},
		{cfg.Output.MarginRight, &m.Right},
		{cfg.Output.MarginBottom, &m.Bottom},
		{cfg.Output.MarginLeft, &m.Left},
	} {
		if mm, ok := parseLengthMM(edge.raw); ok {
			*edge.dst = mm
		}
	}
	return m
}

// millimetersPerUnit converts the CSS absolute length units to millimeters.
// Relative units (em, %, vh) are deliberately absent: they have no fixed
// meaning for a page box, and guessing one would be worse than ignoring the
// value.
var millimetersPerUnit = map[string]float64{
	"mm": 1,
	"cm": 10,
	"in": 25.4,
	"pt": 25.4 / 72,
	"pc": 25.4 / 6,
	"px": 25.4 / 96,
}

// parseLengthMM converts a CSS absolute length to millimeters. A bare number is
// read as millimeters, matching style.margin's own unit.
func parseLengthMM(raw string) (float64, bool) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return 0, false
	}
	for unit, factor := range millimetersPerUnit {
		if numeric, found := strings.CutSuffix(s, unit); found {
			value, err := strconv.ParseFloat(strings.TrimSpace(numeric), 64)
			if err != nil || value < 0 {
				return 0, false
			}
			return value * factor, true
		}
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}
