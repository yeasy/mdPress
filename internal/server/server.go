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
	"net/url"
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

// WebSocket message type constants and timing
const (
	msgTypeBuildStart   = "build-start"
	msgTypeReload       = "reload"
	msgTypeCSSUpdate    = "css-update"
	msgTypeBuildErr     = "build-error"
	shutdownTimeout     = 5 * time.Second
	debounceInterval    = 500 * time.Millisecond
	fileWatcherInterval = 1 * time.Second
	browserOpenDelay    = 500 * time.Millisecond
	wsReadLimit         = 4096
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

// buildWSMessage constructs a WebSocket JSON message with timestamp.
func buildWSMessage(msgType string) []byte {
	return []byte(fmt.Sprintf(`{"type":"%s","timestamp":%d}`, msgType, time.Now().UnixMilli()))
}

// buildWSErrorMessage constructs a build-error WebSocket message with escaped error text.
func buildWSErrorMessage(errMsg string) ([]byte, error) {
	// Use json.Marshal to properly escape all special characters
	escapedBytes, err := json.Marshal(errMsg)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf(`{"type":"%s","timestamp":%d,"error":%s}`, msgTypeBuildErr, time.Now().UnixMilli(), escapedBytes)), nil
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

	// Debounce state for file-change rebuilds.
	debounceTimer    *time.Timer
	debounceMu       sync.Mutex
	lastTriggeredExt string
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
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true // non-browser clients
				}
				u, err := url.Parse(origin)
				if err != nil {
					return false
				}
				return u.Host == r.Host
			},
		},
	}
}

