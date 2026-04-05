// Package plantuml provides rendering of PlantUML diagrams in Markdown content.
//
// It detects ```plantuml code blocks in HTML content and converts them to SVG
// using either the PlantUML online server or a local plantuml command.
package plantuml

import (
	"bytes"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yeasy/mdpress/pkg/utils"
)

const (
	// defaultPlantUMLServer is the base URL for the public PlantUML rendering service.
	defaultPlantUMLServer = "https://www.plantuml.com/plantuml"
	// maxResponseSize limits PlantUML server responses to 10 MB to prevent DoS.
	maxResponseSize = 10 * 1024 * 1024
	// maxErrorBodySize caps how much of an error response body we read for logging.
	maxErrorBodySize = 1024
	// maxStderrSize caps captured stderr from local PlantUML to 1 MB.
	maxStderrSize = 1024 * 1024
	// plantumlHTTPTimeout is the timeout for HTTP requests to the PlantUML server.
	plantumlHTTPTimeout = 30 * time.Second
	// localPlantumlTimeout caps local PlantUML execution to prevent hangs
	// when the caller's context has no deadline (e.g. context.Background).
	localPlantumlTimeout = 120 * time.Second
)

// Renderer handles the detection and conversion of PlantUML diagrams.
type Renderer struct {
	// serverURL is the base URL for the PlantUML server (e.g. "https://www.plantuml.com/plantuml")
	serverURL string
	// useLocal determines whether to use a local plantuml command instead of the server
	useLocal bool
	// cache stores rendered SVGs to avoid repeated network calls
	cache sync.Map // map[string]string - key is plantuml code, value is SVG HTML
	// httpClient is used for making requests to the PlantUML server
	httpClient *http.Client
}

// NewRenderer creates a new PlantUML renderer.
// serverURL should be the base URL without the /svg path (e.g. "https://www.plantuml.com/plantuml")
func NewRenderer(serverURL string, useLocal bool) (*Renderer, error) {
	if serverURL == "" {
		serverURL = defaultPlantUMLServer
	}
	serverURL = strings.TrimSuffix(serverURL, "/")

	// Validate the server URL to prevent SSRF via crafted book.yaml.
	if err := validatePlantUMLServer(serverURL); err != nil {
		return nil, fmt.Errorf("invalid plantuml server URL: %w", err)
	}

	return &Renderer{
		serverURL: serverURL,
		useLocal:  useLocal,
		// HTTP timeout for PlantUML server requests. This balances network
		// latency and rendering time for typical diagrams while preventing indefinite hangs.
		httpClient: &http.Client{
			Timeout: plantumlHTTPTimeout,
			// Use SSRF-safe transport that validates resolved IPs at dial time
			// to prevent DNS rebinding attacks against custom PlantUML servers.
			Transport: utils.SSRFSafeTransport(),
			// Validate redirect targets to prevent SSRF via 302 to internal IPs.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= utils.MaxHTTPRedirects {
					return errors.New("too many redirects")
				}
				return utils.CheckURLNotPrivate(req.URL)
			},
		},
	}, nil
}

// ClearCache removes all cached PlantUML renderings, freeing memory.
// This should be called between rebuild cycles in long-running processes.
func (r *Renderer) ClearCache() {
	r.cache.Range(func(key, _ any) bool {
		r.cache.Delete(key)
		return true
	})
}

// newRendererNoValidation creates a Renderer without SSRF validation.
// This is intended for tests that use local mock servers.
func newRendererNoValidation(serverURL string, useLocal bool) *Renderer {
	if serverURL == "" {
		serverURL = defaultPlantUMLServer
	}
	return &Renderer{
		serverURL:  strings.TrimSuffix(serverURL, "/"),
		useLocal:   useLocal,
		httpClient: &http.Client{Timeout: plantumlHTTPTimeout},
	}
}

// validatePlantUMLServer checks that the PlantUML server URL uses HTTPS and
// does not resolve to a private/internal address.
func validatePlantUMLServer(serverURL string) error {
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("scheme %q not allowed; use http or https", u.Scheme)
	}
	return utils.CheckURLNotPrivate(u)
}

// plantumlPattern matches <pre><code class="language-plantuml">...</code></pre>
var plantumlPattern = regexp.MustCompile(
	`<pre[^>]*><code[^>]*class="[^"]*language-plantuml[^"]*"[^>]*>([\s\S]*?)</code></pre>`)

