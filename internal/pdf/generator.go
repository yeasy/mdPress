// Package pdf renders HTML documents to PDF using Chromium.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	// ── macOS system fonts (last resort) ───────────────────────────────────
	// Chrome may not embed Core Text–managed system fonts cleanly, but they
	// are better than no font at all on machines without user-installed CJK fonts.
	{
		family: "Hiragino Sans GB",
		paths:  []string{"/System/Library/Fonts/Hiragino Sans GB.ttc"},
	},
	{
		family: "Heiti SC",
		paths: []string{
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
		},
	},
	{
		family: "Songti SC",
		paths:  []string{"/System/Library/Fonts/Supplemental/Songti.ttc"},
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
func buildCJKFontFaceCSS() string {
	const cjkRange = "U+2E80-2EFF, U+3000-303F, U+3400-4DBF, U+4E00-9FFF, " +
		"U+F900-FAFF, U+FE30-FE4F, U+FF00-FFEF, " +
		"U+20000-2A6DF, U+2A700-2B73F, U+2B740-2B81F, U+2B820-2CEAF"

	// Find the first available CJK font file.
	var font cjkFontSource
	for _, c := range systemCJKFontCandidates {
		for _, p := range c.paths {
			if _, err := os.Stat(p); err == nil {
				font = cjkFontSource{path: p}
				break
			}
		}
		if font.path != "" {
			break
		}
	}
	if font.path == "" {
		return "" // no CJK font found; PDF may show blank squares for CJK characters
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
	return css.String()
}

func cjkFontSrc(src cjkFontSource) string {
	fontURL := fileURLForCSS(src.path)
	switch strings.ToLower(filepath.Ext(src.path)) {
	case ".ttc", ".otc":
		// Font collections require an explicit fragment to select a single face.
		// Per CSS Fonts, when the container has no custom fragment scheme, a
		// 1-based index is used, so "#1" refers to the first face in the collection.
		return fmt.Sprintf("url(%q) format(collection)", fontURL+"#1")
	case ".otf":
		return fmt.Sprintf("url(%q) format(opentype)", fontURL)
	case ".ttf":
		return fmt.Sprintf("url(%q) format(truetype)", fontURL)
	default:
		return fmt.Sprintf("url(%q)", fontURL)
	}
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
func injectCJKFontFaceCSS(htmlContent string) string {
	css := buildCJKFontFaceCSS()
	if css == "" {
		return htmlContent
	}
	block := "<style data-cjk-fonts=\"1\">\n" + css + "</style>\n"
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

// WithPrintBackground toggles background printing.
func WithPrintBackground(print bool) GeneratorOption {
	return func(g *Generator) { g.printBackground = print }
}

// WithHeaderFooter toggles header and footer rendering.
func WithHeaderFooter(enable bool) GeneratorOption {
	return func(g *Generator) { g.displayHeaderFooter = enable }
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
	htmlContent = injectCJKFontFaceCSS(htmlContent)

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
			pdfBuf, _, err = page.PrintToPDF().
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
				WithGenerateTaggedPDF(g.generateTaggedPDF).
				Do(ctx)
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
			_ = os.RemoveAll(root)
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
	_ = os.Remove(tmpOutput)

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
