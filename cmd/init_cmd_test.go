package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/pkg/utils"
)

// extractTitleFromFile is a test wrapper for utils.ExtractTitleFromFile.
// This allows tests to call the function without the package prefix.
func extractTitleFromFile(path string) string {
	return utils.ExtractTitleFromFile(path)
}

// TestInferTitleFromPathBasic tests basic title inference from file paths.
func TestInferTitleFromPathBasic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "chapter01/README.md",
			expected: "Chapter01",
		},
		{
			input:    "preface.md",
			expected: "Preface",
		},
		{
			input:    "part1/intro.md",
			expected: "Part1 - intro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferTitleFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("inferTitleFromPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestInferTitleFromPathWithMultipleDirectories tests paths with nested directories.
func TestInferTitleFromPathWithMultipleDirectories(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "part1/chapter1/section1.md",
			expected: "Part1 - chapter1 - section1",
		},
		{
			input:    "docs/advanced/networking.md",
			expected: "Docs - advanced - networking",
		},
		{
			input:    "a/b/c/d.md",
			expected: "A - b - c - d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferTitleFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("inferTitleFromPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestInferTitleFromPathREADMEHandling tests README file naming special cases.
func TestInferTitleFromPathREADMEHandling(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "README.md",
			expected: "README",
		},
		{
			input:    "section/README.md",
			expected: "Section",
		},
		{
			input:    "part1/chapter2/README.MD",
			expected: "Part1 - chapter2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferTitleFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("inferTitleFromPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestInferTitleFromPathMixedCase tests case handling.
func TestInferTitleFromPathMixedCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "MyProject.md",
			expected: "MyProject",
		},
		{
			input:    "mYfILE.md",
			expected: "MYfILE",
		},
		{
			input:    "chapter/INTRO.md",
			expected: "Chapter - INTRO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferTitleFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("inferTitleFromPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestExtractTitleFromFileBasic tests H1 heading extraction.
func TestExtractTitleFromFileBasic(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		filename string
		content  string
		expected string
	}{
		{
			filename: "simple.md",
			content:  "# Hello World\nSome content",
			expected: "Hello World",
		},
		{
			filename: "with_spaces.md",
			content:  "#   Padded Title   \nContent",
			expected: "Padded Title",
		},
		{
			filename: "no_heading.md",
			content:  "## Second level\nNo H1 here",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result := extractTitleFromFile(path)
			if result != tt.expected {
				t.Errorf("extractTitleFromFile(%q) = %q, want %q", path, result, tt.expected)
			}
		})
	}
}

// TestExtractTitleFromFileWithMultipleHeadings tests that only the first H1 is extracted.
func TestExtractTitleFromFileWithMultipleHeadings(t *testing.T) {
	tmpDir := t.TempDir()

	content := `# First Heading
Some content

# Second Heading
More content`

	path := filepath.Join(tmpDir, "multi.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := extractTitleFromFile(path)
	expected := "First Heading"
	if result != expected {
		t.Errorf("extractTitleFromFile() = %q, want %q", result, expected)
	}
}

// TestExtractTitleFromFileEmptyFile tests extraction from empty files.
func TestExtractTitleFromFileEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.md")

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := extractTitleFromFile(path)
	if result != "" {
		t.Errorf("extractTitleFromFile() on empty file = %q, want empty string", result)
	}
}

// TestExtractTitleFromFileNonexistent tests error handling for missing files.
func TestExtractTitleFromFileNonexistent(t *testing.T) {
	result := extractTitleFromFile("/nonexistent/path/file.md")
	if result != "" {
		t.Errorf("extractTitleFromFile() on missing file = %q, want empty string", result)
	}
}

// TestExtractTitleFromFileFirstFiftyLines tests the 50-line limit.
func TestExtractTitleFromFileFirstFiftyLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Create content with H1 on line 51 (should not be extracted)
	lines := make([]string, 52)
	for i := 0; i < 50; i++ {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	lines[50] = "# Should Not Extract"
	lines[51] = "Content after line 50"

	content := strings.Join(lines, "\n")
	path := filepath.Join(tmpDir, "fifty.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := extractTitleFromFile(path)
	if result != "" {
		t.Errorf("extractTitleFromFile() = %q, want empty (H1 beyond line 50)", result)
	}
}

// TestExtractTitleFromFileAtLineLimit tests H1 exactly at line 50.
func TestExtractTitleFromFileAtLineLimit(t *testing.T) {
	tmpDir := t.TempDir()

	lines := make([]string, 50)
	for i := 0; i < 49; i++ {
		lines[i] = fmt.Sprintf("Line %d", i+1)
	}
	lines[49] = "# At Line 50"

	content := strings.Join(lines, "\n")
	path := filepath.Join(tmpDir, "at_limit.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := extractTitleFromFile(path)
	expected := "At Line 50"
	if result != expected {
		t.Errorf("extractTitleFromFile() = %q, want %q", result, expected)
	}
}

// TestScanMarkdownFilesBasic tests directory scanning for Markdown files.
func TestScanMarkdownFilesBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.md"), []byte("# File 1"), 0o644); err != nil {
		t.Fatalf("failed to create file1.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.md"), []byte("# File 2"), 0o644); err != nil {
		t.Fatalf("failed to create file2.md: %v", err)
	}

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("scanMarkdownFiles() found %d files, want 2", len(files))
	}
}

