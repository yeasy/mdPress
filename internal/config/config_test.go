package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultConfig 测试默认配置的合理性
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Book.Title != "Untitled Book" {
		t.Errorf("默认标题错误: got %q, want %q", cfg.Book.Title, "Untitled Book")
	}
	if cfg.Book.Language != "zh-CN" {
		t.Errorf("默认语言错误: got %q, want %q", cfg.Book.Language, "zh-CN")
	}
	if cfg.Style.PageSize != "A4" {
		t.Errorf("默认页面尺寸错误: got %q, want %q", cfg.Style.PageSize, "A4")
	}
	if cfg.Style.Theme != "technical" {
		t.Errorf("默认主题错误: got %q, want %q", cfg.Style.Theme, "technical")
	}
	if cfg.Style.LineHeight != 1.6 {
		t.Errorf("默认行高错误: got %f, want %f", cfg.Style.LineHeight, 1.6)
	}
	if cfg.Output.Filename != "output.pdf" {
		t.Errorf("默认输出文件名错误: got %q, want %q", cfg.Output.Filename, "output.pdf")
	}
	if !cfg.Output.TOC {
		t.Error("默认应启用目录")
	}
	if !cfg.Output.Cover {
		t.Error("默认应启用封面")
	}
	if cfg.Style.Margin.Top != 25 {
		t.Errorf("默认上边距错误: got %f, want %f", cfg.Style.Margin.Top, 25.0)
	}
}

// TestLoadValidConfig 测试加载合法配置文件
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("写入章节文件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch02.md"), []byte("# Chapter 2"), 0644); err != nil {
		t.Fatalf("写入章节文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Book.Title != "测试图书" {
		t.Errorf("标题错误: got %q, want %q", cfg.Book.Title, "测试图书")
	}
	if cfg.Book.Author != "测试作者" {
		t.Errorf("作者错误: got %q, want %q", cfg.Book.Author, "测试作者")
	}
	if cfg.Book.Version != "2.0.0" {
		t.Errorf("版本错误: got %q, want %q", cfg.Book.Version, "2.0.0")
	}
	if cfg.Book.Language != "en-US" {
		t.Errorf("语言错误: got %q, want %q", cfg.Book.Language, "en-US")
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("章节数错误: got %d, want %d", len(cfg.Chapters), 2)
	}
	if cfg.Style.Theme != "elegant" {
		t.Errorf("主题错误: got %q, want %q", cfg.Style.Theme, "elegant")
	}
	if cfg.Style.PageSize != "A5" {
		t.Errorf("页面尺寸错误: got %q, want %q", cfg.Style.PageSize, "A5")
	}
	if cfg.Output.Filename != "test.pdf" {
		t.Errorf("输出文件名错误: got %q, want %q", cfg.Output.Filename, "test.pdf")
	}
	if cfg.Output.TOC {
		t.Error("TOC 应被禁用")
	}
	if cfg.Output.Cover {
		t.Error("封面应被禁用")
	}
}

// TestLoadNonExistentFile 测试加载不存在的文件
func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/book.yaml")
	if err == nil {
		t.Error("加载不存在的文件应返回错误")
	}
}

// TestLoadInvalidYAML 测试加载无效 YAML
func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	content := `invalid yaml: [broken: {`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("加载无效 YAML 应返回错误")
	}
}

// TestValidateEmptyTitle 测试空标题验证
func TestValidateEmptyTitle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = ""
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}

	err := cfg.Validate()
	if err == nil {
		t.Error("空标题应验证失败")
	}
}

// TestValidateNoChapters 测试无章节验证
func TestValidateNoChapters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = nil

	err := cfg.Validate()
	if err == nil {
		t.Error("无章节应验证失败")
	}
}

// TestValidateEmptyChapterFile 测试章节文件为空
func TestValidateEmptyChapterFile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: ""}}

	err := cfg.Validate()
	if err == nil {
		t.Error("空文件路径应验证失败")
	}
}

// TestValidateInvalidPageSize 测试无效页面尺寸
func TestValidateInvalidPageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Style.PageSize = "INVALID"

	err := cfg.Validate()
	if err == nil {
		t.Error("无效页面尺寸应验证失败")
	}
}

