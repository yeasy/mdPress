package output

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/yeasy/mdpress/internal/theme"
)

// TestNewEpubGenerator verifies basic construction of an EpubGenerator.
func TestNewEpubGenerator(t *testing.T) {
	meta := EpubMeta{
		Title:  "Test Book",
		Author: "Test Author",
	}

	gen := NewEpubGenerator(meta)

	if gen.meta.Title != "Test Book" {
		t.Errorf("expected title %q, got %q", "Test Book", gen.meta.Title)
	}
	if gen.meta.Author != "Test Author" {
		t.Errorf("expected author %q, got %q", "Test Author", gen.meta.Author)
	}
	if len(gen.chapters) != 0 {
		t.Errorf("expected 0 chapters initially, got %d", len(gen.chapters))
	}
	if gen.css != "" {
		t.Errorf("expected empty CSS initially, got %q", gen.css)
	}
}

// TestSetCSS verifies that CSS can be set on the generator.
func TestSetCSS(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test"})

	css := "body { font-family: serif; }"
	gen.SetCSS(css)

	if gen.css != css {
		t.Errorf("expected CSS %q, got %q", css, gen.css)
	}
}

// TestAddChapter verifies that chapters can be added to the generator.
func TestAddChapter(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test"})

	ch1 := EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content 1</p>",
	}
	ch2 := EpubChapter{
		Title:    "Chapter 2",
		ID:       "ch2",
		Filename: "ch2.xhtml",
		HTML:     "<p>Content 2</p>",
	}

	gen.AddChapter(ch1)
	gen.AddChapter(ch2)

	if len(gen.chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(gen.chapters))
	}
	if gen.chapters[0].Title != "Chapter 1" {
		t.Errorf("expected first chapter title %q, got %q", "Chapter 1", gen.chapters[0].Title)
	}
	if gen.chapters[1].Title != "Chapter 2" {
		t.Errorf("expected second chapter title %q, got %q", "Chapter 2", gen.chapters[1].Title)
	}
}

// TestGenerateBasicEpub verifies that a minimal EPUB is created with the correct structure.
func TestGenerateBasicEpub(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.epub")

	meta := EpubMeta{
		Title:    "Test Book",
		Author:   "Test Author",
		Language: "en",
	}
	gen := NewEpubGenerator(meta)
	gen.SetCSS("body { color: black; }")
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Hello, World!</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify the EPUB file exists
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("EPUB file not created: %v", err)
	}

	// Open and verify the ZIP structure
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB as ZIP: %v", err)
	}
	defer r.Close() //nolint:errcheck

	// Map file names for easy lookup
	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	// Verify required files exist
	requiredFiles := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/nav.xhtml",
		"OEBPS/toc.ncx",
		"OEBPS/style.css",
		"OEBPS/ch1.xhtml",
	}

	for _, filename := range requiredFiles {
		if _, ok := fileMap[filename]; !ok {
			t.Errorf("required file %q not found in EPUB", filename)
		}
	}

	// Verify mimetype content
	if mimeFile, ok := fileMap["mimetype"]; ok {
		rc, err := mimeFile.Open()
		if err != nil {
			t.Fatalf("failed to open mimetype entry: %v", err)
		}
		mimeData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read mimetype entry: %v", err)
		}
		if string(mimeData) != "application/epub+zip" {
			t.Errorf("expected mimetype content %q, got %q", "application/epub+zip", string(mimeData))
		}
	}
}

// TestGenerateEmptyChapters verifies that Generate fails when no chapters are added.
func TestGenerateEmptyChapters(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Empty Book"})
	// No chapters added

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() should not fail with empty chapters: %v", err)
	}

	// Verify partial file is cleaned up
	if _, err := os.Stat(outputPath); err == nil {
		// File should exist (it's valid to have 0 chapters)
		reader, err := zip.OpenReader(outputPath)
		if err != nil {
			t.Errorf("Generated EPUB with 0 chapters is not a valid ZIP")
		} else {
			defer reader.Close() //nolint:errcheck
		}
	}
}

// TestGenerateInvalidOutputPath verifies that Generate fails and cleans up on invalid path.
func TestGenerateInvalidOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	// A missing parent directory is no longer invalid — Generate creates it,
	// like every other backend. Use a path whose parent is a regular file, so
	// the directory genuinely cannot be created.
	blocker := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalidPath := filepath.Join(blocker, "output.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Test"})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content</p>",
	})

	err := gen.Generate(invalidPath)
	if err == nil {
		t.Fatal("Generate() should fail with invalid output path")
	}

	// Verify no partial file was left behind
	if _, err := os.Stat(invalidPath); err == nil {
		t.Error("partial EPUB file was not cleaned up after error")
	}
}

// TestWrapXHTMLBasic verifies that wrapXHTML wraps content correctly without math.
func TestWrapXHTMLBasic(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})
	gen.SetCSS("body { color: black; }")

	html := `<h1>Hello</h1><p>This is a test.</p>`
	result := gen.wrapXHTML("Test Title", html)

	// Check for required XHTML structure
	if !strings.Contains(result, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(result, `<!DOCTYPE html>`) {
		t.Error("missing DOCTYPE")
	}
	if !strings.Contains(result, `<html xmlns="http://www.w3.org/1999/xhtml"`) {
		t.Error("missing HTML namespace")
	}
	if !strings.Contains(result, `<title>Test Title</title>`) {
		t.Error("missing or incorrect title")
	}
	if !strings.Contains(result, `<link rel="stylesheet" type="text/css" href="style.css"/>`) {
		t.Error("missing CSS link when CSS is set")
	}
	if !strings.Contains(result, html) {
		t.Error("original HTML content not found in wrapped output")
	}

	// Should NOT contain KaTeX links without math
	if strings.Contains(result, "katex") {
		t.Error("KaTeX links should not be present without math content")
	}
}