// Listen reserves the configured port and returns the listener.
func (s *Server) Listen() (net.Listener, error) {
	addr := net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
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
	// On Unix, the underlying error is syscall.EADDRINUSE.
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	// On Windows, the underlying Winsock error (WSAEADDRINUSE = 10048)
	// may not match the invented syscall.EADDRINUSE constant.
	// Fall back to string matching for cross-platform reliability.
	msg := err.Error()
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "Only one usage of each socket address")
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
		Addr:              ln.Addr().String(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
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

	// Open the browser when requested, respecting context cancellation.
	if s.AutoOpen {
		go func() {
			select {
			case <-time.After(browserOpenDelay):
				openBrowser(s.browserURL())
			case <-ctx.Done():
			}
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
	msg := buildWSMessage(msgTypeReload)
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// notifyBuildStart sends a rebuild-started message to all connected WebSocket clients.
func (s *Server) notifyBuildStart() {
	msg := buildWSMessage(msgTypeBuildStart)
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// notifyCSSUpdate sends a CSS-only update message to all connected WebSocket clients.
func (s *Server) notifyCSSUpdate() {
	msg := buildWSMessage(msgTypeCSSUpdate)
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// notifyBuildError sends a build error message to all connected WebSocket clients.
func (s *Server) notifyBuildError(errMsg string) {
	msg, err := buildWSErrorMessage(errMsg)
	if err != nil {
		s.logger.Error("Failed to marshal build error message", slog.String("error", err.Error()))
		return
	}
	for _, client := range s.snapshotClients() {
		if err := client.writeMessage(websocket.TextMessage, msg); err != nil {
			s.logger.Debug("Failed to send WebSocket message", slog.String("error", err.Error()))
		}
	}
}

// maxWSClients is the maximum number of concurrent WebSocket connections.
const maxWSClients = 100

// handleWebSocket upgrades an HTTP request to a WebSocket connection.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade first, then atomically check-and-register under a single lock.
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", slog.String("error", err.Error()))
		return
	}

	// Wrap the connection so writes remain thread-safe.
	client := &wsClient{conn: conn}

	// Atomically check the limit and register, preventing TOCTOU races.
	s.clientsMu.Lock()
	if len(s.clients) >= maxWSClients {
		s.clientsMu.Unlock()
		conn.Close() //nolint:errcheck
		return
	}
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

	// Limit incoming message size — the server only reads to detect disconnects.
	conn.SetReadLimit(wsReadLimit)
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
	serveInfoJSON, _ := json.Marshal(map[string]string{
		"address":   s.browserURL(),
		"watchDir":  s.WatchDir,
		"outputDir": s.OutputDir,
	})
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
  var serveInfo = ` + string(serveInfoJSON) + `;
  var serveStatusKey = 'mdpress-serve-flash';
  var serveUI = null;
  var serveState = { ws: 'connecting', last: 'Waiting for changes', error: '' };

  function ensureServeUI() {
    if (serveUI) return serveUI;
    var style = document.createElement('style');
    style.textContent =
      '#mdpress-serve-status{position:fixed;top:0;left:0;right:0;z-index:99997;display:none;padding:10px 16px;font:13px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#fff;box-shadow:0 8px 24px rgba(0,0,0,.16)}' +
      '#mdpress-serve-status[data-kind="building"]{background:#1d4ed8}' +
      '#mdpress-serve-status[data-kind="success"]{background:#15803d}' +
      '#mdpress-serve-status[data-kind="error"]{background:#b91c1c}' +
      '#mdpress-serve-status[data-kind="warning"]{background:#92400e}' +
      '#mdpress-serve-status .row{display:flex;align-items:center;gap:10px;flex-wrap:wrap}' +
      '#mdpress-serve-status .text{font-weight:600}' +
      '#mdpress-serve-status .meta{opacity:.85;font-size:12px}' +
      '#mdpress-serve-status .actions{margin-left:auto;display:flex;gap:8px}' +
      '#mdpress-serve-status button{border:0;background:rgba(255,255,255,.18);color:inherit;padding:4px 10px;border-radius:999px;cursor:pointer;font:inherit}' +
      '#mdpress-serve-status pre{display:none;width:100%;margin:10px 0 0;padding:10px 12px;border-radius:10px;background:rgba(0,0,0,.18);white-space:pre-wrap;overflow:auto;font:12px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace}' +
      '#mdpress-serve-status.show-details pre{display:block}' +
      '#mdpress-serve-panel-toggle{position:fixed;right:18px;bottom:18px;z-index:99996;border:1px solid rgba(0,0,0,.06);border-radius:999px;background:rgba(255,255,255,.82);color:#6b7280;padding:7px 10px;cursor:pointer;font:600 10px/1 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;box-shadow:0 8px 24px rgba(0,0,0,.10);opacity:.32;transition:opacity .15s ease, transform .15s ease, background .15s ease}' +
      '#mdpress-serve-panel-toggle:hover,#mdpress-serve-panel-toggle:focus-visible{opacity:.92;transform:translateY(-1px);background:rgba(255,255,255,.96);outline:none}' +
      '#mdpress-serve-panel{position:fixed;right:18px;bottom:56px;z-index:99996;width:min(320px,calc(100vw - 24px));background:rgba(255,255,255,.96);color:#374151;border:1px solid rgba(0,0,0,.08);border-radius:14px;padding:12px 13px;box-shadow:0 18px 40px rgba(0,0,0,.18);backdrop-filter:blur(14px);display:none;font:12px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}' +
      '#mdpress-serve-panel.open{display:block}' +
      '#mdpress-serve-panel h2{margin:0 0 8px;font-size:12px;color:#111827}' +
      '#mdpress-serve-panel .line{margin:6px 0}' +
      '#mdpress-serve-panel .label{display:block;color:#6b7280;font-size:10px;text-transform:uppercase;letter-spacing:.05em}' +
      '#mdpress-serve-panel .value{display:block;color:#111827;word-break:break-word}' +
      '#mdpress-serve-panel details{margin-top:8px}' +
      '#mdpress-serve-panel summary{cursor:pointer;color:#4b5563;font-weight:600}' +
      '#mdpress-serve-panel .hint{margin-top:8px;color:#6b7280;font-size:11px}';
    document.head.appendChild(style);

    var status = document.createElement('div');
    status.id = 'mdpress-serve-status';
    status.innerHTML = '<div class="row"><span class="text"></span><span class="meta"></span><div class="actions"><button type="button" class="details">Details</button><button type="button" class="dismiss">Dismiss</button></div></div><pre></pre>';
    document.body.appendChild(status);

    var toggle = document.createElement('button');
    toggle.id = 'mdpress-serve-panel-toggle';
    toggle.type = 'button';
    toggle.textContent = 'Dev';
    document.body.appendChild(toggle);

    var panel = document.createElement('section');
    panel.id = 'mdpress-serve-panel';
    panel.innerHTML = '<h2>Live Preview</h2>' +
      '<div class="line"><span class="label">Status</span><span class="value" data-field="status"></span></div>' +
      '<div class="line"><span class="label">Page</span><span class="value" data-field="page"></span></div>' +
      '<div class="line"><span class="label">WebSocket</span><span class="value" data-field="ws"></span></div>' +
      '<details><summary>Paths</summary>' +
      '<div class="line"><span class="label">Address</span><span class="value" data-field="address"></span></div>' +
      '<div class="line"><span class="label">Watching</span><span class="value" data-field="watch"></span></div>' +
      '<div class="line"><span class="label">Output</span><span class="value" data-field="output"></span></div>' +
      '</details>' +
      '<div class="hint">Serve-only tools and rebuild state live here. Static site output is unchanged.</div>';
    document.body.appendChild(panel);

    toggle.addEventListener('click', function() {
      panel.classList.toggle('open');
    });
    status.querySelector('.dismiss').addEventListener('click', function() {
      status.style.display = 'none';
      status.classList.remove('show-details');
    });
    status.querySelector('.details').addEventListener('click', function() {
      status.classList.toggle('show-details');
    });

    serveUI = {
      status: status,
      text: status.querySelector('.text'),
      meta: status.querySelector('.meta'),
      detail: status.querySelector('pre'),
      detailsBtn: status.querySelector('.details'),
      panel: panel,
      statusField: panel.querySelector('[data-field="status"]'),
      addressField: panel.querySelector('[data-field="address"]'),
      watchField: panel.querySelector('[data-field="watch"]'),
      outputField: panel.querySelector('[data-field="output"]'),
      pageField: panel.querySelector('[data-field="page"]'),
      wsField: panel.querySelector('[data-field="ws"]')
    };
    refreshServePanel();
    restoreServeFlash();
    return serveUI;
  }

  function refreshServePanel() {
    var ui = ensureServeUI();
    ui.statusField.textContent = serveState.last || 'Waiting for changes';
    ui.addressField.textContent = serveInfo.address;
    ui.watchField.textContent = serveInfo.watchDir;
    ui.outputField.textContent = serveInfo.outputDir;
    ui.pageField.textContent = window.location.pathname + window.location.hash;
    ui.wsField.textContent = serveState.ws;
  }
  window.__mdpressRefreshServePanel = refreshServePanel;

  function setServeStatus(kind, text, detail, sticky) {
    var ui = ensureServeUI();
    serveState.last = text;
    if (detail) serveState.error = detail;
    ui.status.dataset.kind = kind;
    ui.text.textContent = text;
    ui.meta.textContent = new Date().toLocaleTimeString();
    ui.detail.textContent = detail || '';
    ui.detailsBtn.style.display = detail ? '' : 'none';
    ui.status.classList.toggle('show-details', !!detail && kind === 'error');
    ui.status.style.display = 'block';
    refreshServePanel();
    if (!sticky) {
      window.setTimeout(function() {
        if (ui.status.dataset.kind === kind) {
          ui.status.style.display = 'none';
          ui.status.classList.remove('show-details');
        }
      }, 2200);
    }
  }

  function rememberServeFlash(kind, text) {
    try {
      sessionStorage.setItem(serveStatusKey, JSON.stringify({ kind: kind, text: text, ts: Date.now() }));
    } catch (e) {}
  }

  function restoreServeFlash() {
    try {
      var raw = sessionStorage.getItem(serveStatusKey);
      if (!raw) return;
      sessionStorage.removeItem(serveStatusKey);
      var payload = JSON.parse(raw);
      if (!payload || (Date.now() - payload.ts) > 5000) return;
      setServeStatus(payload.kind || 'success', payload.text || 'Updated', '', false);
    } catch (e) {}
  }

  function connect() {
    var ws = new WebSocket(wsURL);

    ws.onopen = function() {
      console.log('[mdpress] WebSocket connected');
      serveState.ws = 'connected';
      currentInterval = reconnectInterval;
      refreshServePanel();
    };

    ws.onmessage = function(e) {
      var data = e.data;
      // Support both legacy string and JSON messages.
      if (data === 'reload') {
        rememberServeFlash('success', 'Rebuild complete');
        location.reload();
        return;
      }
      try {
        var msg = JSON.parse(data);
        if (msg.type === '` + msgTypeBuildStart + `') {
          setServeStatus('building', 'Rebuilding…', '', true);
        } else if (msg.type === 'reload') {
          rememberServeFlash('success', 'Rebuild complete');
          location.reload();
        } else if (msg.type === 'css-update') {
          setServeStatus('success', 'Styles updated', '', false);
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
          setServeStatus('error', 'Build failed', msg.error, true);
        }
      } catch(err) {
        // Unknown message format, ignore.
      }
    };

    ws.onclose = function() {
      serveState.ws = 'reconnecting';
      refreshServePanel();
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

  ensureServeUI();
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
			path += "index.html"
		}

		filePath := filepath.Join(s.OutputDir, filepath.Clean(path))

		// Protect against path traversal by keeping access within OutputDir.
		// Use Clean to normalize paths, EvalSymlinks to resolve symlinks, and
		// case-insensitive comparison to prevent bypasses on case-insensitive
		// filesystems (e.g., Windows, macOS).
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
		// Resolve symlinks so that a symlink pointing outside OutputDir is caught.
		// Only resolve the output dir when the file path also resolves, so that
		// both stay in the same namespace.  When the file does not exist,
		// EvalSymlinks fails and the output dir may resolve differently (e.g.
		// /tmp → /private/tmp on macOS), causing a false path-traversal block.
		fileSymlinksResolved := false
		if resolved, err := filepath.EvalSymlinks(absFilePath); err == nil {
			absFilePath = resolved
			fileSymlinksResolved = true
		}
		if fileSymlinksResolved {
			if resolved, err := filepath.EvalSymlinks(absOutputDir); err == nil {
				absOutputDir = resolved
			}
		}
		// Normalize paths and perform case-insensitive comparison on case-insensitive systems
		cleanFilePath := filepath.Clean(absFilePath)
		cleanOutputDir := filepath.Clean(absOutputDir)
		caseInsensitiveCheck := strings.ToLower(cleanFilePath)
		caseInsensitiveOutputDir := strings.ToLower(cleanOutputDir)

		isWithinOutputDir := caseInsensitiveCheck == caseInsensitiveOutputDir ||
			strings.HasPrefix(caseInsensitiveCheck, caseInsensitiveOutputDir+string(filepath.Separator))

		if !isWithinOutputDir {
			s.logger.Warn("Blocked path traversal attempt", slog.String("path", r.URL.Path))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		content, err := os.ReadFile(absFilePath)
		if err != nil {
			// Fallback: if the requested HTML page does not exist and this is
			// not already the root path, redirect to the home page so that
			// mistyped or stale URLs land on a useful page instead of a 404.
			// Only redirect page navigations (no extension or .html); static
			// asset requests (.css, .js, .png, etc.) should still 404.
			if os.IsNotExist(err) && r.URL.Path != "/" {
				ext := filepath.Ext(r.URL.Path)
				if ext == "" || ext == ".html" {
					s.logger.Warn("Page not found, redirecting to /", slog.String("path", r.URL.Path))
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			}
			next.ServeHTTP(w, r)
			return
		}

		// Inject the reload script before </body>.
		html := string(content)
		injected := strings.Replace(html, "</body>", reloadScript+"</body>", 1)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
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

// stopDebounceTimer cancels any pending debounce timer. The caller must hold
// s.debounceMu.
func (s *Server) stopDebounceTimer() {
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}
}

// debouncedRebuild resets the debounce timer and schedules a rebuild after
// debounceInterval. triggerFile is the file that changed (used for logging).
// ext is the lowercased file extension of the trigger (e.g. ".css", ".md").
// When ext is ".css" and no non-CSS change was batched, only a CSS-only update
// is sent; otherwise a full page reload is triggered.
func (s *Server) debouncedRebuild(ctx context.Context, triggerFile, ext string) {
	s.debounceMu.Lock()
	s.stopDebounceTimer()

	// Track the "most significant" extension in the current debounce window.
	// A non-CSS change escalates any pending CSS-only update to a full reload.
	if ext != ".css" {
		s.lastTriggeredExt = ext
	} else if s.lastTriggeredExt == "" {
		s.lastTriggeredExt = ext
	}
	// else: keep the previous non-CSS ext so a full reload is triggered

	capturedExt := s.lastTriggeredExt
	s.debounceTimer = time.AfterFunc(debounceInterval, func() {
		if ctx.Err() != nil {
			return
		}
		s.logger.Info("File change detected, rebuilding...", slog.String("trigger", filepath.Base(triggerFile)))
		s.notifyBuildStart()
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
		s.debounceMu.Lock()
		s.lastTriggeredExt = ""
		s.debounceMu.Unlock()
	})
	s.debounceMu.Unlock()
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

	// Stop any pending debounce timer when the watcher exits to prevent
	// the callback from firing after the server has begun shutting down.
	defer func() {
		s.debounceMu.Lock()
		s.stopDebounceTimer()
		s.debounceMu.Unlock()
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
			s.debouncedRebuild(ctx, event.Name, ext)

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

	ticker := time.NewTicker(fileWatcherInterval)
	defer ticker.Stop()

	// Stop any pending debounce timer when the watcher exits to prevent
	// the callback from firing after the server has begun shutting down.
	defer func() {
		s.debounceMu.Lock()
		s.stopDebounceTimer()
		s.debounceMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, changedFile := s.checkForChanges(lastModTimes)
			if changed {
				ext := strings.ToLower(filepath.Ext(changedFile))
				s.debouncedRebuild(ctx, changedFile, ext)
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
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" || base == "vendor" {
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
func (s *Server) checkForChanges(modTimes map[string]time.Time) (bool, string) {
	changed := false
	changedFile := ""
	newModTimes := make(map[string]time.Time)

	if err := filepath.Walk(s.WatchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		if info.IsDir() {
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".yaml" || ext == ".yml" || ext == ".css" {
			newModTimes[path] = info.ModTime()
			if prevTime, ok := modTimes[path]; !ok || !prevTime.Equal(info.ModTime()) {
				changed = true
				if changedFile == "" {
					changedFile = path
				}
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
			if changedFile == "" {
				changedFile = path
			}
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

	return changed, changedFile
}

// openBrowser opens the default browser. Only http/https URLs are allowed.
func openBrowser(rawURL string) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return
	}

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{rawURL}
	case "linux":
		cmd = "xdg-open"
		args = []string{rawURL}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	c := exec.CommandContext(ctx, cmd, args...)
	if err := c.Start(); err != nil {
		cancel()
		return
	}
	// Collect exit status in background to prevent zombie processes.
	go func() {
		defer cancel()
		c.Wait() //nolint:errcheck
	}()
}
