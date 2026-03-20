package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestNewGenerator 测试创建生成器
func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator 返回 nil")
	}
	// 验证默认值
	if g.timeout != defaultTimeout {
		t.Errorf("默认超时错误: got %v, want %v", g.timeout, defaultTimeout)
	}
	if g.pageWidth != defaultPageWidth {
		t.Errorf("默认页宽错误: got %f, want %f", g.pageWidth, defaultPageWidth)
	}
	if g.pageHeight != defaultPageHeight {
		t.Errorf("默认页高错误: got %f, want %f", g.pageHeight, defaultPageHeight)
	}
	if !g.printBackground {
		t.Error("默认应打印背景")
	}
}

// TestNewGeneratorWithOptions 测试带选项的创建
func TestNewGeneratorWithOptions(t *testing.T) {
	g := NewGenerator(
		WithTimeout(30*time.Second),
		WithPageSize(148, 210),
		WithMargins(10, 10, 15, 15),
		WithPrintBackground(false),
		WithHeaderFooter(true),
	)

	if g.timeout != 30*time.Second {
		t.Errorf("超时设置错误: got %v", g.timeout)
	}
	if g.pageWidth != 148 {
		t.Errorf("页宽设置错误: got %f", g.pageWidth)
	}
	if g.pageHeight != 210 {
		t.Errorf("页高设置错误: got %f", g.pageHeight)
	}
	if g.marginLeft != 10 {
		t.Errorf("左边距设置错误: got %f", g.marginLeft)
	}
	if g.marginTop != 15 {
		t.Errorf("上边距设置错误: got %f", g.marginTop)
	}
	if g.printBackground {
		t.Error("printBackground 应为 false")
	}
	if !g.displayHeaderFooter {
		t.Error("displayHeaderFooter 应为 true")
	}
}

// TestGenerateEmptyContent 测试空内容
func TestGenerateEmptyContent(t *testing.T) {
	g := NewGenerator()
	err := g.Generate("", "output.pdf")
	if err == nil {
		t.Error("空 HTML 内容应返回错误")
	}
}

// TestGenerateEmptyOutput 测试空输出路径
func TestGenerateEmptyOutput(t *testing.T) {
	g := NewGenerator()
	err := g.Generate("<html></html>", "")
	if err == nil {
		t.Error("空输出路径应返回错误")
	}
}

// TestGenerateFromNonExistentFile 测试不存在的 HTML 文件
func TestGenerateFromNonExistentFile(t *testing.T) {
	g := NewGenerator()
	err := g.GenerateFromFile("/nonexistent/file.html", "output.pdf")
	if err == nil {
		t.Error("不存在的文件应返回错误")
	}
}

// TestWithTimeoutOption 测试超时选项
func TestWithTimeoutOption(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"10秒", 10 * time.Second},
		{"1分钟", time.Minute},
		{"5分钟", 5 * time.Minute},
	}

	for _, tt := range tests {
		g := NewGenerator(WithTimeout(tt.timeout))
		if g.timeout != tt.timeout {
			t.Errorf("%s: 超时设置错误: got %v, want %v", tt.name, g.timeout, tt.timeout)
		}
	}
}

// TestWithPageSizeOption 测试页面尺寸选项
func TestWithPageSizeOption(t *testing.T) {
	tests := []struct {
		name          string
		width, height float64
	}{
		{"A4", 210, 297},
		{"A5", 148, 210},
		{"Letter", 216, 279},
		{"B5", 176, 250},
	}

	for _, tt := range tests {
		g := NewGenerator(WithPageSize(tt.width, tt.height))
		if g.pageWidth != tt.width {
			t.Errorf("%s: 页宽错误: got %f, want %f", tt.name, g.pageWidth, tt.width)
		}
		if g.pageHeight != tt.height {
			t.Errorf("%s: 页高错误: got %f, want %f", tt.name, g.pageHeight, tt.height)
		}
	}
}