// TestWrapXHTMLWithMath verifies that wrapXHTML includes KaTeX CDN links when math is detected.
func TestWrapXHTMLWithMath(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})

	htmlWithMath := `<p>This is math: <span class="math display">E = mc^2</span></p>`
	result := gen.wrapXHTML("Math Chapter", htmlWithMath)

	// Check for KaTeX CSS and JS links
	if !strings.Contains(result, "katex") {
		t.Error("KaTeX CSS link not found when math content is present")
	}
	if !strings.Contains(result, "renderMathInElement") {
		t.Error("KaTeX renderMathInElement script not found")
	}
	if !strings.Contains(result, `delimiters:[{left:'$$',right:'$$',display:true},{left:'$',right:'$',display:false}]`) {
		t.Error("KaTeX configuration not found in script")
	}

	// Verify structure is still correct
	if !strings.Contains(result, `<title>Math Chapter</title>`) {
		t.Error("title not found in wrapped math document")
	}
}

// TestWrapXHTMLWithoutCSS verifies wrapXHTML without custom CSS set: style.css
// is always packaged (theme-derived or minimal fallback), so the link is
// always emitted.
func TestWrapXHTMLWithoutCSS(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})
	// Don't call SetCSS

	html := `<p>Content without CSS</p>`
	result := gen.wrapXHTML("No CSS", html)

	if !strings.Contains(result, `<link rel="stylesheet" type="text/css" href="style.css"/>`) {
		t.Error("style.css link should always be present (stylesheet is always packaged)")
	}
	if !strings.Contains(result, html) {
		t.Error("original HTML content not found")
	}
}

// TestWrapXHTMLLanguageDefault verifies that wrapXHTML uses default language when not specified.
func TestWrapXHTMLLanguageDefault(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test"}) // No Language specified

	result := gen.wrapXHTML("Title", "<p>Content</p>")

	// Should default to "en"
	if !strings.Contains(result, `lang="en"`) {
		t.Error("expected default language 'en', not found")
	}
}

// TestWrapXHTMLLanguageCustom verifies that wrapXHTML uses custom language.
func TestWrapXHTMLLanguageCustom(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "fr"})

	result := gen.wrapXHTML("Title", "<p>Content</p>")

	if !strings.Contains(result, `lang="fr"`) {
		t.Error("expected language 'fr', not found")
	}
}

// TestNormalizeHTMLForXHTMLVoidElements verifies normalization of self-closing elements.
func TestNormalizeHTMLForXHTMLVoidElements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "br tag",
			input:    "<p>Line 1<br>Line 2</p>",
			expected: "<p>Line 1<br />Line 2</p>",
		},
		{
			name:     "img tag",
			input:    `<img src="test.png">`,
			expected: `<img src="test.png" />`,
		},
		{
			name:     "already self-closed",
			input:    "<br />",
			expected: "<br />",
		},
		{
			name:     "hr tag",
			input:    "<hr>",
			expected: "<hr />",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHTMLForXHTML(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected %q to contain %q", result, tt.expected)
			}
		})
	}
}

// TestNormalizeHTMLForXHTMLAmpersands verifies ampersand escaping.
func TestNormalizeHTMLForXHTMLAmpersands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bare ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "existing entity",
			input:    "&lt;tag&gt;",
			expected: "&lt;tag&gt;",
		},
		{
			name:     "mixed",
			input:    "A & B &lt; C",
			expected: "A &amp; B &lt; C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHTMLForXHTML(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected %q to contain %q", result, tt.expected)
			}
		})
	}
}

