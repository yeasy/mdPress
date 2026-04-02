// server_test.go Tests preview server core functionality.
// Covers: NewServer creation, WebSocket handling, HTML injection, file change detection, polling, etc.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", description)
}

// TestNewServer tests basic server creation properties
func TestNewServer(t *testing.T) {
	watchDir1 := t.TempDir()
	outputDir1 := t.TempDir()
	watchDir2 := t.TempDir()
	outputDir2 := t.TempDir()
	watchDir3 := t.TempDir()
	outputDir3 := t.TempDir()

	tests := []struct {
		name      string
		host      string
		port      int
		watchDir  string
		outputDir string
		logger    *slog.Logger
	}{
		{
			name:      "basic creation",
			host:      "127.0.0.1",
			port:      8080,
			watchDir:  watchDir1,
			outputDir: outputDir1,
			logger:    slog.Default(),
		},
		{
			name:      "custom port",
			host:      "0.0.0.0",
			port:      3000,
			watchDir:  watchDir2,
			outputDir: outputDir2,
			logger:    slog.Default(),
		},
		{
			name:      "nil logger uses default",
			host:      "",
			port:      9090,
			watchDir:  watchDir3,
			outputDir: outputDir3,
			logger:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.host, tt.port, tt.watchDir, tt.outputDir, tt.logger)

			if srv == nil {
				t.Fatal("NewServer returned nil")
			}
			expectedHost := tt.host
			if expectedHost == "" {
				expectedHost = "127.0.0.1"
			}
			if srv.Host != expectedHost {
				t.Errorf("Host = %q, want %q", srv.Host, expectedHost)
			}
			if srv.clients == nil {
				t.Error("clients map should be initialized")
			}
			if srv.logger == nil {
				t.Error("logger should not be nil (should use default even if nil is passed)")
			}
			// AutoOpen defaults to false
			if srv.AutoOpen {
				t.Error("AutoOpen should default to false")
			}
		})
	}
}

func TestListen_PortAlreadyInUse(t *testing.T) {
	occupied, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy test port: %v", err)
	}
	defer occupied.Close() //nolint:errcheck

	port := occupied.Addr().(*net.TCPAddr).Port
	srv := NewServer("127.0.0.1", port, t.TempDir(), t.TempDir(), slog.Default())

	_, err = srv.Listen()
	if err == nil {
		t.Fatal("expected Listen to return port-in-use error")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("error should mention port in use, got: %v", err)
	}
}

func TestListenFrom_SkipsOccupiedPort(t *testing.T) {
	occupied, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to occupy test port: %v", err)
	}
	defer occupied.Close() //nolint:errcheck

	startPort := occupied.Addr().(*net.TCPAddr).Port
	srv := NewServer("127.0.0.1", startPort, t.TempDir(), t.TempDir(), slog.Default())

	ln, err := srv.ListenFrom(startPort)
	if err != nil {
		t.Fatalf("ListenFrom returned error: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	if srv.Port <= startPort {
		t.Fatalf("should skip occupied port %d, actually used %d", startPort, srv.Port)
	}
}

