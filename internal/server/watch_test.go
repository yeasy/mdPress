// watch_test.go tests the file watcher / rebuild interplay.
// Covers: debouncedRebuild coalescing and CSS escalation, the ignore rules
// applied to fsnotify events, and the regression where the serve atomic-swap
// artifacts (_book.old, mdpress-serve-*.tmp) re-triggered the watcher and
// caused an infinite rebuild loop.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// discardLogger returns a logger that swallows all output, keeping the
// watcher tests quiet.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// dialTestWS connects a WebSocket client to the server's handler and consumes
// the initial "connected" acknowledgment.
func dialTestWS(t *testing.T, srv *Server) *websocket.Conn {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	t.Cleanup(ts.Close)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	t.Cleanup(func() { conn.Close() }) //nolint:errcheck

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("failed to read connected ack: %v", err)
	}
	return conn
}

// readTerminalWSMessage reads messages until a terminal one (reload,
// css-update, or build-error) arrives, skipping build-start notifications.
func readTerminalWSMessage(t *testing.T, conn *websocket.Conn) string {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatalf("failed to set read deadline: %v", err)
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("failed to read WebSocket message: %v", err)
		}
		s := string(msg)
		if strings.Contains(s, `"type":"`+msgTypeBuildStart+`"`) {
			continue
		}
		return s
	}
}

// editUntilRebuild repeatedly edits mdPath (spaced beyond the debounce
// window) until the build counter moves past prev. This absorbs the startup
// race where an edit can land before the watcher has registered the
// directory tree.
func editUntilRebuild(t *testing.T, mdPath string, builds *atomic.Int32, prev int32) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for builds.Load() == prev && time.Now().Before(deadline) {
		content := fmt.Sprintf("# Chapter\n\nedit %d\n", time.Now().UnixNano())
		if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to edit source file: %v", err)
		}
		time.Sleep(2 * debounceInterval)
	}
	if builds.Load() == prev {
		t.Fatal("watcher never picked up the source edit")
	}
}

func TestDebouncedRebuild_CoalescesRapidChanges(t *testing.T) {
	t.Parallel()

	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		srv.debouncedRebuild(ctx, fmt.Sprintf("ch%02d.md", i), ".md")
		time.Sleep(10 * time.Millisecond)
	}

	waitForCondition(t, 5*time.Second, func() bool { return builds.Load() >= 1 }, "debounced rebuild to fire")

	// No further rebuilds may fire once the burst has been coalesced.
	time.Sleep(2 * debounceInterval)
	if got := builds.Load(); got != 1 {
		t.Fatalf("5 rapid changes should coalesce into exactly 1 rebuild, got %d", got)
	}
}

func TestDebouncedRebuild_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv.debouncedRebuild(ctx, "ch01.md", ".md")

	time.Sleep(debounceInterval + 200*time.Millisecond)
	if got := builds.Load(); got != 0 {
		t.Fatalf("no rebuild should run after context cancellation, got %d", got)
	}
}

func TestDebouncedRebuild_CSSOnlyChangeSendsCSSUpdate(t *testing.T) {
	t.Parallel()

	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}
	conn := dialTestWS(t, srv)

	srv.debouncedRebuild(context.Background(), "custom.css", ".css")

	msg := readTerminalWSMessage(t, conn)
	if !strings.Contains(msg, `"type":"css-update"`) {
		t.Fatalf("CSS-only change should send css-update, got: %q", msg)
	}
	if got := builds.Load(); got != 1 {
		t.Errorf("expected exactly 1 rebuild, got %d", got)
	}
}

func TestDebouncedRebuild_NonCSSChangeEscalatesToReload(t *testing.T) {
	t.Parallel()

	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())
	srv.BuildFunc = func() error { return nil }
	conn := dialTestWS(t, srv)

	ctx := context.Background()
	srv.debouncedRebuild(ctx, "custom.css", ".css")
	srv.debouncedRebuild(ctx, "ch01.md", ".md")
	// A trailing CSS change must not de-escalate the pending full reload.
	srv.debouncedRebuild(ctx, "custom.css", ".css")

	msg := readTerminalWSMessage(t, conn)
	if !strings.Contains(msg, `"type":"reload"`) {
		t.Fatalf("mixed css+md changes should trigger a full reload, got: %q", msg)
	}
}

