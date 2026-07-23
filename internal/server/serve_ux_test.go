package server

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// TestListenFrom_WarnsOnPortFallback covers the case where the default port is
// taken: the server quietly moved to another port, leaving the user reloading
// a dead tab on the port they expected.
func TestListenFrom_WarnsOnPortFallback(t *testing.T) {
	// Occupy a port so the first attempt must fall through.
	occupied, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy a port: %v", err)
	}
	defer occupied.Close() //nolint:errcheck
	startPort := occupied.Addr().(*net.TCPAddr).Port

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
	srv := NewServer("127.0.0.1", startPort, t.TempDir(), t.TempDir(), logger)

	ln, err := srv.ListenFrom(startPort)
	if err != nil {
		t.Fatalf("ListenFrom failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	if srv.Port == startPort {
		t.Fatalf("expected a different port than the occupied %d", startPort)
	}
	out := logs.String()
	if !strings.Contains(out, "already in use") {
		t.Errorf("expected a warning that the port was in use, got %q", out)
	}
	if !strings.Contains(out, strconv.Itoa(srv.Port)) {
		t.Errorf("expected the warning to name the port actually used (%d), got %q", srv.Port, out)
	}
}

// TestListenFrom_QuietWhenPortIsFree makes sure the fallback notice is not
// printed on the normal path.
func TestListenFrom_QuietWhenPortIsFree(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
	srv := NewServer("127.0.0.1", 0, t.TempDir(), t.TempDir(), logger)

	// Port 0 always binds, so startPort and the resulting port match.
	ln, err := srv.ListenFrom(0)
	if err != nil {
		t.Fatalf("ListenFrom failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	if strings.Contains(logs.String(), "already in use") {
		t.Errorf("unexpected port fallback warning: %q", logs.String())
	}
}

// TestNetworkURLs covers `mdpress serve --host 0.0.0.0`, which used to print
// only http://localhost:PORT — useless for the phone or laptop the wildcard
// binding was chosen for.
func TestNetworkURLs(t *testing.T) {
	loopback := NewServer("127.0.0.1", 9000, t.TempDir(), t.TempDir(), discardLogger())
	if urls := loopback.networkURLs(); len(urls) != 0 {
		t.Errorf("a loopback binding is not reachable from the network, got %v", urls)
	}

	wildcard := NewServer("0.0.0.0", 9000, t.TempDir(), t.TempDir(), discardLogger())
	for _, raw := range wildcard.networkURLs() {
		u, err := url.Parse(raw)
		if err != nil {
			t.Errorf("networkURLs returned an unparseable URL %q: %v", raw, err)
			continue
		}
		if u.Port() != "9000" {
			t.Errorf("expected the listening port in %q", raw)
		}
		ip := net.ParseIP(u.Hostname())
		if ip == nil {
			t.Errorf("expected a literal IP in %q", raw)
			continue
		}
		if ip.IsLoopback() {
			t.Errorf("loopback address %q is not useful as a network URL", raw)
		}
	}
}

// TestIsPageRequest guards the split between page URLs (which get the site's
// 404 page) and static assets (which keep the file server's response). Chapter
// slugs commonly contain dots, so a naive filepath.Ext check misclassifies them.
func TestIsPageRequest(t *testing.T) {
	pages := []string{
		"/chapter-3",
		"/02_reasoning-2.4_reflexion",
		"/missing.html",
		"/missing.htm",
		"/",
	}
	for _, p := range pages {
		if !isPageRequest(p) {
			t.Errorf("%q should be treated as a page request", p)
		}
	}
	assets := []string{"/style.css", "/app.js", "/logo.PNG", "/font.woff2", "/data.json"}
	for _, p := range assets {
		if isPageRequest(p) {
			t.Errorf("%q should be treated as a static asset", p)
		}
	}
}

// TestServeNotFound_FallsBackWithout404Page checks the degraded case: a site
// that has no 404.html still gets a real 404 status rather than a redirect.
func TestServeNotFound_FallsBackWithout404Page(t *testing.T) {
	outputDir := t.TempDir()
	srv := NewServer("127.0.0.1", 8080, t.TempDir(), outputDir, discardLogger())
	handler := srv.injectLiveReload(http.FileServer(http.Dir(outputDir)))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("expected no redirect, got Location %q", loc)
	}
}

// TestServeNotFound_InjectsLiveReload verifies the 404 page keeps the live
// reload connection, so the tab recovers by itself once the page is created.
func TestServeNotFound_InjectsLiveReload(t *testing.T) {
	outputDir := t.TempDir()
	writeTestFile(t, outputDir, "404.html", "<html><head><title>Page not found - Demo</title></head><body>404</body></html>")
	srv := NewServer("127.0.0.1", 8080, t.TempDir(), outputDir, discardLogger())
	handler := srv.injectLiveReload(http.FileServer(http.Dir(outputDir)))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "__mdpress_ws") {
		t.Error("expected the live reload script in the 404 page")
	}
}
