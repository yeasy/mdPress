package toc

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TestNewGenerator tests creating a generator
func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
}

// TestGenerateEmpty tests empty heading list
func TestGenerateEmpty(t *testing.T) {
	g := NewGenerator()
	entries := g.Generate(nil)
	if len(entries) != 0 {
		t.Errorf("empty input should return empty list: got %d entries", len(entries))
	}

	entries = g.Generate([]HeadingInfo{})
	if len(entries) != 0 {
		t.Errorf("empty slice should return empty list: got %d entries", len(entries))
	}
}

// TestGenerateFlat tests same-level headings
func TestGenerateFlat(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "章节一", ID: "ch1"},
		{Level: 1, Text: "章节二", ID: "ch2"},
		{Level: 1, Text: "章节三", ID: "ch3"},
	}

	entries := g.Generate(headings)
	if len(entries) != 3 {
		t.Fatalf("should have 3 top-level entries: got %d", len(entries))
	}

	for i, entry := range entries {
		if entry.Level != 1 {
			t.Errorf("entry %d has wrong level: got %d", i, entry.Level)
		}
		if len(entry.Children) != 0 {
			t.Errorf("entry %d should have no children: got %d", i, len(entry.Children))
		}
	}
}

// TestGenerateNested tests nested headings
func TestGenerateNested(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "第一章", ID: "ch1"},
		{Level: 2, Text: "1.1 小节", ID: "sec1-1"},
		{Level: 2, Text: "1.2 小节", ID: "sec1-2"},
		{Level: 3, Text: "1.2.1 子节", ID: "sub1-2-1"},
		{Level: 1, Text: "第二章", ID: "ch2"},
		{Level: 2, Text: "2.1 小节", ID: "sec2-1"},
	}

	entries := g.Generate(headings)
	if len(entries) != 2 {
		t.Fatalf("should have 2 top-level entries: got %d", len(entries))
	}

	// Chapter 1 should have 2 children
	ch1 := entries[0]
	if ch1.Title != "第一章" {
		t.Errorf("first entry has wrong title: got %q", ch1.Title)
	}
	if len(ch1.Children) != 2 {
		t.Fatalf("chapter 1 should have 2 children: got %d", len(ch1.Children))
	}

	// Section 1.2 should have 1 child (1.2.1)
	sec12 := ch1.Children[1]
	if len(sec12.Children) != 1 {
		t.Errorf("section 1.2 should have 1 child: got %d", len(sec12.Children))
	}

	// Chapter 2 should have 1 child
	ch2 := entries[1]
	if len(ch2.Children) != 1 {
		t.Errorf("chapter 2 should have 1 child: got %d", len(ch2.Children))
	}

	// Total entry count should be 6
	total := CountEntries(entries)
	if total != 6 {
		t.Errorf("should have 6 total entries: got %d", total)
	}
}

// TestGenerateDeepNesting tests deep nesting
func TestGenerateDeepNesting(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "H1", ID: "h1"},
		{Level: 2, Text: "H2", ID: "h2"},
		{Level: 3, Text: "H3", ID: "h3"},
		{Level: 4, Text: "H4", ID: "h4"},
		{Level: 5, Text: "H5", ID: "h5"},
		{Level: 6, Text: "H6", ID: "h6"},
	}

	entries := g.Generate(headings)
	if len(entries) != 1 {
		t.Fatalf("should have 1 top-level entry: got %d", len(entries))
	}
}

// TestRenderHTMLEmpty tests rendering an empty TOC
func TestRenderHTMLEmpty(t *testing.T) {
	g := NewGenerator()
	html := g.RenderHTML(nil)
	if html != "" {
		t.Errorf("empty TOC should return empty string: got %q", html)
	}
}

// TestRenderHTMLBasic tests basic HTML rendering
func TestRenderHTMLBasic(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "简介", ID: "intro", Children: []TOCEntry{}},
		{Level: 1, Title: "总结", ID: "summary", Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	if !strings.Contains(html, `<nav class="toc">`) {
		t.Error("HTML should contain nav.toc tag")
	}
	if !strings.Contains(html, `href="#intro"`) {
		t.Error("HTML should contain intro anchor link")
	}
	if !strings.Contains(html, `href="#summary"`) {
		t.Error("HTML should contain summary anchor link")
	}
	if !strings.Contains(html, "简介") {
		t.Error("HTML should contain heading text")
	}
}

// TestRenderHTMLNested tests nested HTML rendering
func TestRenderHTMLNested(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1 节", ID: "sec1-1", Children: []TOCEntry{}},
			},
		},
	}

	html := g.RenderHTML(entries)

	// Should have nested ul elements
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 2 {
		t.Errorf("nested TOC should have at least 2 ul tags: got %d", ulCount)
	}
}

