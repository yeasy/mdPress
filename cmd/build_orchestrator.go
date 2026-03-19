package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/plugin"
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
	// PluginManager manages loaded plugins and dispatches hook calls throughout the
	// build pipeline.  It is an empty (no-op) Manager when no plugins are configured.
	PluginManager *plugin.Manager
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

	// Load plugins declared in book.yaml.  A loading error produces a warning
	// but does not abort the build.
	pluginMgr := plugin.MustLoadPlugins(cfg, func(msg string) {
		logger.Warn(msg)
	})

	return &BuildOrchestrator{
		Config:        cfg,
		Theme:         thm,
		Parser:        parser,
		Gloss:         gloss,
		Logger:        logger,
		PluginManager: pluginMgr,
	}, nil
}

// ProcessChapters runs the ChapterPipeline and returns results.
func (o *BuildOrchestrator) ProcessChapters(ctxOpts ...context.Context) (*ChapterPipelineResult, error) {
	ctx := context.Background()
	if len(ctxOpts) > 0 && ctxOpts[0] != nil {
		ctx = ctxOpts[0]
	}
	return o.ProcessChaptersWithOptions(ctx, ChapterPipelineOptions{})
}

// ProcessChaptersWithOptions runs the ChapterPipeline with caller-provided options.
func (o *BuildOrchestrator) ProcessChaptersWithOptions(ctx context.Context, options ChapterPipelineOptions) (*ChapterPipelineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pipeline := NewChapterPipeline(o.Config, o.Theme, o.Parser, o.Gloss, o.Logger, o.PluginManager)
	return pipeline.ProcessWithOptions(ctx, options)
}

// LoadCustomCSS loads user-provided CSS.
func (o *BuildOrchestrator) LoadCustomCSS() string {
	return loadCustomCSS(o.Config, o.Logger)
}
