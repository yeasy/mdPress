// Package config loads and validates mdpress configuration.
// It reads book metadata, chapter definitions, style settings, and output options from book.yaml.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
	"gopkg.in/yaml.v3"
)

// Pre-compiled patterns for style validation.
var (
	fontFamilyPattern = regexp.MustCompile(`^[a-zA-Z0-9 ,\-'.]+$`)
	fontSizePattern   = regexp.MustCompile(`^\d+(\.\d+)?(px|pt|em|rem|%)$`)
	codeThemePattern  = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
)

// BookConfig is the top-level configuration for a book.
type BookConfig struct {
	Book     BookMeta     `yaml:"book"`
	Chapters []ChapterDef `yaml:"chapters"`
	Style    StyleConfig  `yaml:"style"`
	Output   OutputConfig `yaml:"output"`
	// Plugins lists the plugins to run during the build, in declaration order.
	Plugins []PluginConfig `yaml:"plugins"`

	// These fields are auto-detected by Load instead of being set directly in YAML.
	GlossaryFile string `yaml:"-"` // Path to GLOSSARY.md, if present.
	LangsFile    string `yaml:"-"` // Path to LANGS.md, if present.

	// baseDir is the directory that contains the config file.
	baseDir string `yaml:"-"`
}

// PluginConfig describes a single plugin entry in book.yaml.
//
// Example:
//
//	plugins:
//	  - name: word-count
//	    path: ./plugins/word-count
//	    config:
//	      warn_threshold: 500
type PluginConfig struct {
	// Name is the unique plugin identifier (lowercase, hyphen-separated).
	Name string `yaml:"name"`
	// Path is the path to the plugin executable, relative to book.yaml.
	Path string `yaml:"path"`
	// Config contains arbitrary key-value pairs passed to the plugin.
	Config map[string]any `yaml:"config"`
}

// BookMeta contains book metadata.
type BookMeta struct {
	Title       string    `yaml:"title"`
	Subtitle    string    `yaml:"subtitle"`
	Author      string    `yaml:"author"`
	Version     string    `yaml:"version"`
	Language    string    `yaml:"language"`
	Description string    `yaml:"description"`
	Cover       CoverMeta `yaml:"cover"`
}

// CoverMeta stores cover configuration.
type CoverMeta struct {
	Image      string `yaml:"image"`
	Background string `yaml:"background"` // Background color, for example "#1a1a2e".
}

// ChapterDef defines a chapter and its nested sections.
type ChapterDef struct {
	Title    string       `yaml:"title"`
	File     string       `yaml:"file"`
	Sections []ChapterDef `yaml:"sections"`
}

// StyleConfig stores style-related settings.
type StyleConfig struct {
	Theme      string            `yaml:"theme"`
	PageSize   string            `yaml:"page_size"`
	FontFamily string            `yaml:"font_family"`
	FontSize   string            `yaml:"font_size"`
	CodeTheme  string            `yaml:"code_theme"`
	LineHeight float64           `yaml:"line_height"`
	Margin     MarginConfig      `yaml:"margin"`
	Header     HeaderFooterStyle `yaml:"header"`
	Footer     HeaderFooterStyle `yaml:"footer"`
	CustomCSS  string            `yaml:"custom_css"`
}

// MarginConfig stores page margins in millimeters.
type MarginConfig struct {
	Top    float64 `yaml:"top"`
	Bottom float64 `yaml:"bottom"`
	Left   float64 `yaml:"left"`
	Right  float64 `yaml:"right"`
}

// HeaderFooterStyle stores header and footer text templates.
type HeaderFooterStyle struct {
	Left   string `yaml:"left"`
	Center string `yaml:"center"`
	Right  string `yaml:"right"`
}

