package cmd

import (
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/markdown"
)

func TestParsedChapterCacheRoundTrip(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	chapterPath := filepath.Join("/book", "chapter.md")
	content := "# Title\n\nbody"
	codeTheme := "monokai"
	want := &cachedParsedChapter{
		HTML: "<h1>Title</h1>",
		Headings: []markdown.HeadingInfo{
			{Level: 1, Text: "Title", ID: "title"},
		},
		Diagnostics: []markdown.Diagnostic{
			{Rule: "example", Line: 1, Column: 1, Message: "msg"},
		},
	}

	if err := storeParsedChapterCache(chapterPath, content, codeTheme, want); err != nil {
		t.Fatalf("storeParsedChapterCache 失败: %v", err)
	}

	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil {
		t.Fatalf("loadParsedChapterCache 失败: %v", err)
	}
	if !ok {
		t.Fatal("应命中缓存")
	}
	if got.HTML != want.HTML {
		t.Fatalf("HTML = %q, want %q", got.HTML, want.HTML)
	}
	if len(got.Headings) != 1 || got.Headings[0].Text != "Title" {
		t.Fatalf("Headings = %#v", got.Headings)
	}
	if len(got.Diagnostics) != 1 || got.Diagnostics[0].Rule != "example" {
		t.Fatalf("Diagnostics = %#v", got.Diagnostics)
	}
}

func TestParsedChapterCacheDisabled(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())
	t.Setenv("MDPRESS_DISABLE_CACHE", "1")

	chapterPath := filepath.Join("/book", "chapter.md")
	content := "# Title\n\nbody"
	codeTheme := "monokai"
	payload := &cachedParsedChapter{HTML: "<h1>Title</h1>"}

	if err := storeParsedChapterCache(chapterPath, content, codeTheme, payload); err != nil {
		t.Fatalf("storeParsedChapterCache 在 no-cache 模式下不应报错: %v", err)
	}

	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil {
		t.Fatalf("loadParsedChapterCache 在 no-cache 模式下不应报错: %v", err)
	}
	if ok || got != nil {
		t.Fatal("禁用缓存时不应命中解析缓存")
	}
}
