package output

import (
	"regexp"
	"strings"
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
		"on_this_page":       "本页目录",
		"copy":               "复制",
		"copied":             "已复制！",
		"hide_sidebar":       "隐藏侧边栏",
		"light_mode":         "浅色模式",
		"dark_mode":          "深色模式",
		"system_default":     "跟随系统",
		"search_kbd":         "⌘K",
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
		"on_this_page":       "このページの目次",
		"copy":               "コピー",
		"copied":             "コピー済み！",
		"hide_sidebar":       "サイドバーを隠す",
		"light_mode":         "ライトモード",
		"dark_mode":          "ダークモード",
		"system_default":     "システムデフォルト",
		"search_kbd":         "⌘K",
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
		"on_this_page":       "ON THIS PAGE",
		"copy":               "Copy",
		"copied":             "Copied!",
		"hide_sidebar":       "Hide sidebar",
		"light_mode":         "Light mode",
		"dark_mode":          "Dark mode",
		"system_default":     "System default",
		"search_kbd":         "⌘K",
		"page_of":            "Page %d of %d",
		"built_with":         "Built with %s",
	},
}

// htmlTagPattern strips HTML tags for plain-text extraction.
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

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
	d.UIonThisPage = uiString(lang, "on_this_page")
	d.UIcopy = uiString(lang, "copy")
	d.UIcopied = uiString(lang, "copied")
	d.UIhideSidebar = uiString(lang, "hide_sidebar")
	d.UIlightMode = uiString(lang, "light_mode")
	d.UIdarkMode = uiString(lang, "dark_mode")
	d.UIsystemDefault = uiString(lang, "system_default")
	d.UIsearchKbd = uiString(lang, "search_kbd")
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