// RenderHTML processes HTML content and replaces PlantUML code blocks with SVG output.
func (r *Renderer) RenderHTML(ctx context.Context, html string) (string, error) {
	var errs []error
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
		svg, cacheErr := r.getSVG(ctx, code)
		if cacheErr != nil {
			errs = append(errs, cacheErr)
			return match // Return original on error
		}

		// Sanitize SVG to prevent script injection from compromised servers.
		svg = sanitizeSVG(svg)
		// Return the SVG wrapped in a div for consistency
		return fmt.Sprintf(`<div class="plantuml-diagram">%s</div>`, svg)
	})
	return result, errors.Join(errs...)
}

// getSVG returns the SVG for the given PlantUML code, using cache when available.
func (r *Renderer) getSVG(ctx context.Context, code string) (string, error) {
	// Sanitize before caching or rendering — applies to both local and server paths.
	code = sanitizePlantUMLCode(code)

	// Check cache first
	if cached, ok := r.cache.Load(code); ok {
		if s, ok := cached.(string); ok {
			return s, nil
		}
	}

	var svg string
	var err error

	if r.useLocal {
		svg, err = r.renderLocal(ctx, code)
	} else {
		svg, err = r.renderServer(ctx, code)
	}

	if err != nil {
		return "", err
	}

	// Cache the result
	r.cache.Store(code, svg)
	return svg, nil
}

