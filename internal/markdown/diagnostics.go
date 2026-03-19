package markdown

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
)

// Diagnostic 表示构建期间发现的文档问题。
type Diagnostic struct {
	Rule    string
	Line    int
	Column  int
	Message string
}

// Position 返回适合日志输出的位置字符串。
func (d Diagnostic) Position() string {
	if d.Line <= 0 {
		return "-"
	}
	if d.Column <= 0 {
		return fmt.Sprintf("%d", d.Line)
	}
	return fmt.Sprintf("%d:%d", d.Line, d.Column)
}

var (
	orderedListMarkerPattern = regexp.MustCompile(`^(\s*)(\d+)([.)])\s+`)
	fenceLinePattern         = regexp.MustCompile(`^( {0,3})(` + "```" + `+|~~~+)([ \t]*)(.*)$`)
)

var mermaidDiagramPrefixes = []string{
	"architecture-beta",
	"block-beta",
	"c4component",
	"c4container",
	"c4context",
	"c4deployment",
	"c4dynamic",
	"classdiagram",
	"erdiagram",
	"flowchart",
	"gantt",
	"gitgraph",
	"journey",
	"kanban",
	"mindmap",
	"packet-beta",
	"pie",
	"quadrantchart",
	"requirementdiagram",
	"sankey-beta",
	"sequencediagram",
	"statediagram",
	"statediagram-v2",
	"timeline",
	"xychart-beta",
	"graph",
}

type sourceIndex struct {
	source     []byte
	lineStarts []int
}

func newSourceIndex(source []byte) *sourceIndex {
	starts := []int{0}
	for i, b := range source {
		if b == '\n' && i+1 < len(source) {
			starts = append(starts, i+1)
		}
	}
	return &sourceIndex{
		source:     source,
		lineStarts: starts,
	}
}

func (s *sourceIndex) lineCol(offset int) (int, int) {
	if len(s.lineStarts) == 0 {
		return 1, 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(s.source) {
		offset = len(s.source)
	}
	lineIdx := sort.Search(len(s.lineStarts), func(i int) bool {
		return s.lineStarts[i] > offset
	}) - 1
	if lineIdx < 0 {
		lineIdx = 0
	}
	lineStart := s.lineStarts[lineIdx]
	col := runeCountBytes(s.source[lineStart:offset]) + 1
	return lineIdx + 1, col
}

func (s *sourceIndex) lineText(line int) string {
	if line <= 0 {
		return ""
	}
	idx := line - 1
	if idx >= len(s.lineStarts) {
		return ""
	}
	start := s.lineStarts[idx]
	end := len(s.source)
	if idx+1 < len(s.lineStarts) {
		end = s.lineStarts[idx+1]
	}
	return strings.TrimRight(string(s.source[start:end]), "\r\n")
}

// CollectDiagnostics 收集 Markdown 文档中的结构化 warning。
func CollectDiagnostics(document ast.Node, source []byte) []Diagnostic {
	index := newSourceIndex(source)
	diagnostics := collectOrderedListDiagnostics(document, source, index)
	diagnostics = append(diagnostics, collectMermaidDiagnostics(source)...)
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Line != diagnostics[j].Line {
			return diagnostics[i].Line < diagnostics[j].Line
		}
		if diagnostics[i].Column != diagnostics[j].Column {
			return diagnostics[i].Column < diagnostics[j].Column
		}
		return diagnostics[i].Rule < diagnostics[j].Rule
	})
	return diagnostics
}

func collectOrderedListDiagnostics(document ast.Node, source []byte, index *sourceIndex) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)
	_ = document

	lastMarkerByIndent := make(map[int]int)
	lines := strings.Split(string(source), "\n")
	for i, rawLine := range lines {
		lineText := strings.TrimRight(rawLine, "\r")
		match := orderedListMarkerPattern.FindStringSubmatchIndex(lineText)
		if match == nil {
			if strings.TrimSpace(lineText) != "" {
				clear(lastMarkerByIndent)
			}
			continue
		}

		indentWidth := utf8.RuneCountInString(lineText[:match[2]])
		actual, err := strconv.Atoi(lineText[match[4]:match[5]])
		if err != nil {
			continue
		}
		column := utf8.RuneCountInString(lineText[:match[4]]) + 1

		for depth := range lastMarkerByIndent {
			if depth > indentWidth {
				delete(lastMarkerByIndent, depth)
			}
		}

		if prevMarker, ok := lastMarkerByIndent[indentWidth]; ok && !(prevMarker == 1 && actual == 1) && actual != prevMarker+1 {
			diagnostics = append(diagnostics, Diagnostic{
				Rule:    "ordered-list-sequence",
				Line:    i + 1,
				Column:  column,
				Message: fmt.Sprintf("有序列表编号不连续：期望 %d，实际为 %d", prevMarker+1, actual),
			})
		}
		lastMarkerByIndent[indentWidth] = actual
	}
	return diagnostics
}

type mermaidFence struct {
	startLine   int
	startColumn int
	content     []string
}

