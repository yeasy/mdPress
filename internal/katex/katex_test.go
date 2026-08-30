package katex

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TestStylesheetReferencesOnlyShippedFonts is the assertion that keeps an EPUB
// valid: the stylesheet may not point at a file the package does not carry.
// Upstream lists WOFF and TrueType alongside WOFF2, and only WOFF2 ships.
func TestStylesheetReferencesOnlyShippedFonts(t *testing.T) {
	css, err := Stylesheet()
	if err != nil {
		t.Fatal(err)
	}
	assetList, err := Assets()
	if err != nil {
		t.Fatal(err)
	}
	shipped := make(map[string]bool, len(assetList))
	for _, a := range assetList {
		shipped[a.Path] = true
	}

	refs := regexp.MustCompile(`url\(([^)]+)\)`).FindAllStringSubmatch(css, -1)
	if len(refs) == 0 {
		t.Fatal("stylesheet references no fonts at all")
	}
	var woff2 int
	for _, m := range refs {
		ref := strings.Trim(m[1], `'"`)
		if !shipped[ref] {
			t.Errorf("stylesheet references %q, which is not packaged", ref)
		}
		if strings.HasSuffix(ref, ".woff2") {
			woff2++
		} else {
			t.Errorf("stylesheet still references a non-WOFF2 font: %q", ref)
		}
	}
	if woff2 < 20 {
		t.Errorf("expected the full WOFF2 font set to survive the rewrite, got %d references", woff2)
	}
	// The rewrite must not leave a dangling separator where the dropped
	// sources used to be.
	for _, bad := range []string{"src:,", ",,", ",}"} {
		if strings.Contains(css, bad) {
			t.Errorf("rewritten stylesheet contains %q", bad)
		}
	}
}

// TestAssetsAreComplete: a missing script or font is invisible until a reader
// opens the book, so check the set here.
func TestAssetsAreComplete(t *testing.T) {
	assetList, err := Assets()
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]Asset, len(assetList))
	var fonts int
	for _, a := range assetList {
		if len(a.Data) == 0 {
			t.Errorf("asset %s is empty", a.Path)
		}
		if a.MediaType == "" {
			t.Errorf("asset %s has no media type", a.Path)
		}
		byPath[a.Path] = a
		if strings.HasPrefix(a.Path, "fonts/") {
			fonts++
		}
	}
	for path, wantType := range map[string]string{
		"katex.min.css":      "text/css",
		"katex.min.js":       "text/javascript",
		"auto-render.min.js": "text/javascript",
		"LICENSE":            "text/plain",
	} {
		a, ok := byPath[path]
		if !ok {
			t.Errorf("%s is not packaged", path)
			continue
		}
		if a.MediaType != wantType {
			t.Errorf("%s media type = %q, want %q", path, a.MediaType, wantType)
		}
	}
	if fonts < 20 {
		t.Errorf("packaged %d fonts, want the full KaTeX set", fonts)
	}
}

// TestVendoredVersionMatchesCDN keeps the bundled copy and the CDN URLs the
// web builds use on the same release, so a formula does not render one way in
// the browser and another in the EPUB.
func TestVendoredVersionMatchesCDN(t *testing.T) {
	for _, url := range []string{utils.KaTeXCSSURL, utils.KaTeXJSURL, utils.KaTeXAutoRenderURL} {
		if !strings.Contains(url, "katex@"+Version+"/") {
			t.Errorf("CDN URL %q does not use the vendored KaTeX %s", url, Version)
		}
	}
	js, err := assets.ReadFile("assets/katex.min.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `version:"`+Version+`"`) {
		t.Errorf("vendored katex.min.js does not report version %s", Version)
	}
}
