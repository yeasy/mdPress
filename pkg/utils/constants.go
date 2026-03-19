package utils

// MermaidCDNURL is the CDN URL for the Mermaid diagram library.
// Centralized here so every part of the codebase uses the same version.
const MermaidCDNURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"

// KaTeX CDN URL，统一版本号避免各处不一致。
const (
	// KaTeXCSSURL 是 KaTeX 样式表的 CDN 地址
	KaTeXCSSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css"
	// KaTeXJSURL 是 KaTeX 核心 JS 的 CDN 地址
	KaTeXJSURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.js"
	// KaTeXAutoRenderURL 是 KaTeX 自动渲染扩展的 CDN 地址，负责扫描文档中的 $...$ 和 $$...$$ 并渲染
	KaTeXAutoRenderURL = "https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/contrib/auto-render.min.js"
)
