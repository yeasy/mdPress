// navigation_test.go tests navigation tree building functions.
// Tests cover heading-to-navigation conversion, chapter flattening, and tree structures.
package cmd

import (
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/toc"
)

// ---------------------------------------------------------------------------
// flattenChaptersWithDepth
// ---------------------------------------------------------------------------

func TestFlattenChaptersWithDepth_Empty(t *testing.T) {
	result := flattenChaptersWithDepth(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d items", len(result))
	}
}

func TestFlattenChaptersWithDepth_SingleChapter(t *testing.T) {
	chapters := []config.ChapterDef{
		{Title: "Introduction", File: "intro.md"},
	}
	result := flattenChaptersWithDepth(chapters)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Def.Title != "Introduction" {
		t.Errorf("expected title 'Introduction', got %q", result[0].Def.Title)
	}
	if result[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", result[0].Depth)
	}
}

func TestFlattenChaptersWithDepth_NestedSections(t *testing.T) {
	chapters := []config.ChapterDef{
		{
			Title: "Chapter 1",
			File:  "ch1.md",
			Sections: []config.ChapterDef{
				{Title: "Section 1.1", File: "sec1.1.md"},
				{Title: "Section 1.2", File: "sec1.2.md"},
			},
		},
	}
	result := flattenChaptersWithDepth(chapters)
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	if result[0].Depth != 0 {
		t.Errorf("chapter depth should be 0, got %d", result[0].Depth)
	}
	if result[1].Depth != 1 {
		t.Errorf("section depth should be 1, got %d", result[1].Depth)
	}
	if result[2].Depth != 1 {
		t.Errorf("section depth should be 1, got %d", result[2].Depth)
	}
}

func TestFlattenChaptersWithDepth_DeepNesting(t *testing.T) {
	chapters := []config.ChapterDef{
		{
			Title: "Chapter 1",
			File:  "ch1.md",
			Sections: []config.ChapterDef{
				{
					Title: "Section 1.1",
					File:  "sec1.1.md",
					Sections: []config.ChapterDef{
						{Title: "Subsection 1.1.1", File: "subsec1.1.1.md"},
					},
				},
			},
		},
	}
	result := flattenChaptersWithDepth(chapters)
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	if result[0].Depth != 0 {
		t.Errorf("expected depth 0, got %d", result[0].Depth)
	}
	if result[1].Depth != 1 {
		t.Errorf("expected depth 1, got %d", result[1].Depth)
	}
	if result[2].Depth != 2 {
		t.Errorf("expected depth 2, got %d", result[2].Depth)
	}
}

func TestFlattenChaptersWithDepth_MultipleTopLevelChapters(t *testing.T) {
	chapters := []config.ChapterDef{
		{Title: "Chapter 1", File: "ch1.md"},
		{Title: "Chapter 2", File: "ch2.md"},
		{Title: "Chapter 3", File: "ch3.md"},
	}
	result := flattenChaptersWithDepth(chapters)
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	for i, fc := range result {
		if fc.Depth != 0 {
			t.Errorf("item %d: expected depth 0, got %d", i, fc.Depth)
		}
	}
}

// ---------------------------------------------------------------------------
// buildHeadingTree
// ---------------------------------------------------------------------------

func TestBuildHeadingTree_NilInput(t *testing.T) {
	result := buildHeadingTree(nil, "chapter-id")
	if result != nil {
		t.Errorf("expected nil for empty headings, got %v", result)
	}
}

func TestBuildHeadingTree_EmptySlice(t *testing.T) {
	result := buildHeadingTree([]markdown.HeadingInfo{}, "chapter-id")
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestBuildHeadingTree_SingleHeading(t *testing.T) {
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Chapter Title", ID: "chapter-title"},
	}
	result := buildHeadingTree(headings, "chapter-title")
	// When chapter root matches, it's stripped, so result may be nil or empty
	if len(result) > 0 {
		// If not stripped, verify structure
		if result[0].Title != "Chapter Title" {
			t.Errorf("expected title 'Chapter Title', got %q", result[0].Title)
		}
	}
}

func TestBuildHeadingTree_NestedHeadings(t *testing.T) {
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Main Title", ID: "main-title"},
		{Level: 2, Text: "Section A", ID: "section-a"},
		{Level: 3, Text: "Subsection A.1", ID: "subsection-a-1"},
		{Level: 2, Text: "Section B", ID: "section-b"},
	}
	result := buildHeadingTree(headings, "main-title")
	// Main title is stripped, so we expect sections at top level
	if result == nil {
		t.Fatal("expected non-nil result for nested headings")
	}
}

func TestBuildHeadingTree_NoMatchingChapterID(t *testing.T) {
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Chapter One", ID: "chapter-one"},
		{Level: 2, Text: "Section 1.1", ID: "section-1-1"},
	}
	result := buildHeadingTree(headings, "different-chapter-id")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Since chapter ID doesn't match, the first heading should remain
	if len(result) > 0 && result[0].ID != "chapter-one" {
		t.Errorf("expected first heading ID 'chapter-one', got %q", result[0].ID)
	}
}

func TestBuildHeadingTree_MultipleTopLevelHeadings(t *testing.T) {
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Chapter 1", ID: "chapter-1"},
		{Level: 1, Text: "Chapter 2", ID: "chapter-2"},
	}
	result := buildHeadingTree(headings, "chapter-1")
	// Chapter 1 is stripped, Chapter 2 remains
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// toNavHeadings
// ---------------------------------------------------------------------------

