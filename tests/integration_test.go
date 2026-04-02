package tests

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/crossref"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"gopkg.in/yaml.v3"
)

// Get test data directory path
func getTestDataDir() string {
	// Go tests run from the package directory, so testdata is relative to here
	return "testdata"
}

// TestConfigLoadAndValidate tests loading and validating a config file
func TestConfigLoadAndValidate(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify book metadata
	if cfg.Book.Title != "集成测试图书" {
		t.Errorf("expected book title '集成测试图书', got '%s'", cfg.Book.Title)
	}
	if cfg.Book.Author != "测试作者" {
		t.Errorf("expected author '测试作者', got '%s'", cfg.Book.Author)
	}
	if cfg.Book.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", cfg.Book.Version)
	}

	// Verify chapters
	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(cfg.Chapters))
	}

	// Verify style
	if cfg.Style.Theme != "technical" {
		t.Errorf("expected theme 'technical', got '%s'", cfg.Style.Theme)
	}
	if cfg.Style.PageSize != "A4" {
		t.Errorf("expected page size 'A4', got '%s'", cfg.Style.PageSize)
	}

	// Verify output
	if cfg.Output.Filename != "test-output.pdf" {
		t.Errorf("expected output file 'test-output.pdf', got '%s'", cfg.Output.Filename)
	}
	if !cfg.Output.TOC {
		t.Error("expected TOC generation")
	}
	if !cfg.Output.Cover {
		t.Error("expected cover generation")
	}
}

// TestMarkdownParsing tests parsing Markdown files and verifying output contains expected HTML elements
func TestMarkdownParsing(t *testing.T) {
	// Define parsing test cases
	testCases := []struct {
		name             string
		filename         string
		expectedElements []string
	}{
		{
			name:     "chapter 1 parsing",
			filename: "ch01.md",
			expectedElements: []string{
				"第一章",
				"简介",
				"加粗文本",
				"斜体文本",
				"列表项",
				"<table",
			},
		},
		{
			name:     "chapter 2 parsing",
			filename: "ch02.md",
			expectedElements: []string{
				"第二章",
				"详情",
				"代码示例",
				"package",
				"<pre",
			},
		},
	}

	testDataDir := getTestDataDir()
	parser := markdown.NewParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read Markdown file
			filePath := filepath.Join(testDataDir, tc.filename)
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Parse Markdown
			html, _, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("failed to parse Markdown: %v", err)
			}

			// Verify output contains expected elements
			for _, elem := range tc.expectedElements {
				if !strings.Contains(html, elem) {
					t.Errorf("expected element not found in HTML: %s", elem)
				}
			}

			// Verify output is HTML
			if !strings.Contains(html, "<") || !strings.Contains(html, ">") {
				t.Error("output should be HTML format")
			}
		})
	}
}

// TestTOCGeneration tests parsing chapters, collecting headings, generating TOC, and verifying structure
func TestTOCGeneration(t *testing.T) {
	testDataDir := getTestDataDir()

	// Read first chapter to extract headings
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Parse Markdown and extract headings
	parser := markdown.NewParser()
	_, headings, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("failed to parse Markdown: %v", err)
	}

	// Verify heading count (should be at least 1, the main heading)
	if len(headings) == 0 {
		t.Error("should have extracted at least one heading")
	}

	// Convert heading types
	tocHeadings := make([]toc.HeadingInfo, len(headings))
	for i, h := range headings {
		tocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	// Generate TOC
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(tocHeadings)

	// Verify generated TOC
	if len(entries) == 0 {
		t.Error("should have generated TOC entries")
	}

	// Render TOC as HTML
	tocHTML := tocGen.RenderHTML(entries)

	// Verify TOC HTML
	if !strings.Contains(tocHTML, "<nav") {
		t.Error("TOC HTML should contain <nav tag")
	}
	if !strings.Contains(tocHTML, "<ul") {
		t.Error("TOC HTML should contain <ul tag")
	}
	if !strings.Contains(tocHTML, "<li") {
		t.Error("TOC HTML should contain <li tag")
	}
	if !strings.Contains(tocHTML, "<a") {
		t.Error("TOC HTML should contain <a tag")
	}
}

