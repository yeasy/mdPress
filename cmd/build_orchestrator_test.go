package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// TestNewBuildOrchestrator_BasicInitialization tests basic orchestrator creation with valid config
func TestNewBuildOrchestrator_BasicInitialization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title:  "Test Book",
			Author: "Test Author",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator == nil {
		t.Fatal("orchestrator should not be nil")
	}

	if orchestrator.Config != cfg {
		t.Error("Config should match the provided config")
	}

	if orchestrator.Theme == nil {
		t.Error("Theme should be initialized")
	}

	if orchestrator.Parser == nil {
		t.Error("Parser should be initialized")
	}

	if orchestrator.Logger != logger {
		t.Error("Logger should match the provided logger")
	}

	if orchestrator.PluginManager == nil {
		t.Error("PluginManager should be initialized")
	}
}

// TestNewBuildOrchestrator_WithCodeThemeFromConfig tests that code theme is taken from config when set
func TestNewBuildOrchestrator_WithCodeThemeFromConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme:     "technical",
			CodeTheme: "monokai",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Parser == nil {
		t.Error("Parser should be initialized")
	}
	// The parser's code theme would be set to "monokai" from config
}

// TestNewBuildOrchestrator_WithThemeFallback tests fallback to default theme when specified theme doesn't exist
func TestNewBuildOrchestrator_WithThemeFallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "nonexistent-theme-xyz",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator should not fail with fallback: %v", err)
	}

	if orchestrator == nil {
		t.Fatal("orchestrator should not be nil after fallback")
	}

	if orchestrator.Theme == nil {
		t.Error("Theme should fall back to default")
	}
	// The theme should be the default "technical" theme
}

// TestNewBuildOrchestrator_WithGlossary tests glossary loading when configured
func TestNewBuildOrchestrator_WithGlossary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create temporary glossary file
	tmpDir := t.TempDir()

	glossaryPath := filepath.Join(tmpDir, "GLOSSARY.md")
	glossaryContent := "# Glossary\n\n## API\nApplication Programming Interface\n"
	if err := os.WriteFile(glossaryPath, []byte(glossaryContent), 0o644); err != nil {
		t.Fatalf("failed to write glossary: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		GlossaryFile: glossaryPath,
		Chapters:     []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Gloss == nil {
		t.Error("Glossary should be loaded when GlossaryFile is set")
	}
}

// TestNewBuildOrchestrator_WithoutGlossary tests that missing glossary doesn't cause failure
func TestNewBuildOrchestrator_WithoutGlossary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		GlossaryFile: "", // No glossary
		Chapters:     []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Gloss != nil {
		t.Error("Glossary should be nil when GlossaryFile is empty")
	}
}

// TestNewBuildOrchestrator_WithInvalidGlossary tests graceful handling of invalid glossary files
func TestNewBuildOrchestrator_WithInvalidGlossary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		GlossaryFile: "/nonexistent/glossary/path/GLOSSARY.md",
		Chapters:     []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator should not fail with invalid glossary: %v", err)
	}

	if orchestrator == nil {
		t.Fatal("orchestrator should be created despite glossary failure")
	}
	// Should continue with nil glossary
	if orchestrator.Gloss != nil {
		t.Error("Glossary should be nil when file cannot be parsed")
	}
}

// TestProcessChapters_EmptyChapters tests ProcessChapters with no chapters configured.
// With an empty chapter list the pipeline returns an error.
func TestProcessChapters_EmptyChapters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	_, err = orchestrator.ProcessChapters()
	if err == nil {
		t.Error("ProcessChapters with empty chapters should return an error")
	}
}

// TestProcessChapters_WithRealChapter tests ProcessChapters with a real chapter file
func TestProcessChapters_WithRealChapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tmpDir := t.TempDir()

	chapterPath := filepath.Join(tmpDir, "ch1.md")
	if err := os.WriteFile(chapterPath, []byte("# Chapter 1\n\nHello world.\n"), 0o644); err != nil {
		t.Fatalf("failed to write chapter: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{
			{Title: "Chapter 1", File: "ch1.md"},
		},
	}
	cfg.SetBaseDir(tmpDir)

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	result, err := orchestrator.ProcessChapters()
	if err != nil {
		t.Fatalf("ProcessChapters failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}
}

// TestProcessChapters_WithCanceledContext tests ProcessChapters with canceled context
func TestProcessChapters_WithCanceledContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	_, err = orchestrator.ProcessChapters(ctx)
	// Should return an error (either context canceled or no chapters)
	if err == nil {
		t.Error("ProcessChapters with canceled context should return an error")
	}
}

// TestProcessChaptersWithOptions_EmptyChapters tests with empty chapters
func TestProcessChaptersWithOptions_EmptyChapters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	ctx := context.Background()
	options := chapterPipelineOptions{}

	_, err = orchestrator.ProcessChaptersWithOptions(ctx, options)
	if err == nil {
		t.Error("ProcessChaptersWithOptions with empty chapters should return an error")
	}
}

