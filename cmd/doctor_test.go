package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestDoctorEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor on empty directory should not error, got: %v", err)
	}
}

func TestDoctorWithBookYAML(t *testing.T) {
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test Book"
  author: "Test Author"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
`
	bookPath := filepath.Join(tmpDir, "book.yaml")
	if err := os.WriteFile(bookPath, []byte(bookYAML), 0644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create the chapter file so config can load successfully
	chPath := filepath.Join(tmpDir, "ch1.md")
	if err := os.WriteFile(chPath, []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor with book.yaml should not error, got: %v", err)
	}
}

func TestDoctorWithSummaryMD(t *testing.T) {
	tmpDir := t.TempDir()

	summaryContent := `# Summary

- [Introduction](README.md)
- [Chapter 1](ch01.md)
`
	summaryPath := filepath.Join(tmpDir, "SUMMARY.md")
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0644); err != nil {
		t.Fatalf("failed to write SUMMARY.md: %v", err)
	}

	// Create the referenced files
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Intro"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor with SUMMARY.md should not error, got: %v", err)
	}
}

func TestDoctorNonExistentDir(t *testing.T) {
	nonExistentDir := "/this/path/should/not/exist/ever"

	err := executeDoctor(nonExistentDir)
	if err == nil {
		t.Error("executeDoctor on non-existent directory should return an error")
	}
}

func TestDoctorWithValidBookConfig(t *testing.T) {
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Sample Book"
  author: "John Doe"
  version: "1.0.0"
chapters:
  - title: "Preface"
    file: "preface.md"
  - title: "Chapter 1"
    file: "ch1.md"
`
	bookPath := filepath.Join(tmpDir, "book.yaml")
	if err := os.WriteFile(bookPath, []byte(bookYAML), 0644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "preface.md"), []byte("# Preface"), 0644); err != nil {
		t.Fatalf("failed to write preface.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("failed to write ch1.md: %v", err)
	}

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor with valid config should not error, got: %v", err)
	}

	// Verify config can be loaded
	cfg, err := config.Load(bookPath)
	if err != nil {
		t.Fatalf("config.Load should succeed: %v", err)
	}
	if cfg.Book.Title != "Sample Book" {
		t.Errorf("expected title 'Sample Book', got %q", cfg.Book.Title)
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(cfg.Chapters))
	}
}

func TestDoctorWithLangsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create LANGS.md
	langsContent := `# Languages

- [English](en/)
- [中文](zh/)
`
	langsPath := filepath.Join(tmpDir, "LANGS.md")
	if err := os.WriteFile(langsPath, []byte(langsContent), 0644); err != nil {
		t.Fatalf("failed to write LANGS.md: %v", err)
	}

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor with LANGS.md should not error, got: %v", err)
	}
}

func TestDoctorReportPath(t *testing.T) {
	tmpDir := t.TempDir()

	bookYAML := `book:
  title: "Test"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	bookPath := filepath.Join(tmpDir, "book.yaml")
	if err := os.WriteFile(bookPath, []byte(bookYAML), 0644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	// Test JSON report generation
	reportPath := filepath.Join(tmpDir, "report.json")
	doctorReportPath = reportPath

	err := executeDoctor(tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor should not error: %v", err)
	}

	// Verify report file was created
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report file should exist: %v", err)
	}

	// Reset doctorReportPath after test
	doctorReportPath = ""
}

func TestDoctorReportsCacheStatus(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("MDPRESS_CACHE_DIR", filepath.Join(tmpDir, "cache"))
	t.Setenv("MDPRESS_DISABLE_CACHE", "1")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := executeDoctor(tmpDir)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("executeDoctor should not error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()
	if !strings.Contains(output, "Runtime cache is disabled") {
		t.Fatalf("doctor 输出应包含 cache 状态，实际: %s", output)
	}
}
