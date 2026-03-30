package theme

import (
	"errors"
	"fmt"
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
	const maxThemeSize = 1 * 1024 * 1024 // 1 MB
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("theme file not found: %w", err)
	}
	if info.Size() > int64(maxThemeSize) {
		return nil, fmt.Errorf("theme file is too large (%d bytes; max %d bytes)", info.Size(), maxThemeSize)
	}

	// Read file contents.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	// Parse YAML.
	theme := &Theme{}
	if err := yaml.Unmarshal(data, theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme file: %w", err)
	}

	// Auto-derive name from filename before validation (so nameless
	// YAML files don't fail the required-name check).
	if theme.Name == "" {
		theme.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}

	if err := theme.Validate(); err != nil {
		return nil, fmt.Errorf("theme validation failed: %w", err)
	}
	tm.themes[theme.Name] = theme

	return theme, nil
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

// ToCSS converts theme settings to CSS code.
func (t *Theme) ToCSS() string {
	var css strings.Builder

	css.WriteString("/* Auto-generated theme CSS */\n")
	css.WriteString(":root {\n")
	fmt.Fprintf(&css, "  --font-family: %s;\n", quoteFontFamily(t.FontFamily))
	fmt.Fprintf(&css, "  --font-family-mono: %s;\n", quoteFontFamily(defaultCJKMonoFontFamily))
	fmt.Fprintf(&css, "  --font-size: %dpt;\n", t.FontSize)
	fmt.Fprintf(&css, "  --line-height: %.2f;\n", t.LineHeight)
	fmt.Fprintf(&css, "  --color-text: %s;\n", t.Colors.Text)
	fmt.Fprintf(&css, "  --color-background: %s;\n", t.Colors.Background)
	fmt.Fprintf(&css, "  --color-heading: %s;\n", t.Colors.Heading)
	fmt.Fprintf(&css, "  --color-link: %s;\n", t.Colors.Link)
	fmt.Fprintf(&css, "  --color-code-bg: %s;\n", t.Colors.CodeBg)
	fmt.Fprintf(&css, "  --color-code-text: %s;\n", t.Colors.CodeText)
	fmt.Fprintf(&css, "  --color-accent: %s;\n", t.Colors.Accent)
	fmt.Fprintf(&css, "  --color-border: %s;\n", t.Colors.Border)
	fmt.Fprintf(&css, "  --margin-top: %.2fmm;\n", t.Margins.Top)
	fmt.Fprintf(&css, "  --margin-bottom: %.2fmm;\n", t.Margins.Bottom)
	fmt.Fprintf(&css, "  --margin-left: %.2fmm;\n", t.Margins.Left)
	fmt.Fprintf(&css, "  --margin-right: %.2fmm;\n", t.Margins.Right)
	css.WriteString("}\n\n")

	// Base styles.
	css.WriteString("body {\n")
	css.WriteString("  font-family: var(--font-family);\n")
	css.WriteString("  font-size: var(--font-size);\n")
	css.WriteString("  line-height: var(--line-height);\n")
	css.WriteString("  color: var(--color-text);\n")
	css.WriteString("  background-color: var(--color-background);\n")
	css.WriteString("  margin: var(--margin-top) var(--margin-right) var(--margin-bottom) var(--margin-left);\n")
	css.WriteString("}\n\n")

	// Heading styles.
	css.WriteString("h1, h2, h3, h4, h5, h6 {\n")
	css.WriteString("  color: var(--color-heading);\n")
	css.WriteString("  font-weight: bold;\n")
	css.WriteString("  margin-top: 1em;\n")
	css.WriteString("  margin-bottom: 0.5em;\n")
	css.WriteString("}\n\n")

	// Link styles.
	css.WriteString("a {\n")
	css.WriteString("  color: var(--color-link);\n")
	css.WriteString("  text-decoration: underline;\n")
	css.WriteString("}\n\n")

	// Code styles -- no background color, professional book style.
	css.WriteString("code, pre {\n")
	css.WriteString("  background: none;\n")
	css.WriteString("  color: var(--color-code-text);\n")
	css.WriteString("  font-family: var(--font-family-mono);\n")
	css.WriteString("}\n\n")

	css.WriteString("pre {\n")
	css.WriteString("  padding: 0.8em 1em;\n")
	css.WriteString("  font-size: 0.82em;\n")
	css.WriteString("  line-height: 1.5;\n")
	css.WriteString("  border: 1px solid var(--color-border);\n")
	css.WriteString("  border-radius: 3px;\n")
	css.WriteString("  overflow-x: auto;\n")
	css.WriteString("  white-space: pre-wrap;\n")
	css.WriteString("  overflow-wrap: anywhere;\n")
	css.WriteString("  word-break: break-all;\n")
	css.WriteString("}\n\n")

	// Blockquote styles.
	css.WriteString("blockquote {\n")
	css.WriteString("  border-left: 4px solid var(--color-accent);\n")
	css.WriteString("  margin-left: 0;\n")
	css.WriteString("  padding-left: 1em;\n")
	css.WriteString("  color: var(--color-text);\n")
	css.WriteString("  opacity: 0.8;\n")
	css.WriteString("}\n\n")

	// Table styles.
	css.WriteString("table {\n")
	css.WriteString("  border-collapse: collapse;\n")
	css.WriteString("  width: 100%;\n")
	css.WriteString("  margin: 1em 0;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th, table td {\n")
	css.WriteString("  border: 1px solid var(--color-border);\n")
	css.WriteString("  padding: 0.5em;\n")
	css.WriteString("  text-align: left;\n")
	css.WriteString("  overflow-wrap: anywhere;\n")
	css.WriteString("  word-break: break-word;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th {\n")
	css.WriteString("  background-color: var(--color-code-bg);\n")
	css.WriteString("  color: var(--color-heading);\n")
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
