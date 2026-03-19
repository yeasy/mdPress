package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAutoDiscoverWithYamlAndSummary 测试具有 book.yaml 和 SUMMARY.md 的目录
func TestAutoDiscoverWithYamlAndSummary(t *testing.T) {
	dir := t.TempDir()

	// 创建 book.yaml
	yamlContent := `book:
  title: "Auto Discover Test"
chapters: []
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	// 创建 SUMMARY.md
	summaryContent := `# Summary

* [Intro](intro.md)
* [Chapter 1](ch1.md)
`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summaryContent), 0644); err != nil {
		t.Fatalf("写入 SUMMARY.md 失败: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"intro.md", "ch1.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}

	cfg, err := Discover(dir)
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

// TestAutoDiscoverCompleteDirectory 测试包含所有文件的完整目录
func TestAutoDiscoverCompleteDirectory(t *testing.T) {
	dir := t.TempDir()

	// 创建 book.yaml
	yamlContent := `book:
  title: "Complete Directory"
  author: "Test Author"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	// 创建 SUMMARY.md
	summaryContent := `* [Intro](intro.md)`
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summaryContent), 0644); err != nil {
		t.Fatalf("写入 SUMMARY.md 失败: %v", err)
	}

	// 创建 GLOSSARY.md
	glossaryContent := `# Glossary

## API
Application Programming Interface
`
	if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte(glossaryContent), 0644); err != nil {
		t.Fatalf("写入 GLOSSARY.md 失败: %v", err)
	}

	// 创建 LANGS.md
	langsContent := `# Languages

* [English](en/)
* [中文](zh/)
`
	if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte(langsContent), 0644); err != nil {
		t.Fatalf("写入 LANGS.md 失败: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if cfg.Book.Title != "Complete Directory" {
		t.Errorf("expected title 'Complete Directory', got %q", cfg.Book.Title)
	}
	if cfg.Book.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got %q", cfg.Book.Author)
	}
	// book.yaml 定义了 chapters，所以应该使用 YAML 中的
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

// TestAutoDiscoverOnlyYamlNoSummary 测试仅有 book.yaml 但没有 SUMMARY.md 的目录
func TestAutoDiscoverOnlyYamlNoSummary(t *testing.T) {
	dir := t.TempDir()

	// 创建 book.yaml，包含章节定义
	yamlContent := `book:
  title: "Only YAML"
chapters:
  - title: "Chapter 1"
    file: "ch1.md"
  - title: "Chapter 2"
    file: "ch2.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"ch1.md", "ch2.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}

	cfg, err := Discover(dir)
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

// TestAutoDiscoverOnlyYamlNoChapters 测试 book.yaml 存在但没有章节，也没有 SUMMARY.md
func TestAutoDiscoverOnlyYamlNoChapters(t *testing.T) {
	dir := t.TempDir()

	// 创建 book.yaml，但没有定义章节
	yamlContent := `book:
  title: "No Chapters"
chapters: []
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	_, err := Discover(dir)
	// 如果没有章节，Load() 会在 Validate() 时失败
	if err == nil {
		t.Error("expected error when no chapters defined")
	}
}

// TestAutoDiscoverEmptyYaml 测试空 book.yaml 内容
func TestAutoDiscoverEmptyYaml(t *testing.T) {
	dir := t.TempDir()

	// 创建空的 book.yaml
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(""), 0644); err != nil {
		t.Fatalf("写入空 book.yaml 失败: %v", err)
	}

	_, err := Discover(dir)
	// 空 YAML 会导致默认值被使用，但没有章节时应该失败
	if err == nil {
		t.Error("expected error for empty config without chapters")
	}
}

// TestLoadAutoDiscoverTableDriven 表驱动的自动发现测试
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
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644); err != nil {
					return err
				}
				summary := `* [Intro](intro.md)`
				if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644); err != nil {
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
				// YAML 中定义了 chapters，所以应该使用 YAML，不读 SUMMARY.md
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
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("## Term\nDef"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte("* [En](en/)"), 0644); err != nil {
					return err
				}
				// Create chapter file
				return os.WriteFile(filepath.Join(dir, "ch.md"), []byte("# Content"), 0644)
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
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md", "ch2.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644); err != nil {
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
				if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644); err != nil {
					return err
				}
				summary := `* [Ch1](ch1.md)
* [Ch2](ch2.md)
`
				if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0644); err != nil {
					return err
				}
				// Create chapter files
				for _, file := range []string{"ch1.md", "ch2.md"} {
					if err := os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644); err != nil {
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

			cfg, err := Discover(dir)
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
