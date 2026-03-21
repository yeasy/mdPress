package typst

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTypstDocument_BasicMetadata(t *testing.T) {
	tests := []struct {
		name     string
		data     TypstTemplateData
		shouldContain []string
	}{
		{
			name: "full metadata",
			data: TypstTemplateData{
				Title:       "Test Title",
				Subtitle:    "Test Subtitle",
				Author:      "John Doe",
				Date:        "2026-03-21",
				Version:     "1.0.0",
				Language:    "en",
				Content:     "= Introduction\n\nThis is content.",
				PageWidth:   "210mm",
				PageHeight:  "297mm",
				MarginTop:   "20mm",
				MarginRight: "20mm",
				MarginBottom: "20mm",
				MarginLeft:  "20mm",
				FontFamily:  "Helvetica",
				FontSize:    "12pt",
				LineHeight:  1.5,
			},
			shouldContain: []string{
				"= Test Title",
				"Test Subtitle",
				"John Doe",
				"2026-03-21",
				"Version 1.0.0",
				"= Introduction",
				"paper: \"210mm-x-297mm\"",
				"lang: \"en\"",
				"size: 12pt",
			},
		},
		{
			name: "minimal metadata",
			data: TypstTemplateData{
				Title:       "Minimal",
				Content:     "Hello world",
				PageWidth:   "210mm",
				PageHeight:  "297mm",
				MarginTop:   "20mm",
				MarginRight: "20mm",
				MarginBottom: "20mm",
				MarginLeft:  "20mm",
				FontFamily:  "Helvetica",
				FontSize:    "12pt",
				LineHeight:  1.5,
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
			result, err := RenderTypstDocument(tt.data)
			if err != nil {
				t.Fatalf("RenderTypstDocument failed: %v", err)
			}

			if result == "" {
				t.Fatal("RenderTypstDocument returned empty string")
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

			result, err := RenderTypstDocument(data)
			if err != nil {
				t.Fatalf("RenderTypstDocument failed: %v", err)
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

	err := WriteTypstFile(filePath, content)
	if err != nil {
		t.Fatalf("WriteTypstFile failed: %v", err)
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

	// Note: WriteTypstFile doesn't create parent directories, so this should fail
	err := WriteTypstFile(filePath, content)
	if err == nil {
		t.Error("Expected WriteTypstFile to fail when parent directories don't exist")
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
		{"", "210mm", "297mm"},         // Default to A4
		{"a4", "210mm", "297mm"},       // Case-sensitive, defaults to A4
	}

	for _, tt := range tests {
		t.Run(tt.pageSize, func(t *testing.T) {
			w, h := GetPageDimensions(tt.pageSize)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("GetPageDimensions(%q) = (%q, %q), want (%q, %q)", tt.pageSize, w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestConvertMarginToTypst(t *testing.T) {
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

func TestCurrentDate(t *testing.T) {
	result := CurrentDate()
	// Should match YYYY-MM-DD format
	parts := strings.Split(result, "-")
	if len(parts) != 3 {
		t.Errorf("CurrentDate() returned %q, expected YYYY-MM-DD format", result)
	}

	if len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		t.Errorf("CurrentDate() returned %q, not in proper YYYY-MM-DD format", result)
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
			got := PrepareTypstContent(tt.input)
			if got != tt.expected {
				t.Errorf("PrepareTypstContent(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMakeTypstFont(t *testing.T) {
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
			got := MakeTypstFont(tt.input)
			if got != tt.expected {
				t.Errorf("MakeTypstFont(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMakeTypstFontSize(t *testing.T) {
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
			got := MakeTypstFontSize(tt.input)
			if got != tt.expected {
				t.Errorf("MakeTypstFontSize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCreateTypstDir(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := CreateTypstDir(tmpDir)
	if err != nil {
		t.Fatalf("CreateTypstDir failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".typst")
	if result != expectedPath {
		t.Errorf("CreateTypstDir returned %q, want %q", result, expectedPath)
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
	if err := os.MkdirAll(typstDir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Call again
	result, err := CreateTypstDir(tmpDir)
	if err != nil {
		t.Fatalf("CreateTypstDir should not fail on existing dir: %v", err)
	}

	if result != typstDir {
		t.Errorf("CreateTypstDir returned %q, want %q", result, typstDir)
	}
}

func TestRenderTypstDocument_AllPageSizes(t *testing.T) {
	pageSizes := []string{"A4", "A5", "Letter", "Legal"}

	for _, size := range pageSizes {
		t.Run(size, func(t *testing.T) {
			w, h := GetPageDimensions(size)
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

			result, err := RenderTypstDocument(data)
			if err != nil {
				t.Fatalf("RenderTypstDocument failed: %v", err)
			}

			expectedDimensions := `"` + w + `-x-` + h + `"`
			if !strings.Contains(result, expectedDimensions) {
				t.Errorf("Expected dimensions %s not found in output", expectedDimensions)
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

			result, err := RenderTypstDocument(data)
			if err != nil {
				t.Fatalf("RenderTypstDocument failed: %v", err)
			}

			// Just verify the template renders without error
			if !strings.Contains(result, "Test") {
				t.Error("Expected title in output")
			}
		})
	}
}
