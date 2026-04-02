// Package pdf renders HTML documents to PDF using Chromium.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/yeasy/mdpress/pkg/utils"
)

// allowedChromiumFlags is an allowlist of safe Chromium command-line flags
// that users are permitted to set via the CHROME_FLAGS environment variable.
var allowedChromiumFlags = map[string]bool{
	"no-sandbox":                       true,
	"disable-gpu":                      true,
	"headless":                         true,
	"disable-dev-shm-usage":            true,
	"font-render-hinting":              true,
	"disable-software-rasterizer":      true,
	"disable-extensions":               true,
	"disable-background-networking":    true,
	"disable-sync":                     true,
	"disable-translate":                true,
	"mute-audio":                       true,
	"no-first-run":                     true,
	"safebrowsing-disable-auto-update": true,
	"hide-scrollbars":                  true,
	"disable-notifications":            true,
	"disable-crash-reporter":           true,
	"noerrdialogs":                     true,
	"allow-file-access-from-files":     true,
	"no-pdf-header-footer":             true,
	"print-to-pdf":                     true,
	"user-data-dir":                    true,
}

// newlineStripper removes CR and LF characters from log messages to prevent
// log injection via user-controlled flag names.
var newlineStripper = strings.NewReplacer("\n", "", "\r", "")

// cjkFontCandidate describes a CJK font with filesystem paths that Chromium
// can load via file:// URL for embedding in PDF output.
//
// Only file:// sources are used — no local() — because fonts resolved through
// local() are managed by the OS font API (Core Text on macOS) and Chrome's
// Skia PDF backend cannot embed them.  Fonts loaded via file:// URL are read
// as raw bytes and can be embedded, ensuring CJK glyphs appear in the PDF.
type cjkFontCandidate struct {
	// family is the original font family name (used only for documentation).
	family string
	// paths are candidate filesystem paths checked in order; the first
	// existing file is used as the file:// URL source in the @font-face rule.
	paths []string
}

type cjkFontSource struct {
	path string
}

// systemCJKFontCandidates lists CJK font paths in preference order.
// User-installed fonts come first because Chrome can embed them via file:// URL.
// macOS system fonts (managed by Core Text) come last as a fallback; Chrome
// may not be able to embed them cleanly, but they are better than nothing.
var systemCJKFontCandidates = []cjkFontCandidate{
	// ── User-installed fonts (macOS ~/Library/Fonts) ────────────────────────
	// Confirmed embeddable: Chrome reads raw bytes via file:// and embeds them.
	{
		family: "Microsoft YaHei",
		paths: func() []string {
			paths := []string{
				"/Library/Fonts/msyh.ttc",
				`C:\Windows\Fonts\msyh.ttc`,
			}
			if home, err := os.UserHomeDir(); err == nil {
				paths = append([]string{filepath.Join(home, "Library", "Fonts", "msyh.ttc")}, paths...)
			}
			return paths
		}(),
	},
	// ── Linux / Docker (Noto CJK, WenQuanYi) ──────────────────────────────
	{
		family: "Noto Sans SC",
		paths: []string{
			"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/google-noto-cjk/NotoSansCJKsc-Regular.otf",
			"/usr/share/fonts/noto-cjk/NotoSansCJKsc-Regular.otf",
		},
	},
	{
		family: "WenQuanYi Micro Hei",
		paths: []string{
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
			"/usr/share/fonts/wqy-microhei/wqy-microhei.ttc",
		},
	},
	// ── Homebrew / user-installed Noto CJK (macOS) ───────────────────────
	{
		family: "Noto Sans CJK SC (Homebrew)",
		paths: func() []string {
			prefixes := []string{"/opt/homebrew/share/fonts", "/usr/local/share/fonts"}
			paths := make([]string, 0, len(prefixes)*2)
			for _, prefix := range prefixes {
				paths = append(paths,
					prefix+"/NotoSansCJKsc-Regular.otf",
					prefix+"/NotoSansCJK-Regular.ttc",
				)
			}
			return paths
		}(),
	},
	// ── macOS system fonts ────────────────────────────────────────────────
	// PingFang SC is the default CJK font on macOS 10.11+.
	// Chrome can embed fonts loaded via file:// URL even from the sealed
	// system volume, because Chromium reads raw bytes via open(2), bypassing
	// Core Text.
	{
		family: "PingFang SC",
		paths:  []string{"/System/Library/Fonts/PingFang.ttc"},
	},
	{
		family: "Hiragino Sans GB",
		paths: []string{
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
			"/System/Library/Fonts/HiraginoSans-W3.ttc",
		},
	},
	{
		family: "Songti SC",
		paths: []string{
			"/System/Library/Fonts/Supplemental/Songti.ttc",
			"/System/Library/Fonts/STSong.ttf",
		},
	},
	{
		family: "Heiti SC",
		paths: []string{
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
		},
	},
}

