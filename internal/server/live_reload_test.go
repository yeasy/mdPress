package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHandleWebSocket_ReplaysUnresolvedBuildError covers the case where a
// rebuild fails and the reader then refreshes the page: the new connection
// must be told the build is broken, otherwise the browser shows the last good
// build with no indication that the latest edit never compiled.
func TestHandleWebSocket_ReplaysUnresolvedBuildError(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())

	// A rebuild fails while no browser is attached.
	srv.notifyBuildError("chapter-2.md:14: unexpected end of input")

	conn := dialTestWS(t, srv)
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected a replayed build-error frame, got read error: %v", err)
	}
	var msg struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("failed to decode replayed frame %q: %v", raw, err)
	}
	if msg.Type != msgTypeBuildErr {
		t.Errorf("expected a %q frame, got %q", msgTypeBuildErr, msg.Type)
	}
	if !strings.Contains(msg.Error, "unexpected end of input") {
		t.Errorf("expected the original build error text, got %q", msg.Error)
	}
}

// TestHandleWebSocket_NoReplayAfterSuccessfulBuild verifies the sticky error is
// cleared once a build succeeds, so a later refresh does not show a stale
// failure banner.
func TestHandleWebSocket_NoReplayAfterSuccessfulBuild(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())

	srv.notifyBuildError("broken")
	if srv.LastBuildError() == "" {
		t.Fatal("expected the failed build to be remembered")
	}
	// A later successful rebuild clears it (see debouncedRebuild).
	srv.setLastBuildErr("")

	conn := dialTestWS(t, srv)
	if err := conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	if _, raw, err := conn.ReadMessage(); err == nil {
		t.Errorf("expected no frame after the ack, got %q", raw)
	}
}

// TestDebouncedRebuild_ClearsBuildErrorOnSuccess exercises the real rebuild
// path: a failure is remembered, and the next successful build forgets it.
func TestDebouncedRebuild_ClearsBuildErrorOnSuccess(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), discardLogger())

	var buildErr error
	srv.BuildFunc = func() error { return buildErr }

	buildErr = errors.New("chapter-2.md:14: unexpected end of input")
	srv.debouncedRebuild(context.Background(), "intro.md", ".md")
	waitForCondition(t, 5*time.Second, func() bool {
		return srv.LastBuildError() != ""
	}, "failed build to be recorded")

	buildErr = nil
	srv.debouncedRebuild(context.Background(), "intro.md", ".md")
	waitForCondition(t, 5*time.Second, func() bool {
		return srv.LastBuildError() == ""
	}, "successful build to clear the recorded failure")
}

// TestReloadScript_ReloadsOnReconnect guards the browser-side handling of a
// server restart. The tab reconnects but used to keep rendering content built
// by the previous server process, staying stale until a manual refresh.
func TestReloadScript_ReloadsOnReconnect(t *testing.T) {
	script := injectedScript(t)

	if !strings.Contains(script, "var everConnected = false;") {
		t.Error("live reload script does not track whether the tab was ever connected")
	}
	// doReload must live outside ws.onmessage so ws.onopen can call it.
	onopen := between(script, "ws.onopen = function() {", "};")
	if !strings.Contains(onopen, "doReload()") {
		t.Errorf("ws.onopen does not reload on reconnect; body was:\n%s", onopen)
	}
	if !strings.Contains(onopen, "if (everConnected)") {
		t.Errorf("ws.onopen reloads unconditionally, which would loop on first connect; body was:\n%s", onopen)
	}
}

// injectedScript returns the live-reload script the server injects into pages.
func injectedScript(t *testing.T) string {
	t.Helper()
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "index.html", "<html><body>Home</body></html>")

	srv := NewServer("127.0.0.1", 8080, t.TempDir(), outputDir, discardLogger())
	handler := srv.injectLiveReload(http.FileServer(http.Dir(outputDir)))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Body.String()
}

// writeTestFile writes name inside dir, failing the test on error.
func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func between(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	rest := s[i+len(start):]
	j := strings.Index(rest, end)
	if j < 0 {
		return rest
	}
	return rest[:j]
}
