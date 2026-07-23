// Package config loads and validates mdpress configuration.
// It reads book metadata, chapter definitions, style settings, and output options from book.yaml.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
	"gopkg.in/yaml.v3"
)

// Pre-compiled patterns for style validation.
var (
	fontFamilyPattern = regexp.MustCompile(`^[\p{L}\p{N} ,\-'.]+$`)
	fontSizePattern   = regexp.MustCompile(`^\d+(\.\d+)?(px|pt|em|rem|%)$`)
	codeThemePattern  = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
)

// validPageSizes lists the accepted page size names (uppercase keys).
var validPageSizes = map[string]bool{
	"A4": true, "A5": true, "LETTER": true, "LEGAL": true, "B5": true,
}

// IsValidPageSize reports whether s is a recognized page size name (case-insensitive).
func IsValidPageSize(s string) bool {
	return validPageSizes[strings.ToUpper(s)]
}

// validPageSizeNames returns the list of accepted page size names in sorted order.
func validPageSizeNames() []string {
	names := make([]string, 0, len(validPageSizes))
	for k := range validPageSizes {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// BookConfig is the top-level configuration for a book.
type BookConfig struct {
	Book     BookMeta       `yaml:"book"`
	Chapters []ChapterDef   `yaml:"chapters"`
	Style    StyleConfig    `yaml:"style"`
	Output   OutputConfig   `yaml:"output"`
	Markdown MarkdownConfig `yaml:"markdown"`
	// Plugins lists the plugins to run during the build, in declaration order.
	Plugins []PluginConfig `yaml:"plugins"`
	// Variables are user-defined template variables, usable in Markdown as
	// {{ key }} alongside the built-in book.*/style.*/output.* values.
	Variables map[string]string `yaml:"variables"`

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

// MarkdownConfig controls how Markdown sources are parsed.
type MarkdownConfig struct {
	// AllowHTML controls whether raw HTML written in Markdown reaches the
	// output. A nil pointer means "not configured" and keeps the default.
	AllowHTML *bool `yaml:"allow_html"`
}

// AllowRawHTML reports whether raw HTML embedded in Markdown should be
// rendered as HTML rather than escaped.
//
// mdpress treats Markdown sources as trusted input — they come from the same
// repository as book.yaml — so this defaults to true and raw HTML passes
// through unfiltered, including <script> and <iframe>. A project that renders
// Markdown it did not write (community contributions, user submissions) should
// set `markdown.allow_html: false`.
func (c *BookConfig) AllowRawHTML() bool {
	return c.Markdown.AllowHTML == nil || *c.Markdown.AllowHTML
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
	// Favicon is the site icon: a project-relative image path, or an absolute
	// URL. Empty keeps mdpress's built-in book emoji.
	Favicon string `yaml:"favicon"`
	// Logo is an image shown above the title in the site sidebar: a
	// project-relative image path, or an absolute URL. Empty shows no logo.
	Logo string `yaml:"logo"`
	// Copyright is a short notice rendered in each page's footer, e.g.
	// "© 2026 Acme Inc.". Empty renders no notice.
	Copyright string `yaml:"copyright"`
}

// CoverMeta stores cover configuration.
type CoverMeta struct {
	Image      string `yaml:"image"`
	Background string `yaml:"background"` // Background color, for example "#1a1a2e".
}

// ChapterDef defines a chapter and its nested sections.
type ChapterDef struct {
	Title string `yaml:"title"`
	File  string `yaml:"file"`
	// Section is an optional group label rendered above this chapter in the
	// site sidebar, starting a new group. SUMMARY.md "## Heading" lines set it
	// on the chapter that follows them; book.yaml can set it directly. It is
	// carried on a real chapter rather than modeled as a file-less entry, so
	// nothing downstream has to cope with a chapter that has no content.
	Section  string       `yaml:"section"`
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
	MarginTop         string   `yaml:"margin_top"`         // e.g., "20mm"; unset means style.margin.top
	MarginBottom      string   `yaml:"margin_bottom"`      // e.g., "20mm"; unset means style.margin.bottom
	MarginLeft        string   `yaml:"margin_left"`        // e.g., "25mm"; unset means style.margin.left
	MarginRight       string   `yaml:"margin_right"`       // e.g., "25mm"; unset means style.margin.right
	GenerateBookmarks bool     `yaml:"generate_bookmarks"` // Generate PDF bookmarks from headings (default true)
	SiteURL           string   `yaml:"site_url"`           // Public base URL of the deployed site (e.g. https://user.github.io/repo); enables sitemap.xml
	EditBase          string   `yaml:"edit_base"`          // Base URL for "edit this page" links (e.g. https://github.com/user/repo/edit/main/)
	TaggedPDF         *bool    `yaml:"tagged_pdf"`         // Generate accessible tagged PDF (default true; false produces smaller files)
	// FooterHTML replaces the site's default "Built with mdPress" footer line.
	// A nil pointer means "not configured" and keeps the default; an explicit
	// empty string removes the line. It is a pointer for exactly that reason —
	// a plain string cannot tell "unset" from "the user wants no footer".
	// Its value is emitted as raw HTML, on the same trust footing as raw HTML
	// in the Markdown sources (see BookConfig.AllowRawHTML).
	FooterHTML *string `yaml:"footer_html"`
	// ShowThemeBadge renders the theme name as a badge in the site sidebar.
	// Off by default: it is mdpress advertising itself on someone else's
	// published site, and removing it used to require a CSS hack.
	ShowThemeBadge bool `yaml:"show_theme_badge"`
}

// DefaultBookTitle is the placeholder used when book.title is unset. It is
// exported so callers can tell "the user named their book" from "nothing took
// effect" — reporting the placeholder as a valid title hid typo'd config keys.
const DefaultBookTitle = "Untitled Book"

// DefaultConfig returns a config populated with reasonable defaults.
func DefaultConfig() *BookConfig {
	return &BookConfig{
		Book: BookMeta{
			Title:  DefaultBookTitle,
			Author: "",
			// Matches what `mdpress init` scaffolds. Zero-config discovery
			// overrides this by sniffing the content, so a Chinese book still
			// gets zh-CN without configuring anything.
			Version:  "1.0.0",
			Language: "en-US",
		},
		Style: StyleConfig{
			Theme:    "technical",
			PageSize: "A4",
			// Typography defaults live in the theme, not here: an empty value
			// means "inherit from the theme", which is what lets `elegant`
			// stay serif and `minimal` keep its own scale. A non-empty value
			// is an explicit user override and wins (see ApplyTypography).
			FontFamily: "",
			FontSize:   "",
			CodeTheme:  "", // empty inherits the theme's code_theme (e.g. github for technical, bw for minimal)
			LineHeight: 0,
			Margin: MarginConfig{
				Top:    25,
				Bottom: 25,
				Left:   20,
				Right:  20,
			},
			// Unset means "not configured". These used to carry the same
			// values the manual gives as its example, and the PDF builder
			// decided "the user customized this" by comparing against them —
			// so copying the documented example produced no header at all.
			// The effective defaults live in the PDF builder.
			Header: HeaderFooterStyle{},
			Footer: HeaderFooterStyle{},
		},
		Output: OutputConfig{
			// Empty means "derive the name from the book title". A literal
			// default of "output.pdf" forced the build to special-case that
			// exact string as "unset", which silently ignored users who really
			// did write `filename: "output.pdf"` in book.yaml.
			Filename:         "",
			TOC:              true,
			TOCMaxDepth:      2,
			Cover:            true,
			Header:           true,
			Footer:           true,
			PDFTimeout:       120,
			Watermark:        "",
			WatermarkOpacity: 0.1,
			// Unset means "use style.margin". A literal default here is
			// indistinguishable from a user's explicit choice, so it would
			// override style.margin on every build — the same trap the
			// typography defaults had.
			MarginTop:         "",
			MarginBottom:      "",
			MarginLeft:        "",
			MarginRight:       "",
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
	// Use os.Open + Fstat + LimitReader to avoid TOCTOU between stat and read.
	const maxConfigSize = 10 * 1024 * 1024 // 10MB
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w (ensure %s exists and is readable)", err, path)
	}
	defer f.Close() //nolint:errcheck
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file: %w", err)
	}
	if info.Size() > int64(maxConfigSize) {
		return nil, fmt.Errorf("config file is too large (%d bytes; max allowed is %d bytes)", info.Size(), maxConfigSize)
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(maxConfigSize)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w (ensure %s exists and is readable)", err, path)
	}
	if int64(len(data)) > int64(maxConfigSize) {
		return nil, fmt.Errorf("config file exceeds size limit during read (%d bytes; max %d)", len(data), maxConfigSize)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w (check the YAML syntax in %s)", err, path)
	}
	// A key mdpress does not recognize is almost always a typo or wrong
	// nesting, and silently dropping it is the worst outcome: the user edits
	// book.yaml, rebuilds, sees no change, and has nothing to go on. Report it
	// as a warning rather than an error so an unknown key never breaks a build
	// that used to work.
	if unknown := FindUnknownKeys(data); len(unknown) > 0 {
		for _, key := range unknown {
			slog.Warn("unknown key in config file; it will be ignored",
				slog.String("config", path),
				slog.String("key", key.Path),
				slog.String("hint", key.Hint()))
		}
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

	cfg.detectAuxFiles()

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

// detectAuxFiles auto-detects GLOSSARY.md and LANGS.md in the base directory
// and sets the corresponding config fields if the files exist.
func (c *BookConfig) detectAuxFiles() {
	if c.baseDir == "" {
		return
	}
	glossaryPath := filepath.Join(c.baseDir, "GLOSSARY.md")
	if _, err := os.Stat(glossaryPath); err == nil {
		c.GlossaryFile = glossaryPath
	}
	langsPath := filepath.Join(c.baseDir, "LANGS.md")
	if _, err := os.Stat(langsPath); err == nil {
		c.LangsFile = langsPath
	}
}

// Validate checks the configuration for completeness and validity.
//
// It reports every independent problem it finds rather than stopping at the
// first one: a book.yaml with five mistakes used to take five edit-and-rerun
// cycles to clean up. The result is an [errors.Join] value, so callers that
// want to render one line per problem can unwrap it with ValidationErrors.
func (c *BookConfig) Validate() error {
	var errs []error

	if c.Book.Title == "" {
		errs = append(errs, errors.New("book title cannot be empty (set book.title in book.yaml)"))
	}

	if len(c.Chapters) == 0 {
		// A multi-language project's root book.yaml carries shared metadata;
		// the chapters live in each language directory. Requiring chapters here
		// made the obvious thing to write — a root book.yaml with the title and
		// author — fail the build with an error about chapters that named a
		// file the user had not been asked to create.
		if c.LangsFile == "" {
			errs = append(errs, errors.New("at least one chapter is required (add chapters in book.yaml or create SUMMARY.md)"))
		}
	} else {
		errs = append(errs, c.validateChapters(c.Chapters, "")...)
	}

	if c.Style.PageSize != "" && !IsValidPageSize(c.Style.PageSize) {
		errs = append(errs, fmt.Errorf("unsupported page size: %q (supported: A4, A5, Letter, Legal, B5)", c.Style.PageSize))
	}

	// Validate the theme name. Besides the built-ins, a theme may be a YAML
	// file path (style.theme: mytheme.yaml) or the name of a project theme
	// file at themes/<name>.yaml — both are loaded by the build orchestrator.
	validThemes := map[string]bool{
		"technical": true, "elegant": true, "minimal": true, "": true,
	}
	if !validThemes[c.Style.Theme] && !isThemeFileRef(c.Style.Theme) && !c.hasProjectThemeFile(c.Style.Theme) {
		errs = append(errs, fmt.Errorf("unknown theme: %q (built-ins: technical, elegant, minimal; or provide themes/%s.yaml; run mdpress themes list for details)", c.Style.Theme, c.Style.Theme))
	}

	// Validate output formats.
	validFormats := map[string]bool{"pdf": true, "html": true, "epub": true, "site": true, "typst": true}
	for _, f := range c.Output.Formats {
		if !validFormats[f] {
			errs = append(errs, fmt.Errorf("unsupported output format: %q (supported: pdf, html, epub, site, typst)", f))
		}
	}

	// Validate TOCMaxDepth range (1-6, or 0 for default).
	if c.Output.TOCMaxDepth != 0 && (c.Output.TOCMaxDepth < 1 || c.Output.TOCMaxDepth > 6) {
		errs = append(errs, fmt.Errorf("toc_max_depth must be between 1 and 6 (got %d)", c.Output.TOCMaxDepth))
	}

	// Validate WatermarkOpacity range (0.0-1.0, or 0 for not set).
	if c.Output.WatermarkOpacity != 0 && (c.Output.WatermarkOpacity < 0.0 || c.Output.WatermarkOpacity > 1.0) {
		errs = append(errs, fmt.Errorf("watermark_opacity must be between 0.0 and 1.0 (got %f)", c.Output.WatermarkOpacity))
	}

	// Validate Watermark text: reject template injection markers and enforce length.
	if c.Output.Watermark != "" {
		if len(c.Output.Watermark) > 200 {
			errs = append(errs, fmt.Errorf("watermark text is too long (%d characters; max 200)", len(c.Output.Watermark)))
		}
		if strings.Contains(c.Output.Watermark, "{{") || strings.Contains(c.Output.Watermark, "}}") {
			errs = append(errs, errors.New("watermark text must not contain template markers ({{ or }})"))
		}
	}

	// Validate PDFTimeout range (5-3600 seconds, or 0 for default).
	if c.Output.PDFTimeout != 0 && (c.Output.PDFTimeout < 5 || c.Output.PDFTimeout > 3600) {
		errs = append(errs, fmt.Errorf("pdf_timeout must be between 5 and 3600 seconds (got %d)", c.Output.PDFTimeout))
	}

	// Validate font_family: allow Unicode letters (including CJK), digits, spaces, commas, hyphens, single quotes, and periods.
	if c.Style.FontFamily != "" {
		if !fontFamilyPattern.MatchString(c.Style.FontFamily) {
			errs = append(errs, errors.New("font_family contains invalid characters (only letters, digits, spaces, commas, hyphens, periods, and single quotes are allowed)"))
		}
	}

	// Validate font_size: must match a simple CSS size pattern (e.g. 14px, 1.2em, 16pt, 100%%).
	if c.Style.FontSize != "" {
		if !fontSizePattern.MatchString(c.Style.FontSize) {
			errs = append(errs, fmt.Errorf("font_size %q is not a valid CSS size (expected a number followed by px, pt, em, rem, or %%)", c.Style.FontSize))
		}
	}

	// Validate code_theme: only allow alphanumeric, hyphens, and underscores.
	if c.Style.CodeTheme != "" {
		if !codeThemePattern.MatchString(c.Style.CodeTheme) {
			errs = append(errs, fmt.Errorf("code_theme %q contains invalid characters (only alphanumeric, hyphens, and underscores are allowed)", c.Style.CodeTheme))
		}
	}

	// Validate line_height: must be a positive value in a reasonable range.
	if c.Style.LineHeight != 0 && (c.Style.LineHeight < 0.5 || c.Style.LineHeight > 5.0) {
		errs = append(errs, fmt.Errorf("line_height must be between 0.5 and 5.0 (got %g)", c.Style.LineHeight))
	}

	// Validate output filename: strip directory components to prevent path traversal.
	if c.Output.Filename != "" {
		base := filepath.Base(c.Output.Filename)
		if base != c.Output.Filename {
			errs = append(errs, fmt.Errorf("output filename %q must not contain directory components", c.Output.Filename))
		}
	}

	// Validate that custom_css does not escape the project directory.
	if c.Style.CustomCSS != "" {
		if _, err := utils.SafeJoin(c.baseDir, c.Style.CustomCSS); err != nil {
			errs = append(errs, fmt.Errorf("custom_css: %w", err))
		}
	}

	// Validate that cover image does not escape the project directory.
	if c.Book.Cover.Image != "" {
		if _, err := utils.SafeJoin(c.baseDir, c.Book.Cover.Image); err != nil {
			errs = append(errs, fmt.Errorf("cover.image: %w", err))
		}
	}

	// Branding images are copied into the published site, so a project-relative
	// path must stay inside the project. Absolute URLs are left to the browser.
	for _, ref := range []struct{ key, value string }{
		{"book.favicon", c.Book.Favicon},
		{"book.logo", c.Book.Logo},
	} {
		if ref.value == "" || utils.IsExternalAssetRef(ref.value) {
			continue
		}
		if _, err := utils.SafeJoin(c.baseDir, ref.value); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ref.key, err))
		}
	}

	return errors.Join(errs...)
}

// ValidationErrors flattens an error returned by [BookConfig.Validate] (or a
// wrapper around one) into the individual problems it reports, so a caller can
// render one line per problem instead of a single blob of joined text.
// A nil error yields nil; an error that is not a join yields a one-element slice.
func ValidationErrors(err error) []error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok { //nolint:errorlint // matching the join node itself, not a target
		var out []error
		for _, e := range joined.Unwrap() {
			out = append(out, ValidationErrors(e)...)
		}
		return out
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		// A wrapper such as "config validation failed: %w" only adds a prefix;
		// keep it when it wraps a single error, but look through it when it
		// hides a join so each problem still gets its own line.
		if inner := ValidationErrors(wrapped); len(inner) > 1 {
			return inner
		}
	}
	return []error{err}
}

const maxChapterNestingDepth = 20

// validateChapters recursively validates chapter definitions and their nested
// sections, returning every problem it finds. Four missing chapter files used
// to mean four runs of `mdpress validate`, one per file.
func (c *BookConfig) validateChapters(chapters []ChapterDef, prefix string) []error {
	return c.validateChaptersDepth(chapters, prefix, 0)
}

func (c *BookConfig) validateChaptersDepth(chapters []ChapterDef, prefix string, depth int) []error {
	if depth > maxChapterNestingDepth {
		return []error{fmt.Errorf("chapter nesting exceeds maximum depth of %d", maxChapterNestingDepth)}
	}
	var errs []error
	for i, ch := range chapters {
		label := fmt.Sprintf("%s%d", prefix, i+1)
		if ch.File == "" {
			errs = append(errs, fmt.Errorf("chapter %s is missing a file path", label))
			continue
		}
		// Reject absolute paths and paths that escape the project directory.
		if filepath.IsAbs(ch.File) {
			errs = append(errs, fmt.Errorf("chapter %s: absolute path not allowed: %s", label, ch.File))
			continue
		}
		resolvedPath := c.ResolvePath(ch.File)
		absResolved, err := filepath.Abs(resolvedPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("chapter %s: invalid path: %w", label, err))
			continue
		}
		// Resolve symlinks so that a symlink inside the project pointing
		// outside cannot bypass the containment check. Only apply symlink
		// resolution when both paths resolve successfully to keep them at
		// the same "resolution level".
		absBase, err := filepath.Abs(c.baseDir)
		if err != nil {
			errs = append(errs, fmt.Errorf("chapter %s: cannot resolve base dir: %w", label, err))
			continue
		}
		if evaledR, errR := filepath.EvalSymlinks(absResolved); errR == nil {
			if evaledB, errB := filepath.EvalSymlinks(absBase); errB == nil {
				absResolved = evaledR
				absBase = evaledB
			}
		}
		if !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) && absResolved != absBase {
			errs = append(errs, fmt.Errorf("chapter %s: path escapes project directory: %s", label, ch.File))
			continue
		}
		// Check whether the referenced chapter file exists.
		if _, err := os.Stat(resolvedPath); errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("chapter %s references a missing file: %s (paths are relative to book.yaml)", label, ch.File))
			continue
		}
		// Recursively validate nested sections.
		if len(ch.Sections) > 0 {
			errs = append(errs, c.validateChaptersDepth(ch.Sections, label+".", depth+1)...)
		}
	}
	return errs
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
//
// Security: This function does NOT verify that the result is contained within
// the project directory. It is the caller's responsibility to ensure the input
// path has been validated (e.g. via [utils.SafeJoin] or [BookConfig.Validate])
// before passing it here. Absolute paths are returned unchanged, so an
// attacker-controlled absolute path could escape the project root.
func (c *BookConfig) ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.baseDir, p)
}

// isThemeFileRef reports whether the theme value is a YAML file reference
// (style.theme: mytheme.yaml) rather than a theme name.
func isThemeFileRef(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

// hasProjectThemeFile reports whether the project contains a custom theme file
// at themes/<name>.yaml (or .yml) for the given theme name.
func (c *BookConfig) hasProjectThemeFile(name string) bool {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return false
	}
	for _, ext := range []string{".yaml", ".yml"} {
		if info, err := os.Stat(filepath.Join(c.BaseDir(), "themes", name+ext)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}
