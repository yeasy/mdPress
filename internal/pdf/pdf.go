// Package pdf renders HTML documents to PDF using Chromium.
package pdf

// PDFRenderer abstracts PDF generation so tests can use a mock.
type PDFRenderer interface {
	Generate(htmlContent string, outputPath string) error
	GenerateFromFile(htmlFilePath string, outputPath string) error
}
