package pdf

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildCJKFontFaceEdgeCases tests edge cases of CJK font face CSS generation
func TestBuildCJKFontFaceEdgeCases(t *testing.T) {
	result := buildCJKFontFaceCSS()

	// CSS is either empty (no fonts found) or contains expected structure
	if result.css == "" {
		t.Skip("No CJK fonts found on this system, skipping buildCJKFontFaceCSS structure test")
	}

	if result.family == "" {
		t.Error("family should not be empty when CSS is present")
	}
	if result.fontPath == "" {
		t.Error("fontPath should not be empty when CSS is present")
	}

	// When CSS is not empty, it should contain @font-face
	if !strings.Contains(result.css, "@font-face") {
		t.Error("CSS should contain @font-face rule when fonts are available")
	}

	// Should contain unicode-range
	if !strings.Contains(result.css, "unicode-range") {
		t.Error("CSS should contain unicode-range")
	}

	// Should contain body font-family override
	if !strings.Contains(result.css, "body") {
		t.Error("CSS should contain body styling")
	}
}

// TestInjectCJKFontFaceHeadInjection tests CSS injection before </head>
func TestInjectCJKFontFaceHeadInjection(t *testing.T) {
	html := "<html><head><title>Test</title></head><body>content</body></html>"
	result := injectCJKFontFaceCSS(html, nil)

	// </head> should still be in the result
	if !strings.Contains(result, "</head>") {
		t.Fatal("</head> tag should be preserved")
	}

	headIdx := strings.Index(result, "</head>")
	styleIdx := strings.Index(result, "<style data-cjk-fonts")

	if styleIdx == -1 {
		t.Skip("No CJK fonts found on this system, skipping injection position test")
	}

	if styleIdx >= headIdx {
		t.Errorf("CJK style (pos %d) should be before </head> (pos %d)", styleIdx, headIdx)
	}
}

// TestInjectCJKFontFacePrependFallback tests CSS prepending when no </head>
func TestInjectCJKFontFacePrependFallback(t *testing.T) {
	html := "<body>content</body>"
	result := injectCJKFontFaceCSS(html, nil)

	switch {
	case strings.HasPrefix(result, "<style data-cjk-fonts"):
		// Fonts available: verify original HTML is preserved after the injected style.
		if !strings.Contains(result, html) {
			t.Errorf("original HTML should be preserved in result, got %q", result)
		}
	case result == html:
		t.Skip("No CJK fonts found on this system, skipping prepend fallback test")
	default:
		t.Errorf("unexpected result: should either prepend <style data-cjk-fonts or return original HTML, got %q", result)
	}
}

// TestFileURLForCSSFormat tests file URL format generation
func TestFileURLForCSSFormat(t *testing.T) {
	result := fileURLForCSS("/path/to/font.ttf")

	if !strings.HasPrefix(result, "file://") {
		t.Errorf("File URL should start with file://, got %q", result)
	}

	// Should have proper slashes
	if strings.Count(result, "//") < 1 {
		t.Error("File URL format incorrect")
	}
}

// TestCJKFontSrcRelativeURL tests that cjkFontSrc returns a relative URL with
// format() hint for the local HTTP server approach.
func TestCJKFontSrcRelativeURL(t *testing.T) {
	src := cjkFontSource{path: "/any/font.ttc"}
	result := cjkFontSrc(src)

	expected := `url("/cjk-font") format("collection")`
	if result != expected {
		t.Errorf("cjkFontSrc() = %q, want %q", result, expected)
	}
}

// TestCJKFontSrcFallback tests that the fallback function produces file:// URL
func TestCJKFontSrcFallback(t *testing.T) {
	src := cjkFontSource{path: "/nonexistent/font.ttc"}
	result := cjkFontSrcFallback(src)

	if !strings.Contains(result, "file://") {
		t.Errorf("Expected file:// URL, got %q", result)
	}
	if !strings.Contains(result, "url(") {
		t.Error("Should contain url() wrapper")
	}
}