func TestDebouncedRebuild_BuildErrorSendsBuildError(t *testing.T) {
	t.Parallel()

	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())
	srv.BuildFunc = func() error { return errors.New("boom: chapter failed") }
	conn := dialTestWS(t, srv)

	srv.debouncedRebuild(context.Background(), "ch01.md", ".md")

	msg := readTerminalWSMessage(t, conn)
	if !strings.Contains(msg, `"type":"build-error"`) {
		t.Fatalf("failed build should send build-error, got: %q", msg)
	}
	if !strings.Contains(msg, "boom: chapter failed") {
		t.Errorf("build-error message should carry the error text, got: %q", msg)
	}
}

// TestWatchFsnotify_AtomicSwapDoesNotRetrigger is the regression test for the
// infinite rebuild loop: cmd/serve.go's BuildFunc stages the site in a
// mdpress-serve-*.tmp directory next to the output dir, renames the previous
// output to _book.old, and swaps the new build in. Those artifacts live
// inside the watched root, so the watcher must ignore them or every rebuild
// schedules the next one forever.
func TestWatchFsnotify_AtomicSwapDoesNotRetrigger(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	outputDir := filepath.Join(watchDir, "_book")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("v0"), 0o644); err != nil {
		t.Fatalf("failed to seed output dir: %v", err)
	}
	mdPath := filepath.Join(watchDir, "ch01.md")
	if err := os.WriteFile(mdPath, []byte("# Chapter\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	srv := NewServer("127.0.0.1", 0, watchDir, outputDir, discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		// Replicate the cmd/serve.go atomic swap inside the watched root.
		tempOutput, err := os.MkdirTemp(filepath.Dir(outputDir), "mdpress-serve-*.tmp")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tempOutput, "index.html"), []byte(time.Now().String()), 0o644); err != nil {
			return err
		}
		backupDir := outputDir + ".old"
		if err := os.RemoveAll(backupDir); err != nil {
			return err
		}
		if err := os.Rename(outputDir, backupDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.Rename(tempOutput, outputDir); err != nil {
			return err
		}
		return os.RemoveAll(backupDir)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.watchFilesWithFsnotify(ctx)

	editUntilRebuild(t, mdPath, &builds, 0)

	// The swap artifacts created by the rebuild (_book rename, _book.old,
	// mdpress-serve-*.tmp) must not schedule further rebuilds: the count has
	// to settle once the edit has been processed.
	settled := builds.Load()
	time.Sleep(6 * debounceInterval)
	if got := builds.Load(); got != settled {
		t.Fatalf("rebuild count did not settle after one edit: %d then %d (watcher re-triggered by swap artifacts)", settled, got)
	}
}

