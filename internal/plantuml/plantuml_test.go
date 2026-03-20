package plantuml

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestEncodeForServer tests the encoding of PlantUML code for the server.
func TestEncodeForServer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We just check that it's not empty and uses valid chars
	}{
		{
			name:  "simple sequence diagram",
			input: "Alice -> Bob: Hello",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "multiline diagram",
			input: "graph TD\n  A --> B\n  B --> C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := encodeForServer(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check that encoding is not empty (unless input was empty)
			if tt.input != "" && encoded == "" {
				t.Fatal("encoded string is empty")
			}

			// Verify the encoding only contains valid PlantUML alphabet characters
			validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
			for _, ch := range encoded {
				if !strings.ContainsRune(validChars, ch) {
					t.Fatalf("encoded string contains invalid character: %c", ch)
				}
			}
		})
	}
}

// TestEncodeForServerRoundtrip verifies that we can decode what we encode.
func TestEncodeForServerRoundtrip(t *testing.T) {
	original := "Alice -> Bob: Hello\nBob -> Alice: World"

	encoded, err := encodeForServer(original)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	// Decode it back
	// Add back the padding that was removed
	padding := (4 - len(encoded)%4) % 4
	paddedEncoded := encoded + strings.Repeat("=", padding)

	// Reverse the character replacements
	reversedEncoded := strings.NewReplacer(
		"-", "+",
		"_", "/",
	).Replace(paddedEncoded)

	decoded, err := base64.StdEncoding.DecodeString(reversedEncoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	// Decompress with zlib
	r, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		t.Fatalf("zlib reader creation failed: %v", err)
	}
	defer r.Close()

	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(r); err != nil {
		t.Fatalf("zlib decompression failed: %v", err)
	}

	if decompressed.String() != original {
		t.Fatalf("roundtrip failed: got %q, want %q", decompressed.String(), original)
	}
}

