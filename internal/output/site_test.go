package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSiteGeneratorIndexPage verifies that index.html renders the first chapter
// directly (no HTTP redirect) so the SPA loads instantly at the site root.
func TestSiteGeneratorIndexPage(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add a chapter with explicit filename.
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1>",
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify index.html exists.
	indexPath := filepath.Join(dir, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.html not created: %v", err)
	}

	indexHTML := string(data)

	if !strings.Contains(indexHTML, "<!DOCTYPE html>") {
		t.Error("index.html should be valid HTML")
	}
	// The page should embed the first chapter's content, not a redirect.
	if !strings.Contains(indexHTML, "Chapter 1") {
		t.Error("index.html should contain the first chapter content")
	}
	// The SPA router should recognize ch1.html as the active file.
	if !strings.Contains(indexHTML, `class="sidebar-home-link" href="/index.html"`) {
		t.Error("index.html should point the sidebar title to index.html")
	}
	if strings.Contains(indexHTML, `<span class="bc-sep">›</span>`) {
		t.Error("index.html should not render an extra breadcrumb segment for the first chapter")
	}
	if strings.Contains(indexHTML, `href="../index.html"`) {
		t.Error("index.html should not use parent-directory home links")
	}
	// No meta-refresh redirect.
	if strings.Contains(indexHTML, "meta http-equiv=\"refresh\"") {
		t.Error("index.html must not use a meta-refresh redirect")
	}
}

// TestSiteGeneratorPrevNextNavigation creates 3 chapters and verifies each page has correct prev/next links.
func TestSiteGeneratorPrevNextNavigation(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add 3 chapters
	for i := 1; i <= 3; i++ {
		gen.AddChapter(SiteChapter{
			Title:    fmt.Sprintf("Chapter %d", i),
			ID:       fmt.Sprintf("ch%d", i),
			Filename: fmt.Sprintf("ch%d.html", i),
			Content:  fmt.Sprintf("<h1>Chapter %d</h1>", i),
		})
	}

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Test first page: no prev, has next
	ch1Data, err := os.ReadFile(filepath.Join(dir, "ch1.html"))
	if err != nil {
		t.Fatalf("read ch1.html failed: %v", err)
	}
	ch1HTML := string(ch1Data)

	// First page should not have a prev link
	if strings.Contains(ch1HTML, `href="ch0.html"`) {
		t.Error("first page should not have a previous link")
	}
	// But should have a next link
	if !strings.Contains(ch1HTML, `href="/ch2.html"`) {
		t.Error("first page should link to next page (ch2.html)")
	}
	if !strings.Contains(ch1HTML, "Chapter 2") {
		t.Error("first page should show next page title (Chapter 2)")
	}

	// Test middle page: has both prev and next
	ch2Data, err := os.ReadFile(filepath.Join(dir, "ch2.html"))
	if err != nil {
		t.Fatalf("read ch2.html failed: %v", err)
	}
	ch2HTML := string(ch2Data)

	if !strings.Contains(ch2HTML, `href="/ch1.html"`) {
		t.Error("middle page should have a previous link to ch1.html")
	}
	if !strings.Contains(ch2HTML, "Chapter 1") {
		t.Error("middle page should show previous page title (Chapter 1)")
	}
	if !strings.Contains(ch2HTML, `href="/ch3.html"`) {
		t.Error("middle page should have a next link to ch3.html")
	}
	if !strings.Contains(ch2HTML, "Chapter 3") {
		t.Error("middle page should show next page title (Chapter 3)")
	}

	// Test last page: has prev, no next
	ch3Data, err := os.ReadFile(filepath.Join(dir, "ch3.html"))
	if err != nil {
		t.Fatalf("read ch3.html failed: %v", err)
	}
	ch3HTML := string(ch3Data)

	if !strings.Contains(ch3HTML, `href="/ch2.html"`) {
		t.Error("last page should have a previous link to ch2.html")
	}
	if !strings.Contains(ch3HTML, "Chapter 2") {
		t.Error("last page should show previous page title (Chapter 2)")
	}
	// Last page should not have a next link
	if strings.Contains(ch3HTML, `class="next" href="ch4.html"`) {
		t.Error("last page should not have a next link")
	}
}