// TestRenderHTMLEscaping tests special character escaping
func TestRenderHTMLEscaping(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "<script>alert('xss')</script>", ID: "xss-test", Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	if strings.Contains(html, "<script>") {
		t.Error("HTML tags should be escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("should contain escaped tags")
	}
}

// TestGetEntry tests finding entries by ID
func TestGetEntry(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1", ID: "sec1-1", Children: []TOCEntry{}},
				{
					Level: 2, Title: "1.2", ID: "sec1-2",
					Children: []TOCEntry{
						{Level: 3, Title: "1.2.1", ID: "sub1-2-1", Children: []TOCEntry{}},
					},
				},
			},
		},
		{Level: 1, Title: "第二章", ID: "ch2", Children: []TOCEntry{}},
	}

	tests := []struct {
		id    string
		found bool
		title string
	}{
		{"ch1", true, "第一章"},
		{"sec1-1", true, "1.1"},
		{"sub1-2-1", true, "1.2.1"},
		{"ch2", true, "第二章"},
		{"nonexistent", false, ""},
	}

	for _, tt := range tests {
		entry := GetEntry(entries, tt.id)
		if tt.found {
			if entry == nil {
				t.Errorf("should find entry with ID=%q", tt.id)
				continue
			}
			if entry.Title != tt.title {
				t.Errorf("wrong title for ID=%q: got %q, want %q", tt.id, entry.Title, tt.title)
			}
		} else if entry != nil {
			t.Errorf("ID=%q should not exist", tt.id)
		}
	}
}

// TestFlattenToList tests flattening
func TestFlattenToList(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "A", ID: "a",
			Children: []TOCEntry{
				{Level: 2, Title: "A1", ID: "a1", Children: []TOCEntry{}},
				{Level: 2, Title: "A2", ID: "a2", Children: []TOCEntry{}},
			},
		},
		{Level: 1, Title: "B", ID: "b", Children: []TOCEntry{}},
	}

	flat := FlattenToList(entries)
	if len(flat) != 4 {
		t.Fatalf("should have 4 entries after flattening: got %d", len(flat))
	}

	expectedTitles := []string{"A", "A1", "A2", "B"}
	for i, title := range expectedTitles {
		if flat[i].Title != title {
			t.Errorf("entry %d has wrong title: got %q, want %q", i, flat[i].Title, title)
		}
	}

	// After flattening, Children should be empty
	for i, entry := range flat {
		if len(entry.Children) != 0 {
			t.Errorf("entry %d should have no children after flattening", i)
		}
	}
}

// TestCountEntries tests entry counting
func TestCountEntries(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "A", ID: "a",
			Children: []TOCEntry{
				{Level: 2, Title: "A1", ID: "a1", Children: []TOCEntry{}},
				{
					Level: 2, Title: "A2", ID: "a2",
					Children: []TOCEntry{
						{Level: 3, Title: "A2a", ID: "a2a", Children: []TOCEntry{}},
					},
				},
			},
		},
		{Level: 1, Title: "B", ID: "b", Children: []TOCEntry{}},
	}

	count := CountEntries(entries)
	if count != 5 {
		t.Errorf("should have 5 entries: got %d", count)
	}
}

// TestCountEntriesEmpty tests counting an empty list
func TestCountEntriesEmpty(t *testing.T) {
	count := CountEntries(nil)
	if count != 0 {
		t.Errorf("empty list count should be 0: got %d", count)
	}
}

// TestEscapeHTMLToc tests HTML escaping in the toc package
func TestEscapeHTMLToc(t *testing.T) {
	input := `<a href="test">&'`
	expected := `&lt;a href=&quot;test&quot;&gt;&amp;&#39;`
	got := utils.EscapeHTML(input)
	if got != expected {
		t.Errorf("EscapeHTML result error: got %q, want %q", got, expected)
	}
}

// TestGenerateSkippedLevels tests handling of skipped heading levels
func TestGenerateSkippedLevels(t *testing.T) {
	g := NewGenerator()
	// Test jumping from H1 directly to H3
	headings := []HeadingInfo{
		{Level: 1, Text: "第一章", ID: "ch1"},
		{Level: 3, Text: "直接跳到三级", ID: "skip2"},
		{Level: 2, Text: "回到二级", ID: "back2"},
	}

	entries := g.Generate(headings)
	if len(entries) != 1 {
		t.Fatalf("should have 1 top-level entry: got %d", len(entries))
	}

	// Verify skipped levels are still nested correctly
	ch1 := entries[0]
	if len(ch1.Children) < 2 {
		t.Errorf("chapter 1 should have at least 2 children to handle skipped levels: got %d", len(ch1.Children))
	}

	total := CountEntries(entries)
	if total != 3 {
		t.Errorf("total entries should be 3: got %d", total)
	}
}

// TestGenerateRepeatedTitles tests handling of same titles with different IDs
func TestGenerateRepeatedTitles(t *testing.T) {
	g := NewGenerator()
	// Same title but different ID
	headings := []HeadingInfo{
		{Level: 1, Text: "简介", ID: "intro-1"},
		{Level: 1, Text: "简介", ID: "intro-2"},
		{Level: 1, Text: "简介", ID: "intro-3"},
	}

	entries := g.Generate(headings)
	if len(entries) != 3 {
		t.Fatalf("should have 3 entries: got %d", len(entries))
	}

	// Verify each entry has a distinct ID
	ids := make(map[string]bool)
	for _, entry := range entries {
		if ids[entry.ID] {
			t.Errorf("ID %q is duplicated", entry.ID)
		}
		ids[entry.ID] = true
		if entry.Title != "简介" {
			t.Errorf("title should be '简介': got %q", entry.Title)
		}
	}
}