// TestScanMarkdownFilesIgnoresNonMarkdown tests that non-Markdown files are skipped.
func TestScanMarkdownFilesIgnoresNonMarkdown(t *testing.T) {
	tmpDir := t.TempDir()

	// Create various files
	os.WriteFile(filepath.Join(tmpDir, "file1.md"), []byte("# File 1"), 0o644)   //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("Text file"), 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "file3.json"), []byte("{}"), 0o644)       //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("scanMarkdownFiles() found %d files, want 1", len(files))
	}
	if len(files) > 0 && files[0].RelPath != "file1.md" {
		t.Errorf("scanMarkdownFiles() found %q, want file1.md", files[0].RelPath)
	}
}

// TestScanMarkdownFilesNestedDirectories tests scanning nested directories.
func TestScanMarkdownFilesNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "part1", "chapter1"), 0o755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	os.WriteFile(filepath.Join(tmpDir, "intro.md"), []byte("# Intro"), 0o644)                          //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "part1", "overview.md"), []byte("# Overview"), 0o644)           //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "part1", "chapter1", "content.md"), []byte("# Content"), 0o644) //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("scanMarkdownFiles() found %d files, want 3", len(files))
	}

	// Check sorting: should be by depth first
	expectedPaths := []string{"intro.md", "part1/overview.md", "part1/chapter1/content.md"}
	for i, expected := range expectedPaths {
		if i < len(files) && files[i].RelPath != expected {
			t.Errorf("files[%d].RelPath = %q, want %q", i, files[i].RelPath, expected)
		}
	}
}

// TestScanMarkdownFilesSkipsHiddenDirs tests that hidden directories are skipped.
func TestScanMarkdownFilesSkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create visible and hidden directories
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)                                     //nolint:errcheck
	os.MkdirAll(filepath.Join(tmpDir, "visible"), 0o755)                                  //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, ".git", "file.md"), []byte("# Git"), 0o644)        //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "visible", "file.md"), []byte("# Visible"), 0o644) //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("scanMarkdownFiles() found %d files, want 1 (hidden dirs should be skipped)", len(files))
	}
}

// TestScanMarkdownFilesSkipsDependencyDirs tests that vendor and node_modules are skipped.
func TestScanMarkdownFilesSkipsDependencyDirs(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "vendor"), 0o755)                                            //nolint:errcheck
	os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0o755)                                      //nolint:errcheck
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0o755)                                               //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "vendor", "file.md"), []byte("# Vendor"), 0o644)            //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "node_modules", "file.md"), []byte("# NodeModules"), 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "src", "file.md"), []byte("# Source"), 0o644)               //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("scanMarkdownFiles() found %d files, want 1", len(files))
	}
	if len(files) > 0 && files[0].RelPath != "src/file.md" {
		t.Errorf("scanMarkdownFiles() = %q, want src/file.md", files[0].RelPath)
	}
}

