package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/pdf"
)

// pdfPageText returns the text of one page of a PDF, laid out as printed.
func pdfPageText(t *testing.T, path string, page int) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "pdftotext", "-layout", //nolint:gosec // G204: test-controlled arguments
		"-f", strconv.Itoa(page), "-l", strconv.Itoa(page), path, "-").Output()
	if err != nil {
		t.Fatalf("pdftotext page %d of %s: %v", page, path, err)
	}
	return string(out)
}

// TestBuildPDFTableOfContents builds a real PDF and reads its table of contents
// back with pdftotext. The printed TOC used to have no heading, no page numbers
// and the chapters' file headings rather than the titles configured in
// book.yaml — so it named chapters differently from every other output format
// and gave a print reader nothing to look a chapter up by.
func TestBuildPDFTableOfContents(t *testing.T) {
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

	// Long chapters so that each one starts on a different page and a wrong
	// page number cannot accidentally be right.
	filler := strings.Repeat("Filler paragraph for pagination.\n\n", 90)
	write("book.yaml", `book:
  title: "TOC Page Numbers"
  author: "Tester"
  language: "en"
chapters:
  - title: "Configured First"
    file: "one.md"
  - title: "Configured Second"
    file: "two.md"
output:
  formats: ["pdf"]
  toc: true
  cover: true
`)
	write("one.md", "# File Heading One\n\nStart.\n\n"+filler+"## Section One\n\nEnd.\n")
	write("two.md", "# File Heading Two\n\nStart.\n\n"+filler+"## Section Two\n\nEnd.\n")

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
	pdfPath := filepath.Join(dir, "TOC-Page-Numbers.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		t.Fatalf("expected generated PDF at %s: %v", pdfPath, err)
	}

	// The cover is page 1, so the table of contents is page 2.
	toc := pdfPageText(t, pdfPath, 2)

	if !strings.Contains(toc, "Contents") {
		t.Errorf("the table of contents page has no heading:\n%s", toc)
	}
	if strings.Contains(toc, "File Heading") {
		t.Errorf("the table of contents uses the file h1 instead of the book.yaml title:\n%s", toc)
	}

	entryLine := regexp.MustCompile(`(?m)^\s*(Configured First|Configured Second|Section One|Section Two)\s+(\d+)\s*$`)
	matches := entryLine.FindAllStringSubmatch(toc, -1)
	if len(matches) != 4 {
		t.Fatalf("expected 4 table-of-contents entries ending in a page number, got %d:\n%s",
			len(matches), toc)
	}

	for _, m := range matches {
		title, page := m[1], m[2]
		pageNum, err := strconv.Atoi(page)
		if err != nil {
			t.Fatalf("page number %q for %q is not a number", page, title)
		}
		// The number printed in the TOC has to be the page the entry is
		// actually on, not a guess and not a counter.
		if got := pdfPageText(t, pdfPath, pageNum); !strings.Contains(got, title) {
			t.Errorf("%q is listed on page %d, but page %d reads:\n%s", title, pageNum, pageNum, got)
		}
	}
}
