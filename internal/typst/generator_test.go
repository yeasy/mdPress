package typst

import (
	"strings"
	"testing"
	"time"
)

// TestConverterHeadings tests heading conversion.
func TestConverterHeadings(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		input    string
		contains string
	}{
		{
			input:    "# Heading 1",
			contains: "= Heading 1",
		},
		{
			input:    "## Heading 2",
			contains: "== Heading 2",
		},
		{
			input:    "### Heading 3",
			contains: "=== Heading 3",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.contains) {
			t.Errorf("input %q: expected to contain %q, got %q", test.input, test.contains, result)
		}
	}
}

// TestConverterBold tests bold formatting conversion.
func TestConverterBold(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		input    string
		notEmpty bool
	}{
		{
			input:    "This is **bold** text",
			notEmpty: true,
		},
		{
			input:    "This is __bold__ text",
			notEmpty: true,
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if test.notEmpty && result == "" {
			t.Errorf("input %q: expected non-empty result", test.input)
		}
		// In Typst, both * and _ can denote emphasis, so just verify it has content
		if !strings.Contains(result, "bold") {
			t.Errorf("input %q: expected 'bold' in result, got %q", test.input, result)
		}
	}
}

// TestConverterItalic tests italic formatting conversion.
func TestConverterItalic(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		input string
		check string
	}{
		{
			input: "This is *italic* text",
			check: "italic",
		},
		{
			input: "This is _italic_ text",
			check: "italic",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.check) {
			t.Errorf("input %q: expected to contain %q, got %q", test.input, test.check, result)
		}
	}
}

// TestConverterCodeSpans tests inline code conversion.
func TestConverterCodeSpans(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		input    string
		contains string
	}{
		{
			input:    "Use `code` here",
			contains: "Use `code` here",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.contains) {
			t.Errorf("input %q: expected to contain %q, got %q", test.input, test.contains, result)
		}
	}
}

// TestConverterLinks tests link conversion.
func TestConverterLinks(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := "Visit [example](https://example.com) for more"
	result := converter.Convert(input)

	if !strings.Contains(result, `#link("https://example.com")`) {
		t.Errorf("input %q: expected to contain link markup, got %q", input, result)
	}
	if !strings.Contains(result, "[example]") {
		t.Errorf("input %q: expected to contain link text, got %q", input, result)
	}
}

// TestConverterImages tests image conversion.
func TestConverterImages(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := "![Alt text](image.png)"
	result := converter.Convert(input)

	if !strings.Contains(result, `#image("image.png")`) {
		t.Errorf("input %q: expected to contain image markup, got %q", input, result)
	}
}

// TestConverterLists tests list conversion.
func TestConverterLists(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		input    string
		contains string
	}{
		{
			input:    "- Item 1\n- Item 2",
			contains: "- Item 1",
		},
		{
			input:    "* Item 1\n* Item 2",
			contains: "- Item 1",
		},
		{
			input:    "1. Item 1\n2. Item 2",
			contains: "+ Item 1",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.contains) {
			t.Errorf("input %q: expected to contain %q, got %q", test.input, test.contains, result)
		}
	}
}

// TestConverterBlockquote tests blockquote conversion.
func TestConverterBlockquote(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := "> This is a quote"
	result := converter.Convert(input)

	if !strings.Contains(result, "> This is a quote") {
		t.Errorf("input %q: expected blockquote, got %q", input, result)
	}
}

// TestConverterCodeBlock tests code block conversion.
func TestConverterCodeBlock(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := "```go\nfunc main() {}\n```"
	result := converter.Convert(input)

	if !strings.Contains(result, "```go") {
		t.Errorf("input %q: expected code block with language, got %q", input, result)
	}
	if !strings.Contains(result, "func main() {}") {
		t.Errorf("input %q: expected code content, got %q", input, result)
	}
}

