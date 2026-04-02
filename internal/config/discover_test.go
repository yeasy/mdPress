package config

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// extractTitleFromFile is a test helper that extracts the first H1 heading from a markdown file.
// The production code inlined this logic into ExtractReadmeMetadata.
func extractTitleFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}
	return ""
}

// TestDiscoverWithBookYaml tests Discover prioritizes book.yaml
func TestDiscoverWithBookYaml(t *testing.T) {
	dir := t.TempDir()

	// Create book.yaml
	yaml := `book:
  title: "Book YAML Title"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write book.yaml failed: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Content"), 0o644); err != nil {
		t.Fatalf("write ch1.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Book YAML Title" {
		t.Errorf("expected 'Book YAML Title', got %q", cfg.Book.Title)
	}
	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter, got %d", len(cfg.Chapters))
	}
}

// TestDiscoverWithBookJSON tests Discover prioritizes book.json over SUMMARY.md
func TestDiscoverWithBookJSON(t *testing.T) {
	dir := t.TempDir()

	// Create book.json
	json := `{
  "title": "Book JSON Title"
}`
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(json), 0o644); err != nil {
		t.Fatalf("write book.json failed: %v", err)
	}

	// Create SUMMARY.md (chapters are loaded from here, not from book.json)
	// NOTE: chapter file must NOT be named "summary.md" because on
	// case-insensitive file systems (macOS) it overwrites SUMMARY.md.
	summary := `# Summary

* [Summary Ch](chapter.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write SUMMARY.md failed: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(dir, "chapter.md"), []byte("# Content"), 0o644); err != nil {
		t.Fatalf("write chapter.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Book JSON Title" {
		t.Errorf("expected 'Book JSON Title', got %q", cfg.Book.Title)
	}
	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter from SUMMARY.md, got %d", len(cfg.Chapters))
	}
}

// TestDiscoverWithSummaryOnly tests Discover falls back to SUMMARY.md
func TestDiscoverWithSummaryOnly(t *testing.T) {
	dir := t.TempDir()

	// Create SUMMARY.md
	summary := `# Summary

* [Intro](intro.md)
* [Chapter 1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write SUMMARY.md failed: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"intro.md", "ch1.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", file, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters from SUMMARY.md, got %d", len(cfg.Chapters))
	}
	if cfg.Chapters[0].Title != "Intro" {
		t.Errorf("expected first chapter 'Intro', got %q", cfg.Chapters[0].Title)
	}
}

// TestDiscoverAutoDiscoverMarkdown tests Discover auto-discovers markdown files
func TestDiscoverAutoDiscoverMarkdown(t *testing.T) {
	dir := t.TempDir()

	// Create markdown files
	files := []struct {
		name    string
		content string
	}{
		{"01_intro.md", "# Introduction\nContent here"},
		{"02_chapter.md", "# Main Chapter\nMore content"},
		{"03_conclusion.md", "# Conclusion\nEnd"},
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte(f.content), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", f.name, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 3 {
		t.Errorf("expected 3 chapters, got %d", len(cfg.Chapters))
	}

	// Verify lexical ordering
	if cfg.Chapters[0].Title != "Introduction" {
		t.Errorf("expected first chapter 'Introduction', got %q", cfg.Chapters[0].Title)
	}
	if cfg.Chapters[1].Title != "Main Chapter" {
		t.Errorf("expected second chapter 'Main Chapter', got %q", cfg.Chapters[1].Title)
	}
	if cfg.Chapters[2].Title != "Conclusion" {
		t.Errorf("expected third chapter 'Conclusion', got %q", cfg.Chapters[2].Title)
	}
}

// TestDiscoverEmptyDirectory tests Discover fails on empty directory
func TestDiscoverEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	_, err := Discover(context.Background(), dir)
	if err == nil {
		t.Error("expected error for empty directory")
	}

	discoverErr, ok := err.(*DiscoverError)
	if !ok {
		t.Errorf("expected DiscoverError, got %T", err)
	}
	if discoverErr == nil {
		t.Fatal("discoverErr is nil")
	}
	if !strings.Contains(discoverErr.Error(), "no .md files") {
		t.Errorf("expected 'no .md files' error, got %q", discoverErr.Error())
	}
}

// TestDiscoverNonExistentDirectory tests Discover with non-existent directory
func TestDiscoverNonExistentDirectory(t *testing.T) {
	_, err := Discover(context.Background(), "/nonexistent/path/to/directory")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

// TestDiscoverWithReadmeAsFirstChapter tests that README.md becomes first chapter
func TestDiscoverWithReadmeAsFirstChapter(t *testing.T) {
	dir := t.TempDir()

	// Create README.md as first chapter
	readme := `# Project Overview
This is the main overview of the project.
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	// Create additional chapters
	if err := os.WriteFile(filepath.Join(dir, "chapter1.md"), []byte("# Chapter 1\nContent"), 0o644); err != nil {
		t.Fatalf("write chapter1.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(cfg.Chapters))
	}

	// README should be first
	if cfg.Chapters[0].File != "README.md" {
		t.Errorf("expected first chapter to be README.md, got %q", cfg.Chapters[0].File)
	}
	if cfg.Chapters[0].Title != "Project Overview" {
		t.Errorf("expected first chapter title 'Project Overview', got %q", cfg.Chapters[0].Title)
	}

	// Book title should come from README
	if cfg.Book.Title != "Project Overview" {
		t.Errorf("expected book title 'Project Overview', got %q", cfg.Book.Title)
	}
}

// TestDiscoverReadmeWithoutH1 tests README.md without H1 uses "Preface"
func TestDiscoverReadmeWithoutH1(t *testing.T) {
	dir := t.TempDir()

	// Create README.md without H1
	readme := "Just some intro text without heading\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	// Create a chapter with H1
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# First Chapter\nContent"), 0o644); err != nil {
		t.Fatalf("write ch1.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Chapters[0].Title != "Preface" {
		t.Errorf("expected README title 'Preface', got %q", cfg.Chapters[0].Title)
	}
}

// TestDiscoverSkipsSpecialFiles tests that special files are skipped
func TestDiscoverSkipsSpecialFiles(t *testing.T) {
	dir := t.TempDir()

	// Create special files that should be skipped during auto-discovery,
	// plus one regular chapter. Note: SUMMARY.md is omitted because its
	// presence triggers summary-based discovery instead of auto-discovery.
	files := map[string]string{
		"GLOSSARY.md":     "## Term\nDefinition\n",
		"LANGS.md":        "* [English](en/)\n",
		"CHANGELOG.md":    "# Changelog\n## v1.0.0\n- Initial release\n",
		"CONTRIBUTING.md": "# Contributing\nPR welcome\n",
		"LICENSE.md":      "MIT License\n",
		"regular.md":      "# Regular Chapter\nContent",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should only have the regular chapter, not the special files
	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter, got %d", len(cfg.Chapters))
	}
	if cfg.Chapters[0].Title != "Regular Chapter" {
		t.Errorf("expected 'Regular Chapter', got %q", cfg.Chapters[0].Title)
	}
}

// TestDiscoverDetectsGlossary tests Discover detects GLOSSARY.md
func TestDiscoverDetectsGlossary(t *testing.T) {
	dir := t.TempDir()

	// Create chapter
	if err := os.WriteFile(filepath.Join(dir, "ch.md"), []byte("# Chapter\nContent"), 0o644); err != nil {
		t.Fatalf("write ch.md failed: %v", err)
	}

	// Create GLOSSARY.md
	if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("## API\nApplication Programming Interface\n"), 0o644); err != nil {
		t.Fatalf("write GLOSSARY.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.GlossaryFile != filepath.Join(dir, "GLOSSARY.md") {
		t.Errorf("expected GlossaryFile to be set, got %q", cfg.GlossaryFile)
	}
}

// TestAutoDiscoverWithNestedDirectories tests markdown discovery in nested dirs
func TestAutoDiscoverWithNestedDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure
	if err := os.Mkdir(filepath.Join(dir, "part1"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "part2"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Create files in nested directories
	files := map[string]string{
		"intro.md":          "# Introduction\nIntro text",
		"part1/chapter1.md": "# Part 1 Chapter\nContent",
		"part2/chapter2.md": "# Part 2 Chapter\nContent",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", path, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 3 {
		t.Errorf("expected 3 chapters from nested dirs, got %d", len(cfg.Chapters))
	}
}

// TestAutoDiscoverSkipsHiddenDirectories tests that hidden directories are skipped
func TestAutoDiscoverSkipsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create normal chapter
	if err := os.WriteFile(filepath.Join(dir, "normal.md"), []byte("# Normal\nContent"), 0o644); err != nil {
		t.Fatalf("write normal.md failed: %v", err)
	}

	// Create hidden directory
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.Mkdir(hiddenDir, 0o755); err != nil {
		t.Fatalf("mkdir .hidden failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.md"), []byte("# Secret\nContent"), 0o644); err != nil {
		t.Fatalf("write secret.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter (hidden dir skipped), got %d", len(cfg.Chapters))
	}
	if cfg.Chapters[0].Title != "Normal" {
		t.Errorf("expected 'Normal', got %q", cfg.Chapters[0].Title)
	}
}

// TestAutoDiscoverSkipsNodeModules tests that node_modules is skipped
func TestAutoDiscoverSkipsNodeModules(t *testing.T) {
	dir := t.TempDir()

	// Create normal chapter
	if err := os.WriteFile(filepath.Join(dir, "normal.md"), []byte("# Normal\nContent"), 0o644); err != nil {
		t.Fatalf("write normal.md failed: %v", err)
	}

	// Create node_modules directory
	nodeDir := filepath.Join(dir, "node_modules")
	if err := os.Mkdir(nodeDir, 0o755); err != nil {
		t.Fatalf("mkdir node_modules failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(nodeDir, "module.md"), []byte("# Module\nContent"), 0o644); err != nil {
		t.Fatalf("write module.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter (node_modules skipped), got %d", len(cfg.Chapters))
	}
}

// TestFindMarkdownFiles tests finding markdown files
func TestFindMarkdownFiles(t *testing.T) {
	dir := t.TempDir()

	// Create structure
	files := map[string]bool{
		"ch1.md":    true,
		"ch2.md":    true,
		"README.md": true,
		"doc.txt":   false,
		"data.json": false,
	}

	for name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}

	// Create nested markdown
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "subdir/nested.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write nested.md failed: %v", err)
	}

	found, err := findMarkdownFiles(dir)
	if err != nil {
		t.Fatalf("findMarkdownFiles failed: %v", err)
	}

	if len(found) != 4 {
		t.Errorf("expected 4 markdown files, got %d", len(found))
	}

	// Check that only .md files are found
	for _, path := range found {
		if !strings.HasSuffix(path, ".md") {
			t.Errorf("found non-.md file: %s", path)
		}
	}
}

