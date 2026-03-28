package utils

import (
	"strings"
	"testing"
)

func TestResolveCDNPlaceholders(t *testing.T) {
	t.Run("replaces CDN placeholders", func(t *testing.T) {
		input := `<script src="{{MERMAID_CDN_URL}}"></script>` +
			`<link href="{{KATEX_CSS_URL}}">` +
			`<script src="{{KATEX_JS_URL}}"></script>` +
			`<script src="{{KATEX_AUTO_RENDER_URL}}"></script>`

		got := ResolveCDNPlaceholders(input)

		if got == input {
			t.Fatal("expected placeholders to be replaced, but output is identical to input")
		}
		for _, tc := range []struct {
			name, url string
		}{
			{"MermaidCDNURL", MermaidCDNURL},
			{"KaTeXCSSURL", KaTeXCSSURL},
			{"KaTeXJSURL", KaTeXJSURL},
			{"KaTeXAutoRenderURL", KaTeXAutoRenderURL},
		} {
			if !strings.Contains(got, tc.url) {
				t.Errorf("expected output to contain %s (%s)", tc.name, tc.url)
			}
		}
	})

	t.Run("returns string unchanged when no placeholders", func(t *testing.T) {
		input := `<p>Hello, world!</p>`
		got := ResolveCDNPlaceholders(input)
		if got != input {
			t.Errorf("expected %q, got %q", input, got)
		}
	})

	t.Run("returns empty string for empty input", func(t *testing.T) {
		got := ResolveCDNPlaceholders("")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("replaces only the matching placeholder among mixed text", func(t *testing.T) {
		input := `before {{MERMAID_CDN_URL}} after`
		got := ResolveCDNPlaceholders(input)
		if !strings.Contains(got, MermaidCDNURL) {
			t.Errorf("expected mermaid URL in output, got %q", got)
		}
		if strings.Contains(got, "{{MERMAID_CDN_URL}}") {
			t.Error("placeholder should have been replaced")
		}
		if !strings.HasPrefix(got, "before ") || !strings.HasSuffix(got, " after") {
			t.Errorf("surrounding text should be preserved, got %q", got)
		}
	})

	t.Run("replaces multiple occurrences of same placeholder", func(t *testing.T) {
		input := `{{KATEX_JS_URL}} and {{KATEX_JS_URL}}`
		got := ResolveCDNPlaceholders(input)
		if count := strings.Count(got, KaTeXJSURL); count != 2 {
			t.Errorf("expected 2 occurrences of KaTeX JS URL, got %d in %q", count, got)
		}
	})

	t.Run("leaves unknown placeholders untouched", func(t *testing.T) {
		input := `{{UNKNOWN_PLACEHOLDER}}`
		got := ResolveCDNPlaceholders(input)
		if got != input {
			t.Errorf("unknown placeholder should be untouched, got %q", got)
		}
	})

	t.Run("CDN URLs contain expected version strings", func(t *testing.T) {
		if !strings.Contains(MermaidCDNURL, "mermaid") {
			t.Error("MermaidCDNURL should contain 'mermaid'")
		}
		if !strings.Contains(KaTeXCSSURL, "katex") {
			t.Error("KaTeXCSSURL should contain 'katex'")
		}
		if !strings.Contains(KaTeXJSURL, "katex") {
			t.Error("KaTeXJSURL should contain 'katex'")
		}
		if !strings.Contains(KaTeXAutoRenderURL, "auto-render") {
			t.Error("KaTeXAutoRenderURL should contain 'auto-render'")
		}
	})
}

func TestGetPageDimensions(t *testing.T) {
	tests := []struct {
		name         string
		size         string
		wantWidth    float64
		wantHeight   float64
		wantWidthMM  string
		wantHeightMM string
	}{
		{"A4", "A4", 210, 297, "210mm", "297mm"},
		{"A4 lowercase", "a4", 210, 297, "210mm", "297mm"},
		{"A5", "A5", 148, 210, "148mm", "210mm"},
		{"Letter", "Letter", 216, 279, "216mm", "279mm"},
		{"Legal", "LEGAL", 216, 356, "216mm", "356mm"},
		{"B5", "B5", 176, 250, "176mm", "250mm"},
		{"unknown defaults to A4", "Unknown", 210, 297, "210mm", "297mm"},
		{"empty defaults to A4", "", 210, 297, "210mm", "297mm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := GetPageDimensions(tt.size)
			if d.Width != tt.wantWidth {
				t.Errorf("Width = %v, want %v", d.Width, tt.wantWidth)
			}
			if d.Height != tt.wantHeight {
				t.Errorf("Height = %v, want %v", d.Height, tt.wantHeight)
			}
			if d.WidthMM() != tt.wantWidthMM {
				t.Errorf("WidthMM() = %q, want %q", d.WidthMM(), tt.wantWidthMM)
			}
			if d.HeightMM() != tt.wantHeightMM {
				t.Errorf("HeightMM() = %q, want %q", d.HeightMM(), tt.wantHeightMM)
			}
		})
	}
}