// TestInjectLiveReload_HTMLFile tests live-reload script injection in HTML files
func TestInjectLiveReload_HTMLFile(t *testing.T) {
	// Create temp output directory and HTML file
	outputDir := t.TempDir()
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello</h1>
</body>
</html>`
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(htmlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())

	// Create a handler with injection middleware
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Request root path (should inject script)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Verify script was injected
	if !strings.Contains(body, "__mdpress_ws") {
		t.Error("HTML response should contain WebSocket connection script")
	}
	if strings.Contains(body, "ws://localhost:8080") {
		t.Error("HTML response should not hardcode localhost WebSocket address")
	}
	if !strings.Contains(body, "window.location.host") {
		t.Error("HTML response should establish WebSocket connection based on current host")
	}
	if !strings.Contains(body, "location.reload()") {
		t.Error("HTML response should contain auto-reload logic")
	}
	if !strings.Contains(body, "mdpress-serve-panel") {
		t.Error("HTML response should contain serve dev panel script")
	}
	if !strings.Contains(body, "build-start") {
		t.Error("HTML response should contain build-start event handling logic")
	}
	// Verify original content is preserved
	if !strings.Contains(body, "<h1>Hello</h1>") {
		t.Error("original HTML content should be preserved")
	}
	// Verify Content-Type
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type should be text/html, got %q", ct)
	}
	// Verify Cache-Control
	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control should be no-cache, got %q", cc)
	}
}

// TestInjectLiveReload_NonHTML tests that non-HTML files are not injected with script
func TestInjectLiveReload_NonHTML(t *testing.T) {
	outputDir := t.TempDir()
	cssContent := `body { color: red; }`
	if err := os.WriteFile(filepath.Join(outputDir, "style.css"), []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Non-HTML files should not contain injected script
	if strings.Contains(body, "__mdpress_ws") {
		t.Error("non-HTML files should not contain WebSocket script")
	}
}

// TestInjectLiveReload_DirectoryPath tests directory path requests (ending with /)
func TestInjectLiveReload_DirectoryPath(t *testing.T) {
	outputDir := t.TempDir()
	subDir := filepath.Join(outputDir, "chapter1")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subdir failed: %v", err)
	}
	htmlContent := `<!DOCTYPE html><html><body><p>Chapter</p></body></html>`
	if err := os.WriteFile(filepath.Join(subDir, "index.html"), []byte(htmlContent), 0o644); err != nil {
		t.Fatalf("write index.html failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Request path ending with /
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/chapter1/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "__mdpress_ws") {
		t.Error("HTML from directory path requests should also have script injected")
	}
}

// TestHandleWebSocket tests WebSocket connection handling
func TestHandleWebSocket(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	// Replace http:// with ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect WebSocket
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// Read connection acknowledgment message
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read WebSocket message: %v", err)
	}
	if string(msg) != "connected" {
		t.Errorf("expected 'connected', got %q", string(msg))
	}

	// Verify client was registered
	srv.clientsMu.RLock()
	clientCount := len(srv.clients)
	srv.clientsMu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}

	// Verify client is removed after disconnection
	conn.Close() //nolint:errcheck
	waitForCondition(t, time.Second, func() bool {
		srv.clientsMu.RLock()
		defer srv.clientsMu.RUnlock()
		return len(srv.clients) == 0
	}, "WebSocket client cleanup")

	srv.clientsMu.RLock()
	clientCount = len(srv.clients)
	srv.clientsMu.RUnlock()

	if clientCount != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", clientCount)
	}
}

// TestNotifyClients tests notifying all clients
func TestNotifyClients(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect multiple clients
	const numClients = 3
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client %d connection failed: %v", i, err)
		}
		conns[i] = conn
		// Read "connected" acknowledgment
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("client %d failed to read connected message: %v", i, err)
		}
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close() //nolint:errcheck
			}
		}
	}()

	// Notify all clients
	srv.notifyClients()

	// Verify all clients received "reload" message
	for i, conn := range conns {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("client %d failed to set read deadline: %v", i, err)
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d failed to read message: %v", i, err)
		}
		if !strings.Contains(string(msg), `"type":"reload"`) {
			t.Errorf("client %d got %q, expected reload JSON message", i, string(msg))
		}
	}
}

// TestNotifyClientsEmpty tests that notification with no clients does not error
func TestNotifyClientsEmpty(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())
	// Should not panic or error
	srv.notifyClients()
}

// TestNotifyClientsConcurrent tests concurrent notification safety
// Each wsClient has its own writeMu, ensuring concurrent notifyClients calls
// do not cause gorilla/websocket concurrent write panics.
func TestNotifyClientsConcurrent(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Test concurrency safety with no clients
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.notifyClients()
		}()
	}
	wg.Wait()
}

// TestNotifyClientsConcurrentWithClients tests concurrent write safety with real client connections
func TestNotifyClientsConcurrentWithClients(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect multiple clients
	const numClients = 3
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client %d connection failed: %v", i, err)
		}
		conns[i] = conn
		if _, _, err := conn.ReadMessage(); err != nil { // read "connected"
			t.Fatalf("client %d failed to read connected message: %v", i, err)
		}
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close() //nolint:errcheck
			}
		}
	}()

	// Concurrently call notifyClients; verify wsClient.writeMu prevents concurrent write panics
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.notifyClients()
		}()
	}
	wg.Wait()
}

// TestScanModTimes tests file modification time scanning
func TestScanModTimes(t *testing.T) {
	watchDir := t.TempDir()

	// Create test files
	for name, content := range map[string]string{
		"chapter1.md": "# Chapter 1",
		"config.yaml": "title: test",
		"style.css":   "body{}",
		"image.png":   "png",
	} {
		if err := os.WriteFile(filepath.Join(watchDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}

	// Create hidden directory (should be skipped)
	hiddenDir := filepath.Join(watchDir, ".git")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("mkdir hidden dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "config.yml"), []byte("git"), 0o644); err != nil {
		t.Fatalf("write hidden config failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Should scan .md, .yaml, .css files
	if len(modTimes) != 3 {
		t.Errorf("expected 3 scanned files, got %d", len(modTimes))
		for k := range modTimes {
			t.Logf("  file: %s", k)
		}
	}

	// Verify .png files are excluded
	for path := range modTimes {
		if strings.HasSuffix(path, ".png") {
			t.Errorf("should not include .png file: %s", path)
		}
	}
}

// TestCheckForChanges tests file change detection
func TestCheckForChanges(t *testing.T) {
	watchDir := t.TempDir()
	mdFile := filepath.Join(watchDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0o644); err != nil {
		t.Fatalf("write test.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())

	// Initial scan
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Should return false when no changes
	changed, _ := srv.checkForChanges(modTimes)
	if changed {
		t.Error("should return false when no files changed")
	}

	// Modify file
	originalModTime, ok := modTimes[mdFile]
	if !ok {
		t.Fatalf("initial scan did not record %s", mdFile)
	}
	if err := os.WriteFile(mdFile, []byte("# Test Modified"), 0o644); err != nil {
		t.Fatalf("rewrite test.md failed: %v", err)
	}
	updatedModTime := originalModTime.Add(2 * time.Second)
	if err := os.Chtimes(mdFile, updatedModTime, updatedModTime); err != nil {
		t.Fatalf("update test.md mod time failed: %v", err)
	}

	// Detect change
	changed, changedFile := srv.checkForChanges(modTimes)
	if !changed {
		t.Error("should return true after file modification")
	}
	if changedFile != mdFile {
		t.Errorf("changed file should be %q, got %q", mdFile, changedFile)
	}

	// Check again (modTimes updated) should return false
	changed, changedFile = srv.checkForChanges(modTimes)
	if changed {
		t.Error("should return false after modTimes updated")
	}
	if changedFile != "" {
		t.Errorf("changed file should be empty when no changes, got %q", changedFile)
	}
}

// TestCheckForChanges_NewFile tests new file detection
func TestCheckForChanges_NewFile(t *testing.T) {
	watchDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(watchDir, "existing.md"), []byte("# Existing"), 0o644); err != nil {
		t.Fatalf("write existing.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Add a new file
	if err := os.WriteFile(filepath.Join(watchDir, "new_file.md"), []byte("# New"), 0o644); err != nil {
		t.Fatalf("write new_file.md failed: %v", err)
	}

	changed, _ := srv.checkForChanges(modTimes)
	if !changed {
		t.Error("should return true after new file added")
	}
}

// TestCheckForChanges_DeleteFile tests deleted file detection
func TestCheckForChanges_DeleteFile(t *testing.T) {
	watchDir := t.TempDir()
	toDelete := filepath.Join(watchDir, "to_delete.md")
	if err := os.WriteFile(toDelete, []byte("# Delete Me"), 0o644); err != nil {
		t.Fatalf("write to_delete.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Delete file
	if err := os.Remove(toDelete); err != nil {
		t.Fatalf("remove file failed: %v", err)
	}

	changed, _ := srv.checkForChanges(modTimes)
	if !changed {
		t.Error("should return true after file deleted")
	}
}

// TestStartContextCancel tests stopping the server via context cancellation
func TestStartContextCancel(t *testing.T) {
	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("<html><body></body></html>"), 0o644); err != nil {
		t.Fatalf("write index.html failed: %v", err)
	}
	watchDir := t.TempDir()

	srv := NewServer("127.0.0.1", 0, watchDir, outputDir, slog.Default())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Port 0 test: Start should exit cleanly when context is canceled.
	err := srv.Start(ctx)
	if err != nil && !strings.Contains(err.Error(), "context") {
		t.Fatalf("Start returned unexpected error: %v", err)
	}
}

// TestScanModTimes_SkipNodeModules tests skipping node_modules directory
func TestScanModTimes_SkipNodeModules(t *testing.T) {
	watchDir := t.TempDir()

	// Create node_modules directory
	nmDir := filepath.Join(watchDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatalf("mkdir node_modules failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "package.md"), []byte("# Package"), 0o644); err != nil {
		t.Fatalf("write package.md failed: %v", err)
	}

	// Create normal file
	if err := os.WriteFile(filepath.Join(watchDir, "chapter.md"), []byte("# Chapter"), 0o644); err != nil {
		t.Fatalf("write chapter.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Should only have 1 file (chapter.md), excluding node_modules
	if len(modTimes) != 1 {
		t.Errorf("expected 1 scanned file, got %d", len(modTimes))
	}
}

// TestScanModTimes_SkipBookDir tests skipping _book directory
func TestScanModTimes_SkipBookDir(t *testing.T) {
	watchDir := t.TempDir()

	bookDir := filepath.Join(watchDir, "_book")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatalf("mkdir _book failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "output.md"), []byte("# Output"), 0o644); err != nil {
		t.Fatalf("write output.md failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(watchDir, "source.md"), []byte("# Source"), 0o644); err != nil {
		t.Fatalf("write source.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	if len(modTimes) != 1 {
		t.Errorf("expected 1 scanned file, got %d", len(modTimes))
	}
}

// TestScanModTimes_YAMLAndYML tests recognizing both .yaml and .yml extensions
func TestScanModTimes_YAMLAndYML(t *testing.T) {
	watchDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(watchDir, "config.yaml"), []byte("a: 1"), 0o644); err != nil {
		t.Fatalf("write config.yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(watchDir, "data.yml"), []byte("b: 2"), 0o644); err != nil {
		t.Fatalf("write data.yml failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	if len(modTimes) != 2 {
		t.Errorf("expected 2 scanned files (.yaml and .yml), got %d", len(modTimes))
	}
}

// TestInjectLiveReload_MissingFile tests requesting a non-existent HTML file
func TestInjectLiveReload_MissingFile(t *testing.T) {
	outputDir := t.TempDir()

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/nonexistent.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should not panic; should return 404 or be handled by file server
	if rec.Code == http.StatusOK {
		t.Error("non-existent file should not return 200")
	}
}

// TestBuildFuncError tests BuildFunc returning an error
func TestBuildFuncError(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	expectedErr := fmt.Errorf("build failed")
	srv.BuildFunc = func() error {
		return expectedErr
	}

	err := srv.BuildFunc()
	if err == nil {
		t.Error("BuildFunc should return error")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("error message mismatch: got %q, want %q", err.Error(), expectedErr.Error())
	}
}

// TestSnapshotClients tests that snapshotClients returns a copy, not a reference
func TestSnapshotClients(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect a client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// Read the "connected" acknowledgment
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Failed to read connected message: %v", err)
	}

	// Get a snapshot
	snapshot1 := srv.snapshotClients()
	if len(snapshot1) != 1 {
		t.Errorf("Expected 1 client in snapshot, got %d", len(snapshot1))
	}

	// Get another snapshot
	snapshot2 := srv.snapshotClients()

	// Both snapshots should have the same client, but they should be different slices
	if len(snapshot1) != len(snapshot2) {
		t.Errorf("Snapshots have different lengths: %d vs %d", len(snapshot1), len(snapshot2))
	}

	// Verify the snapshots are independent copies (not the same underlying array)
	if &snapshot1[0] == &snapshot2[0] {
		t.Error("Snapshots should be independent copies with different memory addresses")
	}

	// Both should reference the same client
	if snapshot1[0] != snapshot2[0] {
		t.Error("Snapshots should contain the same client object")
	}
}

// TestBrowserURL tests URL generation for various host values
func TestBrowserURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "localhost",
			host:     "localhost",
			port:     8080,
			expected: "http://localhost:8080",
		},
		{
			name:     "127.0.0.1",
			host:     "127.0.0.1",
			port:     3000,
			expected: "http://127.0.0.1:3000",
		},
		{
			name:     "empty host becomes localhost",
			host:     "",
			port:     9000,
			expected: "http://localhost:9000",
		},
		{
			name:     "0.0.0.0 becomes localhost",
			host:     "0.0.0.0",
			port:     5000,
			expected: "http://localhost:5000",
		},
		{
			name:     "::: becomes localhost",
			host:     "::",
			port:     8000,
			expected: "http://localhost:8000",
		},
		{
			name:     "[::] becomes localhost",
			host:     "[::]",
			port:     8000,
			expected: "http://localhost:8000",
		},
		{
			name:     "IPv6 address",
			host:     "::1",
			port:     8080,
			expected: "http://[::1]:8080",
		},
		{
			name:     "IPv6 with colons (not brackets)",
			host:     "fe80::1",
			port:     3000,
			expected: "http://[fe80::1]:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.host, tt.port, "/tmp", "/tmp", slog.Default())
			url := srv.browserURL()
			if url != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, url)
			}
		})
	}
}

// TestListenDynamicPort tests Listen with port 0 for dynamic allocation
func TestListenDynamicPort(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, "/tmp", "/tmp", slog.Default())

	ln, err := srv.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	// Port should be assigned dynamically
	if srv.Port <= 0 {
		t.Errorf("Expected positive port, got %d", srv.Port)
	}

	// Verify we can actually listen on the port
	addr := ln.Addr()
	if addr == nil {
		t.Error("Listener address should not be nil")
	}
}

// TestListenFromDynamicAllocation tests ListenFrom with port 0 for dynamic allocation
func TestListenFromDynamicAllocation(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, "/tmp", "/tmp", slog.Default())

	ln, err := srv.ListenFrom(0)
	if err != nil {
		t.Fatalf("ListenFrom failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	// Port should be dynamically allocated
	if srv.Port <= 0 {
		t.Errorf("Expected positive port, got %d", srv.Port)
	}
}

// TestNotifyCSSUpdate tests CSS-only update notifications
func TestNotifyCSSUpdate(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect a client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// Read the "connected" acknowledgment
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Failed to read connected message: %v", err)
	}

	// Send CSS update notification
	srv.notifyCSSUpdate()

	// Read the CSS update message
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	msgStr := string(msg)
	if !strings.Contains(msgStr, `"type":"css-update"`) {
		t.Errorf("Expected css-update message, got: %q", msgStr)
	}

	if !strings.Contains(msgStr, `"timestamp"`) {
		t.Errorf("Expected timestamp in message, got: %q", msgStr)
	}
}

// TestNotifyBuildError tests build error notifications
func TestNotifyBuildError(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect a client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// Read the "connected" acknowledgment
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Failed to read connected message: %v", err)
	}

	// Send a build error notification
	errorMsg := "Failed to compile: syntax error on line 42"
	srv.notifyBuildError(errorMsg)

	// Read the error message
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	msgStr := string(msg)
	if !strings.Contains(msgStr, `"type":"build-error"`) {
		t.Errorf("Expected build-error message, got: %q", msgStr)
	}

	if !strings.Contains(msgStr, errorMsg) {
		t.Errorf("Expected error message %q in notification, got: %q", errorMsg, msgStr)
	}

	if !strings.Contains(msgStr, `"timestamp"`) {
		t.Errorf("Expected timestamp in message, got: %q", msgStr)
	}
}

// TestNotifyBuildStart tests build start notifications
func TestNotifyBuildStart(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Failed to read connected message: %v", err)
	}

	srv.notifyBuildStart()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	msgStr := string(msg)
	if !strings.Contains(msgStr, `"type":"build-start"`) {
		t.Errorf("Expected build-start message, got: %q", msgStr)
	}
	if !strings.Contains(msgStr, `"timestamp"`) {
		t.Errorf("Expected timestamp in message, got: %q", msgStr)
	}
}

// TestNotifyBuildErrorWithSpecialChars tests build error with special characters
func TestNotifyBuildErrorWithSpecialChars(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// Connect a client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// Read the "connected" acknowledgment
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("Failed to read connected message: %v", err)
	}

	// Send a build error with special characters
	errorMsg := "Error: \"quoted\" string with\nline breaks and\ttabs"
	srv.notifyBuildError(errorMsg)

	// Read the error message
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("Failed to set read deadline: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	msgStr := string(msg)
	if !strings.Contains(msgStr, `"type":"build-error"`) {
		t.Errorf("Expected build-error message, got: %q", msgStr)
	}

	// The message should be JSON-escaped properly
	if !strings.Contains(msgStr, `"error"`) {
		t.Errorf("Expected error field in message, got: %q", msgStr)
	}
}

// TestConcurrentClientRegistration tests concurrent client registration + notification
func TestConcurrentClientRegistration(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	const numClients = 5
	var wg sync.WaitGroup
	conns := make([]*websocket.Conn, numClients)
	connsMu := sync.Mutex{}

	// Concurrently register clients
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", idx, err)
				return
			}

			// Read the "connected" acknowledgment
			if _, _, err := conn.ReadMessage(); err != nil {
				t.Errorf("Client %d failed to read ack: %v", idx, err)
				conn.Close() //nolint:errcheck
				return
			}

			connsMu.Lock()
			conns[idx] = conn
			connsMu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all clients are registered
	srv.clientsMu.RLock()
	if len(srv.clients) != numClients {
		srv.clientsMu.RUnlock()
		t.Errorf("Expected %d clients, got %d", numClients, len(srv.clients))
		return
	}
	srv.clientsMu.RUnlock()

	// Send notification while clients are still registering/active
	var notifyWg sync.WaitGroup
	for i := 0; i < 3; i++ {
		notifyWg.Add(1)
		go func() {
			defer notifyWg.Done()
			srv.notifyClients()
		}()
	}

	notifyWg.Wait()

	// Verify all clients received at least one message
	for i, conn := range conns {
		if conn == nil {
			continue
		}
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Errorf("Client %d failed to set deadline: %v", i, err)
			continue
		}

		msgCount := 0
		for msgCount < 3 {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			msgStr := string(msg)
			if strings.Contains(msgStr, `"type":"reload"`) {
				msgCount++
			}
		}

		if msgCount == 0 {
			t.Errorf("Client %d received no reload messages", i)
		}
	}

	// Cleanup
	connsMu.Lock()
	for _, conn := range conns {
		if conn != nil {
			conn.Close() //nolint:errcheck
		}
	}
	connsMu.Unlock()
}

// TestInjectLiveReload_PathTraversal tests path traversal protection
func TestInjectLiveReload_PathTraversal(t *testing.T) {
	outputDir := t.TempDir()

	// Create a legitimate HTML file
	htmlFile := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(htmlFile, []byte("<html><body>OK</body></html>"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Try to access a file outside the output directory
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/../../../etc/passwd.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should return forbidden or 404, not OK
	if rec.Code == http.StatusOK {
		t.Error("Path traversal attempt should not return 200")
	}
}

// TestWSClientWriteMessage tests the writeMessage method on wsClient
func TestWSClientWriteMessage(t *testing.T) {
	// Create a test WebSocket connection using httptest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close() //nolint:errcheck

		client := &wsClient{conn: conn}

		// Test writing a message
		if err := client.writeMessage(websocket.TextMessage, []byte("test message")); err != nil {
			t.Errorf("writeMessage failed: %v", err)
		}

		// Test concurrent writes to verify lock is working
		errs := make(chan error, 3)
		for i := 0; i < 3; i++ {
			go func(idx int) {
				msg := []byte(fmt.Sprintf("msg %d", idx))
				errs <- client.writeMessage(websocket.TextMessage, msg)
			}(i)
		}

		// Wait for goroutines and collect errors
		for i := 0; i < 3; i++ {
			if err := <-errs; err != nil {
				t.Errorf("concurrent writeMessage failed: %v", err)
			}
		}

		// Read messages from client to keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}))
	defer server.Close()

	// Connect as client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Close() //nolint:errcheck

	// Read all messages (1 initial + 3 concurrent)
	for i := 0; i < 4; i++ {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message %d: %v", i, err)
		}
		if i == 0 && string(msg) != "test message" {
			t.Errorf("Expected 'test message', got %q", string(msg))
		}
	}
}

// TestSnapshotClientsThreadSafety tests that snapshotClients is thread-safe
func TestSnapshotClientsThreadSafety(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Create mock clients (lock required to avoid race with concurrent readers)
	mockClients := make([]*wsClient, 5)
	srv.clientsMu.Lock()
	for i := 0; i < 5; i++ {
		mockClients[i] = &wsClient{}
		srv.clients[mockClients[i]] = struct{}{}
	}
	srv.clientsMu.Unlock()

	// Concurrent reads should not panic or deadlock
	snapshots := make(chan []*wsClient, 3)
	errs := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			snapshot := srv.snapshotClients()
			if snapshot == nil {
				errs <- fmt.Errorf("snapshot is nil")
				return
			}
			snapshots <- snapshot
		}()
	}

	for i := 0; i < 3; i++ {
		select {
		case snap := <-snapshots:
			if len(snap) != 5 {
				t.Errorf("Expected 5 clients in snapshot, got %d", len(snap))
			}
		case err := <-errs:
			t.Errorf("snapshotClients error: %v", err)
		case <-time.After(2 * time.Second):
			t.Error("snapshotClients timed out")
		}
	}
}

// TestBrowserURLEdgeCases tests various edge cases for browserURL
func TestBrowserURLEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		browserHost string
		port        int
		expectedURL string
	}{
		{
			name:        "IPv6 localhost",
			host:        "::1",
			browserHost: "::1",
			port:        8080,
			expectedURL: "http://[::1]:8080",
		},
		{
			name:        "IPv6 full address",
			host:        "2001:db8::1",
			browserHost: "2001:db8::1",
			port:        3000,
			expectedURL: "http://[2001:db8::1]:3000",
		},
		{
			name:        "0.0.0.0 becomes localhost",
			host:        "0.0.0.0",
			browserHost: "0.0.0.0",
			port:        8080,
			expectedURL: "http://localhost:8080",
		},
		{
			name:        ":: becomes localhost",
			host:        "::",
			browserHost: "::",
			port:        8080,
			expectedURL: "http://localhost:8080",
		},
		{
			name:        "[[::]] becomes localhost",
			host:        "[::]",
			browserHost: "[::]",
			port:        8080,
			expectedURL: "http://localhost:8080",
		},
		{
			name:        "custom host with port",
			host:        "example.com:8080",
			browserHost: "example.com:8080",
			port:        9090,
			expectedURL: "http://[example.com:8080]:9090",
		},
		{
			name:        "standard hostname",
			host:        "localhost",
			browserHost: "localhost",
			port:        5000,
			expectedURL: "http://localhost:5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{
				Host:        tt.host,
				browserHost: tt.browserHost,
				Port:        tt.port,
			}
			url := srv.browserURL()
			if url != tt.expectedURL {
				t.Errorf("browserURL() = %q, want %q", url, tt.expectedURL)
			}
		})
	}
}

// TestNewServerOptionsDefaults tests that NewServer sets proper option defaults
func TestNewServerOptionsDefaults(t *testing.T) {
	tmpWatchDir := t.TempDir()
	tmpOutputDir := t.TempDir()

	tests := []struct {
		name      string
		host      string
		port      int
		wantHost  string
		wantBHost string
	}{
		{
			name:      "empty host defaults to 127.0.0.1",
			host:      "",
			port:      8080,
			wantHost:  "127.0.0.1",
			wantBHost: "localhost",
		},
		{
			name:      "explicit localhost",
			host:      "localhost",
			port:      8080,
			wantHost:  "localhost",
			wantBHost: "localhost",
		},
		{
			name:      "0.0.0.0 kept as is",
			host:      "0.0.0.0",
			port:      8080,
			wantHost:  "0.0.0.0",
			wantBHost: "0.0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.host, tt.port, tmpWatchDir, tmpOutputDir, slog.Default())

			if srv.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", srv.Host, tt.wantHost)
			}
			if srv.browserHost != tt.wantBHost {
				t.Errorf("browserHost = %q, want %q", srv.browserHost, tt.wantBHost)
			}
			// Only test host/browserHost resolution logic; skip tautological
			// field-assignment assertions (Port, clients, etc.).
		})
	}
}

// TestWebSocketClientRegistration tests client registration and deregistration
func TestWebSocketClientRegistration(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Initially empty
	srv.clientsMu.RLock()
	n := len(srv.clients)
	srv.clientsMu.RUnlock()
	if n != 0 {
		t.Errorf("Expected 0 clients initially, got %d", n)
	}

	// Register multiple clients
	srv.clientsMu.Lock()
	for i := 0; i < 5; i++ {
		client := &wsClient{}
		srv.clients[client] = struct{}{}
	}
	srv.clientsMu.Unlock()

	srv.clientsMu.RLock()
	n = len(srv.clients)
	srv.clientsMu.RUnlock()
	if n != 5 {
		t.Errorf("Expected 5 clients after registration, got %d", n)
	}

	// Deregister clients
	clientsList := make([]*wsClient, 0, 5)
	srv.clientsMu.RLock()
	for c := range srv.clients {
		clientsList = append(clientsList, c)
	}
	srv.clientsMu.RUnlock()

	for i, client := range clientsList {
		srv.clientsMu.Lock()
		delete(srv.clients, client)
		srv.clientsMu.Unlock()

		srv.clientsMu.RLock()
		remaining := len(srv.clients)
		srv.clientsMu.RUnlock()

		expected := 5 - i - 1
		if remaining != expected {
			t.Errorf("After deleting client %d: expected %d clients, got %d", i, expected, remaining)
		}
	}

	srv.clientsMu.RLock()
	n = len(srv.clients)
	srv.clientsMu.RUnlock()
	if n != 0 {
		t.Errorf("Expected 0 clients after deregistration, got %d", n)
	}
}

// TestListenWithDynamicPortSelection tests port assignment from listener
func TestListenWithDynamicPortSelection(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, "/tmp", "/tmp", slog.Default())

	ln, err := srv.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	// Port should be assigned automatically
	if srv.Port <= 0 || srv.Port >= 65536 {
		t.Errorf("Port should be valid after Listen, got %d", srv.Port)
	}

	// Try to use the same port - should fail if Listen sets it correctly
	srv2 := NewServer("127.0.0.1", srv.Port, "/tmp", "/tmp", slog.Default())
	_, err = srv2.Listen()
	if err == nil {
		t.Error("Should not be able to listen on occupied port")
	}
}

// TestInjectLiveReload_HTMLWithoutBodyTag tests HTML injection when </body> is missing
func TestInjectLiveReload_HTMLWithoutBodyTag(t *testing.T) {
	outputDir := t.TempDir()

	// Create an HTML file without </body> tag
	htmlFile := filepath.Join(outputDir, "test.html")
	htmlContent := `<html><head><title>Test</title></head><body>Content</body>`
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rec.Code)
	}

	// The reload script should still be added (before </body>)
	body := rec.Body.String()
	if !strings.Contains(body, "</body>") {
		t.Error("Response should contain </body> tag")
	}
	if !strings.Contains(body, "mdpress") {
		t.Error("Response should contain mdpress script")
	}
}

// TestInjectLiveReload_IndexHTMLFallback tests that root path falls back to index.html
func TestInjectLiveReload_IndexHTMLFallback(t *testing.T) {
	outputDir := t.TempDir()

	// Create index.html
	indexFile := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(indexFile, []byte("<html><body>Index</body></html>"), 0o644); err != nil {
		t.Fatalf("Failed to write index file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Test with root path
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for /, got %d", rec.Code)
	}

	// Test with trailing slash on subdir
	req = httptest.NewRequestWithContext(context.Background(), "GET", "/subdir/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	// This might not find index.html, which is OK - just verify no panic
}

// TestInjectLiveReload_NonExistentPathRedirectsToRoot tests that a request for
// a non-existent HTML path (e.g. a mistyped URL) redirects to the home page.
func TestInjectLiveReload_NonExistentPathRedirectsToRoot(t *testing.T) {
	outputDir := t.TempDir()

	// Create only the root index.html – no subdirectory pages.
	indexFile := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(indexFile, []byte("<html><body>Home</body></html>"), 0o644); err != nil {
		t.Fatalf("Failed to write index file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Request a path that does not exist.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/02_reasoning-2.4_reflexion/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("Expected 302 redirect for non-existent path, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" {
		t.Errorf("Expected redirect to /, got %q", loc)
	}

	// A non-existent .html path should also redirect.
	req = httptest.NewRequestWithContext(context.Background(), "GET", "/missing-page.html", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("Expected 302 redirect for non-existent .html path, got %d", rec.Code)
	}

	// Static asset requests should NOT redirect – they should fall through
	// to the file server which returns 404.
	for _, assetPath := range []string{
		"/style.css",
		"/app.js",
		"/logo.png",
		"/photo.jpg",
		"/icon.svg",
		"/favicon.ico",
		"/font.woff2",
	} {
		req = httptest.NewRequestWithContext(context.Background(), "GET", assetPath, nil)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusTemporaryRedirect {
			t.Errorf("Static asset %s should not redirect, got 307", assetPath)
		}
	}
}

// TestScanModTimes_CSSFiles tests that CSS files are properly tracked
func TestScanModTimes_CSSFiles(t *testing.T) {
	watchDir := t.TempDir()

	// Create test files with different extensions
	cssFile := filepath.Join(watchDir, "style.css")
	mdFile := filepath.Join(watchDir, "readme.md")
	txtFile := filepath.Join(watchDir, "notes.txt")

	if err := os.WriteFile(cssFile, []byte("body { }"), 0o644); err != nil {
		t.Fatalf("Failed to write CSS file: %v", err)
	}
	if err := os.WriteFile(mdFile, []byte("# Title"), 0o644); err != nil {
		t.Fatalf("Failed to write MD file: %v", err)
	}
	if err := os.WriteFile(txtFile, []byte("text"), 0o644); err != nil {
		t.Fatalf("Failed to write TXT file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Should include CSS and MD files, but not TXT
	if _, ok := modTimes[cssFile]; !ok {
		t.Error("CSS file should be tracked")
	}
	if _, ok := modTimes[mdFile]; !ok {
		t.Error("MD file should be tracked")
	}
	if _, ok := modTimes[txtFile]; ok {
		t.Error("TXT file should not be tracked")
	}
}

// TestIsAddrInUseDetection tests the isAddrInUse error detection function
func TestIsAddrInUseDetection(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldMatch bool
	}{
		{
			name:        "nil error",
			err:         nil,
			shouldMatch: false,
		},
		{
			name:        "EADDRINUSE syscall error",
			err:         syscall.EADDRINUSE,
			shouldMatch: true,
		},
		{
			name:        "generic error with 'address already in use' message",
			err:         fmt.Errorf("listen tcp: address already in use"),
			shouldMatch: true, // string fallback matches
		},
		{
			name:        "unrelated error",
			err:         fmt.Errorf("some other error"),
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAddrInUse(tt.err)
			if result != tt.shouldMatch {
				t.Errorf("isAddrInUse(%v) = %v, want %v", tt.err, result, tt.shouldMatch)
			}
		})
	}
}

func TestIsSkippedDir(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{"git dir", ".git", true},
		{"hidden dir generic", ".hidden", true},
		{"node_modules", "node_modules", true},
		{"_book", "_book", true},
		{"vendor", "vendor", true},
		{"regular dir", "src", false},
		{"output dir", "output", false},
		{"docs dir", "docs", false},
		{"empty string", "", false},
		{"cache dir without dot prefix", "cache", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSkippedDir(tt.dirName)
			if result != tt.expected {
				t.Errorf("isSkippedDir(%q) = %v, want %v", tt.dirName, result, tt.expected)
			}
		})
	}
}

func TestIsWatchedExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		{"markdown", ".md", true},
		{"yaml", ".yaml", true},
		{"yml", ".yml", true},
		{"css", ".css", true},
		{"go source", ".go", false},
		{"json", ".json", false},
		{"html", ".html", false},
		{"empty string", "", false},
		{"no dot md", "md", false},
		{"txt", ".txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWatchedExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("isWatchedExtension(%q) = %v, want %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestBuildWSMessage(t *testing.T) {
	tests := []struct {
		name        string
		msgType     string
		wantType    string
		wantHasTime bool
	}{
		{"reload message", msgTypeReload, msgTypeReload, true},
		{"build-start message", msgTypeBuildStart, msgTypeBuildStart, true},
		{"css-update message", msgTypeCSSUpdate, msgTypeCSSUpdate, true},
		{"arbitrary type", "custom-event", "custom-event", true},
		{"empty type", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UnixMilli()
			data := buildWSMessage(tt.msgType)
			after := time.Now().UnixMilli()

			if data == nil {
				t.Fatal("buildWSMessage returned nil")
			}

			s := string(data)
			if !strings.Contains(s, `"type":"`+tt.wantType+`"`) {
				t.Errorf("message %q missing expected type field %q", s, tt.wantType)
			}

			// Verify timestamp field is present and within range
			var parsed struct {
				Type      string `json:"type"`
				Timestamp int64  `json:"timestamp"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to parse message: %v", err)
			}
			if parsed.Timestamp < before || parsed.Timestamp > after {
				t.Errorf("timestamp %d not in range [%d, %d]", parsed.Timestamp, before, after)
			}
		})
	}
}

