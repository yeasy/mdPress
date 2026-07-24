package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/pdf"
)

// TestBuildPDFTableOfContents_NoFooterNoPageNumbers builds a real PDF with
// output.footer:false and checks that the table of contents does not cite page
// numbers. It used to list "Deep Dive .... 10" while no sheet of the document
// carried a folio, so a reader of a printed copy had no way to find page 10 —
// the two halves of the same feature disagreed.
func TestBuildPDFTableOfContents_NoFooterNoPageNumbers(t *testing.T) {
	if testing.Short() {
		t.Skip("renders a PDF with Chromium; skipped in -short mode")
	}
	if err := pdf.CheckChromiumAvailable(); err != nil {
		t.Skipf("Chromium is not available: %v", err)
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext (poppler) is not installed")
	}

	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Long chapters so the entries land on distinct, multi-digit-free pages
	// that a stray number could not be mistaken for something else.
	filler := strings.Repeat("Filler paragraph for pagination.\n\n", 90)
	write("book.yaml", `book:
  title: "No Folio"
  author: "Tester"
  language: "en"
chapters:
  - title: "Chapter Alpha"
    file: "one.md"
  - title: "Chapter Beta"
    file: "two.md"
output:
  formats: ["pdf"]
  toc: true
  cover: true
  footer: false
`)
	write("one.md", "# Chapter Alpha\n\nStart.\n\n"+filler+"## Section Alpha\n\nEnd.\n")
	write("two.md", "# Chapter Beta\n\nStart.\n\n"+filler+"## Section Beta\n\nEnd.\n")

	t.Chdir(dir)
	cfgFile = "book.yaml"
	buildFormat = "pdf"
	buildOutput = ""
	buildSummary = ""
	buildSubDir = ""
	buildBranch = ""
	quiet = true
	verbose = false

	if err := executeBuild(context.Background(), ""); err != nil {
		t.Fatalf("executeBuild() returned error: %v", err)
	}
	pdfPath := filepath.Join(dir, "No-Folio.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("expected generated PDF at %s: %v", pdfPath, err)
	}

	// The cover is page 1, so the table of contents is page 2.
	toc := pdfPageText(t, pdfPath, 2)
	if !strings.Contains(toc, "Chapter Alpha") {
		t.Fatalf("the table of contents is missing its entries:\n%s", toc)
	}

	numbered := regexp.MustCompile(`(?m)^\s*(Chapter Alpha|Chapter Beta|Section Alpha|Section Beta)\s+\d+\s*$`)
	if m := numbered.FindAllString(toc, -1); len(m) > 0 {
		t.Errorf("the table of contents cites page numbers that no page carries: %q\n%s", m, toc)
	}
}

// TestPDFPrintsPageNumbers checks the guard that decides whether the table of
// contents may cite page numbers: the folio can come from either the footer or
// a header configured with {page}.
func TestPDFPrintsPageNumbers(t *testing.T) {
	tests := []struct {
		name           string
		header, footer string
		want           bool
	}{
		{"default footer", "", defaultPDFFooterTemplate, true},
		{"no header or footer", "", "", false},
		{"header without a page token", "<div>Title</div>", "", false},
		{"header carrying {page}", "<div>" + pdfPageNumberSpan + "</div>", "", true},
		{"footer with only the page count", "", "<div>" + pdfTotalPagesSpan + "</div>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pdfPrintsPageNumbers(tt.header, tt.footer); got != tt.want {
				t.Errorf("pdfPrintsPageNumbers() = %v, want %v", got, tt.want)
			}
		})
	}
}
