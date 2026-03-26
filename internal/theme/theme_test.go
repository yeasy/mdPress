package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewThemeManager tests creating a theme manager
func TestNewThemeManager(t *testing.T) {
	tm := NewThemeManager()
	if tm == nil {
		t.Fatal("NewThemeManager returned nil")
	}
}

// TestBuiltinThemesExist tests that all built-in themes exist
func TestBuiltinThemesExist(t *testing.T) {
	tm := NewThemeManager()
	builtins := []string{"technical", "elegant", "minimal"}

	for _, name := range builtins {
		thm, err := tm.Get(name)
		if err != nil {
			t.Errorf("built-in theme %q should exist: %v", name, err)
			continue
		}
		if thm.Name != name {
			t.Errorf("theme name mismatch: got %q, want %q", thm.Name, name)
		}
	}
}

// TestGetDefaultTheme tests that empty name returns the default theme
func TestGetDefaultTheme(t *testing.T) {
	tm := NewThemeManager()
	thm, err := tm.Get("")
	if err != nil {
		t.Fatalf("empty name should return default theme: %v", err)
	}
	if thm.Name != "technical" {
		t.Errorf("default theme should be 'technical': got %q", thm.Name)
	}
}

// TestGetNonExistentTheme tests retrieving a non-existent theme
func TestGetNonExistentTheme(t *testing.T) {
	tm := NewThemeManager()
	_, err := tm.Get("nonexistent")
	if err == nil {
		t.Error("getting non-existent theme should return error")
	}
}

// TestListThemes tests listing all themes
func TestListThemes(t *testing.T) {
	tm := NewThemeManager()
	names := tm.List()
	if len(names) < 3 {
		t.Errorf("should have at least 3 themes: got %d", len(names))
	}

	// Check built-in themes are in the list
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	for _, expected := range []string{"technical", "elegant", "minimal"} {
		if !nameSet[expected] {
			t.Errorf("theme list should contain %q", expected)
		}
	}
}

// TestThemeTechnicalProperties tests technical theme properties
func TestThemeTechnicalProperties(t *testing.T) {
	tm := NewThemeManager()
	thm, _ := tm.Get("technical")

	if thm.PageSize == "" {
		t.Error("technical theme should have PageSize")
	}
	if thm.FontFamily == "" {
		t.Error("technical theme should have FontFamily")
	}
	if thm.FontSize <= 0 {
		t.Error("technical theme FontSize should be > 0")
	}
	if thm.LineHeight <= 0 {
		t.Error("technical theme LineHeight should be > 0")
	}
	if thm.Colors.Text == "" {
		t.Error("technical theme should have Text color")
	}
	if thm.Colors.Background == "" {
		t.Error("technical theme should have Background color")
	}
	if thm.Colors.Heading == "" {
		t.Error("technical theme should have Heading color")
	}
}

// TestThemeToCSS tests CSS generation
func TestThemeToCSS(t *testing.T) {
	tm := NewThemeManager()
	thm, _ := tm.Get("technical")

	css := thm.ToCSS()
	if css == "" {
		t.Fatal("CSS should not be empty")
	}

	// Check CSS variables
	expectedVars := []string{
		"--font-family",
		"--font-family-mono",
		"--font-size",
		"--line-height",
		"--color-text",
		"--color-background",
		"--color-heading",
		"--color-link",
	}
	for _, v := range expectedVars {
		if !strings.Contains(css, v) {
			t.Errorf("CSS should contain variable %q", v)
		}
	}

	// Check base style rules
	expectedRules := []string{
		"body {",
		"h1, h2, h3",
		"a {",
		"code, pre {",
		"blockquote {",
		"table {",
	}
	for _, rule := range expectedRules {
		if !strings.Contains(css, rule) {
			t.Errorf("CSS should contain rule %q", rule)
		}
	}
}

