package plantuml

import (
	"bytes"
	"compress/flate"
	"context"
	"io"
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

	// Decode using the PlantUML custom base64 alphabet.
	// Add back padding that was stripped.
	padding := (4 - len(encoded)%4) % 4
	padded := encoded + strings.Repeat("=", padding)

	raw := make([]byte, 0, len(padded)*3/4)
	for i := 0; i+3 < len(padded); i += 4 {
		b0 := plantumlDecode6bit(padded[i])
		b1 := plantumlDecode6bit(padded[i+1])
		b2 := plantumlDecode6bit(padded[i+2])
		b3 := plantumlDecode6bit(padded[i+3])
		raw = append(raw, (b0<<2)|(b1>>4))
		if padded[i+2] != '=' {
			raw = append(raw, (b1<<4)|(b2>>2))
		}
		if padded[i+3] != '=' {
			raw = append(raw, (b2<<6)|b3)
		}
	}

	// Decompress with raw deflate.
	r := flate.NewReader(bytes.NewReader(raw))
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("deflate decompression failed: %v", err)
	}

	if string(decompressed) != original {
		t.Fatalf("roundtrip failed: got %q, want %q", string(decompressed), original)
	}
}

// plantumlDecode6bit reverses plantumlEncode6bit for testing.
func plantumlDecode6bit(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'Z':
		return c - 'A' + 10
	case c >= 'a' && c <= 'z':
		return c - 'a' + 36
	case c == '-':
		return 62
	case c == '_':
		return 63
	default:
		return 0
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

	renderer := newRendererNoValidation(mockServer.URL, false)

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

	renderer := newRendererNoValidation(mockServer.URL, false)
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

	renderer := newRendererNoValidation(mockServer.URL, false)

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

	renderer := newRendererNoValidation(mockServer.URL, false)
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

	renderer := newRendererNoValidation(mockServer.URL, false)
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
	r := newRendererNoValidation("", true)
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
	r := newRendererNoValidation("", true)
	svg, err := r.renderLocal(ctx, "Alice -> Bob: Hello")
	if err != nil {
		t.Fatalf("renderLocal failed: %v", err)
	}
	if !strings.Contains(svg, "<svg") {
		t.Fatalf("expected SVG output, got: %s", svg)
	}
}

// TestSanitizePlantUMLCode tests the directive sanitizer.
func TestSanitizePlantUMLCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSafe bool // true if the output should NOT contain the directive
	}{
		{"include directive", "!include /etc/passwd", true},
		{"import directive", "!import http://evil.com/file", true},
		{"include with whitespace", "  !INCLUDE foo.puml", true},
		{"include_many", "!include_many files.txt", true},
		{"define directive", "!define MY_VAR", true},
		{"theme directive", "!theme cerulean", true},
		{"theme from url", "!theme cerulean from http://evil.com", true},
		{"pragma directive", "!pragma useVerticalIf on", true},
		{"%load_json", "%load_json(\"/etc/passwd\")", true},
		{"%getenv", "%getenv(\"SECRET\")", true},
		{"%filename", "title %filename()", true},
		{"%dirpath", "title %dirpath()", true},
		{"normal code", "Alice -> Bob: Hello", false},
		{"comment", "' this is a comment", false},
		{"participant", "participant Alice", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePlantUMLCode(tt.input)
			hasRemoval := strings.Contains(result, "[directive removed for security]")
			if tt.wantSafe && !hasRemoval {
				t.Errorf("expected directive to be removed, got: %q", result)
			} else if tt.wantSafe && strings.Contains(result, tt.input) {
				t.Errorf("original directive should not appear in output")
			} else if !tt.wantSafe {
				if hasRemoval {
					t.Errorf("safe input should not be modified, got: %q", result)
				}
				if result != tt.input {
					t.Errorf("safe input should be unchanged: want %q, got %q", tt.input, result)
				}
			}
		})
	}
}
