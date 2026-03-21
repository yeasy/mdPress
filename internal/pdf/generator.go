// Package pdf renders HTML documents to PDF using Chromium.
package pdf

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
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
			var paths []string
			for _, prefix := range []string{"/opt/homebrew/share/fonts", "/usr/local/share/fonts"} {
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
	css.WriteString("body {\n" +
		"  font-family: \"CJK-Embedded\", -apple-system, BlinkMacSystemFont, \"Segoe UI\",\n" +
		"    \"PingFang SC\", \"Hiragino Sans GB\", \"Heiti SC\", \"Heiti TC\",\n" +
		"    \"Microsoft YaHei\", \"Noto Sans SC\", \"Noto Sans CJK SC\",\n" +
		"    \"Source Han Sans SC\", \"WenQuanYi Micro Hei\",\n" +
		"    \"Roboto\", \"Droid Sans\", \"Helvetica Neue\", sans-serif;\n" +
		"}\n")
	return cjkFontResult{css: css.String(), fontPath: font.path, family: family}
}

func cjkFontSrc(src cjkFontSource) string {
	// Primary strategy: embed the font as a data: URI.
	// Chrome's headless PrintToPDF on macOS cannot reliably embed fonts loaded
	// via file:// @font-face into the PDF output (glyphs render on screen but
	// are silently dropped during PDF serialization).  Inlining the font bytes
	// as a data: URI guarantees Chrome has the raw font data in memory and can
	// embed the glyphs into the PDF.
	data, err := os.ReadFile(src.path)
	if err == nil {
		encoded := base64.StdEncoding.EncodeToString(data)
		mime := "font/ttf"
		switch strings.ToLower(filepath.Ext(src.path)) {
		case ".ttc", ".otc":
			mime = "font/collection"
		case ".otf":
			mime = "font/otf"
		case ".woff":
			mime = "font/woff"
		case ".woff2":
			mime = "font/woff2"
		}
		return fmt.Sprintf("url(data:%s;base64,%s)", mime, encoded)
	}
	// Fallback: file:// URL (may not work for PDF embedding on macOS).
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
	defaultTimeout    = 60 * time.Second
	defaultPageWidth  = 210.0 // A4
	defaultPageHeight = 297.0
	defaultMargin     = 20.0
)

// parseMarginString converts a margin string (e.g., "20mm", "1in", "2.5cm") to millimeters.
// If the input is empty or invalid, it returns the default margin value.
func parseMarginString(s string, defaultMM float64) float64 {
	if s == "" {
		return defaultMM
	}
	s = strings.TrimSpace(s)
	// Match number with optional unit (mm, cm, in, pt, px) - case insensitive
	re := regexp.MustCompile(`(?i)^([-+]?[\d.]+)\s*(mm|cm|in|pt|px)?$`)
	matches := re.FindStringSubmatch(s)
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

// Generate renders an HTML string to a PDF file.
func (g *Generator) Generate(htmlContent string, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}
	if htmlContent == "" {
		return fmt.Errorf("HTML content cannot be empty")
	}

	// Inject @font-face rules for CJK system fonts before writing the HTML
	// to disk.  Injecting into the HTML (rather than via JS after page load)
	// ensures Chrome's font-matching pass sees the rules during initial layout.
	htmlContent = injectCJKFontFaceCSS(htmlContent, slog.Default())

	// Write the HTML to a temporary file first.
	tmpFile, err := os.CreateTemp("", "mdpress-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	tmpFile.Close() //nolint:errcheck

	return g.GenerateFromFile(tmpPath, outputPath)
}

// GenerateFromFile renders a local HTML file to PDF.
func (g *Generator) GenerateFromFile(htmlFilePath string, outputPath string) error {
	absHTMLPath, err := filepath.Abs(htmlFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve HTML file path: %w", err)
	}
	if _, err := os.Stat(absHTMLPath); err != nil {
		return fmt.Errorf("HTML file does not exist: %w", err)
	}

	chromePath, err := resolveChromiumPath()
	if err != nil {
		return err
	}

	runtimeDirs, err := prepareChromiumRuntimeDirs()
	if err != nil {
		return fmt.Errorf("failed to prepare Chrome runtime directories: %w", err)
	}
	defer runtimeDirs.cleanup()

	var chromeOutput bytes.Buffer
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), chromiumAllocatorOptions(chromePath, runtimeDirs, &chromeOutput)...)
	defer cancel()

	// Create the chromedp context.
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Apply the timeout.
	ctx, cancel = context.WithTimeout(ctx, g.timeout)
	defer cancel()

	fileURL := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absHTMLPath),
	}).String()

	// Convert millimeters to inches because PrintToPDF expects inches.
	mmToInch := func(mm float64) float64 { return mm / 25.4 }

	var pdfBuf []byte
	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		// Wait for all @font-face sources (including file:// URLs) to finish
		// loading.  document.fonts.ready is a Promise that resolves only after
		// all pending font resources have been fetched and parsed, ensuring
		// the CJK glyphs are available to Skia when PrintToPDF runs.
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(`document.fonts.ready.then(() => 'ok')`).
				WithAwaitPromise(true).
				Do(ctx)
			if exp != nil {
				// Non-fatal: log but do not abort; the screenshot step below
				// provides an additional render-sync barrier.
				_ = exp
			}
			return err
		}),
		// Force a full compositor paint pass before generating the PDF.
		// Even after document.fonts.ready resolves, calling CaptureScreenshot
		// ensures the compositor has produced a fully-rendered frame with all
		// font metrics applied — a second barrier against timing edge cases.
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
					// Empty header to avoid Chrome's default header.
					cmd = cmd.WithHeaderTemplate("<span></span>")
				}
				if g.footerTemplate != "" {
					cmd = cmd.WithFooterTemplate(g.footerTemplate)
				}
			}
			pdfBuf, _, err = cmd.Do(ctx)
			return err
		}),
	)
	if err != nil {
		if fallbackErr := generatePDFViaChromeCLI(chromePath, runtimeDirs, absHTMLPath, outputPath); fallbackErr == nil {
			return nil
		} else {
			details := strings.TrimSpace(chromeOutput.String())
			if details != "" {
				return fmt.Errorf("failed to generate PDF with Chrome at %q (flags: %s): %w\nchrome output:\n%s\nfallback error: %v", chromePath, strings.TrimSpace(os.Getenv("CHROME_FLAGS")), err, details, fallbackErr)
			}
			return fmt.Errorf("failed to generate PDF with Chrome at %q (flags: %s): %w\nfallback error: %v", chromePath, strings.TrimSpace(os.Getenv("CHROME_FLAGS")), err, fallbackErr)
		}
	}

	// Write the generated PDF bytes.
	if err := os.WriteFile(outputPath, pdfBuf, 0644); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}

	return nil
}

