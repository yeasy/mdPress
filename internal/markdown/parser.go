// Package markdown provides Markdown parsing and HTML conversion.
// Built on the goldmark library, it supports GFM extensions, syntax highlighting, footnotes, and more.
package markdown

import (
	"bytes"
	"fmt"
	"regexp"

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

// HeadingInfo holds heading metadata, used for TOC generation.
type HeadingInfo struct {
	Level  int    // Heading level (1-6)
	Text   string // Heading text content
	ID     string // Heading ID, used for cross-references
	Line   int    // Line number of the heading
	Column int    // Column number of the heading
}

// ParserOption is a functional option type.
type ParserOption func(*Parser)

// Parser is the Markdown parser.
type Parser struct {
	md        goldmark.Markdown
	codeTheme string
}

// NewParser creates and returns a new Markdown parser instance.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		codeTheme: "github",
	}

	for _, opt := range opts {
		opt(p)
	}

	p.initGoldmark()
	return p
}

// initGoldmark initializes the goldmark parser and all extensions.
func (p *Parser) initGoldmark() {
	exts := []goldmark.Extender{
		// GFM extensions.
		extension.NewTable(),
		extension.Strikethrough,
		extension.TaskList,
		extension.Linkify,
		// Footnotes.
		extension.Footnote,
		// Syntax highlighting.
		highlighting.NewHighlighting(
			highlighting.WithStyle(p.codeTheme),
		),
	}

	p.md = goldmark.New(
		goldmark.WithExtensions(exts...),
		goldmark.WithParserOptions(
			// NOTE: Do NOT use parser.WithAutoHeadingID() here — Goldmark's
			// built-in auto-ID generator strips CJK characters, producing
			// meaningless IDs like "heading" or "41-".  Our custom
			// headingIDTransformer preserves Unicode letters (\p{L}) and
			// generates correct CJK-aware IDs such as "41-归纳法与机器学习".
			parser.WithASTTransformers(
				util.Prioritized(newHeadingIDTransformer(), 100),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow raw HTML.
		),
	)
}

// Parse parses Markdown source and returns HTML and heading information.
func (p *Parser) Parse(source []byte) (string, []HeadingInfo, error) {
	html, headings, _, err := p.ParseWithDiagnostics(source)
	return html, headings, err
}

// ParseWithDiagnostics parses Markdown and also returns build-time warnings.
func (p *Parser) ParseWithDiagnostics(source []byte) (string, []HeadingInfo, []Diagnostic, error) {
	if len(source) == 0 {
		return "", []HeadingInfo{}, nil, nil
	}

	// Pre-process math formulas: replace $$...$$ and $...$ with safe placeholder
	// tokens to prevent goldmark from treating _ inside formulas as emphasis.
	mathProc := newMathPreprocessor()
	processedSource := []byte(mathProc.preprocess(string(source)))

	// Parse the pre-processed source into an AST.
	reader := text.NewReader(processedSource)
	document := p.md.Parser().Parse(reader)
	diagnostics := collectDiagnostics(document, processedSource)

	// Walk the AST to collect heading information using local state
	// (no shared mutable struct fields) so concurrent Parse calls are safe.
	headings := p.collectHeadings(document, processedSource, newSourceIndex(processedSource))

	// Render AST to HTML.
	var buf bytes.Buffer
	if err := p.md.Renderer().Render(&buf, processedSource, document); err != nil {
		return "", nil, nil, fmt.Errorf("failed to render Markdown: %w", err)
	}

	// Post-process: GFM Alerts, Mermaid code blocks, etc.
	htmlResult := postProcess(buf.String())

	// Restore math placeholders to KaTeX-recognizable HTML span elements.
	htmlResult = mathProc.postprocess(htmlResult)

	return htmlResult, headings, diagnostics, nil
}

// collectHeadings recursively walks the AST to collect heading information.
// It returns the collected headings instead of mutating struct state, making
// concurrent Parse calls on the same Parser safe.
func (p *Parser) collectHeadings(node ast.Node, source []byte, index *sourceIndex) []HeadingInfo {
	var headings []HeadingInfo
	p.collectHeadingsRecurse(node, source, index, &headings)
	return headings
}

// collectHeadingsRecurse is the recursive helper for collectHeadings.
func (p *Parser) collectHeadingsRecurse(node ast.Node, source []byte, index *sourceIndex, headings *[]HeadingInfo) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if heading, ok := child.(*ast.Heading); ok {
			*headings = append(*headings, p.extractHeadingInfo(heading, source, index))
		}
		if child.HasChildren() {
			p.collectHeadingsRecurse(child, source, index, headings)
		}
	}
}

// extractHeadingInfo extracts information from a heading node.
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

// SetCodeTheme sets the syntax highlighting theme.
func (p *Parser) SetCodeTheme(theme string) {
	p.codeTheme = theme
	// Note: The highlighting library falls back to a default style on invalid themes,
	// so no validation is needed here. The goldmark library does not expose errors for
	// invalid themes during style initialization.
	p.initGoldmark()
}

// generateHeadingID generates a normalized heading ID.
func generateHeadingID(text string) string {
	id := bytes.ToLower([]byte(text))
	// Remove non-alphanumeric characters (preserving CJK and other Unicode letters).
	id = headingIDStripRegexp.ReplaceAll(id, []byte(""))
	// Replace spaces with hyphens.
	id = headingIDSpaceRegexp.ReplaceAll(id, []byte("-"))
	id = bytes.Trim(id, "-")
	if len(id) == 0 {
		return "heading"
	}
	return string(id)
}

// WithCodeTheme is an option that sets the syntax highlighting theme.
func WithCodeTheme(theme string) ParserOption {
	return func(p *Parser) {
		p.codeTheme = theme
	}
}
