package plantuml

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
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
		if _, err := w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"><rect width="100" height="100" fill="white"/></svg>`)); err != nil {
			t.Fatalf("failed to write mock SVG response: %v", err)
		}
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
			ctx := context.Background()
			result, err := renderer.RenderHTML(ctx, tt.html)
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
			} else if !strings.Contains(result, tt.html) {
				t.Fatal("result should contain original HTML when no plantuml found")
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
		if _, err := w.Write([]byte(`<svg></svg>`)); err != nil {
			t.Fatalf("failed to write cached mock response: %v", err)
		}
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)
	ctx := context.Background()

	code := "Alice -> Bob: Hello"

	// First call should hit the server
	svg1, err := renderer.getSVG(ctx, code)
	if err != nil {
		t.Fatalf("first getSVG failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("first call should hit server, but callCount=%d", callCount)
	}

	// Second call should use cache
	svg2, err := renderer.getSVG(ctx, code)
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
		if _, err := w.Write([]byte(`<svg width="100" height="100"></svg>`)); err != nil {
			t.Fatalf("failed to write diagram response: %v", err)
		}
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
			ctx := context.Background()
			html := `<pre><code class="language-plantuml">` + code + `</code></pre>`
			result, err := renderer.RenderHTML(ctx, html)
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
		if _, err := w.Write([]byte("Internal Server Error")); err != nil {
			t.Fatalf("failed to write error response: %v", err)
		}
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)
	ctx := context.Background()

	html := `<pre><code class="language-plantuml">Alice -> Bob</code></pre>`
	result, err := renderer.RenderHTML(ctx, html)
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
		if _, err := w.Write([]byte(`<svg></svg>`)); err != nil {
			t.Fatalf("failed to write whitespace test response: %v", err)
		}
	}))
	defer mockServer.Close()

	renderer := NewRenderer(mockServer.URL, false)
	ctx := context.Background()

	html := `<pre><code class="language-plantuml">
    Alice -> Bob: Hello

    Bob -> Alice: Hi
  </code></pre>`
	result, err := renderer.RenderHTML(ctx, html)
	if err != nil {
		t.Fatalf("rendering failed: %v", err)
	}
	if !strings.Contains(result, `class="plantuml-diagram"`) {
		t.Fatal("should handle leading/trailing whitespace")
	}
}

// TestLocalPlantumlCmdNoneAvailable verifies a helpful error is returned when
// neither PLANTUML_JAR nor a plantuml binary is available.
func TestLocalPlantumlCmdNoneAvailable(t *testing.T) {
	// Ensure PLANTUML_JAR is not set for this test.
	t.Setenv("PLANTUML_JAR", "")

	// Override PATH with an empty directory so LookPath("plantuml") fails.
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	ctx := context.Background()
	_, err := localPlantumlCmd(ctx)
	if err == nil {
		t.Fatal("expected an error when plantuml is not available")
	}
	if !strings.Contains(err.Error(), "plantuml not found") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestLocalPlantumlCmdWithJar verifies PLANTUML_JAR is respected.
func TestLocalPlantumlCmdWithJar(t *testing.T) {
	jarPath := "/opt/plantuml.jar"
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		t.Skip("plantuml.jar not found, skipping")
	}
	t.Setenv("PLANTUML_JAR", jarPath)

	ctx := context.Background()
	cmd, err := localPlantumlCmd(ctx)
	if err != nil {
		// Acceptable if java is not on PATH in this environment.
		if strings.Contains(err.Error(), "java is not in PATH") {
			t.Skip("java not available; skipping PLANTUML_JAR test")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the constructed command uses java -jar.
	if len(cmd.Args) < 3 {
		t.Fatalf("expected at least 3 args, got %v", cmd.Args)
	}
	if !strings.HasSuffix(cmd.Args[0], "java") {
		t.Fatalf("expected java executable, got %s", cmd.Args[0])
	}
	if cmd.Args[1] != "-jar" {
		t.Fatalf("expected -jar flag, got %s", cmd.Args[1])
	}
	if cmd.Args[2] != "/opt/plantuml.jar" {
		t.Fatalf("expected jar path, got %s", cmd.Args[2])
	}
}

// TestRenderLocalNotFound verifies renderLocal returns a clear error when
// plantuml is not installed.
func TestRenderLocalNotFound(t *testing.T) {
	t.Setenv("PLANTUML_JAR", "")
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	ctx := context.Background()
	r := NewRenderer("", true)
	_, err := r.renderLocal(ctx, "Alice -> Bob")
	if err == nil {
		t.Fatal("expected an error when plantuml is not available")
	}
	if !strings.Contains(err.Error(), "plantuml not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRenderLocalWithFakePlantuml runs renderLocal against a fake plantuml
// script that emits a minimal SVG, verifying the happy path without a real
// installation.
func TestRenderLocalWithFakePlantuml(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test; skipping on Windows")
	}
	// Write a tiny shell script that prints a valid SVG to stdout.
	dir := t.TempDir()
	script := dir + "/plantuml"
	scriptContent := "#!/bin/sh\necho '<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>'\n"
	if err := os.WriteFile(script, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to create fake plantuml script: %v", err)
	}

	t.Setenv("PLANTUML_JAR", "")
	t.Setenv("PATH", dir)

	ctx := context.Background()
	r := NewRenderer("", true)
	svg, err := r.renderLocal(ctx, "Alice -> Bob: Hello")
	if err != nil {
		t.Fatalf("renderLocal failed: %v", err)
	}
	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected SVG output, got: %s", svg)
	}
}
