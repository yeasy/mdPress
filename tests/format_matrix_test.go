// format_matrix_test.go builds one book containing every construct that has
// historically been handled by some output formats and quietly dropped by
// others, then asserts the same invariants across site, standalone HTML and
// ePub.
//
// The recurring defect this guards against is not any single bug: it is that
// new capabilities get wired into three formats and missed in the fourth, so
// the gap only surfaces when a reader opens the neglected artifact. Each
// assertion below is stated once and checked for every format.
package tests

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// matrixBook writes the fixture book and returns its directory.
//
// Every element here exists to catch a specific class of format-specific
// regression; do not simplify it without replacing the coverage.
func matrixBook(t *testing.T) string {
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

	write("book.yaml", `book:
  title: "Format Matrix"
  author: "Test"
  language: "en-US"
chapters:
  - title: "Intro"
    file: "intro.md"
  - title: "Alpha Overview"
    file: "alpha/overview.md"
  - title: "Beta Overview"
    file: "beta/overview.md"
  - title: "中文章节"
    file: "cjk.md"
    sections:
      - title: "Nested Child"
        file: "nested.md"
`)

	// The prose deliberately uses words from the XHTML boolean-attribute list
	// ("multiple", "open", "required"). They must survive as plain text.
	write("intro.md", `# Intro

mdPress supports multiple output formats and does not require a browser.

See [the alpha overview](alpha/overview.md) for details.

`+"```go\n// Build multiple formats\nfunc main() {}\n```"+`

$E = mc^2$

`+"```mermaid\nflowchart LR\n  A[Edit] --> B[Build]\n```"+`
`)

	// Two chapters with the SAME heading text in different directories: their
	// derived IDs must not collide into a single output file.
	write("alpha/overview.md", `# Overview

ALPHA_UNIQUE_MARKER lives here.
`)
	write("beta/overview.md", `# Overview

BETA_UNIQUE_MARKER lives here.
`)

	write("nested.md", `# Nested Child

NESTED_UNIQUE_MARKER lives here.
`)

	// A CJK-titled chapter: its slug becomes a non-ASCII packaged filename,
	// which every package-document reference must percent-encode.
	write("cjk.md", `# 中文章节

CJK_UNIQUE_MARKER 在这里。
`)

	return dir
}

// buildFormat runs the real CLI, mirroring what a user types.
func buildFormat(t *testing.T, sourceDir, format string) {
	t.Helper()
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".", "build", "--format", format, sourceDir)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build --format %s failed: %v\n%s", format, err, out)
	}
}

// epubText concatenates every XHTML chapter in the generated ePub, and also
// returns the ZIP entry names so structural assertions can inspect them.
func epubText(t *testing.T, epubPath string) (string, []string) {
	t.Helper()
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close() //nolint:errcheck

	var body strings.Builder
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
		if !strings.HasSuffix(f.Name, ".xhtml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close() //nolint:errcheck
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		body.Write(data)
	}
	return body.String(), names
}

// readEpubDoc returns one named entry from the generated ePub.
func readEpubDoc(t *testing.T, epubPath, name string) string {
	t.Helper()
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close() //nolint:errcheck

	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close() //nolint:errcheck
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(data)
	}
	t.Fatalf("ePub entry %q not found", name)
	return ""
}

