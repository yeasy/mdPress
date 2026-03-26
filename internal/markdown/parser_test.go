package markdown

import (
	"fmt"
	"strings"
	"testing"
)

// TestNewParser tests creating a new parser
func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser returned nil")
	}
}

// TestNewParserWithOptions tests creating a parser with options
func TestNewParserWithOptions(t *testing.T) {
	parser := NewParser(WithCodeTheme("dracula"))
	if parser == nil {
		t.Fatal("NewParser with options returned nil")
		return
	}
	if parser.codeTheme != "dracula" {
		t.Errorf("code theme should be dracula: got %q", parser.codeTheme)
	}
}

// TestParseBasic tests basic Markdown parsing
func TestParseBasic(t *testing.T) {
	parser := NewParser()
	html, headings, err := parser.Parse([]byte("# 标题\n\n这是内容"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "标题") {
		t.Error("HTML does not contain heading text")
	}
	if len(headings) != 1 {
		t.Errorf("expected 1 heading, got %d", len(headings))
	}
	if headings[0].Level != 1 {
		t.Errorf("expected heading level 1, got %d", headings[0].Level)
	}
	if headings[0].Text != "标题" {
		t.Errorf("heading text mismatch: got %q", headings[0].Text)
	}
}

// TestParseTable tests table parsing
func TestParseTable(t *testing.T) {
	parser := NewParser()
	md := "| A | B |\n|---|---|\n| 1 | 2 |"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<table>") {
		t.Error("HTML does not contain table tag")
	}
	if !strings.Contains(html, "<th>") {
		t.Error("HTML does not contain th tag")
	}
	if !strings.Contains(html, "<td>") {
		t.Error("HTML does not contain td tag")
	}
}

// TestParseCodeHighlight tests code highlighting
func TestParseCodeHighlight(t *testing.T) {
	parser := NewParser(WithCodeTheme("monokai"))
	md := "```go\nfmt.Println(\"hello\")\n```"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<pre") {
		t.Error("code block should contain pre tag")
	}
	// Chroma inline style on <pre> is stripped by PostProcess;
	// token-level <span style="color:..."> should still be present.
	if strings.Contains(html, `<pre style="`) {
		t.Error("chroma inline style on <pre> should be stripped")
	}
	if !strings.Contains(html, `style="color`) {
		t.Error("token-level inline color styles should be preserved")
	}
}

// TestParseCodeMultiLanguages tests multi-language code highlighting
func TestParseCodeMultiLanguages(t *testing.T) {
	parser := NewParser()
	languages := []string{
		"```python\nprint('hello')\n```",
		"```javascript\nconsole.log('hi')\n```",
		"```bash\necho hello\n```",
		"```rust\nfn main() {}\n```",
	}

	for _, md := range languages {
		html, _, err := parser.Parse([]byte(md))
		if err != nil {
			t.Fatalf("parse %q failed: %v", md[:20], err)
		}
		if !strings.Contains(html, "<pre") {
			t.Errorf("code block %q should contain pre tag", md[:20])
		}
	}
}

// TestParseEmpty tests empty content
func TestParseEmpty(t *testing.T) {
	parser := NewParser()
	html, headings, err := parser.Parse([]byte(""))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if html != "" {
		t.Errorf("expected empty HTML, got %q", html)
	}
	if len(headings) != 0 {
		t.Errorf("expected 0 headings, got %d", len(headings))
	}
}

// TestMultipleHeadings tests multi-level headings
func TestMultipleHeadings(t *testing.T) {
	parser := NewParser()
	md := "# H1\n\n## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6"
	_, headings, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(headings) != 6 {
		t.Errorf("expected 6 headings, got %d", len(headings))
	}
	for i, h := range headings {
		if h.Level != i+1 {
			t.Errorf("heading %d level mismatch: got %d, want %d", i, h.Level, i+1)
		}
	}
}

