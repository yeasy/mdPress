package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// writeMiniProject writes a one-chapter project and returns its directory.
func writeMiniProject(t *testing.T, dir, bookYAML string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ch1.md"), []byte("# Chapter One\n\nBody.\n"), 0o600); err != nil {
		t.Fatalf("write ch1.md: %v", err)
	}
	if bookYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0o600); err != nil {
			t.Fatalf("write book.yaml: %v", err)
		}
	}
	return dir
}

// A title with spaces must not leak them into the artifact name: "My Book.pdf"
// needs quoting in every shell and becomes a %20 URL for site builds.
func TestDeriveOutputFilename_TitleSpacesBecomeHyphens(t *testing.T) {
	dir := writeMiniProject(t, filepath.Join(t.TempDir(), "proj"), `book:
  title: "My Book"
chapters:
  - title: "Chapter One"
    file: "ch1.md"
`)
	cfg, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := deriveOutputFilename(cfg); got != "My-Book.pdf" {
		t.Errorf("deriveOutputFilename() = %q, want %q", got, "My-Book.pdf")
	}
}

// output.filename: "output.pdf" used to be indistinguishable from "unset" and
// was silently ignored. It is an explicit choice and must be honored.
func TestDeriveOutputFilename_ExplicitOutputPDFHonored(t *testing.T) {
	dir := writeMiniProject(t, filepath.Join(t.TempDir(), "proj"), `book:
  title: "My Book"
chapters:
  - title: "Chapter One"
    file: "ch1.md"
output:
  filename: "output.pdf"
`)
	cfg, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := deriveOutputFilename(cfg); got != "output.pdf" {
		t.Errorf("deriveOutputFilename() = %q, want %q", got, "output.pdf")
	}
}

// A zero-config directory names its artifact after the directory, matching
// what `mdpress init` would have written into book.yaml.
func TestDeriveOutputFilename_ZeroConfigUsesDirName(t *testing.T) {
	dir := writeMiniProject(t, filepath.Join(t.TempDir(), "field-guide"), "")
	cfg, err := config.Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if got := deriveOutputFilename(cfg); got != "Field-guide.pdf" {
		t.Errorf("deriveOutputFilename() = %q, want %q", got, "Field-guide.pdf")
	}
}

// `mdpress init` and a zero-config build of the same directory must agree on
// the artifact name; they used to differ (underscores vs. spaces).
func TestInitAndZeroConfigAgreeOnOutputName(t *testing.T) {
	defer suppressOutput(t)()
	dir := writeMiniProject(t, filepath.Join(t.TempDir(), "my project"), "")

	zeroCfg, err := config.Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	zeroName := deriveOutputFilename(zeroCfg)

	if err := executeInit(context.Background(), dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	initCfg, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("load generated book.yaml: %v", err)
	}
	if got := deriveOutputFilename(initCfg); got != zeroName {
		t.Errorf("init produces %q but zero-config produces %q", got, zeroName)
	}
}