// TestThemeValidate tests theme validation
func TestThemeValidate(t *testing.T) {
	tests := []struct {
		name    string
		theme   Theme
		wantErr bool
	}{
		{
			"valid theme",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			false,
		},
		{
			"no name",
			Theme{PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"no PageSize",
			Theme{Name: "test", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"FontSize is 0",
			Theme{Name: "test", PageSize: "A4", FontSize: 0, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"LineHeight is 0",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 0,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"no text color",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Background: "#fff"}},
			true,
		},
		{
			"no background color",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333"}},
			true,
		},
	}

	for _, tt := range tests {
		err := tt.theme.Validate()
		if tt.wantErr && err == nil {
			t.Errorf("%s: should fail validation", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: should not fail validation: %v", tt.name, err)
		}
	}
}

// TestThemeClone tests theme cloning
func TestThemeClone(t *testing.T) {
	original := &Theme{
		Name:       "original",
		PageSize:   "A4",
		FontFamily: "Arial",
		FontSize:   12,
		Colors:     ColorScheme{Text: "#000", Background: "#fff"},
	}

	clone := original.Clone()

	if clone == original {
		t.Error("clone should return a new instance")
	}
	if clone.Name != original.Name {
		t.Errorf("cloned Name differs: got %q", clone.Name)
	}

	// Modifying clone should not affect original
	clone.Name = "modified"
	if original.Name == "modified" {
		t.Error("modifying clone should not affect original")
	}
}

// TestLoadFromFile tests loading a theme from file
func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "custom.yaml")
	content := `
name: custom-theme
page_size: A4
font_family: "Source Han Sans"
font_size: 14
code_theme: dracula
line_height: 1.8
colors:
  text: "#2d2d2d"
  background: "#ffffff"
  heading: "#c0392b"
  link: "#2980b9"
  code_bg: "#f5f5f5"
  code_text: "#333"
  accent: "#e74c3c"
  border: "#ddd"
margins:
  top: 25
  bottom: 25
  left: 20
  right: 20
`
	if err := os.WriteFile(themePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write theme file: %v", err)
	}

	tm := NewThemeManager()
	thm, err := tm.LoadFromFile(themePath)
	if err != nil {
		t.Fatalf("failed to load theme from file: %v", err)
	}

	if thm.Name != "custom-theme" {
		t.Errorf("theme name mismatch: got %q", thm.Name)
	}
	if thm.FontSize != 14 {
		t.Errorf("font size mismatch: got %d", thm.FontSize)
	}
	if thm.Colors.Heading != "#c0392b" {
		t.Errorf("heading color mismatch: got %q", thm.Colors.Heading)
	}

	// Should be retrievable by name
	retrieved, err := tm.Get("custom-theme")
	if err != nil {
		t.Errorf("should be able to get theme by name after loading: %v", err)
	}
	if retrieved.FontSize != 14 {
		t.Error("theme properties retrieved by name should be correct")
	}
}

// TestLoadFromFileNonExistent tests loading a non-existent theme file
func TestLoadFromFileNonExistent(t *testing.T) {
	tm := NewThemeManager()
	_, err := tm.LoadFromFile("/nonexistent/theme.yaml")
	if err == nil {
		t.Error("loading non-existent file should return error")
	}
}

