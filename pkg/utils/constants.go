package utils

import (
	"fmt"
	"strings"
)

// MermaidCDNURL is the CDN URL for the Mermaid diagram library.
// Centralized here so every part of the codebase uses the same version.
const MermaidCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"

// KaTeX CDN URLs. Centralized here to keep the version consistent across the codebase.
const (
	// KaTeXCSSURL is the CDN URL for the KaTeX stylesheet.
	KaTeXCSSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/katex.min.css"
	// KaTeXJSURL is the CDN URL for the KaTeX core JavaScript library.
	KaTeXJSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/katex.min.js"
	// KaTeXAutoRenderURL is the CDN URL for the KaTeX auto-render extension, which scans for $...$ and $$...$$ in the document and renders them.
	KaTeXAutoRenderURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/contrib/auto-render.min.js"
)

// MaxHTTPRedirects is the maximum number of HTTP redirects to follow.
// Shared across image downloads, upgrade checks, and PlantUML rendering.
const MaxHTTPRedirects = 10

// cdnReplacer is a pre-built replacer for CDN URL placeholders.
// Promoted to package level to avoid repeated allocation per call.
var cdnReplacer = strings.NewReplacer(
	"{{MERMAID_CDN_URL}}", MermaidCDNURL,
	"{{KATEX_CSS_URL}}", KaTeXCSSURL,
	"{{KATEX_JS_URL}}", KaTeXJSURL,
	"{{KATEX_AUTO_RENDER_URL}}", KaTeXAutoRenderURL,
)

// ResolveCDNPlaceholders replaces CDN URL placeholders in a template string
// with the actual CDN URLs defined in this package.
func ResolveCDNPlaceholders(tmpl string) string {
	return cdnReplacer.Replace(tmpl)
}

// PageDimensions holds page width and height in millimeters.
type PageDimensions struct {
	Width  float64
	Height float64
}

// WidthMM returns the width as a Typst-compatible "Xmm" string.
func (d PageDimensions) WidthMM() string { return fmt.Sprintf("%.0fmm", d.Width) }

// HeightMM returns the height as a Typst-compatible "Xmm" string.
func (d PageDimensions) HeightMM() string { return fmt.Sprintf("%.0fmm", d.Height) }

// Standard page dimensions in millimeters (width x height).
var pageDimensions = map[string]PageDimensions{
	"A4":     {210, 297},
	"A5":     {148, 210},
	"B5":     {176, 250},
	"LETTER": {216, 279},
	"LEGAL":  {216, 356},
}

// GetPageDimensions returns the page dimensions for a named page size.
// If the size is unknown, A4 dimensions are returned.
func GetPageDimensions(size string) PageDimensions {
	if d, ok := pageDimensions[strings.ToUpper(size)]; ok {
		return d
	}
	return pageDimensions["A4"]
}
