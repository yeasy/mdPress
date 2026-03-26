package utils

// MermaidCDNURL is the CDN URL for the Mermaid diagram library.
// Centralized here so every part of the codebase uses the same version.
const MermaidCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"

// KaTeX CDN URLs. Centralized here to keep the version consistent across the codebase.
const (
	// KaTeXCSSURL is the CDN URL for the KaTeX stylesheet.
	KaTeXCSSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css"
	// KaTeXJSURL is the CDN URL for the KaTeX core JavaScript library.
	KaTeXJSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.js"
	// KaTeXAutoRenderURL is the CDN URL for the KaTeX auto-render extension, which scans for $...$ and $$...$$ in the document and renders them.
	KaTeXAutoRenderURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/contrib/auto-render.min.js"
)