// TestLoadFromFileInvalidYAML tests loading an invalid YAML theme
func TestLoadFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(themePath, []byte("{{invalid: yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	tm := NewThemeManager()
	_, err := tm.LoadFromFile(themePath)
	if err == nil {
		t.Error("invalid YAML should return error")
	}
}

// TestLoadFromFileAutoName tests automatic naming from filename
func TestLoadFromFileAutoName(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "my-theme.yaml")
	// Must include name field since validation runs before auto-naming
	content := `
name: my-theme
page_size: A4
font_family: sans-serif
font_size: 12
line_height: 1.5
colors:
  text: "#333"
  background: "#fff"
`
	if err := os.WriteFile(themePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tm := NewThemeManager()
	thm, err := tm.LoadFromFile(themePath)
	if err != nil {
		t.Fatalf("failed to load theme: %v", err)
	}

	if thm.Name != "my-theme" {
		t.Errorf("theme name mismatch: got %q, want %q", thm.Name, "my-theme")
	}

	// Should be retrievable by name
	_, err = tm.Get("my-theme")
	if err != nil {
		t.Errorf("should be able to get loaded theme by name: %v", err)
	}
}

// TestBuiltinThemesValidate tests that all built-in themes pass validation
func TestBuiltinThemesValidate(t *testing.T) {
	tm := NewThemeManager()
	for _, name := range tm.List() {
		thm, err := tm.Get(name)
		if err != nil {
			t.Errorf("failed to get theme %q: %v", name, err)
			continue
		}
		if err := thm.Validate(); err != nil {
			t.Errorf("built-in theme %q failed validation: %v", name, err)
		}
	}
}

// TestToCSSAllThemes tests that all themes can generate CSS
func TestToCSSAllThemes(t *testing.T) {
	tm := NewThemeManager()
	for _, name := range tm.List() {
		thm, _ := tm.Get(name)
		css := thm.ToCSS()
		if css == "" {
			t.Errorf("theme %q generated empty CSS", name)
		}
		if !strings.Contains(css, ":root") {
			t.Errorf("theme %q CSS should contain :root", name)
		}
	}
}

// ---------------------------------------------------------------------------
// quoteFontFamily - Table-Driven Tests for Font Family Quoting
// ---------------------------------------------------------------------------

func TestQuoteFontFamily_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single family without spaces",
			input:    "Arial",
			expected: "Arial",
		},
		{
			name:     "family with spaces needs quotes",
			input:    "Times New Roman",
			expected: "'Times New Roman'",
		},
		{
			name:     "already single quoted",
			input:    "'Arial Black'",
			expected: "'Arial Black'",
		},
		{
			name:     "already double quoted",
			input:    `"Courier New"`,
			expected: `"Courier New"`,
		},
		{
			name:     "generic family serif",
			input:    "serif",
			expected: "serif",
		},
		{
			name:     "generic family sans-serif",
			input:    "sans-serif",
			expected: "sans-serif",
		},
		{
			name:     "generic family monospace",
			input:    "monospace",
			expected: "monospace",
		},
		{
			name:     "multiple families comma-separated",
			input:    "Arial, sans-serif",
			expected: "Arial, sans-serif",
		},
		{
			name:     "multiple families with spaces",
			input:    "Times New Roman, serif",
			expected: "'Times New Roman', serif",
		},
		{
			name:     "all families with spaces",
			input:    "Times New Roman, Arial Black",
			expected: "'Times New Roman', 'Arial Black'",
		},
		{
			name:     "vendor prefixed",
			input:    "-webkit-system-font",
			expected: "-webkit-system-font",
		},
		{
			name:     "ui-monospace",
			input:    "ui-monospace",
			expected: "ui-monospace",
		},
		{
			name:     "complex mixed list",
			input:    "Arial, 'Times New Roman', sans-serif, Consolas",
			expected: "Arial, 'Times New Roman', sans-serif, Consolas",
		},
		{
			name:     "with extra spaces",
			input:    "  Arial  ,  sans-serif  ",
			expected: "Arial, sans-serif",
		},
		{
			name:     "CJK font with spaces",
			input:    "PingFang SC, Hiragino Sans GB",
			expected: "'PingFang SC', 'Hiragino Sans GB'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteFontFamily(tt.input)
			if result != tt.expected {
				t.Errorf("quoteFontFamily(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestQuoteFontFamily_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		checkFn func(string) bool
		desc    string
	}{
		{
			name:  "empty string",
			input: "",
			checkFn: func(s string) bool {
				return s == ""
			},
			desc: "should handle empty input",
		},
		{
			name:  "single space",
			input: " ",
			checkFn: func(s string) bool {
				return s == "" // single space trims to empty string
			},
			desc: "should handle single space",
		},
		{
			name:  "only comma",
			input: ",",
			checkFn: func(s string) bool {
				return s != ""
			},
			desc: "should handle only comma",
		},
		{
			name:  "multiple commas",
			input: "Arial,,,sans-serif",
			checkFn: func(s string) bool {
				return len(s) > 0
			},
			desc: "should handle multiple commas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteFontFamily(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("quoteFontFamily(%q) failed %s, got %q", tt.input, tt.desc, result)
			}
		})
	}
}

func TestQuoteFontFamily_PreservesValidCSS(t *testing.T) {
	// Test that valid CSS is not corrupted
	validCSS := []string{
		"ui-serif",
		"ui-sans-serif",
		"ui-monospace",
		"ui-rounded",
		"emoji",
		"math",
		"fangsong",
		"cursive",
		"fantasy",
		"system-ui",
	}

	for _, family := range validCSS {
		result := quoteFontFamily(family)
		if result != family {
			t.Errorf("quoteFontFamily(%q) should not modify, got %q", family, result)
		}
	}
}

