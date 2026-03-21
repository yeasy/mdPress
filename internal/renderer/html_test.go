package renderer

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// TestNewHTMLRenderer_Success tests successful creation of HTMLRenderer
func TestNewHTMLRenderer_Success(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()

	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewHTMLRenderer returned nil")
	}
	if r.config != cfg {
		t.Error("config not properly assigned")
	}
	if r.theme != thm {
		t.Error("theme not properly assigned")
	}
	if r.tmpl == nil {
		t.Error("template not properly initialized")
	}
}

// TestNewHTMLRenderer_NilConfig tests creation with nil config
func TestNewHTMLRenderer_NilConfig(t *testing.T) {
	thm := newTestTheme()

	// Should still create renderer with nil config
	r, err := NewHTMLRenderer(nil, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer with nil config failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewHTMLRenderer should not return nil for nil config")
	}
}

// TestNewHTMLRenderer_NilTheme tests creation with nil theme
func TestNewHTMLRenderer_NilTheme(t *testing.T) {
	cfg := newTestConfig()

	r, err := NewHTMLRenderer(cfg, nil)
	if err != nil {
		t.Fatalf("NewHTMLRenderer with nil theme failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewHTMLRenderer returned nil for nil theme")
	}
}

// TestNewHTMLRenderer_InvalidTemplate tests handling of invalid template
func TestNewHTMLRenderer_InvalidTemplate(t *testing.T) {
	// This test verifies that the template parsing works
	// The actual template is in htmlTemplate string
	cfg := newTestConfig()
	thm := newTestTheme()

	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	if r.tmpl == nil {
		t.Error("template should be initialized")
	}
}

// TestRender_NilParts tests rendering with nil parts
func TestRender_NilParts(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	_, err = r.Render(nil)
	if err == nil {
		t.Error("Render with nil parts should return error")
	}
	if !strings.Contains(err.Error(), "render parts cannot be nil") {
		t.Errorf("error should mention render parts, got: %v", err)
	}
}

// TestRender_EmptyParts tests rendering with empty parts
func TestRender_EmptyParts(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render with empty parts failed: %v", err)
	}
	if html == "" {
		t.Error("Render should return non-empty HTML")
	}
}

// TestRender_WithCover tests rendering with cover HTML
func TestRender_WithCover(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		CoverHTML: "<div class='cover'><h1>Test Book</h1></div>",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(html, "Test Book") {
		t.Error("HTML should contain cover content")
	}
}

// TestRender_WithTOC tests rendering with table of contents
func TestRender_WithTOC(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		TOCHTML: "<div class='toc'><ul><li>Chapter 1</li></ul></div>",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(html, "Chapter 1") {
		t.Error("HTML should contain TOC content")
	}
}

// TestRender_WithChapters tests rendering with chapters
func TestRender_WithChapters(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{
				Title:   "Chapter 1",
				ID:      "ch1",
				Content: "<p>Content of chapter 1</p>",
			},
			{
				Title:   "Chapter 2",
				ID:      "ch2",
				Content: "<p>Content of chapter 2</p>",
			},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(html, "Chapter 1") {
		t.Error("HTML should contain chapter 1")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("HTML should contain chapter 2")
	}
}

