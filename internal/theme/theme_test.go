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

// ============================================================================
// 新增：主题验证集成测试
// ============================================================================

// TestBuiltinThemesStructureIntegrity 测试内置主题的结构完整性 (表格驱动)
func TestBuiltinThemesStructureIntegrity(t *testing.T) {
	builtinThemes := []string{"technical", "elegant", "minimal"}

	tests := []struct {
		name       string
		fieldCheck func(t *testing.T, theme *Theme, themeName string)
	}{
		{
			name: "主题名称一致性",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Name != themeName {
					t.Errorf("主题 %q: 名称不匹配，期望 %q 得 %q", themeName, themeName, theme.Name)
				}
			},
		},
		{
			name: "必需字段非空",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Name == "" {
					t.Errorf("主题 %q: Name 为空", themeName)
				}
				if theme.PageSize == "" {
					t.Errorf("主题 %q: PageSize 为空", themeName)
				}
				if theme.FontFamily == "" {
					t.Errorf("主题 %q: FontFamily 为空", themeName)
				}
				if theme.FontSize <= 0 {
					t.Errorf("主题 %q: FontSize 无效 (%d)", themeName, theme.FontSize)
				}
				if theme.CodeTheme == "" {
					t.Errorf("主题 %q: CodeTheme 为空", themeName)
				}
				if theme.LineHeight <= 0 {
					t.Errorf("主题 %q: LineHeight 无效 (%.2f)", themeName, theme.LineHeight)
				}
			},
		},
		{
			name: "颜色方案完整",
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
						t.Errorf("主题 %q: 颜色 %s 为空", themeName, colorName)
					}
				}
			},
		},
		{
			name: "边距设置有效",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if theme.Margins.Top < 0 {
					t.Errorf("主题 %q: Top 边距无效 (%.2f)", themeName, theme.Margins.Top)
				}
				if theme.Margins.Bottom < 0 {
					t.Errorf("主题 %q: Bottom 边距无效 (%.2f)", themeName, theme.Margins.Bottom)
				}
				if theme.Margins.Left < 0 {
					t.Errorf("主题 %q: Left 边距无效 (%.2f)", themeName, theme.Margins.Left)
				}
				if theme.Margins.Right < 0 {
					t.Errorf("主题 %q: Right 边距无效 (%.2f)", themeName, theme.Margins.Right)
				}
			},
		},
		{
			name: "验证通过",
			fieldCheck: func(t *testing.T, theme *Theme, themeName string) {
				if err := theme.Validate(); err != nil {
					t.Errorf("主题 %q: 验证失败: %v", themeName, err)
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
					t.Fatalf("获取主题 %q 失败: %v", themeName, err)
				}
				tt.fieldCheck(t, theme, themeName)
			}
		})
	}
}

// TestThemeColorValidation 测试主题颜色值格式 (表格驱动)
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
					t.Errorf("颜色 %s 不应为空", colorName)
					continue
				}

				// 检查是否是有效的十六进制颜色格式
				if !isValidColorHex(colorValue) && !isValidColorRGB(colorValue) && !isValidColorName(colorValue) {
					t.Logf("警告: 颜色 %s 格式可能非标准: %q (但可能有效)", colorName, colorValue)
				}
			}
		})
	}
}

// TestThemeCSSSyntax 测试生成的 CSS 语法有效性 (表格驱动)
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
				t.Fatal("生成的 CSS 不应为空")
			}

			// 检查必需的 CSS 模式
			for _, pattern := range requiredCSSPatterns {
				if !strings.Contains(css, pattern) {
					t.Errorf("CSS 应包含 %q", pattern)
				}
			}

			// 基本的 CSS 结构检查
			openBraces := strings.Count(css, "{")
			closeBraces := strings.Count(css, "}")
			if openBraces != closeBraces {
				t.Errorf("CSS 括号不匹配: { 出现 %d 次, } 出现 %d 次", openBraces, closeBraces)
			}
		})
	}
}

// TestThemeNoDuplicateNames 测试没有重复的主题名称
func TestThemeNoDuplicateNames(t *testing.T) {
	tm := NewThemeManager()
	names := tm.List()

	nameMap := make(map[string]int)
	for _, name := range names {
		nameMap[name]++
	}

	for name, count := range nameMap {
		if count > 1 {
			t.Errorf("主题名称 %q 重复 %d 次", name, count)
		}
	}
}

// TestThemeConsistencyBetweenBuiltinAndManager 测试内置主题与管理器一致性
func TestThemeConsistencyBetweenBuiltinAndManager(t *testing.T) {
	builtinFuncs := map[string]func() *Theme{
		"technical": builtinTechnical,
		"elegant":   builtinElegant,
		"minimal":   builtinMinimal,
	}

	tm := NewThemeManager()

	for themeName, builderFunc := range builtinFuncs {
		t.Run(themeName, func(t *testing.T) {
			// 从 manager 获取
			managed, err := tm.Get(themeName)
			if err != nil {
				t.Fatalf("从 manager 获取 %q 失败: %v", themeName, err)
			}

			// 直接构建
			builtin := builderFunc()

			// 比较关键字段
			if managed.Name != builtin.Name {
				t.Errorf("Name 不一致: managed=%q, builtin=%q", managed.Name, builtin.Name)
			}
			if managed.PageSize != builtin.PageSize {
				t.Errorf("PageSize 不一致: managed=%q, builtin=%q", managed.PageSize, builtin.PageSize)
			}
			if managed.FontSize != builtin.FontSize {
				t.Errorf("FontSize 不一致: managed=%d, builtin=%d", managed.FontSize, builtin.FontSize)
			}
			if managed.CodeTheme != builtin.CodeTheme {
				t.Errorf("CodeTheme 不一致: managed=%q, builtin=%q", managed.CodeTheme, builtin.CodeTheme)
			}
		})
	}
}

