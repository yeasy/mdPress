// katex_inline.go turns the vendored KaTeX distribution into something a
// single-file HTML document can carry: a stylesheet whose fonts are data
// URIs rather than sibling files. The formulas themselves are typeset by
// internal/katex, which every self-contained format shares.
//
// The standalone format is advertised as portable — one file you can mail to
// someone. It used to fetch KaTeX from a CDN at view time, so a reader with no
// network got raw LaTeX source in the one format whose entire selling point is
// being self-contained. Rendering here, in Go, means the shipped document
// needs neither the network nor JavaScript to show a formula.
package renderer

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/internal/katex"
)

// katexFontURL matches a font reference in the vendored stylesheet. Stylesheet()
// has already stripped the WOFF and TrueType sources, so only the WOFF2 files
// that actually ship are left to resolve.
var katexFontURL = regexp.MustCompile(`url\((fonts/[^)]+)\)`)

// katexInlineStylesheet returns the vendored KaTeX stylesheet with every font
// embedded as a data URI, ready to drop into a <style> block.
func katexInlineStylesheet() (string, error) {
	css, err := katex.Stylesheet()
	if err != nil {
		return "", err
	}
	assets, err := katex.Assets()
	if err != nil {
		return "", err
	}
	return inlineKaTeXFonts(css, assets)
}

// inlineKaTeXFonts rewrites `url(fonts/KaTeX_*.woff2)` into
// `url(data:font/woff2;base64,…)` using the packaged assets.
//
// A reference that does not resolve is a build error, not a warning: the whole
// point of this format is that nothing is fetched later, so a font left as a
// relative path is a glyph that silently falls back to the system serif in
// every copy of the document that ever gets mailed anywhere.
func inlineKaTeXFonts(css string, assets []katex.Asset) (string, error) {
	byPath := make(map[string]katex.Asset, len(assets))
	for _, a := range assets {
		byPath[a.Path] = a
	}

	var missing []string
	out := katexFontURL.ReplaceAllStringFunc(css, func(match string) string {
		ref := katexFontURL.FindStringSubmatch(match)[1]
		asset, ok := byPath[ref]
		if !ok {
			missing = append(missing, ref)
			return match
		}
		mediaType := asset.MediaType
		if mediaType == "" {
			mediaType = "font/woff2"
		}
		return "url(data:" + mediaType + ";base64," +
			base64.StdEncoding.EncodeToString(asset.Data) + ")"
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("vendored KaTeX stylesheet references fonts that are not packaged: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}
