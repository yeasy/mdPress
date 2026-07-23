// allow_html_test.go drives the real CLI to check that markdown.allow_html
// reaches the parse workers, not just the orchestrator's own parser: chapter
// HTML is produced by per-worker parsers, so a setting wired only into the
// orchestrator would look correct in unit tests and do nothing in a build.
package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rawHTMLChapter = "# Intro\n\n" +
	"<script>alert('xss')</script>\n\n" +
	"<iframe src=\"https://evil.example\"></iframe>\n\n" +
	"Inline <img src=x onerror=\"alert(1)\"> text.\n"

func writeRawHTMLProject(t *testing.T, markdownSettings string) string {
	t.Helper()
	dir := t.TempDir()
	book := `book:
  title: "Raw HTML"
  author: "Test"
  language: "en-US"
chapters:
  - title: "Intro"
    file: "intro.md"
` + markdownSettings
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(book), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "intro.md"), []byte(rawHTMLChapter), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func readBuiltHTML(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "Raw-HTML.html")) //nolint:gosec // G304: test-controlled path
	if err != nil {
		t.Fatalf("read standalone html: %v", err)
	}
	return string(data)
}

func TestAllowHTMLFalseStripsRawHTMLFromBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}
	dir := writeRawHTMLProject(t, "markdown:\n  allow_html: false\n")
	buildFormat(t, dir, "html")

	body := readBuiltHTML(t, dir)
	for _, forbidden := range []string{"<script>alert", "<iframe", "onerror"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("allow_html: false left %q in the built HTML", forbidden)
		}
	}
}

func TestRawHTMLPassesThroughByDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}
	dir := writeRawHTMLProject(t, "")
	buildFormat(t, dir, "html")

	body := readBuiltHTML(t, dir)
	if !strings.Contains(body, "<iframe") {
		t.Error("raw HTML should still pass through when allow_html is not configured")
	}
}