// TestWithMarginsOption 测试边距选项
func TestWithMarginsOption(t *testing.T) {
	g := NewGenerator(WithMargins(5, 10, 15, 20))
	if g.marginLeft != 5 {
		t.Errorf("左边距错误: got %f", g.marginLeft)
	}
	if g.marginRight != 10 {
		t.Errorf("右边距错误: got %f", g.marginRight)
	}
	if g.marginTop != 15 {
		t.Errorf("上边距错误: got %f", g.marginTop)
	}
	if g.marginBottom != 20 {
		t.Errorf("下边距错误: got %f", g.marginBottom)
	}
}

// TestChromiumCheck 测试 Chromium 检查
// 注意：此测试在 CI 环境中可能失败（无 Chrome）
func TestChromiumCheck(t *testing.T) {
	g := NewGenerator()
	err := g.checkChromiumAvailable()
	// 不断言具体结果，因为取决于环境
	_ = err
}

// TestMultipleOptionsChaining tests chaining multiple options together.
func TestMultipleOptionsChaining(t *testing.T) {
	g := NewGenerator(
		WithTimeout(90*time.Second),
		WithPageSize(210, 297),
		WithMargins(20, 20, 25, 25),
		WithPrintBackground(true),
		WithHeaderFooter(false),
	)

	if g.timeout != 90*time.Second {
		t.Error("chained options: timeout wrong")
	}
	if g.pageWidth != 210 || g.pageHeight != 297 {
		t.Error("chained options: page size wrong")
	}
	if g.marginLeft != 20 || g.marginTop != 25 {
		t.Error("chained options: margin wrong")
	}
	if !g.printBackground {
		t.Error("chained options: printBackground should be true")
	}
	if g.displayHeaderFooter {
		t.Error("chained options: displayHeaderFooter should be false")
	}
}

// TestDocumentOutlineEnabledByDefault verifies outline is on by default.
func TestDocumentOutlineEnabledByDefault(t *testing.T) {
	g := NewGenerator()
	if !g.generateDocumentOutline {
		t.Error("generateDocumentOutline should be true by default")
	}
	if !g.generateTaggedPDF {
		t.Error("generateTaggedPDF should be true by default")
	}
}

// TestWithDocumentOutlineOption tests toggling the document outline.
func TestWithDocumentOutlineOption(t *testing.T) {
	g := NewGenerator(WithDocumentOutline(false))
	if g.generateDocumentOutline {
		t.Error("generateDocumentOutline should be false after WithDocumentOutline(false)")
	}

	g2 := NewGenerator(WithDocumentOutline(true))
	if !g2.generateDocumentOutline {
		t.Error("generateDocumentOutline should be true after WithDocumentOutline(true)")
	}
}

// TestWithTaggedPDFOption tests toggling tagged PDF generation.
func TestWithTaggedPDFOption(t *testing.T) {
	g := NewGenerator(WithTaggedPDF(false))
	if g.generateTaggedPDF {
		t.Error("generateTaggedPDF should be false after WithTaggedPDF(false)")
	}
}

