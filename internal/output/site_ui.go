package output

import (
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// uiStrings holds localized UI labels keyed by language prefix.
// The first matching prefix wins, e.g. "zh-hans" matches "zh".
var uiStrings = map[string]map[string]string{
	"zh": {
		"previous":           "上一章",
		"next":               "下一章",
		"search_placeholder": "输入关键词搜索…",
		"search_button":      "搜索",
		"no_results":         "未找到相关结果：",
		"search_unavailable": "搜索不可用",
		"search_results_one": "1 个结果",
		"search_results":     "%d 个结果",
		"recent_pages":       "最近访问",
		"recent_empty":       "还没有最近访问的页面",
		"search_navigate":    "选择",
		"search_open":        "打开",
		"search_close":       "关闭",
		"search_match_title": "标题",
		"search_match_path":  "路径",
		"search_match_text":  "正文",
		"search_matched":     "已定位到：%s",
		"on_this_page":       "本页目录",
		"copy":               "复制",
		"copied":             "已复制！",
		"hide_sidebar":       "隐藏侧边栏",
		"light_mode":         "浅色模式",
		"dark_mode":          "深色模式",
		"system_default":     "跟随系统",
		"search_kbd":         "Ctrl/⌘ K",
		"page_of":            "第 %d 页，共 %d 页",
		"built_with":         "使用 %s 构建",
	},
	"ja": {
		"previous":           "前へ",
		"next":               "次へ",
		"search_placeholder": "検索…",
		"search_button":      "検索",
		"no_results":         "結果なし：",
		"search_unavailable": "検索利用不可",
		"search_results_one": "1 件の結果",
		"search_results":     "%d 件の結果",
		"recent_pages":       "最近のページ",
		"recent_empty":       "最近開いたページはまだありません",
		"search_navigate":    "移動",
		"search_open":        "開く",
		"search_close":       "閉じる",
		"search_match_title": "タイトル",
		"search_match_path":  "パス",
		"search_match_text":  "本文",
		"search_matched":     "一致箇所：%s",
		"on_this_page":       "このページの目次",
		"copy":               "コピー",
		"copied":             "コピー済み！",
		"hide_sidebar":       "サイドバーを隠す",
		"light_mode":         "ライトモード",
		"dark_mode":          "ダークモード",
		"system_default":     "システムデフォルト",
		"search_kbd":         "Ctrl/⌘ K",
		"page_of":            "%d / %d ページ",
		"built_with":         "%s で構築",
	},
	// Default (English) is the fallback.
	"en": {
		"previous":           "Previous",
		"next":               "Next",
		"search_placeholder": "Type to search…",
		"search_button":      "Search",
		"no_results":         "No results for",
		"search_unavailable": "Search unavailable",
		"search_results_one": "1 result",
		"search_results":     "%d results",
		"recent_pages":       "Recent pages",
		"recent_empty":       "No recent pages yet",
		"search_navigate":    "navigate",
		"search_open":        "open",
		"search_close":       "close",
		"search_match_title": "title",
		"search_match_path":  "path",
		"search_match_text":  "text",
		"search_matched":     "Matched: %s",
		"on_this_page":       "ON THIS PAGE",
		"copy":               "Copy",
		"copied":             "Copied!",
		"hide_sidebar":       "Hide sidebar",
		"light_mode":         "Light mode",
		"dark_mode":          "Dark mode",
		"system_default":     "System default",
		"search_kbd":         "Ctrl/⌘ K",
		"page_of":            "Page %d of %d",
		"built_with":         "Built with %s",
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
	d.UIcopy = uiString(lang, "copy")
	d.UIcopied = uiString(lang, "copied")
	d.UIhideSidebar = uiString(lang, "hide_sidebar")
	d.UIlightMode = uiString(lang, "light_mode")
	d.UIdarkMode = uiString(lang, "dark_mode")
	d.UIsystemDefault = uiString(lang, "system_default")
	d.UIsearchKbd = uiString(lang, "search_kbd")
	d.UIpageOf = uiString(lang, "page_of")
	d.UIbuiltWith = uiString(lang, "built_with")
}

// extractDescription returns the first ~160 characters of plain text from HTML
// content, suitable for use as a meta description.
func extractDescription(htmlContent string) string {
	text := htmlTagPattern.ReplaceAllString(htmlContent, " ")
	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) > 160 {
		// Truncate at word boundary.
		truncated := string(runes[:160])
		if idx := strings.LastIndex(truncated, " "); idx > 80 {
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
	// Fast path: if there is an <h1> anywhere, we never need a generated title.
	if strings.Contains(strings.ToLower(html), "<h1") {
		return true
	}
	m := contentLeadingHeadingPattern.FindStringSubmatch(html)
	if m == nil {
		return false
	}
	// Strip any inner tags (e.g. <a>, <code>) from the matched heading text
	// and compare with pageTitle after normalising whitespace.
	headingText := htmlTagPattern.ReplaceAllString(m[1], "")
	headingText = strings.TrimSpace(strings.Join(strings.Fields(headingText), " "))
	return headingText == strings.TrimSpace(pageTitle)
}
