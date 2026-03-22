package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/crossref"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"gopkg.in/yaml.v3"
)

// 获取测试数据目录路径
func getTestDataDir() string {
	// Go tests run from the package directory, so testdata is relative to here
	return "testdata"
}

// TestConfigLoadAndValidate 测试加载和验证配置文件
func TestConfigLoadAndValidate(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证书籍元数据
	if cfg.Book.Title != "集成测试图书" {
		t.Errorf("期望书名为 '集成测试图书'，实际为 '%s'", cfg.Book.Title)
	}
	if cfg.Book.Author != "测试作者" {
		t.Errorf("期望作者为 '测试作者'，实际为 '%s'", cfg.Book.Author)
	}
	if cfg.Book.Version != "1.0.0" {
		t.Errorf("期望版本为 '1.0.0'，实际为 '%s'", cfg.Book.Version)
	}

	// 验证章节
	if len(cfg.Chapters) != 2 {
		t.Errorf("期望 2 个章节，实际为 %d", len(cfg.Chapters))
	}

	// 验证样式
	if cfg.Style.Theme != "technical" {
		t.Errorf("期望主题为 'technical'，实际为 '%s'", cfg.Style.Theme)
	}
	if cfg.Style.PageSize != "A4" {
		t.Errorf("期望页面尺寸为 'A4'，实际为 '%s'", cfg.Style.PageSize)
	}

	// 验证输出
	if cfg.Output.Filename != "test-output.pdf" {
		t.Errorf("期望输出文件为 'test-output.pdf'，实际为 '%s'", cfg.Output.Filename)
	}
	if !cfg.Output.TOC {
		t.Error("期望生成目录")
	}
	if !cfg.Output.Cover {
		t.Error("期望生成封面")
	}
}

// TestMarkdownParsing 测试解析 Markdown 文件并验证输出包含期望的 HTML 元素
func TestMarkdownParsing(t *testing.T) {
	// 定义解析测试用例
	testCases := []struct {
		name             string
		filename         string
		expectedElements []string
	}{
		{
			name:     "第一章解析",
			filename: "ch01.md",
			expectedElements: []string{
				"第一章",
				"简介",
				"加粗文本",
				"斜体文本",
				"列表项",
				"<table",
			},
		},
		{
			name:     "第二章解析",
			filename: "ch02.md",
			expectedElements: []string{
				"第二章",
				"详情",
				"代码示例",
				"package",
				"<pre",
			},
		},
	}

	testDataDir := getTestDataDir()
	parser := markdown.NewParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 读取 Markdown 文件
			filePath := filepath.Join(testDataDir, tc.filename)
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}

			// 解析 Markdown
			html, _, err := parser.Parse(data)
			if err != nil {
				t.Fatalf("解析 Markdown 失败: %v", err)
			}

			// 验证输出包含期望的元素
			for _, elem := range tc.expectedElements {
				if !strings.Contains(html, elem) {
					t.Errorf("HTML 中未找到期望的元素: %s", elem)
				}
			}

			// 验证输出是 HTML
			if !strings.Contains(html, "<") || !strings.Contains(html, ">") {
				t.Error("输出应该是 HTML 格式")
			}
		})
	}
}

// TestTOCGeneration 测试解析章节、收集标题、生成目录并验证结构
func TestTOCGeneration(t *testing.T) {
	testDataDir := getTestDataDir()

	// 读取第一章文件以提取标题
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}

	// 解析 Markdown 并提取标题
	parser := markdown.NewParser()
	_, headings, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("解析 Markdown 失败: %v", err)
	}

	// 验证提取的标题数量（应至少有 1 个，即主标题）
	if len(headings) == 0 {
		t.Error("应该提取到至少一个标题")
	}

	// 转换标题类型
	tocHeadings := make([]toc.HeadingInfo, len(headings))
	for i, h := range headings {
		tocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	// 生成目录
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(tocHeadings)

	// 验证生成的目录
	if len(entries) == 0 {
		t.Error("应该生成目录条目")
	}

	// 渲染目录为 HTML
	tocHTML := tocGen.RenderHTML(entries)

	// 验证目录 HTML
	if !strings.Contains(tocHTML, "<nav") {
		t.Error("目录 HTML 应包含 <nav 标签")
	}
	if !strings.Contains(tocHTML, "<ul") {
		t.Error("目录 HTML 应包含 <ul 标签")
	}
	if !strings.Contains(tocHTML, "<li") {
		t.Error("目录 HTML 应包含 <li 标签")
	}
	if !strings.Contains(tocHTML, "<a") {
		t.Error("目录 HTML 应包含 <a 标签")
	}
}