// TestValidateValidPageSizes 测试所有合法页面尺寸
func TestValidateValidPageSizes(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	sizes := []string{"A4", "A5", "Letter", "Legal", "B5"}
	for _, size := range sizes {
		cfg := DefaultConfig()
		cfg.baseDir = tmpDir
		cfg.Book.Title = "Test"
		cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
		cfg.Style.PageSize = size

		if err := cfg.Validate(); err != nil {
			t.Errorf("页面尺寸 %q 应通过验证，得到错误: %v", size, err)
		}
	}
}

// TestResolvePath 测试路径解析
func TestResolvePath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.baseDir = "/home/user/book"

	tests := []struct {
		input string
		want  string
	}{
		{"ch01.md", "/home/user/book/ch01.md"},
		{"sub/ch01.md", "/home/user/book/sub/ch01.md"},
		{"/absolute/path.md", "/absolute/path.md"},
	}

	for _, tt := range tests {
		got := cfg.ResolvePath(tt.input)
		if got != tt.want {
			t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestBaseDir 测试基础目录
func TestBaseDir(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseDir() != "." {
		t.Errorf("默认 BaseDir 应为 '.', got %q", cfg.BaseDir())
	}

	cfg.baseDir = "/test/path"
	if cfg.BaseDir() != "/test/path" {
		t.Errorf("BaseDir 应为 '/test/path', got %q", cfg.BaseDir())
	}
}

// TestLoadSetsBaseDir 测试加载时设置 BaseDir
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.BaseDir() != tmpDir {
		t.Errorf("BaseDir 应为 %q, got %q", tmpDir, cfg.BaseDir())
	}
}

// TestNestedChapters 测试嵌套子章节
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	// Create chapter files
	for _, file := range []string{"part1.md", "s1.md", "s2.md"} {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("# Section"), 0644); err != nil {
			t.Fatalf("创建文件失败: %v", err)
		}
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(cfg.Chapters) != 1 {
		t.Fatalf("顶层章节数错误: got %d, want %d", len(cfg.Chapters), 1)
	}
	if len(cfg.Chapters[0].Sections) != 2 {
		t.Errorf("子章节数错误: got %d, want %d", len(cfg.Chapters[0].Sections), 2)
	}
}

// TestDefaultValuesPreserved 测试部分配置保留默认值
func TestDefaultValuesPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	// 只设置最少的必要字段
	content := `
book:
  title: "Minimal Config"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证默认值被保留
	if cfg.Style.PageSize != "A4" {
		t.Errorf("默认 PageSize 应保留: got %q", cfg.Style.PageSize)
	}
	if cfg.Output.Filename != "output.pdf" {
		t.Errorf("默认 Filename 应保留: got %q", cfg.Output.Filename)
	}
	if !cfg.Output.TOC {
		t.Error("默认 TOC 应保留为 true")
	}
}

// TestValidateTableDriven 表驱动测试，覆盖多个验证场景
func TestValidateTableDriven(t *testing.T) {
	// Create temp directory and file for successful validation cases
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
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

// TestLoadConfigOverrideDefaults 测试 YAML 值正确覆盖默认值
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证覆盖的值
	if cfg.Book.Language != "fr-FR" {
		t.Errorf("Language 未被覆盖: got %q, want %q", cfg.Book.Language, "fr-FR")
	}
	if cfg.Style.LineHeight != 2.0 {
		t.Errorf("LineHeight 未被覆盖: got %f, want %f", cfg.Style.LineHeight, 2.0)
	}
	if cfg.Style.PageSize != "A5" {
		t.Errorf("PageSize 未被覆盖: got %q, want %q", cfg.Style.PageSize, "A5")
	}
	if cfg.Style.Theme != "elegant" {
		t.Errorf("Theme 未被覆盖: got %q, want %q", cfg.Style.Theme, "elegant")
	}
	if cfg.Style.Margin.Top != 30 {
		t.Errorf("Margin.Top 未被覆盖: got %f, want %f", cfg.Style.Margin.Top, 30.0)
	}
	if cfg.Style.Margin.Bottom != 35 {
		t.Errorf("Margin.Bottom 未被覆盖: got %f, want %f", cfg.Style.Margin.Bottom, 35.0)
	}
	if cfg.Output.Filename != "custom.pdf" {
		t.Errorf("Filename 未被覆盖: got %q, want %q", cfg.Output.Filename, "custom.pdf")
	}
	if cfg.Output.TOC {
		t.Error("TOC 未被正确覆盖为 false")
	}
	if cfg.Output.Cover {
		t.Error("Cover 未被正确覆盖为 false")
	}
}

// TestLoadAutoDetectsLangs 测试 LANGS.md 自动检测
func TestLoadAutoDetectsLangs(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")
	langsPath := filepath.Join(tmpDir, "LANGS.md")

	// 创建配置文件
	cfgContent := `
