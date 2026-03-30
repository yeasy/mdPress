// Package golden implements golden file tests for mdpress output.
// Golden files capture the expected output of HTML generation and are stored in testdata/golden/.
// Run with -update to regenerate golden files after intentional changes.
package golden

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
)

var update = flag.Bool("update", false, "update golden files instead of comparing")

// testdataDir returns the absolute path to the golden testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("failed to resolve testdata dir: %v", err)
	}
	return dir
}

// goldenDir returns the absolute path to the golden output directory.
func goldenDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "golden"))
	if err != nil {
		t.Fatalf("failed to resolve golden dir: %v", err)
	}
	return dir
}

// datePattern matches ISO dates like 2006-01-02 embedded in HTML output.
var datePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// normalizeOutput replaces volatile fields (dates) with a stable placeholder
// and normalizes line endings to LF for cross-platform consistency.
func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return datePattern.ReplaceAllString(s, "DATE_PLACEHOLDER")
}

// checkGolden compares got against the golden file at path.
// If -update is set the golden file is written instead.
// If the golden file does not yet exist it is created and the test is skipped.
func checkGolden(t *testing.T, path, got string) {
	t.Helper()
	normalized := normalizeOutput(got)

	if *update {
		if err := os.WriteFile(path, []byte(normalized), 0o644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", path, err)
		}
		t.Logf("updated golden file: %s", path)
		return
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// First run: write the golden file and skip so the developer can review.
		if err2 := os.WriteFile(path, []byte(normalized), 0o644); err2 != nil {
			t.Fatalf("failed to create golden file %s: %v", path, err2)
		}
		t.Skipf("golden file created at %s — review and re-run tests", path)
		return
	}
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}

	want := strings.ReplaceAll(string(data), "\r\n", "\n")
	if normalized != want {
		t.Errorf("output does not match golden file %s\n\nwant:\n%s\n\ngot:\n%s",
			path, want, normalized)
	}
}

// loadConfig loads the test book configuration.
func loadConfig(t *testing.T) *config.BookConfig {
	t.Helper()
	cfg, err := config.Load(filepath.Join(testdataDir(t), "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

// parseChapters parses all chapter files and returns their HTML and collected headings.
func parseChapters(t *testing.T, cfg *config.BookConfig) ([]renderer.ChapterHTML, []toc.HeadingInfo) {
	t.Helper()
	parser := markdown.NewParser()
	var chapters []renderer.ChapterHTML
	var allHeadings []toc.HeadingInfo

	flat := config.FlattenChapters(cfg.Chapters)
	for _, ch := range flat {
		path := cfg.ResolvePath(ch.File)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read chapter %s: %v", ch.File, err)
		}
		html, headings, err := parser.Parse(data)
		if err != nil {
			t.Fatalf("failed to parse chapter %s: %v", ch.File, err)
		}
		id := strings.TrimSuffix(filepath.Base(ch.File), ".md")
		chapters = append(chapters, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      id,
			Content: html,
		})
		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{
				Level: h.Level,
				Text:  h.Text,
				ID:    h.ID,
			})
		}
	}
	return chapters, allHeadings
}

// TestGoldenHTML generates the full HTML document and compares it against the golden file.
func TestGoldenHTML(t *testing.T) {
	cfg := loadConfig(t)

	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("failed to load theme %q: %v", cfg.Style.Theme, err)
	}

	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	chapters, allHeadings := parseChapters(t, cfg)

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chapters,
	}

	r, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("failed to create HTML renderer: %v", err)
	}
	got, err := r.Render(parts)
	if err != nil {
		t.Fatalf("failed to render HTML: %v", err)
	}

	checkGolden(t, filepath.Join(goldenDir(t), "full.html"), got)
}

// TestGoldenTOC verifies the table of contents structure against the golden file.
func TestGoldenTOC(t *testing.T) {
	cfg := loadConfig(t)

	_, allHeadings := parseChapters(t, cfg)

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	got := tocGen.RenderHTML(tocEntries)

	// Sanity checks before golden comparison.
	if !strings.Contains(got, "<nav") {
		t.Error("TOC HTML must contain <nav> element")
	}
	if !strings.Contains(got, "<ul") {
		t.Error("TOC HTML must contain <ul> element")
	}
	if !strings.Contains(got, "<a") {
		t.Error("TOC HTML must contain <a> elements")
	}
	// All three chapters should have headings in the TOC.
	for _, title := range []string{"Introduction", "Tables", "中文"} {
		if !strings.Contains(got, title) {
			t.Errorf("TOC HTML missing expected heading text %q", title)
		}
	}

	checkGolden(t, filepath.Join(goldenDir(t), "toc.html"), got)
}

