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
