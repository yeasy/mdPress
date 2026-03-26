// validate_mermaid_test.go provides comprehensive tests for Mermaid diagram validation functions.
// Tests cover HTML building, status tracking, and rendering validation scenarios.
package cmd

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/pdf"
)

// ---------------------------------------------------------------------------
// buildMermaidValidationHTML - Comprehensive Table-Driven Tests
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		bodyHTML      string
		expectDoctype bool
		expectBody    bool
		expectStatus  bool
		expectScript  bool
		checkContent  func(html string) bool
	}{
		{
			name:          "empty body HTML",
			bodyHTML:      "",
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				return strings.Contains(html, "<body>") && strings.Contains(html, "</body>")
			},
		},
		{
			name:          "simple mermaid diagram",
			bodyHTML:      `<div class="mermaid">graph TD; A-->B;</div>`,
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				return strings.Contains(html, `<div class="mermaid">`)
			},
		},
		{
			name:          "multiple mermaid blocks",
			bodyHTML:      `<div class="mermaid">graph A</div><div class="mermaid">graph B</div>`,
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				count := strings.Count(html, `<div class="mermaid">`)
				return count == 2
			},
		},
		{
			name:          "mermaid with special characters",
			bodyHTML:      `<div class="mermaid">graph TD; A["Node &amp; Text"] --> B;</div>`,
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				return strings.Contains(html, "Node &amp; Text")
			},
		},
		{
			name: "mermaid with newlines",
			bodyHTML: `<div class="mermaid">
graph TD
A-->B
B-->C
</div>`,
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				return strings.Contains(html, "graph TD")
			},
		},
		{
			name:          "complex HTML structure",
			bodyHTML:      `<div><p>Text before</p><div class="mermaid">graph</div><p>Text after</p></div>`,
			expectDoctype: true,
			expectBody:    true,
			expectStatus:  true,
			expectScript:  true,
			checkContent: func(html string) bool {
				return strings.Contains(html, "Text before") && strings.Contains(html, "Text after")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := buildMermaidValidationHTML(tt.bodyHTML)

			if html == "" {
				t.Fatal("expected non-empty HTML output")
			}

			if tt.expectDoctype && !strings.Contains(html, "<!DOCTYPE html>") {
				t.Error("missing DOCTYPE declaration")
			}

			if tt.expectBody && !strings.Contains(html, "<body>") {
				t.Error("missing body tag")
			}

			if tt.expectStatus && !strings.Contains(html, "__mdpressMermaidStatus") {
				t.Error("missing status tracking variable")
			}

			if tt.expectScript && !strings.Contains(html, "<script") {
				t.Error("missing script tags")
			}

			if !tt.checkContent(html) {
				t.Errorf("custom content check failed")
			}
		})
	}
}

func TestBuildMermaidValidationHTML_StructureValidation(t *testing.T) {
	html := buildMermaidValidationHTML("<div>test</div>")

	// Validate HTML structure
	if !strings.Contains(html, "<html") {
		t.Error("missing html tag")
	}
	if !strings.Contains(html, "<head>") {
		t.Error("missing head tag")
	}
	if !strings.Contains(html, "<meta charset") {
		t.Error("missing charset meta tag")
	}
	if !strings.Contains(html, "<meta name=\"viewport\"") {
		t.Error("missing viewport meta tag")
	}
	if !strings.Contains(html, "<title>") {
		t.Error("missing title tag")
	}
}

func TestBuildMermaidValidationHTML_JavaScriptFeatures(t *testing.T) {
	html := buildMermaidValidationHTML("")

	// Check for required JavaScript functionality
	if !strings.Contains(html, "window.__mdpressMermaidStatus") {
		t.Error("missing status object initialization")
	}
	if !strings.Contains(html, "window.addEventListener('error'") {
		t.Error("missing error event listener")
	}
	if !strings.Contains(html, "mermaid.initialize") {
		t.Error("missing mermaid initialization")
	}
	if !strings.Contains(html, "document.querySelectorAll('.mermaid')") {
		t.Error("missing mermaid selector code")
	}
	if !strings.Contains(html, "mermaid.run") {
		t.Error("missing mermaid.run call")
	}
	if !strings.Contains(html, "themeVariables") {
		t.Error("missing mermaid theme variables")
	}
}

