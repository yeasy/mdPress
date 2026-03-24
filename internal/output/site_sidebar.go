package output

import (
	"fmt"
	"html/template"
	"strings"
)

// buildSidebar renders the sidebar navigation HTML.
func (g *SiteGenerator) buildSidebar(chapters []SiteChapter, activeFile string) string {
	var b strings.Builder
	g.renderSidebarItems(&b, chapters, activeFile)
	return b.String()
}

// renderSidebarItems recursively renders sidebar chapter navigation items as HTML.
// It processes the chapters list and their children, generating nested navigation
// elements with appropriate classes and states.
func (g *SiteGenerator) renderSidebarItems(b *strings.Builder, chapters []SiteChapter, activeFile string) {
	for _, ch := range chapters {
		filename := ch.Filename
		if filename == "" {
			filename = "#"
		}
		groupClass := "nav-group"
		hasChildren := (maxSidebarHeadingDepth > 0 && len(ch.Headings) > 0) ||
			(ch.Depth < maxSidebarChapterDepth && len(ch.Children) > 0)
		if hasChildren {
			if g.isChapterBranchActive(ch, activeFile) {
				groupClass += " expanded"
			} else {
				groupClass += " collapsed"
			}
		}

		fmt.Fprintf(b, `<div class="%s" data-group-file="%s">`, groupClass, template.HTMLEscapeString(filename))
		b.WriteString(`<div class="nav-row">`)
		if hasChildren {
			expanded := "false"
			if g.isChapterBranchActive(ch, activeFile) {
				expanded = "true"
			}
			fmt.Fprintf(b, `<button class="nav-toggle" type="button" aria-label="Toggle section" aria-expanded="%s"></button>`, expanded)
		} else {
			b.WriteString(`<span class="nav-toggle nav-toggle-placeholder"></span>`)
		}
		escapedFilename := template.HTMLEscapeString(filename)
		fmt.Fprintf(b,
			`<a class="nav-item nav-chapter nav-depth-%d" href="%s" data-file="%s" data-group-link="%t">%s</a>`,
			ch.Depth+1,
			escapedFilename,
			escapedFilename,
			hasChildren,
			template.HTMLEscapeString(ch.Title))
		b.WriteString(`</div>`)

		if hasChildren {
			b.WriteString(`<div class="nav-children">`)
			b.WriteString(`<div class="nav-children-inner">`)
			if len(ch.Headings) > 0 {
				g.renderSidebarHeadings(b, ch.Filename, ch.Headings, 0)
			}
			if len(ch.Children) > 0 {
				g.renderSidebarItems(b, ch.Children, activeFile)
			}
			b.WriteString(`</div>`)
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>` + "\n")
	}
}

// isChapterBranchActive recursively checks whether a chapter or any of its
// descendants matches the active file.
func (g *SiteGenerator) isChapterBranchActive(ch SiteChapter, activeFile string) bool {
	if ch.Filename == activeFile {
		return true
	}
	for _, child := range ch.Children {
		if g.isChapterBranchActive(child, activeFile) {
			return true
		}
	}
	return false
}

// buildBreadcrumbs returns the ancestor chain from root to the page identified
// by filename.  For example, for a page nested under "Part 1 > Chapter 2" it
// returns [{Part 1, part1.html}, {Chapter 2, ch2.html}].
func (g *SiteGenerator) buildBreadcrumbs(chapters []SiteChapter, filename string) []breadcrumb {
	const maxDepth = 20
	var walk func([]SiteChapter, []breadcrumb, int) []breadcrumb
	walk = func(chs []SiteChapter, path []breadcrumb, depth int) []breadcrumb {
		if depth > maxDepth {
			return nil
		}
		for _, ch := range chs {
			cur := make([]breadcrumb, len(path)+1)
			copy(cur, path)
			cur[len(path)] = breadcrumb{Title: ch.Title, Filename: ch.Filename}
			if ch.Filename == filename {
				return cur
			}
			if found := walk(ch.Children, cur, depth+1); found != nil {
				return found
			}
		}
		return nil
	}
	return walk(chapters, nil, 0)
}

// maxSidebarChapterDepth limits how deep chapter nesting goes in the sidebar.
// Chapters at this depth or deeper are rendered as flat links without expand
// triangles, and their Children are not shown.  A value of 1 means only
// top-level groups (e.g. "第二章") expand to show their direct children
// (2.1, 2.2, …); those children appear as plain links without further nesting.
const maxSidebarChapterDepth = 1

// maxSidebarHeadingDepth limits how many levels of headings appear in the
// sidebar navigation.  Set to 0 to show only chapter titles — no in-page
// headings (h2, h3, …) in the sidebar.
const maxSidebarHeadingDepth = 0

// renderSidebarHeadings recursively renders heading navigation items for the sidebar.
func (g *SiteGenerator) renderSidebarHeadings(b *strings.Builder, filename string, headings []SiteNavHeading, depth int) {
	if depth >= maxSidebarHeadingDepth {
		return
	}
	for _, heading := range headings {
		fmt.Fprintf(b,
			`<a class="nav-item nav-heading nav-heading-depth-%d" href="%s#%s" data-file="%s" data-target="%s">%s</a>`,
			depth+1,
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(heading.Title))
		if len(heading.Children) > 0 {
			g.renderSidebarHeadings(b, filename, heading.Children, depth+1)
		}
	}
}
