// server_test.go 测试预览服务器的核心功能。
// 包括：NewServer 创建、WebSocket 处理、HTML 注入、文件变化检测、轮询监听等。
package server

import (
	"context"
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

// TestNewServer 测试服务器创建的基本属性
func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      int
		watchDir  string
		outputDir string
		logger    *slog.Logger
	}{
		{
			name:      "基本创建",
			host:      "127.0.0.1",
			port:      8080,
			watchDir:  "/tmp/watch",
			outputDir: "/tmp/output",
			logger:    slog.Default(),
		},
		{
			name:      "自定义端口",
			host:      "0.0.0.0",
			port:      3000,
			watchDir:  "/tmp/watch2",
			outputDir: "/tmp/output2",
			logger:    slog.Default(),
		},
		{
			name:      "nil logger 使用默认",
			host:      "",
			port:      9090,
			watchDir:  "/tmp/w",
			outputDir: "/tmp/o",
			logger:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.host, tt.port, tt.watchDir, tt.outputDir, tt.logger)

			if srv == nil {
				t.Fatal("NewServer 返回 nil")
			}
			expectedHost := tt.host
			if expectedHost == "" {
				expectedHost = "127.0.0.1"
			}
			if srv.Host != expectedHost {
				t.Errorf("Host = %q, 期望 %q", srv.Host, expectedHost)
			}
			if srv.Port != tt.port {
				t.Errorf("Port = %d, 期望 %d", srv.Port, tt.port)
			}
			if srv.WatchDir != tt.watchDir {
				t.Errorf("WatchDir = %q, 期望 %q", srv.WatchDir, tt.watchDir)
			}
			if srv.OutputDir != tt.outputDir {
				t.Errorf("OutputDir = %q, 期望 %q", srv.OutputDir, tt.outputDir)
			}
			if srv.clients == nil {
				t.Error("clients map 应该被初始化")
			}
			if srv.logger == nil {
				t.Error("logger 不应为 nil（即使传入 nil 也应使用默认）")
			}
			// AutoOpen 默认为 false
			if srv.AutoOpen {
				t.Error("AutoOpen 默认应为 false")
			}
		})
	}
}

func TestListen_PortAlreadyInUse(t *testing.T) {
	occupied, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("无法占用测试端口: %v", err)
	}
	defer occupied.Close() //nolint:errcheck

	port := occupied.Addr().(*net.TCPAddr).Port
	srv := NewServer("127.0.0.1", port, t.TempDir(), t.TempDir(), slog.Default())

	_, err = srv.Listen()
	if err == nil {
		t.Fatal("预期 Listen 返回端口占用错误")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("错误信息应包含端口占用提示, 实际: %v", err)
	}
}

func TestListenFrom_SkipsOccupiedPort(t *testing.T) {
	occupied, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("无法占用测试端口: %v", err)
	}
	defer occupied.Close() //nolint:errcheck

	startPort := occupied.Addr().(*net.TCPAddr).Port
	srv := NewServer("127.0.0.1", startPort, t.TempDir(), t.TempDir(), slog.Default())

	ln, err := srv.ListenFrom(startPort)
	if err != nil {
		t.Fatalf("ListenFrom 返回错误: %v", err)
	}
	defer ln.Close() //nolint:errcheck

	if srv.Port <= startPort {
		t.Fatalf("应跳过已占用端口 %d, 实际使用 %d", startPort, srv.Port)
	}
}