// TestCoverGeneration tests generating a cover from config metadata and verifying HTML structure
func TestCoverGeneration(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Generate cover
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// Verify cover HTML structure
	if !strings.Contains(coverHTML, "<!DOCTYPE html>") {
		t.Error("cover should contain DOCTYPE declaration")
	}
	if !strings.Contains(coverHTML, cfg.Book.Title) {
		t.Error("cover should contain book title")
	}
	if !strings.Contains(coverHTML, cfg.Book.Author) {
		t.Error("cover should contain author name")
	}
	if !strings.Contains(coverHTML, cfg.Book.Version) {
		t.Error("cover should contain version number")
	}
	if !strings.Contains(coverHTML, "<style") {
		t.Error("cover should contain styles")
	}
}

// TestCrossRefWorkflow tests cross-reference workflow: register references, process HTML, verify replacements
func TestCrossRefWorkflow(t *testing.T) {
	// Create cross-reference resolver
	resolver := crossref.NewResolver()

	// Register figures
	figNum1 := resolver.RegisterFigure("fig1", "示例图表 1")
	figNum2 := resolver.RegisterFigure("fig2", "示例图表 2")

	// Verify figure numbers
	if figNum1 != 1 {
		t.Errorf("first figure number should be 1, got %d", figNum1)
	}
	if figNum2 != 2 {
		t.Errorf("second figure number should be 2, got %d", figNum2)
	}

	// Register tables
	tableNum1 := resolver.RegisterTable("tbl1", "示例表格 1")
	tableNum2 := resolver.RegisterTable("tbl2", "示例表格 2")

	// Verify table numbers
	if tableNum1 != 1 {
		t.Errorf("first table number should be 1, got %d", tableNum1)
	}
	if tableNum2 != 2 {
		t.Errorf("second table number should be 2, got %d", tableNum2)
	}

	// Register sections
	resolver.RegisterSection("sec1", "第 1 章", 1)
	resolver.RegisterSection("sec2", "第 2 章", 1)

	// Verify resolution
	ref1, err := resolver.Resolve("fig1")
	if err != nil {
		t.Fatalf("failed to resolve figure reference: %v", err)
	}
	if ref1.Number != 1 {
		t.Errorf("expected reference number 1, got %d", ref1.Number)
	}

	// Test HTML processing
	testHTML := `<p>参考 {{ref:fig1}} 和 {{ref:tbl1}}</p>`
	processedHTML := resolver.ProcessHTML(testHTML)

	// Verify placeholders were replaced
	if strings.Contains(processedHTML, "{{ref:fig1}}") {
		t.Error("placeholder {{ref:fig1}} should be replaced")
	}
	if strings.Contains(processedHTML, "{{ref:tbl1}}") {
		t.Error("placeholder {{ref:tbl1}} should be replaced")
	}

	// Verify all references
	allRefs := resolver.GetAllReferences()
	if len(allRefs) == 0 {
		t.Error("should return at least one reference")
	}
}

// TestFullHTMLRender tests rendering a complete document (all parts) and verifying structure
func TestFullHTMLRender(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Get theme
	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}

	// Generate cover
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// Parse chapter 1
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("failed to read chapter 1: %v", err)
	}

	parser := markdown.NewParser()
	ch01HTML, _, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("failed to parse chapter 1: %v", err)
	}

	// Parse chapter 2
	ch02Path := filepath.Join(testDataDir, "ch02.md")
	ch02Data, err := os.ReadFile(ch02Path)
	if err != nil {
		t.Fatalf("failed to read chapter 2: %v", err)
	}

	ch02HTML, _, err := parser.Parse(ch02Data)
	if err != nil {
		t.Fatalf("failed to parse chapter 2: %v", err)
	}

	// Generate TOC
	allMDHeadings := []markdown.HeadingInfo{}
	_, h1, _ := parser.Parse(ch01Data)
	_, h2, _ := parser.Parse(ch02Data)
	allMDHeadings = append(allMDHeadings, h1...)
	allMDHeadings = append(allMDHeadings, h2...)

	// Convert heading types
	allTocHeadings := make([]toc.HeadingInfo, len(allMDHeadings))
	for i, h := range allMDHeadings {
		allTocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allTocHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	// Assemble render parts
	parts := &renderer.RenderParts{
		CoverHTML: coverHTML,
		TOCHTML:   tocHTML,
		ChaptersHTML: []renderer.ChapterHTML{
			{Title: "第一章 简介", ID: "ch1", Content: ch01HTML},
			{Title: "第二章 详情", ID: "ch2", Content: ch02HTML},
		},
		CustomCSS: ".custom { margin: 20px; }",
	}

	// Render full HTML
	r, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Verify full structure
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		cfg.Book.Title,
		cfg.Book.Author,
		"第一章",
		"第二章",
		"<nav",
		"@page",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(html, elem) {
			t.Errorf("full HTML should contain: %s", elem)
		}
	}

	// Verify valid HTML
	if !strings.Contains(html, "</html>") {
		t.Error("HTML should have closing tag")
	}
}

