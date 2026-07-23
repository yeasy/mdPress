package theme

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// cssValueSafe rejects CSS values containing characters that could break out
// of a CSS property declaration (semicolons, braces, backslashes, angle brackets).
var cssValueSafe = regexp.MustCompile(`^[^;{}<>\\]*$`)

const defaultCJKMonoFontFamily = "ui-monospace, 'SF Mono', Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans Mono CJK SC', monospace"

// Theme defines the document theme styling.
type Theme struct {
	Name           string         `yaml:"name"`
	PageSize       string         `yaml:"page_size"` // e.g. A4, Letter
	FontFamily     string         `yaml:"font_family"`
	FontSize       int            `yaml:"font_size"` // in pt
	CodeTheme      string         `yaml:"code_theme"`
	LineHeight     float64        `yaml:"line_height"`
	Colors         ColorScheme    `yaml:"colors"`
	Margins        MarginSettings `yaml:"margins"`
	HeaderTemplate string         `yaml:"header_template"`
	FooterTemplate string         `yaml:"footer_template"`

	// FontSizeCSS overrides FontSize with a literal CSS length. book.yaml's
	// style.font_size accepts any CSS unit (px, em, rem, %), which the integer
	// point value above cannot represent, so the resolved config sets this
	// when the user configured a size. Empty means "use FontSize".
	FontSizeCSS string `yaml:"-"`
}

// TypographyOverride carries the typography settings from book.yaml's `style`
// block. They win over the theme's own values, so a user can retune a built-in
// theme without forking it.
type TypographyOverride struct {
	FontFamily string
	FontSize   string
	LineHeight float64
}

// ResolvedFontSize returns the theme's body font size as a CSS length, taking
// a user override into account. Use it instead of formatting FontSize
// directly, so every backend agrees on the effective size.
func (t *Theme) ResolvedFontSize() string {
	if t.FontSizeCSS != "" {
		return t.FontSizeCSS
	}
	return fmt.Sprintf("%dpt", t.FontSize)
}

// ApplyTypography overlays non-empty user typography onto the theme. Without
// it, style.font_family / font_size / line_height are parsed and validated but
// never reach any renderer.
func (t *Theme) ApplyTypography(o TypographyOverride) {
	if o.FontFamily != "" {
		t.FontFamily = o.FontFamily
	}
	if o.FontSize != "" {
		t.FontSizeCSS = o.FontSize
	}
	if o.LineHeight > 0 {
		t.LineHeight = o.LineHeight
	}
}

// ColorScheme defines the color palette for a theme.
type ColorScheme struct {
	Text       string `yaml:"text"`
	Background string `yaml:"background"`
	Heading    string `yaml:"heading"`
	Link       string `yaml:"link"`
	CodeBg     string `yaml:"code_bg"`
	CodeText   string `yaml:"code_text"`
	Accent     string `yaml:"accent"`
	Border     string `yaml:"border"`
}

// MarginSettings defines page margins in millimeters.
type MarginSettings struct {
	Top    float64 `yaml:"top"`
	Bottom float64 `yaml:"bottom"`
	Left   float64 `yaml:"left"`
	Right  float64 `yaml:"right"`
}

// ThemeManager manages theme loading and retrieval.
type ThemeManager struct {
	themes map[string]*Theme
}

// NewThemeManager creates a new theme manager pre-loaded with built-in themes.
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]*Theme),
	}

	tm.themes["technical"] = builtinTechnical()
	tm.themes["elegant"] = builtinElegant()
	tm.themes["minimal"] = builtinMinimal()

	return tm
}

// Get returns the theme with the given name.
func (tm *ThemeManager) Get(name string) (*Theme, error) {
	if name == "" {
		return tm.themes["technical"], nil
	}

	theme, exists := tm.themes[name]
	if !exists {
		return nil, fmt.Errorf("theme '%s' not found", name)
	}

	return theme, nil
}

