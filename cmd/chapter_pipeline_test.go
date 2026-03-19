package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/theme"
)

// TestChapterPipelineBasic tests basic chapter processing with a simple markdown file.
func TestChapterPipelineBasic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple markdown file with a heading
	chapterContent := `# Chapter One

This is the first chapter with some content.

## Section 1.1

Some subsection text.
`
	chapterFile := filepath.Join(tmpDir, "chapter1.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	// Create config pointing to the temp directory
	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title:  "Test Book",
			Author: "Test Author",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter One",
				File:  "chapter1.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	// Create parser and theme
	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, err := themeManager.Get("technical")
	if err != nil {
		t.Fatalf("Failed to get theme: %v", err)
	}

	// Create logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create and run pipeline
	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Verify chapters were returned
	if result == nil {
		t.Fatal("Pipeline result is nil")
		return
	}

	// Verify chapter count
	if len(result.Chapters) != 1 {
		t.Errorf("Expected 1 chapter, got %d", len(result.Chapters))
	}

	// Verify chapter content
	if len(result.Chapters) > 0 {
		chapter := result.Chapters[0]

		// Check title
		if chapter.Title != "Chapter One" {
			t.Errorf("Expected chapter title 'Chapter One', got %q", chapter.Title)
		}

		// Check that content contains rendered HTML
		if !strings.Contains(chapter.Content, "Chapter One") {
			t.Errorf("Chapter content does not contain expected heading text")
		}

		// Check that content contains section text
		if !strings.Contains(chapter.Content, "first chapter") {
			t.Errorf("Chapter content does not contain expected body text")
		}

		// Verify ID is set (either from heading or fallback)
		if chapter.ID == "" {
			t.Error("Chapter ID is empty")
		}
	}

	// Verify chapter files list
	if len(result.ChapterFiles) != 1 {
		t.Errorf("Expected 1 chapter file, got %d", len(result.ChapterFiles))
	}
}

// TestChapterPipelineNoChapters tests that an error is returned when no chapters are processed.
func TestChapterPipelineNoChapters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a chapter file that doesn't exist
	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title:  "Test Book",
			Author: "Test Author",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Missing Chapter",
				File:  "nonexistent.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	// Should return an error about no chapters processed
	if err == nil {
		t.Error("Expected an error when no chapters are processed")
	}

	if result != nil {
		t.Error("Expected nil result when no chapters are processed")
	}

	// Check error message
	if !strings.Contains(err.Error(), "no chapters were processed") {
		t.Errorf("Expected error about no chapters processed, got: %v", err)
	}
}

// TestChapterPipelineFallbackID tests that fallback chapter IDs are generated correctly.
func TestChapterPipelineFallbackID(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a markdown file WITHOUT a heading
	chapterContent := `This is a chapter without a heading.

Just some regular paragraph text.
`
	chapterFile := filepath.Join(tmpDir, "chapter1.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter One",
				File:  "chapter1.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	if len(result.Chapters) != 1 {
		t.Fatalf("Expected 1 chapter, got %d", len(result.Chapters))
	}

	// Verify fallback ID is 1-based (chapter-1)
	expectedID := "chapter-1"
	if result.Chapters[0].ID != expectedID {
		t.Errorf("Expected fallback ID %q, got %q", expectedID, result.Chapters[0].ID)
	}
}

// TestChapterPipelineMissingFile tests that missing chapter files are skipped.
func TestChapterPipelineMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid chapter file
	chapterContent := `# Chapter Two

This is the second chapter.
`
	chapterFile := filepath.Join(tmpDir, "chapter2.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Missing Chapter",
				File:  "nonexistent.md",
			},
			{
				Title: "Chapter Two",
				File:  "chapter2.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	// Should succeed because at least one chapter was processed
	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Should have exactly 1 chapter (the missing one is skipped)
	if len(result.Chapters) != 1 {
		t.Errorf("Expected 1 chapter (missing one skipped), got %d", len(result.Chapters))
	}

	if len(result.Chapters) > 0 {
		if result.Chapters[0].Title != "Chapter Two" {
			t.Errorf("Expected remaining chapter to be 'Chapter Two', got %q", result.Chapters[0].Title)
		}
	}
}

// TestChapterPipelineGlossaryIntegration tests glossary integration.
// Note: This is a simplified test that verifies the glossary parameter is accepted.
// Full glossary functionality would require setting up actual glossary terms.
func TestChapterPipelineGlossaryIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a chapter with a word that could be highlighted
	chapterContent := `# Chapter One

This chapter mentions a term that should be glossarized.
`
	chapterFile := filepath.Join(tmpDir, "chapter1.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter One",
				File:  "chapter1.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Run pipeline with nil glossary (most tests will use nil)
	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process with nil glossary failed: %v", err)
	}

	if len(result.Chapters) != 1 {
		t.Fatalf("Expected 1 chapter, got %d", len(result.Chapters))
	}

	// Just verify the chapter was processed successfully
	chapter := result.Chapters[0]
	if chapter.Title != "Chapter One" {
		t.Errorf("Expected chapter title 'Chapter One', got %q", chapter.Title)
	}

	// Verify content is present
	if chapter.Content == "" {
		t.Error("Chapter content is empty")
	}
}

