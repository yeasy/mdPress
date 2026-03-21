// Package server provides the live preview HTTP server.
// It watches files with fsnotify and pushes reload notifications over WebSocket.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

// wsClient wraps a single WebSocket connection with a dedicated write lock.
type wsClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// writeMessage sends a message to the connection safely.
func (c *wsClient) writeMessage(msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(msgType, data)
}

// Server implements the live preview server.
type Server struct {
	// Configuration
	Host        string // Listening host or IP.
	Port        int    // Listening port.
	WatchDir    string // Source directory to watch.
	OutputDir   string // Output directory.
	AutoOpen    bool   // Whether to open the browser automatically.
	browserHost string

	// BuildFunc is provided by the caller and rebuilds the project output.
	BuildFunc func() error

	// Internal state
	clients   map[*wsClient]struct{} // Connected WebSocket clients.
	clientsMu sync.RWMutex
	logger    *slog.Logger
	upgrader  websocket.Upgrader // WebSocket upgrader.
}

// NewServer creates a preview server.
func NewServer(host string, port int, watchDir, outputDir string, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	browserHost := host
	if host == "" {
		host = "127.0.0.1"
		browserHost = "localhost"
	}
	return &Server{
		Host:        host,
		Port:        port,
		WatchDir:    watchDir,
		OutputDir:   outputDir,
		browserHost: browserHost,
		clients:     make(map[*wsClient]struct{}),
		logger:      logger,
		upgrader: websocket.Upgrader{
			// Allow all origins for local development.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Listen reserves the configured port and returns the listener.
func (s *Server) Listen() (net.Listener, error) {
	addr := net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("address %s is already in use (try mdpress serve --host %s --port %d): %w", addr, s.Host, s.Port+1, err)
	}
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
		s.Port = tcpAddr.Port
	}
	return ln, nil
}

// ListenFrom reserves the first available port from startPort upward.
func (s *Server) ListenFrom(startPort int) (net.Listener, error) {
	for port := startPort; port <= 65535; port++ {
		s.Port = port
		ln, err := s.Listen()
		if err == nil {
			return ln, nil
		}
		if isAddrInUse(err) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("no available port found from %d to 65535", startPort)
}

func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.EADDRINUSE)
}

// Start runs the server until the context is canceled.
func (s *Server) Start(ctx context.Context) error {
	ln, err := s.Listen()
	if err != nil {
		return err
	}
	return s.StartWithListener(ctx, ln)
}

// StartWithListener runs the server using a pre-bound listener.
func (s *Server) StartWithListener(ctx context.Context, ln net.Listener) error {
	if ln == nil {
		return fmt.Errorf("listener is nil")
	}
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
		s.Port = tcpAddr.Port
	}

	// Create the router.
	mux := http.NewServeMux()

	// Static file server with injected live reload support.
	fileServer := http.FileServer(http.Dir(s.OutputDir))
	mux.Handle("/", s.injectLiveReload(fileServer))

	// WebSocket endpoint for reload notifications.
	mux.HandleFunc("/__mdpress_ws", s.handleWebSocket)

	// Start file watching.
	go s.watchFilesWithFsnotify(ctx)

	server := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: mux,
	}

	// Graceful shutdown.
	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down server...")
		// Close all WebSocket clients.
		s.clientsMu.Lock()
		for client := range s.clients {
			client.conn.Close() //nolint:errcheck
		}
		s.clientsMu.Unlock()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Debug("Server shutdown returned error", slog.String("error", err.Error()))
		}
	}()

	fmt.Printf("\n📖 mdpress Live Preview Server\n\n")
	fmt.Printf("  Address: %s\n", s.browserURL())
	fmt.Printf("  Binding: %s\n", net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port)))
	fmt.Printf("  Watching: %s\n", s.WatchDir)
	fmt.Printf("  Output: %s\n", s.OutputDir)
	fmt.Printf("\n  File changes automatically trigger browser reloads (fsnotify + WebSocket)\n")
	fmt.Printf("  Press Ctrl+C to stop the server\n\n")

	// Open the browser when requested.
	if s.AutoOpen {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(s.browserURL())
		}()
	}

	err := server.Serve(ln)
	if err != nil && errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		return nil
	}
	return err
}

