package pdf

import (
	"fmt"
	"os"
)

// Compile-time interface check.
var _ PDFRenderer = (*mockGenerator)(nil)

// mockGenerator is a test double that writes a minimal valid PDF header
// without requiring Chromium. Use it in tests to verify PDF generation
// flow without external dependencies.
type mockGenerator struct {
	GenerateCalled  bool
	LastHTMLContent string
	LastOutputPath  string
	GenerateError   error // Set this to simulate errors
}

// Generate records the call arguments and writes a minimal PDF file.
func (m *mockGenerator) Generate(htmlContent string, outputPath string) error {
	m.GenerateCalled = true
	m.LastHTMLContent = htmlContent
	m.LastOutputPath = outputPath
	if m.GenerateError != nil {
		return m.GenerateError
	}
	// Write a minimal PDF file (just the header)
	return os.WriteFile(outputPath, []byte("%PDF-1.4\n%%EOF\n"), 0644)
}

// GenerateFromFile reads the HTML file and calls Generate with its content.
func (m *mockGenerator) GenerateFromFile(htmlFilePath string, outputPath string) error {
	content, err := os.ReadFile(htmlFilePath)
	if err != nil {
		return fmt.Errorf("read HTML for mock PDF: %w", err)
	}
	return m.Generate(string(content), outputPath)
}
