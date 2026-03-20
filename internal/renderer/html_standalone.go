// html_standalone.go renders a self-contained single-page HTML document.
// The output embeds CSS and JavaScript and implements a GitBook-style three-column
// layout: left sidebar (global TOC), centre content area, right in-page TOC.
//
// Additional features: dark/light/system theme toggle, code copy button with
// language label, callout boxes, full-text search (⌘K), prev/next navigation,
// image lightbox, Mermaid diagrams, and KaTeX math formulas.
package renderer

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// StandaloneHTMLRenderer 渲染自包含单页 HTML 文档
type StandaloneHTMLRenderer struct {
	config *config.BookConfig
	theme  *theme.Theme
	tmpl   *template.Template
}

// standaloneData 是模板数据模型
type standaloneData struct {
	Title       string
	Author      string
	CSS         template.CSS
	Chapters    []standaloneChapter
	SidebarHTML template.HTML
}

// standaloneChapter 存储章节渲染数据（含前后章导航）
type standaloneChapter struct {
	Title     string
	ID        string
	Content   template.HTML
	PrevTitle string // 上一章标题（空则无上一章）
	PrevID    string // 上一章 ID
	NextTitle string // 下一章标题（空则无下一章）
	NextID    string // 下一章 ID
}

// standaloneSidebarChapter 是侧边栏树节点
type standaloneSidebarChapter struct {
	ChapterHTML
	Children []standaloneSidebarChapter
}

// NewStandaloneHTMLRenderer creates a single-page HTML renderer.
func NewStandaloneHTMLRenderer(cfg *config.BookConfig, thm *theme.Theme) (*StandaloneHTMLRenderer, error) {
	// Substitute CDN URL placeholders before parsing the template so that the
	// template engine never needs to evaluate them as Go template expressions.
	resolved := strings.ReplaceAll(standaloneHTMLTemplate, "{{MERMAID_CDN_URL}}", utils.MermaidCDNURL)
	resolved = strings.ReplaceAll(resolved, "{{KATEX_CSS_URL}}", utils.KaTeXCSSURL)
	resolved = strings.ReplaceAll(resolved, "{{KATEX_JS_URL}}", utils.KaTeXJSURL)
	resolved = strings.ReplaceAll(resolved, "{{KATEX_AUTO_RENDER_URL}}", utils.KaTeXAutoRenderURL)

	tmpl, err := template.New("standalone").Funcs(template.FuncMap{
		"safeHTML": func(s template.HTML) template.HTML { return s },
	}).Parse(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standalone HTML template: %w", err)
	}
	return &StandaloneHTMLRenderer{
		config: cfg,
		theme:  thm,
		tmpl:   tmpl,
	}, nil
}

// Render 渲染完整的单页 HTML 文档
func (r *StandaloneHTMLRenderer) Render(parts *RenderParts) (string, error) {
	if parts == nil {
		return "", fmt.Errorf("render parts cannot be nil")
	}

	// 组装 CSS 包（主题 CSS + 自定义 CSS）
	var cssBuilder strings.Builder
	if r.theme != nil {
		cssBuilder.WriteString(r.theme.ToCSS())
		cssBuilder.WriteString("\n")
	}
	if parts.CustomCSS != "" {
		cssBuilder.WriteString(parts.CustomCSS)
		cssBuilder.WriteString("\n")
	}

	// 转换章节数据，预计算前后章导航信息
	chapters := make([]standaloneChapter, 0, len(parts.ChaptersHTML))
	for i, ch := range parts.ChaptersHTML {
		chID := ch.ID
		if chID == "" {
			chID = fmt.Sprintf("chapter-%d", i+1)
		}

		// 计算前后章信息
		var prevTitle, prevID, nextTitle, nextID string
		if i > 0 {
			prev := parts.ChaptersHTML[i-1]
			prevTitle = prev.Title
			prevID = prev.ID
			if prevID == "" {
				prevID = fmt.Sprintf("chapter-%d", i)
			}
		}
		if i < len(parts.ChaptersHTML)-1 {
			next := parts.ChaptersHTML[i+1]
			nextTitle = next.Title
			nextID = next.ID
			if nextID == "" {
				nextID = fmt.Sprintf("chapter-%d", i+2)
			}
		}

		chapters = append(chapters, standaloneChapter{
			Title:     ch.Title,
			ID:        chID,
			Content:   template.HTML(ch.Content),
			PrevTitle: prevTitle,
			PrevID:    prevID,
			NextTitle: nextTitle,
			NextID:    nextID,
		})
	}

	data := standaloneData{
		Title:       r.config.Book.Title,
		Author:      r.config.Book.Author,
		CSS:         template.CSS(cssBuilder.String()),
		Chapters:    chapters,
		SidebarHTML: template.HTML(r.buildSidebar(parts.ChaptersHTML)),
	}

	var result strings.Builder
	if err := r.tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to render standalone HTML: %w", err)
	}
	return result.String(), nil
}

// buildSidebar 生成左侧全局 TOC 侧边栏 HTML
func (r *StandaloneHTMLRenderer) buildSidebar(chapters []ChapterHTML) string {
	var b strings.Builder
	for _, ch := range buildStandaloneSidebarTree(chapters) {
		r.renderSidebarChapter(&b, ch)
	}
	return b.String()
}

// renderSidebarChapter 递归渲染一个侧边栏章节条目。
// 注意：保留 toc-group、data-group-id、data-group-link 等属性以维持测试兼容性。
func (r *StandaloneHTMLRenderer) renderSidebarChapter(b *strings.Builder, ch standaloneSidebarChapter) {
	hasChildren := len(ch.Headings) > 0 || len(ch.Children) > 0

	// 保持 toc-group 类名（测试兼容）并添加新的 toc-item 类
	groupClass := "toc-group toc-item"
	if hasChildren {
		groupClass += " has-children"
	}

	fmt.Fprintf(b, `<div class="%s" data-group-id="%s">`,
		groupClass, template.HTMLEscapeString(ch.ID))
	b.WriteString(`<div class="toc-row">`)

	if hasChildren {
		b.WriteString(`<button class="toc-toggle" type="button" aria-label="展开/折叠" aria-expanded="false"></button>`)
	} else {
		b.WriteString(`<span class="toc-spacer"></span>`)
	}

	// 保留 data-group-link="true"（测试兼容）
	fmt.Fprintf(b,
		`<a href="#%s" class="toc-link toc-link-chapter toc-depth-%d" data-target="%s" data-group-link="true">%s</a>`,
		template.HTMLEscapeString(ch.ID),
		ch.Depth+1,
		template.HTMLEscapeString(ch.ID),
		template.HTMLEscapeString(ch.Title))
	b.WriteString(`</div>`)

	if hasChildren {
		// 默认折叠（hidden 属性）
		b.WriteString(`<div class="toc-children" hidden>`)
		if len(ch.Headings) > 0 {
			r.renderSidebarHeadings(b, ch.Headings, 0)
		}
		for _, child := range ch.Children {
			r.renderSidebarChapter(b, child)
		}
		b.WriteString(`</div>`)
	}

	b.WriteString(`</div>`)
}

// renderSidebarHeadings 递归渲染侧边栏中的标题条目
func (r *StandaloneHTMLRenderer) renderSidebarHeadings(b *strings.Builder, headings []NavHeading, depth int) {
	for _, h := range headings {
		fmt.Fprintf(b,
			`<a href="#%s" class="toc-link toc-link-heading toc-heading-depth-%d" data-target="%s">%s</a>`,
			template.HTMLEscapeString(h.ID),
			depth+1,
			template.HTMLEscapeString(h.ID),
			template.HTMLEscapeString(h.Title))
		if len(h.Children) > 0 {
			r.renderSidebarHeadings(b, h.Children, depth+1)
		}
	}
}