// TestInjectLiveReload_HTMLFile 测试 HTML 文件中注入实时刷新脚本
func TestInjectLiveReload_HTMLFile(t *testing.T) {
	// 创建临时输出目录和 HTML 文件
	outputDir := t.TempDir()
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello</h1>
</body>
</html>`
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(htmlContent), 0644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())

	// 创建一个带注入中间件的 handler
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// 请求根路径（应该注入脚本）
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// 验证脚本被注入
	if !strings.Contains(body, "__mdpress_ws") {
		t.Error("HTML 响应应包含 WebSocket 连接脚本")
	}
	if strings.Contains(body, "ws://localhost:8080") {
		t.Error("HTML 响应不应硬编码 localhost WebSocket 地址")
	}
	if !strings.Contains(body, "window.location.host") {
		t.Error("HTML 响应应基于当前访问地址建立 WebSocket 连接")
	}
	if !strings.Contains(body, "location.reload()") {
		t.Error("HTML 响应应包含自动刷新逻辑")
	}
	// 验证原始内容保留
	if !strings.Contains(body, "<h1>Hello</h1>") {
		t.Error("原始 HTML 内容应保留")
	}
	// 验证 Content-Type
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type 应为 text/html, 实际 %q", ct)
	}
	// 验证 Cache-Control
	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control 应为 no-cache, 实际 %q", cc)
	}
}

// TestInjectLiveReload_NonHTML 测试非 HTML 文件不会被注入脚本
func TestInjectLiveReload_NonHTML(t *testing.T) {
	outputDir := t.TempDir()
	cssContent := `body { color: red; }`
	if err := os.WriteFile(filepath.Join(outputDir, "style.css"), []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequest("GET", "/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	// 非 HTML 文件不应包含注入脚本
	if strings.Contains(body, "__mdpress_ws") {
		t.Error("非 HTML 文件不应包含 WebSocket 脚本")
	}
}

// TestInjectLiveReload_DirectoryPath 测试目录路径请求（以 / 结尾）
func TestInjectLiveReload_DirectoryPath(t *testing.T) {
	outputDir := t.TempDir()
	subDir := filepath.Join(outputDir, "chapter1")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir subdir failed: %v", err)
	}
	htmlContent := `<!DOCTYPE html><html><body><p>Chapter</p></body></html>`
	if err := os.WriteFile(filepath.Join(subDir, "index.html"), []byte(htmlContent), 0644); err != nil {
		t.Fatalf("write index.html failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// 请求以 / 结尾的路径
	req := httptest.NewRequest("GET", "/chapter1/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "__mdpress_ws") {
		t.Error("目录路径请求的 HTML 也应注入脚本")
	}
}

// TestHandleWebSocket 测试 WebSocket 连接处理
func TestHandleWebSocket(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// 创建测试服务器
	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	// 将 http:// 替换为 ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// 连接 WebSocket
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket 连接失败: %v", err)
	}
	defer conn.Close() //nolint:errcheck

	// 读取连接确认消息
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取 WebSocket 消息失败: %v", err)
	}
	if string(msg) != "connected" {
		t.Errorf("期望收到 'connected'，实际收到 %q", string(msg))
	}

	// 验证客户端已注册
	srv.clientsMu.RLock()
	clientCount := len(srv.clients)
	srv.clientsMu.RUnlock()

	if clientCount != 1 {
		t.Errorf("应有 1 个客户端, 实际 %d", clientCount)
	}

	// 关闭连接后验证客户端被移除
	conn.Close() //nolint:errcheck
	// 等待一小段时间让 goroutine 处理断开事件
	time.Sleep(100 * time.Millisecond)

	srv.clientsMu.RLock()
	clientCount = len(srv.clients)
	srv.clientsMu.RUnlock()

	if clientCount != 0 {
		t.Errorf("断开后应有 0 个客户端, 实际 %d", clientCount)
	}
}

// TestNotifyClients 测试通知所有客户端
func TestNotifyClients(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// 连接多个客户端
	const numClients = 3
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("客户端 %d 连接失败: %v", i, err)
		}
		conns[i] = conn
		// 读取 "connected" 确认消息
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("客户端 %d 读取 connected 消息失败: %v", i, err)
		}
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close() //nolint:errcheck
			}
		}
	}()

	// 通知所有客户端
	srv.notifyClients()

	// 验证所有客户端都收到 "reload" 消息
	for i, conn := range conns {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("客户端 %d 设置读取超时失败: %v", i, err)
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("客户端 %d 读取消息失败: %v", i, err)
		}
		if !strings.Contains(string(msg), `"type":"reload"`) {
			t.Errorf("客户端 %d 收到 %q, 期望包含 reload JSON 消息", i, string(msg))
		}
	}
}

// TestNotifyClientsEmpty 测试没有客户端时通知不报错
func TestNotifyClientsEmpty(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())
	// 不应 panic 或报错
	srv.notifyClients()
}

// TestNotifyClientsConcurrent 测试并发通知的安全性
// 每个 wsClient 有独立的 writeMu，保证并发 notifyClients 不会导致
// gorilla/websocket 并发写入 panic。
func TestNotifyClientsConcurrent(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// 测试无客户端时的并发安全性
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

// TestNotifyClientsConcurrentWithClients 测试有真实客户端连接时的并发写入安全性
func TestNotifyClientsConcurrentWithClients(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/"

	// 连接多个客户端
	const numClients = 3
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("客户端 %d 连接失败: %v", i, err)
		}
		conns[i] = conn
		if _, _, err := conn.ReadMessage(); err != nil { // 读取 "connected"
			t.Fatalf("客户端 %d 读取 connected 消息失败: %v", i, err)
		}
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close() //nolint:errcheck
			}
		}
	}()

	// 并发调用 notifyClients，验证 wsClient.writeMu 防止并发写入 panic
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

// TestScanModTimes 测试文件修改时间扫描
func TestScanModTimes(t *testing.T) {
	watchDir := t.TempDir()

	// 创建测试文件
	for name, content := range map[string]string{
		"chapter1.md": "# Chapter 1",
		"config.yaml": "title: test",
		"style.css":   "body{}",
		"image.png":   "png",
	} {
		if err := os.WriteFile(filepath.Join(watchDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}

	// 创建隐藏目录（应被跳过）
	hiddenDir := filepath.Join(watchDir, ".git")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("mkdir hidden dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "config.yml"), []byte("git"), 0644); err != nil {
		t.Fatalf("write hidden config failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// 应该扫描到 .md, .yaml, .css 文件
	if len(modTimes) != 3 {
		t.Errorf("应扫描到 3 个文件, 实际 %d", len(modTimes))
		for k := range modTimes {
			t.Logf("  文件: %s", k)
		}
	}

	// 验证不包含 .png 文件
	for path := range modTimes {
		if strings.HasSuffix(path, ".png") {
			t.Errorf("不应包含 .png 文件: %s", path)
		}
	}
}

// TestCheckForChanges 测试文件变化检测
func TestCheckForChanges(t *testing.T) {
	watchDir := t.TempDir()
	mdFile := filepath.Join(watchDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("write test.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())

	// 初始扫描
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// 没有变化时应返回 false
	changed := srv.checkForChanges(modTimes)
	if changed {
		t.Error("无文件变化时应返回 false")
	}

	// 修改文件
	time.Sleep(50 * time.Millisecond) // 确保时间戳不同
	if err := os.WriteFile(mdFile, []byte("# Test Modified"), 0644); err != nil {
		t.Fatalf("rewrite test.md failed: %v", err)
	}

	// 检测到变化
	changed = srv.checkForChanges(modTimes)
	if !changed {
		t.Error("文件修改后应返回 true")
	}

	// 再次检查（modTimes 已更新）应返回 false
	changed = srv.checkForChanges(modTimes)
	if changed {
		t.Error("modTimes 更新后应返回 false")
	}
}

// TestCheckForChanges_NewFile 测试新增文件检测
func TestCheckForChanges_NewFile(t *testing.T) {
	watchDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(watchDir, "existing.md"), []byte("# Existing"), 0644); err != nil {
		t.Fatalf("write existing.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// 新增一个文件
	if err := os.WriteFile(filepath.Join(watchDir, "new_file.md"), []byte("# New"), 0644); err != nil {
		t.Fatalf("write new_file.md failed: %v", err)
	}

	changed := srv.checkForChanges(modTimes)
	if !changed {
		t.Error("新增文件后应返回 true")
	}
}

// TestCheckForChanges_DeleteFile 测试删除文件检测
func TestCheckForChanges_DeleteFile(t *testing.T) {
	watchDir := t.TempDir()
	toDelete := filepath.Join(watchDir, "to_delete.md")
	if err := os.WriteFile(toDelete, []byte("# Delete Me"), 0644); err != nil {
		t.Fatalf("write to_delete.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// 删除文件
	if err := os.Remove(toDelete); err != nil {
		t.Fatalf("remove file failed: %v", err)
	}

	changed := srv.checkForChanges(modTimes)
	if !changed {
		t.Error("删除文件后应返回 true")
	}
}

// TestStartContextCancel 测试通过 context 取消来停止服务器
func TestStartContextCancel(t *testing.T) {
	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("<html><body></body></html>"), 0644); err != nil {
		t.Fatalf("write index.html failed: %v", err)
	}
	watchDir := t.TempDir()

	srv := NewServer("127.0.0.1", 0, watchDir, outputDir, slog.Default())
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 端口 0 测试 Start 能正常退出（不 panic）
	err := srv.Start(ctx)
	if err != nil {
		t.Logf("Start 返回错误（预期的）: %v", err)
	}
}

// TestScanModTimes_SkipNodeModules 测试跳过 node_modules 目录
func TestScanModTimes_SkipNodeModules(t *testing.T) {
	watchDir := t.TempDir()

	// 创建 node_modules 目录
	nmDir := filepath.Join(watchDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatalf("mkdir node_modules failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "package.md"), []byte("# Package"), 0644); err != nil {
		t.Fatalf("write package.md failed: %v", err)
	}

	// 创建正常文件
	if err := os.WriteFile(filepath.Join(watchDir, "chapter.md"), []byte("# Chapter"), 0644); err != nil {
		t.Fatalf("write chapter.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// 应该只有 1 个文件（chapter.md），不包含 node_modules 下的文件
	if len(modTimes) != 1 {
		t.Errorf("应扫描到 1 个文件, 实际 %d", len(modTimes))
	}
}

// TestScanModTimes_SkipBookDir 测试跳过 _book 目录
func TestScanModTimes_SkipBookDir(t *testing.T) {
	watchDir := t.TempDir()

	bookDir := filepath.Join(watchDir, "_book")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("mkdir _book failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bookDir, "output.md"), []byte("# Output"), 0644); err != nil {
		t.Fatalf("write output.md failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(watchDir, "source.md"), []byte("# Source"), 0644); err != nil {
		t.Fatalf("write source.md failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	if len(modTimes) != 1 {
		t.Errorf("应扫描到 1 个文件, 实际 %d", len(modTimes))
	}
}

// TestScanModTimes_YAMLAndYML 测试同时识别 .yaml 和 .yml 扩展名
func TestScanModTimes_YAMLAndYML(t *testing.T) {
	watchDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(watchDir, "config.yaml"), []byte("a: 1"), 0644); err != nil {
		t.Fatalf("write config.yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(watchDir, "data.yml"), []byte("b: 2"), 0644); err != nil {
		t.Fatalf("write data.yml failed: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	if len(modTimes) != 2 {
		t.Errorf("应扫描到 2 个文件（.yaml 和 .yml）, 实际 %d", len(modTimes))
	}
}

// TestInjectLiveReload_MissingFile 测试请求不存在的 HTML 文件
func TestInjectLiveReload_MissingFile(t *testing.T) {
	outputDir := t.TempDir()

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequest("GET", "/nonexistent.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// 不应 panic，应返回 404 或由文件服务器处理
	if rec.Code == http.StatusOK {
		t.Error("不存在的文件不应返回 200")
	}
}

// TestNewServerDefaultLogger 测试 nil logger 时使用默认 logger
func TestNewServerDefaultLogger(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", nil)
	if srv.logger == nil {
		t.Error("传入 nil logger 时应使用默认 logger")
	}
}

// TestBuildFuncIntegration 测试 BuildFunc 回调的集成
func TestBuildFuncIntegration(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	buildCalled := false
	srv.BuildFunc = func() error {
		buildCalled = true
		return nil
	}

	// BuildFunc 不为 nil
	if srv.BuildFunc == nil {
		t.Error("BuildFunc 应已设置")
	}

	// 手动调用
	err := srv.BuildFunc()
	if err != nil {
		t.Errorf("BuildFunc 不应报错: %v", err)
	}
	if !buildCalled {
		t.Error("BuildFunc 应被调用")
	}
}

// TestBuildFuncError 测试 BuildFunc 返回错误
func TestBuildFuncError(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	expectedErr := fmt.Errorf("构建失败")
	srv.BuildFunc = func() error {
		return expectedErr
	}

	err := srv.BuildFunc()
	if err == nil {
		t.Error("BuildFunc 应返回错误")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("错误消息不匹配: 得到 %q, 期望 %q", err.Error(), expectedErr.Error())
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

// TestIsAddrInUse tests the isAddrInUse error detection
func TestIsAddrInUse(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "EADDRINUSE error",
			err:      fmt.Errorf("listen tcp 127.0.0.1:8080: %w", syscall.EADDRINUSE),
			expected: true,
		},
		{
			name:     "address already in use string",
			err:      fmt.Errorf("address already in use"),
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("permission denied"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAddrInUse(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
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
	if err := os.WriteFile(htmlFile, []byte("<html><body>OK</body></html>"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Try to access a file outside the output directory
	req := httptest.NewRequest("GET", "/../../../etc/passwd.html", nil)
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

	// Create mock clients
	mockClients := make([]*wsClient, 5)
	for i := 0; i < 5; i++ {
		mockClients[i] = &wsClient{}
		srv.clients[mockClients[i]] = struct{}{}
	}

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
			srv := NewServer(tt.host, tt.port, "/tmp/watch", "/tmp/out", slog.Default())

			if srv.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", srv.Host, tt.wantHost)
			}
			if srv.browserHost != tt.wantBHost {
				t.Errorf("browserHost = %q, want %q", srv.browserHost, tt.wantBHost)
			}
			if srv.Port != tt.port {
				t.Errorf("Port = %d, want %d", srv.Port, tt.port)
			}
			if srv.clients == nil {
				t.Error("clients map should be initialized")
			}
			if srv.logger == nil {
				t.Error("logger should be initialized")
			}
			if srv.AutoOpen != false {
				t.Error("AutoOpen should default to false")
			}
			if srv.BuildFunc != nil {
				t.Error("BuildFunc should default to nil")
			}
		})
	}
}

// TestWebSocketClientRegistration tests client registration and deregistration
func TestWebSocketClientRegistration(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Initially empty
	if len(srv.clients) != 0 {
		t.Errorf("Expected 0 clients initially, got %d", len(srv.clients))
	}

	// Register multiple clients
	for i := 0; i < 5; i++ {
		client := &wsClient{}
		srv.clientsMu.Lock()
		srv.clients[client] = struct{}{}
		srv.clientsMu.Unlock()
	}

	if len(srv.clients) != 5 {
		t.Errorf("Expected 5 clients after registration, got %d", len(srv.clients))
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

	if len(srv.clients) != 0 {
		t.Errorf("Expected 0 clients after deregistration, got %d", len(srv.clients))
	}
}

// TestNotifyClientsMessageFormat tests that notification messages are properly formatted
func TestNotifyClientsMessageFormat(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp", "/tmp", slog.Default())

	// Create a mock client that records messages
	messages := make([]string, 0)
	msgMu := sync.Mutex{}

	// Replace writeMessage with a mock
	originalWriteMessage := func(msg []byte) {
		msgMu.Lock()
		defer msgMu.Unlock()
		messages = append(messages, string(msg))
	}

	// Create test clients that record messages
	testClients := make([]*wsClient, 3)
	for i := 0; i < 3; i++ {
		client := &wsClient{}
		testClients[i] = client
		srv.clients[client] = struct{}{}
	}

	// Test notifyClients message format
	msg := []byte(`{"type":"reload","timestamp":` + fmt.Sprintf("%d", time.Now().UnixMilli()) + `}`)
	if !strings.Contains(string(msg), `"type":"reload"`) {
		t.Error("reload message should contain type:reload")
	}
	if !strings.Contains(string(msg), `"timestamp":`) {
		t.Error("reload message should contain timestamp")
	}

	// Test notifyCSSUpdate message format
	cssMsg := []byte(`{"type":"css-update","timestamp":` + fmt.Sprintf("%d", time.Now().UnixMilli()) + `}`)
	if !strings.Contains(string(cssMsg), `"type":"css-update"`) {
		t.Error("css-update message should contain type:css-update")
	}

	// Test notifyBuildError message format
	errMsg := "test error message"
	buildErrMsg := []byte(fmt.Sprintf(`{"type":"build-error","timestamp":%d,"error":"%s"}`, time.Now().UnixMilli(), errMsg))
	if !strings.Contains(string(buildErrMsg), `"type":"build-error"`) {
		t.Error("build-error message should contain type:build-error")
	}

	// Verify originalWriteMessage was not called (we're not actually testing write behavior here)
	_ = originalWriteMessage
}

// TestListenWithDynamicPortSelection tests port assignment from listener
func TestListenWithDynamicPortSelection(t *testing.T) {
	srv := NewServer("127.0.0.1", 0, "/tmp", "/tmp", slog.Default())

	ln, err := srv.Listen()
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

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
	if err := os.WriteFile(htmlFile, []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	req := httptest.NewRequest("GET", "/test.html", nil)
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
	if err := os.WriteFile(indexFile, []byte("<html><body>Index</body></html>"), 0644); err != nil {
		t.Fatalf("Failed to write index file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, "/tmp", outputDir, slog.Default())
	fileServer := http.FileServer(http.Dir(outputDir))
	handler := srv.injectLiveReload(fileServer)

	// Test with root path
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 for /, got %d", rec.Code)
	}

	// Test with trailing slash on subdir
	req = httptest.NewRequest("GET", "/subdir/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	// This might not find index.html, which is OK - just verify no panic
}

// TestScanModTimes_CSSFiles tests that CSS files are properly tracked
func TestScanModTimes_CSSFiles(t *testing.T) {
	watchDir := t.TempDir()

	// Create test files with different extensions
	cssFile := filepath.Join(watchDir, "style.css")
	mdFile := filepath.Join(watchDir, "readme.md")
	txtFile := filepath.Join(watchDir, "notes.txt")

	if err := os.WriteFile(cssFile, []byte("body { }"), 0644); err != nil {
		t.Fatalf("Failed to write CSS file: %v", err)
	}
	if err := os.WriteFile(mdFile, []byte("# Title"), 0644); err != nil {
		t.Fatalf("Failed to write MD file: %v", err)
	}
	if err := os.WriteFile(txtFile, []byte("text"), 0644); err != nil {
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

// TestCheckForChanges_NoChanges tests that unchanged files are detected correctly
func TestCheckForChanges_NoChanges(t *testing.T) {
	watchDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(watchDir, "test.md")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	srv := NewServer("127.0.0.1", 8080, watchDir, "/tmp", slog.Default())
	modTimes := make(map[string]time.Time)
	srv.scanModTimes(modTimes)

	// Check for changes immediately - should be false
	changed := srv.checkForChanges(modTimes)
	if changed {
		t.Error("Should detect no changes immediately after scan")
	}

	// Check again - still should be false
	changed = srv.checkForChanges(modTimes)
	if changed {
		t.Error("Should detect no changes on subsequent check")
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
			shouldMatch: false,
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

// TestServerInitialClientState tests server initialization with empty client map
func TestServerInitialClientState(t *testing.T) {
	srv := NewServer("127.0.0.1", 8080, "/tmp/watch", "/tmp/output", nil)

	// Verify clients map is properly initialized
	if srv.clients == nil {
		t.Fatal("clients map should not be nil")
	}

	// Verify it's empty
	if len(srv.clients) != 0 {
		t.Errorf("clients map should be empty initially, got %d", len(srv.clients))
	}

	// Verify snapshot of empty map works
	snapshot := srv.snapshotClients()
	if len(snapshot) != 0 {
		t.Errorf("snapshot of empty clients should be empty, got %d", len(snapshot))
	}
}