// renderServer renders PlantUML using the online server.
func (r *Renderer) renderServer(ctx context.Context, code string) (string, error) {
	// Encode the PlantUML code using deflate + base64 encoding
	// PlantUML uses a custom encoding alphabet
	encoded, err := encodeForServer(code)
	if err != nil {
		return "", fmt.Errorf("failed to encode plantuml code: %w", err)
	}

	// Construct the URL
	url := fmt.Sprintf("%s/svg/%s", r.serverURL, encoded)

	// Fetch the SVG
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for plantuml diagram: %w", err)
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch plantuml diagram: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		// Read error response body (capped to prevent excessive logging)
		var errBuf bytes.Buffer
		_, _ = io.Copy(&errBuf, io.LimitReader(resp.Body, maxErrorBodySize))
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

// sanitizePlantUMLCode strips potentially dangerous PlantUML directives
// that could read arbitrary local files, fetch remote resources, or leak
// environment variables when running a local PlantUML process.
func sanitizePlantUMLCode(code string) string {
	lines := strings.Split(code, "\n")
	var sb strings.Builder
	sb.Grow(len(code))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if isDangerousPlantUMLDirective(lower) {
			sb.WriteString("' [directive removed for security]")
		} else {
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// isDangerousPlantUMLDirective returns true if the lowercased, trimmed line
// starts with a PlantUML preprocessor directive that could access the
// filesystem, network, or environment.
func isDangerousPlantUMLDirective(lower string) bool {
	// File inclusion and imports
	if strings.HasPrefix(lower, "!include") || strings.HasPrefix(lower, "!import") {
		return true
	}
	// Macro definitions (can reference external resources)
	if strings.HasPrefix(lower, "!define") {
		return true
	}
	// Theme loading (can load external files)
	if strings.HasPrefix(lower, "!theme") {
		return true
	}
	// Pragma directives (can alter security behavior)
	if strings.HasPrefix(lower, "!pragma") {
		return true
	}
	// Function/procedure definitions (can call dangerous built-ins or
	// construct dynamic include paths via string concatenation)
	if strings.HasPrefix(lower, "!function") || strings.HasPrefix(lower, "!procedure") ||
		strings.HasPrefix(lower, "!unquoted") || strings.HasPrefix(lower, "!log") ||
		strings.HasPrefix(lower, "!local") {
		return true
	}
	// Sub-part directives (used with !include to selectively include file parts)
	if strings.HasPrefix(lower, "!startsub") || strings.HasPrefix(lower, "!endsub") {
		return true
	}
	// Built-in functions that leak filesystem or environment info
	if strings.Contains(lower, "%load_json") || strings.Contains(lower, "%getenv") ||
		strings.Contains(lower, "%filename") || strings.Contains(lower, "%dirpath") ||
		strings.Contains(lower, "%file_exists") || strings.Contains(lower, "%fileexists") {
		return true
	}
	// Allow safe control-flow directives that do not access the
	// filesystem, network, or environment. These must be checked before
	// the catch-all below so that legitimate conditional/loop syntax
	// in PlantUML diagrams is preserved.
	safeControlFlow := []string{
		"!if", "!ifdef", "!ifndef", "!elseif", "!else", "!endif",
		"!while", "!endwhile", "!foreach", "!endfor", "!endforeach",
		"!return", "!assert",
		"!endfunction", "!endprocedure",
	}
	for _, safe := range safeControlFlow {
		if lower == safe || strings.HasPrefix(lower, safe+" ") || strings.HasPrefix(lower, safe+"\t") {
			return false
		}
	}
	// Catch-all: block unknown preprocessor directives (future-proofing)
	if len(lower) > 1 && lower[0] == '!' && lower[1] >= 'a' && lower[1] <= 'z' {
		return true
	}
	return false
}

// renderLocal renders PlantUML using a local plantuml installation.
//
// Detection order:
//  1. PLANTUML_JAR env var — runs "java -jar $PLANTUML_JAR -tsvg -pipe"
//  2. "plantuml" on PATH   — runs "plantuml -tsvg -pipe"
//
// If neither is available the error message includes install instructions.
func (r *Renderer) renderLocal(ctx context.Context, code string) (string, error) {
	// Ensure a deadline exists even if the caller passes context.Background(),
	// preventing a malicious diagram from hanging indefinitely.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, localPlantumlTimeout)
		defer cancel()
	}
	cmd, err := localPlantumlCmd(ctx)
	if err != nil {
		return "", err
	}

	cmd.Stdin = strings.NewReader(code)

	// Limit captured output to maxResponseSize to prevent memory exhaustion
	// from malicious or runaway local PlantUML processes.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &utils.LimitedWriter{W: &stdout, N: maxResponseSize}
	cmd.Stderr = &utils.LimitedWriter{W: &stderr, N: maxStderrSize}

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("plantuml execution failed: %s", msg)
	}

	svg := stdout.String()
	if svg == "" {
		return "", errors.New("plantuml produced no output")
	}

	return svg, nil
}

// localPlantumlCmd builds an exec.Cmd for the local PlantUML renderer.
// It checks PLANTUML_JAR first, then falls back to "plantuml" on PATH.
func localPlantumlCmd(ctx context.Context) (*exec.Cmd, error) {
	// Prefer an explicit jar path from the environment.
	if jar := os.Getenv("PLANTUML_JAR"); jar != "" {
		if _, err := os.Stat(jar); err != nil {
			return nil, fmt.Errorf("PLANTUML_JAR points to %q but the file does not exist: %w", jar, err)
		}
		javaPath, err := exec.LookPath("java")
		if err != nil {
			return nil, fmt.Errorf("PLANTUML_JAR is set but java is not in PATH: %w", err)
		}
		return exec.CommandContext(ctx, javaPath, "-jar", jar, "-tsvg", "-pipe", "-charset", "UTF-8"), nil
	}

	// Fall back to the plantuml wrapper script / binary.
	path, err := exec.LookPath("plantuml")
	if err != nil {
		return nil, errors.New(
			"plantuml not found: install plantuml (e.g. brew install plantuml) " +
				"or set PLANTUML_JAR=/path/to/plantuml.jar",
		)
	}
	return exec.CommandContext(ctx, path, "-tsvg", "-pipe", "-charset", "UTF-8"), nil
}

// encodeForServer encodes PlantUML code for the online server.
// PlantUML uses raw deflate (not zlib) followed by a custom base64 encoding
// with the alphabet: 0-9, A-Z, a-z, -, _ (different order from standard base64).
func encodeForServer(code string) (string, error) {
	// Compress with raw deflate (no zlib header/checksum).
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := w.Write([]byte(code)); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	compressed := buf.Bytes()

	// Encode using PlantUML's custom base64 alphabet.
	var sb strings.Builder
	for i := 0; i < len(compressed); i += 3 {
		switch {
		case i+2 < len(compressed):
			sb.WriteByte(plantumlEncode6bit(compressed[i] >> 2))
			sb.WriteByte(plantumlEncode6bit(((compressed[i] & 0x3) << 4) | (compressed[i+1] >> 4)))
			sb.WriteByte(plantumlEncode6bit(((compressed[i+1] & 0xF) << 2) | (compressed[i+2] >> 6)))
			sb.WriteByte(plantumlEncode6bit(compressed[i+2] & 0x3F))
		case i+1 < len(compressed):
			sb.WriteByte(plantumlEncode6bit(compressed[i] >> 2))
			sb.WriteByte(plantumlEncode6bit(((compressed[i] & 0x3) << 4) | (compressed[i+1] >> 4)))
			sb.WriteByte(plantumlEncode6bit((compressed[i+1] & 0xF) << 2))
			sb.WriteByte('=')
		default:
			sb.WriteByte(plantumlEncode6bit(compressed[i] >> 2))
			sb.WriteByte(plantumlEncode6bit((compressed[i] & 0x3) << 4))
			sb.WriteByte('=')
			sb.WriteByte('=')
		}
	}

	return strings.TrimRight(sb.String(), "="), nil
}

// plantumlEncode6bit maps a 6-bit value to the PlantUML base64 alphabet.
// Alphabet order: 0-9 A-Z a-z - _
func plantumlEncode6bit(b byte) byte {
	b &= 0x3F
	switch {
	case b < 10:
		return '0' + b
	case b < 36:
		return 'A' + b - 10
	case b < 62:
		return 'a' + b - 36
	case b == 62:
		return '-'
	default:
		return '_'
	}
}

// unescapeHTML reverses HTML entity escaping done by goldmark.
// Order matters: &amp; must be processed last to avoid double-unescaping
// (e.g., "&amp;lt;" -> "&lt;" -> "<" if &amp; were processed first).
func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&amp;", "&") // must be last
	return s
}

