// validate.go implements the validate subcommand.
// It checks the book config, referenced files, and image paths,
// then prints a readable validation report.
package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
		return executeValidate(cmd.Context(), targetDir)
	},
}

func init() {
	validateCmd.Flags().StringVar(&validateReportPath, "report", "", "Write validation report to .json or .md")
}

// validateResult represents a single validation result.
type validateResult struct {
	OK      bool   `json:"ok"`                // Whether the check passed.
	Warning bool   `json:"warning,omitempty"` // True for non-fatal warnings (OK is also true).
	Message string `json:"message"`           // Human-readable description.
}

// executeValidate runs the full validation flow.
func executeValidate(ctx context.Context, targetDir string) error {
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
			OK:      true,
			Message: fmt.Sprintf("Config file found: %s", configPath),
		})

		cfg, err = config.Load(configPath)
		if err != nil {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("Invalid config syntax: %v", err),
			})
			return finalizeValidate(results, true, 0)
		}
		results = append(results, validateResult{
			OK:      true,
			Message: "Config syntax is valid",
		})
	} else {
		cfg, err = config.Discover(ctx, absTargetDir)
		if err != nil {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("No buildable project found: %v", err),
			})
			return finalizeValidate(results, true, 0)
		}
		results = append(results, validateResult{
			OK:      true,
			Message: fmt.Sprintf("No book.yaml found, using auto-discovery from: %s", cfg.BaseDir()),
		})
		configPath = filepath.Join(cfg.BaseDir(), "book.yaml")
	}

	// ========== 3. Check required fields ==========
	// Title
	if cfg.Book.Title == "" {
		results = append(results, validateResult{OK: false, Message: "Missing book title (book.title)"})
		hasError = true
	} else {
		results = append(results, validateResult{OK: true, Message: fmt.Sprintf("Title: %s", cfg.Book.Title)})
	}

	// Author
	if cfg.Book.Author == "" {
		results = append(results, validateResult{OK: true, Warning: true, Message: "Missing author (book.author) (recommended)"})
	} else {
		results = append(results, validateResult{OK: true, Message: fmt.Sprintf("Author: %s", cfg.Book.Author)})
	}

	// Chapter list
	if len(cfg.Chapters) == 0 {
		results = append(results, validateResult{OK: false, Message: "No chapters defined (chapters)"})
		hasError = true
	} else {
		results = append(results, validateResult{OK: true, Message: fmt.Sprintf("Chapter count: %d", countChapterDefs(cfg.Chapters))})
	}

	// ========== 4. Check referenced Markdown files ==========
	flatChapters := config.FlattenChapters(cfg.Chapters)
	missingFiles := 0
	for _, ch := range flatChapters {
		filePath := cfg.ResolvePath(ch.File)
		if utils.FileExists(filePath) {
			results = append(results, validateResult{
				OK:      true,
				Message: fmt.Sprintf("Chapter file found: %s", ch.File),
			})
		} else {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("Chapter file not found: %s", ch.File),
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
				OK:      true,
				Message: fmt.Sprintf("Cover image found: %s", cfg.Book.Cover.Image),
			})
		} else {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("Cover image not found: %s", cfg.Book.Cover.Image),
			})
			hasError = true
		}
	}

	// ========== 6. Check custom CSS ==========
	if cfg.Style.CustomCSS != "" {
		cssPath := cfg.ResolvePath(cfg.Style.CustomCSS)
		if utils.FileExists(cssPath) {
			results = append(results, validateResult{OK: true, Message: fmt.Sprintf("Custom CSS found: %s", cfg.Style.CustomCSS)})
		} else {
			results = append(results, validateResult{OK: false, Message: fmt.Sprintf("Custom CSS not found: %s", cfg.Style.CustomCSS)})
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
			if utils.IsRemoteURL(img) {
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
					OK:      false,
					Message: fmt.Sprintf("Image not found: %s (referenced from %s)", img, ch.File),
				})
				imageErrors++
				hasError = true
			}
		}
	}
	if imageChecked > 0 && imageErrors == 0 {
		results = append(results, validateResult{
			OK:      true,
			Message: fmt.Sprintf("Image reference check passed (%d images)", imageChecked),
		})
	}

	// Print progress so far before the slow content validation step.
	printedSoFar := 0
	if !quiet {
		printResults(results)
		printedSoFar = len(results)
		fmt.Printf("\n  Validating chapter content (%d files)...\n", len(flatChapters))
	}

	// ========== 8. Check chapter numbering and Mermaid diagnostics ==========
	contentIssues, contentWarnings, contentErr := validateChapterContentAndSequence(cfg)
	if contentErr != nil {
		results = append(results, validateResult{
			OK:      false,
			Message: fmt.Sprintf("Content validation failed: %v", contentErr),
		})
		hasError = true
	} else if len(contentIssues) == 0 && len(contentWarnings) == 0 {
		results = append(results, validateResult{
			OK:      true,
			Message: "Chapter numbering, title, and Mermaid checks passed",
		})
	} else {
		if len(contentIssues) == 0 {
			results = append(results, validateResult{
				OK:      true,
				Message: "Chapter numbering and Mermaid checks passed",
			})
		}
		for _, issue := range contentIssues {
			results = append(results, validateResult{
				OK:      false,
				Message: issue,
			})
		}
		for _, warn := range contentWarnings {
			results = append(results, validateResult{
				OK:      true,
				Warning: true,
				Message: warn,
			})
		}
		if len(contentIssues) > 0 {
			hasError = true
		}
	}

	// ========== 9. Check Markdown chapter links ==========
	unresolvedLinks, linkErr := findUnresolvedMarkdownLinks(cfg)
	if linkErr != nil {
		results = append(results, validateResult{
			OK:      false,
			Message: fmt.Sprintf("Markdown link check failed: %v", linkErr),
		})
		hasError = true
	} else if len(unresolvedLinks) == 0 {
		results = append(results, validateResult{
			OK:      true,
			Message: "Markdown chapter link check passed",
		})
	} else {
		for _, item := range unresolvedLinks {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("Markdown link target is outside the build graph: %s (referenced from %s)", item.Target, item.Source),
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
			results = append(results, validateResult{OK: true, Message: fmt.Sprintf("Detected %s (%s)", file, desc)})
		}
	}

	return finalizeValidate(results, hasError, printedSoFar)
}

