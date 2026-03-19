// Package renderer 负责将各个部分组装成最终的 HTML 文档。
// 使用 html/template 进行安全的 HTML 渲染，支持主题 CSS、封面、目录和章节内容的组合。
package renderer

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// HTMLRenderer 用于将各个部分组装成最终的 HTML 文档
type HTMLRenderer struct {
	config *config.BookConfig
	theme  *theme.Theme
	tmpl   *template.Template
}

// ChapterHTML 表示单个章节的 HTML 内容
type ChapterHTML struct {
	Title    string       // 章节标题
	ID       string       // 章节唯一标识符
	Content  string       // 章节 HTML 内容
	Depth    int          // 章节在书籍结构中的层级（从 0 开始）
	Headings []NavHeading // 章节内标题树，用于导航
}

// NavHeading 表示章节内的导航标题树
type NavHeading struct {
	Title    string
	ID       string
	Children []NavHeading
}

// RenderParts 包含需要渲染的各个部分
type RenderParts struct {
	CoverHTML    string        // 封面 HTML
	TOCHTML      string        // 目录 HTML
	ChaptersHTML []ChapterHTML // 所有章节
	CustomCSS    string        // 自定义 CSS
}

// 模板数据结构
type templateData struct {
	Title      string
	Author     string
	Language   string
	CSS        template.CSS
	CoverHTML  template.HTML
	TOCHTML    template.HTML
	Chapters   []templateChapter
	HeaderText string
	FooterText string
}

type templateChapter struct {
	Title   string
	ID      string
	Content template.HTML
}

// NewHTMLRenderer creates a new HTML renderer used for PDF generation.
func NewHTMLRenderer(cfg *config.BookConfig, thm *theme.Theme) (*HTMLRenderer, error) {
	// Substitute CDN URL placeholders so the template does not need to import
	// the utils package at template execution time.
	resolvedTemplate := strings.ReplaceAll(htmlTemplate, "{{MERMAID_CDN_URL}}", utils.MermaidCDNURL)
	resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{{KATEX_CSS_URL}}", utils.KaTeXCSSURL)
	resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{{KATEX_JS_URL}}", utils.KaTeXJSURL)
	resolvedTemplate = strings.ReplaceAll(resolvedTemplate, "{{KATEX_AUTO_RENDER_URL}}", utils.KaTeXAutoRenderURL)
	tmpl, err := template.New("book").Parse(resolvedTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return &HTMLRenderer{
		config: cfg,
		theme:  thm,
		tmpl:   tmpl,
	}, nil
}

// Render 将各个部分组装成完整的 HTML 文档
func (r *HTMLRenderer) Render(parts *RenderParts) (string, error) {
	if parts == nil {
		return "", fmt.Errorf("渲染部件不能为空")
	}

	// 组装 CSS：主题 CSS + 自定义 CSS + 打印 CSS
	fullCSS := r.buildFullCSS(parts.CustomCSS)

	// 转换章节数据
	chapters := make([]templateChapter, len(parts.ChaptersHTML))
	for i, ch := range parts.ChaptersHTML {
		chapters[i] = templateChapter{
			Title:   ch.Title,
			ID:      ch.ID,
			Content: template.HTML(ch.Content),
		}
	}

	// 构建页眉页脚文本
	headerText := r.config.Style.Header.Left
	footerText := r.config.Style.Footer.Center

	data := templateData{
		Title:      r.config.Book.Title,
		Author:     r.config.Book.Author,
		Language:   r.config.Book.Language,
		CSS:        template.CSS(fullCSS),
		CoverHTML:  template.HTML(parts.CoverHTML),
		TOCHTML:    template.HTML(parts.TOCHTML),
		Chapters:   chapters,
		HeaderText: headerText,
		FooterText: footerText,
	}

	var result strings.Builder
	if err := r.tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("渲染 HTML 模板失败: %w", err)
	}

	return result.String(), nil
}

// buildFullCSS 组装完整的 CSS
func (r *HTMLRenderer) buildFullCSS(customCSS string) string {
	var css strings.Builder

	// 主题 CSS
	if r.theme != nil {
		css.WriteString(r.theme.ToCSS())
		css.WriteString("\n")
	}

	// 自定义 CSS
	if customCSS != "" {
		css.WriteString(customCSS)
		css.WriteString("\n")
	}

	// 打印 CSS
	css.WriteString(r.buildPrintCSS())

	return css.String()
}

// buildPrintCSS 生成打印相关的 CSS 规则
func (r *HTMLRenderer) buildPrintCSS() string {
	var css strings.Builder

	pageSize := r.config.Style.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}

	fmt.Fprintf(&css, `
@page {
  size: %s;
  margin: %.0fmm %.0fmm %.0fmm %.0fmm;
}

@page :first {
  margin: 0;
}

.chapter {
  page-break-before: always;
}

.cover-page {
  page-break-after: always;
}

.toc-page {
  page-break-after: always;
}
`, pageSize,
		r.config.Style.Margin.Top,
		r.config.Style.Margin.Right,
		r.config.Style.Margin.Bottom,
		r.config.Style.Margin.Left)

	return css.String()
}
