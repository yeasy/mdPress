package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestQuickstartCreatesProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	// Verify book.yaml was created
	bookPath := filepath.Join(projectDir, "book.yaml")
	if _, err := os.Stat(bookPath); err != nil {
		t.Errorf("book.yaml should exist: %v", err)
	}

	// Verify at least one markdown file was created
	readmePath := filepath.Join(projectDir, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("README.md should exist: %v", err)
	}

	// Verify preface was created
	prefacePath := filepath.Join(projectDir, "preface.md")
	if _, err := os.Stat(prefacePath); err != nil {
		t.Errorf("preface.md should exist: %v", err)
	}

	// Verify at least one chapter was created
	ch01Path := filepath.Join(projectDir, "chapter01", "README.md")
	if _, err := os.Stat(ch01Path); err != nil {
		t.Errorf("chapter01/README.md should exist: %v", err)
	}

	// Verify images directory and cover were created
	coverPath := filepath.Join(projectDir, "images", "cover.svg")
	if _, err := os.Stat(coverPath); err != nil {
		t.Errorf("images/cover.svg should exist: %v", err)
	}
}

func TestQuickstartDoesNotOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "existing-project")

	// Create the project directory and add a file
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	existingFile := filepath.Join(projectDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	// Try to run quickstart on the non-empty directory
	err := executeQuickstart(projectDir)
	if err == nil {
		t.Error("executeQuickstart should error when directory is not empty")
	}

	// Verify the existing file was not modified
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}
	if string(content) != "existing content" {
		t.Error("existing file should not be modified")
	}
}

func TestQuickstartProjectLoadable(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "loadable-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	// Attempt to load the generated book.yaml
	bookPath := filepath.Join(projectDir, "book.yaml")
	cfg, err := config.Load(bookPath)
	if err != nil {
		t.Fatalf("config.Load should succeed: %v", err)
	}

	// Verify basic properties of the loaded config
	if cfg.Book.Title == "" {
		t.Error("loaded config should have a title")
	}
	if len(cfg.Chapters) == 0 {
		t.Error("loaded config should have chapters")
	}

	// Verify the output filename is set
	if cfg.Output.Filename == "" {
		t.Error("loaded config should have an output filename")
	}
}

func TestQuickstartGeneratesAllChapters(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "full-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	// Verify all three chapters exist
	chapters := []string{
		filepath.Join(projectDir, "chapter01", "README.md"),
		filepath.Join(projectDir, "chapter02", "README.md"),
		filepath.Join(projectDir, "chapter03", "README.md"),
	}

	for _, chPath := range chapters {
		if _, err := os.Stat(chPath); err != nil {
			t.Errorf("chapter should exist at %s: %v", chPath, err)
		}
	}
}

func TestQuickstartCreatesImageDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "images-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	// Verify images directory exists
	imagesDir := filepath.Join(projectDir, "images")
	if _, err := os.Stat(imagesDir); err != nil {
		t.Errorf("images directory should exist: %v", err)
	}

	// Verify images/README.md exists
	imagesReadme := filepath.Join(imagesDir, "README.md")
	if _, err := os.Stat(imagesReadme); err != nil {
		t.Errorf("images/README.md should exist: %v", err)
	}

	// Verify cover.svg exists
	coverSVG := filepath.Join(imagesDir, "cover.svg")
	if _, err := os.Stat(coverSVG); err != nil {
		t.Errorf("cover.svg should exist: %v", err)
	}
}

func TestQuickstartWithCustomProjectName(t *testing.T) {
	tmpDir := t.TempDir()
	projectName := "my-awesome-book"
	projectDir := filepath.Join(tmpDir, projectName)

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	// Load the config and verify the project name is used
	bookPath := filepath.Join(projectDir, "book.yaml")
	cfg, err := config.Load(bookPath)
	if err != nil {
		t.Fatalf("config.Load should succeed: %v", err)
	}

	// The output filename should use the project name
	expectedFilename := projectName + ".pdf"
	if cfg.Output.Filename != expectedFilename {
		t.Errorf("expected output filename %q, got %q", expectedFilename, cfg.Output.Filename)
	}
}

func TestQuickstartConfigHasRequiredFields(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "required-fields-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	bookPath := filepath.Join(projectDir, "book.yaml")
	cfg, err := config.Load(bookPath)
	if err != nil {
		t.Fatalf("config.Load should succeed: %v", err)
	}

	// Verify required fields
	if cfg.Book.Title == "" {
		t.Error("book title should not be empty")
	}
	if cfg.Book.Author == "" {
		t.Error("book author should not be empty")
	}
	if cfg.Book.Version == "" {
		t.Error("book version should not be empty")
	}
	if len(cfg.Chapters) < 3 {
		t.Error("book should have at least 3 chapters (preface + 3 chapters)")
	}
	if cfg.Style.PageSize == "" {
		t.Error("page size should be set")
	}
	if cfg.Output.Filename == "" {
		t.Error("output filename should be set")
	}
}

func TestQuickstartCanValidateGeneratedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "validate-project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should not error, got: %v", err)
	}

	bookPath := filepath.Join(projectDir, "book.yaml")
	cfg, err := config.Load(bookPath)
	if err != nil {
		t.Fatalf("config.Load should succeed: %v", err)
	}

	// Validate the loaded config
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("generated config should be valid: %v", err)
	}
}
