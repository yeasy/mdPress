package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// headingIDTransformer auto-generates unique ID attributes for headings.
// It is safe for concurrent use: each Transform call uses a local map so
// multiple documents can be parsed in parallel without interference.
type headingIDTransformer struct{}

func newHeadingIDTransformer() parser.ASTTransformer {
	return &headingIDTransformer{}
}

// Transform walks the AST and generates IDs for all heading nodes.
// A fresh usedIDs map is created per call so heading IDs are scoped to a
// single document and concurrent Transform calls don't interfere.
func (t *headingIDTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	usedIDs := make(map[string]int)
	source := reader.Source()

	var headings []*ast.Heading
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := n.(*ast.Heading); ok {
			headings = append(headings, heading)
		}
		return ast.WalkContinue, nil
	})

	// Ids an author wrote as "## Heading {#custom-id}" are claimed before any
	// slug is derived: the author wrote {#intro} precisely so that links to
	// #intro land on that heading, so an unrelated heading that happens to
	// slugify to "intro" is the one that must yield.
	for _, heading := range headings {
		claimCustomHeadingID(heading, usedIDs)
	}
	for _, heading := range headings {
		processHeading(heading, source, usedIDs)
	}
}

// claimCustomHeadingID reserves the custom id of a heading written as
// "## Heading {#custom-id}", keeping the author's spelling verbatim — the whole
// point of a custom id is that links to it stay valid. Two headings carrying the
// same custom id still get the "-N" treatment: duplicate ids are invalid HTML
// and the browser silently jumps to whichever comes first.
func claimCustomHeadingID(heading *ast.Heading, usedIDs map[string]int) {
	attr, ok := heading.AttributeString("id")
	if !ok {
		return
	}
	idBytes, ok := attr.([]byte)
	if !ok || len(idBytes) == 0 {
		return
	}
	heading.SetAttributeString("id", []byte(claimUniqueID(string(idBytes), usedIDs)))
}

// processHeading sets the ID attribute for a single heading node.
func processHeading(heading *ast.Heading, source []byte, usedIDs map[string]int) {
	if _, ok := heading.AttributeString("id"); ok {
		return
	}

	headingText := extractNodeText(heading, source)
	if headingText == "" {
		return
	}

	id := generateUniqueID(headingText, usedIDs)
	heading.SetAttributeString("id", []byte(id))
}

// generateUniqueID generates a unique heading ID, auto-appending a suffix on duplicates.
func generateUniqueID(text string, usedIDs map[string]int) string {
	baseID := generateHeadingID(text)
	if baseID == "" {
		baseID = "heading"
	}
	return claimUniqueID(baseID, usedIDs)
}

// claimUniqueID reserves baseID in usedIDs, returning it unchanged the first
// time and "baseID-N" afterwards. The suffix keeps climbing until it lands on a
// name nobody holds: a custom {#intro-2} can occupy the very name the counter
// would otherwise hand to the second "Intro" heading.
func claimUniqueID(baseID string, usedIDs map[string]int) string {
	count, exists := usedIDs[baseID]
	if !exists {
		usedIDs[baseID] = 1
		return baseID
	}

	for {
		count++
		candidate := fmt.Sprintf("%s-%d", baseID, count)
		if _, taken := usedIDs[candidate]; !taken {
			usedIDs[baseID] = count
			usedIDs[candidate] = 1
			return candidate
		}
	}
}

// extractNodeText recursively extracts plain text from an AST node.
func extractNodeText(node ast.Node, source []byte) string {
	var result strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			result.Write(textNode.Segment.Value(source))
		} else if codeSpan, ok := child.(*ast.CodeSpan); ok {
			for c := codeSpan.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					result.Write(t.Segment.Value(source))
				}
			}
		} else if child.HasChildren() {
			result.WriteString(extractNodeText(child, source))
		}
	}
	return result.String()
}

var _ parser.ASTTransformer = (*headingIDTransformer)(nil)