// TestGoldenCover verifies the cover page HTML against the golden file.
func TestGoldenCover(t *testing.T) {
	cfg := loadConfig(t)

	coverGen := cover.NewCoverGenerator(cfg.Book)
	got := coverGen.RenderHTML()

	// Sanity checks before golden comparison.
	if !strings.Contains(got, "<!DOCTYPE html>") {
		t.Error("cover must contain DOCTYPE declaration")
	}
	if !strings.Contains(got, cfg.Book.Title) {
		t.Errorf("cover must contain book title %q", cfg.Book.Title)
	}
	if !strings.Contains(got, cfg.Book.Author) {
		t.Errorf("cover must contain author %q", cfg.Book.Author)
	}
	if !strings.Contains(got, cfg.Book.Version) {
		t.Errorf("cover must contain version %q", cfg.Book.Version)
	}
	if !strings.Contains(got, cfg.Book.Subtitle) {
		t.Errorf("cover must contain subtitle %q", cfg.Book.Subtitle)
	}

	checkGolden(t, filepath.Join(goldenDir(t), "cover.html"), got)
}

// TestGoldenCodeBlocks tests rendering of Markdown with code blocks.
func TestGoldenCodeBlocks(t *testing.T) {
	parser := markdown.NewParser()

	markdownContent := "# Code Block Test\n\n## Go Example\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n## Python Example\n```python\ndef hello():\n    print(\"Hello, World!\")\n\nif __name__ == \"__main__\":\n    hello()\n```\n\n## Inline code\nThis is inline `code` in the text.\n\n## Code with syntax highlighting\n```javascript\nconst greeting = \"Hello, World!\";\nconsole.log(greeting);\n```\n"

	html, headings, err := parser.Parse([]byte(markdownContent))
	if err != nil {
		t.Fatalf("failed to parse markdown with code blocks: %v", err)
	}

	// Sanity checks before golden comparison.
	if !strings.Contains(html, "<pre") {
		t.Error("code blocks should be wrapped in <pre> tags")
	}
	if !strings.Contains(html, "<code") {
		t.Error("code should be wrapped in <code> tags")
	}
	if !strings.Contains(html, "Hello, World!") {
		t.Error("code content should be preserved")
	}
	if !strings.Contains(html, "func") || !strings.Contains(html, "main") {
		t.Error("Go code should be present")
	}

	// Verify headings were extracted
	if len(headings) < 2 {
		t.Errorf("expected at least 2 headings, got %d", len(headings))
	}

	checkGolden(t, filepath.Join(goldenDir(t), "code_blocks.html"), html)
}

// TestGoldenTables tests rendering of Markdown with tables.
func TestGoldenTables(t *testing.T) {
	parser := markdown.NewParser()

	markdownContent := "# Table Test\n\n## Simple Table\n| Name    | Age | City      |\n|---------|-----|-----------|" +
		"\n| Alice   | 30  | New York  |\n| Bob     | 25  | London    |\n| Charlie | 35  | Paris     |" +
		"\n\n## Complex Table\n| Header 1 | Header 2       | Header 3  |\n|----------|----------------|-----------|" +
		"\n| Cell 1.1 | Cell 1.2       | Cell 1.3  |\n| Cell 2.1 | Multi-line     | Cell 2.3  |" +
		"\n|          | Cell 2.2 cont. |           |\n| Cell 3.1 | Cell 3.2       | Cell 3.3  |" +
		"\n\n## Nested Formatting in Table\n| **Bold** | *Italic* | `Code` |\n|----------|----------|---------|" +
		"\n| Text     | More     | More    |\n"

	html, headings, err := parser.Parse([]byte(markdownContent))
	if err != nil {
		t.Fatalf("failed to parse markdown with tables: %v", err)
	}

	// Sanity checks before golden comparison.
	if !strings.Contains(html, "<table") {
		t.Error("tables should be rendered with <table> tags")
	}
	if !strings.Contains(html, "<thead") {
		t.Error("tables should have <thead> for headers")
	}
	if !strings.Contains(html, "<tbody") {
		t.Error("tables should have <tbody> for content")
	}
	if !strings.Contains(html, "<tr") {
		t.Error("tables should have rows")
	}
	if !strings.Contains(html, "<td") {
		t.Error("tables should have data cells")
	}

	// Verify table content
	if !strings.Contains(html, "Alice") || !strings.Contains(html, "New York") {
		t.Error("table content should be preserved")
	}

	// Verify headings were extracted
	if len(headings) < 2 {
		t.Errorf("expected at least 2 headings, got %d", len(headings))
	}

	checkGolden(t, filepath.Join(goldenDir(t), "tables.html"), html)
}