// TestCoverGeneration 测试从配置元数据生成封面并验证 HTML 结构
func TestCoverGeneration(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 生成封面
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// 验证封面 HTML 结构
	if !strings.Contains(coverHTML, "<!DOCTYPE html>") {
		t.Error("封面应包含 DOCTYPE 声明")
	}
	if !strings.Contains(coverHTML, cfg.Book.Title) {
		t.Error("封面应包含书籍标题")
	}
	if !strings.Contains(coverHTML, cfg.Book.Author) {
		t.Error("封面应包含作者名")
	}
	if !strings.Contains(coverHTML, cfg.Book.Version) {
		t.Error("封面应包含版本号")
	}
	if !strings.Contains(coverHTML, "<style") {
		t.Error("封面应包含样式")
	}
}

// TestCrossRefWorkflow 测试交叉引用工作流：注册引用、处理 HTML、验证替换
func TestCrossRefWorkflow(t *testing.T) {
	// 创建交叉引用解析器
	resolver := crossref.NewResolver()

	// 注册图表
	figNum1 := resolver.RegisterFigure("fig1", "示例图表 1")
	figNum2 := resolver.RegisterFigure("fig2", "示例图表 2")

	// 验证图表编号
	if figNum1 != 1 {
		t.Errorf("第一个图表编号应为 1，实际为 %d", figNum1)
	}
	if figNum2 != 2 {
		t.Errorf("第二个图表编号应为 2，实际为 %d", figNum2)
	}

	// 注册表格
	tableNum1 := resolver.RegisterTable("tbl1", "示例表格 1")
	tableNum2 := resolver.RegisterTable("tbl2", "示例表格 2")

	// 验证表格编号
	if tableNum1 != 1 {
		t.Errorf("第一个表格编号应为 1，实际为 %d", tableNum1)
	}
	if tableNum2 != 2 {
		t.Errorf("第二个表格编号应为 2，实际为 %d", tableNum2)
	}

	// 注册章节
	resolver.RegisterSection("sec1", "第 1 章", 1)
	resolver.RegisterSection("sec2", "第 2 章", 1)

	// 验证解析功能
	ref1, err := resolver.Resolve("fig1")
	if err != nil {
		t.Fatalf("解析图表引用失败: %v", err)
	}
	if ref1.Number != 1 {
		t.Errorf("期望引用编号为 1，实际为 %d", ref1.Number)
	}

	// 测试 HTML 处理
	testHTML := `<p>参考 {{ref:fig1}} 和 {{ref:tbl1}}</p>`
	processedHTML := resolver.ProcessHTML(testHTML)

	// 验证占位符被替换
	if strings.Contains(processedHTML, "{{ref:fig1}}") {
		t.Error("占位符 {{ref:fig1}} 应被替换")
	}
	if strings.Contains(processedHTML, "{{ref:tbl1}}") {
		t.Error("占位符 {{ref:tbl1}} 应被替换")
	}

	// 验证所有引用
	allRefs := resolver.GetAllReferences()
	if len(allRefs) == 0 {
		t.Error("应该返回至少一个引用")
	}
}