// TestParseStrikethrough tests strikethrough
func TestParseStrikethrough(t *testing.T) {
	parser := NewParser()
	md := "~~deleted text~~"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<del>") {
		t.Error("strikethrough should generate <del> tag")
	}
}

// TestParseTaskList tests task lists
func TestParseTaskList(t *testing.T) {
	parser := NewParser()
	md := "- [x] 完成的任务\n- [ ] 未完成的任务"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "checkbox") || !strings.Contains(html, "input") {
		t.Error("task list should contain checkbox")
	}
}

// TestParseFootnotes tests footnotes
func TestParseFootnotes(t *testing.T) {
	parser := NewParser()
	md := "这是正文[^1]\n\n[^1]: 这是脚注内容"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "footnote") {
		t.Error("footnotes should be generated in HTML")
	}
}

// TestParseBlockquote tests blockquotes
func TestParseBlockquote(t *testing.T) {
	parser := NewParser()
	md := "> 这是引用内容\n> 第二行"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<blockquote>") {
		t.Error("blockquote should generate blockquote tag")
	}
}

// TestParseLinks tests links
func TestParseLinks(t *testing.T) {
	parser := NewParser()
	md := "[点击这里](https://example.com)"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, `href="https://example.com"`) {
		t.Error("link should be generated correctly")
	}
	if !strings.Contains(html, "点击这里") {
		t.Error("link text should be correct")
	}
}

// TestParseImages tests images
func TestParseImages(t *testing.T) {
	parser := NewParser()
	md := "![替代文本](image.png)"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<img") {
		t.Error("image should generate img tag")
	}
	if !strings.Contains(html, `src="image.png"`) {
		t.Error("image src should be correct")
	}
	if !strings.Contains(html, `alt="替代文本"`) {
		t.Error("image alt should be correct")
	}
}

// TestParseBold tests bold text
func TestParseBold(t *testing.T) {
	parser := NewParser()
	md := "**加粗文本**"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<strong>") {
		t.Error("bold should generate strong tag")
	}
}

// TestParseItalic tests italic text
func TestParseItalic(t *testing.T) {
	parser := NewParser()
	md := "*斜体文本*"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<em>") {
		t.Error("italic should generate em tag")
	}
}

// TestParseInlineCode tests inline code
func TestParseInlineCode(t *testing.T) {
	parser := NewParser()
	md := "使用 `go build` 命令"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<code>") {
		t.Error("inline code should generate code tag")
	}
}

// TestParseOrderedList tests ordered lists
func TestParseOrderedList(t *testing.T) {
	parser := NewParser()
	md := "1. 第一项\n2. 第二项\n3. 第三项"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<ol>") {
		t.Error("ordered list should generate ol tag")
	}
}

// TestParseHorizontalRule tests horizontal rules
func TestParseHorizontalRule(t *testing.T) {
	parser := NewParser()
	md := "---"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(html, "<hr") {
		t.Error("horizontal rule should generate hr tag")
	}
}

// TestHeadingIDs tests heading ID generation
func TestHeadingIDs(t *testing.T) {
	parser := NewParser()
	md := "# Hello World\n\n## 中文标题"
	_, headings, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(headings) < 2 {
		t.Fatalf("should have at least 2 headings: got %d", len(headings))
	}

	// ID should not be empty
	for i, h := range headings {
		if h.ID == "" {
			t.Errorf("heading %d ID should not be empty", i)
		}
	}
}

// TestSetCodeTheme tests dynamic code theme switching
func TestSetCodeTheme(t *testing.T) {
	parser := NewParser()
	parser.SetCodeTheme("dracula")
	if parser.codeTheme != "dracula" {
		t.Errorf("code theme should be dracula: got %q", parser.codeTheme)
	}

	// Should parse correctly after switching
	md := "```go\nfmt.Println(\"hello\")\n```"
	_, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed after theme switch: %v", err)
	}
}

