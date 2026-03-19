// e2e_test.go 端到端测试。
// 测试完整的 init → build → 验证输出 流程，包括零配置模式和 HTML 输出。
package tests

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/cover"
	"github.com/yeasy/mdpress/internal/crossref"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/output"
	"github.com/yeasy/mdpress/internal/renderer"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/internal/toc"
	"github.com/yeasy/mdpress/internal/variables"
	"github.com/yeasy/mdpress/pkg/utils"
)

// TestE2E_QuickstartBuildVerify 测试完整的 quickstart → build → 验证输出 流程
func TestE2E_QuickstartBuildVerify(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 1. 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 2. 初始化主题
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		thm, err = tm.Get("technical")
		if err != nil {
			t.Fatalf("加载默认主题失败: %v", err)
		}
	}

	// 3. 初始化解析器
	parser := markdown.NewParser()

	// 4. 初始化交叉引用
	resolver := crossref.NewResolver()

	// 5. 解析所有章节
	var allHeadings []toc.HeadingInfo
	chaptersHTML := make([]renderer.ChapterHTML, 0)

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			t.Logf("跳过章节 %s: %v", ch.File, err)
			continue
		}

		// 模板变量替换
		content = variables.Expand(content, cfg)

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("解析章节 %s 失败: %v", ch.File, err)
		}

		// 图片处理
		chapterDir := filepath.Dir(chapterPath)
		htmlContent, _ = utils.ProcessImages(htmlContent, chapterDir, true)

		// 收集标题
		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
			resolver.RegisterSection(h.ID, h.Text, h.Level)
		}

		// 交叉引用
		htmlContent = resolver.ProcessHTML(htmlContent)
		htmlContent = resolver.AddCaptions(htmlContent)

		chapterID := headings[0].ID
		if len(headings) == 0 {
			chapterID = "chapter-" + string(rune('0'+i))
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      chapterID,
			Content: htmlContent,
		})
	}

	if len(chaptersHTML) == 0 {
		t.Fatal("没有成功处理任何章节")
	}

	// 6. 生成封面
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// 7. 生成目录
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// 8. 组装 HTML
	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("渲染 HTML 失败: %v", err)
	}

	// 9. 验证最终输出
	if fullHTML == "" {
		t.Fatal("输出 HTML 不应为空")
	}
	if !strings.Contains(fullHTML, "<!DOCTYPE html>") {
		t.Error("输出应包含 DOCTYPE 声明")
	}
	if !strings.Contains(fullHTML, cfg.Book.Title) {
		t.Error("输出应包含书名")
	}
	if !strings.Contains(fullHTML, cfg.Book.Author) {
		t.Error("输出应包含作者")
	}
	if !strings.Contains(fullHTML, "<nav") {
		t.Error("输出应包含目录导航")
	}
	// 验证封面
	if !strings.Contains(fullHTML, cfg.Book.Version) {
		t.Error("输出应包含版本号")
	}
	// 验证所有章节都在输出中
	for _, ch := range chaptersHTML {
		if !strings.Contains(fullHTML, ch.Title) {
			t.Errorf("输出应包含章节: %s", ch.Title)
		}
	}

	t.Logf("端到端测试完成: 输出 HTML 大小 %d 字节, 章节 %d 个", len(fullHTML), len(chaptersHTML))
}

