// Package glossary parses glossary definitions and highlights terms in HTML.
//
// GLOSSARY.md format:
//
//	# Glossary
//
//	## API
//	Application Programming Interface.
//
//	## Markdown
//	A lightweight markup language.
package glossary

import (
	"bufio"
	"cmp"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/yeasy/mdpress/pkg/utils"
)

// maxGlossaryLineSize is the maximum line size (in bytes) for the glossary scanner.
const maxGlossaryLineSize = 1024 * 1024

// Package-level compiled regexps to avoid recompilation in hot paths.
// The skip pattern covers every region a term must not be rewritten in: an
// existing anchor (a term already linked, or a term appearing inside the
// author's own link — nesting <a> there would be invalid HTML) and any tag,
// so terms are never matched inside attribute values.
var (
	slugifyRegexp       = regexp.MustCompile(`[^a-z0-9\-\p{L}]`)
	glossarySkipPattern = regexp.MustCompile(`(?s)<a\b[^>]*>.*?</a>|<[^>]+>`)
)

// Term represents a single glossary entry.
type Term struct {
	Name       string // Term name.
	Definition string // Term definition.
}

// Glossary stores parsed glossary terms.
type Glossary struct {
	Terms []Term

	prepareOnce  sync.Once
	sortedTerms  []Term
	termPatterns map[string]*regexp.Regexp
}

// ParseFile parses a glossary from GLOSSARY.md.
func ParseFile(path string) (*Glossary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GLOSSARY.md: %w", err)
	}
	defer f.Close() //nolint:errcheck

	g := &Glossary{}
	var currentTerm string
	var currentDef strings.Builder

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxGlossaryLineSize)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip the top-level heading.
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			continue
		}

		// A second-level heading starts a new term.
		if term, ok := strings.CutPrefix(trimmed, "## "); ok {
			// Flush the previous term.
			if currentTerm != "" {
				g.Terms = append(g.Terms, Term{
					Name:       currentTerm,
					Definition: strings.TrimSpace(currentDef.String()),
				})
			}
			currentTerm = strings.TrimSpace(term)
			currentDef.Reset()
			continue
		}

		// Accumulate definition text.
		if currentTerm != "" && trimmed != "" {
			if currentDef.Len() > 0 {
				currentDef.WriteString(" ")
			}
			currentDef.WriteString(trimmed)
		}
	}

	// Flush the final term.
	if currentTerm != "" {
		g.Terms = append(g.Terms, Term{
			Name:       currentTerm,
			Definition: strings.TrimSpace(currentDef.String()),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read GLOSSARY.md: %w", err)
	}

	return g, nil
}

// prepare sorts terms and pre-compiles regex patterns once.
func (g *Glossary) prepare() {
	g.prepareOnce.Do(func() {
		g.sortedTerms = make([]Term, len(g.Terms))
		copy(g.sortedTerms, g.Terms)
		slices.SortFunc(g.sortedTerms, func(a, b Term) int {
			return cmp.Compare(len(b.Name), len(a.Name))
		})

		g.termPatterns = make(map[string]*regexp.Regexp, len(g.sortedTerms))
		for _, term := range g.sortedTerms {
			escapedName := regexp.QuoteMeta(term.Name)
			if utils.ContainsCJK(term.Name) {
				g.termPatterns[term.Name] = regexp.MustCompile(`(?i)` + escapedName)
			} else {
				g.termPatterns[term.Name] = regexp.MustCompile(`(?i)\b` + escapedName + `\b`)
			}
		}
	})
}

// ProcessHTML highlights glossary terms in HTML body text.
func (g *Glossary) ProcessHTML(html string) string {
	if len(g.Terms) == 0 {
		return html
	}

	g.prepare()

	for _, term := range g.sortedTerms {
		html = highlightTerm(html, term, g.termPatterns[term.Name])
	}

	return html
}

// RenderHTML renders the glossary as an HTML page.
func (g *Glossary) RenderHTML() string {
	if len(g.Terms) == 0 {
		return ""
	}

	// Sort terms alphabetically.
	sorted := make([]Term, len(g.Terms))
	copy(sorted, g.Terms)
	slices.SortFunc(sorted, func(a, b Term) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	// No <h1> here: every consumer renders the chapter title itself (the
	// PDF/HTML templates from ChapterHTML.Title, ePub and the site by
	// re-adding one when the body has none), so emitting a heading printed
	// "Glossary" twice on the page.
	var b strings.Builder
	b.WriteString("<div class=\"glossary-page\">\n")
	b.WriteString("<dl class=\"glossary-list\">\n")
	for _, term := range sorted {
		fmt.Fprintf(&b, "  <dt id=\"glossary-%s\"><strong>%s</strong></dt>\n",
			slugify(term.Name), utils.EscapeHTML(term.Name))
		fmt.Fprintf(&b, "  <dd>%s</dd>\n", utils.EscapeHTML(term.Definition))
	}
	b.WriteString("</dl>\n")
	b.WriteString("</div>\n")

	return b.String()
}

// linkTerm returns the replacement function for one term. The occurrence
// becomes a link to the glossary entry, not just a title tooltip: a tooltip
// needs a mouse, so on paper, on e-ink readers and on touch screens the
// definition was unreachable even though the docs promised terms were linked.
// The tooltip is kept for pointer users.
func linkTerm(term Term) func(string) string {
	return func(match string) string {
		return fmt.Sprintf(`<a href="#glossary-%s" class="glossary-term" title="%s">%s</a>`,
			slugify(term.Name), utils.EscapeAttr(term.Definition), match)
	}
}

// highlightTerm highlights a single term while avoiding tag replacement.
func highlightTerm(html string, term Term, pattern *regexp.Regexp) string {
	skipPattern := glossarySkipPattern
	skipPositions := skipPattern.FindAllStringIndex(html, -1)

	// Build safe replacement segments outside tags and existing spans.
	var result strings.Builder
	lastEnd := 0

	for _, pos := range skipPositions {
		// Replace terms in text before the skipped region.
		if pos[0] > lastEnd {
			textSegment := html[lastEnd:pos[0]]
			textSegment = pattern.ReplaceAllStringFunc(textSegment, linkTerm(term))
			result.WriteString(textSegment)
		}
		// Preserve the skipped region as-is.
		result.WriteString(html[pos[0]:pos[1]])
		lastEnd = pos[1]
	}

	// Replace terms in the trailing text segment.
	if lastEnd < len(html) {
		textSegment := html[lastEnd:]
		textSegment = pattern.ReplaceAllStringFunc(textSegment, linkTerm(term))
		result.WriteString(textSegment)
	}

	return result.String()
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return slugifyRegexp.ReplaceAllString(s, "")
}
