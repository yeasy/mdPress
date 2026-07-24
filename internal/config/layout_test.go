package config

import (
	"os"
	"path/filepath"
	"testing"
)

// loadLayoutConfig writes book.yaml plus one chapter and loads it.
func loadLayoutConfig(t *testing.T, yaml string) *BookConfig {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.md"), []byte("# One\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "book.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

const layoutChapters = `
chapters:
  - title: One
    file: one.md
`

// TestApplyThemeLayoutFillsUnconfiguredGeometry is the regression test for the
// A4-forever bug: DefaultConfig pre-filled style.page_size and style.margin, so
// a theme file's page_size/margins reached no renderer and an A5 theme still
// printed A4 pages.
func TestApplyThemeLayoutFillsUnconfiguredGeometry(t *testing.T) {
	cfg := loadLayoutConfig(t, "book:\n  title: T\n"+layoutChapters)

	cfg.ApplyThemeLayout("A5", MarginConfig{Top: 30, Bottom: 30, Left: 30, Right: 30})

	if cfg.Style.PageSize != "A5" {
		t.Errorf("page size = %q, want A5 from the theme", cfg.Style.PageSize)
	}
	want := MarginConfig{Top: 30, Bottom: 30, Left: 30, Right: 30}
	if cfg.Style.Margin != want {
		t.Errorf("margins = %+v, want %+v from the theme", cfg.Style.Margin, want)
	}
}

// TestApplyThemeLayoutKeepsConfiguredGeometry checks the other direction: the
// style block still beats the theme, per edge.
func TestApplyThemeLayoutKeepsConfiguredGeometry(t *testing.T) {
	cfg := loadLayoutConfig(t, `book:
  title: T
style:
  page_size: Letter
  margin:
    top: 5
`+layoutChapters)

	cfg.ApplyThemeLayout("A5", MarginConfig{Top: 30, Bottom: 30, Left: 30, Right: 30})

	if cfg.Style.PageSize != "Letter" {
		t.Errorf("page size = %q, want the configured Letter", cfg.Style.PageSize)
	}
	want := MarginConfig{Top: 5, Bottom: 30, Left: 30, Right: 30}
	if cfg.Style.Margin != want {
		t.Errorf("margins = %+v, want %+v (configured top, theme elsewhere)", cfg.Style.Margin, want)
	}
}

// TestApplyThemeLayoutIgnoresEmptyTheme keeps the built-in fallbacks: a theme
// that declares no margins must not blank the page out.
func TestApplyThemeLayoutIgnoresEmptyTheme(t *testing.T) {
	cfg := loadLayoutConfig(t, "book:\n  title: T\n"+layoutChapters)
	before := cfg.Style

	cfg.ApplyThemeLayout("", MarginConfig{})

	if cfg.Style.PageSize != before.PageSize {
		t.Errorf("page size = %q, want the default %q", cfg.Style.PageSize, before.PageSize)
	}
	if cfg.Style.Margin != before.Margin {
		t.Errorf("margins = %+v, want the defaults %+v", cfg.Style.Margin, before.Margin)
	}
}