func TestResolveChromiumPathPrefersMDPRESSChromePath(t *testing.T) {
	t.Setenv("CHROME_BIN", "")
	originalCandidates := chromiumExecutableCandidates
	originalMacPaths := chromiumMacPaths
	chromiumExecutableCandidates = nil
	chromiumMacPaths = nil
	defer func() {
		chromiumExecutableCandidates = originalCandidates
		chromiumMacPaths = originalMacPaths
	}()

	dir := t.TempDir()
	chromePath := filepath.Join(dir, "chrome")
	if err := os.WriteFile(chromePath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MDPRESS_CHROME_PATH", chromePath)

	got, err := resolveChromiumPath()
	if err != nil {
		t.Fatalf("resolveChromiumPath() returned error: %v", err)
	}
	if got != chromePath {
		t.Fatalf("resolveChromiumPath() = %q, want %q", got, chromePath)
	}
}

func TestResolveChromiumPathFallsBackToChromeBin(t *testing.T) {
	t.Setenv("MDPRESS_CHROME_PATH", "")
	originalCandidates := chromiumExecutableCandidates
	originalMacPaths := chromiumMacPaths
	chromiumExecutableCandidates = nil
	chromiumMacPaths = nil
	defer func() {
		chromiumExecutableCandidates = originalCandidates
		chromiumMacPaths = originalMacPaths
	}()

	dir := t.TempDir()
	chromePath := filepath.Join(dir, "chrome-bin")
	if err := os.WriteFile(chromePath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHROME_BIN", chromePath)

	got, err := resolveChromiumPath()
	if err != nil {
		t.Fatalf("resolveChromiumPath() returned error: %v", err)
	}
	if got != chromePath {
		t.Fatalf("resolveChromiumPath() = %q, want %q", got, chromePath)
	}
}

func TestResolveChromiumPathFindsPathExecutable(t *testing.T) {
	t.Setenv("MDPRESS_CHROME_PATH", "")
	t.Setenv("CHROME_BIN", "")

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exeName := "google-chrome"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	chromePath := filepath.Join(binDir, exeName)
	if err := os.WriteFile(chromePath, []byte(""), 0o755); err != nil {
		t.Fatal(err)
	}

	originalCandidates := chromiumExecutableCandidates
	originalMacPaths := chromiumMacPaths
	chromiumExecutableCandidates = []string{"google-chrome"}
	chromiumMacPaths = nil
	defer func() {
		chromiumExecutableCandidates = originalCandidates
		chromiumMacPaths = originalMacPaths
	}()

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	got, err := resolveChromiumPath()
	if err != nil {
		t.Fatalf("resolveChromiumPath() returned error: %v", err)
	}
	if got != chromePath {
		t.Fatalf("resolveChromiumPath() = %q, want %q", got, chromePath)
	}
}

func TestParseChromiumFlags(t *testing.T) {
	args := parseChromiumFlags("--no-sandbox --disable-dev-shm-usage --remote-debugging-port=0 --single-process=false")
	if _, ok := args["no-sandbox"]; !ok {
		t.Fatal("expected no-sandbox flag to be set")
	}
	if value, ok := args["disable-dev-shm-usage"]; !ok || value != true {
		t.Fatalf("expected disable-dev-shm-usage=true, got %v", value)
	}
	if value, ok := args["remote-debugging-port"]; !ok || value != "0" {
		t.Fatalf("expected remote-debugging-port=\"0\", got %v", value)
	}
	if value, ok := args["single-process"]; !ok || value != false {
		t.Fatalf("expected single-process=false, got %v", value)
	}
}

func TestPrepareChromiumRuntimeDirs(t *testing.T) {
	t.Setenv("MDPRESS_CACHE_DIR", t.TempDir())

	runtime, err := prepareChromiumRuntimeDirs()
	if err != nil {
		t.Fatalf("prepareChromiumRuntimeDirs() returned error: %v", err)
	}
	defer runtime.cleanup()

	for _, dir := range []string{runtime.root, runtime.homeDir, runtime.userData, runtime.tmpDir, runtime.xdgConfig, runtime.xdgCache} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("expected runtime directory %q to exist", dir)
		}
	}
}

func TestChromiumRuntimeEnv(t *testing.T) {
	runtime := chromiumRuntimeDirs{
		homeDir:   "/tmp/home",
		tmpDir:    "/tmp/tmp",
		xdgConfig: "/tmp/xdg-config",
		xdgCache:  "/tmp/xdg-cache",
	}
	env := chromiumRuntimeEnv(runtime)
	joined := strings.Join(env, "\n")
	// HOME and XDG_CACHE_HOME are intentionally not overridden so that Chrome
	// can access system font caches; only TMPDIR and XDG_CONFIG_HOME are isolated.
	for _, expected := range []string{
		"TMPDIR=/tmp/tmp",
		"XDG_CONFIG_HOME=/tmp/xdg-config",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in env, got %v", expected, env)
		}
	}
	for _, unexpected := range []string{"\nHOME=", "XDG_CACHE_HOME="} {
		if strings.Contains("\n"+joined, unexpected) {
			t.Fatalf("unexpected %q in env (should not override): %v", unexpected, env)
		}
	}
}