// TestChapterPipelineMultipleChapters tests processing multiple chapters.
func TestChapterPipelineMultipleChapters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple chapter files
	chapters := []struct {
		filename string
		content  string
		title    string
	}{
		{
			filename: "intro.md",
			content:  "# Introduction\n\nThis is the introduction.",
			title:    "Introduction",
		},
		{
			filename: "chapter1.md",
			content:  "# Chapter 1\n\nFirst chapter content.",
			title:    "Chapter 1",
		},
		{
			filename: "chapter2.md",
			content:  "# Chapter 2\n\nSecond chapter content.",
			title:    "Chapter 2",
		},
	}

	var chapterDefs []config.ChapterDef
	for _, ch := range chapters {
		filepath := filepath.Join(tmpDir, ch.filename)
		if err := os.WriteFile(filepath, []byte(ch.content), 0644); err != nil {
			t.Fatalf("Failed to write chapter file: %v", err)
		}
		chapterDefs = append(chapterDefs, config.ChapterDef{
			Title: ch.title,
			File:  ch.filename,
		})
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: chapterDefs,
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Verify all chapters were processed
	if len(result.Chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(result.Chapters))
	}

	// Verify order and content
	expectedTitles := []string{"Introduction", "Chapter 1", "Chapter 2"}
	for i, expectedTitle := range expectedTitles {
		if i >= len(result.Chapters) {
			break
		}
		if result.Chapters[i].Title != expectedTitle {
			t.Errorf("Chapter %d: expected title %q, got %q", i, expectedTitle, result.Chapters[i].Title)
		}
	}

	// Verify chapter files list
	if len(result.ChapterFiles) != 3 {
		t.Errorf("Expected 3 chapter files, got %d", len(result.ChapterFiles))
	}
}

// TestChapterPipelineHeadings tests that headings are collected properly.
func TestChapterPipelineHeadings(t *testing.T) {
	tmpDir := t.TempDir()

	chapterContent := `# Main Heading

## Subheading 1

Some content here.

### Sub-subheading 1.1

More content.

## Subheading 2

Final section.
`
	chapterFile := filepath.Join(tmpDir, "chapter.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter",
				File:  "chapter.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Verify headings were collected
	if len(result.AllHeadings) == 0 {
		t.Error("Expected headings to be collected, got none")
	}

	// Verify we have at least the main heading
	foundMainHeading := false
	for _, heading := range result.AllHeadings {
		if strings.Contains(heading.Text, "Main Heading") {
			foundMainHeading = true
			if heading.Level != 1 {
				t.Errorf("Main heading should be level 1, got %d", heading.Level)
			}
		}
	}

	if !foundMainHeading {
		t.Error("Expected to find 'Main Heading' in collected headings")
	}
}

// TestChapterPipelineResolverIntegration tests that the cross-reference resolver is initialized.
func TestChapterPipelineResolverIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	chapterContent := `# Chapter One

See [Section below](#section-reference) for details.

## Section Reference

Details here.
`
	chapterFile := filepath.Join(tmpDir, "chapter.md")
	if err := os.WriteFile(chapterFile, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter One",
				File:  "chapter.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Verify resolver is present and initialized
	if result.Resolver == nil {
		t.Error("Expected resolver to be initialized")
	}
}

// TestChapterPipelineNestedChapters tests processing nested chapter sections.
func TestChapterPipelineNestedChapters(t *testing.T) {
	tmpDir := t.TempDir()

	// Create chapter files
	chapterContent := `# Part One

Introductory content.
`
	part1File := filepath.Join(tmpDir, "part1.md")
	if err := os.WriteFile(part1File, []byte(chapterContent), 0644); err != nil {
		t.Fatalf("Failed to write part1 file: %v", err)
	}

	sectionContent := `# Section One

Section content.
`
	section1File := filepath.Join(tmpDir, "section1.md")
	if err := os.WriteFile(section1File, []byte(sectionContent), 0644); err != nil {
		t.Fatalf("Failed to write section1 file: %v", err)
	}

	// Create nested chapter structure
	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Part One",
				File:  "part1.md",
				Sections: []config.ChapterDef{
					{
						Title: "Section One",
						File:  "section1.md",
					},
				},
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	result, err := pipeline.Process(context.Background())

	if err != nil {
		t.Fatalf("Pipeline process failed: %v", err)
	}

	// Should process both parent and child sections (flattened)
	if len(result.Chapters) != 2 {
		t.Errorf("Expected 2 chapters (parent + child), got %d", len(result.Chapters))
	}

	// Verify depth is set correctly
	if len(result.Chapters) >= 2 {
		if result.Chapters[0].Depth != 0 {
			t.Errorf("First chapter depth should be 0, got %d", result.Chapters[0].Depth)
		}
		if result.Chapters[1].Depth != 1 {
			t.Errorf("Second chapter depth should be 1, got %d", result.Chapters[1].Depth)
		}
	}
}

func TestChapterPipelineCanceledContext(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "chapter.md"), []byte("# Title\n\ncontent"), 0644); err != nil {
		t.Fatalf("Failed to write chapter file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Chapters: []config.ChapterDef{
			{
				Title: "Chapter 1",
				File:  "chapter.md",
			},
		},
	}
	cfg.SetBaseDir(tmpDir)

	parser := markdown.NewParser()
	themeManager := theme.NewThemeManager()
	thm, _ := themeManager.Get("technical")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pipeline := NewChapterPipeline(cfg, thm, parser, nil, logger, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pipeline.Process(ctx)
	if err == nil {
		t.Fatal("Expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
}
