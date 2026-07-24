package cmd

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/renderer"
)

func TestRewriteMarkdownLinksInHTML(t *testing.T) {
	targets := map[string]string{
		"README.md":             "preface",
		"chapter1/README.md":    "chapter-1",
		"chapter2/section.md":   "section-2",
		"appendix/reference.md": "appendix-ref",
	}

	html := `<p><a href="../README.md">Home</a></p>
<p><a href="section.md">Sibling</a></p>
<p><a href="../appendix/reference.md#tips">Appendix</a></p>
<p><a href="https://example.com/README.md">External</a></p>
<p><a href="#local-anchor">Local</a></p>`

	got := rewriteMarkdownLinksInHTML(html, "chapter2/current.md", targets)

	for want, name := range map[string]string{
		`href="#preface"`:                      "root readme should rewrite to chapter anchor",
		`href="#section-2"`:                    "relative markdown link should rewrite to target chapter anchor",
		`href="#tips"`:                         "markdown link with fragment should keep target fragment for single-page output",
		`href="https://example.com/README.md"`: "external markdown-looking URL should remain unchanged",
		`href="#local-anchor"`:                 "local anchor should remain unchanged",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%s: got %q, want substring %q", name, got, want)
		}
	}
}

func TestRewriteChapterLinksLenMismatch(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{Title: "Only", ID: "only", Content: `<a href="other.md">Other</a>`},
	}

	got := rewriteChapterLinks(chapters, nil)
	if got[0].Content != chapters[0].Content {
		t.Fatalf("length mismatch should leave content unchanged, got %q", got[0].Content)
	}
}

// TestRewriteEpubGlossaryLinksRepointsTermLinks pins the fix for dead ePub
// glossary links: Glossary.ProcessHTML injects same-document "#glossary-<term>"
// links into every chapter, correct for the single-file HTML and PDF, but in an
// ePub each chapter is its own flat OEBPS/*.xhtml file and the "glossary-<term>"
// anchors live only in glossary.xhtml, so left alone every highlighted term
// jumped nowhere.
func TestRewriteEpubGlossaryLinksRepointsTermLinks(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{Title: "One", ID: "one", Content: `<p><a href="#glossary-api">API</a></p>`},
		{Title: "Two", ID: "two", Content: `<p><a href="#glossary-api">API</a></p>`},
		{Title: glossaryChapterTitle, ID: glossaryChapterID, Content: `<dt id="glossary-api">API</dt> <a href="#glossary-markdown">see</a>`},
	}
	chapterFiles := []string{"one.md", "two.md", ""}

	got := rewriteEpubGlossaryLinks(chapters, chapterFiles)

	// ePub documents are flat, so a chapter reaches the glossary by bare filename.
	for _, i := range []int{0, 1} {
		if want := `href="glossary.xhtml#glossary-api"`; !strings.Contains(got[i].Content, want) {
			t.Errorf("chapter %d should link to %s, got %q", i, want, got[i].Content)
		}
		if strings.Contains(got[i].Content, `href="#glossary-`) {
			t.Errorf("chapter %d still has a dead same-document glossary link: %q", i, got[i].Content)
		}
	}

	// The glossary chapter's own definition anchor and cross-term links must stay
	// same-document: those anchors resolve within glossary.xhtml itself.
	if !strings.Contains(got[2].Content, `<dt id="glossary-api">`) {
		t.Errorf("glossary definition anchor was altered: %q", got[2].Content)
	}
	if !strings.Contains(got[2].Content, `href="#glossary-markdown"`) {
		t.Errorf("glossary self cross-term link was clobbered: %q", got[2].Content)
	}

	// The caller's slice must not be mutated in place.
	if strings.Contains(chapters[0].Content, "glossary.xhtml") {
		t.Error("rewriteEpubGlossaryLinks mutated its input")
	}

	// A book without a synthesized glossary appendix must be returned untouched.
	noGloss := []renderer.ChapterHTML{{Title: "One", ID: "one", Content: `<p><a href="#glossary-api">API</a></p>`}}
	if out := rewriteEpubGlossaryLinks(noGloss, []string{"one.md"}); out[0].Content != noGloss[0].Content {
		t.Errorf("without a glossary chapter content must be unchanged, got %q", out[0].Content)
	}
}

func TestRewriteMarkdownLinksInHTMLMarksUnresolvedMarkdownLinks(t *testing.T) {
	got := rewriteMarkdownLinksInHTML(`<p><a href="../appendix/missing.md">Missing</a></p>`, "chapter1/current.md", map[string]string{
		"chapter1/current.md": "current",
	})

	if !strings.Contains(got, `data-mdpress-link="unresolved-markdown"`) {
		t.Fatalf("expected unresolved markdown link marker, got %q", got)
	}
	if !strings.Contains(got, `href="../appendix/missing.md"`) {
		t.Fatalf("expected original href to be preserved, got %q", got)
	}
}
