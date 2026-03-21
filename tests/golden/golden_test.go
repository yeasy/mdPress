// Package golden implements golden file tests for mdpress output.
// Golden files capture the expected output of HTML generation and are stored in testdata/golden/.
// Run with -update to regenerate golden files after intentional changes.
package golden

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
)

var update = flag.Bool("update", false, "update golden files instead of comparing")

// testdataDir returns the absolute path to the golden testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("failed to resolve testdata dir: %v", err)
	}
	return dir
}

// goldenDir returns the absolute path to the golden output directory.
func goldenDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "golden"))
	if err != nil {
		t.Fatalf("failed to resolve golden dir: %v", err)
	}
	return dir
}

// datePattern matches ISO dates like 2006-01-02 embedded in HTML output.
var datePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// normalizeOutput replaces volatile fields (dates) with a stable placeholder.
func normalizeOutput(s string) string {
	return datePattern.ReplaceAllString(s, "DATE_PLACEHOLDER")
}

// checkGolden compares got against the golden file at path.
// If -update is set the golden file is written instead.
// If the golden file does not yet exist it is created and the test is skipped.
func checkGolden(t *testing.T, path, got string) {
	t.Helper()
	normalized := normalizeOutput(got)

	if *update {
		if err := os.WriteFile(path, []byte(normalized), 0o644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", path, err)
		}
		t.Logf("updated golden file: %s", path)
		return
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: write the golden file and skip so the developer can review.
		if err2 := os.WriteFile(path, []byte(normalized), 0o644); err2 != nil {
			t.Fatalf("failed to create golden file %s: %v", path, err2)
		}
		t.Skipf("golden file created at %s — review and re-run tests", path)
		return
	}
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}

	want := string(data)
	if normalized != want {
		t.Errorf("output does not match golden file %s\n\nwant:\n%s\n\ngot:\n%s",
			path, want, normalized)
	}
}

// loadConfig loads the test book configuration.
func loadConfig(t *testing.T) *config.BookConfig {
	t.Helper()
	cfg, err := config.Load(filepath.Join(testdataDir(t), "book.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

// parseChapters parses all chapter files and returns their HTML and collected headings.
func parseChapters(t *testing.T, cfg *config.BookConfig) ([]renderer.ChapterHTML, []toc.HeadingInfo) {
	t.Helper()
	parser := markdown.NewParser()
	var chapters []renderer.ChapterHTML
	var allHeadings []toc.HeadingInfo

	flat := config.FlattenChapters(cfg.Chapters)
	for _, ch := range flat {
		path := cfg.ResolvePath(ch.File)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read chapter %s: %v", ch.File, err)
		}
		html, headings, err := parser.Parse(data)
		if err != nil {
			t.Fatalf("failed to parse chapter %s: %v", ch.File, err)
		}
		id := strings.TrimSuffix(filepath.Base(ch.File), ".md")
		chapters = append(chapters, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      id,
			Content: html,
		})
		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{
				Level: h.Level,
				Text:  h.Text,
				ID:    h.ID,
			})
		}
	}
	return chapters, allHeadings
}

// TestGoldenHTML generates the full HTML document and compares it against the golden file.
func TestGoldenHTML(t *testing.T) {
	cfg := loadConfig(t)

	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("failed to load theme %q: %v", cfg.Style.Theme, err)
	}

	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	chapters, allHeadings := parseChapters(t, cfg)

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chapters,
	}

	r, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("failed to create HTML renderer: %v", err)
	}
	got, err := r.Render(parts)
	if err != nil {
		t.Fatalf("failed to render HTML: %v", err)
	}

	checkGolden(t, filepath.Join(goldenDir(t), "full.html"), got)
}

// TestGoldenTOC verifies the table of contents structure against the golden file.
func TestGoldenTOC(t *testing.T) {
	cfg := loadConfig(t)

	_, allHeadings := parseChapters(t, cfg)

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	got := tocGen.RenderHTML(tocEntries)

	// Sanity checks before golden comparison.
	if !strings.Contains(got, "<nav") {
		t.Error("TOC HTML must contain <nav> element")
	}
	if !strings.Contains(got, "<ul") {
		t.Error("TOC HTML must contain <ul> element")
	}
	if !strings.Contains(got, "<a") {
		t.Error("TOC HTML must contain <a> elements")
	}
	// All three chapters should have headings in the TOC.
	for _, title := range []string{"Introduction", "Tables", "中文"} {
		if !strings.Contains(got, title) {
			t.Errorf("TOC HTML missing expected heading text %q", title)
		}
	}

	checkGolden(t, filepath.Join(goldenDir(t), "toc.html"), got)
}

// TestGoldenCover verifies the cover page HTML against the golden file.
func TestGoldenCover(t *testing.T) {
	cfg := loadConfig(t)

	coverGen := cover.NewCoverGenerator(cfg.Book)
	got := coverGen.RenderHTML()

	// Sanity checks before golden comparison.
	if !strings.Contains(got, "<!DOCTYPE html>") {
		t.Error("cover must contain DOCTYPE declaration")
	}
	if !strings.Contains(got, cfg.Book.Title) {
		t.Errorf("cover must contain book title %q", cfg.Book.Title)
	}
	if !strings.Contains(got, cfg.Book.Author) {
		t.Errorf("cover must contain author %q", cfg.Book.Author)
	}
	if !strings.Contains(got, cfg.Book.Version) {
		t.Errorf("cover must contain version %q", cfg.Book.Version)
	}
	if !strings.Contains(got, cfg.Book.Subtitle) {
		t.Errorf("cover must contain subtitle %q", cfg.Book.Subtitle)
	}

	checkGolden(t, filepath.Join(goldenDir(t), "cover.html"), got)
}
