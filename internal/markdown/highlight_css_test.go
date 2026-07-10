package markdown

import (
	"strings"
	"testing"
)

// TestResolveCodeTheme tests the code theme name resolution and fallback.
func TestResolveCodeTheme(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"github", "github"},
		{"GitHub", "github"},
		{" monokai ", "monokai"},
		{"dracula", "dracula"},
		{"github-dark", "github-dark"},
		{"default", "github"}, // legacy value, not a chroma style
		{"", "github"},
		{"no-such-style", "github"},
	}
	for _, tt := range tests {
		if got := resolveCodeTheme(tt.in); got != tt.want {
			t.Errorf("resolveCodeTheme(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestDarkCodeTheme tests the light->dark counterpart mapping.
func TestDarkCodeTheme(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"github", "github-dark"},
		{"default", "github-dark"},
		{"", "github-dark"},
		{"monokai", "monokai"},
		{"dracula", "dracula"},
		{"solarized-light", "github-dark"},
		{"no-such-style", "github-dark"},
	}
	for _, tt := range tests {
		if got := darkCodeTheme(tt.in); got != tt.want {
			t.Errorf("darkCodeTheme(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestHighlightCSSLight tests the light-mode stylesheet generation.
func TestHighlightCSSLight(t *testing.T) {
	css := HighlightCSSLight("github")
	if css == "" {
		t.Fatal("light CSS for github should not be empty")
	}
	if !strings.Contains(css, ".chroma") {
		t.Error("light CSS should be scoped to .chroma")
	}
	if strings.Contains(css, "style=") {
		t.Error("light CSS should not contain inline style attributes")
	}
	if strings.Contains(css, `[data-theme="dark"]`) || strings.Contains(css, "html.dark") {
		t.Error("light CSS must not carry dark-mode prefixes")
	}
	// The style's mode class must be stripped so rules match .chroma directly.
	if strings.Contains(css, ".chroma.light") || strings.Contains(css, ".chroma.dark") {
		t.Error("light CSS should not keep chroma mode classes in selectors")
	}
	// The standalone-page .bg helper rules should be dropped.
	for _, line := range strings.Split(css, "\n") {
		if line != "" && !strings.HasPrefix(line, ".chroma") {
			t.Errorf("unexpected non-.chroma rule in light CSS: %q", line)
		}
	}
}

// TestHighlightCSSLightFallback tests that unknown themes fall back to github.
func TestHighlightCSSLightFallback(t *testing.T) {
	github := HighlightCSSLight("github")
	for _, name := range []string{"", "default", "no-such-style"} {
		if got := HighlightCSSLight(name); got != github {
			t.Errorf("HighlightCSSLight(%q) should fall back to the github stylesheet", name)
		}
	}
}

// TestHighlightCSSDark tests the dark-mode stylesheet generation.
func TestHighlightCSSDark(t *testing.T) {
	css := HighlightCSSDark("github")
	if css == "" {
		t.Fatal("dark CSS for github should not be empty")
	}
	if css == HighlightCSSLight("github") {
		t.Error("dark CSS should differ from light CSS for github")
	}
	for _, line := range strings.Split(strings.TrimRight(css, "\n"), "\n") {
		for _, prefix := range DarkModeSelectors {
			if !strings.Contains(line, prefix+" .chroma") {
				t.Errorf("dark rule missing prefix %q: %q", prefix, line)
			}
		}
	}
}

// TestHighlightCSSDarkCounterparts tests that dark styles keep themselves and
// light styles use their github-dark counterpart.
func TestHighlightCSSDarkCounterparts(t *testing.T) {
	if HighlightCSSDark("monokai") == HighlightCSSDark("github") {
		t.Error("monokai dark CSS should come from monokai, not github-dark")
	}
	if HighlightCSSDark("default") != HighlightCSSDark("github") {
		t.Error(`legacy "default" theme should map to the github-dark stylesheet`)
	}
}

// TestScopeChromaRules tests rule filtering and prefixing.
func TestScopeChromaRules(t *testing.T) {
	css := ".bg { color: #000 }\n.chroma { color: #111 }\n.chroma .kd { color: #222 }\n"
	got := scopeChromaRules(css, nil)
	if strings.Contains(got, ".bg") {
		t.Error(".bg rules should be dropped")
	}
	if !strings.Contains(got, ".chroma { color: #111 }") {
		t.Errorf(".chroma rule should be kept, got %q", got)
	}

	scoped := scopeChromaRules(css, []string{"html.x", "html.y"})
	if !strings.Contains(scoped, "html.x .chroma .kd, html.y .chroma .kd { color: #222 }") {
		t.Errorf("prefixed rule missing, got %q", scoped)
	}
}
