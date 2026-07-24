// extra_funcs_test.go adds coverage for pure utility functions in the cmd package
// that were previously untested: format predicates, multilingual helpers,
// navigation converters, build-issue reporting, and theme listing.
package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/output"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
)

// ---------------------------------------------------------------------------
// containsBuildFormat
// ---------------------------------------------------------------------------

func TestContainsBuildFormat(t *testing.T) {
	tests := []struct {
		formats []string
		target  string
		want    bool
	}{
		{[]string{"pdf", "html"}, "pdf", true},
		{[]string{"PDF", "HTML"}, "pdf", true}, // case-insensitive
		{[]string{" pdf "}, "pdf", true},       // whitespace trimmed
		{[]string{"html", "epub"}, "pdf", false},
		{[]string{}, "pdf", false},
		{nil, "pdf", false},
		{[]string{"site"}, "site", true},
	}
	for _, tt := range tests {
		if got := containsBuildFormat(tt.formats, tt.target); got != tt.want {
			t.Errorf("containsBuildFormat(%v, %q) = %v, want %v", tt.formats, tt.target, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// containsAnyNonPDFFormat
// ---------------------------------------------------------------------------

func TestContainsAnyNonPDFFormat(t *testing.T) {
	tests := []struct {
		formats []string
		want    bool
	}{
		{[]string{"pdf"}, false},
		{[]string{"PDF"}, false},
		{[]string{"html"}, true},
		{[]string{"pdf", "html"}, true},
		{[]string{}, false},
		{nil, false},
		{[]string{"site", "epub"}, true},
	}
	for _, tt := range tests {
		if got := containsAnyNonPDFFormat(tt.formats); got != tt.want {
			t.Errorf("containsAnyNonPDFFormat(%v) = %v, want %v", tt.formats, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// resolveRequestedBuildOutput
// ---------------------------------------------------------------------------

func TestResolveRequestedBuildOutput(t *testing.T) {
	// Empty string returns empty without error.
	got, err := resolveRequestedBuildOutput("")
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}

	// Non-empty path should be resolved to an absolute path.
	got, err = resolveRequestedBuildOutput("output/book.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty absolute path")
	}

	// A trailing separator (explicit directory intent) must be preserved
	// through the filepath.Abs normalization.
	got, err = resolveRequestedBuildOutput("output" + string(os.PathSeparator))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasTrailingPathSeparator(got) {
		t.Errorf("expected trailing separator to be preserved, got %q", got)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

func TestHasTrailingPathSeparator(t *testing.T) {
	if hasTrailingPathSeparator("") {
		t.Error("empty string should not report a trailing separator")
	}
	if hasTrailingPathSeparator("dist") {
		t.Error("plain name should not report a trailing separator")
	}
	if !hasTrailingPathSeparator("dist/") {
		t.Error("dist/ should report a trailing separator")
	}
}

// ---------------------------------------------------------------------------
// predictedOutputLinks
// ---------------------------------------------------------------------------

func TestPredictedOutputLinks(t *testing.T) {
	base := "/out/mybook"
	links := predictedOutputLinks(base, "", []string{"pdf", "html", "site", "epub"})

	if links["pdf"] != "/out/mybook.pdf" {
		t.Errorf("pdf link = %q", links["pdf"])
	}
	if links["html"] != "/out/mybook.html" {
		t.Errorf("html link = %q", links["html"])
	}
	if links["epub"] != "/out/mybook.epub" {
		t.Errorf("epub link = %q", links["epub"])
	}
	// With no explicit site dir, site falls back to <base>_site/index.html,
	// matching siteBuilder's multi-language fallback.
	if links["site"] != filepath.Join("/out/mybook_site", "index.html") {
		t.Errorf("site link = %q, want %q", links["site"], filepath.Join("/out/mybook_site", "index.html"))
	}

	// An explicit site dir must be advertised verbatim: the same directory
	// the site builder actually writes into.
	linksSite := predictedOutputLinks(base, "/proj/_book", []string{"site"})
	if linksSite["site"] != filepath.Join("/proj/_book", "index.html") {
		t.Errorf("site link = %q, want %q", linksSite["site"], filepath.Join("/proj/_book", "index.html"))
	}

	// typst produces a distinct PDF filename.
	linksTypst := predictedOutputLinks(base, "", []string{"typst"})
	if linksTypst["typst"] != "/out/mybook-typst.pdf" {
		t.Errorf("typst link = %q, want %q", linksTypst["typst"], "/out/mybook-typst.pdf")
	}

	// Unknown format is ignored.
	links2 := predictedOutputLinks(base, "", []string{"docx"})
	if len(links2) != 0 {
		t.Errorf("expected empty map for unknown format, got %v", links2)
	}
}

// ---------------------------------------------------------------------------
// defaultLanguageTarget
// ---------------------------------------------------------------------------

func TestDefaultLanguageTarget(t *testing.T) {
	// Empty summaries returns empty string.
	if got := defaultLanguageTarget("/landing", nil); got != "" {
		t.Errorf("expected empty string for no summaries, got %q", got)
	}

	// Summary with no outputs returns empty string.
	summaries := []languageBuildSummary{
		{Name: "English", Dir: "en", Outputs: map[string]string{}},
	}
	if got := defaultLanguageTarget("/landing", summaries); got != "" {
		t.Errorf("expected empty for summary with no outputs, got %q", got)
	}

	// Summary with html output returns a relative path.
	summaries2 := []languageBuildSummary{
		{Name: "English", Dir: "en", Outputs: map[string]string{
			"html": "/landing/en/book.html",
		}},
	}
	got := defaultLanguageTarget("/landing", summaries2)
	if got == "" {
		t.Error("expected non-empty target when html output present")
	}
}

// ---------------------------------------------------------------------------
// preferredLanguageFile
// ---------------------------------------------------------------------------

func TestPreferredLanguageFile(t *testing.T) {
	// No outputs.
	s := languageBuildSummary{Outputs: map[string]string{}}
	if got := preferredLanguageFile(s); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// HTML preferred over site.
	s.Outputs = map[string]string{"html": "/out/book.html", "site": "/out/site/index.html"}
	if got := preferredLanguageFile(s); got != "/out/book.html" {
		t.Errorf("expected html output, got %q", got)
	}

	// Only site available.
	s.Outputs = map[string]string{"site": "/out/site/index.html"}
	if got := preferredLanguageFile(s); got != "/out/site/index.html" {
		t.Errorf("expected site output, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// buildLanguageSwitcherHTML
// ---------------------------------------------------------------------------

func TestBuildLanguageSwitcherHTML(t *testing.T) {
	summaries := []languageBuildSummary{
		{Name: "English", Dir: "en", Outputs: map[string]string{"html": "/out/en/book.html"}},
		{Name: "中文", Dir: "zh", Outputs: map[string]string{"html": "/out/zh/book.html"}},
	}

	html, err := buildLanguageSwitcherHTML("/out/en", "/out/index.html", summaries, "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Current language should be a <span class="current">, not a link.
	if !strings.Contains(html, `<span class="current">English</span>`) {
		t.Errorf("expected current language span, got: %s", html)
	}
	// Other language should be a link.
	if !strings.Contains(html, `<a href=`) {
		t.Error("expected link for non-current language")
	}
	if !strings.Contains(html, "中文") {
		t.Error("expected Chinese language name in output")
	}
	// Should contain landing page link.
	if !strings.Contains(html, "All languages") {
		t.Error("expected 'All languages' link")
	}
}

func TestBuildLanguageSwitcherHTML_WindowsAbsolutePaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific path regression")
	}

	summaries := []languageBuildSummary{
		{Name: "English", Dir: "en", Outputs: map[string]string{"site": `C:\book\en_site\index.html`}},
		{Name: "中文", Dir: "zh", Outputs: map[string]string{"site": `C:\book\zh_site\index.html`}},
	}

	html, err := buildLanguageSwitcherHTML(`C:\book\en_site`, `C:\book\_mdpress_langs.html`, summaries, "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `href="../zh_site/index.html"`) {
		t.Fatalf("expected normalized site link, got: %s", html)
	}
	if !strings.Contains(html, `href="../_mdpress_langs.html"`) {
		t.Fatalf("expected normalized landing link, got: %s", html)
	}
	if strings.Contains(html, `\`) {
		t.Fatalf("expected slash-normalized links, got: %s", html)
	}
}

func TestBuildLanguageSwitcherHTML_WindowsHTMLPreferredAndNestedPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific path regression")
	}

	summaries := []languageBuildSummary{
		{
			Name: "English",
			Dir:  "en",
			Outputs: map[string]string{
				"html": `C:\book\out\en\pages\book.html`,
				"site": `C:\book\out\en_site\index.html`,
			},
		},
		{
			Name: "中文",
			Dir:  "zh",
			Outputs: map[string]string{
				"html": `C:\book\out\zh\pages\book.html`,
				"site": `C:\book\out\zh_site\index.html`,
			},
		},
	}

	html, err := buildLanguageSwitcherHTML(`C:\book\out\en\pages`, `C:\book\out\_mdpress_langs.html`, summaries, "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `<span class="current">English</span>`) {
		t.Fatalf("expected current language span, got: %s", html)
	}
	if !strings.Contains(html, `href="../../zh/pages/book.html"`) {
		t.Fatalf("expected html output to be preferred for non-current language, got: %s", html)
	}
	if !strings.Contains(html, `href="../../_mdpress_langs.html"`) {
		t.Fatalf("expected nested landing link, got: %s", html)
	}
	if strings.Contains(html, `\`) {
		t.Fatalf("expected slash-normalized links, got: %s", html)
	}
}

func TestBuildLanguageSwitcherHTML_WindowsCrossDriveReturnsError(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific path regression")
	}

	summaries := []languageBuildSummary{
		{Name: "English", Dir: "en", Outputs: map[string]string{"html": `C:\book\en\book.html`}},
		{Name: "中文", Dir: "zh", Outputs: map[string]string{"html": `D:\book\zh\book.html`}},
	}

	if _, err := buildLanguageSwitcherHTML(`C:\book\en`, `D:\book\_mdpress_langs.html`, summaries, "en"); err == nil {
		t.Fatal("expected cross-drive relative path calculation to fail")
	}
}

func TestBuildLanguageSwitcherHTML_NoOutputs(t *testing.T) {
	summaries := []languageBuildSummary{
		{Name: "Empty", Dir: "empty", Outputs: map[string]string{}},
	}
	html, err := buildLanguageSwitcherHTML("/out", "/out/index.html", summaries, "other")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Language with no preferred output should be skipped.
	if strings.Contains(html, "Empty") {
		t.Error("expected language with no outputs to be skipped")
	}
}

// ---------------------------------------------------------------------------
// rendererHeadingsToSiteHeadings
// ---------------------------------------------------------------------------

func TestRendererHeadingsToSiteHeadings(t *testing.T) {
	// Empty input returns empty slice.
	result := rendererHeadingsToSiteHeadings(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}

	// Single heading without children.
	input := []renderer.NavHeading{
		{Title: "Introduction", ID: "intro"},
	}
	result = rendererHeadingsToSiteHeadings(input)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Title != "Introduction" || result[0].ID != "intro" {
		t.Errorf("unexpected result: %+v", result[0])
	}

	// Nested headings are preserved.
	nested := []renderer.NavHeading{
		{Title: "Chapter 1", ID: "ch1", Children: []renderer.NavHeading{
			{Title: "Section 1.1", ID: "s11"},
		}},
	}
	result = rendererHeadingsToSiteHeadings(nested)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level, got %d", len(result))
	}
	if len(result[0].Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(result[0].Children))
	}
}

// ---------------------------------------------------------------------------
// rewriteChapterLinksForSite
// ---------------------------------------------------------------------------

func TestRewriteChapterLinksForSite(t *testing.T) {
	// Mismatched lengths return original chapters unchanged.
	chapters := []renderer.ChapterHTML{{Title: "Ch1", ID: "ch1"}}
	files := []string{"ch1.md"}
	pages := []string{"ch_000.html", "ch_001.html"} // wrong length
	result := rewriteChapterLinksForSite(chapters, files, pages)
	if len(result) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(result))
	}

	// Empty chapters return empty.
	result2 := rewriteChapterLinksForSite(nil, nil, nil)
	if len(result2) != 0 {
		t.Errorf("expected empty, got %d", len(result2))
	}

	// Matching lengths are processed without error.
	ch := []renderer.ChapterHTML{{Title: "Ch1", ID: "ch1", Content: "<p>hello</p>"}}
	f := []string{"ch1.md"}
	p := []string{"ch_000.html"}
	result3 := rewriteChapterLinksForSite(ch, f, p)
	if len(result3) != 1 {
		t.Fatalf("expected 1, got %d", len(result3))
	}
}

// ---------------------------------------------------------------------------
// runPluginHook
// ---------------------------------------------------------------------------

func TestRunPluginHook_NilManager(t *testing.T) {
	// nil manager should be a no-op, not a panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runPluginHook panicked with nil manager: %v", r)
		}
	}()
	runPluginHook(nil, nil, slog.Default())
}

// ---------------------------------------------------------------------------
// formatIssueSummary
// ---------------------------------------------------------------------------

func TestFormatIssueSummary(t *testing.T) {
	// Empty returns empty string.
	if got := formatIssueSummary(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// Single issue.
	issues := []projectIssue{{Rule: "chapter-title-sequence"}}
	got := formatIssueSummary(issues)
	if !strings.Contains(got, "chapter-title-sequence=1") {
		t.Errorf("unexpected summary: %q", got)
	}

	// Multiple rules are sorted and joined.
	issues2 := []projectIssue{
		{Rule: "book-title-style"},
		{Rule: "book-title-style"},
		{Rule: "chapter-title-sequence"},
	}
	got2 := formatIssueSummary(issues2)
	if !strings.Contains(got2, "book-title-style=2") {
		t.Errorf("unexpected summary: %q", got2)
	}
	if !strings.Contains(got2, "chapter-title-sequence=1") {
		t.Errorf("unexpected summary: %q", got2)
	}
}

// ---------------------------------------------------------------------------
// flattenChaptersWithDepth
// ---------------------------------------------------------------------------

func TestFlattenChaptersWithDepth(t *testing.T) {
	// Empty input.
	result := flattenChaptersWithDepth(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}

	// Single chapter, no sections.
	chapters := []config.ChapterDef{{Title: "Intro", File: "intro.md"}}
	result = flattenChaptersWithDepth(chapters)
	if len(result) != 1 || result[0].Depth != 0 {
		t.Errorf("unexpected result: %+v", result)
	}

	// Nested chapters increment depth.
	nested := []config.ChapterDef{
		{Title: "Ch1", File: "ch1.md", Sections: []config.ChapterDef{
			{Title: "Sec1.1", File: "sec11.md"},
		}},
	}
	result = flattenChaptersWithDepth(nested)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].Depth != 0 || result[1].Depth != 1 {
		t.Errorf("depths: %d, %d; want 0, 1", result[0].Depth, result[1].Depth)
	}
}

// ---------------------------------------------------------------------------
// buildHeadingTree
// ---------------------------------------------------------------------------

func TestBuildHeadingTree_Empty(t *testing.T) {
	result := buildHeadingTree(nil, "chapter-id")
	if result != nil {
		t.Errorf("expected nil for empty headings, got %v", result)
	}
}

func TestBuildHeadingTree_WithHeadings(t *testing.T) {
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Chapter One", ID: "chapter-one"},
		{Level: 2, Text: "Section 1.1", ID: "section-1-1"},
	}
	result := buildHeadingTree(headings, "chapter-one")
	// The chapter root heading (level 1) is stripped; only section-level entries remain.
	if result == nil {
		t.Fatal("expected non-nil result for non-empty headings")
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 section heading after stripping root, got %d", len(result))
	}
	if result[0].Title != "Section 1.1" {
		t.Errorf("expected heading title %q, got %q", "Section 1.1", result[0].Title)
	}
}

// ---------------------------------------------------------------------------
// toNavHeadings and toRendererNavHeadings
// ---------------------------------------------------------------------------

func TestToNavHeadings_RoundTrip(t *testing.T) {
	navs := []navHeading{
		{Title: "Introduction", ID: "intro"},
		{Title: "Chapter 1", ID: "ch1", Children: []navHeading{
			{Title: "Sub", ID: "sub"},
		}},
	}
	// toRendererNavHeadings should preserve structure.
	rNavs := toRendererNavHeadings(navs)
	if len(rNavs) != 2 {
		t.Fatalf("expected 2 items, got %d", len(rNavs))
	}
	if rNavs[1].Title != "Chapter 1" {
		t.Errorf("title mismatch: %q", rNavs[1].Title)
	}
	if len(rNavs[1].Children) != 1 {
		t.Errorf("children count: %d", len(rNavs[1].Children))
	}
}

// ---------------------------------------------------------------------------
// buildMermaidValidationHTML
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML(t *testing.T) {
	body := `<div class="mermaid">graph TD; A-->B;</div>`
	html := buildMermaidValidationHTML(body)

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(html, body) {
		t.Error("body content not embedded")
	}
	if !strings.Contains(html, "__mdpressMermaidStatus") {
		t.Error("missing status tracking variable")
	}
}

func TestBuildMermaidValidationHTML_EmptyBody(t *testing.T) {
	html := buildMermaidValidationHTML("")
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("missing DOCTYPE")
	}
}

// ---------------------------------------------------------------------------
// validateRenderedMermaidHTML
// ---------------------------------------------------------------------------

func TestValidateRenderedMermaidHTML_EmptyContent(t *testing.T) {
	// Empty content is a no-op; chromium is not required.
	err := validateRenderedMermaidHTML("")
	if err != nil {
		t.Errorf("expected nil for empty content, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// getAvailableThemes (extended coverage)
// ---------------------------------------------------------------------------

func TestGetAvailableThemes_Colors(t *testing.T) {
	themes := getAvailableThemes()
	for _, th := range themes {
		if th.colors.background == "" {
			t.Errorf("theme %q has empty background color", th.name)
		}
		if th.colors.codeBg == "" {
			t.Errorf("theme %q has empty code background color", th.name)
		}
	}
}

// ---------------------------------------------------------------------------
// executeThemesList (extended)
// ---------------------------------------------------------------------------

func TestExecuteThemesList_NoError(t *testing.T) {
	defer suppressOutput(t)()
	// Should not return an error.
	if err := executeThemesList(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// executeThemesShow
// ---------------------------------------------------------------------------

func TestExecuteThemesShow(t *testing.T) {
	defer suppressOutput(t)()
	// Known theme should succeed.
	if err := executeThemesShow("technical"); err != nil {
		t.Errorf("executeThemesShow(technical) unexpected error: %v", err)
	}
	if err := executeThemesShow("elegant"); err != nil {
		t.Errorf("executeThemesShow(elegant) unexpected error: %v", err)
	}

	// Unknown theme should return an error.
	if err := executeThemesShow("nonexistent-theme"); err == nil {
		t.Error("expected error for unknown theme, got nil")
	}
}

// ---------------------------------------------------------------------------
// NewBuildOrchestrator
// ---------------------------------------------------------------------------

func TestNewBuildOrchestrator_MinimalConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.Default()

	orch, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
		return
	}
	if orch.Config == nil {
		t.Error("orchestrator config is nil")
	}
	if orch.Theme == nil {
		t.Error("orchestrator theme is nil")
	}
	if orch.Parser == nil {
		t.Error("orchestrator parser is nil")
	}
	if orch.Logger == nil {
		t.Error("orchestrator logger is nil")
	}
}

func TestNewBuildOrchestrator_InvalidThemeFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Style.Theme = "this-theme-does-not-exist"
	logger := slog.Default()

	// Should fall back to "technical" and succeed.
	orch, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orch.Theme == nil {
		t.Error("theme should be non-nil after fallback")
	}
}

// ---------------------------------------------------------------------------
// buildSiteChapterTree
// ---------------------------------------------------------------------------

func TestBuildSiteChapterTree_Empty(t *testing.T) {
	result := buildSiteChapterTree(nil, nil, nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestBuildSiteChapterTree_Mismatch(t *testing.T) {
	defs := []config.ChapterDef{{Title: "Ch1", File: "ch1.md"}}
	// No chapters/filenames provided.
	result := buildSiteChapterTree(defs, nil, nil, nil, nil)
	// Chapter has a file but no matching chapter data, so result should be empty.
	if len(result) != 0 {
		t.Errorf("expected 0 items when no chapter HTML, got %d", len(result))
	}
}

func TestBuildSiteChapterTree_WithData(t *testing.T) {
	defs := []config.ChapterDef{
		{Title: "Ch1", File: "ch1.md"},
		{Title: "Ch2", File: "ch2.md"},
	}
	chapters := []renderer.ChapterHTML{
		{Title: "Chapter One", ID: "ch1", Content: "<p>one</p>"},
		{Title: "Chapter Two", ID: "ch2", Content: "<p>two</p>"},
	}
	chapterFiles := []string{"ch1.md", "ch2.md"}
	pages := []string{"ch_000.html", "ch_001.html"}
	markdown := []string{"# One", "# Two"}

	result := buildSiteChapterTree(defs, chapters, chapterFiles, pages, markdown)
	if len(result) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(result))
	}
	if result[0].ID != "ch1" {
		t.Errorf("first chapter ID = %q, want ch1", result[0].ID)
	}
	if result[0].Markdown != "# One" {
		t.Errorf("first chapter markdown = %q, want # One", result[0].Markdown)
	}
}

func TestBuildSiteChapterTree_SkippedChapter(t *testing.T) {
	// Simulate a case where chapter B was skipped during processing.
	defs := []config.ChapterDef{
		{Title: "A", File: "a.md"},
		{Title: "B", File: "b.md"},
		{Title: "C", File: "c.md"},
	}
	// Only chapters A and C were processed (B was skipped).
	chapters := []renderer.ChapterHTML{
		{Title: "Chapter A", ID: "a", Content: "<p>A</p>"},
		{Title: "Chapter C", ID: "c", Content: "<p>C</p>"},
	}
	chapterFiles := []string{"a.md", "c.md"}
	pages := []string{"ch_000.html", "ch_001.html"}
	markdown := []string{"# A", "# C"}

	result := buildSiteChapterTree(defs, chapters, chapterFiles, pages, markdown)
	// Only A and C should appear (B was skipped).
	if len(result) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(result))
	}
	if result[0].ID != "a" {
		t.Errorf("first chapter ID = %q, want a", result[0].ID)
	}
	if result[1].ID != "c" {
		t.Errorf("second chapter ID = %q, want c", result[1].ID)
	}
	if result[1].Content != "<p>C</p>" {
		t.Errorf("second chapter content = %q, want <p>C</p>", result[1].Content)
	}
	// Verify filenames are correctly assigned (core fix for slug collisions).
	if result[0].Filename != "ch_000.html" {
		t.Errorf("first chapter filename = %q, want ch_000.html", result[0].Filename)
	}
	if result[1].Filename != "ch_001.html" {
		t.Errorf("second chapter filename = %q, want ch_001.html", result[1].Filename)
	}
}

// ---------------------------------------------------------------------------
// output.SiteNavHeading round-trip via rendererHeadingsToSiteHeadings
// ---------------------------------------------------------------------------

func TestRendererHeadingsToSiteHeadings_Nested(t *testing.T) {
	input := []renderer.NavHeading{
		{Title: "Top", ID: "top", Children: []renderer.NavHeading{
			{Title: "Mid", ID: "mid", Children: []renderer.NavHeading{
				{Title: "Leaf", ID: "leaf"},
			}},
		}},
	}
	result := rendererHeadingsToSiteHeadings(input)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
	if result[0].Children[0].Children[0].Title != "Leaf" {
		t.Errorf("nested title not preserved")
	}
}

// Compile-time check: ensure output.SiteNavHeading is referenced.
var _ output.SiteNavHeading

// ---------------------------------------------------------------------------
// pdfHeaderFooterTemplates / expandPDFTemplateTokens
// ---------------------------------------------------------------------------

func TestPDFHeaderFooterTemplatesDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	header, footer := pdfHeaderFooterTemplates(cfg)
	if header != "" {
		t.Errorf("default header should be empty, got %q", header)
	}
	if footer != defaultPDFFooterTemplate {
		t.Errorf("default footer = %q, want the centered page-number template", footer)
	}
	if !strings.Contains(footer, "pageNumber") {
		t.Error("default footer must contain the pageNumber span")
	}
	if strings.Contains(footer, "mdPress") {
		t.Error("default footer must not carry branding")
	}
}

func TestPDFHeaderFooterTemplatesDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Header = false
	cfg.Output.Footer = false
	header, footer := pdfHeaderFooterTemplates(cfg)
	if header != "" || footer != "" {
		t.Errorf("output.header/footer=false should disable both, got header=%q footer=%q", header, footer)
	}
}

func TestPDFHeaderFooterTemplatesCustom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "My <Book> & Others"
	cfg.Style.Footer = config.HeaderFooterStyle{Left: "{title}", Center: "{page}", Right: "v1"}
	cfg.Style.Header = config.HeaderFooterStyle{Center: "{{.Book.Title}}"}
	header, footer := pdfHeaderFooterTemplates(cfg)
	if !strings.Contains(footer, "<span class='pageNumber'></span>") {
		t.Errorf("custom footer should expand {page}: %q", footer)
	}
	if !strings.Contains(footer, "My &lt;Book&gt; &amp; Others") {
		t.Errorf("custom footer should expand {title} HTML-escaped: %q", footer)
	}
	if !strings.Contains(footer, "v1") {
		t.Errorf("custom footer should keep literal text: %q", footer)
	}
	if !strings.Contains(header, "My &lt;Book&gt; &amp; Others") {
		t.Errorf("custom header should render the escaped book title: %q", header)
	}
}

func TestExpandPDFTemplateTokens(t *testing.T) {
	got := expandPDFTemplateTokens("<script>alert(1)</script> {page} of {pages}", config.BookMeta{Title: "T"})
	if strings.Contains(got, "<script>") {
		t.Errorf("user text must be HTML-escaped, got %q", got)
	}
	if !strings.Contains(got, "<span class='pageNumber'></span>") {
		t.Errorf("{page} must expand to the pageNumber span, got %q", got)
	}
	if !strings.Contains(got, "<span class='totalPages'></span>") {
		t.Errorf("{pages} must expand to the totalPages span, got %q", got)
	}
	if expandPDFTemplateTokens("{{.Chapter.Title}}", config.BookMeta{Title: "T"}) != "" {
		t.Error("{{.Chapter.Title}} has no Chrome equivalent and should expand to nothing")
	}
	if expandPDFTemplateTokens("", config.BookMeta{Title: "T"}) != "" {
		t.Error("empty input should stay empty")
	}
}

// ---------------------------------------------------------------------------
// site output: safety check, atomic swap, and pruning of stale pages
// ---------------------------------------------------------------------------

func TestEnsureReplaceableSiteDir(t *testing.T) {
	root := t.TempDir()

	t.Run("missing directory is fine", func(t *testing.T) {
		if err := ensureReplaceableSiteDir(filepath.Join(root, "missing")); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty directory is fine", func(t *testing.T) {
		dir := filepath.Join(root, "empty")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := ensureReplaceableSiteDir(dir); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("previously generated site is fine", func(t *testing.T) {
		dir := filepath.Join(root, "generated")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ensureReplaceableSiteDir(dir); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("foreign content is refused", func(t *testing.T) {
		dir := filepath.Join(root, "userdata")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep me"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ensureReplaceableSiteDir(dir); err == nil {
			t.Error("expected refusal for a non-empty non-generated directory")
		}
	})

	t.Run("existing file is refused", func(t *testing.T) {
		path := filepath.Join(root, "afile")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ensureReplaceableSiteDir(path); err == nil {
			t.Error("expected refusal when the site output path is a file")
		}
	})
}

func TestSwapSiteDir(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "site")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "stale.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	staging, err := newSiteStaging(root, "mdpress-site-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging.Site, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := swapSiteDir(staging, target, slog.Default()); err != nil {
		t.Fatalf("swapSiteDir() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "index.html")); err != nil {
		t.Errorf("fresh build not swapped in: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "stale.html")); !os.IsNotExist(err) {
		t.Error("stale page survived the swap")
	}
	if _, err := os.Stat(target + ".old"); !os.IsNotExist(err) {
		t.Error("backup directory left behind after swap")
	}
	if _, err := os.Stat(staging.Root); !os.IsNotExist(err) {
		t.Error("staging directory left behind after swap")
	}
}

// newSiteBuildContext builds a minimal one-chapter buildContext for site tests.
func newSiteBuildContext(t *testing.T, siteDir string, logger *slog.Logger) *buildContext {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Book.Title = "Swap Test Book"
	cfg.Chapters = []config.ChapterDef{{Title: "One", File: "ch1.md"}}
	thm, err := theme.NewThemeManager().Get("technical")
	if err != nil {
		t.Fatalf("load builtin theme: %v", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &buildContext{
		Config:          cfg,
		Theme:           thm,
		ChaptersHTML:    []renderer.ChapterHTML{{Title: "One", ID: "one", Content: "<p>hello</p>"}},
		ChapterFiles:    []string{"ch1.md"},
		ChapterMarkdown: []string{"# One\n\nhello\n"},
		SiteDir:         siteDir,
		Logger:          logger,
	}
}

func TestSiteBuilderSwapPrunesStalePages(t *testing.T) {
	root := t.TempDir()
	siteDir := filepath.Join(root, "_book")
	bc := newSiteBuildContext(t, siteDir, nil)
	base := filepath.Join(root, "book")

	if err := (&siteBuilder{}).Build(context.Background(), bc, base); err != nil {
		t.Fatalf("first site build failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(siteDir, "index.html")); err != nil {
		t.Fatalf("index.html missing after build: %v", err)
	}

	// Plant a stale page (e.g. from a renamed chapter) and rebuild: the
	// atomic swap must leave only the current build's files.
	if err := os.WriteFile(filepath.Join(siteDir, "stale.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (&siteBuilder{}).Build(context.Background(), bc, base); err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(siteDir, "stale.html")); !os.IsNotExist(err) {
		t.Error("stale page survived the rebuild")
	}
	if _, err := os.Stat(filepath.Join(siteDir, "index.html")); err != nil {
		t.Errorf("index.html missing after rebuild: %v", err)
	}
	if _, err := os.Stat(siteDir + ".old"); !os.IsNotExist(err) {
		t.Error("backup directory left behind")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "mdpress-site-") {
			t.Errorf("temp build directory left behind: %s", e.Name())
		}
	}
}

// TestSiteBuilderOutputIsWorldReadable guards the published site root's mode.
// The atomic swap renames a staging directory into place, and os.MkdirTemp
// creates those at 0700 — a site root nobody but the owner can traverse means
// a 403 from nginx/httpd and a broken deploy via rsync -a / docker COPY.
func TestSiteBuilderOutputIsWorldReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	root := t.TempDir()
	siteDir := filepath.Join(root, "_book")
	bc := newSiteBuildContext(t, siteDir, nil)

	if err := (&siteBuilder{}).Build(context.Background(), bc, filepath.Join(root, "book")); err != nil {
		t.Fatalf("site build failed: %v", err)
	}
	info, err := os.Stat(siteDir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("site root mode = %04o, want 0755 (a 0700 site root breaks deploys)", perm)
	}
}

func TestSiteBuilderRefusesForeignDirectory(t *testing.T) {
	root := t.TempDir()
	siteDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(siteDir, "notes.txt")
	if err := os.WriteFile(userFile, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	bc := newSiteBuildContext(t, siteDir, nil)
	if err := (&siteBuilder{}).Build(context.Background(), bc, filepath.Join(root, "book")); err == nil {
		t.Fatal("expected the site build to refuse replacing a directory with user data")
	}
	if data, err := os.ReadFile(userFile); err != nil || string(data) != "keep me" {
		t.Errorf("user data was modified: %v %q", err, data)
	}
}

func TestSiteBuilderInPlaceWhenSharingOutputDir(t *testing.T) {
	// --output <dir> makes the site share its directory with the other
	// formats' files; the builder must generate in place without deleting
	// sibling outputs.
	root := t.TempDir()
	sibling := filepath.Join(root, "book.pdf")
	if err := os.WriteFile(sibling, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}

	bc := newSiteBuildContext(t, root, nil)
	if err := (&siteBuilder{}).Build(context.Background(), bc, filepath.Join(root, "book")); err != nil {
		t.Fatalf("in-place site build failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		t.Errorf("index.html missing after in-place build: %v", err)
	}
	if data, err := os.ReadFile(sibling); err != nil || string(data) != "pdf" {
		t.Errorf("sibling output was clobbered: %v %q", err, data)
	}
}

func TestSiteBuilderLegacySiteDirHint(t *testing.T) {
	root := t.TempDir()
	siteDir := filepath.Join(root, "_book")
	base := filepath.Join(root, "book")
	if err := os.MkdirAll(base+"_site", 0o755); err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	bc := newSiteBuildContext(t, siteDir, logger)
	if err := (&siteBuilder{}).Build(context.Background(), bc, base); err != nil {
		t.Fatalf("site build failed: %v", err)
	}
	if !strings.Contains(buf.String(), "moved to _book/") {
		t.Errorf("expected a legacy <name>_site migration hint in the logs, got:\n%s", buf.String())
	}
}

// TestPDFHeaderFooterHonorsDocumentedExample covers a configuration that could
// not work: the manual's example header was byte-for-byte the built-in default,
// and the builder decided "the user customized this" by comparing against that
// default — so copying the example produced no header and no explanation.
func TestPDFHeaderFooterHonorsDocumentedExample(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "Header Book"
	cfg.Book.Author = "Ann Author"
	cfg.Output.Header = true
	cfg.Style.Header = config.HeaderFooterStyle{
		Left:  "{{.Book.Title}}",
		Right: "{{.Book.Author}}",
	}

	header, _ := pdfHeaderFooterTemplates(cfg)
	if header == "" {
		t.Fatal("the documented example produced no header")
	}
	for _, want := range []string{"Header Book", "Ann Author"} {
		if !strings.Contains(header, want) {
			t.Errorf("header is missing %q: %s", want, header)
		}
	}
	if strings.Contains(header, "{{") {
		t.Errorf("an unexpanded token reached the page: %s", header)
	}
}

// TestPDFHeaderDefaultsToNone keeps the documented default: no header unless
// style.header is configured.
func TestPDFHeaderDefaultsToNone(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Header = true

	if header, _ := pdfHeaderFooterTemplates(cfg); header != "" {
		t.Errorf("expected no header without style.header, got %q", header)
	}
}

// TestExpandPDFTemplateTokensStripsUnknown keeps an unsupported token off the
// paper: Chrome prints the running head verbatim, so "{{.Whatever}}" would
// appear on every page.
func TestExpandPDFTemplateTokensStripsUnknown(t *testing.T) {
	got := expandPDFTemplateTokens("{{.Nope}} {title}", config.BookMeta{Title: "T"})
	if strings.Contains(got, "{{") {
		t.Errorf("unknown token survived: %q", got)
	}
	if !strings.Contains(got, "T") {
		t.Errorf("known token was lost: %q", got)
	}
}
