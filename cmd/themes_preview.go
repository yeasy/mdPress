package cmd

import (
	"fmt"
	"html"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// themes_preview.go contains the implementation for the "themes preview" subcommand.
// It generates a self-contained HTML file showcasing all built-in themes with sample
// content styled by each theme's real pipeline stylesheet (theme.ToCSS), so the
// preview always matches what builds actually produce.

// executeThemesPreview generates an HTML preview of all themes.
func executeThemesPreview(outputPath string) error {
	logger := slog.Default()

	// Use default output path if empty
	if outputPath == "" {
		outputPath = "themes-preview.html"
	}

	logger.Debug("Generating themes preview", slog.String("output", outputPath))

	themes := getAvailableThemes()

	previewHTML := generatePreviewHTML(themes)

	// Write to file
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	if err := os.WriteFile(absPath, []byte(previewHTML), 0o644); err != nil {
		return fmt.Errorf("failed to write preview file: %w", err)
	}

	if !quiet {
		fmt.Printf("✓ Theme preview generated: %s\n", absPath)
	}
	return nil
}

// generatePreviewHTML creates a self-contained HTML document showcasing all themes.
func generatePreviewHTML(themes []themeInfo) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
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
            max-width: 1400px;
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
            grid-template-columns: repeat(auto-fit, minmax(420px, 1fr));
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

        .theme-props {
            padding: 10px 20px;
            font-size: 0.85em;
            color: #666;
            border-bottom: 1px solid #e0e0e0;
        }

        .theme-props li {
            margin-left: 1.2em;
        }

        .theme-colors {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
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
            height: 40px;
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

        .theme-sample {
            border: 0;
            width: 100%;
            height: 660px;
            display: block;
            background: white;
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
            <p>Each sample below is rendered with the theme's real build stylesheet</p>
        </div>

        <div class="themes-grid">
`)

	// Generate preview for each theme
	for _, thm := range themes {
		sb.WriteString(generateThemePreviewSection(thm))
	}

	sb.WriteString(`        </div>

        <div class="footer">
            <p>Generated by mdpress &mdash; customize any theme by creating themes/&lt;name&gt;.yaml next to book.yaml (see 'mdpress themes show &lt;name&gt;')</p>
        </div>
    </div>
</body>
</html>
`)

	return sb.String()
}

// generateThemePreviewSection creates an HTML section for a single theme.
// The sample content is rendered inside a sandboxed iframe whose stylesheet
// is the theme's actual pipeline CSS (theme.ToCSS), so headings, links,
// inline-code chips, code blocks, blockquotes, and zebra-striped tables all
// show the exact styling that builds produce.
func generateThemePreviewSection(thm themeInfo) string {
	displayName := thm.displayName
	if thm.isDefault {
		displayName += " (default)"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `            <div class="theme-section">
                <div class="theme-header" style="background: linear-gradient(135deg, %s 0%%, %s 100%%);">
                    <div class="theme-name">%s</div>
                    <div class="theme-code">%s</div>
                </div>

                <div class="theme-info">
                    %s
                </div>
`,
		thm.colors.primary,
		thm.colors.secondary,
		html.EscapeString(displayName),
		html.EscapeString(thm.name),
		html.EscapeString(thm.description),
	)

	if len(thm.features) > 0 {
		sb.WriteString(`
                <ul class="theme-props">
`)
		for _, feature := range thm.features {
			fmt.Fprintf(&sb, "                    <li>%s</li>\n", html.EscapeString(feature))
		}
		sb.WriteString("                </ul>\n")
	}

	sb.WriteString(`
                <div class="theme-colors">
`)
	for _, swatch := range []struct{ label, value string }{
		{"Heading", thm.colors.primary},
		{"Link", thm.colors.secondary},
		{"Accent", thm.colors.accent},
		{"Text", thm.colors.text},
		{"Background", thm.colors.background},
		{"CodeBg", thm.colors.codeBg},
		{"CodeText", thm.colors.codeText},
		{"Border", thm.colors.border},
	} {
		fmt.Fprintf(&sb, `                    <div class="color-swatch">
                        <div class="color-box" style="background-color: %s;"></div>
                        <div class="color-label">%s</div>
                        <div class="color-value">%s</div>
                    </div>
`, swatch.value, html.EscapeString(swatch.label), html.EscapeString(swatch.value))
	}
	sb.WriteString("                </div>\n")

	// Embed the sample document in a srcdoc iframe so each theme's global
	// stylesheet (body, headings, tables, ...) stays isolated per sample.
	fmt.Fprintf(&sb, `
                <iframe class="theme-sample" title="%s sample" srcdoc="%s"></iframe>
            </div>
`,
		html.EscapeString(thm.name),
		html.EscapeString(themeSampleDocument(thm)),
	)

	return sb.String()
}

// themeSampleDocument builds the standalone HTML document shown inside a
// theme's preview iframe, styled by the theme's real ToCSS output.
func themeSampleDocument(thm themeInfo) string {
	css := ""
	if thm.theme != nil {
		css = thm.theme.ToCSS()
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<style>\n")
	sb.WriteString(css)
	// Preview-only accommodation: the theme's page margins are meant for
	// print pagination; keep the sample compact inside the card.
	sb.WriteString("\nbody { margin: 18px 22px; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n")
	sb.WriteString(`<h1>Sample Heading</h1>
<p>This is a sample paragraph showcasing the theme's text styling with a <a href="#">sample link</a> and <code>inline code</code>. The quick brown fox jumps over the lazy dog.</p>
<h2>Section Heading</h2>
<blockquote>A well-designed theme enhances readability and creates a professional appearance for your documentation.</blockquote>
<pre><code>// Sample code block
func greet(name string) string {
    return "Hello, " + name
}</code></pre>
<table>
<thead>
<tr><th>Feature</th><th>Value</th></tr>
</thead>
<tbody>
<tr><td>Table header</td><td>tinted with accent underline</td></tr>
<tr><td>Zebra striping</td><td>even rows tinted</td></tr>
<tr><td>Inline code</td><td>subtle chip background</td></tr>
<tr><td>Links</td><td>underline on hover</td></tr>
</tbody>
</table>
`)
	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}
