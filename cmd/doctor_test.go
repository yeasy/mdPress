package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestDoctorEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	err := executeDoctor(context.Background(), tmpDir)
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

	err := executeDoctor(context.Background(), tmpDir)
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

	err := executeDoctor(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("executeDoctor with SUMMARY.md should not error, got: %v", err)
	}
}

func TestDoctorNonExistentDir(t *testing.T) {
	nonExistentDir := "/this/path/should/not/exist/ever"

	err := executeDoctor(context.Background(), nonExistentDir)
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

	err := executeDoctor(context.Background(), tmpDir)
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

	err := executeDoctor(context.Background(), tmpDir)
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

	err := executeDoctor(context.Background(), tmpDir)
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

	err := executeDoctor(context.Background(), tmpDir)

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

// TestDoctorCmdStructure tests that doctorCmd is properly configured
func TestDoctorCmdStructure(t *testing.T) {
	if doctorCmd == nil {
		t.Fatal("doctorCmd should not be nil")
	}
	if doctorCmd.Use != "doctor [directory]" {
		t.Errorf("doctorCmd.Use expected 'doctor [directory]', got %q", doctorCmd.Use)
	}
	if doctorCmd.Short == "" {
		t.Error("doctorCmd.Short should not be empty")
	}
	if doctorCmd.Long == "" {
		t.Error("doctorCmd.Long should not be empty")
	}
}

// TestDoctorCmdFlags tests that doctor command has correct flags
func TestDoctorCmdFlags(t *testing.T) {
	if doctorCmd == nil {
		t.Fatal("doctorCmd should not be nil")
	}
	reportFlag := doctorCmd.Flags().Lookup("report")
	if reportFlag == nil {
		t.Error("doctorCmd should have 'report' flag")
	}
	if reportFlag != nil && reportFlag.Usage == "" {
		t.Error("report flag should have usage description")
	}
}

// TestSearchPlantUMLInDir_NoPlantUML tests searchPlantUMLInDir when no plantuml blocks exist
func TestSearchPlantUMLInDir_NoPlantUML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create markdown files without plantuml blocks
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("# Test\n\nNo diagrams here"), 0644); err != nil {
		t.Fatalf("failed to write test.md: %v", err)
	}

	result := searchPlantUMLInDir(tmpDir)
	if result {
		t.Error("searchPlantUMLInDir should return false when no plantuml blocks exist")
	}
}

// TestSearchPlantUMLInDir_WithPlantUML tests searchPlantUMLInDir when plantuml blocks exist
func TestSearchPlantUMLInDir_WithPlantUML(t *testing.T) {
	tmpDir := t.TempDir()

	content := `# Diagram Test

` + "```" + `plantuml
@startuml
Actor -> Server: Request
Server -> Actor: Response
@enduml
` + "```" + `

Done.`

	if err := os.WriteFile(filepath.Join(tmpDir, "diagram.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write diagram.md: %v", err)
	}

	result := searchPlantUMLInDir(tmpDir)
	if !result {
		t.Error("searchPlantUMLInDir should return true when plantuml blocks exist")
	}
}

// TestSearchPlantUMLInDir_Nested tests searchPlantUMLInDir in nested directories
func TestSearchPlantUMLInDir_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	nestedDir := filepath.Join(tmpDir, "docs", "diagrams")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	// Create a file with plantuml block deep in the tree
	content := "# Architecture\n\n" + "```" + "plantuml\n@startuml\nComponent A\n@enduml\n" + "```"
	if err := os.WriteFile(filepath.Join(nestedDir, "arch.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write arch.md: %v", err)
	}

	result := searchPlantUMLInDir(tmpDir)
	if !result {
		t.Error("searchPlantUMLInDir should find plantuml blocks in nested directories")
	}
}

// TestSearchPlantUMLInDir_SkipsHidden tests that searchPlantUMLInDir skips hidden directories
func TestSearchPlantUMLInDir_SkipsHidden(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hidden directory with plantuml block
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("failed to create hidden directory: %v", err)
	}

	content := "```" + "plantuml\ndiagram\n" + "```"
	if err := os.WriteFile(filepath.Join(hiddenDir, "secret.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write secret.md: %v", err)
	}

	result := searchPlantUMLInDir(tmpDir)
	if result {
		t.Error("searchPlantUMLInDir should skip hidden directories")
	}
}

