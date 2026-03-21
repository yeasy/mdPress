// Package plantuml provides rendering of PlantUML diagrams in Markdown content.
//
// It detects ```plantuml code blocks in HTML content and converts them to SVG
// using either the PlantUML online server or a local plantuml command.
package plantuml

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// maxResponseSize limits PlantUML server responses to 10 MB to prevent DoS.
	maxResponseSize = 10 * 1024 * 1024
	// plantumlHTTPTimeout is the timeout for HTTP requests to the PlantUML server.
	plantumlHTTPTimeout = 30 * time.Second
)

// Renderer handles the detection and conversion of PlantUML diagrams.
type Renderer struct {
	// serverURL is the base URL for the PlantUML server (e.g. "http://www.plantuml.com/plantuml")
	serverURL string
	// useLocal determines whether to use a local plantuml command instead of the server
	useLocal bool
	// cache stores rendered SVGs to avoid repeated network calls
	cache sync.Map // map[string]string - key is plantuml code, value is SVG HTML
	// httpClient is used for making requests to the PlantUML server
	httpClient *http.Client
}

// NewRenderer creates a new PlantUML renderer.
// serverURL should be the base URL without the /svg path (e.g. "http://www.plantuml.com/plantuml")
func NewRenderer(serverURL string, useLocal bool) *Renderer {
	if serverURL == "" {
		serverURL = "http://www.plantuml.com/plantuml"
	}
	return &Renderer{
		serverURL: strings.TrimSuffix(serverURL, "/"),
		useLocal:  useLocal,
		// HTTP timeout for PlantUML server requests. This balances network
		// latency and rendering time for typical diagrams while preventing indefinite hangs.
		httpClient: &http.Client{Timeout: plantumlHTTPTimeout},
	}
}

// plantumlPattern matches <pre><code class="language-plantuml">...</code></pre>
var plantumlPattern = regexp.MustCompile(
	`<pre[^>]*><code[^>]*class="[^"]*language-plantuml[^"]*"[^>]*>([\s\S]*?)</code></pre>`)

// RenderHTML processes HTML content and replaces PlantUML code blocks with SVG output.
func (r *Renderer) RenderHTML(html string) (string, error) {
	var err error
	result := plantumlPattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := plantumlPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		// Unescape HTML entities (goldmark escapes < > &, etc.)
		code := parts[1]
		code = unescapeHTML(code)
		code = strings.TrimSpace(code)

		// Try to get SVG (from cache or by rendering)
		svg, cacheErr := r.getSVG(code)
		if cacheErr != nil {
			err = cacheErr
			return match // Return original on error
		}

		// Return the SVG wrapped in a div for consistency
		return fmt.Sprintf(`<div class="plantuml-diagram">%s</div>`, svg)
	})
	return result, err
}

// getSVG returns the SVG for the given PlantUML code, using cache when available.
func (r *Renderer) getSVG(code string) (string, error) {
	// Check cache first
	if cached, ok := r.cache.Load(code); ok {
		return cached.(string), nil
	}

	var svg string
	var err error

	if r.useLocal {
		svg, err = r.renderLocal(code)
	} else {
		svg, err = r.renderServer(code)
	}

	if err != nil {
		return "", err
	}

	// Cache the result
	r.cache.Store(code, svg)
	return svg, nil
}

// renderServer renders PlantUML using the online server.
func (r *Renderer) renderServer(code string) (string, error) {
	// Encode the PlantUML code using deflate + base64 encoding
	// PlantUML uses a custom encoding alphabet
	encoded, err := encodeForServer(code)
	if err != nil {
		return "", fmt.Errorf("failed to encode plantuml code: %w", err)
	}

	// Construct the URL
	url := fmt.Sprintf("%s/svg/%s", r.serverURL, encoded)

	// Fetch the SVG
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for plantuml diagram: %w", err)
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plantuml diagram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read error response body (capped to prevent excessive logging)
		var errBuf bytes.Buffer
		_, _ = io.Copy(&errBuf, io.LimitReader(resp.Body, 1024))
		errMsg := errBuf.String()
		if errMsg != "" {
			return "", fmt.Errorf("plantuml server returned status %d: %s", resp.StatusCode, errMsg)
		}
		return "", fmt.Errorf("plantuml server returned status %d", resp.StatusCode)
	}

	// Read the SVG content (capped at maxResponseSize to prevent DoS)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(resp.Body, maxResponseSize+1)); err != nil {
		return "", fmt.Errorf("failed to read plantuml response: %w", err)
	}
	if buf.Len() > maxResponseSize {
		return "", fmt.Errorf("plantuml response exceeds maximum size (%d bytes)", maxResponseSize)
	}

	return buf.String(), nil
}

