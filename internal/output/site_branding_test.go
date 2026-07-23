package output

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// onePixelPNG is the smallest valid PNG, used as a stand-in favicon/logo.
const onePixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// brandingBookRoot writes a project directory containing an image at each of
// the given relative paths.
func brandingBookRoot(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	data, err := base64.StdEncoding.DecodeString(onePixelPNG)
	if err != nil {
		t.Fatalf("decode fixture png: %v", err)
	}
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	return root
}

// generateBrandedSite renders a two-page site (one at the root, one nested)
// and returns the output directory.
func generateBrandedSite(t *testing.T, meta SiteMeta, bookRoot string) string {
	t.Helper()
	dir := t.TempDir()
	gen := NewSiteGenerator(meta)
	gen.BookRoot = bookRoot
	gen.AddChapter(SiteChapter{Title: "Chapter 1", ID: "ch1", Filename: "ch1.html", Content: "<h1>Chapter 1</h1>"})
	gen.AddChapter(SiteChapter{Title: "Chapter 2", ID: "ch2", Filename: "part/ch2.html", Content: "<h1>Chapter 2</h1>"})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	return dir
}

func readSitePage(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// TestSiteDefaultBrandingKeepsFooterAndDropsThemeBadge pins the out-of-the-box
// look. The theme badge used to be rendered unconditionally on every published
// site, carrying mdPress's own marketing copy about its themes as a tooltip
// ("Clean, professional style for technical documentation and IT books") — on
// the reader's sidebar, with no way to remove it short of a CSS hack.
func TestSiteDefaultBrandingKeepsFooterAndDropsThemeBadge(t *testing.T) {
	dir := generateBrandedSite(t, SiteMeta{
		Title:            "Test Book",
		Language:         "en-US",
		Theme:            "technical",
		ThemeDescription: "Clean, professional style for technical documentation and IT books",
	}, "")
	page := readSitePage(t, dir, "ch1.html")

	if !strings.Contains(page, "Built with mdPress") {
		t.Error("the default footer line should still be rendered")
	}
	if strings.Contains(page, `class="theme-badge"`) {
		t.Error("the theme badge must not be rendered unless it is turned on")
	}
	if strings.Contains(page, "Clean, professional style") {
		t.Error("mdPress's theme description must not appear on a published site")
	}
	// Falls back to the built-in emoji icon when no favicon is configured.
	if !strings.Contains(page, `<link rel="icon" href="data:image/svg+xml,`) {
		t.Error("expected the built-in emoji favicon by default")
	}
}

// TestSiteBrandingConfiguration checks that a project can put its own favicon,
// logo, copyright and footer on the site, and turn the theme badge back on.
func TestSiteBrandingConfiguration(t *testing.T) {
	root := brandingBookRoot(t, "brand/icon.png", "brand/logo.png")
	footer := `<a href="https://example.com">Acme Docs</a>`
	dir := generateBrandedSite(t, SiteMeta{
		Title:          "Test Book",
		Language:       "en-US",
		Theme:          "technical",
		Favicon:        "brand/icon.png",
		Logo:           "brand/logo.png",
		Copyright:      "© 2026 Acme Inc.",
		FooterHTML:     &footer,
		ShowThemeBadge: true,
	}, root)
	page := readSitePage(t, dir, "ch1.html")

	faviconHref := regexp.MustCompile(`<link rel="icon" href="(assets/favicon-[0-9a-f]+\.png)">`).FindStringSubmatch(page)
	if faviconHref == nil {
		t.Fatal("configured favicon was not rendered as the page icon")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(faviconHref[1]))); err != nil {
		t.Errorf("favicon %s was not written into the site: %v", faviconHref[1], err)
	}

	logoHref := regexp.MustCompile(`<img class="sidebar-logo" src="(assets/logo-[0-9a-f]+\.png)"`).FindStringSubmatch(page)
	if logoHref == nil {
		t.Fatal("configured logo was not rendered in the sidebar")
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(logoHref[1]))); err != nil {
		t.Errorf("logo %s was not written into the site: %v", logoHref[1], err)
	}

	if !strings.Contains(page, "© 2026 Acme Inc.") {
		t.Error("configured copyright notice is missing from the footer")
	}
	if !strings.Contains(page, `<a href="https://example.com">Acme Docs</a>`) {
		t.Error("configured footer HTML is missing")
	}
	if strings.Contains(page, "Built with mdPress") {
		t.Error("a configured footer should replace the default mdPress line")
	}
	if !strings.Contains(page, `<div class="theme-badge">technical</div>`) {
		t.Error("theme badge should appear when explicitly enabled")
	}
}

