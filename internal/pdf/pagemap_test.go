package pdf

import (
	"fmt"
	"strings"
	"testing"
)

// buildTestPDFObjects assembles a classic-cross-reference PDF from object bodies
// indexed by object number, so a test can describe a page tree and its named
// destinations without hand-counting byte offsets.
func buildTestPDFObjects(objects map[int]string, rootNum int) []byte {
	maxNum := 0
	for num := range objects {
		if num > maxNum {
			maxNum = num
		}
	}

	var buf strings.Builder
	buf.WriteString("%PDF-1.4\n")
	offsets := make(map[int]int, len(objects))
	for num := 1; num <= maxNum; num++ {
		body, ok := objects[num]
		if !ok {
			continue
		}
		offsets[num] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", num, body)
	}

	startxref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxNum+1)
	buf.WriteString("0000000000 65535 f \n")
	for num := 1; num <= maxNum; num++ {
		off, ok := offsets[num]
		if !ok {
			buf.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<</Size %d\n/Root %d 0 R>>\nstartxref\n%d\n%%%%EOF\n",
		maxNum+1, rootNum, startxref)
	return []byte(buf.String())
}

func TestNamedDestinationPagesResolvesPageNumbers(t *testing.T) {
	data := buildTestPDFObjects(map[int]string{
		1: "<</Type /Catalog\n/Pages 2 0 R\n/Dests 6 0 R>>",
		2: "<</Type /Pages\n/Kids [3 0 R 7 0 R]\n/Count 3>>",
		3: "<</Type /Page\n/Parent 2 0 R>>",
		4: "<</Type /Page\n/Parent 7 0 R>>",
		5: "<</Type /Page\n/Parent 7 0 R>>",
		6: "<</intro [3 0 R /XYZ 0 700 0]\n" +
			"/middle [4 0 R /XYZ 0 700 0]\n" +
			"/last <</D [5 0 R /XYZ 0 700 0]>>>>",
		// A nested /Pages node: page numbering must follow reading order
		// across the whole tree, not just the top-level kids.
		7: "<</Type /Pages\n/Parent 2 0 R\n/Kids [4 0 R 5 0 R]\n/Count 2>>",
	}, 1)

	pages, err := namedDestinationPages(data)
	if err != nil {
		t.Fatalf("namedDestinationPages() error = %v", err)
	}
	want := map[string]int{"intro": 1, "middle": 2, "last": 3}
	for name, page := range want {
		if pages[name] != page {
			t.Errorf("destination %q on page %d, want %d", name, pages[name], page)
		}
	}
}

func TestNamedDestinationPagesDecodesEscapedNames(t *testing.T) {
	// Chrome percent-encodes a non-ASCII fragment and then escapes the "%" as
	// "#25" when writing it as a PDF name. The HTML anchor it came from is
	// spelled in plain UTF-8, so both spellings must resolve.
	data := buildTestPDFObjects(map[int]string{
		1: "<</Type /Catalog\n/Pages 2 0 R\n/Dests 4 0 R>>",
		2: "<</Type /Pages\n/Kids [3 0 R]\n/Count 1>>",
		3: "<</Type /Page\n/Parent 2 0 R>>",
		4: "<</#25E6#25A6#2582#25E8#25BF#25B0 [3 0 R /XYZ 0 700 0]>>",
	}, 1)

	pages, err := namedDestinationPages(data)
	if err != nil {
		t.Fatalf("namedDestinationPages() error = %v", err)
	}
	if pages["概述"] != 1 {
		t.Errorf("destination %q not resolved; got map %v", "概述", pages)
	}
}

func TestNamedDestinationPagesReportsUnreadableDocuments(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"not a pdf", []byte("hello")},
		{"no destinations", buildTestPDFObjects(map[int]string{
			1: "<</Type /Catalog\n/Pages 2 0 R>>",
			2: "<</Type /Pages\n/Kids [3 0 R]\n/Count 1>>",
			3: "<</Type /Page\n/Parent 2 0 R>>",
		}, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := namedDestinationPages(tt.data); err == nil {
				t.Error("expected an error so the caller can fall back to a TOC without page numbers")
			}
		})
	}
}

func TestDecodePDFName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain-id", "plain-id"},
		{"a#20b", "a b"},
		{"#25E6", "%E6"},
		{"trailing#", "trailing#"},
	}
	for _, tt := range tests {
		if got := decodePDFName(tt.in); got != tt.want {
			t.Errorf("decodePDFName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