// TestScanMarkdownFilesSkipsTopLevelREADME tests that top-level README.md is skipped.
func TestScanMarkdownFilesSkipsTopLevelREADME(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Top Level"), 0o644)          //nolint:errcheck
	os.MkdirAll(filepath.Join(tmpDir, "chapter"), 0o755)                                    //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "chapter", "README.md"), []byte("# Chapter"), 0o644) //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("scanMarkdownFiles() found %d files, want 1", len(files))
	}
	if len(files) > 0 && files[0].RelPath != "chapter/README.md" {
		t.Errorf("scanMarkdownFiles() = %q, want chapter/README.md", files[0].RelPath)
	}
}

// TestScanMarkdownFilesSortingByDepth tests that files are sorted by depth first.
func TestScanMarkdownFilesSortingByDepth(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files at different depths (intentionally out of order)
	os.MkdirAll(filepath.Join(tmpDir, "a", "b", "c"), 0o755)                               //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "a", "b", "c", "deep.md"), []byte("# Deep"), 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "a", "shallow.md"), []byte("# Shallow"), 0o644)     //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "root.md"), []byte("# Root"), 0o644)                //nolint:errcheck

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("scanMarkdownFiles() found %d files, want 3", len(files))
	}

	expectedOrder := []string{"root.md", "a/shallow.md", "a/b/c/deep.md"}
	for i, expected := range expectedOrder {
		if i < len(files) && files[i].RelPath != expected {
			t.Errorf("files[%d].RelPath = %q, want %q", i, files[i].RelPath, expected)
		}
	}
}

// TestDetectCoverImageBasic tests cover image detection.
func TestDetectCoverImageBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that PNG is found
	os.WriteFile(filepath.Join(tmpDir, "cover.png"), []byte{}, 0o644) //nolint:errcheck

	result := detectCoverImage(tmpDir)
	if result != "cover.png" {
		t.Errorf("detectCoverImage() = %q, want cover.png", result)
	}
}

// TestDetectCoverImageMultipleCandidates tests priority when multiple covers exist.
func TestDetectCoverImageMultipleCandidates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple candidates (function should return first found)
	os.WriteFile(filepath.Join(tmpDir, "cover.png"), []byte{}, 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "cover.jpg"), []byte{}, 0o644) //nolint:errcheck
	os.WriteFile(filepath.Join(tmpDir, "Cover.png"), []byte{}, 0o644) //nolint:errcheck

	result := detectCoverImage(tmpDir)
	// Should find one of the candidates
	candidates := map[string]bool{"cover.png": true, "cover.jpg": true, "Cover.png": true}
	if !candidates[result] {
		t.Errorf("detectCoverImage() = %q, want one of cover.png, cover.jpg, Cover.png", result)
	}
}

// TestDetectCoverImageNone tests when no cover exists.
func TestDetectCoverImageNone(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create any cover files
	result := detectCoverImage(tmpDir)
	if result != "" {
		t.Errorf("detectCoverImage() = %q, want empty string when no cover found", result)
	}
}

// TestDetectCoverImageVariants tests different image formats.
func TestDetectCoverImageVariants(t *testing.T) {
	tests := []string{"cover.png", "cover.jpg", "cover.jpeg", "cover.svg", "Cover.png", "Cover.jpg", "Cover.jpeg"}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.WriteFile(filepath.Join(tmpDir, filename), []byte{}, 0o644) //nolint:errcheck

			result := detectCoverImage(tmpDir)
			if result != filename {
				t.Errorf("detectCoverImage() = %q, want %q", result, filename)
			}
		})
	}
}