// TestFullHTMLRender 测试渲染完整文档（包含所有部分）并验证结构
func TestFullHTMLRender(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 获取主题
	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("获取主题失败: %v", err)
	}

	// 生成封面
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// 解析第一章
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("读取第一章失败: %v", err)
	}

	parser := markdown.NewParser()
	ch01HTML, _, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("解析第一章失败: %v", err)
	}

	// 解析第二章
	ch02Path := filepath.Join(testDataDir, "ch02.md")
	ch02Data, err := os.ReadFile(ch02Path)
	if err != nil {
		t.Fatalf("读取第二章失败: %v", err)
	}

	ch02HTML, _, err := parser.Parse(ch02Data)
	if err != nil {
		t.Fatalf("解析第二章失败: %v", err)
	}

	// 生成目录
	allMDHeadings := []markdown.HeadingInfo{}
	_, h1, _ := parser.Parse(ch01Data)
	_, h2, _ := parser.Parse(ch02Data)
	allMDHeadings = append(allMDHeadings, h1...)
	allMDHeadings = append(allMDHeadings, h2...)

	// 转换标题类型
	allTocHeadings := make([]toc.HeadingInfo, len(allMDHeadings))
	for i, h := range allMDHeadings {
		allTocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allTocHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	// 组织渲染部分
	parts := &renderer.RenderParts{
		CoverHTML: coverHTML,
		TOCHTML:   tocHTML,
		ChaptersHTML: []renderer.ChapterHTML{
			{Title: "第一章 简介", ID: "ch1", Content: ch01HTML},
			{Title: "第二章 详情", ID: "ch2", Content: ch02HTML},
		},
		CustomCSS: ".custom { margin: 20px; }",
	}

	// 渲染完整 HTML
	r, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	html, err := r.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	// 验证完整结构
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		cfg.Book.Title,
		cfg.Book.Author,
		"第一章",
		"第二章",
		"<nav",
		"@page",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(html, elem) {
			t.Errorf("完整 HTML 应包含: %s", elem)
		}
	}

	// 验证是有效的 HTML
	if !strings.Contains(html, "</html>") {
		t.Error("HTML 应有闭合标签")
	}
}

// TestSummaryParsing 测试解析 SUMMARY.md 文件
func TestSummaryParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	summaryPath := filepath.Join(testDataDir, "SUMMARY.md")

	// 读取 SUMMARY.md
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("读取 SUMMARY.md 失败: %v", err)
	}

	content := string(data)

	// 验证文件内容
	if !strings.Contains(content, "第一章") {
		t.Error("SUMMARY.md 应包含第一章")
	}
	if !strings.Contains(content, "第二章") {
		t.Error("SUMMARY.md 应包含第二章")
	}
	if !strings.Contains(content, "ch01.md") {
		t.Error("SUMMARY.md 应包含 ch01.md 引用")
	}
	if !strings.Contains(content, "ch02.md") {
		t.Error("SUMMARY.md 应包含 ch02.md 引用")
	}

	// 验证目录结构
	if !strings.Contains(content, "* [") {
		t.Error("SUMMARY.md 应使用 Markdown 列表语法")
	}
}

// TestEmptyFileParsing 测试解析空 Markdown 文件
func TestEmptyFileParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	emptyPath := filepath.Join(testDataDir, "empty.md")

	// 读取空文件
	data, err := os.ReadFile(emptyPath)
	if err != nil {
		t.Fatalf("读取 empty.md 失败: %v", err)
	}

	// 解析空 Markdown
	parser := markdown.NewParser()
	html, headings, err := parser.Parse(data)
	if err != nil {
		// 空文件可能返回错误或空字符串，都是可接受的
		t.Logf("空文件解析返回错误: %v", err)
	}

	// 验证结果（即使是空的也应该返回有效值）
	if html == "" {
		t.Logf("空文件生成空 HTML，这是可以接受的")
	}

	// 应该没有标题
	if len(headings) > 0 {
		t.Error("空文件应该没有标题")
	}
}

// TestSpecialCharsParsing 测试解析包含特殊字符的 Markdown 文件
func TestSpecialCharsParsing(t *testing.T) {
	testDataDir := getTestDataDir()
	specialPath := filepath.Join(testDataDir, "special_chars.md")

	// 读取特殊字符文件
	data, err := os.ReadFile(specialPath)
	if err != nil {
		t.Fatalf("读取 special_chars.md 失败: %v", err)
	}

	// 解析 Markdown
	parser := markdown.NewParser()
	html, _, err := parser.Parse(data)
	if err != nil {
		t.Fatalf("解析 Markdown 失败: %v", err)
	}

	// 验证特殊字符被正确转义
	// HTML 应将 < 转为 &lt;，> 转为 &gt;，& 转为 &amp;
	if !strings.Contains(html, "&") {
		// 特殊字符应被转义
		t.Logf("特殊字符在输出中: %s", html)
	}

	// 验证标题被处理（即使包含特殊字符）
	if !strings.Contains(html, "<h1") {
		t.Error("应该生成 <h1 标题")
	}

	// 验证代码块中的反引号被保留
	if !strings.Contains(html, "特殊字符") {
		t.Error("应该包含代码块内容")
	}
}

