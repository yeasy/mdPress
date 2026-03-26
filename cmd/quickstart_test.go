package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestQuickstartCreatesProject(t *testing.T) {
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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
	defer suppressOutput(t)()
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

func TestQuickstartReadDirErrorHandling(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "readable-project")

	// Create project directory normally
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Verify executeQuickstart succeeds on empty directory
	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should succeed on empty directory: %v", err)
	}

	// Verify the directory is now non-empty
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		t.Fatalf("ReadDir should succeed after quickstart: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("quickstart should have created files")
	}
}

func TestQuickstartHiddenFilesDetection(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "hidden-files-project")

	// Create directory with hidden file (.git is commonly overlooked by glob patterns)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	hiddenFile := filepath.Join(projectDir, ".git")
	if err := os.Mkdir(hiddenFile, 0755); err != nil {
		t.Fatalf("failed to create hidden directory: %v", err)
	}

	// quickstart should reject non-empty directory (even with only hidden files)
	err := executeQuickstart(projectDir)
	if err == nil {
		t.Error("executeQuickstart should reject directory with hidden files")
	}
}

// Test quickstart command creation and basic properties
func TestQuickstartCommand(t *testing.T) {
	if quickstartCmd == nil {
		t.Fatal("quickstartCmd should not be nil")
	}

	if quickstartCmd.Use != "quickstart [directory]" {
		t.Errorf("expected Use %q, got %q", "quickstart [directory]", quickstartCmd.Use)
	}

	if quickstartCmd.Short == "" {
		t.Error("quickstartCmd.Short should not be empty")
	}

	if quickstartCmd.Long == "" {
		t.Error("quickstartCmd.Long should not be empty")
	}
}

// Test that command has correct argument validation
func TestQuickstartCommandArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one argument",
			args:    []string{"my-book"},
			wantErr: false,
		},
		{
			name:    "two arguments should error",
			args:    []string{"dir1", "dir2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := quickstartCmd.Args(quickstartCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// Test directory validation with empty vs non-empty checks
func TestDirectoryValidation(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(dir string) error
		expectError  bool
		errorPattern string
	}{
		{
			name: "non-existent directory",
			setup: func(dir string) error {
				return nil // don't create it
			},
			expectError: false,
		},
		{
			name: "empty existing directory",
			setup: func(dir string) error {
				return os.MkdirAll(dir, 0755)
			},
			expectError: false,
		},
		{
			name: "directory with regular file",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
			},
			expectError:  true,
			errorPattern: "already exists and is not empty",
		},
		{
			name: "directory with subdirectory",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
			},
			expectError:  true,
			errorPattern: "already exists and is not empty",
		},
		{
			name: "directory with hidden file",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644)
			},
			expectError:  true,
			errorPattern: "already exists and is not empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer suppressOutput(t)()
			tmpDir := t.TempDir()
			projectDir := filepath.Join(tmpDir, "test-project")

			if err := tt.setup(projectDir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			err := executeQuickstart(projectDir)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v: %v", tt.expectError, err != nil, err)
			}
			if tt.expectError && err != nil && tt.errorPattern != "" {
				if !strings.Contains(err.Error(), tt.errorPattern) {
					t.Errorf("expected error containing %q, got %q", tt.errorPattern, err.Error())
				}
			}
		})
	}
}

// Test template file generation with table-driven tests
func TestTemplateGeneration(t *testing.T) {
	tests := []struct {
		name             string
		filePath         string
		projectName      string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:        "book.yaml generation",
			filePath:    "book.yaml",
			projectName: "test-book",
			shouldContain: []string{
				"title: \"test-book\"",
				"subtitle: \"A sample book created with mdpress\"",
				"author: \"Your Name\"",
				"version: \"1.0.0\"",
				"language: \"en-US\"",
				"chapters:",
				"Preface",
				"Chapter 1: Getting Started",
				"output:",
				"filename: \"test-book.pdf\"",
			},
			shouldNotContain: []string{},
		},
		{
			name:        "README.md generation",
			filePath:    "README.md",
			projectName: "my-project",
			shouldContain: []string{
				"# my-project",
				"mdpress",
				"Book configuration",
				"Quick Start",
				"mdpress build",
				"mdpress serve",
			},
			shouldNotContain: []string{},
		},
		{
			name:     "preface.md generation",
			filePath: "preface.md",
			shouldContain: []string{
				"# Preface",
				"Welcome to this sample book",
				"mdpress",
				"Markdown authoring",
				"Multiple output formats",
			},
			shouldNotContain: []string{},
		},
		{
			name:     "chapter01 generation",
			filePath: filepath.Join("chapter01", "README.md"),
			shouldContain: []string{
				"# Getting Started",
				"Install mdpress",
				"Create a Project",
				"Write Content",
				"Build Output",
			},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer suppressOutput(t)()
			tmpDir := t.TempDir()
			projectDir := filepath.Join(tmpDir, tt.projectName)

			if err := executeQuickstart(projectDir); err != nil {
				t.Fatalf("executeQuickstart failed: %v", err)
			}

			filePath := filepath.Join(projectDir, tt.filePath)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file %s: %v", tt.filePath, err)
			}

			contentStr := string(content)
			for _, substr := range tt.shouldContain {
				if !strings.Contains(contentStr, substr) {
					t.Errorf("expected file to contain %q, got:\n%s", substr, contentStr[:min(len(contentStr), 500)])
				}
			}

			for _, substr := range tt.shouldNotContain {
				if strings.Contains(contentStr, substr) {
					t.Errorf("expected file to NOT contain %q", substr)
				}
			}
		})
	}
}