// TestGenerateInteractiveBookYAML tests interactive YAML generation.
func TestGenerateInteractiveBookYAML(t *testing.T) {
	answers := initAnswers{
		Title:    "My Interactive Book",
		Author:   "John Doe",
		Language: "zh-CN",
		Theme:    "elegant",
	}

	yaml := generateInteractiveBookYAML(answers)

	requiredStrings := []string{
		"title: \"My Interactive Book\"",
		"author: \"John Doe\"",
		"language: \"zh-CN\"",
		"theme: \"elegant\"",
		"preface.md",
		"chapter01/README.md",
	}

	for _, req := range requiredStrings {
		if !strings.Contains(yaml, req) {
			t.Errorf("generateInteractiveBookYAML() missing %q", req)
		}
	}
}

// TestCountChapterDefsSimple tests counting flat chapter definitions.
func TestCountChapterDefsSimple(t *testing.T) {
	chapters := []config.ChapterDef{
		{Title: "Chapter 1", Sections: []config.ChapterDef{}},
		{Title: "Chapter 2", Sections: []config.ChapterDef{}},
	}

	count := countChapterDefs(chapters)
	if count != 2 {
		t.Errorf("countChapterDefs() = %d, want 2", count)
	}
}

// TestCountChapterDefsNested tests counting nested chapter definitions.
func TestCountChapterDefsNested(t *testing.T) {
	chapters := []config.ChapterDef{
		{
			Title: "Chapter 1",
			Sections: []config.ChapterDef{
				{Title: "Section 1.1", Sections: []config.ChapterDef{}},
				{Title: "Section 1.2", Sections: []config.ChapterDef{}},
			},
		},
		{Title: "Chapter 2", Sections: []config.ChapterDef{}},
	}

	count := countChapterDefs(chapters)
	// 2 top-level + 2 sections under chapter 1 = 4
	if count != 4 {
		t.Errorf("countChapterDefs() = %d, want 4", count)
	}
}

// TestCountChapterDefsDeepNesting tests deeply nested chapter definitions.
func TestCountChapterDefsDeepNesting(t *testing.T) {
	chapters := []config.ChapterDef{
		{
			Title: "Part 1",
			Sections: []config.ChapterDef{
				{
					Title: "Chapter 1",
					Sections: []config.ChapterDef{
						{Title: "Section 1.1.1", Sections: []config.ChapterDef{}},
					},
				},
			},
		},
	}

	count := countChapterDefs(chapters)
	// 1 part + 1 chapter + 1 section = 3
	if count != 3 {
		t.Errorf("countChapterDefs() = %d, want 3", count)
	}
}

// TestCountChapterDefsEmpty tests counting empty definitions.
func TestCountChapterDefsEmpty(t *testing.T) {
	chapters := []config.ChapterDef{}

	count := countChapterDefs(chapters)
	if count != 0 {
		t.Errorf("countChapterDefs() = %d, want 0", count)
	}
}

// TestCreateStarterTemplateBasic tests basic template creation.
func TestCreateStarterTemplateBasic(t *testing.T) {
	tmpDir := t.TempDir()

	err := createStarterTemplate(tmpDir)
	if err != nil {
		t.Fatalf("createStarterTemplate() failed: %v", err)
	}

	// Check that required files were created
	prefacePath := filepath.Join(tmpDir, "preface.md")
	if _, err := os.Stat(prefacePath); os.IsNotExist(err) {
		t.Error("createStarterTemplate() did not create preface.md")
	}

	ch01ReadmePath := filepath.Join(tmpDir, "chapter01", "README.md")
	if _, err := os.Stat(ch01ReadmePath); os.IsNotExist(err) {
		t.Error("createStarterTemplate() did not create chapter01/README.md")
	}
}