// OutputConfig stores output-related settings.
type OutputConfig struct {
	Filename          string   `yaml:"filename"`
	TOC               bool     `yaml:"toc"`
	TOCMaxDepth       int      `yaml:"toc_max_depth"` // Maximum heading level to include in TOC (1-6, default 2). Level 1 = h1 only, 2 = h1+h2, etc.
	Cover             bool     `yaml:"cover"`
	Header            bool     `yaml:"header"`
	Footer            bool     `yaml:"footer"`
	Formats           []string `yaml:"formats"`            // Output formats: pdf, html, epub, site (default ["pdf"]).
	PDFTimeout        int      `yaml:"pdf_timeout"`        // PDF generation timeout in seconds (default 120).
	Watermark         string   `yaml:"watermark"`          // Watermark text (e.g., "DRAFT", "CONFIDENTIAL")
	WatermarkOpacity  float64  `yaml:"watermark_opacity"`  // Opacity 0.0-1.0 (default 0.1)
	MarginTop         string   `yaml:"margin_top"`         // e.g., "20mm" (default "15mm")
	MarginBottom      string   `yaml:"margin_bottom"`      // e.g., "20mm" (default "15mm")
	MarginLeft        string   `yaml:"margin_left"`        // e.g., "25mm" (default "20mm")
	MarginRight       string   `yaml:"margin_right"`       // e.g., "25mm" (default "20mm")
	GenerateBookmarks bool     `yaml:"generate_bookmarks"` // Generate PDF bookmarks from headings (default true)
}

// DefaultConfig returns a config populated with reasonable defaults.
func DefaultConfig() *BookConfig {
	return &BookConfig{
		Book: BookMeta{
			Title:    "Untitled Book",
			Author:   "",
			Version:  "1.0.0",
			Language: "zh-CN",
		},
		Style: StyleConfig{
			Theme:      "technical",
			PageSize:   "A4",
			FontFamily: "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans CJK SC', 'Noto Sans SC', 'Source Han Sans SC', 'Segoe UI', 'Helvetica Neue', Arial, sans-serif",
			FontSize:   "12pt",
			CodeTheme:  "github",
			LineHeight: 1.6,
			Margin: MarginConfig{
				Top:    25,
				Bottom: 25,
				Left:   20,
				Right:  20,
			},
			Header: HeaderFooterStyle{
				Left:  "{{.Book.Title}}",
				Right: "{{.Chapter.Title}}",
			},
			Footer: HeaderFooterStyle{
				Center: "{{.PageNum}}",
			},
		},
		Output: OutputConfig{
			Filename:          "output.pdf",
			TOC:               true,
			TOCMaxDepth:       2,
			Cover:             true,
			Header:            true,
			Footer:            true,
			PDFTimeout:        120,
			Watermark:         "",
			WatermarkOpacity:  0.1,
			MarginTop:         "15mm",
			MarginBottom:      "15mm",
			MarginLeft:        "20mm",
			MarginRight:       "20mm",
			GenerateBookmarks: true,
		},
		baseDir: ".",
	}
}