// TestE2E_ZeroConfigMode 测试零配置模式的端到端流程
func TestE2E_ZeroConfigMode(t *testing.T) {
	// 创建临时目录模拟零配置场景
	tempDir := t.TempDir()

	// 创建一些 Markdown 文件（不包含 book.yaml）
	if err := os.WriteFile(filepath.Join(tempDir, "intro.md"), []byte(`# 简介

这是一个零配置测试。

## 功能特点

- 自动发现 Markdown 文件
- 无需配置文件
`), 0644); err != nil {
		t.Fatalf("write intro.md failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "chapter1.md"), []byte(`# 第一章 快速开始

## 安装

使用以下命令安装：

`+"```bash\ngo install github.com/yeasy/mdpress@latest\n```"+`

## 使用

运行 mdpress build 即可。
`), 0644); err != nil {
		t.Fatalf("write chapter1.md failed: %v", err)
	}

	// 使用 Discover 进行零配置加载
	cfg, err := config.Discover(tempDir)
	if err != nil {
		t.Fatalf("零配置发现失败: %v", err)
	}

	// 验证自动发现的配置
	if len(cfg.Chapters) == 0 {
		t.Fatal("零配置应自动发现章节")
	}

	t.Logf("零配置发现 %d 个章节", len(cfg.Chapters))

	// 初始化主题和解析器
	tm := theme.NewThemeManager()
	thm, _ := tm.Get("technical")
	parser := markdown.NewParser()

	// 解析所有发现的章节
	chaptersHTML := make([]renderer.ChapterHTML, 0)
	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			continue
		}

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			continue
		}

		chapterID := "chapter-auto-" + string(rune('0'+i))
		if len(headings) > 0 {
			chapterID = headings[0].ID
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      chapterID,
			Content: htmlContent,
		})
	}

	if len(chaptersHTML) == 0 {
		t.Fatal("零配置模式应至少处理一个章节")
	}

	// 渲染 HTML
	parts := &renderer.RenderParts{
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer 失败: %v", err)
	}
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("零配置渲染失败: %v", err)
	}

	if !strings.Contains(fullHTML, "<!DOCTYPE html>") {
		t.Error("零配置输出应包含完整 HTML 结构")
	}

	t.Logf("零配置端到端完成: 输出 %d 字节", len(fullHTML))
}

// TestE2E_HTMLOutput 测试 --format html 端到端输出
func TestE2E_HTMLOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, _ := tm.Get("technical")
	parser := markdown.NewParser()

	// 解析章节
	chaptersHTML := make([]renderer.ChapterHTML, 0)
	var allHeadings []toc.HeadingInfo

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			continue
		}

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			continue
		}

		chapterDir := filepath.Dir(chapterPath)
		htmlContent, _ = utils.ProcessImages(htmlContent, chapterDir, true)

		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
		}

		chapterID := "ch-" + string(rune('0'+i))
		if len(headings) > 0 {
			chapterID = headings[0].ID
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      chapterID,
			Content: htmlContent,
		})
	}

	// 生成目录
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// 使用 Standalone HTML 渲染器（对应 --format html）
	standaloneRenderer, err := renderer.NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer 失败: %v", err)
	}
	parts := &renderer.RenderParts{
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	standaloneHTML, err := standaloneRenderer.Render(parts)
	if err != nil {
		t.Fatalf("单页 HTML 渲染失败: %v", err)
	}

	// 验证单页 HTML 输出
	if standaloneHTML == "" {
		t.Fatal("单页 HTML 输出不应为空")
	}
	if !strings.Contains(standaloneHTML, "<!DOCTYPE html>") {
		t.Error("单页 HTML 应包含 DOCTYPE")
	}
	if !strings.Contains(standaloneHTML, "<style") {
		t.Error("单页 HTML 应包含内联样式（自包含）")
	}

	// 写入临时文件验证
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "test-output.html")
	err = os.WriteFile(outputPath, []byte(standaloneHTML), 0644)
	if err != nil {
		t.Fatalf("写入 HTML 文件失败: %v", err)
	}

	// 验证文件已创建且非空
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("输出文件不存在: %v", err)
	}
	if info.Size() == 0 {
		t.Error("输出 HTML 文件不应为空")
	}

	t.Logf("HTML 端到端完成: 文件大小 %d 字节", info.Size())
}

// TestE2E_EPubOutput 测试 ePub 输出流程
func TestE2E_EPubOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, _ := tm.Get("technical")
	parser := markdown.NewParser()

	chaptersHTML := make([]renderer.ChapterHTML, 0)

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			continue
		}
		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			continue
		}

		chapterID := "epub-ch-" + string(rune('0'+i))
		if len(headings) > 0 {
			chapterID = headings[0].ID
		}

		chaptersHTML = append(chaptersHTML, renderer.ChapterHTML{
			Title:   ch.Title,
			ID:      chapterID,
			Content: htmlContent,
		})
	}

	// 生成 ePub
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "test-output.epub")

	epubGen := output.NewEpubGenerator(output.EpubMeta{
		Title:        cfg.Book.Title,
		Subtitle:     cfg.Book.Subtitle,
		Author:       cfg.Book.Author,
		Language:     cfg.Book.Language,
		Version:      cfg.Book.Version,
		Description:  cfg.Book.Description,
		IncludeCover: cfg.Output.Cover,
	})
	epubGen.SetCSS(thm.ToCSS())
	for _, ch := range chaptersHTML {
		epubGen.AddChapter(output.EpubChapter{
			Title:    ch.Title,
			ID:       ch.ID,
			Filename: ch.ID + ".xhtml",
			HTML:     ch.Content,
		})
	}

	err = epubGen.Generate(outputPath)
	if err != nil {
		t.Fatalf("生成 ePub 失败: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("ePub 文件不存在: %v", err)
	}
	if info.Size() == 0 {
		t.Error("ePub 文件不应为空")
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("打开 ePub zip 失败: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	opf := readEpubEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `version="3.0"`) {
		t.Error("ePub 输出应使用 EPUB 3 package 版本")
	}

	nav := readEpubEntry(t, reader.File, "OEBPS/nav.xhtml")
	if !strings.Contains(nav, "Contents") {
		t.Error("ePub 输出应包含 nav.xhtml 导航文档")
	}

	t.Logf("ePub 端到端完成: 文件大小 %d 字节", info.Size())
}

// TestE2E_GlossaryIntegration 测试术语表集成流程
func TestE2E_GlossaryIntegration(t *testing.T) {
	testDataDir := getTestDataDir()
	glossaryPath := filepath.Join(testDataDir, "GLOSSARY.md")

	// 检查 GLOSSARY.md 是否存在
	if _, err := os.Stat(glossaryPath); os.IsNotExist(err) {
		t.Skip("测试数据中没有 GLOSSARY.md")
	}

	gloss, err := glossary.ParseFile(glossaryPath)
	if err != nil {
		t.Fatalf("解析术语表失败: %v", err)
	}

	if len(gloss.Terms) == 0 {
		t.Skip("术语表为空，跳过")
	}

	// 测试术语高亮功能
	testHTML := "<p>这是一个包含术语的测试段落。</p>"
	processedHTML := gloss.ProcessHTML(testHTML)

	// 渲染术语表页面
	glossHTML := gloss.RenderHTML()
	if glossHTML == "" {
		t.Error("术语表 HTML 不应为空")
	}

	t.Logf("术语表集成: %d 个术语, 处理后 HTML 长度 %d", len(gloss.Terms), len(processedHTML))
}

// TestE2E_MultiChapterTOC 测试多章节目录生成的完整性
func TestE2E_MultiChapterTOC(t *testing.T) {
	testDataDir := getTestDataDir()
	parser := markdown.NewParser()

	// 读取所有 Markdown 文件
	files := []string{"ch01.md", "ch02.md"}
	var allHeadings []toc.HeadingInfo

	for _, file := range files {
		filePath := filepath.Join(testDataDir, file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		_, headings, err := parser.Parse(data)
		if err != nil {
			continue
		}

		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{
				Level: h.Level,
				Text:  h.Text,
				ID:    h.ID,
			})
		}
	}

	if len(allHeadings) == 0 {
		t.Fatal("应提取到标题")
	}

	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// 验证目录结构
	if !strings.Contains(tocHTML, "<nav") {
		t.Error("多章节目录应包含 nav 标签")
	}
	if !strings.Contains(tocHTML, "<a") {
		t.Error("目录应包含链接")
	}

	// 验证所有标题都在目录中
	for _, h := range allHeadings {
		if h.Level <= 2 && !strings.Contains(tocHTML, h.Text) {
			t.Errorf("目录应包含标题: %s (level %d)", h.Text, h.Level)
		}
	}

	totalEntries := toc.CountEntries(entries)
	t.Logf("多章节目录: %d 个标题, %d 个目录条目", len(allHeadings), totalEntries)
}

func readEpubEntry(t *testing.T, files []*zip.File, name string) string {
	t.Helper()

	for _, file := range files {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("打开 ePub 条目 %s 失败: %v", name, err)
		}
		defer rc.Close() //nolint:errcheck

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("读取 ePub 条目 %s 失败: %v", name, err)
		}
		return string(data)
	}

	t.Fatalf("ePub 条目不存在: %s", name)
	return ""
}

// TestE2E_SiteOutput 测试 HTML 站点输出
func TestE2E_SiteOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, _ := tm.Get("technical")
	parser := markdown.NewParser()

	// 创建站点生成器
	siteGen := output.NewSiteGenerator(output.SiteMeta{
		Title:    cfg.Book.Title,
		Author:   cfg.Book.Author,
		Language: cfg.Book.Language,
	})
	siteGen.SetCSS(thm.ToCSS())

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			continue
		}
		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			continue
		}
		chapterID := "site-ch"
		if len(headings) > 0 {
			chapterID = headings[0].ID
		}
		filename := "ch_" + string(rune('0'+i)) + ".html"
		siteGen.AddChapter(output.SiteChapter{
			Title:    ch.Title,
			ID:       chapterID,
			Filename: filename,
			Content:  htmlContent,
		})
	}

	outputDir := t.TempDir()
	err = siteGen.Generate(outputDir)
	if err != nil {
		t.Fatalf("站点生成失败: %v", err)
	}

	// 验证站点输出
	indexPath := filepath.Join(outputDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("站点应生成 index.html")
	}

	indexContent, _ := os.ReadFile(indexPath)
	if len(indexContent) == 0 {
		t.Error("index.html 不应为空")
	}

	t.Logf("站点输出完成: %s", outputDir)
}

// TestE2E_VariableExpansion 测试模板变量替换的端到端流程
func TestE2E_VariableExpansion(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 创建包含模板变量的内容（使用 {{ book.title }} 格式）
	content := []byte("# {{ book.title }}\n\n作者: {{ book.author }}\n\n版本: {{ book.version }}")

	// 进行变量替换
	expanded := variables.Expand(content, cfg)

	// 解析替换后的 Markdown
	parser := markdown.NewParser()
	html, _, err := parser.Parse(expanded)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 验证变量被正确替换
	if strings.Contains(html, "{{ book.title }}") {
		t.Error("模板变量 {{ book.title }} 应被替换")
	}
	if !strings.Contains(html, cfg.Book.Title) {
		t.Error("HTML 应包含实际书名")
	}
	if !strings.Contains(html, cfg.Book.Author) {
		t.Error("HTML 应包含实际作者名")
	}
}
