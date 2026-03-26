package renderer

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// resolveTemplatePlaceholders replaces CDN URL placeholders that are not
// Go template actions but look like them (e.g. {{MERMAID_CDN_URL}}).
// Production code does the same replacement before parsing.
func resolveTemplatePlaceholders(tmpl string) string {
	r := strings.NewReplacer(
		"{{MERMAID_CDN_URL}}", "https://cdn.example.com/mermaid.min.js",
		"{{KATEX_CSS_URL}}", "https://cdn.example.com/katex.min.css",
		"{{KATEX_JS_URL}}", "https://cdn.example.com/katex.min.js",
		"{{KATEX_AUTO_RENDER_URL}}", "https://cdn.example.com/auto-render.min.js",
	)
	return r.Replace(tmpl)
}

// TestTemplateConstantExists verifies the htmlTemplate constant is defined and non-empty
func TestTemplateConstantExists(t *testing.T) {
	if htmlTemplate == "" {
		t.Fatal("htmlTemplate constant is empty")
	}
	if !strings.Contains(htmlTemplate, "<!DOCTYPE html>") {
		t.Error("htmlTemplate should contain DOCTYPE declaration")
	}
}

// TestTemplateParseable verifies the template string can be parsed by Go's template engine
func TestTemplateParseable(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse htmlTemplate: %v", err)
	}
	if tmpl == nil {
		t.Fatal("Template should not be nil after successful parsing")
	}
}

// TestTemplateHTMLStructure verifies the basic HTML5 structure is present
func TestTemplateHTMLStructure(t *testing.T) {
	requiredElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"<head>",
		"</head>",
		"<body>",
		"</body>",
		"<meta charset=\"UTF-8\">",
		"<meta name=\"viewport\"",
		"<style>",
		"</style>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(htmlTemplate, elem) {
			t.Errorf("htmlTemplate should contain %q", elem)
		}
	}
}

// TestTemplateMetaTags verifies required meta tags are present
func TestTemplateMetaTags(t *testing.T) {
	metaTags := []string{
		"<meta charset=\"UTF-8\">",
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">",
		"<meta http-equiv=\"X-UA-Compatible\" content=\"IE=edge\">",
		"<meta name=\"author\"",
		"<meta name=\"description\"",
	}

	for _, tag := range metaTags {
		if !strings.Contains(htmlTemplate, tag) {
			t.Errorf("htmlTemplate should contain meta tag: %q", tag)
		}
	}
}

// TestTemplateLanguageSupport verifies language attribute template variable
func TestTemplateLanguageSupport(t *testing.T) {
	if !strings.Contains(htmlTemplate, "{{if .Language}}{{.Language}}{{else}}en{{end}}") {
		t.Error("htmlTemplate should support Language variable with English fallback")
	}
}

// TestTemplateCoverSection verifies cover page section structure
func TestTemplateCoverSection(t *testing.T) {
	coverMarkers := []string{
		"{{if .CoverHTML}}",
		"<div class=\"cover-page\">",
		"{{.CoverHTML}}",
		"</div>",
	}

	for _, marker := range coverMarkers {
		if !strings.Contains(htmlTemplate, marker) {
			t.Errorf("htmlTemplate should contain cover section marker: %q", marker)
		}
	}
}

// TestTemplateTOCSection verifies table of contents section
func TestTemplateTOCSection(t *testing.T) {
	tocMarkers := []string{
		"{{if .TOCHTML}}",
		"<div class=\"toc-page\">",
		"{{.TOCHTML}}",
	}

	for _, marker := range tocMarkers {
		if !strings.Contains(htmlTemplate, marker) {
			t.Errorf("htmlTemplate should contain TOC section marker: %q", marker)
		}
	}
}

// TestTemplateChaptersSection verifies chapters section with range loop
func TestTemplateChaptersSection(t *testing.T) {
	if !strings.Contains(htmlTemplate, "{{range .Chapters}}") {
		t.Error("htmlTemplate should iterate over Chapters")
	}
	if !strings.Contains(htmlTemplate, "<div class=\"chapter\" id=\"{{.ID}}\">") {
		t.Error("htmlTemplate should render chapter with ID attribute")
	}
	if !strings.Contains(htmlTemplate, "<h1 class=\"chapter-title\">{{.Title}}</h1>") {
		t.Error("htmlTemplate should render chapter title")
	}
	if !strings.Contains(htmlTemplate, "<div class=\"chapter-content\">") {
		t.Error("htmlTemplate should have chapter content div")
	}
	if !strings.Contains(htmlTemplate, "{{.Content}}") {
		t.Error("htmlTemplate should render chapter content")
	}
	if !strings.Contains(htmlTemplate, "{{end}}") {
		t.Error("htmlTemplate should close range loop")
	}
}

