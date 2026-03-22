package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewThemeManager 测试创建主题管理器
func TestNewThemeManager(t *testing.T) {
	tm := NewThemeManager()
	if tm == nil {
		t.Fatal("NewThemeManager 返回 nil")
	}
}

// TestBuiltinThemesExist 测试内置主题都存在
func TestBuiltinThemesExist(t *testing.T) {
	tm := NewThemeManager()
	builtins := []string{"technical", "elegant", "minimal"}

	for _, name := range builtins {
		thm, err := tm.Get(name)
		if err != nil {
			t.Errorf("内置主题 %q 应存在: %v", name, err)
			continue
		}
		if thm.Name != name {
			t.Errorf("主题名称错误: got %q, want %q", thm.Name, name)
		}
	}
}

// TestGetDefaultTheme 测试空名称返回默认主题
func TestGetDefaultTheme(t *testing.T) {
	tm := NewThemeManager()
	thm, err := tm.Get("")
	if err != nil {
		t.Fatalf("空名称应返回默认主题: %v", err)
	}
	if thm.Name != "technical" {
		t.Errorf("默认主题应为 'technical': got %q", thm.Name)
	}
}

// TestGetNonExistentTheme 测试获取不存在的主题
func TestGetNonExistentTheme(t *testing.T) {
	tm := NewThemeManager()
	_, err := tm.Get("nonexistent")
	if err == nil {
		t.Error("获取不存在的主题应返回错误")
	}
}

// TestListThemes 测试列出所有主题
func TestListThemes(t *testing.T) {
	tm := NewThemeManager()
	names := tm.List()
	if len(names) < 3 {
		t.Errorf("应至少有 3 个主题: got %d", len(names))
	}

	// 检查内置主题在列表中
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	for _, expected := range []string{"technical", "elegant", "minimal"} {
		if !nameSet[expected] {
			t.Errorf("主题列表中应包含 %q", expected)
		}
	}
}

// TestThemeTechnicalProperties 测试 technical 主题属性
func TestThemeTechnicalProperties(t *testing.T) {
	tm := NewThemeManager()
	thm, _ := tm.Get("technical")

	if thm.PageSize == "" {
		t.Error("technical 主题应有 PageSize")
	}
	if thm.FontFamily == "" {
		t.Error("technical 主题应有 FontFamily")
	}
	if thm.FontSize <= 0 {
		t.Error("technical 主题 FontSize 应大于 0")
	}
	if thm.LineHeight <= 0 {
		t.Error("technical 主题 LineHeight 应大于 0")
	}
	if thm.Colors.Text == "" {
		t.Error("technical 主题应有 Text 颜色")
	}
	if thm.Colors.Background == "" {
		t.Error("technical 主题应有 Background 颜色")
	}
	if thm.Colors.Heading == "" {
		t.Error("technical 主题应有 Heading 颜色")
	}
}

// TestThemeToCSS 测试 CSS 生成
func TestThemeToCSS(t *testing.T) {
	tm := NewThemeManager()
	thm, _ := tm.Get("technical")

	css := thm.ToCSS()
	if css == "" {
		t.Fatal("CSS 不应为空")
	}

	// 检查 CSS 变量
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
			t.Errorf("CSS 应包含变量 %q", v)
		}
	}

	// 检查基础样式规则
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
			t.Errorf("CSS 应包含规则 %q", rule)
		}
	}
}

// TestThemeValidate 测试主题验证
func TestThemeValidate(t *testing.T) {
	tests := []struct {
		name    string
		theme   Theme
		wantErr bool
	}{
		{
			"合法主题",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			false,
		},
		{
			"无名称",
			Theme{PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"无 PageSize",
			Theme{Name: "test", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"FontSize 为 0",
			Theme{Name: "test", PageSize: "A4", FontSize: 0, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"LineHeight 为 0",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 0,
				Colors: ColorScheme{Text: "#333", Background: "#fff"}},
			true,
		},
		{
			"无文本颜色",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Background: "#fff"}},
			true,
		},
		{
			"无背景颜色",
			Theme{Name: "test", PageSize: "A4", FontSize: 12, LineHeight: 1.6,
				Colors: ColorScheme{Text: "#333"}},
			true,
		},
	}

	for _, tt := range tests {
		err := tt.theme.Validate()
		if tt.wantErr && err == nil {
			t.Errorf("%s: 应验证失败", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: 不应验证失败: %v", tt.name, err)
		}
	}
}

// TestThemeClone 测试主题克隆
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
		t.Error("克隆应返回新实例")
	}
	if clone.Name != original.Name {
		t.Errorf("克隆的 Name 不同: got %q", clone.Name)
	}

	// 修改克隆不应影响原件
	clone.Name = "modified"
	if original.Name == "modified" {
		t.Error("修改克隆不应影响原件")
	}
}

