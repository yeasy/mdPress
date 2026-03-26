package renderer

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
)

func newTestConfig() *config.BookConfig {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "测试图书"
	cfg.Book.Author = "测试作者"
	return cfg
}

func newTestTheme() *theme.Theme {
	tm := theme.NewThemeManager()
	thm, _ := tm.Get("technical")
	return thm
}

// TestNewHTMLRenderer tests creating a renderer
func TestNewHTMLRenderer(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewHTMLRenderer returned nil")
	}
}

// TestRenderEmpty tests rendering nil parts
func TestRenderEmpty(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	_, err = r.Render(nil)
	if err == nil {
		t.Error("nil parts should return an error")
	}
}

// TestRenderBasic tests basic rendering
func TestRenderBasic(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CoverHTML: "<div>cover</div>",
		ChaptersHTML: []ChapterHTML{
			{Title: "第一章", ID: "ch1", Content: "<p>内容</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should contain DOCTYPE declaration")
	}
	if !strings.Contains(html, `lang="zh-CN"`) {
		t.Error("PDF HTML should have book language lang attribute")
	}
	if !strings.Contains(html, "测试图书") {
		t.Error("should contain book title")
	}
	if !strings.Contains(html, "测试作者") {
		t.Error("should contain author name")
	}
	// Brand footer appears only inside the cover-page div.
	if !strings.Contains(html, `Build with md<span class="brand-accent">Press</span>`) {
		t.Error("should contain default brand footer (in cover)")
	}
	if !strings.Contains(html, "第一章") {
		t.Error("should contain chapter title")
	}
	if !strings.Contains(html, "<p>内容</p>") {
		t.Error("should contain chapter content")
	}
}

// TestRenderWithCover tests rendering with cover
func TestRenderWithCover(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CoverHTML: `<div class="test-cover">封面</div>`,
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "test-cover") {
		t.Error("should contain cover content")
	}
}

// TestRenderWithTOC tests rendering with TOC
func TestRenderWithTOC(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		TOCHTML: `<nav class="toc"><ul><li>Chapter 1</li></ul></nav>`,
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "toc") {
		t.Error("should contain TOC content")
	}
}

// TestRenderMultipleChapters tests multi-chapter rendering
func TestRenderMultipleChapters(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "第一章", ID: "ch1", Content: "<p>Chapter 1</p>"},
			{Title: "第二章", ID: "ch2", Content: "<p>Chapter 2</p>"},
			{Title: "第三章", ID: "ch3", Content: "<p>Chapter 3</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "Chapter 1") {
		t.Error("should contain chapter 1 content")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("should contain chapter 2 content")
	}
	if !strings.Contains(html, "Chapter 3") {
		t.Error("should contain chapter 3 content")
	}
}

func TestStandaloneRenderNestedSidebar(t *testing.T) {
	r, err := NewStandaloneHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{
				Title:   "第一章",
				ID:      "ch1",
				Content: "<h1>第一章</h1><h2 id=\"sec-1\">背景</h2>",
				Headings: []NavHeading{
					{
						Title: "背景",
						ID:    "sec-1",
						Children: []NavHeading{
							{Title: "细节", ID: "detail-1"},
						},
					},
				},
			},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("standalone HTML render failed: %v", err)
	}

	if !strings.Contains(html, "toc-group") {
		t.Error("sidebar should contain collapsible chapter group")
	}
	if !strings.Contains(html, "data-group-link=\"true\"") {
		t.Error("chapter link should be marked as collapsible group")
	}
	if !strings.Contains(html, "href=\"#sec-1\"") {
		t.Error("sidebar should contain in-chapter heading anchors")
	}
	if !strings.Contains(html, "细节") {
		t.Error("sidebar should render nested headings")
	}
}

