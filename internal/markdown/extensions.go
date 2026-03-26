package markdown

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// headingIDTransformer 为标题自动生成唯一 ID 属性.
// It is safe for concurrent use: each Transform call uses a local map so
// multiple documents can be parsed in parallel without interference.
type headingIDTransformer struct{}

func newHeadingIDTransformer() parser.ASTTransformer {
	return &headingIDTransformer{}
}

// Transform 遍历 AST，为所有标题节点生成 ID.
// A fresh usedIDs map is created per call so heading IDs are scoped to a
// single document and concurrent Transform calls don't interfere.
func (t *headingIDTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	usedIDs := make(map[string]int)
	source := reader.Source()
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := n.(*ast.Heading); ok {
			processHeading(heading, source, usedIDs)
		}
		return ast.WalkContinue, nil
	})
}

// processHeading 为单个标题节点设置 ID 属性
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

// generateUniqueID 生成唯一的标题 ID，遇到重复自动添加后缀
func generateUniqueID(text string, usedIDs map[string]int) string {
	baseID := generateHeadingID(text)
	if baseID == "" {
		baseID = "heading"
	}

	count, exists := usedIDs[baseID]
	if !exists {
		usedIDs[baseID] = 1
		return baseID
	}

	usedIDs[baseID] = count + 1
	return fmt.Sprintf("%s-%d", baseID, count+1)
}

// extractNodeText 从 AST 节点中递归提取纯文本
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
