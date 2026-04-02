package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestAutoDiscoverWithYamlAndSummary tests a directory with book.yaml and SUMMARY.md
func TestAutoDiscoverWithYamlAndSummary(t *testing.T) {
	dir := t.TempDir()

	// Create book.yaml
	yamlContent := `book:
  title: "Auto Discover Test"
chapters: []
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create SUMMARY.md
	summaryContent := `# Summary

* [Intro](intro.md)
* [Chapter 1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summaryContent), 0o644); err != nil {
		t.Fatalf("failed to write SUMMARY.md: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"intro.md", "ch1.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Auto Discover Test" {
		t.Errorf("expected title 'Auto Discover Test', got %q", cfg.Book.Title)
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters from SUMMARY.md, got %d", len(cfg.Chapters))
	}
	if cfg.Chapters[0].Title != "Intro" {
		t.Errorf("expected 'Intro', got %q", cfg.Chapters[0].Title)
	}
}

// TestAutoDiscoverCompleteDirectory tests a directory with all files present
func TestAutoDiscoverCompleteDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create book.yaml
	yamlContent := `book:
  title: "Complete Directory"
  author: "Test Author"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create SUMMARY.md
	summaryContent := `* [Intro](intro.md)`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summaryContent), 0o644); err != nil {
		t.Fatalf("failed to write SUMMARY.md: %v", err)
	}

	// Create GLOSSARY.md
	glossaryContent := `# Glossary

## API
Application Programming Interface
`
	if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte(glossaryContent), 0o644); err != nil {
		t.Fatalf("failed to write GLOSSARY.md: %v", err)
	}

	// Create LANGS.md
	langsContent := `# Languages

* [English](en/)
* [中文](zh/)
`
	if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte(langsContent), 0o644); err != nil {
		t.Fatalf("failed to write LANGS.md: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Complete Directory" {
		t.Errorf("expected title 'Complete Directory', got %q", cfg.Book.Title)
	}
	if cfg.Book.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got %q", cfg.Book.Author)
	}
	// book.yaml defines chapters, so YAML ones should be used
	if len(cfg.Chapters) != 1 {
		t.Errorf("expected 1 chapter from YAML, got %d", len(cfg.Chapters))
	}
	if cfg.GlossaryFile == "" {
		t.Error("should detect GLOSSARY.md")
	}
	if cfg.LangsFile == "" {
		t.Error("should detect LANGS.md")
	}
}

// TestAutoDiscoverOnlyYamlNoSummary tests a directory with only book.yaml and no SUMMARY.md
func TestAutoDiscoverOnlyYamlNoSummary(t *testing.T) {
	dir := t.TempDir()

	// Create book.yaml with chapter definitions
	yamlContent := `book:
  title: "Only YAML"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
  - title: "Chapter 2"
    file: "ch2.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"ch1.md", "ch2.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Only YAML" {
		t.Errorf("expected title 'Only YAML', got %q", cfg.Book.Title)
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("expected 2 chapters from YAML, got %d", len(cfg.Chapters))
	}
}

// TestAutoDiscoverOnlyYamlNoChapters tests book.yaml without chapters and no SUMMARY.md
func TestAutoDiscoverOnlyYamlNoChapters(t *testing.T) {
	dir := t.TempDir()

	// Create book.yaml without chapter definitions
	yamlContent := `book:
  title: "No Chapters"
chapters: []
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	_, err := Discover(context.Background(), dir)
	// Without chapters, Load() fails at Validate()
	if err == nil {
		t.Error("expected error when no chapters defined")
	}
}

// TestAutoDiscoverEmptyYaml tests empty book.yaml content
func TestAutoDiscoverEmptyYaml(t *testing.T) {
	dir := t.TempDir()

	// Create empty book.yaml
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write empty book.yaml: %v", err)
	}

	_, err := Discover(context.Background(), dir)
	// Empty YAML uses defaults, but should fail without chapters
	if err == nil {
		t.Error("expected error for empty config without chapters")
	}
}

// TestLoadAutoDiscoverTableDriven table-driven auto-discovery tests
func TestLoadAutoDiscoverTableDriven(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(dir string) error
		wantErr     bool
		checkFields func(t *testing.T, cfg *BookConfig)
	}{
		{
			name: "book.yaml with chapters and SUMMARY.md",
			setup: func(dir string) error {
				yaml := `book:
  title: "Test Book"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o644); err != nil {
					return err
				}
				summary := `* [Intro](intro.md)`
				if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *BookConfig) {
				if cfg.Book.Title != "Test Book" {
					t.Errorf("expected 'Test Book', got %q", cfg.Book.Title)
				}
				// YAML defines chapters, so YAML is used instead of SUMMARY.md
				if len(cfg.Chapters) != 1 {
					t.Errorf("expected 1 chapter from YAML, got %d", len(cfg.Chapters))
				}
			},
		},
		{
			name: "book.yaml with GLOSSARY.md and LANGS.md",
			setup: func(dir string) error {
				yaml := `book:
  title: "Test"
chapters:
  - title: "Ch"
    file: "ch.md"
`
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("## Term\nDef"), 0o644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte("* [En](en/)"), 0o644); err != nil {
					return err
				}
				// Create chapter file
				return os.WriteFile(filepath.Join(dir, "ch.md"), []byte("# Content"), 0o644)
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *BookConfig) {
				if cfg.GlossaryFile == "" {
					t.Error("should detect GLOSSARY.md")
				}
				if cfg.LangsFile == "" {
					t.Error("should detect LANGS.md")
				}
			},
		},
		{
			name: "book.yaml with chapters, no SUMMARY.md",
			setup: func(dir string) error {
				yaml := `book:
  title: "Test"
chapters:
  - title: "Ch1"
    file: "ch1.md"
  - title: "Ch2"
    file: "ch2.md"
`
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md", "ch2.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *BookConfig) {
				if len(cfg.Chapters) != 2 {
					t.Errorf("expected 2 chapters, got %d", len(cfg.Chapters))
				}
			},
		},
		{
			name: "book.yaml without chapters, with SUMMARY.md",
			setup: func(dir string) error {
				yaml := `book:
  title: "From Summary"
chapters: []
`
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o644); err != nil {
					return err
				}
				summary := `* [Ch1](ch1.md)
* [Ch2](ch2.md)
`
				if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0o644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md", "ch2.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0o644); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *BookConfig) {
				if len(cfg.Chapters) != 2 {
					t.Errorf("expected 2 chapters from SUMMARY.md, got %d", len(cfg.Chapters))
				}
				if cfg.Chapters[0].Title != "Ch1" {
					t.Errorf("expected 'Ch1', got %q", cfg.Chapters[0].Title)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(dir); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			cfg, err := Discover(context.Background(), dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Discover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.checkFields != nil {
				tt.checkFields(t, cfg)
			}
		})
	}
}
