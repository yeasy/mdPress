// postprocess.go performs post-processing on HTML emitted by goldmark.
// Includes: GFM Alert conversion ([!NOTE] etc.) and Mermaid code block conversion.
package markdown

import (
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// alertTypes maps GFM Alert type names to their display style.
var alertTypes = map[string]alertStyle{
	"NOTE":      {icon: "ℹ️", color: "#0969da", bg: "#ddf4ff", border: "#54aeff", label: "Note"},
	"TIP":       {icon: "💡", color: "#1a7f37", bg: "#dafbe1", border: "#4ac26b", label: "Tip"},
	"IMPORTANT": {icon: "🔔", color: "#8250df", bg: "#fbefff", border: "#c297ff", label: "Important"},
	"WARNING":   {icon: "⚠️", color: "#9a6700", bg: "#fff8c5", border: "#d4a72c", label: "Warning"},
	"CAUTION":   {icon: "🔴", color: "#cf222e", bg: "#ffebe9", border: "#ff8182", label: "Caution"},
}

type alertStyle struct {
	icon, color, bg, border, label string
}

// alertPattern matches the [!TYPE] marker inside a blockquote.
// goldmark renders "> [!NOTE]" as "<blockquote>\n<p>[!NOTE]...".
var alertPattern = regexp.MustCompile(
	`<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)

// PostProcess applies all post-processing transforms to goldmark-rendered HTML.
func PostProcess(html string) string {
	html = processAlerts(html)
	html = processMermaid(html)
	return html
}

// processAlerts converts GFM Alert syntax to styled HTML divs.
//
// Input (goldmark-rendered HTML):
//
//	<blockquote>
//	<p>[!NOTE]
//	This is a note.</p>
//	</blockquote>
//
// Output:
//
//	<div class="alert alert-note">
//	<p class="alert-title">ℹ️ Note</p>
//	<p>This is a note.</p>
//	</div>
func processAlerts(html string) string {
	// 找到所有 alert blockquote 的起始位置
	for {
		loc := alertPattern.FindStringSubmatchIndex(html)
		if loc == nil {
			break
		}

		// loc[0]:loc[1] = 整个匹配
		// loc[2]:loc[3] = 类型名（NOTE/TIP/...）
		alertType := html[loc[2]:loc[3]]
		style, ok := alertTypes[alertType]
		if !ok {
			break
		}

		// 找到对应的 </blockquote>
		startIdx := loc[0]
		closeTag := "</blockquote>"
		closeIdx := strings.Index(html[startIdx:], closeTag)
		if closeIdx == -1 {
			break
		}
		closeIdx += startIdx

		// 提取 blockquote 内部内容
		inner := html[loc[1]:closeIdx]

		// 移除 [!TYPE] 后可能紧跟的换行/空白和 </p><p> 等
		inner = strings.TrimPrefix(inner, "\n")

		// 构建 alert div
		alertHTML := "<div class=\"alert alert-" + strings.ToLower(alertType) + "\" " +
			"style=\"border-left:4px solid " + style.border + ";background:" + style.bg +
			";padding:12px 16px;margin:1em 0;border-radius:0 6px 6px 0;\">\n" +
			"<p class=\"alert-title\" style=\"color:" + style.color +
			";font-weight:600;margin:0 0 4px;\">" + style.icon + " " + style.label + "</p>\n" +
			inner + "\n</div>"

		html = html[:startIdx] + alertHTML + html[closeIdx+len(closeTag):]
	}

	return html
}

// mermaidPattern matches <pre><code class="language-mermaid">...</code></pre>
// as well as the chroma-highlighted variant produced by goldmark-highlighting.
var mermaidPattern = regexp.MustCompile(
	`<pre[^>]*><code[^>]*class="[^"]*language-mermaid[^"]*"[^>]*>([\s\S]*?)</code></pre>`)

// processMermaid converts mermaid code blocks to <div class="mermaid"> elements
// for client-side rendering by the Mermaid JS library.
func processMermaid(html string) string {
	return mermaidPattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := mermaidPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		// 还原 HTML 实体（goldmark 会转义 < > 等）
		code := parts[1]
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&amp;", "&")
		code = strings.ReplaceAll(code, "&quot;", `"`)
		code = strings.ReplaceAll(code, "&#39;", "'")
		code = strings.TrimSpace(code)

		return "<div class=\"mermaid\">\n" + code + "\n</div>"
	})
}

// MermaidScript returns the <script> tags needed to load and initialise Mermaid.
// Only include this when the HTML contains .mermaid elements.
func MermaidScript() string {
	return `<script src="` + utils.MermaidCDNURL + `"></script>
<script>mermaid.initialize({startOnLoad:true,theme:'default'});</script>`
}

// NeedsMermaid reports whether the HTML contains any Mermaid diagram elements.
func NeedsMermaid(html string) bool {
	return strings.Contains(html, `class="mermaid"`)
}

// NeedsKaTeX reports whether the HTML contains any math formula elements
// produced by the math preprocessor.
func NeedsKaTeX(html string) bool {
	return strings.Contains(html, `class="math `)
}

// KaTeXScript returns the HTML tags (link + scripts) needed to load KaTeX and
// its auto-render extension. The auto-render extension scans the document for
// $...$ and $$...$$ delimiters and renders them with KaTeX.
// Only include this when the HTML contains math elements (see NeedsKaTeX).
func KaTeXScript() string {
	return `<link rel="stylesheet" href="` + utils.KaTeXCSSURL + `">` +
		`<script defer src="` + utils.KaTeXJSURL + `"></script>` +
		`<script defer src="` + utils.KaTeXAutoRenderURL + `"` +
		` onload="renderMathInElement(document.body,{` +
		`delimiters:[` +
		`{left:'$$',right:'$$',display:true},` +
		`{left:'$',right:'$',display:false}` +
		`],throwOnError:false});"></script>`
}

// KaTeXScriptForEpub returns XHTML-compatible KaTeX script tags for use inside
// EPUB XHTML documents. Some EPUB readers (e.g. Apple Books) support JavaScript,
// so KaTeX can render math formulas in those readers.
func KaTeXScriptForEpub() string {
	return "\n" +
		`<link rel="stylesheet" href="` + utils.KaTeXCSSURL + `"/>` + "\n" +
		`<script src="` + utils.KaTeXJSURL + `"></script>` + "\n" +
		`<script src="` + utils.KaTeXAutoRenderURL + `"></script>` + "\n" +
		`<script>` + "\n" +
		`if(typeof renderMathInElement==='function'){` + "\n" +
		`  renderMathInElement(document.body,{` + "\n" +
		`    delimiters:[{left:'$$',right:'$$',display:true},{left:'$',right:'$',display:false}],` + "\n" +
		`    throwOnError:false` + "\n" +
		`  });` + "\n" +
		`}` + "\n" +
		`</script>`
}