book:
  title: "Test Book"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	// Create chapter file
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	// 创建 LANGS.md
	langsContent := `# Languages

- [English](en/)
- [中文](zh/)
`
	if err := os.WriteFile(langsPath, []byte(langsContent), 0644); err != nil {
		t.Fatalf("写入 LANGS.md 失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.LangsFile == "" {
		t.Error("应该检测到 LANGS.md")
	}
	if cfg.LangsFile != langsPath {
		t.Errorf("LangsFile 路径错误: got %q, want %q", cfg.LangsFile, langsPath)
	}
}

// TestResolvePathTableDriven 表驱动的路径解析测试，包含边界情况
func TestResolvePathTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		input   string
		want    string
	}{
		{
			name:    "relative path",
			baseDir: "/home/user/book",
			input:   "ch01.md",
			want:    "/home/user/book/ch01.md",
		},
		{
			name:    "nested relative path",
			baseDir: "/home/user/book",
			input:   "chapters/ch01.md",
			want:    "/home/user/book/chapters/ch01.md",
		},
		{
			name:    "absolute path",
			baseDir: "/home/user/book",
			input:   "/etc/hosts",
			want:    "/etc/hosts",
		},
		{
			name:    "dot relative path",
			baseDir: "/home/user/book",
			input:   "./ch01.md",
			want:    "/home/user/book/ch01.md",
		},
		{
			name:    "parent directory",
			baseDir: "/home/user/book",
			input:   "../shared/ch01.md",
			want:    "/home/user/shared/ch01.md",
		},
		{
			name:    "deeply nested relative",
			baseDir: "/a/b/c",
			input:   "d/e/f/g.md",
			want:    "/a/b/c/d/e/f/g.md",
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
			if got != tt.want {
				t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestSetBaseDir 测试 SetBaseDir 方法
func TestSetBaseDir(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaseDir() != "." {
		t.Errorf("默认 BaseDir 应为 '.', got %q", cfg.BaseDir())
	}

	cfg.SetBaseDir("/new/path")
	if cfg.BaseDir() != "/new/path" {
		t.Errorf("SetBaseDir 后 BaseDir 应为 '/new/path', got %q", cfg.BaseDir())
	}
}

// TestFlattenChapters 测试 FlattenChapters 函数
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

// TestLoadAutoDetectsGlossary 测试 GLOSSARY.md 自动检测
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
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	glossaryContent := `# Glossary

## Term 1
Definition 1

## Term 2
Definition 2
`
	if err := os.WriteFile(glossaryPath, []byte(glossaryContent), 0644); err != nil {
		t.Fatalf("写入 GLOSSARY.md 失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.GlossaryFile == "" {
		t.Error("应该检测到 GLOSSARY.md")
	}
	if cfg.GlossaryFile != glossaryPath {
		t.Errorf("GlossaryFile 路径错误: got %q, want %q", cfg.GlossaryFile, glossaryPath)
	}
}

// TestValidateMissingChapterFile 测试章节文件不存在的验证
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
		t.Error("验证应该失败：文件不存在")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("错误消息应提及 missing: %v", err)
	}
}

// TestValidateNestedChaptersMissingFile 测试嵌套章节文件不存在
func TestValidateNestedChaptersMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建主章节文件但不创建子章节文件
	if err := os.WriteFile(filepath.Join(tmpDir, "main.md"), []byte("# Main"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
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
		t.Error("验证应该失败：子章节文件不存在")
	}
}

// TestValidateInvalidTheme 测试无效主题验证
func TestValidateInvalidTheme(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Style.Theme = "invalid-theme"

	err := cfg.Validate()
	if err == nil {
		t.Error("无效主题应验证失败")
	}
	if !strings.Contains(err.Error(), "theme") {
		t.Errorf("错误应提及 theme: %v", err)
	}
}

// TestValidateInvalidOutputFormat 测试无效输出格式验证
func TestValidateInvalidOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg := DefaultConfig()
	cfg.baseDir = tmpDir
	cfg.Book.Title = "Test"
	cfg.Chapters = []ChapterDef{{Title: "ch1", File: "ch1.md"}}
	cfg.Output.Formats = []string{"invalid_format"}

	err := cfg.Validate()
	if err == nil {
		t.Error("无效输出格式应验证失败")
	}
}

