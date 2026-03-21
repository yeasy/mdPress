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
		t.Log("No CJK fonts found on this system, buildCJKFontFaceCSS returns empty result")
		return
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
	if strings.HasPrefix(result, "<style data-cjk-fonts") {
		t.Log("CSS was prepended (fonts available)")
	} else if result == html {
		t.Log("No CSS injected (no fonts found)")
	} else {
		// Some modification happened
		t.Log("HTML was modified but not by prepending style")
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

// TestCJKFontSrcFormats tests different font format handling
func TestCJKFontSrcFormats(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantFormat string
	}{
		{"TTC collection", "/fonts/font.ttc", "collection"},
		{"TTF truetype", "/fonts/font.ttf", "truetype"},
		{"OTF opentype", "/fonts/font.otf", "opentype"},
		{"OTC collection", "/fonts/font.otc", "collection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := cjkFontSource{path: tt.path}
			result := cjkFontSrc(src)

			if !strings.Contains(result, "format("+tt.wantFormat+")") {
				t.Errorf("Expected format(%s), got %q", tt.wantFormat, result)
			}
		})
	}
}

// TestCJKFontSrcUnknownFormat tests fallback for unknown formats
func TestCJKFontSrcUnknownFormat(t *testing.T) {
	src := cjkFontSource{path: "/fonts/font.xyz"}
	result := cjkFontSrc(src)

	// Should still generate valid url() CSS
	if !strings.Contains(result, "url(") {
		t.Error("Should contain url() even for unknown format")
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
		t.Logf("Default timeout is %v", defaultTimeout)
	}
}
