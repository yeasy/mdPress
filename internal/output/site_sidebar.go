package output

import (
	"fmt"
	"html/template"
	"strings"
)

// buildSidebar renders the sidebar navigation HTML.
func (g *SiteGenerator) buildSidebar(chapters []SiteChapter, activeFile string) string {
	var b strings.Builder
	g.renderSidebarItems(&b, sidebarChapters(chapters, g.Meta.Title), activeFile, 0)
	return b.String()
}

func sidebarChapters(chapters []SiteChapter, siteTitle string) []SiteChapter {
	if len(chapters) == 0 {
		return chapters
	}
	first := chapters[0]
	lowerFile := strings.ToLower(first.Filename)
	if strings.Contains(lowerFile, "readme/") || strings.Contains(lowerFile, "readme.") || strings.EqualFold(first.Title, siteTitle) {
		return chapters[1:]
	}
	return chapters
}

// renderSidebarItems recursively renders sidebar chapter navigation items as HTML.
// It processes the chapters list and their children, generating nested navigation
// elements with appropriate classes and states.
// The depth argument counts nesting levels actually rendered so far; it only
// guards against a malformed chapter tree recursing forever, it is not a
// display limit.
func (g *SiteGenerator) renderSidebarItems(b *strings.Builder, chapters []SiteChapter, activeFile string, depth int) {
	if depth >= maxSidebarChapterDepth {
		return
	}
	for _, ch := range chapters {
		// A group label starts a new run of chapters. Without this a long book
		// renders as one flat, unscannable list.
		if ch.Section != "" {
			fmt.Fprintf(b, `<div class="nav-section">%s</div>`, template.HTMLEscapeString(ch.Section))
		}
		filename := ch.Filename
		if filename == "" {
			filename = "#"
		}
		href := filename
		if href != "#" {
			href = relativeSiteHref(activeFile, filename)
		}
		groupClass := "nav-group"
		hasChildren := (maxSidebarHeadingDepth > 0 && len(ch.Headings) > 0) ||
			(depth+1 < maxSidebarChapterDepth && len(ch.Children) > 0)
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
		}
		escapedFilename := template.HTMLEscapeString(filename)
		escapedHref := template.HTMLEscapeString(href)
		fmt.Fprintf(b,
			`<a class="nav-item nav-chapter nav-depth-%d" href="%s" data-file="%s" data-group-link="%t">%s</a>`,
			ch.Depth+1,
			escapedHref,
			escapedFilename,
			hasChildren,
			template.HTMLEscapeString(ch.Title))
		b.WriteString(`</div>`)

		if hasChildren {
			b.WriteString(`<div class="nav-children">`)
			b.WriteString(`<div class="nav-children-inner">`)
			if len(ch.Headings) > 0 {
				g.renderSidebarHeadings(b, activeFile, ch.Filename, ch.Headings, 0)
			}
			if len(ch.Children) > 0 {
				g.renderSidebarItems(b, ch.Children, activeFile, depth+1)
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

// maxSidebarChapterDepth is a runaway-recursion guard, not a display budget.
// It used to be 1, which meant a book with the ordinary three-level structure
// (Part > Chapter > Section) generated the section pages, listed them in
// sitemap.xml and indexed them for search, but never linked them from the
// sidebar on any page — the reader could only reach them by walking prev/next
// or guessing the URL, and the build said nothing.  Real books nest three or
// four levels; the limit only exists so a malformed chapter tree cannot make
// the renderer recurse forever.  It matches the breadcrumb walker's cap.
const maxSidebarChapterDepth = 20

// maxSidebarHeadingDepth limits how many levels of headings appear in the
// sidebar navigation.  Set to 0 to show only chapter titles — no in-page
// headings (h2, h3, …) in the sidebar.
const maxSidebarHeadingDepth = 0

// renderSidebarHeadings recursively renders heading navigation items for the sidebar.
func (g *SiteGenerator) renderSidebarHeadings(b *strings.Builder, activeFile string, filename string, headings []SiteNavHeading, depth int) {
	if depth >= maxSidebarHeadingDepth {
		return
	}
	for _, heading := range headings {
		href := relativeSiteHref(activeFile, filename)
		fmt.Fprintf(b,
			`<a class="nav-item nav-heading nav-heading-depth-%d" href="%s#%s" data-file="%s" data-target="%s">%s</a>`,
			depth+1,
			template.HTMLEscapeString(href),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(heading.Title))
		if len(heading.Children) > 0 {
			g.renderSidebarHeadings(b, activeFile, filename, heading.Children, depth+1)
		}
	}
}
