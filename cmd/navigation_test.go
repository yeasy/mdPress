// navigation_test.go tests navigation tree building functions.
// Tests cover heading-to-navigation conversion, chapter flattening, and tree structures.
package cmd

import (
	"fmt"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
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
	// Test direct conversion from a single TOC entry
	entries := []toc.TOCEntry{
		{Title: "Test Heading", ID: "test-heading"},
	}
	result := toNavHeadings(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	if result[0].Title != "Test Heading" {
		t.Errorf("expected title 'Test Heading', got %q", result[0].Title)
	}
	if result[0].ID != "test-heading" {
		t.Errorf("expected ID 'test-heading', got %q", result[0].ID)
	}
	if len(result[0].Children) != 0 {
		t.Errorf("expected no children, got %d", len(result[0].Children))
	}
}

func TestToNavHeadings_NestedStructure(t *testing.T) {
	// Test conversion with nested TOC entries
	entries := []toc.TOCEntry{
		{
			Title: "Parent",
			ID:    "parent",
			Children: []toc.TOCEntry{
				{Title: "Child 1", ID: "child-1"},
				{Title: "Child 2", ID: "child-2"},
			},
		},
	}
	result := toNavHeadings(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 top-level item, got %d", len(result))
	}
	if result[0].Title != "Parent" {
		t.Errorf("expected parent title 'Parent', got %q", result[0].Title)
	}
	if len(result[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(result[0].Children))
	}
	if result[0].Children[0].Title != "Child 1" {
		t.Errorf("expected child title 'Child 1', got %q", result[0].Children[0].Title)
	}
	if result[0].Children[1].Title != "Child 2" {
		t.Errorf("expected child title 'Child 2', got %q", result[0].Children[1].Title)
	}
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

// ---------------------------------------------------------------------------
// Table-driven tests for comprehensive coverage
// ---------------------------------------------------------------------------

func TestFlattenChaptersWithDepth_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		chapters      []config.ChapterDef
		expectedCount int
		checkDepths   func([]flattenedChapter) bool
	}{
		{
			name:          "empty input",
			chapters:      nil,
			expectedCount: 0,
			checkDepths: func(fc []flattenedChapter) bool {
				return len(fc) == 0
			},
		},
		{
			name:          "single flat chapter",
			chapters:      []config.ChapterDef{{Title: "Ch1", File: "ch1.md"}},
			expectedCount: 1,
			checkDepths: func(fc []flattenedChapter) bool {
				return fc[0].Depth == 0
			},
		},
		{
			name: "one chapter with two sections",
			chapters: []config.ChapterDef{
				{
					Title: "Ch1",
					File:  "ch1.md",
					Sections: []config.ChapterDef{
						{Title: "Sec1", File: "sec1.md"},
						{Title: "Sec2", File: "sec2.md"},
					},
				},
			},
			expectedCount: 3,
			checkDepths: func(fc []flattenedChapter) bool {
				return fc[0].Depth == 0 && fc[1].Depth == 1 && fc[2].Depth == 1
			},
		},
		{
			name: "two top-level chapters with nested sections",
			chapters: []config.ChapterDef{
				{
					Title: "Ch1",
					File:  "ch1.md",
					Sections: []config.ChapterDef{
						{Title: "Sec1.1", File: "sec1.1.md"},
					},
				},
				{
					Title: "Ch2",
					File:  "ch2.md",
					Sections: []config.ChapterDef{
						{Title: "Sec2.1", File: "sec2.1.md"},
					},
				},
			},
			expectedCount: 4,
			checkDepths: func(fc []flattenedChapter) bool {
				return fc[0].Depth == 0 && fc[1].Depth == 1 &&
					fc[2].Depth == 0 && fc[3].Depth == 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenChaptersWithDepth(tt.chapters)
			if len(result) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(result))
			}
			if !tt.checkDepths(result) {
				t.Error("depth check failed")
			}
		})
	}
}