// renderLocal renders PlantUML using a local plantuml installation.
//
// Detection order:
//  1. PLANTUML_JAR env var — runs "java -jar $PLANTUML_JAR -tsvg -pipe"
//  2. "plantuml" on PATH   — runs "plantuml -tsvg -pipe"
//
// If neither is available the error message includes install instructions.
func (r *Renderer) renderLocal(code string) (string, error) {
	cmd, err := localPlantumlCmd()
	if err != nil {
		return "", err
	}

	cmd.Stdin = strings.NewReader(code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("plantuml execution failed: %s", msg)
	}

	svg := stdout.String()
	if svg == "" {
		return "", fmt.Errorf("plantuml produced no output")
	}

	return svg, nil
}

// localPlantumlCmd builds an exec.Cmd for the local PlantUML renderer.
// It checks PLANTUML_JAR first, then falls back to "plantuml" on PATH.
func localPlantumlCmd() (*exec.Cmd, error) {
	// Prefer an explicit jar path from the environment.
	if jar := os.Getenv("PLANTUML_JAR"); jar != "" {
		if _, err := os.Stat(jar); err != nil {
			return nil, fmt.Errorf("PLANTUML_JAR points to %q but the file does not exist: %w", jar, err)
		}
		javaPath, err := exec.LookPath("java")
		if err != nil {
			return nil, fmt.Errorf("PLANTUML_JAR is set but java is not in PATH: %w", err)
		}
		return exec.CommandContext(context.Background(), javaPath, "-jar", jar, "-tsvg", "-pipe", "-charset", "UTF-8"), nil
	}

	// Fall back to the plantuml wrapper script / binary.
	path, err := exec.LookPath("plantuml")
	if err != nil {
		return nil, fmt.Errorf(
			"plantuml not found: install plantuml (e.g. brew install plantuml) " +
				"or set PLANTUML_JAR=/path/to/plantuml.jar",
		)
	}
	return exec.CommandContext(context.Background(), path, "-tsvg", "-pipe", "-charset", "UTF-8"), nil
}

// encodeForServer encodes PlantUML code for the online server using deflate + base64.
// PlantUML uses a custom base64 alphabet: 0-9, A-Z, a-z, -, _
func encodeForServer(code string) (string, error) {
	// Compress with deflate
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte(code)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	compressed := buf.Bytes()

	// Encode with standard base64
	encoded := base64.StdEncoding.EncodeToString(compressed)

	// Replace characters to use PlantUML's custom alphabet
	// Standard base64: A-Z, a-z, 0-9, +, /
	// PlantUML:       A-Z, a-z, 0-9, -, _
	encoded = strings.NewReplacer(
		"+", "-",
		"/", "_",
	).Replace(encoded)

	// Remove padding
	encoded = strings.TrimRight(encoded, "=")

	return encoded, nil
}

// unescapeHTML reverses HTML entity escaping done by goldmark.
func unescapeHTML(s string) string {
	replacements := map[string]string{
		"&lt;":   "<",
		"&gt;":   ">",
		"&amp;":  "&",
		"&quot;": `"`,
		"&#39;":  "'",
	}
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}

// NeedsPlantuml reports whether the HTML contains any PlantUML diagram elements.
func NeedsPlantuml(html string) bool {
	return strings.Contains(html, `class="plantuml-diagram"`) ||
		strings.Contains(html, `language-plantuml`)
}
