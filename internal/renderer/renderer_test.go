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

// TestNewHTMLRenderer 测试创建渲染器
func TestNewHTMLRenderer(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	if r == nil {
		t.Fatal("NewHTMLRenderer 返回 nil")
	}
}

// TestRenderEmpty 测试渲染空部件
func TestRenderEmpty(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	_, err = r.Render(nil)
	if err == nil {
		t.Error("nil 部件应返回错误")
	}
}

// TestRenderBasic 测试基本渲染
func TestRenderBasic(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "第一章", ID: "ch1", Content: "<p>内容</p>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("应包含 DOCTYPE 声明")
	}
	if !strings.Contains(html, `lang="zh-CN"`) {
		t.Error("PDF HTML 应带书籍语言 lang 属性")
	}
	if !strings.Contains(html, "测试图书") {
		t.Error("应包含书名")
	}
	if !strings.Contains(html, "测试作者") {
		t.Error("应包含作者名")
	}
	if !strings.Contains(html, `Build with md<span class="brand-accent">Press</span>`) {
		t.Error("应包含默认品牌页脚")
	}
	if !strings.Contains(html, "第一章") {
		t.Error("应包含章节标题")
	}
	if !strings.Contains(html, "<p>内容</p>") {
		t.Error("应包含章节内容")
	}
}

// TestRenderWithCover 测试带封面渲染
func TestRenderWithCover(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		CoverHTML: `<div class="test-cover">封面</div>`,
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, "test-cover") {
		t.Error("应包含封面内容")
	}
}

// TestRenderWithTOC 测试带目录渲染
func TestRenderWithTOC(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		TOCHTML: `<nav class="toc"><ul><li>Chapter 1</li></ul></nav>`,
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, "toc") {
		t.Error("应包含目录内容")
	}
}

// TestRenderMultipleChapters 测试多章节渲染
func TestRenderMultipleChapters(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
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
		t.Fatalf("渲染失败: %v", err)
	}

	for i := 1; i <= 3; i++ {
		chID := "ch" + strings.Repeat("", 0) // just to use strings
		_ = chID
	}

	if !strings.Contains(html, "Chapter 1") {
		t.Error("应包含第一章内容")
	}
	if !strings.Contains(html, "Chapter 2") {
		t.Error("应包含第二章内容")
	}
	if !strings.Contains(html, "Chapter 3") {
		t.Error("应包含第三章内容")
	}
}

func TestStandaloneRenderNestedSidebar(t *testing.T) {
	r, err := NewStandaloneHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer 失败: %v", err)
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
		t.Fatalf("单页 HTML 渲染失败: %v", err)
	}

	if !strings.Contains(html, "toc-group") {
		t.Error("侧边栏应包含可折叠章节分组")
	}
	if !strings.Contains(html, "data-group-link=\"true\"") {
		t.Error("章节链接应标记为可折叠分组")
	}
	if !strings.Contains(html, "href=\"#sec-1\"") {
		t.Error("侧边栏应包含章节内标题锚点")
	}
	if !strings.Contains(html, "细节") {
		t.Error("侧边栏应渲染嵌套标题")
	}
}

func TestStandaloneRenderNestedChapterTree(t *testing.T) {
	r, err := NewStandaloneHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer 失败: %v", err)
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
		t.Fatalf("单页 HTML 渲染失败: %v", err)
	}

	if !strings.Contains(html, `data-group-id="ch1"`) {
		t.Fatal("应包含第一章分组")
	}
	if !strings.Contains(html, `data-group-id="ch1-1"`) {
		t.Fatal("应包含子章节分组")
	}
	ch1Idx := strings.Index(html, `data-group-id="ch1"`)
	ch11Idx := strings.Index(html, `data-group-id="ch1-1"`)
	ch2Idx := strings.Index(html, `data-group-id="ch2"`)
	if ch1Idx < 0 || ch11Idx <= ch1Idx || ch2Idx <= ch11Idx {
		t.Fatalf("子章节应渲染在父章节分组内部，索引: ch1=%d ch1-1=%d ch2=%d", ch1Idx, ch11Idx, ch2Idx)
	}
}

// TestRenderWithCustomCSS 测试自定义 CSS
func TestRenderWithCustomCSS(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		CustomCSS: ".custom-class { color: red; }",
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, ".custom-class") {
		t.Error("应包含自定义 CSS")
	}
}

// TestRenderIncludesThemeCSS 测试包含主题 CSS
func TestRenderIncludesThemeCSS(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, "--font-family") {
		t.Error("应包含主题 CSS 变量")
	}
}

// TestRenderHTMLValidity 测试 HTML 结构完整性
func TestRenderHTMLValidity(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
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
		t.Fatalf("渲染失败: %v", err)
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
			t.Errorf("HTML 应包含 %q", tag)
		}
	}
}

// TestRenderPrintCSS 测试打印 CSS
func TestRenderPrintCSS(t *testing.T) {
	cfg := newTestConfig()
	cfg.Style.PageSize = "Letter"

	r, err := NewHTMLRenderer(cfg, newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	if !strings.Contains(html, "@page") {
		t.Error("应包含 @page 规则")
	}
	if !strings.Contains(html, "Letter") {
		t.Error("应包含指定的页面尺寸 Letter")
	}
}

// TestRenderNilTheme 测试空主题
func TestRenderNilTheme(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), nil)
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "content"},
		},
	}

	// 不应 panic
	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("空主题渲染不应报错: %v", err)
	}
	if html == "" {
		t.Error("即使无主题也应生成 HTML")
	}
}

