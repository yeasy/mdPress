package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seoTestSite builds a two-chapter site whose first chapter is not README, so
// index.html and the first chapter page are duplicates of each other.
func seoTestSite(t *testing.T, meta SiteMeta) string {
	t.Helper()
	dir := t.TempDir()
	gen := NewSiteGenerator(meta)
	gen.AddChapter(SiteChapter{Title: "Preface", Filename: "preface.html", Content: "<p>Welcome.</p>"})
	gen.AddChapter(SiteChapter{Title: "Chapter One", Filename: "chapter01/section1.html", Content: "<p>Body.</p>"})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	return dir
}

func readSiteFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s failed: %v", name, err)
	}
	return string(data)
}

func TestSiteCanonicalAndHomeTitle(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US", SiteURL: "https://example.com/docs"})

	index := readSiteFile(t, dir, "index.html")
	if !strings.Contains(index, "<title>Test Book</title>") {
		t.Error("the site root should be titled with the book, not with the first chapter")
	}
	if !strings.Contains(index, `<meta property="og:title" content="Test Book">`) {
		t.Error("og:title on the site root should be the book title")
	}
	if !strings.Contains(index, `<link rel="canonical" href="https://example.com/docs/">`) {
		t.Error("index.html should declare the site root as canonical")
	}
	if !strings.Contains(index, `<meta property="og:url" content="https://example.com/docs/">`) {
		t.Error("index.html should declare og:url")
	}

	// The first chapter's own page is byte-identical to the site root, so it
	// must point crawlers at the root instead of competing with it.
	preface := readSiteFile(t, dir, "preface.html")
	if !strings.Contains(preface, `<link rel="canonical" href="https://example.com/docs/">`) {
		t.Error("the duplicated first chapter should canonicalize to the site root")
	}

	inner := readSiteFile(t, dir, "chapter01/section1.html")
	want := `<link rel="canonical" href="https://example.com/docs/chapter01/section1.html">`
	if !strings.Contains(inner, want) {
		t.Errorf("nested page should declare its own canonical URL, want %s", want)
	}
	if !strings.Contains(inner, "<title>Chapter One - Test Book</title>") {
		t.Error("content pages should keep the chapter-prefixed title")
	}
}

func TestSiteCanonicalOmittedWithoutSiteURL(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US"})
	index := readSiteFile(t, dir, "index.html")
	if strings.Contains(index, `rel="canonical"`) || strings.Contains(index, "og:url") {
		t.Error("without output.site_url there is no absolute URL to declare, so no canonical/og:url should be emitted")
	}
}

func TestSiteSitemapHasNoDuplicateEntries(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US", SiteURL: "https://example.com/docs"})
	sitemap := readSiteFile(t, dir, "sitemap.xml")
	if got := strings.Count(sitemap, "<loc>"); got != 2 {
		t.Errorf("sitemap should list one URL per distinct page, got %d entries in:\n%s", got, sitemap)
	}
	if strings.Contains(sitemap, "<loc>https://example.com/docs/preface.html</loc>") {
		t.Error("sitemap should not list the first chapter, which the site root already serves")
	}
	if !strings.Contains(sitemap, "<loc>https://example.com/docs/chapter01/section1.html</loc>") {
		t.Error("sitemap should list the remaining pages")
	}
}

func TestSiteWritesHostingFiles(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US", SiteURL: "https://example.com/docs"})

	if _, err := os.Stat(filepath.Join(dir, ".nojekyll")); err != nil {
		t.Errorf(".nojekyll should be generated so GitHub Pages does not run Jekyll: %v", err)
	}
	robots := readSiteFile(t, dir, "robots.txt")
	if !strings.Contains(robots, "User-agent: *") {
		t.Errorf("robots.txt should allow crawling, got:\n%s", robots)
	}
	if !strings.Contains(robots, "Sitemap: https://example.com/docs/sitemap.xml") {
		t.Errorf("robots.txt should point at the generated sitemap, got:\n%s", robots)
	}
}

func TestSiteHostingFilesOmitSitemapLineWithoutSiteURL(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US"})
	robots := readSiteFile(t, dir, "robots.txt")
	if strings.Contains(robots, "Sitemap:") {
		t.Errorf("robots.txt must not advertise a sitemap that was never written, got:\n%s", robots)
	}
}

func TestSiteHostingFilesYieldToStaticDir(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "robots.txt"), []byte("User-agent: *\nDisallow: /\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: "en-US"})
	gen.BookRoot = root
	gen.AddChapter(SiteChapter{Title: "Preface", Filename: "preface.html", Content: "<p>Welcome.</p>"})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	robots := readSiteFile(t, dir, "robots.txt")
	if !strings.Contains(robots, "Disallow: /") {
		t.Errorf("a robots.txt shipped in static/ must win over the generated one, got:\n%s", robots)
	}
}

func TestSiteIndexHighlightsFirstChapterInSidebar(t *testing.T) {
	dir := seoTestSite(t, SiteMeta{Title: "Test Book", Language: "en-US"})
	index := readSiteFile(t, dir, "index.html")
	if !strings.Contains(index, `navFile: "preface.html"`) {
		t.Error("index.html should highlight the first chapter it re-serves, not a non-existent index.html nav entry")
	}
	preface := readSiteFile(t, dir, "preface.html")
	if !strings.Contains(preface, `navFile: ""`) {
		t.Error("content pages should fall back to their own file for nav highlighting")
	}
}

func TestSite404UsesThemeAccentAndAbsoluteHome(t *testing.T) {
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: "en-US"})
	gen.SetCSS(":root {\n  --color-accent: #b5651d;\n  --color-link: #1c5a9e;\n}\nbody { color: #333; }")
	gen.AddChapter(SiteChapter{Title: "Preface", Filename: "preface.html", Content: "<p>Welcome.</p>"})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	page := readSiteFile(t, dir, "404.html")
	if !strings.Contains(page, "--color-accent: #b5651d;") {
		t.Errorf("404.html should define the book's accent color, not fall back to the default blue:\n%s", page)
	}
	if !strings.Contains(page, "mdpress-theme") {
		t.Error("404.html should honor the reader's stored theme choice")
	}
	if !strings.Contains(page, `href="/"`) {
		t.Error("404.html home link should be absolute so it works at any served depth")
	}
}
