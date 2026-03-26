package pdf

import (
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

	t.Logf("Found CJK font: family=%s path=%s", result.family, result.fontPath)

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
		t.Error("</head> tag should be preserved")
	}

	// If CSS was injected, it should be before </head>
	if headIdx := strings.Index(result, "</head>"); headIdx != -1 {
		if styleIdx := strings.Index(result, "<style data-cjk-fonts"); styleIdx != -1 && styleIdx < headIdx {
			// Good - style is before head
		} else if styleIdx != -1 {
			t.Error("CSS should be injected before </head>")
		}
	}
}

// TestInjectCJKFontFacePrependFallback tests CSS prepending when no </head>
func TestInjectCJKFontFacePrependFallback(t *testing.T) {
	html := "<body>content</body>"
	result := injectCJKFontFaceCSS(html, nil)

	// Either no CSS (no fonts), or CSS is prepended
	switch {
	case strings.HasPrefix(result, "<style data-cjk-fonts"):
		// Fonts available: result should be non-empty and start with the style tag
		if result == "" {
			t.Error("result should be non-empty when CJK fonts are available")
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

// TestPageSizeAndMarginConstants tests that page size constants are correct
func TestPageSizeAndMarginConstants(t *testing.T) {
	// A4 dimensions in millimeters
	if defaultPageWidth != 210 || defaultPageHeight != 297 {
		t.Errorf("A4 dimensions should be 210x297, got %.0fx%.0f", defaultPageWidth, defaultPageHeight)
	}

	if defaultMargin != 20 {
		t.Errorf("Default margin should be 20mm, got %.0f", defaultMargin)
	}

	if defaultTimeout.String() != "1m0s" {
		t.Errorf("Default timeout should be 1m0s, got %v", defaultTimeout)
	}
}