// TestSearchPlantUMLInDir_SkipsNodeModules tests that searchPlantUMLInDir skips node_modules
func TestSearchPlantUMLInDir_SkipsNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules directory with plantuml block
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("failed to create node_modules directory: %v", err)
	}

	content := "```" + "plantuml\ndiagram\n" + "```"
	if err := os.WriteFile(filepath.Join(nodeModulesDir, "package.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write package.md: %v", err)
	}

	result := searchPlantUMLInDir(tmpDir)
	if result {
		t.Error("searchPlantUMLInDir should skip node_modules")
	}
}

// TestHasPlantUMLBlocks tests hasPlantUMLBlocks wrapper
func TestHasPlantUMLBlocks(t *testing.T) {
	tmpDir := t.TempDir()

	// Initially no plantuml blocks
	if hasPlantUMLBlocks(tmpDir) {
		t.Error("hasPlantUMLBlocks should return false for empty directory")
	}

	// Add plantuml block
	content := "```" + "plantuml\n@startuml\n@enduml\n" + "```"
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test.md: %v", err)
	}

	if !hasPlantUMLBlocks(tmpDir) {
		t.Error("hasPlantUMLBlocks should return true after adding plantuml block")
	}
}

// TestRenderDoctorMarkdown tests markdown rendering of doctor report
func TestRenderDoctorMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		report  doctorReport
		wantStr []string
	}{
		{
			name: "basic_report",
			report: doctorReport{
				Platform:          "linux/amd64",
				GoVersion:         "go1.24.2",
				ChromiumAvailable: true,
				CJKFontsAvailable: false,
				PlantUMLNeeded:    false,
				PlantUMLAvailable: false,
				BookYAMLFound:     true,
				SummaryFound:      true,
				LangsFound:        false,
				ProjectLoadable:   true,
				ProjectTitle:      "Test Project",
				TopLevelChapters:  3,
			},
			wantStr: []string{
				"# mdpress Doctor Report",
				"Platform: linux/amd64",
				"Go version: go1.24.2",
				"Chromium available: true",
				"CJK fonts available: false",
				"Project title: Test Project",
				"Top-level chapters: 3",
			},
		},
		{
			name: "with_warnings",
			report: doctorReport{
				Platform:  "darwin/arm64",
				GoVersion: "go1.25.0",
				Warnings: []string{
					"Chromium not available",
					"No CJK fonts detected",
				},
			},
			wantStr: []string{
				"# mdpress Doctor Report",
				"## Warnings",
				"Chromium not available",
				"No CJK fonts detected",
			},
		},
		{
			name: "with_unresolved_links",
			report: doctorReport{
				Platform:  "windows/amd64",
				GoVersion: "go1.24.0",
				UnresolvedMarkdown: []unresolvedMarkdownLink{
					{Source: "ch1.md", Target: "missing.md"},
					{Source: "ch2.md", Target: "notfound.md"},
				},
			},
			wantStr: []string{
				"## Unresolved Markdown Links",
				"missing.md (from ch1.md)",
				"notfound.md (from ch2.md)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderDoctorMarkdown(tt.report)
			for _, want := range tt.wantStr {
				if !strings.Contains(result, want) {
					t.Errorf("renderDoctorMarkdown missing %q in output:\n%s", want, result)
				}
			}
		})
	}
}

// TestWriteDoctorReport_JSON tests JSON report writing
func TestWriteDoctorReport_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.json")

	report := doctorReport{
		Platform:          "linux/amd64",
		GoVersion:         "go1.24.2",
		ChromiumAvailable: true,
		ProjectLoadable:   true,
		ProjectTitle:      "Test",
		TopLevelChapters:  2,
	}

	if err := writeDoctorReport(reportPath, report); err != nil {
		t.Fatalf("writeDoctorReport failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	// Verify JSON is valid
	var loaded doctorReport
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("invalid JSON in report: %v", err)
	}

	if loaded.Platform != report.Platform {
		t.Errorf("expected Platform %q, got %q", report.Platform, loaded.Platform)
	}
	if loaded.ProjectTitle != report.ProjectTitle {
		t.Errorf("expected ProjectTitle %q, got %q", report.ProjectTitle, loaded.ProjectTitle)
	}
}