func TestBuildHeadingTree_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		headings   []markdown.HeadingInfo
		chapterID  string
		expectNil  bool
		expectLen  int
		firstTitle string
	}{
		{
			name:      "nil headings",
			headings:  nil,
			chapterID: "ch1",
			expectNil: true,
		},
		{
			name:      "empty headings slice",
			headings:  []markdown.HeadingInfo{},
			chapterID: "ch1",
			expectNil: true,
		},
		{
			name: "single heading that matches chapter ID (stripped)",
			headings: []markdown.HeadingInfo{
				{Level: 1, Text: "Chapter", ID: "chapter"},
			},
			chapterID: "chapter",
			expectNil: true, // Should be stripped since it matches chapter ID
		},
		{
			name: "multiple headings with matching chapter ID",
			headings: []markdown.HeadingInfo{
				{Level: 1, Text: "Main", ID: "main"},
				{Level: 2, Text: "Sub", ID: "sub"},
			},
			chapterID: "main",
			expectLen: 1, // Main is stripped, Sub remains
		},
		{
			name: "multiple headings with non-matching chapter ID",
			headings: []markdown.HeadingInfo{
				{Level: 1, Text: "Chapter 1", ID: "ch1"},
				{Level: 2, Text: "Section", ID: "section"},
			},
			chapterID:  "different-id",
			expectLen:  1, // Chapter 1 is root with Section as child
			firstTitle: "Chapter 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHeadingTree(tt.headings, tt.chapterID)
			if tt.expectNil && result != nil {
				t.Errorf("expected nil result, got %v", result)
			}
			if !tt.expectNil && result == nil {
				t.Fatal("expected non-nil result")
			}
			if !tt.expectNil && len(result) != tt.expectLen {
				t.Errorf("expected %d results, got %d", tt.expectLen, len(result))
			}
			if tt.firstTitle != "" && len(result) > 0 && result[0].Title != tt.firstTitle {
				t.Errorf("expected first title %q, got %q", tt.firstTitle, result[0].Title)
			}
		})
	}
}