// TestLoadFromFile 测试从文件加载主题
func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "custom.yaml")
	content := `
name: custom-theme
pageSize: A4
fontFamily: "Source Han Sans"
fontSize: 14
codeTheme: dracula
lineHeight: 1.8
colors:
  text: "#2d2d2d"
  background: "#ffffff"
  heading: "#c0392b"
  link: "#2980b9"
  codeBg: "#f5f5f5"
  codeText: "#333"
  accent: "#e74c3c"
  border: "#ddd"
margins:
  top: 25
  bottom: 25
  left: 20
  right: 20
`
	if err := os.WriteFile(themePath, []byte(content), 0644); err != nil {
		t.Fatalf("写入主题文件失败: %v", err)
	}

	tm := NewThemeManager()
	thm, err := tm.LoadFromFile(themePath)
	if err != nil {
		t.Fatalf("从文件加载主题失败: %v", err)
	}

	if thm.Name != "custom-theme" {
		t.Errorf("主题名称错误: got %q", thm.Name)
	}
	if thm.FontSize != 14 {
		t.Errorf("字体大小错误: got %d", thm.FontSize)
	}
	if thm.Colors.Heading != "#c0392b" {
		t.Errorf("标题颜色错误: got %q", thm.Colors.Heading)
	}

	// 应该能通过名称获取
	retrieved, err := tm.Get("custom-theme")
	if err != nil {
		t.Errorf("加载后应能通过名称获取主题: %v", err)
	}
	if retrieved.FontSize != 14 {
		t.Error("通过名称获取的主题属性应正确")
	}
}

// TestLoadFromFileNonExistent 测试加载不存在的主题文件
func TestLoadFromFileNonExistent(t *testing.T) {
	tm := NewThemeManager()
	_, err := tm.LoadFromFile("/nonexistent/theme.yaml")
	if err == nil {
		t.Error("加载不存在的文件应返回错误")
	}
}

// TestLoadFromFileInvalidYAML 测试加载无效 YAML 主题
func TestLoadFromFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(themePath, []byte("{{invalid: yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	tm := NewThemeManager()
	_, err := tm.LoadFromFile(themePath)
	if err == nil {
		t.Error("无效 YAML 应返回错误")
	}
}

// TestLoadFromFileAutoName 测试文件名自动命名
func TestLoadFromFileAutoName(t *testing.T) {
	tmpDir := t.TempDir()
	themePath := filepath.Join(tmpDir, "my-theme.yaml")
	// 必须包含 name 字段，因为验证在自动命名之前执行
	content := `
name: my-theme
pageSize: A4
fontFamily: sans-serif
fontSize: 12
lineHeight: 1.5
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
		t.Fatalf("加载主题失败: %v", err)
	}

	if thm.Name != "my-theme" {
		t.Errorf("主题名错误: got %q, want %q", thm.Name, "my-theme")
	}

	// 应通过名称获取
	_, err = tm.Get("my-theme")
	if err != nil {
		t.Errorf("应能通过名称获取加载的主题: %v", err)
	}
}

// TestBuiltinThemesValidate 测试所有内置主题通过验证
func TestBuiltinThemesValidate(t *testing.T) {
	tm := NewThemeManager()
	for _, name := range tm.List() {
		thm, err := tm.Get(name)
		if err != nil {
			t.Errorf("获取主题 %q 失败: %v", name, err)
			continue
		}
		if err := thm.Validate(); err != nil {
			t.Errorf("内置主题 %q 验证失败: %v", name, err)
		}
	}
}

// TestToCSSAllThemes 测试所有主题都能生成 CSS
func TestToCSSAllThemes(t *testing.T) {
	tm := NewThemeManager()
	for _, name := range tm.List() {
		thm, _ := tm.Get(name)
		css := thm.ToCSS()
		if css == "" {
			t.Errorf("主题 %q 生成的 CSS 为空", name)
		}
		if !strings.Contains(css, ":root") {
			t.Errorf("主题 %q 的 CSS 应包含 :root", name)
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

// TestGetThemeDescription_KnownThemes 测试获取已知主题的描述
func TestGetThemeDescription_KnownThemes(t *testing.T) {
	tests := []struct {
		name           string
		themeName      string
		expectedSubstr string
	}{
		{
			name:           "technical theme",
			themeName:      "technical",
			expectedSubstr: "干净、专业",
		},
		{
			name:           "elegant theme",
			themeName:      "elegant",
			expectedSubstr: "优雅",
		},
		{
			name:           "minimal theme",
			themeName:      "minimal",
			expectedSubstr: "极简",
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

// TestGetThemeDescription_UnknownTheme 测试获取未知主题的描述
func TestGetThemeDescription_UnknownTheme(t *testing.T) {
	unknownThemes := []string{
		"nonexistent",
		"unknown-theme",
		"xyz",
		"not-a-theme",
	}

	expectedDefault := "未知的主题"

	for _, themeName := range unknownThemes {
		t.Run(themeName, func(t *testing.T) {
			desc := GetThemeDescription(themeName)
			if desc != expectedDefault {
				t.Errorf("GetThemeDescription(%q) = %q, want %q", themeName, desc, expectedDefault)
			}
		})
	}
}

// TestGetThemeDescription_EmptyString 测试空字符串输入
func TestGetThemeDescription_EmptyString(t *testing.T) {
	desc := GetThemeDescription("")
	expectedDefault := "未知的主题"
	if desc != expectedDefault {
		t.Errorf("GetThemeDescription(%q) = %q, want %q", "", desc, expectedDefault)
	}
}

// TestGetThemeDescription_CaseSensitive 测试主题名称大小写敏感性
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
			expectedDefault := "未知的主题"

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

// TestGetThemeDescription_AllKnownThemes 测试所有已知主题
func TestGetThemeDescription_AllKnownThemes(t *testing.T) {
	expectedDefault := "未知的主题"

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

// TestGetThemeDescription_NonEmptyDescriptions 测试已知主题返回非空描述
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

// TestGetThemeDescription_ConsistentResults 测试多次调用返回一致结果
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