// buildCJKFontFaceCSS generates a @font-face rule that aliases the first
// available CJK font file as "CJK-Embedded", plus a body font-family override
// that places "CJK-Embedded" first in the stack.
//
// Design rationale:
//   - Chrome's Skia PDF backend silently drops glyphs from fonts that it
//     cannot embed.  Fonts selected via the normal font-family stack (e.g.
//     PingFang SC, Hiragino Sans GB) are managed by Core Text and cannot be
//     embedded; only fonts loaded from raw bytes via file:// URL can be embedded.
//   - By using a unique alias "CJK-Embedded" backed by a file:// URL and
//     placing it first in body's font-family, we force Chrome to load the font
//     from disk for CJK code points — making those glyphs embeddable.
//   - unicode-range limits "CJK-Embedded" to actual CJK code points; Latin
//     and other characters continue to use the remaining font-family entries.
//
// cjkFontResult holds the @font-face CSS and the path of the selected font.
type cjkFontResult struct {
	css      string // empty when no CJK font is found
	fontPath string // filesystem path of the selected font
	family   string // family name of the selected candidate
}

func buildCJKFontFaceCSS() cjkFontResult {
	const cjkRange = "U+2E80-2EFF, U+3000-303F, U+3400-4DBF, U+4E00-9FFF, " +
		"U+F900-FAFF, U+FE30-FE4F, U+FF00-FFEF, " +
		"U+20000-2A6DF, U+2A700-2B73F, U+2B740-2B81F, U+2B820-2CEAF"

	// Find the first available CJK font file.
	var font cjkFontSource
	var family string
	for _, c := range systemCJKFontCandidates {
		for _, p := range c.paths {
			if _, err := os.Stat(p); err == nil {
				font = cjkFontSource{path: p}
				family = c.family
				break
			}
		}
		if font.path != "" {
			break
		}
	}
	if font.path == "" {
		return cjkFontResult{} // no CJK font found
	}

	var css strings.Builder
	// Declare the embeddable CJK alias restricted to CJK code points.
	fmt.Fprintf(&css,
		"@font-face {\n  font-family: \"CJK-Embedded\";\n  src: %s;\n  font-style: normal;\n  font-weight: 400;\n  unicode-range: %s;\n}\n\n",
		cjkFontSrc(font), cjkRange)
	// Override body font-family so Chrome selects "CJK-Embedded" for CJK code
	// points instead of a Core Text–managed system font.
	// Use !important to win the CSS cascade over theme CSS which sets
	// body { font-family: var(--font-family); } and would otherwise
	// override this injection, causing CJK characters to render as tofu.
	css.WriteString("body {\n" +
		"  font-family: \"CJK-Embedded\", -apple-system, BlinkMacSystemFont, \"Segoe UI\",\n" +
		"    \"PingFang SC\", \"Hiragino Sans GB\", \"Heiti SC\", \"Heiti TC\",\n" +
		"    \"Microsoft YaHei\", \"Noto Sans SC\", \"Noto Sans CJK SC\",\n" +
		"    \"Source Han Sans SC\", \"WenQuanYi Micro Hei\",\n" +
		"    \"Roboto\", \"Droid Sans\", \"Helvetica Neue\", sans-serif !important;\n" +
		"}\n")
	// Override monospace elements too — code/pre have their own font-family
	// that does not inherit from body, so CJK characters in code blocks would
	// be missing without this rule.
	css.WriteString("code, pre, kbd, samp, .hljs {\n" +
		"  font-family: \"CJK-Embedded\", ui-monospace, \"SF Mono\", Menlo, Monaco,\n" +
		"    Consolas, \"Liberation Mono\", \"Courier New\",\n" +
		"    \"PingFang SC\", \"Hiragino Sans GB\", \"Microsoft YaHei\",\n" +
		"    \"Noto Sans Mono CJK SC\", monospace !important;\n" +
		"}\n")
	// Override Mermaid SVG text elements — Mermaid renders diagrams as SVG
	// and sets font-family directly on <text>/<foreignObject> elements.
	// Without this rule, CJK characters in diagrams render as tofu in PDF.
	// Mermaid renders diagrams as SVG.  SVG <text> elements resolve fonts
	// differently from HTML — Chrome's PDF backend may fail to fall back to
	// system Latin fonts after a unicode-range-restricted CJK @font-face.
	// Place common Latin fonts explicitly before the CJK stack so digits
	// and ASCII characters always render in the PDF.
	css.WriteString(".mermaid text, .mermaid tspan, .mermaid foreignObject,\n" +
		".mermaid .label, .mermaid .nodeLabel, .mermaid .edgeLabel,\n" +
		".mermaid .cluster-label, .mermaid .titleText,\n" +
		".mermaid [class*=\"Label\"] {\n" +
		"  font-family: -apple-system, BlinkMacSystemFont, \"Segoe UI\",\n" +
		"    Helvetica, Arial, \"CJK-Embedded\", \"PingFang SC\", \"Hiragino Sans GB\",\n" +
		"    \"Microsoft YaHei\", \"Noto Sans SC\", \"Noto Sans CJK SC\",\n" +
		"    \"Source Han Sans SC\", sans-serif !important;\n" +
		"}\n")
	return cjkFontResult{css: css.String(), fontPath: font.path, family: family}
}

