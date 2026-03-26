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

// PostProcess applies all post-processing transforms to goldmark-rendered HTML.
func PostProcess(html string) string {
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

		// Find the matching </blockquote>.
		startIdx := loc[0]
		closeTag := "</blockquote>"
		closeIdx := strings.Index(html[startIdx:], closeTag)
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
		// Unescape HTML entities that goldmark added (e.g. &lt; &gt; &amp;),
		// then re-escape to produce safe HTML. This preserves Mermaid syntax
		// characters while preventing XSS via injected HTML tags or attributes.
		code := parts[1]
		code = htmlpkg.UnescapeString(code)
		code = strings.TrimSpace(code)
		code = htmlpkg.EscapeString(code)

		return "<div class=\"mermaid\">\n" + code + "\n</div>"
	})
}

// MermaidScript returns the <script> tags needed to load and initialize Mermaid.
// Only include this when the HTML contains .mermaid elements.
func MermaidScript() string {
	return `<script src="` + utils.MermaidCDNURL + `"></script>
<script>mermaid.initialize({startOnLoad:true,theme:'default',themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}});</script>`
}

// NeedsMermaid reports whether the HTML contains any Mermaid diagram elements.
func NeedsMermaid(html string) bool {
	return strings.Contains(html, `class="mermaid"`)
}