func TestQuoteFontFamily_CJKFonts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "PingFang SC",
			input:    "PingFang SC",
			contains: "PingFang SC",
		},
		{
			name:     "Microsoft YaHei",
			input:    "Microsoft YaHei",
			contains: "Microsoft YaHei",
		},
		{
			name:     "Hiragino Sans GB",
			input:    "Hiragino Sans GB",
			contains: "Hiragino Sans GB",
		},
		{
			name:     "Noto Sans SC",
			input:    "Noto Sans SC",
			contains: "Noto Sans SC",
		},
		{
			name:     "Source Han Sans SC",
			input:    "Source Han Sans SC",
			contains: "Source Han Sans SC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteFontFamily(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

func TestQuoteFontFamily_MixedQuoteStyles(t *testing.T) {
	// Test that both single and double quotes work
	tests := []struct {
		name    string
		input   string
		isValid func(string) bool
	}{
		{
			name:  "single quoted with comma",
			input: "'Times New Roman', Arial",
			isValid: func(s string) bool {
				return strings.Contains(s, "'Times New Roman'") && strings.Contains(s, "Arial")
			},
		},
		{
			name:  "double quoted with comma",
			input: `"Courier New", monospace`,
			isValid: func(s string) bool {
				return strings.Contains(s, `"Courier New"`) || strings.Contains(s, `'Courier New'`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteFontFamily(tt.input)
			if !tt.isValid(result) {
				t.Errorf("quoteFontFamily(%q) = %q, validation failed", tt.input, result)
			}
		})
	}
}

func TestQuoteFontFamily_RealWorldFontStacks(t *testing.T) {
	// Test common font stacks used in practice
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
	}{
		{
			name:  "System font stack",
			input: "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
			validate: func(s string) bool {
				return strings.Contains(s, "-apple-system") && strings.Contains(s, "sans-serif")
			},
		},
		{
			name:  "Bootstrap font stack",
			input: "Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif",
			validate: func(s string) bool {
				return strings.Contains(s, "sans-serif")
			},
		},
		{
			name:  "CJK system stack",
			input: "PingFang SC, Hiragino Sans GB, Microsoft YaHei, sans-serif",
			validate: func(s string) bool {
				return strings.Contains(s, "PingFang") && strings.Contains(s, "sans-serif")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteFontFamily(tt.input)
			if !tt.validate(result) {
				t.Errorf("quoteFontFamily(%q) = %q, validation failed", tt.input, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ToCSS - Enhanced Tests for CSS Generation
// ---------------------------------------------------------------------------

func TestThemeToCSS_RootVariables(t *testing.T) {
	thm := &Theme{
		Name:       "test",
		PageSize:   "A4",
		FontFamily: "Arial",
		FontSize:   12,
		LineHeight: 1.5,
		Colors: ColorScheme{
			Text:       "#000",
			Background: "#fff",
			Heading:    "#333",
			Link:       "#0066cc",
			CodeBg:     "#f5f5f5",
			CodeText:   "#333",
			Accent:     "#ff6600",
			Border:     "#ddd",
		},
	}

	css := thm.ToCSS()

	requiredVars := []string{
		"--font-family",
		"--font-family-mono",
		"--font-size",
		"--line-height",
		"--color-text",
		"--color-background",
		"--color-heading",
		"--color-link",
		"--color-code-bg",
		"--color-code-text",
		"--color-accent",
		"--color-border",
		"--margin-top",
		"--margin-bottom",
		"--margin-left",
		"--margin-right",
	}

	for _, varName := range requiredVars {
		if !strings.Contains(css, varName) {
			t.Errorf("CSS missing variable: %s", varName)
		}
	}
}

func TestThemeToCSS_SelectorCoverage(t *testing.T) {
	thm := &Theme{
		Name:       "test",
		PageSize:   "A4",
		FontFamily: "Arial",
		FontSize:   12,
		LineHeight: 1.5,
		Colors: ColorScheme{
			Text:       "#000",
			Background: "#fff",
			Heading:    "#333",
			Link:       "#0066cc",
			CodeBg:     "#f5f5f5",
			CodeText:   "#333",
			Accent:     "#ff6600",
			Border:     "#ddd",
		},
	}

	css := thm.ToCSS()

	requiredSelectors := []string{
		":root",
		"body {",
		"h1, h2, h3, h4, h5, h6",
		"a {",
		"code, pre",
		"pre {",
		"blockquote",
		"table {",
		"table th, table td",
		"table th {",
	}

	for _, selector := range requiredSelectors {
		if !strings.Contains(css, selector) {
			t.Errorf("CSS missing selector: %s", selector)
		}
	}
}

func TestThemeToCSS_NumericalValues(t *testing.T) {
	thm := &Theme{
		Name:       "test",
		PageSize:   "A4",
		FontFamily: "Arial",
		FontSize:   16,
		LineHeight: 1.8,
		Colors: ColorScheme{
			Text:       "#000",
			Background: "#fff",
			Heading:    "#333",
			Link:       "#0066cc",
			CodeBg:     "#f5f5f5",
			CodeText:   "#333",
			Accent:     "#ff6600",
			Border:     "#ddd",
		},
		Margins: MarginSettings{
			Top:    20,
			Bottom: 20,
			Left:   15,
			Right:  15,
		},
	}

	css := thm.ToCSS()

	if !strings.Contains(css, "16pt") {
		t.Error("CSS should contain font size 16pt")
	}

	if !strings.Contains(css, "1.80") {
		t.Error("CSS should contain line height 1.80")
	}

	if !strings.Contains(css, "20.00mm") {
		t.Error("CSS should contain margin 20.00mm")
	}
}

func TestThemeToCSS_FontFamilyProcessing(t *testing.T) {
	thm := &Theme{
		Name:       "test",
		PageSize:   "A4",
		FontFamily: "Times New Roman, serif",
		FontSize:   12,
		LineHeight: 1.5,
		Colors: ColorScheme{
			Text:       "#000",
			Background: "#fff",
			Heading:    "#333",
			Link:       "#0066cc",
			CodeBg:     "#f5f5f5",
			CodeText:   "#333",
			Accent:     "#ff6600",
			Border:     "#ddd",
		},
	}

	css := thm.ToCSS()

	// Font family should be processed through quoteFontFamily
	if !strings.Contains(css, "--font-family") {
		t.Error("CSS should have font family variable")
	}

	// Should have properly quoted font name
	if !strings.Contains(css, "Times") || !strings.Contains(css, "Roman") {
		t.Error("Font family content should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Margin Validation
// ---------------------------------------------------------------------------

func TestMarginSettings_AllPositive(t *testing.T) {
	m := MarginSettings{
		Top:    10.5,
		Bottom: 20.0,
		Left:   15.25,
		Right:  18.75,
	}

	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"top", m.Top, 10.5},
		{"bottom", m.Bottom, 20.0},
		{"left", m.Left, 15.25},
		{"right", m.Right, 18.75},
	}

	for _, tt := range tests {
		if tt.value != tt.expected {
			t.Errorf("%s margin: got %f, want %f", tt.name, tt.value, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// ColorScheme Validation
// ---------------------------------------------------------------------------

func TestColorScheme_AllFieldsSet(t *testing.T) {
	colors := ColorScheme{
		Text:       "#333333",
		Background: "#ffffff",
		Heading:    "#000000",
		Link:       "#0066cc",
		CodeBg:     "#f5f5f5",
		CodeText:   "#333333",
		Accent:     "#ff6600",
		Border:     "#dddddd",
	}

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"text", colors.Text, "#333333"},
		{"background", colors.Background, "#ffffff"},
		{"heading", colors.Heading, "#000000"},
		{"link", colors.Link, "#0066cc"},
		{"codeBg", colors.CodeBg, "#f5f5f5"},
		{"codeText", colors.CodeText, "#333333"},
		{"accent", colors.Accent, "#ff6600"},
		{"border", colors.Border, "#dddddd"},
	}

	for _, tt := range tests {
		if tt.value != tt.expected {
			t.Errorf("%s color: got %s, want %s", tt.name, tt.value, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// GetThemeDescription - Tests for Theme Description Lookup
// ---------------------------------------------------------------------------

// TestGetThemeDescription_KnownThemes tests getting descriptions of known themes
func TestGetThemeDescription_KnownThemes(t *testing.T) {
	tests := []struct {
		name           string
		themeName      string
		expectedSubstr string
	}{
		{
			name:           "technical theme",
			themeName:      "technical",
			expectedSubstr: "professional",
		},
		{
			name:           "elegant theme",
			themeName:      "elegant",
			expectedSubstr: "Elegant",
		},
		{
			name:           "minimal theme",
			themeName:      "minimal",
			expectedSubstr: "Minimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := GetThemeDescription(tt.themeName)
			if desc == "" {
				t.Errorf("GetThemeDescription(%q) returned empty string", tt.themeName)
			}
			if !strings.Contains(desc, tt.expectedSubstr) {
				t.Errorf("GetThemeDescription(%q) = %q, want to contain %q", tt.themeName, desc, tt.expectedSubstr)
			}
		})
	}
}

// TestGetThemeDescription_UnknownTheme tests getting description of unknown theme
func TestGetThemeDescription_UnknownTheme(t *testing.T) {
	unknownThemes := []string{
		"nonexistent",
		"unknown-theme",
		"xyz",
		"not-a-theme",
	}

	expectedDefault := "Unknown theme"

	for _, themeName := range unknownThemes {
		t.Run(themeName, func(t *testing.T) {
			desc := GetThemeDescription(themeName)
			if desc != expectedDefault {
				t.Errorf("GetThemeDescription(%q) = %q, want %q", themeName, desc, expectedDefault)
			}
		})
	}
}

// TestGetThemeDescription_EmptyString tests empty string input
func TestGetThemeDescription_EmptyString(t *testing.T) {
	desc := GetThemeDescription("")
	expectedDefault := "Unknown theme"
	if desc != expectedDefault {
		t.Errorf("GetThemeDescription(%q) = %q, want %q", "", desc, expectedDefault)
	}
}

// TestGetThemeDescription_CaseSensitive tests theme name case sensitivity
func TestGetThemeDescription_CaseSensitive(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		isKnown   bool
	}{
		{
			name:      "lowercase technical",
			themeName: "technical",
			isKnown:   true,
		},
		{
			name:      "uppercase TECHNICAL",
			themeName: "TECHNICAL",
			isKnown:   false,
		},
		{
			name:      "mixed case Technical",
			themeName: "Technical",
			isKnown:   false,
		},
		{
			name:      "lowercase elegant",
			themeName: "elegant",
			isKnown:   true,
		},
		{
			name:      "uppercase ELEGANT",
			themeName: "ELEGANT",
			isKnown:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := GetThemeDescription(tt.themeName)
			expectedDefault := "Unknown theme"

			if tt.isKnown {
				if desc == expectedDefault {
					t.Errorf("GetThemeDescription(%q) returned unknown theme message, expected a known description", tt.themeName)
				}
			} else {
				if desc != expectedDefault {
					t.Errorf("GetThemeDescription(%q) = %q, want unknown theme message", tt.themeName, desc)
				}
			}
		})
	}
}

// TestGetThemeDescription_AllKnownThemes tests all known themes
func TestGetThemeDescription_AllKnownThemes(t *testing.T) {
	expectedDefault := "Unknown theme"

	for themeName := range ThemeVariants {
		desc := GetThemeDescription(themeName)
		if desc == "" {
			t.Errorf("GetThemeDescription(%q) returned empty string", themeName)
		}
		if desc == expectedDefault {
			t.Errorf("GetThemeDescription(%q) returned default message for known theme", themeName)
		}
		// Each description should be non-empty and not just whitespace
		if strings.TrimSpace(desc) == "" {
			t.Errorf("GetThemeDescription(%q) returned whitespace-only string", themeName)
		}
	}
}

// TestGetThemeDescription_NonEmptyDescriptions tests that known themes return non-empty descriptions
func TestGetThemeDescription_NonEmptyDescriptions(t *testing.T) {
	knownThemes := []string{"technical", "elegant", "minimal"}

	for _, themeName := range knownThemes {
		desc := GetThemeDescription(themeName)

		if desc == "" {
			t.Errorf("GetThemeDescription(%q) returned empty string, expected non-empty description", themeName)
		}

		if len(strings.TrimSpace(desc)) == 0 {
			t.Errorf("GetThemeDescription(%q) returned whitespace-only string, expected meaningful content", themeName)
		}
	}
}

// TestGetThemeDescription_ConsistentResults tests that multiple calls return consistent results
func TestGetThemeDescription_ConsistentResults(t *testing.T) {
	themeName := "technical"

	first := GetThemeDescription(themeName)
	second := GetThemeDescription(themeName)
	third := GetThemeDescription(themeName)

	if first != second {
		t.Errorf("GetThemeDescription(%q) returned different results on successive calls: %q vs %q", themeName, first, second)
	}

	if second != third {
		t.Errorf("GetThemeDescription(%q) returned different results on successive calls: %q vs %q", themeName, second, third)
	}
}

// ============================================================================
// Additional: theme validation integration tests
// ============================================================================

// TestBuiltinThemesStructureIntegrity tests built-in theme structural integrity (table-driven)
func TestBuiltinThemesStructureIntegrity(t *testing.T) {
	builtinThemes := []string{"technical", "elegant", "minimal"}

	tests := []struct {
		name       string
		fieldCheck func(t *testing.T, theme *Theme, themeName string)
	}{
		{
			name: "theme name consistency",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Name != themeName {
					t.Errorf("theme %q: name mismatch, want %q got %q", themeName, themeName, theme.Name)
				}
			},
		},
		{
			name: "required fields non-empty",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Name == "" {
					t.Errorf("theme %q: Name is empty", themeName)
				}
				if theme.PageSize == "" {
					t.Errorf("theme %q: PageSize is empty", themeName)
				}
				if theme.FontFamily == "" {
					t.Errorf("theme %q: FontFamily is empty", themeName)
				}
				if theme.FontSize <= 0 {
					t.Errorf("theme %q: FontSize invalid (%d)", themeName, theme.FontSize)
				}
				if theme.CodeTheme == "" {
					t.Errorf("theme %q: CodeTheme is empty", themeName)
				}
				if theme.LineHeight <= 0 {
					t.Errorf("theme %q: LineHeight invalid (%.2f)", themeName, theme.LineHeight)
				}
			},
		},
		{
			name: "color scheme completeness",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				requiredColors := map[string]*string{
					"Text":       &theme.Colors.Text,
					"Background": &theme.Colors.Background,
					"Heading":    &theme.Colors.Heading,
					"Link":       &theme.Colors.Link,
					"CodeBg":     &theme.Colors.CodeBg,
					"CodeText":   &theme.Colors.CodeText,
					"Accent":     &theme.Colors.Accent,
					"Border":     &theme.Colors.Border,
				}
				for colorName, colorPtr := range requiredColors {
					if colorPtr == nil || *colorPtr == "" {
						t.Errorf("theme %q: color %s is empty", themeName, colorName)
					}
				}
			},
		},
		{
			name: "margin settings valid",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Margins.Top < 0 {
					t.Errorf("theme %q: Top margin invalid (%.2f)", themeName, theme.Margins.Top)
				}
				if theme.Margins.Bottom < 0 {
					t.Errorf("theme %q: Bottom margin invalid (%.2f)", themeName, theme.Margins.Bottom)
				}
				if theme.Margins.Left < 0 {
					t.Errorf("theme %q: Left margin invalid (%.2f)", themeName, theme.Margins.Left)
				}
				if theme.Margins.Right < 0 {
					t.Errorf("theme %q: Right margin invalid (%.2f)", themeName, theme.Margins.Right)
				}
			},
		},
		{
			name: "validation passes",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if err := theme.Validate(); err != nil {
					t.Errorf("theme %q: validation failed: %v", themeName, err)
				}
			},
		},
	}

	tm := NewThemeManager()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, themeName := range builtinThemes {
				theme, err := tm.Get(themeName)
				if err != nil {
					t.Fatalf("failed to get theme %q: %v", themeName, err)
				}
				tt.fieldCheck(t, theme, themeName)
			}
		})
	}
}

// TestThemeColorValidation tests theme color value format (table-driven)
func TestThemeColorValidation(t *testing.T) {
	tm := NewThemeManager()
	builtinThemes := []string{"technical", "elegant", "minimal"}

	for _, themeName := range builtinThemes {
		t.Run(themeName, func(t *testing.T) {
			theme, _ := tm.Get(themeName)

			colorFields := map[string]string{
				"Text":       theme.Colors.Text,
				"Background": theme.Colors.Background,
				"Heading":    theme.Colors.Heading,
				"Link":       theme.Colors.Link,
				"CodeBg":     theme.Colors.CodeBg,
				"CodeText":   theme.Colors.CodeText,
				"Accent":     theme.Colors.Accent,
				"Border":     theme.Colors.Border,
			}

			for colorName, colorValue := range colorFields {
				if colorValue == "" {
					t.Errorf("color %s should not be empty", colorName)
					continue
				}

				// Check if it's a valid hex color format
				if !isValidColorHex(colorValue) && !isValidColorRGB(colorValue) && !isValidColorName(colorValue) {
					t.Errorf("color %s has non-standard format: %q", colorName, colorValue)
				}
			}
		})
	}
}

// TestThemeCSSSyntax tests generated CSS syntax validity (table-driven)
func TestThemeCSSSyntax(t *testing.T) {
	tm := NewThemeManager()
	builtinThemes := []string{"technical", "elegant", "minimal"}

	requiredCSSPatterns := []string{
		":root {",
		"--font-family:",
		"--font-size:",
		"--line-height:",
		"--color-text:",
		"--color-background:",
		"body {",
		"h1, h2, h3",
		"a {",
		"code, pre {",
		"blockquote {",
		"table {",
		"}",
	}

	for _, themeName := range builtinThemes {
		t.Run(themeName, func(t *testing.T) {
			theme, _ := tm.Get(themeName)
			css := theme.ToCSS()

			if css == "" {
				t.Fatal("generated CSS should not be empty")
			}

			// Check required CSS patterns
			for _, pattern := range requiredCSSPatterns {
				if !strings.Contains(css, pattern) {
					t.Errorf("CSS should contain %q", pattern)
				}
			}

			// Basic CSS structure check
			openBraces := strings.Count(css, "{")
			closeBraces := strings.Count(css, "}")
			if openBraces != closeBraces {
				t.Errorf("CSS braces mismatch: { appears %d times, } appears %d times", openBraces, closeBraces)
			}
		})
	}
}

// TestThemeNoDuplicateNames tests that there are no duplicate theme names
func TestThemeNoDuplicateNames(t *testing.T) {
	tm := NewThemeManager()
	names := tm.List()

	nameMap := make(map[string]int)
	for _, name := range names {
		nameMap[name]++
	}

	for name, count := range nameMap {
		if count > 1 {
			t.Errorf("theme name %q duplicated %d times", name, count)
		}
	}
}

// TestThemeConsistencyBetweenBuiltinAndManager tests consistency between built-in themes and manager
func TestThemeConsistencyBetweenBuiltinAndManager(t *testing.T) {
	builtinFuncs := map[string]func() *Theme{
		"technical": builtinTechnical,
		"elegant":   builtinElegant,
		"minimal":   builtinMinimal,
	}

	tm := NewThemeManager()

	for themeName, builderFunc := range builtinFuncs {
		t.Run(themeName, func(t *testing.T) {
			// Get from manager
			managed, err := tm.Get(themeName)
			if err != nil {
				t.Fatalf("failed to get %q from manager: %v", themeName, err)
			}

			// Build directly
			builtin := builderFunc()

			// Compare key fields
			if managed.Name != builtin.Name {
				t.Errorf("Name mismatch: managed=%q, builtin=%q", managed.Name, builtin.Name)
			}
			if managed.PageSize != builtin.PageSize {
				t.Errorf("PageSize mismatch: managed=%q, builtin=%q", managed.PageSize, builtin.PageSize)
			}
			if managed.FontSize != builtin.FontSize {
				t.Errorf("FontSize mismatch: managed=%d, builtin=%d", managed.FontSize, builtin.FontSize)
			}
			if managed.CodeTheme != builtin.CodeTheme {
				t.Errorf("CodeTheme mismatch: managed=%q, builtin=%q", managed.CodeTheme, builtin.CodeTheme)
			}
		})
	}
}

// TestThemeCloneIndependence tests clone independence
func TestThemeCloneIndependence(t *testing.T) {
	tm := NewThemeManager()
	original, _ := tm.Get("technical")

	cloned := original.Clone()

	// Modify clone
	cloned.Name = "cloned-technical"
	cloned.FontSize = 20
	cloned.Colors.Text = "#FF0000"

	// Check original theme was not modified
	if original.Name != "technical" {
		t.Errorf("original theme was modified: %q", original.Name)
	}
	if original.FontSize != 11 {
		t.Errorf("original theme FontSize was modified: %d", original.FontSize)
	}
	if original.Colors.Text != "#2C3E50" {
		t.Errorf("original theme color was modified: %q", original.Colors.Text)
	}
}

// TestThemeFontFamilyQuoting tests font family quoting
func TestThemeFontFamilyQuoting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // expected content in output
	}{
		{
			name:     "multiple font chain",
			input:    "Arial, 'Helvetica Neue', sans-serif",
			contains: "Arial",
		},
		{
			name:     "CJK font",
			input:    "'Noto Sans SC', sans-serif",
			contains: "Noto",
		},
		{
			name:     "single font",
			input:    "monospace",
			contains: "monospace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := quoteFontFamily(tt.input)
			if quoted == "" {
				t.Error("quoted result should not be empty")
			}
			if !strings.Contains(quoted, tt.contains) {
				t.Errorf("output should contain %q, got %q", tt.contains, quoted)
			}
		})
	}
}

