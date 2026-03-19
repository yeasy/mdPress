package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSummaryBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `# Summary

* [Preface](preface.md)
* [Chapter 1](ch01.md)
* [Chapter 2](ch02.md)
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 3 {
		t.Fatalf("expected 3 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "Preface" {
		t.Errorf("first chapter title: got %q", chapters[0].Title)
	}
	if chapters[0].File != "preface.md" {
		t.Errorf("first chapter file: got %q", chapters[0].File)
	}
}

func TestParseSummaryNested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `# Table of Contents

* [Part 1](part1/README.md)
  * [Section 1.1](part1/s1.md)
  * [Section 1.2](part1/s2.md)
    * [Sub 1.2.1](part1/sub.md)
* [Part 2](part2/README.md)
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 top-level, got %d", len(chapters))
	}
	if len(chapters[0].Sections) != 2 {
		t.Fatalf("Part 1 should have 2 sections, got %d", len(chapters[0].Sections))
	}
	if len(chapters[0].Sections[1].Sections) != 1 {
		t.Errorf("Section 1.2 should have 1 sub-section, got %d", len(chapters[0].Sections[1].Sections))
	}
}

func TestParseSummaryEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	os.WriteFile(path, []byte("# Summary\n\n"), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestParseSummarySkipAnchors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `* [Intro](#intro)
* [Chapter 1](ch01.md)
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, _ := ParseSummary(path)
	if len(chapters) != 1 {
		t.Errorf("anchor links should be skipped, got %d chapters", len(chapters))
	}
}

func TestParseSummaryNonExistent(t *testing.T) {
	_, err := ParseSummary("/nonexistent/SUMMARY.md")
	if err == nil {
		t.Error("should fail for non-existent file")
	}
}

func TestParseSummaryWithTabs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := "* [A](a.md)\n\t* [B](b.md)\n"
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 top-level, got %d", len(chapters))
	}
	if len(chapters[0].Sections) != 1 {
		t.Errorf("A should have 1 sub, got %d", len(chapters[0].Sections))
	}
}

func TestLoadWithSummary(t *testing.T) {
	dir := t.TempDir()

	// book.yaml without chapters
	yaml := `book:
  title: "Test"
`
	os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644)

	// SUMMARY.md provides chapters
	summary := "* [Intro](intro.md)\n* [Ch1](ch1.md)\n"
	os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0644)

	// Create chapter files
	for _, file := range []string{"intro.md", "ch1.md"} {
		os.WriteFile(filepath.Join(dir, file), []byte("# Content"), 0644)
	}

	cfg, err := Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("should load 2 chapters from SUMMARY.md, got %d", len(cfg.Chapters))
	}
}

func TestLoadDetectsGlossary(t *testing.T) {
	dir := t.TempDir()
	yaml := "book:\n  title: Test\nchapters:\n  - title: ch\n    file: ch.md\n"
	os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0644)
	os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("## API\nfoo\n"), 0644)

	// Create chapter file
	os.WriteFile(filepath.Join(dir, "ch.md"), []byte("# Content"), 0644)

	cfg, err := Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.GlossaryFile == "" {
		t.Error("should detect GLOSSARY.md")
	}
}

// TestParseSummarySpecialChars 测试标题中包含特殊字符的 SUMMARY
func TestParseSummarySpecialChars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `# Summary

* [Chapter & Guide](ch1.md)
* [API < REST >](ch2.md)
* [Quote "marks"](ch3.md)
* [Math: a & b](ch4.md)
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 4 {
		t.Fatalf("expected 4 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "Chapter & Guide" {
		t.Errorf("expected 'Chapter & Guide', got %q", chapters[0].Title)
	}
	if chapters[1].Title != "API < REST >" {
		t.Errorf("expected 'API < REST >', got %q", chapters[1].Title)
	}
	if chapters[2].Title != "Quote \"marks\"" {
		t.Errorf("expected 'Quote \"marks\"', got %q", chapters[2].Title)
	}
	if chapters[3].Title != "Math: a & b" {
		t.Errorf("expected 'Math: a & b', got %q", chapters[3].Title)
	}
}

// TestParseSummaryDeepNesting 测试 4+ 层嵌套
func TestParseSummaryDeepNesting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `# Summary

* [Level 1](l1.md)
  * [Level 2](l2.md)
    * [Level 3](l3.md)
      * [Level 4](l4.md)
        * [Level 5](l5.md)
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 1 {
		t.Fatalf("expected 1 top-level, got %d", len(chapters))
	}
	if chapters[0].Title != "Level 1" {
		t.Errorf("expected 'Level 1', got %q", chapters[0].Title)
	}

	// 验证深层嵌套结构
	level2 := chapters[0].Sections
	if len(level2) != 1 {
		t.Fatalf("expected 1 level 2, got %d", len(level2))
	}
	if level2[0].Title != "Level 2" {
		t.Errorf("expected 'Level 2', got %q", level2[0].Title)
	}

	level3 := level2[0].Sections
	if len(level3) != 1 {
		t.Fatalf("expected 1 level 3, got %d", len(level3))
	}
	if level3[0].Title != "Level 3" {
		t.Errorf("expected 'Level 3', got %q", level3[0].Title)
	}

	level4 := level3[0].Sections
	if len(level4) != 1 {
		t.Fatalf("expected 1 level 4, got %d", len(level4))
	}
	if level4[0].Title != "Level 4" {
		t.Errorf("expected 'Level 4', got %q", level4[0].Title)
	}

	level5 := level4[0].Sections
	if len(level5) != 1 {
		t.Fatalf("expected 1 level 5, got %d", len(level5))
	}
	if level5[0].Title != "Level 5" {
		t.Errorf("expected 'Level 5', got %q", level5[0].Title)
	}
}

// TestParseSummaryMixedIndent 测试混合空格和 tab 的缩进
func TestParseSummaryMixedIndent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	// 混合使用空格和 tab - 第一级 2 个空格，第二级 1 个 tab（算作 2 个空格）
	content := "* [Part A](a.md)\n  * [Section A1](a1.md)\n\t* [Section A2](a2.md)\n* [Part B](b.md)\n"
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 top-level, got %d", len(chapters))
	}
	if len(chapters[0].Sections) != 2 {
		t.Errorf("Part A should have 2 sections, got %d", len(chapters[0].Sections))
	}
	if chapters[0].Sections[0].Title != "Section A1" {
		t.Errorf("expected 'Section A1', got %q", chapters[0].Sections[0].Title)
	}
	if chapters[0].Sections[1].Title != "Section A2" {
		t.Errorf("expected 'Section A2', got %q", chapters[0].Sections[1].Title)
	}
}

// TestCountIndent 单元测试 countIndent 函数
func TestCountIndent(t *testing.T) {
	tests := []struct {
		name string
		line string
		want int
	}{
		{
			name: "no indent",
			line: "* [Title](file.md)",
			want: 0,
		},
		{
			name: "two spaces",
			line: "  * [Title](file.md)",
			want: 2,
		},
		{
			name: "four spaces",
			line: "    * [Title](file.md)",
			want: 4,
		},
		{
			name: "one tab",
			line: "\t* [Title](file.md)",
			want: 2,
		},
		{
			name: "two tabs",
			line: "\t\t* [Title](file.md)",
			want: 4,
		},
		{
			name: "mixed tab and space",
			line: "\t  * [Title](file.md)",
			want: 4, // 1 tab (2) + 2 spaces
		},
		{
			name: "eight spaces",
			line: "        * [Title](file.md)",
			want: 8,
		},
		{
			name: "space after non-space",
			line: "  x  spaces",
			want: 2, // 只计算前导空格/tab
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countIndent(tt.line)
			if got != tt.want {
				t.Errorf("countIndent(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}

// TestParseSummaryNoLinks 测试不包含 markdown 链接的文件
func TestParseSummaryNoLinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SUMMARY.md")
	content := `# Summary

This is just plain text, no markdown links here.

Some plain text item without link
Another item

And some more content...
`
	os.WriteFile(path, []byte(content), 0644)

	chapters, err := ParseSummary(path)
	if err != nil {
		t.Fatalf("ParseSummary failed: %v", err)
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters (no links), got %d", len(chapters))
	}
}