// TestGoldenImages tests rendering of Markdown with image references.
func TestGoldenImages(t *testing.T) {
	parser := markdown.NewParser()

	markdownContent := "# Image Test\n\n## Image with alt text\n![A sample image](./images/sample.png)\n\n" +
		"## Image with link\n[![Click me](./images/button.png)](https://example.com)\n\n" +
		"## Multiple images\n![First](./img1.png)\n![Second](./img2.png)\n\n" +
		"## Image in paragraph\nThis paragraph contains an ![inline image](./inline.png) within text.\n\n" +
		"## Image with title\n![Image with title](./titled.png \"Image Title\")\n"

	html, headings, err := parser.Parse([]byte(markdownContent))
	if err != nil {
		t.Fatalf("failed to parse markdown with images: %v", err)
	}

	// Sanity checks before golden comparison.
	if !strings.Contains(html, "<img") {
		t.Error("images should be rendered with <img> tags")
	}
	if !strings.Contains(html, "src=") {
		t.Error("images should have src attribute")
	}
	if !strings.Contains(html, "alt=") {
		t.Error("images should have alt attribute for accessibility")
	}

	// Verify headings were extracted
	if len(headings) < 2 {
		t.Errorf("expected at least 2 headings, got %d", len(headings))
	}

	checkGolden(t, filepath.Join(goldenDir(t), "images.html"), html)
}

// TestGoldenCJKContent tests rendering of CJK (Chinese, Japanese, Korean) characters.
func TestGoldenCJKContent(t *testing.T) {
	parser := markdown.NewParser()

	markdownContent := "# CJK 字符测试\n\n## 中文内容\n这是一个包含中文字符的测试。mdPress 应该能够正确处理 CJK 字符，包括：\n\n" +
		"- **粗体中文** 和 *斜体中文*\n- 中文代码变量：`变量名`\n- 中文链接：[示例链接](https://example.com)\n\n" +
		"### 中文代码块\n```go\n// 中文注释\nfunc 主程序() {\n    格式.打印(\"你好，世界！\")\n}\n```\n\n" +
		"## 混合内容\nEnglish and 中文 mixed together.\n\n" +
		"| 列1   | 列2       |\n|-------|-----------|" +
		"\n| 中文  | 英文      |\n| 数据1 | 数据2     |\n\n" +
		"## 日文内容\nこれは日本語のテストです。\n\n" +
		"## 韓文內容\n이것은 한국어 테스트입니다.\n\n" +
		"## 特殊符号\n- 数学表达式：x² + y² = z²\n- 货币：¥100, $50, €75\n- 其他符号：©2024, ®Brand, ™Mark\n"

	html, headings, err := parser.Parse([]byte(markdownContent))
	if err != nil {
		t.Fatalf("failed to parse markdown with CJK content: %v", err)
	}

	// Sanity checks before golden comparison.
	if !strings.Contains(html, "中文") {
		t.Error("Chinese content should be preserved")
	}
	if !strings.Contains(html, "日本語") {
		t.Error("Japanese content should be preserved")
	}
	if !strings.Contains(html, "한국어") {
		t.Error("Korean content should be preserved")
	}
	if !strings.Contains(html, "粗体中文") {
		t.Error("CJK in bold should be preserved")
	}
	if !strings.Contains(html, "变量名") {
		t.Error("CJK in code should be preserved")
	}
	if !strings.Contains(html, "<table") {
		t.Error("tables with CJK content should render")
	}
	if !strings.Contains(html, "<code") {
		t.Error("code blocks with CJK should render")
	}

	// Verify headings were extracted (including CJK headings)
	if len(headings) < 3 {
		t.Errorf("expected at least 3 headings, got %d", len(headings))
	}

	// Check for CJK heading extraction
	var foundCJKHeading bool
	for _, h := range headings {
		if strings.Contains(h.Text, "中文") || strings.Contains(h.Text, "日本語") || strings.Contains(h.Text, "한국어") {
			foundCJKHeading = true
			break
		}
	}
	if !foundCJKHeading {
		t.Error("CJK headings should be extracted")
	}

	checkGolden(t, filepath.Join(goldenDir(t), "cjk_content.html"), html)
}