func collectMermaidDiagnostics(source []byte) []Diagnostic {
	lines := strings.Split(string(source), "\n")
	diagnostics := make([]Diagnostic, 0)

	var current *mermaidFence
	var fenceChar byte
	var fenceLen int

	for i, rawLine := range lines {
		lineNo := i + 1
		line := strings.TrimRight(rawLine, "\r")

		if current == nil {
			info, ok := parseFenceOpen(line)
			if !ok {
				continue
			}
			if strings.EqualFold(firstFenceToken(info.rest), "mermaid") {
				current = &mermaidFence{
					startLine:   lineNo,
					startColumn: len(info.indent) + 1,
				}
				fenceChar = info.marker[0]
				fenceLen = len(info.marker)
			}
			continue
		}

		if isFenceClose(line, fenceChar, fenceLen) {
			diagnostics = append(diagnostics, validateMermaidFence(current)...)
			current = nil
			fenceChar = 0
			fenceLen = 0
			continue
		}

		current.content = append(current.content, line)
	}

	if current != nil {
		diagnostics = append(diagnostics, Diagnostic{
			Rule:    "mermaid-unclosed-fence",
			Line:    current.startLine,
			Column:  current.startColumn,
			Message: "Mermaid 代码块未闭合，缺少结束 fence",
		})
	}

	return diagnostics
}

type fenceOpen struct {
	indent string
	marker string
	rest   string
}

func parseFenceOpen(line string) (fenceOpen, bool) {
	matches := fenceLinePattern.FindStringSubmatch(line)
	if matches == nil {
		return fenceOpen{}, false
	}
	return fenceOpen{
		indent: matches[1],
		marker: matches[2],
		rest:   strings.TrimSpace(matches[4]),
	}, true
}

func isFenceClose(line string, fenceChar byte, fenceLen int) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return false
	}
	if len(trimmed) < fenceLen {
		return false
	}
	if strings.Trim(trimmed[fenceLen:], " \t") != "" {
		return false
	}
	for i := 0; i < fenceLen; i++ {
		if trimmed[i] != fenceChar {
			return false
		}
	}
	for i := fenceLen; i < len(trimmed); i++ {
		if trimmed[i] != fenceChar {
			return false
		}
	}
	return true
}

func firstFenceToken(rest string) string {
	if rest == "" {
		return ""
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func validateMermaidFence(block *mermaidFence) []Diagnostic {
	diagnostics := make([]Diagnostic, 0)

	firstLine, firstColumn, firstContent := firstMeaningfulMermaidLine(block)
	if firstContent == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Rule:    "mermaid-empty",
			Line:    block.startLine,
			Column:  block.startColumn,
			Message: "Mermaid 图为空，没有可渲染的内容",
		})
		return diagnostics
	}

	if !isKnownMermaidDiagram(firstContent) {
		diagnostics = append(diagnostics, Diagnostic{
			Rule:    "mermaid-unknown-diagram",
			Line:    firstLine,
			Column:  firstColumn,
			Message: fmt.Sprintf("Mermaid 图首个有效语句不是已知图类型：%q", firstContent),
		})
	}

	if diag, ok := findMermaidBracketIssue(block); ok {
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics
}

func firstMeaningfulMermaidLine(block *mermaidFence) (int, int, string) {
	for i, line := range block.content {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "%%") {
			continue
		}
		col := utf8.RuneCountInString(line[:len(line)-len(strings.TrimLeft(line, " \t"))]) + 1
		return block.startLine + i + 1, col, trimmed
	}
	return 0, 0, ""
}

func isKnownMermaidDiagram(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	for _, prefix := range mermaidDiagramPrefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+" ") {
			return true
		}
	}
	return false
}

type mermaidBracket struct {
	char   rune
	line   int
	column int
}

func findMermaidBracketIssue(block *mermaidFence) (Diagnostic, bool) {
	stack := make([]mermaidBracket, 0)
	var quote rune

	for i, rawLine := range block.content {
		lineNo := block.startLine + i + 1
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(strings.TrimSpace(line), "%%") {
			continue
		}

		for byteIdx, r := range line {
			if quote != 0 {
				if r == quote {
					quote = 0
				}
				continue
			}

			if r == '"' || r == '\'' {
				quote = r
				continue
			}

			switch r {
			case '(', '[', '{':
				stack = append(stack, mermaidBracket{
					char:   r,
					line:   lineNo,
					column: utf8.RuneCountInString(line[:byteIdx]) + 1,
				})
			case ')', ']', '}':
				if len(stack) == 0 {
					return Diagnostic{
						Rule:    "mermaid-bracket-mismatch",
						Line:    lineNo,
						Column:  utf8.RuneCountInString(line[:byteIdx]) + 1,
						Message: fmt.Sprintf("Mermaid 图括号不匹配：多余的 %q", string(r)),
					}, true
				}
				open := stack[len(stack)-1]
				if !isMatchingBracket(open.char, r) {
					return Diagnostic{
						Rule:    "mermaid-bracket-mismatch",
						Line:    lineNo,
						Column:  utf8.RuneCountInString(line[:byteIdx]) + 1,
						Message: fmt.Sprintf("Mermaid 图括号不匹配：遇到 %q，但最近的未闭合括号是 %q", string(r), string(open.char)),
					}, true
				}
				stack = stack[:len(stack)-1]
			}
		}
	}

	if len(stack) > 0 {
		open := stack[len(stack)-1]
		return Diagnostic{
			Rule:    "mermaid-bracket-mismatch",
			Line:    open.line,
			Column:  open.column,
			Message: fmt.Sprintf("Mermaid 图括号未闭合：缺少与 %q 对应的闭合符", string(open.char)),
		}, true
	}

	return Diagnostic{}, false
}

func isMatchingBracket(open, close rune) bool {
	switch {
	case open == '(' && close == ')':
		return true
	case open == '[' && close == ']':
		return true
	case open == '{' && close == '}':
		return true
	default:
		return false
	}
}

func runeCountBytes(b []byte) int {
	return utf8.RuneCountInString(string(b))
}