// TestLoadFromSummaryMD 测试从 SUMMARY.md 加载章节
func TestLoadFromSummaryMD(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")

	// 创建最小的配置，不包含 chapters
	cfgContent := `
book:
  title: "Test Book"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	// 创建 SUMMARY.md
	summaryPath := filepath.Join(tmpDir, "SUMMARY.md")
	summaryContent := `# Summary

- [Chapter 1](ch1.md)
- [Chapter 2](ch2.md)
`
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0644); err != nil {
		t.Fatalf("写入 SUMMARY.md 失败: %v", err)
	}

	// 创建章节文件
	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "ch2.md"), []byte("# Chapter 2"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(cfg.Chapters) == 0 {
		t.Error("应该从 SUMMARY.md 加载章节")
	}
}

// TestLoadEmptyInput 测试加载空 YAML
func TestLoadEmptyInput(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "book.yaml")

	// 创建空的配置文件
	if err := os.WriteFile(cfgPath, []byte(""), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("加载空配置应返回错误")
	}
}

// TestPluginConfig 测试插件配置
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "ch1.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if len(cfg.Plugins) != 1 {
		t.Fatalf("插件数量错误: got %d, want 1", len(cfg.Plugins))
	}

	plugin := cfg.Plugins[0]
	if plugin.Name != "word-count" {
		t.Errorf("插件名错误: got %q, want 'word-count'", plugin.Name)
	}
	if plugin.Path != "./plugins/word-count" {
		t.Errorf("插件路径错误: got %q", plugin.Path)
	}
}

// TestDefaultConfigWatermarkSettings tests watermark default values.
func TestDefaultConfigWatermarkSettings(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Output.Watermark != "" {
		t.Errorf("默认 watermark 应为空, got %q", cfg.Output.Watermark)
	}
	if cfg.Output.WatermarkOpacity != 0.1 {
		t.Errorf("默认 watermark opacity 应为 0.1, got %f", cfg.Output.WatermarkOpacity)
	}
}

// TestDefaultConfigMarginSettings tests margin default values.
func TestDefaultConfigMarginSettings(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Output.MarginTop != "15mm" {
		t.Errorf("默认 margin_top 应为 '15mm', got %q", cfg.Output.MarginTop)
	}
	if cfg.Output.MarginBottom != "15mm" {
		t.Errorf("默认 margin_bottom 应为 '15mm', got %q", cfg.Output.MarginBottom)
	}
	if cfg.Output.MarginLeft != "20mm" {
		t.Errorf("默认 margin_left 应为 '20mm', got %q", cfg.Output.MarginLeft)
	}
	if cfg.Output.MarginRight != "20mm" {
		t.Errorf("默认 margin_right 应为 '20mm', got %q", cfg.Output.MarginRight)
	}
}

// TestDefaultConfigBookmarkSettings tests bookmark generation default value.
func TestDefaultConfigBookmarkSettings(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Output.GenerateBookmarks {
		t.Error("默认 generate_bookmarks 应为 true")
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("写入章节文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Output.Watermark != "CONFIDENTIAL" {
		t.Errorf("watermark 应为 'CONFIDENTIAL', got %q", cfg.Output.Watermark)
	}
	if cfg.Output.WatermarkOpacity != 0.2 {
		t.Errorf("watermark_opacity 应为 0.2, got %f", cfg.Output.WatermarkOpacity)
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("写入章节文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Output.MarginTop != "20mm" {
		t.Errorf("margin_top 应为 '20mm', got %q", cfg.Output.MarginTop)
	}
	if cfg.Output.MarginBottom != "25mm" {
		t.Errorf("margin_bottom 应为 '25mm', got %q", cfg.Output.MarginBottom)
	}
	if cfg.Output.MarginLeft != "30mm" {
		t.Errorf("margin_left 应为 '30mm', got %q", cfg.Output.MarginLeft)
	}
	if cfg.Output.MarginRight != "35mm" {
		t.Errorf("margin_right 应为 '35mm', got %q", cfg.Output.MarginRight)
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
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}

	// Create chapter files
	if err := os.WriteFile(filepath.Join(tmpDir, "ch01.md"), []byte("# Chapter 1"), 0644); err != nil {
		t.Fatalf("写入章节文件失败: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Output.GenerateBookmarks {
		t.Error("generate_bookmarks 应为 false")
	}
}