// siteText concatenates every generated page of the site output.
func siteText(t *testing.T, siteDir string) string {
	t.Helper()
	var body strings.Builder
	err := filepath.WalkDir(siteDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // G304: test-controlled path
		if readErr != nil {
			return readErr
		}
		body.Write(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk site: %v", err)
	}
	return body.String()
}

func TestFormatMatrix(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}
	dir := matrixBook(t)
	buildFormat(t, dir, "site,html,epub")

	epubPath := filepath.Join(dir, "Format-Matrix.epub")
	epubBody, epubNames := epubText(t, epubPath)
	formats := map[string]string{
		"site": siteText(t, filepath.Join(dir, "_book")),
		"html": func() string {
			data, err := os.ReadFile(filepath.Join(dir, "Format-Matrix.html")) //nolint:gosec // G304: test-controlled path
			if err != nil {
				t.Fatalf("read standalone html: %v", err)
			}
			return string(data)
		}(),
		"epub": epubBody,
	}

	for name, body := range formats {
		t.Run(name+"/prose is not mangled by XHTML normalization", func(t *testing.T) {
			// Words on the boolean-attribute list must stay plain text.
			for _, phrase := range []string{
				"supports multiple output formats",
				"does not require a browser",
				"Build multiple formats",
			} {
				if !strings.Contains(body, phrase) {
					t.Errorf("prose %q did not survive into %s output", phrase, name)
				}
			}
			for _, corrupted := range []string{
				`multiple="multiple"`, `required="required"`, `open="open"`, `hidden="hidden"`,
			} {
				if strings.Contains(body, corrupted) {
					t.Errorf("%s output contains %s — a boolean-attribute rewrite leaked into text", name, corrupted)
				}
			}
		})

		t.Run(name+"/chapters with duplicate titles both survive", func(t *testing.T) {
			for _, marker := range []string{"ALPHA_UNIQUE_MARKER", "BETA_UNIQUE_MARKER", "CJK_UNIQUE_MARKER", "NESTED_UNIQUE_MARKER"} {
				if !strings.Contains(body, marker) {
					t.Errorf("%s output lost the chapter containing %s (ID collision)", name, marker)
				}
			}
		})

		t.Run(name+"/cross-chapter links are rewritten", func(t *testing.T) {
			// A raw .md href is dead in every published format.
			if strings.Contains(body, `href="alpha/overview.md"`) ||
				strings.Contains(body, `href="./alpha/overview.md"`) {
				t.Errorf("%s output still links to the Markdown source instead of the generated page", name)
			}
		})
	}

	t.Run("epub/zip has no duplicate entries", func(t *testing.T) {
		seen := map[string]bool{}
		for _, n := range epubNames {
			if seen[n] {
				t.Errorf("duplicate ZIP entry %q — readers show only one of the colliding chapters", n)
			}
			seen[n] = true
		}
	})

	t.Run("epub/navigation keeps the chapter hierarchy", func(t *testing.T) {
		// A book using `sections:` must not be flattened: reading systems show
		// the navigation document as the table of contents, and a flat list
		// loses the structure the author wrote.
		nav := readEpubDoc(t, epubPath, "OEBPS/nav.xhtml")
		if !regexp.MustCompile(`(?s)<li>.*?<ol>`).MatchString(nav) {
			t.Error("nav.xhtml has no nested list; the sections hierarchy was flattened")
		}
		ncx := readEpubDoc(t, epubPath, "OEBPS/toc.ncx")
		if !regexp.MustCompile(`(?s)<navPoint[^>]*>.*?<content[^>]*/>\s*<navPoint`).MatchString(ncx) {
			t.Error("toc.ncx has no nested navPoint; the sections hierarchy was flattened")
		}
	})

	t.Run("epub/identifiers agree", func(t *testing.T) {
		// A dtb:uid that does not match the OPF unique identifier is an
		// epubcheck error and confuses library de-duplication.
		opf := readEpubDoc(t, epubPath, "OEBPS/content.opf")
		ncx := readEpubDoc(t, epubPath, "OEBPS/toc.ncx")
		uid := regexp.MustCompile(`dtb:uid"?\s*content="([^"]*)"`).FindStringSubmatch(ncx)
		ident := regexp.MustCompile(`<dc:identifier[^>]*>([^<]*)<`).FindStringSubmatch(opf)
		if uid == nil || ident == nil {
			t.Fatal("could not read both identifiers")
		}
		if uid[1] != ident[1] {
			t.Errorf("dtb:uid %q does not match the OPF identifier %q", uid[1], ident[1])
		}
	})

	t.Run("epub/code blocks keep their highlighting", func(t *testing.T) {
		// Chapters carry chroma class markup; without the matching rules every
		// code block renders as undifferentiated plain text.
		css := readEpubDoc(t, epubPath, "OEBPS/style.css")
		if !strings.Contains(css, ".chroma") && !regexp.MustCompile(`\.(kd|nf|s1|k)\s*\{`).MatchString(css) {
			t.Error("packaged stylesheet has no syntax-highlighting rules")
		}
	})

	t.Run("epub/package references are valid URIs", func(t *testing.T) {
		// OCF requires every path in the package document, NCX and nav to be a
		// valid URI. A CJK chapter title yields a non-ASCII filename, which
		// must therefore be percent-encoded in those references — the ZIP
		// entry itself keeps the readable UTF-8 name.
		for _, doc := range []string{"OEBPS/content.opf", "OEBPS/toc.ncx", "OEBPS/nav.xhtml"} {
			body := readEpubDoc(t, epubPath, doc)
			for _, attr := range []string{"href", "src"} {
				for _, ref := range regexp.MustCompile(attr+`="([^"]+)"`).FindAllStringSubmatch(body, -1) {
					for _, r := range ref[1] {
						if r > 127 {
							t.Errorf("%s contains a non-percent-encoded %s=%q", doc, attr, ref[1])
							break
						}
					}
				}
			}
		}
	})
}

// mdpressBinary builds the CLI once per test run and returns its path.
//
// Tests that need the working directory to be the book (so a relative --output
// resolves against the project) cannot use "go run": it requires the module
// directory as cwd.
var mdpressBinary = sync.OnceValues(func() (string, error) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		return "", err
	}
	// Windows will not exec a file without the .exe suffix, so the extension
	// is part of the name rather than cosmetic.
	name := "mdpress-testbin"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(os.TempDir(), name)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	cmd.Dir = repoRoot
	if out, buildErr := cmd.CombinedOutput(); buildErr != nil {
		return "", fmt.Errorf("build mdpress: %w\n%s", buildErr, out)
	}
	return bin, nil
})

// buildFormatIn runs the CLI from inside sourceDir, so a relative --output
// resolves against the project rather than the repository.
func buildFormatIn(t *testing.T, sourceDir, format string, extra ...string) {
	t.Helper()
	bin, err := mdpressBinary()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	args := append([]string{"build", "--format", format, "--no-cache"}, extra...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = sourceDir
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("build --format %s %v failed: %v\n%s", format, extra, runErr, out)
	}
}