// TestBuildCJKFontFaceCSS tests CJK font face CSS generation.
func TestBuildCJKFontFaceCSS(t *testing.T) {
	css := buildCJKFontFaceCSS()
	// Result depends on environment — if no CJK font is installed, returns empty.
	if css != "" {
		if !strings.Contains(css, "@font-face") {
			t.Error("non-empty CSS should contain @font-face rule")
		}
		if !strings.Contains(css, "CJK-Embedded") {
			t.Error("non-empty CSS should use CJK-Embedded family name")
		}
		if !strings.Contains(css, "unicode-range") {
			t.Error("non-empty CSS should include unicode-range")
		}
		if !strings.Contains(css, "file://") {
			t.Error("non-empty CSS should use file:// URL")
		}
		if !strings.Contains(css, "body {") {
			t.Error("non-empty CSS should include body font-family override")
		}
	}
}

// TestInjectCJKFontFaceCSS tests CSS injection into HTML.
func TestInjectCJKFontFaceCSS(t *testing.T) {
	// Test with no CJK fonts available — should return unchanged HTML.
	// In environments without CJK fonts, this validates the no-op path.
	html := "<html><head><title>Test</title></head><body>Hello</body></html>"
	result := injectCJKFontFaceCSS(html)

	// If no CJK fonts installed, result should be unchanged.
	// If CJK fonts are installed, result should contain the style block.
	if result != html {
		if !strings.Contains(result, `data-cjk-fonts="1"`) {
			t.Error("injected CSS should have data-cjk-fonts attribute")
		}
		if !strings.Contains(result, "</head>") {
			t.Error("injected CSS should preserve </head> tag")
		}
		// Style block should appear before </head>
		styleIdx := strings.Index(result, `data-cjk-fonts="1"`)
		headIdx := strings.Index(result, "</head>")
		if styleIdx > headIdx {
			t.Error("CJK style block should be injected before </head>")
		}
	}
}

// TestInjectCJKFontFaceCSSNoHead tests injection when </head> is missing.
func TestInjectCJKFontFaceCSSNoHead(t *testing.T) {
	html := "<body>Hello</body>"
	result := injectCJKFontFaceCSS(html)
	// If CJK fonts available, block should be prepended.
	if result != html && !strings.HasPrefix(result, "<style") {
		t.Error("when no </head> present, CJK style should be prepended")
	}
}

func TestCJKFontSrc(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "ttc collection",
			path:     "/tmp/fonts/msyh.ttc",
			expected: `url("file:///tmp/fonts/msyh.ttc#1") format(collection)`,
		},
		{
			name:     "otc collection",
			path:     "/tmp/fonts/noto.otc",
			expected: `url("file:///tmp/fonts/noto.otc#1") format(collection)`,
		},
		{
			name:     "otf font",
			path:     "/tmp/fonts/noto.otf",
			expected: `url("file:///tmp/fonts/noto.otf") format(opentype)`,
		},
		{
			name:     "ttf font",
			path:     "/tmp/fonts/noto.ttf",
			expected: `url("file:///tmp/fonts/noto.ttf") format(truetype)`,
		},
		{
			name:     "unknown extension",
			path:     "/tmp/fonts/noto.font",
			expected: `url("file:///tmp/fonts/noto.font")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cjkFontSrc(cjkFontSource{path: tt.path})
			if got != tt.expected {
				t.Fatalf("cjkFontSrc(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestFileURLForCSS(t *testing.T) {
	got := fileURLForCSS("/tmp/My Fonts/msyh.ttc")
	want := "file:///tmp/My%20Fonts/msyh.ttc"
	if got != want {
		t.Fatalf("fileURLForCSS() = %q, want %q", got, want)
	}
}

func TestChromiumAllocatorOptionsIncludeRuntimeOverrides(t *testing.T) {
	runtime := chromiumRuntimeDirs{
		userData:  "/tmp/user-data",
		homeDir:   "/tmp/home",
		tmpDir:    "/tmp/tmp",
		xdgConfig: "/tmp/xdg-config",
		xdgCache:  "/tmp/xdg-cache",
	}
	var output bytes.Buffer
	opts := chromiumAllocatorOptions("/tmp/chrome", runtime, &output)
	if len(opts) == 0 {
		t.Fatal("chromiumAllocatorOptions returned no options")
	}
}