// TestWatchFsnotify_IgnoresGeneratedOutputDirs creates every generated
// output/swap directory mdpress produces inside the watched root and asserts
// that none of them triggers a rebuild, while a real source edit still does.
func TestWatchFsnotify_IgnoresGeneratedOutputDirs(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	outputDir := filepath.Join(watchDir, "_book")
	mdPath := filepath.Join(watchDir, "ch01.md")
	if err := os.WriteFile(mdPath, []byte("# Chapter\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	srv := NewServer("127.0.0.1", 0, watchDir, outputDir, discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.watchFilesWithFsnotify(ctx)

	// Probe edit proves the watcher is active before asserting negatives.
	editUntilRebuild(t, mdPath, &builds, 0)
	probeBuilds := builds.Load()

	// Create the artifacts serve/build produce inside the watched root.
	for _, dir := range []string{"_book", "_book.old", "_output", "guide_site"} {
		if err := os.MkdirAll(filepath.Join(watchDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(watchDir, dir, "stale.md"), []byte("# stale\n"), 0o644); err != nil {
			t.Fatalf("failed to write file in %s: %v", dir, err)
		}
	}
	if _, err := os.MkdirTemp(watchDir, "mdpress-serve-*.tmp"); err != nil {
		t.Fatalf("failed to create serve temp dir: %v", err)
	}

	time.Sleep(6 * debounceInterval)
	if got := builds.Load(); got != probeBuilds {
		t.Fatalf("creating output/swap directories triggered %d extra rebuild(s)", got-probeBuilds)
	}

	// Sanity: a real source edit still triggers a rebuild.
	editUntilRebuild(t, mdPath, &builds, probeBuilds)
}

// TestWatchFsnotify_NewDirectoryIsWatched verifies the positive path of the
// directory-Create handler: a regular new directory triggers a rebuild and
// files created inside it afterwards are watched too.
func TestWatchFsnotify_NewDirectoryIsWatched(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	mdPath := filepath.Join(watchDir, "ch01.md")
	if err := os.WriteFile(mdPath, []byte("# Chapter\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	srv := NewServer("127.0.0.1", 0, watchDir, filepath.Join(watchDir, "_book"), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.watchFilesWithFsnotify(ctx)

	editUntilRebuild(t, mdPath, &builds, 0)

	// Creating a regular directory triggers a rebuild (it may contain files).
	prev := builds.Load()
	newDir := filepath.Join(watchDir, "chapters")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("failed to create new dir: %v", err)
	}
	waitForCondition(t, 5*time.Second, func() bool { return builds.Load() > prev }, "rebuild after new directory")

	// Let the directory-creation burst settle, then verify a file created
	// inside the new directory is watched as well.
	time.Sleep(2 * debounceInterval)
	editUntilRebuild(t, filepath.Join(newDir, "ch02.md"), &builds, builds.Load())
}

func TestWatchFilesPolling_TriggersRebuildOnChange(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	mdPath := filepath.Join(watchDir, "ch01.md")
	if err := os.WriteFile(mdPath, []byte("# Chapter\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	srv := NewServer("127.0.0.1", 0, watchDir, filepath.Join(watchDir, "_book"), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.watchFilesPolling(ctx)

	// Let the initial scan record baseline modification times, then edit.
	time.Sleep(300 * time.Millisecond)
	deadline := time.Now().Add(10 * time.Second)
	for builds.Load() == 0 && time.Now().Before(deadline) {
		content := fmt.Sprintf("# Chapter\n\nedit %d\n", time.Now().UnixNano())
		if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to edit source file: %v", err)
		}
		time.Sleep(fileWatcherInterval + debounceInterval)
	}
	if builds.Load() == 0 {
		t.Fatal("polling watcher never triggered a rebuild")
	}
}

func TestScanModTimes_SkipsGeneratedOutputDirs(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	for _, dir := range []string{"_book", "_book.old", "_output", "guide_site", "mdpress-serve-42.tmp"} {
		if err := os.MkdirAll(filepath.Join(watchDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(watchDir, dir, "inside.md"), []byte("# x\n"), 0o644); err != nil {
			t.Fatalf("failed to write file in %s: %v", dir, err)
		}
	}
	realMD := filepath.Join(watchDir, "real.md")
	if err := os.WriteFile(realMD, []byte("# real\n"), 0o644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	srv := NewServer("127.0.0.1", 0, watchDir, filepath.Join(watchDir, "_book"), discardLogger())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	if _, ok := modTimes[realMD]; !ok {
		t.Errorf("scanModTimes should record %s", realMD)
	}
	if len(modTimes) != 1 {
		t.Errorf("scanModTimes should record only the real source file, got %d entries: %v", len(modTimes), modTimes)
	}
}

// TestWatchFsnotify_DeletionTriggersRebuild covers chapter removal: the
// fsnotify event mask used to accept only Write|Create, so deleting or
// renaming a chapter produced no rebuild and the deleted page kept being
// served until some unrelated edit happened to trigger one.
func TestWatchFsnotify_DeletionTriggersRebuild(t *testing.T) {
	t.Parallel()

	watchDir := t.TempDir()
	keepPath := filepath.Join(watchDir, "ch01.md")
	dropPath := filepath.Join(watchDir, "ch02.md")
	for _, p := range []string{keepPath, dropPath} {
		if err := os.WriteFile(p, []byte("# Chapter\n"), 0o644); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}
	}

	srv := NewServer("127.0.0.1", 0, watchDir, filepath.Join(watchDir, "_book"), discardLogger())
	var builds atomic.Int32
	srv.BuildFunc = func() error {
		builds.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.watchFilesWithFsnotify(ctx)

	editUntilRebuild(t, keepPath, &builds, 0)
	time.Sleep(2 * debounceInterval)

	prev := builds.Load()
	if err := os.Remove(dropPath); err != nil {
		t.Fatalf("failed to remove chapter: %v", err)
	}
	waitForCondition(t, 5*time.Second, func() bool { return builds.Load() > prev }, "rebuild after chapter deletion")
}
