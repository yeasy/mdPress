package cmd

import (
	"bytes"
	"context"
	"io"
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
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(testBookYAML), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte(testChapterContent), 0o644); err != nil {
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
	defer suppressOutput(t)()
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	if err := executeValidate(context.Background(), tmpDir); err != nil {
		t.Errorf("valid project should pass validation: %v", err)
	}
}

func TestExecuteValidate_MissingChapterFile(t *testing.T) {
	defer suppressOutput(t)()
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
chapters:
  - title: "Missing Chapter"
    file: "nonexistent.md"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	if err := executeValidate(context.Background(), tmpDir); err == nil {
		t.Error("project with missing chapter file should fail validation")
	}
}

func TestExecuteValidate_MissingTitle(t *testing.T) {
	defer suppressOutput(t)()
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
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte(testChapterContent), 0o644); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}

	if err := executeValidate(context.Background(), tmpDir); err != nil {
		t.Errorf("book without explicit title should pass (defaults to 'Untitled Book'): %v", err)
	}
}

func TestExecuteValidate_NoChapters(t *testing.T) {
	defer suppressOutput(t)()
	defer withValidateReportPath(t, "")()
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
chapters: []
`
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	if err := executeValidate(context.Background(), tmpDir); err == nil {
		t.Error("book without chapters should fail validation")
	}
}

func TestExecuteValidate_WithCoverImageMissing(t *testing.T) {
	defer suppressOutput(t)()
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
	if err := os.WriteFile(filepath.Join(tmpDir, "book.yaml"), []byte(bookYAML), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte(testChapterContent), 0o644); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}

	if err := executeValidate(context.Background(), tmpDir); err == nil {
		t.Error("book with missing cover image should fail validation")
	}
}

func TestExecuteValidate_WithJSONReport(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	reportPath := filepath.Join(tmpDir, "report.json")
	defer withValidateReportPath(t, reportPath)()

	if err := executeValidate(context.Background(), tmpDir); err != nil {
		t.Fatalf("executeValidate failed: %v", err)
	}

	if _, err := os.Stat(reportPath); err != nil {
		t.Errorf("JSON report should have been written: %v", err)
	}
}

func TestExecuteValidate_WithMDReport(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	reportPath := filepath.Join(tmpDir, "report.md")
	defer withValidateReportPath(t, reportPath)()

	if err := executeValidate(context.Background(), tmpDir); err != nil {
		t.Fatalf("executeValidate failed: %v", err)
	}

	if _, err := os.Stat(reportPath); err != nil {
		t.Errorf("MD report should have been written: %v", err)
	}
}

func TestFinalizeValidate_NoError(t *testing.T) {
	defer withValidateReportPath(t, "")()

	results := []validateResult{
		{OK: true, Message: "check 1 passed"},
		{OK: true, Message: "check 2 passed"},
	}
	if err := finalizeValidate(results, false); err != nil {
		t.Errorf("finalizeValidate with no error should return nil: %v", err)
	}
}

func TestFinalizeValidate_WithError(t *testing.T) {
	defer withValidateReportPath(t, "")()

	results := []validateResult{
		{OK: true, Message: "check passed"},
		{OK: false, Message: "check failed"},
	}
	err := finalizeValidate(results, true)
	if err == nil {
		t.Fatal("finalizeValidate with error should return non-nil error")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error should mention 'validation failed', got: %v", err)
	}
}

func TestFinalizeValidate_WritesJSONReport(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")
	defer withValidateReportPath(t, reportPath)()

	results := []validateResult{{OK: true, Message: "all good"}}
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

	results := []validateResult{{OK: false, Message: "failed check"}}
	err := finalizeValidate(results, true)
	if err == nil {
		t.Error("should still return error even in quiet mode")
	}
}

func TestPrintResults_Normal(t *testing.T) {
	results := []validateResult{
		{OK: true, Message: "passed"},
		{OK: false, Message: "failed"},
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	printResults(results)

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut) //nolint:errcheck
	io.Copy(&bufErr, rErr) //nolint:errcheck
	rOut.Close()
	rErr.Close()

	combined := bufOut.String() + bufErr.String()
	if combined == "" {
		t.Error("printResults produced no output for non-quiet mode")
	}
}

func TestPrintResults_QuietMode(t *testing.T) {
	oldQuiet := quiet
	quiet = true
	defer func() { quiet = oldQuiet }()

	results := []validateResult{
		{OK: true, Message: "passed"},
		{OK: false, Message: "failed"},
	}

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	printResults(results)

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut) //nolint:errcheck
	io.Copy(&bufErr, rErr) //nolint:errcheck
	rOut.Close()
	rErr.Close()

	if bufOut.Len() != 0 || bufErr.Len() != 0 {
		t.Errorf("printResults should produce no output in quiet mode, got stdout=%q stderr=%q", bufOut.String(), bufErr.String())
	}
}

func TestPrintResults_Empty(t *testing.T) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr

	printResults(nil)
	printResults([]validateResult{})

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut) //nolint:errcheck
	io.Copy(&bufErr, rErr) //nolint:errcheck
	rOut.Close()
	rErr.Close()

	combined := strings.TrimSpace(bufOut.String() + bufErr.String())
	if strings.Contains(combined, "passed") || strings.Contains(combined, "failed") {
		t.Errorf("empty results should not produce pass/fail messages, got: %q", combined)
	}
}

func TestValidateChapterContentAndSequence_ValidProject(t *testing.T) {
	tmpDir := t.TempDir()
	createTestProject(t, tmpDir)

	cfg, err := config.Load(filepath.Join(tmpDir, "book.yaml"))
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	issues, _, err := validateChapterContentAndSequence(cfg)
	if err != nil {
		t.Errorf("should not error for valid project: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for valid project, got %d: %v", len(issues), issues)
	}
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

	_, _, err := validateChapterContentAndSequence(cfg)
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

	issues, _, err := validateChapterContentAndSequence(cfg)
	if err != nil {
		t.Errorf("empty chapters should not error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("empty chapters should have no issues, got: %v", issues)
	}
}
