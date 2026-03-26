// Package renderer assembles individual parts into the final HTML document.
// It uses html/template for safe HTML rendering, combining theme CSS, cover, TOC, and chapter content.
package renderer

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// HTMLRenderer assembles individual parts into the final HTML document.
type HTMLRenderer struct {
	config *config.BookConfig
	theme  *theme.Theme
	tmpl   *template.Template
}

// ChapterHTML represents the HTML content of a single chapter.
type ChapterHTML struct {
	Title    string       // Chapter title
	ID       string       // Chapter unique identifier
	Content  string       // Chapter HTML content
	Depth    int          // Chapter depth in book structure (0-based)
	Headings []NavHeading // Heading tree within the chapter, used for navigation
}

// NavHeading represents a navigation heading tree within a chapter.
type NavHeading struct {
	Title    string
	ID       string
	Children []NavHeading
}

// RenderParts contains the individual parts to be rendered.
type RenderParts struct {
	CoverHTML    string        // Cover page HTML
	TOCHTML      string        // Table of contents HTML
	ChaptersHTML []ChapterHTML // All chapters
	CustomCSS    string        // Custom CSS
}

// Template data structure.
type templateData struct {
	Title            string
	Author           string
	Language         string
	CSS              template.CSS
	CoverHTML        template.HTML
	TOCHTML          template.HTML
	Chapters         []templateChapter
	HeaderText       string
	FooterText       string
	Watermark        string
	WatermarkOpacity float64
}

type templateChapter struct {
	Title   string
	ID      string
	Content template.HTML
}

// NewHTMLRenderer creates a new HTML renderer used for PDF generation.
func NewHTMLRenderer(cfg *config.BookConfig, thm *theme.Theme) (*HTMLRenderer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
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

// Render assembles all parts into a complete HTML document.
func (r *HTMLRenderer) Render(parts *RenderParts) (string, error) {
	if parts == nil {
		return "", fmt.Errorf("render parts cannot be nil")
	}

	// Assemble CSS: theme CSS + custom CSS + print CSS.
	fullCSS := r.buildFullCSS(parts.CustomCSS)

	// Convert chapter data.
	chapters := make([]templateChapter, len(parts.ChaptersHTML))
	for i, ch := range parts.ChaptersHTML {
		chapters[i] = templateChapter{
			Title:   ch.Title,
			ID:      ch.ID,
			Content: template.HTML(ch.Content),
		}
	}

	// Build header and footer text.
	headerText := r.config.Style.Header.Left
	footerText := r.config.Style.Footer.Center

	data := templateData{
		Title:            r.config.Book.Title,
		Author:           r.config.Book.Author,
		Language:         r.config.Book.Language,
		CSS:              template.CSS(fullCSS),
		CoverHTML:        template.HTML(parts.CoverHTML),
		TOCHTML:          template.HTML(parts.TOCHTML),
		Chapters:         chapters,
		HeaderText:       headerText,
		FooterText:       footerText,
		Watermark:        r.config.Output.Watermark,
		WatermarkOpacity: r.config.Output.WatermarkOpacity,
	}

	var result strings.Builder
	if err := r.tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to render HTML template: %w", err)
	}

	return result.String(), nil
}

// buildFullCSS assembles the complete CSS.
func (r *HTMLRenderer) buildFullCSS(customCSS string) string {
	var css strings.Builder

	// Theme CSS.
	if r.theme != nil {
		css.WriteString(r.theme.ToCSS())
		css.WriteString("\n")
	}

	// Custom CSS.
	if customCSS != "" {
		css.WriteString(customCSS)
		css.WriteString("\n")
	}

	// Print CSS.
	css.WriteString(r.buildPrintCSS())

	return css.String()
}

// buildPrintCSS generates print-related CSS rules.
func (r *HTMLRenderer) buildPrintCSS() string {
	var css strings.Builder

	pageSize := r.config.Style.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}
	// Defense-in-depth: validate at point of use, not just config load time.
	validSizes := map[string]bool{"A4": true, "A5": true, "Letter": true, "Legal": true, "B5": true}
	if !validSizes[pageSize] {
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
