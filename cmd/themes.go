package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeasy/mdpress/internal/theme"
)

// defaultThemeName is the theme used when book.yaml does not set style.theme.
const defaultThemeName = "technical"

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
		// cobra routes an unmatched sub-command here. Returning nil made a
		// typo look like success: `mdpress themes lst` printed nothing and
		// exited 0.
		hint := ""
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			hint = fmt.Sprintf("\n\nDid you mean this?\n\t%s", strings.Join(suggestions, "\n\t"))
		}
		return fmt.Errorf("unknown themes sub-command %q%s\n\nRun 'mdpress themes --help' to see the available sub-commands", args[0], hint)
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

The preview renders each theme with the exact stylesheet the build pipeline
produces, including headings, links, inline code, code blocks, blockquotes,
and tables with header and zebra striping.

Examples:
  mdpress themes preview
  mdpress themes preview --output custom-preview.html`,

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		// The documented example writes into ./artifacts/, which did not exist
		// and made the command fail with a bare ENOENT from os.WriteFile.
		if dir := filepath.Dir(output); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("failed to create preview output directory %s: %w", dir, err)
			}
		}
		return executeThemesPreview(output)
	},
}

func init() {
	// cobra only defaults this inside its own suggestion path; SuggestionsFor
	// compares against the raw value, and 0 means "never suggest".
	themesCmd.SuggestionsMinimumDistance = 2

	// Register theme subcommands.
	themesCmd.AddCommand(themesListCmd)
	themesCmd.AddCommand(themesShowCmd)
	themesCmd.AddCommand(themesPreviewCmd)

	// Add flags for preview command
	themesPreviewCmd.Flags().StringP("output", "o", "themes-preview.html", "Output file path for the HTML preview")
}

// themeInfo describes a theme for CLI display. Every palette and property
// value is derived from the live theme definitions in internal/theme, so the
// CLI output can never drift from what builds actually produce.
type themeInfo struct {
	name        string
	displayName string
	description string
	isDefault   bool
	features    []string
	colors      themeColors
	theme       *theme.Theme
}

// themeColors stores theme color values mapped from theme.ColorScheme
// (primary=heading, secondary=link).
type themeColors struct {
	primary    string
	secondary  string
	accent     string
	text       string
	background string
	codeBg     string
	codeText   string
	border     string
}

// getAvailableThemes returns the built-in themes, default theme first.
func getAvailableThemes() []themeInfo {
	tm := theme.NewThemeManager()

	// tm.List() is sorted; surface the default theme first.
	names := tm.List()
	ordered := make([]string, 0, len(names))
	for _, name := range names {
		if name == defaultThemeName {
			ordered = append([]string{name}, ordered...)
		} else {
			ordered = append(ordered, name)
		}
	}

	infos := make([]themeInfo, 0, len(ordered))
	for _, name := range ordered {
		thm, err := tm.Get(name)
		if err != nil || thm == nil {
			continue
		}
		infos = append(infos, newThemeInfo(thm))
	}
	return infos
}

// newThemeInfo builds the CLI display record from a live theme definition.
func newThemeInfo(thm *theme.Theme) themeInfo {
	return themeInfo{
		name:        thm.Name,
		displayName: displayThemeName(thm.Name),
		description: theme.GetThemeDescription(thm.Name),
		isDefault:   thm.Name == defaultThemeName,
		features:    themeFeatures(thm),
		colors: themeColors{
			primary:    thm.Colors.Heading,
			secondary:  thm.Colors.Link,
			accent:     thm.Colors.Accent,
			text:       thm.Colors.Text,
			background: thm.Colors.Background,
			codeBg:     thm.Colors.CodeBg,
			codeText:   thm.Colors.CodeText,
			border:     thm.Colors.Border,
		},
		theme: thm,
	}
}

// displayThemeName renders a theme name for headings (first letter uppercased).
func displayThemeName(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// primaryFontName returns the first entry of a CSS font-family list with
// surrounding quotes stripped.
func primaryFontName(family string) string {
	first := family
	if i := strings.Index(family, ","); i >= 0 {
		first = family[:i]
	}
	return strings.Trim(strings.TrimSpace(first), `'"`)
}

// themeFeatures derives display features from the live theme settings.
func themeFeatures(thm *theme.Theme) []string {
	return []string{
		fmt.Sprintf("Font: %s", primaryFontName(thm.FontFamily)),
		fmt.Sprintf("Base size %dpt, line height %.2f", thm.FontSize, thm.LineHeight),
		fmt.Sprintf("Code highlighting: %s", thm.CodeTheme),
		fmt.Sprintf("Page %s, margins %.0f/%.0f/%.0f/%.0f mm (top/right/bottom/left)",
			thm.PageSize, thm.Margins.Top, thm.Margins.Right, thm.Margins.Bottom, thm.Margins.Left),
	}
}