func cjkFontSrc(src cjkFontSource) string {
	// Use a relative URL that will be resolved by the local HTTP server
	// started in Generate().  Previous approaches (file:// URLs and data:
	// URIs) failed on macOS because:
	//   - file:// fonts: Chrome's Skia PDF backend cannot embed fonts loaded
	//     from file:// @font-face on macOS (glyphs render on screen but are
	//     silently dropped during PDF serialization).
	//   - data: URIs: TTC font files are 20+ MB; base64 encoding produces
	//     30+ MB of inline CSS that Chrome cannot reliably parse.
	// Serving fonts via HTTP (localhost) lets Chrome's network stack fetch
	// the raw font bytes, which Skia can then embed into the PDF output.
	//
	// A format() hint helps Chrome correctly identify the font format,
	// especially for TTC (TrueType Collection) files like PingFang.ttc.
	ext := strings.ToLower(filepath.Ext(src.path))
	switch ext {
	case ".ttc", ".otc":
		return `url("/cjk-font") format("collection")`
	case ".otf":
		return `url("/cjk-font") format("opentype")`
	case ".woff":
		return `url("/cjk-font") format("woff")`
	case ".woff2":
		return `url("/cjk-font") format("woff2")`
	default: // .ttf or unknown
		return `url("/cjk-font") format("truetype")`
	}
}

// cjkFontSrcFallback returns a file:// URL for use in non-HTTP contexts
// (e.g. when the HTML is loaded directly from disk without the font server).
func cjkFontSrcFallback(src cjkFontSource) string {
	fontURL := fileURLForCSS(src.path)
	return fmt.Sprintf("url(%q)", fontURL)
}

func fileURLForCSS(path string) string {
	return (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(path),
	}).String()
}

// injectCJKFontFaceCSS inserts a <style> block with CJK @font-face rules
// immediately before </head> in htmlContent and returns the modified string.
// When there is no </head> tag the block is prepended to the content.
// Injecting into the HTML (rather than via JavaScript after page load) ensures
// Chrome's font-matching pass sees the rules during the initial layout.
func injectCJKFontFaceCSS(htmlContent string, logger *slog.Logger) string {
	result := buildCJKFontFaceCSS()
	if result.css == "" {
		if logger != nil {
			logger.Warn("No CJK font file found on system — PDF may show blank squares for CJK text")
		}
		return htmlContent
	}
	if logger != nil {
		logger.Info("CJK font for PDF embedding", slog.String("family", result.family), slog.String("path", result.fontPath))
	}
	block := "<style data-cjk-fonts=\"1\">\n" + result.css + "</style>\n"
	if idx := strings.Index(htmlContent, "</head>"); idx != -1 {
		return htmlContent[:idx] + block + htmlContent[idx:]
	}
	return block + htmlContent
}

// Generator converts HTML into PDF files.
type Generator struct {
	timeout                 time.Duration
	pageWidth               float64 // Millimeters.
	pageHeight              float64 // Millimeters.
	marginLeft              float64
	marginRight             float64
	marginTop               float64
	marginBottom            float64
	printBackground         bool
	displayHeaderFooter     bool
	headerTemplate          string
	footerTemplate          string
	generateDocumentOutline bool // Generate clickable PDF bookmarks from heading hierarchy.
	generateTaggedPDF       bool // Generate tagged (accessible) PDF.
}

type chromiumRuntimeDirs struct {
	root      string
	homeDir   string
	userData  string
	tmpDir    string
	xdgConfig string
	xdgCache  string
	cleanup   func()
}

// GeneratorOption customizes a PDF generator.
type GeneratorOption func(*Generator)

const (
	defaultTimeout           = 60 * time.Second
	chromiumPrintTimeout     = 120 * time.Second
	defaultPageWidth         = 210.0 // A4
	defaultPageHeight        = 297.0
	defaultMargin            = 20.0
	fontSrvReadHeaderTimeout = 10 * time.Second
	fontSrvReadTimeout       = 30 * time.Second
	fontSrvWriteTimeout      = 60 * time.Second
	fontSrvIdleTimeout       = 60 * time.Second
)