// checkChromiumAvailable verifies that Chrome or Chromium is installed.
// It first checks the MDPRESS_CHROME_PATH environment variable, then looks
// for common Chrome/Chromium executables in PATH and standard install locations.
func (g *Generator) checkChromiumAvailable() error {
	_, err := resolveChromiumPath()
	return err
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

	return "", fmt.Errorf(
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

func parseChromiumFlags(raw string) map[string]interface{} {
	flags := make(map[string]interface{})
	for _, item := range strings.Fields(raw) {
		if !strings.HasPrefix(item, "--") {
			continue
		}
		flag := strings.TrimPrefix(item, "--")
		if flag == "" {
			continue
		}
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

func prepareChromiumRuntimeDirs() (chromiumRuntimeDirs, error) {
	rootBase := filepath.Join(utils.CacheRootDir(), "chrome-runtime")
	if utils.CacheDisabled() {
		rootBase = filepath.Join(os.TempDir(), "mdpress-chrome-runtime")
	}
	if err := os.MkdirAll(rootBase, 0o755); err != nil {
		return chromiumRuntimeDirs{}, err
	}

	root, err := os.MkdirTemp(rootBase, "run-*")
	if err != nil {
		return chromiumRuntimeDirs{}, err
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
				slog.Debug("Failed to clean up Chrome runtime directory", slog.String("dir", root), slog.String("error", err.Error()))
			}
		},
	}
	for _, dir := range []string{runtime.homeDir, runtime.userData, runtime.tmpDir, runtime.xdgConfig, runtime.xdgCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			runtime.cleanup()
			return chromiumRuntimeDirs{}, err
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
	if err := os.Remove(tmpOutput); err != nil {
		slog.Debug("Failed to remove temporary PDF output file", slog.String("file", tmpOutput), slog.String("error", err.Error()))
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
	args = append(args, strings.Fields(os.Getenv("CHROME_FLAGS"))...)
	args = append(args, fileURL)

	cmd := exec.Command(chromePath, args...)
	cmd.Env = append(os.Environ(), chromiumRuntimeEnv(runtime)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details != "" {
			return fmt.Errorf("chrome CLI fallback failed: %w\nchrome output:\n%s", err, details)
		}
		return fmt.Errorf("chrome CLI fallback failed: %w", err)
	}
	info, err := os.Stat(tmpOutput)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("chrome CLI fallback did not produce a PDF")
	}
	if err := os.Rename(tmpOutput, outputPath); err != nil {
		return fmt.Errorf("failed to finalize fallback PDF output: %w", err)
	}
	return nil
}
