package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// buildTestPDF assembles a minimal but structurally valid PDF with a classic
// cross-reference table, mirroring what Chrome's print pipeline emits. Object 1
// is the /Info dictionary, whose body the caller supplies.
func buildTestPDF(infoDict string) []byte {
	bodies := []string{
		"1 0 obj\n" + infoDict + "\nendobj\n",
		"2 0 obj\n<</Type /Catalog /Pages 3 0 R>>\nendobj\n",
		"3 0 obj\n<</Type /Pages /Kids [4 0 R] /Count 1>>\nendobj\n",
		"4 0 obj\n<</Type /Page /Parent 3 0 R /MediaBox [0 0 595 842]>>\nendobj\n",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(bodies))
	for i, body := range bodies {
		offsets[i] = buf.Len()
		buf.WriteString(body)
	}
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", len(bodies)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<</Size %d\n/Root 2 0 R\n/Info 1 0 R>>\nstartxref\n%d\n%%%%EOF\n",
		len(bodies)+1, xrefOffset)
	return buf.Bytes()
}

// readInfoValues resolves the /Info dictionary the way a PDF reader would —
// through the trailer and cross-reference table — and returns its decoded
// entries. Asserting on this rather than on raw bytes keeps the tests about
// observable metadata.
func readInfoValues(t *testing.T, data []byte) map[string]string {
	t.Helper()
	offsets, trailer, _, ok := resolveXref(data)
	if !ok {
		t.Fatalf("resolveXref failed on rewritten PDF")
	}
	num, gen, ok := dictObjectRef(trailer, "Info")
	if !ok {
		t.Fatalf("trailer has no /Info reference: %s", trailer)
	}
	off, ok := offsets[num]
	if !ok {
		t.Fatalf("no xref entry for /Info object %d", num)
	}
	_, _, body, ok := readIndirectDict(data, off, num, gen)
	if !ok {
		t.Fatalf("/Info object %d is not a dictionary", num)
	}
	entries, ok := parsePDFDict(body)
	if !ok {
		t.Fatalf("cannot parse /Info dictionary: %s", body)
	}
	values := make(map[string]string, len(entries))
	for _, e := range entries {
		values[e.key] = decodePDFString(t, e.value)
	}
	return values
}