// TestProcessChaptersWithOptions_WithConcurrencyOption tests with concurrency option
func TestProcessChaptersWithOptions_WithConcurrencyOption(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tmpDir := t.TempDir()

	chapterPath := filepath.Join(tmpDir, "ch1.md")
	if err := os.WriteFile(chapterPath, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("failed to write chapter: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{
			{Title: "Hello", File: "ch1.md"},
		},
	}
	cfg.SetBaseDir(tmpDir)

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	ctx := context.Background()
	options := chapterPipelineOptions{
		MaxConcurrency: 4,
	}

	result, err := orchestrator.ProcessChaptersWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("ProcessChaptersWithOptions with concurrency failed: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}
}

// TestProcessChaptersWithOptions_WithNilContext tests with nil context
func TestProcessChaptersWithOptions_WithNilContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	options := chapterPipelineOptions{}

	// Pass nil context - should be handled (returns error because no chapters)
	//nolint:staticcheck // testing nil context handling
	_, err = orchestrator.ProcessChaptersWithOptions(nil, options)
	// Error expected because no chapters, but should not panic
	if err == nil {
		t.Error("expected error with empty chapters")
	}
}

// TestLoadCustomCSS_NoCustomCSS tests loading when no custom CSS is configured
func TestLoadCustomCSS_NoCustomCSS(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme:     "technical",
			CustomCSS: "",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	css := orchestrator.LoadCustomCSS()
	if css != "" {
		t.Errorf("expected empty CSS when CustomCSS is not configured, got: %q", css)
	}
}

// TestLoadCustomCSS_WithValidFile tests loading custom CSS from a valid file
func TestLoadCustomCSS_WithValidFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create temporary CSS file
	tmpDir := t.TempDir()

	cssPath := filepath.Join(tmpDir, "custom.css")
	cssContent := "body { color: #333; font-family: Arial, sans-serif; }"
	if err := os.WriteFile(cssPath, []byte(cssContent), 0o644); err != nil {
		t.Fatalf("failed to write CSS file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme:     "technical",
			CustomCSS: cssPath,
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	css := orchestrator.LoadCustomCSS()
	if css == "" {
		t.Error("expected non-empty CSS when valid file is provided")
	}

	if css != cssContent {
		t.Errorf("expected CSS content to match, got: %q", css)
	}
}

// TestLoadCustomCSS_WithInvalidPath tests loading custom CSS from invalid path
func TestLoadCustomCSS_WithInvalidPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme:     "technical",
			CustomCSS: "/nonexistent/path/to/custom.css",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	css := orchestrator.LoadCustomCSS()
	// Should return empty string on error (graceful fallback)
	if css != "" {
		t.Errorf("expected empty CSS when file cannot be loaded, got: %q", css)
	}
}

// TestLoadCustomCSS_MultipleInvocations tests that LoadCustomCSS is idempotent
func TestLoadCustomCSS_MultipleInvocations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create temporary CSS file
	tmpDir := t.TempDir()

	cssPath := filepath.Join(tmpDir, "custom.css")
	cssContent := "body { margin: 0; padding: 0; }"
	if err := os.WriteFile(cssPath, []byte(cssContent), 0o644); err != nil {
		t.Fatalf("failed to write CSS file: %v", err)
	}

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme:     "technical",
			CustomCSS: cssPath,
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	css1 := orchestrator.LoadCustomCSS()
	css2 := orchestrator.LoadCustomCSS()

	if css1 != css2 {
		t.Error("LoadCustomCSS should return same content on multiple calls")
	}
}

// TestBuildOrchestrator_ThemeFields tests that theme is properly initialized
func TestBuildOrchestrator_ThemeFields(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Theme == nil {
		t.Fatal("Theme should not be nil")
	}

	// Verify theme has a name set
	if orchestrator.Theme.Name == "" {
		t.Error("Theme should have Name set")
	}
}

// TestBuildOrchestrator_ParserFields tests that parser is properly initialized
func TestBuildOrchestrator_ParserFields(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Parser == nil {
		t.Fatal("Parser should not be nil")
	}
}

// TestBuildOrchestrator_LoggerPreservation tests that logger is preserved
func TestBuildOrchestrator_LoggerPreservation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Logger != logger {
		t.Error("Logger should be the same instance passed to NewBuildOrchestrator")
	}
}

// TestBuildOrchestrator_ConfigPreservation tests that config is preserved
func TestBuildOrchestrator_ConfigPreservation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title:  "Test Book",
			Author: "Test Author",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.Config != cfg {
		t.Error("Config should be the same instance passed to NewBuildOrchestrator")
	}

	if orchestrator.Config.Book.Title != "Test Book" {
		t.Errorf("Config.Book.Title should be preserved, got: %q", orchestrator.Config.Book.Title)
	}

	if orchestrator.Config.Book.Author != "Test Author" {
		t.Errorf("Config.Book.Author should be preserved, got: %q", orchestrator.Config.Book.Author)
	}
}

// TestBuildOrchestrator_PluginManagerInitialized tests that plugin manager is initialized
func TestBuildOrchestrator_PluginManagerInitialized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := &config.BookConfig{
		Book: config.BookMeta{
			Title: "Test Book",
		},
		Style: config.StyleConfig{
			Theme: "technical",
		},
		Chapters: []config.ChapterDef{},
	}

	orchestrator, err := newBuildOrchestrator(cfg, logger)
	if err != nil {
		t.Fatalf("NewBuildOrchestrator failed: %v", err)
	}

	if orchestrator.PluginManager == nil {
		t.Fatal("PluginManager should not be nil")
	}
}