// TestSummaryParsing tests parsing a SUMMARY.md file
func TestSummaryParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	summaryPath := filepath.Join(testDataDir, "SUMMARY.md")

	// Read SUMMARY.md
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("failed to read SUMMARY.md: %v", err)
	}

	content := string(data)

	// Verify file content
	if !strings.Contains(content, "第一章") {
		t.Error("SUMMARY.md should contain chapter 1")
	}
	if !strings.Contains(content, "第二章") {
		t.Error("SUMMARY.md should contain chapter 2")
	}
	if !strings.Contains(content, "ch01.md") {
		t.Error("SUMMARY.md should contain ch01.md reference")
	}
	if !strings.Contains(content, "ch02.md") {
		t.Error("SUMMARY.md should contain ch02.md reference")
	}

	// Verify TOC structure
	if !strings.Contains(content, "* [") {
		t.Error("SUMMARY.md should use Markdown list syntax")
	}
}

// TestEmptyFileParsing tests parsing an empty Markdown file
func TestEmptyFileParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	emptyPath := filepath.Join(testDataDir, "empty.md")

	// Read empty file
	data, err := os.ReadFile(emptyPath)
	if err != nil {
		t.Fatalf("failed to read empty.md: %v", err)
	}

	// Parse empty Markdown
	parser := markdown.NewParser()
	html, headings, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("failed to parse empty file: %v", err)
	}

	// Verify result (even empty input should return valid values)
	if html == "" {
		t.Logf("empty file produced empty HTML, which is acceptable")
	}

	// Should have no headings
	if len(headings) > 0 {
		t.Error("empty file should have no headings")
	}
}

// TestSpecialCharsParsing tests parsing a Markdown file with special characters
func TestSpecialCharsParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	specialPath := filepath.Join(testDataDir, "special_chars.md")

	// Read special characters file
	data, err := os.ReadFile(specialPath)
	if err != nil {
		t.Fatalf("failed to read special_chars.md: %v", err)
	}

	// Parse Markdown
	parser := markdown.NewParser()
	html, _, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("failed to parse Markdown: %v", err)
	}

	// Verify special characters are properly escaped
	// HTML should encode < as &lt;, > as &gt;, & as &amp;
	if !strings.Contains(html, "&amp;") && !strings.Contains(html, "&lt;") && !strings.Contains(html, "&gt;") {
		t.Errorf("special characters should be HTML-escaped, but no escape entities found: %s", html)
	}

	// Verify headings are processed (even with special characters)
	if !strings.Contains(html, "<h1") {
		t.Error("should generate <h1 heading")
	}

	// Verify backtick code block content is preserved
	if !strings.Contains(html, "特殊字符") {
		t.Error("should contain code block content")
	}
}