// TestConverterComplexDocument tests a more complex markdown document.
func TestConverterComplexDocument(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	input := `# Title

This is a paragraph with **bold** and *italic* text.

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
`

	result := converter.Convert(input)

	// Check that key elements are present
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
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected to find %q in result, got:\n%s", check, result)
		}
	}
}

// TestPageDimensions tests page dimension lookup.
func TestPageDimensions(t *testing.T) {
	tests := []struct {
		size           string
		expectedWidth  string
		expectedHeight string
	}{
		{"A4", "210mm", "297mm"},
		{"A5", "148mm", "210mm"},
		{"Letter", "216mm", "279mm"},
		{"Legal", "216mm", "356mm"},
		{"unknown", "210mm", "297mm"}, // Should default to A4
	}

	for _, test := range tests {
		width, height := GetPageDimensions(test.size)
		if width != test.expectedWidth || height != test.expectedHeight {
			t.Errorf("size %q: expected %s x %s, got %s x %s",
				test.size, test.expectedWidth, test.expectedHeight, width, height)
		}
	}
}

// TestTemplateRendering tests Typst template rendering.
func TestTemplateRendering(t *testing.T) {
	data := TypstTemplateData{
		Title:        "Test Book",
		Author:       "Test Author",
		Date:         "2026-03-19",
		Version:      "1.0.0",
		Language:     "en",
		Content:      "# Chapter 1\n\nContent here.",
		PageWidth:    "210",
		PageHeight:   "297",
		MarginTop:    "20mm",
		MarginRight:  "20mm",
		MarginBottom: "20mm",
		MarginLeft:   "20mm",
		FontFamily:   "Segoe UI",
		FontSize:     "12pt",
		LineHeight:   1.6,
	}

	result, err := RenderTypstDocument(data)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	// Check that key elements are in the template
	checks := []string{
		"#set page(",
		"#set text(",
		"Test Book",
		"Test Author",
		"# Chapter 1",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected template to contain %q, got:\n%s", check, result)
		}
	}
}

// TestGeneratorCreation tests that a Generator can be created with options.
func TestGeneratorCreation(t *testing.T) {
	gen := NewGenerator(
		WithTitle("My Book"),
		WithAuthor("John Doe"),
		WithPageSize("Letter"),
		WithFontSize("14pt"),
		WithLineHeight(1.8),
	)

	if gen.title != "My Book" {
		t.Errorf("expected title %q, got %q", "My Book", gen.title)
	}
	if gen.author != "John Doe" {
		t.Errorf("expected author %q, got %q", "John Doe", gen.author)
	}
	if gen.pageSize != "Letter" {
		t.Errorf("expected pageSize %q, got %q", "Letter", gen.pageSize)
	}
	if gen.fontSize != "14pt" {
		t.Errorf("expected fontSize %q, got %q", "14pt", gen.fontSize)
	}
	if gen.lineHeight != 1.8 {
		t.Errorf("expected lineHeight %f, got %f", 1.8, gen.lineHeight)
	}
}

// TestCurrentDate tests that CurrentDate returns a reasonable date.
func TestCurrentDate(t *testing.T) {
	date := CurrentDate()
	if len(date) == 0 {
		t.Error("expected non-empty date")
	}
	// Check basic YYYY-MM-DD format
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		t.Errorf("expected YYYY-MM-DD format, got %q", date)
	}
}

// TestMakeTypstFont tests font family conversion.
func TestMakeTypstFont(t *testing.T) {
	result := MakeTypstFont("Arial, sans-serif")
	if result == "" {
		t.Error("expected non-empty font result")
	}
	if !strings.Contains(result, "Arial") && !strings.Contains(result, "sans-serif") {
		t.Errorf("expected font to contain familiar names, got %q", result)
	}
}

