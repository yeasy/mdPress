// html_standalone.go renders a self-contained single-page HTML document.
// The output embeds CSS and JavaScript and implements a GitBook-style three-column
// layout: left sidebar (global TOC), center content area, right in-page TOC.
//
// Additional features: dark/light/system theme toggle, code copy button with
// language label, callout boxes, full-text search (⌘K), prev/next navigation,
// image lightbox, Mermaid diagrams, and KaTeX math formulas.
package renderer

import (
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// StandaloneHTMLRenderer renders a self-contained single-page HTML document.
type StandaloneHTMLRenderer struct {
	config *config.BookConfig
	theme  *theme.Theme
	tmpl   *template.Template
}

// standaloneData is the template data model.
type standaloneData struct {
	Title       string
	Author      string
	Language    string
	CSS         template.CSS
	Chapters    []standaloneChapter
	SidebarHTML template.HTML
}

// standaloneChapter stores chapter rendering data (with prev/next navigation).
type standaloneChapter struct {
	Title     string
	ID        string
	Content   template.HTML
	PrevTitle string // Previous chapter title (empty if none)
	PrevID    string // Previous chapter ID
	NextTitle string // Next chapter title (empty if none)
	NextID    string // Next chapter ID
}

// standaloneSidebarChapter is a sidebar tree node.
type standaloneSidebarChapter struct {
	ChapterHTML
	Children []standaloneSidebarChapter
}

// NewStandaloneHTMLRenderer creates a single-page HTML renderer.
func NewStandaloneHTMLRenderer(cfg *config.BookConfig, thm *theme.Theme) (*StandaloneHTMLRenderer, error) {
	// Compose the complete template from separate parts
	standaloneHTMLTemplate := standaloneHTMLHead + standaloneCSS + standaloneHTMLMiddle + standaloneJS + standaloneHTMLTail

	// Substitute CDN URL placeholders before parsing the template so that the
	// template engine never needs to evaluate them as Go template expressions.
	resolved := utils.ResolveCDNPlaceholders(standaloneHTMLTemplate)

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

// Render renders the complete single-page HTML document.
func (r *StandaloneHTMLRenderer) Render(parts *RenderParts) (string, error) {
	if parts == nil {
		return "", errors.New("render parts cannot be nil")
	}

	// Assemble CSS bundle (theme CSS + custom CSS).
	var cssBuilder strings.Builder
	if r.theme != nil {
		cssBuilder.WriteString(r.theme.ToCSS())
		cssBuilder.WriteString("\n")
	}
	if parts.CustomCSS != "" {
		cssBuilder.WriteString(utils.SanitizeCSS(parts.CustomCSS))
		cssBuilder.WriteString("\n")
	}

	// Convert chapter data and pre-compute prev/next navigation info.
	chapters := make([]standaloneChapter, 0, len(parts.ChaptersHTML))
	for i, ch := range parts.ChaptersHTML {
		chID := ch.ID
		if chID == "" {
			chID = fmt.Sprintf("chapter-%d", i+1)
		}

		// Compute prev/next chapter info.
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
		Language:    r.config.Book.Language,
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

// buildSidebar generates the left-side global TOC sidebar HTML.
func (r *StandaloneHTMLRenderer) buildSidebar(chapters []ChapterHTML) string {
	var b strings.Builder
	for _, ch := range buildStandaloneSidebarTree(chapters) {
		r.renderSidebarChapter(&b, ch)
	}
	return b.String()
}

// renderSidebarChapter recursively renders a sidebar chapter entry.
// Note: toc-group, data-group-id, data-group-link attributes are preserved for test compatibility.
func (r *StandaloneHTMLRenderer) renderSidebarChapter(b *strings.Builder, ch standaloneSidebarChapter) {
	hasChildren := len(ch.Headings) > 0 || len(ch.Children) > 0

	// Keep toc-group class (test compat) and add new toc-item class.
	groupClass := "toc-group toc-item"
	if hasChildren {
		groupClass += " has-children"
	}

	fmt.Fprintf(b, `<div class="%s" data-group-id="%s">`,
		groupClass, template.HTMLEscapeString(ch.ID))
	b.WriteString(`<div class="toc-row">`)

	if hasChildren {
		b.WriteString(`<button class="toc-toggle" type="button" aria-label="Expand/Collapse" aria-expanded="false"></button>`)
	} else {
		b.WriteString(`<span class="toc-spacer"></span>`)
	}

	// Preserve data-group-link="true" (test compat).
	fmt.Fprintf(b,
		`<a href="#%s" class="toc-link toc-link-chapter toc-depth-%d" data-target="%s" data-group-link="true">%s</a>`,
		template.HTMLEscapeString(ch.ID),
		ch.Depth+1,
		template.HTMLEscapeString(ch.ID),
		template.HTMLEscapeString(ch.Title))
	b.WriteString(`</div>`)

	if hasChildren {
		// Collapsed by default (hidden attribute).
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

// renderSidebarHeadings recursively renders heading entries in the sidebar.
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

// buildStandaloneSidebarTree converts a flat chapter list into a tree structure.
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