// Load reads a config file from disk.
// If chapters are empty, it attempts to load them from SUMMARY.md in the same directory.
// It also auto-detects GLOSSARY.md and LANGS.md.
func Load(path string) (*BookConfig, error) {
	// Limit config size to guard against malformed or malicious YAML inputs.
	// Check size via os.Stat before reading to avoid loading large files into memory.
	const maxConfigSize = 10 * 1024 * 1024 // 10MB
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w (ensure %s exists and is readable)", err, path)
	}
	if info.Size() > int64(maxConfigSize) {
		return nil, fmt.Errorf("config file is too large (%d bytes; max allowed is %d bytes)", info.Size(), maxConfigSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w (ensure %s exists and is readable)", err, path)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w (check the YAML syntax in %s)", err, path)
	}

	// Resolve the config base directory.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}
	cfg.baseDir = filepath.Dir(absPath)

	// If chapters are missing in YAML, try SUMMARY.md.
	if len(cfg.Chapters) == 0 {
		summaryPath := filepath.Join(cfg.baseDir, "SUMMARY.md")
		if _, err := os.Stat(summaryPath); err == nil {
			chapters, err := ParseSummary(summaryPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SUMMARY.md: %w", err)
			}
			cfg.Chapters = chapters
		}
	}

	// Auto-detect GLOSSARY.md.
	glossaryPath := filepath.Join(cfg.baseDir, "GLOSSARY.md")
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	// Auto-detect LANGS.md.
	langsPath := filepath.Join(cfg.baseDir, "LANGS.md")
	if _, err := os.Stat(langsPath); err == nil {
		cfg.LangsFile = langsPath
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// BaseDir returns the directory containing the config file.
func (c *BookConfig) BaseDir() string {
	return c.baseDir
}

// SetBaseDir overrides the base directory used to resolve relative paths.
// It is primarily useful for tests and for constructing configs in memory.
func (c *BookConfig) SetBaseDir(dir string) {
	c.baseDir = dir
}

// Validate checks the configuration for completeness and validity.
func (c *BookConfig) Validate() error {
	if c.Book.Title == "" {
		return errors.New("book title cannot be empty (set book.title in book.yaml)")
	}

	if len(c.Chapters) == 0 {
		return errors.New("at least one chapter is required (add chapters in book.yaml or create SUMMARY.md)")
	}

	if err := c.validateChapters(c.Chapters, ""); err != nil {
		return fmt.Errorf("chapter validation failed: %w", err)
	}

	validSizes := map[string]bool{
		"A4": true, "A5": true, "Letter": true, "Legal": true, "B5": true,
	}
	if c.Style.PageSize != "" && !validSizes[c.Style.PageSize] {
		return fmt.Errorf("unsupported page size: %q (supported: A4, A5, Letter, Legal, B5)", c.Style.PageSize)
	}

	// Validate the theme name.
	validThemes := map[string]bool{
		"technical": true, "elegant": true, "minimal": true, "": true,
	}
	if !validThemes[c.Style.Theme] {
		return fmt.Errorf("unknown theme: %q (built-ins: technical, elegant, minimal; run mdpress themes list for details)", c.Style.Theme)
	}

	// Validate output formats.
	validFormats := map[string]bool{"pdf": true, "html": true, "epub": true, "site": true, "typst": true}
	for _, f := range c.Output.Formats {
		if !validFormats[f] {
			return fmt.Errorf("unsupported output format: %q (supported: pdf, html, epub, site, typst)", f)
		}
	}

	// Validate TOCMaxDepth range (1-6, or 0 for default).
	if c.Output.TOCMaxDepth != 0 && (c.Output.TOCMaxDepth < 1 || c.Output.TOCMaxDepth > 6) {
		return fmt.Errorf("toc_max_depth must be between 1 and 6 (got %d)", c.Output.TOCMaxDepth)
	}

	// Validate WatermarkOpacity range (0.0-1.0, or 0 for not set).
	if c.Output.WatermarkOpacity != 0 && (c.Output.WatermarkOpacity < 0.0 || c.Output.WatermarkOpacity > 1.0) {
		return fmt.Errorf("watermark_opacity must be between 0.0 and 1.0 (got %f)", c.Output.WatermarkOpacity)
	}

	// Validate Watermark text: reject template injection markers and enforce length.
	if c.Output.Watermark != "" {
		if len(c.Output.Watermark) > 200 {
			return fmt.Errorf("watermark text is too long (%d characters; max 200)", len(c.Output.Watermark))
		}
		if strings.Contains(c.Output.Watermark, "{{") || strings.Contains(c.Output.Watermark, "}}") {
			return errors.New("watermark text must not contain template markers ({{ or }})")
		}
	}

	// Validate PDFTimeout range (5-3600 seconds, or 0 for default).
	if c.Output.PDFTimeout != 0 && (c.Output.PDFTimeout < 5 || c.Output.PDFTimeout > 3600) {
		return fmt.Errorf("pdf_timeout must be between 5 and 3600 seconds (got %d)", c.Output.PDFTimeout)
	}

	// Validate font_family: only allow alphanumeric, spaces, commas, hyphens, single quotes, and periods.
	if c.Style.FontFamily != "" {
		if !fontFamilyPattern.MatchString(c.Style.FontFamily) {
			return errors.New("font_family contains invalid characters (only alphanumeric, spaces, commas, hyphens, periods, and single quotes are allowed)")
		}
	}

	// Validate font_size: must match a simple CSS size pattern (e.g. 14px, 1.2em, 16pt, 100%%).
	if c.Style.FontSize != "" {
		if !fontSizePattern.MatchString(c.Style.FontSize) {
			return fmt.Errorf("font_size %q is not a valid CSS size (expected a number followed by px, pt, em, rem, or %%)", c.Style.FontSize)
		}
	}

	// Validate code_theme: only allow alphanumeric, hyphens, and underscores.
	if c.Style.CodeTheme != "" {
		if !codeThemePattern.MatchString(c.Style.CodeTheme) {
			return fmt.Errorf("code_theme %q contains invalid characters (only alphanumeric, hyphens, and underscores are allowed)", c.Style.CodeTheme)
		}
	}

	// Validate line_height: must be a positive value in a reasonable range.
	if c.Style.LineHeight != 0 && (c.Style.LineHeight < 0.5 || c.Style.LineHeight > 5.0) {
		return fmt.Errorf("line_height must be between 0.5 and 5.0 (got %g)", c.Style.LineHeight)
	}

	// Validate output filename: strip directory components to prevent path traversal.
	if c.Output.Filename != "" {
		base := filepath.Base(c.Output.Filename)
		if base != c.Output.Filename {
			return fmt.Errorf("output filename %q must not contain directory components", c.Output.Filename)
		}
	}

	// Validate that custom_css does not escape the project directory.
	if c.Style.CustomCSS != "" {
		if _, err := utils.SafeJoin(c.baseDir, c.Style.CustomCSS); err != nil {
			return fmt.Errorf("custom_css: %w", err)
		}
	}

	// Validate that cover image does not escape the project directory.
	if c.Book.Cover.Image != "" {
		if _, err := utils.SafeJoin(c.baseDir, c.Book.Cover.Image); err != nil {
			return fmt.Errorf("cover.image: %w", err)
		}
	}

	return nil
}

const maxChapterNestingDepth = 20

// validateChapters recursively validates chapter definitions and their nested sections.
func (c *BookConfig) validateChapters(chapters []ChapterDef, prefix string) error {
	return c.validateChaptersDepth(chapters, prefix, 0)
}

func (c *BookConfig) validateChaptersDepth(chapters []ChapterDef, prefix string, depth int) error {
	if depth > maxChapterNestingDepth {
		return fmt.Errorf("chapter nesting exceeds maximum depth of %d", maxChapterNestingDepth)
	}
	for i, ch := range chapters {
		label := fmt.Sprintf("%s%d", prefix, i+1)
		if ch.File == "" {
			return fmt.Errorf("chapter %s is missing a file path", label)
		}
		// Reject absolute paths and paths that escape the project directory.
		if filepath.IsAbs(ch.File) {
			return fmt.Errorf("chapter %s: absolute path not allowed: %s", label, ch.File)
		}
		resolvedPath := c.ResolvePath(ch.File)
		absResolved, err := filepath.Abs(resolvedPath)
		if err != nil {
			return fmt.Errorf("chapter %s: invalid path: %w", label, err)
		}
		// Resolve symlinks so that a symlink inside the project pointing
		// outside cannot bypass the containment check. Only apply symlink
		// resolution when both paths resolve successfully to keep them at
		// the same "resolution level".
		absBase, err := filepath.Abs(c.baseDir)
		if err != nil {
			return fmt.Errorf("chapter %s: cannot resolve base dir: %w", label, err)
		}
		if evaledR, errR := filepath.EvalSymlinks(absResolved); errR == nil {
			if evaledB, errB := filepath.EvalSymlinks(absBase); errB == nil {
				absResolved = evaledR
				absBase = evaledB
			}
		}
		if !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) && absResolved != absBase {
			return fmt.Errorf("chapter %s: path escapes project directory: %s", label, ch.File)
		}
		// Check whether the referenced chapter file exists.
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return fmt.Errorf("chapter %s references a missing file: %s (paths are relative to book.yaml)", label, ch.File)
		}
		// Recursively validate nested sections.
		if len(ch.Sections) > 0 {
			if err := c.validateChaptersDepth(ch.Sections, label+".", depth+1); err != nil {
				return fmt.Errorf("nested section validation failed: %w", err)
			}
		}
	}
	return nil
}

// FlattenChapters expands nested chapter definitions into a flat list.
// This is the canonical implementation; callers should use this instead of
// maintaining their own flattening logic.
func FlattenChapters(chapters []ChapterDef) []ChapterDef {
	var result []ChapterDef
	for _, ch := range chapters {
		result = append(result, ch)
		if len(ch.Sections) > 0 {
			result = append(result, FlattenChapters(ch.Sections)...)
		}
	}
	return result
}

// ResolvePath resolves a path relative to the config directory.
func (c *BookConfig) ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.baseDir, p)
}
