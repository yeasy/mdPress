package pdf

import (
	"fmt"
	"strings"
	"testing"
)

// The streams below are trimmed copies of what Chrome's PrintToPDF actually
// wrote for a book with a cover, style.header "{title}"/"{author}" and
// style.footer "Page {page} of {pages}": every page ends with the same
// self-contained "q … Q" group holding the running head and the folio, and only
// the folio's glyphs differ from sheet to sheet.

const testCoverArtwork = `1 0 0 -1 0 842 cm
q
1 1 1 rg
0 0 595 842 re
f
Q
q
0 0 595 842 re
W* n
/NonStruct <</MCID 0 >>BDC
BT
/F5 46 Tf
1 0 0 -1 100 400 Tm
<00370048> Tj
ET
EMC
Q
`

const testTOCArtwork = `1 0 0 -1 0 842 cm
q
1 1 1 rg
0 0 595 842 re
f
Q
q
0 0 595 842 re
W* n
/NonStruct <</MCID 0 >>BDC
BT
/F9 16 Tf
1 0 0 -1 60 120 Tm
<0026005200510057> Tj
ET
EMC
Q
`

// testRunningHead is the trailing group Chrome appends to every sheet. folio is
// the glyph of the page number, the only thing that changes between sheets.
func testRunningHead(folio string) string {
	return `q
0 0 595 842 re
W* n
q
.75 0 0 .75 0 0 cm
.3176 .3373 .3647 rg
/G3 gs
BT
/F24 9 Tf
1 0 0 -1 37.78125 28 Tm
<002600520059004800550003> Tj
ET
BT
/F24 9 Tf
1 0 0 -1 373.8125 1101 Tm
` + folio + ` Tj
ET
Q
Q
`
}

func TestStripRunningHeadRemovesTheFolioFromTheCover(t *testing.T) {
	cover := testCoverArtwork + testRunningHead("<0014>")
	next := testTOCArtwork + testRunningHead("<0015>")

	got, err := stripRunningHead([]byte(cover), []byte(next))
	if err != nil {
		t.Fatalf("stripRunningHead() error = %v", err)
	}
	if strings.Contains(string(got), "/F24") {
		t.Errorf("the running head's font is still drawn on the cover:\n%s", got)
	}
	// The cover's own artwork must survive untouched — the whole point of
	// refusing to guess is that this text belongs to the book.
	for _, want := range []string{"/F5 46 Tf", "<00370048> Tj", "0 0 595 842 re"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("cover artwork lost %q:\n%s", want, got)
		}
	}
}

func TestStripRunningHeadKeepsMarkedContentBalanced(t *testing.T) {
	// Chrome closes the content's last marked-content sequence from inside the
	// running-head group. Dropping that EMC with the group would leave the
	// sequence open, and a reader walking the tagged structure would then
	// attribute the rest of the book to an element that starts on the cover.
	openSequence := "/NonStruct <</MCID 9 >>BDC\nq\n1 0 0 1 0 0 cm\n"
	cover := testCoverArtwork + openSequence + "Q\n" +
		strings.Replace(testRunningHead("<0014>"), "/G3 gs\n", "/G3 gs\nEMC\n", 1)
	next := testTOCArtwork + testRunningHead("<0015>")

	got, err := stripRunningHead([]byte(cover), []byte(next))
	if err != nil {
		t.Fatalf("stripRunningHead() error = %v", err)
	}
	if opens, closes := strings.Count(string(got), "BDC"), strings.Count(string(got), "EMC"); opens != closes {
		t.Errorf("marked content left unbalanced: %d BDC, %d EMC\n%s", opens, closes, got)
	}
}

func TestStripRunningHeadRejectsUnrelatedTrailingGroups(t *testing.T) {
	// No header or footer was printed, so each page's last group is its own
	// artwork. Removing it would erase part of the cover.
	if _, err := stripRunningHead([]byte(testCoverArtwork), []byte(testTOCArtwork)); err == nil {
		t.Fatal("stripRunningHead() removed a group that is not a running head")
	}
}

