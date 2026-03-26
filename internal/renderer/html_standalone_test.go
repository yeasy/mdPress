package renderer

import (
	"strings"
	"testing"
)

// TestNewStandaloneHTMLRendererSuccess tests successful creation of StandaloneHTMLRenderer.
func TestNewStandaloneHTMLRendererSuccess(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)

	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	if r == nil {
		t.Fatal("NewStandaloneHTMLRenderer returned nil")
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

// TestNewStandaloneHTMLRendererNilConfig tests error handling with nil config.
func TestNewStandaloneHTMLRendererNilConfig(t *testing.T) {
	thm := newTestTheme(t)
	r, _ := NewStandaloneHTMLRenderer(nil, thm)
	// Should create renderer even with nil config (panic happens on Render)
	if r == nil {
		t.Fatal("NewStandaloneHTMLRenderer should not return nil for nil config")
	}
}

// TestNewStandaloneHTMLRendererNilTheme tests creation with nil theme.
func TestNewStandaloneHTMLRendererNilTheme(t *testing.T) {
	cfg := newTestConfig()
	r, err := NewStandaloneHTMLRenderer(cfg, nil)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed with nil theme: %v", err)
	}
	if r == nil {
		t.Fatal("NewStandaloneHTMLRenderer returned nil for nil theme")
	}
}

// TestStandaloneRenderBasic tests basic standalone HTML rendering.
func TestStandaloneRenderBasic(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter 1", ID: "ch1", Content: "<p>Content 1</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("HTML should contain html tag")
	}
	if !strings.Contains(html, "Chapter 1") {
		t.Error("HTML should contain chapter title")
	}
	if !strings.Contains(html, "<p>Content 1</p>") {
		t.Error("HTML should contain chapter content")
	}
}

// TestStandaloneRenderNilParts tests error handling with nil parts.
func TestStandaloneRenderNilParts(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	_, err = r.Render(nil)
	if err == nil {
		t.Error("Render(nil) should return error")
	}
}

