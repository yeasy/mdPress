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