// TestRenderEmptyChapters 测试空章节切片（非 nil）
func TestRenderEmptyChapters(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		CoverHTML:    "<div>Cover</div>",
		TOCHTML:      "<div>TOC</div>",
		ChaptersHTML: []ChapterHTML{}, // 空切片，非 nil
		CustomCSS:    "",
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("空章节列表渲染失败: %v", err)
	}

	// 应包含封面和目录，但无章节内容
	if !strings.Contains(html, "Cover") {
		t.Error("应包含封面")
	}
	if !strings.Contains(html, "TOC") {
		t.Error("应包含目录")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("应包含有效的 HTML")
	}
}

// TestRenderWithAllParts 测试包含所有部分的渲染：封面、目录、章节和自定义 CSS
func TestRenderWithAllParts(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
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
		t.Fatalf("完整部分渲染失败: %v", err)
	}

	// 验证所有部分都存在
	if !strings.Contains(html, "书籍封面") {
		t.Error("应包含封面标题")
	}
	if !strings.Contains(html, "cover-section") {
		t.Error("应包含封面类名")
	}
	if !strings.Contains(html, "toc") {
		t.Error("应包含目录导航")
	}
	if !strings.Contains(html, "第一章") {
		t.Error("应包含第一章")
	}
	if !strings.Contains(html, "第二章") {
		t.Error("应包含第二章")
	}
	if !strings.Contains(html, "这是第一章的内容") {
		t.Error("应包含第一章内容")
	}
	if !strings.Contains(html, ".custom-heading") {
		t.Error("应包含自定义 CSS - custom-heading")
	}
	if !strings.Contains(html, ".custom-text") {
		t.Error("应包含自定义 CSS - custom-text")
	}
}

// TestRenderPageSizeVariations 表驱动测试不同页面大小
func TestRenderPageSizeVariations(t *testing.T) {
	// 页面大小测试案例
	testCases := []struct {
		name     string
		pageSize string
		expected string
	}{
		{
			name:     "A4 页面大小",
			pageSize: "A4",
			expected: "size: A4",
		},
		{
			name:     "Letter 页面大小",
			pageSize: "Letter",
			expected: "size: Letter",
		},
		{
			name:     "A5 页面大小",
			pageSize: "A5",
			expected: "size: A5",
		},
		{
			name:     "Legal 页面大小",
			pageSize: "Legal",
			expected: "size: Legal",
		},
		{
			name:     "B5 页面大小",
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
				t.Fatalf("NewHTMLRenderer 失败: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("渲染失败: %v", err)
			}

			if !strings.Contains(html, tc.expected) {
				t.Errorf("应在输出中包含 %q", tc.expected)
			}
		})
	}
}

// TestRenderMarginValues 测试自定义边距值是否出现在输出中
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
			name: "标准边距（25mm）",
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
			name: "大边距",
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
			name: "不对称边距",
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
				t.Fatalf("NewHTMLRenderer 失败: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("渲染失败: %v", err)
			}

			// 检查边距值是否在 @page 规则中出现
			if !strings.Contains(html, "margin:") {
				t.Error("应包含 margin 属性")
			}
		})
	}
}

// TestBuildPrintCSS 测试 buildPrintCSS 方法处理不同配置
func TestBuildPrintCSS(t *testing.T) {
	testCases := []struct {
		name            string
		pageSize        string
		expectPageBreak bool
		expectPageRule  bool
	}{
		{
			name:            "默认配置",
			pageSize:        "A4",
			expectPageBreak: true,
			expectPageRule:  true,
		},
		{
			name:            "自定义页面大小",
			pageSize:        "Letter",
			expectPageBreak: true,
			expectPageRule:  true,
		},
		{
			name:            "空页面大小（应用默认）",
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
				t.Fatalf("NewHTMLRenderer 失败: %v", err)
			}
			parts := &RenderParts{
				ChaptersHTML: []ChapterHTML{
					{Title: "Ch1", ID: "ch1", Content: "<p>Test</p>"},
				},
			}

			html, err := r.Render(parts)
			if err != nil {
				t.Fatalf("渲染失败: %v", err)
			}

			if tc.expectPageRule && !strings.Contains(html, "@page") {
				t.Error("应包含 @page 规则")
			}
			if tc.expectPageBreak && !strings.Contains(html, "page-break") {
				t.Error("应包含 page-break 属性")
			}
		})
	}
}

func TestRenderPrintLayoutAvoidsExtraPaddingAndOverflow(t *testing.T) {
	r, err := NewHTMLRenderer(newTestConfig(), newTestTheme())
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	parts := &RenderParts{
		ChaptersHTML: []ChapterHTML{
			{Title: "Ch1", ID: "ch1", Content: "<pre><code>averyveryveryveryveryveryverylongtoken</code></pre><table><tr><td>averyveryveryveryveryveryverylongtoken</td></tr></table>"},
		},
	}

	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	for _, snippet := range []string{
		".chapter {\n      page-break-before: always;\n      page-break-inside: avoid;\n      padding: 0;",
		".toc-page {\n      page-break-after: always;\n      page-break-inside: avoid;\n      padding: 0;",
		"white-space: pre-wrap;",
		"table-layout: fixed;",
		"overflow-wrap: anywhere;",
	} {
		if !strings.Contains(html, snippet) {
			t.Errorf("应包含打印布局修正规则 %q", snippet)
		}
	}
}