// printResults prints validation results, optionally skipping already-printed ones.
func printResults(results []validateResult, skipFirst ...int) {
	if quiet {
		return
	}
	skip := 0
	if len(skipFirst) > 0 {
		skip = skipFirst[0]
	}
	fmt.Println()
	for i, r := range results {
		if i < skip {
			continue
		}
		if r.Warning {
			utils.Warning("%s", r.Message)
		} else if r.OK {
			utils.Success("%s", r.Message)
		} else {
			utils.Error("%s", r.Message)
		}
	}
}

// extractImagePaths extracts image references from a Markdown file.
func extractImagePaths(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image path file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	// Match Markdown image syntax: ![alt](path)
	imgRegex := mdImagePattern
	// Match HTML img tags: <img src="path">
	htmlImgRegex := htmlImgSrcPattern

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
	flatChapters := config.FlattenChapters(cfg.Chapters)
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
			return nil, fmt.Errorf("failed to extract markdown links from %s: %w", ch.File, err)
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
		return nil, fmt.Errorf("failed to open markdown link file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	linkRegex := mdLinkPattern
	htmlLinkRegex := htmlLinkHrefPattern

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

func validateChapterContentAndSequence(cfg *config.BookConfig) (issues []string, warnings []string, err error) {
	issues = validateChapterSequence(cfg.Chapters)
	parser := markdown.NewParser()

	for _, flat := range flattenChaptersWithDepth(cfg.Chapters) {
		filePath := cfg.ResolvePath(flat.Def.File)
		content, readErr := utils.ReadFile(filePath)
		if readErr != nil {
			return issues, warnings, fmt.Errorf("failed to read chapter file %s: %w", flat.Def.File, readErr)
		}

		htmlContent, headings, diagnostics, parseErr := parser.ParseWithDiagnostics(content)
		if parseErr != nil {
			return issues, warnings, fmt.Errorf("failed to parse chapter %s: %w", flat.Def.File, parseErr)
		}

		if diag := validateChapterTitleSequence(flat.Def.Title, headings); diag != nil {
			issues = append(issues, fmt.Sprintf("Chapter title numbering mismatch: %s (rule=%s)", flat.Def.File, diag.Rule))
		}

		// Check SUMMARY title vs file heading mismatch (warning, not error).
		if flat.Def.Title != "" && len(headings) > 0 && headings[0].Text != "" {
			summaryNorm := normalizeChapterTitle(flat.Def.Title)
			headingNorm := normalizeChapterTitle(headings[0].Text)
			if summaryNorm != "" && headingNorm != "" && summaryNorm != headingNorm {
				warnings = append(warnings, fmt.Sprintf("Title mismatch in %s: SUMMARY %q vs file heading %q (SUMMARY title takes precedence)",
					flat.Def.File, flat.Def.Title, headings[0].Text))
			}
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

	return issues, warnings, nil
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

// Pre-compiled regexps for content validation.
var (
	relaxedChineseTitleSequencePattern = regexp.MustCompile(`^\s*第\s*([一二三四五六七八九十百零〇两\d]+)(?:\s*([章节篇部卷]))?(?:\s+|$)`)
	mdImagePattern                     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	htmlImgSrcPattern                  = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
	mdLinkPattern                      = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	htmlLinkHrefPattern                = regexp.MustCompile(`<a[^>]+href=["']([^"']+\.md(?:#[^"']*)?)["']`)
)

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

func finalizeValidate(results []validateResult, hasError bool, alreadyPrinted ...int) error {
	passed, failed := summarizeValidationResults(results)
	warned := 0
	for _, r := range results {
		if r.Warning {
			warned++
		}
	}

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

	skip := 0
	if len(alreadyPrinted) > 0 {
		skip = alreadyPrinted[0]
	}
	printResults(results, skip)

	if !quiet {
		fmt.Println()
		warnSuffix := ""
		if warned > 0 {
			warnSuffix = fmt.Sprintf(", %s warnings", utils.Yellow(strconv.Itoa(warned)))
		}
		if hasError {
			fmt.Printf("  %s %d checks total, %s passed, %s failed%s\n",
				utils.Red("Result:"),
				len(results),
				utils.Green(strconv.Itoa(passed)),
				utils.Red(strconv.Itoa(failed)),
				warnSuffix,
			)
			fmt.Println()
		} else if warned > 0 {
			fmt.Printf("  %s %d checks passed, %s warnings ✓\n",
				utils.Green("Result:"),
				passed,
				utils.Yellow(strconv.Itoa(warned)),
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
		return errors.New("validation failed; fix the issues above and try again")
	}
	return nil
}

func summarizeValidationResults(results []validateResult) (passed int, failed int) {
	for _, r := range results {
		if r.OK && !r.Warning {
			passed++
		} else if !r.OK {
			failed++
		}
		// Warnings are counted separately (not in passed or failed).
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
		return fmt.Errorf("resolve report path: %w", err)
	}
	if err := utils.EnsureDir(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	switch strings.ToLower(filepath.Ext(absPath)) {
	case ".json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal report: %w", err)
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
		if !result.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", status, result.Message)
	}
	return b.String()
}