// SVG sanitization patterns to strip dangerous content from PlantUML output.
var (
	svgScriptPattern        = regexp.MustCompile(`(?i)<script[\s>][\s\S]*?</script>`)
	svgEventHandlerPattern  = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]*)`)
	svgJavascriptURLPattern = regexp.MustCompile(`(?i)(href|xlink:href)\s*=\s*(?:"(?:javascript|data|vbscript):[^"]*"|'(?:javascript|data|vbscript):[^']*')`)
	svgForeignObjectPattern = regexp.MustCompile(`(?i)<foreignObject[\s>][\s\S]*?</foreignObject>`)
	svgUsePattern           = regexp.MustCompile(`(?i)<use[^>]+(?:xlink:)?href\s*=\s*(?:"[^"#][^"]*"|'[^'#][^']*')[^>]*>`)
	// Strip <style> blocks that could exfiltrate data via @import/url().
	svgStylePattern = regexp.MustCompile(`(?i)<style[\s>][\s\S]*?</style>`)
	// Strip SVG animation elements that could alter attributes post-sanitization.
	svgAnimatePattern = regexp.MustCompile(`(?i)<(?:animate|set|animateTransform|animateMotion)[\s>][\s\S]*?</(?:animate|set|animateTransform|animateMotion)>`)
	// Self-closing animation/iframe elements.
	svgAnimateSCPattern = regexp.MustCompile(`(?i)<(?:animate|set|animateTransform|animateMotion|iframe)\b[^>]*/?>`)
)

// sanitizeSVG strips potentially dangerous elements from SVG content
// to prevent XSS from compromised or malicious PlantUML servers.
func sanitizeSVG(svg string) string {
	svg = svgScriptPattern.ReplaceAllString(svg, "")
	svg = svgForeignObjectPattern.ReplaceAllString(svg, "")
	svg = svgStylePattern.ReplaceAllString(svg, "")
	svg = svgAnimatePattern.ReplaceAllString(svg, "")
	svg = svgAnimateSCPattern.ReplaceAllString(svg, "")
	svg = svgEventHandlerPattern.ReplaceAllString(svg, "")
	svg = svgJavascriptURLPattern.ReplaceAllString(svg, "")
	svg = svgUsePattern.ReplaceAllString(svg, "")
	return svg
}

// needsPlantuml reports whether the HTML contains any PlantUML diagram elements.
func needsPlantuml(html string) bool {
	return strings.Contains(html, `class="plantuml-diagram"`) ||
		strings.Contains(html, `language-plantuml`)
}