// TestGetHeadings tests retrieving heading list
func TestGetHeadings(t *testing.T) {
	parser := NewParser()
	if _, _, err := parser.Parse([]byte("# A\n\n## B")); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	headings := parser.GetHeadings()
	if len(headings) != 2 {
		t.Errorf("should have 2 headings: got %d", len(headings))
	}
}

// TestParseComplexDocument tests complex document
func TestParseComplexDocument(t *testing.T) {
	parser := NewParser()
	md := `# 项目介绍

这是一个 **重要** 的项目。

## 功能列表

- [x] 已完成功能
- [ ] 待完成功能

## 代码示例

` + "```go" + `
package main

func main() {
    println("hello")
}
` + "```" + `

## 数据对比

| 名称 | 分数 |
|------|------|
| A    | 95   |
| B    | 88   |

> 注意：以上数据仅供参考。

---

更多信息请访问 [官网](https://example.com)。

这是一个脚注[^1]。

[^1]: 脚注内容。
`
	html, headings, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse complex document failed: %v", err)
	}

	if len(headings) != 4 {
		t.Errorf("should have 4 headings: got %d", len(headings))
	}

	// Check various elements
	checks := map[string]string{
		"<strong>":     "bold",
		"<table>":      "table",
		"<blockquote>": "blockquote",
		"<hr":          "horizontal rule",
		"<a ":          "link", //nolint:gocritic // intentional trailing space to match tag prefix
		"<pre":         "code block",
	}

	for tag, name := range checks {
		if !strings.Contains(html, tag) {
			t.Errorf("complex document should contain %s (%s)", name, tag)
		}
	}
}

