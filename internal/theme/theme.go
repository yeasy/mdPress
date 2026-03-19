package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Theme 定义了文档的主题样式
type Theme struct {
	// 主题名称
	Name string `yaml:"name"`
	// 页面大小 (A4, Letter等)
	PageSize string `yaml:"pageSize"`
	// 字体族
	FontFamily string `yaml:"fontFamily"`
	// 字体大小 (pt)
	FontSize int `yaml:"fontSize"`
	// 代码主题
	CodeTheme string `yaml:"codeTheme"`
	// 行高
	LineHeight float64 `yaml:"lineHeight"`

	// 颜色设置
	Colors ColorScheme `yaml:"colors"`

	// 边距设置
	Margins MarginSettings `yaml:"margins"`

	// 页眉模板
	HeaderTemplate string `yaml:"headerTemplate"`
	// 页脚模板
	FooterTemplate string `yaml:"footerTemplate"`
}

// ColorScheme 定义颜色方案
type ColorScheme struct {
	// 文本颜色
	Text string `yaml:"text"`
	// 背景颜色
	Background string `yaml:"background"`
	// 标题颜色
	Heading string `yaml:"heading"`
	// 链接颜色
	Link string `yaml:"link"`
	// 代码块背景颜色
	CodeBg string `yaml:"codeBg"`
	// 代码文本颜色
	CodeText string `yaml:"codeText"`
	// 强调颜色
	Accent string `yaml:"accent"`
	// 边框颜色
	Border string `yaml:"border"`
}

// MarginSettings 定义边距设置 (单位: mm)
type MarginSettings struct {
	// 上边距
	Top float64 `yaml:"top"`
	// 下边距
	Bottom float64 `yaml:"bottom"`
	// 左边距
	Left float64 `yaml:"left"`
	// 右边距
	Right float64 `yaml:"right"`
}

// ThemeManager 管理主题的加载和获取
type ThemeManager struct {
	themes map[string]*Theme
}

// NewThemeManager 创建一个新的主题管理器，包含内置主题
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]*Theme),
	}

	// 加载内置主题
	tm.themes["technical"] = builtinTechnical()
	tm.themes["elegant"] = builtinElegant()
	tm.themes["minimal"] = builtinMinimal()

	return tm
}

// Get 根据名称获取主题
func (tm *ThemeManager) Get(name string) (*Theme, error) {
	if name == "" {
		return tm.themes["technical"], nil
	}

	theme, exists := tm.themes[name]
	if !exists {
		return nil, fmt.Errorf("主题 '%s' 不存在", name)
	}

	return theme, nil
}

// LoadFromFile 从YAML文件加载主题
func (tm *ThemeManager) LoadFromFile(path string) (*Theme, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("主题文件不存在: %w", err)
	}

	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取主题文件失败: %w", err)
	}

	// 解析YAML
	theme := &Theme{}
	if err := yaml.Unmarshal(data, theme); err != nil {
		return nil, fmt.Errorf("解析主题文件失败: %w", err)
	}

	// 验证主题
	if err := theme.Validate(); err != nil {
		return nil, fmt.Errorf("主题验证失败: %w", err)
	}

	// 将主题添加到管理器
	if theme.Name == "" {
		theme.Name = strings.TrimSuffix(filepath.Base(path), ".yaml")
	}
	tm.themes[theme.Name] = theme

	return theme, nil
}

// List 返回所有可用的主题名称列表
func (tm *ThemeManager) List() []string {
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// Validate 验证主题的有效性
func (t *Theme) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("主题名称不能为空")
	}

	if t.PageSize == "" {
		return fmt.Errorf("页面大小不能为空")
	}

	if t.FontSize <= 0 {
		return fmt.Errorf("字体大小必须大于0")
	}

	if t.LineHeight <= 0 {
		return fmt.Errorf("行高必须大于0")
	}

	if t.Colors.Text == "" {
		return fmt.Errorf("文本颜色不能为空")
	}

	if t.Colors.Background == "" {
		return fmt.Errorf("背景颜色不能为空")
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
			parts[i] = " " + trimmed
			continue
		}
		// CSS generic family or vendor-prefixed value (starts with -).
		if genericFamilies[strings.ToLower(trimmed)] || strings.HasPrefix(trimmed, "-") {
			parts[i] = " " + trimmed
			continue
		}
		// Contains spaces but not quoted — wrap in single quotes.
		if strings.Contains(trimmed, " ") {
			parts[i] = " '" + trimmed + "'"
		} else {
			parts[i] = " " + trimmed
		}
	}
	result := strings.Join(parts, ",")
	return strings.TrimSpace(result)
}

