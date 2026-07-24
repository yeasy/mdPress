package pdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestGenerateStopsWhenContextIsCanceled pins the behavior behind Ctrl+C.
//
// The generator used to build its chromedp allocator from context.Background(),
// so the caller had no handle on a render in progress. On a book that keeps
// Chrome busy for a minute, an interrupt was recorded by the CLI and then
// ignored: the build sat on a frozen "[5/5] Generating output (pdf)" until the
// whole book had been printed, and only then reported "build canceled". This
// test cancels mid-render and requires the call to come back promptly with the
// cancellation itself, not with a Chrome failure or an expired pdf_timeout.
func TestGenerateStopsWhenContextIsCanceled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in browser is a /bin/sh script")
	}
	dir := t.TempDir()
	// A browser that never speaks the DevTools protocol stands in for one that
	// is busy laying out a long document: either way the render does not finish
	// on its own, so what is measured is the cancellation and nothing else.
	fakeChrome := filepath.Join(dir, "fake-chrome")
	if err := os.WriteFile(fakeChrome, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MDPRESS_CHROME_PATH", fakeChrome)
	t.Setenv("CHROME_BIN", "")

	outputPath := filepath.Join(dir, "out.pdf")
	// A timeout far longer than the test's patience: if cancellation were still
	// ignored, the only way out would be this deadline, and the test would fail
	// on elapsed time rather than hanging forever.
	g := NewGenerator(WithTimeout(60 * time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := g.Generate(ctx, "<html><head></head><body>hello</body></html>", outputPath)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled: an interrupt must not be reported as a Chrome or timeout failure", err)
	}
	if elapsed > 15*time.Second {
		t.Errorf("Generate returned %s after the cancel; it should stop within a second, not wait out pdf_timeout", elapsed)
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Errorf("a canceled render left %s behind; a half-printed book on disk looks current to make and to the reader", outputPath)
	}
}

// TestGenerateFromFileStopsWhenContextIsCanceled covers the file:// entry point,
// which reaches Chrome through a different path and additionally has a CLI
// fallback. Canceling must not be mistaken for a chromedp failure worth
// retrying: the fallback would start a second browser the user already said
// they did not want, and make them wait out that render too.
func TestGenerateFromFileStopsWhenContextIsCanceled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in browser is a /bin/sh script")
	}
	dir := t.TempDir()
	fakeChrome := filepath.Join(dir, "fake-chrome")
	if err := os.WriteFile(fakeChrome, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MDPRESS_CHROME_PATH", fakeChrome)
	t.Setenv("CHROME_BIN", "")

	htmlPath := filepath.Join(dir, "in.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body>hello</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(dir, "out.pdf")
	g := NewGenerator(WithTimeout(60 * time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := g.GenerateFromFile(ctx, htmlPath, outputPath)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if elapsed > 15*time.Second {
		t.Errorf("GenerateFromFile returned %s after the cancel, and the CLI fallback must not run either", elapsed)
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Errorf("a canceled render left %s behind", outputPath)
	}
}

// TestKillRunningBrowsersIsSafeWhenIdle guards the hard-exit path: it runs from
// a signal handler, where a panic or a nil dereference would be the last thing
// the user sees.
func TestKillRunningBrowsersIsSafeWhenIdle(t *testing.T) {
	KillRunningBrowsers()
	KillRunningBrowsers()
}

// TestTrackBrowserIgnoresContextsWithoutABrowser checks the same guard from the
// other side: a context that never allocated a browser has no process to track.
func TestTrackBrowserIgnoresContextsWithoutABrowser(t *testing.T) {
	if release := trackBrowser(context.Background()); release != nil {
		t.Error("trackBrowser returned a release func for a context with no browser")
	}
}