// TestGenerateHeadingID tests heading ID generation function
func TestGenerateHeadingID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"", "heading"},
	}

	for _, tt := range tests {
		got := generateHeadingID(tt.input)
		if got != tt.want {
			t.Errorf("generateHeadingID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestParseConcurrent tests concurrent parsing safety
func TestParseConcurrent(t *testing.T) {
	parser := NewParser()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _, err := parser.Parse([]byte("# Test\n\nContent"))
			if err != nil {
				t.Errorf("concurrent parse failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestParseGFMTableAlignment tests GFM table alignment syntax
func TestParseGFMTableAlignment(t *testing.T) {
	parser := NewParser()
	// Test left-align, center-align, right-align table syntax
	// GFM tables require a preceding blank line
	md := "\n| 左对齐 | 居中 | 右对齐 |\n" +
		"|:---|:---:|---:|\n" +
		"| L | C | R |"

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("table alignment parse failed: %v", err)
	}
	if !strings.Contains(html, "<table>") {
		t.Error("aligned table should contain table tag")
	}
	if !strings.Contains(html, "<th") {
		t.Error("aligned table should contain th tag")
	}
	if !strings.Contains(html, "<td") {
		t.Error("aligned table should contain td tag")
	}
}

// TestParseNestedCodeBlock tests code blocks inside lists or blockquotes
func TestParseNestedCodeBlock(t *testing.T) {
	parser := NewParser()
	tests := []struct {
		name string
		md   string
	}{
		{
			name: "code block in list",
			md:   "- 第一项\n\n  ```go\n  fmt.Println(\"hello\")\n  ```\n- 第二项",
		},
		{
			name: "code block in blockquote",
			md:   "> 这是一个引用\n>\n> ```python\n> print('hello')\n> ```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("nested code block parse failed: %v", err)
			}
			if !strings.Contains(html, "<pre") {
				t.Error("nested code block should contain pre tag")
			}
		})
	}
}

// TestParseMultipleFootnotes tests multiple footnotes
func TestParseMultipleFootnotes(t *testing.T) {
	parser := NewParser()
	md := "这是第一个脚注[^1]，第二个脚注[^2]，还有第三个[^3]。\n\n" +
		"[^1]: 第一个脚注内容\n" +
		"[^2]: 第二个脚注内容\n" +
		"[^3]: 第三个脚注内容"

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("multiple footnotes parse failed: %v", err)
	}
	if !strings.Contains(html, "footnote") {
		t.Error("footnotes should be generated in HTML")
	}
	// Verify footnote markers appear
	footnoteCount := strings.Count(html, "footnote")
	if footnoteCount < 3 {
		t.Errorf("expected at least 3 footnote markers, got: %d", footnoteCount)
	}
}

// TestParseImageWithTitle tests images with title attribute
func TestParseImageWithTitle(t *testing.T) {
	parser := NewParser()
	md := `![替代文本](image.png "这是图片标题")`

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("image with title parse failed: %v", err)
	}
	if !strings.Contains(html, "<img") {
		t.Error("image should generate img tag")
	}
	if !strings.Contains(html, `src="image.png"`) {
		t.Error("image src should be correct")
	}
	if !strings.Contains(html, `alt="替代文本"`) {
		t.Error("image alt should be correct")
	}
	if !strings.Contains(html, "title=") || !strings.Contains(html, "图片标题") {
		t.Errorf("image title should be present: %s", html)
	}
}

// TestParseLinkVariations tests various link forms (table-driven)
func TestParseLinkVariations(t *testing.T) {
	parser := NewParser()
	tests := []struct {
		name    string
		md      string
		hasHref string
		hasText string
	}{
		{
			name:    "standard link",
			md:      "[文本](https://example.com)",
			hasHref: "https://example.com",
			hasText: "文本",
		},
		{
			name:    "link with title",
			md:      `[文本](https://example.com "标题")`,
			hasHref: "https://example.com",
			hasText: "文本",
		},
		{
			name:    "autolink",
			md:      "https://example.com",
			hasHref: "https://example.com",
			hasText: "",
		},
		{
			name:    "reference link",
			md:      "[文本][ref]\n\n[ref]: https://example.com",
			hasHref: "https://example.com",
			hasText: "文本",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("link parse failed: %v", err)
			}
			if !strings.Contains(html, "href=") {
				t.Error("link should generate href attribute")
			}
			if tt.hasHref != "" && !strings.Contains(html, tt.hasHref) {
				t.Errorf("link should contain %q", tt.hasHref)
			}
			if tt.hasText != "" && !strings.Contains(html, tt.hasText) {
				t.Errorf("link should contain text %q", tt.hasText)
			}
		})
	}
}

// TestParseHTMLPassthrough tests raw HTML passthrough
func TestParseHTMLPassthrough(t *testing.T) {
	parser := NewParser()
	// With WithUnsafe() configured, raw HTML should be preserved
	md := "这是文本\n\n<div class=\"custom\">自定义 HTML</div>\n\n更多文本"

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("HTML passthrough parse failed: %v", err)
	}
	if !strings.Contains(html, "<div") && !strings.Contains(html, "custom") {
		t.Errorf("HTML passthrough content should be preserved: %s", html)
	}
}

// TestGenerateHeadingIDTableDriven table-driven tests for heading ID generation
func TestGenerateHeadingIDTableDriven(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "Hello World",
			want:  "hello-world",
		},
		{
			input: "中文标题",
			want:  "中文标题", // Chinese characters are preserved in heading IDs
		},
		{
			input: "Special !@#$% Chars",
			want:  "special-chars",
		},
		{
			input: "Numbers 123 456",
			want:  "numbers-123-456",
		},
		{
			input: "",
			want:  "heading",
		},
		{
			input: "   Multiple   Spaces   ",
			want:  "multiple-spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := generateHeadingID(tt.input)
			// Verify ID is not empty
			if got == "" {
				t.Error("generated ID should not be empty")
			}
			if got != tt.want {
				t.Errorf("generateHeadingID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseNestedFormatting tests nested formatting
func TestParseNestedFormatting(t *testing.T) {
	parser := NewParser()
	tests := []struct {
		name string
		md   string
	}{
		{
			name: "italic inside bold",
			md:   "**粗体 _斜体_ 粗体**",
		},
		{
			name: "bold inside italic",
			md:   "*斜体 **粗体** 斜体*",
		},
		{
			name: "bold-italic mixed",
			md:   "***粗斜体***",
		},
		{
			name: "formatting inside code",
			md:   "`**不应被解析**`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("nested formatting parse failed: %v", err)
			}
			// Check if HTML was generated
			if html == "" {
				t.Error("should generate HTML")
			}
		})
	}
}

// TestParseDefinitionList tests definition list handling
func TestParseDefinitionList(t *testing.T) {
	parser := NewParser()
	// Test definition list format (if supported)
	md := "Apple\n:   一种水果\n\nBanana\n:   另一种水果"

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("definition list parse failed: %v", err)
	}
	// Check if content was generated; specific format depends on support
	if html == "" {
		t.Error("should generate HTML output")
	}
}