// LoadFromFile loads a theme from a YAML file.
func (tm *ThemeManager) LoadFromFile(path string) (*Theme, error) {
	// Use os.Open + Fstat + LimitReader to avoid TOCTOU between stat and read.
	const maxThemeSize = 1 * 1024 * 1024 // 1 MB
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("theme file not found: %w", err)
	}
	defer f.Close() //nolint:errcheck
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat theme file: %w", err)
	}
	if info.Size() > int64(maxThemeSize) {
		return nil, fmt.Errorf("theme file is too large (%d bytes; max %d bytes)", info.Size(), maxThemeSize)
	}

	// Read file contents.
	data, err := io.ReadAll(io.LimitReader(f, int64(maxThemeSize)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}
	if int64(len(data)) > int64(maxThemeSize) {
		return nil, fmt.Errorf("theme file exceeds size limit during read (%d bytes; max %d)", len(data), maxThemeSize)
	}

	// Probe the file for its name so the right built-in can serve as the base.
	probe := &Theme{}
	if err := yaml.Unmarshal(data, probe); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}

	// Auto-derive name from filename before validation (so nameless
	// YAML files don't fail the required-name check).
	name := probe.Name
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	// A custom theme overlays a built-in rather than replacing it. Themes
	// carry a dozen colors, a font stack and four margins; a file listing only
	// the handful of fields Validate insists on used to produce empty CSS
	// values (link color collapsing into body text) and 0mm page margins.
	theme := builtinBaseFor(name)
	if err := yaml.Unmarshal(data, theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}
	theme.Name = name

	if err := theme.Validate(); err != nil {
		return nil, fmt.Errorf("theme validation failed: %w", err)
	}
	tm.themes[theme.Name] = theme

	return theme, nil
}

// builtinBaseFor returns a fresh built-in theme to overlay a custom theme
// onto: the same-named built-in when the file customizes one, otherwise the
// default. It deliberately ignores previously loaded custom themes so that
// loading order cannot change what a theme file means.
func builtinBaseFor(name string) *Theme {
	switch name {
	case "elegant":
		return builtinElegant()
	case "minimal":
		return builtinMinimal()
	default:
		return builtinTechnical()
	}
}

// List returns the names of all available themes in sorted order.
func (tm *ThemeManager) List() []string {
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Validate checks theme fields for correctness.
func (t *Theme) Validate() error {
	if t.Name == "" {
		return errors.New("theme name must not be empty")
	}

	if t.PageSize == "" {
		return errors.New("page size must not be empty")
	}

	if t.FontSize <= 0 {
		return errors.New("font size must be greater than 0")
	}

	if t.LineHeight <= 0 {
		return errors.New("line height must be greater than 0")
	}

	if t.Colors.Text == "" {
		return errors.New("text color must not be empty")
	}

	if t.Colors.Background == "" {
		return errors.New("background color must not be empty")
	}

	// Reject color/font values containing CSS injection characters.
	for _, cv := range []struct{ name, value string }{
		{"text color", t.Colors.Text},
		{"background color", t.Colors.Background},
		{"heading color", t.Colors.Heading},
		{"link color", t.Colors.Link},
		{"code background", t.Colors.CodeBg},
		{"code text", t.Colors.CodeText},
		{"accent color", t.Colors.Accent},
		{"border color", t.Colors.Border},
		{"font family", t.FontFamily},
	} {
		if cv.value != "" && !cssValueSafe.MatchString(cv.value) {
			return fmt.Errorf("%s contains unsafe characters", cv.name)
		}
	}

	return nil
}

// quoteFontFamily ensures each font name containing spaces is wrapped in quotes.
// Names that are already quoted (single or double) or CSS generic families are left unchanged.
func quoteFontFamily(family string) string {
	genericFamilies := map[string]bool{
		"serif": true, "sans-serif": true, "monospace": true,
		"cursive": true, "fantasy": true, "system-ui": true,
		"ui-serif": true, "ui-sans-serif": true, "ui-monospace": true,
		"ui-rounded": true, "math": true, "emoji": true, "fangsong": true,
	}

	parts := strings.Split(family, ",")
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		// Already quoted.
		if (strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) ||
			(strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) {
			parts[i] = trimmed
			continue
		}
		// CSS generic family or vendor-prefixed value (starts with -).
		if genericFamilies[strings.ToLower(trimmed)] || strings.HasPrefix(trimmed, "-") {
			parts[i] = trimmed
			continue
		}
		// Strip stray single quotes to avoid CSS syntax errors, then
		// wrap in single quotes if the name contains spaces.
		cleaned := strings.ReplaceAll(trimmed, "'", "")
		if strings.Contains(cleaned, " ") {
			parts[i] = "'" + cleaned + "'"
		} else {
			parts[i] = cleaned
		}
	}
	return strings.Join(parts, ", ")
}

