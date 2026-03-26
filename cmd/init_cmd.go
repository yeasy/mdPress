// init_cmd.go implements the init subcommand.
// It scans Markdown files, extracts structure and titles, and generates book.yaml.
// When the target directory is empty it creates starter files.
// Interactive mode collects project metadata, with sensible defaults for non-interactive terminals.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/pkg/utils"
)

// initInteractive controls whether interactive mode is enabled.
var initInteractive bool

// scanSkipDirs lists directories that scanMarkdownFiles should skip.
var scanSkipDirs = map[string]bool{"node_modules": true, "vendor": true, ".git": true}

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a book project by scanning Markdown files",
	Long: `Scan the target directory for .md files, extract structure and titles,
and generate a book.yaml configuration file automatically.

If the directory contains no .md files, mdpress creates a starter template.
Use --interactive to answer a few prompts for title, author, language, and theme.

Examples:
  mdpress init
  mdpress init ./my-book
  mdpress init --interactive
  mdpress init ./my-book -i`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		return executeInit(cmd.Context(), dir)
	},
}

func init() {
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "Enable interactive prompts")
}

// initAnswers stores interactive prompt results.
type initAnswers struct {
	Title    string
	Author   string
	Language string
	Theme    string
}

// isTerminalInteractive reports whether stdin is interactive.
func isTerminalInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Character devices indicate an interactive terminal.
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// promptUser asks a question and falls back to the default value on empty input.
func promptUser(reader *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s %s[%s]%s: ", question, utils.Dim(""), utils.Dim(defaultVal), "")
	} else {
		fmt.Printf("  %s: ", question)
	}
	answer, err := reader.ReadString('\n')
	// On error (including EOF), fall back to default. In non-interactive mode,
	// EOF is expected and harmless.
	if err != nil {
		return defaultVal
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultVal
	}
	return answer
}

// promptChoice shows a list of options and returns the selected value.
func promptChoice(reader *bufio.Reader, question string, options []string, defaultIdx int) string {
	fmt.Printf("  %s\n", question)
	for i, opt := range options {
		marker := "  "
		if i == defaultIdx {
			marker = "→ "
		}
		fmt.Printf("    %s%d) %s\n", marker, i+1, opt)
	}
	fmt.Printf("  Choose %s[%d]%s: ", utils.Dim(""), defaultIdx+1, "")
	answer, err := reader.ReadString('\n')
	// On error (including EOF), fall back to default. In non-interactive mode,
	// EOF is expected and harmless.
	if err != nil {
		return options[defaultIdx]
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return options[defaultIdx]
	}
	// Try numeric selection first.
	for i, opt := range options {
		if answer == fmt.Sprintf("%d", i+1) {
			return opt
		}
	}
	// Fall back to name matching.
	for _, opt := range options {
		if strings.EqualFold(answer, opt) || strings.HasPrefix(strings.ToLower(opt), strings.ToLower(answer)) {
			return opt
		}
	}
	return options[defaultIdx]
}

// runInteractiveInit runs the interactive questionnaire.
func runInteractiveInit(projectName string) initAnswers {
	answers := initAnswers{
		Title:    projectName,
		Author:   "",
		Language: "en-US",
		Theme:    "technical",
	}

	// Fall back to defaults if the terminal is not interactive.
	if !isTerminalInteractive() {
		utils.Warning("Interactive input is not available; using defaults")
		return answers
	}

	reader := bufio.NewReader(os.Stdin)

	utils.Header("Initialize Book Project")
	fmt.Println("  Answer the following questions. Press Enter to accept the default.")
	fmt.Println()

	// Title
	answers.Title = promptUser(reader, "Title", projectName)

	// Author
	answers.Author = promptUser(reader, "Author", "")

	// Language
	languages := []string{"en-US", "zh-CN", "ja-JP", "ko-KR", "zh-TW"}
	answers.Language = promptChoice(reader, "Language:", languages, 0)

	// Theme
	themes := []string{"technical", "elegant", "minimal"}
	answers.Theme = promptChoice(reader, "Theme:", themes, 0)

	fmt.Println()
	return answers
}