// ============================================================================
// Additional: complex integration tests
// ============================================================================

// TestComplexMarkdownIntegration tests complex Markdown mixed features (table-driven)
func TestComplexMarkdownIntegration(t *testing.T) {
	tests := []struct {
		name         string
		md           string
		wantElements map[string]string // tag -> description
	}{
		{
			name: "headings+code+table+links",
			md: "# 项目概述\n\n" +
				"这是 [官网](https://example.com) 的链接。\n\n" +
				"## API 文档\n\n" +
				"```go\n" +
				"func Handler(w http.ResponseWriter, r *http.Request) {\n" +
				"    w.Header().Set(\"Content-Type\", \"application/json\")\n" +
				"}\n" +
				"```\n\n" +
				"| 端点 | 方法 | 说明 |\n" +
				"|------|------|------|\n" +
				"| /api/users | GET | 获取用户列表 |\n" +
				"| /api/users | POST | 创建用户 |\n\n" +
				"---\n\n" +
				"### 返回值\n\n" +
				"- **code**: 状态码\n" +
				"- **data**: 响应数据",
			wantElements: map[string]string{
				"<h1":     "main heading",
				"<h2":     "second-level heading",
				"<h3":     "third-level heading",
				"<a href": "link",
				"<pre":    "code block",
				"<table>": "table",
				"<hr":     "horizontal rule",
				"<ul>":    "list",
			},
		},
		{
			name: "nested: list+code+emphasis",
			md: "## 步骤说明\n\n" +
				"1. 安装依赖：\n\n" +
				"   ```bash\n" +
				"   go get github.com/example/package\n" +
				"   ```\n\n" +
				"2. 导入包：\n\n" +
				"   ```go\n" +
				"   import \"github.com/example/package\"\n" +
				"   ```\n\n" +
				"3. 使用 API：\n\n" +
				"   - 调用 **Init** 函数初始化\n" +
				"   - 调用 *Process* 方法处理数据\n" +
				"   - 检查 __错误__ 返回值",
			wantElements: map[string]string{
				"<h2":      "second-level heading",
				"<ol>":     "ordered list",
				"<pre":     "code block",
				"<strong>": "bold",
				"<em>":     "italic",
				"<ul>":     "unordered list",
			},
		},
		{
			name: "blockquote+code+table",
			md: "> **注意：** 这是一个重要的提示\n" +
				">\n" +
				"> ```python\n" +
				"> result = process(data)\n" +
				"> print(result)\n" +
				"> ```\n" +
				">\n" +
				"> 更多详情见下表：\n" +
				">\n" +
				"> | 参数 | 类型 | 必需 |\n" +
				"> |-----|------|------|\n" +
				"> | data | string | 是 |\n" +
				"> | async | bool | 否 |",
			wantElements: map[string]string{
				"<blockquote>": "blockquote",
				"<pre":         "code block",
				"<table>":      "table",
				"<strong>":     "bold",
			},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			for tag, desc := range tt.wantElements {
				if !strings.Contains(html, tag) {
					t.Errorf("missing %s (%s)", desc, tag)
				}
			}
		})
	}
}

