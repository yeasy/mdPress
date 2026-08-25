package output

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAssetExtractorExtract pins assetExtractor.Extract's contract. Extract
// walks matches by index in a single pass rather than re-running the pattern
// inside a replace callback -- a match here is a whole base64 image, so the
// second scan dominated site builds -- and every fallback below has to keep
// behaving as it did, because each one leaves an image inline rather than
// losing it.
func TestAssetExtractorExtract(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfake-image-bytes"))
	other := base64.StdEncoding.EncodeToString([]byte("a different image"))

	t.Run("rewrites and dedups by content", func(t *testing.T) {
		dir := t.TempDir()
		a := newAssetExtractor(dir)
		in := `<p><img src="data:image/png;base64,` + png + `" alt="a">` +
			`<img src="data:image/png;base64,` + png + `" alt="same">` +
			`<img src="data:image/png;base64,` + other + `" alt="b"></p>`
		out, err := a.Extract(in, "index.html")
		if err != nil {
			t.Fatalf("Extract: %v", err)
		}
		if strings.Contains(out, "base64,") {
			t.Errorf("data URI survived extraction: %s", out)
		}
		// Identical bytes share one file; different bytes get their own.
		if len(a.names) != 2 {
			t.Errorf("expected 2 distinct assets, got %d: %v", len(a.names), a.names)
		}
		if n := strings.Count(out, `src="`); n != 3 {
			t.Errorf("expected 3 rewritten src attributes, got %d: %s", n, out)
		}
		// Surrounding markup and attribute order are preserved.
		if !strings.Contains(out, `alt="a"`) || !strings.HasPrefix(out, "<p>") || !strings.HasSuffix(out, "</p>") {
			t.Errorf("surrounding markup was disturbed: %s", out)
		}
		entries, _ := os.ReadDir(filepath.Join(dir, siteAssetDir))
		if len(entries) != 2 {
			t.Errorf("expected 2 files written, got %d", len(entries))
		}
	})

	t.Run("href is relative to the page depth", func(t *testing.T) {
		dir := t.TempDir()
		a := newAssetExtractor(dir)
		in := `<img src="data:image/png;base64,` + png + `">`
		root, err := a.Extract(in, "index.html")
		if err != nil {
			t.Fatal(err)
		}
		nested, err := a.Extract(in, "part/ch1.html")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(root, "../") {
			t.Errorf("root page should not reference a parent directory: %s", root)
		}
		if !strings.Contains(nested, "../") {
			t.Errorf("nested page should reach the asset dir via ../: %s", nested)
		}
	})

	t.Run("unconvertible images stay inline", func(t *testing.T) {
		dir := t.TempDir()
		a := newAssetExtractor(dir)
		cases := map[string]string{
			"unknown media type": `<img src="data:image/tiff;base64,` + png + `">`,
			"undecodable base64": `<img src="data:image/png;base64,!!!not-base64!!!">`,
		}
		for name, in := range cases {
			out, err := a.Extract(in, "index.html")
			if err != nil {
				t.Errorf("%s: unexpected error: %v", name, err)
			}
			if out != in {
				t.Errorf("%s: expected the image to be left untouched\n got: %s\nwant: %s", name, out, in)
			}
		}
	})

	t.Run("html without images is returned unchanged", func(t *testing.T) {
		a := newAssetExtractor(t.TempDir())
		in := `<p>No images here, just <code>src="data:"</code> in prose.</p>`
		out, err := a.Extract(in, "index.html")
		if err != nil {
			t.Fatal(err)
		}
		if out != in {
			t.Errorf("content without a data URI was modified:\n got: %s\nwant: %s", out, in)
		}
	})
}
