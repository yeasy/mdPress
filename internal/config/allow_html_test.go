package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowRawHTMLDefaultsToTrue(t *testing.T) {
	if !DefaultConfig().AllowRawHTML() {
		t.Error("raw HTML should be allowed unless a project opts out")
	}
}

func TestAllowRawHTMLReadsConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# One\n"), 0o600); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}
	for _, tc := range []struct {
		name  string
		yaml  string
		allow bool
	}{
		{"unset", "", true},
		{"explicit true", "markdown:\n  allow_html: true\n", true},
		{"explicit false", "markdown:\n  allow_html: false\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, "book.yaml")
			body := "book:\n  title: \"T\"\nchapters:\n  - title: \"One\"\n    file: \"ch1.md\"\n" + tc.yaml
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write book.yaml: %v", err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if got := cfg.AllowRawHTML(); got != tc.allow {
				t.Errorf("AllowRawHTML() = %v, want %v", got, tc.allow)
			}
		})
	}
}

// markdown.allow_html has to be a recognized key, or setting it would draw an
// "unknown key" warning and do nothing.
func TestAllowHTMLIsARecognizedKey(t *testing.T) {
	data := []byte("markdown:\n  allow_html: false\n")
	if unknown := FindUnknownKeys(data); len(unknown) != 0 {
		t.Errorf("markdown.allow_html reported as unknown: %v", unknown)
	}
}