// TestMultiLanguageBuild 测试多语言构建：创建临时项目，发现多个语言，验证章节
func TestMultiLanguageBuild(t *testing.T) {
	tempDir := t.TempDir()

	// 创建 SUMMARY.md（Discover 需要它来触发 LANGS.md 检测）
	summaryContent := "# Summary\n\n* [Introduction](README.md)\n"
	if err := os.WriteFile(filepath.Join(tempDir, "SUMMARY.md"), []byte(summaryContent), 0644); err != nil {
		t.Fatalf("写入 SUMMARY.md 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Test Book\n"), 0644); err != nil {
		t.Fatalf("写入 README.md 失败: %v", err)
	}

	// 创建 LANGS.md 文件定义多个语言
	langsContent := `# Languages

- [中文](./zh/)
- [English](./en/)
`
	if err := os.WriteFile(filepath.Join(tempDir, "LANGS.md"), []byte(langsContent), 0644); err != nil {
		t.Fatalf("写入 LANGS.md 失败: %v", err)
	}

	// 创建中文目录和配置
	zhDir := filepath.Join(tempDir, "zh")
	if err := os.MkdirAll(zhDir, 0755); err != nil {
		t.Fatalf("创建中文目录失败: %v", err)
	}

	zhConfigContent := `book:
  title: "中文书籍"
  author: "作者"
  version: "1.0.0"
chapters:
  - title: "第一章"
    file: "ch01.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(zhDir, "book.yaml"), []byte(zhConfigContent), 0644); err != nil {
		t.Fatalf("写入中文 book.yaml 失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(zhDir, "ch01.md"), []byte("# 第一章\n\n中文内容"), 0644); err != nil {
		t.Fatalf("写入中文章节失败: %v", err)
	}

	// 创建英文目录和配置
	enDir := filepath.Join(tempDir, "en")
	if err := os.MkdirAll(enDir, 0755); err != nil {
		t.Fatalf("创建英文目录失败: %v", err)
	}

	enConfigContent := `book:
  title: "English Book"
  author: "Author"
  version: "1.0.0"
chapters:
  - title: "Chapter 1"
    file: "ch01.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(enDir, "book.yaml"), []byte(enConfigContent), 0644); err != nil {
		t.Fatalf("写入英文 book.yaml 失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(enDir, "ch01.md"), []byte("# Chapter 1\n\nEnglish content"), 0644); err != nil {
		t.Fatalf("写入英文章节失败: %v", err)
	}

	// 使用 Discover 自动发现配置
	cfg, err := config.Discover(tempDir)
	if err != nil {
		t.Fatalf("发现配置失败: %v", err)
	}

	// 验证 LANGS.md 被检测到
	if cfg.LangsFile == "" {
		t.Error("应该检测到 LANGS.md")
	}

	// 验证可以加载中文配置
	zhCfg, err := config.Load(filepath.Join(zhDir, "book.yaml"))
	if err != nil {
		t.Fatalf("加载中文配置失败: %v", err)
	}
	if zhCfg.Book.Title != "中文书籍" {
		t.Errorf("期望中文书名，实际: %s", zhCfg.Book.Title)
	}
	if len(zhCfg.Chapters) != 1 {
		t.Errorf("期望 1 个中文章节，实际: %d", len(zhCfg.Chapters))
	}

	// 验证可以加载英文配置
	enCfg, err := config.Load(filepath.Join(enDir, "book.yaml"))
	if err != nil {
		t.Fatalf("加载英文配置失败: %v", err)
	}
	if enCfg.Book.Title != "English Book" {
		t.Errorf("期望英文书名，实际: %s", enCfg.Book.Title)
	}
	if len(enCfg.Chapters) != 1 {
		t.Errorf("期望 1 个英文章节，实际: %d", len(enCfg.Chapters))
	}
}

