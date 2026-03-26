// migrate.go implements the `mdpress migrate` command.
//
// It detects a GitBook / HonKit project in the given directory, converts
// book.json to book.yaml, rewrites GitBook-specific template tags in Markdown
// files, and prints a migration report.
package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/pkg/utils"
	"gopkg.in/yaml.v3"
)

// migrateCmd implements the migrate sub-command.
var migrateCmd = &cobra.Command{
	Use:   "migrate [directory]",
	Short: "Migrate a GitBook / HonKit project to mdpress",
	Long: `Detect a GitBook or HonKit project and convert it to mdpress format.

What migrate does:
  1. Reads book.json and converts it to book.yaml.
  2. Rewrites GitBook template tags in Markdown files:
       {% hint style="info" %}...{% endhint %}  →  blockquote
       {% tabs %}...{% endtabs %}               →  kept with a comment
       {% code title="..." %}...{% endcode %}   →  fenced code block
  3. Keeps SUMMARY.md intact (mdpress already supports this format).
  4. Prints a migration report listing all changes made.

Examples:
  mdpress migrate
  mdpress migrate ./my-gitbook
  mdpress migrate --dry-run ./my-gitbook`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return fmt.Errorf("failed to read --dry-run flag: %w", err)
		}
		return executeMigrate(dir, dryRun)
	},
}

func init() {
	migrateCmd.Flags().Bool("dry-run", false, "Preview changes without writing any files")
}

// ---- GitBook config model ----

// gitBookConfig represents the structure of a GitBook book.json file.
type gitBookConfig struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	Language    string         `json:"language"`
	Plugins     []string       `json:"plugins"`
	GitBook     string         `json:"gitbook"`
	Structure   map[string]any `json:"structure"`
	Links       map[string]any `json:"links"`
	Styles      map[string]any `json:"styles"`
	Variables   map[string]any `json:"variables"`
}

// ---- Migration report ----

// migrateReport accumulates messages describing what the migration did or will do.
type migrateReport struct {
	created  []string
	modified []string
	skipped  []string
	warnings []string
}

func (r *migrateReport) addCreated(msg string)  { r.created = append(r.created, msg) }
func (r *migrateReport) addModified(msg string) { r.modified = append(r.modified, msg) }
func (r *migrateReport) addSkipped(msg string)  { r.skipped = append(r.skipped, msg) }
func (r *migrateReport) addWarning(msg string)  { r.warnings = append(r.warnings, msg) }

func (r *migrateReport) print() {
	fmt.Println("\n=== Migration Report ===")
	if len(r.created) > 0 {
		fmt.Println("\nCreated:")
		for _, m := range r.created {
			fmt.Println("  + " + m)
		}
	}
	if len(r.modified) > 0 {
		fmt.Println("\nModified:")
		for _, m := range r.modified {
			fmt.Println("  ~ " + m)
		}
	}
	if len(r.skipped) > 0 {
		fmt.Println("\nSkipped:")
		for _, m := range r.skipped {
			fmt.Println("  - " + m)
		}
	}
	if len(r.warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, m := range r.warnings {
			fmt.Println("  ! " + m)
		}
	}
	fmt.Println()
}

// ---- Main migration logic ----

func executeMigrate(dir string, dryRun bool) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("cannot resolve directory %q: %w", dir, err)
	}

	fmt.Printf("Scanning %s for a GitBook / HonKit project...\n", absDir)

	report := &migrateReport{}

	// Detect GitBook structure.
	bookJSONPath := filepath.Join(absDir, "book.json")
	summaryPath := filepath.Join(absDir, "SUMMARY.md")

	hasBookJSON := utils.FileExists(bookJSONPath)
	hasSummary := utils.FileExists(summaryPath)

	if !hasBookJSON && !hasSummary {
		return fmt.Errorf("no GitBook project detected in %s (expected book.json or SUMMARY.md)", absDir)
	}

	var gb *gitBookConfig
	if !hasBookJSON {
		report.addSkipped("book.json not found — skipping config conversion")
	} else {
		var err error
		gb, err = migrateBookJSON(bookJSONPath, absDir, dryRun, report)
		if err != nil {
			return err
		}
	}

	if hasSummary {
		report.addSkipped("SUMMARY.md is already compatible with mdpress — no changes needed")
	}

	// Rewrite GitBook template tags in all Markdown files.
	if err := migrateMarkdownFiles(absDir, dryRun, report); err != nil {
		return err
	}

	if dryRun {
		fmt.Println("\n[dry-run] No files were written.")
	}

	// Summarize known GitBook plugins that have no mdpress equivalent yet.
	if gb != nil {
		summarisePluginWarnings(gb, report)
	}

	report.print()

	fmt.Println("Migration complete.  Run `mdpress validate` to check the result.")
	return nil
}

