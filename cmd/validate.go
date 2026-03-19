// validate.go implements the validate subcommand.
// It checks the book config, referenced files, and image paths,
// then prints a readable validation report.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/pkg/utils"
)

var validateReportPath string

// validateCmd validates project configuration and references.
var validateCmd = &cobra.Command{
	Use:   "validate [directory]",
	Short: "Validate project config and file references",
	Long: `Validate the book.yaml configuration, including:
  - Valid config syntax
  - Required fields such as title and chapters
  - Referenced .md files
  - Image paths referenced from Markdown
  - Cover image paths

Examples:
  mdpress validate
  mdpress validate /path/to/book
  mdpress validate /path/to/book --report validate-report.json
  mdpress validate --config path/to/book.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}
		return executeValidate(targetDir)
	},
}

func init() {
	validateCmd.Flags().StringVar(&validateReportPath, "report", "", "Write validation report to .json or .md")
}

// validateResult represents a single validation result.
type validateResult struct {
	ok      bool   // Whether the check passed.
	message string // Human-readable description.
}

// executeValidate runs the full validation flow.
func executeValidate(targetDir string) error {
	utils.Header("mdpress Validation")

	var results []validateResult
	hasError := false

	// ========== 1. Resolve config or auto-discover project ==========
	absTargetDir, absErr := filepath.Abs(targetDir)
	if absErr != nil {
		return fmt.Errorf("validation failed: cannot resolve target directory: %w", absErr)
	}

	configPath := cfgFile
	if targetDir != "." {
		configPath = filepath.Join(absTargetDir, "book.yaml")
	}
	var cfg *config.BookConfig
	var err error
	if utils.FileExists(configPath) {
		results = append(results, validateResult{
			ok:      true,
			message: fmt.Sprintf("Config file found: %s", configPath),
		})

		cfg, err = config.Load(configPath)
		if err != nil {
			results = append(results, validateResult{
				ok:      false,
				message: fmt.Sprintf("Invalid config syntax: %v", err),
			})
			return finalizeValidate(results, true)
		}
		results = append(results, validateResult{
			ok:      true,
			message: "Config syntax is valid",
		})
	} else {
		cfg, err = config.Discover(absTargetDir)
		if err != nil {
			results = append(results, validateResult{
				ok:      false,
				message: fmt.Sprintf("No buildable project found: %v", err),
			})
			return finalizeValidate(results, true)
		}
		results = append(results, validateResult{
			ok:      true,
			message: fmt.Sprintf("No book.yaml found, using auto-discovery from: %s", cfg.BaseDir()),
		})
		configPath = filepath.Join(cfg.BaseDir(), "book.yaml")
	}

	// ========== 3. Check required fields ==========
	// Title
	if cfg.Book.Title == "" {
		results = append(results, validateResult{ok: false, message: "Missing book title (book.title)"})
		hasError = true
	} else {
		results = append(results, validateResult{ok: true, message: fmt.Sprintf("Title: %s", cfg.Book.Title)})
	}

	// Author
	if cfg.Book.Author == "" {
		results = append(results, validateResult{ok: false, message: "Missing author (book.author) (recommended)"})
		// Missing author is a warning, not a hard error.
	} else {
		results = append(results, validateResult{ok: true, message: fmt.Sprintf("Author: %s", cfg.Book.Author)})
	}

	// Chapter list
	if len(cfg.Chapters) == 0 {
		results = append(results, validateResult{ok: false, message: "No chapters defined (chapters)"})
		hasError = true
	} else {
		results = append(results, validateResult{ok: true, message: fmt.Sprintf("Chapter count: %d", countChapterDefs(cfg.Chapters))})
	}

	// ========== 4. Check referenced Markdown files ==========
	flatChapters := flattenChapterDefs(cfg.Chapters)
	missingFiles := 0
	for _, ch := range flatChapters {
		filePath := cfg.ResolvePath(ch.File)
		if utils.FileExists(filePath) {
			results = append(results, validateResult{
				ok:      true,
				message: fmt.Sprintf("Chapter file found: %s", ch.File),
			})
		} else {
			results = append(results, validateResult{
				ok:      false,
				message: fmt.Sprintf("Chapter file not found: %s", ch.File),
			})
			missingFiles++
			hasError = true
		}
	}

	// ========== 5. Check cover image ==========
	if cfg.Book.Cover.Image != "" {
		coverPath := cfg.ResolvePath(cfg.Book.Cover.Image)
		if utils.FileExists(coverPath) {
			results = append(results, validateResult{
				ok:      true,
				message: fmt.Sprintf("Cover image found: %s", cfg.Book.Cover.Image),
			})
		} else {
			results = append(results, validateResult{
				ok:      false,
				message: fmt.Sprintf("Cover image not found: %s", cfg.Book.Cover.Image),
			})
			hasError = true
		}
	}

	// ========== 6. Check custom CSS ==========
	if cfg.Style.CustomCSS != "" {
		cssPath := cfg.ResolvePath(cfg.Style.CustomCSS)
		if utils.FileExists(cssPath) {
			results = append(results, validateResult{ok: true, message: fmt.Sprintf("Custom CSS found: %s", cfg.Style.CustomCSS)})
		} else {
			results = append(results, validateResult{ok: false, message: fmt.Sprintf("Custom CSS not found: %s", cfg.Style.CustomCSS)})
			hasError = true
		}
	}

	// ========== 7. Check images referenced from Markdown ==========
	imageErrors := 0
	imageChecked := 0
	for _, ch := range flatChapters {
		filePath := cfg.ResolvePath(ch.File)
		if !utils.FileExists(filePath) {
			continue
		}
		images, err := extractImagePaths(filePath)
		if err != nil {
			continue
		}
		chapterDir := filepath.Dir(filePath)
		for _, img := range images {
			// Skip remote URLs.
			if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
				continue
			}
			imageChecked++
			// Resolve image paths relative to the chapter file first.
			imgPath := filepath.Join(chapterDir, img)
			// Also try the project root.
			if !utils.FileExists(imgPath) {
				imgPath = cfg.ResolvePath(img)
			}
			if !utils.FileExists(imgPath) {
				results = append(results, validateResult{
					ok:      false,
					message: fmt.Sprintf("Image not found: %s (referenced from %s)", img, ch.File),
				})
				imageErrors++
				hasError = true
			}
		}
	}
	if imageChecked > 0 && imageErrors == 0 {
		results = append(results, validateResult{
			ok:      true,
			message: fmt.Sprintf("Image reference check passed (%d images)", imageChecked),
		})
	}

	// ========== 8. Check chapter numbering and Mermaid diagnostics ==========
	contentIssues, contentErr := validateChapterContentAndSequence(cfg)
	if contentErr != nil {
		results = append(results, validateResult{
			ok:      false,
			message: fmt.Sprintf("Content validation failed: %v", contentErr),
		})
		hasError = true
	} else if len(contentIssues) == 0 {
		results = append(results, validateResult{
			ok:      true,
			message: "Chapter numbering and Mermaid checks passed",
		})
	} else {
		for _, issue := range contentIssues {
			results = append(results, validateResult{
				ok:      false,
				message: issue,
			})
		}
		hasError = true
	}

	// ========== 9. Check Markdown chapter links ==========
	unresolvedLinks, linkErr := findUnresolvedMarkdownLinks(cfg)
	if linkErr != nil {
		results = append(results, validateResult{
			ok:      false,
			message: fmt.Sprintf("Markdown link check failed: %v", linkErr),
		})
		hasError = true
	} else if len(unresolvedLinks) == 0 {
		results = append(results, validateResult{
			ok:      true,
			message: "Markdown chapter link check passed",
		})
	} else {
		for _, item := range unresolvedLinks {
			results = append(results, validateResult{
				ok:      false,
				message: fmt.Sprintf("Markdown link target is outside the build graph: %s (referenced from %s)", item.Target, item.Source),
			})
		}
		hasError = true
	}

	// ========== 10. Detect special files ==========
	configDir := filepath.Dir(configPath)
	if absDir, err := filepath.Abs(configDir); err == nil {
		configDir = absDir
	}

	specialFiles := map[string]string{
		"SUMMARY.md":  "table of contents definition",
		"GLOSSARY.md": "glossary definition",
		"LANGS.md":    "multi-language configuration",
	}
	for file, desc := range specialFiles {
		path := filepath.Join(configDir, file)
		if utils.FileExists(path) {
			results = append(results, validateResult{ok: true, message: fmt.Sprintf("Detected %s (%s)", file, desc)})
		}
	}

	return finalizeValidate(results, hasError)
}

// printResults prints all validation results.
func printResults(results []validateResult) {
	if quiet {
		return
	}
	fmt.Println()
	for _, r := range results {
		if r.ok {
			utils.Success("%s", r.message)
		} else {
			utils.Error("%s", r.message)
		}
	}
}

// flattenChapterDefs delegates to the canonical config.FlattenChapters.
func flattenChapterDefs(chapters []config.ChapterDef) []config.ChapterDef {
	return config.FlattenChapters(chapters)
}

// extractImagePaths extracts image references from a Markdown file.
func extractImagePaths(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	// Match Markdown image syntax: ![alt](path)
	imgRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	// Match HTML img tags: <img src="path">
	htmlImgRegex := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

	var images []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Markdown images
		matches := imgRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) >= 3 {
				imgPath := strings.TrimSpace(m[2])
				// Strip any optional title segment (for example: path/to/image.png "title")
				if idx := strings.Index(imgPath, " "); idx > 0 {
					imgPath = imgPath[:idx]
				}
				images = append(images, imgPath)
			}
		}

		// HTML img tags
		htmlMatches := htmlImgRegex.FindAllStringSubmatch(line, -1)
		for _, m := range htmlMatches {
			if len(m) >= 2 {
				images = append(images, strings.TrimSpace(m[1]))
			}
		}
	}

	return images, scanner.Err()
}

type unresolvedMarkdownLink struct {
	Source string
	Target string
}

func findUnresolvedMarkdownLinks(cfg *config.BookConfig) ([]unresolvedMarkdownLink, error) {
	flatChapters := flattenChapterDefs(cfg.Chapters)
	if len(flatChapters) == 0 {
		return nil, nil
	}

	targets := make(map[string]struct{}, len(flatChapters))
	for _, ch := range flatChapters {
		targets[linkrewrite.NormalizePath(ch.File)] = struct{}{}
	}

	var unresolved []unresolvedMarkdownLink
	for _, ch := range flatChapters {
		filePath := cfg.ResolvePath(ch.File)
		links, err := extractMarkdownLinks(filePath)
		if err != nil {
			return nil, err
		}
		currentDir := filepath.Dir(linkrewrite.NormalizePath(ch.File))
		for _, link := range links {
			targetPath := linkrewrite.NormalizePath(filepath.Join(currentDir, link))
			if _, ok := targets[targetPath]; ok {
				continue
			}
			unresolved = append(unresolved, unresolvedMarkdownLink{
				Source: ch.File,
				Target: link,
			})
		}
	}

	return unresolved, nil
}

func extractMarkdownLinks(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	htmlLinkRegex := regexp.MustCompile(`<a[^>]+href=["']([^"']+\.md(?:#[^"']*)?)["']`)

	var links []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		matches := linkRegex.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			if len(m) < 6 {
				continue
			}
			if m[0] > 0 && line[m[0]-1] == '!' {
				continue
			}
			target := normalizeMarkdownLinkTarget(line[m[4]:m[5]])
			if target != "" {
				links = append(links, target)
			}
		}

		htmlMatches := htmlLinkRegex.FindAllStringSubmatch(line, -1)
		for _, m := range htmlMatches {
			if len(m) < 2 {
				continue
			}
			target := normalizeMarkdownLinkTarget(m[1])
			if target != "" {
				links = append(links, target)
			}
		}
	}

	return links, scanner.Err()
}

