package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/glossary"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/plugin"
	"github.com/yeasy/mdpress/internal/theme"
)

// allowPlugins opts in to executing plugins declared in a remote project's
// book.yaml.  It is registered as the --allow-plugins flag on both the build
// and serve commands.  Plugins are executed at load time (probe), so building
// or serving an untrusted remote repository would otherwise run arbitrary code
// from that repository's book.yaml.  For local sources plugins always load.
var allowPlugins bool

// buildSourceIsRemote signals that the currently resolved build/serve source is
// a remote (e.g. GitHub) repository rather than a local path.  It is set by the
// build and serve commands right after source.Detect resolves the input and is
// consulted by newBuildOrchestrator when deciding whether to load book.yaml
// plugins.  Using a package-level flag keeps newBuildOrchestrator's signature
// unchanged for existing callers.
var buildSourceIsRemote bool

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

// customThemePath returns the on-disk YAML file the configured theme should
// be loaded from, or "" when the theme should resolve to a built-in.
//
// Two custom-theme sources are supported (this is the mechanism advertised by
// `mdpress themes show`):
//  1. style.theme set to a YAML file path (ending in .yaml or .yml),
//     resolved relative to the directory containing book.yaml.
//  2. A project-local override at <project>/themes/<name>.yaml, where <name>
//     is the configured theme name (the default "technical" when unset).
//     When the file exists it replaces the built-in theme of the same name.
func customThemePath(cfg *config.BookConfig) string {
	name := cfg.Style.Theme
	if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		return cfg.ResolvePath(name)
	}
	if name == "" {
		name = defaultThemeName
	}
	for _, ext := range []string{".yaml", ".yml"} {
		override := cfg.ResolvePath(filepath.Join("themes", name+ext))
		if info, err := os.Stat(override); err == nil && info.Mode().IsRegular() {
			return override
		}
	}
	return ""
}

// NewBuildOrchestrator creates a fully initialized orchestrator from config.
// It loads the theme (custom theme file, or built-in with fallback), creates
// the parser, and loads the glossary.
func newBuildOrchestrator(cfg *config.BookConfig, logger *slog.Logger) (*buildOrchestrator, error) {
	// Initialize the theme. A project-local custom theme (explicit
	// style.theme YAML path or themes/<name>.yaml override) takes
	// precedence over the built-ins; a custom theme that fails to load or
	// validate is a hard error rather than a silent fallback.
	tm := theme.NewThemeManager()
	var thm *theme.Theme
	var err error
	if path := customThemePath(cfg); path != "" {
		thm, err = tm.LoadFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("custom theme %s: %w", path, err)
		}
		logger.Debug("loaded custom theme", slog.String("path", path), slog.String("theme", thm.Name))
	} else {
		thm, err = tm.Get(cfg.Style.Theme)
		if err != nil {
			logger.Warn("theme lookup failed, falling back to default", slog.String("theme", cfg.Style.Theme), slog.Any("error", err))
			thm, err = tm.Get(defaultThemeName)
			if err != nil {
				return nil, fmt.Errorf("failed to load default theme: %w", err)
			}
		}
	}

	// book.yaml's `style` typography wins over the theme's own values, so a
	// user can retune a built-in theme without forking it. Applied here, once,
	// so every renderer that reads the theme picks it up.
	thm.ApplyTypography(theme.TypographyOverride{
		FontFamily: cfg.Style.FontFamily,
		FontSize:   cfg.Style.FontSize,
		LineHeight: cfg.Style.LineHeight,
	})

	// Initialize the Markdown parser.
	codeTheme := cfg.Style.CodeTheme
	if codeTheme == "" {
		codeTheme = thm.CodeTheme
	}
	parser := markdown.NewParser(markdownParserOptions(cfg, codeTheme)...)

	// Load the glossary when configured.
	var gloss *glossary.Glossary
	if cfg.GlossaryFile != "" {
		gloss, err = glossary.ParseFile(cfg.GlossaryFile)
		if err != nil {
			logger.Warn("failed to parse GLOSSARY.md", slog.Any("error", err))
		}
	}

	// Load plugins declared in book.yaml.  A loading error produces a warning
	// but does not abort the build.
	//
	// Plugins execute at load time (probe), so refuse to run plugins declared
	// by a remote project unless the user explicitly opts in with
	// --allow-plugins.  Local sources keep the existing behavior.
	var pluginMgr *plugin.Manager
	if buildSourceIsRemote && !allowPlugins && cfg != nil && len(cfg.Plugins) > 0 {
		logger.Warn(fmt.Sprintf("Refusing to run %d plugin(s) from a remote project; pass --allow-plugins to trust and execute them.", len(cfg.Plugins)))
		pluginMgr = plugin.NewManager()
	} else {
		pluginMgr = plugin.MustLoadPlugins(cfg, func(msg string) {
			logger.Warn(msg)
		})
	}

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