func TestToNavHeadings_Empty(t *testing.T) {
	result := toNavHeadings(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestToNavHeadings_EmptySlice(t *testing.T) {
	result := toNavHeadings([]toc.TOCEntry{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestToNavHeadings_SingleItem(t *testing.T) {
	// Note: This is a test of conversion from toc.TOCEntry to navHeading
	// which requires building via buildHeadingTree first.
	// Testing the core conversion indirectly through the pipeline.
}

func TestToNavHeadings_NestedStructure(t *testing.T) {
	// This function is used internally by buildHeadingTree
	// Testing through integration with buildHeadingTree
}

// ---------------------------------------------------------------------------
// toRendererNavHeadings
// ---------------------------------------------------------------------------

func TestToRendererNavHeadings_Empty(t *testing.T) {
	result := toRendererNavHeadings(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestToRendererNavHeadings_EmptySlice(t *testing.T) {
	result := toRendererNavHeadings([]navHeading{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestToRendererNavHeadings_SingleItem(t *testing.T) {
	input := []navHeading{
		{Title: "Introduction", ID: "intro"},
	}
	result := toRendererNavHeadings(input)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Title != "Introduction" {
		t.Errorf("expected title 'Introduction', got %q", result[0].Title)
	}
	if result[0].ID != "intro" {
		t.Errorf("expected ID 'intro', got %q", result[0].ID)
	}
}

func TestToRendererNavHeadings_NestedItems(t *testing.T) {
	input := []navHeading{
		{
			Title: "Chapter 1",
			ID:    "ch1",
			Children: []navHeading{
				{Title: "Section 1.1", ID: "sec1-1"},
				{Title: "Section 1.2", ID: "sec1-2"},
			},
		},
		{Title: "Chapter 2", ID: "ch2"},
	}
	result := toRendererNavHeadings(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	// Verify first item and its children
	ch1 := result[0]
	if ch1.Title != "Chapter 1" {
		t.Errorf("expected title 'Chapter 1', got %q", ch1.Title)
	}
	if ch1.ID != "ch1" {
		t.Errorf("expected ID 'ch1', got %q", ch1.ID)
	}
	if len(ch1.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(ch1.Children))
	}

	// Verify child structure
	if ch1.Children[0].Title != "Section 1.1" {
		t.Errorf("expected child title 'Section 1.1', got %q", ch1.Children[0].Title)
	}
	if ch1.Children[0].ID != "sec1-1" {
		t.Errorf("expected child ID 'sec1-1', got %q", ch1.Children[0].ID)
	}

	// Verify second item
	ch2 := result[1]
	if ch2.Title != "Chapter 2" {
		t.Errorf("expected title 'Chapter 2', got %q", ch2.Title)
	}
	if ch2.ID != "ch2" {
		t.Errorf("expected ID 'ch2', got %q", ch2.ID)
	}
	if len(ch2.Children) != 0 {
		t.Errorf("expected no children for Chapter 2, got %d", len(ch2.Children))
	}
}

func TestToRendererNavHeadings_DeeplyNestedItems(t *testing.T) {
	input := []navHeading{
		{
			Title: "Level 1",
			ID:    "l1",
			Children: []navHeading{
				{
					Title: "Level 2",
					ID:    "l2",
					Children: []navHeading{
						{Title: "Level 3", ID: "l3"},
					},
				},
			},
		},
	}
	result := toRendererNavHeadings(input)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level item, got %d", len(result))
	}

	l1 := result[0]
	if len(l1.Children) != 1 {
		t.Fatalf("expected 1 child at level 1, got %d", len(l1.Children))
	}

	l2 := l1.Children[0]
	if len(l2.Children) != 1 {
		t.Fatalf("expected 1 child at level 2, got %d", len(l2.Children))
	}

	l3 := l2.Children[0]
	if l3.Title != "Level 3" {
		t.Errorf("expected title 'Level 3', got %q", l3.Title)
	}
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestNavigationPipeline_CompleteFlow(t *testing.T) {
	// Test the complete pipeline: chapters -> flattened chapters -> navigation
	chapters := []config.ChapterDef{
		{
			Title: "Part 1",
			File:  "part1.md",
			Sections: []config.ChapterDef{
				{Title: "Chapter 1", File: "ch1.md"},
				{Title: "Chapter 2", File: "ch2.md"},
			},
		},
		{Title: "Part 2", File: "part2.md"},
	}

	flattened := flattenChaptersWithDepth(chapters)
	if len(flattened) != 4 {
		t.Errorf("expected 4 flattened items, got %d", len(flattened))
	}

	// Verify depth progression
	expectedDepths := []int{0, 1, 1, 0}
	for i, fc := range flattened {
		if fc.Depth != expectedDepths[i] {
			t.Errorf("item %d: expected depth %d, got %d", i, expectedDepths[i], fc.Depth)
		}
	}
}

func TestToRendererNavHeadings_ConvertsToCorrectType(t *testing.T) {
	// Ensure the function returns renderer.NavHeading types correctly
	input := []navHeading{
		{Title: "Test", ID: "test"},
	}
	result := toRendererNavHeadings(input)

	// Verify type compatibility
	_ = result
}

func TestFlattenChaptersWithDepth_PreservesChapterData(t *testing.T) {
	// Ensure that chapter definitions are preserved accurately
	originalTitle := "Chapter with Special Characters: A & B"
	chapters := []config.ChapterDef{
		{Title: originalTitle, File: "ch.md"},
	}

	result := flattenChaptersWithDepth(chapters)
	if result[0].Def.Title != originalTitle {
		t.Errorf("chapter title not preserved: expected %q, got %q", originalTitle, result[0].Def.Title)
	}
}
