package typst

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTypstDocument_BasicMetadata(t *testing.T) {
	tests := []struct {
		name          string
		data          TypstTemplateData
		shouldContain []string
	}{
		{
			name: "full metadata",
			data: TypstTemplateData{
				Title:        "Test Title",
				Subtitle:     "Test Subtitle",
				Author:       "John Doe",
				Date:         "2026-03-21",
				Version:      "1.0.0",
				Language:     "en",
				Content:      "= Introduction\n\nThis is content.",
				PageWidth:    "210mm",
				PageHeight:   "297mm",
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   1.5,
			},
			shouldContain: []string{
				"= Test Title",
				"Test Subtitle",
				"John Doe",
				"2026-03-21",
				"Version 1.0.0",
				"= Introduction",
				"width: 210mm",
				"height: 297mm",
				"lang: \"en\"",
				"size: 12pt",
			},
		},
		{
			name: "minimal metadata",
			data: TypstTemplateData{
				Title:        "Minimal",
				Content:      "Hello world",
				PageWidth:    "210mm",
				PageHeight:   "297mm",
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   1.5,
			},
			shouldContain: []string{
				"= Minimal",
				"Hello world",
			},
		},
		{
			name: "empty optional fields",
			data: TypstTemplateData{
				Title:        "Only Title",
				Subtitle:     "",
				Author:       "",
				Date:         "",
				Version:      "",
				Content:      "Body",
				PageWidth:    "210mm",
				PageHeight:   "297mm",
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   1.5,
			},
			shouldContain: []string{
				"= Only Title",
				"Body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderTypstDocument(tt.data)
			if err != nil {
				t.Fatalf("renderTypstDocument failed: %v", err)
			}

			if result == "" {
				t.Fatal("renderTypstDocument returned empty string")
			}

			for _, expected := range tt.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected to find %q in result, but it was not found", expected)
				}
			}
		})
	}
}

func TestRenderTypstDocument_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "unicode characters",
			content: "中文测试 日本語テスト 한국어 테스트",
		},
		{
			name:    "CJK text",
			content: "这是一个测试文本，包含多行\n第二行内容\n第三行",
		},
		{
			name:    "mixed ascii and unicode",
			content: "English text mixed with 中文 and more English",
		},
		{
			name:    "special markdown characters",
			content: "# Heading\n**bold** _italic_ `code`",
		},
		{
			name:    "emoji in content",
			content: "Test with emoji: 🎉 ✓ ✗ ⚠",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := TypstTemplateData{
				Title:        "Test",
				Content:      tt.content,
				PageWidth:    "210mm",
				PageHeight:   "297mm",
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   1.5,
			}

			result, err := renderTypstDocument(data)
			if err != nil {
				t.Fatalf("renderTypstDocument failed: %v", err)
			}

			if !strings.Contains(result, tt.content) {
				t.Errorf("Content %q not preserved in output", tt.content)
			}
		})
	}
}

func TestWriteTypstFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.typ")
	content := "Test Typst content"

	err := writeTypstFile(filePath, content)
	if err != nil {
		t.Fatalf("writeTypstFile failed: %v", err)
	}

	// Verify file was written
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch: got %q, want %q", string(data), content)
	}
}

func TestWriteTypstFile_CreateParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "subdir", "nested", "test.typ")
	content := "Nested file content"

	// Note: writeTypstFile doesn't create parent directories, so this should fail
	err := writeTypstFile(filePath, content)
	if err == nil {
		t.Error("Expected writeTypstFile to fail when parent directories don't exist")
	}
}

