// Package toc generates and renders tables of contents.
// It builds a hierarchical structure from headings and renders HTML navigation.
package toc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// HeadingInfo stores heading metadata parsed from Markdown.
type HeadingInfo struct {
	Level int    // Heading level from 1 to 6.
	Text  string // Heading text.
	ID    string // Unique heading identifier, typically used as an anchor.
}

// TOCEntry represents one node in the TOC tree.
type TOCEntry struct {
	Level    int        // Heading level from 1 to 6.
	Title    string     // Heading text.
	ID       string     // Unique identifier used for anchor links.
	PageNum  int        // Optional page number for print-oriented output.
	Children []TOCEntry // Child entries.
}

// Generator builds hierarchical TOC trees from flat heading lists.
type Generator struct {
	// language is a book language code such as "en" or "zh-CN". It only
	// selects the wording of the rendered "Contents" heading.
	language string
}

// NewGenerator creates a new TOC generator that renders an English heading.
func NewGenerator() *Generator {
	return &Generator{}
}

// NewGeneratorForLanguage creates a TOC generator whose rendered heading is
// written in the book's language.
func NewGeneratorForLanguage(language string) *Generator {
	return &Generator{language: language}
}

// contentsHeadings maps a base language code to the wording of the TOC page
// heading. A printed table of contents that carries no heading at all reads as
// a stray list of links, so one is always rendered; unknown languages fall
// back to English, matching the default language in book.yaml.
var contentsHeadings = map[string]string{
	"en": "Contents",
	"zh": "目录",
	"ja": "目次",
	"ko": "목차",
	"fr": "Sommaire",
	"de": "Inhalt",
	"es": "Índice",
	"pt": "Sumário",
	"it": "Indice",
	"ru": "Содержание",
}

// contentsHeading returns the localized "Contents" wording for language.
func contentsHeading(language string) string {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(language)), "-")
	if heading, ok := contentsHeadings[base]; ok {
		return heading
	}
	return contentsHeadings["en"]
}

// Generate builds a hierarchical TOC tree from document-order headings.
func (g *Generator) Generate(headings []HeadingInfo) []TOCEntry {
	if len(headings) == 0 {
		return []TOCEntry{}
	}

	// Track nesting with a pointer stack to avoid losing child references.
	var stack []*TOCEntry
	var root []*TOCEntry

	for _, heading := range headings {
		entry := &TOCEntry{
			Level:    heading.Level,
			Title:    heading.Text,
			ID:       heading.ID,
			Children: []TOCEntry{},
		}

		// Pop until the parent level is lower than the current entry level.
		for len(stack) > 0 && stack[len(stack)-1].Level >= entry.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			root = append(root, entry)
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, *entry)
			// Re-point to the appended child entry stored inside parent.Children.
			entry = &parent.Children[len(parent.Children)-1]
		}

		stack = append(stack, entry)
	}

	// Convert the root pointer slice back into values.
	result := make([]TOCEntry, len(root))
	for i, r := range root {
		result[i] = *r
	}
	return result
}

// RenderHTML renders the TOC tree as nested HTML navigation.
func (g *Generator) RenderHTML(entries []TOCEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(`<nav class="toc">` + "\n")
	fmt.Fprintf(&buf, "  <h1 class=\"toc-title\">%s</h1>\n", utils.EscapeHTML(contentsHeading(g.language)))
	g.renderEntries(&buf, entries, 0)
	buf.WriteString(`</nav>` + "\n")

	return buf.String()
}

// PageSlotAttr marks the element that holds an entry's printed page number.
// The PDF generator prints the document once to learn which page each anchor
// landed on, then fills these slots in and prints again — the slot is left
// empty and reserved on the first pass so that filling it in cannot change the
// layout, and therefore cannot invalidate the very page numbers being written.
const PageSlotAttr = "data-toc-page"

// renderEntries renders TOC entries recursively.
func (g *Generator) renderEntries(buf *strings.Builder, entries []TOCEntry, depth int) {
	if len(entries) == 0 {
		return
	}

	indent := strings.Repeat("  ", depth+1)
	buf.WriteString(indent + `<ul>` + "\n")

	for _, entry := range entries {
		itemIndent := strings.Repeat("  ", depth+2)
		buf.WriteString(itemIndent + `<li>`)
		// The title, the dot leader and the page-number slot are three flex
		// items so the number is flushed right and the leader stretches
		// between them; see the .toc rules in internal/renderer/template.go.
		pageNum := ""
		if entry.PageNum > 0 {
			pageNum = strconv.Itoa(entry.PageNum)
		}
		fmt.Fprintf(buf,
			`<a href="#%s"><span class="toc-entry-title">%s</span><span class="toc-leader"></span><span class="toc-pageno" %s="%s">%s</span></a>`,
			utils.EscapeAttr(entry.ID), utils.EscapeHTML(entry.Title),
			PageSlotAttr, utils.EscapeAttr(entry.ID), pageNum)

		// Render child entries recursively when present.
		if len(entry.Children) > 0 {
			buf.WriteString("\n")
			g.renderEntries(buf, entry.Children, depth+1)
			buf.WriteString(itemIndent)
		}

		buf.WriteString(`</li>` + "\n")
	}

	buf.WriteString(indent + `</ul>` + "\n")
}

// getEntry performs a depth-first lookup by entry ID.
func getEntry(entries []TOCEntry, id string) *TOCEntry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
		if found := getEntry(entries[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}

// flattenToList flattens the TOC tree into a linear list while preserving order.
func flattenToList(entries []TOCEntry) []TOCEntry {
	var result []TOCEntry

	var flatten func([]TOCEntry)
	flatten = func(children []TOCEntry) {
		for _, entry := range children {
			result = append(result, TOCEntry{
				Level:   entry.Level,
				Title:   entry.Title,
				ID:      entry.ID,
				PageNum: entry.PageNum,
			})
			flatten(entry.Children)
		}
	}

	flatten(entries)
	return result
}

// CountEntries returns the total number of TOC entries, including descendants.
func CountEntries(entries []TOCEntry) int {
	count := len(entries)
	for _, entry := range entries {
		count += CountEntries(entry.Children)
	}
	return count
}
