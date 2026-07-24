// Package pdf renders HTML documents to PDF using Chromium.
package pdf

import "context"

// PDFRenderer abstracts PDF generation so tests can use a mock.
//
// Both methods take a context because rendering a book-sized document keeps
// Chrome busy for minutes. Without one, Ctrl+C could not reach the render in
// progress: the CLI's signal handler canceled a context nothing in here was
// watching, so an interrupted build sat on a frozen progress line until Chrome
// had finished the whole book anyway.
type PDFRenderer interface {
	Generate(ctx context.Context, htmlContent string, outputPath string) error
	GenerateFromFile(ctx context.Context, htmlFilePath string, outputPath string) error
}
