// Package markdown 提供 Markdown 解析和 HTML 转换功能。
// 基于 goldmark 库，支持 GFM 扩展、代码高亮、脚注等特性。
package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"sync"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Package-level compiled regexps for generateHeadingID (called per heading).
var (
	headingIDStripRegexp = regexp.MustCompile(`[^\p{L}\p{N}\s\-]`)
	headingIDSpaceRegexp = regexp.MustCompile(`\s+`)
)

// HeadingInfo 标题信息结构体，用于目录生成
type HeadingInfo struct {
	Level  int    // 标题等级 (1-6)
	Text   string // 标题文本内容
	ID     string // 标题 ID，用于交叉引用
	Line   int    // 标题所在行
	Column int    // 标题所在列
}

// ParserOption 函数式选项类型
type ParserOption func(*Parser)

// Parser Markdown 解析器
type Parser struct {
	md         goldmark.Markdown
	headings   []HeadingInfo
	headingsMu sync.RWMutex
	codeTheme  string
}

// NewParser 创建并返回一个新的 Markdown 解析器实例
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		headings:  make([]HeadingInfo, 0),
		codeTheme: "github",
	}

	for _, opt := range opts {
		opt(p)
	}

	p.initGoldmark()
	return p
}

// initGoldmark 初始化 goldmark 解析器和所有扩展
func (p *Parser) initGoldmark() {
	exts := []goldmark.Extender{
		// GFM 扩展
		extension.NewTable(),
		extension.Strikethrough,
		extension.TaskList,
		extension.Linkify,
		// 脚注
		extension.Footnote,
		// 代码高亮
		highlighting.NewHighlighting(
			highlighting.WithStyle(p.codeTheme),
		),
	}

	p.md = goldmark.New(
		goldmark.WithExtensions(exts...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(newHeadingIDTransformer(), 100),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // 允许原始 HTML
		),
	)
}

// Parse 解析 Markdown 源代码，返回 HTML 和标题信息
func (p *Parser) Parse(source []byte) (string, []HeadingInfo, error) {
	html, headings, _, err := p.ParseWithDiagnostics(source)
	return html, headings, err
}

// ParseWithDiagnostics 解析 Markdown，并返回构建期 warning。
func (p *Parser) ParseWithDiagnostics(source []byte) (string, []HeadingInfo, []Diagnostic, error) {
	if len(source) == 0 {
		return "", []HeadingInfo{}, nil, nil
	}

	// Pre-process math formulas: replace $$...$$ and $...$ with safe placeholder
	// tokens to prevent goldmark from treating _ inside formulas as emphasis.
	mathProc := newMathPreprocessor()
	processedSource := []byte(mathProc.preprocess(string(source)))

	// Reset heading collection for this parse run.
	p.headingsMu.Lock()
	p.headings = make([]HeadingInfo, 0)
	p.headingsMu.Unlock()

	// Parse the pre-processed source into an AST.
	reader := text.NewReader(processedSource)
	document := p.md.Parser().Parse(reader)
	diagnostics := CollectDiagnostics(document, processedSource)

	// Walk the AST to collect heading information.
	p.collectHeadings(document, processedSource, newSourceIndex(processedSource))

	// Render AST to HTML.
	var buf bytes.Buffer
	if err := p.md.Renderer().Render(&buf, processedSource, document); err != nil {
		return "", nil, nil, fmt.Errorf("渲染 Markdown 失败: %w", err)
	}

	// Post-process: GFM Alerts, Mermaid code blocks, etc.
	htmlResult := PostProcess(buf.String())

	// Restore math placeholders to KaTeX-recognizable HTML span elements.
	htmlResult = mathProc.postprocess(htmlResult)

	p.headingsMu.RLock()
	headingsCopy := make([]HeadingInfo, len(p.headings))
	copy(headingsCopy, p.headings)
	p.headingsMu.RUnlock()

	return htmlResult, headingsCopy, diagnostics, nil
}

// collectHeadings 递归遍历 AST 收集标题信息
func (p *Parser) collectHeadings(node ast.Node, source []byte, index *sourceIndex) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if heading, ok := child.(*ast.Heading); ok {
			info := p.extractHeadingInfo(heading, source, index)
			p.headingsMu.Lock()
			p.headings = append(p.headings, info)
			p.headingsMu.Unlock()
		}
		if child.HasChildren() {
			p.collectHeadings(child, source, index)
		}
	}
}

// extractHeadingInfo 从标题节点中提取信息
func (p *Parser) extractHeadingInfo(heading *ast.Heading, source []byte, index *sourceIndex) HeadingInfo {
	headingText := extractNodeText(heading, source)

	id := ""
	if idAttr, ok := heading.AttributeString("id"); ok {
		if idBytes, ok := idAttr.([]byte); ok {
			id = string(idBytes)
		}
	}
	if id == "" {
		id = generateHeadingID(headingText)
	}

	line, column := 0, 0
	if heading.Lines() != nil && heading.Lines().Len() > 0 {
		line, column = index.lineCol(heading.Lines().At(0).Start)
	}

	return HeadingInfo{
		Level:  heading.Level,
		Text:   headingText,
		ID:     id,
		Line:   line,
		Column: column,
	}
}

// SetCodeTheme 设置代码高亮主题
func (p *Parser) SetCodeTheme(theme string) {
	p.codeTheme = theme
	// Note: The highlighting library falls back to a default style on invalid themes,
	// so no validation is needed here. The goldmark library does not expose errors for
	// invalid themes during style initialization.
	p.initGoldmark()
}

// GetHeadings 获取当前收集的所有标题信息
func (p *Parser) GetHeadings() []HeadingInfo {
	p.headingsMu.RLock()
	defer p.headingsMu.RUnlock()
	headingsCopy := make([]HeadingInfo, len(p.headings))
	copy(headingsCopy, p.headings)
	return headingsCopy
}

// generateHeadingID 生成规范化的标题 ID
func generateHeadingID(text string) string {
	id := bytes.ToLower([]byte(text))
	// 移除非字母数字字符（保留中文等 Unicode 字符）
	id = headingIDStripRegexp.ReplaceAll(id, []byte(""))
	// 空格替换为连字符
	id = headingIDSpaceRegexp.ReplaceAll(id, []byte("-"))
	id = bytes.Trim(id, "-")
	if len(id) == 0 {
		return "heading"
	}
	return string(id)
}

// WithCodeTheme 选项：设置代码高亮主题
func WithCodeTheme(theme string) ParserOption {
	return func(p *Parser) {
		p.codeTheme = theme
	}
}