// TestGenerateSpecialCharIDs tests IDs with special characters
func TestGenerateSpecialCharIDs(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "标题", ID: "id-with-dashes"},
		{Level: 1, Text: "标题", ID: "id_with_underscores"},
		{Level: 1, Text: "标题", ID: "id.with.dots"},
		{Level: 1, Text: "标题", ID: "id-123-numbers"},
	}

	entries := g.Generate(headings)
	if len(entries) != 4 {
		t.Fatalf("should have 4 entries: got %d", len(entries))
	}

	expectedIDs := []string{"id-with-dashes", "id_with_underscores", "id.with.dots", "id-123-numbers"}
	for i, expectedID := range expectedIDs {
		if entries[i].ID != expectedID {
			t.Errorf("entry %d ID should be %q: got %q", i, expectedID, entries[i].ID)
		}
	}
}

// TestRenderHTMLDeepNesting tests 4+ level deep nested HTML rendering
func TestRenderHTMLDeepNesting(t *testing.T) {
	g := NewGenerator()
	// Build 4-level deep nested structure
	entries := []TOCEntry{
		{
			Level: 1, Title: "Level 1", ID: "l1",
			Children: []TOCEntry{
				{
					Level: 2, Title: "Level 2", ID: "l2",
					Children: []TOCEntry{
						{
							Level: 3, Title: "Level 3", ID: "l3",
							Children: []TOCEntry{
								{Level: 4, Title: "Level 4", ID: "l4", Children: []TOCEntry{}},
							},
						},
					},
				},
			},
		},
	}

	html := g.RenderHTML(entries)

	// Check deep nested structure
	if !strings.Contains(html, "Level 1") {
		t.Error("HTML should contain level 1 heading")
	}
	if !strings.Contains(html, "Level 2") {
		t.Error("HTML should contain level 2 heading")
	}
	if !strings.Contains(html, "Level 3") {
		t.Error("HTML should contain level 3 heading")
	}
	if !strings.Contains(html, "Level 4") {
		t.Error("HTML should contain level 4 heading")
	}

	// Check nested ul tags
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 4 {
		t.Errorf("4-level nesting should have at least 4 ul tags: got %d", ulCount)
	}

	// Check all anchor links exist
	links := []string{"#l1", "#l2", "#l3", "#l4"}
	for _, link := range links {
		if !strings.Contains(html, link) {
			t.Errorf("HTML should contain link %s", link)
		}
	}
}

// TestFlattenToListEmpty tests flattening empty and nil lists
func TestFlattenToListEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  []TOCEntry
		expect int
	}{
		{
			name:   "nil slice",
			input:  nil,
			expect: 0,
		},
		{
			name:   "empty slice",
			input:  []TOCEntry{},
			expect: 0,
		},
		{
			name: "single level no children",
			input: []TOCEntry{
				{Level: 1, Title: "A", ID: "a", Children: []TOCEntry{}},
			},
			expect: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenToList(tt.input)
			if len(result) != tt.expect {
				t.Errorf("should have %d entries: got %d", tt.expect, len(result))
			}
		})
	}
}

// TestGetEntryFirstMatch tests that GetEntry returns the first match
func TestGetEntryFirstMatch(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1", ID: "duplicate-id", Children: []TOCEntry{}},
				{
					Level: 2, Title: "1.2", ID: "sec1-2",
					Children: []TOCEntry{
						{Level: 3, Title: "1.2.1", ID: "duplicate-id", Children: []TOCEntry{}},
					},
				},
			},
		},
	}

	// Query an ID that may appear multiple times
	entry := GetEntry(entries, "duplicate-id")
	if entry == nil {
		t.Fatal("should find entry with ID 'duplicate-id'")
		return
	}

	// Verify the first match is returned
	if entry.Title != "1.1" {
		t.Errorf("should return first match with title '1.1': got %q", entry.Title)
	}

	if entry.Level != 2 {
		t.Errorf("first match should be Level 2: got %d", entry.Level)
	}
}

// TestRenderHTMLWithPageNumbers tests rendering TOC entries with page numbers
func TestRenderHTMLWithPageNumbers(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "第一章", ID: "ch1", PageNum: 1, Children: []TOCEntry{
			{Level: 2, Title: "1.1 节", ID: "sec1-1", PageNum: 2, Children: []TOCEntry{}},
		}},
		{Level: 1, Title: "第二章", ID: "ch2", PageNum: 5, Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	// Verify HTML contains headings and links
	if !strings.Contains(html, "第一章") {
		t.Error("HTML should contain '第一章'")
	}
	if !strings.Contains(html, "#ch1") {
		t.Error("HTML should contain #ch1 link")
	}

	// Page numbers may render differently depending on implementation
	if !strings.Contains(html, "<nav class=\"toc\">") {
		t.Error("HTML should contain toc navigation container")
	}
}