// migrateBookJSON converts book.json → book.yaml.
// Returns the parsed gitBookConfig for further processing or error if parsing fails.
func migrateBookJSON(bookJSONPath, projectDir string, dryRun bool, report *migrateReport) (*gitBookConfig, error) {
	const maxBookJSONSize = 10 * 1024 * 1024 // 10 MB
	info, err := os.Stat(bookJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat book.json: %w", err)
	}
	if info.Size() > maxBookJSONSize {
		return nil, fmt.Errorf("book.json is too large (%d bytes; max %d bytes)", info.Size(), maxBookJSONSize)
	}
	data, err := os.ReadFile(bookJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read book.json: %w", err)
	}

	var gb gitBookConfig
	if err := json.Unmarshal(data, &gb); err != nil {
		return nil, fmt.Errorf("failed to parse book.json: %w", err)
	}

	// Build the mdpress book.yaml structure using only the basic Go types that
	// yaml.v3 can marshal correctly.
	bookYAML := map[string]any{
		"book": map[string]any{
			"title":       nonEmpty(gb.Title, "Untitled Book"),
			"author":      nonEmpty(gb.Author, "Unknown"),
			"language":    nonEmpty(gb.Language, "en"),
			"description": gb.Description,
		},
		"style": map[string]any{
			"theme": "technical",
		},
		"output": map[string]any{
			"filename": "output",
			"toc":      true,
			"cover":    false,
		},
	}

	yamlBytes, err := yaml.Marshal(bookYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to generate book.yaml: %w", err)
	}

	header := "# Generated by `mdpress migrate` from book.json\n" +
		"# Review and adjust as needed.\n\n"

	outPath := filepath.Join(projectDir, "book.yaml")
	if dryRun {
		fmt.Printf("[dry-run] Would write %s\n", outPath)
		report.addCreated("book.yaml (dry-run)")
		return &gb, nil
	}

	if utils.FileExists(outPath) {
		slog.Warn("overwriting existing book.yaml", slog.String("path", outPath))
	}

	if err := os.WriteFile(outPath, []byte(header+string(yamlBytes)), 0644); err != nil {
		return nil, fmt.Errorf("failed to write book.yaml: %w", err)
	}
	report.addCreated("book.yaml")
	return &gb, nil
}

// nonEmpty returns s when non-empty, otherwise returns fallback.
func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

// ---- GitBook syntax rewriting ----

// hintRE matches {% hint style="TYPE" %}...{% endhint %} (possibly multi-line).
var hintRE = regexp.MustCompile(`(?s)\{%\s*hint\s+style="([^"]+)"\s*%\}(.*?)\{%\s*endhint\s*%\}`)

// codeTagRE matches {% code title="TITLE" %}...{% endcode %}.
var codeTagRE = regexp.MustCompile(`(?s)\{%\s*code\s+[^%]*%\}(.*?)\{%\s*endcode\s*%\}`)

// tabsRE matches the entire {% tabs %}...{% endtabs %} block.
var tabsRE = regexp.MustCompile(`(?s)\{%\s*tabs\s*%\}.*?\{%\s*endtabs\s*%\}`)

// tabTitleRE matches {% tab title="NAME" %}...{% endtab %} inside a tabs block.
var tabTitleRE = regexp.MustCompile(`\{%\s*tab\s+title="([^"]+)"\s*%\}`)

// stripTabsRE, stripEndtabsRE, and stripEndtabRE remove the outer tab markers
// left after tab title replacement.
var (
	stripTabsRE    = regexp.MustCompile(`\{%\s*tabs\s*%\}`)
	stripEndtabsRE = regexp.MustCompile(`\{%\s*endtabs\s*%\}`)
	stripEndtabRE  = regexp.MustCompile(`\{%\s*endtab\s*%\}`)
)