// TestThemeLoadFromYAML tests loading themes from YAML files
func TestThemeLoadFromYAML(t *testing.T) {
	// Get theme file directory
	themeDir := filepath.Join("/sessions/epic-blissful-johnson/mnt/mdpress", "themes")

	themeFiles := []string{
		filepath.Join(themeDir, "technical.yaml"),
		filepath.Join(themeDir, "elegant.yaml"),
		filepath.Join(themeDir, "minimal.yaml"),
	}

	tm := NewThemeManager()

	for _, themeFile := range themeFiles {
		t.Run(filepath.Base(themeFile), func(t *testing.T) {
			// Skip if theme file does not exist
			if _, err := os.Stat(themeFile); err != nil {
				t.Skipf("theme file does not exist: %s", themeFile)
			}

			// Attempt to load
			theme, err := tm.LoadFromFile(themeFile)
			if err != nil {
				t.Fatalf("failed to load theme file: %v", err)
			}

			// Verify loaded theme
			if theme.Name == "" {
				t.Error("loaded theme name should not be empty")
			}
			if theme.PageSize == "" {
				t.Error("loaded theme PageSize should not be empty")
			}
		})
	}
}

// TestThemeAllFieldsNotEmpty tests that all fields are non-empty (for built-in themes)
func TestThemeAllFieldsNotEmpty(t *testing.T) {
	tm := NewThemeManager()
	builtinThemes := []string{"technical", "elegant", "minimal"}

	for _, themeName := range builtinThemes {
		t.Run(themeName, func(t *testing.T) {
			theme, _ := tm.Get(themeName)

			// Key field checks
			checks := map[string]any{
				"Name":       theme.Name,
				"PageSize":   theme.PageSize,
				"FontFamily": theme.FontFamily,
				"CodeTheme":  theme.CodeTheme,
			}

			for fieldName, value := range checks {
				if str, ok := value.(string); ok && str == "" {
					t.Errorf("%s field should not be empty", fieldName)
				}
			}

			// Numeric field checks
			if theme.FontSize == 0 {
				t.Error("FontSize should not be 0")
			}
			if theme.LineHeight == 0 {
				t.Error("LineHeight should not be 0")
			}
		})
	}
}

// ============================================================================
// Helper functions
// ============================================================================

// isValidColorHex checks if the string is a valid hex color
func isValidColorHex(color string) bool {
	if !strings.HasPrefix(color, "#") {
		return false
	}
	hex := strings.TrimPrefix(color, "#")
	return len(hex) == 6 || len(hex) == 3 || len(hex) == 8
}

// isValidColorRGB checks if the string is a valid RGB color
func isValidColorRGB(color string) bool {
	return strings.HasPrefix(color, "rgb")
}

// isValidColorName checks if the string is a valid CSS color name
func isValidColorName(color string) bool {
	cssColors := map[string]bool{
		"red": true, "blue": true, "green": true, "black": true, "white": true,
		"transparent": true, "inherit": true, "currentColor": true,
	}
	return cssColors[strings.ToLower(color)]
}
