package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultConfig tests default config sanity
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Book.Title != "Untitled Book" {
		t.Errorf("wrong default title: got %q, want %q", cfg.Book.Title, "Untitled Book")
	}
	if cfg.Book.Language != "zh-CN" {
		t.Errorf("wrong default language: got %q, want %q", cfg.Book.Language, "zh-CN")
	}
	if cfg.Style.PageSize != "A4" {
		t.Errorf("wrong default page size: got %q, want %q", cfg.Style.PageSize, "A4")
	}
	if cfg.Style.Theme != "technical" {
		t.Errorf("wrong default theme: got %q, want %q", cfg.Style.Theme, "technical")
	}
	if cfg.Style.LineHeight != 1.6 {
		t.Errorf("wrong default line height: got %f, want %f", cfg.Style.LineHeight, 1.6)
	}
	if cfg.Output.Filename != "output.pdf" {
		t.Errorf("wrong default output filename: got %q, want %q", cfg.Output.Filename, "output.pdf")
	}
	if !cfg.Output.TOC {
		t.Error("TOC should be enabled by default")
	}
	if !cfg.Output.Cover {
		t.Error("cover should be enabled by default")
	}
	if cfg.Style.Margin.Top != 25 {
		t.Errorf("wrong default top margin: got %f, want %f", cfg.Style.Margin.Top, 25.0)
	}
}

// TestLoadValidConfig tests loading a valid config file
func TestLoadValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "测试图书"
  author: "测试作者"
  version: "2.0.0"
  language: "en-US"

chapters:
  - title: "第一章"
    file: "ch01.md"
  - title: "第二章"
    file: "ch02.md"

style:
  theme: "elegant"
  page_size: "A5"
  font_family: "Arial"
  code_theme: "dracula"

output:
  filename: "test.pdf"
  toc: false
  cover: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch02.md"), []byte("# Chapter 2"), 0o644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Book.Title != "测试图书" {
		t.Errorf("wrong title: got %q, want %q", cfg.Book.Title, "测试图书")
	}
	if cfg.Book.Author != "测试作者" {
		t.Errorf("wrong author: got %q, want %q", cfg.Book.Author, "测试作者")
	}
	if cfg.Book.Version != "2.0.0" {
		t.Errorf("wrong version: got %q, want %q", cfg.Book.Version, "2.0.0")
	}
	if cfg.Book.Language != "en-US" {
		t.Errorf("wrong language: got %q, want %q", cfg.Book.Language, "en-US")
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("wrong chapter count: got %d, want %d", len(cfg.Chapters), 2)
	}
	if cfg.Style.Theme != "elegant" {
		t.Errorf("wrong theme: got %q, want %q", cfg.Style.Theme, "elegant")
	}
	if cfg.Style.PageSize != "A5" {
		t.Errorf("wrong page size: got %q, want %q", cfg.Style.PageSize, "A5")
	}
	if cfg.Output.Filename != "test.pdf" {
		t.Errorf("wrong output filename: got %q, want %q", cfg.Output.Filename, "test.pdf")
	}
	if cfg.Output.TOC {
		t.Error("TOC should be disabled")
	}
	if cfg.Output.Cover {
		t.Error("cover should be disabled")
	}
}

// TestLoadNonExistentFile tests loading a non-existent file
func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/book.yaml")
	if err == nil {
		t.Error("loading a non-existent file should return an error")
	}
}

// TestLoadInvalidYAML tests loading invalid YAML
func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `invalid yaml: [broken: {`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("loading invalid YAML should return an error")
	}
}

// TestValidateEmptyTitle tests empty title validation
func TestValidateEmptyTitle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = ""
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}

	err := cfg.Validate()
	if err == nil {
		t.Error("empty title should fail validation")
	}
}

// TestValidateNoChapters tests no-chapters validation
func TestValidateNoChapters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = nil

	err := cfg.Validate()
	if err == nil {
		t.Error("no chapters should fail validation")
	}
}