// TestMakeTypstFontSize tests font size conversion.
func TestMakeTypstFontSize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12pt", "12pt"},
		{"14px", "10.5pt"},
		{"", "12pt"}, // Default
	}

	for _, test := range tests {
		result := MakeTypstFontSize(test.input)
		if result != test.expected {
			t.Errorf("input %q: expected %q, got %q", test.input, test.expected, result)
		}
	}
}

// TestConvertMarginToTypst tests margin conversion.
func TestConvertMarginToTypst(t *testing.T) {
	tests := []struct {
		margin     string
		defaultVal string
		expected   string
	}{
		{"20mm", "15mm", "20mm"},
		{"", "15mm", "15mm"},
		{"1in", "20mm", "1in"},
	}

	for _, test := range tests {
		result := ConvertMarginToTypst(test.margin, test.defaultVal)
		if result != test.expected {
			t.Errorf("input %q with default %q: expected %q, got %q",
				test.margin, test.defaultVal, test.expected, result)
		}
	}
}

// TestHelperFunctions tests various helper functions.
func TestHelperFunctions(t *testing.T) {
	// Test countLeadingChars
	if countLeadingChars("###", '#') != 3 {
		t.Error("countLeadingChars failed for ###")
	}
	if countLeadingChars("# heading", '#') != 1 {
		t.Error("countLeadingChars failed for # heading")
	}

	// Test countLeadingSpaces
	if countLeadingSpaces("   text") != 3 {
		t.Error("countLeadingSpaces failed")
	}
	if countLeadingSpaces("text") != 0 {
		t.Error("countLeadingSpaces failed for no leading spaces")
	}

	// Test isOrderedListItem
	if !isOrderedListItem("1. Item") {
		t.Error("isOrderedListItem failed for valid item")
	}
	if isOrderedListItem("- Item") {
		t.Error("isOrderedListItem failed for unordered item")
	}

	// Test isHorizontalRule
	if !isHorizontalRule("---") {
		t.Error("isHorizontalRule failed for ---")
	}
	if !isHorizontalRule("***") {
		t.Error("isHorizontalRule failed for ***")
	}
	if isHorizontalRule("--") {
		t.Error("isHorizontalRule failed for --")
	}
}

// TestConvertImagesComprehensive tests convertImages with table-driven test cases.
func TestConvertImagesComprehensive(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "normal image with alt text",
			input:          "![alt text](image.png)",
			expectedOutput: `#image("image.png")`,
		},
		{
			name:           "image with empty alt text",
			input:          "![](image.png)",
			expectedOutput: `#image("image.png")`,
		},
		{
			name:           "multiple images in one line",
			input:          "![img1](a.png) and ![img2](b.png)",
			expectedOutput: `#image("a.png") and #image("b.png")`,
		},
		{
			name:           "image in text",
			input:          "Here is ![alt](img.png) in text",
			expectedOutput: `Here is #image("img.png") in text`,
		},
		{
			name:           "no image match",
			input:          "This is regular text without images",
			expectedOutput: "This is regular text without images",
		},
		{
			name:           "image with URL containing special chars",
			input:          "![alt](http://example.com/image.png?size=100)",
			expectedOutput: `#image("http://example.com/image.png?size=100")`,
		},
		{
			name:           "image with relative path",
			input:          "![alt](../images/pic.jpg)",
			expectedOutput: `#image("../images/pic.jpg")`,
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.expectedOutput) {
			t.Errorf("%s: expected output to contain %q, got %q", test.name, test.expectedOutput, result)
		}
	}
}