// discoveredFile represents a Markdown file found during scanning.
type discoveredFile struct {
	RelPath string // Path relative to the project root.
	Title   string // First extracted H1 heading, if any.
	Depth   int    // Directory depth for sorting.
}

func executeInit(ctx context.Context, dir string) error {
	logger := slog.Default()

	// Resolve the target path early to avoid cwd-dependent behavior.
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}
	dir = absDir
	projectName := filepath.Base(absDir)

	// Ensure the target directory exists.
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Refuse to overwrite an existing book.yaml.
	cfgPath := filepath.Join(dir, "book.yaml")
	if utils.FileExists(cfgPath) {
		return fmt.Errorf("book.yaml already exists in %s (delete it before reinitializing)", dir)
	}

	// Scan for Markdown files.
	logger.Info("Scanning directory for Markdown files", slog.String("path", absDir))
	files, err := scanMarkdownFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	if len(files) == 0 {
		// Empty project: create starter files and a default config.
		logger.Info("No Markdown files found; creating starter template")

		// Collect interactive answers when requested.
		var answers initAnswers
		if initInteractive {
			answers = runInteractiveInit(projectName)
		} else {
			answers = initAnswers{
				Title:    projectName,
				Author:   "",
				Language: "en-US",
				Theme:    "technical",
			}
		}

		if err := createStarterTemplate(dir); err != nil {
			return fmt.Errorf("failed to create starter template: %w", err)
		}
		yaml := generateInteractiveBookYAML(answers)
		if err := utils.WriteFile(cfgPath, []byte(yaml)); err != nil {
			return fmt.Errorf("failed to write book.yaml: %w", err)
		}

		utils.Header("Project Created")
		utils.Success("Created a new mdpress project: %s", dir)
		fmt.Println()
		fmt.Println("  Created files:")
		fmt.Println("    • book.yaml")
		fmt.Println("    • preface.md")
		fmt.Println("    • chapter01/README.md")
		fmt.Printf("\n  Next steps:\n")
		if absDir != "." {
			fmt.Printf("    cd %s\n", dir)
		}
		fmt.Println("    # Edit book.yaml and your Markdown files")
		fmt.Println("    mdpress build")
		fmt.Println()
		return nil
	}

	// Detect special project files.
	summaryPath := filepath.Join(dir, "SUMMARY.md")
	hasSummary := utils.FileExists(summaryPath)
	glossaryPath := filepath.Join(dir, "GLOSSARY.md")
	hasGlossary := utils.FileExists(glossaryPath)
	langsPath := filepath.Join(dir, "LANGS.md")
	hasLangs := utils.FileExists(langsPath)

	coverImage := detectCoverImage(dir)

	// Try to extract metadata from README.md to provide better defaults.
	readmePath := filepath.Join(dir, "README.md")
	readmeMeta := config.ExtractReadmeMetadata(ctx, readmePath)
	defaultTitle := projectName
	defaultAuthor := ""
	defaultLanguage := "en-US"
	if readmeMeta.Title != "" {
		defaultTitle = readmeMeta.Title
	}
	if readmeMeta.Author != "" {
		defaultAuthor = readmeMeta.Author
	}
	if readmeMeta.Language != "" {
		defaultLanguage = readmeMeta.Language
	}

	// Collect metadata overrides when interactive mode is enabled.
	var answers initAnswers
	if initInteractive {
		answers = runInteractiveInit(defaultTitle)
		// Pre-fill from detected metadata.
		if answers.Author == "" && defaultAuthor != "" {
			answers.Author = defaultAuthor
		}
	} else {
		answers = initAnswers{
			Title:    defaultTitle,
			Author:   defaultAuthor,
			Language: defaultLanguage,
			Theme:    "technical",
		}
	}

	// When SUMMARY.md exists, omit chapters because build will read them from SUMMARY.md.
	// Otherwise generate chapters from the scanned file list.
	var yamlContent string
	if hasSummary {
		logger.Info("Detected SUMMARY.md; chapters will be loaded from it at build time")
		yamlContent = generateBookYAMLWithMeta(answers, coverImage, nil)
	} else {
		logger.Info("Discovered Markdown files", slog.Int("count", len(files)))
		yamlContent = generateBookYAMLWithMeta(answers, coverImage, files)
	}

	if err := utils.WriteFile(cfgPath, []byte(yamlContent)); err != nil {
		return fmt.Errorf("failed to write book.yaml: %w", err)
	}

	// Print the result summary.
	fmt.Printf("\n✅ Initialized an mdpress project in %s\n\n", dir)
	if hasSummary {
		// Parse SUMMARY.md to show chapter counts.
		chapters, err := config.ParseSummary(summaryPath)
		if err == nil {
			topLevel := len(chapters)
			total := countChapterDefs(chapters)
			fmt.Printf("  📄 SUMMARY.md: %d top-level chapters, %d entries total\n", topLevel, total)
		}
	} else {
		fmt.Printf("  Discovered %d chapters\n", len(files))
	}
	if hasGlossary {
		fmt.Printf("  📖 Detected GLOSSARY.md (terms will be highlighted automatically)\n")
	}
	if hasLangs {
		fmt.Printf("  🌐 Detected LANGS.md (multi-language project)\n")
	}
	fmt.Printf("\n  Generated: book.yaml\n")
	fmt.Printf("\n  Next steps:\n")
	if absDir != "." {
		fmt.Printf("    cd %s\n", dir)
	}
	fmt.Printf("    # Review book.yaml, then run:\n")
	fmt.Printf("    mdpress build\n")

	return nil
}