func normalizeMarkdownLinkTarget(raw string) string {
	target := strings.TrimSpace(raw)
	if idx := strings.Index(target, "#"); idx >= 0 {
		target = target[:idx]
	}
	if target == "" || strings.ToLower(filepath.Ext(target)) != ".md" {
		return ""
	}
	lower := strings.ToLower(target)
	for _, prefix := range []string{"http://", "https://", "mailto:", "tel:", "javascript:", "data:"} {
		if strings.HasPrefix(lower, prefix) {
			return ""
		}
	}
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return ""
	}
	return target
}

func validateChapterContentAndSequence(cfg *config.BookConfig) ([]string, error) {
	issues := validateChapterSequence(cfg.Chapters)
	parser := markdown.NewParser()

	for _, flat := range flattenChaptersWithDepth(cfg.Chapters) {
		filePath := cfg.ResolvePath(flat.Def.File)
		content, err := utils.ReadFile(filePath)
		if err != nil {
			return issues, err
		}

		_, headings, diagnostics, err := parser.ParseWithDiagnostics(content)
		if err != nil {
			return issues, err
		}
		htmlContent, _, err := parser.Parse(content)
		if err != nil {
			return issues, err
		}

		if diag := validateChapterTitleSequence(flat.Def.Title, headings); diag != nil {
			issues = append(issues, fmt.Sprintf("Chapter title numbering mismatch: %s (rule=%s)", flat.Def.File, diag.Rule))
		}

		for _, diag := range diagnostics {
			if strings.HasPrefix(diag.Rule, "mermaid-") {
				issues = append(issues, fmt.Sprintf("Mermaid issue in %s at %d:%d: %s", flat.Def.File, diag.Line, diag.Column, diag.Message))
			}
		}
		if markdown.NeedsMermaid(htmlContent) {
			if err := validateRenderedMermaidHTML(htmlContent); err != nil {
				issues = append(issues, fmt.Sprintf("Mermaid render check failed in %s: %v", flat.Def.File, err))
			}
		}
	}

	return issues, nil
}