func TestSiteGeneratorCurrentFileUsesRelativePath(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "chapter-1/index.html",
		Content:  "<h1>Chapter 1</h1>",
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pageData, err := os.ReadFile(filepath.Join(dir, "chapter-1", "index.html"))
	if err != nil {
		t.Fatalf("read nested page failed: %v", err)
	}

	if !strings.Contains(string(pageData), `var currentFile = 'chapter-1\/index.html';`) {
		t.Error("page script should keep the full relative path for currentFile")
	}
}

// TestSiteGeneratorFlattenNestedChapters creates chapters with children and verifies flattenChapters produces correct order and count.
func TestSiteGeneratorFlattenNestedChapters(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Create a chapter hierarchy: Ch1 (with Ch1.1 and Ch1.2), Ch2
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1>",
		Children: []SiteChapter{
			{
				Title:    "Chapter 1.1",
				ID:       "ch1.1",
				Filename: "ch1.1.html",
				Content:  "<h1>Chapter 1.1</h1>",
			},
			{
				Title:    "Chapter 1.2",
				ID:       "ch1.2",
				Filename: "ch1.2.html",
				Content:  "<h1>Chapter 1.2</h1>",
			},
		},
	})
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 2",
		ID:       "ch2",
		Filename: "ch2.html",
		Content:  "<h1>Chapter 2</h1>",
	})

	// Call flattenChapters (unexported, but testable since we're in the same package)
	flattened := gen.flattenChapters(gen.Chapters)

	// Should have 4 pages total: ch1, ch1.1, ch1.2, ch2
	if len(flattened) != 4 {
		t.Errorf("flattenChapters should return 4 pages, got %d", len(flattened))
	}

	// Verify order
	expectedTitles := []string{"Chapter 1", "Chapter 1.1", "Chapter 1.2", "Chapter 2"}
	for i, chapter := range flattened {
		if chapter.Title != expectedTitles[i] {
			t.Errorf("page %d: expected title %q, got %q", i, expectedTitles[i], chapter.Title)
		}
	}

	// Verify filenames
	expectedFilenames := []string{"ch1.html", "ch1.1.html", "ch1.2.html", "ch2.html"}
	for i, chapter := range flattened {
		if chapter.Filename != expectedFilenames[i] {
			t.Errorf("page %d: expected filename %q, got %q", i, expectedFilenames[i], chapter.Filename)
		}
	}
}

