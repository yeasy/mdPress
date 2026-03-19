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
	"github.com/chromedp/chromedp"
	"github.com/yeasy/mdpress/pkg/utils"
)

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
	return []string{
		"HOME=" + runtime.homeDir,
		"TMPDIR=" + runtime.tmpDir,
		"XDG_CONFIG_HOME=" + runtime.xdgConfig,
		"XDG_CACHE_HOME=" + runtime.xdgCache,
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