func validateChapterSequence(chapters []config.ChapterDef) []string {
	var issues []string
	var walk func([]config.ChapterDef, int)
	walk = func(items []config.ChapterDef, depth int) {
		prevSeq := []int(nil)
		prevStyle := ""
		prevTitle := ""
		for _, ch := range items {
			style := titleSequenceStyle(ch.Title)
			seq, ok := parseSequenceParts(ch.Title)
			if ok {
				if prevStyle == style && len(prevSeq) == len(seq) && len(seq) > 0 {
					expected := append([]int(nil), prevSeq...)
					expected[len(expected)-1]++
					if !equalIntSlices(expected, seq) {
						issues = append(issues, fmt.Sprintf("Chapter sequence gap or mismatch at depth %d: expected %s after %q, got %q",
							depth, formatSequenceParts(expected), prevTitle, ch.Title))
					}
				}
				prevSeq = seq
				prevStyle = style
				prevTitle = ch.Title
			}
			if len(ch.Sections) > 0 {
				walk(ch.Sections, depth+1)
			}
		}
	}

	walk(chapters, 0)
	return issues
}

var relaxedChineseTitleSequencePattern = regexp.MustCompile(`^\s*第\s*([一二三四五六七八九十百零〇两\d]+)(?:\s*([章节篇部卷]))?(?:\s+|$)`)