// decodePDFString turns a PDF string object back into Go text.
func decodePDFString(t *testing.T, raw []byte) string {
	t.Helper()
	s := string(raw)
	switch {
	case strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")"):
		inner := s[1 : len(s)-1]
		r := strings.NewReplacer(`\(`, "(", `\)`, ")", `\\`, `\`)
		return r.Replace(inner)
	case strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">"):
		hex := s[1 : len(s)-1]
		if !strings.HasPrefix(hex, "FEFF") {
			t.Fatalf("hex string %q is not UTF-16BE with a BOM", s)
		}
		var units []uint16
		for i := 4; i+3 < len(hex)+1 && i+4 <= len(hex); i += 4 {
			var u uint16
			if _, err := fmt.Sscanf(hex[i:i+4], "%04X", &u); err != nil {
				t.Fatalf("bad UTF-16 unit in %q: %v", s, err)
			}
			units = append(units, u)
		}
		return string(decodeUTF16(units))
	default:
		return s
	}
}

func decodeUTF16(units []uint16) []rune {
	var runes []rune
	for i := 0; i < len(units); i++ {
		u := units[i]
		if u >= 0xD800 && u <= 0xDBFF && i+1 < len(units) {
			lo := units[i+1]
			runes = append(runes, ((rune(u)-0xD800)<<10|(rune(lo)-0xDC00))+0x10000)
			i++
			continue
		}
		runes = append(runes, rune(u))
	}
	return runes
}

const chromeLikeInfo = "<</Title (Old Title)\n" +
	"/Creator (Mozilla/5.0 \\(Macintosh; Intel Mac OS X 10_15_7\\) AppleWebKit/537.36 " +
	"\\(KHTML, like Gecko\\) HeadlessChrome/150.0.0.0 Safari/537.36)\n" +
	"/Producer (Skia/PDF m150)\n" +
	"/CreationDate (D:20260723153154+00'00')\n" +
	"/ModDate (D:20260723153154+00'00')>>"

func TestSetDocumentInfoWritesBookMetadata(t *testing.T) {
	data := buildTestPDF(chromeLikeInfo)
	ts := time.Date(2023, time.November, 14, 22, 13, 20, 0, time.UTC)
	meta := DocumentMetadata{
		Title:   "Metadata Probe Book: A Subtitle",
		Author:  "Jane Author",
		Subject: "A book used to probe PDF metadata.",
		Creator: "mdPress 1.2.3",
	}

	out, err := SetDocumentInfo(data, meta, ts)
	if err != nil {
		t.Fatalf("SetDocumentInfo: %v", err)
	}

	values := readInfoValues(t, out)
	for key, want := range map[string]string{
		"Title":        meta.Title,
		"Author":       meta.Author,
		"Subject":      meta.Subject,
		"Creator":      meta.Creator,
		"CreationDate": "D:20231114221320+00'00'",
		"ModDate":      "D:20231114221320+00'00'",
		// Producer names the rendering engine and must survive the rewrite.
		"Producer": "Skia/PDF m150",
	} {
		if got := values[key]; got != want {
			t.Errorf("/%s = %q, want %q", key, got, want)
		}
	}
	if strings.Contains(string(out), "HeadlessChrome") {
		t.Error("the headless Chrome user agent should no longer appear in the PDF")
	}
	if strings.Contains(string(out), "D:20260723153154") {
		t.Error("Chrome's wall-clock timestamp should no longer appear in the PDF")
	}
	if len(out) != len(data) {
		t.Errorf("rewrite changed the file length (%d -> %d); cross-reference offsets would be stale",
			len(data), len(out))
	}
}

// A dictionary too large for the space the old one occupied must still end up
// readable — the rewriter falls back to an appended incremental update.
func TestSetDocumentInfoAppendsWhenTooLarge(t *testing.T) {
	data := buildTestPDF("<</Title (T)/CreationDate (D:20260723153154+00'00')>>")
	ts := time.Date(2023, time.November, 14, 22, 13, 20, 0, time.UTC)
	meta := DocumentMetadata{
		Title:   "Metadata Probe Book",
		Author:  "Jane Author",
		Subject: strings.Repeat("a very long description ", 20),
		Creator: "mdPress 1.2.3",
	}

	out, err := SetDocumentInfo(data, meta, ts)
	if err != nil {
		t.Fatalf("SetDocumentInfo: %v", err)
	}
	if len(out) <= len(data) {
		t.Fatalf("expected an appended incremental update, file did not grow")
	}

	values := readInfoValues(t, out)
	if values["Subject"] != meta.Subject {
		t.Errorf("/Subject = %q, want %q", values["Subject"], meta.Subject)
	}
	if values["Author"] != meta.Author {
		t.Errorf("/Author = %q, want %q", values["Author"], meta.Author)
	}
	if strings.Contains(string(out), "D:20260723153154") {
		t.Error("the superseded wall-clock timestamp should have been blanked out")
	}
}

func TestSetDocumentInfoEncodesNonASCII(t *testing.T) {
	data := buildTestPDF(chromeLikeInfo)
	meta := DocumentMetadata{Title: "中文手册", Author: "作者 (试)"}

	out, err := SetDocumentInfo(data, meta, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("SetDocumentInfo: %v", err)
	}
	values := readInfoValues(t, out)
	if values["Title"] != "中文手册" {
		t.Errorf("/Title = %q, want %q", values["Title"], "中文手册")
	}
	if values["Author"] != "作者 (试)" {
		t.Errorf("/Author = %q, want %q", values["Author"], "作者 (试)")
	}
}

// Parentheses in metadata must not break out of the PDF literal string.
func TestSetDocumentInfoEscapesLiteralStrings(t *testing.T) {
	data := buildTestPDF(chromeLikeInfo)
	meta := DocumentMetadata{Title: `Escapes ()\ here`}

	out, err := SetDocumentInfo(data, meta, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("SetDocumentInfo: %v", err)
	}
	if got := readInfoValues(t, out)["Title"]; got != `Escapes ()\ here` {
		t.Errorf("/Title = %q, want %q", got, `Escapes ()\ here`)
	}
}

// An unparseable document must come back byte-for-byte unchanged: a build
// should lose its metadata rather than emit a corrupt PDF.
func TestSetDocumentInfoLeavesUnknownFormatsAlone(t *testing.T) {
	for name, data := range map[string][]byte{
		"not a pdf":     []byte("hello world"),
		"no startxref":  []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"),
		"xref stream":   []byte("%PDF-1.5\n5 0 obj\n<</Type /XRef>>\nstream\nendstream\nendobj\nstartxref\n9\n%%EOF\n"),
		"empty content": {},
	} {
		out, err := SetDocumentInfo(data, DocumentMetadata{Title: "x"}, time.Unix(0, 0).UTC())
		if err == nil {
			t.Errorf("%s: expected an error", name)
		}
		if !bytes.Equal(out, data) {
			t.Errorf("%s: input was modified despite the error", name)
		}
	}
}

func TestBuildTimeHonorsSourceDateEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")
	got := buildTime()
	if want := time.Unix(1700000000, 0).UTC(); !got.Equal(want) {
		t.Errorf("buildTime() = %v, want %v", got, want)
	}

	t.Setenv("SOURCE_DATE_EPOCH", "not-a-number")
	if buildTime().IsZero() {
		t.Error("an unparseable SOURCE_DATE_EPOCH should fall back to the current time")
	}
}

func TestPDFDateFormat(t *testing.T) {
	got := pdfDate(time.Date(2023, time.November, 14, 22, 13, 20, 0, time.UTC))
	if want := "D:20231114221320+00'00'"; got != want {
		t.Errorf("pdfDate = %q, want %q", got, want)
	}
}
