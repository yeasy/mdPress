package cmd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupServeLeftovers removes exactly the scratch directories the serve
// rebuild swap creates, and nothing else.
func TestCleanupServeLeftovers(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "_book")

	for _, dir := range []string{
		outputDir,
		outputDir + ".old",
		filepath.Join(root, "mdpress-serve-123456.tmp"),
		filepath.Join(root, "mdpress-serve-abcdef.tmp"),
		filepath.Join(root, "chapters"),
		filepath.Join(root, "mdpress-site-999.tmp"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	cleanupServeLeftovers(outputDir, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, gone := range []string{
		outputDir + ".old",
		filepath.Join(root, "mdpress-serve-123456.tmp"),
		filepath.Join(root, "mdpress-serve-abcdef.tmp"),
	} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s should have been cleaned up", filepath.Base(gone))
		}
	}
	for _, kept := range []string{
		outputDir,
		filepath.Join(root, "chapters"),
		filepath.Join(root, "mdpress-site-999.tmp"),
	} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("%s should not have been touched: %v", filepath.Base(kept), err)
		}
	}
}

// TestExecuteServe_ClearsLeftoversFromAKilledRun starts serve in a project that
// still holds the scratch dirs from a previous run that was killed mid-rebuild,
// and asserts they are gone. They used to accumulate in the project forever.
func TestExecuteServe_ClearsLeftoversFromAKilledRun(t *testing.T) {
	root := t.TempDir()
	book := "book:\n  title: \"T\"\nchapters:\n  - title: \"Intro\"\n    file: \"README.md\"\n"
	if err := os.WriteFile(filepath.Join(root, "book.yaml"), []byte(book), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Intro\n\nhello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(root, "_book")
	staleBackup := outDir + ".old"
	staleStaging := filepath.Join(root, "mdpress-serve-stale.tmp")
	for _, dir := range []string{staleBackup, staleStaging} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// A canceled context keeps this bounded: the initial build aborts and
	// serve returns instead of blocking. The sweep runs before that, so the
	// leftovers must be gone either way.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_ = executeServe(ctx, root, serveOptions{
		Host:        "127.0.0.1",
		Port:        0,
		OutputDir:   outDir,
		PortChanged: true,
	})

	for _, gone := range []string{staleBackup, staleStaging} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("serve left %s behind", filepath.Base(gone))
		}
	}
}