// TestSiteGeneratorAutoFilenames adds chapters without filenames and verifies they get auto-assigned filenames.
func TestSiteGeneratorAutoFilenames(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add chapters without explicit filenames
	gen.AddChapter(SiteChapter{
		Title:   "Chapter 1",
		ID:      "ch1",
		Content: "<h1>Chapter 1</h1>",
		// Filename intentionally omitted
	})
	gen.AddChapter(SiteChapter{
		Title:   "Chapter 2",
		ID:      "ch2",
		Content: "<h1>Chapter 2</h1>",
		// Filename intentionally omitted
	})
	gen.AddChapter(SiteChapter{
		Title:   "Chapter 3",
		ID:      "ch3",
		Content: "<h1>Chapter 3</h1>",
		// Filename intentionally omitted
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify auto-assigned filenames exist
	expectedFiles := []string{"page_0.html", "page_1.html", "page_2.html", "index.html"}
	for _, filename := range expectedFiles {
		filePath := filepath.Join(dir, filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("expected file %s not created: %v", filename, err)
		}
	}

	// Verify the pages link to each other with auto-assigned filenames
	page0Data, err := os.ReadFile(filepath.Join(dir, "page_0.html"))
	if err != nil {
		t.Fatalf("read page_0.html failed: %v", err)
	}
	page0HTML := string(page0Data)

	if !strings.Contains(page0HTML, `href="/page_1.html"`) {
		t.Error("page_0.html should link to page_1.html")
	}
}

// TestSiteGeneratorBranchActive tests isChapterBranchActive: a chapter that is the active file should return true.
// A chapter whose child is active should also return true. Unrelated chapters should return false.
func TestSiteGeneratorBranchActive(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Create a hierarchy for testing
	parent := SiteChapter{
		Title:    "Parent",
		ID:       "parent",
		Filename: "parent.html",
		Children: []SiteChapter{
			{
				Title:    "Child 1",
				ID:       "child1",
				Filename: "child1.html",
			},
			{
				Title:    "Child 2",
				ID:       "child2",
				Filename: "child2.html",
			},
		},
	}

	sibling := SiteChapter{
		Title:    "Sibling",
		ID:       "sibling",
		Filename: "sibling.html",
	}

	gen.AddChapter(parent)
	gen.AddChapter(sibling)

	// Test 1: parent chapter itself is active
	if !gen.isChapterBranchActive(parent, "parent.html") {
		t.Error("parent chapter should be active when its own filename is active")
	}

	// Test 2: parent chapter is active when a child is active
	if !gen.isChapterBranchActive(parent, "child1.html") {
		t.Error("parent chapter should be active when child1.html is active")
	}
	if !gen.isChapterBranchActive(parent, "child2.html") {
		t.Error("parent chapter should be active when child2.html is active")
	}

	// Test 3: sibling chapter is not active when parent or children are active
	if gen.isChapterBranchActive(sibling, "parent.html") {
		t.Error("sibling chapter should not be active when parent is active")
	}
	if gen.isChapterBranchActive(sibling, "child1.html") {
		t.Error("sibling chapter should not be active when child1.html is active")
	}

	// Test 4: unrelated file doesn't make any chapter active
	if gen.isChapterBranchActive(parent, "unrelated.html") {
		t.Error("parent chapter should not be active for unrelated files")
	}
	if gen.isChapterBranchActive(sibling, "unrelated.html") {
		t.Error("sibling chapter should not be active for unrelated files")
	}
}

// TestSiteGeneratorHeadingsInSidebar adds chapters with headings and verifies sidebar contains heading links.
func TestSiteGeneratorHeadingsInSidebar(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add a chapter with nested headings
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1><h2 id=\"sec-1\">Section 1</h2><h3 id=\"subsec-1\">Subsection 1</h3><h2 id=\"sec-2\">Section 2</h2>",
		Headings: []SiteNavHeading{
			{
				Title: "Section 1",
				ID:    "sec-1",
				Children: []SiteNavHeading{
					{
						Title: "Subsection 1",
						ID:    "subsec-1",
					},
				},
			},
			{
				Title: "Section 2",
				ID:    "sec-2",
			},
		},
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read the generated page
	pagePath := filepath.Join(dir, "ch1.html")
	data, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}
	html := string(data)

	// With maxSidebarHeadingDepth = 0, NO in-page headings should appear
	// in the sidebar — only chapter titles are shown.
	if strings.Contains(html, `href="ch1.html#sec-1"`) {
		t.Error("sidebar should NOT contain heading links when maxSidebarHeadingDepth is 0")
	}
	if strings.Contains(html, `href="ch1.html#subsec-1"`) {
		t.Error("sidebar should NOT contain sub-heading links when maxSidebarHeadingDepth is 0")
	}

	// Chapter-level content (Section 1, Section 2) still appears in page body
	if !strings.Contains(html, "Section 1") {
		t.Error("page content should contain 'Section 1' text")
	}
	if !strings.Contains(html, "Section 2") {
		t.Error("page content should contain 'Section 2' text")
	}
}

// TestSiteGeneratorLanguageAttribute verifies the generated HTML has the correct lang attribute from SiteMeta.Language.
func TestSiteGeneratorLanguageAttribute(t *testing.T) {
	tests := []struct {
		lang string
		name string
	}{
		{"en-US", "English (US)"},
		{"fr-FR", "French"},
		{"zh-CN", "Chinese (Simplified)"},
		{"ja-JP", "Japanese"},
		{"", "Empty language"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			gen := NewSiteGenerator(SiteMeta{
				Title:    "Test Book",
				Author:   "Author",
				Language: tt.lang,
			})
			gen.AddChapter(SiteChapter{
				Title:    "Chapter 1",
				ID:       "ch1",
				Filename: "ch1.html",
				Content:  "<h1>Chapter 1</h1>",
			})

			if err := gen.Generate(dir); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			pagePath := filepath.Join(dir, "ch1.html")
			data, err := os.ReadFile(pagePath)
			if err != nil {
				t.Fatalf("read page failed: %v", err)
			}
			html := string(data)

			// Verify lang attribute exists in the html tag
			if tt.lang == "" {
				// If language is empty, it should still have the lang attribute with the value
				if !strings.Contains(html, `<html lang=""`) {
					t.Error("html tag should have lang attribute even if empty")
				}
			} else {
				langAttr := fmt.Sprintf(`<html lang="%s"`, tt.lang)
				if !strings.Contains(html, langAttr) {
					t.Errorf("html tag should have lang=\"%s\"", tt.lang)
				}
			}
		})
	}
}