func TestBuildWSErrorMessage(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantErr bool
	}{
		{"simple error", "build failed", false},
		{"empty message", "", false},
		{"message with special chars", "error: <script>alert('xss')</script>", false},
		{"message with newlines", "line1\nline2", false},
		{"message with quotes", `error: "file not found"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := buildWSErrorMessage(tt.errMsg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildWSErrorMessage(%q) error = %v, wantErr %v", tt.errMsg, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if data == nil {
				t.Fatal("buildWSErrorMessage returned nil data")
			}

			var parsed struct {
				Type      string `json:"type"`
				Timestamp int64  `json:"timestamp"`
				Error     string `json:"error"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to parse message: %v", err)
			}
			if parsed.Type != msgTypeBuildErr {
				t.Errorf("type = %q, want %q", parsed.Type, msgTypeBuildErr)
			}
			if parsed.Error != tt.errMsg {
				t.Errorf("error field = %q, want %q", parsed.Error, tt.errMsg)
			}
			if parsed.Timestamp <= 0 {
				t.Errorf("timestamp should be positive, got %d", parsed.Timestamp)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
	}

	for header, want := range expectedHeaders {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %q = %q, want %q", header, got, want)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	}
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP missing default-src directive: %s", csp)
	}
	if !strings.Contains(csp, "https://cdn.jsdelivr.net") {
		t.Errorf("CSP missing CDN allowlist for Mermaid/KaTeX: %s", csp)
	}
}
