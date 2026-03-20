// Package plantuml provides rendering of PlantUML diagrams in Markdown content.
//
// It detects ```plantuml code blocks in HTML content and converts them to SVG
// using either the PlantUML online server or a local plantuml command.
package plantuml

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
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
		serverURL:  strings.TrimSuffix(serverURL, "/"),
		useLocal:   useLocal,
		httpClient: &http.Client{Timeout: 30000000000}, // 30 second timeout
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
	resp, err := r.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plantuml diagram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("plantuml server returned status %d", resp.StatusCode)
	}

	// Read the SVG content
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read plantuml response: %w", err)
	}

	return buf.String(), nil
}

// renderLocal renders PlantUML using a local plantuml command.
// This is a placeholder for future implementation.
func (r *Renderer) renderLocal(code string) (string, error) {
	// TODO: Implement local PlantUML rendering using os/exec
	return "", fmt.Errorf("local plantuml rendering not yet implemented")
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