// TestCreateStarterTemplateContent tests the content of created files.
func TestCreateStarterTemplateContent(t *testing.T) {
	tmpDir := t.TempDir()

	err := createStarterTemplate(tmpDir)
	if err != nil {
		t.Fatalf("createStarterTemplate() failed: %v", err)
	}

	prefaceContent, err := os.ReadFile(filepath.Join(tmpDir, "preface.md"))
	if err != nil {
		t.Fatalf("failed to read preface.md: %v", err)
	}

	if !strings.Contains(string(prefaceContent), "# Preface") {
		t.Error("preface.md should contain '# Preface' heading")
	}

	ch01Content, err := os.ReadFile(filepath.Join(tmpDir, "chapter01", "README.md"))
	if err != nil {
		t.Fatalf("failed to read chapter01/README.md: %v", err)
	}

	if !strings.Contains(string(ch01Content), "# Chapter 1") {
		t.Error("chapter01/README.md should contain '# Chapter 1' heading")
	}
	if !strings.Contains(string(ch01Content), "Hello, mdpress!") {
		t.Error("chapter01/README.md should contain example code")
	}
}

// TestCreateStarterTemplateDirectoryExists tests template creation in existing directory.
func TestCreateStarterTemplateDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory and add a file to it
	if err := os.WriteFile(filepath.Join(tmpDir, "existing.md"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	err := createStarterTemplate(tmpDir)
	if err != nil {
		t.Fatalf("createStarterTemplate() failed: %v", err)
	}

	// Original file should still exist
	if _, err := os.Stat(filepath.Join(tmpDir, "existing.md")); os.IsNotExist(err) {
		t.Error("existing file was removed")
	}

	// New files should be created
	if _, err := os.Stat(filepath.Join(tmpDir, "preface.md")); os.IsNotExist(err) {
		t.Error("preface.md was not created")
	}
}

// TestIsTerminalInteractive tests terminal detection does not panic.
func TestIsTerminalInteractive(t *testing.T) {
	// isTerminalInteractive checks if stdin is a real terminal.
	// In test environments this typically returns false.
	result := isTerminalInteractive()
	t.Logf("isTerminalInteractive() = %v", result)
	// In CI/test environments, stdin is not a terminal
	if result {
		t.Log("Running in a real terminal environment")
	}
}

// TestInferTitleFromPathEdgeCases tests edge cases in title inference.
func TestInferTitleFromPathEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    ".md",
			expected: "",
		},
		{
			input:    "a.md",
			expected: "A",
		},
		{
			input:    "README.MD",
			expected: "README",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := inferTitleFromPath(tt.input)
			if result != tt.expected {
				t.Errorf("inferTitleFromPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestScanMarkdownFilesEmpty tests scanning empty directory.
func TestScanMarkdownFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := scanMarkdownFiles(tmpDir)
	if err != nil {
		t.Fatalf("scanMarkdownFiles() failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("scanMarkdownFiles() on empty dir = %d files, want 0", len(files))
	}
}

// TestExtractTitleFromFileWithSpaces tests title extraction with special spacing.
func TestExtractTitleFromFileWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()

	content := `#  Title With Multiple  Spaces
More content`

	path := filepath.Join(tmpDir, "spaces.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := extractTitleFromFile(path)
	expected := "Title With Multiple  Spaces"
	if result != expected {
		t.Errorf("extractTitleFromFile() = %q, want %q", result, expected)
	}
}

// TestPromptUserBasic tests user prompt with default fallback (captured via buffer).
func TestPromptUserBasic(t *testing.T) {
	input := bytes.NewBufferString("\n")
	reader := bufio.NewReader(input)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	result := promptUser(reader, "Enter something", "defaultValue")

	w.Close()
	os.Stdout = oldStdout
	_, _ = io.ReadAll(r) // drain pipe

	if result != "defaultValue" {
		t.Errorf("promptUser() with empty input = %q, want %q", result, "defaultValue")
	}
}

// TestPromptChoiceBasic tests choice selection with default (captured via buffer).
func TestPromptChoiceBasic(t *testing.T) {
	input := bytes.NewBufferString("\n")
	reader := bufio.NewReader(input)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	options := []string{"Option1", "Option2", "Option3"}
	result := promptChoice(reader, "Pick one", options, 1)

	w.Close()
	os.Stdout = oldStdout
	_, _ = io.ReadAll(r) // drain pipe

	if result != "Option2" {
		t.Errorf("promptChoice() with empty input = %q, want Option2", result)
	}
}