// snapshotClients returns a snapshot of the current client set, allowing
// callers to iterate without holding the lock. This prevents a slow client's
// writeMessage from blocking new connection registrations.
func (s *Server) snapshotClients() []*wsClient {
	s.clientsMu.RLock()
	clients := make([]*wsClient, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.clientsMu.RUnlock()
	return clients
}

// notifyClients sends a reload message to all connected WebSocket clients.
func (s *Server) notifyClients() {
	msg := []byte(`{"type":"reload","timestamp":` + fmt.Sprintf("%d", time.Now().UnixMilli()) + `}`)
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// notifyCSSUpdate sends a CSS-only update message to all connected WebSocket clients.
func (s *Server) notifyCSSUpdate() {
	msg := []byte(`{"type":"css-update","timestamp":` + fmt.Sprintf("%d", time.Now().UnixMilli()) + `}`)
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// notifyBuildError sends a build error message to all connected WebSocket clients.
func (s *Server) notifyBuildError(errMsg string) {
	// Use json.Marshal to properly escape all special characters including
	// \b, \f, Unicode control characters, etc.
	escapedBytes, err := json.Marshal(errMsg)
	if err != nil {
		s.logger.Error("Failed to marshal build error message", slog.String("error", err.Error()))
		return
	}
	msg := []byte(fmt.Sprintf(`{"type":"build-error","timestamp":%d,"error":%s}`, time.Now().UnixMilli(), escapedBytes))
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// handleWebSocket upgrades an HTTP request to a WebSocket connection.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", slog.String("error", err.Error()))
		return
	}

	// Wrap the connection so writes remain thread-safe.
	client := &wsClient{conn: conn}

	// Register the client.
	s.clientsMu.Lock()
	s.clients[client] = struct{}{}
	total := len(s.clients)
	s.clientsMu.Unlock()

	s.logger.Debug("WebSocket client connected", slog.Int("total", total))

	// Send the connection acknowledgment.
	if err := client.writeMessage(websocket.TextMessage, []byte("connected")); err != nil {
		s.logger.Debug("Failed to send WebSocket ack", slog.String("error", err.Error()))
	}

	// Keep reading to detect disconnects.
	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, client)
		s.clientsMu.Unlock()
		conn.Close() //nolint:errcheck
		s.logger.Debug("WebSocket client disconnected")
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

// injectLiveReload injects live-reload JavaScript into HTML responses.
func (s *Server) injectLiveReload(next http.Handler) http.Handler {
	// Browser-side script: connect over WebSocket and reload on change.
	reloadScript := `
<!-- mdpress live reload (WebSocket) -->
<script>
(function() {
  'use strict';
  var scheme = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
  var wsURL = scheme + window.location.host + '/__mdpress_ws';
  var reconnectInterval = 2000;
  var maxReconnectInterval = 30000;
  var currentInterval = reconnectInterval;

  // Error overlay management
  var overlay = null;
  function showErrorOverlay(msg) {
    removeErrorOverlay();
    overlay = document.createElement('div');
    overlay.id = 'mdpress-error-overlay';
    overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.85);color:#ff6b6b;z-index:99999;padding:32px;font-family:monospace;font-size:14px;overflow:auto;white-space:pre-wrap;';
    var header = document.createElement('div');
    header.style.cssText = 'font-size:20px;font-weight:bold;margin-bottom:16px;color:#ff6b6b;';
    header.textContent = 'Build Error';
    var close = document.createElement('button');
    close.textContent = 'Dismiss';
    close.style.cssText = 'position:fixed;top:16px;right:16px;background:#ff6b6b;color:#fff;border:none;padding:8px 16px;border-radius:4px;cursor:pointer;font-size:14px;';
    close.onclick = removeErrorOverlay;
    var content = document.createElement('pre');
    content.style.cssText = 'color:#ffa0a0;margin:0;line-height:1.6;';
    content.textContent = msg;
    overlay.appendChild(header);
    overlay.appendChild(close);
    overlay.appendChild(content);
    document.body.appendChild(overlay);
  }
  function removeErrorOverlay() {
    if (overlay && overlay.parentNode) {
      overlay.parentNode.removeChild(overlay);
      overlay = null;
    }
  }

  function connect() {
    var ws = new WebSocket(wsURL);

    ws.onopen = function() {
      console.log('[mdpress] WebSocket connected');
      currentInterval = reconnectInterval;
    };

    ws.onmessage = function(e) {
      var data = e.data;
      // Support both legacy string and JSON messages.
      if (data === 'reload') {
        removeErrorOverlay();
        location.reload();
        return;
      }
      try {
        var msg = JSON.parse(data);
        if (msg.type === 'reload') {
          removeErrorOverlay();
          location.reload();
        } else if (msg.type === 'css-update') {
          removeErrorOverlay();
          // Reload all stylesheets without page flash.
          var links = document.querySelectorAll('link[rel="stylesheet"]');
          links.forEach(function(link) {
            var href = link.getAttribute('href');
            if (href) {
              link.setAttribute('href', href.split('?')[0] + '?t=' + Date.now());
            }
          });
          // Also reload inline styles from server
          console.log('[mdpress] CSS updated without page reload');
        } else if (msg.type === 'build-error') {
          console.error('[mdpress] Build error:', msg.error);
          showErrorOverlay(msg.error);
        }
      } catch(err) {
        // Unknown message format, ignore.
      }
    };

    ws.onclose = function() {
      console.log('[mdpress] WebSocket disconnected, retrying in ' + (currentInterval/1000) + 's...');
      setTimeout(function() {
        currentInterval = Math.min(currentInterval * 1.5, maxReconnectInterval);
        connect();
      }, currentInterval);
    };

    ws.onerror = function() {
      ws.close();
    };
  }

  connect();
})();
</script>
`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Proxy non-HTML requests directly.
		if !strings.HasSuffix(r.URL.Path, ".html") && r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, "/") {
			next.ServeHTTP(w, r)
			return
		}

		// For HTML responses, inject the reload script.
		path := r.URL.Path
		if path == "/" || strings.HasSuffix(path, "/") {
			path = path + "index.html"
		}

		filePath := filepath.Join(s.OutputDir, filepath.Clean(path))

		// Protect against path traversal by keeping access within OutputDir.
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		absOutputDir, err := filepath.Abs(s.OutputDir)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(absFilePath, absOutputDir+string(filepath.Separator)) && absFilePath != absOutputDir {
			s.logger.Warn("Blocked path traversal attempt", slog.String("path", r.URL.Path))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Inject the reload script before </body>.
		html := string(content)
		injected := strings.Replace(html, "</body>", reloadScript+"</body>", 1)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write([]byte(injected)); err != nil {
			s.logger.Debug("Failed to write HTTP response", slog.String("error", err.Error()))
		}
	})
}

