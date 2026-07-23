// style_wiring_test.go asserts that typography settings in book.yaml actually
// reach the generated artifacts.
//
// The recurring failure mode this guards is a config field that is declared,
// defaulted, validated and documented, but read by nothing on the path that
// matters. To the user it is invisible: they edit book.yaml, rebuild, see no
// change, and get no diagnostic. A grep-based check cannot catch that (the
// field is usually read by *some* backend), so this drives the real CLI and
// looks for the configured value in the output.
package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStyleConfigReachesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI; skipped in -short mode")
	}

	dir := t.TempDir()
	// Distinctive values that cannot appear by chance in the templates.
	const (
		wantFont       = "MdpressWiringProbe"
		wantFontSize   = "17pt"
		wantLineHeight = "2.35"
	)
	book := `book:
  title: "Style Wiring"
  author: "Test"
  language: "en-US"
style:
  theme: "technical"
  font_family: "` + wantFont + `, serif"
  font_size: "` + wantFontSize + `"
  line_height: ` + wantLineHeight + `
chapters:
  - title: "Intro"
    file: "intro.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(book), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "intro.md"), []byte("# Intro\n\nbody text\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	buildFormat(t, dir, "site,html")

	outputs := map[string]string{
		"site": siteText(t, filepath.Join(dir, "_book")),
		"html": func() string {
			data, err := os.ReadFile(filepath.Join(dir, "Style-Wiring.html")) //nolint:gosec // G304: test-controlled path
			if err != nil {
				t.Fatalf("read standalone html: %v", err)
			}
			return string(data)
		}(),
	}

	for name, body := range outputs {
		t.Run(name, func(t *testing.T) {
			for setting, want := range map[string]string{
				"style.font_family": wantFont,
				"style.font_size":   wantFontSize,
				"style.line_height": wantLineHeight,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s is not honored in %s output: %q does not appear anywhere", setting, name, want)
				}
			}
		})
	}
}
