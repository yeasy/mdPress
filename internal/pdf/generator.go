// Package pdf renders HTML documents to PDF using Chromium.
package pdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
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

// GeneratorOption customizes a PDF generator.
type GeneratorOption func(*Generator)

const (
	defaultTimeout    = 60 * time.Second
	defaultPageWidth  = 210.0 // A4
	defaultPageHeight = 297.0
	defaultMargin     = 20.0
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
	if _, err := os.Stat(htmlFilePath); err != nil {
		return fmt.Errorf("HTML file does not exist: %w", err)
	}

	// Ensure Chromium or Chrome is available.
	if err := g.checkChromiumAvailable(); err != nil {
		return err
	}

	// Create the chromedp context.
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Apply the timeout.
	ctx, cancel = context.WithTimeout(ctx, g.timeout)
	defer cancel()

	fileURL := "file://" + htmlFilePath

	// Convert millimeters to inches because PrintToPDF expects inches.
	mmToInch := func(mm float64) float64 { return mm / 25.4 }

	var pdfBuf []byte
	err := chromedp.Run(ctx,
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
		return fmt.Errorf("failed to generate PDF: %w", err)
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
	// Honor an explicit override from the environment.
	if envPath := os.Getenv("MDPRESS_CHROME_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return nil
		}
		return fmt.Errorf("MDPRESS_CHROME_PATH is set to %q but the file does not exist", envPath)
	}

	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
	}

	for _, exe := range candidates {
		if _, err := exec.LookPath(exe); err == nil {
			return nil
		}
	}

	// Check common macOS install locations.
	macPaths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	for _, p := range macPaths {
		if _, err := os.Stat(p); err == nil {
			return nil
		}
	}

	return fmt.Errorf(
		"Chrome/Chromium was not found. Install one of the following:\n" +
			"  macOS:   brew install chromium or install Google Chrome\n" +
			"  Ubuntu:  sudo apt install chromium-browser\n" +
			"  Windows: install Google Chrome (https://www.google.com/chrome/)\n" +
			"  Or set MDPRESS_CHROME_PATH to a custom Chrome/Chromium path")
}

// CheckChromiumAvailable reports whether Chrome or Chromium is installed.
func CheckChromiumAvailable() error {
	return NewGenerator().checkChromiumAvailable()
}
