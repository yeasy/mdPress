// Package katex vendors the KaTeX distribution so formats that must be
// self-contained can render math without reaching a CDN.
//
// The site and standalone HTML builds load KaTeX from jsDelivr, which is the
// right trade for a page opened in a browser. An EPUB is not that: EPUB 3.3
// expects a publication to carry its resources, a reader is routinely offline,
// and a book whose formulas resolve to raw LaTeX source is simply broken. The
// files here are the upstream release, byte for byte, so refreshing KaTeX is a
// straight copy; the only transformation (dropping the WOFF and TrueType font
// sources the stylesheet also lists) happens in Stylesheet below, at build
// time, so nothing has to be hand-edited on an upgrade.
package katex

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

// Version is the vendored KaTeX release. It must match the version in the CDN
// URLs in pkg/utils/constants.go so a book renders the same math whether it is
// read on the web or out of an EPUB; TestVendoredVersionMatchesCDN pins that.
const Version = "0.16.44"

//go:embed assets
var assets embed.FS

// Asset is one vendored file, ready to be packaged next to the document that
// references it. Path is relative to the directory the caller roots the
// distribution at (the stylesheet's own font URLs assume this layout).
type Asset struct {
	Path      string
	MediaType string
	Data      []byte
}

// nonWOFF2FontSource matches one entry of a CSS `src:` font list pointing at a
// WOFF or TrueType file. `\.woff\)` cannot match a `.woff2)` URL — the digit
// stands where the closing parenthesis must be — so the WOFF2 entry survives.
var nonWOFF2FontSource = regexp.MustCompile(`,?\s*url\([^)]+\.(?:woff|ttf)\)\s*format\("[^"]*"\)`)

// Stylesheet returns katex.min.css with every font source other than WOFF2
// removed.
//
// Upstream lists three formats per face. Shipping all of them would put 1.1 MB
// of duplicate fonts in every book with a formula, and shipping only WOFF2
// while leaving the other two in the stylesheet would leave the publication
// referencing files that are not in it — which is exactly the class of defect
// this package exists to fix. WOFF2 alone is what every EPUB 3 reading system
// that runs KaTeX at all supports.
func Stylesheet() (string, error) {
	data, err := assets.ReadFile("assets/katex.min.css")
	if err != nil {
		return "", fmt.Errorf("read vendored katex.min.css: %w", err)
	}
	css := nonWOFF2FontSource.ReplaceAllString(string(data), "")
	// Guard against a future release that orders the sources differently and
	// leaves a dangling separator behind.
	css = strings.ReplaceAll(css, "src:,", "src:")
	css = strings.ReplaceAll(css, ",,", ",")
	css = strings.ReplaceAll(css, ",}", "}")
	return css, nil
}

// Assets returns every file needed to render math offline: the rewritten
// stylesheet, the two scripts, the WOFF2 fonts, and KaTeX's MIT license, which
// travels with the code it covers.
func Assets() ([]Asset, error) {
	css, err := Stylesheet()
	if err != nil {
		return nil, err
	}
	out := []Asset{{Path: "katex.min.css", MediaType: "text/css", Data: []byte(css)}}

	for _, name := range []string{"katex.min.js", "auto-render.min.js"} {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return nil, fmt.Errorf("read vendored %s: %w", name, err)
		}
		out = append(out, Asset{Path: name, MediaType: "text/javascript", Data: data})
	}

	fonts, err := fs.ReadDir(assets, "assets/fonts")
	if err != nil {
		return nil, fmt.Errorf("read vendored fonts: %w", err)
	}
	for _, entry := range fonts {
		if entry.IsDir() || path.Ext(entry.Name()) != ".woff2" {
			continue
		}
		data, err := assets.ReadFile("assets/fonts/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read vendored font %s: %w", entry.Name(), err)
		}
		out = append(out, Asset{
			Path:      "fonts/" + entry.Name(),
			MediaType: "font/woff2",
			Data:      data,
		})
	}

	license, err := assets.ReadFile("assets/LICENSE")
	if err != nil {
		return nil, fmt.Errorf("read vendored LICENSE: %w", err)
	}
	out = append(out, Asset{Path: "LICENSE", MediaType: "text/plain", Data: license})

	return out, nil
}