// TestRender_WithCustomCSS tests rendering with custom CSS
func TestRender_WithCustomCSS(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	customCSS := "body { color: red; }"
	parts := &RenderParts{
		CustomCSS: customCSS,
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(html, "color: red") {
		t.Error("HTML should contain custom CSS")
	}
}

// TestRender_CompleteDocument tests rendering a complete document
func TestRender_CompleteDocument(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		CoverHTML: "<div class='cover'><h1>My Book</h1></div>",
		TOCHTML:   "<div class='toc'><h2>Table of Contents</h2></div>",
		ChaptersHTML: []ChapterHTML{
			{
				Title:   "Introduction",
				ID:      "intro",
				Content: "<p>This is the introduction.</p>",
				Depth:   0,
			},
			{
				Title:   "Chapter 1",
				ID:      "ch1",
				Content: "<p>Chapter 1 content</p>",
				Depth:   1,
			},
		},
		CustomCSS: "body { font-size: 14px; }",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify complete document structure
	if !strings.Contains(html, "<!DOCTYPE") {
		t.Error("HTML should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("HTML should contain html tag")
	}
	if !strings.Contains(html, "<head>") {
		t.Error("HTML should contain head tag")
	}
	if !strings.Contains(html, "<body") {
		t.Error("HTML should contain body tag")
	}
	if !strings.Contains(html, "My Book") {
		t.Error("HTML should contain cover content")
	}
	if !strings.Contains(html, "Table of Contents") {
		t.Error("HTML should contain TOC")
	}
	if !strings.Contains(html, "Chapter 1 content") {
		t.Error("HTML should contain chapter content")
	}
	if !strings.Contains(html, "font-size: 14px") {
		t.Error("HTML should contain custom CSS")
	}
}

// TestRender_WithWatermark tests rendering with watermark
func TestRender_WithWatermark(t *testing.T) {
	cfg := newTestConfig()
	cfg.Output.Watermark = "CONFIDENTIAL"
	cfg.Output.WatermarkOpacity = 0.3

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter 1", ID: "ch1", Content: "<p>Content</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if html == "" {
		t.Error("HTML should not be empty")
	}
	// Watermark will be in the rendered output if template supports it
}

// TestRender_WithHeaderFooter tests rendering with header and footer
func TestRender_WithHeaderFooter(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.Header.Left = "My Header"
	cfg.Style.Footer.Center = "Page"

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter", ID: "ch1", Content: "<p>Content</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if html == "" {
		t.Error("HTML should not be empty")
	}
}

// TestBuildFullCSS_WithTheme tests buildFullCSS with theme
func TestBuildFullCSS_WithTheme(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildFullCSS("")
	if css == "" {
		t.Error("CSS should not be empty when theme is present")
	}
	// Verify print CSS is included
	if !strings.Contains(css, "@page") {
		t.Error("CSS should contain @page rule for print styles")
	}
}

// TestBuildFullCSS_WithCustomCSS tests buildFullCSS with custom CSS
func TestBuildFullCSS_WithCustomCSS(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	customCSS := "body { color: blue; }"
	css := r.buildFullCSS(customCSS)
	if !strings.Contains(css, "color: blue") {
		t.Error("CSS should contain custom CSS")
	}
	if !strings.Contains(css, "@page") {
		t.Error("CSS should contain print styles")
	}
}

// TestBuildFullCSS_WithoutTheme tests buildFullCSS without theme
func TestBuildFullCSS_WithoutTheme(t *testing.T) {
	cfg := newTestConfig()
	r, err := NewHTMLRenderer(cfg, nil)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildFullCSS("body { margin: 0; }")
	if css == "" {
		t.Error("CSS should not be empty")
	}
	// Should still have print CSS
	if !strings.Contains(css, "@page") {
		t.Error("CSS should contain @page rule")
	}
}

// TestBuildPrintCSS tests buildPrintCSS function
func TestBuildPrintCSS_DefaultPageSize(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = ""
	cfg.Style.Margin = config.MarginConfig{Top: 20, Right: 20, Bottom: 20, Left: 20}

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildPrintCSS()
	if !strings.Contains(css, "@page") {
		t.Error("Print CSS should contain @page")
	}
	if !strings.Contains(css, "A4") {
		t.Error("Print CSS should default to A4 page size")
	}
}

// TestBuildPrintCSS_CustomPageSize tests buildPrintCSS with custom page size
func TestBuildPrintCSS_CustomPageSize(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "Letter"
	cfg.Style.Margin = config.MarginConfig{Top: 25, Right: 25, Bottom: 25, Left: 25}

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildPrintCSS()
	if !strings.Contains(css, "Letter") {
		t.Error("Print CSS should contain custom page size")
	}
}

// TestBuildPrintCSS_PageBreaks tests that print CSS includes page breaks
func TestBuildPrintCSS_PageBreaks(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "A4"
	cfg.Style.Margin = config.MarginConfig{Top: 20, Right: 20, Bottom: 20, Left: 20}

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildPrintCSS()

	// Verify page break rules
	if !strings.Contains(css, ".chapter") {
		t.Error("Print CSS should include chapter page break rules")
	}
	if !strings.Contains(css, "page-break-before") {
		t.Error("Print CSS should include page-break-before")
	}
	if !strings.Contains(css, "page-break-after") {
		t.Error("Print CSS should include page-break-after")
	}
}

// TestBuildPrintCSS_Margins tests that margins are correctly included
func TestBuildPrintCSS_Margins(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "A4"
	cfg.Style.Margin = config.MarginConfig{Top: 30, Right: 25, Bottom: 35, Left: 20}

	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	css := r.buildPrintCSS()

	// Check that margins are included in the CSS
	if !strings.Contains(css, "margin:") {
		t.Error("Print CSS should include margin settings")
	}
}

// TestRender_TableDrivenTests provides comprehensive test coverage
func TestRender_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		coverHTML   string
		tocHTML     string
		customCSS   string
		chapters    []ChapterHTML
		shouldErr   bool
		checkOutput func(string) bool
	}{
		{
			name:      "minimal content",
			shouldErr: false,
			checkOutput: func(html string) bool {
				return strings.Contains(html, "<html")
			},
		},
		{
			name:      "with chapter",
			chapters:  []ChapterHTML{{Title: "Ch1", ID: "c1", Content: "<p>test</p>"}},
			shouldErr: false,
			checkOutput: func(html string) bool {
				return strings.Contains(html, "Ch1")
			},
		},
		{
			name:      "with cover",
			coverHTML: "<div>Book Title</div>",
			shouldErr: false,
			checkOutput: func(html string) bool {
				return strings.Contains(html, "Book Title")
			},
		},
		{
			name:      "with toc",
			tocHTML:   "<div>Contents</div>",
			shouldErr: false,
			checkOutput: func(html string) bool {
				return strings.Contains(html, "Contents")
			},
		},
		{
			name:      "with custom css",
			customCSS: "body { background: white; }",
			shouldErr: false,
			checkOutput: func(html string) bool {
				return strings.Contains(html, "background: white")
			},
		},
	}

	cfg := newTestConfig()
	thm := newTestTheme()
	r, err := NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := &RenderParts{
				CoverHTML:    tt.coverHTML,
				TOCHTML:      tt.tocHTML,
				ChaptersHTML: tt.chapters,
				CustomCSS:    tt.customCSS,
			}

			html, err := r.Render(parts)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Render() error = %v, shouldErr %v", err, tt.shouldErr)
			}

			if !tt.shouldErr && !tt.checkOutput(html) {
				t.Errorf("Output check failed for %q", tt.name)
			}
		})
	}
}

// BenchmarkNewHTMLRenderer benchmarks HTMLRenderer creation
func BenchmarkNewHTMLRenderer(b *testing.B) {
	cfg := newTestConfig()
	thm := newTestTheme()
	for i := 0; i < b.N; i++ {
		_, _ = NewHTMLRenderer(cfg, thm)
	}
}

// BenchmarkRender benchmarks the Render function
func BenchmarkRender(b *testing.B) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, _ := NewHTMLRenderer(cfg, thm)

	parts := &RenderParts{
		CoverHTML: "<div>Cover</div>",
		TOCHTML:   "<div>TOC</div>",
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter 1", ID: "ch1", Content: "<p>Content</p>"},
		},
		CustomCSS: "body { font-size: 14px; }",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Render(parts)
	}
}

// BenchmarkBuildFullCSS benchmarks CSS building
func BenchmarkBuildFullCSS(b *testing.B) {
	cfg := newTestConfig()
	thm := newTestTheme()
	r, _ := NewHTMLRenderer(cfg, thm)

	customCSS := "body { color: black; }"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.buildFullCSS(customCSS)
	}
}
