// layout.go resolves the page geometry a paginated build uses.
//
// Page size and margins can come from three places: the theme file, book.yaml's
// `style` block, and book.yaml's `output.margin_*` keys. Only the last two ever
// reached a renderer. DefaultConfig pre-fills style.page_size ("A4") and
// style.margin (25/25/20/20), and every consumer reads those config fields, so
// a themes/<name>.yaml declaring `page_size: A5` and wider `margins:` — values
// theme validation *requires* and `mdpress themes show` advertises — changed
// nothing at all in the finished PDF.
package config

// ApplyThemeLayout fills in page geometry the project did not configure, using
// the theme's own values. book.yaml's `style` block still wins wherever it
// actually said something; ApplyTypography does the same for fonts.
//
// Margins are taken as a block: all-zero margins are what a theme file that
// omits the `margins:` key yields, and theme.ToCSS already reads that as
// "unset" rather than as an edge-to-edge page. Individual edges are still
// honored one by one, so `style: {margin: {top: 5}}` keeps the theme's other
// three sides.
func (c *BookConfig) ApplyThemeLayout(pageSize string, margins MarginConfig) {
	if pageSize != "" && !c.IsSet("style.page_size") {
		c.Style.PageSize = pageSize
	}

	if margins == (MarginConfig{}) {
		return
	}
	for _, edge := range []struct {
		key  string
		from float64
		to   *float64
	}{
		{"style.margin.top", margins.Top, &c.Style.Margin.Top},
		{"style.margin.bottom", margins.Bottom, &c.Style.Margin.Bottom},
		{"style.margin.left", margins.Left, &c.Style.Margin.Left},
		{"style.margin.right", margins.Right, &c.Style.Margin.Right},
	} {
		if !c.IsSet(edge.key) {
			*edge.to = edge.from
		}
	}
}