func (s *Server) browserURL() string {
	host := s.Host
	if s.browserHost != "" {
		host = s.browserHost
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "localhost"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%d", host, s.Port)
}

// watchFilesWithFsnotify uses fsnotify to watch for changes and trigger rebuilds.
func (s *Server) watchFilesWithFsnotify(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		s.logger.Error("Failed to create fsnotify watcher, falling back to polling", slog.String("error", err.Error()))
		s.watchFilesPolling(ctx)
		return
	}
	defer watcher.Close() //nolint:errcheck

	// Recursively add watched directories.
	err = filepath.Walk(s.WatchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			// Skip hidden, dependency, and output directories.
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" || base == "vendor" {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		s.logger.Error("Failed to add watch directory", slog.String("error", err.Error()))
		return
	}

	s.logger.Info("fsnotify watcher started", slog.String("dir", s.WatchDir))

	// Debounce rebuilds to avoid repeated triggers on file save.
	var debounceTimer *time.Timer
	var debounceMu sync.Mutex
	var lastTriggeredExt string

	// Stop any pending debounce timer when the watcher exits to prevent
	// the callback from firing after the server has begun shutting down.
	defer func() {
		debounceMu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only rebuild on changes to .md, .yaml, .yml, and .css files.
			ext := strings.ToLower(filepath.Ext(event.Name))
			if ext != ".md" && ext != ".yaml" && ext != ".yml" && ext != ".css" {
				continue
			}

			// Only react to write and create events.
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Watch newly created directories for recursive monitoring.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					base := filepath.Base(event.Name)
					if !strings.HasPrefix(base, ".") && base != "node_modules" && base != "_book" && base != "vendor" {
						if addErr := watcher.Add(event.Name); addErr != nil {
							s.logger.Warn("Failed to watch new directory", slog.String("dir", event.Name), slog.String("error", addErr.Error()))
						} else {
							s.logger.Debug("Added new directory to watcher", slog.String("dir", event.Name))
						}
					}
					continue
				}
			}

			s.logger.Debug("Detected file change", slog.String("file", event.Name), slog.String("op", event.Op.String()))

			// Capture values for the closure to avoid referencing the loop variable.
			triggerFile := event.Name
			triggerExt := ext

			// Debounce multiple changes within 500ms into one rebuild.
			// When a non-CSS file changes, escalate to a full reload so that
			// a Markdown edit followed by a quick CSS save still triggers a
			// full page reload instead of a CSS-only update.
			debounceMu.Lock()
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			if triggerExt != ".css" {
				lastTriggeredExt = triggerExt
			} else if lastTriggeredExt == "" {
				lastTriggeredExt = triggerExt
			}
			// else: keep the previous non-CSS ext so a full reload is triggered
			capturedExt := lastTriggeredExt
			debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
				s.logger.Info("File change detected, rebuilding...", slog.String("trigger", filepath.Base(triggerFile)))
				if s.BuildFunc != nil {
					if err := s.BuildFunc(); err != nil {
						s.logger.Error("Rebuild failed", slog.String("error", err.Error()))
						s.notifyBuildError(err.Error())
						return
					}
				}
				s.logger.Info("Build completed, notifying browser to reload")
				if capturedExt == ".css" {
					s.notifyCSSUpdate()
				} else {
					s.notifyClients()
				}
				// Reset for next debounce cycle.
				debounceMu.Lock()
				lastTriggeredExt = ""
				debounceMu.Unlock()
			})
			debounceMu.Unlock()

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			s.logger.Error("fsnotify error", slog.String("error", err.Error()))
		}
	}
}