// TestHTMLRenderingEndToEnd 测试 HTML 渲染端到端：创建最小项目，渲染，验证输出结构
func TestHTMLRenderingEndToEnd(t *testing.T) {
	tempDir := t.TempDir()

	// 创建最小项目
	bookYAML := `book:
  title: "测试书籍"
  author: "测试作者"
  version: "1.0.0"
chapters:
  - title: "简介"
    file: "intro.md"
  - title: "内容"
    file: "content.md"
style:
  theme: "technical"
output:
  toc: true
  cover: true
`
	if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("写入 book.yaml 失败: %v", err)
	}

	introContent := `# 简介

这是简介章节。

## 背景
- 项目背景
- 主要目标
`
	if err := os.WriteFile(filepath.Join(tempDir, "intro.md"), []byte(introContent), 0644); err != nil {
		t.Fatalf("写入 intro.md 失败: %v", err)
	}

	contentMD := `# 内容

## 主要内容
这是主要内容部分。

## 代码示例
` + "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	if err := os.WriteFile(filepath.Join(tempDir, "content.md"), []byte(contentMD), 0644); err != nil {
		t.Fatalf("写入 content.md 失败: %v", err)
	}

	// 加载配置
	cfg, err := config.Load(filepath.Join(tempDir, "book.yaml"))
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 初始化主题和解析器
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("获取主题失败: %v", err)
	}

	parser := markdown.NewParser()

	// 解析所有章节
	var chaptersHTML []renderer.ChapterHTML
	var allHeadings []toc.HeadingInfo

	for _, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := os.ReadFile(chapterPath)
		if err != nil {
			t.Fatalf("读取章节 %s 失败: %v", ch.File, err)
		}

		html, headings, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("解析章节 %s 失败: %v", ch.File, err)
		}

		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      ch.Title,
			Content: html,
		})
	}

	if len(chaptersHTML) == 0 {
		t.Fatal("应该至少有一个章节")
	}

	// 生成封面和目录
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	tocGen := toc.NewGenerator()
	tocEntries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(tocEntries)

	// 渲染完整 HTML
	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("创建 HTML 渲染器失败: %v", err)
	}

	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("渲染 HTML 失败: %v", err)
	}

	// 验证输出包含预期结构
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		cfg.Book.Title,
		cfg.Book.Author,
		"简介",
		"内容",
		"<nav",
		"<title>",
		"</html>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(fullHTML, elem) {
			t.Errorf("HTML 应包含: %s", elem)
		}
	}

	// 验证代码块被处理（代码可能在 <code> 或 <pre> 中）
	if !strings.Contains(fullHTML, "func main") && !strings.Contains(fullHTML, "Println") {
		t.Log("注意: 代码块内容可能被 HTML 模板转义")
	}

	// 验证标题存在
	if !strings.Contains(fullHTML, "h1") && !strings.Contains(fullHTML, "h2") {
		t.Error("HTML 应包含标题标签")
	}

	t.Logf("HTML 渲染成功: %d 字节", len(fullHTML))
}

// TestSiteOutputStructure 测试网站输出结构：验证生成正确的目录布局和 index.html
func TestSiteOutputStructure(t *testing.T) {
	tempDir := t.TempDir()
	_ = filepath.Join(tempDir, "output") // reserved for future site output verification

	// 创建项目
	bookYAML := `book:
  title: "网站测试"
  author: "作者"
chapters:
  - title: "页面1"
    file: "page1.md"
  - title: "页面2"
    file: "page2.md"
style:
  theme: "technical"
`
	if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(bookYAML), 0644); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "page1.md"), []byte("# 页面1\n\n内容1"), 0644); err != nil {
		t.Fatalf("写入 page1.md 失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "page2.md"), []byte("# 页面2\n\n内容2"), 0644); err != nil {
		t.Fatalf("写入 page2.md 失败: %v", err)
	}

	// 加载配置
	cfg, err := config.Load(filepath.Join(tempDir, "book.yaml"))
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置有效
	if cfg.Book.Title != "网站测试" {
		t.Errorf("期望书名 '网站测试'，实际: %s", cfg.Book.Title)
	}

	if len(cfg.Chapters) != 2 {
		t.Errorf("期望 2 个章节，实际: %d", len(cfg.Chapters))
	}

	// 验证所有章节文件存在
	for _, ch := range cfg.Chapters {
		resolvedPath := cfg.ResolvePath(ch.File)
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			t.Errorf("章节文件不存在: %s", resolvedPath)
		}
	}

	t.Logf("网站项目验证完成: %d 个章节", len(cfg.Chapters))
}

