// postprocess.go performs post-processing on HTML emitted by goldmark.
// Includes: GFM Alert conversion ([!NOTE] etc.) and Mermaid code block conversion.
package markdown

import (
	htmlpkg "html"
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

// postProcess applies all post-processing transforms to goldmark-rendered HTML.
func postProcess(html string) string {
	html = processAlerts(html)
	html = processMermaid(html)
	html = stripChromaPreStyle(html)
	html = addLazyLoading(html)
	return html
}

// imgTagPattern matches any <img ...> tag so we can inspect it in a callback.
var imgTagPattern = regexp.MustCompile(`<img\b[^>]*>`)

// addLazyLoading inserts loading="lazy" on <img> tags that lack it.
func addLazyLoading(html string) string {
	return imgTagPattern.ReplaceAllStringFunc(html, func(tag string) string {
		if strings.Contains(tag, "loading=") {
			return tag
		}
		// Insert loading="lazy" right after "<img"
		return "<img loading=\"lazy\"" + tag[len("<img"):]
	})
}

// chromaPreStylePattern matches the inline style attribute that chroma adds to
// <pre> elements (e.g. style="background-color:#fff;"). Removing it lets each
// output format's CSS control code block appearance without specificity fights.
var chromaPreStylePattern = regexp.MustCompile(`(<pre)\s+style="[^"]*"`)

// stripChromaPreStyle removes inline style attributes from <pre> tags injected
// by chroma's HTML formatter. Chroma sets background-color (and sometimes color)
// on <pre> via inline styles, which override the site/standalone/PDF CSS and can
// cause invisible text when the inline bg conflicts with the CSS text color.
func stripChromaPreStyle(html string) string {
	return chromaPreStylePattern.ReplaceAllString(html, "$1")
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
	// Find all alert blockquote start positions.
	for {
		loc := alertPattern.FindStringSubmatchIndex(html)
		if loc == nil {
			break
		}

		// loc[0]:loc[1] = full match
		// loc[2]:loc[3] = alert type name (NOTE/TIP/...)
		alertType := html[loc[2]:loc[3]]
		style, ok := alertTypes[alertType]
		if !ok {
			break
		}

		// Find the matching </blockquote>, accounting for nested blockquotes.
		startIdx := loc[0]
		closeTag := "</blockquote>"
		openTag := "<blockquote"
		closeIdx := findMatchingClose(html[startIdx:], openTag, closeTag)
		if closeIdx == -1 {
			break
		}
		closeIdx += startIdx

		// Extract inner content of the blockquote.
		inner := html[loc[1]:closeIdx]

		// Strip any leading newline/whitespace after the [!TYPE] marker.
		inner = strings.TrimPrefix(inner, "\n")

		// Build the alert div.
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
		// Unescape HTML entities that goldmark added (e.g. &lt; &gt; &amp; &#34;)
		// so Mermaid JS can parse the diagram syntax correctly.
		// Primary XSS protection is Mermaid's securityLevel:'strict' mode.
		// As defense-in-depth (e.g. if Mermaid CDN fails to load), strip
		// <script> tags and event handler attributes from the unescaped content.
		code := parts[1]
		code = htmlpkg.UnescapeString(code)
		code = strings.TrimSpace(code)
		code = sanitizeMermaidCode(code)

		return "<div class=\"mermaid\">\n" + code + "\n</div>"
	})
}

// dangerousTagPattern matches HTML tags that can execute code, load external
// resources, or inject styles: <script>, <iframe>, <object>, <embed>, <form>,
// <base>, <link>, <meta>, <style>. Mermaid diagrams never need these tags;
// stripping them prevents XSS/CSS-injection when Mermaid JS fails to load and
// the browser renders the raw HTML.
// The pattern handles '>' inside quoted attribute values to avoid premature
// match termination (e.g. <script data-x="a>b">).
var dangerousTagPattern = regexp.MustCompile(
	`(?i)</?(?:script|iframe|object|embed|form|base|link|meta|style)\b(?:[^>"']|"[^"]*"|'[^']*')*>`)

// eventHandlerPattern matches HTML event handler attributes like onclick, onload, etc.
var eventHandlerPattern = regexp.MustCompile(`(?i)\s+on[a-z]+\s*=\s*("[^"]*"|'[^']*'|[^\s>]*)`)

// jsURIPattern matches href or src attributes with javascript:, vbscript:,
// or data: URIs. The full match covers the attribute name + value (including
// quotes), so ReplaceAllString replaces the entire attribute cleanly.
var jsURIPattern = regexp.MustCompile(`(?i)(?:href|src)\s*=\s*(?:"[^"]*(?:javascript|vbscript|data)\s*:[^"]*"|'[^']*(?:javascript|vbscript|data)\s*:[^']*'|[^\s>]*(?:javascript|vbscript|data)\s*:[^\s>]*)`)

// sanitizeMermaidCode strips dangerous HTML tags, event handler attributes,
// and javascript:/data: URIs from Mermaid diagram code as defense-in-depth
// against XSS if Mermaid JS fails to load.
func sanitizeMermaidCode(code string) string {
	code = dangerousTagPattern.ReplaceAllString(code, "")
	code = eventHandlerPattern.ReplaceAllString(code, "")
	code = jsURIPattern.ReplaceAllString(code, "data-blocked-uri=\"removed\"")
	return code
}

// mermaidScript returns the <script> tags needed to load and initialize Mermaid.
// Only include this when the HTML contains .mermaid elements.
func mermaidScript() string {
	return `<script src="` + utils.MermaidCDNURL + `"></script>
<script>mermaid.initialize({startOnLoad:true,theme:'default',securityLevel:'strict',themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}});</script>`
}

// NeedsMermaid reports whether the HTML contains any Mermaid diagram elements.
func NeedsMermaid(html string) bool {
	return strings.Contains(html, `class="mermaid"`)
}

// findMatchingClose finds the position of the closing tag that matches the
// first opening tag in s, correctly handling nested tags of the same type.
// s is expected to start at or before the first opening tag.
// Returns -1 if no matching close tag is found.
func findMatchingClose(s, openTag, closeTag string) int {
	depth := 0
	i := 0
	for i < len(s) {
		openIdx := strings.Index(s[i:], openTag)
		closeIdx := strings.Index(s[i:], closeTag)

		if closeIdx == -1 {
			return -1
		}

		// If there's no more open tags, or close comes first.
		if openIdx == -1 || closeIdx < openIdx {
			depth--
			if depth <= 0 {
				return i + closeIdx
			}
			i += closeIdx + len(closeTag)
		} else {
			depth++
			i += openIdx + len(openTag)
		}
	}
	return -1
}
