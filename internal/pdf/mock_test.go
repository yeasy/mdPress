package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestMockGeneratorSatisfiesInterface verifies that MockGenerator implements PDFRenderer.
func TestMockGeneratorSatisfiesInterface(t *testing.T) {
	var _ PDFRenderer = (*MockGenerator)(nil)
}

// TestMockGeneratorWritesPDF verifies that MockGenerator creates a file with a valid PDF header.
func TestMockGeneratorWritesPDF(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")

	mock := &MockGenerator{}
	htmlContent := "<html><body>Test</body></html>"

	err := mock.Generate(htmlContent, outputPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("PDF file was not created: %v", err)
	}

	// Verify the file contains a valid PDF header
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read PDF file: %v", err)
	}

	expectedHeader := []byte("%PDF-1.4\n%%EOF\n")
	if !bytes.Equal(content, expectedHeader) {
		t.Errorf("PDF header mismatch.\nGot: %v\nExpected: %v", content, expectedHeader)
	}
}

// TestMockGeneratorRecordsArgs verifies that MockGenerator records HTML content and output path.
func TestMockGeneratorRecordsArgs(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")
	htmlContent := "<html><body>Test Content</body></html>"

	mock := &MockGenerator{}
	err := mock.Generate(htmlContent, outputPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !mock.GenerateCalled {
		t.Error("GenerateCalled should be true")
	}

	if mock.LastHTMLContent != htmlContent {
		t.Errorf("LastHTMLContent mismatch.\nGot: %q\nExpected: %q", mock.LastHTMLContent, htmlContent)
	}

	if mock.LastOutputPath != outputPath {
		t.Errorf("LastOutputPath mismatch.\nGot: %q\nExpected: %q", mock.LastOutputPath, outputPath)
	}
}

// TestMockGeneratorSimulatesError verifies that error simulation works.
func TestMockGeneratorSimulatesError(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.pdf")

	testError := "simulated error"
	mock := &MockGenerator{
		GenerateError: &mockError{message: testError},
	}

	err := mock.Generate("<html></html>", outputPath)
	if err == nil {
		t.Fatal("Expected error, but got nil")
	}

	if err.Error() != testError {
		t.Errorf("Error message mismatch.\nGot: %q\nExpected: %q", err.Error(), testError)
	}

	// Verify the call was still recorded despite the error
	if !mock.GenerateCalled {
		t.Error("GenerateCalled should be true even when error is returned")
	}
}

// mockError is a simple error type for testing.
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
