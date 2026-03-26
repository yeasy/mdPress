package markdown

import (
	"testing"
)

// FuzzParse fuzzes the Markdown parser with arbitrary input.
func FuzzParse(f *testing.F) {
	// Seed with common Markdown patterns.
	seeds := []string{
		"# Heading\n\nParagraph text.",
		"## Sub heading\n\n- item 1\n- item 2",
		"```go\nfunc main() {}\n```",
		"| A | B |\n| --- | --- |\n| 1 | 2 |",
		"[link](https://example.com)",
		"![image](img.png)",
		"> blockquote\n> line 2",
		"---",
		"**bold** and *italic*",
		"text with `inline code`",
		"1. ordered\n2. list",
		"- [ ] task\n- [x] done",
		"footnote[^1]\n\n[^1]: definition",
		"~~strikethrough~~",
		"<div>html content</div>",
		"# 第一章 中文标题\n\n这是一段中文。",
		"",
		"# " + string(make([]byte, 1000)),
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Create a fresh parser per iteration to avoid stale usedIDs state.
		parser := NewParser()
		// The parser should never panic on any input.
		_, _, _ = parser.Parse(data)
		_, _, _, _ = parser.ParseWithDiagnostics(data)
	})
}