func TestStripRunningHeadRejectsUnsafeOperators(t *testing.T) {
	// A running head carrying an image would take a form XObject — which may
	// hold marked content of its own — out of the page, so the rewrite backs
	// off instead.
	withImage := func(folio string) string {
		return strings.Replace(testRunningHead(folio), "/G3 gs\n", "/G3 gs\n/X7 Do\n", 1)
	}
	_, err := stripRunningHead([]byte(testCoverArtwork+withImage("<0014>")),
		[]byte(testTOCArtwork+withImage("<0015>")))
	if err == nil {
		t.Fatal("stripRunningHead() removed a group containing an XObject")
	}
	if !strings.Contains(err.Error(), `"Do"`) {
		t.Errorf("error should name the operator it refused, got %v", err)
	}
}

func TestStripRunningHeadRefusesToBlankAPage(t *testing.T) {
	// A page whose entire drawing is one group has no running head to take off;
	// removing the group would leave a blank sheet.
	only := "1 0 0 -1 0 842 cm\n" + testRunningHead("<0014>")
	if _, err := stripRunningHead([]byte(only), []byte(only)); err == nil {
		t.Fatal("stripRunningHead() emptied a page that holds a single group")
	}
}

func TestStripRunningHeadRejectsUnbalancedStreams(t *testing.T) {
	broken := testCoverArtwork + "q\n0 0 595 842 re\nW* n\n"
	if _, err := stripRunningHead([]byte(broken), []byte(testTOCArtwork+testRunningHead("<0015>"))); err == nil {
		t.Fatal("stripRunningHead() accepted a stream with an unclosed group")
	}
}

// testStreamObject renders an uncompressed content stream object body.
func testStreamObject(payload string) string {
	return fmt.Sprintf("<</Length %d>> stream\n%s\nendstream", len(payload), payload)
}

// buildTestCoverPDF assembles a two-page document whose pages carry the given
// content streams.
func buildTestCoverPDF(coverStream, nextStream string) []byte {
	return buildTestPDFObjects(map[int]string{
		1: "<</Type /Catalog\n/Pages 2 0 R>>",
		2: "<</Type /Pages\n/Kids [3 0 R 4 0 R]\n/Count 2>>",
		3: "<</Type /Page\n/Parent 2 0 R\n/Contents 5 0 R\n/StructParents 0>>",
		4: "<</Type /Page\n/Parent 2 0 R\n/Contents 6 0 R\n/StructParents 1>>",
		5: testStreamObject(coverStream),
		6: testStreamObject(nextStream),
	}, 1)
}

func TestStripCoverHeaderFooterRewritesOnlyTheCover(t *testing.T) {
	data := buildTestCoverPDF(
		testCoverArtwork+testRunningHead("<0014>"),
		testTOCArtwork+testRunningHead("<0015>"),
	)

	out, err := StripCoverHeaderFooter(data)
	if err != nil {
		t.Fatalf("StripCoverHeaderFooter() error = %v", err)
	}

	offsets, trailer, _, ok := resolveXref(out)
	if !ok {
		t.Fatal("the rewritten document's cross-reference chain is not readable")
	}
	rootNum, rootGen, _ := dictObjectRef(trailer, "Root")
	catalog, ok := indirectDictBody(out, offsets, rootNum, rootGen)
	if !ok {
		t.Fatal("the rewritten document's catalog is not readable")
	}
	pages, err := orderedPageObjects(out, offsets, catalog)
	if err != nil {
		t.Fatalf("orderedPageObjects() error = %v", err)
	}
	if len(pages) != 2 || pages[0] != 3 || pages[1] != 4 {
		t.Fatalf("page objects changed: got %v, want [3 4]", pages)
	}

	cover, err := readPage(out, offsets, pages[0])
	if err != nil {
		t.Fatalf("readPage(cover) error = %v", err)
	}
	if strings.Contains(string(cover.content), "/F24") {
		t.Errorf("the cover still carries the running head:\n%s", cover.content)
	}
	if !strings.Contains(string(cover.content), "<00370048> Tj") {
		t.Errorf("the cover lost its own artwork:\n%s", cover.content)
	}
	// The page keeps its object number and every other entry, so named
	// destinations, outline entries and the structure tree still resolve.
	if !strings.Contains(string(cover.dict), "/StructParents 0") {
		t.Errorf("the rewritten page dropped /StructParents:\n%s", cover.dict)
	}

	second, err := readPage(out, offsets, pages[1])
	if err != nil {
		t.Fatalf("readPage(second) error = %v", err)
	}
	if !strings.Contains(string(second.content), "/F24") {
		t.Errorf("the running head was taken off page two as well:\n%s", second.content)
	}
}