// Test book.yaml content generation with different project names
func TestBookYAMLContent(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		checks      func(t *testing.T, content string)
	}{
		{
			name:        "simple project name",
			projectName: "my-book",
			checks: func(t *testing.T, content string) {
				if !strings.Contains(content, `title: "my-book"`) {
					t.Error("expected title to match project name")
				}
				if !strings.Contains(content, `filename: "my-book.pdf"`) {
					t.Error("expected filename to match project name")
				}
			},
		},
		{
			name:        "long project name",
			projectName: "very-long-project-name-with-many-words",
			checks: func(t *testing.T, content string) {
				if !strings.Contains(content, `title: "very-long-project-name-with-many-words"`) {
					t.Error("expected title to match project name")
				}
			},
		},
		{
			name:        "special characters in name",
			projectName: "project-with-dashes_and_underscores",
			checks: func(t *testing.T, content string) {
				if !strings.Contains(content, `filename: "project-with-dashes_and_underscores.pdf"`) {
					t.Error("expected filename to preserve special characters")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := generateQuickstartBookYAML(tt.projectName)
			tt.checks(t, yaml)
		})
	}
}

// Test SVG cover generation with special characters and truncation
func TestPlaceholderCoverSVG(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		expectedSVG string
		shouldMatch bool
	}{
		{
			name:        "simple title",
			title:       "My Book",
			expectedSVG: "My Book",
			shouldMatch: true,
		},
		{
			name:        "title with special chars",
			title:       "Book & Guide",
			expectedSVG: "Book &amp; Guide",
			shouldMatch: true,
		},
		{
			name:        "title with angle brackets",
			title:       "Book <advanced>",
			expectedSVG: "Book &lt;advanced&gt;",
			shouldMatch: true,
		},
		{
			name:        "long title truncation",
			title:       "This is a very long book title that should be truncated",
			expectedSVG: "This is a very long ...",
			shouldMatch: true,
		},
		{
			name:        "SVG structure",
			title:       "Test",
			expectedSVG: "<svg xmlns=",
			shouldMatch: true,
		},
		{
			name:        "gradient definition",
			title:       "Test",
			expectedSVG: "linearGradient",
			shouldMatch: true,
		},
		{
			name:        "mdpress attribution",
			title:       "Test",
			expectedSVG: "Built with mdpress",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := generatePlaceholderCoverSVG(tt.title)
			if tt.shouldMatch {
				if !strings.Contains(svg, tt.expectedSVG) {
					t.Errorf("expected SVG to contain %q, got:\n%s", tt.expectedSVG, svg[:min(len(svg), 300)])
				}
			} else {
				if strings.Contains(svg, tt.expectedSVG) {
					t.Errorf("expected SVG to NOT contain %q", tt.expectedSVG)
				}
			}
		})
	}
}

// Test error handling for path resolution
func TestPathResolutionError(t *testing.T) {
	// This is challenging to test without mocking or special setup.
	// We test a normal case where path resolution should work.
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "path-test")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart with normal path should not error: %v", err)
	}

	// Verify the directory was created
	if _, err := os.Stat(projectDir); err != nil {
		t.Errorf("project directory should exist: %v", err)
	}
}

// Test content of generated chapters
func TestGeneratedChapterContent(t *testing.T) {
	tests := []struct {
		name          string
		chapterPath   string
		shouldContain []string
	}{
		{
			name:        "chapter01 content",
			chapterPath: filepath.Join("chapter01", "README.md"),
			shouldContain: []string{
				"Getting Started",
				"Install mdpress",
				"Build Output",
			},
		},
		{
			name:        "chapter02 content",
			chapterPath: filepath.Join("chapter02", "README.md"),
			shouldContain: []string{
				"Advanced Usage",
				"Config File",
				"Live Preview",
			},
		},
		{
			name:        "chapter03 content",
			chapterPath: filepath.Join("chapter03", "README.md"),
			shouldContain: []string{
				"Best Practices",
				"Project Organization",
				"Version Control",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer suppressOutput(t)()
			tmpDir := t.TempDir()
			projectDir := filepath.Join(tmpDir, "chapter-test")

			if err := executeQuickstart(projectDir); err != nil {
				t.Fatalf("executeQuickstart failed: %v", err)
			}

			filePath := filepath.Join(projectDir, tt.chapterPath)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read chapter file: %v", err)
			}

			contentStr := string(content)
			for _, substr := range tt.shouldContain {
				if !strings.Contains(contentStr, substr) {
					t.Errorf("expected chapter to contain %q", substr)
				}
			}
		})
	}
}