// watchFilesPolling polls for file changes as a fallback when fsnotify is unavailable.
func (s *Server) watchFilesPolling(ctx context.Context) {
	lastModTimes := make(map[string]time.Time)
	s.scanModTimes(lastModTimes)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var debounceTimer *time.Timer
	var debounceMu sync.Mutex

	// Stop any pending debounce timer when the watcher exits to prevent
	// the callback from firing after the server has begun shutting down.
	defer func() {
		debounceMu.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed := s.checkForChanges(lastModTimes)
			if changed {
				debounceMu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
					s.logger.Info("File change detected, rebuilding...")
					if s.BuildFunc != nil {
						if err := s.BuildFunc(); err != nil {
							s.logger.Error("Rebuild failed", slog.String("error", err.Error()))
							return
						}
					}
					s.logger.Info("Build completed, notifying browser to reload")
					s.notifyClients()
				})
				debounceMu.Unlock()
			}
		}
	}
}

// scanModTimes records file modification times.
func (s *Server) scanModTimes(modTimes map[string]time.Time) {
	if err := filepath.Walk(s.WatchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".yaml" || ext == ".yml" || ext == ".css" {
			modTimes[path] = info.ModTime()
		}
		return nil
	}); err != nil {
		s.logger.Debug("Failed to scan modification times", slog.String("error", err.Error()))
	}
}

// checkForChanges reports whether any watched files changed.
func (s *Server) checkForChanges(modTimes map[string]time.Time) bool {
	changed := false
	newModTimes := make(map[string]time.Time)

	if err := filepath.Walk(s.WatchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".yaml" || ext == ".yml" || ext == ".css" {
			newModTimes[path] = info.ModTime()
			if prevTime, ok := modTimes[path]; !ok || !prevTime.Equal(info.ModTime()) {
				changed = true
			}
		}
		return nil
	}); err != nil {
		s.logger.Debug("Failed to walk watch directory", slog.String("error", err.Error()))
	}

	// Detect deleted files.
	for path := range modTimes {
		if _, ok := newModTimes[path]; !ok {
			changed = true
		}
	}

	// Refresh the modification time map.
	for path, t := range newModTimes {
		modTimes[path] = t
	}
	for path := range modTimes {
		if _, ok := newModTimes[path]; !ok {
			delete(modTimes, path)
		}
	}

	return changed
}

// openBrowser opens the default browser.
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		return
	}
}
