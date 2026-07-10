package markdown

// HighlightCSSLight returns the light-mode syntax-highlighting stylesheet for
// the given chroma style name (theme.CodeTheme). It returns an empty string
// until class-based highlighting is enabled by the markdown renderer.
func HighlightCSSLight(codeTheme string) string {
	_ = codeTheme
	return ""
}

// HighlightCSSDark returns the dark-mode syntax-highlighting stylesheet for
// the given chroma style name, with every rule scoped so it only applies under
// [data-theme="dark"]. It returns an empty string until class-based
// highlighting is enabled by the markdown renderer.
func HighlightCSSDark(codeTheme string) string {
	_ = codeTheme
	return ""
}