// parseMarginString converts a margin string (e.g., "20mm", "1in", "2.5cm") to millimeters.
// If the input is empty or invalid, it returns the default margin value.
func parseMarginString(s string, defaultMM float64) float64 {
	if s == "" {
		return defaultMM
	}
	s = strings.TrimSpace(s)
	matches := marginRe.FindStringSubmatch(s)
	if matches == nil {
		return defaultMM
	}
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return defaultMM
	}
	unit := strings.ToLower(matches[2])
	if unit == "" {
		unit = "mm" // Default to mm if not specified
	}
	// Convert to millimeters
	switch unit {
	case "mm":
		return value
	case "cm":
		return value * 10
	case "in":
		return value * 25.4
	case "pt":
		return value * 25.4 / 72.0
	case "px":
		return value * 25.4 / 96.0 // Assume 96 DPI
	default:
		return defaultMM
	}
}

var marginRe = regexp.MustCompile(`(?i)^(\d+\.?\d*)\s*(mm|cm|in|pt|px)?$`)

var (
	chromiumExecutableCandidates = []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}
	chromiumMacPaths = []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
)

// NewGenerator creates a PDF generator.
// By default, it generates a document outline (clickable bookmarks) and tagged PDF.
func NewGenerator(opts ...GeneratorOption) *Generator {
	g := &Generator{
		timeout:                 defaultTimeout,
		pageWidth:               defaultPageWidth,
		pageHeight:              defaultPageHeight,
		marginLeft:              defaultMargin,
		marginRight:             defaultMargin,
		marginTop:               defaultMargin,
		marginBottom:            defaultMargin,
		printBackground:         true,
		generateDocumentOutline: true,
		generateTaggedPDF:       true,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// WithTimeout sets the operation timeout.
func WithTimeout(d time.Duration) GeneratorOption {
	return func(g *Generator) { g.timeout = d }
}

// WithPageSize sets the page size in millimeters.
func WithPageSize(width, height float64) GeneratorOption {
	return func(g *Generator) {
		g.pageWidth = width
		g.pageHeight = height
	}
}

// WithMargins sets page margins in millimeters.
func WithMargins(left, right, top, bottom float64) GeneratorOption {
	return func(g *Generator) {
		g.marginLeft = left
		g.marginRight = right
		g.marginTop = top
		g.marginBottom = bottom
	}
}

// WithMarginStrings sets page margins from string values (e.g., "20mm", "1in").
// Uses default values if parsing fails.
func WithMarginStrings(left, right, top, bottom string) GeneratorOption {
	return func(g *Generator) {
		g.marginLeft = parseMarginString(left, defaultMargin)
		g.marginRight = parseMarginString(right, defaultMargin)
		g.marginTop = parseMarginString(top, defaultMargin)
		g.marginBottom = parseMarginString(bottom, defaultMargin)
	}
}

// WithPrintBackground toggles background printing.
func WithPrintBackground(print bool) GeneratorOption {
	return func(g *Generator) { g.printBackground = print }
}

// WithHeaderFooter toggles header and footer rendering.
func WithHeaderFooter(enable bool) GeneratorOption {
	return func(g *Generator) { g.displayHeaderFooter = enable }
}

// WithFooterTemplate sets a custom HTML footer template for PDF pages.
// The template is rendered by Chrome's PrintToPDF and supports CSS styling.
// Chrome provides special classes: "pageNumber", "totalPages", "date", "title", "url".
func WithFooterTemplate(tmpl string) GeneratorOption {
	return func(g *Generator) {
		g.footerTemplate = tmpl
		g.displayHeaderFooter = true
	}
}

// WithDocumentOutline toggles PDF bookmark/outline generation from heading hierarchy.
// Enabled by default. Requires Chrome 128+ for full support.
func WithDocumentOutline(enable bool) GeneratorOption {
	return func(g *Generator) { g.generateDocumentOutline = enable }
}

// WithTaggedPDF toggles tagged (accessible) PDF generation.
// Enabled by default. Tagged PDFs include structural metadata for screen readers.
func WithTaggedPDF(enable bool) GeneratorOption {
	return func(g *Generator) { g.generateTaggedPDF = enable }
}

// WarnIfCJKFontsMissing checks whether the HTML content contains CJK characters
// and warns the user if no CJK fonts are installed on the system.
// This is a best-effort check — it logs a warning but does not block PDF generation.
func WarnIfCJKFontsMissing(htmlContent string, logger interface{ Warn(string, ...any) }) {
	if !utils.ContainsCJK(htmlContent) {
		return
	}
	status := utils.CheckCJKFonts()
	if status.Available {
		return
	}
	if logger != nil {
		logger.Warn("CJK characters detected but no CJK fonts found on the system. " +
			"PDF output may show blank squares instead of Chinese/Japanese/Korean text. " +
			utils.CJKFontInstallHint())
	}
}

// fontServer serves HTML and CJK font files over localhost HTTP so that
// Chrome can load fonts via standard HTTP requests.  This avoids file:// and
// data: URI issues with font embedding on macOS.
type fontServer struct {
	listener net.Listener
	server   *http.Server
	baseURL  string
}

// newFontServer starts an HTTP server on a random localhost port, serving
// the given HTML content at "/" and the CJK font file at "/cjk-font".
func newFontServer(htmlContent string, fontPath string) (*fontServer, error) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start font server: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlContent)) //nolint:errcheck
	})
	if fontPath != "" {
		mux.HandleFunc("/cjk-font", func(w http.ResponseWriter, r *http.Request) {
			ext := strings.ToLower(filepath.Ext(fontPath))
			switch ext {
			case ".ttf":
				w.Header().Set("Content-Type", "font/ttf")
			case ".otf":
				w.Header().Set("Content-Type", "font/otf")
			case ".ttc", ".otc":
				w.Header().Set("Content-Type", "font/collection")
			case ".woff":
				w.Header().Set("Content-Type", "font/woff")
			case ".woff2":
				w.Header().Set("Content-Type", "font/woff2")
			default:
				w.Header().Set("Content-Type", "application/octet-stream")
			}
			http.ServeFile(w, r, fontPath)
		})
	}
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: fontSrvReadHeaderTimeout,
		ReadTimeout:       fontSrvReadTimeout,
		WriteTimeout:      fontSrvWriteTimeout,
		IdleTimeout:       fontSrvIdleTimeout,
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Debug("Font server error", slog.Any("error", err))
		}
	}()
	return &fontServer{
		listener: listener,
		server:   server,
		baseURL:  fmt.Sprintf("http://%s", listener.Addr().String()),
	}, nil
}

