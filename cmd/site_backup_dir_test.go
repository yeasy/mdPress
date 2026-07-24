package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// discardLogger returns a logger that swallows output, for tests that only
// care about filesystem effects.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestSwapSiteDir_KeepsUnrelatedDotOldSibling pins the fix for the swap
// scratch name. swapSiteDir used to stage the previous site at
// "<outputDir>.old" and RemoveAll it first, so `build --format site -o
// ~/Sites/book` silently destroyed the user's own ~/Sites/book.old backup.
func TestSwapSiteDir_KeepsUnrelatedDotOldSibling(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "site")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	// A hand-made backup of the previous release, sitting where people put it.
	userBackup := target + ".old"
	if err := os.MkdirAll(userBackup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userBackup, "index.html"), []byte("last release"), 0o644); err != nil {
		t.Fatal(err)
	}

	staging, err := newSiteStaging(root, "mdpress-site-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging.Site, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := swapSiteDir(staging, target, discardLogger()); err != nil {
		t.Fatalf("swapSiteDir() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(userBackup, "index.html"))
	if err != nil {
		t.Fatalf("the user's %s backup was destroyed by the swap: %v", filepath.Base(userBackup), err)
	}
	if string(content) != "last release" {
		t.Errorf("the user's backup was overwritten: got %q", content)
	}

	// The swap itself must still have happened, and must not leave scratch behind.
	if got, err := os.ReadFile(filepath.Join(target, "index.html")); err != nil || string(got) != "new" {
		t.Errorf("fresh build not swapped in: %q, %v", got, err)
	}
	assertNoSwapScratchLeftBehind(t, root)
}

// assertNoSwapScratchLeftBehind fails when the swap leaves any mdpress-owned
// scratch directory in parent.
func assertNoSwapScratchLeftBehind(t *testing.T, parent string) {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "mdpress-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("scratch directory %s left behind after the swap", e.Name())
		}
	}
}

// TestCleanupServeLeftovers_LeavesDotOldAlone pins that the leftover sweep no
// longer claims the sibling "<outputDir>.old" name. It used to RemoveAll it
// unconditionally, so `serve --output ~/public` deleted ~/public.old.
func TestCleanupServeLeftovers_LeavesDotOldAlone(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "_book")
	userBackup := outputDir + ".old"
	if err := os.MkdirAll(userBackup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userBackup, "index.html"), []byte("previous release"), 0o644); err != nil {
		t.Fatal(err)
	}

	cleanupServeLeftovers(outputDir, discardLogger())

	if _, err := os.Stat(filepath.Join(userBackup, "index.html")); err != nil {
		t.Fatalf("cleanupServeLeftovers deleted the user's %s: %v", filepath.Base(userBackup), err)
	}
}

// TestExecuteServe_RefusedRunTouchesNothing pins the ordering fix: when serve
// refuses the output directory it must not have deleted anything first. The
// sweep used to run before ensureReplaceableSiteDir, so the command destroyed
// the sibling backup and *then* told the user it had done nothing.
func TestExecuteServe_RefusedRunTouchesNothing(t *testing.T) {
	root := t.TempDir()
	book := "book:\n  title: \"T\"\nchapters:\n  - title: \"Intro\"\n    file: \"README.md\"\n"
	if err := os.WriteFile(filepath.Join(root, "book.yaml"), []byte(book), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Intro\n\nhello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// An output directory serve must refuse: non-empty, not a generated site.
	outDir := filepath.Join(root, "public")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "README.txt"), []byte("user file"), 0o644); err != nil {
		t.Fatal(err)
	}
	userBackup := outDir + ".old"
	if err := os.MkdirAll(userBackup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userBackup, "index.html"), []byte("previous release"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := executeServe(context.Background(), root, serveOptions{
		Host:        "127.0.0.1",
		Port:        0,
		OutputDir:   outDir,
		PortChanged: true,
	})
	if err == nil {
		t.Fatal("expected serve to refuse the output directory")
	}

	if _, statErr := os.Stat(filepath.Join(userBackup, "index.html")); statErr != nil {
		t.Errorf("a refused serve run deleted %s: %v", filepath.Base(userBackup), statErr)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "README.txt")); statErr != nil {
		t.Errorf("a refused serve run touched the output directory: %v", statErr)
	}
}