// TestConvertBoldComprehensive tests convertBold with table-driven test cases.
func TestConvertBoldComprehensive(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "bold with double asterisks",
			input:          "This is **bold** text",
			expectedOutput: "This is *bold* text",
		},
		{
			name:           "bold with double underscores",
			input:          "This is __bold__ text",
			expectedOutput: "This is *bold* text",
		},
		{
			name:           "multiple bold sections with asterisks",
			input:          "**first** and **second**",
			expectedOutput: "*first* and *second*",
		},
		{
			name:           "multiple bold sections with mixed markers",
			input:          "**first** and __second__",
			expectedOutput: "*first* and *second*",
		},
		{
			name:           "bold with content containing spaces",
			input:          "**bold text here**",
			expectedOutput: "*bold text here*",
		},
		{
			name:           "no match - no bold markers",
			input:          "This is regular text",
			expectedOutput: "This is regular text",
		},
		{
			name:           "bold at start of text",
			input:          "**Start** of line",
			expectedOutput: "*Start* of line",
		},
		{
			name:           "bold at end of text",
			input:          "End of **line**",
			expectedOutput: "End of *line*",
		},
		{
			name:           "bold with numbers and punctuation",
			input:          "**123** and **test!**",
			expectedOutput: "*123* and *test!*",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)
		if !strings.Contains(result, test.expectedOutput) {
			t.Errorf("%s: expected output to contain %q, got %q", test.name, test.expectedOutput, result)
		}
	}
}

// TestReplaceLinksSingle tests replaceLinks with single markdown link conversion.
func TestReplaceLinksSingle(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "simple markdown link",
			input:          "[text](url)",
			expectedOutput: `#link("url")[text]`,
		},
		{
			name:           "link in sentence",
			input:          "Click [here](https://example.com) now",
			expectedOutput: `Click #link("https://example.com")[here] now`,
		},
		{
			name:           "link with empty text",
			input:          "[](https://example.com)",
			expectedOutput: `#link("https://example.com")[]`,
		},
		{
			name:           "no links in text",
			input:          "This is plain text without any links",
			expectedOutput: "This is plain text without any links",
		},
		{
			name:           "multiple links in one line",
			input:          "[first](url1) and [second](url2)",
			expectedOutput: `#link("url1")[first]` + " and " + `#link("url2")[second]`,
		},
		{
			name:           "link with special chars in URL",
			input:          "[link](https://example.com/path?param=value&other=123)",
			expectedOutput: `#link("https://example.com/path?param=value&other=123")[link]`,
		},
		{
			name:           "link at start of line",
			input:          "[start](url) of line",
			expectedOutput: `#link("url")[start] of line`,
		},
		{
			name:           "link at end of line",
			input:          "end of [line](url)",
			expectedOutput: `end of #link("url")[line]`,
		},
		{
			name:           "link with punctuation in text",
			input:          "[click here!](url)",
			expectedOutput: `#link("url")[click here!]`,
		},
		{
			name:           "link with special chars in text",
			input:          "[foo & bar](url)",
			expectedOutput: `#link("url")[foo & bar]`,
		},
		{
			name:           "image is not converted as link",
			input:          "![alt](img.png)",
			expectedOutput: "![alt](img.png)",
		},
	}

	for _, test := range tests {
		result := converter.replaceLinks(test.input)
		if !strings.Contains(result, test.expectedOutput) {
			t.Errorf("%s: expected output to contain %q, got %q", test.name, test.expectedOutput, result)
		}
	}
}

