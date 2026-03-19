// postprocess.go 对 goldmark 渲染后的 HTML 做后处理。
// 包括：GFM Alerts（[!NOTE] 等）转换、Mermaid 代码块转换。
package markdown

import (
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// alertTypes 支持的 GFM Alert 类型及其图标/颜色
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

// alertPattern 匹配 blockquote 内的 [!TYPE] 标记
// goldmark 将 > [!NOTE] 渲染为 <blockquote>\n<p>[!NOTE]...
var alertPattern = regexp.MustCompile(
	`<blockquote>\s*<p>\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)

// PostProcess 对渲染后的 HTML 做后处理
func PostProcess(html string) string {
	html = processAlerts(html)
	html = processMermaid(html)
	return html
}

// processAlerts 将 GFM Alert 语法转换为带样式的 HTML。
//
// 输入（goldmark 渲染后的 HTML）：
//
//	<blockquote>
//	<p>[!NOTE]
//	This is a note.</p>
//	</blockquote>
//
// 输出：
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

// mermaidPattern 匹配 <pre><code class="language-mermaid">...</code></pre>
// 或 goldmark-highlighting 输出的带 chroma style 的 mermaid 块
var mermaidPattern = regexp.MustCompile(
	`<pre[^>]*><code[^>]*class="[^"]*language-mermaid[^"]*"[^>]*>([\s\S]*?)</code></pre>`)

// processMermaid 将 mermaid 代码块转换为 <div class="mermaid"> 以供客户端渲染。
// Mermaid JS 库会在浏览器中自动找到 .mermaid 类的 div 并渲染为 SVG。
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

// MermaidScript 返回引入 Mermaid JS 的 <script> 标签。
// 只有当 HTML 中包含 .mermaid div 时才需要引入。
func MermaidScript() string {
	return `<script src="` + utils.MermaidCDNURL + `"></script>
<script>mermaid.initialize({startOnLoad:true,theme:'default'});</script>`
}

// NeedsMermaid 检查 HTML 中是否包含 mermaid 图表
func NeedsMermaid(html string) bool {
	return strings.Contains(html, `class="mermaid"`)
}
