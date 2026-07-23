package output

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// jsdelivrURLPattern finds every third-party asset URL a published page tells
// the reader's browser to fetch.
var jsdelivrURLPattern = regexp.MustCompile(`https://cdn\.jsdelivr\.net/[^'"\s)]+`)

// pinnedNPMVersionPattern matches an npm path pinned to an exact version
// (npm/pkg@1.2.3/...), as opposed to a floating range such as npm/mermaid@11.
var pinnedNPMVersionPattern = regexp.MustCompile(`/npm/[^/@]+@\d+\.\d+\.\d+/`)

// buildCDNTestSite renders a one-page site containing both a Mermaid diagram
// and a math span, and returns the rendered page.
func buildCDNTestSite(t *testing.T, language string) string {
	t.Helper()
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: language})
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  `<h1>Chapter 1</h1><div class="mermaid">graph TD; A--&gt;B;</div><span class="math">$E = mc^2$</span>`,
	})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "ch1.html"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	return string(data)
}

// TestSiteCDNAssetsArePinnedAndIntegrityChecked guards the supply chain of the
// two libraries a published site loads from a third party. Before this was
// fixed, Mermaid was requested as "mermaid@11" — a floating range whose bytes
// could change under readers with no mdpress release — and none of the four
// assets carried a Subresource Integrity digest, so a tampered CDN response
// would simply execute.
func TestSiteCDNAssetsArePinnedAndIntegrityChecked(t *testing.T) {
	page := buildCDNTestSite(t, "en-US")

	urls := jsdelivrURLPattern.FindAllString(page, -1)
	if len(urls) == 0 {
		t.Fatal("expected the page to reference CDN assets")
	}
	for _, u := range urls {
		if !pinnedNPMVersionPattern.MatchString(u) {
			t.Errorf("CDN asset %q is not pinned to an exact version", u)
		}
	}

	// Every CDN asset must be accompanied by its own integrity digest.
	if got := strings.Count(page, "sha384-"); got != len(urls) {
		t.Errorf("got %d integrity digests for %d CDN assets; every asset needs one", got, len(urls))
	}
	if !strings.Contains(page, "crossOrigin = 'anonymous'") {
		t.Error("CDN assets must be requested with crossorigin=anonymous, or the integrity check cannot run")
	}
}

// TestSiteCDNFailureIsVisibleToTheReader checks that an unreachable or
// tampered CDN degrades into something the reader can see and act on. It used
// to produce only a console.warn: an offline reader got a blank area where the
// diagram should be and raw "$E = mc^2$" where the formula should be, with
// nothing on the page explaining why.
func TestSiteCDNFailureIsVisibleToTheReader(t *testing.T) {
	page := buildCDNTestSite(t, "en-US")

	for _, want := range []string{
		"markAssetFailure('.mermaid'",
		"markAssetFailure('.math'",
		"Diagram not rendered: the Mermaid library could not be loaded",
		"Some formulas on this page are not rendered: the KaTeX library could not be loaded",
		".content .asset-error",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page is missing the CDN failure fallback: %q", want)
		}
	}
}

// TestSiteCDNFailureNoticeIsLocalized keeps the fallback notice in the book's
// language rather than always English.
func TestSiteCDNFailureNoticeIsLocalized(t *testing.T) {
	page := buildCDNTestSite(t, "zh-CN")

	if !strings.Contains(page, "图表未渲染") {
		t.Error("Chinese site should carry the Chinese diagram-failure notice")
	}
	if strings.Contains(page, "Diagram not rendered") {
		t.Error("Chinese site should not fall back to the English notice")
	}
}