func (fs *fontServer) Close() {
	// server.Close() already closes the underlying listener, so we only
	// need to close the server itself.
	if err := fs.server.Close(); err != nil {
		slog.Debug("Failed to close font server", slog.Any("error", err))
	}
}

// Generate renders an HTML string to a PDF file.
func (g *Generator) Generate(htmlContent string, outputPath string) error {
	if outputPath == "" {
		return errors.New("generate: output path cannot be empty")
	}
	if htmlContent == "" {
		return errors.New("generate: HTML content cannot be empty")
	}

	// Find a CJK font and inject @font-face CSS into the HTML.
	cjkResult := buildCJKFontFaceCSS()
	if cjkResult.css != "" {
		slog.Info("CJK font for PDF embedding",
			slog.String("family", cjkResult.family),
			slog.String("path", cjkResult.fontPath))
		block := "<style data-cjk-fonts=\"1\">\n" + cjkResult.css + "</style>\n"
		if idx := strings.Index(htmlContent, "</head>"); idx != -1 {
			htmlContent = htmlContent[:idx] + block + htmlContent[idx:]
		} else {
			htmlContent = block + htmlContent
		}
	} else {
		slog.Warn("No CJK font file found on system — PDF may show blank squares for CJK text")
	}

	// Start a local HTTP server to serve HTML + CJK font.  Chrome loads the
	// page from http://localhost which lets it fetch the font via standard
	// HTTP — avoiding file:// and data: URI embedding issues on macOS.
	srv, err := newFontServer(htmlContent, cjkResult.fontPath)
	if err != nil {
		// Fall back to file-based approach if HTTP server fails.
		slog.Warn("Failed to start font server, falling back to file:// approach",
			slog.Any("error", err))
		return g.generateFromString(htmlContent, outputPath)
	}
	defer srv.Close() //nolint:errcheck

	return g.generateFromURL(srv.baseURL, outputPath)
}

// generateFromString writes HTML to a temp file and generates PDF from it.
func (g *Generator) generateFromString(htmlContent string, outputPath string) error {
	tmpFile, err := os.CreateTemp("", "mdpress-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	return g.GenerateFromFile(tmpPath, outputPath)
}

// GenerateFromFile renders a local HTML file to PDF.
func (g *Generator) GenerateFromFile(htmlFilePath string, outputPath string) error {
	absHTMLPath, err := filepath.Abs(htmlFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve HTML file path: %w", err)
	}
	if _, err := os.Stat(absHTMLPath); err != nil {
		return fmt.Errorf("html file does not exist: %w", err)
	}

	fileURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absHTMLPath),
	}).String()

	return g.generateFromURL(fileURL, outputPath)
}