func TestStandaloneRenderNestedChapterTree(t *testing.T) {
	r, err := NewStandaloneHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "第一章", ID: "ch1", Content: "<h1>第一章</h1>", Depth: 0},
			{Title: "1.1 小节", ID: "ch1-1", Content: "<h1>1.1 小节</h1>", Depth: 1},
			{Title: "1.2 小节", ID: "ch1-2", Content: "<h1>1.2 小节</h1>", Depth: 1},
			{Title: "第二章", ID: "ch2", Content: "<h1>第二章</h1>", Depth: 0},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("standalone HTML render failed: %v", err)
	}

	if !strings.Contains(html, `data-group-id="ch1"`) {
		t.Fatal("should contain chapter 1 group")
	}
	if !strings.Contains(html, `data-group-id="ch1-1"`) {
		t.Fatal("should contain sub-chapter group")
	}
	ch1Idx := strings.Index(html, `data-group-id="ch1"`)
	ch11Idx := strings.Index(html, `data-group-id="ch1-1"`)
	ch2Idx := strings.Index(html, `data-group-id="ch2"`)
	if ch1Idx < 0 || ch11Idx <= ch1Idx || ch2Idx <= ch11Idx {
		t.Fatalf("sub-chapters should render inside parent chapter group, indices: ch1=%d ch1-1=%d ch2=%d", ch1Idx, ch11Idx, ch2Idx)
	}
}

// TestRenderWithCustomCSS tests custom CSS
func TestRenderWithCustomCSS(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CustomCSS: ".custom-class { color: red; }",
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, ".custom-class") {
		t.Error("should contain custom CSS")
	}
}

// TestRenderIncludesThemeCSS tests inclusion of theme CSS
func TestRenderIncludesThemeCSS(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "--font-family") {
		t.Error("should contain theme CSS variables")
	}
}

// TestRenderHTMLValidity tests HTML structural validity
func TestRenderHTMLValidity(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CoverHTML: "<div>Cover</div>",
		TOCHTML:   "<div>TOC</div>",
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "<p>Content</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	requiredTags := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<style>",
		"</style>",
	}

	for _, tag := range requiredTags {
		if !strings.Contains(html, tag) {
			t.Errorf("HTML should contain %q", tag)
		}
	}
}

// TestRenderPrintCSS tests print CSS
func TestRenderPrintCSS(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "Letter"

	r, err := NewHTMLRenderer(cfg, newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(html, "@page") {
		t.Error("should contain @page rule")
	}
	if !strings.Contains(html, "Letter") {
		t.Error("should contain specified page size Letter")
	}
}

// TestRenderNilTheme tests nil theme
func TestRenderNilTheme(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), nil)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	// Should not panic
	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("nil theme render should not error: %v", err)
	}
	if html == "" {
		t.Error("should generate HTML even without a theme")
	}
}

// TestRenderEmptyChapters tests empty chapter slice (non-nil)
func TestRenderEmptyChapters(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CoverHTML:    "<div>Cover</div>",
		TOCHTML:      "<div>TOC</div>",
		ChaptersHTML: []ChapterHTML{}, // empty slice, not nil
		CustomCSS:    "",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("empty chapter list render failed: %v", err)
	}

	// Should contain cover and TOC but no chapter content
	if !strings.Contains(html, "Cover") {
		t.Error("should contain cover")
	}
	if !strings.Contains(html, "TOC") {
		t.Error("should contain TOC")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should contain valid HTML")
	}
}

// TestRenderWithAllParts tests rendering with all parts: cover, TOC, chapters, and custom CSS
func TestRenderWithAllParts(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		CoverHTML: `<div class="cover-section"><h1>书籍封面</h1></div>`,
		TOCHTML:   `<nav class="toc"><ul><li><a href="#ch1">第一章</a></li><li><a href="#ch2">第二章</a></li></ul></nav>`,
		ChaptersHTML: []ChapterHTML{
			{Title: "第一章", ID: "ch1", Content: "<p>这是第一章的内容</p>"},
			{Title: "第二章", ID: "ch2", Content: "<p>这是第二章的内容</p>"},
		},
		CustomCSS: `
.custom-heading {
  color: #2c3e50;
  font-weight: bold;
}
.custom-text {
  font-size: 14px;
}`,
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("full parts render failed: %v", err)
	}

	// Verify all parts are present
	if !strings.Contains(html, "书籍封面") {
		t.Error("should contain cover title")
	}
	if !strings.Contains(html, "cover-section") {
		t.Error("should contain cover class name")
	}
	if !strings.Contains(html, "toc") {
		t.Error("should contain TOC navigation")
	}
	if !strings.Contains(html, "第一章") {
		t.Error("should contain chapter 1")
	}
	if !strings.Contains(html, "第二章") {
		t.Error("should contain chapter 2")
	}
	if !strings.Contains(html, "这是第一章的内容") {
		t.Error("should contain chapter 1 content")
	}
	if !strings.Contains(html, ".custom-heading") {
		t.Error("should contain custom CSS - custom-heading")
	}
	if !strings.Contains(html, ".custom-text") {
		t.Error("should contain custom CSS - custom-text")
	}
}

