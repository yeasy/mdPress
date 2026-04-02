package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "Manage built-in themes",
	Long: `List and inspect the built-in themes available in mdpress.

Subcommands:
  themes list      List all available themes and color palettes
  themes show      Show theme details and sample configuration
  themes preview   Generate an HTML preview of all themes

Examples:
  mdpress themes list
  mdpress themes show technical
  mdpress themes show elegant
  mdpress themes preview`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return nil
	},
}

var themesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all themes",
	Long: `List every built-in theme available in mdpress.

Each theme includes its name, description, and color settings.

Example:
  mdpress themes list`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return executeThemesList()
	},
}

var themesShowCmd = &cobra.Command{
	Use:   "show <theme-name>",
	Short: "Show theme details",
	Long: `Show detailed information and configuration hints for a theme.

Examples:
  mdpress themes show technical
  mdpress themes show elegant
  mdpress themes show minimal`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeThemesShow(args[0])
	},
}

var themesPreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Generate an HTML preview of all themes",
	Long: `Generate a self-contained HTML file that showcases all built-in themes applied to sample content.

The preview page displays each theme with sample elements including headings, paragraphs, code blocks, tables, and blockquotes.

Examples:
  mdpress themes preview
  mdpress themes preview --output custom-preview.html`,

	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		return executeThemesPreview(output)
	},
}

func init() {
	// Register theme subcommands.
	themesCmd.AddCommand(themesListCmd)
	themesCmd.AddCommand(themesShowCmd)
	themesCmd.AddCommand(themesPreviewCmd)

	// Add flags for preview command
	themesPreviewCmd.Flags().StringP("output", "o", "themes-preview.html", "Output file path for the HTML preview")
}

// themeInfo describes a built-in theme.
type themeInfo struct {
	name        string
	displayName string
	description string
	author      string
	version     string
	license     string
	features    []string
	colors      themeColors
}

// themeColors stores theme color values.
type themeColors struct {
	primary    string
	secondary  string
	accent     string
	text       string
	background string
	codeBg     string
}

// getAvailableThemes returns the built-in themes.
func getAvailableThemes() []themeInfo {
	return []themeInfo{
		{
			name:        "technical",
			displayName: "Technical",
			description: "A clean and professional style for technical books and documentation.",
			author:      "mdpress Team",
			version:     "1.0.0",
			license:     "MIT",
			features: []string{
				"Clear typography",
				"Code highlighting support",
				"Responsive layout",
				"Professional font pairing",
			},
			colors: themeColors{
				primary:    "#1A5490",
				secondary:  "#0066CC",
				accent:     "#0066CC",
				text:       "#2C3E50",
				background: "#FFFFFF",
				codeBg:     "#F5F7F9",
			},
		},
		{
			name:        "elegant",
			displayName: "Elegant",
			description: "A refined style suited for essays, academic writing, and literary work.",
			author:      "mdpress Team",
			version:     "1.0.0",
			license:     "MIT",
			features: []string{
				"Classic typography",
				"Decorative accents",
				"Careful spacing",
				"Chapter dividers",
			},
			colors: themeColors{
				primary:    "#34495e",
				secondary:  "#16a085",
				accent:     "#d35400",
				text:       "#2c3e50",
				background: "#ecf0f1",
				codeBg:     "#e8e8e8",
			},
		},
		{
			name:        "minimal",
			displayName: "Minimal",
			description: "A minimal design focused on clarity and efficient reading.",
			author:      "mdpress Team",
			version:     "1.0.0",
			license:     "MIT",
			features: []string{
				"Minimal styling",
				"High contrast",
				"Fast loading",
				"Print-friendly",
			},
			colors: themeColors{
				primary:    "#000000",
				secondary:  "#555555",
				accent:     "#0066cc",
				text:       "#000000",
				background: "#ffffff",
				codeBg:     "#f0f0f0",
			},
		},
	}
}

// executeThemesList prints the built-in themes.
func executeThemesList() error {
	logger := slog.Default()
	logger.Info("Listing available themes")

	themes := getAvailableThemes()

	fmt.Println()
	fmt.Println("Available themes:")
	fmt.Println()

	for i, theme := range themes {
		fmt.Printf("%d. %s (%s)\n", i+1, theme.displayName, theme.name)
		fmt.Printf("   Description: %s\n", theme.description)
		fmt.Printf("   Author: %s | Version: %s | License: %s\n", theme.author, theme.version, theme.license)
		fmt.Printf("   Colors: %s (primary) / %s (secondary) / %s (accent)\n", theme.colors.primary, theme.colors.secondary, theme.colors.accent)

		if len(theme.features) > 0 {
			fmt.Printf("   Features:\n")
			for _, feature := range theme.features {
				fmt.Printf("     - %s\n", feature)
			}
		}
		fmt.Println()
	}

	fmt.Printf("Run 'mdpress themes show <theme-name>' to view theme details.\n")
	fmt.Printf("Example: mdpress themes show elegant\n\n")

	return nil
}

// executeThemesShow prints details for a single theme.
func executeThemesShow(themeName string) error {
	logger := slog.Default()
	logger.Debug("Showing theme details", slog.String("theme", themeName))

	themes := getAvailableThemes()

	var thm *themeInfo
	for i := range themes {
		if themes[i].name == themeName {
			thm = &themes[i]
			break
		}
	}

	if thm == nil {
		return fmt.Errorf("theme not found: %q\n\nRun 'mdpress themes list' to view available themes", themeName)
	}

	// Print theme details.
	fmt.Println()
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Printf("Theme: %s (%s)\n", thm.displayName, thm.name)
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("Description: %s\n", thm.description)
	fmt.Printf("Author:      %s\n", thm.author)
	fmt.Printf("Version:     %s\n", thm.version)
	fmt.Printf("License:     %s\n", thm.license)
	fmt.Println()

	fmt.Println("Features:")
	for _, feature := range thm.features {
		fmt.Printf("  ✓ %s\n", feature)
	}
	fmt.Println()

	fmt.Println("Colors:")
	fmt.Printf("  Primary:    %s\n", thm.colors.primary)
	fmt.Printf("  Secondary:  %s\n", thm.colors.secondary)
	fmt.Printf("  Accent:     %s\n", thm.colors.accent)
	fmt.Printf("  Text:       %s\n", thm.colors.text)
	fmt.Printf("  Background: %s\n", thm.colors.background)
	fmt.Printf("  CodeBg:     %s\n", thm.colors.codeBg)
	fmt.Println()

	fmt.Println("Configuration:")
	fmt.Printf("  Use this theme in book.yaml:\n")
	fmt.Printf("    theme: \"%s\"\n", thm.name)
	fmt.Println()

	fmt.Println("Customization:")
	fmt.Printf("  Create 'themes/%s/' in your project to customize this theme.\n", thm.name)
	fmt.Printf("  Reference: https://github.com/yeasy/mdpress/tree/main/docs/manual/en/themes\n")
	fmt.Println()

	return nil
}