// generateFromURL opens pageURL in Chrome and prints it to PDF.
func (g *Generator) generateFromURL(pageURL string, outputPath string) error {
	chromePath, err := resolveChromiumPath()
	if err != nil {
		return fmt.Errorf("resolve Chrome path: %w", err)
	}

	runtimeDirs, err := prepareChromiumRuntimeDirs()
	if err != nil {
		return fmt.Errorf("failed to prepare Chrome runtime directories: %w", err)
	}
	defer runtimeDirs.cleanup()

	var chromeOutput bytes.Buffer
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromiumAllocatorOptions(chromePath, runtimeDirs, &chromeOutput)...)
	defer allocCancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, g.timeout)
	defer timeoutCancel()

	mmToInch := func(mm float64) float64 { return mm / 25.4 }

	var pdfBuf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		// Wait up to 15 s (polling every 200 ms) for Mermaid diagrams to
		// finish rendering. Mermaid loads async from CDN and replaces
		// .mermaid divs with SVGs.
		chromedp.ActionFunc(func(ctx context.Context) error {
			const mermaidWaitJS = `(function() {
				var els = document.querySelectorAll('.mermaid');
				if (els.length === 0) return 'none';
				var maxWait = 15000, interval = 200, elapsed = 0;
				return new Promise(function(resolve) {
					function check() {
						var done = true;
						els.forEach(function(el) {
							if (!el.querySelector('svg')) done = false;
						});
						if (done) return resolve('ok');
						elapsed += interval;
						if (elapsed >= maxWait) return resolve('timeout');
						setTimeout(check, interval);
					}
					check();
				});
			})()`
			result, exp, err := runtime.Evaluate(mermaidWaitJS).
				WithAwaitPromise(true).
				Do(ctx)
			if exp != nil {
				slog.Warn("mermaid wait exception", slog.String("text", exp.Text))
			}
			if err == nil && result != nil && result.Value != nil {
				status := string(result.Value)
				if status != `"none"` {
					slog.Debug("mermaid rendering status", slog.String("result", status))
				}
			}
			if err != nil {
				return fmt.Errorf("mermaid wait evaluation: %w", err)
			}
			return nil
		}),
		// Wait for all @font-face sources to finish loading.
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(`document.fonts.ready.then(() => 'ok')`).
				WithAwaitPromise(true).
				Do(ctx)
			if exp != nil {
				slog.Warn("font loading exception", slog.String("text", exp.Text))
			}
			if err != nil {
				return fmt.Errorf("font-face loading wait: %w", err)
			}
			return nil
		}),
		// Log font loading diagnostics for CJK debugging.
		chromedp.ActionFunc(func(ctx context.Context) error {
			result, _, err := runtime.Evaluate(
				`(function() {` +
					`var diag = {` +
					`  fontsLoaded: document.fonts.size,` +
					`  cjkAvailable: document.fonts.check('16px "CJK-Embedded"'),` +
					`  hasChinese: /[\u4e00-\u9fff]/.test(document.body.innerText.substring(0,1000)),` +
					`  bodyFont: getComputedStyle(document.body).fontFamily.substring(0,150),` +
					`  codeFont: document.querySelector('code') ? getComputedStyle(document.querySelector('code')).fontFamily.substring(0,150) : 'no-code-element'` +
					`};` +
					// Check all loaded font faces for CJK-related entries
					`var cjkFonts = [];` +
					`document.fonts.forEach(function(f) {` +
					`  if (f.family.indexOf('CJK') >= 0 || f.status !== 'unloaded') {` +
					`    cjkFonts.push(f.family + ':' + f.status);` +
					`  }` +
					`});` +
					`diag.fontFaces = cjkFonts.join(', ');` +
					`return JSON.stringify(diag);` +
					`})()`).Do(ctx)
			if err == nil && result != nil && result.Value != nil {
				slog.Info("PDF font diagnostics", slog.String("info", string(result.Value)))
			}
			return nil // non-fatal
		}),
		// Force a full compositor paint pass before generating the PDF.
		chromedp.ActionFunc(func(ctx context.Context) error {
			var buf []byte
			return chromedp.CaptureScreenshot(&buf).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cmd := page.PrintToPDF().
				WithPaperWidth(mmToInch(g.pageWidth)).
				WithPaperHeight(mmToInch(g.pageHeight)).
				WithMarginLeft(mmToInch(g.marginLeft)).
				WithMarginRight(mmToInch(g.marginRight)).
				WithMarginTop(mmToInch(g.marginTop)).
				WithMarginBottom(mmToInch(g.marginBottom)).
				WithPrintBackground(g.printBackground).
				WithDisplayHeaderFooter(g.displayHeaderFooter).
				WithPreferCSSPageSize(true).
				WithGenerateDocumentOutline(g.generateDocumentOutline).
				WithGenerateTaggedPDF(g.generateTaggedPDF)
			if g.displayHeaderFooter {
				if g.headerTemplate != "" {
					cmd = cmd.WithHeaderTemplate(g.headerTemplate)
				} else {
					cmd = cmd.WithHeaderTemplate("<span></span>")
				}
				if g.footerTemplate != "" {
					cmd = cmd.WithFooterTemplate(g.footerTemplate)
				}
			}
			pdfBuf, _, err = cmd.Do(ctx)
			if err != nil {
				return fmt.Errorf("chromedp print to PDF: %w", err)
			}
			return nil
		}),
	)
	if err != nil {
		// Only try CLI fallback for file:// URLs.
		if htmlPath, ok := strings.CutPrefix(pageURL, "file://"); ok {
			if fallbackErr := generatePDFViaChromeCLI(chromePath, runtimeDirs, htmlPath, outputPath); fallbackErr == nil {
				return nil
			}
		}
		details := strings.TrimSpace(chromeOutput.String())
		if details != "" {
			return fmt.Errorf("failed to generate PDF with Chrome at %q: %w\nchrome output:\n%s", chromePath, err, details)
		}
		return fmt.Errorf("failed to generate PDF with Chrome at %q: %w", chromePath, err)
	}

	if err := os.WriteFile(outputPath, pdfBuf, 0o644); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}

	return nil
}