// rewriteGitBookSyntax converts known GitBook template tags to standard Markdown.
func rewriteGitBookSyntax(content string) (string, bool) {
	original := content

	// {% hint style="TYPE" %}BODY{% endhint %} → blockquote
	content = hintRE.ReplaceAllStringFunc(content, func(match string) string {
		groups := hintRE.FindStringSubmatch(match)
		if len(groups) < 3 {
			return match
		}
		style := groups[1]
		body := strings.TrimSpace(groups[2])
		// Prefix each line of the body with "> " to form a blockquote.
		lines := strings.Split(body, "\n")
		quoted := make([]string, 0, len(lines)+1)
		quoted = append(quoted, fmt.Sprintf("> **%s:** ", strings.ToUpper(style)))
		for _, line := range lines {
			quoted = append(quoted, "> "+line)
		}
		return strings.Join(quoted, "\n")
	})

	// {% code title="TITLE" %}BODY{% endcode %} → ```\nBODY\n```
	content = codeTagRE.ReplaceAllStringFunc(content, func(match string) string {
		groups := codeTagRE.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		body := strings.TrimSpace(groups[1])
		return "```\n" + body + "\n```"
	})

	// {% tabs %}...{% endtabs %}: flatten each tab to a heading + content.
	content = tabsRE.ReplaceAllStringFunc(content, func(match string) string {
		// Replace {% tab title="NAME" %} with a Markdown heading.
		result := tabTitleRE.ReplaceAllStringFunc(match, func(m string) string {
			titleGroups := tabTitleRE.FindStringSubmatch(m)
			if len(titleGroups) < 2 {
				return m
			}
			return "#### " + titleGroups[1]
		})
		// Strip the outer {% tabs %} / {% endtabs %} and {% endtab %} markers.
		result = stripTabsRE.ReplaceAllString(result, "")
		result = stripEndtabsRE.ReplaceAllString(result, "")
		result = stripEndtabRE.ReplaceAllString(result, "")
		return strings.TrimSpace(result)
	})

	return content, content != original
}

// migrateMarkdownFiles walks projectDir, rewrites GitBook syntax in .md files.
func migrateMarkdownFiles(projectDir string, dryRun bool, report *migrateReport) error {
	return filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden and vendor directories.
			base := info.Name()
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		// Reject unreasonably large markdown files (50 MB).
		const maxMigrateFileSize = 50 << 20
		if info.Size() > maxMigrateFileSize {
			report.addWarning(fmt.Sprintf("skipping %s: file too large (%d bytes)", path, info.Size()))
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			report.addWarning(fmt.Sprintf("cannot read %s: %v", path, err))
			return nil
		}

		rewritten, changed := rewriteGitBookSyntax(string(data))
		if !changed {
			return nil
		}

		relPath, err := filepath.Rel(projectDir, path)
		if err != nil {
			relPath = path // fallback to absolute path
		}
		if dryRun {
			fmt.Printf("[dry-run] Would rewrite GitBook tags in %s\n", relPath)
			report.addModified(relPath + " (dry-run)")
			return nil
		}

		if err := os.WriteFile(path, []byte(rewritten), info.Mode()); err != nil {
			report.addWarning(fmt.Sprintf("failed to write %s: %v", relPath, err))
			return nil
		}
		report.addModified(relPath)
		return nil
	})
}

// summarisePluginWarnings warns about GitBook plugins that have no known mdpress equivalent.
// It accepts the already-parsed gitBookConfig to avoid redundantly re-reading book.json.
func summarisePluginWarnings(gb *gitBookConfig, report *migrateReport) {
	if gb == nil {
		return
	}
	// Plugins that mdpress handles natively or via the plugin system.
	knownPlugins := map[string]bool{
		"highlight":     true,
		"search":        true,
		"lunr":          true,
		"sharing":       true,
		"fontsettings":  true,
		"theme-default": true,
	}
	for _, p := range gb.Plugins {
		p = strings.TrimPrefix(p, "-") // "-plugin" disables it in GitBook
		p = strings.TrimSpace(p)
		if p == "" || knownPlugins[p] {
			continue
		}
		report.addWarning(fmt.Sprintf(
			"GitBook plugin %q has no direct mdpress equivalent — check the mdpress plugin registry",
			p,
		))
	}
}
