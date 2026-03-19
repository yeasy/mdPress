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

// Theme describes a built-in theme.
type Theme struct {
	Name        string
	DisplayName string
	Description string
	Author      string
	Version     string
	License     string
	Features    []string
	Colors      ThemeColors
}

// ThemeColors stores theme color values.
type ThemeColors struct {
	Primary    string
	Secondary  string
	Accent     string
	Text       string
	Background string
	CodeBg     string
}

// getAvailableThemes returns the built-in themes.
func getAvailableThemes() []Theme {
	return []Theme{
		{
			Name:        "technical",
			DisplayName: "Technical",
			Description: "A clean and professional style for technical books and documentation.",
			Author:      "mdpress Team",
			Version:     "1.0.0",
			License:     "MIT",
			Features: []string{
				"Clear typography",
				"Code highlighting support",
				"Responsive layout",
				"Professional font pairing",
			},
			Colors: ThemeColors{
				Primary:    "#1A5490",
				Secondary:  "#0066CC",
				Accent:     "#0066CC",
				Text:       "#2C3E50",
				Background: "#FFFFFF",
				CodeBg:     "#F5F7F9",
			},
		},
		{
			Name:        "elegant",
			DisplayName: "Elegant",
			Description: "A refined style suited for essays, academic writing, and literary work.",
			Author:      "mdpress Team",
			Version:     "1.0.0",
			License:     "MIT",
			Features: []string{
				"Classic typography",
				"Decorative accents",
				"Careful spacing",
				"Chapter dividers",
			},
			Colors: ThemeColors{
				Primary:    "#34495e",
				Secondary:  "#16a085",
				Accent:     "#d35400",
				Text:       "#2c3e50",
				Background: "#ecf0f1",
				CodeBg:     "#e8e8e8",
			},
		},
		{
			Name:        "minimal",
			DisplayName: "Minimal",
			Description: "A minimal design focused on clarity and efficient reading.",
			Author:      "mdpress Team",
			Version:     "1.0.0",
			License:     "MIT",
			Features: []string{
				"Minimal styling",
				"High contrast",
				"Fast loading",
				"Print-friendly",
			},
			Colors: ThemeColors{
				Primary:    "#000000",
				Secondary:  "#555555",
				Accent:     "#0066cc",
				Text:       "#000000",
				Background: "#ffffff",
				CodeBg:     "#f0f0f0",
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
		fmt.Printf("%d. %s (%s)\n", i+1, theme.DisplayName, theme.Name)
		fmt.Printf("   Description: %s\n", theme.Description)
		fmt.Printf("   Author: %s | Version: %s | License: %s\n", theme.Author, theme.Version, theme.License)
		fmt.Printf("   Colors: %s (primary) / %s (secondary) / %s (accent)\n", theme.Colors.Primary, theme.Colors.Secondary, theme.Colors.Accent)

		if len(theme.Features) > 0 {
			fmt.Printf("   Features:\n")
			for _, feature := range theme.Features {
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

	var theme *Theme
	for i := range themes {
		if themes[i].Name == themeName {
			theme = &themes[i]
			break
		}
	}

	if theme == nil {
		return fmt.Errorf("theme not found: %q\n\nRun 'mdpress themes list' to view available themes", themeName)
	}

	// Print theme details.
	fmt.Println()
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Printf("Theme: %s (%s)\n", theme.DisplayName, theme.Name)
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("Description: %s\n", theme.Description)
	fmt.Printf("Author:      %s\n", theme.Author)
	fmt.Printf("Version:     %s\n", theme.Version)
	fmt.Printf("License:     %s\n", theme.License)
	fmt.Println()

	fmt.Println("Features:")
	for _, feature := range theme.Features {
		fmt.Printf("  ✓ %s\n", feature)
	}
	fmt.Println()

	fmt.Println("Colors:")
	fmt.Printf("  Primary:    %s\n", theme.Colors.Primary)
	fmt.Printf("  Secondary:  %s\n", theme.Colors.Secondary)
	fmt.Printf("  Accent:     %s\n", theme.Colors.Accent)
	fmt.Printf("  Text:       %s\n", theme.Colors.Text)
	fmt.Printf("  Background: %s\n", theme.Colors.Background)
	fmt.Printf("  CodeBg:     %s\n", theme.Colors.CodeBg)
	fmt.Println()

	fmt.Println("Configuration:")
	fmt.Printf("  Use this theme in book.yaml:\n")
	fmt.Printf("    theme: \"%s\"\n", theme.Name)
	fmt.Println()

	fmt.Println("Customization:")
	fmt.Printf("  Create 'themes/%s/' in your project to customize this theme.\n", theme.Name)
	fmt.Printf("  Reference: https://github.com/yeasy/mdpress/tree/main/docs/themes\n")
	fmt.Println()

	return nil
}