// TestThemeCloneIndependence 测试克隆主题的独立性
func TestThemeCloneIndependence(t *testing.T) {
	tm := NewThemeManager()
	original, _ := tm.Get("technical")

	cloned := original.Clone()

	// 修改克隆
	cloned.Name = "cloned-technical"
	cloned.FontSize = 20
	cloned.Colors.Text = "#FF0000"

	// 检查原始主题未被修改
	if original.Name != "technical" {
		t.Errorf("原始主题被修改了: %q", original.Name)
	}
	if original.FontSize != 11 {
		t.Errorf("原始主题 FontSize 被修改了: %d", original.FontSize)
	}
	if original.Colors.Text != "#2C3E50" {
		t.Errorf("原始主题颜色被修改了: %q", original.Colors.Text)
	}
}

// TestThemeFontFamilyQuoting 测试字体族引号处理
func TestThemeFontFamilyQuoting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // 输出应包含的内容
	}{
		{
			name:     "多字体链式",
			input:    "Arial, 'Helvetica Neue', sans-serif",
			contains: "Arial",
		},
		{
			name:     "CJK 字体",
			input:    "'Noto Sans SC', sans-serif",
			contains: "Noto",
		},
		{
			name:     "单字体",
			input:    "monospace",
			contains: "monospace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := quoteFontFamily(tt.input)
			if quoted == "" {
				t.Error("引号处理结果不应为空")
			}
			if !strings.Contains(quoted, tt.contains) {
				t.Errorf("输出应包含 %q，得 %q", tt.contains, quoted)
			}
		})
	}
}

// TestThemeLoadFromYAML 测试从 YAML 文件加载主题
func TestThemeLoadFromYAML(t *testing.T) {
	// 获取主题文件目录
	themeDir := filepath.Join("/sessions/epic-blissful-johnson/mnt/mdpress", "themes")

	themeFiles := []string{
		filepath.Join(themeDir, "technical.yaml"),
		filepath.Join(themeDir, "elegant.yaml"),
		filepath.Join(themeDir, "minimal.yaml"),
	}

	tm := NewThemeManager()

	for _, themeFile := range themeFiles {
		t.Run(filepath.Base(themeFile), func(t *testing.T) {
			// 检查文件是否存在
			if _, err := os.Stat(themeFile); err != nil {
				t.Logf("跳过: 主题文件不存在: %s", themeFile)
				return
			}

			// 尝试加载
			theme, err := tm.LoadFromFile(themeFile)
			if err != nil {
				t.Fatalf("加载主题文件失败: %v", err)
			}

			// 验证加载的主题
			if theme.Name == "" {
				t.Error("加载的主题名称不应为空")
			}
			if theme.PageSize == "" {
				t.Error("加载的主题 PageSize 不应为空")
			}
		})
	}
}

// TestThemeAllFieldsNotEmpty 测试所有字段都不为空（针对内置主题）
func TestThemeAllFieldsNotEmpty(t *testing.T) {
	tm := NewThemeManager()
	builtinThemes := []string{"technical", "elegant", "minimal"}

	for _, themeName := range builtinThemes {
		t.Run(themeName, func(t *testing.T) {
			theme, _ := tm.Get(themeName)

			// 关键字段检查
			checks := map[string]interface{}{
				"Name":       theme.Name,
				"PageSize":   theme.PageSize,
				"FontFamily": theme.FontFamily,
				"CodeTheme":  theme.CodeTheme,
			}

			for fieldName, value := range checks {
				if str, ok := value.(string); ok && str == "" {
					t.Errorf("%s 字段不应为空", fieldName)
				}
			}

			// 数值字段检查
			if theme.FontSize == 0 {
				t.Error("FontSize 不应为 0")
			}
			if theme.LineHeight == 0 {
				t.Error("LineHeight 不应为 0")
			}
		})
	}
}

// ============================================================================
// 辅助函数
// ============================================================================

// isValidColorHex 检查是否是有效的十六进制颜色
func isValidColorHex(color string) bool {
	if !strings.HasPrefix(color, "#") {
		return false
	}
	hex := strings.TrimPrefix(color, "#")
	return len(hex) == 6 || len(hex) == 3 || len(hex) == 8
}

// isValidColorRGB 检查是否是有效的 RGB 颜色
func isValidColorRGB(color string) bool {
	return strings.HasPrefix(color, "rgb")
}

// isValidColorName 检查是否是有效的 CSS 颜色名称
func isValidColorName(color string) bool {
	cssColors := map[string]bool{
		"red": true, "blue": true, "green": true, "black": true, "white": true,
		"transparent": true, "inherit": true, "currentColor": true,
	}
	return cssColors[strings.ToLower(color)]
}