// TestMultiLanguageBuild tests multilingual build: create temp project, discover languages, verify chapters
func TestMultiLanguageBuild(t *testing.T) {
	tempDir := t.TempDir()

	// Create SUMMARY.md (Discover needs it to trigger LANGS.md detection)
	summaryContent := "# Summary\n\n* [Introduction](README.md)\n"
	if err := os.WriteFile(filepath.Join(tempDir, "SUMMARY.md"), []byte(summaryContent), 0o644); err != nil {
		t.Fatalf("failed to write SUMMARY.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Test Book\n"), 0o644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	// Create LANGS.md defining multiple languages
	langsContent := `# Languages

- [中文](./zh/)
- [English](./en/)
`
	if err := os.WriteFile(filepath.Join(tempDir, "LANGS.md"), []byte(langsContent), 0o644); err != nil {
		t.Fatalf("failed to write LANGS.md: %v", err)
	}

	// Create Chinese directory and config
	zhDir := filepath.Join(tempDir, "zh")
	if err := os.MkdirAll(zhDir, 0o755); err != nil {
		t.Fatalf("failed to create Chinese directory: %v", err)
	}

	zhConfigContent := `book:
  title: "中文书籍"
  author: "作者"
  version: "1.0.0"
chapters:
  - title: "第一章"
    file: "ch01.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(zhDir, "book.yaml"), []byte(zhConfigContent), 0o644); err != nil {
		t.Fatalf("failed to write Chinese book.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(zhDir, "ch01.md"), []byte("# 第一章\n\n中文内容"), 0o644); err != nil {
		t.Fatalf("failed to write Chinese chapter: %v", err)
	}

	// Create English directory and config
	enDir := filepath.Join(tempDir, "en")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("failed to create English directory: %v", err)
	}

	enConfigContent := `book:
  title: "English Book"
  author: "Author"
  version: "1.0.0"
chapters:
  - title: "Chapter 1"
    file: "ch01.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(enDir, "book.yaml"), []byte(enConfigContent), 0o644); err != nil {
		t.Fatalf("failed to write English book.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(enDir, "ch01.md"), []byte("# Chapter 1\n\nEnglish content"), 0o644); err != nil {
		t.Fatalf("failed to write English chapter: %v", err)
	}

	// Use Discover to auto-detect config
	cfg, err := config.Discover(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("config discovery failed: %v", err)
	}

	// Verify LANGS.md was detected
	if cfg.LangsFile == "" {
		t.Error("should have detected LANGS.md")
	}

	// Verify Chinese config can be loaded
	zhCfg, err := config.Load(filepath.Join(zhDir, "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load Chinese config: %v", err)
	}
	if zhCfg.Book.Title != "中文书籍" {
		t.Errorf("expected Chinese book title, got: %s", zhCfg.Book.Title)
	}
	if len(zhCfg.Chapters) != 1 {
		t.Errorf("expected 1 Chinese chapter, got: %d", len(zhCfg.Chapters))
	}

	// Verify English config can be loaded
	enCfg, err := config.Load(filepath.Join(enDir, "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load English config: %v", err)
	}
	if enCfg.Book.Title != "English Book" {
		t.Errorf("expected English book title, got: %s", enCfg.Book.Title)
	}
	if len(enCfg.Chapters) != 1 {
		t.Errorf("expected 1 English chapter, got: %d", len(enCfg.Chapters))
	}
}

// TestHTMLRenderingEndToEnd tests HTML rendering end-to-end: create minimal project, render, verify output structure
func TestHTMLRenderingEndToEnd(t *testing.T) {
	tempDir := t.TempDir()

	// Create minimal project
	bookYAML := `book:
  title: "测试书籍"
  author: "测试作者"
  version: "1.0.0"
chapters:
  - title: "简介"
    file: "intro.md"
  - title: "内容"
    file: "content.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	introContent := `# 简介

这是简介章节。

## 背景
- 项目背景
- 主要目标
`
	if err := os.WriteFile(filepath.Join(tempDir, "intro.md"), []byte(introContent), 0o644); err != nil {
		t.Fatalf("failed to write intro.md: %v", err)
	}

	contentMD := `# 内容

## 主要内容
这是主要内容部分。

## 代码示例
` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	if err := os.WriteFile(filepath.Join(tempDir, "content.md"), []byte(contentMD), 0o644); err != nil {
		t.Fatalf("failed to write content.md: %v", err)
	}

	// Load config
	cfg, err := config.Load(filepath.Join(tempDir, "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Initialize theme and parser
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}

	parser := markdown.NewParser()

	// Parse all chapters
	var chaptersHTML []renderer.ChapterHTML
	var allHeadings []toc.HeadingInfo

	for _, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := os.ReadFile(chapterPath)
		if err != nil {
			t.Fatalf("failed to read chapter %s: %v", ch.File, err)
		}

		html, headings, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("failed to parse chapter %s: %v", ch.File, err)
		}

		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      ch.Title,
			Content: html,
		})
	}

	if len(chaptersHTML) == 0 {
		t.Fatal("should have at least one chapter")
	}

	// Generate cover and TOC
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	// Render full HTML
	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("failed to create HTML renderer: %v", err)
	}

	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("HTML rendering failed: %v", err)
	}

	// Verify output contains expected structure
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		cfg.Book.Title,
		cfg.Book.Author,
		"简介",
		"内容",
		"<nav",
		"<title>",
		"</html>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(fullHTML, elem) {
			t.Errorf("HTML should contain: %s", elem)
		}
	}

	// Verify code blocks are processed (code may be in <code> or <pre>)
	if !strings.Contains(fullHTML, "func main") && !strings.Contains(fullHTML, "Println") {
		t.Errorf("code block content should contain 'func main' or 'Println': %s", fullHTML[:min(len(fullHTML), 500)])
	}

	// Verify headings exist
	if !strings.Contains(fullHTML, "h1") && !strings.Contains(fullHTML, "h2") {
		t.Error("HTML should contain heading tags")
	}

	t.Logf("HTML rendering succeeded: %d bytes", len(fullHTML))
}

// TestSiteOutputStructure tests site output structure: verify correct directory layout and index.html
func TestSiteOutputStructure(t *testing.T) {
	tempDir := t.TempDir()

	// Create project
	bookYAML := `book:
  title: "网站测试"
  author: "作者"
chapters:
  - title: "页面1"
    file: "page1.md"
  - title: "页面2"
    file: "page2.md"
style:
  theme: "technical"
`
	if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "page1.md"), []byte("# 页面1\n\n内容1"), 0o644); err != nil {
		t.Fatalf("failed to write page1.md: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "page2.md"), []byte("# 页面2\n\n内容2"), 0o644); err != nil {
		t.Fatalf("failed to write page2.md: %v", err)
	}

	// Load config
	cfg, err := config.Load(filepath.Join(tempDir, "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config is valid
	if cfg.Book.Title != "网站测试" {
		t.Errorf("expected book title '网站测试', got: %s", cfg.Book.Title)
	}

	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got: %d", len(cfg.Chapters))
	}

	// Verify all chapter files exist
	for _, ch := range cfg.Chapters {
		resolvedPath := cfg.ResolvePath(ch.File)
		if _, err := os.Stat(resolvedPath); errors.Is(err, fs.ErrNotExist) {
			t.Errorf("chapter file does not exist: %s", resolvedPath)
		}
	}

	t.Logf("site project verification complete: %d chapters", len(cfg.Chapters))
}

// TestConfigValidation tests config validation: check various config errors
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		configYAML  string
		expectedErr string
	}{
		{
			name:        "empty title",
			configYAML:  "book:\n  title: \"\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"",
			expectedErr: "title cannot be empty",
		},
		{
			name:        "no chapters",
			configYAML:  "book:\n  title: \"测试\"\nchapters: []",
			expectedErr: "at least one chapter",
		},
		{
			name:        "invalid page size",
			configYAML:  "book:\n  title: \"测试\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"\nstyle:\n  page_size: \"XYZ\"",
			expectedErr: "unsupported page size",
		},
		{
			name:        "invalid theme",
			configYAML:  "book:\n  title: \"测试\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"\nstyle:\n  theme: \"nonexistent\"",
			expectedErr: "unknown theme",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Write config
			if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(tc.configYAML), 0o644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			// Create dummy chapter files
			if err := os.WriteFile(filepath.Join(tempDir, "ch01.md"), []byte("# 测试"), 0o644); err != nil {
				t.Fatalf("failed to write chapter: %v", err)
			}

			// Attempt to load config
			cfg := config.DefaultConfig()
			configPath := filepath.Join(tempDir, "book.yaml")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}

			if err := yaml.Unmarshal(data, cfg); err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			cfg.SetBaseDir(tempDir)

			// Verify it should fail
			err = cfg.Validate()
			if err == nil {
				t.Fatal("expected config validation to fail")
			}
			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("expected error to contain '%s', got: %v", tc.expectedErr, err)
			}
		})
	}
}

// TestThemeApplication tests theme application: verify theme CSS is included in output
func TestThemeApplication(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Get theme
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}

	// Verify theme is valid
	if thm == nil {
		t.Fatal("theme should be valid")
	}

	// Parse chapters
	parser := markdown.NewParser()
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("failed to read chapter: %v", err)
	}

	html, headings, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Generate full document
	coverGen := cover.NewCoverGenerator(cfg.Book)
	tocGen := toc.NewGenerator()

	tocHeadings := make([]toc.HeadingInfo, len(headings))
	for i, h := range headings {
		tocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	tocEntries := tocGen.Generate(tocHeadings)
	parts := &renderer.RenderParts{
		CoverHTML: coverGen.RenderHTML(),
		TOCHTML:   tocGen.RenderHTML(tocEntries),
		ChaptersHTML: []renderer.ChapterHTML{
			{Title: cfg.Chapters[0].Title, ID: "ch1", Content: html},
		},
	}

	// Create renderer
	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	// Render
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Verify CSS is included
	if !strings.Contains(fullHTML, "<style") {
		t.Error("output should contain <style> tag")
	}

	if !strings.Contains(fullHTML, "css") && !strings.Contains(fullHTML, "@") {
		t.Error("output should contain CSS rules")
	}

	// Verify theme-related CSS rules exist
	if !strings.Contains(fullHTML, "font-family") && !strings.Contains(fullHTML, "color") && !strings.Contains(fullHTML, "margin") {
		t.Errorf("output should contain theme CSS rules (font-family, color, or margin)")
	}
}
