// e2e_test.go End-to-end tests.
// Tests the full init -> build -> verify output flow, including zero-config mode and HTML output.
package tests

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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

// TestE2E_QuickstartBuildVerify tests the full quickstart -> build -> verify output flow
func TestE2E_QuickstartBuildVerify(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	// 1. Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 2. Initialize theme
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		thm, err = tm.Get("technical")
		if err != nil {
			t.Fatalf("failed to load default theme: %v", err)
		}
	}

	// 3. Initialize parser
	parser := markdown.NewParser()

	// 4. Initialize cross-references
	resolver := crossref.NewResolver()

	// 5. Parse all chapters
	var allHeadings []toc.HeadingInfo
	chaptersHTML := make([]renderer.ChapterHTML, 0)

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			t.Fatalf("failed to read chapter %s: %v", ch.File, err)
		}

		// Template variable substitution
		content = variables.Expand(content, cfg)

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Fatalf("failed to parse chapter %s: %v", ch.File, err)
		}

		// Image processing
		chapterDir := filepath.Dir(chapterPath)
		htmlContent, err = utils.ProcessImages(htmlContent, chapterDir, true)
		if err != nil {
			t.Fatalf("ProcessImages failed for chapter %s: %v", ch.File, err)
		}

		// Collect headings
		for _, h := range headings {
			allHeadings = append(allHeadings, toc.HeadingInfo{Level: h.Level, Text: h.Text, ID: h.ID})
			resolver.RegisterSection(h.ID, h.Text, h.Level)
		}

		// Cross-references
		htmlContent = resolver.ProcessHTML(htmlContent)
		htmlContent = resolver.AddCaptions(htmlContent)

		chapterID := "chapter-" + strconv.Itoa(i)
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
		t.Fatal("no chapters were successfully processed")
	}

	// 6. Generate cover
	coverGen := cover.NewCoverGenerator(cfg.Book)
	coverHTML := coverGen.RenderHTML()

	// 7. Generate TOC
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// 8. Assemble HTML
	parts := &renderer.RenderParts{
		CoverHTML:    coverHTML,
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("HTML rendering failed: %v", err)
	}

	// 9. Verify final output
	if fullHTML == "" {
		t.Fatal("output HTML should not be empty")
	}
	if !strings.Contains(fullHTML, "<!DOCTYPE html>") {
		t.Error("output should contain DOCTYPE declaration")
	}
	if !strings.Contains(fullHTML, cfg.Book.Title) {
		t.Error("output should contain book title")
	}
	if !strings.Contains(fullHTML, cfg.Book.Author) {
		t.Error("output should contain author")
	}
	if !strings.Contains(fullHTML, "<nav") {
		t.Error("output should contain TOC navigation")
	}
	// Verify cover
	if !strings.Contains(fullHTML, cfg.Book.Version) {
		t.Error("output should contain version number")
	}
	// Verify all chapters are in the output
	for _, ch := range chaptersHTML {
		if !strings.Contains(fullHTML, ch.Title) {
			t.Errorf("output should contain chapter: %s", ch.Title)
		}
	}

	t.Logf("end-to-end test complete: output HTML size %d bytes, %d chapters", len(fullHTML), len(chaptersHTML))
}