// TestStandaloneLanguageAttribute tests language attribute rendering.
func TestStandaloneLanguageAttribute(t *testing.T) {
	testCases := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "Chinese",
			language: "zh-CN",
			expected: `lang="zh-CN"`,
		},
		{
			name:     "English",
			language: "en",
			expected: `lang="en"`,
		},
		{
			name:     "Empty defaults to en",
			language: "",
			expected: `lang="en"`,
		},
		{
			name:     "Japanese",
			language: "ja",
			expected: `lang="ja"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Book.Language = tc.language
			thm := newTestTheme(t)

			r, err := NewStandaloneHTMLRenderer(cfg, thm)
			if err != nil {
				t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
			}

			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "content"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			if !strings.Contains(html, tc.expected) {
				t.Errorf("expected HTML to contain %q, got:\n%s", tc.expected, html[:500])
			}
		})
	}
}

// TestStandaloneMetadataRendering tests title and author metadata rendering.
func TestStandaloneMetadataRendering(t *testing.T) {
	cfg := newTestConfig()
	cfg.Book.Title = "Test Book Title"
	cfg.Book.Author = "Test Author Name"
	thm := newTestTheme(t)

	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "Test Book Title") {
		t.Error("HTML should contain book title")
	}
	if !strings.Contains(html, "Test Author Name") {
		t.Error("HTML should contain author name")
	}
}

// TestStandaloneEmptyChapters tests rendering with empty chapters list.
func TestStandaloneEmptyChapters(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render with empty chapters failed: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should still be valid even with empty chapters")
	}
}

// TestStandaloneMultipleChapters tests rendering with multiple chapters.
func TestStandaloneMultipleChapters(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter 1", ID: "ch1", Content: "<p>Content 1</p>"},
			{Title: "Chapter 2", ID: "ch2", Content: "<p>Content 2</p>"},
			{Title: "Chapter 3", ID: "ch3", Content: "<p>Content 3</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "Content 1") {
		t.Error("HTML should contain Content 1")
	}
	if !strings.Contains(html, "Content 2") {
		t.Error("HTML should contain Content 2")
	}
	if !strings.Contains(html, "Content 3") {
		t.Error("HTML should contain Content 3")
	}
}

// TestStandaloneChapterIDGeneration tests automatic ID generation for chapters.
func TestStandaloneChapterIDGeneration(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "", Content: "content1", Depth: 0},
			{Title: "Ch2", ID: "custom-id", Content: "content2", Depth: 0},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Generated IDs appear in chapter elements and navigation
	// The sidebar uses empty IDs since it's built from original chapters
	if !strings.Contains(html, `data-group-id="custom-id"`) {
		t.Error("HTML should contain custom-id in sidebar")
	}
	// Generated IDs are in chapter elements themselves
	if !strings.Contains(html, "content1") && !strings.Contains(html, "content2") {
		t.Error("HTML should contain chapter content")
	}
}

// TestStandaloneChapterNavigation tests prev/next chapter navigation.
func TestStandaloneChapterNavigation(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "First", ID: "ch1", Content: "content1"},
			{Title: "Second", ID: "ch2", Content: "content2"},
			{Title: "Third", ID: "ch3", Content: "content3"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// First chapter should have no previous but have next
	if !strings.Contains(html, "Second") {
		t.Error("First chapter should reference next chapter title")
	}

	// Last chapter should have previous but no next
	if !strings.Contains(html, "Third") {
		t.Error("Last chapter should be present")
	}
}

// TestStandaloneCustomCSS tests custom CSS inclusion.
func TestStandaloneCustomCSS(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
		CustomCSS: ".custom-class { color: red; }",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, ".custom-class") {
		t.Error("HTML should contain custom CSS")
	}
	if !strings.Contains(html, "color: red") {
		t.Error("HTML should contain custom CSS properties")
	}
}

// TestStandaloneThemeCSS tests that theme CSS is included.
func TestStandaloneThemeCSS(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should contain CSS variables from theme
	if !strings.Contains(html, "--color") && !strings.Contains(html, "color") {
		t.Error("HTML should contain theme CSS or color definitions")
	}
}

// TestBuildStandaloneSidebarTreeFlat tests flat chapter structure.
func TestBuildStandaloneSidebarTreeFlat(t *testing.T) {
	chapters := []ChapterHTML{
		{Title: "Ch1", ID: "ch1", Depth: 0},
		{Title: "Ch2", ID: "ch2", Depth: 0},
		{Title: "Ch3", ID: "ch3", Depth: 0},
	}

	tree := buildStandaloneSidebarTree(chapters)

	if len(tree) != 3 {
		t.Errorf("expected 3 root nodes, got %d", len(tree))
	}

	for i, node := range tree {
		if node.Title != chapters[i].Title {
			t.Errorf("node %d title mismatch: expected %q, got %q", i, chapters[i].Title, node.Title)
		}
		if node.ID != chapters[i].ID {
			t.Errorf("node %d ID mismatch: expected %q, got %q", i, chapters[i].ID, node.ID)
		}
		if len(node.Children) != 0 {
			t.Errorf("node %d should have no children, got %d", i, len(node.Children))
		}
	}
}

// TestBuildStandaloneSidebarTreeNested tests nested chapter structure.
func TestBuildStandaloneSidebarTreeNested(t *testing.T) {
	chapters := []ChapterHTML{
		{Title: "Ch1", ID: "ch1", Depth: 0},
		{Title: "Ch1.1", ID: "ch1-1", Depth: 1},
		{Title: "Ch1.2", ID: "ch1-2", Depth: 1},
		{Title: "Ch2", ID: "ch2", Depth: 0},
		{Title: "Ch2.1", ID: "ch2-1", Depth: 1},
	}

	tree := buildStandaloneSidebarTree(chapters)

	if len(tree) != 2 {
		t.Errorf("expected 2 root nodes, got %d", len(tree))
	}

	if tree[0].Title != "Ch1" {
		t.Errorf("first root should be Ch1, got %q", tree[0].Title)
	}
	if len(tree[0].Children) != 2 {
		t.Errorf("Ch1 should have 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Title != "Ch1.1" {
		t.Errorf("first child of Ch1 should be Ch1.1, got %q", tree[0].Children[0].Title)
	}

	if tree[1].Title != "Ch2" {
		t.Errorf("second root should be Ch2, got %q", tree[1].Title)
	}
	if len(tree[1].Children) != 1 {
		t.Errorf("Ch2 should have 1 child, got %d", len(tree[1].Children))
	}
}

// TestBuildStandaloneSidebarTreeDeepNesting tests deeply nested structure.
func TestBuildStandaloneSidebarTreeDeepNesting(t *testing.T) {
	chapters := []ChapterHTML{
		{Title: "L0", ID: "l0", Depth: 0},
		{Title: "L1", ID: "l1", Depth: 1},
		{Title: "L2", ID: "l2", Depth: 2},
		{Title: "L3", ID: "l3", Depth: 3},
	}

	tree := buildStandaloneSidebarTree(chapters)

	if len(tree) != 1 {
		t.Errorf("expected 1 root, got %d", len(tree))
	}

	// Navigate through nesting
	node := tree[0]
	if node.Title != "L0" || len(node.Children) == 0 {
		t.Fatal("L0 node not properly formed")
	}

	node = node.Children[0]
	if node.Title != "L1" || len(node.Children) == 0 {
		t.Fatal("L1 node not properly formed")
	}

	node = node.Children[0]
	if node.Title != "L2" || len(node.Children) == 0 {
		t.Fatal("L2 node not properly formed")
	}

	node = node.Children[0]
	if node.Title != "L3" {
		t.Errorf("deepest node should be L3, got %q", node.Title)
	}
	if len(node.Children) != 0 {
		t.Errorf("L3 should have no children")
	}
}

// TestBuildStandaloneSidebarTreeEmpty tests empty input.
func TestBuildStandaloneSidebarTreeEmpty(t *testing.T) {
	tree := buildStandaloneSidebarTree([]ChapterHTML{})
	if len(tree) != 0 {
		t.Errorf("empty input should produce empty tree, got %d nodes", len(tree))
	}
}

// TestBuildStandaloneSidebarTreeSkipInvalidDepth tests depth skip handling.
func TestBuildStandaloneSidebarTreeSkipInvalidDepth(t *testing.T) {
	// Depth jumps from 0 to 2, skipping depth 1
	chapters := []ChapterHTML{
		{Title: "L0", ID: "l0", Depth: 0},
		{Title: "L2", ID: "l2", Depth: 2}, // Skip depth 1
		{Title: "L0b", ID: "l0b", Depth: 0},
	}

	tree := buildStandaloneSidebarTree(chapters)

	if len(tree) != 2 {
		t.Errorf("expected 2 root nodes, got %d", len(tree))
	}

	// L2 should be skipped and not appear as root
	if tree[0].Title != "L0" {
		t.Errorf("first node should be L0, got %q", tree[0].Title)
	}
	if tree[1].Title != "L0b" {
		t.Errorf("second node should be L0b, got %q", tree[1].Title)
	}
}

// TestStandaloneSidebarRendering tests sidebar HTML generation.
func TestStandaloneSidebarRendering(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Chapter 1", ID: "ch1", Content: "content", Depth: 0},
			{Title: "Section 1.1", ID: "sec-1-1", Content: "content", Depth: 1},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Check sidebar structure
	if !strings.Contains(html, "toc-group") {
		t.Error("sidebar should contain toc-group")
	}
	if !strings.Contains(html, "toc-link") {
		t.Error("sidebar should contain toc-link")
	}
	if !strings.Contains(html, "Chapter 1") {
		t.Error("sidebar should contain chapter title")
	}
}

// TestStandaloneHeadingsNavigation tests nested heading structure in sidebar.
func TestStandaloneHeadingsNavigation(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{
				Title:   "Chapter 1",
				ID:      "ch1",
				Content: "<h1>Chapter 1</h1><h2 id='h1'>Section A</h2>",
				Headings: []NavHeading{
					{
						Title: "Section A",
						ID:    "h1",
						Children: []NavHeading{
							{Title: "Subsection A.1", ID: "h1-1"},
						},
					},
				},
			},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, "Section A") {
		t.Error("should render heading title")
	}
	if !strings.Contains(html, "Subsection A.1") {
		t.Error("should render nested heading title")
	}
}

// TestStandaloneDataPopulation tests standalone data struct population.
func TestStandaloneDataPopulation(t *testing.T) {
	cfg := newTestConfig()
	cfg.Book.Title = "Test Title"
	cfg.Book.Author = "Test Author"
	cfg.Book.Language = "zh-CN"

	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content1"},
			{Title: "Ch2", ID: "ch2", Content: "content2"},
		},
		CustomCSS: ".custom { color: blue; }",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify all components are populated correctly
	if !strings.Contains(html, "Test Title") {
		t.Error("title not populated")
	}
	if !strings.Contains(html, "Test Author") {
		t.Error("author not populated")
	}
	if !strings.Contains(html, `lang="zh-CN"`) {
		t.Error("language not populated")
	}
	if !strings.Contains(html, ".custom") {
		t.Error("CSS not populated")
	}
	if !strings.Contains(html, "content1") && !strings.Contains(html, "Ch1") {
		t.Error("chapter content not populated")
	}
}

// TestStandaloneSpecialCharactersEscape tests proper HTML escaping.
func TestStandaloneSpecialCharactersEscape(t *testing.T) {
	cfg := newTestConfig()
	cfg.Book.Title = "Title with <script>"
	cfg.Book.Author = "Author & Co."

	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1 < Ch2", ID: "ch1&ch2", Content: "<p>Safe HTML</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should escape special chars in attributes but preserve safe HTML in content
	if strings.Contains(html, "Title with <script>") {
		t.Error("should escape <script> tag in title")
	}
	if strings.Contains(html, `title="Ch1 < Ch2"`) && strings.Count(html, "&lt;") == 0 {
		t.Error("should escape < character in chapter title")
	}
	if !strings.Contains(html, "<p>Safe HTML</p>") {
		t.Error("should preserve safe HTML in content")
	}
}

// TestStandaloneHTMLStructure tests overall HTML document structure.
func TestStandaloneHTMLStructure(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<style>",
		"</style>",
		"<script>",
		"</script>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(html, elem) {
			t.Errorf("HTML should contain %q", elem)
		}
	}
}

// TestStandaloneCDNURLSubstitution tests that CDN URLs are substituted in template.
func TestStandaloneCDNURLSubstitution(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	if r.tmpl == nil {
		t.Fatal("template not initialized")
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Should contain actual CDN URLs, not placeholders
	if strings.Contains(html, "{{MERMAID_CDN_URL}}") {
		t.Error("Mermaid CDN URL placeholder should be substituted")
	}
	if strings.Contains(html, "{{KATEX_CSS_URL}}") {
		t.Error("KaTeX CSS URL placeholder should be substituted")
	}
}

// TestStandaloneRenderWithAllComponentsCombined tests rendering with all components.
func TestStandaloneRenderWithAllComponentsCombined(t *testing.T) {
	cfg := newTestConfig()
	cfg.Book.Title = "Complete Book"
	cfg.Book.Author = "Complete Author"
	cfg.Book.Language = "en"

	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{
				Title:   "Chapter 1",
				ID:      "ch1",
				Content: "<h1>Chapter 1</h1><p>Intro</p>",
				Depth:   0,
				Headings: []NavHeading{
					{Title: "Section 1.1", ID: "sec-1-1"},
				},
			},
			{
				Title:   "Chapter 2",
				ID:      "ch2",
				Content: "<h1>Chapter 2</h1><p>More content</p>",
				Depth:   0,
			},
		},
		CustomCSS: "body { font-size: 16px; }",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify all components
	if !strings.Contains(html, "Complete Book") {
		t.Error("title missing")
	}
	if !strings.Contains(html, "Complete Author") {
		t.Error("author missing")
	}
	if !strings.Contains(html, `lang="en"`) {
		t.Error("language attribute missing")
	}
	if !strings.Contains(html, "Chapter 1") {
		t.Error("first chapter title missing")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("second chapter title missing")
	}
	if !strings.Contains(html, "Section 1.1") {
		t.Error("nested heading missing")
	}
	if !strings.Contains(html, "body { font-size: 16px; }") {
		t.Error("custom CSS missing")
	}
	if !strings.Contains(html, "toc-group") {
		t.Error("sidebar structure missing")
	}
}

// TestStandaloneChapterIDsInContent tests chapter IDs are accessible in rendered HTML.
func TestStandaloneChapterIDsInContent(t *testing.T) {
	cfg := newTestConfig()
	thm := newTestTheme(t)
	r, err := NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}

	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "chapter-one", Content: "content"},
			{Title: "Ch2", ID: "chapter-two", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(html, `chapter-one`) {
		t.Error("chapter ID should be in HTML")
	}
	if !strings.Contains(html, `chapter-two`) {
		t.Error("chapter ID should be in HTML")
	}
}
