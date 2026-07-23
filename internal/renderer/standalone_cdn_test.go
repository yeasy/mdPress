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
// for the single-file HTML output, which is the format documented as readable
// offline. Its Mermaid URL used to float on the major version and none of the
// four assets carried a Subresource Integrity digest.
func TestStandaloneCDNAssetsArePinnedAndIntegrityChecked(t *testing.T) {
	html := renderStandalone(t, `<div class="mermaid">graph TD; A--&gt;B;</div><span class="math">$E = mc^2$</span>`)

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
	html := renderStandalone(t, `<div class="mermaid">graph TD; A--&gt;B;</div><span class="math">$E = mc^2$</span>`)

	for _, want := range []string{
		"mdpressAssetFailure('.mermaid'",
		"mdpressAssetFailure('.math'",
		"Diagram not rendered: the Mermaid library could not be loaded",
		"Some formulas on this page are not rendered: the KaTeX library could not be loaded",
		".asset-error {",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("document is missing the CDN failure fallback: %q", want)
		}
	}
}
