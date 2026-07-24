package cmd

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// customThemeYAML is a complete project theme: theme validation requires
// page_size, font_size, line_height and the two colors, so a user who wants
// custom margins is forced to write a page_size too.
const customThemeYAML = `name: mytheme
page_size: A5
font_family: "Georgia, serif"
font_size: 11
line_height: 1.5
code_theme: github
colors:
  text: "#000000"
  background: "#ffffff"
  heading: "#111111"
  link: "#0066cc"
  code_bg: "#f5f5f5"
  code_text: "#333333"
  accent: "#0066cc"
  border: "#dddddd"
margins:
  top: 60
  bottom: 60
  left: 60
  right: 60
`

// loadThemedProject writes a project with themes/mytheme.yaml and the given
// book.yaml, and returns the loaded config.
func loadThemedProject(t *testing.T, bookYAML string) *config.BookConfig {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join("themes", "mytheme.yaml"), customThemeYAML)
	write("one.md", "# One\n\nBody.\n")
	write("book.yaml", bookYAML)

	cfg, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

// TestOrchestratorAppliesThemePageGeometry is the regression test for a theme
// file whose page_size and margins did nothing. Every renderer reads page
// geometry off the config, and DefaultConfig had already filled style.page_size
// with "A4" and style.margin with 25/25/20/20, so themes/mytheme.yaml saying
// `page_size: A5` still produced A4 pages with A4 margins.
func TestOrchestratorAppliesThemePageGeometry(t *testing.T) {
	cfg := loadThemedProject(t, `book:
  title: Pocket Handbook
style:
  theme: mytheme
chapters:
  - title: One
    file: one.md
`)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if _, err := newBuildOrchestrator(cfg, logger); err != nil {
		t.Fatalf("newBuildOrchestrator: %v", err)
	}

	if cfg.Style.PageSize != "A5" {
		t.Errorf("style.page_size = %q, want A5 from themes/mytheme.yaml", cfg.Style.PageSize)
	}
	want := config.MarginConfig{Top: 60, Bottom: 60, Left: 60, Right: 60}
	if cfg.Style.Margin != want {
		t.Errorf("style.margin = %+v, want %+v from themes/mytheme.yaml", cfg.Style.Margin, want)
	}
}

// TestOrchestratorKeepsConfiguredPageGeometry checks the precedence: book.yaml
// still beats the theme, edge by edge, the way style typography beats it.
func TestOrchestratorKeepsConfiguredPageGeometry(t *testing.T) {
	cfg := loadThemedProject(t, `book:
  title: Pocket Handbook
style:
  theme: mytheme
  page_size: Letter
  margin:
    top: 5
chapters:
  - title: One
    file: one.md
`)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if _, err := newBuildOrchestrator(cfg, logger); err != nil {
		t.Fatalf("newBuildOrchestrator: %v", err)
	}

	if cfg.Style.PageSize != "Letter" {
		t.Errorf("style.page_size = %q, want the configured Letter", cfg.Style.PageSize)
	}
	want := config.MarginConfig{Top: 5, Bottom: 60, Left: 60, Right: 60}
	if cfg.Style.Margin != want {
		t.Errorf("style.margin = %+v, want %+v (configured top, theme elsewhere)", cfg.Style.Margin, want)
	}
}

// TestConfigShowReportsThemePageGeometry keeps `config show` honest: it is the
// documented first debugging step for "I set it and nothing happened", so it
// must print the geometry the build uses, not DefaultConfig's.
func TestConfigShowReportsThemePageGeometry(t *testing.T) {
	cfg := loadThemedProject(t, `book:
  title: Pocket Handbook
style:
  theme: mytheme
chapters:
  - title: One
    file: one.md
`)
	report := newConfigShowReport(cfg, filepath.Join(cfg.BaseDir(), "book.yaml"), false)

	if report.Style.PageSize != "A5" {
		t.Errorf("config show page_size = %q, want A5 from themes/mytheme.yaml", report.Style.PageSize)
	}
	if report.Style.Margin.Top != 60 {
		t.Errorf("config show margin.top = %v, want 60 from themes/mytheme.yaml", report.Style.Margin.Top)
	}
}