// buildStandaloneSidebarTree 将扁平章节列表转换为树形结构
func buildStandaloneSidebarTree(chapters []ChapterHTML) []standaloneSidebarChapter {
	var build func(start, depth int) ([]standaloneSidebarChapter, int)
	build = func(start, depth int) ([]standaloneSidebarChapter, int) {
		var result []standaloneSidebarChapter
		i := start
		for i < len(chapters) {
			ch := chapters[i]
			if ch.Depth < depth {
				break
			}
			if ch.Depth > depth {
				i++
				continue
			}
			node := standaloneSidebarChapter{ChapterHTML: ch}
			i++
			if i < len(chapters) && chapters[i].Depth > depth {
				node.Children, i = build(i, depth+1)
			}
			result = append(result, node)
		}
		return result, i
	}
	tree, _ := build(0, 0)
	return tree
}

// standaloneHTMLTemplate 自包含单页 HTML 模板（GitBook 风格三栏布局）
const standaloneHTMLTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="author" content="{{.Author}}">
  <title>{{.Title}}</title>
  <!--
    防止主题闪烁（FOUC）：在页面渲染前从 localStorage 读取主题设置并立即应用。
    此脚本必须放在 <head> 内，在任何 CSS 之前执行。
  -->
  <script>
  (function() {
    try {
      var t = localStorage.getItem('mdpress-theme') || 'system';
      var dark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
      if (dark) document.documentElement.setAttribute('data-theme', 'dark');
    } catch(e) {}
  })();
  </script>
  <style>
    /* ============================================================
       CSS 变量 - 亮色模式默认值
       ============================================================ */
    :root {
      --color-bg:           #ffffff;
      --color-bg-alt:       #f8f9fa;
      --color-bg-sidebar:   #f5f5f7;
      --color-text:         #1f2328;
      --color-text-muted:   #656d76;
      --color-heading:      #0d1117;
      --color-link:         #0969da;
      --color-link-hover:   #0550ae;
      --color-border:       #d0d7de;
      --color-accent:       #0969da;
      --color-accent-light: rgba(9, 105, 218, 0.08);

      /* 代码块 */
      --color-code-bg:      #f6f8fa;
      --color-code-border:  #d0d7de;
      --color-code-text:    #cf222e;
      --color-code-lang:    #57606a;

      /* 侧边栏 */
      --color-sidebar-hover:  rgba(9, 105, 218, 0.06);
      --color-sidebar-active: #0969da;
      --color-sidebar-active-bg: rgba(9, 105, 218, 0.1);

      /* 表格 */
      --color-table-header: #f6f8fa;
      --color-table-stripe: #ffffff;
      --color-table-stripe-alt: #f6f8fa;
      --color-table-hover:  #eef2ff;

      /* Callout 提示框 */
      --callout-note-bg:        #dbeafe;
      --callout-note-border:    #2563eb;
      --callout-note-color:     #1e40af;
      --callout-warning-bg:     #fef3c7;
      --callout-warning-border: #d97706;
      --callout-warning-color:  #92400e;
      --callout-tip-bg:         #dcfce7;
      --callout-tip-border:     #16a34a;
      --callout-tip-color:      #15803d;
      --callout-important-bg:   #fee2e2;
      --callout-important-border: #dc2626;
      --callout-important-color: #9f1239;

      /* 进度条 */
      --color-progress: #0969da;

      /* 阴影 */
      --shadow-sm: 0 1px 3px rgba(31,35,40,0.06), 0 1px 2px rgba(31,35,40,0.04);
      --shadow-md: 0 4px 8px rgba(31,35,40,0.08), 0 2px 4px rgba(31,35,40,0.06);

      /* 布局 */
      --toolbar-height:      56px;
      --left-sidebar-width:  260px;
      --right-sidebar-width: 220px;
      --content-max-width:   800px;
    }

    /* ============================================================
       CSS 变量 - 暗色模式
       ============================================================ */
    [data-theme="dark"] {
      --color-bg:           #0d1117;
      --color-bg-alt:       #161b22;
      --color-bg-sidebar:   #13191f;
      --color-text:         #c9d1d9;
      --color-text-muted:   #8b949e;
      --color-heading:      #f0f6fc;
      --color-link:         #58a6ff;
      --color-link-hover:   #79b8ff;
      --color-border:       #30363d;
      --color-accent:       #58a6ff;
      --color-accent-light: rgba(88, 166, 255, 0.1);

      --color-code-bg:      #161b22;
      --color-code-border:  #30363d;
      --color-code-text:    #ff7b72;
      --color-code-lang:    #8b949e;

      --color-sidebar-hover:    rgba(88, 166, 255, 0.06);
      --color-sidebar-active:   #58a6ff;
      --color-sidebar-active-bg: rgba(88, 166, 255, 0.1);

      --color-table-header:    #161b22;
      --color-table-stripe:    #0d1117;
      --color-table-stripe-alt: #161b22;
      --color-table-hover:     #1c2b3a;

      --callout-note-bg:        #1a3558;
      --callout-note-border:    #388bfd;
      --callout-note-color:     #79b8ff;
      --callout-warning-bg:     #2d1f00;
      --callout-warning-border: #d29922;
      --callout-warning-color:  #e3b341;
      --callout-tip-bg:         #0a2813;
      --callout-tip-border:     #3fb950;
      --callout-tip-color:      #56d364;
      --callout-important-bg:   #2d0b10;
      --callout-important-border: #f85149;
      --callout-important-color: #ff7b72;

      --color-progress: #58a6ff;
      --shadow-sm: 0 1px 3px rgba(0,0,0,0.3);
      --shadow-md: 0 4px 8px rgba(0,0,0,0.4);
    }

    /* ============================================================
       基础重置
       ============================================================ */
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    [hidden] { display: none !important; }

    html {
      font-size: 16px;
      scroll-behavior: smooth;
      -webkit-font-smoothing: antialiased;
      -moz-osx-font-smoothing: grayscale;
    }

    body {
      font-family: system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
                   'Noto Sans SC', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'WenQuanYi Micro Hei', sans-serif;
      font-size: 16px;
      line-height: 1.7;
      color: var(--color-text);
      background: var(--color-bg);
      transition: background-color 0.2s ease, color 0.2s ease;
    }

    /* ============================================================
       阅读进度条（页面顶部细条）
       ============================================================ */
    #reading-progress {
      position: fixed;
      top: 0; left: 0;
      height: 3px;
      width: 0%;
      background: var(--color-progress);
      z-index: 9999;
      transition: width 0.1s linear;
      border-radius: 0 2px 2px 0;
    }

    /* ============================================================
       顶部工具栏
       ============================================================ */
    .toolbar {
      position: fixed;
      top: 0; left: 0; right: 0;
      height: var(--toolbar-height);
      background: var(--color-bg);
      border-bottom: 1px solid var(--color-border);
      display: flex;
      align-items: center;
      padding: 0 1rem;
      gap: 0.5rem;
      z-index: 1000;
      box-shadow: var(--shadow-sm);
    }

    .toolbar-brand {
      font-size: 0.95rem;
      font-weight: 600;
      color: var(--color-heading);
      text-decoration: none;
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      margin: 0 0.5rem;
    }

    .toolbar-btn {
      display: inline-flex;
      align-items: center;
      gap: 0.3rem;
      padding: 5px 10px;
      border: 1px solid var(--color-border);
      border-radius: 6px;
      background: transparent;
      color: var(--color-text-muted);
      font-size: 0.82rem;
      cursor: pointer;
      white-space: nowrap;
      flex-shrink: 0;
      transition: background 0.15s, color 0.15s, border-color 0.15s;
    }

    .toolbar-btn:hover {
      background: var(--color-accent-light);
      color: var(--color-accent);
      border-color: var(--color-accent);
    }

    .toolbar-btn.icon-only { padding: 5px 8px; }

    /* ============================================================
       整体布局
       ============================================================ */
    .app-body {
      display: flex;
      padding-top: var(--toolbar-height);
      min-height: 100vh;
    }

    /* ============================================================
       左侧全局 TOC 侧边栏
       ============================================================ */
    .left-sidebar {
      position: fixed;
      top: var(--toolbar-height);
      left: 0;
      bottom: 0;
      width: var(--left-sidebar-width);
      background: var(--color-bg-sidebar);
      border-right: 1px solid var(--color-border);
      overflow-y: auto;
      overflow-x: hidden;
      z-index: 100;
      transition: transform 0.25s ease;
      scrollbar-width: thin;
      scrollbar-color: var(--color-border) transparent;
    }

    .left-sidebar::-webkit-scrollbar { width: 4px; }
    .left-sidebar::-webkit-scrollbar-thumb { background: var(--color-border); border-radius: 2px; }

    /* 桌面端折叠：推走侧边栏 */
    .left-sidebar.sidebar-collapsed { transform: translateX(calc(-1 * var(--left-sidebar-width))); }

    /* 移动端：默认不显示，点击 hamburger 后滑入 */
    .left-sidebar.mobile-open { transform: translateX(0) !important; }

    .sidebar-header {
      padding: 1rem 1rem 0.5rem;
      font-size: 0.7rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--color-text-muted);
      border-bottom: 1px solid var(--color-border);
    }

    .sidebar-nav { padding: 0.5rem 0 3rem; }

    /* ============================================================
       左侧 TOC 条目样式
       ============================================================ */
    /* toc-group 保留（测试兼容），与 toc-item 共用 */
    .toc-group { position: relative; }

    .toc-row {
      display: flex;
      align-items: center;
    }

    .toc-toggle {
      flex: 0 0 28px;
      height: 30px;
      border: none;
      background: transparent;
      color: var(--color-text-muted);
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 4px;
      margin: 1px 0 1px 4px;
      transition: background 0.15s, color 0.15s;
    }

    /* 折叠/展开箭头（CSS 绘制，无图标字体依赖）*/
    .toc-toggle::after {
      content: '';
      display: block;
      width: 6px;
      height: 6px;
      border-right: 1.5px solid currentColor;
      border-bottom: 1.5px solid currentColor;
      transform: rotate(-45deg) translateY(1px);
      transition: transform 0.2s ease;
    }

    .toc-toggle[aria-expanded="true"]::after {
      transform: rotate(45deg) translateY(-1px);
    }

    .toc-toggle:hover { background: var(--color-sidebar-hover); color: var(--color-text); }

    .toc-spacer { flex: 0 0 28px; margin: 1px 0 1px 4px; }

    .toc-link {
      flex: 1;
      display: block;
      padding: 5px 10px 5px 2px;
      color: var(--color-text-muted);
      text-decoration: none;
      font-size: 0.875rem;
      border-radius: 4px;
      transition: background 0.15s, color 0.15s;
      line-height: 1.4;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      margin: 1px 4px 1px 0;
    }

    .toc-link:hover {
      background: var(--color-sidebar-hover);
      color: var(--color-text);
    }

    .toc-link.active, .toc-link.toc-link-active {
      background: var(--color-sidebar-active-bg);
      color: var(--color-sidebar-active);
      font-weight: 600;
    }

    .toc-link-chapter { font-weight: 500; }

    /* 章节层级缩进 */
    .toc-depth-1 { padding-left: 2px; }
    .toc-depth-2 { padding-left: 10px; }
    .toc-depth-3 { padding-left: 18px; }
    .toc-depth-4 { padding-left: 26px; }

    /* 标题层级缩进 */
    .toc-link-heading { font-size: 0.82rem; }
    .toc-heading-depth-1 { padding-left: 18px; }
    .toc-heading-depth-2 { padding-left: 28px; }
    .toc-heading-depth-3 { padding-left: 36px; font-size: 0.78rem; }

    /* Expand/collapse child list with a smooth max-height transition. */
    .toc-children {
      overflow: hidden;
      transition: max-height 0.3s ease;
    }

    /* ============================================================
       主内容区
       ============================================================ */
    .main-content {
      flex: 1;
      min-width: 0;
      margin-left: var(--left-sidebar-width);
      margin-right: var(--right-sidebar-width);
      transition: margin 0.25s ease;
    }

    /* 左侧边栏折叠时，主内容左扩 */
    .main-content.left-expanded { margin-left: 0; }
    /* 右侧边栏不显示时，主内容右扩 */
    .main-content.right-expanded { margin-right: 0; }

    .content-inner {
      max-width: var(--content-max-width);
      margin: 0 auto;
      padding: 2.5rem 2rem;
    }

    /* ============================================================
       右侧页内 TOC
       ============================================================ */
    .right-sidebar {
      position: fixed;
      top: var(--toolbar-height);
      right: 0;
      bottom: 0;
      width: var(--right-sidebar-width);
      overflow-y: auto;
      padding: 1.25rem 0 3rem;
      z-index: 100;
      border-left: 1px solid var(--color-border);
      background: var(--color-bg);
      scrollbar-width: thin;
      scrollbar-color: var(--color-border) transparent;
    }

    .right-sidebar::-webkit-scrollbar { width: 3px; }
    .right-sidebar::-webkit-scrollbar-thumb { background: var(--color-border); border-radius: 2px; }

    .right-toc-title {
      font-size: 0.7rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--color-text-muted);
      padding: 0 0.75rem 0.5rem;
      margin-bottom: 0.25rem;
    }

    .right-toc-link {
      display: block;
      padding: 3px 0.75rem;
      font-size: 0.8rem;
      color: var(--color-text-muted);
      text-decoration: none;
      border-left: 2px solid transparent;
      transition: all 0.15s ease;
      line-height: 1.5;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .right-toc-link:hover { color: var(--color-text); background: var(--color-sidebar-hover); }

    .right-toc-link.active {
      color: var(--color-sidebar-active);
      border-left-color: var(--color-sidebar-active);
    }

    .right-toc-link.rtoc-d1 { padding-left: 0.75rem; }
    .right-toc-link.rtoc-d2 { padding-left: 1.25rem; font-size: 0.76rem; }
    .right-toc-link.rtoc-d3 { padding-left: 1.75rem; font-size: 0.73rem; }

    /* ============================================================
       章节文章区
       ============================================================ */
    .chapter {
      margin-bottom: 4rem;
      padding-bottom: 2.5rem;
      border-bottom: 1px solid var(--color-border);
    }

    .chapter:last-of-type { border-bottom: none; }

    .chapter-content { line-height: 1.7; }

    /* ============================================================
       排版 - 标题
       ============================================================ */
    .chapter-content h1 {
      font-size: 2em;
      font-weight: 700;
      color: var(--color-heading);
      margin: 0 0 1.25rem;
      padding-bottom: 0.5rem;
      border-bottom: 1px solid var(--color-border);
      line-height: 1.25;
    }

    .chapter-content h2 {
      font-size: 1.5em;
      font-weight: 600;
      color: var(--color-heading);
      margin: 2rem 0 0.75rem;
      padding-bottom: 0.3rem;
      border-bottom: 1px solid var(--color-border);
      line-height: 1.3;
    }

    .chapter-content h3 {
      font-size: 1.25em;
      font-weight: 600;
      color: var(--color-heading);
      margin: 1.75rem 0 0.5rem;
      line-height: 1.35;
    }

    .chapter-content h4, .chapter-content h5, .chapter-content h6 {
      font-size: 1em;
      font-weight: 600;
      color: var(--color-heading);
      margin: 1.5rem 0 0.4rem;
    }

    /* ============================================================
       排版 - 正文
       ============================================================ */
    .chapter-content p { margin: 0 0 1em; }

    .chapter-content ul, .chapter-content ol {
      margin: 0 0 1em;
      padding-left: 1.5em;
    }

    .chapter-content li { margin: 0.3em 0; }
    .chapter-content li > ul, .chapter-content li > ol { margin: 0.3em 0; }

    .chapter-content a {
      color: var(--color-link);
      text-decoration: none;
      border-bottom: 1px solid rgba(9, 105, 218, 0.3);
      transition: color 0.15s, border-color 0.15s;
    }

    .chapter-content a:hover {
      color: var(--color-link-hover);
      border-bottom-color: var(--color-link-hover);
    }

    /* ============================================================
       内联代码
       ============================================================ */
    .chapter-content code {
      font-family: 'JetBrains Mono', 'Fira Code', 'Monaco', 'Consolas', 'Courier New', monospace;
      font-size: 0.875em;
      background: var(--color-code-bg);
      color: var(--color-code-text);
      padding: 0.15em 0.4em;
      border-radius: 4px;
      border: 1px solid var(--color-code-border);
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    /* ============================================================
       代码块（带语言标签 + 复制按钮）
       ============================================================ */
    .code-block-wrapper {
      position: relative;
      margin: 1.25rem 0;
      border-radius: 8px;
      border: 1px solid var(--color-code-border);
      overflow: hidden;
      box-shadow: var(--shadow-sm);
    }

    .code-block-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 6px 12px;
      background: var(--color-code-bg);
      border-bottom: 1px solid var(--color-code-border);
    }

    .code-lang-label {
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      color: var(--color-code-lang);
    }

    .code-copy-btn {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 2px 8px;
      border: 1px solid var(--color-code-border);
      border-radius: 4px;
      background: transparent;
      color: var(--color-text-muted);
      font-size: 0.72rem;
      cursor: pointer;
      transition: all 0.15s ease;
    }

    .code-copy-btn:hover {
      background: var(--color-accent-light);
      color: var(--color-accent);
      border-color: var(--color-accent);
    }

    .code-copy-btn.copied { color: #22c55e; border-color: #22c55e; }

    .code-block-wrapper pre {
      margin: 0;
      padding: 1rem 1.25rem;
      background: var(--color-code-bg);
      overflow-x: auto;
      border: none;
      border-radius: 0;
      font-size: 0.875em;
      line-height: 1.6;
    }

    .code-block-wrapper pre code {
      background: transparent;
      border: none;
      padding: 0;
      font-size: inherit;
      color: var(--color-text);
      border-radius: 0;
    }

    /* 未包装的独立 pre（无语言标识时的 fallback）*/
    .chapter-content pre {
      border-radius: 8px;
      border: 1px solid var(--color-code-border);
      padding: 1rem 1.25rem;
      overflow-x: auto;
      margin: 1.25rem 0;
      background: var(--color-code-bg);
      line-height: 1.6;
      box-shadow: var(--shadow-sm);
    }

    .chapter-content pre code {
      background: transparent;
      border: none;
      padding: 0;
      color: var(--color-text);
    }

    /* ============================================================
       Callout 提示框（由 JS 从 blockquote 转换生成）
       ============================================================ */
    .callout {
      display: flex;
      gap: 0.75rem;
      padding: 0.875rem 1rem;
      border-radius: 8px;
      border-left: 4px solid;
      margin: 1.25rem 0;
    }

    .callout-icon { font-size: 1.1em; flex-shrink: 0; margin-top: 2px; }
    .callout-body { flex: 1; min-width: 0; }
    .callout-body > *:last-child { margin-bottom: 0; }

    .callout-note     { background: var(--callout-note-bg);      border-color: var(--callout-note-border);      color: var(--callout-note-color); }
    .callout-warning  { background: var(--callout-warning-bg);   border-color: var(--callout-warning-border);  color: var(--callout-warning-color); }
    .callout-tip      { background: var(--callout-tip-bg);       border-color: var(--callout-tip-border);      color: var(--callout-tip-color); }
    .callout-important { background: var(--callout-important-bg); border-color: var(--callout-important-border); color: var(--callout-important-color); }

    /* 普通 blockquote（非 callout）*/
    .chapter-content blockquote {
      border-left: 4px solid var(--color-accent);
      margin: 1.25rem 0;
      padding: 0.5rem 1rem;
      background: var(--color-bg-alt);
      border-radius: 0 6px 6px 0;
      color: var(--color-text-muted);
    }

    .chapter-content blockquote p { margin-bottom: 0.25em; }

    /* ============================================================
       表格（带斑马纹、hover 高亮）
       ============================================================ */
    .table-wrapper {
      overflow-x: auto;
      margin: 1.25rem 0;
      border-radius: 8px;
      border: 1px solid var(--color-border);
      box-shadow: var(--shadow-sm);
    }

    .chapter-content table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.9em;
    }

    .chapter-content table th {
      background: var(--color-table-header);
      border: none;
      border-bottom: 2px solid var(--color-border);
      padding: 10px 14px;
      text-align: left;
      font-weight: 600;
      color: var(--color-heading);
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    .chapter-content table td {
      border: none;
      border-bottom: 1px solid var(--color-border);
      padding: 8px 14px;
      vertical-align: top;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    .chapter-content table tbody tr:last-child td { border-bottom: none; }
    .chapter-content table tbody tr:nth-child(even) { background: var(--color-table-stripe-alt); }
    .chapter-content table tbody tr:hover { background: var(--color-table-hover); }

    /* ============================================================
       图片（圆角、阴影、灯箱交互）
       ============================================================ */
    .chapter-content img {
      max-width: 100%;
      height: auto;
      display: block;
      margin: 1.5rem auto;
      border-radius: 8px;
      box-shadow: var(--shadow-md);
      cursor: zoom-in;
    }

    /* ============================================================
       水平线
       ============================================================ */
    .chapter-content hr {
      border: none;
      height: 1px;
      background: var(--color-border);
      margin: 2rem 0;
    }

    /* ============================================================
       搜索高亮
       ============================================================ */
    .search-highlight {
      background: rgba(255, 220, 0, 0.4);
      border-radius: 2px;
      padding: 0 1px;
    }

    /* ============================================================
       上一页/下一页章节导航
       ============================================================ */
    .chapter-nav {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
      margin-top: 2.5rem;
      padding-top: 1.5rem;
      border-top: 1px solid var(--color-border);
    }

    .chapter-nav-btn {
      display: flex;
      flex-direction: column;
      gap: 0.2rem;
      padding: 0.875rem 1rem;
      border: 1px solid var(--color-border);
      border-radius: 8px;
      text-decoration: none;
      color: var(--color-text);
      background: var(--color-bg-alt);
      transition: border-color 0.2s, background 0.2s, transform 0.15s;
    }

    .chapter-nav-btn:hover {
      border-color: var(--color-accent);
      background: var(--color-accent-light);
      transform: translateY(-1px);
      box-shadow: var(--shadow-md);
    }

    .chapter-nav-btn.next { text-align: right; }

    .chapter-nav-label {
      font-size: 0.72rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      color: var(--color-text-muted);
    }

    .chapter-nav-title {
      font-size: 0.88rem;
      font-weight: 600;
      color: var(--color-link);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    /* ============================================================
       搜索模态框
       ============================================================ */
    .search-overlay {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.5);
      z-index: 5000;
      align-items: flex-start;
      justify-content: center;
      padding-top: 8vh;
    }

    .search-overlay.visible { display: flex; }

    .search-modal {
      width: 100%;
      max-width: 580px;
      background: var(--color-bg);
      border: 1px solid var(--color-border);
      border-radius: 12px;
      box-shadow: 0 24px 64px rgba(0,0,0,0.3);
      overflow: hidden;
      margin: 0 1rem;
    }

    .search-input-row {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.75rem 1rem;
      border-bottom: 1px solid var(--color-border);
    }

    .search-icon-glyph { color: var(--color-text-muted); flex-shrink: 0; }

    .search-input {
      flex: 1;
      border: none;
      background: transparent;
      color: var(--color-text);
      font-size: 1rem;
      outline: none;
      caret-color: var(--color-accent);
    }

    .search-input::placeholder { color: var(--color-text-muted); }

    .search-kbd {
      font-size: 0.72rem;
      color: var(--color-text-muted);
      background: var(--color-bg-alt);
      border: 1px solid var(--color-border);
      border-radius: 4px;
      padding: 2px 6px;
      flex-shrink: 0;
      font-family: inherit;
    }

    .search-results-list { max-height: 60vh; overflow-y: auto; }

    .search-result {
      padding: 10px 1rem;
      cursor: pointer;
      border-bottom: 1px solid var(--color-border);
      transition: background 0.1s;
    }

    .search-result:last-child { border-bottom: none; }
    .search-result:hover, .search-result.focused { background: var(--color-accent-light); }

    .search-result-title {
      font-weight: 600;
      font-size: 0.875rem;
      color: var(--color-heading);
      margin-bottom: 2px;
    }

    .search-result-excerpt {
      font-size: 0.8rem;
      color: var(--color-text-muted);
      line-height: 1.4;
      overflow: hidden;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
    }

    .search-result-excerpt mark {
      background: rgba(255, 220, 0, 0.4);
      color: inherit;
      border-radius: 2px;
      padding: 0 1px;
    }

    .search-footer {
      padding: 6px 1rem;
      font-size: 0.72rem;
      color: var(--color-text-muted);
      background: var(--color-bg-alt);
      border-top: 1px solid var(--color-border);
      display: flex;
      justify-content: space-between;
    }

    .search-no-results {
      padding: 2rem 1rem;
      text-align: center;
      color: var(--color-text-muted);
      font-size: 0.9rem;
    }

    /* ============================================================
       图片灯箱
       ============================================================ */
    .img-lightbox {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.92);
      z-index: 9000;
      display: none;
      align-items: center;
      justify-content: center;
      cursor: zoom-out;
    }

    .img-lightbox.visible { display: flex; }

    .img-lightbox img {
      max-width: 90vw;
      max-height: 90vh;
      border-radius: 6px;
      object-fit: contain;
      box-shadow: 0 8px 40px rgba(0,0,0,0.5);
      cursor: default;
    }

    /* ============================================================
       回到顶部按钮
       ============================================================ */
    #back-to-top {
      position: fixed;
      bottom: 1.5rem;
      right: 1.5rem;
      width: 40px;
      height: 40px;
      border-radius: 50%;
      background: var(--color-accent);
      color: #fff;
      border: none;
      cursor: pointer;
      display: none;
      align-items: center;
      justify-content: center;
      font-size: 1rem;
      box-shadow: var(--shadow-md);
      transition: transform 0.2s, opacity 0.2s;
      z-index: 500;
    }

    #back-to-top.visible { display: flex; }
    #back-to-top:hover { transform: translateY(-2px); opacity: 0.9; }

    /* ============================================================
       移动端遮罩层
       ============================================================ */
    .sidebar-overlay {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.4);
      z-index: 99;
    }

    .sidebar-overlay.visible { display: block; }

    /* ============================================================
       响应式断点
       ============================================================ */

    /* 中等屏幕：隐藏右侧 TOC */
    @media (max-width: 1200px) {
      .right-sidebar { display: none; }
      .main-content { margin-right: 0; }
      .main-content.right-expanded { margin-right: 0; }
    }

    /* 小屏幕：左侧边栏默认隐藏，hamburger 控制 */
    @media (max-width: 768px) {
      .left-sidebar {
        transform: translateX(calc(-1 * var(--left-sidebar-width)));
        z-index: 200;
      }
      .main-content { margin-left: 0 !important; margin-right: 0 !important; }
      .content-inner { padding: 1.5rem 1rem; }
      .chapter-nav { grid-template-columns: 1fr; }
      .chapter-nav-btn.next { text-align: left; }
    }

    /* ============================================================
       打印样式
       ============================================================ */
    @media print {
      #reading-progress, .toolbar, .left-sidebar, .right-sidebar,
      #back-to-top, .search-overlay, .sidebar-overlay,
      .chapter-nav, .img-lightbox { display: none !important; }

      .app-body { display: block; padding-top: 0; }
      .main-content { margin: 0 !important; }
      .content-inner { max-width: none; padding: 0; }

      .chapter { page-break-before: always; border: none; margin: 0; padding: 0; }
      .chapter:first-child { page-break-before: avoid; }

      body { color: #000; background: #fff; }
      a { color: #000; text-decoration: underline; }
      h1, h2, h3, h4, h5, h6 { page-break-after: avoid; color: #000; }
      pre, figure, table { page-break-inside: avoid; }
      pre { border: 1px solid #ccc; }
    }

    @page { size: A4; margin: 2cm; }

    /* ============================================================
       自定义主题覆盖
       ============================================================ */
    {{.CSS}}
  </style>
</head>
<body>
  <!-- 阅读进度条 -->
  <div id="reading-progress"></div>

  <!-- 顶部工具栏 -->
  <header class="toolbar">
    <button class="toolbar-btn icon-only" id="btn-sidebar" title="切换目录" aria-label="切换目录">☰</button>
    <a class="toolbar-brand" href="#">{{.Title}}</a>
    <button class="toolbar-btn" id="btn-search" title="全文搜索 (⌘K / Ctrl+K)" aria-label="搜索">🔍 搜索</button>
    <button class="toolbar-btn icon-only" id="btn-theme" title="切换主题" aria-label="切换主题">🌙</button>
  </header>

  <!-- 移动端侧边栏遮罩 -->
  <div class="sidebar-overlay" id="sidebar-overlay"></div>

  <div class="app-body">
    <!-- 左侧全局目录 -->
    <nav class="left-sidebar" id="left-sidebar" aria-label="全局目录导航">
      <div class="sidebar-header">目录</div>
      <div class="sidebar-nav" id="sidebar-nav">
        {{safeHTML .SidebarHTML}}
      </div>
    </nav>

    <!-- 主内容区 -->
    <main class="main-content" id="main-content" role="main">
      <div class="content-inner">
        {{range .Chapters}}
        <article class="chapter" id="{{.ID}}">
          <div class="chapter-content">
            {{.Content}}
          </div>

          <!-- 上一页/下一页导航 -->
          <nav class="chapter-nav" aria-label="章节导航">
            {{if .PrevID}}
            <a href="#{{.PrevID}}" class="chapter-nav-btn prev" aria-label="上一节：{{.PrevTitle}}">
              <span class="chapter-nav-label">← 上一节</span>
              <span class="chapter-nav-title">{{.PrevTitle}}</span>
            </a>
            {{else}}
            <div></div>
            {{end}}
            {{if .NextID}}
            <a href="#{{.NextID}}" class="chapter-nav-btn next" aria-label="下一节：{{.NextTitle}}">
              <span class="chapter-nav-label">下一节 →</span>
              <span class="chapter-nav-title">{{.NextTitle}}</span>
            </a>
            {{else}}
            <div></div>
            {{end}}
          </nav>
        </article>
        {{end}}

        <!-- 页脚 -->
        <footer style="margin-top:3rem;padding-top:1.5rem;border-top:1px solid var(--color-border);text-align:center;color:var(--color-text-muted);font-size:0.8rem;">
          <a href="https://github.com/yeasy/mdpress" target="_blank" rel="noopener noreferrer" style="color:inherit;border-bottom:none;">Built with mdpress</a>
        </footer>
      </div>
    </main>

    <!-- 右侧页内 TOC -->
    <aside class="right-sidebar" id="right-sidebar" aria-label="页内目录">
      <div class="right-toc-title">本页目录</div>
      <nav id="right-toc-nav" aria-label="页内标题导航"></nav>
    </aside>
  </div>

  <!-- 搜索模态框 -->
  <div class="search-overlay" id="search-overlay" role="dialog" aria-modal="true" aria-label="搜索">
    <div class="search-modal">
      <div class="search-input-row">
        <span class="search-icon-glyph" aria-hidden="true">🔍</span>
        <input class="search-input" id="search-input" type="text"
               placeholder="搜索内容..." autocomplete="off" autocorrect="off" spellcheck="false">
        <kbd class="search-kbd">Esc</kbd>
      </div>
      <div class="search-results-list" id="search-results-list"></div>
      <div class="search-footer">
        <span id="search-count-label"></span>
        <span>↑↓ 导航 · Enter 跳转 · Esc 关闭</span>
      </div>
    </div>
  </div>

  <!-- 图片灯箱 -->
  <div class="img-lightbox" id="img-lightbox" role="dialog" aria-modal="true" aria-label="图片预览">
    <img id="img-lightbox-src" src="" alt="">
  </div>

  <!-- 回到顶部 -->
  <button id="back-to-top" aria-label="回到顶部">↑</button>

  <script>
  (function() {
    'use strict';

    // ============================================================
    // 主题管理：三档切换（light / dark / system），无闪烁
    // ============================================================
    var THEME_KEY = 'mdpress-theme';
    var themeBtn  = document.getElementById('btn-theme');
    var themes    = ['light', 'dark', 'system'];
    var themeIcons  = { light: '☀️', dark: '🌙', system: '🖥' };
    var themeLabels = { light: '亮色', dark: '暗色', system: '跟随系统' };
    var currentTheme = localStorage.getItem(THEME_KEY) || 'system';

    function applyTheme(t) {
      currentTheme = t;
      try { localStorage.setItem(THEME_KEY, t); } catch(e) {}
      var dark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
      document.documentElement.setAttribute('data-theme', dark ? 'dark' : '');
      themeBtn.textContent = themeIcons[t];
      themeBtn.title = '主题：' + themeLabels[t] + '（点击切换）';
    }

    themeBtn.addEventListener('click', function() {
      var idx = themes.indexOf(currentTheme);
      applyTheme(themes[(idx + 1) % themes.length]);
    });

    // 监听系统主题变化（仅在 system 模式下生效）
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
      if (currentTheme === 'system') applyTheme('system');
    });

    applyTheme(currentTheme);

    // ============================================================
    // 侧边栏控制（桌面推拉 + 移动端滑入）
    // ============================================================
    var leftSidebar   = document.getElementById('left-sidebar');
    var mainContent   = document.getElementById('main-content');
    var sidebarOverlay = document.getElementById('sidebar-overlay');
    var sidebarHidden = false;

    function isMobile() { return window.innerWidth <= 768; }

    function showSidebar() {
      sidebarHidden = false;
      if (isMobile()) {
        leftSidebar.classList.add('mobile-open');
        sidebarOverlay.classList.add('visible');
      } else {
        leftSidebar.classList.remove('sidebar-collapsed');
        mainContent.classList.remove('left-expanded');
      }
    }

    function hideSidebar() {
      sidebarHidden = true;
      if (isMobile()) {
        leftSidebar.classList.remove('mobile-open');
        sidebarOverlay.classList.remove('visible');
      } else {
        leftSidebar.classList.add('sidebar-collapsed');
        mainContent.classList.add('left-expanded');
      }
    }

    document.getElementById('btn-sidebar').addEventListener('click', function() {
      sidebarHidden ? showSidebar() : hideSidebar();
    });

    sidebarOverlay.addEventListener('click', hideSidebar);

    // 移动端点击链接后自动关闭侧边栏
    leftSidebar.addEventListener('click', function(e) {
      if (e.target.tagName === 'A' && isMobile()) hideSidebar();
    });

    // ============================================================
    // 左侧 TOC 折叠/展开（带过渡动画，支持多个章节同时展开）
    // ============================================================

    // 展开子章节列表（带 max-height 动画）
    function expandTocGroup(children, btn) {
      children.style.maxHeight = '0';
      children.removeAttribute('hidden');
      // 强制重排，确保 CSS transition 生效
      void children.offsetHeight;
      children.style.maxHeight = children.scrollHeight + 'px';
      if (btn) btn.setAttribute('aria-expanded', 'true');
      // 动画结束后清除 maxHeight，允许嵌套展开时重新计算高度
      children.addEventListener('transitionend', function onEnd() {
        children.removeEventListener('transitionend', onEnd);
        if (!children.hidden) children.style.maxHeight = '';
      });
    }

    // 折叠子章节列表（带 max-height 动画）
    function collapseTocGroup(children, btn) {
      children.style.maxHeight = children.scrollHeight + 'px';
      void children.offsetHeight;
      children.style.maxHeight = '0';
      if (btn) btn.setAttribute('aria-expanded', 'false');
      children.addEventListener('transitionend', function onEnd() {
        children.removeEventListener('transitionend', onEnd);
        children.setAttribute('hidden', '');
        children.style.maxHeight = '';
      });
    }

    document.querySelectorAll('.toc-toggle').forEach(function(btn) {
      btn.addEventListener('click', function(e) {
        e.stopPropagation();
        var item = btn.closest('.toc-group');
        var children = item ? item.querySelector('.toc-children') : null;
        if (!children) return;
        var expanded = btn.getAttribute('aria-expanded') === 'true';
        // 直接切换当前章节，不关闭其他已展开章节（非手风琴模式）
        if (expanded) {
          collapseTocGroup(children, btn);
        } else {
          expandTocGroup(children, btn);
        }
      });
    });

    // ============================================================
    // 平滑滚动：拦截 TOC 链接和章节导航按钮的点击，防止页面闪烁
    // ============================================================
    function handleAnchorClick(e) {
      var href = this.getAttribute('href');
      if (!href || href.charAt(0) !== '#') return;
      var targetId = href.slice(1);
      if (!document.getElementById(targetId)) return;
      e.preventDefault();
      // 使用 JS 平滑滚动，避免浏览器默认的瞬间跳转（闪烁）
      document.getElementById(targetId).scrollIntoView({ behavior: 'smooth', block: 'start' });
      // 更新地址栏 hash，不触发浏览器默认跳转
      if (history.pushState) history.pushState(null, '', href);
    }

    function initSmoothNav() {
      // 侧边栏目录链接
      document.querySelectorAll('#sidebar-nav .toc-link').forEach(function(link) {
        link.addEventListener('click', handleAnchorClick);
      });
      // 上一页/下一页章节导航按钮
      document.querySelectorAll('.chapter-nav-btn').forEach(function(link) {
        link.addEventListener('click', handleAnchorClick);
      });
    }

    // ============================================================
    // Scroll Spy: highlight left TOC + update right TOC.
    // Uses IntersectionObserver so updates fire only when elements enter or
    // leave the observation zone, eliminating per-frame flicker that occurs
    // with scroll-event polling during smooth navigation.
    // ============================================================
    var activeChapterId = '';
    var activeHeadingId = '';

    function initScrollSpy() {
      var chapters = Array.from(document.querySelectorAll('.chapter'));
      var headings = Array.from(
        document.querySelectorAll('.chapter-content h1[id], .chapter-content h2[id], .chapter-content h3[id], .chapter-content h4[id]')
      );

      // Pre-map each heading to its parent chapter id for O(1) lookup.
      headings.forEach(function(h) {
        h._chapterId = h.closest('.chapter') ? h.closest('.chapter').id : '';
      });

      // Visibility state: true when element is inside the observation zone.
      var visibleHeadings = {};
      var visibleChapters = {};

      // Determine which chapter/heading is currently active and push updates
      // to the left and right TOC components only when the state actually changes.
      function syncActive() {
        // The topmost visible heading (first in DOM order) wins.
        var newHeadingId = '';
        var newChapterId = '';
        for (var i = 0; i < headings.length; i++) {
          if (visibleHeadings[headings[i].id]) {
            newHeadingId = headings[i].id;
            newChapterId = headings[i]._chapterId;
            break;
          }
        }
        // No heading in zone — fall back to the topmost visible chapter.
        if (!newChapterId) {
          for (var j = 0; j < chapters.length; j++) {
            if (visibleChapters[chapters[j].id]) { newChapterId = chapters[j].id; break; }
          }
          if (!newChapterId && chapters.length > 0) newChapterId = chapters[0].id;
        }

        if (newChapterId !== activeChapterId || newHeadingId !== activeHeadingId) {
          activeChapterId = newChapterId;
          activeHeadingId = newHeadingId;
          updateLeftTOC(newChapterId, newHeadingId);
          updateRightTOC(newChapterId, newHeadingId);
        }
      }

      // Observe headings in a band from 80 px below the viewport top (below
      // the fixed toolbar) down to 50 % up from the bottom.  The observer
      // fires only on entry/exit — not on every scroll frame.
      var headingObserver = new IntersectionObserver(function(entries) {
        entries.forEach(function(e) { visibleHeadings[e.target.id] = e.isIntersecting; });
        syncActive();
      }, { rootMargin: '-80px 0px -50% 0px', threshold: 0 });

      headings.forEach(function(h) { headingObserver.observe(h); });

      // Observe chapters with a wider band to handle chapters that have no headings.
      var chapterObserver = new IntersectionObserver(function(entries) {
        entries.forEach(function(e) { visibleChapters[e.target.id] = e.isIntersecting; });
        syncActive();
      }, { rootMargin: '-80px 0px -20% 0px', threshold: 0 });

      chapters.forEach(function(ch) { chapterObserver.observe(ch); });
    }

    // 更新左侧 TOC 高亮
    function updateLeftTOC(chapterId, headingId) {
      var activeTarget = headingId || chapterId;
      document.querySelectorAll('#sidebar-nav .toc-link').forEach(function(link) {
        var target = link.getAttribute('data-target');
        link.classList.toggle('active', target === activeTarget);
      });

      // 展开包含活跃链接的章节组
      var activeLink = document.querySelector('#sidebar-nav .toc-link.active');
      if (activeLink) {
        var group = activeLink.closest('.toc-group');
        while (group) {
          var toggle = group.querySelector(':scope > .toc-row > .toc-toggle');
          var children = group.querySelector(':scope > .toc-children');
          if (toggle && children && children.hidden) {
            // 带动画展开（scroll spy 触发时同样使用过渡效果）
            expandTocGroup(children, toggle);
          }
          var parent = group.parentElement;
          group = parent ? parent.closest('.toc-group') : null;
        }
      }
    }

    // 右侧页内 TOC 缓存（避免每帧重建 DOM）
    var rightTOCCache = {};
    var currentRightChapter = '';

    function updateRightTOC(chapterId, headingId) {
      var rightNav = document.getElementById('right-toc-nav');
      if (!rightNav) return;

      // 章节切换时重新构建右侧 TOC 链接列表
      if (chapterId !== currentRightChapter) {
        currentRightChapter = chapterId;
        if (!rightTOCCache[chapterId]) {
          var chapter = document.getElementById(chapterId);
          rightTOCCache[chapterId] = chapter
            ? Array.from(chapter.querySelectorAll('.chapter-content h1[id], .chapter-content h2[id], .chapter-content h3[id]'))
                .map(function(h) { return { id: h.id, text: h.textContent, level: parseInt(h.tagName.slice(1)) }; })
            : [];
        }
        rightNav.innerHTML = '';
        rightTOCCache[chapterId].forEach(function(item) {
          var a = document.createElement('a');
          a.href = '#' + item.id;
          a.className = 'right-toc-link rtoc-d' + (item.level - 1);
          a.setAttribute('data-target', item.id);
          a.textContent = item.text;
          // Use smooth scroll + history.pushState (same behaviour as left TOC links).
          a.addEventListener('click', handleAnchorClick);
          rightNav.appendChild(a);
        });
      }

      // 高亮当前标题
      rightNav.querySelectorAll('.right-toc-link').forEach(function(link) {
        link.classList.toggle('active', link.getAttribute('data-target') === headingId);
      });
    }

    // onScroll only handles the reading progress bar and the back-to-top button.
    // Chapter/heading tracking is now handled by IntersectionObserver in initScrollSpy.
    function onScroll() {
      var scrollTop = window.scrollY || document.documentElement.scrollTop;
      var docH = document.documentElement.scrollHeight - window.innerHeight;
      var pct  = docH > 0 ? Math.min(100, (scrollTop / docH) * 100) : 0;
      document.getElementById('reading-progress').style.width = pct + '%';
      document.getElementById('back-to-top').classList.toggle('visible', scrollTop > 400);
    }

    // Throttle scroll events with requestAnimationFrame to avoid excessive repaints.
    var rafPending = false;
    window.addEventListener('scroll', function() {
      if (rafPending) return;
      rafPending = true;
      requestAnimationFrame(function() { onScroll(); rafPending = false; });
    }, { passive: true });

    // ============================================================
    // 代码块增强：自动包装 pre > code，添加语言标签和复制按钮
    // ============================================================
    function enhanceCodeBlocks() {
      document.querySelectorAll('.chapter-content pre').forEach(function(pre) {
        // 已处理过的跳过（幂等）
        if (pre.parentElement && pre.parentElement.classList.contains('code-block-wrapper')) return;

        var code = pre.querySelector('code');
        var lang = '';
        if (code) {
          Array.from(code.classList).some(function(cls) {
            var m = cls.match(/^language-(.+)$/);
            if (m) { lang = m[1]; return true; }
            return false;
          });
        }

        // 创建包装容器
        var wrapper = document.createElement('div');
        wrapper.className = 'code-block-wrapper';

        // 头部：语言标签 + 复制按钮
        var header = document.createElement('div');
        header.className = 'code-block-header';

        var langLabel = document.createElement('span');
        langLabel.className = 'code-lang-label';
        langLabel.textContent = lang || 'text';

        var copyBtn = document.createElement('button');
        copyBtn.className = 'code-copy-btn';
        copyBtn.textContent = '复制';
        copyBtn.title = '复制代码';
        copyBtn.setAttribute('aria-label', '复制代码');

        // 复制逻辑（优先 navigator.clipboard，降级 execCommand）
        copyBtn.addEventListener('click', function() {
          var text = code ? code.textContent : pre.textContent;
          var doFeedback = function() {
            copyBtn.textContent = '已复制 ✓';
            copyBtn.classList.add('copied');
            setTimeout(function() {
              copyBtn.textContent = '复制';
              copyBtn.classList.remove('copied');
            }, 2000);
          };
          if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(doFeedback).catch(function() {
              fallbackCopy(text, doFeedback);
            });
          } else {
            fallbackCopy(text, doFeedback);
          }
        });

        header.appendChild(langLabel);
        header.appendChild(copyBtn);
        wrapper.appendChild(header);

        // 将原 pre 移入 wrapper
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
      });
    }

    // execCommand 降级复制（用于不支持 Clipboard API 的环境）
    function fallbackCopy(text, cb) {
      var ta = document.createElement('textarea');
      ta.value = text;
      ta.style.cssText = 'position:fixed;top:-9999px;opacity:0';
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand('copy'); } catch(e) {}
      document.body.removeChild(ta);
      if (cb) cb();
    }

    // ============================================================
    // Callout 提示框：将特定格式的 blockquote 转换为彩色提示框
    //
    // 支持格式（Markdown 中）：
    //   > **Note**: 内容
    //   > **Warning**: 内容
    //   > **Tip**: 内容
    //   > **Important**: 内容
    // ============================================================
    var CALLOUT_MAP = {
      'Note':      { type: 'note',      icon: 'ℹ️' },
      'Warning':   { type: 'warning',   icon: '⚠️' },
      'Tip':       { type: 'tip',       icon: '💡' },
      'Important': { type: 'important', icon: '❗' },
      'Danger':    { type: 'important', icon: '🚨' },
      '注意':      { type: 'note',      icon: 'ℹ️' },
      '警告':      { type: 'warning',   icon: '⚠️' },
      '提示':      { type: 'tip',       icon: '💡' },
      '重要':      { type: 'important', icon: '❗' },
    };

    function transformCallouts() {
      document.querySelectorAll('.chapter-content blockquote').forEach(function(bq) {
        var firstP = bq.querySelector('p:first-child');
        if (!firstP) return;

        var firstStrong = firstP.querySelector('strong:first-child');
        if (!firstStrong) return;

        var keyword = firstStrong.textContent.replace(/:$/, '').trim();
        var info    = CALLOUT_MAP[keyword];
        if (!info)  return;

        // 构建 callout 容器
        var callout = document.createElement('div');
        callout.className = 'callout callout-' + info.type;

        var icon = document.createElement('span');
        icon.className = 'callout-icon';
        icon.setAttribute('aria-hidden', 'true');
        icon.textContent = info.icon;

        var body = document.createElement('div');
        body.className = 'callout-body';

        // 清理 strong 标签和紧跟的冒号/空格
        firstStrong.remove();
        var firstTextNode = firstP.firstChild;
        if (firstTextNode && firstTextNode.nodeType === 3) {
          firstTextNode.textContent = firstTextNode.textContent.replace(/^[:\s]+/, '');
        }

        // 将 blockquote 中的内容移入 body
        while (bq.firstChild) body.appendChild(bq.firstChild);

        callout.appendChild(icon);
        callout.appendChild(body);
        bq.parentNode.replaceChild(callout, bq);
      });
    }

    // ============================================================
    // 表格包装：使宽表格可横向滚动
    // ============================================================
    function wrapTables() {
      document.querySelectorAll('.chapter-content table').forEach(function(table) {
        if (table.parentElement && table.parentElement.classList.contains('table-wrapper')) return;
        var wrapper = document.createElement('div');
        wrapper.className = 'table-wrapper';
        table.parentNode.insertBefore(wrapper, table);
        wrapper.appendChild(table);
      });
    }

    // ============================================================
    // 图片灯箱：点击图片全屏查看
    // ============================================================
    var lightbox    = document.getElementById('img-lightbox');
    var lightboxImg = document.getElementById('img-lightbox-src');

    function openLightbox(src, alt) {
      lightboxImg.src = src;
      lightboxImg.alt = alt || '';
      lightbox.classList.add('visible');
      document.body.style.overflow = 'hidden';
    }

    function closeLightbox() {
      lightbox.classList.remove('visible');
      document.body.style.overflow = '';
      // 延迟清空 src，避免图片闪烁
      setTimeout(function() { if (!lightbox.classList.contains('visible')) lightboxImg.src = ''; }, 300);
    }

    lightbox.addEventListener('click', function(e) {
      if (e.target !== lightboxImg) closeLightbox();
    });

    function initLightbox() {
      document.querySelectorAll('.chapter-content img').forEach(function(img) {
        img.addEventListener('click', function() { openLightbox(img.src, img.alt); });
      });
    }

    // ============================================================
    // 全文搜索（⌘K / Ctrl+K 打开模态框，支持中文）
    // ============================================================
    var searchOverlay     = document.getElementById('search-overlay');
    var searchInput       = document.getElementById('search-input');
    var searchResultsList = document.getElementById('search-results-list');
    var searchCountLabel  = document.getElementById('search-count-label');
    var searchFocusIdx    = -1;

    function openSearch() {
      searchOverlay.classList.add('visible');
      searchInput.focus();
      searchInput.select();
    }

    function closeSearch() {
      searchOverlay.classList.remove('visible');
      searchFocusIdx = -1;
    }

    document.getElementById('btn-search').addEventListener('click', openSearch);

    searchOverlay.addEventListener('click', function(e) {
      if (e.target === searchOverlay) closeSearch();
    });

    document.addEventListener('keydown', function(e) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        searchOverlay.classList.contains('visible') ? closeSearch() : openSearch();
        return;
      }
      if (e.key === 'Escape') {
        if (lightbox.classList.contains('visible')) { closeLightbox(); return; }
        closeSearch();
        return;
      }
      if (!searchOverlay.classList.contains('visible')) return;
      if (e.key === 'ArrowDown') { e.preventDefault(); moveFocus(1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); moveFocus(-1); }
      else if (e.key === 'Enter') { e.preventDefault(); activateFocused(); }
    });

    function moveFocus(delta) {
      var items = searchResultsList.querySelectorAll('.search-result');
      if (!items.length) return;
      searchFocusIdx = Math.max(0, Math.min(items.length - 1, searchFocusIdx + delta));
      items.forEach(function(item, i) { item.classList.toggle('focused', i === searchFocusIdx); });
    }

    function activateFocused() {
      var item = searchResultsList.querySelector('.search-result.focused');
      if (item) {
        scrollToId(item.getAttribute('data-target'));
        closeSearch();
      }
    }

    function scrollToId(id) {
      var el = document.getElementById(id);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    function escapeRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

    var searchTimer = null;
    searchInput.addEventListener('input', function() {
      searchFocusIdx = -1;
      if (searchTimer) clearTimeout(searchTimer);
      searchTimer = setTimeout(doSearch, 200);
    });

    function doSearch() {
      var query = searchInput.value.trim();
      searchResultsList.innerHTML = '';
      if (!query) { searchCountLabel.textContent = ''; return; }

      var re = new RegExp(escapeRe(query), 'gi');
      var results = [];

      document.querySelectorAll('.chapter').forEach(function(chapter) {
        if (results.length >= 50) return;
        var content = chapter.querySelector('.chapter-content');
        if (!content) return;

        var walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT, {
          acceptNode: function(node) {
            var tag = node.parentElement ? node.parentElement.tagName : '';
            if (tag === 'SCRIPT' || tag === 'STYLE') return NodeFilter.FILTER_REJECT;
            return NodeFilter.FILTER_ACCEPT;
          }
        });

        var seen = new Set();
        var node;
        while ((node = walker.nextNode()) && results.length < 50) {
          var text = node.textContent;
          if (!re.test(text)) { re.lastIndex = 0; continue; }
          re.lastIndex = 0;

          var match;
          while ((match = re.exec(text)) !== null && results.length < 50) {
            var s = Math.max(0, match.index - 40);
            var e = Math.min(text.length, match.index + query.length + 40);
            var excerpt = (s > 0 ? '…' : '') + text.slice(s, e) + (e < text.length ? '…' : '');

            // 找最近标题作为结果标题
            var nearH = node.parentElement ? node.parentElement.closest('h1,h2,h3,h4') : null;
            var itemTitle  = nearH ? nearH.textContent : (chapter.querySelector('.chapter-content h1,h2,h3') || {textContent: chapter.id}).textContent;
            var targetId   = nearH ? nearH.id : chapter.id;

            var key = targetId + '|' + excerpt.slice(0, 20);
            if (!seen.has(key)) {
              seen.add(key);
              results.push({ title: itemTitle, excerpt: excerpt, targetId: targetId });
            }
          }
          re.lastIndex = 0;
        }
      });

      searchCountLabel.textContent = results.length + ' 条结果' + (results.length >= 50 ? '（前 50 条）' : '');

      if (!results.length) {
        var q = query.replace(/</g, '&lt;');
        searchResultsList.innerHTML = '<div class="search-no-results">未找到与 "' + q + '" 相关的内容</div>';
        return;
      }

      var re2 = new RegExp('(' + escapeRe(query) + ')', 'gi');
      results.forEach(function(r, i) {
        var div = document.createElement('div');
        div.className = 'search-result';
        div.setAttribute('data-target', r.targetId);

        var title = document.createElement('div');
        title.className = 'search-result-title';
        title.textContent = r.title;

        var excerpt = document.createElement('div');
        excerpt.className = 'search-result-excerpt';
        excerpt.innerHTML = r.excerpt.replace(/</g, '&lt;').replace(re2, '<mark>$1</mark>');

        div.appendChild(title);
        div.appendChild(excerpt);
        div.addEventListener('click', function() {
          scrollToId(r.targetId);
          closeSearch();
        });
        searchResultsList.appendChild(div);
      });
    }

    // ============================================================
    // 回到顶部按钮
    // ============================================================
    document.getElementById('back-to-top').addEventListener('click', function() {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    });

    // ============================================================
    // 初始化：DOM 就绪后执行所有增强操作
    // ============================================================
    function init() {
      initScrollSpy();
      initSmoothNav();
      enhanceCodeBlocks();
      transformCallouts();
      wrapTables();
      initLightbox();
      onScroll(); // 初始化进度条和 TOC 高亮
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', init);
    } else {
      init();
    }

  })();
  </script>

  <!-- Mermaid: auto-detect and load only when diagrams are present -->
  <script>
  if (document.querySelector('.mermaid')) {
    var s = document.createElement('script');
    s.src = '{{MERMAID_CDN_URL}}';
    s.onload = function() { mermaid.initialize({startOnLoad:true, theme:'default'}); };
    document.body.appendChild(s);
  }
  </script>

  <!-- KaTeX: auto-detect and load only when math formulas are present -->
  <script>
  if (document.querySelector('.math')) {
    var link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = '{{KATEX_CSS_URL}}';
    document.head.appendChild(link);
    var s = document.createElement('script');
    s.src = '{{KATEX_JS_URL}}';
    s.onload = function() {
      var ar = document.createElement('script');
      ar.src = '{{KATEX_AUTO_RENDER_URL}}';
      ar.onload = function() {
        renderMathInElement(document.body, {
          delimiters: [
            {left: '$$', right: '$$', display: true},
            {left: '$',  right: '$',  display: false}
          ],
          throwOnError: false
        });
      };
      document.body.appendChild(ar);
    };
    document.body.appendChild(s);
  }
  </script>
</body>
</html>
`
