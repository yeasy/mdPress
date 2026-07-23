// Package crossref provides cross-reference and auto-numbering functionality.
// It supports numbering and referencing figures, tables, and sections, replacing placeholders in HTML with actual numbers.
package crossref

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/yeasy/mdpress/internal/i18n"
	"github.com/yeasy/mdpress/pkg/utils"
)

// Package-level compiled regexps to avoid recompilation per call.
var (
	refPlaceholderRegexp = regexp.MustCompile(`\{\{ref:([a-zA-Z0-9_\-]+)\}\}`)
	figureCaptionRegexp  = regexp.MustCompile(`(?s)<figure\s+id="([^"]+)"([^>]*)>(.*?)</figure>`)
	tableCaptionRegexp   = regexp.MustCompile(`(?s)<table\s+id="([^"]+)"([^>]*)>(.*?)</table>`)
	figcaptionRegexp     = regexp.MustCompile(`(?s)<figcaption[^>]*>(.*?)</figcaption>`)
	captionRegexp        = regexp.MustCompile(`(?s)<caption[^>]*>(.*?)</caption>`)
	imgAltRegexp         = regexp.MustCompile(`(?s)<img[^>]*\salt="([^"]*)"`)
)

// referenceType defines reference type constants.
type referenceType string

const (
	typeFigure  referenceType = "figure"  // Figure
	typeTable   referenceType = "table"   // Table
	typeSection referenceType = "section" // Section
)

// Reference represents a tracked reference object.
type Reference struct {
	Type      referenceType // Reference type (figure, table, or section)
	ID        string        // Unique identifier
	Number    int           // Auto-assigned number
	Title     string        // Title or description
	Level     int           // Heading level for sections; 0 for other types
	NumberStr string        // Hierarchical number string, e.g. "1.2.3" (sections only)
}

// Resolver manages all cross-references and auto-numbering.
type Resolver struct {
	mu            sync.RWMutex          // Mutex for concurrent access
	figures       map[string]*Reference // Figure ID to reference mapping
	tables        map[string]*Reference // Table ID to reference mapping
	sections      map[string]*Reference // Section ID to reference mapping
	figCount      int                   // Figure counter
	tabCount      int                   // Table counter
	sectionCounts map[int]int           // Section counters by heading level
	labels        i18n.CrossRefLabels   // Localized "Figure"/"Table"/"Section" wording
}

// NewResolver creates a new cross-reference resolver instance labeling
// references in English, the default book language.
func NewResolver() *Resolver {
	return NewResolverForLanguage("")
}

// NewResolverForLanguage creates a resolver whose labels and captions follow
// the book's language code (book.language), e.g. "Figure 1" or "图1".
func NewResolverForLanguage(lang string) *Resolver {
	return &Resolver{
		figures:       make(map[string]*Reference),
		tables:        make(map[string]*Reference),
		sections:      make(map[string]*Reference),
		sectionCounts: make(map[int]int),
		labels:        i18n.CrossRefLabelsFor(lang),
	}
}

// RegisterFigure registers a figure and returns its auto-assigned number.
// The id parameter is the figure's unique identifier (typically used for HTML anchors).
// The title parameter is the figure's caption or description.
func (r *Resolver) RegisterFigure(id, title string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered.
	if ref, exists := r.figures[id]; exists {
		return ref.Number
	}

	r.figCount++
	ref := &Reference{
		Type:   typeFigure,
		ID:     id,
		Number: r.figCount,
		Title:  title,
		Level:  0,
	}

	r.figures[id] = ref
	return r.figCount
}

// RegisterTable registers a table and returns its auto-assigned number.
// The id parameter is the table's unique identifier.
// The title parameter is the table's caption or description.
func (r *Resolver) RegisterTable(id, title string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered.
	if ref, exists := r.tables[id]; exists {
		return ref.Number
	}

	r.tabCount++
	ref := &Reference{
		Type:   typeTable,
		ID:     id,
		Number: r.tabCount,
		Title:  title,
		Level:  0,
	}

	r.tables[id] = ref
	return r.tabCount
}