// TestTemplateWatermark verifies watermark support
func TestTemplateWatermark(t *testing.T) {
	if !strings.Contains(htmlTemplate, "{{if .Watermark}}") {
		t.Error("htmlTemplate should support Watermark conditional")
	}
	if !strings.Contains(htmlTemplate, "<div class=\"watermark\">{{.Watermark}}</div>") {
		t.Error("htmlTemplate should render watermark div")
	}
	if !strings.Contains(htmlTemplate, "{{.WatermarkOpacity}}") {
		t.Error("htmlTemplate should include WatermarkOpacity in CSS")
	}
}

// TestTemplateCSSInclusion verifies CSS is properly embedded
func TestTemplateCSSInclusion(t *testing.T) {
	if !strings.Contains(htmlTemplate, "<style>") {
		t.Error("htmlTemplate should have style tag")
	}
	if !strings.Contains(htmlTemplate, "{{.CSS}}") {
		t.Error("htmlTemplate should include CSS template variable")
	}
	if !strings.Contains(htmlTemplate, "/* ============================================") {
		t.Error("htmlTemplate should contain CSS comments sections")
	}
}

// TestTemplateBaseStyles verifies base CSS styling rules
func TestTemplateBaseStyles(t *testing.T) {
	baseStyles := []string{
		"box-sizing: border-box;",
		"font-family:",
		"line-height: 1.6;",
		"color: #333;",
		"background-color: #fff;",
	}

	for _, style := range baseStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain base style: %q", style)
		}
	}
}

// TestTemplateCoverPageStyles verifies cover page specific styles
func TestTemplateCoverPageStyles(t *testing.T) {
	coverStyles := []string{
		".cover-page",
		"height: 100vh;",
		"display: flex;",
		"align-items: center;",
		"justify-content: center;",
		"page-break-after: always;",
	}

	for _, style := range coverStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain cover style: %q", style)
		}
	}
}

// TestTemplateTOCStyles verifies TOC page specific styles
func TestTemplateTOCStyles(t *testing.T) {
	tocStyles := []string{
		".toc-page",
		".toc-title",
		".toc-list",
		".toc-item",
		"list-style: none;",
	}

	for _, style := range tocStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain TOC style: %q", style)
		}
	}
}

// TestTemplateChapterStyles verifies chapter specific styles
func TestTemplateChapterStyles(t *testing.T) {
	chapterStyles := []string{
		".chapter",
		".chapter-title",
		".chapter-content",
		"page-break-before: always;",
		"page-break-inside: avoid;",
	}

	for _, style := range chapterStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain chapter style: %q", style)
		}
	}
}

// TestTemplateHeadingStyles verifies heading (h1-h6) styles
func TestTemplateHeadingStyles(t *testing.T) {
	headingStyles := map[string]string{
		"h1": "2em",
		"h2": "1.5em",
		"h3": "1.2em",
	}

	for heading, fontSize := range headingStyles {
		pattern := heading + " {"
		if !strings.Contains(htmlTemplate, pattern) {
			t.Errorf("htmlTemplate should contain %s styles", heading)
		}
		if !strings.Contains(htmlTemplate, fontSize) {
			t.Errorf("htmlTemplate should define font-size for %s", heading)
		}
	}
}

// TestTemplateParagraphStyles verifies paragraph styles
func TestTemplateParagraphStyles(t *testing.T) {
	if !strings.Contains(htmlTemplate, "p {") {
		t.Error("htmlTemplate should contain paragraph styles")
	}
	if !strings.Contains(htmlTemplate, "text-align: justify;") {
		t.Error("htmlTemplate paragraphs should be justified")
	}
}

// TestTemplateListStyles verifies list styles
func TestTemplateListStyles(t *testing.T) {
	listStyles := []string{
		"ul, ol {",
		"padding-left: 2rem;",
		"ul li, ol li {",
	}

	for _, style := range listStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain list style: %q", style)
		}
	}
}

