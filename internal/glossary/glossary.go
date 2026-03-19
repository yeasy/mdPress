// Package glossary parses glossary definitions and highlights terms in HTML.
//
// GLOSSARY.md 格式：
//
//	# Glossary
//
//	## API
//	Application Programming Interface，应用程序编程接口。
//
//	## Markdown
//	一种轻量级标记语言。
package glossary

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// Term represents a single glossary entry.
type Term struct {
	Name       string // Term name.
	Definition string // Term definition.
}

// Glossary stores parsed glossary terms.
type Glossary struct {
	Terms []Term
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
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip the top-level heading.
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			continue
		}

		// A second-level heading starts a new term.
		if strings.HasPrefix(trimmed, "## ") {
			// Flush the previous term.
			if currentTerm != "" {
				g.Terms = append(g.Terms, Term{
					Name:       currentTerm,
					Definition: strings.TrimSpace(currentDef.String()),
				})
			}
			currentTerm = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
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

// ProcessHTML highlights glossary terms in HTML body text.
func (g *Glossary) ProcessHTML(html string) string {
	if len(g.Terms) == 0 {
		return html
	}

	// Match longer terms first.
	sorted := make([]Term, len(g.Terms))
	copy(sorted, g.Terms)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Name) > len(sorted[j].Name)
	})

	for _, term := range sorted {
		html = highlightTerm(html, term)
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
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	var b strings.Builder
	b.WriteString("<div class=\"glossary-page\">\n")
	b.WriteString("<h1>Glossary</h1>\n")
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

// highlightTerm highlights a single term while avoiding tag replacement.
func highlightTerm(html string, term Term) string {
	// Use word boundaries to avoid partial matches.
	escapedName := regexp.QuoteMeta(term.Name)
	pattern := regexp.MustCompile(`(?i)\b` + escapedName + `\b`)

	// Split tags from text and only replace in text nodes.
	tagPattern := regexp.MustCompile(`<[^>]+>`)
	tagPositions := tagPattern.FindAllStringIndex(html, -1)

	// Build safe replacement segments outside tags.
	var result strings.Builder
	lastEnd := 0

	for _, pos := range tagPositions {
		// Replace terms in text before the tag.
		if pos[0] > lastEnd {
			textSegment := html[lastEnd:pos[0]]
			textSegment = pattern.ReplaceAllStringFunc(textSegment, func(match string) string {
				tooltip := utils.EscapeAttr(term.Definition)
				return fmt.Sprintf(`<span class="glossary-term" title="%s">%s</span>`, tooltip, match)
			})
			result.WriteString(textSegment)
		}
		// Preserve tags as-is.
		result.WriteString(html[pos[0]:pos[1]])
		lastEnd = pos[1]
	}

	// Replace terms in the trailing text segment.
	if lastEnd < len(html) {
		textSegment := html[lastEnd:]
		textSegment = pattern.ReplaceAllStringFunc(textSegment, func(match string) string {
			tooltip := utils.EscapeAttr(term.Definition)
			return fmt.Sprintf(`<span class="glossary-term" title="%s">%s</span>`, tooltip, match)
		})
		result.WriteString(textSegment)
	}

	return result.String()
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9\-\p{L}]`)
	return reg.ReplaceAllString(s, "")
}