func parseSequenceParts(title string) ([]int, bool) {
	if matches := decimalTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return splitSequenceParts(matches[1])
	}
	if matches := englishTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		return splitSequenceParts(matches[1])
	}
	if matches := relaxedChineseTitleSequencePattern.FindStringSubmatch(title); len(matches) >= 2 {
		value := parseChineseOrdinal(matches[1])
		if value <= 0 && !strings.ContainsAny(matches[1], "零〇0") {
			return nil, false
		}
		return []int{value}, true
	}
	return nil, false
}

func splitSequenceParts(raw string) ([]int, bool) {
	parts := strings.Split(raw, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, value)
	}
	return out, true
}

func equalIntSlices(a, b []int) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func formatSequenceParts(parts []int) string {
	if len(parts) == 0 {
		return ""
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strconv.Itoa(part))
	}
	return strings.Join(out, ".")
}

type validationReport struct {
	Status      string           `json:"status"`
	TotalChecks int              `json:"total_checks"`
	Passed      int              `json:"passed"`
	Failed      int              `json:"failed"`
	Results     []validateResult `json:"results"`
}

func finalizeValidate(results []validateResult, hasError bool) error {
	passed, failed := summarizeValidationResults(results)

	if validateReportPath != "" {
		if err := writeValidationReport(validateReportPath, validationReport{
			Status:      validationStatus(hasError),
			TotalChecks: len(results),
			Passed:      passed,
			Failed:      failed,
			Results:     results,
		}); err != nil {
			return fmt.Errorf("validation failed to write report: %w", err)
		}
	}

	printResults(results)

	if !quiet {
		fmt.Println()
		if hasError {
			fmt.Printf("  %s %d checks total, %s passed, %s failed\n",
				utils.Red("Result:"),
				len(results),
				utils.Green(fmt.Sprintf("%d", passed)),
				utils.Red(fmt.Sprintf("%d", failed)),
			)
			fmt.Println()
		} else {
			fmt.Printf("  %s all %d checks passed ✓\n",
				utils.Green("Result:"),
				len(results),
			)
			fmt.Println()
		}
	}

	if hasError {
		return fmt.Errorf("validation failed; fix the issues above and try again")
	}
	return nil
}

func summarizeValidationResults(results []validateResult) (passed int, failed int) {
	for _, r := range results {
		if r.ok {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func validationStatus(hasError bool) string {
	if hasError {
		return "failed"
	}
	return "passed"
}

func writeValidationReport(path string, report validationReport) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := utils.EnsureDir(filepath.Dir(absPath)); err != nil {
		return err
	}

	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(absPath, data, 0644)
	case ".md":
		return os.WriteFile(absPath, []byte(renderValidationMarkdown(report)), 0644)
	default:
		return fmt.Errorf("unsupported report extension: %s (use .json or .md)", filepath.Ext(absPath))
	}
}

func renderValidationMarkdown(report validationReport) string {
	var b strings.Builder
	b.WriteString("# mdpress Validation Report\n\n")
	fmt.Fprintf(&b, "- Status: %s\n", report.Status)
	fmt.Fprintf(&b, "- Total checks: %d\n", report.TotalChecks)
	fmt.Fprintf(&b, "- Passed: %d\n", report.Passed)
	fmt.Fprintf(&b, "- Failed: %d\n\n", report.Failed)
	b.WriteString("## Results\n\n")
	for _, result := range report.Results {
		status := "PASS"
		if !result.ok {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", status, result.message)
	}
	return b.String()
}