// TestTemplateCodeStyles verifies code and pre block styles
func TestTemplateCodeStyles(t *testing.T) {
	codeStyles := []string{
		"code {",
		"ui-monospace",
		"pre {",
		"border: 1px solid #ddd;",
		"overflow-x: auto;",
	}

	for _, style := range codeStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain code style: %q", style)
		}
	}
}

// TestTemplateTableStyles verifies table styles
func TestTemplateTableStyles(t *testing.T) {
	tableStyles := []string{
		"table {",
		"border-collapse: collapse;",
		"table-layout: fixed;",
		"table th {",
		"table td {",
		"table tbody tr:nth-child(even)",
	}

	for _, style := range tableStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain table style: %q", style)
		}
	}
}

// TestTemplateBlockquoteStyles verifies blockquote styles
func TestTemplateBlockquoteStyles(t *testing.T) {
	bqStyles := []string{
		"blockquote {",
		"border-left: 4px solid #667eea;",
		"background-color: #f9f9f9;",
	}

	for _, style := range bqStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain blockquote style: %q", style)
		}
	}
}

// TestTemplateImageStyles verifies image and figure styles
func TestTemplateImageStyles(t *testing.T) {
	imgStyles := []string{
		"img {",
		"max-width: 100%;",
		"height: auto;",
		"figure {",
		"figcaption {",
	}

	for _, style := range imgStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain image style: %q", style)
		}
	}
}

// TestTemplateLinkStyles verifies anchor and link styles
func TestTemplateLinkStyles(t *testing.T) {
	linkStyles := []string{
		"a {",
		"color: #0066cc;",
		"a:hover {",
		"text-decoration: underline;",
	}

	for _, style := range linkStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain link style: %q", style)
		}
	}
}

// TestTemplatePrintMediaStyles verifies @media print styles
func TestTemplatePrintMediaStyles(t *testing.T) {
	printStyles := []string{
		"@media print {",
		".no-print {",
		"display: none !important;",
	}

	for _, style := range printStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain print style: %q", style)
		}
	}
}

// TestTemplatePageRules verifies @page rules
func TestTemplatePageRules(t *testing.T) {
	pageRules := []string{
		"@page {",
		"size: A4;",
		"margin: 2cm;",
		"@page :first {",
	}

	for _, rule := range pageRules {
		if !strings.Contains(htmlTemplate, rule) {
			t.Errorf("htmlTemplate should contain page rule: %q", rule)
		}
	}
}

// TestTemplateWatermarkStyles verifies watermark CSS styles
func TestTemplateWatermarkStyles(t *testing.T) {
	watermarkStyles := []string{
		".watermark {",
		"position: fixed;",
		"transform: translate(-50%, -50%) rotate(-45deg);",
		"font-size: 80px;",
		"white-space: nowrap;",
	}

	for _, style := range watermarkStyles {
		if !strings.Contains(htmlTemplate, style) {
			t.Errorf("htmlTemplate should contain watermark style: %q", style)
		}
	}
}

// TestTemplateMermaidScript verifies Mermaid diagram support script
func TestTemplateMermaidScript(t *testing.T) {
	if !strings.Contains(htmlTemplate, "document.querySelector('.mermaid')") {
		t.Error("htmlTemplate should check for Mermaid diagrams")
	}
	if !strings.Contains(htmlTemplate, "{{MERMAID_CDN_URL}}") {
		t.Error("htmlTemplate should have Mermaid CDN URL placeholder")
	}
	if !strings.Contains(htmlTemplate, "mermaid.initialize") {
		t.Error("htmlTemplate should initialize Mermaid")
	}
}

// TestTemplateKaTeXScript verifies KaTeX math formula support script
func TestTemplateKaTeXScript(t *testing.T) {
	if !strings.Contains(htmlTemplate, "document.querySelector('.math')") {
		t.Error("htmlTemplate should check for math formulas")
	}
	if !strings.Contains(htmlTemplate, "{{KATEX_CSS_URL}}") {
		t.Error("htmlTemplate should have KaTeX CSS URL placeholder")
	}
	if !strings.Contains(htmlTemplate, "{{KATEX_JS_URL}}") {
		t.Error("htmlTemplate should have KaTeX JS URL placeholder")
	}
	if !strings.Contains(htmlTemplate, "{{KATEX_AUTO_RENDER_URL}}") {
		t.Error("htmlTemplate should have KaTeX auto-render URL placeholder")
	}
	if !strings.Contains(htmlTemplate, "renderMathInElement") {
		t.Error("htmlTemplate should call renderMathInElement")
	}
}

