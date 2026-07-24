package cmd

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/toc"
)

// TestBuildSiteChapterTreeKeepsSynthesizedGlossary pins the fix for the dead
// glossary links on every published site: buildSiteChapterTree walked
// cfg.Chapters, and the glossary appendix has no ChapterDef, so it was dropped
// — the build injected "#glossary-term" links into every chapter and then
// never published the page they point at.
func TestBuildSiteChapterTreeKeepsSynthesizedGlossary(t *testing.T) {
	defs := []config.ChapterDef{{Title: "One", File: "one.md"}}
	chapters := []renderer.ChapterHTML{
		{Title: "One", ID: "one", Content: "<p>body</p>"},
		{Title: glossaryChapterTitle, ID: glossaryChapterID, Content: `<dt id="glossary-api">API</dt>`},
	}
	chapterFiles := []string{"one.md", ""}
	pages := []string{"one.html", siteGlossaryPageName}

	tree := buildSiteChapterTree(defs, chapters, chapterFiles, pages, nil)
	if len(tree) != 2 {
		t.Fatalf("expected the glossary to be published as a page, got %d chapters: %+v", len(tree), tree)
	}
	if tree[1].ID != glossaryChapterID {
		t.Errorf("trailing site chapter = %q, want %q", tree[1].ID, glossaryChapterID)
	}
	if tree[1].Filename != siteGlossaryPageName {
		t.Errorf("glossary page filename = %q, want %q", tree[1].Filename, siteGlossaryPageName)
	}
}

// TestResolveSiteGlossaryPageRepointsTermLinks covers the other half of the
// dead-link defect: the term links are same-document ("#glossary-api"), which
// is right for the PDF and the single-file HTML but points at the chapter the
// reader is already on once every chapter is its own page.
func TestResolveSiteGlossaryPageRepointsTermLinks(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{Title: "One", ID: "one", Content: `<p><a href="#glossary-api">API</a></p>`},
		{Title: "Deep", ID: "deep", Content: `<p><a href="#glossary-api">API</a></p>`},
		{Title: glossaryChapterTitle, ID: glossaryChapterID, Content: `<dt id="glossary-api">API</dt>`},
	}
	chapterFiles := []string{"one.md", "guide/deep.md", ""}
	pages := []string{"one.html", "guide/deep.html", "ch_002.html"}

	got, gotPages := resolveSiteGlossaryPage(chapters, chapterFiles, pages)

	if gotPages[2] != siteGlossaryPageName {
		t.Errorf("glossary page = %q, want %q", gotPages[2], siteGlossaryPageName)
	}
	if want := `href="glossary.html#glossary-api"`; !strings.Contains(got[0].Content, want) {
		t.Errorf("top-level chapter should link to %s, got %q", want, got[0].Content)
	}
	if want := `href="../glossary.html#glossary-api"`; !strings.Contains(got[1].Content, want) {
		t.Errorf("nested chapter should link to %s, got %q", want, got[1].Content)
	}
	// The glossary page's own definition anchors must stay where they are.
	if !strings.Contains(got[2].Content, `<dt id="glossary-api">`) {
		t.Errorf("glossary page content was rewritten: %q", got[2].Content)
	}
	// The caller's slices are reused across a two-pass site generation, so
	// rewriting must not mutate them in place.
	if pages[2] != "ch_002.html" || strings.Contains(chapters[0].Content, "glossary.html") {
		t.Error("resolveSiteGlossaryPage mutated its inputs")
	}
}

// TestBuildSiteChapterTreeSetsSourcePath covers the "Edit this page" links:
// SourcePath was never populated, so site.go reverse-derived the source from
// the sanitized page filename and any chapter path with a space in it shipped
// a link that 404s on GitHub.
func TestBuildSiteChapterTreeSetsSourcePath(t *testing.T) {
	defs := []config.ChapterDef{{
		Title: "Guide",
		File:  "user guide/README.md",
		Sections: []config.ChapterDef{
			{Title: "Deep", File: "user guide/deep.md"},
		},
	}}
	chapters := []renderer.ChapterHTML{
		{Title: "Guide", ID: "guide"},
		{Title: "Deep", ID: "deep"},
	}
	chapterFiles := []string{"user guide/README.md", "user guide/deep.md"}
	pages := []string{"userguide/index.html", "userguide/deep.html"}

	tree := buildSiteChapterTree(defs, chapters, chapterFiles, pages, nil)
	if len(tree) != 1 || len(tree[0].Children) != 1 {
		t.Fatalf("unexpected tree shape: %+v", tree)
	}
	if tree[0].SourcePath != "user guide/README.md" {
		t.Errorf("SourcePath = %q, want %q", tree[0].SourcePath, "user guide/README.md")
	}
	if tree[0].Children[0].SourcePath != "user guide/deep.md" {
		t.Errorf("nested SourcePath = %q, want %q", tree[0].Children[0].SourcePath, "user guide/deep.md")
	}
}

func TestRelativeSitePage(t *testing.T) {
	tests := []struct {
		from, to, want string
	}{
		{"index.html", "glossary.html", "glossary.html"},
		{"one.html", "glossary.html", "glossary.html"},
		{"guide/deep.html", "glossary.html", "../glossary.html"},
		{"a/b/c.html", "glossary.html", "../../glossary.html"},
	}
	for _, tt := range tests {
		if got := relativeSitePage(tt.from, tt.to); got != tt.want {
			t.Errorf("relativeSitePage(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
		}
	}
}

// TestTOCHeadingsForBookIncludesGlossary pins the printed contents: the
// glossary was printed in the PDF and listed in its bookmarks but missing from
// the table of contents, which in a printed book is the only index there is.
func TestTOCHeadingsForBookIncludesGlossary(t *testing.T) {
	all := []toc.HeadingInfo{
		{Level: 1, Text: "One", ID: "one"},
		{Level: 2, Text: "Sub", ID: "sub"},
	}

	got := tocHeadingsForBook(all, true, 0)
	if len(got) != 3 || got[2].Text != glossaryChapterTitle || got[2].ID != glossaryChapterID {
		t.Fatalf("glossary heading missing from the TOC: %+v", got)
	}
	if got[2].Level != 1 {
		t.Errorf("glossary should be a top-level TOC entry, got level %d", got[2].Level)
	}

	// A book without a GLOSSARY.md must not grow a phantom entry, and the
	// depth cap still applies to everything.
	if got := tocHeadingsForBook(all, false, 0); len(got) != 2 {
		t.Errorf("no glossary should mean no extra entry, got %+v", got)
	}
	if got := tocHeadingsForBook(all, true, 1); len(got) != 2 || got[1].ID != glossaryChapterID {
		t.Errorf("toc_max_depth 1 should keep both level-1 entries, got %+v", got)
	}

	// The caller reuses allHeadings for other outputs.
	if len(all) != 2 {
		t.Error("tocHeadingsForBook mutated allHeadings")
	}
}