// ToCSS converts theme settings to CSS code.
func (t *Theme) ToCSS() string {
	var css strings.Builder

	css.WriteString("/* Auto-generated theme CSS */\n")
	css.WriteString(":root {\n")
	css.WriteString(fmt.Sprintf("  --font-family: %s;\n", quoteFontFamily(t.FontFamily)))
	css.WriteString(fmt.Sprintf("  --font-size: %dpt;\n", t.FontSize))
	css.WriteString(fmt.Sprintf("  --line-height: %.2f;\n", t.LineHeight))
	css.WriteString(fmt.Sprintf("  --color-text: %s;\n", t.Colors.Text))
	css.WriteString(fmt.Sprintf("  --color-background: %s;\n", t.Colors.Background))
	css.WriteString(fmt.Sprintf("  --color-heading: %s;\n", t.Colors.Heading))
	css.WriteString(fmt.Sprintf("  --color-link: %s;\n", t.Colors.Link))
	css.WriteString(fmt.Sprintf("  --color-code-bg: %s;\n", t.Colors.CodeBg))
	css.WriteString(fmt.Sprintf("  --color-code-text: %s;\n", t.Colors.CodeText))
	css.WriteString(fmt.Sprintf("  --color-accent: %s;\n", t.Colors.Accent))
	css.WriteString(fmt.Sprintf("  --color-border: %s;\n", t.Colors.Border))
	css.WriteString(fmt.Sprintf("  --margin-top: %.2fmm;\n", t.Margins.Top))
	css.WriteString(fmt.Sprintf("  --margin-bottom: %.2fmm;\n", t.Margins.Bottom))
	css.WriteString(fmt.Sprintf("  --margin-left: %.2fmm;\n", t.Margins.Left))
	css.WriteString(fmt.Sprintf("  --margin-right: %.2fmm;\n", t.Margins.Right))
	css.WriteString("}\n\n")

	// 基础样式
	css.WriteString("body {\n")
	css.WriteString("  font-family: var(--font-family);\n")
	css.WriteString("  font-size: var(--font-size);\n")
	css.WriteString("  line-height: var(--line-height);\n")
	css.WriteString("  color: var(--color-text);\n")
	css.WriteString("  background-color: var(--color-background);\n")
	css.WriteString("  margin: var(--margin-top) var(--margin-right) var(--margin-bottom) var(--margin-left);\n")
	css.WriteString("}\n\n")

	// 标题样式
	css.WriteString("h1, h2, h3, h4, h5, h6 {\n")
	css.WriteString("  color: var(--color-heading);\n")
	css.WriteString("  font-weight: bold;\n")
	css.WriteString("  margin-top: 1em;\n")
	css.WriteString("  margin-bottom: 0.5em;\n")
	css.WriteString("}\n\n")

	// 链接样式
	css.WriteString("a {\n")
	css.WriteString("  color: var(--color-link);\n")
	css.WriteString("  text-decoration: underline;\n")
	css.WriteString("}\n\n")

	// 代码样式
	css.WriteString("code, pre {\n")
	css.WriteString("  background-color: var(--color-code-bg);\n")
	css.WriteString("  color: var(--color-code-text);\n")
	css.WriteString("  font-family: 'Courier New', monospace;\n")
	css.WriteString("  padding: 0.2em 0.4em;\n")
	css.WriteString("  border-radius: 3px;\n")
	css.WriteString("}\n\n")

	css.WriteString("pre {\n")
	css.WriteString("  padding: 1em;\n")
	css.WriteString("  overflow-x: auto;\n")
	css.WriteString("}\n\n")

	// 块引用样式
	css.WriteString("blockquote {\n")
	css.WriteString("  border-left: 4px solid var(--color-accent);\n")
	css.WriteString("  margin-left: 0;\n")
	css.WriteString("  padding-left: 1em;\n")
	css.WriteString("  color: var(--color-text);\n")
	css.WriteString("  opacity: 0.8;\n")
	css.WriteString("}\n\n")

	// 表格样式
	css.WriteString("table {\n")
	css.WriteString("  border-collapse: collapse;\n")
	css.WriteString("  width: 100%;\n")
	css.WriteString("  margin: 1em 0;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th, table td {\n")
	css.WriteString("  border: 1px solid var(--color-border);\n")
	css.WriteString("  padding: 0.5em;\n")
	css.WriteString("  text-align: left;\n")
	css.WriteString("}\n\n")

	css.WriteString("table th {\n")
	css.WriteString("  background-color: var(--color-code-bg);\n")
	css.WriteString("  color: var(--color-heading);\n")
	css.WriteString("}\n\n")

	return css.String()
}

// Clone 创建主题的深拷贝
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