// TestTemplateWithMinimalData tests rendering with minimal data
func TestTemplateWithMinimalData(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "",
		Author:           "",
		Language:         "",
		CSS:              "",
		CoverHTML:        "",
		TOCHTML:          "",
		Chapters:         []any{},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with minimal data: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Output should contain DOCTYPE")
	}
	if !strings.Contains(result, "<html") {
		t.Error("Output should contain html tag")
	}
}

// TestTemplateWithCompleteData tests rendering with complete data
func TestTemplateWithCompleteData(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Test Book",
		Author:    "Test Author",
		Language:  "en",
		CSS:       template.CSS("body { font-size: 14px; }"),
		CoverHTML: template.HTML("<div class='cover'><h1>Book Title</h1></div>"),
		TOCHTML:   template.HTML("<div class='toc'><h2>Contents</h2></div>"),
		Chapters: []chapter{
			{Title: "Chapter 1", ID: "ch1", Content: template.HTML("<p>Content 1</p>")},
			{Title: "Chapter 2", ID: "ch2", Content: template.HTML("<p>Content 2</p>")},
		},
		HeaderText:       "My Header",
		FooterText:       "Page",
		Watermark:        "CONFIDENTIAL",
		WatermarkOpacity: 0.3,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with complete data: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "Test Book") {
		t.Error("Output should contain title")
	}
	if !strings.Contains(result, "Test Author") {
		t.Error("Output should contain author")
	}
	if !strings.Contains(result, "lang=\"en\"") {
		t.Error("Output should contain language attribute")
	}
	if !strings.Contains(result, "font-size: 14px") {
		t.Error("Output should contain custom CSS")
	}
	if !strings.Contains(result, "Book Title") {
		t.Error("Output should contain cover content")
	}
	if !strings.Contains(result, "Contents") {
		t.Error("Output should contain TOC content")
	}
	if !strings.Contains(result, "Chapter 1") {
		t.Error("Output should contain chapter 1")
	}
	if !strings.Contains(result, "Chapter 2") {
		t.Error("Output should contain chapter 2")
	}
	if !strings.Contains(result, "CONFIDENTIAL") {
		t.Error("Output should contain watermark")
	}
}

// TestTemplateWithEmptyChapters tests rendering with empty chapters list
func TestTemplateWithEmptyChapters(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS("body { }"),
		CoverHTML:        template.HTML("<div>Cover</div>"),
		TOCHTML:          template.HTML("<div>TOC</div>"),
		Chapters:         []any{},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with empty chapters: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "Cover") {
		t.Error("Output should contain cover even with empty chapters")
	}
	if !strings.Contains(result, "TOC") {
		t.Error("Output should contain TOC even with empty chapters")
	}
}

// TestTemplateWithNilCoverAndTOC tests rendering with nil cover and TOC
func TestTemplateWithNilCoverAndTOC(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Test",
		Author:    "Author",
		Language:  "en",
		CSS:       template.CSS(""),
		CoverHTML: template.HTML(""), // Empty
		TOCHTML:   template.HTML(""), // Empty
		Chapters: []chapter{
			{Title: "Ch1", ID: "ch1", Content: template.HTML("<p>Content</p>")},
		},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with nil cover/TOC: %v", err)
	}

	result := buf.String()
	// Should still render chapters
	if !strings.Contains(result, "Ch1") {
		t.Error("Output should contain chapter even without cover/TOC")
	}
	if !strings.Contains(result, "Content") {
		t.Error("Output should contain chapter content")
	}
}

// TestTemplateWithSpecialCharacters tests rendering with special HTML characters
func TestTemplateWithSpecialCharacters(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Test & Book <Title>",
		Author:    "Author \"Name\" & Co.",
		Language:  "en",
		CSS:       template.CSS("/* comment */"),
		CoverHTML: template.HTML("<div>A &amp; B</div>"),
		TOCHTML:   template.HTML(""),
		Chapters: []chapter{
			{Title: "Chapter & Content", ID: "ch-1", Content: template.HTML("<p>Test &lt; &gt;</p>")},
		},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with special characters: %v", err)
	}

	result := buf.String()
	if result == "" {
		t.Error("Output should not be empty")
	}
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Output should still contain valid HTML structure")
	}
}

