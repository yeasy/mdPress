// site_assets_test.go covers what a deployed site is made of: image files
// rather than inlined bytes, and the project's own static files.
package tests

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testPNG builds a valid PNG of the given size so the fixture exercises the
// real image pipeline rather than a placeholder the decoder would reject.
func testPNG(width, height int) []byte {
	var raw bytes.Buffer
	for y := 0; y < height; y++ {
		raw.WriteByte(0) // filter: none
		for x := 0; x < width; x++ {
			raw.Write([]byte{0xff, 0x00, 0x00})
		}
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	zw.Write(raw.Bytes()) //nolint:errcheck // in-memory writer
	zw.Close()            //nolint:errcheck

	chunk := func(kind string, payload []byte) []byte {
		var b bytes.Buffer
		binary.Write(&b, binary.BigEndian, uint32(len(payload))) //nolint:errcheck
		body := append([]byte(kind), payload...)
		b.Write(body)
		binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(body)) //nolint:errcheck
		return b.Bytes()
	}

	var ihdr bytes.Buffer
	binary.Write(&ihdr, binary.BigEndian, uint32(width))  //nolint:errcheck
	binary.Write(&ihdr, binary.BigEndian, uint32(height)) //nolint:errcheck
	ihdr.Write([]byte{8, 2, 0, 0, 0})                     // 8-bit RGB

	var png bytes.Buffer
	png.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	png.Write(chunk("IHDR", ihdr.Bytes()))
	png.Write(chunk("IDAT", compressed.Bytes()))
	png.Write(chunk("IEND", nil))
	return png.Bytes()
}

func TestSiteAssets(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}

	dir := t.TempDir()
	write := func(name string, data []byte) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write("book.yaml", []byte(`book:
  title: "Assets"
  language: "en-US"
chapters:
  - title: "One"
    file: "one.md"
  - title: "Two"
    file: "two.md"
`))
	write("pic.png", testPNG(80, 80))
	// The same image on two pages must be stored once, not per page.
	write("one.md", []byte("# One\n\n![shot](pic.png)\n"))
	write("two.md", []byte("# Two\n\n![shot again](pic.png)\n"))
	// static/ is the supported way to ship files the generator does not
	// produce; they must survive into the site root.
	write("static/CNAME", []byte("docs.example.com\n"))
	write("static/.nojekyll", []byte(""))
	write("static/img/logo.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`))

	buildFormat(t, dir, "site")
	siteDir := filepath.Join(dir, "_book")

	t.Run("images are files, not inlined bytes", func(t *testing.T) {
		body := siteText(t, siteDir)
		if strings.Contains(body, "data:image/png;base64") {
			t.Error("page still embeds the image as a data URI; every page pays for it and nothing caches")
		}
		entries, err := os.ReadDir(filepath.Join(siteDir, "assets"))
		if err != nil {
			t.Fatalf("no assets directory: %v", err)
		}
		pngs := 0
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".png") {
				pngs++
			}
		}
		if pngs != 1 {
			t.Errorf("wrote %d png assets, want 1 (identical images must be deduplicated)", pngs)
		}
	})

	t.Run("pages reference the extracted asset", func(t *testing.T) {
		page, err := os.ReadFile(filepath.Join(siteDir, "one.html")) //nolint:gosec // G304: test-controlled path
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(page), "assets/img-") {
			t.Error("page does not reference the extracted asset")
		}
	})

	t.Run("static files land in the site root", func(t *testing.T) {
		for name, want := range map[string]string{
			"CNAME":        "docs.example.com",
			".nojekyll":    "",
			"img/logo.svg": "<svg",
		} {
			data, err := os.ReadFile(filepath.Join(siteDir, name)) //nolint:gosec // G304: test-controlled path
			if err != nil {
				t.Errorf("static file %s missing from the site: %v", name, err)
				continue
			}
			if want != "" && !strings.Contains(string(data), want) {
				t.Errorf("static file %s has unexpected content %q", name, data)
			}
		}
	})
}
