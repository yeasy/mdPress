package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// themes_preview.go contains the implementation for the "themes preview" subcommand.
// It generates a self-contained HTML file showcasing all built-in themes with sample content.

// executeThemesPreview generates an HTML preview of all themes.
func executeThemesPreview(outputPath string) error {
	// Use default output path if empty
	if outputPath == "" {
		outputPath = "themes-preview.html"
	}

	logger := slog.Default()
	logger.Info("Generating themes preview", slog.String("output", outputPath))

	themes := getAvailableThemes()

	html := generatePreviewHTML(themes)

	// Write to file
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("failed to write preview file: %w", err)
	}

	fmt.Printf("✓ Theme preview generated: %s\n", absPath)
	return nil
}

// generatePreviewHTML creates a self-contained HTML document showcasing all themes.
func generatePreviewHTML(themes []Theme) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>mdpress Themes Preview</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans SC', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f5f5f5;
            padding: 40px 20px;
            line-height: 1.6;
            color: #333;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        .header {
            text-align: center;
            margin-bottom: 60px;
            border-bottom: 3px solid #333;
            padding-bottom: 30px;
        }

        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
            color: #1a1a1a;
        }

        .header p {
            font-size: 1.1em;
            color: #666;
        }

        .themes-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
            gap: 40px;
            margin-bottom: 60px;
        }

        .theme-section {
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            transition: box-shadow 0.3s ease;
        }

        .theme-section:hover {
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
        }

        .theme-header {
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border-bottom: 4px solid #333;
        }

        .theme-name {
            font-size: 1.8em;
            font-weight: bold;
            margin-bottom: 5px;
        }

        .theme-code {
            font-size: 0.9em;
            opacity: 0.9;
            font-family: 'Monaco', 'Courier New', 'PingFang SC', 'Noto Sans Mono CJK SC', monospace;
        }

        .theme-info {
            padding: 15px 20px;
            background: #f9f9f9;
            font-size: 0.9em;
            color: #666;
            border-bottom: 1px solid #e0e0e0;
        }

        .theme-colors {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 8px;
            padding: 15px 20px;
            border-bottom: 1px solid #e0e0e0;
        }

        .color-swatch {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 5px;
        }

        .color-box {
            width: 100%;
            height: 50px;
            border-radius: 4px;
            border: 1px solid #ddd;
            box-shadow: inset 0 0 1px rgba(0, 0, 0, 0.1);
        }

        .color-label {
            font-size: 0.75em;
            font-weight: 600;
            text-align: center;
            width: 100%;
            word-break: break-word;
        }

        .color-value {
            font-size: 0.7em;
            color: #999;
            font-family: 'Monaco', 'Courier New', 'PingFang SC', 'Noto Sans Mono CJK SC', monospace;
        }

        .theme-content {
            padding: 30px;
        }

        .theme-content h2 {
            margin-bottom: 15px;
            padding-bottom: 8px;
            border-bottom: 2px solid;
        }

        .theme-content p {
            margin-bottom: 15px;
            line-height: 1.7;
        }

        .theme-content code {
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Monaco', 'Courier New', 'PingFang SC', 'Noto Sans Mono CJK SC', monospace;
            font-size: 0.9em;
        }

        .theme-content pre {
            margin: 15px 0;
            padding: 12px;
            border-radius: 4px;
            overflow-x: auto;
            font-family: 'Monaco', 'Courier New', 'PingFang SC', 'Noto Sans Mono CJK SC', monospace;
            font-size: 0.85em;
            line-height: 1.4;
        }

        .theme-content pre code {
            padding: 0;
            background: none;
        }

        .theme-content table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0;
            font-size: 0.95em;
        }

        .theme-content table th {
            padding: 10px;
            text-align: left;
            font-weight: 600;
            border-bottom: 2px solid;
        }

        .theme-content table td {
            padding: 10px;
            border-bottom: 1px solid;
        }

        .theme-content blockquote {
            margin: 15px 0;
            padding: 15px;
            border-left: 4px solid;
            border-radius: 0 4px 4px 0;
            font-style: italic;
        }

        .footer {
            text-align: center;
            padding: 30px;
            color: #666;
            font-size: 0.9em;
            border-top: 1px solid #ddd;
            margin-top: 60px;
        }

        @media (max-width: 768px) {
            .themes-grid {
                grid-template-columns: 1fr;
            }

            .header h1 {
                font-size: 1.8em;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>mdpress Theme Showcase</h1>
            <p>Interactive preview of all built-in themes</p>
        </div>

        <div class="themes-grid">
`

	// Generate preview for each theme
	for _, theme := range themes {
		html += generateThemePreviewSection(theme)
	}

	html += `        </div>

        <div class="footer">
            <p>Generated by mdpress &mdash; All themes are customizable via your project's themes/ directory</p>
        </div>
    </div>
</body>
</html>
`

	return html
}

// generateThemePreviewSection creates an HTML section for a single theme.
func generateThemePreviewSection(theme Theme) string {
	section := fmt.Sprintf(`            <div class="theme-section">
                <div class="theme-header" style="background: linear-gradient(135deg, %s 0%%, %s 100%%);">
                    <div class="theme-name">%s</div>
                    <div class="theme-code">%s</div>
                </div>

                <div class="theme-info">
                    %s
                </div>

                <div class="theme-colors">
                    <div class="color-swatch">
                        <div class="color-box" style="background-color: %s;"></div>
                        <div class="color-label">Primary</div>
                        <div class="color-value">%s</div>
                    </div>
                    <div class="color-swatch">
                        <div class="color-box" style="background-color: %s;"></div>
                        <div class="color-label">Secondary</div>
                        <div class="color-value">%s</div>
                    </div>
                    <div class="color-swatch">
                        <div class="color-box" style="background-color: %s;"></div>
                        <div class="color-label">Accent</div>
                        <div class="color-value">%s</div>
                    </div>
                </div>

                <div class="theme-content" style="color: %s; background-color: %s;">
                    <h2 style="color: %s; border-bottom-color: %s;">Sample Heading</h2>
                    <p>This is a sample paragraph showcasing the theme's text styling. The quick brown fox jumps over the lazy dog.</p>

                    <p>You can use <code style="color: %s; background-color: %s;">inline code</code> for simple terms and references.</p>

                    <pre style="color: %s; background-color: %s; border: 1px solid %s;"><code>// Sample code block
func greet(name string) string {
    return "Hello, " + name
}</code></pre>

                    <table style="border-color: %s;">
                        <thead>
                            <tr style="border-bottom-color: %s; background-color: %s;">
                                <th>Feature</th>
                                <th>Value</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr style="border-bottom-color: %s;">
                                <td><strong>Author</strong></td>
                                <td>%s</td>
                            </tr>
                            <tr style="border-bottom-color: %s;">
                                <td><strong>Version</strong></td>
                                <td>%s</td>
                            </tr>
                        </tbody>
                    </table>

                    <blockquote style="border-left-color: %s; color: %s; background-color: rgba(0, 0, 0, 0.05);">A well-designed theme enhances readability and creates a professional appearance for your documentation.</blockquote>
                </div>
            </div>
`,
		theme.Colors.Primary,
		theme.Colors.Secondary,
		theme.DisplayName,
		theme.Name,
		theme.Description,
		theme.Colors.Primary,
		theme.Colors.Primary,
		theme.Colors.Secondary,
		theme.Colors.Secondary,
		theme.Colors.Accent,
		theme.Colors.Accent,
		theme.Colors.Text,
		theme.Colors.Background,
		theme.Colors.Primary,
		theme.Colors.Accent,
		theme.Colors.Text,
		theme.Colors.CodeBg,
		theme.Colors.Text,
		theme.Colors.CodeBg,
		theme.Colors.Secondary,
		theme.Colors.Secondary,
		theme.Colors.Secondary,
		theme.Colors.CodeBg,
		theme.Colors.Secondary,
		theme.Author,
		theme.Colors.Secondary,
		theme.Version,
		theme.Colors.Accent,
		theme.Colors.Text,
	)

	return section
}