// TestTemplateWithCJKCharacters tests rendering with CJK (Chinese/Japanese/Korean) characters
func TestTemplateWithCJKCharacters(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "测试图书",
		Author:    "测试作者",
		Language:  "zh-CN",
		CSS:       template.CSS(""),
		CoverHTML: template.HTML("<div>封面</div>"),
		TOCHTML:   template.HTML("<div>目录</div>"),
		Chapters: []chapter{
			{Title: "第一章", ID: "ch1", Content: template.HTML("<p>内容</p>")},
		},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with CJK characters: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "测试图书") {
		t.Error("Output should contain Chinese title")
	}
	if !strings.Contains(result, "zh-CN") {
		t.Error("Output should contain correct language attribute")
	}
}

// TestTemplateLongContent tests rendering with very long content
func TestTemplateLongContent(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	// Create long content
	longContent := strings.Repeat("<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>", 100)

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Long Content Test",
		Author:    "Author",
		Language:  "en",
		CSS:       template.CSS(""),
		CoverHTML: template.HTML(""),
		TOCHTML:   template.HTML(""),
		Chapters: []chapter{
			{Title: "Chapter 1", ID: "ch1", Content: template.HTML(longContent)},
		},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with long content: %v", err)
	}

	result := buf.String()
	if len(result) == 0 {
		t.Error("Output should not be empty with long content")
	}
}

// TestTemplateMultipleChapters tests rendering with multiple chapters
func TestTemplateMultipleChapters(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	chapters := make([]chapter, 0)
	for i := 1; i <= 10; i++ {
		chapters = append(chapters, chapter{
			Title:   "Chapter " + string(rune(i+'0'-1)),
			ID:      "ch" + string(rune(i+'0'-1)),
			Content: template.HTML("<p>Content " + string(rune(i+'0'-1)) + "</p>"),
		})
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Multi-Chapter Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS(""),
		CoverHTML:        template.HTML(""),
		TOCHTML:          template.HTML(""),
		Chapters:         chapters,
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template with multiple chapters: %v", err)
	}

	result := buf.String()
	// Verify all chapters are rendered
	for i := 1; i <= 10; i++ {
		chNum := string(rune(i + '0' - 1))
		if !strings.Contains(result, "Chapter "+chNum) {
			t.Errorf("Output should contain Chapter %s", chNum)
		}
	}
}

// TestTemplateChapterIDAsPoundAnchor tests chapter IDs work as anchor links
func TestTemplateChapterIDAsPoundAnchor(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Test",
		Author:    "Author",
		Language:  "en",
		CSS:       template.CSS(""),
		CoverHTML: template.HTML(""),
		TOCHTML:   template.HTML(""),
		Chapters: []chapter{
			{Title: "Introduction", ID: "intro", Content: template.HTML("<p>Intro</p>")},
			{Title: "Chapter One", ID: "chapter-one", Content: template.HTML("<p>One</p>")},
		},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, `id="intro"`) {
		t.Error("Output should contain chapter ID as id attribute")
	}
	if !strings.Contains(result, `id="chapter-one"`) {
		t.Error("Output should contain hyphenated chapter ID")
	}
}

// TestTemplateCSSIsProperlyEscaped tests CSS content is not over-escaped
func TestTemplateCSSIsProperlyEscaped(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS("body { content: \"value\"; }"),
		CoverHTML:        template.HTML(""),
		TOCHTML:          template.HTML(""),
		Chapters:         []any{},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	// CSS should not be HTML-escaped (template.CSS prevents that)
	if !strings.Contains(result, "content: \"value\";") {
		t.Error("CSS should not be HTML-escaped")
	}
}

// TestTemplateHTMLInCoverIsNotEscaped tests HTML in cover is rendered properly
func TestTemplateHTMLInCoverIsNotEscaped(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS(""),
		CoverHTML:        template.HTML("<div class='cover-wrapper'><h1>My Book</h1><p>Subtitle</p></div>"),
		TOCHTML:          template.HTML(""),
		Chapters:         []any{},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	// HTML should be rendered, not escaped
	if !strings.Contains(result, "<div class='cover-wrapper'>") {
		t.Error("Cover HTML should not be escaped")
	}
	if !strings.Contains(result, "<h1>My Book</h1>") {
		t.Error("Cover HTML tags should be rendered")
	}
}