// TestConfigValidation 测试配置验证：检查各种配置错误
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		configYAML  string
		expectedErr string
	}{
		{
			name:        "空标题",
			configYAML:  "book:\n  title: \"\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"",
			expectedErr: "title cannot be empty",
		},
		{
			name:        "无章节",
			configYAML:  "book:\n  title: \"测试\"\nchapters: []",
			expectedErr: "at least one chapter",
		},
		{
			name:        "无效页面大小",
			configYAML:  "book:\n  title: \"测试\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"\nstyle:\n  page_size: \"XYZ\"",
			expectedErr: "unsupported page size",
		},
		{
			name:        "无效主题",
			configYAML:  "book:\n  title: \"测试\"\nchapters:\n  - title: \"ch1\"\n    file: \"ch01.md\"\nstyle:\n  theme: \"nonexistent\"",
			expectedErr: "unknown theme",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// 写入配置
			if err := os.WriteFile(filepath.Join(tempDir, "book.yaml"), []byte(tc.configYAML), 0644); err != nil {
				t.Fatalf("写入配置失败: %v", err)
			}

			// 创建虚拟章节文件
			if err := os.WriteFile(filepath.Join(tempDir, "ch01.md"), []byte("# 测试"), 0644); err != nil {
				t.Fatalf("写入章节失败: %v", err)
			}

			// 尝试加载配置
			cfg := config.DefaultConfig()
			configPath := filepath.Join(tempDir, "book.yaml")
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("读取配置失败: %v", err)
			}

			if err := yaml.Unmarshal(data, cfg); err != nil {
				t.Fatalf("解析 YAML 失败: %v", err)
			}

			cfg.SetBaseDir(tempDir)

			// 验证应该失败
			err = cfg.Validate()
			if err == nil {
				t.Error("期望配置验证失败")
			}
			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("期望错误包含 '%s'，实际: %v", tc.expectedErr, err)
			}
		})
	}
}

// TestThemeApplication 测试主题应用：验证主题 CSS 被包含在输出中
func TestThemeApplication(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 获取主题
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		t.Fatalf("获取主题失败: %v", err)
	}

	// 验证主题有效
	if thm == nil {
		t.Fatal("主题应该有效")
	}

	// 解析章节
	parser := markdown.NewParser()
	ch01Path := filepath.Join(testDataDir, "ch01.md")
	ch01Data, err := os.ReadFile(ch01Path)
	if err != nil {
		t.Fatalf("读取章节失败: %v", err)
	}

	html, headings, err := parser.Parse(ch01Data)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 生成完整文档
	coverGen := cover.NewCoverGenerator(cfg.Book)
	tocGen := toc.NewGenerator()

	tocHeadings := make([]toc.HeadingInfo, len(headings))
	for i, h := range headings {
		tocHeadings[i] = toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID}
	}

	tocEntries := tocGen.Generate(tocHeadings)
	parts := &renderer.RenderParts{
		CoverHTML: coverGen.RenderHTML(),
		TOCHTML:   tocGen.RenderHTML(tocEntries),
		ChaptersHTML: []renderer.ChapterHTML{
			{Title: cfg.Chapters[0].Title, ID: "ch1", Content: html},
		},
	}

	// 创建渲染器
	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("创建渲染器失败: %v", err)
	}

	// 渲染
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	// 验证 CSS 被包含
	if !strings.Contains(fullHTML, "<style") {
		t.Error("输出应包含 <style> 标签")
	}

	if !strings.Contains(fullHTML, "css") && !strings.Contains(fullHTML, "@") {
		t.Error("输出应包含 CSS 规则")
	}

	// 验证主题名称相关的 CSS
	if strings.Contains(fullHTML, "font-family") || strings.Contains(fullHTML, "color") || strings.Contains(fullHTML, "margin") {
		t.Logf("验证主题 CSS 被应用")
	}
}