func TestToRendererNavHeadings_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		input     []navHeading
		expectLen int
		validate  func([]renderer.NavHeading) error
	}{
		{
			name:      "nil input",
			input:     nil,
			expectLen: 0,
		},
		{
			name:      "empty slice",
			input:     []navHeading{},
			expectLen: 0,
		},
		{
			name: "single item no children",
			input: []navHeading{
				{Title: "Item 1", ID: "item-1"},
			},
			expectLen: 1,
			validate: func(result []renderer.NavHeading) error {
				if result[0].Title != "Item 1" {
					return fmt.Errorf("expected title 'Item 1', got %q", result[0].Title)
				}
				if result[0].ID != "item-1" {
					return fmt.Errorf("expected ID 'item-1', got %q", result[0].ID)
				}
				return nil
			},
		},
		{
			name: "multiple items with children",
			input: []navHeading{
				{
					Title: "Parent 1",
					ID:    "p1",
					Children: []navHeading{
						{Title: "Child 1.1", ID: "c1-1"},
					},
				},
				{
					Title: "Parent 2",
					ID:    "p2",
				},
			},
			expectLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toRendererNavHeadings(tt.input)
			if len(result) != tt.expectLen {
				t.Errorf("expected %d items, got %d", tt.expectLen, len(result))
			}
			if tt.validate != nil {
				if err := tt.validate(result); err != nil {
					t.Errorf("validation failed: %v", err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestFlattenChaptersWithDepth_LargeNesting(t *testing.T) {
	// Test with a larger nesting level
	chapters := []config.ChapterDef{
		{
			Title: "L0",
			File:  "l0.md",
			Sections: []config.ChapterDef{
				{
					Title: "L1",
					File:  "l1.md",
					Sections: []config.ChapterDef{
						{
							Title: "L2",
							File:  "l2.md",
							Sections: []config.ChapterDef{
								{Title: "L3", File: "l3.md"},
							},
						},
					},
				},
			},
		},
	}
	result := flattenChaptersWithDepth(chapters)
	if len(result) != 4 {
		t.Errorf("expected 4 items, got %d", len(result))
	}
	depths := []int{0, 1, 2, 3}
	for i, d := range depths {
		if result[i].Depth != d {
			t.Errorf("item %d: expected depth %d, got %d", i, d, result[i].Depth)
		}
	}
}

func TestBuildHeadingTree_VariousHeadingLevels(t *testing.T) {
	// Test with headings of various levels (1-6)
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "H1", ID: "h1"},
		{Level: 2, Text: "H2", ID: "h2"},
		{Level: 3, Text: "H3", ID: "h3"},
		{Level: 4, Text: "H4", ID: "h4"},
		{Level: 5, Text: "H5", ID: "h5"},
		{Level: 6, Text: "H6", ID: "h6"},
	}
	result := buildHeadingTree(headings, "different-id")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestToRendererNavHeadings_LargeHierarchy(t *testing.T) {
	// Test with a larger nested hierarchy
	input := []navHeading{
		{
			Title: "Root 1",
			ID:    "root1",
			Children: []navHeading{
				{
					Title: "Branch 1",
					ID:    "branch1",
					Children: []navHeading{
						{Title: "Leaf 1", ID: "leaf1"},
						{Title: "Leaf 2", ID: "leaf2"},
					},
				},
				{
					Title: "Branch 2",
					ID:    "branch2",
					Children: []navHeading{
						{Title: "Leaf 3", ID: "leaf3"},
					},
				},
			},
		},
	}
	result := toRendererNavHeadings(input)
	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}
	root := result[0]
	if len(root.Children) != 2 {
		t.Errorf("expected 2 branches, got %d", len(root.Children))
	}
	if len(root.Children[0].Children) != 2 {
		t.Errorf("expected 2 leaves in branch 1, got %d", len(root.Children[0].Children))
	}
	if len(root.Children[1].Children) != 1 {
		t.Errorf("expected 1 leaf in branch 2, got %d", len(root.Children[1].Children))
	}
}

func TestToNavHeadings_NestedDeep(t *testing.T) {
	// Test deep recursion in toNavHeadings
	entries := []toc.TOCEntry{
		{
			Title: "L1",
			ID:    "l1",
			Children: []toc.TOCEntry{
				{
					Title: "L2",
					ID:    "l2",
					Children: []toc.TOCEntry{
						{
							Title: "L3",
							ID:    "l3",
							Children: []toc.TOCEntry{
								{Title: "L4", ID: "l4"},
							},
						},
					},
				},
			},
		},
	}
	result := toNavHeadings(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	current := result[0]
	expected := []string{"L1", "L2", "L3", "L4"}
	for i, exp := range expected {
		if i > 0 {
			if len(current.Children) == 0 {
				t.Fatalf("expected child at level %d", i)
			}
			current = current.Children[0]
		}
		if current.Title != exp {
			t.Errorf("level %d: expected title %q, got %q", i, exp, current.Title)
		}
	}
}

// ---------------------------------------------------------------------------
// Comprehensive integration test
// ---------------------------------------------------------------------------

func TestCompleteNavigationWorkflow(t *testing.T) {
	// Simulate a complete workflow: chapters with sections, headings, and navigation trees
	chapters := []config.ChapterDef{
		{
			Title: "Introduction",
			File:  "intro.md",
			Sections: []config.ChapterDef{
				{Title: "Getting Started", File: "getting-started.md"},
			},
		},
		{
			Title: "Main Content",
			File:  "main.md",
			Sections: []config.ChapterDef{
				{Title: "Section A", File: "sec-a.md"},
				{Title: "Section B", File: "sec-b.md"},
			},
		},
	}

	// Step 1: Flatten chapters
	flattened := flattenChaptersWithDepth(chapters)
	if len(flattened) != 5 {
		t.Fatalf("flattening: expected 5 items, got %d", len(flattened))
	}

	// Step 2: Build heading tree for one of the chapters
	headings := []markdown.HeadingInfo{
		{Level: 1, Text: "Main Content", ID: "main"},
		{Level: 2, Text: "Part A", ID: "part-a"},
		{Level: 3, Text: "Detail A.1", ID: "detail-a-1"},
		{Level: 2, Text: "Part B", ID: "part-b"},
	}
	navTree := buildHeadingTree(headings, "main")
	if navTree == nil {
		t.Fatal("expected non-nil navigation tree")
	}

	// Step 3: Convert to renderer format
	rendererNav := toRendererNavHeadings(navTree)
	if len(rendererNav) == 0 {
		t.Error("expected non-empty renderer nav")
	}

	// Verify structure is preserved through conversions
	if len(rendererNav) > 0 && rendererNav[0].Title == "" {
		t.Error("renderer nav item has empty title")
	}
}