// countChapterDefs recursively counts chapter definitions.
func countChapterDefs(chapters []config.ChapterDef) int {
	count := len(chapters)
	for _, ch := range chapters {
		count += countChapterDefs(ch.Sections)
	}
	return count
}

// scanMarkdownFiles recursively scans a directory for Markdown files.
func scanMarkdownFiles(root string) ([]discoveredFile, error) {
	var files []discoveredFile

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files.
		}

		// Skip hidden directories.
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		// Skip common dependency directories.
		if info.IsDir() {
			if scanSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only keep .md files.
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		// Normalize to forward slashes.
		relPath = filepath.ToSlash(relPath)

		// Skip top-level documentation files that are not book content.
		// Keep README.md files inside subdirectories as chapter entry files.
		if filepath.Dir(relPath) == "." {
			baseLower := strings.ToLower(info.Name())
			if baseLower == "readme.md" || baseLower == "changelog.md" ||
				baseLower == "contributing.md" || baseLower == "license.md" {
				return nil
			}
		}

		title := utils.ExtractTitleFromFile(path)
		depth := strings.Count(relPath, "/")

		files = append(files, discoveredFile{
			RelPath: relPath,
			Title:   title,
			Depth:   depth,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan markdown files in %s: %w", root, err)
	}

	// Sort by depth first, then path name.
	sort.Slice(files, func(i, j int) bool {
		if files[i].Depth != files[j].Depth {
			return files[i].Depth < files[j].Depth
		}
		return files[i].RelPath < files[j].RelPath
	})

	return files, nil
}

// detectCoverImage looks for common cover image file names.
func detectCoverImage(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	existing := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		existing[entry.Name()] = struct{}{}
	}

	candidates := []string{
		"cover.png", "cover.jpg", "cover.jpeg", "cover.svg",
		"Cover.png", "Cover.jpg", "Cover.jpeg",
	}
	for _, name := range candidates {
		if _, ok := existing[name]; ok {
			return name
		}
	}
	return ""
}

