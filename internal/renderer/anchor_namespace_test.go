package renderer

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/toc"
)

// renderTestTOC builds a table of contents exactly the way the build pipeline
// does, so the tests exercise the real markup (link plus page-number slot).
func renderTestTOC(headings []toc.HeadingInfo) string {
	gen := toc.NewGenerator()
	return gen.RenderHTML(gen.Generate(headings))
}

// TestRemapTOC_DuplicateHeadingAcrossChapters covers the printed PDF table of
// contents pointing two "Overview" entries at the same page: heading ids are
// minted per chapter, so the second chapter's anchor collided with the first
// and Chrome recorded only one destination for it.
func TestRemapTOC_DuplicateHeadingAcrossChapters(t *testing.T) {
	chapters := []struct{ id, html string }{
		{"alpha-chapter", `<h2 id="overview">Overview</h2>`},
		{"beta-chapter", `<h2 id="overview">Overview</h2>`},
	}
	tocHTML := renderTestTOC([]toc.HeadingInfo{
		{Level: 1, Text: "Alpha Chapter", ID: "alpha-chapter"},
		{Level: 2, Text: "Overview", ID: "overview"},
		{Level: 1, Text: "Beta Chapter", ID: "beta-chapter"},
		{Level: 2, Text: "Overview", ID: "overview"},
	})

	ns := newAnchorNamespacer()
	for _, ch := range chapters {
		ns.Reserve(ch.id)
	}
	for _, ch := range chapters {
		ns.MarkChapter(ch.id)
		ns.Rewrite(ch.id, ch.html)
	}
	remapped := ns.RemapTOC(tocHTML, 0)

	if !strings.Contains(remapped, `href="#overview"`) {
		t.Errorf("first Overview entry should keep its original anchor:\n%s", remapped)
	}
	if !strings.Contains(remapped, `href="#beta-chapter--overview"`) {
		t.Errorf("second Overview entry should point at the second chapter's unique anchor:\n%s", remapped)
	}
	// The page-number slot repeats the anchor id; if it is not remapped with
	// the link, the printed page number is still looked up under the colliding
	// id and both entries print the first chapter's page.
	if !strings.Contains(remapped, toc.PageSlotAttr+`="beta-chapter--overview"`) {
		t.Errorf("page-number slot should be remapped with the link:\n%s", remapped)
	}
	if strings.Count(remapped, `href="#overview"`) != 1 {
		t.Errorf("expected exactly one entry left on the colliding id:\n%s", remapped)
	}
}

// TestRemapTOC_HeadingSharingALaterChapterID pins the ordering rule: a heading
// slug that also names a later chapter must resolve to the heading, because the
// table of contents lists them in document order.
func TestRemapTOC_HeadingSharingALaterChapterID(t *testing.T) {
	chapters := []struct{ id, html string }{
		{"intro", `<h2 id="notes">Notes</h2>`},
		{"notes", ``},
	}
	tocHTML := renderTestTOC([]toc.HeadingInfo{
		{Level: 1, Text: "Intro", ID: "intro"},
		{Level: 2, Text: "Notes", ID: "notes"},
		{Level: 1, Text: "Notes", ID: "notes"},
	})

	ns := newAnchorNamespacer()
	for _, ch := range chapters {
		ns.Reserve(ch.id)
	}
	for _, ch := range chapters {
		ns.MarkChapter(ch.id)
		ns.Rewrite(ch.id, ch.html)
	}
	remapped := ns.RemapTOC(tocHTML, 0)

	if !strings.Contains(remapped, `href="#intro--notes"`) {
		t.Errorf("the h2 must point at the renamed heading, not the chapter named Notes:\n%s", remapped)
	}
	if !strings.Contains(remapped, `href="#notes"`) {
		t.Errorf("the chapter entry must keep pointing at the chapter:\n%s", remapped)
	}
}

// TestRemapTOC_DepthFilterKeepsEntriesAligned makes sure headings the TOC
// generator dropped for exceeding output.toc_max_depth do not consume an entry
// and shift every following one onto the wrong anchor.
func TestRemapTOC_DepthFilterKeepsEntriesAligned(t *testing.T) {
	chapters := []struct{ id, html string }{
		{"alpha-chapter", `<h3 id="overview">Overview</h3>`},
		{"beta-chapter", `<h2 id="overview">Overview</h2>`},
	}
	// With toc_max_depth 2 the h3 never reaches the TOC, so the only printed
	// "Overview" is the second chapter's.
	tocHTML := renderTestTOC([]toc.HeadingInfo{
		{Level: 1, Text: "Alpha Chapter", ID: "alpha-chapter"},
		{Level: 1, Text: "Beta Chapter", ID: "beta-chapter"},
		{Level: 2, Text: "Overview", ID: "overview"},
	})

	ns := newAnchorNamespacer()
	for _, ch := range chapters {
		ns.Reserve(ch.id)
	}
	for _, ch := range chapters {
		ns.MarkChapter(ch.id)
		ns.Rewrite(ch.id, ch.html)
	}
	remapped := ns.RemapTOC(tocHTML, 2)

	if !strings.Contains(remapped, `href="#beta-chapter--overview"`) {
		t.Errorf("the printed entry must point at the h2 that produced it:\n%s", remapped)
	}
}

// TestRender_DuplicateHeadingIDsAcrossChapters is the end-to-end version: the
// assembled PDF document must expose one destination per heading, and the
// printed table of contents must link each entry to its own chapter.
func TestRender_DuplicateHeadingIDsAcrossChapters(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme(t))
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		TOCHTML: renderTestTOC([]toc.HeadingInfo{
			{Level: 1, Text: "Alpha Chapter", ID: "alpha-chapter"},
			{Level: 2, Text: "Overview", ID: "overview"},
			{Level: 1, Text: "Beta Chapter", ID: "beta-chapter"},
			{Level: 2, Text: "Overview", ID: "overview"},
		}),
		ChaptersHTML: []ChapterHTML{
			{Title: "Alpha Chapter", ID: "alpha-chapter", Content: `<h2 id="overview">Overview</h2><p><a href="#overview">see</a></p>`},
			{Title: "Beta Chapter", ID: "beta-chapter", Content: `<h2 id="overview">Overview</h2><p><a href="#overview">see</a></p>`},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if got := strings.Count(html, `id="overview"`); got != 1 {
		t.Errorf("assembled document should carry the colliding id once, got %d occurrences", got)
	}
	if !strings.Contains(html, `id="beta-chapter--overview"`) {
		t.Error("the second chapter's heading should get a document-wide-unique id")
	}
	if !strings.Contains(html, `href="#beta-chapter--overview"`) {
		t.Error("the TOC entry and the in-chapter link should follow the renamed heading")
	}
	if got := strings.Count(html, `href="#beta-chapter--overview"`); got != 2 {
		t.Errorf("expected the TOC entry and the chapter's own link to be remapped, got %d", got)
	}
}