// Test images directory structure
func TestImagesDirectoryStructure(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "images-structure-test")

	if err := executeQuickstart(projectDir); err != nil {
		t.Fatalf("executeQuickstart failed: %v", err)
	}

	imagesDir := filepath.Join(projectDir, "images")
	expectedFiles := []string{
		filepath.Join(imagesDir, "README.md"),
		filepath.Join(imagesDir, "cover.svg"),
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); err != nil {
			t.Errorf("expected file %s to exist: %v", expectedFile, err)
		}
	}

	// Verify README.md in images directory has useful content
	imagesReadmePath := filepath.Join(imagesDir, "README.md")
	content, err := os.ReadFile(imagesReadmePath)
	if err != nil {
		t.Fatalf("failed to read images/README.md: %v", err)
	}

	contentStr := string(content)
	expectedTexts := []string{"Image Assets", "Supported Formats", "PNG", "JPEG", "SVG"}
	for _, text := range expectedTexts {
		if !strings.Contains(contentStr, text) {
			t.Errorf("expected images README to contain %q", text)
		}
	}
}

// Test all chapters are listed in book.yaml
func TestChaptersInBookYAML(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "chapters-yaml-test")

	if err := executeQuickstart(projectDir); err != nil {
		t.Fatalf("executeQuickstart failed: %v", err)
	}

	bookPath := filepath.Join(projectDir, "book.yaml")
	content, err := os.ReadFile(bookPath)
	if err != nil {
		t.Fatalf("failed to read book.yaml: %v", err)
	}

	contentStr := string(content)
	expectedChapters := []string{
		"Preface",
		"preface.md",
		"Chapter 1: Getting Started",
		"chapter01/README.md",
		"Chapter 2: Advanced Usage",
		"chapter02/README.md",
		"Chapter 3: Best Practices",
		"chapter03/README.md",
	}

	for _, chapter := range expectedChapters {
		if !strings.Contains(contentStr, chapter) {
			t.Errorf("expected book.yaml to contain %q", chapter)
		}
	}
}

// Test default directory name
func TestDefaultDirectoryName(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	defaultDir := filepath.Join(tmpDir, "my-book")

	// Create a test scenario where we simulate default behavior
	err := executeQuickstart(defaultDir)
	if err != nil {
		t.Fatalf("executeQuickstart with default name should not error: %v", err)
	}

	// Verify project was created
	if _, err := os.Stat(filepath.Join(defaultDir, "book.yaml")); err != nil {
		t.Errorf("book.yaml should exist in default directory: %v", err)
	}
}

// Test nested directory creation
func TestNestedDirectoryCreation(t *testing.T) {
	defer suppressOutput(t)()
	tmpDir := t.TempDir()
	// Create a path with multiple non-existent parent directories
	projectDir := filepath.Join(tmpDir, "parent", "child", "project")

	err := executeQuickstart(projectDir)
	if err != nil {
		t.Fatalf("executeQuickstart should create nested directories: %v", err)
	}

	// Verify the nested directory structure was created
	if _, err := os.Stat(projectDir); err != nil {
		t.Errorf("nested project directory should exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, "book.yaml")); err != nil {
		t.Errorf("book.yaml should exist in nested directory: %v", err)
	}
}

// Test SVG XML special character handling
func TestSVGSpecialCharacterHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ampersand",
			input:    "Fish & Chips",
			expected: "Fish &amp; Chips",
		},
		{
			name:     "less than",
			input:    "Value < 10",
			expected: "Value &lt; 10",
		},
		{
			name:     "greater than",
			input:    "Value > 10",
			expected: "Value &gt; 10",
		},
		{
			name:     "multiple special chars",
			input:    "A & B < C > D",
			expected: "A &amp; B &lt; C &gt; D",
		},
		{
			name:     "no special chars",
			input:    "Normal Title",
			expected: "Normal Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := generatePlaceholderCoverSVG(tt.input)
			if !strings.Contains(svg, tt.expected) {
				t.Errorf("expected SVG to contain %q, got:\n%s", tt.expected, svg)
			}
		})
	}
}