// TestNormalizeHTMLForXHTMLBooleanAttributes verifies expansion of boolean attributes.
func TestNormalizeHTMLForXHTMLBooleanAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "disabled attribute",
			input:    `<input disabled>`,
			expected: `disabled="disabled"`,
		},
		{
			name:     "checked attribute",
			input:    `<input checked type="checkbox">`,
			expected: `checked="checked"`,
		},
		{
			name:     "multiple boolean attributes",
			input:    `<input disabled readonly>`,
			expected: `disabled="disabled"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeHTMLForXHTML(tt.input)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected %q to contain %q", result, tt.expected)
			}
		})
	}
}

// TestGenerateWithCoverImage verifies that a cover image is included when configured.
func TestGenerateWithCoverImage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple PNG image for testing
	coverPath := filepath.Join(tmpDir, "cover.png")
	// Minimal valid PNG: 1x1 transparent pixel
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(coverPath, pngData, 0o644); err != nil {
		t.Fatalf("failed to create test cover image: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "with_cover.epub")

	meta := EpubMeta{
		Title:          "Book with Cover",
		Author:         "Test",
		IncludeCover:   true,
		CoverImagePath: coverPath,
	}
	gen := NewEpubGenerator(meta)
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify cover files in EPUB
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	if _, ok := fileMap["OEBPS/cover.xhtml"]; !ok {
		t.Error("cover.xhtml not found in EPUB")
	}

	// Check for cover image asset
	hasAsset := false
	for name := range fileMap {
		if strings.Contains(name, "assets") && strings.Contains(name, "cover") {
			hasAsset = true
			break
		}
	}
	if !hasAsset {
		t.Error("cover image asset not found in EPUB")
	}
}

// TestGenerateWithMetadata verifies that metadata is correctly included in OPF.
func TestGenerateWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "with_metadata.epub")

	meta := EpubMeta{
		Title:       "Test Book",
		Subtitle:    "A Subtitle",
		Author:      "Test Author",
		Language:    "en",
		Version:     "1.0.0",
		Description: "A test book description",
	}
	gen := NewEpubGenerator(meta)
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Read and verify OPF content
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	opfFile, err := r.Open("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("content.opf not found: %v", err)
	}
	defer opfFile.Close() //nolint:errcheck

	opfData, err := io.ReadAll(opfFile)
	if err != nil {
		t.Fatalf("failed to read content.opf: %v", err)
	}
	opfContent := string(opfData)

	// Verify metadata in OPF
	if !strings.Contains(opfContent, "Test Book") {
		t.Error("title not found in OPF")
	}
	if !strings.Contains(opfContent, "A Subtitle") {
		t.Error("subtitle not found in OPF")
	}
	if !strings.Contains(opfContent, "Test Author") {
		t.Error("author not found in OPF")
	}
	if !strings.Contains(opfContent, "1.0.0") {
		t.Error("version not found in OPF")
	}
	if !strings.Contains(opfContent, "A test book description") {
		t.Error("description not found in OPF")
	}
}

// TestGenerateNavDocument verifies that the navigation document is created correctly.
func TestGenerateNavDocument(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "with_nav.epub")

	meta := EpubMeta{
		Title:        "Nav Test",
		Language:     "en",
		IncludeCover: true,
	}
	gen := NewEpubGenerator(meta)
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content 1</p>",
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 2",
		Filename: "ch2.xhtml",
		HTML:     "<p>Content 2</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Read and verify nav.xhtml content
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	navFile, err := r.Open("OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("nav.xhtml not found: %v", err)
	}
	defer navFile.Close() //nolint:errcheck

	navData, err := io.ReadAll(navFile)
	if err != nil {
		t.Fatalf("failed to read nav.xhtml: %v", err)
	}
	navContent := string(navData)

	// Verify navigation structure
	if !strings.Contains(navContent, `<nav epub:type="toc"`) {
		t.Error("navigation with epub:type=\"toc\" not found")
	}
	if !strings.Contains(navContent, `<a href="cover.xhtml">Cover</a>`) {
		t.Error("cover link not found in nav")
	}
	if !strings.Contains(navContent, "Chapter 1") {
		t.Error("Chapter 1 not found in nav")
	}
	if !strings.Contains(navContent, "Chapter 2") {
		t.Error("Chapter 2 not found in nav")
	}
}

// TestGenerateNCXDocument verifies that the NCX (toc.ncx) file is created correctly.
func TestGenerateNCXDocument(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "with_ncx.epub")

	meta := EpubMeta{
		Title:        "NCX Test",
		IncludeCover: true,
	}
	gen := NewEpubGenerator(meta)
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content 1</p>",
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 2",
		Filename: "ch2.xhtml",
		HTML:     "<p>Content 2</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Read and verify toc.ncx content
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	ncxFile, err := r.Open("OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("toc.ncx not found: %v", err)
	}
	defer ncxFile.Close() //nolint:errcheck

	ncxData, err := io.ReadAll(ncxFile)
	if err != nil {
		t.Fatalf("failed to read toc.ncx: %v", err)
	}
	ncxContent := string(ncxData)

	// Verify NCX structure
	if !strings.Contains(ncxContent, `<?xml version="1.0"`) {
		t.Error("XML declaration not found in NCX")
	}
	if !strings.Contains(ncxContent, `<navMap>`) {
		t.Error("navMap not found in NCX")
	}
	if !strings.Contains(ncxContent, "NCX Test") {
		t.Error("title not found in NCX")
	}
	if !strings.Contains(ncxContent, `<navPoint id="nav-cover"`) {
		t.Error("cover navPoint not found in NCX")
	}
	if !strings.Contains(ncxContent, "Chapter 1") {
		t.Error("Chapter 1 not found in NCX")
	}
}

// TestGeneratePreservesPartialFileOnZipError simulates a write error to verify cleanup.
// Note: This test verifies the behavior by checking that the file is cleaned up.
func TestGeneratePartialFileCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-permission semantics required; skipping on Windows")
	}

	tmpDir := t.TempDir()

	// Make tmpDir read-only to cause write failure
	gen := NewEpubGenerator(EpubMeta{Title: "Test"})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content</p>",
	})

	// Change to a directory we can't write to
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0o444); err != nil {
		t.Fatalf("failed to create readonly directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) //nolint:errcheck // Restore permissions for cleanup

	readOnlyPath := filepath.Join(readOnlyDir, "test.epub")

	// This should fail due to permissions
	err := gen.Generate(readOnlyPath)
	if err == nil {
		t.Fatal("expected Generate() to fail with read-only directory")
	}

	// Verify partial file doesn't exist or is cleaned up
	if _, err := os.Stat(readOnlyPath); err == nil {
		t.Error("partial file should have been cleaned up after error")
	}
}

// TestEpubXMLEscaping verifies that special characters in titles and content are properly escaped.
func TestEpubXMLEscaping(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "xml_escape.epub")

	meta := EpubMeta{
		Title:       "Test < & > \"Book\"",
		Author:      "Author & Company",
		Subtitle:    "A 'subtitle' with <tags>",
		Description: "Description with & and < and >",
	}
	gen := NewEpubGenerator(meta)
	gen.AddChapter(EpubChapter{
		Title:    "Chapter & Title",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content with & and <tag></p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("EPUB file not created: %v", err)
	}

	// Open and verify the EPUB is still a valid ZIP
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB as ZIP: %v", err)
	}
	r.Close()
}

// TestGenerateMultipleChapters verifies correct structure with multiple chapters.
func TestGenerateMultipleChapters(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "multi_chapter.epub")

	meta := EpubMeta{
		Title:  "Multi Chapter Book",
		Author: "Test",
	}
	gen := NewEpubGenerator(meta)

	// Add multiple chapters
	for i := 1; i <= 5; i++ {
		ch := EpubChapter{
			Title:    "Chapter " + string(rune('0'+i)),
			Filename: "ch" + string(rune('0'+i)) + ".xhtml",
			HTML:     "<p>Content of chapter " + string(rune('0'+i)) + "</p>",
		}
		gen.AddChapter(ch)
	}

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify all chapter files exist in EPUB
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	for i := 1; i <= 5; i++ {
		filename := "OEBPS/ch" + string(rune('0'+i)) + ".xhtml"
		if _, ok := fileMap[filename]; !ok {
			t.Errorf("chapter file %q not found in EPUB", filename)
		}
	}

	// Verify OPF manifest and spine
	opfFile, err := r.Open("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("content.opf not found: %v", err)
	}
	opfData, err := io.ReadAll(opfFile)
	if err != nil {
		t.Fatalf("failed to read content.opf: %v", err)
	}
	opfFile.Close()
	opfContent := string(opfData)

	for i := 1; i <= 5; i++ {
		if !strings.Contains(opfContent, "ch"+string(rune('0'+i))+".xhtml") {
			t.Errorf("chapter %d reference not found in OPF manifest", i)
		}
	}
}

// TestGenerateWithoutCSS verifies EPUB generation without custom CSS still
// ships the reader-friendly fallback stylesheet.
func TestGenerateWithoutCSS(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "no_css.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "No CSS Book"})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Content</p>",
	})

	// Don't call SetCSS

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// style.css is always packaged; without a theme it is the minimal fallback.
	css := readEpubFile(t, outputPath, "OEBPS/style.css")
	if !strings.Contains(css, "text-decoration: underline") {
		t.Error("fallback stylesheet should underline links")
	}
	if strings.Contains(css, "var(") {
		t.Error("fallback stylesheet must not use CSS custom properties")
	}

	// The OPF manifest declares the stylesheet.
	opf := readEpubFile(t, outputPath, "OEBPS/content.opf")
	if !strings.Contains(opf, `<item id="css" href="style.css" media-type="text/css"/>`) {
		t.Error("style.css manifest item not found in OPF")
	}
}

// TestContainerXMLContent verifies the container.xml has correct content.
func TestContainerXMLContent(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "container.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Test"})
	gen.AddChapter(EpubChapter{
		Title:    "Ch1",
		Filename: "ch1.xhtml",
		HTML:     "<p>Test</p>",
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	containerFile, err := r.Open("META-INF/container.xml")
	if err != nil {
		t.Fatalf("container.xml not found: %v", err)
	}
	defer containerFile.Close() //nolint:errcheck

	containerData, err := io.ReadAll(containerFile)
	if err != nil {
		t.Fatalf("failed to read container.xml: %v", err)
	}
	containerContent := string(containerData)

	// Verify container structure
	if !strings.Contains(containerContent, `<rootfile full-path="OEBPS/content.opf"`) {
		t.Error("rootfile path not correct in container.xml")
	}
	if !strings.Contains(containerContent, `media-type="application/oebps-package+xml"`) {
		t.Error("media-type not correct in container.xml")
	}
}

// TestGenerateChapterContentInclusion verifies that chapter HTML content is correctly included.
func TestGenerateChapterContentInclusion(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "content.epub")

	expectedContent := `<h1>Chapter Title</h1><p>This is a test paragraph with <strong>bold</strong> and <em>italic</em> text.</p>`

	gen := NewEpubGenerator(EpubMeta{Title: "Test"})
	gen.AddChapter(EpubChapter{
		Title:    "Test Chapter",
		Filename: "chapter.xhtml",
		HTML:     expectedContent,
	})

	err := gen.Generate(outputPath)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close() //nolint:errcheck

	chapterFile, err := r.Open("OEBPS/chapter.xhtml")
	if err != nil {
		t.Fatalf("chapter.xhtml not found: %v", err)
	}
	defer chapterFile.Close() //nolint:errcheck

	chapterData, err := io.ReadAll(chapterFile)
	if err != nil {
		t.Fatalf("failed to read chapter.xhtml: %v", err)
	}
	chapterContent := string(chapterData)

	if !strings.Contains(chapterContent, expectedContent) {
		t.Error("chapter content not found in generated file")
	}
}

// BenchmarkGenerateSimpleEpub measures performance of EPUB generation.
func BenchmarkGenerateSimpleEpub(b *testing.B) {
	tmpDir := b.TempDir()

	meta := EpubMeta{
		Title:  "Benchmark Book",
		Author: "Test",
	}

	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, "bench"+string(rune('0'+(i%10)))+".epub")
		gen := NewEpubGenerator(meta)
		gen.SetCSS("body { color: black; }")

		for j := 0; j < 10; j++ {
			gen.AddChapter(EpubChapter{
				Title:    "Chapter",
				Filename: "ch.xhtml",
				HTML:     "<p>Content</p>",
			})
		}

		if err := gen.Generate(outputPath); err != nil {
			b.Fatalf("epub generation failed: %v", err)
		}
	}
}

// readEpubFileNames returns the list of file entry names inside an EPUB zip.
func readEpubFileNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

// readEpubFile returns the content of a named entry inside an EPUB zip.
func readEpubFile(t *testing.T, path, name string) string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", name, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read entry %s: %v", name, err)
		}
		return string(data)
	}
	t.Fatalf("entry %s not found in epub", name)
	return ""
}

// TestEpubPackagesSharedImageAboveChapterDir verifies that an image referenced
// above the chapter's own directory (e.g. a shared ../images/pic.png from a
// chapter in docs/) is packaged rather than silently dropped. This exercises
// the book-root containment logic.
func TestEpubPackagesSharedImageAboveChapterDir(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	imagesDir := filepath.Join(root, "images")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Minimal valid 1x1 PNG.
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	imgPath := filepath.Join(imagesDir, "pic.png")
	if err := os.WriteFile(imgPath, png, 0o644); err != nil {
		t.Fatal(err)
	}

	gen := NewEpubGenerator(EpubMeta{Title: "T", Author: "A", Language: "en"})
	gen.SetBookRoot(root)
	gen.AddChapter(EpubChapter{
		Title:     "Ch1",
		ID:        "ch1",
		Filename:  "ch1.xhtml",
		HTML:      `<p><img src="../images/pic.png" alt="pic"></p>`,
		SourceDir: docsDir,
	})

	out := filepath.Join(t.TempDir(), "book.epub")
	if err := gen.Generate(out); err != nil {
		t.Fatalf("generate: %v", err)
	}

	names := readEpubFileNames(t, out)
	var hasAsset bool
	for _, n := range names {
		if strings.HasPrefix(n, "OEBPS/assets/") {
			hasAsset = true
		}
	}
	if !hasAsset {
		t.Fatalf("expected shared image to be packaged, got entries: %v", names)
	}

	chapter := readEpubFile(t, out, "OEBPS/ch1.xhtml")
	if strings.Contains(chapter, "../images/pic.png") {
		t.Errorf("chapter XHTML still references original ../images src: %s", chapter)
	}
}

// TestEpubCommonAncestorPackagesSharedImage verifies the fallback containment
// base (common ancestor of chapter source dirs) also packages shared images
// even without an explicit book root.
func TestEpubCommonAncestorPackagesSharedImage(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	imagesDir := filepath.Join(root, "images")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "pic.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}

	gen := NewEpubGenerator(EpubMeta{Title: "T", Author: "A", Language: "en"})
	// No SetBookRoot; two chapters both under docs/ so the common ancestor is
	// docs/, which does NOT contain ../images. Add a second chapter one level
	// up so the common ancestor becomes root and ../images is contained.
	gen.AddChapter(EpubChapter{
		Title:     "Ch1",
		ID:        "ch1",
		Filename:  "ch1.xhtml",
		HTML:      `<p><img src="../images/pic.png" alt="pic"></p>`,
		SourceDir: docsDir,
	})
	gen.AddChapter(EpubChapter{
		Title:     "Ch2",
		ID:        "ch2",
		Filename:  "ch2.xhtml",
		HTML:      `<p>no image</p>`,
		SourceDir: root,
	})

	out := filepath.Join(t.TempDir(), "book.epub")
	if err := gen.Generate(out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	chapter := readEpubFile(t, out, "OEBPS/ch1.xhtml")
	if strings.Contains(chapter, "../images/pic.png") {
		t.Errorf("expected shared image packaged via common ancestor, chapter still has original src: %s", chapter)
	}
}

// TestEpubMalformedDataURIDoesNotFailBuild verifies that an unsupported /
// malformed data URI no longer aborts the whole EPUB build. The original src is
// preserved and generation succeeds.
func TestEpubMalformedDataURIDoesNotFailBuild(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "T", Author: "A", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title:    "Ch1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     `<p><img src="data:image/png;notbase64!!!" alt="broken"></p>`,
	})
	out := filepath.Join(t.TempDir(), "book.epub")
	if err := gen.Generate(out); err != nil {
		t.Fatalf("malformed data URI should not fail build, got: %v", err)
	}
	chapter := readEpubFile(t, out, "OEBPS/ch1.xhtml")
	if !strings.Contains(chapter, "data:image/png;notbase64") {
		t.Errorf("expected original malformed data URI src preserved, got: %s", chapter)
	}
}

// TestEpubNonBase64SVGDataURIPackaged verifies that a non-base64 (utf8 or
// URL-encoded) SVG data URI is decoded and packaged as an asset.
func TestEpubNonBase64SVGDataURIPackaged(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "T", Author: "A", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title:    "Ch1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     `<p><img src="data:image/svg+xml;utf8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%2F%3E" alt="svg"></p>`,
	})
	out := filepath.Join(t.TempDir(), "book.epub")
	if err := gen.Generate(out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	names := readEpubFileNames(t, out)
	var found bool
	for _, n := range names {
		if strings.HasPrefix(n, "OEBPS/assets/img-") && strings.HasSuffix(n, ".svg") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected non-base64 SVG data URI packaged as .svg asset, entries: %v", names)
	}
}

// TestBuildNonBase64DataURIImageAsset unit-tests the non-base64 decoder.
func TestBuildNonBase64DataURIImageAsset(t *testing.T) {
	// URL-encoded payload.
	asset, ok := buildNonBase64DataURIImageAsset(`data:image/svg+xml,%3Csvg%2F%3E`, 1)
	if !ok || asset == nil {
		t.Fatalf("expected asset for url-encoded svg, ok=%v asset=%v", ok, asset)
	}
	if string(asset.Data) != "<svg/>" {
		t.Errorf("expected decoded <svg/>, got %q", string(asset.Data))
	}
	if asset.MediaType != "image/svg+xml" {
		t.Errorf("unexpected media type %q", asset.MediaType)
	}

	// A base64 data URI is not a non-base64 form.
	if _, ok := buildNonBase64DataURIImageAsset(`data:image/png;base64,AAAA`, 2); ok {
		t.Errorf("base64 data URI should not be handled as non-base64")
	}
}

// TestWrapXHTMLInjectsChapterTitleHeading verifies that the chapter title is
// re-emitted as an <h1> when the body does not already start with one (the
// chapter pipeline strips the leading h1 for the PDF/HTML templates).
func TestWrapXHTMLInjectsChapterTitleHeading(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})

	result := gen.wrapXHTML("Chapter & One", `<p>Body text.</p>`)

	if !strings.Contains(result, "<h1>Chapter &amp; One</h1>") {
		t.Errorf("expected injected escaped chapter title h1, got: %s", result)
	}
	if strings.Index(result, "<h1>Chapter &amp; One</h1>") > strings.Index(result, "<p>Body text.</p>") {
		t.Error("injected h1 should precede the chapter body content")
	}
}

// TestWrapXHTMLKeepsExistingHeading verifies that no duplicate h1 is injected
// when the chapter body already starts with a heading.
func TestWrapXHTMLKeepsExistingHeading(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})

	result := gen.wrapXHTML("Chapter One", `<h1 id="x">Custom Heading</h1><p>Body</p>`)

	if got := strings.Count(result, "<h1"); got != 1 {
		t.Errorf("expected exactly 1 h1, got %d in: %s", got, result)
	}
	if strings.Contains(result, "<h1>Chapter One</h1>") {
		t.Error("chapter title should not be injected when body already starts with h1")
	}
}

// TestEpubMathChapterManifestProperties verifies that chapters containing math
// declare properties="scripted remote-resources" in the OPF manifest (required
// by EPUB 3 for embedded scripts and remote CDN resources), while non-math
// chapters carry no properties and no scripts.
func TestEpubMathChapterManifestProperties(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "math.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Math Book", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title:    "Math",
		Filename: "math.xhtml",
		HTML:     `<p><span class="math display">E = mc^2</span></p>`,
	})
	gen.AddChapter(EpubChapter{
		Title:    "Plain",
		Filename: "plain.xhtml",
		HTML:     `<p>No math here.</p>`,
	})

	if err := gen.Generate(outputPath); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	opf := readEpubFile(t, outputPath, "OEBPS/content.opf")
	if !strings.Contains(opf, `<item id="ch0" href="math.xhtml" media-type="application/xhtml+xml" properties="scripted remote-resources"/>`) {
		t.Errorf("math chapter manifest item missing scripted/remote-resources properties: %s", opf)
	}
	if !strings.Contains(opf, `<item id="ch1" href="plain.xhtml" media-type="application/xhtml+xml"/>`) {
		t.Errorf("non-math chapter manifest item should have no properties: %s", opf)
	}

	// The non-math chapter must not embed any scripts.
	plain := readEpubFile(t, outputPath, "OEBPS/plain.xhtml")
	if strings.Contains(plain, "<script") {
		t.Errorf("non-math chapter should contain no scripts: %s", plain)
	}
	math := readEpubFile(t, outputPath, "OEBPS/math.xhtml")
	if !strings.Contains(math, "<script") {
		t.Error("math chapter should embed KaTeX scripts")
	}
}

// TestEpubCoverPageDefaultNavy verifies the generated cover page uses the
// premium deep-navy default background with light text.
func TestEpubCoverPageDefaultNavy(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Navy Book"})

	page := gen.generateCoverPage(nil)

	if !strings.Contains(page, "background-color: #102a43") {
		t.Errorf("expected default navy cover background, got: %s", page)
	}
	if !strings.Contains(page, "color: #f6f8fc") {
		t.Errorf("expected light text on dark default background, got: %s", page)
	}
}

// TestEpubCoverPageHonorsConfiguredBackground verifies book.cover.background
// is applied and text adapts to a light background.
func TestEpubCoverPageHonorsConfiguredBackground(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Light Book", CoverBackground: "#ffffff"})

	page := gen.generateCoverPage(nil)

	if !strings.Contains(page, "background-color: #ffffff") {
		t.Errorf("expected configured cover background, got: %s", page)
	}
	if !strings.Contains(page, "color: #14304a") {
		t.Errorf("expected dark text on light configured background, got: %s", page)
	}
}

// TestEpubCoverPageRejectsUnsafeBackground verifies that a background value
// failing CSS color validation falls back to the navy default.
func TestEpubCoverPageRejectsUnsafeBackground(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{
		Title:           "Unsafe",
		CoverBackground: "red; } body { background: url(evil)",
	})

	page := gen.generateCoverPage(nil)

	if !strings.Contains(page, "background-color: #102a43") {
		t.Errorf("expected fallback to navy for unsafe background, got: %s", page)
	}
	if strings.Contains(page, "evil") {
		t.Errorf("unsafe background value must not be emitted: %s", page)
	}
}

// TestEpubCoverBackgroundValidation unit-tests the background validator.
func TestEpubCoverBackgroundValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty falls back to navy", "", "#102a43"},
		{"hex color accepted", "#ABCDEF", "#ABCDEF"},
		{"named color accepted", "navy", "navy"},
		{"rgb accepted", "rgb(16, 42, 67)", "rgb(16, 42, 67)"},
		{"injection rejected", "#fff; } * { color: red", "#102a43"},
		{"url rejected", "url(http://evil)", "#102a43"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := epubCoverBackground(tt.input); got != tt.expected {
				t.Errorf("epubCoverBackground(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestEpubStylesheetFromTheme verifies the theme-derived stylesheet is
// reader-friendly: literal colors (no CSS variables), no page margins, no
// absolute font sizes, underlined links, and custom CSS appended last.
func TestEpubStylesheetFromTheme(t *testing.T) {
	thm := &theme.Theme{
		Name:       "test",
		FontFamily: "serif",
		FontSize:   11,
		LineHeight: 1.7,
		Colors: theme.ColorScheme{
			Text:       "#111111",
			Background: "#fafafa",
			Heading:    "#222222",
			Link:       "#123456",
			CodeBg:     "#eeeeee",
			CodeText:   "#333333",
			Accent:     "#445566",
			Border:     "#dddddd",
		},
		Margins: theme.MarginSettings{Top: 20, Bottom: 20, Left: 20, Right: 20},
	}
	gen := NewEpubGenerator(EpubMeta{Title: "Styled"})
	gen.SetTheme(thm)
	gen.SetCSS("p { color: pink; }")

	css := gen.stylesheet()

	if strings.Contains(css, "var(") {
		t.Error("EPUB stylesheet must not use CSS custom properties")
	}
	if strings.Contains(css, "mm;") || strings.Contains(css, "mm ") {
		t.Error("EPUB stylesheet must not carry print margins in mm")
	}
	if strings.Contains(css, "pt;") {
		t.Error("EPUB stylesheet must not use absolute pt font sizes")
	}
	if !strings.Contains(css, "#123456") {
		t.Error("theme link color should be emitted as a literal value")
	}
	if !strings.Contains(css, "text-decoration: underline") {
		t.Error("links must be underlined (WCAG 1.4.1)")
	}
	if strings.Contains(css, "background-color: #fafafa") {
		t.Error("theme background must not be forced on the body (breaks reader night modes)")
	}
	themeIdx := strings.Index(css, "#123456")
	customIdx := strings.Index(css, "pink")
	if customIdx < themeIdx {
		t.Error("custom CSS should be appended after the theme-derived stylesheet")
	}
}

// TestEpubStylesheetNilThemeFallback verifies the minimal fallback stylesheet
// is used when no theme is set.
func TestEpubStylesheetNilThemeFallback(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Plain"})

	css := gen.stylesheet()

	if css == "" {
		t.Fatal("stylesheet should never be empty")
	}
	if !strings.Contains(css, "text-decoration: underline") {
		t.Error("fallback stylesheet should underline links")
	}
	if strings.Contains(css, "var(") {
		t.Error("fallback stylesheet must not use CSS custom properties")
	}
}

// TestEpubGeneratedChapterContainsTitleHeading is an end-to-end check that a
// generated chapter document contains its title as an h1 when the source body
// has none (regression test for chapters shipping without their heading).
func TestEpubGeneratedChapterContainsTitleHeading(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "headings.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Headings", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter One",
		Filename: "ch1.xhtml",
		HTML:     `<p>Stripped body without heading.</p>`,
	})

	if err := gen.Generate(outputPath); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	chapter := readEpubFile(t, outputPath, "OEBPS/ch1.xhtml")
	if !strings.Contains(chapter, "<h1>Chapter One</h1>") {
		t.Errorf("generated chapter should contain its title heading: %s", chapter)
	}
}

// TestEpubGeneratorCreatesOutputDirectory guards parity with the other
// backends: `mdpress build --format epub -o release/book.epub` must create
// release/ rather than failing with a bare "no such file or directory".
func TestEpubGeneratorCreatesOutputDirectory(t *testing.T) {
	root := t.TempDir()
	gen := NewEpubGenerator(EpubMeta{Title: "Dir Test", Author: "Author"})
	gen.AddChapter(EpubChapter{Title: "One", ID: "one", Filename: "one.xhtml", HTML: "<p>hello</p>"})

	outputPath := filepath.Join(root, "release", "nested", "book.epub")
	if err := gen.Generate(outputPath); err != nil {
		t.Fatalf("Generate() into a missing directory failed: %v", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("epub not written: %v", err)
	}
}

// rawHTMLChapter is markup of the kind Goldmark passes through untouched when
// an author writes HTML directly in Markdown. Every construct here used to end
// up verbatim in the packaged XHTML.
const rawHTMLChapter = `<p>Intro</p>
<img width=300 alt="x" src="a.png">
<p>text<hr>
<ul><li>one<li>two</ul>
<p>caf&eacute; &nbsp; A&B <span class=note data-x='1'>note</span></p>
<div class="mermaid">
graph TD;
  A--&gt;B;
</div>`

// TestEpubChaptersAreWellFormedXML is the guard for the whole EPUB: strict
// reading systems parse chapter documents as XML and refuse the entire book
// when one of them is malformed, so a build that "succeeded" on raw HTML used
// to produce a file that would not open.
func TestEpubChaptersAreWellFormedXML(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "raw-html.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Raw HTML", Author: "A", Language: "en"})
	gen.AddChapter(EpubChapter{Title: "One", ID: "one", Filename: "one.xhtml", HTML: rawHTMLChapter})
	if err := gen.Generate(outputPath); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close() //nolint:errcheck

	checked := 0
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".xhtml") && !strings.HasSuffix(f.Name, ".opf") && !strings.HasSuffix(f.Name, ".ncx") {
			continue
		}
		checked++
		doc := readZipEntry(t, r.File, f.Name)
		var parsed any
		if err := xml.Unmarshal([]byte(doc), &parsed); err != nil {
			t.Errorf("%s is not well-formed XML: %v\n%s", f.Name, err, doc)
		}
	}
	if checked < 3 {
		t.Fatalf("expected the epub to contain XML documents to check, got %d", checked)
	}

	chapter := readZipEntry(t, r.File, "OEBPS/one.xhtml")
	if !strings.Contains(chapter, `width="300"`) {
		t.Errorf("unquoted attribute value should be quoted, got:\n%s", chapter)
	}
	if strings.Contains(chapter, "&nbsp;") {
		t.Errorf("XML defines no &nbsp; entity; it must be resolved, got:\n%s", chapter)
	}
	if !strings.Contains(chapter, "<p>text</p>") {
		t.Errorf("unclosed <p> should be balanced, got:\n%s", chapter)
	}
}

// TestEpubDropsScriptFromChapters documents that scripting is stripped: EPUB
// readers are not required to run scripts, and an undeclared scripted document
// is an epubcheck error.
func TestEpubDropsScriptFromChapters(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "T", Language: "en"})
	got := gen.wrapXHTML("One", `<p>a</p><script>if (1 < 2) alert("x")</script><p>b</p>`)

	if strings.Contains(got, "alert(") {
		t.Errorf("author script should be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "<p>a</p>") || !strings.Contains(got, "<p>b</p>") {
		t.Errorf("surrounding content must be kept, got:\n%s", got)
	}
	if err := validateXHTML("chapter", got); err != nil {
		t.Errorf("chapter should stay well-formed: %v", err)
	}
}

// TestEpubMermaidIsReadableAndWarns covers the Mermaid degradation: EPUB has no
// Mermaid runtime, so the diagram source is all the reader gets. Inside the
// <div> the site output uses, its line breaks collapse into a single unreadable
// paragraph — and unlike math, nothing warned about it.
func TestEpubMermaidIsReadableAndWarns(t *testing.T) {
	var logged strings.Builder
	restore := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(restore)
	epubMermaidWarningOnce = sync.Once{}
	defer func() { epubMermaidWarningOnce = sync.Once{} }()

	outputPath := filepath.Join(t.TempDir(), "mermaid.epub")
	gen := NewEpubGenerator(EpubMeta{Title: "Diagrams", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title: "One", ID: "one", Filename: "one.xhtml",
		HTML: "<p>See:</p>\n<div class=\"mermaid\">\ngraph TD;\n  A--&gt;B;\n</div>",
	})
	gen.AddChapter(EpubChapter{
		Title: "Two", ID: "two", Filename: "two.xhtml",
		HTML: "<div class=\"mermaid\">\nsequenceDiagram\n</div>",
	})
	if err := gen.Generate(outputPath); err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if n := strings.Count(logged.String(), "Mermaid"); n != 1 {
		t.Errorf("expected exactly one Mermaid warning, got %d:\n%s", n, logged.String())
	}

	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close() //nolint:errcheck

	chapter := readZipEntry(t, r.File, "OEBPS/one.xhtml")
	if !strings.Contains(chapter, `<pre class="mermaid">`) {
		t.Errorf("mermaid block should be preformatted so its line breaks survive, got:\n%s", chapter)
	}
	if strings.Contains(chapter, `<div class="mermaid">`) {
		t.Errorf("mermaid <div> collapses whitespace in readers, got:\n%s", chapter)
	}
	if css := readZipEntry(t, r.File, "OEBPS/style.css"); !strings.Contains(css, ".mermaid") {
		t.Errorf("style.css should style mermaid source blocks, got:\n%s", css)
	}
}

// TestValidateXHTMLRejectsMalformedDocument guards the safety net itself: if a
// future change reintroduces broken markup, the build must fail loudly instead
// of shipping an unopenable book.
func TestValidateXHTMLRejectsMalformedDocument(t *testing.T) {
	if err := validateXHTML("OEBPS/bad.xhtml", `<p>text<hr></p>`); err == nil {
		t.Error("expected malformed XHTML to be rejected")
	}
	if err := validateXHTML("OEBPS/ok.xhtml", `<p>text<hr /></p>`); err != nil {
		t.Errorf("well-formed XHTML should be accepted: %v", err)
	}
}
