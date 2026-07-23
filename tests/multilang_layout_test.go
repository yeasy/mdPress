// multilang_layout_test.go pins the shape of a multi-language build.
//
// It used to produce three different layouts depending on how --output was
// written — sibling "dist-en_site" directories next to the project, or
// per-language directories named after the book title (so the URL path
// contained spaces and CJK) — and none of them matched the documented layout
// or could be deployed as-is. The switcher was named "_mdpress_langs.html", so
// serving the directory produced a listing instead of the page.
package tests

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func multilangBook(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	write := func(name, content string) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write("LANGS.md", "# Languages\n\n* [English](en/)\n* [中文](zh/)\n")
	// A root book.yaml carrying shared metadata is the obvious thing to write,
	// and used to fail the build with an error about missing chapters.
	write("book.yaml", "book:\n  title: \"Shared Manual\"\n  author: \"Ann\"\n")
	// Titles with a space and with CJK: neither may reach a URL path.
	write("en/book.yaml", "book:\n  title: \"My Manual\"\nchapters:\n  - title: A\n    file: a.md\n")
	write("en/a.md", "# A\n\nenglish body\n")
	write("zh/book.yaml", "book:\n  title: \"我的手册\"\nchapters:\n  - title: A\n    file: a.md\n")
	write("zh/a.md", "# A\n\n中文正文\n")

	return dir
}

func TestMultilangLayoutIsOneTree(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}

	cases := []struct {
		name string
		args []string
		root func(dir string) string
	}{
		{"no --output", nil, func(dir string) string { return filepath.Join(dir, "_book") }},
		{"--output dist", []string{"--output", "dist"}, func(dir string) string { return filepath.Join(dir, "dist") }},
		{"--output dist/", []string{"--output", "dist" + string(os.PathSeparator)}, func(dir string) string { return filepath.Join(dir, "dist") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := multilangBook(t)
			buildFormatIn(t, dir, "site", tc.args...)
			root := tc.root(dir)

			// Every spelling produces the same deployable tree.
			for _, want := range []string{
				filepath.Join(root, "index.html"),
				filepath.Join(root, "en", "index.html"),
				filepath.Join(root, "zh", "index.html"),
			} {
				if _, err := os.Stat(want); err != nil {
					t.Errorf("missing %s: %v", want, err)
				}
			}

			// No directory named after a book title, and nothing beside the
			// project root.
			for _, unwanted := range []string{
				filepath.Join(dir, "dist-en_site"),
				filepath.Join(dir, "dist-index.html"),
				filepath.Join(dir, "_mdpress_langs.html"),
				filepath.Join(root, "en", "My Manual_site"),
			} {
				if _, err := os.Stat(unwanted); err == nil {
					t.Errorf("stale layout artifact still produced: %s", unwanted)
				}
			}

			// The switcher's links must be usable as URLs.
			data, err := os.ReadFile(filepath.Join(root, "index.html")) //nolint:gosec // G304: test-controlled path
			if err != nil {
				t.Fatal(err)
			}
			hrefs := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(string(data), -1)
			if len(hrefs) == 0 {
				t.Fatal("the language switcher has no links")
			}
			for _, h := range hrefs {
				target := h[1]
				if strings.ContainsAny(target, " ") {
					t.Errorf("href %q contains a raw space", target)
				}
				for _, r := range target {
					if r > 127 {
						t.Errorf("href %q contains a raw non-ASCII character", target)
						break
					}
				}
				if _, err := url.Parse(target); err != nil {
					t.Errorf("href %q is not a valid URL: %v", target, err)
				}
			}
		})
	}
}