// checkChromiumAvailable verifies that Chrome or Chromium is installed.
// It first checks the MDPRESS_CHROME_PATH environment variable, then looks
// for common Chrome/Chromium executables in PATH and standard install locations.
func (g *Generator) checkChromiumAvailable() error {
	if _, err := resolveChromiumPath(); err != nil {
		return fmt.Errorf("chromium not available: %w", err)
	}
	return nil
}

// CheckChromiumAvailable reports whether Chrome or Chromium is installed.
func CheckChromiumAvailable() error {
	return NewGenerator().checkChromiumAvailable()
}

func resolveChromiumPath() (string, error) {
	if envPath := strings.TrimSpace(os.Getenv("MDPRESS_CHROME_PATH")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("MDPRESS_CHROME_PATH is set to %q but the file does not exist", envPath)
	}
	if envPath := strings.TrimSpace(os.Getenv("CHROME_BIN")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
		return "", fmt.Errorf("CHROME_BIN is set to %q but the file does not exist", envPath)
	}

	for _, exe := range chromiumExecutableCandidates {
		if path, err := exec.LookPath(exe); err == nil {
			return path, nil
		}
	}

	for _, p := range chromiumMacPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", errors.New(
		"Chrome/Chromium was not found. Install one of the following:\n" +
			"  macOS:   brew install chromium or install Google Chrome\n" +
			"  Ubuntu:  sudo apt install chromium-browser\n" +
			"  Windows: install Google Chrome (https://www.google.com/chrome/)\n" +
			"  Or set MDPRESS_CHROME_PATH to a custom Chrome/Chromium path")
}

func chromiumAllocatorOptions(execPath string, runtime chromiumRuntimeDirs, output *bytes.Buffer) []chromedp.ExecAllocatorOption {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.ExecPath(execPath),
		chromedp.UserDataDir(runtime.userData),
		chromedp.Env(chromiumRuntimeEnv(runtime)...),
		chromedp.Flag("allow-file-access-from-files", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("noerrdialogs", true),
		// Disable font hinting so that CJK glyphs render correctly in headless mode.
		// Without this flag some Chrome builds apply hinting that breaks CJK outlines
		// and the characters appear as blank squares in the generated PDF.
		chromedp.Flag("font-render-hinting", "none"),
	)
	if output != nil {
		opts = append(opts, chromedp.CombinedOutput(output))
	}

	for name, value := range parseChromiumFlags(os.Getenv("CHROME_FLAGS")) {
		opts = append(opts, chromedp.Flag(name, value))
	}

	return opts
}

func parseChromiumFlags(raw string) map[string]any {
	flags := make(map[string]any)
	for _, item := range strings.Fields(raw) {
		flag, ok := strings.CutPrefix(item, "--")
		if !ok || flag == "" {
			continue
		}

		// Extract the flag name (before '=' if present)
		flagName := flag
		if parts := strings.SplitN(flag, "=", 2); len(parts) == 2 {
			flagName = parts[0]
		}

		// Validate against allowlist
		if !allowedChromiumFlags[flagName] {
			sanitized := newlineStripper.Replace(flagName)
			slog.Warn("Rejecting disallowed Chromium flag", slog.String("flag", sanitized))
			continue
		}

		// Parse the flag value
		if parts := strings.SplitN(flag, "=", 2); len(parts) == 2 {
			switch strings.ToLower(parts[1]) {
			case "true":
				flags[parts[0]] = true
			case "false":
				flags[parts[0]] = false
			default:
				flags[parts[0]] = parts[1]
			}
			continue
		}
		flags[flag] = true
	}
	return flags
}

// filterChromiumCLIFlags filters command-line arguments from CHROME_FLAGS environment variable
// to only include allowlisted flags, preventing command injection.
func filterChromiumCLIFlags(flagsString string) []string {
	var filtered []string
	for _, item := range strings.Fields(flagsString) {
		flag, ok := strings.CutPrefix(item, "--")
		if !ok || flag == "" {
			continue
		}

		// Extract the flag name (before '=' if present)
		flagName := flag
		if parts := strings.SplitN(flag, "=", 2); len(parts) == 2 {
			flagName = parts[0]
		}

		// Validate against allowlist
		if !allowedChromiumFlags[flagName] {
			sanitized := newlineStripper.Replace(flagName)
			slog.Warn("Rejecting disallowed Chromium flag", slog.String("flag", sanitized))
			continue
		}

		filtered = append(filtered, item)
	}
	return filtered
}

