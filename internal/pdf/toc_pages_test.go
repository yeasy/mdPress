package pdf

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/toc"
)

// tocDocument renders the table of contents the way the build does, so these
// tests exercise the real markup contract rather than a hand-written copy.
func tocDocument(t *testing.T, entries []toc.TOCEntry) string {
	t.Helper()
	return "<html><body>" + toc.NewGenerator().RenderHTML(entries) + "</body></html>"
}

func TestFillTOCPageNumbers(t *testing.T) {
	doc := tocDocument(t, []toc.TOCEntry{
		{Level: 1, Title: "Introduction", ID: "introduction"},
		{Level: 1, Title: "Appendix", ID: "appendix"},
	})

	if !hasTOCPageSlots(doc) {
		t.Fatal("rendered TOC carries no page-number slots to fill")
	}

	filledDoc, filled := fillTOCPageNumbers(doc, map[string]int{"introduction": 3, "appendix": 17})
	if filled != 2 {
		t.Fatalf("filled %d slots, want 2", filled)
	}
	for _, want := range []string{
		`data-toc-page="introduction">3</span>`,
		`data-toc-page="appendix">17</span>`,
	} {
		if !strings.Contains(filledDoc, want) {
			t.Errorf("filled document is missing %s\ngot: %s", want, filledDoc)
		}
	}
}

func TestFillTOCPageNumbersLeavesUnknownAnchorsEmpty(t *testing.T) {
	// A wrong page number is worse than none: an anchor Chrome never recorded
	// must leave its slot empty rather than borrow a neighbor's number.
	doc := tocDocument(t, []toc.TOCEntry{
		{Level: 1, Title: "Introduction", ID: "introduction"},
		{Level: 1, Title: "Missing", ID: "missing"},
	})

	filledDoc, filled := fillTOCPageNumbers(doc, map[string]int{"introduction": 3})
	if filled != 1 {
		t.Fatalf("filled %d slots, want 1", filled)
	}
	if !strings.Contains(filledDoc, `data-toc-page="missing"></span>`) {
		t.Errorf("unresolved entry did not stay empty\ngot: %s", filledDoc)
	}
}

func TestHasTOCPageSlotsIsFalseWithoutATOC(t *testing.T) {
	if hasTOCPageSlots("<html><body><h1>No table of contents</h1></body></html>") {
		t.Error("a document without a TOC must not trigger the second print pass")
	}
}
