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
type buildOrchestrator struct {
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
func newBuildOrchestrator(cfg *config.BookConfig, logger *slog.Logger) (*buildOrchestrator, error) {
	// Initialize the theme.
	tm := theme.NewThemeManager()
	thm, err := tm.Get(cfg.Style.Theme)
	if err != nil {
		logger.Warn("theme lookup failed, falling back to default", slog.String("theme", cfg.Style.Theme), slog.String("error", err.Error()))
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
			logger.Warn("failed to parse GLOSSARY.md", slog.String("error", err.Error()))
		}
	}

	// Load plugins declared in book.yaml.  A loading error produces a warning
	// but does not abort the build.
	pluginMgr := plugin.MustLoadPlugins(cfg, func(msg string) {
		logger.Warn(msg)
	})

	return &buildOrchestrator{
		Config:        cfg,
		Theme:         thm,
		Parser:        parser,
		Gloss:         gloss,
		Logger:        logger,
		PluginManager: pluginMgr,
	}, nil
}

// ProcessChapters runs the ChapterPipeline and returns results.
func (o *buildOrchestrator) ProcessChapters(ctxOpts ...context.Context) (*chapterPipelineResult, error) {
	ctx := context.Background()
	if len(ctxOpts) > 0 && ctxOpts[0] != nil {
		ctx = ctxOpts[0]
	}
	return o.ProcessChaptersWithOptions(ctx, chapterPipelineOptions{})
}

// ProcessChaptersWithOptions runs the ChapterPipeline with caller-provided options.
func (o *buildOrchestrator) ProcessChaptersWithOptions(ctx context.Context, options chapterPipelineOptions) (*chapterPipelineResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	pipeline := newChapterPipeline(o.Config, o.Theme, o.Parser, o.Gloss, o.Logger, o.PluginManager)
	return pipeline.ProcessWithOptions(ctx, options)
}

// LoadCustomCSS loads user-provided CSS.
func (o *buildOrchestrator) LoadCustomCSS() string {
	return loadCustomCSS(o.Config, o.Logger)
}