// TestReplaceLinksComprehensive tests replaceLinks with comprehensive table-driven test cases.
func TestReplaceLinksComprehensive(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "markdown link single",
			input:    "[text](url)",
			expected: `#link("url")[text]`,
		},
		{
			name:     "no links",
			input:    "This is plain text",
			expected: "This is plain text",
		},
		{
			name:     "multiple links on one line",
			input:    "[first](url1) and [second](url2)",
			expected: `#link("url1")[first] and #link("url2")[second]`,
		},
		{
			name:     "link with special chars",
			input:    "[text](http://example.com/path?a=1&b=2#anchor)",
			expected: `#link("http://example.com/path?a=1&b=2#anchor")[text]`,
		},
		{
			name:     "empty link text",
			input:    "[](url)",
			expected: `#link("url")[]`,
		},
		{
			name:     "link at line start",
			input:    "[link](http://example.com) is here",
			expected: `#link("http://example.com")[link] is here`,
		},
		{
			name:     "link at line end",
			input:    "Visit [example](http://example.com)",
			expected: `Visit #link("http://example.com")[example]`,
		},
		{
			name:     "multiple links with mixed content",
			input:    "See [docs](docs.html) and [FAQ](faq.html) for help",
			expected: `See #link("docs.html")[docs] and #link("faq.html")[FAQ] for help`,
		},
		{
			name:     "unclosed bracket should not convert",
			input:    "[incomplete text and more",
			expected: "[incomplete text and more",
		},
		{
			name:     "bracket without paren should not convert",
			input:    "[text] without paren",
			expected: "[text] without paren",
		},
		{
			name:     "image should skip conversion",
			input:    "![alt](image.png) is not a link",
			expected: "![alt](image.png) is not a link",
		},
		{
			name:     "link with numbers",
			input:    "[123](url)",
			expected: `#link("url")[123]`,
		},
		{
			name:     "link with emoji-like punctuation",
			input:    "[hello!?](url)",
			expected: `#link("url")[hello!?]`,
		},
		{
			name:     "link with relative path",
			input:    "[relative](../path/to/file.html)",
			expected: `#link("../path/to/file.html")[relative]`,
		},
		{
			name:     "link text with multiple spaces",
			input:    "[some longer text](url)",
			expected: `#link("url")[some longer text]`,
		},
	}

	for _, test := range tests {
		result := converter.replaceLinks(test.input)
		if result != test.expected {
			t.Errorf("%s: expected %q, got %q", test.name, test.expected, result)
		}
	}
}

// TestConvertUnclosedCodeBlock tests that unclosed code blocks at EOF trigger warning and content is preserved where possible.
func TestConvertUnclosedCodeBlock(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name           string
		input          string
		shouldContain  string
		shouldNotHave  string
	}{
		{
			name:          "properly closed code block",
			input:         "```go\nfunc main() {}\n```",
			shouldContain: "```go",
		},
		{
			name:          "unclosed code block at EOF",
			input:         "```go\nfunc main() {}\n",
			shouldContain: "func main() {}",
		},
		{
			name:          "empty code block",
			input:         "```\n```",
			shouldContain: "```",
		},
		{
			name:          "code block with language specifier",
			input:         "```python\nprint('hello')\n```",
			shouldContain: "```python",
		},
		{
			name:          "unclosed empty code block",
			input:         "```",
			shouldContain: "",
		},
		{
			name:          "code block with multiline content",
			input:         "```js\nvar x = 1;\nvar y = 2;\n```",
			shouldContain: "var x = 1;",
		},
		{
			name:          "text before unclosed code block",
			input:         "Some text\n```\ncode",
			shouldContain: "Some text",
		},
		{
			name:          "unclosed code block with language",
			input:         "```rust\nfn main() {}\n",
			shouldContain: "fn main() {}",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)

		if test.shouldContain != "" && !strings.Contains(result, test.shouldContain) {
			t.Errorf("%s: expected result to contain %q, got %q", test.name, test.shouldContain, result)
		}

		if test.shouldNotHave != "" && strings.Contains(result, test.shouldNotHave) {
			t.Errorf("%s: expected result to NOT contain %q, got %q", test.name, test.shouldNotHave, result)
		}
	}
}