// TestE2E_ZeroConfigMode tests zero-config mode end-to-end flow
func TestE2E_ZeroConfigMode(t *testing.T) {
	// Create temp directory to simulate zero-config scenario
	tempDir := t.TempDir()

	// Create some Markdown files (without book.yaml)
	if err := os.WriteFile(filepath.Join(tempDir, "intro.md"), []byte(`# 简介

这是一个零配置测试。

## 功能特点

- 自动发现 Markdown 文件
- 无需配置文件
`), 0o644); err != nil {
		t.Fatalf("write intro.md failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "chapter1.md"), []byte(`# 第一章 快速开始

## 安装

使用以下命令安装：

`+"```bash\ngo install github.com/yeasy/mdpress@latest\n```"+`

## 使用

运行 mdpress build 即可。
`), 0o644); err != nil {
		t.Fatalf("write chapter1.md failed: %v", err)
	}

	// Use Discover for zero-config loading
	cfg, err := config.Discover(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("zero-config discovery failed: %v", err)
	}

	// Verify auto-discovered config
	if len(cfg.Chapters) == 0 {
		t.Fatal("zero-config should auto-discover chapters")
	}

	t.Logf("zero-config discovered %d chapters", len(cfg.Chapters))

	// Initialize theme and parser
	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}
	parser := markdown.NewParser()

	// Parse all discovered chapters
	chaptersHTML := make([]renderer.ChapterHTML, 0)
	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			t.Errorf("ReadFile %s failed: %v", chapterPath, err)
			continue
		}

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Errorf("Parse failed for chapter: %v", err)
			continue
		}

		chapterID := fmt.Sprintf("chapter-auto-%d", i)
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
		t.Fatal("zero-config mode should process at least one chapter")
	}

	// Render HTML
	parts := &renderer.RenderParts{
		ChaptersHTML: chaptersHTML,
	}

	htmlRenderer, err := renderer.NewHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewHTMLRenderer failed: %v", err)
	}
	fullHTML, err := htmlRenderer.Render(parts)
	if err != nil {
		t.Fatalf("zero-config rendering failed: %v", err)
	}

	if !strings.Contains(fullHTML, "<!DOCTYPE html>") {
		t.Error("zero-config output should contain complete HTML structure")
	}

	t.Logf("zero-config end-to-end complete: output %d bytes", len(fullHTML))
}

// TestE2E_HTMLOutput tests --format html end-to-end output
func TestE2E_HTMLOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}
	parser := markdown.NewParser()

	// Parse chapters
	chaptersHTML := make([]renderer.ChapterHTML, 0)
	var allHeadings []toc.HeadingInfo

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			t.Errorf("ReadFile %s failed: %v", chapterPath, err)
			continue
		}

		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Errorf("Parse failed for chapter: %v", err)
			continue
		}

		chapterDir := filepath.Dir(chapterPath)
		htmlContent, err = utils.ProcessImages(htmlContent, chapterDir, true)
		if err != nil {
			t.Fatalf("ProcessImages failed for chapter %s: %v", ch.File, err)
		}

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

	// Generate TOC
	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// Use standalone HTML renderer (for --format html)
	standaloneRenderer, err := renderer.NewStandaloneHTMLRenderer(cfg, thm)
	if err != nil {
		t.Fatalf("NewStandaloneHTMLRenderer failed: %v", err)
	}
	parts := &renderer.RenderParts{
		TOCHTML:      tocHTML,
		ChaptersHTML: chaptersHTML,
	}

	standaloneHTML, err := standaloneRenderer.Render(parts)
	if err != nil {
		t.Fatalf("standalone HTML rendering failed: %v", err)
	}

	// Verify standalone HTML output
	if standaloneHTML == "" {
		t.Fatal("standalone HTML output should not be empty")
	}
	if !strings.Contains(standaloneHTML, "<!DOCTYPE html>") {
		t.Error("standalone HTML should contain DOCTYPE")
	}
	if !strings.Contains(standaloneHTML, "<style") {
		t.Error("standalone HTML should contain inline styles (self-contained)")
	}

	// Write to temp file for verification
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "test-output.html")
	err = os.WriteFile(outputPath, []byte(standaloneHTML), 0o644)
	if err != nil {
		t.Fatalf("failed to write HTML file: %v", err)
	}

	// Verify file was created and is non-empty
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output HTML file should not be empty")
	}

	t.Logf("HTML end-to-end complete: file size %d bytes", info.Size())
}

// TestE2E_EPubOutput tests ePub output flow
func TestE2E_EPubOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}
	parser := markdown.NewParser()

	chaptersHTML := make([]renderer.ChapterHTML, 0)

	for i, ch := range cfg.Chapters {
		chapterPath := cfg.ResolvePath(ch.File)
		content, err := utils.ReadFile(chapterPath)
		if err != nil {
			t.Errorf("ReadFile %s failed: %v", chapterPath, err)
			continue
		}
		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Errorf("Parse failed for chapter: %v", err)
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

	// Generate ePub
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
		t.Fatalf("failed to generate ePub: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("ePub file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("ePub file should not be empty")
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("failed to open ePub zip: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	opf := readEpubEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `version="3.0"`) {
		t.Error("ePub output should use EPUB 3 package version")
	}

	nav := readEpubEntry(t, reader.File, "OEBPS/nav.xhtml")
	if !strings.Contains(nav, "Contents") {
		t.Error("ePub output should contain nav.xhtml navigation document")
	}

	t.Logf("ePub end-to-end complete: file size %d bytes", info.Size())
}

// TestE2E_GlossaryIntegration tests glossary integration flow
func TestE2E_GlossaryIntegration(t *testing.T) {
	testDataDir := getTestDataDir()
	glossaryPath := filepath.Join(testDataDir, "GLOSSARY.md")

	// Check if GLOSSARY.md exists
	if _, err := os.Stat(glossaryPath); errors.Is(err, fs.ErrNotExist) {
		t.Skip("no GLOSSARY.md in test data")
	}

	gloss, err := glossary.ParseFile(glossaryPath)
	if err != nil {
		t.Fatalf("failed to parse glossary: %v", err)
	}

	if len(gloss.Terms) == 0 {
		t.Skip("glossary is empty, skipping")
	}

	// Test glossary term highlighting
	testHTML := "<p>这是一个包含术语的测试段落。</p>"
	processedHTML := gloss.ProcessHTML(testHTML)
	if len(processedHTML) == 0 {
		t.Error("ProcessHTML should return non-empty result")
	}

	// Render glossary page
	glossHTML := gloss.RenderHTML()
	if glossHTML == "" {
		t.Error("glossary HTML should not be empty")
	}
}

// TestE2E_MultiChapterTOC tests completeness of multi-chapter TOC generation
func TestE2E_MultiChapterTOC(t *testing.T) {
	testDataDir := getTestDataDir()
	parser := markdown.NewParser()

	// Read all Markdown files
	files := []string{"ch01.md", "ch02.md"}
	var allHeadings []toc.HeadingInfo

	for _, file := range files {
		filePath := filepath.Join(testDataDir, file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("ReadFile %s failed: %v", filePath, err)
			continue
		}

		_, headings, err := parser.Parse(data)
		if err != nil {
			t.Errorf("Parse failed for %s: %v", file, err)
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
		t.Fatal("should have extracted headings")
	}

	tocGen := toc.NewGenerator()
	entries := tocGen.Generate(allHeadings)
	tocHTML := tocGen.RenderHTML(entries)

	// Verify TOC structure
	if !strings.Contains(tocHTML, "<nav") {
		t.Error("multi-chapter TOC should contain nav tag")
	}
	if !strings.Contains(tocHTML, "<a") {
		t.Error("TOC should contain links")
	}

	// Verify all headings are in the TOC
	for _, h := range allHeadings {
		if h.Level <= 2 && !strings.Contains(tocHTML, h.Text) {
			t.Errorf("TOC should contain heading: %s (level %d)", h.Text, h.Level)
		}
	}

	totalEntries := toc.CountEntries(entries)
	t.Logf("multi-chapter TOC: %d headings, %d TOC entries", len(allHeadings), totalEntries)
}

func readEpubEntry(t *testing.T, files []*zip.File, name string) string {
	t.Helper()

	for _, file := range files {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open ePub entry %s: %v", name, err)
		}
		defer rc.Close() //nolint:errcheck

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("failed to read ePub entry %s: %v", name, err)
		}
		return string(data)
	}

	t.Fatalf("ePub entry does not exist: %s", name)
	return ""
}

// TestE2E_SiteOutput tests HTML site output
func TestE2E_SiteOutput(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tm := theme.NewThemeManager()
	thm, err := tm.Get("technical")
	if err != nil {
		t.Fatalf("failed to get theme: %v", err)
	}
	parser := markdown.NewParser()

	// Create site generator
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
			t.Errorf("ReadFile %s failed: %v", chapterPath, err)
			continue
		}
		htmlContent, headings, err := parser.Parse(content)
		if err != nil {
			t.Errorf("Parse failed for chapter: %v", err)
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
		t.Fatalf("site generation failed: %v", err)
	}

	// Verify site output
	indexPath := filepath.Join(outputDir, "index.html")
	if _, err := os.Stat(indexPath); errors.Is(err, fs.ErrNotExist) {
		t.Error("site should generate index.html")
	}

	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	if len(indexContent) == 0 {
		t.Error("index.html should not be empty")
	}

	t.Logf("site output complete: %s", outputDir)
}

// TestE2E_VariableExpansion tests template variable expansion end-to-end flow
func TestE2E_VariableExpansion(t *testing.T) {
	testDataDir := getTestDataDir()
	configPath := filepath.Join(testDataDir, "book.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create content with template variables (using {{ book.title }} format)
	content := []byte("# {{ book.title }}\n\n作者: {{ book.author }}\n\n版本: {{ book.version }}")

	// Perform variable substitution
	expanded := variables.Expand(content, cfg)

	// Parse substituted Markdown
	parser := markdown.NewParser()
	html, _, err := parser.Parse(expanded)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	// Verify variables were correctly replaced
	if strings.Contains(html, "{{ book.title }}") {
		t.Error("template variable {{ book.title }} should be replaced")
	}
	if !strings.Contains(html, cfg.Book.Title) {
		t.Error("HTML should contain actual book title")
	}
	if !strings.Contains(html, cfg.Book.Author) {
		t.Error("HTML should contain actual author name")
	}
}
