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
		t.Fatalf("storeParsedChapterCache failed: %v", err)
	}

	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil {
		t.Fatalf("loadParsedChapterCache failed: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
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
		t.Fatalf("storeParsedChapterCache should not error in no-cache mode: %v", err)
	}

	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil {
		t.Fatalf("loadParsedChapterCache should not error in no-cache mode: %v", err)
	}
	if ok || got != nil {
		t.Fatal("cache should not be hit when disabled")
	}
}

func TestParsedChapterCacheMiss(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	chapterPath := filepath.Join("/book", "missing-chapter.md")
	content := "# Missing Content"
	codeTheme := "monokai"

	// Load non-existent cache
	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil {
		t.Fatalf("loadParsedChapterCache should not error on miss: %v", err)
	}
	if ok || got != nil {
		t.Fatal("cache miss should return ok=false and nil data")
	}
}

func TestParsedChapterCacheInvalidation(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	chapterPath := filepath.Join("/book", "chapter.md")
	codeTheme := "monokai"

	// Store cache with first content
	content1 := "# Title 1\n\nbody1"
	payload1 := &cachedParsedChapter{HTML: "<h1>Title 1</h1>"}
	if err := storeParsedChapterCache(chapterPath, content1, codeTheme, payload1); err != nil {
		t.Fatalf("first store failed: %v", err)
	}

	// Load it back
	_, ok1, err := loadParsedChapterCache(chapterPath, content1, codeTheme)
	if err != nil || !ok1 {
		t.Fatal("first load should hit cache")
	}

	// Change content and try to load (should be cache miss because hash changed)
	content2 := "# Title 2\n\nbody2"
	got2, ok2, err := loadParsedChapterCache(chapterPath, content2, codeTheme)
	if err != nil {
		t.Fatalf("second load should not error: %v", err)
	}
	if ok2 || got2 != nil {
		t.Fatal("changed content should cause cache miss")
	}
}

func TestParsedChapterCacheEmptyContent(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	chapterPath := filepath.Join("/book", "empty.md")
	content := ""
	codeTheme := "monokai"
	payload := &cachedParsedChapter{HTML: ""}

	// Store and load empty content
	if err := storeParsedChapterCache(chapterPath, content, codeTheme, payload); err != nil {
		t.Fatalf("store empty content failed: %v", err)
	}

	got, ok, err := loadParsedChapterCache(chapterPath, content, codeTheme)
	if err != nil || !ok || got == nil {
		t.Fatal("empty content should be cacheable")
	}
	if got.HTML != "" {
		t.Fatalf("empty HTML mismatch: got %q", got.HTML)
	}
}

func TestParsedChapterCacheCodeThemeDifference(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	chapterPath := filepath.Join("/book", "chapter.md")
	content := "# Title\n\nbody"

	// Store with one theme
	theme1 := "monokai"
	payload1 := &cachedParsedChapter{HTML: "<h1>Title</h1>"}
	if err := storeParsedChapterCache(chapterPath, content, theme1, payload1); err != nil {
		t.Fatalf("store with theme1 failed: %v", err)
	}

	// Store with different theme should be separate cache entry
	theme2 := "github"
	payload2 := &cachedParsedChapter{HTML: "<h1 class='github'>Title</h1>"}
	if err := storeParsedChapterCache(chapterPath, content, theme2, payload2); err != nil {
		t.Fatalf("store with theme2 failed: %v", err)
	}

	// Load with theme1 should get payload1
	got1, ok1, err := loadParsedChapterCache(chapterPath, content, theme1)
	if err != nil || !ok1 || got1.HTML != "<h1>Title</h1>" {
		t.Fatal("theme1 cache should be independent")
	}

	// Load with theme2 should get payload2
	got2, ok2, err := loadParsedChapterCache(chapterPath, content, theme2)
	if err != nil || !ok2 || got2.HTML != "<h1 class='github'>Title</h1>" {
		t.Fatal("theme2 cache should be independent")
	}
}
