package typst

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Generator converts HTML/Markdown content to PDF using Typst.
type Generator struct {
	timeout      time.Duration
	pageSize     string
	marginLeft   string
	marginRight  string
	marginTop    string
	marginBottom string
	fontFamily   string
	fontSize     string
	lineHeight   float64
	language     string
	author       string
	title        string
	version      string
	date         string
}

// GeneratorOption customizes a Typst generator.
type GeneratorOption func(*Generator)

const (
	defaultTimeout    = 60 * time.Second
	defaultPageSize   = "A4"
	defaultMargin     = "20mm"
	defaultFontSize   = "12pt"
	defaultLineHeight = 1.6
)

// NewGenerator creates a Typst PDF generator.
func NewGenerator(opts ...GeneratorOption) *Generator {
	g := &Generator{
		timeout:      defaultTimeout,
		pageSize:     defaultPageSize,
		marginTop:    defaultMargin,
		marginBottom: defaultMargin,
		marginLeft:   defaultMargin,
		marginRight:  defaultMargin,
		fontSize:     defaultFontSize,
		lineHeight:   defaultLineHeight,
		language:     "en",
		date:         currentDate(),
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

// WithPageSize sets the page size (e.g., "A4", "Letter").
func WithPageSize(size string) GeneratorOption {
	return func(g *Generator) { g.pageSize = size }
}

// WithMargins sets page margins.
func WithMargins(left, right, top, bottom string) GeneratorOption {
	return func(g *Generator) {
		g.marginLeft = left
		g.marginRight = right
		g.marginTop = top
		g.marginBottom = bottom
	}
}

// WithFontFamily sets the font family.
func WithFontFamily(family string) GeneratorOption {
	return func(g *Generator) { g.fontFamily = family }
}

// WithFontSize sets the font size.
func WithFontSize(size string) GeneratorOption {
	return func(g *Generator) { g.fontSize = size }
}

// WithLineHeight sets the line height.
func WithLineHeight(h float64) GeneratorOption {
	return func(g *Generator) { g.lineHeight = h }
}

// WithLanguage sets the document language.
func WithLanguage(lang string) GeneratorOption {
	return func(g *Generator) { g.language = lang }
}

// WithAuthor sets the document author.
func WithAuthor(author string) GeneratorOption {
	return func(g *Generator) { g.author = author }
}

// WithTitle sets the document title.
func WithTitle(title string) GeneratorOption {
	return func(g *Generator) { g.title = title }
}

// WithVersion sets the document version.
func WithVersion(version string) GeneratorOption {
	return func(g *Generator) { g.version = version }
}

// Generate converts Markdown content to a PDF file via Typst.
func (g *Generator) Generate(markdownContent string, outputPath string) error {
	if outputPath == "" {
		return errors.New("output path cannot be empty")
	}
	if markdownContent == "" {
		return errors.New("markdown content cannot be empty")
	}

	// Check if typst command is available
	if err := g.checkTypstAvailable(); err != nil {
		return fmt.Errorf("check typst: %w", err)
	}

	// Convert Markdown to Typst syntax
	converter := &MarkdownToTypstConverter{}
	typstContent := converter.Convert(markdownContent)

	// Get page dimensions in Typst format
	width, height := getPageDimensions(g.pageSize)

	// Prepare template data
	templateData := TypstTemplateData{
		Title:        g.title,
		Author:       g.author,
		Date:         g.date,
		Version:      g.version,
		Language:     g.language,
		Content:      typstContent,
		PageWidth:    strings.TrimSuffix(width, "mm"),
		PageHeight:   strings.TrimSuffix(height, "mm"),
		MarginTop:    g.marginTop,
		MarginRight:  g.marginRight,
		MarginBottom: g.marginBottom,
		MarginLeft:   g.marginLeft,
		FontFamily:   g.fontFamily,
		FontSize:     g.fontSize,
		LineHeight:   g.lineHeight,
	}

	// Render the Typst document
	typstDocument, err := renderTypstDocument(templateData)
	if err != nil {
		return fmt.Errorf("failed to render Typst document: %w", err)
	}

	// Write to a temporary .typ file
	tmpDir := os.TempDir()
	f, err := os.CreateTemp(tmpDir, "mdpress-*.typ")
	if err != nil {
		return fmt.Errorf("failed to create temporary Typst file: %w", err)
	}
	tmpTypFile := f.Name()
	defer os.Remove(tmpTypFile)

	if _, err := f.WriteString(typstDocument); err != nil {
		f.Close() //nolint:errcheck
		return fmt.Errorf("failed to write temporary Typst file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary Typst file: %w", err)
	}

	// Compile the Typst file to PDF using 'typst compile'
	if err := g.compileToPDF(tmpTypFile, outputPath); err != nil {
		return fmt.Errorf("compile typst to PDF: %w", err)
	}

	return nil
}

// GenerateFromFile reads a Markdown file and generates a PDF.
func (g *Generator) GenerateFromFile(markdownFilePath string, outputPath string) error {
	if _, err := os.Stat(markdownFilePath); err != nil {
		return fmt.Errorf("markdown file does not exist: %w", err)
	}

	content, err := os.ReadFile(markdownFilePath)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	return g.Generate(string(content), outputPath)
}

// compileToPDF runs 'typst compile' to convert .typ to PDF.
func (g *Generator) compileToPDF(typFilePath, outputPath string) error {
	absTypPath, err := filepath.Abs(typFilePath)
	if err != nil {
		return fmt.Errorf("failed to resolve Typst file path: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output file path: %w", err)
	}

	// Run: typst compile <input.typ> <output.pdf>
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "typst", "compile", absTypPath, absOutputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details != "" {
			return fmt.Errorf("typst compile failed: %w\noutput:\n%s", err, details)
		}
		return fmt.Errorf("typst compile failed: %w", err)
	}

	// Verify that the PDF was created
	if _, err := os.Stat(absOutputPath); err != nil {
		return fmt.Errorf("typst compile succeeded but PDF file was not created at %s", absOutputPath)
	}

	return nil
}

// checkTypstAvailable verifies that the 'typst' command is available.
func (g *Generator) checkTypstAvailable() error {
	path, err := exec.LookPath("typst")
	if err != nil {
		return errors.New(
			"typst is not installed or not found in PATH.\n" +
				"Install Typst from https://github.com/typst/typst/releases\n" +
				"Or set it up with: cargo install typst-cli",
		)
	}
	// Verify it's actually a working typst binary by checking version
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("typst command found at %s but failed to run: %w", path, err)
	}
	return nil
}

// checkTypstAvailable checks Typst availability.
func checkTypstAvailable() error {
	return NewGenerator().checkTypstAvailable()
}