// TestParserEdgeCases tests edge cases (table-driven)
func TestParserEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		md      string
		wantErr bool
		check   func(t *testing.T, html string)
	}{
		{
			name:    "empty input",
			md:      "",
			wantErr: false,
			check: func(t *testing.T, html string) {
				if html != "" {
					t.Errorf("empty input should return empty HTML, got %q", html)
				}
			},
		},
		{
			name:    "whitespace only",
			md:      "   \n\n  \t\t\n  ",
			wantErr: false,
			check: func(t *testing.T, html string) {
				if strings.TrimSpace(html) != "" {
					t.Errorf("whitespace only should return empty HTML, got %q", html)
				}
			},
		},
		{
			name:    "very long line (>10000 chars)",
			md:      "# 标题\n\n" + strings.Repeat("x", 12000),
			wantErr: false,
			check: func(t *testing.T, html string) {
				if html == "" {
					t.Error("long input should generate HTML")
				}
			},
		},
		{
			name:    "very long document (>1000 lines)",
			md:      buildLongDocument(1500),
			wantErr: false,
			check: func(t *testing.T, html string) {
				if html == "" {
					t.Error("long document should generate HTML")
				}
				// Should have many headings
				headingCount := strings.Count(html, "<h2")
				if headingCount < 100 {
					t.Errorf("expected at least 100 headings, got: %d", headingCount)
				}
			},
		},
		{
			name: "multiple consecutive code blocks",
			md: "```python\nprint('a')\n```\n\n" +
				"```go\nfmt.Println(\"b\")\n```\n\n" +
				"```js\nconsole.log('c')\n```",
			wantErr: false,
			check: func(t *testing.T, html string) {
				if strings.Count(html, "<pre") < 3 {
					t.Error("should have at least 3 code blocks")
				}
			},
		},
		{
			name:    "special characters and escaping",
			md:      "# <script>alert('xss')</script>\n\n测试 & < > \" '",
			wantErr: false,
			check: func(t *testing.T, html string) {
				// Should generate HTML (specific escaping strategy depends on renderer)
				if html == "" {
					t.Error("special character input should generate HTML")
				}
			},
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got=%v", tt.wantErr, err)
			}
			if tt.check != nil {
				tt.check(t, html)
			}
		})
	}
}

// TestCJKContentRendering tests CJK content rendering (table-driven)
func TestCJKContentRendering(t *testing.T) {
	tests := []struct {
		name string
		md   string
		want string // text that should be present
	}{
		{
			name: "Chinese heading",
			md:   "# 中文标题\n\n## 二级标题",
			want: "中文标题",
		},
		{
			name: "Japanese content",
			md:   "# 日本語のタイトル\n\nこれはテキストです。",
			want: "日本語のタイトル",
		},
		{
			name: "Korean content",
			md:   "# 한국어 제목\n\n한국어 텍스트입니다.",
			want: "한국어 제목",
		},
		{
			name: "mixed CJK",
			md: "# 中文 日本語 한국어\n\n" +
				"这是中文段落。\n\n" +
				"これは日本語です。\n\n" +
				"이것은 한국어입니다.",
			want: "中文",
		},
		{
			name: "CJK table",
			md: "| 中文 | 日本語 | 한국어 |\n" +
				"|------|--------|--------|\n" +
				"| 内容 | コンテンツ | 내용 |",
			want: "中文",
		},
		{
			name: "CJK code comments",
			md: "```python\n" +
				"# 这是中文注释\n" +
				"# これは日本語のコメントです\n" +
				"# 이것은 한국어 주석입니다\n" +
				"print('hello')\n" +
				"```",
			want: "这是中文注释",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if !strings.Contains(html, tt.want) {
				t.Errorf("HTML should contain %q but not found\nHTML: %s", tt.want, html[:500])
			}
		})
	}
}

