package cmd

import (
	"fmt"
	"log/slog"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/theme"
)

// BuildOrchestrator encapsulates the shared build initialization workflow
// used by both `build` and `serve` commands.
type BuildOrchestrator struct {
	Config *config.BookConfig
	Theme  *theme.Theme
	Parser *markdown.Parser
	Gloss  *glossary.Glossary
	Logger *slog.Logger
}

// NewBuildOrchestrator creates a fully initialized orchestrator from config.
// It loads the theme (with fallback), creates the parser, and loads the glossary.
func NewBuildOrchestrator(cfg *config.BookConfig, logger *slog.Logger) (*BuildOrchestrator, error) {
	// Initialize the theme.
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		logger.Warn("Theme lookup failed, falling back to default", slog.String("theme", cfg.Style.Theme), slog.String("error", err.Error()))
		thm, err = tm.Get("technical")
		if err != nil {
			return nil, fmt.Errorf("failed to load default theme: %w", err)
		}
	}

	// Initialize the Markdown parser.
	codeTheme := cfg.Style.CodeTheme
	if codeTheme == "" {
		codeTheme = thm.CodeTheme
	}
	parser := markdown.NewParser(markdown.WithCodeTheme(codeTheme))

	// Load the glossary when configured.
	var gloss *glossary.Glossary
	if cfg.GlossaryFile != "" {
		gloss, err = glossary.ParseFile(cfg.GlossaryFile)
		if err != nil {
			logger.Warn("Failed to parse GLOSSARY.md", slog.String("error", err.Error()))
		}
	}

	return &BuildOrchestrator{
		Config: cfg,
		Theme:  thm,
		Parser: parser,
		Gloss:  gloss,
		Logger: logger,
	}, nil
}

// ProcessChapters runs the ChapterPipeline and returns results.
func (o *BuildOrchestrator) ProcessChapters() (*ChapterPipelineResult, error) {
	pipeline := NewChapterPipeline(o.Config, o.Theme, o.Parser, o.Gloss, o.Logger)
	return pipeline.Process()
}

// LoadCustomCSS loads user-provided CSS.
func (o *BuildOrchestrator) LoadCustomCSS() string {
	return loadCustomCSS(o.Config, o.Logger)
}
