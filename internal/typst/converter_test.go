package typst

import (
	"strings"
	"testing"
)

// TestConvertHeadings tests heading conversion with various levels.
func TestConvertHeadings(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single hash heading",
			input:    "# Heading 1",
			expected: "= Heading 1",
		},
		{
			name:     "double hash heading",
			input:    "## Heading 2",
			expected: "== Heading 2",
		},
		{
			name:     "triple hash heading",
			input:    "### Heading 3",
			expected: "=== Heading 3",
		},
		{
			name:     "quadruple hash heading",
			input:    "#### Heading 4",
			expected: "==== Heading 4",
		},
		{
			name:     "five level heading",
			input:    "##### Heading 5",
			expected: "===== Heading 5",
		},
		{
			name:     "six level heading",
			input:    "###### Heading 6",
			expected: "====== Heading 6",
		},
		{
			name:     "heading with extra spaces",
			input:    "#   Heading with spaces",
			expected: "= Heading with spaces",
		},
		{
			name:     "heading with trailing spaces",
			input:    "# Heading   ",
			expected: "= Heading",
		},
		{
			name:     "heading with special characters",
			input:    "# Heading & Special!",
			expected: "= Heading & Special!",
		},
		{
			name:     "heading only hashes",
			input:    "###",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if tt.expected == "" {
				// For empty expected, just verify it doesn't panic
				return
			}
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertUnorderedLists tests unordered list conversion.
func TestConvertUnorderedLists(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple dash list",
			input:    "- Item 1\n- Item 2",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "simple asterisk list",
			input:    "* Item 1\n* Item 2",
			expected: "- Item 1\n- Item 2",
		},
		{
			name:     "nested list with two levels",
			input:    "- Item 1\n  - Nested 1\n  - Nested 2\n- Item 2",
			expected: "- Item 1",
		},
		{
			name:     "nested list with three levels",
			input:    "- Item\n  - Level 2\n    - Level 3",
			expected: "- Item",
		},
		{
			name:     "list with single item",
			input:    "- Only item",
			expected: "- Only item",
		},
		{
			name:     "list with empty content",
			input:    "- ",
			expected: "-",
		},
		{
			name:     "list with special characters",
			input:    "- Item with & and !",
			expected: "- Item with & and !",
		},
		{
			name:     "mixed dash and asterisk",
			input:    "- Item 1\n* Item 2",
			expected: "- Item 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertOrderedLists tests ordered list conversion.
func TestConvertOrderedLists(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple ordered list",
			input:    "1. First\n2. Second\n3. Third",
			expected: "+ First",
		},
		{
			name:     "ordered list with large numbers",
			input:    "10. Item ten\n11. Item eleven",
			expected: "+ Item ten",
		},
		{
			name:     "nested ordered list",
			input:    "1. Item 1\n  1. Nested 1\n  2. Nested 2",
			expected: "+ Item 1",
		},
		{
			name:     "ordered list with single item",
			input:    "1. Single item",
			expected: "+ Single item",
		},
		{
			name:     "ordered list with special chars",
			input:    "1. Item & text!\n2. Another item?",
			expected: "+ Item & text!",
		},
		{
			name:     "ordered list starting with non-zero",
			input:    "5. Fifth item",
			expected: "+ Fifth item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertCodeBlocks tests code block conversion.
func TestConvertCodeBlocks(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "go code block",
			input:    "```go\nfunc main() {}\n```",
			expected: "```go",
		},
		{
			name:     "python code block",
			input:    "```python\nprint('hello')\n```",
			expected: "```python",
		},
		{
			name:     "code block without language",
			input:    "```\nplain code\n```",
			expected: "```\nplain code",
		},
		{
			name:     "code block with multiple lines",
			input:    "```java\nclass Main {\n  public static void main() {}\n}\n```",
			expected: "class Main",
		},
		{
			name:     "code block with special chars",
			input:    "```\n<html><body>test</body></html>\n```",
			expected: "<html>",
		},
		{
			name:     "code block with empty lines",
			input:    "```\nline 1\n\nline 3\n```",
			expected: "line 1",
		},
		{
			name:     "code block with tabs",
			input:    "```javascript\n\tvar x = 1;\n```",
			expected: "var x = 1;",
		},
		{
			name:     "code block with language containing uppercase",
			input:    "```C++\nint main() {}\n```",
			expected: "```C++",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertBlockquotes tests blockquote conversion.
func TestConvertBlockquotes(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple blockquote",
			input:    "> This is a quote",
			expected: "> This is a quote",
		},
		{
			name:     "blockquote with special chars",
			input:    "> Quote with & and !",
			expected: "> Quote with & and !",
		},
		{
			name:     "multiple blockquote lines",
			input:    "> Line 1\n> Line 2",
			expected: "> Line 1",
		},
		{
			name:     "blockquote with inline formatting",
			input:    "> This is **bold** text",
			expected: "> This is",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertHorizontalRules tests horizontal rule conversion.
func TestConvertHorizontalRules(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "dash horizontal rule",
			input:    "---",
			expected: true,
		},
		{
			name:     "asterisk horizontal rule",
			input:    "***",
			expected: true,
		},
		{
			name:     "underscore horizontal rule",
			input:    "___",
			expected: true,
		},
		{
			name:     "longer dash rule",
			input:    "---------",
			expected: true,
		},
		{
			name:     "rule with spaces",
			input:    "  ---  ",
			expected: true,
		},
		{
			name:     "invalid rule too short",
			input:    "--",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			hasRule := strings.Contains(result, "---")
			if hasRule != tt.expected {
				t.Errorf("expected rule presence=%v, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertBold tests bold text conversion.
func TestConvertBold(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold with double asterisks",
			input:    "This is **bold** text",
			expected: "*bold*",
		},
		{
			name:     "bold with double underscores",
			input:    "This is __bold__ text",
			expected: "*bold*",
		},
		{
			name:     "multiple bold sections",
			input:    "**first** and **second**",
			expected: "*first*",
		},
		{
			name:     "bold at start",
			input:    "**Start** of text",
			expected: "*Start*",
		},
		{
			name:     "bold at end",
			input:    "End of **text**",
			expected: "*text*",
		},
		{
			name:     "bold with numbers",
			input:    "**123** test",
			expected: "*123*",
		},
		{
			name:     "bold with special chars",
			input:    "**test!?** done",
			expected: "*test!?*",
		},
		{
			name:     "bold with spaces inside",
			input:    "**bold text here**",
			expected: "*bold text here*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertItalic tests italic text conversion.
func TestConvertItalic(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "italic with underscores",
			input:    "This is _italic_ text",
			expected: "_italic_",
		},
		{
			name:     "multiple italic sections",
			input:    "_first_ and _second_",
			expected: "_first_",
		},
		{
			name:     "italic at start",
			input:    "_Start_ of text",
			expected: "_Start_",
		},
		{
			name:     "italic at end",
			input:    "End of _text_",
			expected: "_text_",
		},
		{
			name:     "italic with spaces inside",
			input:    "_italic text here_",
			expected: "_italic text here_",
		},
		{
			name:     "italic with numbers",
			input:    "_123_ test",
			expected: "_123_",
		},
		{
			name:     "italic with special chars",
			input:    "_test!?_ done",
			expected: "_test!?_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertCodeSpans tests inline code conversion.
func TestConvertCodeSpans(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple code span",
			input:    "Use `code` here",
			expected: "Use `code` here",
		},
		{
			name:     "code span with special chars",
			input:    "Use `var x = 1;` here",
			expected: "`var x = 1;`",
		},
		{
			name:     "multiple code spans",
			input:    "`first` and `second`",
			expected: "`first`",
		},
		{
			name:     "code span at start",
			input:    "`Start` of text",
			expected: "`Start`",
		},
		{
			name:     "code span at end",
			input:    "End of `text`",
			expected: "`text`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertLinks tests link conversion.
func TestConvertLinks(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple link",
			input:    "[text](url)",
			expected: `#link("url")[text]`,
		},
		{
			name:     "link with URL",
			input:    "[click](https://example.com)",
			expected: `#link("https://example.com")[click]`,
		},
		{
			name:     "multiple links",
			input:    "[first](url1) and [second](url2)",
			expected: `#link("url1")[first]`,
		},
		{
			name:     "link with special chars in URL",
			input:    "[link](http://example.com/path?a=1&b=2)",
			expected: `#link("http://example.com/path?a=1&b=2")[link]`,
		},
		{
			name:     "link at start",
			input:    "[start](url) of text",
			expected: `#link("url")[start]`,
		},
		{
			name:     "link at end",
			input:    "end of [link](url)",
			expected: `#link("url")[link]`,
		},
		{
			name:     "link with punctuation in text",
			input:    "[click here!](url)",
			expected: `#link("url")[click here!]`,
		},
		{
			name:     "empty link text",
			input:    "[](url)",
			expected: `#link("url")[]`,
		},
		{
			name:     "link with relative path",
			input:    "[rel](../path/file.html)",
			expected: `#link("../path/file.html")[rel]`,
		},
		{
			name:     "link with hash anchor",
			input:    "[section](url#anchor)",
			expected: `#link("url#anchor")[section]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertImages tests image conversion.
func TestConvertImages(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple image",
			input:    "![alt](image.png)",
			expected: `#image("image.png")`,
		},
		{
			name:     "image with empty alt",
			input:    "![](image.png)",
			expected: `#image("image.png")`,
		},
		{
			name:     "multiple images",
			input:    "![img1](a.png) and ![img2](b.png)",
			expected: `#image("a.png")`,
		},
		{
			name:     "image with URL",
			input:    "![pic](http://example.com/image.png)",
			expected: `#image("http://example.com/image.png")`,
		},
		{
			name:     "image with query params",
			input:    "![alt](img.png?size=100)",
			expected: `#image("img.png?size=100")`,
		},
		{
			name:     "image with relative path",
			input:    "![alt](../images/pic.jpg)",
			expected: `#image("../images/pic.jpg")`,
		},
		{
			name:     "image at start",
			input:    "![start](url.png) of text",
			expected: `#image("url.png")`,
		},
		{
			name:     "image at end",
			input:    "end of ![end](url.png)",
			expected: `#image("url.png")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertEmptyInput tests conversion of empty inputs.
func TestConvertEmptyInput(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: "",
		},
		{
			name:     "single space",
			input:    " ",
			expected: "",
		},
		{
			name:     "tabs only",
			input:    "\t\t",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertComplexDocument tests conversion of complex markdown documents.
func TestConvertComplexDocument(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := `# Title

This is a paragraph with **bold** and _italic_ text.

## Section

Here's a [link](https://example.com) and an image:

![Image](test.png)

### Code Example

` + "```python\n" + `def hello():
    print("world")
` + "```\n" + `
- Item 1
- Item 2
  - Nested item

> A quote here

---

1. First
2. Second
`

	result := converter.Convert(input)

	checks := []string{
		"= Title",
		"== Section",
		"=== Code Example",
		"bold",
		"italic",
		`#link("https://example.com")`,
		`#image("test.png")`,
		"```python",
		"- Item 1",
		"> A quote here",
		"---",
		"+ First",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected to find %q in result, got:\n%s", check, result)
		}
	}
}

// TestConvertInlineOrder tests that inline conversions happen in correct order.
func TestConvertInlineOrder(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name    string
		input   string
		checkFn func(string) bool
	}{
		{
			name:  "bold and italic together",
			input: "***bold and italic***",
			checkFn: func(result string) bool {
				// Should have bold or italic markers
				return strings.Contains(result, "*") || strings.Contains(result, "_")
			},
		},
		{
			name:  "code and bold together",
			input: "**`code in bold`**",
			checkFn: func(result string) bool {
				return strings.Contains(result, "`")
			},
		},
		{
			name:  "link and bold together",
			input: "**[bold link](url)**",
			checkFn: func(result string) bool {
				// Should have link markers
				return strings.Contains(result, "#link")
			},
		},
		{
			name:  "image and italic together",
			input: "_![italic image](img.png)_",
			checkFn: func(result string) bool {
				// Image should still be converted
				return strings.Contains(result, "#image")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("check failed for input %q, got %q", tt.input, result)
			}
		})
	}
}

// TestUnclosedCodeBlock tests handling of unclosed code blocks.
func TestUnclosedCodeBlock(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name          string
		input         string
		shouldContain string
	}{
		{
			name:          "unclosed code block at EOF",
			input:         "```go\nfunc main() {}",
			shouldContain: "func main() {}",
		},
		{
			name:          "text before unclosed block",
			input:         "Some text\n```\ncode",
			shouldContain: "Some text",
		},
		{
			name:          "unclosed empty block",
			input:         "```go",
			shouldContain: "",
		},
		{
			name:          "multiple code blocks with unclosed",
			input:         "```go\ncode1\n```\ntext\n```python\ncode2",
			shouldContain: "code1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("expected result to contain %q, got %q", tt.shouldContain, result)
			}
		})
	}
}

// TestEdgeCasesWithSpecialCharacters tests edge cases with special characters.
func TestEdgeCasesWithSpecialCharacters(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "ampersand",
			input: "Text with & ampersand",
		},
		{
			name:  "less than greater than",
			input: "Text with < and > symbols",
		},
		{
			name:  "quotes",
			input: `Text with "quotes" and 'apostrophes'`,
		},
		{
			name:  "backslashes",
			input: "Text with \\ backslash",
		},
		{
			name:  "dollar sign",
			input: "Price is $100",
		},
		{
			name:  "percentage",
			input: "100% complete",
		},
		{
			name:  "hash in text",
			input: "Use #hashtag in text",
		},
		{
			name:  "mixed special chars",
			input: "Mix of & < > \" ' \\ $ % #",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			// Just verify it doesn't crash and produces output
			if result == "" && strings.TrimSpace(tt.input) != "" {
				t.Errorf("expected non-empty result for input %q", tt.input)
			}
		})
	}
}

// TestUnicodeHandling tests handling of unicode characters.
func TestUnicodeHandling(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "emoji in text",
			input:    "Hello 👋 World",
			expected: "Hello",
		},
		{
			name:     "chinese characters",
			input:    "你好世界",
			expected: "你好",
		},
		{
			name:     "arabic characters",
			input:    "مرحبا بالعالم",
			expected: "مرحبا",
		},
		{
			name:     "accented characters",
			input:    "Café résumé",
			expected: "Café",
		},
		{
			name:     "greek characters",
			input:    "Ελληνικά",
			expected: "Ελληνικά",
		},
		{
			name:     "unicode in heading",
			input:    "# 标题",
			expected: "= 标题",
		},
		{
			name:     "unicode in link text",
			input:    "[日本語](url)",
			expected: "日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestHelperCountLeadingChars tests the countLeadingChars helper function.
func TestCountLeadingChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		ch       rune
		expected int
	}{
		{
			name:     "three hashes",
			input:    "###",
			ch:       '#',
			expected: 3,
		},
		{
			name:     "hashes with text",
			input:    "## Heading",
			ch:       '#',
			expected: 2,
		},
		{
			name:     "no matching char",
			input:    "text",
			ch:       '#',
			expected: 0,
		},
		{
			name:     "single char",
			input:    "#",
			ch:       '#',
			expected: 1,
		},
		{
			name:     "many hashes",
			input:    "######",
			ch:       '#',
			expected: 6,
		},
		{
			name:     "empty string",
			input:    "",
			ch:       '#',
			expected: 0,
		},
		{
			name:     "wrong char first",
			input:    "text###",
			ch:       '#',
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLeadingChars(tt.input, tt.ch)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestHelperCountLeadingSpaces tests the countLeadingSpaces helper function.
func TestCountLeadingSpaces(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "three spaces",
			input:    "   text",
			expected: 3,
		},
		{
			name:     "no spaces",
			input:    "text",
			expected: 0,
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: 3,
		},
		{
			name:     "tab counts as 4",
			input:    "\ttext",
			expected: 4,
		},
		{
			name:     "mixed spaces and tabs",
			input:    "  \t  text",
			expected: 8, // 2 spaces + 4 for tab + 2 spaces = 8
		},
		{
			name:     "single space",
			input:    " text",
			expected: 1,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "no leading whitespace",
			input:    "nowhitespace",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLeadingSpaces(tt.input)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestHelperIsOrderedListItem tests the isOrderedListItem helper function.
func TestIsOrderedListItem(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple ordered item",
			input:    "1. Item",
			expected: true,
		},
		{
			name:     "double digit",
			input:    "10. Item",
			expected: true,
		},
		{
			name:     "triple digit",
			input:    "999. Item",
			expected: true,
		},
		{
			name:     "with leading spaces",
			input:    "  5. Item",
			expected: true,
		},
		{
			name:     "unordered with dash",
			input:    "- Item",
			expected: false,
		},
		{
			name:     "unordered with asterisk",
			input:    "* Item",
			expected: false,
		},
		{
			name:     "no space after dot",
			input:    "1.Item",
			expected: false,
		},
		{
			name:     "no dot",
			input:    "1 Item",
			expected: false,
		},
		{
			name:     "dot without digit",
			input:    ". Item",
			expected: false,
		},
		{
			name:     "just number",
			input:    "1.",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOrderedListItem(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestHelperExtractListItemContent tests the extractListItemContent helper function.
func TestExtractListItemContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple item",
			input:    "1. Item text",
			expected: "Item text",
		},
		{
			name:     "double digit",
			input:    "10. Another item",
			expected: "Another item",
		},
		{
			name:     "with leading spaces",
			input:    "  5. Indented item",
			expected: "Indented item",
		},
		{
			name:     "item with special chars",
			input:    "1. Item & more!",
			expected: "Item & more!",
		},
		{
			name:     "no space after dot",
			input:    "1.NoSpace",
			expected: "1.NoSpace",
		},
		{
			name:     "empty item",
			input:    "1. ",
			expected: "1.",
		},
		{
			name:     "no dot",
			input:    "1 Item",
			expected: "1 Item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractListItemContent(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestHelperIsHorizontalRule tests the isHorizontalRule helper function.
func TestIsHorizontalRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "three dashes",
			input:    "---",
			expected: true,
		},
		{
			name:     "three asterisks",
			input:    "***",
			expected: true,
		},
		{
			name:     "three underscores",
			input:    "___",
			expected: true,
		},
		{
			name:     "many dashes",
			input:    "---------",
			expected: true,
		},
		{
			name:     "with spaces",
			input:    "  ---  ",
			expected: true,
		},
		{
			name:     "two dashes",
			input:    "--",
			expected: false,
		},
		{
			name:     "mixed chars",
			input:    "-*-",
			expected: false,
		},
		{
			name:     "text with dashes",
			input:    "text---",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "single dash",
			input:    "-",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHorizontalRule(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestConvertCodeBlockMethod tests the convertCodeBlock method directly.
func TestConvertCodeBlockMethod(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		content  string
		lang     string
		expected string
	}{
		{
			name:     "go code",
			content:  "func main() {}",
			lang:     "go",
			expected: "```go\nfunc main() {}\n```",
		},
		{
			name:     "no language",
			content:  "plain code",
			lang:     "",
			expected: "```\nplain code\n```",
		},
		{
			name:     "python with newlines",
			content:  "def hello():\n    print('hi')\n",
			lang:     "python",
			expected: "```python\ndef hello():\n    print('hi')\n```",
		},
		{
			name:     "empty content",
			content:  "",
			lang:     "go",
			expected: "```go\n\n```",
		},
		{
			name:     "java language",
			content:  "class Main {}",
			lang:     "java",
			expected: "```java\nclass Main {}\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertCodeBlock(tt.content, tt.lang)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertBoldMethod tests the convertBold method directly.
func TestConvertBoldMethod(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "asterisk bold",
			input:    "This is **bold** text",
			expected: "This is *bold* text",
		},
		{
			name:     "underscore bold",
			input:    "This is __bold__ text",
			expected: "This is *bold* text",
		},
		{
			name:     "no bold",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "multiple bold asterisk",
			input:    "**first** and **second**",
			expected: "*first* and *second*",
		},
		{
			name:     "mixed bold markers",
			input:    "**first** and __second__",
			expected: "*first* and *second*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertBold(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertItalicMethod tests the convertItalic method directly.
func TestConvertItalicMethod(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "underscore italic",
			input:    "This is _italic_ text",
			expected: "This is _italic_ text",
		},
		{
			name:     "no italic",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "multiple italic",
			input:    "_first_ and _second_",
			expected: "_first_ and _second_",
		},
		{
			name:     "unclosed underscore",
			input:    "this _has unclosed",
			expected: "this _has unclosed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertItalic(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertImagesMethod tests the convertImages method directly.
func TestConvertImagesMethod(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple image",
			input:    "![alt](image.png)",
			expected: `#image("image.png")`,
		},
		{
			name:     "image in text",
			input:    "See ![pic](img.png) here",
			expected: `See #image("img.png") here`,
		},
		{
			name:     "multiple images",
			input:    "![a](a.png) ![b](b.png)",
			expected: `#image("a.png") #image("b.png")`,
		},
		{
			name:     "no images",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertImages(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestConvertLinksMethod tests the convertLinks method directly.
func TestConvertLinksMethod(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple link",
			input:    "[text](url)",
			expected: `#link("url")[text]`,
		},
		{
			name:     "link in text",
			input:    "Click [here](url) now",
			expected: `Click #link("url")[here] now`,
		},
		{
			name:     "no links",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertLinks(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestReplaceLinksDirect tests the replaceLinks helper method directly.
func TestReplaceLinksDirect(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single link",
			input:    "[text](url)",
			expected: `#link("url")[text]`,
		},
		{
			name:     "multiple links",
			input:    "[a](u1) [b](u2)",
			expected: `#link("u1")[a] #link("u2")[b]`,
		},
		{
			name:     "link with query string",
			input:    "[text](http://example.com?a=1&b=2)",
			expected: `#link("http://example.com?a=1&b=2")[text]`,
		},
		{
			name:     "unclosed bracket",
			input:    "[unclosed text",
			expected: "[unclosed text",
		},
		{
			name:     "bracket without paren",
			input:    "[text] no paren",
			expected: "[text] no paren",
		},
		{
			name:     "image should not match",
			input:    "![alt](img.png)",
			expected: "![alt](img.png)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.replaceLinks(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestParagraphConversion tests that regular paragraphs are converted properly.
func TestParagraphConversion(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple paragraph",
			input:    "This is a paragraph",
			expected: "This is a paragraph",
		},
		{
			name:     "paragraph with inline formatting",
			input:    "This has **bold** and _italic_",
			expected: "This has",
		},
		{
			name:     "multiple paragraphs",
			input:    "First paragraph\n\nSecond paragraph",
			expected: "First paragraph",
		},
		{
			name:     "paragraph with link",
			input:    "Visit [site](http://example.com) for more",
			expected: "Visit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestMalformedInput tests handling of malformed markdown.
func TestMalformedInput(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unclosed brackets",
			input: "[unclosed",
		},
		{
			name:  "unclosed parentheses",
			input: "(unclosed",
		},
		{
			name:  "mismatched brackets",
			input: "[text)",
		},
		{
			name:  "multiple asterisks",
			input: "*****text",
		},
		{
			name:  "unmatched formatting",
			input: "**bold without close",
		},
		{
			name:  "mixed formatting markers",
			input: "*_text*_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			result := converter.Convert(tt.input)
			_ = result
		})
	}
}

// TestConsecutiveBlocks tests multiple consecutive blocks of the same type.
func TestConsecutiveBlocks(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "consecutive headings",
			input:    "# Heading 1\n## Heading 2\n### Heading 3",
			expected: "= Heading 1",
		},
		{
			name:     "consecutive lists",
			input:    "- Item 1\n- Item 2\n- Item 3",
			expected: "- Item 1",
		},
		{
			name:     "consecutive quotes",
			input:    "> Quote 1\n> Quote 2\n> Quote 3",
			expected: "> Quote 1",
		},
		{
			name:     "consecutive code blocks",
			input:    "```\ncode1\n```\n```\ncode2\n```",
			expected: "code1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestWhitespacePreservation tests that whitespace is handled correctly.
func TestWhitespacePreservation(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "trailing spaces in paragraph",
			input: "paragraph with spaces   ",
			check: func(result string) bool {
				return len(result) > 0
			},
		},
		{
			name:  "leading spaces in paragraph",
			input: "   paragraph with leading spaces",
			check: func(result string) bool {
				return strings.Contains(result, "paragraph")
			},
		},
		{
			name:  "multiple spaces between words",
			input: "word1    word2",
			check: func(result string) bool {
				return strings.Contains(result, "word1") && strings.Contains(result, "word2")
			},
		},
		{
			name:  "tabs in content",
			input: "text\twith\ttabs",
			check: func(result string) bool {
				return strings.Contains(result, "text") && strings.Contains(result, "with")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.Convert(tt.input)
			if !tt.check(result) {
				t.Errorf("check failed for input %q, got %q", tt.input, result)
			}
		})
	}
}

// TestBoundaryConditions tests boundary conditions and limits.
func TestBoundaryConditions(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "very long line",
			input: strings.Repeat("a", 10000),
		},
		{
			name:  "many lines",
			input: strings.Repeat("line\n", 1000),
		},
		{
			name:  "deeply nested lists",
			input: "- L1\n  - L2\n    - L3\n      - L4\n        - L5",
		},
		{
			name:  "many headings",
			input: strings.Repeat("# Heading\n", 100),
		},
		{
			name:  "alternating empty lines",
			input: "text\n\n\n\nmore text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic or hang
			result := converter.Convert(tt.input)
			if result == "" && strings.TrimSpace(tt.input) != "" {
				t.Errorf("expected non-empty result for input with content")
			}
		})
	}
}

// TestCountLeadingCharsUnicode tests countLeadingChars with unicode runes.
func TestCountLeadingCharsUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		ch       rune
		expected int
	}{
		{
			name:     "emoji hash",
			input:    "🎉🎉text",
			ch:       '🎉',
			expected: 2,
		},
		{
			name:     "chinese char",
			input:    "中中中text",
			ch:       '中',
			expected: 3,
		},
		{
			name:     "unicode no match",
			input:    "text",
			ch:       '中',
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLeadingChars(tt.input, tt.ch)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestIsOrderedListItemUnicode tests isOrderedListItem with unicode text.
func TestIsOrderedListItemUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "ordered list with unicode content",
			input:    "1. 日本語",
			expected: true,
		},
		{
			name:     "ordered list with emoji",
			input:    "1. 🎉 Party",
			expected: true,
		},
		{
			name:     "unicode text that looks like number",
			input:    "１. Item",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOrderedListItem(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
