// extra_funcs_test.go adds coverage for pure utility functions in the cmd package
// that were previously untested: format predicates, multilingual helpers,
// navigation converters, build-issue reporting, and theme listing.
package cmd

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/output"
	"github.com/yeasy/mdpress/internal/renderer"
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
}

// ---------------------------------------------------------------------------
// predictedOutputLinks
// ---------------------------------------------------------------------------

func TestPredictedOutputLinks(t *testing.T) {
	base := "/out/mybook"
	links := predictedOutputLinks(base, []string{"pdf", "html", "site", "epub"})

	if links["pdf"] != "/out/mybook.pdf" {
		t.Errorf("pdf link = %q", links["pdf"])
	}
	if links["html"] != "/out/mybook.html" {
		t.Errorf("html link = %q", links["html"])
	}
	if links["epub"] != "/out/mybook.epub" {
		t.Errorf("epub link = %q", links["epub"])
	}
	// site points to <base>_site/index.html
	if !strings.Contains(links["site"], "index.html") {
		t.Errorf("site link does not end with index.html: %q", links["site"])
	}

	// typst produces a distinct PDF filename.
	linksTypst := predictedOutputLinks(base, []string{"typst"})
	if linksTypst["typst"] != "/out/mybook-typst.pdf" {
		t.Errorf("typst link = %q, want %q", linksTypst["typst"], "/out/mybook-typst.pdf")
	}

	// Unknown format is ignored.
	links2 := predictedOutputLinks(base, []string{"docx"})
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
	result := buildSiteChapterTree(nil, nil, nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestBuildSiteChapterTree_Mismatch(t *testing.T) {
	defs := []config.ChapterDef{{Title: "Ch1", File: "ch1.md"}}
	// No chapters/filenames provided.
	result := buildSiteChapterTree(defs, nil, nil, nil)
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
	pages := []string{"ch_000.html", "ch_001.html"}
	markdown := []string{"# One", "# Two"}

	result := buildSiteChapterTree(defs, chapters, pages, markdown)
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