// TestExtractTitleFromFile tests extracting H1 from markdown
func TestExtractTitleFromFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple H1",
			content:  "# Hello World\n",
			expected: "Hello World",
		},
		{
			name:     "H1 with leading/trailing spaces",
			content:  "#   Title with spaces   \n",
			expected: "Title with spaces",
		},
		{
			name:     "H1 not on first line",
			content:  "Some text\n# Second Title\n",
			expected: "Second Title",
		},
		{
			name:     "no H1",
			content:  "## H2 Title\nContent\n",
			expected: "",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
		{
			name:     "H1 with special characters",
			content:  "# Title & Subtitle (v1.0)\n",
			expected: "Title & Subtitle (v1.0)",
		},
		{
			name:     "Chinese H1",
			content:  "# 项目标题\n内容\n",
			expected: "项目标题",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write failed: %v", err)
			}

			result := extractTitleFromFile(path)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestExtractTitleFromFileNonExistent tests extractTitleFromFile with non-existent file
func TestExtractTitleFromFileNonExistent(t *testing.T) {
	result := extractTitleFromFile("/nonexistent/file.md")
	if result != "" {
		t.Errorf("expected empty string for non-existent file, got %q", result)
	}
}

// TestDetectContentLanguageEnglish tests language detection for English
func TestDetectContentLanguageEnglish(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "pure English",
			content:  "This is a test with English letters only and no CJK characters",
			expected: "en-US",
		},
		{
			name:     "English with numbers",
			content:  "Testing 123 with some numbers and letters",
			expected: "en-US",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "en-US",
		},
		{
			name:     "numbers and punctuation only",
			content:  "123 456 .,!?;:",
			expected: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentLanguage(tt.content)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestDetectContentLanguageChinese tests language detection for Chinese
func TestDetectContentLanguageChinese(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "pure Chinese",
			content:  "这是一个完整的中文文本，包含很多汉字。用来测试中文检测功能。",
			expected: "zh-CN",
		},
		{
			name:     "Chinese with English words",
			content:  "这是一个有英文的中文文本。Some English words here. 更多中文内容在这里。",
			expected: "zh-CN",
		},
		{
			name:     "low ratio Chinese",
			content:  "This is mostly English text with only 一个 Chinese character",
			expected: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectContentLanguage(tt.content)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestInferBookTitle tests book title inference logic
func TestInferBookTitle(t *testing.T) {
	tests := []struct {
		name     string
		h1Title  string
		content  string
		expected string // Will check if result contains this
		dir      string
	}{
		{
			name:     "prefer Chinese badge title",
			h1Title:  "README",
			content:  "[![](https://img.shields.io/badge/Docker%E6%8A%80%E6%9C%AF%E5%85%A5%E9%97%A8-blue])",
			expected: "Docker技术入门",
			dir:      "/test",
		},
		{
			name:     "fallback to meaningful H1",
			h1Title:  "My Project Title",
			content:  "Some content",
			expected: "My Project Title",
			dir:      "/test",
		},
		{
			name:     "ignore generic H1",
			h1Title:  "Preface",
			content:  "Some content",
			expected: "test", // fallback to dir name
			dir:      "/test",
		},
		{
			name:     "ignore README H1",
			h1Title:  "README",
			content:  "Some content",
			expected: "test", // fallback to dir name
			dir:      "/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferBookTitle(tt.h1Title, tt.content, tt.dir)

			if !strings.Contains(strings.ToLower(result), strings.ToLower(tt.expected)) {
				t.Errorf("expected result to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestExtractReadmeMetadataBasic tests extracting metadata from README
func TestExtractReadmeMetadataBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")

	readme := `# My Project
**v1.2.3**

Author: John Doe

This is a test project.
`

	if err := os.WriteFile(path, []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	meta := ExtractReadmeMetadata(context.Background(), path)

	if meta.Title == "" {
		t.Error("expected title to be extracted")
	}
	if meta.Version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got %q", meta.Version)
	}
	if meta.Author != "John Doe" {
		t.Errorf("expected author 'John Doe', got %q", meta.Author)
	}
}

// TestExtractReadmeMetadataChineseAuthor tests Chinese author extraction
func TestExtractReadmeMetadataChineseAuthor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")

	readme := `# 我的项目
作者：张三

内容在这里。
`

	if err := os.WriteFile(path, []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	meta := ExtractReadmeMetadata(context.Background(), path)

	if meta.Author != "张三" {
		t.Errorf("expected author '张三', got %q", meta.Author)
	}
}

// TestExtractReadmeMetadataGitHub tests GitHub username extraction
func TestExtractReadmeMetadataGitHub(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")

	readme := `# Project
GitHub: https://github.com/johndoe/myproject

Content here.
`

	if err := os.WriteFile(path, []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	meta := ExtractReadmeMetadata(context.Background(), path)

	if meta.Author != "johndoe" {
		t.Errorf("expected author 'johndoe' from GitHub, got %q", meta.Author)
	}
}

// TestExtractReadmeMetadataLanguageDetection tests language detection in metadata
func TestExtractReadmeMetadataLanguageDetection(t *testing.T) {
	tests := []struct {
		name             string
		readme           string
		expectedLanguage string
	}{
		{
			name:             "English README",
			readme:           "# My Project\nThis is an English project with lots of English text.",
			expectedLanguage: "en-US",
		},
		{
			name:             "Chinese README",
			readme:           "# 我的项目\n这是一个中文项目，包含很多中文内容。",
			expectedLanguage: "zh-CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "README.md")

			if err := os.WriteFile(path, []byte(tt.readme), 0o644); err != nil {
				t.Fatalf("write README.md failed: %v", err)
			}

			meta := ExtractReadmeMetadata(context.Background(), path)

			if meta.Language != tt.expectedLanguage {
				t.Errorf("expected language %q, got %q", tt.expectedLanguage, meta.Language)
			}
		})
	}
}

// TestExtractReadmeMetadataNotFound tests ExtractReadmeMetadata with missing file
func TestExtractReadmeMetadataNotFound(t *testing.T) {
	meta := ExtractReadmeMetadata(context.Background(), "/nonexistent/README.md")

	// Should return empty metadata, not error
	if meta.Title != "" || meta.Author != "" || meta.Version != "" {
		t.Error("expected empty metadata for non-existent file")
	}
}

// TestDiscoverLexicalOrdering tests that chapters are in lexical order
func TestDiscoverLexicalOrdering(t *testing.T) {
	dir := t.TempDir()

	// Create files with names that test lexical ordering
	files := map[string]string{
		"z_last.md":   "# Z Last\nContent",
		"a_first.md":  "# A First\nContent",
		"m_middle.md": "# M Middle\nContent",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(cfg.Chapters) != 3 {
		t.Errorf("expected 3 chapters, got %d", len(cfg.Chapters))
	}

	// Check lexical ordering
	expected := []string{"A First", "M Middle", "Z Last"}
	for i, exp := range expected {
		if cfg.Chapters[i].Title != exp {
			t.Errorf("chapter %d: expected %q, got %q", i, exp, cfg.Chapters[i].Title)
		}
	}
}

// TestDiscoverAbsolutePathHandling tests Discover with relative paths
func TestDiscoverAbsolutePathHandling(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Ch1\nContent"), 0o644); err != nil {
		t.Fatalf("write ch1.md failed: %v", err)
	}

	// Use relative path
	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover with relative path failed: %v", err)
	}

	if cfg == nil {
		t.Error("expected non-nil config")
	}
}

// TestFileNameToTitle tests converting filename to readable title
func TestFileNameToTitle(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"chapter_one.md", "Chapter one"},
		{"chapter-two.md", "Chapter two"},
		{"01_intro.md", "01 intro"},
		{"README.md", "README"},
		{"my_test_file.md", "My test file"},
		{"a.md", "A"},
	}

	for _, tt := range tests {
		result := fileNameToTitle(tt.filename)
		if result != tt.expected {
			t.Errorf("fileNameToTitle(%q) = %q, want %q", tt.filename, result, tt.expected)
		}
	}
}

// TestDiscoverErrorString tests DiscoverError string representation
func TestDiscoverErrorString(t *testing.T) {
	err := &DiscoverError{
		Dir: "/path/to/dir",
		Msg: "no .md files found in directory",
	}

	errorStr := err.Error()
	if !strings.Contains(errorStr, "no .md files found in directory") {
		t.Errorf("error string should contain message: %s", errorStr)
	}
	if !strings.Contains(errorStr, "/path/to/dir") {
		t.Errorf("error string should contain dir: %s", errorStr)
	}
}

// TestAutoDiscoverBookTitleFromFirstChapter tests fallback to first chapter title
func TestAutoDiscoverBookTitleFromFirstChapter(t *testing.T) {
	dir := t.TempDir()

	// Create files without README.md
	if err := os.WriteFile(filepath.Join(dir, "chapter1.md"), []byte("# First Chapter Title\nContent"), 0o644); err != nil {
		t.Fatalf("write chapter1.md failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "chapter2.md"), []byte("# Second Chapter\nContent"), 0o644); err != nil {
		t.Fatalf("write chapter2.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should use first chapter's title as book title when README is absent
	if cfg.Book.Title != "First Chapter Title" {
		t.Errorf("expected book title from first chapter, got %q", cfg.Book.Title)
	}
}

// TestLoadFromSummaryDetectsMetadata tests loadFromSummary metadata extraction
func TestLoadFromSummaryDetectsMetadata(t *testing.T) {
	dir := t.TempDir()

	// Create SUMMARY.md
	summary := `# My Book

* [Intro](intro.md)
* [Chapter 1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write SUMMARY.md failed: %v", err)
	}

	// Create README.md with metadata
	readme := `# README
**v2.0.0**
Author: Test Author

Description here.
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"intro.md", "ch1.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", file, err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", cfg.Book.Version)
	}
	if cfg.Book.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got %q", cfg.Book.Author)
	}
}

// TestLoadFromSummaryDetectsGlossary tests GLOSSARY.md detection in loadFromSummary path
func TestLoadFromSummaryDetectsGlossary(t *testing.T) {
	dir := t.TempDir()

	// Create SUMMARY.md
	summary := `# Summary

* [Ch1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write SUMMARY.md failed: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Chapter 1\nContent"), 0o644); err != nil {
		t.Fatalf("write ch1.md failed: %v", err)
	}

	// Create GLOSSARY.md
	if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("## API\nDef"), 0o644); err != nil {
		t.Fatalf("write GLOSSARY.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.GlossaryFile == "" {
		t.Error("should detect GLOSSARY.md")
	}
}

// TestLoadFromSummaryDetectsLangs tests LANGS.md detection in loadFromSummary path
func TestLoadFromSummaryDetectsLangs(t *testing.T) {
	dir := t.TempDir()

	// Create SUMMARY.md
	summary := `# Summary

* [Ch1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write SUMMARY.md failed: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Chapter 1\nContent"), 0o644); err != nil {
		t.Fatalf("write ch1.md failed: %v", err)
	}

	// Create LANGS.md
	if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte("* [English](en/)\n"), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.LangsFile == "" {
		t.Error("should detect LANGS.md")
	}
}

// TestExtractReadmeMetadataMultilineVersion tests version extraction from bold text
func TestExtractReadmeMetadataMultilineVersion(t *testing.T) {
	tests := []struct {
		name            string
		readme          string
		expectedVersion string
	}{
		{
			name:            "version with v prefix",
			readme:          "# Project\n**v1.2.3**\nContent",
			expectedVersion: "1.2.3",
		},
		{
			name:            "version without v prefix",
			readme:          "# Project\n**2.0.0**\nContent",
			expectedVersion: "2.0.0",
		},
		{
			name:            "two-digit version",
			readme:          "# Project\n**v1.5**\nContent",
			expectedVersion: "1.5",
		},
		{
			name:            "no version",
			readme:          "# Project\nNo version here",
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "README.md")

			if err := os.WriteFile(path, []byte(tt.readme), 0o644); err != nil {
				t.Fatalf("write README.md failed: %v", err)
			}

			meta := ExtractReadmeMetadata(context.Background(), path)

			if meta.Version != tt.expectedVersion {
				t.Errorf("expected version %q, got %q", tt.expectedVersion, meta.Version)
			}
		})
	}
}
