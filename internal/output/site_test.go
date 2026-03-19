package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSiteGeneratorIndexRedirect verifies that index.html is created and redirects to the first page.
func TestSiteGeneratorIndexRedirect(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Add a chapter with explicit filename
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1>",
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify index.html exists
	indexPath := filepath.Join(dir, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.html not created: %v", err)
	}

	indexHTML := string(data)

	// Verify it's a redirect to the first page
	if !strings.Contains(indexHTML, "<!DOCTYPE html>") {
		t.Error("index.html should be valid HTML")
	}
	if !strings.Contains(indexHTML, "meta http-equiv=\"refresh\"") {
		t.Error("index.html should contain a refresh meta tag")
	}
	if !strings.Contains(indexHTML, "ch1.html") {
		t.Error("index.html should redirect to ch1.html")
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
	if !strings.Contains(ch1HTML, `href="ch2.html"`) {
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

	if !strings.Contains(ch2HTML, `href="ch1.html"`) {
		t.Error("middle page should have a previous link to ch1.html")
	}
	if !strings.Contains(ch2HTML, "Chapter 1") {
		t.Error("middle page should show previous page title (Chapter 1)")
	}
	if !strings.Contains(ch2HTML, `href="ch3.html"`) {
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

	if !strings.Contains(ch3HTML, `href="ch2.html"`) {
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

	if !strings.Contains(page0HTML, `href="page_1.html"`) {
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

	// Verify heading links are in the sidebar
	if !strings.Contains(html, `href="ch1.html#sec-1"`) {
		t.Error("sidebar should contain link to Section 1 (sec-1)")
	}
	if !strings.Contains(html, "Section 1") {
		t.Error("sidebar should display 'Section 1' text")
	}

	if !strings.Contains(html, `href="ch1.html#subsec-1"`) {
		t.Error("sidebar should contain link to Subsection 1 (subsec-1)")
	}
	if !strings.Contains(html, "Subsection 1") {
		t.Error("sidebar should display 'Subsection 1' text")
	}

	if !strings.Contains(html, `href="ch1.html#sec-2"`) {
		t.Error("sidebar should contain link to Section 2 (sec-2)")
	}
	if !strings.Contains(html, "Section 2") {
		t.Error("sidebar should display 'Section 2' text")
	}

	// Verify nested headings use nav-heading-depth classes
	if !strings.Contains(html, "nav-heading") {
		t.Error("sidebar should contain nav-heading elements")
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