func TestStripCoverHeaderFooterLeavesUnrecognizedDocumentsAlone(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "no running head to recognize",
			data: buildTestCoverPDF(testCoverArtwork, testTOCArtwork),
		},
		{
			name: "single page",
			data: buildTestPDFObjects(map[int]string{
				1: "<</Type /Catalog\n/Pages 2 0 R>>",
				2: "<</Type /Pages\n/Kids [3 0 R]\n/Count 1>>",
				3: "<</Type /Page\n/Parent 2 0 R\n/Contents 4 0 R>>",
				4: testStreamObject(testCoverArtwork + testRunningHead("<0014>")),
			}, 1),
		},
		{
			name: "not a PDF",
			data: []byte("this is not a PDF at all"),
		},
		{
			// Comparing a page against a copy of itself would let any trailing
			// group pass as the running head.
			name: "both pages drawn from the same bytes",
			data: buildTestCoverPDF(
				testCoverArtwork+testRunningHead("<0014>"),
				testCoverArtwork+testRunningHead("<0014>"),
			),
		},
		{
			// An inline image's raw bytes are not tokens, so the scanner cannot
			// tell where the page's drawing groups end.
			name: "inline image on the cover",
			data: buildTestCoverPDF(
				"q\nBI /W 2 /H 2 /BPC 8 /CS /G ID \x00qQ\xffq\nEI\nQ\n"+testRunningHead("<0014>"),
				testTOCArtwork+testRunningHead("<0015>"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := StripCoverHeaderFooter(tt.data)
			if err == nil {
				t.Fatal("StripCoverHeaderFooter() rewrote a document it cannot recognize")
			}
			if string(out) != string(tt.data) {
				t.Error("a refused document must be returned byte-for-byte unchanged")
			}
		})
	}
}

func TestStripCoverHeaderFooterSkipsEncryptedDocuments(t *testing.T) {
	data := buildTestCoverPDF(
		testCoverArtwork+testRunningHead("<0014>"),
		testTOCArtwork+testRunningHead("<0015>"),
	)
	// Rewriting a stream inside an encrypted document would write plaintext
	// where the reader expects ciphertext.
	encrypted := strings.Replace(string(data), "/Root 1 0 R>>", "/Root 1 0 R\n/Encrypt 9 0 R>>", 1)
	if _, err := StripCoverHeaderFooter([]byte(encrypted)); err == nil {
		t.Fatal("StripCoverHeaderFooter() modified an encrypted document")
	}
}

func TestGeneratorLeavesTheCoverAloneUnlessAskedTo(t *testing.T) {
	data := buildTestCoverPDF(
		testCoverArtwork+testRunningHead("<0014>"),
		testTOCArtwork+testRunningHead("<0015>"),
	)
	tests := []struct {
		name    string
		options []GeneratorOption
		strip   bool
	}{
		{name: "cover with a footer", options: []GeneratorOption{WithCoverPage(true), WithHeaderFooter(true)}, strip: true},
		// Page one is the table of contents or the first chapter: it wants the
		// running head like any other page.
		{name: "no cover", options: []GeneratorOption{WithCoverPage(false), WithHeaderFooter(true)}},
		{name: "no header or footer", options: []GeneratorOption{WithCoverPage(true), WithHeaderFooter(false)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator(tt.options...)
			got := g.clearCoverHeaderFooter(data)
			if changed := string(got) != string(data); changed != tt.strip {
				t.Errorf("document changed = %v, want %v", changed, tt.strip)
			}
		})
	}
}

func TestScanContentStreamSeparatesOperandsFromOperators(t *testing.T) {
	tokens, ok := scanContentStream([]byte("q\n1 0 0 -1 0 842 cm\n/F24 9 Tf\n[(a) -250 (b)] TJ\nW*\nn\nQ\n"))
	if !ok {
		t.Fatal("scanContentStream() failed on a well-formed stream")
	}
	var operators []string
	for _, tok := range tokens {
		if tok.operator {
			operators = append(operators, tok.text)
		}
	}
	want := []string{"q", "cm", "Tf", "TJ", "W*", "n", "Q"}
	if strings.Join(operators, " ") != strings.Join(want, " ") {
		t.Errorf("operators = %v, want %v", operators, want)
	}
}
