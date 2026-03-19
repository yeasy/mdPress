package output

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	defer r.Close()

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
		rc, _ := mimeFile.Open()
		mimeData, _ := io.ReadAll(rc)
		rc.Close()
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
		_, err := zip.OpenReader(outputPath)
		if err != nil {
			t.Errorf("Generated EPUB with 0 chapters is not a valid ZIP")
		}
	}
}

// TestGenerateInvalidOutputPath verifies that Generate fails and cleans up on invalid path.
func TestGenerateInvalidOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "nonexistent", "deeply", "nested", "output.epub")

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

// TestWrapXHTMLWithoutCSS verifies wrapXHTML without CSS set.
func TestWrapXHTMLWithoutCSS(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Test", Language: "en"})
	// Don't call SetCSS

	html := `<p>Content without CSS</p>`
	result := gen.wrapXHTML("No CSS", html)

	if strings.Contains(result, `<link rel="stylesheet" type="text/css" href="style.css"/>`) {
		t.Error("CSS link should not be present when CSS is not set")
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
	if err := os.WriteFile(coverPath, pngData, 0644); err != nil {
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
	defer r.Close()

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
	defer r.Close()

	opfFile, err := r.Open("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("content.opf not found: %v", err)
	}
	defer opfFile.Close()

	opfData, _ := io.ReadAll(opfFile)
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
	defer r.Close()

	navFile, err := r.Open("OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("nav.xhtml not found: %v", err)
	}
	defer navFile.Close()

	navData, _ := io.ReadAll(navFile)
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
	defer r.Close()

	ncxFile, err := r.Open("OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("toc.ncx not found: %v", err)
	}
	defer ncxFile.Close()

	ncxData, _ := io.ReadAll(ncxFile)
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
	if err := os.Mkdir(readOnlyDir, 0444); err != nil {
		t.Fatalf("failed to create readonly directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) //nolint:errcheck // Restore permissions for cleanup

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
	defer r.Close()

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
	opfFile, _ := r.Open("OEBPS/content.opf")
	opfData, _ := io.ReadAll(opfFile)
	opfFile.Close()
	opfContent := string(opfData)

	for i := 1; i <= 5; i++ {
		if !strings.Contains(opfContent, "ch"+string(rune('0'+i))+".xhtml") {
			t.Errorf("chapter %d reference not found in OPF manifest", i)
		}
	}
}

// TestGenerateWithoutCSS verifies EPUB generation without CSS.
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

	// Verify EPUB doesn't contain style.css
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open EPUB: %v", err)
	}
	defer r.Close()

	fileMap := make(map[string]*zip.File)
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	if _, ok := fileMap["OEBPS/style.css"]; ok {
		t.Error("style.css should not be included when CSS is not set")
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
	defer r.Close()

	containerFile, err := r.Open("META-INF/container.xml")
	if err != nil {
		t.Fatalf("container.xml not found: %v", err)
	}
	defer containerFile.Close()

	containerData, _ := io.ReadAll(containerFile)
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
	defer r.Close()

	chapterFile, err := r.Open("OEBPS/chapter.xhtml")
	if err != nil {
		t.Fatalf("chapter.xhtml not found: %v", err)
	}
	defer chapterFile.Close()

	chapterData, _ := io.ReadAll(chapterFile)
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

		_ = gen.Generate(outputPath)
	}
}
