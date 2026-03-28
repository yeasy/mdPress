package theme

// builtinTechnical returns the "technical" theme — clean, professional style
// suited for technical documentation and IT books.
func builtinTechnical() *Theme {
	return &Theme{
		Name:       "technical",
		PageSize:   "A4",
		FontFamily: "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans CJK SC', 'Noto Sans SC', 'Source Han Sans SC', 'Segoe UI', 'Helvetica Neue', Arial, sans-serif",
		FontSize:   11,
		CodeTheme:  "github",
		LineHeight: 1.6,

		Colors: ColorScheme{
			Text:       "#2C3E50",
			Background: "#FFFFFF",
			Heading:    "#1A5490",
			Link:       "#0066CC",
			CodeBg:     "#F5F7F9",
			CodeText:   "#2C3E50",
			Accent:     "#0066CC",
			Border:     "#D4D4D4",
		},

		Margins: MarginSettings{
			Top:    20.0,
			Bottom: 20.0,
			Left:   20.0,
			Right:  20.0,
		},

		HeaderTemplate: "<div style='text-align: center; font-size: 10pt; color: #999;'>Technical Document</div>",
		FooterTemplate: "<div style='text-align: center; font-size: 10pt; color: #999;'><span class='pageNumber'></span></div>",
	}
}

// builtinElegant returns the "elegant" theme — a serif-based, literary style
// suited for fiction, essays, and publishing.
func builtinElegant() *Theme {
	return &Theme{
		Name:       "elegant",
		PageSize:   "A4",
		FontFamily: "'Songti SC', 'STSong', 'Noto Serif CJK SC', 'Source Han Serif SC', 'Georgia', 'Garamond', 'Times New Roman', serif",
		FontSize:   12,
		CodeTheme:  "github",
		LineHeight: 1.8,

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

		Margins: MarginSettings{
			Top:    25.0,
			Bottom: 25.0,
			Left:   25.0,
			Right:  25.0,
		},

		HeaderTemplate: "<div style='text-align: right; font-size: 11pt; color: #8B6914; border-bottom: 1px solid #D7CCBB; padding-bottom: 5px;'><span class='chapterTitle'></span></div>",
		FooterTemplate: "<div style='text-align: center; font-size: 11pt; color: #8B6914; border-top: 1px solid #D7CCBB; padding-top: 5px;'>- <span class='pageNumber'></span> -</div>",
	}
}

// builtinMinimal returns the "minimal" theme — a clean, whitespace-heavy style
// with maximum readability.
func builtinMinimal() *Theme {
	return &Theme{
		Name:       "minimal",
		PageSize:   "A4",
		FontFamily: "-apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans CJK SC', 'Noto Sans SC', 'Source Han Sans SC', 'San Francisco', 'Segoe UI', Roboto, sans-serif",
		FontSize:   10,
		CodeTheme:  "default",
		LineHeight: 1.7,

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

		Margins: MarginSettings{
			Top:    30.0,
			Bottom: 30.0,
			Left:   30.0,
			Right:  30.0,
		},

		HeaderTemplate: "",
		FooterTemplate: "<div style='text-align: center; font-size: 10pt; color: #666;'><span class='pageNumber'></span></div>",
	}
}

// themeVariants lists available theme variants and their descriptions.
var themeVariants = map[string]string{
	"technical": "Clean, professional style for technical documentation and IT books",
	"elegant":   "Elegant serif-based style for fiction, essays, and publishing",
	"minimal":   "Minimal style with generous whitespace and high readability",
}

// GetThemeDescription returns the human-readable description for a theme.
func GetThemeDescription(themeName string) string {
	if desc, ok := themeVariants[themeName]; ok {
		return desc
	}
	return "Unknown theme"
}