// RegisterSection registers a section.
// The id parameter is the section's unique identifier.
// The title parameter is the section heading text.
// The level parameter is the heading level (1-6), used for hierarchical numbering.
//
// Hierarchical numbering example:
// Level 1: 1. 2. 3. ...
// Level 2: 1.1. 1.2. 2.1. ...
// Level 3: 1.1.1. 1.1.2. ...
func (r *Resolver) RegisterSection(id, title string, level int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already registered.
	if _, exists := r.sections[id]; exists {
		return
	}

	// Reset counters for deeper levels.
	for lv := level + 1; lv <= 6; lv++ {
		delete(r.sectionCounts, lv)
	}

	// Increment the counter for the current level.
	r.sectionCounts[level]++

	// Build hierarchical number.
	var numbers []string
	for lv := 1; lv <= level; lv++ {
		if count, ok := r.sectionCounts[lv]; ok {
			numbers = append(numbers, strconv.Itoa(count))
		} else {
			numbers = append(numbers, "0")
		}
	}

	// Generate number string (e.g. "1.2.3") for display.
	numberStr := strings.Join(numbers, ".")

	ref := &Reference{
		Type:      typeSection,
		ID:        id,
		Number:    r.sectionCounts[level],
		Title:     title,
		Level:     level,
		NumberStr: numberStr,
	}

	r.sections[id] = ref
}

// RegisterFromHTML registers every identified <figure> and <table> found in
// the given HTML, in document order, so that {{ref:...}} placeholders and
// auto-captions work without the author numbering anything by hand. Callers
// run it over the whole book before resolving any placeholder.
//
// Titles are taken from an existing <figcaption>/<caption>, falling back to
// the image's alt text, so a generated caption repeats what the author wrote
// instead of being blank.
func (r *Resolver) RegisterFromHTML(htmlContent string) {
	for _, m := range figureCaptionRegexp.FindAllStringSubmatch(htmlContent, -1) {
		r.RegisterFigure(m[1], figureTitle(m[3]))
	}
	for _, m := range tableCaptionRegexp.FindAllStringSubmatch(htmlContent, -1) {
		r.RegisterTable(m[1], innerText(captionRegexp, m[3]))
	}
}

// figureTitle derives a figure's title from its own caption, or from the alt
// text of the image it wraps.
func figureTitle(content string) string {
	if title := innerText(figcaptionRegexp, content); title != "" {
		return title
	}
	if m := imgAltRegexp.FindStringSubmatch(content); m != nil {
		return strings.TrimSpace(html.UnescapeString(m[1]))
	}
	return ""
}

// innerText returns the plain text of the first match of pattern in content.
func innerText(pattern *regexp.Regexp, content string) string {
	m := pattern.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(utils.StripHTMLTags(m[1])))
}

// Resolve looks up reference information by ID.
// Returns the found Reference pointer, or an error if not found.
func (r *Resolver) Resolve(id string) (*Reference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Look up by priority: figure > table > section.
	if ref, ok := r.figures[id]; ok {
		return ref, nil
	}
	if ref, ok := r.tables[id]; ok {
		return ref, nil
	}
	if ref, ok := r.sections[id]; ok {
		return ref, nil
	}

	return nil, fmt.Errorf("reference not found: %s", id)
}

// ProcessHTML replaces {{ref:id}} placeholders with links to the referenced
// figure, table or section, labeled in the book's language ("Figure 1",
// "图1", "§1.2.3"). Placeholders whose ID is unknown are left as-is; use
// ProcessHTMLWithUnresolved to learn about them.
//
// Example:
// Input: "As shown in {{ref:fig_demo}}, ..."
// Output: "As shown in <a href=\"#fig_demo\" class=\"ref-figure\">Figure 1</a>, ..."
func (r *Resolver) ProcessHTML(html string) string {
	out, _ := r.ProcessHTMLWithUnresolved(html)
	return out
}

// ProcessHTMLWithUnresolved is ProcessHTML plus the IDs it could not resolve,
// in order of first appearance. A placeholder that stays in the text is
// printed verbatim to the reader, so callers report these rather than
// shipping "{{ref:fig_x}}" in the book.
func (r *Resolver) ProcessHTMLWithUnresolved(html string) (string, []string) {
	var unresolved []string
	seen := make(map[string]bool)

	out := refPlaceholderRegexp.ReplaceAllStringFunc(html, func(match string) string {
		// Extract the ID.
		parts := refPlaceholderRegexp.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		id := parts[1]
		ref, err := r.Resolve(id)
		if err != nil {
			if !seen[id] {
				seen[id] = true
				unresolved = append(unresolved, id)
			}
			// If reference not found, return the original placeholder.
			return match
		}

		// Generate reference text based on type.
		switch ref.Type {
		case typeFigure:
			return fmt.Sprintf(`<a href="#%s" class="ref-figure">%s</a>`, utils.EscapeAttr(ref.ID), utils.EscapeHTML(r.labels.FigureLabel(ref.Number)))
		case typeTable:
			return fmt.Sprintf(`<a href="#%s" class="ref-table">%s</a>`, utils.EscapeAttr(ref.ID), utils.EscapeHTML(r.labels.TableLabel(ref.Number)))
		case typeSection:
			label := r.labels.SectionLabel(ref.Number)
			if ref.NumberStr != "" {
				label = "§" + ref.NumberStr
			}
			return fmt.Sprintf(`<a href="#%s" class="ref-section">%s</a>`, utils.EscapeAttr(ref.ID), utils.EscapeHTML(label))
		default:
			return match
		}
	})

	return out, unresolved
}