// TestRenderHTML tests the rendering of PlantUML code blocks in HTML.
func TestRenderHTML(t *testing.T) {
	// Create a mock HTTP server that returns valid SVG
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect width="100" height="100" fill="white"/></svg>`))
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)

	tests := []struct {
		name     string
		html     string
		wantSVG  bool
		wantDiag bool
	}{
		{
			name: "simple plantuml block",
			html: `<p>Here's a diagram:</p>
<pre><code class="language-plantuml">Alice -> Bob: Hello</code></pre>
<p>End of diagram.</p>`,
			wantSVG:  true,
			wantDiag: true,
		},
		{
			name:     "with chroma highlighting",
			html:     `<pre><code class="highlight language-plantuml"><span>Alice -> Bob: Hello</span></code></pre>`,
			wantSVG:  true,
			wantDiag: true,
		},
		{
			name:     "escaped HTML entities",
			html:     `<pre><code class="language-plantuml">&lt;message&gt; &amp; &quot;text&quot;</code></pre>`,
			wantSVG:  true,
			wantDiag: true,
		},
		{
			name:     "no plantuml block",
			html:     `<p>Just text</p>`,
			wantSVG:  false,
			wantDiag: false,
		},
		{
			name: "multiple blocks",
			html: `<pre><code class="language-plantuml">A -> B</code></pre>
<p>More text</p>
<pre><code class="language-plantuml">C -> D</code></pre>`,
			wantSVG:  true,
			wantDiag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.RenderHTML(tt.html)
			if err != nil && tt.wantSVG {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantDiag {
				if !strings.Contains(result, `class="plantuml-diagram"`) {
					t.Fatal("result should contain plantuml-diagram div")
				}
				if tt.wantSVG && !strings.Contains(result, "<svg") {
					t.Fatal("result should contain SVG element")
				}
			} else {
				if !strings.Contains(result, tt.html) {
					t.Fatal("result should contain original HTML when no plantuml found")
				}
			}
		})
	}
}

// TestUnescapeHTML tests HTML entity unescaping.
func TestUnescapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"&lt;tag&gt;", "<tag>"},
		{"&amp;&quot;", `&"`},
		{"&#39;quote&#39;", "'quote'"},
		{"no &entities; here", "no &entities; here"},
		{"mixed &lt; and &gt; and &amp;", "mixed < and > and &"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := unescapeHTML(tt.input)
			if result != tt.expected {
				t.Fatalf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestNeedsPlantuml tests the detection of PlantUML content.
func TestNeedsPlantuml(t *testing.T) {
	tests := []struct {
		html  string
		found bool
	}{
		{`<div class="plantuml-diagram"><svg></svg></div>`, true},
		{`<code class="language-plantuml">text</code>`, true},
		{`<p>Just text</p>`, false},
		{`<code class="language-mermaid">text</code>`, false},
		{`<pre><code class="language-python">code</code></pre>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.html, func(t *testing.T) {
			result := NeedsPlantuml(tt.html)
			if result != tt.found {
				t.Fatalf("got %v, want %v", result, tt.found)
			}
		})
	}
}

// TestCaching tests that rendered SVGs are cached.
func TestCaching(t *testing.T) {
	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<svg></svg>`))
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)

	code := "Alice -> Bob: Hello"

	// First call should hit the server
	svg1, err := renderer.getSVG(code)
	if err != nil {
		t.Fatalf("first getSVG failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("first call should hit server, but callCount=%d", callCount)
	}

	// Second call should use cache
	svg2, err := renderer.getSVG(code)
	if err != nil {
		t.Fatalf("second getSVG failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("second call should use cache, but callCount=%d", callCount)
	}

	// Both should return the same SVG
	if svg1 != svg2 {
		t.Fatal("cached SVG should be identical")
	}
}

// TestVariousDiagramTypes tests different PlantUML diagram types.
func TestVariousDiagramTypes(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<svg width="100" height="100"></svg>`))
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)

	diagrams := map[string]string{
		"sequence": `Alice -> Bob: Hello
Bob -> Alice: Hi`,
		"class": `class MyClass {
  - field1: String
  + method1()
}`,
		"activity": `(*) --> Decision
Decision --> (end)`,
		"state": `[*] --> State1
State1 --> State2`,
	}

	for name, code := range diagrams {
		t.Run(name, func(t *testing.T) {
			html := `<pre><code class="language-plantuml">` + code + `</code></pre>`
			result, err := renderer.RenderHTML(html)
			if err != nil {
				t.Fatalf("rendering failed: %v", err)
			}
			if !strings.Contains(result, `class="plantuml-diagram"`) {
				t.Fatal("should contain plantuml-diagram class")
			}
			if !strings.Contains(result, `<svg`) {
				t.Fatal("should contain SVG")
			}
		})
	}
}

// TestServerError tests error handling when the PlantUML server is unavailable.
func TestServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)

	html := `<pre><code class="language-plantuml">Alice -> Bob</code></pre>`
	result, err := renderer.RenderHTML(html)
	// RenderHTML returns an error when the server fails, but the content
	// is returned unchanged
	if err == nil {
		t.Fatal("RenderHTML should return an error when server fails")
	}
	// When there's an error, the original HTML should be returned
	if !strings.Contains(result, `language-plantuml`) {
		t.Fatal("should return original HTML on server error")
	}
}

// TestWhitespaceHandling tests that whitespace is properly trimmed.
func TestWhitespaceHandling(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<svg></svg>`))
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)

	html := `<pre><code class="language-plantuml">
    Alice -> Bob: Hello

    Bob -> Alice: Hi
  </code></pre>`
	result, err := renderer.RenderHTML(html)
	if err != nil {
		t.Fatalf("rendering failed: %v", err)
	}
	if !strings.Contains(result, `class="plantuml-diagram"`) {
		t.Fatal("should handle leading/trailing whitespace")
	}
}