// TestRenderPageSizeVariations table-driven tests for different page sizes
func TestRenderPageSizeVariations(t *testing.T) {
	// Page size test cases
	testCases := []struct {
		name     string
		pageSize string
		expected string
	}{
		{
			name:     "A4 page size",
			pageSize: "A4",
			expected: "size: A4",
		},
		{
			name:     "Letter page size",
			pageSize: "Letter",
			expected: "size: Letter",
		},
		{
			name:     "A5 page size",
			pageSize: "A5",
			expected: "size: A5",
		},
		{
			name:     "Legal page size",
			pageSize: "Legal",
			expected: "size: Legal",
		},
		{
			name:     "B5 page size",
			pageSize: "B5",
			expected: "size: B5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Style.PageSize = tc.pageSize

			r, err := NewHTMLRenderer(cfg, newTestTheme())
			if err != nil {
				t.Fatalf("NewHTMLRenderer failed: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			if !strings.Contains(html, tc.expected) {
				t.Errorf("output should contain %q", tc.expected)
			}
		})
	}
}

// TestRenderMarginValues tests that custom margin values appear in output
func TestRenderMarginValues(t *testing.T) {
	testCases := []struct {
		name           string
		margins        config.MarginConfig
		expectedTop    string
		expectedBottom string
		expectedLeft   string
		expectedRight  string
	}{
		{
			name: "standard margins (25mm)",
			margins: config.MarginConfig{
				Top:    25,
				Bottom: 25,
				Left:   20,
				Right:  20,
			},
			expectedTop:    "25",
			expectedBottom: "25",
			expectedLeft:   "20",
			expectedRight:  "20",
		},
		{
			name: "large margins",
			margins: config.MarginConfig{
				Top:    30,
				Bottom: 30,
				Left:   30,
				Right:  30,
			},
			expectedTop:    "30",
			expectedBottom: "30",
			expectedLeft:   "30",
			expectedRight:  "30",
		},
		{
			name: "asymmetric margins",
			margins: config.MarginConfig{
				Top:    15,
				Bottom: 25,
				Left:   10,
				Right:  35,
			},
			expectedTop:    "15",
			expectedBottom: "25",
			expectedLeft:   "10",
			expectedRight:  "35",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Style.Margin = tc.margins

			r, err := NewHTMLRenderer(cfg, newTestTheme())
			if err != nil {
				t.Fatalf("NewHTMLRenderer failed: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			// Check if margin values appear in @page rules
			if !strings.Contains(html, "margin:") {
				t.Error("should contain margin property")
			}
		})
	}
}

// TestBuildPrintCSS tests buildPrintCSS method with different configurations
func TestBuildPrintCSS(t *testing.T) {
	testCases := []struct {
		name            string
		pageSize        string
		expectPageBreak bool
		expectPageRule  bool
	}{
		{
			name:            "default config",
			pageSize:        "A4",
			expectPageBreak: true,
			expectPageRule:  true,
		},
		{
			name:            "custom page size",
			pageSize:        "Letter",
			expectPageBreak: true,
			expectPageRule:  true,
		},
		{
			name:            "empty page size (uses default)",
			pageSize:        "",
			expectPageBreak: true,
			expectPageRule:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Style.PageSize = tc.pageSize

			r, err := NewHTMLRenderer(cfg, newTestTheme())
			if err != nil {
				t.Fatalf("NewHTMLRenderer failed: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			if tc.expectPageRule && !strings.Contains(html, "@page") {
				t.Error("should contain @page rule")
			}
			if tc.expectPageBreak && !strings.Contains(html, "page-break") {
				t.Error("should contain page-break property")
			}
		})
	}
}

func TestRenderPrintLayoutAvoidsExtraPaddingAndOverflow(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "<pre><code>averyveryveryveryveryveryverylongtoken</code></pre><table><tr><td>averyveryveryveryveryveryverylongtoken</td></tr></table>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	for _, snippet := range []string{
		".chapter {\n      page-break-before: always;\n      page-break-inside: avoid;\n      padding: 0;",
		".toc-page {\n      page-break-after: always;\n      page-break-inside: avoid;\n      padding: 0;",
		"white-space: pre-wrap;",
		"table-layout: fixed;",
		"overflow-wrap: anywhere;",
	} {
		if !strings.Contains(html, snippet) {
			t.Errorf("should contain print layout fix rule %q", snippet)
		}
	}
}
