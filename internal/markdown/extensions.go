package markdown

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// headingIDTransformer 为标题自动生成唯一 ID 属性
type headingIDTransformer struct {
	usedIDs map[string]int
	mu      sync.Mutex
}

func newHeadingIDTransformer() parser.ASTTransformer {
	return &headingIDTransformer{
		usedIDs: make(map[string]int),
	}
}

// Transform 遍历 AST，为所有标题节点生成 ID
func (t *headingIDTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	if err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := n.(*ast.Heading); ok {
			t.processHeading(heading, source)
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return
	}
}

// processHeading 为单个标题节点设置 ID 属性
func (t *headingIDTransformer) processHeading(heading *ast.Heading, source []byte) {
	if _, ok := heading.AttributeString("id"); ok {
		return
	}

	headingText := extractNodeText(heading, source)
	if headingText == "" {
		return
	}

	id := t.generateUniqueID(headingText)
	heading.SetAttributeString("id", []byte(id))
}

// generateUniqueID 生成唯一的标题 ID，遇到重复自动添加后缀
func (t *headingIDTransformer) generateUniqueID(text string) string {
	baseID := generateHeadingID(text)
	if baseID == "" {
		baseID = "heading"
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	count, exists := t.usedIDs[baseID]
	if !exists {
		t.usedIDs[baseID] = 1
		return baseID
	}

	t.usedIDs[baseID] = count + 1
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