// TestValidateEmptyChapterFile tests empty chapter file
func TestValidateEmptyChapterFile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: ""}}

	err := cfg.Validate()
	if err == nil {
		t.Error("empty file path should fail validation")
	}
}

// TestValidateInvalidPageSize tests invalid page size
func TestValidateInvalidPageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Style.PageSize = "INVALID"

	err := cfg.Validate()
	if err == nil {
		t.Error("invalid page size should fail validation")
	}
}

// TestValidateValidPageSizes tests all valid page sizes
func TestValidateValidPageSizes(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	sizes := []string{"A4", "A5", "Letter", "Legal", "B5"}
	for _, size := range sizes {
		cfg := DefaultConfig()
		cfg.baseDir = tmpDir
		cfg.Book.Title = "Test"
		cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
		cfg.Style.PageSize = size

		if err := cfg.Validate(); err != nil {
			t.Errorf("page size %q should pass validation, got error: %v", size, err)
		}
	}
}

// TestIsValidPageSize tests the page size validator function.
func TestIsValidPageSize(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"A4", true},
		{"A5", true},
		{"Letter", true},
		{"Legal", true},
		{"B5", true},
		{"a4", true},
		{"LETTER", true},
		{"letter", true},
		{"legal", true},
		{"A3", false},
		{"Tabloid", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValidPageSize(tt.input); got != tt.want {
				t.Errorf("IsValidPageSize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test_validPageSizeNames tests that all expected page sizes are returned.
func Test_validPageSizeNames(t *testing.T) {
	names := validPageSizeNames()
	if len(names) != 5 {
		t.Fatalf("expected 5 page sizes, got %d: %v", len(names), names)
	}
	for _, name := range names {
		if !IsValidPageSize(name) {
			t.Errorf("validPageSizeNames() returned %q which is not valid", name)
		}
	}
	// Verify returns a fresh slice each call.
	a := validPageSizeNames()
	b := validPageSizeNames()
	a[0] = "MUTATED"
	for _, name := range b {
		if name == "MUTATED" {
			t.Error("validPageSizeNames should return a new slice each time")
		}
	}
}

// TestResolvePath tests path resolution
func TestResolvePath(t *testing.T) {
	cfg := DefaultConfig()
	baseDir := t.TempDir()
	cfg.baseDir = baseDir

	absolutePath, err := filepath.Abs(filepath.Join(baseDir, "absolute", "path.md"))
	if err != nil {
		t.Fatalf("failed to build absolute path fixture: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"ch01.md", filepath.Join(baseDir, "ch01.md")},
		{"sub/ch01.md", filepath.Join(baseDir, "sub", "ch01.md")},
		{absolutePath, absolutePath},
	}

	for _, tt := range tests {
		got := cfg.ResolvePath(tt.input)
		if got != tt.want {
			t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestBaseDir tests base directory
func TestBaseDir(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseDir() != "." {
		t.Errorf("default BaseDir should be '.', got %q", cfg.BaseDir())
	}

	cfg.baseDir = "/test/path"
	if cfg.BaseDir() != "/test/path" {
		t.Errorf("BaseDir should be '/test/path', got %q", cfg.BaseDir())
	}
}

// TestLoadSetsBaseDir tests that Load sets BaseDir
func TestLoadSetsBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.BaseDir() != tmpDir {
		t.Errorf("BaseDir should be %q, got %q", tmpDir, cfg.BaseDir())
	}
}

// TestNestedChapters tests nested sub-chapters
func TestNestedChapters(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test"
chapters:
  - title: "Part 1"
    file: "part1.md"
    sections:
      - title: "Section 1.1"
        file: "s1.md"
      - title: "Section 1.2"
        file: "s2.md"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"part1.md", "s1.md", "s2.md"} {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("# Section"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Chapters) != 1 {
		t.Fatalf("wrong top-level chapter count: got %d, want %d", len(cfg.Chapters), 1)
	}
	if len(cfg.Chapters[0].Sections) != 2 {
		t.Errorf("wrong sub-chapter count: got %d, want %d", len(cfg.Chapters[0].Sections), 2)
	}
}

// TestDefaultValuesPreserved tests that partial config preserves defaults
func TestDefaultValuesPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	// Only set the minimum required fields
	content := `
book:
  title: "Minimal Config"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify default values are preserved
	if cfg.Style.PageSize != "A4" {
		t.Errorf("default PageSize should be preserved: got %q", cfg.Style.PageSize)
	}
	if cfg.Output.Filename != "output.pdf" {
		t.Errorf("default Filename should be preserved: got %q", cfg.Output.Filename)
	}
	if !cfg.Output.TOC {
		t.Error("default TOC should be preserved as true")
	}
}

// TestValidateTableDriven table-driven tests covering multiple validation scenarios
func TestValidateTableDriven(t *testing.T) {
	// Create temp directory and file for successful validation cases
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	tests := []struct {
		name    string
		setup   func(*BookConfig)
		wantErr bool
	}{
		{
			name: "valid config",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Valid Book"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "A4"
			},
			wantErr: false,
		},
		{
			name: "empty title",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = ""
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
			},
			wantErr: true,
		},
		{
			name: "no chapters",
			setup: func(cfg *BookConfig) {
				cfg.Book.Title = "Test"
				cfg.Chapters = nil
			},
			wantErr: true,
		},
		{
			name: "empty chapter file",
			setup: func(cfg *BookConfig) {
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: ""}}
			},
			wantErr: true,
		},
		{
			name: "invalid page size",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "INVALID_SIZE"
			},
			wantErr: true,
		},
		{
			name: "valid page size A5",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "A5"
			},
			wantErr: false,
		},
		{
			name: "valid page size Letter",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "Letter"
			},
			wantErr: false,
		},
		{
			name: "valid page size Legal",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "Legal"
			},
			wantErr: false,
		},
		{
			name: "valid page size B5",
			setup: func(cfg *BookConfig) {
				cfg.baseDir = tmpDir
				cfg.Book.Title = "Test"
				cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
				cfg.Style.PageSize = "B5"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestLoadConfigOverrideDefaults tests that YAML values correctly override defaults
func TestLoadConfigOverrideDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Custom Book"
  language: "fr-FR"

chapters:
  - title: "Ch1"
    file: "ch1.md"

style:
  line_height: 2.0
  page_size: "A5"
  theme: "elegant"
  margin:
    top: 30
    bottom: 35
    left: 25
    right: 25

output:
  filename: "custom.pdf"
  toc: false
  cover: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify overridden values
	if cfg.Book.Language != "fr-FR" {
		t.Errorf("Language not overridden: got %q, want %q", cfg.Book.Language, "fr-FR")
	}
	if cfg.Style.LineHeight != 2.0 {
		t.Errorf("LineHeight not overridden: got %f, want %f", cfg.Style.LineHeight, 2.0)
	}
	if cfg.Style.PageSize != "A5" {
		t.Errorf("PageSize not overridden: got %q, want %q", cfg.Style.PageSize, "A5")
	}
	if cfg.Style.Theme != "elegant" {
		t.Errorf("Theme not overridden: got %q, want %q", cfg.Style.Theme, "elegant")
	}
	if cfg.Style.Margin.Top != 30 {
		t.Errorf("Margin.Top not overridden: got %f, want %f", cfg.Style.Margin.Top, 30.0)
	}
	if cfg.Style.Margin.Bottom != 35 {
		t.Errorf("Margin.Bottom not overridden: got %f, want %f", cfg.Style.Margin.Bottom, 35.0)
	}
	if cfg.Output.Filename != "custom.pdf" {
		t.Errorf("Filename not overridden: got %q, want %q", cfg.Output.Filename, "custom.pdf")
	}
	if cfg.Output.TOC {
		t.Error("TOC not correctly overridden to false")
	}
	if cfg.Output.Cover {
		t.Error("Cover not correctly overridden to false")
	}
}

// TestLoadAutoDetectsLangs tests LANGS.md auto-detection
func TestLoadAutoDetectsLangs(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	langsPath := filepath.Join(tmpDir, "LANGS.md")

	// Create config file
	cfgContent := `
book:
  title: "Test Book"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create LANGS.md
	langsContent := `# Languages

- [English](en/)
- [中文](zh/)
`
	if err := os.WriteFile(langsPath, []byte(langsContent), 0o644); err != nil {
		t.Fatalf("failed to write LANGS.md: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.LangsFile == "" {
		t.Error("should detect LANGS.md")
	}
	if cfg.LangsFile != langsPath {
		t.Errorf("wrong LangsFile path: got %q, want %q", cfg.LangsFile, langsPath)
	}
}

// TestResolvePathTableDriven table-driven path resolution tests with edge cases
func TestResolvePathTableDriven(t *testing.T) {
	rootDir := t.TempDir()
	nestedBaseDir := filepath.Join(rootDir, "a", "b", "c")
	absolutePath, err := filepath.Abs(filepath.Join(rootDir, "etc", "hosts"))
	if err != nil {
		t.Fatalf("failed to build absolute path fixture: %v", err)
	}

	tests := []struct {
		name    string
		baseDir string
		input   string
		want    string
	}{
		{
			name:    "relative path",
			baseDir: rootDir,
			input:   "ch01.md",
			want:    filepath.Join(rootDir, "ch01.md"),
		},
		{
			name:    "nested relative path",
			baseDir: rootDir,
			input:   "chapters/ch01.md",
			want:    filepath.Join(rootDir, "chapters", "ch01.md"),
		},
		{
			name:    "absolute path",
			baseDir: rootDir,
			input:   absolutePath,
			want:    absolutePath,
		},
		{
			name:    "dot relative path",
			baseDir: rootDir,
			input:   "./ch01.md",
			want:    filepath.Join(rootDir, "ch01.md"),
		},
		{
			name:    "parent directory",
			baseDir: rootDir,
			input:   "../shared/ch01.md",
			want:    filepath.Join(filepath.Dir(rootDir), "shared", "ch01.md"),
		},
		{
			name:    "deeply nested relative",
			baseDir: nestedBaseDir,
			input:   "d/e/f/g.md",
			want:    filepath.Join(nestedBaseDir, "d", "e", "f", "g.md"),
		},
		{
			name:    "dot base dir",
			baseDir: ".",
			input:   "ch01.md",
			want:    "ch01.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.baseDir = tt.baseDir
			got := cfg.ResolvePath(tt.input)
			want := tt.want
			if got != want {
				t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, want)
			}
		})
	}
}

// TestSetBaseDir tests the SetBaseDir method
func TestSetBaseDir(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseDir() != "." {
		t.Errorf("default BaseDir should be '.', got %q", cfg.BaseDir())
	}

	cfg.SetBaseDir("/new/path")
	if cfg.BaseDir() != "/new/path" {
		t.Errorf("after SetBaseDir, BaseDir should be '/new/path', got %q", cfg.BaseDir())
	}
}

// TestFlattenChapters tests the FlattenChapters function
func TestFlattenChapters(t *testing.T) {
	tests := []struct {
		name     string
		chapters []ChapterDef
		wantLen  int
		check    func(t *testing.T, result []ChapterDef)
	}{
		{
			name:     "empty chapters",
			chapters: nil,
			wantLen:  0,
		},
		{
			name: "flat chapters",
			chapters: []ChapterDef{
				{Title: "Ch1", File: "ch1.md"},
				{Title: "Ch2", File: "ch2.md"},
			},
			wantLen: 2,
			check: func(t *testing.T, result []ChapterDef) {
				if result[0].Title != "Ch1" || result[1].Title != "Ch2" {
					t.Error("flat chapters order incorrect")
				}
			},
		},
		{
			name: "nested chapters",
			chapters: []ChapterDef{
				{
					Title: "Part1",
					File:  "part1.md",
					Sections: []ChapterDef{
						{Title: "S1.1", File: "s1.1.md"},
						{Title: "S1.2", File: "s1.2.md"},
					},
				},
				{Title: "Part2", File: "part2.md"},
			},
			wantLen: 4,
			check: func(t *testing.T, result []ChapterDef) {
				if result[0].Title != "Part1" || result[1].Title != "S1.1" || result[2].Title != "S1.2" || result[3].Title != "Part2" {
					t.Error("nested chapters flattening incorrect")
				}
			},
		},
		{
			name: "deeply nested chapters",
			chapters: []ChapterDef{
				{
					Title: "Part1",
					File:  "part1.md",
					Sections: []ChapterDef{
						{
							Title: "Ch1",
							File:  "ch1.md",
							Sections: []ChapterDef{
								{Title: "S1", File: "s1.md"},
							},
						},
					},
				},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenChapters(tt.chapters)
			if len(result) != tt.wantLen {
				t.Errorf("FlattenChapters length = %d, want %d", len(result), tt.wantLen)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestLoadAutoDetectsGlossary tests GLOSSARY.md auto-detection
func TestLoadAutoDetectsGlossary(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	glossaryPath := filepath.Join(tmpDir, "GLOSSARY.md")

	cfgContent := `
book:
  title: "Test Book"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	glossaryContent := `# Glossary

## Term 1
Definition 1

## Term 2
Definition 2
`
	if err := os.WriteFile(glossaryPath, []byte(glossaryContent), 0o644); err != nil {
		t.Fatalf("failed to write GLOSSARY.md: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.GlossaryFile == "" {
		t.Error("should detect GLOSSARY.md")
	}
	if cfg.GlossaryFile != glossaryPath {
		t.Errorf("wrong GlossaryFile path: got %q, want %q", cfg.GlossaryFile, glossaryPath)
	}
}

// TestValidateMissingChapterFile tests validation when chapter file is missing
func TestValidateMissingChapterFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{
		{Title: "Ch1", File: "missing.md"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("validation should fail: file does not exist")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error message should mention missing: %v", err)
	}
}

// TestValidateNestedChaptersMissingFile tests missing nested chapter file
func TestValidateNestedChaptersMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Create main chapter file but not sub-chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "main.md"), []byte("# Main"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{
		{
			Title: "Main",
			File:  "main.md",
			Sections: []ChapterDef{
				{Title: "Sub", File: "missing_sub.md"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("validation should fail: sub-chapter file does not exist")
	}
}

// TestValidateInvalidTheme tests invalid theme validation
func TestValidateInvalidTheme(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Style.Theme = "invalid-theme"

	err := cfg.Validate()
	if err == nil {
		t.Error("invalid theme should fail validation")
	}
	if !strings.Contains(err.Error(), "theme") {
		t.Errorf("error should mention theme: %v", err)
	}
}

// TestValidateInvalidOutputFormat tests invalid output format validation
func TestValidateInvalidOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Output.Formats = []string{"invalid_format"}

	err := cfg.Validate()
	if err == nil {
		t.Error("invalid output format should fail validation")
	}
}

// TestLoadFromSummaryMD tests loading chapters from SUMMARY.md
func TestLoadFromSummaryMD(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")

	// Create minimal config without chapters
	cfgContent := `
book:
  title: "Test Book"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write book.yaml: %v", err)
	}

	// Create SUMMARY.md
	summaryPath := filepath.Join(tmpDir, "SUMMARY.md")
	summaryContent := `# Summary

- [Chapter 1](ch1.md)
- [Chapter 2](ch2.md)
`
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0o644); err != nil {
		t.Fatalf("failed to write SUMMARY.md: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch2.md"), []byte("# Chapter 2"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Chapters) == 0 {
		t.Error("should load chapters from SUMMARY.md")
	}
}

// TestLoadEmptyInput tests loading empty YAML
func TestLoadEmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")

	// Create empty config file
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("loading empty config should return an error")
	}
}

// TestPluginConfig tests plugin configuration
func TestPluginConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test Book"
chapters:
  - title: "Ch1"
    file: "ch1.md"
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.Plugins) != 1 {
		t.Fatalf("wrong plugin count: got %d, want 1", len(cfg.Plugins))
	}

	plugin := cfg.Plugins[0]
	if plugin.Name != "word-count" {
		t.Errorf("wrong plugin name: got %q, want 'word-count'", plugin.Name)
	}
	if plugin.Path != "./plugins/word-count" {
		t.Errorf("wrong plugin path: got %q", plugin.Path)
	}
}

// TestDefaultConfigWatermarkSettings tests watermark default values.
func TestDefaultConfigWatermarkSettings(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Output.Watermark != "" {
		t.Errorf("default watermark should be empty, got %q", cfg.Output.Watermark)
	}
	if cfg.Output.WatermarkOpacity != 0.1 {
		t.Errorf("default watermark opacity should be 0.1, got %f", cfg.Output.WatermarkOpacity)
	}
}

// TestDefaultConfigMarginSettings tests margin default values.
func TestDefaultConfigMarginSettings(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Output.MarginTop != "15mm" {
		t.Errorf("default margin_top should be '15mm', got %q", cfg.Output.MarginTop)
	}
	if cfg.Output.MarginBottom != "15mm" {
		t.Errorf("default margin_bottom should be '15mm', got %q", cfg.Output.MarginBottom)
	}
	if cfg.Output.MarginLeft != "20mm" {
		t.Errorf("default margin_left should be '20mm', got %q", cfg.Output.MarginLeft)
	}
	if cfg.Output.MarginRight != "20mm" {
		t.Errorf("default margin_right should be '20mm', got %q", cfg.Output.MarginRight)
	}
}

// TestDefaultConfigBookmarkSettings tests bookmark generation default value.
func TestDefaultConfigBookmarkSettings(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Output.GenerateBookmarks {
		t.Error("default generate_bookmarks should be true")
	}
}

// TestLoadConfigWithWatermark tests loading a config with watermark settings.
func TestLoadConfigWithWatermark(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test Book"
  author: "Test Author"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"

output:
  watermark: "CONFIDENTIAL"
  watermark_opacity: 0.2
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Output.Watermark != "CONFIDENTIAL" {
		t.Errorf("watermark should be 'CONFIDENTIAL', got %q", cfg.Output.Watermark)
	}
	if cfg.Output.WatermarkOpacity != 0.2 {
		t.Errorf("watermark_opacity should be 0.2, got %f", cfg.Output.WatermarkOpacity)
	}
}

// TestLoadConfigWithCustomMargins tests loading a config with custom margins.
func TestLoadConfigWithCustomMargins(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test Book"
  author: "Test Author"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"

output:
  margin_top: "20mm"
  margin_bottom: "25mm"
  margin_left: "30mm"
  margin_right: "35mm"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Output.MarginTop != "20mm" {
		t.Errorf("margin_top should be '20mm', got %q", cfg.Output.MarginTop)
	}
	if cfg.Output.MarginBottom != "25mm" {
		t.Errorf("margin_bottom should be '25mm', got %q", cfg.Output.MarginBottom)
	}
	if cfg.Output.MarginLeft != "30mm" {
		t.Errorf("margin_left should be '30mm', got %q", cfg.Output.MarginLeft)
	}
	if cfg.Output.MarginRight != "35mm" {
		t.Errorf("margin_right should be '35mm', got %q", cfg.Output.MarginRight)
	}
}

// TestLoadConfigWithGenerateBookmarks tests loading a config with bookmark settings.
func TestLoadConfigWithGenerateBookmarks(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `
book:
  title: "Test Book"
  author: "Test Author"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"

output:
  generate_bookmarks: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0o644); err != nil {
		t.Fatalf("failed to write chapter file: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Output.GenerateBookmarks {
		t.Error("generate_bookmarks should be false")
	}
}
