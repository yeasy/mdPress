package markdown

import (
	"strings"
	"testing"
)

const rawHTMLSource = `# Chapter

<script>alert(1)</script>

<iframe src="https://evil.example"></iframe>

Inline <img src=x onerror="alert(2)"> here.
`

// Raw HTML passes through by default: books rely on inline HTML for layout
// goldmark cannot express, and turning that off would break existing projects.
func TestParserAllowsRawHTMLByDefault(t *testing.T) {
	html, _, err := NewParser().Parse([]byte(rawHTMLSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, want := range []string{"<script>alert(1)</script>", "<iframe", "onerror"} {
		if !strings.Contains(html, want) {
			t.Errorf("default parser dropped %q from the output:\n%s", want, html)
		}
	}
}

// WithAllowHTML(false) is the opt-in for projects that render Markdown they
// did not write; no tag that can execute may survive.
func TestParserWithAllowHTMLFalseDropsRawHTML(t *testing.T) {
	html, _, err := NewParser(WithAllowHTML(false)).Parse([]byte(rawHTMLSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, forbidden := range []string{"<script", "<iframe", "onerror"} {
		if strings.Contains(html, forbidden) {
			t.Errorf("allow_html=false left %q in the output:\n%s", forbidden, html)
		}
	}
	// Ordinary Markdown must still render.
	if !strings.Contains(html, "Chapter") {
		t.Errorf("allow_html=false lost the chapter heading:\n%s", html)
	}
}