// generateBookYAMLWithMeta generates book.yaml using full metadata from initAnswers.
// When files is nil, chapters section is omitted (SUMMARY.md mode).
func generateBookYAMLWithMeta(answers initAnswers, coverImage string, files []discoveredFile) string {
	var b strings.Builder

	if files == nil {
		b.WriteString("# mdpress book configuration\n")
		b.WriteString("# Chapters are loaded automatically from SUMMARY.md\n")
	} else {
		b.WriteString("# mdpress book configuration\n")
		b.WriteString("# Generated automatically by mdpress init\n")
	}
	b.WriteString("# Docs: https://github.com/yeasy/mdpress\n\n")

	b.WriteString("book:\n")
	fmt.Fprintf(&b, "  title: %q\n", answers.Title)
	if answers.Author != "" {
		fmt.Fprintf(&b, "  author: %q\n", answers.Author)
	} else {
		b.WriteString("  author: \"\"\n")
	}
	b.WriteString("  version: \"1.0.0\"\n")
	fmt.Fprintf(&b, "  language: %q\n", answers.Language)

	b.WriteString("  cover:\n")
	if coverImage != "" {
		fmt.Fprintf(&b, "    image: %q\n", coverImage)
	} else {
		b.WriteString("    background: \"#1a1a2e\"\n")
	}

	if files == nil {
		b.WriteString("\n# No chapters section required. Chapters are loaded from SUMMARY.md\n")
	} else {
		b.WriteString("\n# Chapters are listed in reading order\n")
		b.WriteString("# Paths are relative to this config file\n")
		b.WriteString("chapters:\n")
		for _, f := range files {
			title := f.Title
			if title == "" {
				title = inferTitleFromPath(f.RelPath)
			}
			fmt.Fprintf(&b, "  - title: %q\n", title)
			fmt.Fprintf(&b, "    file: %q\n", f.RelPath)
		}
	}

	b.WriteString("\nstyle:\n")
	fmt.Fprintf(&b, "  theme: %q\n", answers.Theme)
	b.WriteString("  page_size: \"A4\"\n")

	b.WriteString("\noutput:\n")
	// Use a sanitized filename based on the title.
	outName := strings.ReplaceAll(answers.Title, " ", "_")
	if outName == "" {
		outName = "output"
	}
	fmt.Fprintf(&b, "  filename: %q\n", outName+".pdf")
	b.WriteString("  toc: true\n")
	b.WriteString("  cover: true\n")

	return b.String()
}

// inferTitleFromPath derives a chapter title from a file path.
func inferTitleFromPath(relPath string) string {
	// "chapter01/README.md" → "chapter01"
	// "preface.md" → "preface"
	// "part1/intro.md" → "part1/intro"
	name := filepath.ToSlash(relPath)

	// Strip the .md suffix.
	name = strings.TrimSuffix(name, ".md")
	name = strings.TrimSuffix(name, ".MD")

	// Use the parent directory when the file is named README.
	if strings.HasSuffix(strings.ToUpper(name), "/README") || strings.EqualFold(name, "README") {
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir != "." {
			name = dir
		}
	}

	// Replace path separators with readable delimiters.
	name = strings.ReplaceAll(name, "/", " - ")

	// Uppercase the first rune (safe for multi-byte UTF-8 characters).
	runes := []rune(name)
	if len(runes) > 0 {
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		name = string(runes)
	}

	return name
}

// generateInteractiveBookYAML builds book.yaml from interactive answers.
func generateInteractiveBookYAML(answers initAnswers) string {
	return fmt.Sprintf(`# mdpress book configuration
# Generated by mdpress init

book:
  title: %q
  author: %q
  version: "1.0.0"
  language: %q

chapters:
  - title: "Preface"
    file: "preface.md"
  - title: "Chapter 1"
    file: "chapter01/README.md"

style:
  theme: %q
  page_size: "A4"

output:
  filename: "output.pdf"
  toc: true
  cover: true
`, answers.Title, answers.Author, answers.Language, answers.Theme)
}

// createStarterTemplate creates starter Markdown files in an empty directory.
func createStarterTemplate(dir string) error {
	prefaceContent := "# Preface\n\nWrite your preface here.\n"

	ch01Content := "# Chapter 1\n\n" +
		"## 1.1 Introduction\n\n" +
		"Start writing the main content here.\n\n" +
		"## 1.2 Code Example\n\n" +
		"```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, mdpress!\")\n}\n```\n"

	if err := utils.WriteFile(filepath.Join(dir, "preface.md"), []byte(prefaceContent)); err != nil {
		return fmt.Errorf("failed to create preface.md: %w", err)
	}

	ch01Dir := filepath.Join(dir, "chapter01")
	if err := utils.EnsureDir(ch01Dir); err != nil {
		return fmt.Errorf("failed to create chapter01/: %w", err)
	}
	if err := utils.WriteFile(filepath.Join(ch01Dir, "README.md"), []byte(ch01Content)); err != nil {
		return fmt.Errorf("failed to create chapter01/README.md: %w", err)
	}

	return nil
}