// TestFontServer tests that the font server starts and serves content
func TestFontServer(t *testing.T) {
	srv, err := newFontServer("<html><body>test</body></html>", "")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if srv.baseURL == "" {
		t.Error("baseURL should not be empty")
	}
	if !strings.HasPrefix(srv.baseURL, "http://127.0.0.1:") {
		t.Errorf("baseURL should start with http://127.0.0.1:, got %q", srv.baseURL)
	}
}

func TestFontServerServesHTML(t *testing.T) {
	htmlContent := "<html><body>Hello World</body></html>"
	srv, err := newFontServer(htmlContent, "")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	client := &http.Client{}

	// GET / should return the HTML content.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.baseURL+"/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	if string(body) != htmlContent {
		t.Errorf("body = %q, want %q", string(body), htmlContent)
	}

	// GET /other should return 404.
	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.baseURL+"/other", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("GET /other failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("GET /other status = %d, want 404", resp2.StatusCode)
	}
}

func TestFontServerServesFontMIME(t *testing.T) {
	tests := []struct {
		ext      string
		wantMIME string
	}{
		{".ttf", "font/ttf"},
		{".otf", "font/otf"},
		{".woff", "font/woff"},
		{".woff2", "font/woff2"},
		{".ttc", "font/collection"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			tmpDir := t.TempDir()
			fontPath := filepath.Join(tmpDir, "test"+tt.ext)
			if err := os.WriteFile(fontPath, []byte("fake-font-data"), 0o644); err != nil {
				t.Fatal(err)
			}

			srv, err := newFontServer("<html></html>", fontPath)
			if err != nil {
				t.Fatal(err)
			}
			defer srv.Close()

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.baseURL+"/cjk-font", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			resp, err := (&http.Client{}).Do(req)
			if err != nil {
				t.Fatalf("GET /cjk-font failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("GET /cjk-font status = %d, want 200", resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, tt.wantMIME) {
				t.Errorf("Content-Type = %q, want prefix %q", ct, tt.wantMIME)
			}
		})
	}
}

func TestFontServerCloseIdempotent(t *testing.T) {
	srv, err := newFontServer("<html></html>", "")
	if err != nil {
		t.Fatal(err)
	}
	srv.Close()
	srv.Close() // second close should not panic
}

// ---------------------------------------------------------------------------
// Generator option setters
// ---------------------------------------------------------------------------

func TestWithFooterTemplate(t *testing.T) {
	tmpl := `<span class="pageNumber"></span>`
	g := NewGenerator(WithFooterTemplate(tmpl))
	if g.footerTemplate != tmpl {
		t.Errorf("footerTemplate = %q, want %q", g.footerTemplate, tmpl)
	}
	if !g.displayHeaderFooter {
		t.Error("displayHeaderFooter should be true when footer template is set")
	}
}

type warnRecorder struct{ msgs []string }

func (w *warnRecorder) Warn(msg string, _ ...any) { w.msgs = append(w.msgs, msg) }

func TestWarnIfCJKFontsMissing_NoCJK(t *testing.T) {
	rec := &warnRecorder{}
	WarnIfCJKFontsMissing("Hello world, this is plain ASCII text.", rec)
	if len(rec.msgs) != 0 {
		t.Errorf("expected no warnings for ASCII-only text, got %d: %v", len(rec.msgs), rec.msgs)
	}
}

func TestWarnIfCJKFontsMissing_WithCJK(t *testing.T) {
	rec := &warnRecorder{}
	WarnIfCJKFontsMissing("这是一段中文内容，用于测试 CJK 字体检测。", rec)
	// On systems with CJK fonts: no warning. On systems without: exactly one warning.
	if len(rec.msgs) > 1 {
		t.Errorf("expected at most 1 warning, got %d: %v", len(rec.msgs), rec.msgs)
	}
	if len(rec.msgs) == 1 && !strings.Contains(rec.msgs[0], "CJK") {
		t.Errorf("warning should mention CJK, got %q", rec.msgs[0])
	}
}
