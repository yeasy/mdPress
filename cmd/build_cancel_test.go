package cmd

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/pdf"
)

// writeCancelTestBook writes a small book and returns its directory.
func writeCancelTestBook(t *testing.T, formats string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("book.yaml", `book:
  title: "Cancel Me"
  language: "en"
chapters:
  - title: "One"
    file: "one.md"
  - title: "Two"
    file: "two.md"
output:
  formats: [`+formats+`]
`)
	body := strings.Repeat("Filler paragraph for pagination.\n\n", 120)
	write("one.md", "# One\n\n"+body)
	write("two.md", "# Two\n\n"+body)
	return dir
}

// TestCanceledBuildFailsWithoutWritingOutput checks that a build whose context
// is already done stops instead of running to completion. Ctrl+C used to be
// swallowed entirely: the signal handler canceled the context, no build stage
// looked at it, and the build finished, printed its summary and exited 0.
func TestCanceledBuildFailsWithoutWritingOutput(t *testing.T) {
	dir := writeCancelTestBook(t, `"html"`)
	cfg, err := config.Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover book: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	quiet = true
	err = executeBuildForConfig(ctx, cfg, []string{"html"}, "", "", logger)
	if !errors.Is(err, errBuildCanceled) {
		t.Fatalf("executeBuildForConfig() error = %v, want it to wrap %v", err, errBuildCanceled)
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("read book dir: %v", readErr)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".html") {
			t.Errorf("canceled build wrote %s", entry.Name())
		}
	}
}

// TestRemoveCanceledBuildArtifacts checks that the files a canceled build
// managed to write are deleted — a leftover file looks current to make, to a
// deploy step and to the reader — while a directory output the generator does
// not own is reported rather than deleted.
func TestRemoveCanceledBuildArtifacts(t *testing.T) {
	dir := t.TempDir()
	baseOutput := filepath.Join(dir, "Book.pdf")
	siteDir := filepath.Join(dir, "_book")

	pdfPath := filepath.Join(dir, "Book.pdf")
	epubPath := filepath.Join(dir, "Book.epub")
	htmlPath := filepath.Join(dir, "Book.html")
	if err := os.MkdirAll(siteDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sitePage := filepath.Join(siteDir, "index.html")
	for _, path := range []string{pdfPath, epubPath, htmlPath, sitePage} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	removeCanceledBuildArtifacts(baseOutput, siteDir, []formatOutcome{
		{Format: "pdf"},
		{Format: "site"},
		{Format: "epub", Err: errors.New("epub failed")},
	}, logger)

	if _, err := os.Stat(pdfPath); !os.IsNotExist(err) {
		t.Error("the PDF a canceled build wrote was left behind")
	}
	if _, err := os.Stat(epubPath); err != nil {
		t.Error("a format that never produced output should not have its path removed")
	}
	if _, err := os.Stat(htmlPath); err != nil {
		t.Error("a format that was not part of this build must not be touched")
	}
	if _, err := os.Stat(sitePage); err != nil {
		t.Error("the site directory is not owned by the generator and must not be deleted")
	}
}

func TestBuildCanceled(t *testing.T) {
	if err := buildCanceled(context.Background()); err != nil {
		t.Errorf("buildCanceled() on a live context = %v, want nil", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := buildCanceled(canceledCtx); !errors.Is(err, errBuildCanceled) {
		t.Errorf("buildCanceled() on a canceled context = %v, want %v", err, errBuildCanceled)
	}

	expiredCtx, expireCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer expireCancel()
	err := buildCanceled(expiredCtx)
	if !errors.Is(err, errBuildCanceled) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("buildCanceled() on an expired context = %v, want both %v and %v",
			err, errBuildCanceled, context.DeadlineExceeded)
	}
}

// TestSIGINTStopsBuildAndRemovesOutput drives the real CLI: it interrupts a
// build once the output stage has started and checks that the process fails
// and leaves no PDF behind. Three Ctrl+C in a row used to do nothing at all —
// the build finished normally and exited 0, which is worse than having no
// signal handler installed.
func TestSIGINTStopsBuildAndRemovesOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI and renders with Chromium; skipped in -short mode")
	}
	// Windows has no SIGINT that one process can deliver to another:
	// os.Process.Signal(os.Interrupt) returns "not supported by windows", and a
	// real Ctrl+C there arrives via GenerateConsoleCtrlEvent to a shared console
	// group, an entirely different mechanism. The signal handling this test
	// exercises is Unix-only, so the test is too.
	if runtime.GOOS == "windows" {
		t.Skip("SIGINT cannot be delivered to another process on Windows")
	}
	if err := pdf.CheckChromiumAvailable(); err != nil {
		t.Skipf("Chromium is not available: %v", err)
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "mdpress-cancel-test")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	build.Dir = repoRoot
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("go build: %v\n%s", buildErr, out)
	}

	dir := writeCancelTestBook(t, `"pdf"`)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "build", "--format", "pdf")
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mdpress: %v", err)
	}

	// Interrupt once the output stage is under way, so the signal lands where
	// it used to be ignored outright rather than before the build got going.
	reader := bufio.NewReader(stdout)
	var progress strings.Builder
	for !strings.Contains(progress.String(), "[5/5]") {
		chunk, readErr := reader.ReadString(']')
		progress.WriteString(chunk)
		if readErr != nil {
			t.Fatalf("build finished before the output stage was reached:\n%s\n%s",
				progress.String(), stderr.String())
		}
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}
	_, _ = io.Copy(io.Discard, reader)

	waitErr := cmd.Wait()
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("interrupted build exited with %v, want a non-zero status\n%s", waitErr, stderr.String())
	}
	if !strings.Contains(stderr.String(), "build canceled") {
		t.Errorf("interrupted build did not report the cancellation:\n%s", stderr.String())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pdf") {
			t.Errorf("interrupted build left %s behind, which looks like a finished build", entry.Name())
		}
	}
}
