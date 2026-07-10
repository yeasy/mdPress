// highlight_css.go generates class-based chroma stylesheets for syntax
// highlighting. The parser renders code blocks with CSS classes instead of
// inline styles (see parser.go), so each output format embeds these light and
// dark stylesheets and code stays readable in both color modes.
package markdown

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// defaultCodeTheme is the chroma style used when the configured code theme is
// empty or not a registered chroma style (e.g. the legacy "default" value).
const defaultCodeTheme = "github"

// DarkModeSelectors lists the root selectors under which dark-mode highlight
// rules apply. They must stay in sync with the renderers: the standalone HTML
// renderer marks dark mode with data-theme="dark" on <html>, and the site
// template toggles the "dark" class on <html>. Renderer CSS that needs to
// override highlight rules in dark mode should use the same prefixes so
// specificity stays predictable.
var DarkModeSelectors = []string{`html[data-theme="dark"]`, `html.dark`}

// resolveCodeTheme maps a configured code theme name to a registered chroma
// style name, falling back to defaultCodeTheme for unknown names so that the
// parser and the generated stylesheets always agree on the style.
func resolveCodeTheme(codeTheme string) string {
	name := strings.ToLower(strings.TrimSpace(codeTheme))
	if _, ok := styles.Registry[name]; ok {
		return name
	}
	return defaultCodeTheme
}

// darkCodeTheme returns the dark-mode counterpart of a configured code theme:
// styles that are already dark (monokai, dracula) keep themselves, everything
// else pairs with github-dark.
func darkCodeTheme(codeTheme string) string {
	switch resolveCodeTheme(codeTheme) {
	case "monokai":
		return "monokai"
	case "dracula":
		return "dracula"
	default:
		return "github-dark"
	}
}

// HighlightCSSLight returns the light-mode syntax-highlighting stylesheet for
// the given chroma style name (theme.CodeTheme), scoped to .chroma. Unknown
// style names fall back to defaultCodeTheme.
func HighlightCSSLight(codeTheme string) string {
	return scopeChromaRules(chromaCSS(resolveCodeTheme(codeTheme)), nil)
}

// HighlightCSSDark returns the dark-mode syntax-highlighting stylesheet for
// the given chroma style name, generated from its dark counterpart style with
// every rule prefixed by DarkModeSelectors so it only applies in dark mode.
//
// A catch-all token rule is prepended: chroma styles only emit rules for the
// token classes they color, so a token styled by the LIGHT stylesheet but not
// by the dark one (e.g. github styles Name ink-dark, github-dark leaves it
// unstyled) would otherwise keep its light ink on the dark background. The
// catch-all (prefix .chroma span) outranks the light token rules (0,2,0) but
// yields to the dark style's own class rules (0,3,1).
func HighlightCSSDark(codeTheme string) string {
	dark := darkCodeTheme(codeTheme)
	var b strings.Builder
	if base := darkBaseTextColor(dark); base != "" {
		scoped := make([]string, len(DarkModeSelectors))
		for i, prefix := range DarkModeSelectors {
			scoped[i] = prefix + " .chroma span"
		}
		b.WriteString(strings.Join(scoped, ", "))
		b.WriteString(" { color: ")
		b.WriteString(base)
		b.WriteString(" }\n")
	}
	b.WriteString(scopeChromaRules(chromaCSS(dark), DarkModeSelectors))
	return b.String()
}

// darkBaseTextColor returns the base text color of a chroma style, used for
// tokens the style leaves uncolored.
func darkBaseTextColor(styleName string) string {
	style := styles.Get(styleName)
	if entry := style.Get(chroma.Text); entry.Colour.IsSet() { //nolint:misspell // Colour is chroma's API name
		return entry.Colour.String() //nolint:misspell // Colour is chroma's API name
	}
	return ""
}

// chromaCSS renders the class-based stylesheet for a registered chroma style
// and strips the style's mode class (".chroma.light" -> ".chroma") so the
// rules match the parser's output regardless of the style's light/dark mode.
func chromaCSS(styleName string) string {
	style := styles.Get(styleName)
	formatter := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.WithCSSComments(false),
	)
	var buf bytes.Buffer
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return ""
	}
	mode := "." + style.Mode().String()
	css := buf.String()
	css = strings.ReplaceAll(css, ".chroma"+mode, ".chroma")
	css = strings.ReplaceAll(css, ".bg"+mode, ".bg")
	return css
}

// scopeChromaRules keeps only the .chroma-scoped rules from css (dropping the
// standalone-page ".bg" helper rules) and, when prefixes is non-empty,
// re-emits each rule under every prefix so it only applies below those roots.
// It relies on chroma's WriteCSS emitting one "SELECTOR { styles }" rule per
// line, which is stable for the pinned chroma version.
func scopeChromaRules(css string, prefixes []string) string {
	var b strings.Builder
	for _, line := range strings.Split(css, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, ".chroma") {
			continue
		}
		if len(prefixes) == 0 {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		selector, rest, ok := strings.Cut(line, " {")
		if !ok {
			continue
		}
		scoped := make([]string, len(prefixes))
		for i, prefix := range prefixes {
			scoped[i] = prefix + " " + selector
		}
		b.WriteString(strings.Join(scoped, ", "))
		b.WriteString(" {")
		b.WriteString(rest)
		b.WriteByte('\n')
	}
	return b.String()
}
