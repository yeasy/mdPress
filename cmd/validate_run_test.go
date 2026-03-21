package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

const testBookYAML = `book:
  title: "Test Book"
  author: "Test Author"
  version: "1.0.0"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
`

const testChapterContent = `# Chapter 1

This is a test chapter.
`

func createTestProject(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(testBookYAML), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte(testChapterContent), 0644); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}
}

func withValidateReportPath(t *testing.T, path string) func() {
	t.Helper()
	old := validateReportPath
	validateReportPath = path
	return func() { validateReportPath = old }
}

func TestExecuteValidate_ValidProject(t *testing.T) {
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	if err := executeValidate(tmpDir); err != nil {
		t.Errorf("valid project should pass validation: %v", err)
	}
}

func TestExecuteValidate_MissingChapterFile(t *testing.T) {
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
chapters:
  - title: "Missing Chapter"
    file: "nonexistent.md"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	if err := executeValidate(tmpDir); err == nil {
		t.Error("project with missing chapter file should fail validation")
	}
}

func TestExecuteValidate_MissingTitle(t *testing.T) {
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	// When title is omitted, config defaults to "Untitled Book",
	// so validation should still pass.
	bookYAML := `book:
  author: "Test Author"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte(testChapterContent), 0644); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}

	if err := executeValidate(tmpDir); err != nil {
		t.Errorf("book without explicit title should pass (defaults to 'Untitled Book'): %v", err)
	}
}

func TestExecuteValidate_NoChapters(t *testing.T) {
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
chapters: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	if err := executeValidate(tmpDir); err == nil {
		t.Error("book without chapters should fail validation")
	}
}

func TestExecuteValidate_WithCoverImageMissing(t *testing.T) {
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
  cover:
    image: "images/cover.png"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte(testChapterContent), 0644); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}

	if err := executeValidate(tmpDir); err == nil {
		t.Error("book with missing cover image should fail validation")
	}
}

func TestExecuteValidate_WithJSONReport(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	reportPath := filepath.Join(tmpDir, "report.json")
	defer withValidateReportPath(t, reportPath)()

	_ = executeValidate(tmpDir)

	if _, err := os.Stat(reportPath); err != nil {
		t.Errorf("JSON report should have been written: %v", err)
	}
}

func TestExecuteValidate_WithMDReport(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	reportPath := filepath.Join(tmpDir, "report.md")
	defer withValidateReportPath(t, reportPath)()

	_ = executeValidate(tmpDir)

	if _, err := os.Stat(reportPath); err != nil {
		t.Errorf("MD report should have been written: %v", err)
	}
}

func TestFinalizeValidate_NoError(t *testing.T) {
	defer withValidateReportPath(t, "")()

	results := []validateResult{
		{ok: true, message: "check 1 passed"},
		{ok: true, message: "check 2 passed"},
	}
	if err := finalizeValidate(results, false); err != nil {
		t.Errorf("finalizeValidate with no error should return nil: %v", err)
	}
}

func TestFinalizeValidate_WithError(t *testing.T) {
	defer withValidateReportPath(t, "")()

	results := []validateResult{
		{ok: true, message: "check passed"},
		{ok: false, message: "check failed"},
	}
	err := finalizeValidate(results, true)
	if err == nil {
		t.Error("finalizeValidate with error should return non-nil error")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error should mention 'validation failed', got: %v", err)
	}
}

func TestFinalizeValidate_WritesJSONReport(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")
	defer withValidateReportPath(t, reportPath)()

	results := []validateResult{{ok: true, message: "all good"}}
	if err := finalizeValidate(results, false); err != nil {
		t.Errorf("finalizeValidate should succeed: %v", err)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Errorf("report file should exist: %v", err)
	}
}

func TestFinalizeValidate_QuietMode(t *testing.T) {
	defer withValidateReportPath(t, "")()
	oldQuiet := quiet
	quiet = true
	defer func() { quiet = oldQuiet }()

	results := []validateResult{{ok: false, message: "failed check"}}
	err := finalizeValidate(results, true)
	if err == nil {
		t.Error("should still return error even in quiet mode")
	}
}

func TestPrintResults_Normal(t *testing.T) {
	results := []validateResult{
		{ok: true, message: "passed"},
		{ok: false, message: "failed"},
	}
	// Just verify no panic
	printResults(results)
}

func TestPrintResults_QuietMode(t *testing.T) {
	oldQuiet := quiet
	quiet = true
	defer func() { quiet = oldQuiet }()

	results := []validateResult{
		{ok: true, message: "passed"},
		{ok: false, message: "failed"},
	}
	// Should return immediately without printing
	printResults(results)
}

func TestPrintResults_Empty(t *testing.T) {
	printResults(nil)
	printResults([]validateResult{})
}

func TestValidateChapterContentAndSequence_ValidProject(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	cfg, err := config.Load(filepath.Join(tmpDir, "book.yaml"))
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	issues, err := validateChapterContentAndSequence(cfg)
	if err != nil {
		t.Errorf("should not error for valid project: %v", err)
	}
	_ = issues
}

func TestValidateChapterContentAndSequence_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Build config manually to bypass config.Load validation
	// which rejects missing chapter files.
	cfg := &config.BookConfig{}
	cfg.SetBaseDir(tmpDir)
	cfg.Book.Title = "Test"
	cfg.Chapters = []config.ChapterDef{
		{Title: "Chapter 1", File: "missing.md"},
	}

	_, err := validateChapterContentAndSequence(cfg)
	if err == nil {
		t.Error("should error when chapter file is missing")
	}
}

func TestValidateChapterContentAndSequence_EmptyChapters(t *testing.T) {
	// Build config manually to bypass config.Load validation
	// which rejects empty chapter lists.
	cfg := &config.BookConfig{}
	cfg.Book.Title = "Test"
	cfg.Chapters = []config.ChapterDef{}

	issues, err := validateChapterContentAndSequence(cfg)
	if err != nil {
		t.Errorf("empty chapters should not error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("empty chapters should have no issues, got: %v", issues)
	}
}