// executeThemesList prints the built-in themes.
func executeThemesList() error {
	themes := getAvailableThemes()

	fmt.Println()
	fmt.Println("Available themes:")
	fmt.Println()

	for i, thm := range themes {
		defaultMark := ""
		if thm.isDefault {
			defaultMark = " [default]"
		}
		fmt.Printf("%d. %s (%s)%s\n", i+1, thm.displayName, thm.name, defaultMark)
		fmt.Printf("   Description: %s\n", thm.description)
		fmt.Printf("   Colors: %s (heading) / %s (link) / %s (accent) / %s (background)\n",
			thm.colors.primary, thm.colors.secondary, thm.colors.accent, thm.colors.background)

		if len(thm.features) > 0 {
			fmt.Printf("   Properties:\n")
			for _, feature := range thm.features {
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
	title := fmt.Sprintf("Theme: %s (%s)", thm.displayName, thm.name)
	if thm.isDefault {
		title += " [default]"
	}
	fmt.Println(title)
	fmt.Println("═════════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("Description: %s\n", thm.description)
	fmt.Println()

	t := thm.theme
	fmt.Println("Typography:")
	fmt.Printf("  Font family: %s\n", t.FontFamily)
	fmt.Printf("  Font size:   %dpt\n", t.FontSize)
	fmt.Printf("  Line height: %.2f\n", t.LineHeight)
	fmt.Printf("  Code theme:  %s\n", t.CodeTheme)
	fmt.Println()

	fmt.Println("Colors:")
	fmt.Printf("  Text:       %s\n", t.Colors.Text)
	fmt.Printf("  Background: %s\n", t.Colors.Background)
	fmt.Printf("  Heading:    %s\n", t.Colors.Heading)
	fmt.Printf("  Link:       %s\n", t.Colors.Link)
	fmt.Printf("  CodeBg:     %s\n", t.Colors.CodeBg)
	fmt.Printf("  CodeText:   %s\n", t.Colors.CodeText)
	fmt.Printf("  Accent:     %s\n", t.Colors.Accent)
	fmt.Printf("  Border:     %s\n", t.Colors.Border)
	fmt.Println()

	fmt.Println("Page:")
	fmt.Printf("  Size:    %s\n", t.PageSize)
	fmt.Printf("  Margins: %.0f/%.0f/%.0f/%.0f mm (top/right/bottom/left)\n",
		t.Margins.Top, t.Margins.Right, t.Margins.Bottom, t.Margins.Left)
	fmt.Println()

	fmt.Println("Configuration:")
	fmt.Printf("  Use this theme in book.yaml:\n")
	fmt.Printf("    style:\n")
	fmt.Printf("      theme: \"%s\"\n", thm.name)
	fmt.Println()

	fmt.Println("Customization:")
	fmt.Printf("  Create 'themes/%s.yaml' next to your book.yaml to replace this theme\n", thm.name)
	fmt.Printf("  for that project, or point style.theme at any theme YAML file, e.g.:\n")
	fmt.Printf("    style:\n")
	fmt.Printf("      theme: \"my-theme.yaml\"\n")
	fmt.Println()
	fmt.Printf("  Starting point with this theme's current values (page_size, font_size,\n")
	fmt.Printf("  line_height, colors.text, and colors.background are required; everything\n")
	fmt.Printf("  else is optional):\n")
	fmt.Println()
	fmt.Print(themeYAMLExample(t, "    "))
	fmt.Println()
	fmt.Printf("  Reference: https://github.com/yeasy/mdpress/tree/main/docs/manual/en/themes\n")
	fmt.Println()

	return nil
}

// themeYAMLExample renders a theme as a ready-to-edit YAML snippet matching
// the schema accepted by themes/<name>.yaml (and style.theme file paths).
func themeYAMLExample(t *theme.Theme, indent string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%sname: %s\n", indent, t.Name)
	fmt.Fprintf(&sb, "%spage_size: %s\n", indent, t.PageSize)
	fmt.Fprintf(&sb, "%sfont_size: %d\n", indent, t.FontSize)
	fmt.Fprintf(&sb, "%sline_height: %.2f\n", indent, t.LineHeight)
	fmt.Fprintf(&sb, "%sfont_family: \"%s\"\n", indent, t.FontFamily)
	fmt.Fprintf(&sb, "%scode_theme: %s\n", indent, t.CodeTheme)
	fmt.Fprintf(&sb, "%scolors:\n", indent)
	fmt.Fprintf(&sb, "%s  text: \"%s\"\n", indent, t.Colors.Text)
	fmt.Fprintf(&sb, "%s  background: \"%s\"\n", indent, t.Colors.Background)
	fmt.Fprintf(&sb, "%s  heading: \"%s\"\n", indent, t.Colors.Heading)
	fmt.Fprintf(&sb, "%s  link: \"%s\"\n", indent, t.Colors.Link)
	fmt.Fprintf(&sb, "%s  code_bg: \"%s\"\n", indent, t.Colors.CodeBg)
	fmt.Fprintf(&sb, "%s  code_text: \"%s\"\n", indent, t.Colors.CodeText)
	fmt.Fprintf(&sb, "%s  accent: \"%s\"\n", indent, t.Colors.Accent)
	fmt.Fprintf(&sb, "%s  border: \"%s\"\n", indent, t.Colors.Border)
	fmt.Fprintf(&sb, "%smargins:\n", indent)
	fmt.Fprintf(&sb, "%s  top: %.0f\n", indent, t.Margins.Top)
	fmt.Fprintf(&sb, "%s  bottom: %.0f\n", indent, t.Margins.Bottom)
	fmt.Fprintf(&sb, "%s  left: %.0f\n", indent, t.Margins.Left)
	fmt.Fprintf(&sb, "%s  right: %.0f\n", indent, t.Margins.Right)
	return sb.String()
}