// TestSiteGeneratorMermaidScript adds chapter content containing mermaid diagrams and verifies the page output contains the mermaid script loading code.
func TestSiteGeneratorMermaidScript(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add a chapter with mermaid diagram
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content: `<h1>Chapter 1</h1>
<div class="mermaid">
graph TD
    A[Start] --> B[Process]
    B --> C[End]
</div>`,
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pagePath := filepath.Join(dir, "ch1.html")
	data, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}
	html := string(data)

	// Verify the mermaid script loading code is present
	if !strings.Contains(html, "ensureMermaid") {
		t.Error("page should contain ensureMermaid function")
	}
	if !strings.Contains(html, "window.mermaid") {
		t.Error("page should reference window.mermaid")
	}
	if !strings.Contains(html, `class="mermaid"`) {
		t.Error("page should contain the mermaid diagram element")
	}

	// Verify the mermaid diagram content is preserved
	if !strings.Contains(html, "graph TD") {
		t.Error("page should contain the mermaid diagram content")
	}
	if !strings.Contains(html, "Start") {
		t.Error("page should contain mermaid diagram text 'Start'")
	}
}

func TestSiteGeneratorMarkdownAndLLMSOutputs(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "LLMS Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content: `<h1>Chapter 1</h1>
<p>Visit <a href="https://example.com">Example</a>.</p>
<figure>
  <img src="cover.png" alt="Cover art">
  <figcaption>Figure caption</figcaption>
</figure>
<div class="mermaid">graph TD; A-->B;</div>`,
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pageHTML, err := os.ReadFile(filepath.Join(dir, "ch1.html"))
	if err != nil {
		t.Fatalf("read ch1.html failed: %v", err)
	}
	html := string(pageHTML)
	if !strings.Contains(html, `class="sidebar-home-link"`) {
		t.Error("sidebar title should be a home link")
	}
	if !strings.Contains(html, `id="search-status"`) {
		t.Error("search modal should include a result status element")
	}
	if !strings.Contains(html, "mdpress-recent-pages") {
		t.Error("search script should persist recent pages")
	}
	if !strings.Contains(html, "pathMatch") {
		t.Error("search script should rank breadcrumb path matches")
	}
	if !strings.Contains(html, "Ctrl/⌘ K") {
		t.Error("search shortcut label should show Ctrl/⌘ K")
	}
	if !strings.Contains(html, "figcaption") {
		t.Error("template should center figure captions by default")
	}
	if !strings.Contains(html, ".mermaid") {
		t.Error("template should include Mermaid centering styles")
	}

	for _, path := range []string{"ch1.html.md", "index.html.md", "llms.txt", "llms-full.txt"} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("%s should not be generated anymore", path)
		}
	}
}

func TestSiteGeneratorCDNScriptDedupingAndReplay(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  `<div class="mermaid">graph TD; A-->B;</div><div class="math">x</div>`,
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ch1.html"))
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}
	html := string(data)

	if !strings.Contains(html, "existing.dataset.mdpressLoaded === 'true'") {
		t.Error("page should replay CDN callbacks after the script has loaded")
	}
	if !strings.Contains(html, "existing.addEventListener('load', onReady, { once: true });") {
		t.Error("page should queue CDN callbacks while the script is still loading")
	}
	if !strings.Contains(html, "tag.replace(/[A-Z]/g") {
		t.Error("page should normalize camelCase tags to data attributes")
	}
}

// TestFlattenChaptersEmpty tests flattenChapters with an empty input slice.
func TestFlattenChaptersEmpty(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	result := gen.flattenChapters([]SiteChapter{})

	if len(result) != 0 {
		t.Errorf("flattenChapters with empty input should return empty slice, got %d items", len(result))
	}
}

// TestFlattenChaptersSingleChapter tests flattenChapters with a single chapter (no children).
func TestFlattenChaptersSingleChapter(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	input := []SiteChapter{
		{
			Title:    "Chapter 1",
			ID:       "ch1",
			Filename: "ch1.html",
			Content:  "<h1>Chapter 1</h1>",
			Depth:    0,
		},
	}

	result := gen.flattenChapters(input)

	if len(result) != 1 {
		t.Errorf("flattenChapters with single chapter should return 1 item, got %d", len(result))
	}

	if result[0].Title != "Chapter 1" {
		t.Errorf("expected title 'Chapter 1', got %q", result[0].Title)
	}
	if result[0].Filename != "ch1.html" {
		t.Errorf("expected filename 'ch1.html', got %q", result[0].Filename)
	}
	if result[0].Depth != 0 {
		t.Errorf("expected Depth 0, got %d", result[0].Depth)
	}
}

// TestFlattenChaptersNestedWithHeadings tests flattenChapters preserves Depth and Headings in nested chapters (T38 fix).
func TestFlattenChaptersNestedWithHeadings(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	input := []SiteChapter{
		{
			Title:    "Chapter 1",
			ID:       "ch1",
			Filename: "ch1.html",
			Content:  "<h1>Chapter 1</h1>",
			Depth:    0,
			Headings: []SiteNavHeading{
				{
					Title: "Section 1.1",
					ID:    "sec-1.1",
				},
			},
			Children: []SiteChapter{
				{
					Title:    "Chapter 1.1",
					ID:       "ch1.1",
					Filename: "ch1.1.html",
					Content:  "<h1>Chapter 1.1</h1>",
					Depth:    1,
					Headings: []SiteNavHeading{
						{
							Title: "Section 1.1.1",
							ID:    "sec-1.1.1",
						},
					},
				},
				{
					Title:    "Chapter 1.2",
					ID:       "ch1.2",
					Filename: "ch1.2.html",
					Content:  "<h1>Chapter 1.2</h1>",
					Depth:    1,
					Headings: []SiteNavHeading{
						{
							Title: "Section 1.2.1",
							ID:    "sec-1.2.1",
						},
					},
				},
			},
		},
	}

	result := gen.flattenChapters(input)

	if len(result) != 3 {
		t.Errorf("flattenChapters should return 3 items, got %d", len(result))
	}

	// Check parent chapter
	if result[0].Title != "Chapter 1" {
		t.Errorf("item 0: expected title 'Chapter 1', got %q", result[0].Title)
	}
	if result[0].Depth != 0 {
		t.Errorf("item 0: expected Depth 0, got %d", result[0].Depth)
	}
	if len(result[0].Headings) != 1 {
		t.Errorf("item 0: expected 1 heading, got %d", len(result[0].Headings))
	}
	if result[0].Headings[0].Title != "Section 1.1" {
		t.Errorf("item 0: expected heading 'Section 1.1', got %q", result[0].Headings[0].Title)
	}

	// Check child 1
	if result[1].Title != "Chapter 1.1" {
		t.Errorf("item 1: expected title 'Chapter 1.1', got %q", result[1].Title)
	}
	if result[1].Depth != 1 {
		t.Errorf("item 1: expected Depth 1, got %d", result[1].Depth)
	}
	if len(result[1].Headings) != 1 {
		t.Errorf("item 1: expected 1 heading, got %d", len(result[1].Headings))
	}

	// Check child 2
	if result[2].Title != "Chapter 1.2" {
		t.Errorf("item 2: expected title 'Chapter 1.2', got %q", result[2].Title)
	}
	if result[2].Depth != 1 {
		t.Errorf("item 2: expected Depth 1, got %d", result[2].Depth)
	}
}

// TestFlattenChaptersDeeplyNested tests flattenChapters with 3+ levels of nesting.
func TestFlattenChaptersDeeplyNested(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	input := []SiteChapter{
		{
			Title:    "Chapter 1",
			ID:       "ch1",
			Filename: "ch1.html",
			Depth:    0,
			Children: []SiteChapter{
				{
					Title:    "Chapter 1.1",
					ID:       "ch1.1",
					Filename: "ch1.1.html",
					Depth:    1,
					Children: []SiteChapter{
						{
							Title:    "Chapter 1.1.1",
							ID:       "ch1.1.1",
							Filename: "ch1.1.1.html",
							Depth:    2,
							Children: []SiteChapter{
								{
									Title:    "Chapter 1.1.1.1",
									ID:       "ch1.1.1.1",
									Filename: "ch1.1.1.1.html",
									Depth:    3,
								},
							},
						},
					},
				},
			},
		},
	}

	result := gen.flattenChapters(input)

	if len(result) != 4 {
		t.Errorf("flattenChapters should return 4 items for 4-level nesting, got %d", len(result))
	}

	// Verify depth progression
	expectedDepths := []int{0, 1, 2, 3}
	for i, expectedDepth := range expectedDepths {
		if result[i].Depth != expectedDepth {
			t.Errorf("item %d: expected Depth %d, got %d", i, expectedDepth, result[i].Depth)
		}
	}

	// Verify order
	expectedTitles := []string{"Chapter 1", "Chapter 1.1", "Chapter 1.1.1", "Chapter 1.1.1.1"}
	for i, expectedTitle := range expectedTitles {
		if result[i].Title != expectedTitle {
			t.Errorf("item %d: expected title %q, got %q", i, expectedTitle, result[i].Title)
		}
	}
}

// TestExtractDescription tests the extractDescription function with various inputs.
func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Short text (<160 chars)",
			input:    "This is a short description.",
			expected: "This is a short description.",
		},
		{
			name:     "Exactly 160 chars",
			input:    strings.Repeat("a", 160),
			expected: strings.Repeat("a", 160),
		},
		{
			name:     "Text >160 chars with word boundaries",
			input:    "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip.",
			expected: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis…",
		},
		{
			name:     "HTML tags stripped",
			input:    "<p>This is <strong>HTML</strong> content with <em>tags</em>.</p>",
			expected: "This is HTML content with tags .",
		},
		{
			name:     "Multiple whitespace collapsed",
			input:    "This   has    multiple\n\nwhitespace\t\tand\r\nnewlines.",
			expected: "This has multiple whitespace and newlines.",
		},
		{
			name:     "Single very long word >160 chars",
			input:    strings.Repeat("a", 200),
			expected: strings.Repeat("a", 160) + "…",
		},
		{
			name:     "HTML tags with text exceeding 160",
			input:    "<h1>Title</h1><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam.</p>",
			expected: "Title Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription(tt.input)
			if result != tt.expected {
				t.Errorf("extractDescription(%q)\n  got:      %q\n  expected: %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

// TestBuildBreadcrumbs tests the buildBreadcrumbs method with various chapter structures.
func TestBuildBreadcrumbs(t *testing.T) {
	tests := []struct {
		name              string
		chapters          []SiteChapter
		targetFilename    string
		expectedTitles    []string
		expectedFilenames []string
	}{
		{
			name: "Root-level page",
			chapters: []SiteChapter{
				{
					Title:    "Chapter 1",
					ID:       "ch1",
					Filename: "ch1.html",
				},
			},
			targetFilename:    "ch1.html",
			expectedTitles:    []string{"Chapter 1"},
			expectedFilenames: []string{"ch1.html"},
		},
		{
			name: "Nested page - full breadcrumb chain",
			chapters: []SiteChapter{
				{
					Title:    "Part 1",
					ID:       "part1",
					Filename: "part1.html",
					Children: []SiteChapter{
						{
							Title:    "Chapter 1.1",
							ID:       "ch1.1",
							Filename: "ch1.1.html",
							Children: []SiteChapter{
								{
									Title:    "Section 1.1.1",
									ID:       "sec1.1.1",
									Filename: "sec1.1.1.html",
								},
							},
						},
					},
				},
			},
			targetFilename:    "sec1.1.1.html",
			expectedTitles:    []string{"Part 1", "Chapter 1.1", "Section 1.1.1"},
			expectedFilenames: []string{"part1.html", "ch1.1.html", "sec1.1.1.html"},
		},
		{
			name: "Page not found",
			chapters: []SiteChapter{
				{
					Title:    "Chapter 1",
					ID:       "ch1",
					Filename: "ch1.html",
				},
			},
			targetFilename:    "nonexistent.html",
			expectedTitles:    nil,
			expectedFilenames: nil,
		},
		{
			name:              "Empty chapter list",
			chapters:          []SiteChapter{},
			targetFilename:    "any.html",
			expectedTitles:    nil,
			expectedFilenames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewSiteGenerator(SiteMeta{
				Title:    "Test Book",
				Author:   "Author",
				Language: "en-US",
			})
			gen.Chapters = tt.chapters

			result := gen.buildBreadcrumbs(tt.chapters, tt.targetFilename)

			if tt.expectedTitles == nil {
				if result != nil {
					t.Errorf("buildBreadcrumbs expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("buildBreadcrumbs expected non-nil result, got nil")
				return
			}

			if len(result) != len(tt.expectedTitles) {
				t.Errorf("breadcrumb length mismatch: got %d, expected %d", len(result), len(tt.expectedTitles))
				return
			}

			for i, expected := range tt.expectedTitles {
				if result[i].Title != expected {
					t.Errorf("breadcrumb[%d].Title: got %q, expected %q", i, result[i].Title, expected)
				}
				if result[i].Filename != tt.expectedFilenames[i] {
					t.Errorf("breadcrumb[%d].Filename: got %q, expected %q", i, result[i].Filename, tt.expectedFilenames[i])
				}
			}
		})
	}
}

// TestSiteGeneratorSitemapAndSearchIndex verifies that Generate() creates sitemap.xml and search-index.json with correct structure.
func TestSiteGeneratorSitemapAndSearchIndex(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Test Author",
		Language: "en-US",
	})

	// Add multiple chapters for search index
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1><p>This is the content of chapter one.</p>",
	})
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 2",
		ID:       "ch2",
		Filename: "ch2.html",
		Content:  "<h1>Chapter 2</h1><p>This is the content of chapter two.</p>",
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Test sitemap.xml
	sitemapPath := filepath.Join(dir, "sitemap.xml")
	sitemapData, err := os.ReadFile(sitemapPath)
	if err != nil {
		t.Fatalf("sitemap.xml not created: %v", err)
	}

	sitemapContent := string(sitemapData)

	// Verify XML structure
	if !strings.Contains(sitemapContent, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("sitemap.xml should contain XML declaration")
	}
	if !strings.Contains(sitemapContent, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Error("sitemap.xml should contain urlset element with correct namespace")
	}
	if !strings.Contains(sitemapContent, `<loc>index.html</loc>`) {
		t.Error("sitemap.xml should contain index.html URL")
	}
	if !strings.Contains(sitemapContent, `<loc>ch1.html</loc>`) {
		t.Error("sitemap.xml should contain ch1.html URL")
	}
	if !strings.Contains(sitemapContent, `<loc>ch2.html</loc>`) {
		t.Error("sitemap.xml should contain ch2.html URL")
	}
	if !strings.Contains(sitemapContent, `</urlset>`) {
		t.Error("sitemap.xml should close with </urlset>")
	}

	// Test search-index.json
	searchIndexPath := filepath.Join(dir, "search-index.json")
	searchIndexData, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatalf("search-index.json not created: %v", err)
	}

	// Verify it is valid JSON
	var searchEntries []any
	if err := json.Unmarshal(searchIndexData, &searchEntries); err != nil {
		t.Fatalf("search-index.json is not valid JSON: %v", err)
	}

	if len(searchEntries) != 2 {
		t.Errorf("search-index.json should have 2 entries, got %d", len(searchEntries))
	}

	// Verify each entry has expected fields (t, f, x)
	for i, entry := range searchEntries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			t.Errorf("entry %d is not a JSON object", i)
			continue
		}

		if _, hasTitle := entryMap["t"]; !hasTitle {
			t.Errorf("entry %d missing field 't' (title)", i)
		}
		if _, hasFilename := entryMap["f"]; !hasFilename {
			t.Errorf("entry %d missing field 'f' (filename)", i)
		}
		if _, hasText := entryMap["x"]; !hasText {
			t.Errorf("entry %d missing field 'x' (text)", i)
		}
	}

	// Verify content
	searchIndexContent := string(searchIndexData)
	if !strings.Contains(searchIndexContent, "Chapter 1") {
		t.Error("search-index.json should contain Chapter 1 title")
	}
	if !strings.Contains(searchIndexContent, "Chapter 2") {
		t.Error("search-index.json should contain Chapter 2 title")
	}
	if !strings.Contains(searchIndexContent, "ch1.html") {
		t.Error("search-index.json should contain ch1.html filename")
	}
	if !strings.Contains(searchIndexContent, "ch2.html") {
		t.Error("search-index.json should contain ch2.html filename")
	}
}

// TestContentStartsWithTitle verifies the duplicate-title detection logic.
func TestContentStartsWithTitle(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		pageTitle string
		expected  bool
	}{
		{
			name:      "h1 present anywhere — always true regardless of title match",
			html:      "<h1>Different</h1><p>body</p>",
			pageTitle: "My Page",
			expected:  true,
		},
		{
			name:      "leading h2 matches page title",
			html:      "<h2>2.4 守护进程与可用性验收</h2><p>body</p>",
			pageTitle: "2.4 守护进程与可用性验收",
			expected:  true,
		},
		{
			name:      "leading h2 does NOT match page title",
			html:      "<h2>Something Else</h2><p>body</p>",
			pageTitle: "My Page",
			expected:  false,
		},
		{
			name:      "leading h3 matches page title",
			html:      "<h3>Chapter 5</h3><p>text</p>",
			pageTitle: "Chapter 5",
			expected:  true,
		},
		{
			name:      "heading with inner tags matches page title",
			html:      "<h2><a href=\"#\">Section 1</a></h2><p>text</p>",
			pageTitle: "Section 1",
			expected:  true,
		},
		{
			name:      "leading whitespace before heading",
			html:      "  \n <h2>My Title</h2><p>text</p>",
			pageTitle: "My Title",
			expected:  true,
		},
		{
			name:      "no heading at all",
			html:      "<p>Just a paragraph</p>",
			pageTitle: "My Page",
			expected:  false,
		},
		{
			name:      "empty content",
			html:      "",
			pageTitle: "Title",
			expected:  false,
		},
		{
			name:      "h2 NOT at start — should not match",
			html:      "<p>Intro</p><h2>My Page</h2>",
			pageTitle: "My Page",
			expected:  false,
		},
		{
			name:      "heading with attributes matches",
			html:      "<h2 id=\"sec\" class=\"chapter\">Overview</h2>",
			pageTitle: "Overview",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contentStartsWithTitle(tt.html, tt.pageTitle)
			if result != tt.expected {
				t.Errorf("contentStartsWithTitle(%q, %q) = %v, want %v",
					tt.html, tt.pageTitle, result, tt.expected)
			}
		})
	}
}

// TestExtractDescriptionEdgeCases verifies the extractDescription function handles various edge cases.
func TestExtractDescriptionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "HTML with HTML entities",
			html:     "Text with &amp; and &lt; entities",
			expected: "Text with &amp; and &lt; entities",
		},
		{
			name:     "Empty string",
			html:     "",
			expected: "",
		},
		{
			name:     "String with only whitespace",
			html:     "   \n\t  \n  ",
			expected: "",
		},
		{
			name:     "Simple HTML with tags",
			html:     "<p>Hello <b>world</b></p>",
			expected: "Hello world",
		},
		{
			name:     "HTML with multiple consecutive spaces",
			html:     "<p>Text    with     spaces</p>",
			expected: "Text with spaces",
		},
		{
			name:     "Text under 160 characters",
			html:     "<p>Short text</p>",
			expected: "Short text",
		},
		{
			name:     "Text exactly at boundary (160 chars)",
			html:     "<p>" + strings.Repeat("word ", 32) + "</p>",
			expected: strings.TrimSpace(strings.Repeat("word ", 32)),
		},
		{
			name: "Long text needing truncation",
			html: "<p>" + strings.Repeat("word ", 50) + "</p>",
			// Will be truncated to around 160 chars with ellipsis
			expected: "", // Will be checked by length and ellipsis below
		},
		{
			name:     "Very long Chinese text with truncation",
			html:     "<p>这是一个很长的中文文本。" + strings.Repeat("这是测试文本。", 30) + "</p>",
			expected: "", // Will be checked by length and ellipsis below
		},
		{
			name:     "HTML with mixed entities and tags",
			html:     "<div>Test &amp; more &lt;stuff&gt;</div>",
			expected: "Test &amp; more &lt;stuff&gt;",
		},
		{
			name:     "Multiple tags and entities combined",
			html:     "<p>Some <span>text</span> with &nbsp; and <b>bold</b> &amp; more</p>",
			expected: "Some text with &nbsp; and bold &amp; more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription(tt.html)

			// For long text tests, verify truncation and ellipsis instead of exact match.
			// extractDescription truncates by rune count (~160 characters) to avoid
			// splitting multi-byte UTF-8 characters.
			if tt.name == "Long text needing truncation" || tt.name == "Very long Chinese text with truncation" {
				if len(result) > 500 {
					t.Errorf("extractDescription should truncate long text: got length %d", len(result))
				}
				if !strings.HasSuffix(result, "…") {
					t.Errorf("extractDescription should end with ellipsis for truncated text")
				}
				return
			}

			if result != tt.expected {
				t.Errorf("extractDescription(%q) = %q, want %q", tt.html, result, tt.expected)
			}
		})
	}
}
