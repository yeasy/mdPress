package renderer

import (
	"regexp"
	"strings"
	"testing"
)

var (
	standaloneCDNURLPattern    = regexp.MustCompile(`https://cdn\.jsdelivr\.net/[^'"\s)]+`)
	standalonePinnedNPMPattern = regexp.MustCompile(`/npm/[^/@]+@\d+\.\d+\.\d+/`)
)

// TestStandaloneCDNAssetsArePinnedAndIntegrityChecked mirrors the site check
// for the single-file HTML output. Math no longer appears here at all — it is
// rendered into the file at build time — but Mermaid is too large to inline,
// so its URL still has to be pinned to an exact version and integrity-checked.
func TestStandaloneCDNAssetsArePinnedAndIntegrityChecked(t *testing.T) {
	html := renderStandalone(t, `<div class="mermaid">graph TD; A--&gt;B;</div>`)

	urls := standaloneCDNURLPattern.FindAllString(html, -1)
	if len(urls) == 0 {
		t.Fatal("expected the document to reference CDN assets")
	}
	for _, u := range urls {
		if !standalonePinnedNPMPattern.MatchString(u) {
			t.Errorf("CDN asset %q is not pinned to an exact version", u)
		}
	}
	if got := strings.Count(html, "sha384-"); got != len(urls) {
		t.Errorf("got %d integrity digests for %d CDN assets; every asset needs one", got, len(urls))
	}
	if strings.Contains(html, "{{MERMAID_SRI}}") || strings.Contains(html, "{{KATEX_JS_SRI}}") {
		t.Error("integrity placeholders were left unresolved in the output")
	}
}

// TestStandaloneCDNFailureIsVisibleToTheReader checks that a blocked or
// tampered CDN leaves a notice on the page instead of a blank gap.
func TestStandaloneCDNFailureIsVisibleToTheReader(t *testing.T) {
	html := renderStandalone(t, `<div class="mermaid">graph TD; A--&gt;B;</div>`)

	for _, want := range []string{
		"mdpressAssetFailure('.mermaid'",
		"Diagram not rendered: the Mermaid library could not be loaded",
		".asset-error {",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("document is missing the CDN failure fallback: %q", want)
		}
	}
}
