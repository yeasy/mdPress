// config_show.go implements `mdpress config show`, which prints the
// configuration a build would actually use.
//
// Until this existed there was no way to see the resolved configuration: an
// unknown key, a theme that failed to resolve, or an output filename derived
// from the book title were all invisible, so "I set it and nothing happened"
// had no first debugging step. `config show` walks the same Load/Discover path
// as `build` and prints the result, plus the values that are computed rather
// than configured (theme source, effective typography, artifact paths).
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// configShowFormat holds the --format value for `config show`.
var configShowFormat string

// configShowFormats lists the output encodings `config show` understands.
var configShowFormats = []string{"yaml", "json"}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect mdpress configuration",
	Long: `Inspect the configuration mdpress resolves for a project.

Subcommands:
  config show      Print the effective configuration a build would use

Examples:
  mdpress config show
  mdpress config show ./my-book
  mdpress config show --format json`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		hint := ""
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			hint = fmt.Sprintf("\n\nDid you mean this?\n\t%s", strings.Join(suggestions, "\n\t"))
		}
		return fmt.Errorf("unknown config sub-command %q%s\n\nRun 'mdpress config --help' to see the available sub-commands", args[0], hint)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [directory]",
	Short: "Print the effective configuration",
	Long: `Print the configuration a build of this project would use.

The output is the book.yaml settings after defaults have been applied (or the
settings auto-discovery inferred when there is no book.yaml), plus a "resolved"
section with the values mdpress computes: which config file was loaded, where
the theme came from, the typography renderers receive after style overrides,
and the file each requested output format would be written to.

Use --format json for scripting:
  mdpress config show --format json | jq -r .style.theme
  mdpress config show --format json | jq -r .resolved.artifacts.pdf

Examples:
  mdpress config show
  mdpress config show ./my-book
  mdpress config show --config release.yaml
  mdpress config show --format json`,

	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}
		return executeConfigShow(cmd.Context(), targetDir, configShowFormat, os.Stdout)
	},
	ValidArgsFunction: completeDirectories,
}

func init() {
	configCmd.SuggestionsMinimumDistance = 2
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().StringVarP(&configShowFormat, "format", "f", "yaml",
		"Output encoding: "+strings.Join(configShowFormats, " or "))
	registerFixedFlagCompletion(configShowCmd, "format", configShowFormats)
}

// configShowReport is the document `config show` prints. Only yaml tags are
// declared: the JSON encoding is produced from the YAML so both formats always
// use the same key names, which is what makes the jq recipes in the help text
// keep working when a field is added.
type configShowReport struct {
	Book      config.BookMeta       `yaml:"book"`
	Chapters  []config.ChapterDef   `yaml:"chapters"`
	Style     config.StyleConfig    `yaml:"style"`
	Output    config.OutputConfig   `yaml:"output"`
	Markdown  configShowMarkdown    `yaml:"markdown"`
	Plugins   []config.PluginConfig `yaml:"plugins"`
	Variables map[string]string     `yaml:"variables"`
	Resolved  configShowResolved    `yaml:"resolved"`
}

// configShowMarkdown reports Markdown parsing options with the "unset" state
// already collapsed, so the reader sees the behavior instead of a null.
type configShowMarkdown struct {
	AllowHTML bool `yaml:"allow_html"`
}

// configShowResolved holds everything mdpress computes rather than reads.
type configShowResolved struct {
	// ConfigFile is the file that was loaded, or "" in zero-config mode.
	ConfigFile string `yaml:"config_file"`
	// Discovered reports whether the config was inferred by auto-discovery.
	Discovered   bool              `yaml:"discovered"`
	BaseDir      string            `yaml:"base_dir"`
	ChapterCount int               `yaml:"chapter_count"`
	GlossaryFile string            `yaml:"glossary_file"`
	LangsFile    string            `yaml:"langs_file"`
	Theme        configShowTheme   `yaml:"theme"`
	Formats      []string          `yaml:"formats"`
	Artifacts    map[string]string `yaml:"artifacts"`
}

// configShowTheme describes the theme a build would load, including where it
// came from — the single most common source of "my theme did nothing".
type configShowTheme struct {
	Name string `yaml:"name"`
	// Source is "built-in" or the path of the theme YAML that replaced it.
	Source string `yaml:"source"`
	// The remaining fields are what renderers receive after book.yaml's
	// `style` typography has been layered onto the theme.
	FontFamily string  `yaml:"font_family"`
	FontSize   string  `yaml:"font_size"`
	LineHeight float64 `yaml:"line_height"`
	CodeTheme  string  `yaml:"code_theme"`
}

// executeConfigShow loads the project the same way `build` does and writes the
// resolved configuration to w.
func executeConfigShow(ctx context.Context, targetDir, format string, w io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "yaml" && format != "json" {
		return fmt.Errorf("unsupported output format %q (supported: %s)", format, strings.Join(configShowFormats, ", "))
	}

	report, err := buildConfigShowReport(ctx, targetDir)
	if err != nil {
		return err
	}

	encoded, err := encodeConfigShowReport(report, format)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, encoded)
	return err
}