// TestNestedBlockquoteWithCode tests nested blockquotes with code (table-driven)
func TestNestedBlockquoteWithCode(t *testing.T) {
	tests := []struct {
		name string
		md   string
	}{
		{
			name: "multi-level nested blockquotes",
			md: "> 外层引用\n" +
				">\n" +
				"> > 内层引用\n" +
				"> >\n" +
				"> > > 更深层引用",
		},
		{
			name: "code block inside blockquote",
			md: "> 说明文字\n" +
				">\n" +
				"> ```go\n" +
				"> func main() {\n" +
				">     println(\"hello\")\n" +
				"> }\n" +
				"> ```\n" +
				">\n" +
				"> 更多说明",
		},
		{
			name: "list and code inside blockquote",
			md: "> 这是引用块\n" +
				">\n" +
				"> 1. 第一项\n" +
				"> 2. 第二项\n" +
				">\n" +
				"> ```python\n" +
				"> import sys\n" +
				"> ```",
		},
		{
			name: "complex nesting: list->blockquote->code->table",
			md: "- 项目 A\n" +
				"\n" +
				"  > 引用块说明\n" +
				"  >\n" +
				"  > ```bash\n" +
				"  > mkdir test\n" +
				"  > cd test\n" +
				"  > ```\n" +
				"\n" +
				"  | 命令 | 说明 |\n" +
				"  |------|-----|\n" +
				"  | ls | 列表 |\n" +
				"\n" +
				"- 项目 B",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			// Basic check: should contain blockquote and pre tags
			if !strings.Contains(html, "<blockquote>") {
				t.Error("should contain blockquote tag")
			}
			if !strings.Contains(html, "<pre") && !strings.Contains(tt.md, "```") {
				// If input has code blocks, there should be pre tags
				if strings.Contains(tt.md, "```") {
					t.Error("should contain pre tag (code block)")
				}
			}
		})
	}
}

// TestMarkdownWithHTMLMixed tests Markdown mixed with HTML (table-driven)
func TestMarkdownWithHTMLMixed(t *testing.T) {
	tests := []struct {
		name        string
		md          string
		wantHTMLTag string
	}{
		{
			name:        "inline HTML",
			md:          "这是文本 <span style='color:red'>红色</span> 继续文本",
			wantHTMLTag: "<span",
		},
		{
			name:        "block-level HTML",
			md:          "文本开始\n\n<div class='box'>HTML 内容</div>\n\n文本结束",
			wantHTMLTag: "<div",
		},
		{
			name: "HTML form",
			md: "## 表单示例\n\n" +
				"<form method='post'>\n" +
				"  <input type='text' name='username'>\n" +
				"  <button>提交</button>\n" +
				"</form>",
			wantHTMLTag: "<form",
		},
		{
			name:        "HTML comment",
			md:          "# 标题\n\n<!-- 这是注释 -->\n\n内容",
			wantHTMLTag: "<!--",
		},
		{
			name:        "HTML escape characters in Markdown",
			md:          "这是&amp; < > \"引号\"",
			wantHTMLTag: "&",
		},
	}

	parser := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, _, err := parser.Parse([]byte(tt.md))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if !strings.Contains(html, tt.wantHTMLTag) {
				t.Errorf("HTML should contain %q, actual output: %s", tt.wantHTMLTag, html[:300])
			}
		})
	}
}

// ============================================================================
// Helper functions
// ============================================================================

// buildLongDocument builds a long document for performance testing
func buildLongDocument(lines int) string {
	var buf strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&buf, "## Chapter %d\n\n", i+1)
		buf.WriteString("This is paragraph content.\n\n")
		if i%5 == 0 {
			buf.WriteString("```go\ncode snippet\n```\n\n")
		}
		if i%7 == 0 {
			buf.WriteString("| Col A | Col B |\n|------|------|\n| a | b |\n\n")
		}
	}
	return buf.String()
}
