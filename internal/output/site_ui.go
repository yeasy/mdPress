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
		"skip_to_content":       "跳到正文",
		"nav_toggle":            "切换导航菜单",
		"breadcrumb":            "面包屑导航",
		"page_navigation":       "翻页导航",
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
		"skip_to_content":       "本文へスキップ",
		"nav_toggle":            "ナビゲーションメニューを切り替え",
		"breadcrumb":            "パンくずリスト",
		"page_navigation":       "ページナビゲーション",
		"assets_mermaid_failed": "図は描画されていません：Mermaid ライブラリを読み込めませんでした（オフラインまたは CDN がブロックされています）。以下はソースです。",
		"assets_katex_failed":   "このページの一部の数式は描画されていません：KaTeX ライブラリを読み込めませんでした（オフラインまたは CDN がブロックされています）。数式は LaTeX ソースのまま表示されます。",
	},
	"ko": {
		"previous":              "이전",
		"next":                  "다음",
		"search_placeholder":    "검색어 입력…",
		"search_button":         "검색",
		"no_results":            "검색 결과 없음:",
		"search_unavailable":    "검색을 사용할 수 없음",
		"search_results_one":    "결과 1개",
		"search_results":        "결과 %d개",
		"recent_pages":          "최근 페이지",
		"recent_empty":          "최근 방문한 페이지가 없습니다",
		"search_navigate":       "이동",
		"search_open":           "열기",
		"search_close":          "닫기",
		"search_match_title":    "제목",
		"search_match_path":     "경로",
		"search_match_text":     "본문",
		"search_matched":        "일치: %s",
		"on_this_page":          "이 페이지 목차",
		"edit_page":             "이 페이지 편집",
		"not_found_title":       "페이지를 찾을 수 없습니다",
		"not_found_home":        "홈으로 돌아가기",
		"copy":                  "복사",
		"copied":                "복사됨!",
		"hide_sidebar":          "사이드바 숨기기",
		"light_mode":            "라이트 모드",
		"dark_mode":             "다크 모드",
		"system_default":        "시스템 기본값",
		"search_kbd":            "Ctrl/⌘ K",
		"page_of":               "%d / %d 페이지",
		"built_with":            "%s(으)로 제작",
		"skip_to_content":       "본문으로 건너뛰기",
		"nav_toggle":            "탐색 메뉴 전환",
		"breadcrumb":            "탐색 경로",
		"page_navigation":       "페이지 탐색",
		"assets_mermaid_failed": "다이어그램이 렌더링되지 않았습니다. Mermaid 라이브러리를 불러올 수 없습니다(오프라인이거나 CDN이 차단됨). 아래는 소스입니다.",
		"assets_katex_failed":   "이 페이지의 일부 수식이 렌더링되지 않았습니다. KaTeX 라이브러리를 불러올 수 없습니다(오프라인이거나 CDN이 차단됨). 수식은 LaTeX 소스로 표시됩니다.",
	},
	"fr": {
		"previous":              "Précédent",
		"next":                  "Suivant",
		"search_placeholder":    "Rechercher…",
		"search_button":         "Rechercher",
		"no_results":            "Aucun résultat pour",
		"search_unavailable":    "Recherche indisponible",
		"search_results_one":    "1 résultat",
		"search_results":        "%d résultats",
		"recent_pages":          "Pages récentes",
		"recent_empty":          "Aucune page récente pour l'instant",
		"search_navigate":       "naviguer",
		"search_open":           "ouvrir",
		"search_close":          "fermer",
		"search_match_title":    "titre",
		"search_match_path":     "chemin",
		"search_match_text":     "texte",
		"search_matched":        "Correspondance : %s",
		"on_this_page":          "SUR CETTE PAGE",
		"edit_page":             "Modifier cette page",
		"not_found_title":       "Page introuvable",
		"not_found_home":        "Retour à l'accueil",
		"copy":                  "Copier",
		"copied":                "Copié !",
		"hide_sidebar":          "Masquer la barre latérale",
		"light_mode":            "Mode clair",
		"dark_mode":             "Mode sombre",
		"system_default":        "Réglage système",
		"search_kbd":            "Ctrl/⌘ K",
		"page_of":               "Page %d sur %d",
		"built_with":            "Créé avec %s",
		"skip_to_content":       "Aller au contenu",
		"nav_toggle":            "Basculer le menu de navigation",
		"breadcrumb":            "Fil d'Ariane",
		"page_navigation":       "Navigation entre pages",
		"assets_mermaid_failed": "Diagramme non rendu : la bibliothèque Mermaid n'a pas pu être chargée (hors ligne, ou CDN bloqué). Sa source est affichée ci-dessous.",
		"assets_katex_failed":   "Certaines formules de cette page ne sont pas rendues : la bibliothèque KaTeX n'a pas pu être chargée (hors ligne, ou CDN bloqué). Elles sont affichées en source LaTeX.",
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
		"skip_to_content":       "Skip to content",
		"nav_toggle":            "Toggle navigation menu",
		"breadcrumb":            "Breadcrumb",
		"page_navigation":       "Page navigation",
		"assets_mermaid_failed": "Diagram not rendered: the Mermaid library could not be loaded (offline, or the CDN is blocked). Its source is shown below.",
		"assets_katex_failed":   "Some formulas on this page are not rendered: the KaTeX library could not be loaded (offline, or the CDN is blocked). They are shown as LaTeX source.",
	},
}

// htmlTagPattern strips HTML tags for plain-text extraction.
// Uses the shared pattern from pkg/utils to avoid duplication.
var htmlTagPattern = utils.HTMLTagPattern

// uiLocalized reports whether the site UI has a translation table covering
// lang (exact or prefix match). English and an empty language count as
// covered.
func uiLocalized(lang string) bool {
	lang = strings.ToLower(lang)
	if lang == "" {
		return true
	}
	if _, ok := uiStrings[lang]; ok {
		return true
	}
	for prefix := range uiStrings {
		if strings.HasPrefix(lang, prefix) {
			return true
		}
	}
	return false
}

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
	d.UIskipToContent = uiString(lang, "skip_to_content")
	d.UInavToggle = uiString(lang, "nav_toggle")
	d.UIbreadcrumb = uiString(lang, "breadcrumb")
	d.UIpageNavigation = uiString(lang, "page_navigation")
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