// cssVar renders a var() reference with a literal fallback value so the rule
// still applies in consumers that drop the :root custom properties (e.g.
// older EPUB reading systems). An empty fallback yields a plain var().
func cssVar(name, fallback string) string {
	if fallback == "" {
		return fmt.Sprintf("var(%s)", name)
	}
	return fmt.Sprintf("var(%s, %s)", name, fallback)
}

// writeDecl emits "prop: var(--name, value);" but omits the declaration
// entirely when the theme has no value, since var() against an undefined
// custom property is invalid at computed-value time and would inherit
// something arbitrary instead.
func writeDecl(css *strings.Builder, prop, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(css, "  %s: %s;\n", prop, cssVar(name, value))
}

// ToCSS converts theme settings to CSS code. Every var() usage carries a
// literal fallback derived from the theme values, so the styling survives
// engines without custom-property support.
func (t *Theme) ToCSS() string {
	var css strings.Builder

	fontFamily := quoteFontFamily(t.FontFamily)
	monoFamily := quoteFontFamily(defaultCJKMonoFontFamily)
	fontSize := t.ResolvedFontSize()
	lineHeight := fmt.Sprintf("%.2f", t.LineHeight)
	marginTop := fmt.Sprintf("%.2fmm", t.Margins.Top)
	marginBottom := fmt.Sprintf("%.2fmm", t.Margins.Bottom)
	marginLeft := fmt.Sprintf("%.2fmm", t.Margins.Left)
	marginRight := fmt.Sprintf("%.2fmm", t.Margins.Right)

	// A theme value that is empty must not become "--color-link: ;": that is a
	// parse error, and the declarations referring to it then resolve to
	// nothing, silently collapsing links into body text. Skip the property
	// instead, so the consuming stylesheet's own default survives.
	writeVar := func(name, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(&css, "  %s: %s;\n", name, value)
	}
	// Margins only mean something on paper, and all-zero margins are what a
	// theme file that omits the block yields — treat that as "unset" rather
	// than as an edge-to-edge page.
	hasMargins := t.Margins.Top != 0 || t.Margins.Bottom != 0 || t.Margins.Left != 0 || t.Margins.Right != 0

	css.WriteString("/* Auto-generated theme CSS */\n")
	css.WriteString(":root {\n")
	writeVar("--font-family", fontFamily)
	writeVar("--font-family-mono", monoFamily)
	writeVar("--font-size", fontSize)
	writeVar("--line-height", lineHeight)
	writeVar("--color-text", t.Colors.Text)
	writeVar("--color-background", t.Colors.Background)
	writeVar("--color-heading", t.Colors.Heading)
	writeVar("--color-link", t.Colors.Link)
	writeVar("--color-code-bg", t.Colors.CodeBg)
	writeVar("--color-code-text", t.Colors.CodeText)
	writeVar("--color-accent", t.Colors.Accent)
	writeVar("--color-border", t.Colors.Border)
	if hasMargins {
		writeVar("--margin-top", marginTop)
		writeVar("--margin-bottom", marginBottom)
		writeVar("--margin-left", marginLeft)
		writeVar("--margin-right", marginRight)
	}
	css.WriteString("}\n\n")

	// Base styles.
	css.WriteString("body {\n")
	writeDecl(&css, "font-family", "--font-family", fontFamily)
	writeDecl(&css, "font-size", "--font-size", fontSize)
	writeDecl(&css, "line-height", "--line-height", lineHeight)
	writeDecl(&css, "color", "--color-text", t.Colors.Text)
	writeDecl(&css, "background-color", "--color-background", t.Colors.Background)
	if hasMargins {
		fmt.Fprintf(&css, "  margin: %s %s %s %s;\n",
			cssVar("--margin-top", marginTop),
			cssVar("--margin-right", marginRight),
			cssVar("--margin-bottom", marginBottom),
			cssVar("--margin-left", marginLeft))
	}
	css.WriteString("}\n\n")

	// Heading styles. Only color/weight/rhythm live here; per-format sizing is
	// owned by each renderer so the type scale can differ between PDF and web.
	css.WriteString("h1, h2, h3, h4, h5, h6 {\n")
	writeDecl(&css, "color", "--color-heading", t.Colors.Heading)
	css.WriteString("  font-weight: 600;\n")
	css.WriteString("  line-height: 1.35;\n")
	css.WriteString("}\n\n")

	// Link styles. No underline by default (cleaner in headings, TOC, and body);
	// underline appears on hover for on-screen affordance.
	css.WriteString("a {\n")
	writeDecl(&css, "color", "--color-link", t.Colors.Link)
	css.WriteString("  text-decoration: none;\n")
	css.WriteString("}\n\n")

	css.WriteString("a:hover {\n")
	css.WriteString("  text-decoration: underline;\n")
	css.WriteString("}\n\n")

	// Code styles. Block code has no background here (the syntax highlighter
	// supplies one); inline code gets a subtle tinted chip for legibility.
	css.WriteString("code, pre {\n")
	css.WriteString("  background: none;\n")
	writeDecl(&css, "color", "--color-code-text", t.Colors.CodeText)
	writeDecl(&css, "font-family", "--font-family-mono", monoFamily)
	css.WriteString("}\n\n")

	css.WriteString(":not(pre) > code {\n")
	writeDecl(&css, "background", "--color-code-bg", t.Colors.CodeBg)
	css.WriteString("  padding: 0.12em 0.36em;\n")
	css.WriteString("  border-radius: 4px;\n")
	css.WriteString("  font-size: 0.88em;\n")
	css.WriteString("}\n\n")

	css.WriteString("pre {\n")
	css.WriteString("  padding: 0.9em 1.1em;\n")
	css.WriteString("  font-size: 0.82em;\n")
	css.WriteString("  line-height: 1.55;\n")
	if t.Colors.Border != "" {
		fmt.Fprintf(&css, "  border: 1px solid %s;\n", cssVar("--color-border", t.Colors.Border))
	}
	css.WriteString("  border-radius: 6px;\n")
	css.WriteString("  overflow-x: auto;\n")
	css.WriteString("  white-space: pre-wrap;\n")
	css.WriteString("  overflow-wrap: anywhere;\n")
	css.WriteString("  word-break: break-all;\n")
	css.WriteString("}\n\n")

	// Blockquote styles: an accent rule with muted text and no heavy fill.
	css.WriteString("blockquote {\n")
	if t.Colors.Accent != "" {
		fmt.Fprintf(&css, "  border-left: 3px solid %s;\n", cssVar("--color-accent", t.Colors.Accent))
	}
	css.WriteString("  margin: 1.2em 0;\n")
	css.WriteString("  padding: 0.2em 0 0.2em 1.1em;\n")
	writeDecl(&css, "color", "--color-text", t.Colors.Text)
	css.WriteString("  opacity: 0.78;\n")
	css.WriteString("}\n\n")

	// Table styles: full width, hairline borders, tinted header with an accent
	// underline, and subtle zebra striping.
	css.WriteString("table {\n")
	css.WriteString("  border-collapse: collapse;\n")
	css.WriteString("  width: 100%;\n")
	css.WriteString("  margin: 1.2em 0;\n")
	css.WriteString("  font-size: 0.96em;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th, table td {\n")
	if t.Colors.Border != "" {
		fmt.Fprintf(&css, "  border: 1px solid %s;\n", cssVar("--color-border", t.Colors.Border))
	}
	css.WriteString("  padding: 0.55em 0.85em;\n")
	css.WriteString("  text-align: left;\n")
	css.WriteString("  overflow-wrap: anywhere;\n")
	css.WriteString("  word-break: break-word;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th {\n")
	writeDecl(&css, "background-color", "--color-code-bg", t.Colors.CodeBg)
	writeDecl(&css, "color", "--color-heading", t.Colors.Heading)
	css.WriteString("  font-weight: 600;\n")
	if t.Colors.Accent != "" {
		fmt.Fprintf(&css, "  border-bottom: 2px solid %s;\n", cssVar("--color-accent", t.Colors.Accent))
	}
	css.WriteString("}\n\n")

	css.WriteString("table tbody tr:nth-child(even) td {\n")
	writeDecl(&css, "background-color", "--color-code-bg", t.Colors.CodeBg)
	css.WriteString("}\n\n")

	return css.String()
}

// Clone creates a deep copy of the theme.
func (t *Theme) Clone() *Theme {
	return &Theme{
		Name:           t.Name,
		PageSize:       t.PageSize,
		FontFamily:     t.FontFamily,
		FontSize:       t.FontSize,
		CodeTheme:      t.CodeTheme,
		LineHeight:     t.LineHeight,
		Colors:         t.Colors,
		Margins:        t.Margins,
		HeaderTemplate: t.HeaderTemplate,
		FooterTemplate: t.FooterTemplate,
	}
}
