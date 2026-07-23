package output

import (
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// uiStrings holds localized UI labels keyed by language prefix.
// The first matching prefix wins, e.g. "zh-hans" matches "zh".
var uiStrings = map[string]map[string]string{
	"zh": {
		"previous":              "上一章",
		"next":                  "下一章",
		"search_placeholder":    "输入关键词搜索…",
		"search_button":         "搜索",
		"no_results":            "未找到相关结果：",
		"search_unavailable":    "搜索不可用",
		"search_results_one":    "1 个结果",
		"search_results":        "%d 个结果",
		"recent_pages":          "最近访问",
		"recent_empty":          "还没有最近访问的页面",
		"search_navigate":       "选择",
		"search_open":           "打开",
		"search_close":          "关闭",
		"search_match_title":    "标题",
		"search_match_path":     "路径",
		"search_match_text":     "正文",
		"search_matched":        "已定位到：%s",
		"on_this_page":          "本页目录",
		"edit_page":             "编辑此页",
		"not_found_title":       "页面未找到",
		"not_found_home":        "返回首页",
		"copy":                  "复制",
		"copied":                "已复制！",
		"hide_sidebar":          "隐藏侧边栏",
		"light_mode":            "浅色模式",
		"dark_mode":             "深色模式",
		"system_default":        "跟随系统",
		"search_kbd":            "Ctrl/⌘ K",
		"page_of":               "第 %d 页，共 %d 页",
		"built_with":            "使用 %s 构建",
		"assets_mermaid_failed": "图表未渲染：无法加载 Mermaid 库（可能处于离线状态或该 CDN 被拦截），以下为图表源码。",
		"assets_katex_failed":   "本页部分公式未渲染：无法加载 KaTeX 库（可能处于离线状态或该 CDN 被拦截），公式以 LaTeX 源码形式显示。",
	},
	"ja": {
		"previous":              "前へ",
		"next":                  "次へ",
		"search_placeholder":    "検索…",
		"search_button":         "検索",
		"no_results":            "結果なし：",
		"search_unavailable":    "検索利用不可",
		"search_results_one":    "1 件の結果",
		"search_results":        "%d 件の結果",
		"recent_pages":          "最近のページ",
		"recent_empty":          "最近開いたページはまだありません",
		"search_navigate":       "移動",
		"search_open":           "開く",
		"search_close":          "閉じる",
		"search_match_title":    "タイトル",
		"search_match_path":     "パス",
		"search_match_text":     "本文",
		"search_matched":        "一致箇所：%s",
		"on_this_page":          "このページの目次",
		"edit_page":             "このページを編集",
		"not_found_title":       "ページが見つかりません",
		"not_found_home":        "ホームに戻る",
		"copy":                  "コピー",
		"copied":                "コピー済み！",
		"hide_sidebar":          "サイドバーを隠す",
		"light_mode":            "ライトモード",
		"dark_mode":             "ダークモード",
		"system_default":        "システムデフォルト",
		"search_kbd":            "Ctrl/⌘ K",
		"page_of":               "%d / %d ページ",
		"built_with":            "%s で構築",
		"assets_mermaid_failed": "図は描画されていません：Mermaid ライブラリを読み込めませんでした（オフラインまたは CDN がブロックされています）。以下はソースです。",
		"assets_katex_failed":   "このページの一部の数式は描画されていません：KaTeX ライブラリを読み込めませんでした（オフラインまたは CDN がブロックされています）。数式は LaTeX ソースのまま表示されます。",
	},
	// Default (English) is the fallback.
	"en": {
		"previous":              "Previous",
		"next":                  "Next",
		"search_placeholder":    "Type to search…",
		"search_button":         "Search",
		"no_results":            "No results for",
		"search_unavailable":    "Search unavailable",
		"search_results_one":    "1 result",
		"search_results":        "%d results",
		"recent_pages":          "Recent pages",
		"recent_empty":          "No recent pages yet",
		"search_navigate":       "navigate",
		"search_open":           "open",
		"search_close":          "close",
		"search_match_title":    "title",
		"search_match_path":     "path",
		"search_match_text":     "text",
		"search_matched":        "Matched: %s",
		"on_this_page":          "ON THIS PAGE",
		"edit_page":             "Edit this page",
		"not_found_title":       "Page not found",
		"not_found_home":        "Back to home",
		"copy":                  "Copy",
		"copied":                "Copied!",
		"hide_sidebar":          "Hide sidebar",
		"light_mode":            "Light mode",
		"dark_mode":             "Dark mode",
		"system_default":        "System default",
		"search_kbd":            "Ctrl/⌘ K",
		"page_of":               "Page %d of %d",
		"built_with":            "Built with %s",
		"assets_mermaid_failed": "Diagram not rendered: the Mermaid library could not be loaded (offline, or the CDN is blocked). Its source is shown below.",
		"assets_katex_failed":   "Some formulas on this page are not rendered: the KaTeX library could not be loaded (offline, or the CDN is blocked). They are shown as LaTeX source.",
	},
}

// htmlTagPattern strips HTML tags for plain-text extraction.
// Uses the shared pattern from pkg/utils to avoid duplication.
var htmlTagPattern = utils.HTMLTagPattern

// uiString returns the localized UI string for the given key and language.
func uiString(lang, key string) string {
	lang = strings.ToLower(lang)
	// Try exact match first, then prefix match, then fallback to English.
	if m, ok := uiStrings[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	// Try prefix match (e.g. "zh-hans" -> "zh").
	for prefix, m := range uiStrings {
		if strings.HasPrefix(lang, prefix) {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	// Fallback to English.
	if m, ok := uiStrings["en"]; ok {
		return m[key]
	}
	return key
}

// populateUIStrings fills the localized UI string fields in pageData.
func populateUIStrings(d *pageData) {
	lang := d.Language
	d.UIprevious = uiString(lang, "previous")
	d.UInext = uiString(lang, "next")
	d.UIsearchPlaceholder = uiString(lang, "search_placeholder")
	d.UIsearchButton = uiString(lang, "search_button")
	d.UInoResults = uiString(lang, "no_results")
	d.UIsearchUnavailable = uiString(lang, "search_unavailable")
	d.UIsearchResultsOne = uiString(lang, "search_results_one")
	d.UIsearchResults = uiString(lang, "search_results")
	d.UIrecentPages = uiString(lang, "recent_pages")
	d.UIrecentEmpty = uiString(lang, "recent_empty")
	d.UIsearchNavigate = uiString(lang, "search_navigate")
	d.UIsearchOpen = uiString(lang, "search_open")
	d.UIsearchClose = uiString(lang, "search_close")
	d.UIsearchMatchTitle = uiString(lang, "search_match_title")
	d.UIsearchMatchPath = uiString(lang, "search_match_path")
	d.UIsearchMatchText = uiString(lang, "search_match_text")
	d.UIsearchMatched = uiString(lang, "search_matched")
	d.UIonThisPage = uiString(lang, "on_this_page")
	d.UIeditPage = uiString(lang, "edit_page")
	d.UIcopy = uiString(lang, "copy")
	d.UIcopied = uiString(lang, "copied")
	d.UIhideSidebar = uiString(lang, "hide_sidebar")
	d.UIlightMode = uiString(lang, "light_mode")
	d.UIdarkMode = uiString(lang, "dark_mode")
	d.UIsystemDefault = uiString(lang, "system_default")
	d.UIsearchKbd = uiString(lang, "search_kbd")
	d.UIpageOf = uiString(lang, "page_of")
	d.UIbuiltWith = uiString(lang, "built_with")
	d.UIassetsMermaidFailed = uiString(lang, "assets_mermaid_failed")
	d.UIassetsKatexFailed = uiString(lang, "assets_katex_failed")
}

// Meta description length limits (in runes).
const (
	maxMetaDescriptionRunes   = 160
	minMetaDescriptionTruncAt = 80
)

// extractDescription returns the first ~160 characters of plain text from HTML
// content, suitable for use as a meta description.
func extractDescription(htmlContent string) string {
	text := htmlTagPattern.ReplaceAllString(htmlContent, " ")
	text = stdhtml.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) > maxMetaDescriptionRunes {
		// Truncate at word boundary.
		truncated := string(runes[:maxMetaDescriptionRunes])
		if idx := strings.LastIndex(truncated, " "); idx > minMetaDescriptionTruncAt {
			text = truncated[:idx] + "…"
		} else {
			text = truncated + "…"
		}
	}
	return text
}

// contentLeadingHeadingPattern matches an opening heading tag (h1–h6) at the
// very start of the HTML content (ignoring leading whitespace).
var contentLeadingHeadingPattern = regexp.MustCompile(`(?i)^\s*<h[1-6]\b[^>]*>(.*?)</h[1-6]>`)

// contentStartsWithTitle reports whether the HTML content already begins with
// a heading (any level h1–h6) whose text matches pageTitle.  This prevents the
// template from inserting a duplicate title above the content.
func contentStartsWithTitle(html, pageTitle string) bool {
	// The check must be anchored to the *start* of the content. It used to
	// return true for an <h1> anywhere in the document, so a chapter with a
	// second <h1> further down (an appendix, say) had its page title
	// suppressed entirely — leaving that inner heading as the page's only H1
	// and the chapter's real title nowhere on the page.
	m := contentLeadingHeadingPattern.FindStringSubmatch(html)
	if m == nil {
		return false
	}
	// A leading <h1> is the content's own title regardless of its wording, so
	// the template must not add a second one above it. (The pipeline normally
	// strips it first, but only when the chapter has a nav title.)
	if leading := strings.ToLower(strings.TrimSpace(m[0])); strings.HasPrefix(leading, "<h1") {
		return true
	}
	// Strip any inner tags (e.g. <a>, <code>) from the matched heading text
	// and compare with pageTitle after normalising whitespace.
	headingText := htmlTagPattern.ReplaceAllString(m[1], "")
	headingText = stdhtml.UnescapeString(headingText)
	headingText = strings.TrimSpace(strings.Join(strings.Fields(headingText), " "))
	return headingText == strings.TrimSpace(pageTitle)
}
