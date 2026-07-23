package utils

import (
	"fmt"
	"strings"
)

// Mermaid and KaTeX are fetched from a public CDN when a reader opens a page
// that needs them. Two rules apply to every URL here:
//
//   - Pin an exact version. `mermaid@11` was a floating range, so the bytes a
//     reader executed could change without any mdpress release, and there was
//     no fixed content to hash.
//   - Ship a Subresource Integrity digest alongside it, so a tampered or
//     substituted CDN response is rejected by the browser instead of running.
//
// To move to a new version, change the URL and recompute the digest:
//
//	curl -sSL --compressed <url> | openssl dgst -sha384 -binary | openssl base64 -A
//
// Verify the result against a second mirror (unpkg.com serves the same npm
// tarball) before committing it.
const (
	// MermaidCDNURL is the CDN URL for the Mermaid diagram library.
	// Centralized here so every part of the codebase uses the same version.
	MermaidCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11.16.0/dist/mermaid.min.js"
	// MermaidSRI is the Subresource Integrity digest of MermaidCDNURL.
	MermaidSRI = "sha384-T/0lMUdJpd2S1ZHtRiofG3htU3xPCrFVeAQ1UUE2TJwlEJSV5NUwn30kP28n238E"

	// KaTeXCSSURL is the CDN URL for the KaTeX stylesheet.
	KaTeXCSSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/katex.min.css"
	// KaTeXCSSSRI is the Subresource Integrity digest of KaTeXCSSURL.
	KaTeXCSSSRI = "sha384-irXK0JiCGinqGL+slwVklbhJetrjczNwaP2lANewD8lKAs9n61SbQ3As28iSqXUE"
	// KaTeXJSURL is the CDN URL for the KaTeX core JavaScript library.
	KaTeXJSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/katex.min.js"
	// KaTeXJSSRI is the Subresource Integrity digest of KaTeXJSURL.
	KaTeXJSSRI = "sha384-m/s9umSlhJbqEdA/j7pQVdGCMx2fHf7GXtgCVhNGOwLuu+1qJQES5AzIE8pn3nKQ"
	// KaTeXAutoRenderURL is the CDN URL for the KaTeX auto-render extension, which scans for $...$ and $$...$$ in the document and renders them.
	KaTeXAutoRenderURL = "https://cdn.jsdelivr.net/npm/katex@0.16.44/dist/contrib/auto-render.min.js"
	// KaTeXAutoRenderSRI is the Subresource Integrity digest of KaTeXAutoRenderURL.
	KaTeXAutoRenderSRI = "sha384-bjyGPfbij8/NDKJhSGZNP/khQVgtHUE5exjm4Ydllo42FwIgYsdLO2lXGmRBf5Mz"
)

// MaxHTTPRedirects is the maximum number of HTTP redirects to follow.
// Shared across image downloads, upgrade checks, and PlantUML rendering.
const MaxHTTPRedirects = 10

// cdnReplacer is a pre-built replacer for CDN URL placeholders.
// Promoted to package level to avoid repeated allocation per call.
var cdnReplacer = strings.NewReplacer(
	"{{MERMAID_CDN_URL}}", MermaidCDNURL,
	"{{MERMAID_SRI}}", MermaidSRI,
	"{{KATEX_CSS_URL}}", KaTeXCSSURL,
	"{{KATEX_CSS_SRI}}", KaTeXCSSSRI,
	"{{KATEX_JS_URL}}", KaTeXJSURL,
	"{{KATEX_JS_SRI}}", KaTeXJSSRI,
	"{{KATEX_AUTO_RENDER_URL}}", KaTeXAutoRenderURL,
	"{{KATEX_AUTO_RENDER_SRI}}", KaTeXAutoRenderSRI,
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
