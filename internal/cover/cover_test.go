package cover

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/pkg/utils"
)

// TestNewCoverGenerator verifies cover generator creation.
func TestNewCoverGenerator(t *testing.T) {
	meta := config.BookMeta{Title: "Test Book"}
	gen := NewCoverGenerator(meta)
	if gen == nil {
		t.Fatal("NewCoverGenerator returned nil")
	}
}

// TestRenderHTMLBasic verifies basic cover rendering.
func TestRenderHTMLBasic(t *testing.T) {
	meta := config.BookMeta{
		Title:   "My Book",
		Author:  "Author",
		Version: "1.0.0",
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "My Book") {
		t.Error("cover should include the title")
	}
	if !strings.Contains(html, "Author") {
		t.Error("cover should include the author")
	}
	if !strings.Contains(html, "1.0.0") {
		t.Error("cover should include the version")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("cover should be a complete HTML document")
	}
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should contain the cover-page class")
	}
}

// TestRenderHTMLWithSubtitle verifies subtitle rendering.
func TestRenderHTMLWithSubtitle(t *testing.T) {
	meta := config.BookMeta{
		Title:    "Main Title",
		Subtitle: "Subtitle Content",
		Author:   "Author",
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Subtitle Content") {
		t.Error("cover should include the subtitle")
	}
	if !strings.Contains(html, "cover-subtitle") {
		t.Error("cover should include the subtitle class")
	}
}

// TestRenderHTMLWithBackground verifies background color rendering.
func TestRenderHTMLWithBackground(t *testing.T) {
	meta := config.BookMeta{
		Title: "Test",
		Cover: config.CoverMeta{Background: "#1a1a2e"},
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "#1a1a2e") {
		t.Error("cover should include the configured background color")
	}
}

// TestRenderHTMLWithCoverImage verifies cover image rendering.
func TestRenderHTMLWithCoverImage(t *testing.T) {
	meta := config.BookMeta{
		Title: "Test",
		Cover: config.CoverMeta{Image: "cover.png"},
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "cover.png") {
		t.Error("cover should include the configured cover image")
	}
	if !strings.Contains(html, "background-image") {
		t.Error("cover should include background image styles")
	}
}

// TestRenderHTMLDefaultGradient verifies the default clean white background.
func TestRenderHTMLDefaultGradient(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "background-color: #ffffff") {
		t.Error("clean white background should be used when no cover background is configured")
	}
}

// TestRenderHTMLEmptyTitle verifies empty-title handling.
func TestRenderHTMLEmptyTitle(t *testing.T) {
	meta := config.BookMeta{Title: "", Author: "Author"}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	// Rendering should still succeed.
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should still render when the title is empty")
	}
}

// TestRenderHTMLEscaping verifies HTML escaping.
func TestRenderHTMLEscaping(t *testing.T) {
	meta := config.BookMeta{
		Title:  "<script>alert('xss')</script>",
		Author: `"injected"`,
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if strings.Contains(html, "<script>") {
		t.Error("HTML tags should be escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("escaped script tag should be present")
	}
	if !strings.Contains(html, "&quot;injected&quot;") {
		t.Error("quotes should be escaped")
	}
}

// TestRenderHTMLContainsDate verifies date rendering.
func TestRenderHTMLContainsDate(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Date") {
		t.Error("cover should contain the date label")
	}
	if !strings.Contains(html, "-") {
		t.Error("cover should contain an ISO-like date format")
	}
}

// TestRenderHTMLStructure verifies the HTML structure.
func TestRenderHTMLStructure(t *testing.T) {
	meta := config.BookMeta{Title: "Test", Author: "Author", Version: "1.0"}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	requiredTags := []string{
		"<!DOCTYPE html>", "<html", "</html>",
		"<head>", "</head>",
		"<body>", "</body>",
		"<style>", "</style>",
		"cover-page", "cover-content", "cover-title",
	}

	for _, tag := range requiredTags {
		if !strings.Contains(html, tag) {
			t.Errorf("cover HTML should contain %q", tag)
		}
	}
}

// TestEscapeHTMLCover verifies HTML escaping in the cover package.
func TestEscapeHTMLCover(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal text", "normal text"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{`he said "hi"`, "he said &quot;hi&quot;"},
		{"a & b", "a &amp; b"},
	}

	for _, tt := range tests {
		got := utils.EscapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestEscapeURL verifies URL escaping.
func TestEscapeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.png", "normal.png"},
		{"it's.png", "it%27s.png"},
		{"file with).png", "file with%29.png"},
		{`path"quote.png`, `path%22quote.png`},
	}

	for _, tt := range tests {
		got := escapeURL(tt.input)
		if got != tt.want {
			t.Errorf("escapeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCoverLabelsLocalization verifies cover label localization.
func TestCoverLabelsLocalization(t *testing.T) {
	tests := []struct {
		lang                          string
		wantAuthor, wantVer, wantDate string
	}{
		{"en-US", "Author", "Version", "Date"},
		{"en", "Author", "Version", "Date"},
		{"", "Author", "Version", "Date"},
		{"zh-CN", "作者", "版本", "日期"},
		{"zh", "作者", "版本", "日期"},
		{"ja-JP", "著者", "バージョン", "日付"},
		{"ko-KR", "저자", "버전", "날짜"},
	}
	for _, tt := range tests {
		a, v, d := coverLabels(tt.lang)
		if a != tt.wantAuthor || v != tt.wantVer || d != tt.wantDate {
			t.Errorf("coverLabels(%q) = (%q,%q,%q), want (%q,%q,%q)",
				tt.lang, a, v, d, tt.wantAuthor, tt.wantVer, tt.wantDate)
		}
	}
}

// TestRenderHTMLChineseLabels verifies Chinese cover labels.
func TestRenderHTMLChineseLabels(t *testing.T) {
	meta := config.BookMeta{
		Title:    "测试书名",
		Author:   "张三",
		Version:  "1.0",
		Language: "zh-CN",
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "作者") {
		t.Error("Chinese cover should use '作者' label")
	}
	if !strings.Contains(html, "版本") {
		t.Error("Chinese cover should use '版本' label")
	}
	if !strings.Contains(html, "日期") {
		t.Error("Chinese cover should use '日期' label")
	}
	if strings.Contains(html, ">Author<") {
		t.Error("Chinese cover should not use English 'Author' label")
	}
}

// TestRenderHTMLEnglishLabels verifies English cover labels (default).
func TestRenderHTMLEnglishLabels(t *testing.T) {
	meta := config.BookMeta{
		Title:    "Test Book",
		Author:   "John",
		Version:  "1.0",
		Language: "en-US",
	}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Author") {
		t.Error("English cover should use 'Author' label")
	}
	if !strings.Contains(html, "Version") {
		t.Error("English cover should use 'Version' label")
	}
	if !strings.Contains(html, "Date") {
		t.Error("English cover should use 'Date' label")
	}
}

// TestRenderHTMLNoAuthor verifies rendering without an author.
func TestRenderHTMLNoAuthor(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta)
	html := gen.RenderHTML()

	// Rendering should succeed without an author block.
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should render successfully")
	}
}
