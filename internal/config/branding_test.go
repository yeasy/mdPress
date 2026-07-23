package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeBrandingProject writes a minimal project with one chapter and the given
// extra book.yaml body, returning the config path.
func writeBrandingProject(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "intro.md"), []byte("# Intro\n"), 0o644); err != nil {
		t.Fatalf("write chapter: %v", err)
	}
	yaml := "book:\n  title: Demo\n" + extra + "chapters:\n  - title: Intro\n    file: intro.md\n"
	path := filepath.Join(dir, "book.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}
	return path
}

// TestBrandingKeysAreRecognized checks that the branding settings load and are
// not reported as typos. Before they existed there was no configuration at all
// for a favicon, logo, copyright, footer or the theme badge.
func TestBrandingKeysAreRecognized(t *testing.T) {
	path := writeBrandingProject(t, `  favicon: brand/icon.png
  logo: brand/logo.png
  copyright: "© 2026 Acme Inc."
output:
  footer_html: "<a href=\"https://example.com\">Acme</a>"
  show_theme_badge: true
`)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if unknown := FindUnknownKeys(data); len(unknown) > 0 {
		t.Fatalf("branding keys reported as unknown: %+v", unknown)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Book.Favicon != "brand/icon.png" {
		t.Errorf("book.favicon = %q", cfg.Book.Favicon)
	}
	if cfg.Book.Logo != "brand/logo.png" {
		t.Errorf("book.logo = %q", cfg.Book.Logo)
	}
	if cfg.Book.Copyright != "© 2026 Acme Inc." {
		t.Errorf("book.copyright = %q", cfg.Book.Copyright)
	}
	if cfg.Output.FooterHTML == nil || !strings.Contains(*cfg.Output.FooterHTML, "Acme") {
		t.Errorf("output.footer_html = %v", cfg.Output.FooterHTML)
	}
	if !cfg.Output.ShowThemeBadge {
		t.Error("output.show_theme_badge should be true")
	}
}

// TestBrandingDefaultsAreUnset keeps the defaults distinguishable from user
// choices: an unset footer must stay nil so the site can tell "keep the
// default line" from "the user asked for no line", and the theme badge must
// default to off.
func TestBrandingDefaultsAreUnset(t *testing.T) {
	cfg, err := Load(writeBrandingProject(t, ""))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Output.FooterHTML != nil {
		t.Errorf("output.footer_html should be nil when unset, got %q", *cfg.Output.FooterHTML)
	}
	if cfg.Output.ShowThemeBadge {
		t.Error("output.show_theme_badge should default to false")
	}
	if cfg.Book.Favicon != "" || cfg.Book.Logo != "" || cfg.Book.Copyright != "" {
		t.Error("branding fields should default to empty")
	}
}

// TestBrandingEmptyFooterIsDistinctFromUnset is the whole reason FooterHTML is
// a pointer.
func TestBrandingEmptyFooterIsDistinctFromUnset(t *testing.T) {
	cfg, err := Load(writeBrandingProject(t, "output:\n  footer_html: \"\"\n"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Output.FooterHTML == nil {
		t.Fatal("an explicitly empty footer_html should not read back as unset")
	}
	if *cfg.Output.FooterHTML != "" {
		t.Errorf("output.footer_html = %q, want empty", *cfg.Output.FooterHTML)
	}
}

// TestBrandingPathsMustStayInsideTheProject: branding images are copied into
// the published site, so they must not be able to name a file outside it.
func TestBrandingPathsMustStayInsideTheProject(t *testing.T) {
	_, err := Load(writeBrandingProject(t, "  favicon: ../../etc/passwd\n"))
	if err == nil {
		t.Fatal("expected an escaping favicon path to be rejected")
	}
	if !strings.Contains(err.Error(), "book.favicon") {
		t.Errorf("error should name the offending key, got: %v", err)
	}
}

// TestBrandingAcceptsAbsoluteURLs: an https:// logo is not a project path and
// must not be run through the containment check.
func TestBrandingAcceptsAbsoluteURLs(t *testing.T) {
	cfg, err := Load(writeBrandingProject(t, "  logo: https://cdn.example.com/logo.svg\n"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Book.Logo != "https://cdn.example.com/logo.svg" {
		t.Errorf("book.logo = %q", cfg.Book.Logo)
	}
}