// TestWriteDoctorReport_Markdown tests Markdown report writing
func TestWriteDoctorReport_Markdown(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.md")

	report := doctorReport{
		Platform:          "darwin/arm64",
		GoVersion:         "go1.24.2",
		ChromiumAvailable: true,
		ProjectLoadable:   true,
		ProjectTitle:      "My Book",
		Warnings: []string{
			"CJK fonts not available",
		},
	}

	if err := writeDoctorReport(reportPath, report); err != nil {
		t.Fatalf("writeDoctorReport failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# mdpress Doctor Report") {
		t.Error("Markdown report missing title")
	}
	if !strings.Contains(content, "darwin/arm64") {
		t.Error("Markdown report missing platform")
	}
	if !strings.Contains(content, "My Book") {
		t.Error("Markdown report missing project title")
	}
}

// TestWriteDoctorReport_InvalidExtension tests error handling for invalid file extensions
func TestWriteDoctorReport_InvalidExtension(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")

	report := doctorReport{
		Platform: "linux/amd64",
	}

	err := writeDoctorReport(reportPath, report)
	if err == nil {
		t.Error("writeDoctorReport should error for unsupported extension")
	}
	if !strings.Contains(err.Error(), "unsupported report extension") {
		t.Errorf("error message should mention unsupported extension, got: %v", err)
	}
}

// TestWriteDoctorReport_CreatesDirectory tests that writeDoctorReport creates parent directories
func TestWriteDoctorReport_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "subdir", "report.json")

	report := doctorReport{
		Platform: "linux/amd64",
	}

	if err := writeDoctorReport(reportPath, report); err != nil {
		t.Fatalf("writeDoctorReport failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report file should exist: %v", err)
	}
}

// TestDoctorReport_JSONSerialization tests JSON marshaling/unmarshaling of doctorReport
func TestDoctorReport_JSONSerialization(t *testing.T) {
	original := doctorReport{
		Platform:          "linux/amd64",
		GoVersion:         "go1.24.2",
		CacheDir:          t.TempDir(),
		CacheDisabled:     false,
		ChromiumAvailable: true,
		CJKFontsAvailable: true,
		PlantUMLAvailable: true,
		PlantUMLNeeded:    true,
		BookYAMLFound:     true,
		SummaryFound:      true,
		LangsFound:        false,
		ProjectLoadable:   true,
		ProjectTitle:      "Test Project",
		TopLevelChapters:  5,
		Warnings: []string{
			"Warning 1",
			"Warning 2",
		},
		UnresolvedMarkdown: []unresolvedMarkdownLink{
			{Source: "ch1.md", Target: "missing.md"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal from JSON
	var loaded doctorReport
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Compare all fields
	if loaded.Platform != original.Platform {
		t.Errorf("Platform mismatch: %q != %q", loaded.Platform, original.Platform)
	}
	if loaded.ProjectTitle != original.ProjectTitle {
		t.Errorf("ProjectTitle mismatch: %q != %q", loaded.ProjectTitle, original.ProjectTitle)
	}
	if loaded.TopLevelChapters != original.TopLevelChapters {
		t.Errorf("TopLevelChapters mismatch: %d != %d", loaded.TopLevelChapters, original.TopLevelChapters)
	}
	if len(loaded.Warnings) != len(original.Warnings) {
		t.Errorf("Warnings length mismatch: %d != %d", len(loaded.Warnings), len(original.Warnings))
	}
	if len(loaded.UnresolvedMarkdown) != len(original.UnresolvedMarkdown) {
		t.Errorf("UnresolvedMarkdown length mismatch: %d != %d", len(loaded.UnresolvedMarkdown), len(original.UnresolvedMarkdown))
	}
}

// TestDoctorPathResolution tests that executeDoctor handles absolute and relative paths
func TestDoctorPathResolution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple project
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test with relative path (current working directory changes during test)
	origCwd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(origCwd) }()

	// Should not error on current directory
	err := executeDoctor(context.Background(), ".")
	if err != nil {
		t.Fatalf("executeDoctor with '.' should not error: %v", err)
	}
}

// TestDoctorWithFile tests that executeDoctor rejects file paths
func TestDoctorWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")

	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	err := executeDoctor(context.Background(), filePath)
	if err == nil {
		t.Error("executeDoctor should error when passed a file instead of directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got: %v", err)
	}
}