// TestSiteBrandingHrefsAreRelativeToEachPage guards the nested-page case: a
// chapter written to part/ch2.html has to reach back up to the asset dir.
func TestSiteBrandingHrefsAreRelativeToEachPage(t *testing.T) {
	root := brandingBookRoot(t, "brand/icon.png", "brand/logo.png")
	dir := generateBrandedSite(t, SiteMeta{
		Title:    "Test Book",
		Language: "en-US",
		Favicon:  "brand/icon.png",
		Logo:     "brand/logo.png",
	}, root)

	nested := readSitePage(t, dir, "part/ch2.html")
	if !regexp.MustCompile(`<link rel="icon" href="\.\./assets/favicon-`).MatchString(nested) {
		t.Error("nested page should reference the favicon with a parent-relative path")
	}
	if !regexp.MustCompile(`<img class="sidebar-logo" src="\.\./assets/logo-`).MatchString(nested) {
		t.Error("nested page should reference the logo with a parent-relative path")
	}
}

// TestSiteFooterCanBeRemovedEntirely covers the difference between "not
// configured" (keep the mdPress line) and "configured to nothing" (no line at
// all) — the reason FooterHTML is a pointer.
func TestSiteFooterCanBeRemovedEntirely(t *testing.T) {
	empty := ""
	dir := generateBrandedSite(t, SiteMeta{
		Title:      "Test Book",
		Language:   "en-US",
		FooterHTML: &empty,
	}, "")
	page := readSitePage(t, dir, "ch1.html")

	if strings.Contains(page, "Built with mdPress") {
		t.Error("an explicitly empty footer should remove the mdPress line")
	}
	if strings.Contains(page, `class="build-meta"`) {
		t.Error("an empty footer should not leave an empty footer block behind")
	}
}

// TestSiteBrandingAcceptsExternalAndStaticReferences covers the two ways a
// branding image can already live outside the project tree: an absolute URL,
// and a file shipped through the project's static/ directory.
func TestSiteBrandingAcceptsExternalAndStaticReferences(t *testing.T) {
	root := brandingBookRoot(t, "static/favicon.png")
	dir := generateBrandedSite(t, SiteMeta{
		Title:    "Test Book",
		Language: "en-US",
		Favicon:  "favicon.png",
		Logo:     "https://cdn.example.com/logo.svg",
	}, root)
	page := readSitePage(t, dir, "ch1.html")

	if !strings.Contains(page, `<link rel="icon" href="favicon.png">`) {
		t.Error("a favicon shipped through static/ should be referenced at its published path")
	}
	if !strings.Contains(page, `src="https://cdn.example.com/logo.svg"`) {
		t.Error("an absolute logo URL should be used as-is")
	}
}

// TestSiteBrandingIgnoresMissingImages keeps a typo in book.yaml from turning
// into a broken image on every page.
func TestSiteBrandingIgnoresMissingImages(t *testing.T) {
	dir := generateBrandedSite(t, SiteMeta{
		Title:    "Test Book",
		Language: "en-US",
		Favicon:  "brand/nope.png",
		Logo:     "brand/nope.png",
	}, t.TempDir())
	page := readSitePage(t, dir, "ch1.html")

	if strings.Contains(page, "nope.png") {
		t.Error("a missing branding image must not be referenced by the page")
	}
	if !strings.Contains(page, `<link rel="icon" href="data:image/svg+xml,`) {
		t.Error("a missing favicon should fall back to the built-in emoji icon")
	}
	if strings.Contains(page, `<img class="sidebar-logo"`) {
		t.Error("a missing logo should render no logo element")
	}
}