func TestGetPageDimensions(t *testing.T) {
	tests := []struct {
		pageSize string
		wantW    string
		wantH    string
	}{
		{"A4", "210mm", "297mm"},
		{"A5", "148mm", "210mm"},
		{"Letter", "216mm", "279mm"},
		{"Legal", "216mm", "356mm"},
		{"Unknown", "210mm", "297mm"}, // Default to A4
		{"", "210mm", "297mm"},        // Default to A4
		{"a4", "210mm", "297mm"},      // Case-sensitive, defaults to A4
	}

	for _, tt := range tests {
		t.Run(tt.pageSize, func(t *testing.T) {
			w, h := getPageDimensions(tt.pageSize)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("getPageDimensions(%q) = (%q, %q), want (%q, %q)", tt.pageSize, w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestConvertMarginToTypst_Extended(t *testing.T) {
	tests := []struct {
		margin     string
		defaultVal string
		want       string
	}{
		{"20mm", "10mm", "20mm"},
		{"1.5in", "10mm", "1.5in"},
		{"", "10mm", "10mm"},
		{"0.5cm", "20mm", "0.5cm"},
		{"5pt", "", "5pt"},
	}

	for _, tt := range tests {
		t.Run(tt.margin, func(t *testing.T) {
			got := ConvertMarginToTypst(tt.margin, tt.defaultVal)
			if got != tt.want {
				t.Errorf("ConvertMarginToTypst(%q, %q) = %q, want %q", tt.margin, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestSanitizeTypstValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"20mm", "20mm"},
		{"Helvetica, Arial", "Helvetica, Arial"},
		{`"Segoe UI"`, `"Segoe UI"`},
		{"12pt", "12pt"},
		{"1.5em", "1.5em"},
		{"hello-world", "hello-world"},
		{"test<script>alert(1)</script>", "testscriptalert1script"},
		{"injection'; drop table;", "injection drop table"},
		{"normal-value", "normal-value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeTypstValue(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeTypstValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCurrentDate_Format(t *testing.T) {
	result := currentDate()
	// Should match YYYY-MM-DD format
	parts := strings.Split(result, "-")
	if len(parts) != 3 {
		t.Errorf("currentDate() returned %q, expected YYYY-MM-DD format", result)
	}

	if len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		t.Errorf("currentDate() returned %q, not in proper YYYY-MM-DD format", result)
	}
}

func TestPrepareTypstContent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple text", "Simple text"},
		{"Text with\nnewlines", "Text with\nnewlines"},
		{"特殊字符 UTF-8", "特殊字符 UTF-8"},
		{"", ""},
		{"Multiple\n\n\nNewlines", "Multiple\n\n\nNewlines"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := prepareTypstContent(tt.input)
			if got != tt.expected {
				t.Errorf("prepareTypstContent(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMakeTypstFont_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", `"Segoe UI", "Helvetica", sans-serif`},
		{"Arial", "Arial"},
		{"Helvetica, sans-serif", "Helvetica, sans-serif"},
		{`"Noto Sans", sans-serif`, `"Noto Sans", sans-serif`},
		{"custom-font", "custom-font"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := makeTypstFont(tt.input)
			if got != tt.expected {
				t.Errorf("makeTypstFont(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMakeTypstFontSize_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "12pt"},
		{"14pt", "14pt"},
		{"12px", "9.0pt"},
		{"16px", "12.0pt"},
		{"10px", "7.5pt"},
		{"20pt", "20pt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := makeTypstFontSize(tt.input)
			if got != tt.expected {
				t.Errorf("makeTypstFontSize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCreateTypstDir(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := createTypstDir(tmpDir)
	if err != nil {
		t.Fatalf("createTypstDir failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".typst")
	if result != expectedPath {
		t.Errorf("createTypstDir returned %q, want %q", result, expectedPath)
	}

	// Verify directory was created
	info, err := os.Stat(result)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestCreateTypstDir_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	typstDir := filepath.Join(tmpDir, ".typst")

	// Create it first
	if err := os.MkdirAll(typstDir, 0o755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Call again
	result, err := createTypstDir(tmpDir)
	if err != nil {
		t.Fatalf("createTypstDir should not fail on existing dir: %v", err)
	}

	if result != typstDir {
		t.Errorf("createTypstDir returned %q, want %q", result, typstDir)
	}
}

func TestRenderTypstDocument_AllPageSizes(t *testing.T) {
	pageSizes := []string{"A4", "A5", "Letter", "Legal"}

	for _, size := range pageSizes {
		t.Run(size, func(t *testing.T) {
			w, h := getPageDimensions(size)
			data := TypstTemplateData{
				Title:        "Page Size Test: " + size,
				Content:      "Testing " + size,
				PageWidth:    w,
				PageHeight:   h,
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   1.5,
			}

			result, err := renderTypstDocument(data)
			if err != nil {
				t.Fatalf("renderTypstDocument failed: %v", err)
			}

			if !strings.Contains(result, "width: "+w) {
				t.Errorf("Expected width: %s not found in output", w)
			}
			if !strings.Contains(result, "height: "+h) {
				t.Errorf("Expected height: %s not found in output", h)
			}
		})
	}
}

func TestSanitizeTypstText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text unchanged", "Hello world", "Hello world"},
		{"backslash escaped", `a\b`, `a\\b`},
		{"hash escaped", "# Heading", `\# Heading`},
		{"dollar escaped", "$math$", `\$math\$`},
		{"equals escaped", "a=b", `a\=b`},
		{"at sign escaped", "user@host", `user\@host`},
		{"less than escaped", "a<b", `a\<b`},
		{"greater than escaped", "a>b", `a\>b`},
		{"underscore escaped", "_italic_", `\_italic\_`},
		{"asterisk escaped", "**bold**", `\*\*bold\*\*`},
		{"go template open stripped", "{{inject}}", "inject"},
		{"go template close stripped", "prefix}}", "prefix"},
		{"empty string", "", ""},
		{"injection attempt stripped", "{{#import bad}}", `\#import bad`},
		{"multiple specials", "#$@", `\#\$\@`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTypstText(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeTypstText(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeTemplateValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain value unchanged", "210mm", "210mm"},
		{"open delimiter stripped", "{{bad", "bad"},
		{"close delimiter stripped", "bad}}", "bad"},
		{"both delimiters stripped", "{{evil}}", "evil"},
		{"nested injection stripped", "{{{{double}}}}", "double"},
		{"empty string", "", ""},
		{"normal dimension", "25mm", "25mm"},
		{"no delimiters", "1.5em", "1.5em"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTemplateValue(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeTemplateValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeDimension(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fallback string
		expected string
	}{
		{"valid mm", "210mm", "10mm", "210mm"},
		{"valid cm", "2.5cm", "10mm", "2.5cm"},
		{"valid in", "1in", "10mm", "1in"},
		{"valid pt", "12pt", "10mm", "12pt"},
		{"valid em", "1.5em", "10mm", "1.5em"},
		{"decimal mm", "25.4mm", "10mm", "25.4mm"},
		{"leading whitespace trimmed", " 20mm", "10mm", "20mm"},
		{"trailing whitespace trimmed", "20mm ", "10mm", "20mm"},
		{"invalid empty", "", "10mm", "10mm"},
		{"invalid word", "invalid", "10mm", "10mm"},
		{"invalid px unit", "12px", "10mm", "10mm"},
		{"template injection", "{{evil}}mm", "10mm", "10mm"},
		{"negative not allowed", "-5mm", "10mm", "10mm"},
		{"bare number no unit", "210", "10mm", "10mm"},
		{"valid uses fallback when invalid", "notadimension", "297mm", "297mm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeDimension(tt.input, tt.fallback)
			if got != tt.expected {
				t.Errorf("sanitizeDimension(%q, %q) = %q, want %q", tt.input, tt.fallback, got, tt.expected)
			}
		})
	}
}

func TestRenderTypstDocument_LineHeight(t *testing.T) {
	tests := []struct {
		lineHeight float64
	}{
		{1.0},
		{1.5},
		{2.0},
		{1.2},
	}

	for _, tt := range tests {
		t.Run("lineheight", func(t *testing.T) {
			data := TypstTemplateData{
				Title:        "Test",
				Content:      "Content",
				PageWidth:    "210mm",
				PageHeight:   "297mm",
				MarginTop:    "20mm",
				MarginRight:  "20mm",
				MarginBottom: "20mm",
				MarginLeft:   "20mm",
				FontFamily:   "Helvetica",
				FontSize:     "12pt",
				LineHeight:   tt.lineHeight,
			}

			result, err := renderTypstDocument(data)
			if err != nil {
				t.Fatalf("renderTypstDocument failed: %v", err)
			}

			// Just verify the template renders without error
			if !strings.Contains(result, "Test") {
				t.Error("Expected title in output")
			}
		})
	}
}