// AddCaptions adds numbered captions to figures and tables.
// Processes HTML like <figure id="fig_1"><img ...></figure>
// and adds <figcaption>图1: Title</figcaption>.
//
// Example:
// Input: <figure id="fig_demo"><img src="demo.png"></figure>
// Output: <figure id="fig_demo"><img src="demo.png"><figcaption>图1: Demo</figcaption></figure>
func (r *Resolver) AddCaptions(html string) string {
	// Copy both maps in a single atomic snapshot
	r.mu.RLock()
	figuresCopy := make(map[string]*Reference, len(r.figures))
	for id, ref := range r.figures {
		figuresCopy[id] = ref
	}
	tablesCopy := make(map[string]*Reference, len(r.tables))
	for id, ref := range r.tables {
		tablesCopy[id] = ref
	}
	r.mu.RUnlock()

	// Add captions to figures.
	html = r.addFigureCaptions(html, figuresCopy)

	// Add captions to tables.
	html = r.addTableCaptions(html, tablesCopy)

	return html
}

// addFigureCaptions adds captions to figure elements.
func (r *Resolver) addFigureCaptions(html string, figures map[string]*Reference) string {
	return figureCaptionRegexp.ReplaceAllStringFunc(html, func(match string) string {
		parts := figureCaptionRegexp.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		id := parts[1]
		attrs := parts[2]
		content := parts[3]

		// Look up the reference for this ID.
		ref, ok := figures[id]
		if !ok {
			return match
		}

		// Skip if a figcaption already exists (avoid duplicates).
		if strings.Contains(content, "<figcaption") {
			return match
		}

		// Build the new figure element with caption.
		caption := fmt.Sprintf(`<figcaption>%s%s</figcaption>`,
			utils.EscapeHTML(r.labels.FigureLabel(ref.Number)), captionSuffix(ref.Title))

		return fmt.Sprintf(`<figure id="%s"%s>%s%s</figure>`,
			utils.EscapeAttr(id), attrs, content, caption)
	})
}

// addTableCaptions adds captions to table elements.
func (r *Resolver) addTableCaptions(html string, tables map[string]*Reference) string {
	return tableCaptionRegexp.ReplaceAllStringFunc(html, func(match string) string {
		parts := tableCaptionRegexp.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		id := parts[1]
		attrs := parts[2]
		content := parts[3]

		// Look up the reference for this ID.
		ref, ok := tables[id]
		if !ok {
			return match
		}

		// Skip if a caption already exists (avoid duplicates).
		if strings.Contains(content, "<caption") {
			return match
		}

		// Build the new table element with caption prepended.
		caption := fmt.Sprintf(`<caption>%s%s</caption>`,
			utils.EscapeHTML(r.labels.TableLabel(ref.Number)), captionSuffix(ref.Title))

		return fmt.Sprintf(`<table id="%s"%s>%s%s</table>`,
			utils.EscapeAttr(id), attrs, caption, content)
	})
}

// captionSuffix renders the ": Title" part of a caption, omitting it when the
// element has no title so the caption does not end in a dangling colon.
func captionSuffix(title string) string {
	if strings.TrimSpace(title) == "" {
		return ""
	}
	return ": " + utils.EscapeHTML(title)
}

// GetAllReferences returns all registered references (for debugging or building reference lists).
// Priority matches Resolve: figures > tables > sections. If the same ID exists in
// multiple categories, only the highest-priority entry is returned.
func (r *Resolver) GetAllReferences() map[string]*Reference {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Reference)

	// Insert in reverse priority order so higher-priority entries overwrite lower ones.
	for id, ref := range r.sections {
		result[id] = ref
	}
	for id, ref := range r.tables {
		result[id] = ref
	}
	for id, ref := range r.figures {
		result[id] = ref
	}

	return result
}

// Reset clears all references and reinitializes the resolver.
// Used when processing multiple independent documents.
func (r *Resolver) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.figures = make(map[string]*Reference)
	r.tables = make(map[string]*Reference)
	r.sections = make(map[string]*Reference)
	r.figCount = 0
	r.tabCount = 0
	r.sectionCounts = make(map[int]int)
}