func prepareChromiumRuntimeDirs() (chromiumRuntimeDirs, error) {
	rootBase := filepath.Join(utils.CacheRootDir(), "chrome-runtime")
	if utils.CacheDisabled() {
		rootBase = filepath.Join(os.TempDir(), "mdpress-chrome-runtime")
	}
	if err := os.MkdirAll(rootBase, 0o755); err != nil {
		return chromiumRuntimeDirs{}, fmt.Errorf("create chrome runtime base dir: %w", err)
	}

	root, err := os.MkdirTemp(rootBase, "run-*")
	if err != nil {
		return chromiumRuntimeDirs{}, fmt.Errorf("create chrome runtime temp dir: %w", err)
	}

	runtime := chromiumRuntimeDirs{
		root:      root,
		homeDir:   filepath.Join(root, "home"),
		userData:  filepath.Join(root, "user-data"),
		tmpDir:    filepath.Join(root, "tmp"),
		xdgConfig: filepath.Join(root, "xdg-config"),
		xdgCache:  filepath.Join(root, "xdg-cache"),
		cleanup: func() {
			if err := os.RemoveAll(root); err != nil {
				slog.Debug("Failed to clean up Chrome runtime directory", slog.String("dir", root), slog.Any("error", err))
			}
		},
	}
	for _, dir := range []string{runtime.homeDir, runtime.userData, runtime.tmpDir, runtime.xdgConfig, runtime.xdgCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			runtime.cleanup()
			return chromiumRuntimeDirs{}, fmt.Errorf("create chrome runtime subdir %q: %w", filepath.Base(dir), err)
		}
	}
	return runtime, nil
}

func chromiumRuntimeEnv(runtime chromiumRuntimeDirs) []string {
	// Do not override HOME: on macOS Chrome uses NSHomeDirectory() (from the passwd
	// database, not the $HOME env var) for its font metrics cache. Overriding HOME
	// does not provide meaningful isolation because --user-data-dir already isolates
	// Chrome's profile, but it can prevent Chrome from finding cached font metrics,
	// causing CJK characters to appear blank in the generated PDF.
	// Do not override XDG_CACHE_HOME: fontconfig stores its glyph/font cache there;
	// pointing it at an empty directory forces a full rescan on every PDF run which
	// can exceed the rendering timeout and leave CJK characters unresolved.
	return []string{
		"TMPDIR=" + runtime.tmpDir,
		"XDG_CONFIG_HOME=" + runtime.xdgConfig,
	}
}

func generatePDFViaChromeCLI(chromePath string, runtime chromiumRuntimeDirs, htmlFilePath, outputPath string) error {
	fileURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(htmlFilePath),
	}).String()
	tmpOutput := outputPath + ".tmp"
	if err := os.Remove(tmpOutput); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Debug("Failed to remove temporary PDF output file", slog.String("file", tmpOutput), slog.Any("error", err))
	}

	args := []string{
		"--headless",
		"--disable-gpu",
		"--allow-file-access-from-files",
		"--disable-crash-reporter",
		"--noerrdialogs",
		"--print-to-pdf=" + tmpOutput,
		"--no-pdf-header-footer",
		"--user-data-dir=" + runtime.userData,
	}
	args = append(args, filterChromiumCLIFlags(os.Getenv("CHROME_FLAGS"))...)
	args = append(args, fileURL)

	ctx, cancel := context.WithTimeout(context.Background(), chromiumPrintTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, chromePath, args...)
	cmd.Env = append(os.Environ(), chromiumRuntimeEnv(runtime)...)
	// Limit captured output to prevent OOM from misbehaving Chrome process.
	const maxChromeOutput = 10 << 20 // 10 MB
	var combinedBuf bytes.Buffer
	lw := &utils.LimitedWriter{W: &combinedBuf, N: maxChromeOutput}
	cmd.Stdout = lw
	cmd.Stderr = lw
	err := cmd.Run()
	if err != nil {
		_ = os.Remove(tmpOutput) // clean up temp file on failure
		details := strings.TrimSpace(combinedBuf.String())
		if details != "" {
			return fmt.Errorf("chrome CLI fallback failed: %w\nchrome output:\n%s", err, details)
		}
		return fmt.Errorf("chrome CLI fallback failed: %w", err)
	}
	info, err := os.Stat(tmpOutput)
	if err != nil || info.Size() == 0 {
		_ = os.Remove(tmpOutput) // clean up empty/missing temp file
		return errors.New("chrome CLI fallback did not produce a PDF")
	}
	if err := os.Rename(tmpOutput, outputPath); err != nil {
		return fmt.Errorf("failed to finalize fallback PDF output: %w", err)
	}
	return nil
}