// TestTemplateDefaultLanguageFallback tests language defaults to English
func TestTemplateDefaultLanguageFallback(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string // Empty string should trigger fallback
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "", // Empty - should default to "en"
		CSS:              template.CSS(""),
		CoverHTML:        template.HTML(""),
		TOCHTML:          template.HTML(""),
		Chapters:         []any{},
		WatermarkOpacity: 0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, `lang="en"`) {
		t.Error("Output should default to lang=\"en\" when Language is empty")
	}
}

// TestTemplateZeroWatermarkOpacity tests watermark opacity of 0
func TestTemplateZeroWatermarkOpacity(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS(""),
		CoverHTML:        template.HTML(""),
		TOCHTML:          template.HTML(""),
		Chapters:         []any{},
		Watermark:        "",
		WatermarkOpacity: 0.0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	// Should not error with zero opacity
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Template should render correctly with zero watermark opacity")
	}
}

// TestTemplateHighWatermarkOpacity tests watermark opacity of 1.0
func TestTemplateHighWatermarkOpacity(t *testing.T) {
	tmpl, err := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []any
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:            "Test",
		Author:           "Author",
		Language:         "en",
		CSS:              template.CSS(""),
		CoverHTML:        template.HTML(""),
		TOCHTML:          template.HTML(""),
		Chapters:         []any{},
		Watermark:        "SECRET",
		WatermarkOpacity: 1.0,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "SECRET") {
		t.Error("Output should contain watermark text")
	}
}

// TestTemplateContainsScriptTags verifies scripts are properly formed
func TestTemplateContainsScriptTags(t *testing.T) {
	if !strings.Contains(htmlTemplate, "<script>") {
		t.Error("htmlTemplate should contain script tags")
	}
	if !strings.Contains(htmlTemplate, "</script>") {
		t.Error("htmlTemplate should close script tags")
	}
	// Count script tags
	openCount := strings.Count(htmlTemplate, "<script>")
	closeCount := strings.Count(htmlTemplate, "</script>")
	if openCount != closeCount {
		t.Errorf("Mismatched script tags: %d open, %d close", openCount, closeCount)
	}
}

// TestTemplateMediaQueryStructure verifies @media queries are properly formed
func TestTemplateMediaQueryStructure(t *testing.T) {
	if !strings.Contains(htmlTemplate, "@media print") {
		t.Error("htmlTemplate should contain @media print query")
	}
	if !strings.Contains(htmlTemplate, "@media print {") {
		t.Error("@media print should have opening brace")
	}
}

// TestTemplateClassNamesConsistency checks CSS class names are used consistently
func TestTemplateClassNamesConsistency(t *testing.T) {
	classNames := []string{
		"cover-page",
		"toc-page",
		"chapter",
		"chapter-title",
		"chapter-content",
		"watermark",
	}

	for _, className := range classNames {
		// Class should be defined in CSS
		cssDefinition := "." + className + " {"
		if !strings.Contains(htmlTemplate, cssDefinition) {
			t.Errorf("CSS should define .%s", className)
		}
	}
}

// BenchmarkTemplateRendering benchmarks basic template rendering
func BenchmarkTemplateRendering(b *testing.B) {
	tmpl, _ := template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))

	type chapter struct {
		Title   string
		ID      string
		Content template.HTML
	}

	data := struct {
		Title            string
		Author           string
		Language         string
		CSS              template.CSS
		CoverHTML        template.HTML
		TOCHTML          template.HTML
		Chapters         []chapter
		HeaderText       string
		FooterText       string
		Watermark        string
		WatermarkOpacity float64
	}{
		Title:     "Test Book",
		Author:    "Test Author",
		Language:  "en",
		CSS:       template.CSS("body { font-size: 14px; }"),
		CoverHTML: template.HTML("<div>Cover</div>"),
		TOCHTML:   template.HTML("<div>TOC</div>"),
		Chapters: []chapter{
			{Title: "Ch1", ID: "ch1", Content: template.HTML("<p>Content</p>")},
		},
		Watermark:        "TEST",
		WatermarkOpacity: 0.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = tmpl.Execute(&buf, data)
	}
}

// BenchmarkTemplateParsing benchmarks template parsing
func BenchmarkTemplateParsing(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = template.New("book").Parse(resolveTemplatePlaceholders(htmlTemplate))
	}
}