// buildConfigShowReport resolves the project configuration into a report.
func buildConfigShowReport(ctx context.Context, targetDir string) (*configShowReport, error) {
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve target directory: %w", err)
	}

	// Mirror `build`/`validate`: an explicit --config wins over the target
	// directory's own book.yaml, and only the implicit default may fall back
	// to auto-discovery.
	sourceDir := ""
	if targetDir != "." {
		sourceDir = absTargetDir
	}
	configPath, allowDiscovery := resolveConfigPath(sourceDir)
	if !allowDiscovery && !utils.FileExists(configPath) {
		return nil, errExplicitConfigMissing(configPath)
	}

	var (
		cfg        *config.BookConfig
		discovered bool
	)
	if utils.FileExists(configPath) {
		cfg, err = config.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		if abs, absErr := filepath.Abs(configPath); absErr == nil {
			configPath = abs
		}
	} else {
		cfg, err = config.Discover(ctx, absTargetDir)
		if err != nil {
			return nil, fmt.Errorf("auto-discovery failed: %w (try running 'mdpress init' to create a config)", err)
		}
		discovered = true
		configPath = ""
	}

	return newConfigShowReport(cfg, configPath, discovered), nil
}

// newConfigShowReport assembles the report from a loaded config.
func newConfigShowReport(cfg *config.BookConfig, configPath string, discovered bool) *configShowReport {
	formats := cfg.Output.Formats
	if len(formats) == 0 {
		// Same fallback as executeBuild, so the reported artifacts match what
		// a bare `mdpress build` would write.
		formats = []string{"pdf"}
	}

	artifacts := map[string]string{}
	if baseOutput, err := resolveBuildBaseOutput(cfg, ""); err == nil {
		siteDir := resolveSiteOutputDir(cfg.BaseDir(), "", formats, nil)
		artifacts = predictedOutputLinks(baseOutput, siteDir, formats)
	}

	return &configShowReport{
		Book:      cfg.Book,
		Chapters:  cfg.Chapters,
		Style:     cfg.Style,
		Output:    cfg.Output,
		Markdown:  configShowMarkdown{AllowHTML: cfg.AllowRawHTML()},
		Plugins:   cfg.Plugins,
		Variables: cfg.Variables,
		Resolved: configShowResolved{
			ConfigFile:   configPath,
			Discovered:   discovered,
			BaseDir:      cfg.BaseDir(),
			ChapterCount: len(config.FlattenChapters(cfg.Chapters)),
			GlossaryFile: cfg.GlossaryFile,
			LangsFile:    cfg.LangsFile,
			Theme:        resolveConfigShowTheme(cfg),
			Formats:      formats,
			Artifacts:    artifacts,
		},
	}
}

// resolveConfigShowTheme reports the theme a build would use. It repeats the
// resolution order of newBuildOrchestrator (custom file, then built-in, then
// the default) so the answer is the one builds act on, and it degrades to the
// configured name when the theme cannot be loaded at all.
func resolveConfigShowTheme(cfg *config.BookConfig) configShowTheme {
	info := configShowTheme{Name: cfg.Style.Theme, Source: "built-in"}
	if info.Name == "" {
		info.Name = defaultThemeName
	}

	tm := theme.NewThemeManager()
	var (
		thm *theme.Theme
		err error
	)
	if path := customThemePath(cfg); path != "" {
		thm, err = tm.LoadFromFile(path)
		if err == nil {
			info.Source = path
		}
	} else {
		thm, err = tm.Get(info.Name)
		if err != nil {
			// A build warns and falls back to the default theme here; say so
			// rather than reporting a name no renderer will ever see.
			thm, err = tm.Get(defaultThemeName)
			if err == nil {
				info.Source = fmt.Sprintf("built-in (fallback: theme %q not found)", cfg.Style.Theme)
			}
		}
	}
	if err != nil || thm == nil {
		return info
	}

	thm.ApplyTypography(theme.TypographyOverride{
		FontFamily: cfg.Style.FontFamily,
		FontSize:   cfg.Style.FontSize,
		LineHeight: cfg.Style.LineHeight,
	})
	info.Name = thm.Name
	info.FontFamily = thm.FontFamily
	info.FontSize = thm.ResolvedFontSize()
	info.LineHeight = thm.LineHeight
	// An empty style.code_theme inherits the theme's, which is what the
	// Markdown parser is actually given.
	info.CodeTheme = cfg.Style.CodeTheme
	if info.CodeTheme == "" {
		info.CodeTheme = thm.CodeTheme
	}
	return info
}

// encodeConfigShowReport renders the report as YAML, or as JSON derived from
// that YAML so both encodings carry identical keys.
func encodeConfigShowReport(report *configShowReport, format string) (string, error) {
	encoded, err := yaml.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("failed to encode configuration: %w", err)
	}
	if format == "yaml" {
		return string(encoded), nil
	}

	var generic map[string]any
	if err := yaml.Unmarshal(encoded, &generic); err != nil {
		return "", fmt.Errorf("failed to encode configuration as JSON: %w", err)
	}
	jsonBytes, err := json.MarshalIndent(generic, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode configuration as JSON: %w", err)
	}
	return string(jsonBytes) + "\n", nil
}
