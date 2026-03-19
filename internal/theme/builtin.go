package theme

// builtinTechnical 返回 "technical" 主题 - 干净、专业的风格
// 适合技术文档和IT领域的书籍
func builtinTechnical() *Theme {
	return &Theme{
		Name:       "technical",
		PageSize:   "A4",
		FontFamily: "'Segoe UI', 'Helvetica Neue', Arial, sans-serif",
		FontSize:   11,
		CodeTheme:  "monokai",
		LineHeight: 1.6,

		// 技术风格的色彩方案 - 深灰色文本，白色背景，蓝色强调
		Colors: ColorScheme{
			Text:      "#2C3E50",
			Background: "#FFFFFF",
			Heading:   "#1A5490",
			Link:      "#0066CC",
			CodeBg:    "#F5F7F9",
			CodeText:  "#2C3E50",
			Accent:    "#0066CC",
			Border:    "#D4D4D4",
		},

		// 标准边距
		Margins: MarginSettings{
			Top:    20.0,
			Bottom: 20.0,
			Left:   20.0,
			Right:  20.0,
		},

		// 页眉页脚模板
		HeaderTemplate: "<div style='text-align: center; font-size: 10pt; color: #999;'>Technical Document</div>",
		FooterTemplate: "<div style='text-align: center; font-size: 10pt; color: #999;'><span class='pageNumber'></span></div>",
	}
}

// builtinElegant 返回 "elegant" 主题 - 优雅的风格
// 使用衬线字体，适合文学和出版类书籍
func builtinElegant() *Theme {
	return &Theme{
		Name:       "elegant",
		PageSize:   "A4",
		FontFamily: "'Georgia', 'Garamond', 'Times New Roman', serif",
		FontSize:   12,
		CodeTheme:  "github",
		LineHeight: 1.8,

		// 优雅风格的色彩方案 - 棕色文本，奶油白背景，金色强调
		Colors: ColorScheme{
			Text:       "#3E2723",
			Background: "#FFFBF0",
			Heading:    "#1B0000",
			Link:       "#8B6914",
			CodeBg:     "#F5F2EB",
			CodeText:   "#3E2723",
			Accent:     "#D4A574",
			Border:     "#D7CCBB",
		},

		// 对称的边距，书籍风格
		Margins: MarginSettings{
			Top:    25.0,
			Bottom: 25.0,
			Left:   25.0,
			Right:  25.0,
		},

		// 优雅的页眉页脚
		HeaderTemplate: "<div style='text-align: right; font-size: 11pt; color: #8B6914; border-bottom: 1px solid #D7CCBB; padding-bottom: 5px;'><span class='chapterTitle'></span></div>",
		FooterTemplate: "<div style='text-align: center; font-size: 11pt; color: #8B6914; border-top: 1px solid #D7CCBB; padding-top: 5px;'>- <span class='pageNumber'></span> -</div>",
	}
}

// builtinMinimal 返回 "minimal" 主题 - 极简风格
// 简洁干净，大量空白，高度可读性
func builtinMinimal() *Theme {
	return &Theme{
		Name:       "minimal",
		PageSize:   "A4",
		FontFamily: "-apple-system, BlinkMacSystemFont, 'San Francisco', 'Segoe UI', Roboto, sans-serif",
		FontSize:   10,
		CodeTheme:  "default",
		LineHeight: 1.7,

		// 极简风格的色彩方案 - 黑色文本，白色背景，最少颜色使用
		Colors: ColorScheme{
			Text:       "#000000",
			Background: "#FFFFFF",
			Heading:    "#000000",
			Link:       "#0000EE",
			CodeBg:     "#EEEEEE",
			CodeText:   "#000000",
			Accent:     "#555555",
			Border:     "#CCCCCC",
		},

		// 宽边距，留出充分的空白
		Margins: MarginSettings{
			Top:    30.0,
			Bottom: 30.0,
			Left:   30.0,
			Right:  30.0,
		},

		// 最小化的页眉页脚
		HeaderTemplate: "",
		FooterTemplate: "<div style='text-align: center; font-size: 10pt; color: #666;'><span class='pageNumber'></span></div>",
	}
}

// ThemeVariants 定义可用的主题变体及其描述
var ThemeVariants = map[string]string{
	"technical": "干净、专业的风格，适合技术文档和IT领域的书籍",
	"elegant":   "优雅的风格，使用衬线字体，适合文学和出版类书籍",
	"minimal":   "极简风格，简洁干净，大量空白，高度可读性",
}

// GetThemeDescription 获取主题的描述
func GetThemeDescription(themeName string) string {
	if desc, ok := ThemeVariants[themeName]; ok {
		return desc
	}
	return "未知的主题"
}