// TestConvertCodeBlocksComprehensive tests code block conversion with table-driven approach.
func TestConvertCodeBlocksComprehensive(t *testing.T) {
	converter := &MarkdownToTypstConverter{}

	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "properly closed go code block",
			input:          "```go\nfunc main() {}\n```",
			expectedOutput: "```go\nfunc main() {}\n```",
		},
		{
			name:           "code block with no language",
			input:          "```\nplain code\n```",
			expectedOutput: "```\nplain code\n```",
		},
		{
			name:           "python code block",
			input:          "```python\nprint('hello')\n```",
			expectedOutput: "```python\nprint('hello')\n```",
		},
		{
			name:           "java code block with multiple lines",
			input:          "```java\npublic static void main() {\n  System.out.println(\"test\");\n}\n```",
			expectedOutput: "System.out.println",
		},
		{
			name:           "code block followed by text",
			input:          "```\ncode\n```\nMore text after",
			expectedOutput: "More text after",
		},
		{
			name:           "multiple code blocks",
			input:          "```js\nvar a = 1;\n```\nSome text\n```py\nprint(a)\n```",
			expectedOutput: "var a = 1;",
		},
		{
			name:           "code block with special characters",
			input:          "```\n<tag>content</tag>\n```",
			expectedOutput: "<tag>content</tag>",
		},
		{
			name:           "code block with empty lines",
			input:          "```\nfirst line\n\nthird line\n```",
			expectedOutput: "first line",
		},
	}

	for _, test := range tests {
		result := converter.Convert(test.input)

		if !strings.Contains(result, test.expectedOutput) {
			t.Errorf("%s: expected output to contain %q, got %q", test.name, test.expectedOutput, result)
		}
	}
}

// TestCheckTypstAvailable tests the Typst availability check
func TestCheckTypstAvailable(t *testing.T) {
	// This test verifies the function does not panic and returns appropriate result
	// In environments where typst is not installed, it should return an error
	err := CheckTypstAvailable()
	if err != nil {
		// It's acceptable for typst to not be installed in test environment
		// Just verify the error message is informative
		if !strings.Contains(err.Error(), "Typst") && !strings.Contains(err.Error(), "typst") {
			t.Errorf("error should mention Typst, got: %v", err)
		}
	}
	// If no error, typst is available and working
}

// TestGeneratorWithMultipleOptions tests Generator creation with various options
func TestGeneratorWithMultipleOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []GeneratorOption
		checkFn  func(*Generator) bool
		errMsg   string
	}{
		{
			name: "all options set",
			opts: []GeneratorOption{
				WithTitle("Test Book"),
				WithAuthor("Test Author"),
				WithVersion("1.0.0"),
				WithPageSize("Letter"),
				WithFontSize("14pt"),
				WithLineHeight(1.8),
				WithLanguage("zh"),
			},
			checkFn: func(g *Generator) bool {
				return g.title == "Test Book" &&
					g.author == "Test Author" &&
					g.version == "1.0.0" &&
					g.pageSize == "Letter" &&
					g.fontSize == "14pt" &&
					g.lineHeight == 1.8 &&
					g.language == "zh"
			},
			errMsg: "Generator with all options failed",
		},
		{
			name: "default timeout set",
			opts: []GeneratorOption{
				WithTimeout(30 * time.Second),
			},
			checkFn: func(g *Generator) bool {
				return g.timeout == 30*time.Second
			},
			errMsg: "timeout option failed",
		},
		{
			name: "margins set",
			opts: []GeneratorOption{
				WithMargins("10mm", "15mm", "20mm", "25mm"),
			},
			checkFn: func(g *Generator) bool {
				return g.marginLeft == "10mm" &&
					g.marginRight == "15mm" &&
					g.marginTop == "20mm" &&
					g.marginBottom == "25mm"
			},
			errMsg: "margins option failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.opts...)
			if !tt.checkFn(gen) {
				t.Error(tt.errMsg)
			}
		})
	}
}

// TestGenerateValidation tests Generate function parameter validation
func TestGenerateValidation(t *testing.T) {
	gen := NewGenerator()

	tests := []struct {
		name    string
		content string
		output  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty output path",
			content: "# Test",
			output:  "",
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "empty content",
			content: "",
			output:  "/tmp/test.pdf",
			wantErr: true,
			errMsg:  "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.Generate(tt.content, tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error should contain %q, got: %v", tt.errMsg, err)
			}
		})
	}
}