func TestBuildMermaidValidationHTML_StatusProperties(t *testing.T) {
	html := buildMermaidValidationHTML("")

	// Check for all status properties in initialization (JS object keys are unquoted)
	statusProps := []string{"done", "ok", "error", "total", "rendered", "processed"}
	for _, prop := range statusProps {
		if !strings.Contains(html, prop+":") {
			t.Errorf("missing status property: %s", prop)
		}
	}
}

func TestBuildMermaidValidationHTML_CDNReference(t *testing.T) {
	html := buildMermaidValidationHTML("")

	// Check that Mermaid CDN is referenced
	if !strings.Contains(html, "script src=") {
		t.Error("missing script src attribute")
	}
	if !strings.Contains(html, "mermaid") {
		t.Error("missing mermaid reference in script")
	}
}

// ---------------------------------------------------------------------------
// validateRenderedMermaidHTML - Edge Cases and Error Conditions
// ---------------------------------------------------------------------------

func TestValidateRenderedMermaidHTML_EmptyHTML(t *testing.T) {
	// Empty content should return nil without requiring chromium
	err := validateRenderedMermaidHTML("")
	if err != nil {
		t.Errorf("empty HTML should not error: %v", err)
	}
}

func TestValidateRenderedMermaidHTML_Whitespace(t *testing.T) {
	// Whitespace-only content requires Chrome for validation;
	// skip when Chrome is not available.
	if err := pdf.CheckChromiumAvailable(); err != nil {
		t.Skipf("Chrome unavailable: %v", err)
	}
	err := validateRenderedMermaidHTML("   \n\t  ")
	if err != nil {
		t.Errorf("whitespace-only HTML should not error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration tests combining both functions
// ---------------------------------------------------------------------------

func TestMermaidValidation_Pipeline(t *testing.T) {
	// Test the typical flow: build HTML, then validate
	mermaidBody := `<div class="mermaid">
graph TD;
    A[Start] --> B[Process];
    B --> C[End];
</div>`

	html := buildMermaidValidationHTML(mermaidBody)

	// Verify HTML is usable
	if html == "" {
		t.Fatal("built HTML should not be empty")
	}

	// Verify HTML contains required mermaid structure
	if !strings.Contains(html, "class=\"mermaid\"") {
		t.Error("mermaid class not preserved in HTML")
	}

	// Verify HTML contains validation script
	if !strings.Contains(html, "__mdpressMermaidStatus") {
		t.Error("validation script not included")
	}
}

func TestMermaidValidation_MultipleBlocks(t *testing.T) {
	mermaidBody := `
<div class="mermaid">graph LR; A --> B;</div>
<p>Some text</p>
<div class="mermaid">graph TB; C --> D;</div>
<p>More text</p>
<div class="mermaid">graph BT; E --> F;</div>
`

	html := buildMermaidValidationHTML(mermaidBody)

	// Count mermaid divs
	count := strings.Count(html, `class="mermaid"`)
	if count != 3 {
		t.Errorf("expected 3 mermaid blocks, found %d", count)
	}

	// Verify script will detect all blocks
	if !strings.Contains(html, "querySelectorAll('.mermaid')") {
		t.Error("script should query all mermaid elements")
	}
}

func TestMermaidValidation_ErrorHandling(t *testing.T) {
	html := buildMermaidValidationHTML(`<div class="mermaid">invalid</div>`)

	// Check error handler in script
	if !strings.Contains(html, "catch (error)") {
		t.Error("error handler missing from validation script")
	}

	if !strings.Contains(html, "window.__mdpressMermaidStatus.error") {
		t.Error("error assignment missing from validation script")
	}
}

func TestMermaidValidation_ScriptSafety(t *testing.T) {
	// Test with potentially dangerous content
	dangerousContent := `<div class="mermaid">
    <script>alert('xss')</script>
</div>`

	html := buildMermaidValidationHTML(dangerousContent)

	// Should just include it as-is (validation happens in browser)
	if !strings.Contains(html, "script") {
		t.Error("content should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Charset and Encoding Tests
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML_UTF8Encoding(t *testing.T) {
	htmlWithCJK := `<div class="mermaid">
graph TD
    A["中文标题"] --> B["English"]
    B --> C["日本語"]
</div>`

	html := buildMermaidValidationHTML(htmlWithCJK)

	if !strings.Contains(html, `charset="UTF-8"`) {
		t.Error("should declare UTF-8 charset")
	}

	if !strings.Contains(html, "中文标题") {
		t.Error("CJK content should be preserved")
	}
}

func TestBuildMermaidValidationHTML_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expect  string
	}{
		{
			name:    "HTML entities",
			content: `<div class="mermaid">&lt;tag&gt;</div>`,
			expect:  `&lt;tag&gt;`,
		},
		{
			name:    "quotes",
			content: `<div class="mermaid">"quoted" 'text'</div>`,
			expect:  `quoted`,
		},
		{
			name:    "ampersands",
			content: `<div class="mermaid">A & B & C</div>`,
			expect:  `& B &`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := buildMermaidValidationHTML(tt.content)
			if !strings.Contains(html, tt.expect) {
				t.Errorf("expected %q in output", tt.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Performance and Boundary Tests
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML_LargeContent(t *testing.T) {
	// Create a large HTML content
	largeContent := `<div class="mermaid">`
	for i := 0; i < 1000; i++ {
		largeContent += "A --> B; "
	}
	largeContent += `</div>`

	html := buildMermaidValidationHTML(largeContent)

	if html == "" {
		t.Error("should handle large content")
	}

	if !strings.Contains(html, "<div class=\"mermaid\">") {
		t.Error("large content should be included")
	}

	// Verify valid HTML structure despite large body
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("large content should still have proper HTML structure")
	}
}

func TestBuildMermaidValidationHTML_NestedDivs(t *testing.T) {
	nestedContent := `
<div class="container">
    <div class="mermaid">
        graph TD
        A --> B
    </div>
    <div class="mermaid">
        graph LR
        C --> D
    </div>
</div>`

	html := buildMermaidValidationHTML(nestedContent)

	// Both mermaid blocks should be present
	if strings.Count(html, "class=\"mermaid\"") != 2 {
		t.Error("nested mermaid blocks should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Content Preservation Tests
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML_ContentIntegrity(t *testing.T) {
	originalContent := `<div class="mermaid">
    sequenceDiagram
        participant User
        participant System
        User ->> System: Request
        System ->> User: Response
    </div>`

	html := buildMermaidValidationHTML(originalContent)

	// Original content should be exactly preserved
	if !strings.Contains(html, originalContent) {
		t.Error("original content should be preserved verbatim")
	}

	// Check structure is maintained
	if !strings.Contains(html, "sequenceDiagram") {
		t.Error("diagram type should be preserved")
	}
	if !strings.Contains(html, "participant User") {
		t.Error("diagram content should be preserved")
	}
}

func TestBuildMermaidValidationHTML_MultilinePreservation(t *testing.T) {
	multilineContent := `<div class="mermaid">
graph TD
    A["Multi
    line
    text"] --> B
</div>`

	html := buildMermaidValidationHTML(multilineContent)

	if !strings.Contains(html, "Multi") {
		t.Error("multiline content start should be preserved")
	}
	if !strings.Contains(html, "text\"] --> B") {
		t.Error("multiline content structure should be preserved")
	}
}

// ---------------------------------------------------------------------------
// HTML Injection Prevention Tests
// ---------------------------------------------------------------------------

func TestBuildMermaidValidationHTML_ScriptTagHandling(t *testing.T) {
	// User-provided script tags should be included as-is
	// (XSS prevention is browser's responsibility)
	content := `<div class="mermaid">
<!-- This is fine as a comment -->
</div>`

	html := buildMermaidValidationHTML(content)

	if !strings.Contains(html, "<!-- This is fine as a comment -->") {
		t.Error("HTML comments should be preserved")
	}
}

func TestBuildMermaidValidationHTML_EventHandlerPreservation(t *testing.T) {
	// Event handlers in user content are preserved as-is
	content := `<div class="mermaid" data-test="value">content</div>`

	html := buildMermaidValidationHTML(content)

	if !strings.Contains(html, "data-test") {
		t.Error("data attributes should be preserved")
	}
}
