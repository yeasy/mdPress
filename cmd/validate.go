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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/linkrewrite"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/pkg/utils"
)

var (
	validateReportPath string
	// validateStrict makes warning-level findings fail the run, mirroring
	// `mdpress doctor --strict`.
	validateStrict bool
)

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

Pass --strict to exit with a non-zero status when any warning is reported
(for example a duplicate chapter entry or an unknown config key), which makes
mdpress validate usable as a CI gate. Without --strict only errors fail the run.

Examples:
  mdpress validate
  mdpress validate /path/to/book
  mdpress validate /path/to/book --report validate-report.json
  mdpress validate /path/to/book --strict
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
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "Exit with a non-zero status when any warning is reported (useful as a CI gate)")
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

	// An explicit --config wins over the target directory's own book.yaml.
	sourceDir := ""
	if targetDir != "." {
		sourceDir = absTargetDir
	}
	configPath, allowDiscovery := resolveConfigPath(sourceDir)
	if !allowDiscovery && !utils.FileExists(configPath) {
		return errExplicitConfigMissing(configPath)
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
			// Load reports every problem it found, not just the first, so give
			// each one its own line: a book.yaml with five mistakes should take
			// one run to diagnose, not five.
			for _, problem := range config.ValidationErrors(err) {
				results = append(results, validateResult{
					OK:      false,
					Message: fmt.Sprintf("Invalid config: %v", problem),
				})
			}
			return finalizeValidate(results, true, 0)
		}
		results = append(results, validateResult{
			OK:      true,
			Message: "Config syntax is valid",
		})
		// Unknown keys parse fine but are dropped on load, so without this the
		// report says the project is healthy while the setting does nothing.
		if data, readErr := os.ReadFile(configPath); readErr == nil { //nolint:gosec // G304: path the user asked us to validate
			for _, key := range config.FindUnknownKeys(data) {
				results = append(results, validateResult{
					OK:      true,
					Warning: true,
					Message: fmt.Sprintf("Unknown config key %q — ignored (%s)", key.Path, key.Hint()),
				})
			}
		}
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
	switch cfg.Book.Title {
	case "":
		results = append(results, validateResult{OK: false, Message: "Missing book title (book.title)"})
		hasError = true
	case config.DefaultBookTitle:
		// The placeholder means book.title never took effect — usually a typo
		// or wrong nesting. Reporting it as a pass hid exactly that.
		results = append(results, validateResult{
			OK:      true,
			Warning: true,
			Message: fmt.Sprintf("Title is still the placeholder %q — set book.title", config.DefaultBookTitle),
		})
	default:
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
	// A file listed twice does not produce two pages: the generators key their
	// output on the source path, so the duplicate silently overwrites the first
	// entry — its title disappears and the page can end up renamed. The list
	// still looks right in book.yaml, which is why this needs saying out loud.
	firstListing := make(map[string]string, len(flatChapters))
	for _, ch := range flatChapters {
		key := linkrewrite.NormalizePath(ch.File)
		if previous, seen := firstListing[key]; seen {
			results = append(results, validateResult{
				OK:      true,
				Warning: true,
				Message: fmt.Sprintf("Duplicate chapter entry: %s is listed more than once (as %q and %q) — only one of them will be built", ch.File, previous, ch.Title),
			})
		} else {
			firstListing[key] = ch.Title
		}

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
	contentIssues, contentWarnings, chapterAnchors, contentErr := validateChapterContentAndSequence(cfg)
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
	unresolvedLinks, deadAnchors, linkErr := findMarkdownLinkIssues(cfg, chapterAnchors)
	switch {
	case linkErr != nil:
		results = append(results, validateResult{
			OK:      false,
			Message: fmt.Sprintf("Markdown link check failed: %v", linkErr),
		})
		hasError = true
	case len(unresolvedLinks) == 0 && len(deadAnchors) == 0:
		results = append(results, validateResult{
			OK:      true,
			Message: "Markdown chapter link and anchor check passed",
		})
	default:
		for _, item := range unresolvedLinks {
			results = append(results, validateResult{
				OK:      false,
				Message: fmt.Sprintf("Markdown link target is outside the build graph: %s (referenced from %s)", item.Target, item.Source),
			})
		}
		if len(unresolvedLinks) > 0 {
			hasError = true
		}
		// A dead #fragment still renders as a working-looking link; it just
		// lands the reader at the top of the page. Report it, but as a
		// warning: raw HTML and plugins can inject ids mdpress cannot see.
		for _, item := range deadAnchors {
			results = append(results, validateResult{
				OK:      true,
				Warning: true,
				Message: fmt.Sprintf("Link anchor not found in %s: %s (referenced from %s)", item.Target, item.Link, item.Source),
			})
		}
	}

	// ========== 10. Check for Markdown files no chapter points at ==========
	if orphans, orphanErr := findOrphanMarkdownFiles(cfg); orphanErr == nil && len(orphans) > 0 {
		// A long list is almost always one mistake (a directory that was never
		// registered), so show enough to recognize it and then summarize.
		const orphanListLimit = 10
		shown := orphans
		if len(shown) > orphanListLimit {
			shown = shown[:orphanListLimit]
		}
		for _, orphan := range shown {
			results = append(results, validateResult{
				OK:      true,
				Warning: true,
				Message: fmt.Sprintf("Markdown file is in no chapter list and will not be built: %s", orphan),
			})
		}
		if len(orphans) > len(shown) {
			results = append(results, validateResult{
				OK:      true,
				Warning: true,
				Message: fmt.Sprintf("... and %d more Markdown file(s) in no chapter list", len(orphans)-len(shown)),
			})
		}
	}

	// ========== 11. Detect special files ==========
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
		// Quiet means "errors only", not "silence". Suppressing the failures
		// too left CI users with a non-zero exit, the line "fix the issues
		// above", and nothing above. Under --strict warnings are failures, so
		// they have to be printed for the same reason.
		for _, r := range results {
			switch {
			case !r.OK:
				utils.Error("%s", r.Message)
			case r.Warning && validateStrict:
				utils.Warning("%s", r.Message)
			}
		}
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
	var fences fenceTracker
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Examples inside code fences are not real references.
		line := scannableLine(scanner.Text(), &fences)
		if line == "" {
			continue
		}

		// Markdown images
		matches := imgRegex.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) >= 3 {
				imgPath := strings.TrimSpace(m[2])
				// Strip any optional title segment (for example: path/to/image.png "title")
				if idx := strings.Index(imgPath, " "); idx > 0 {
					imgPath = imgPath[:idx]
				}
				// URL-decode percent-encoded paths (e.g., "foo%20bar.png" -> "foo bar.png")
				if decoded, err := url.PathUnescape(imgPath); err == nil {
					imgPath = decoded
				}
				images = append(images, imgPath)
			}
		}

		// HTML img tags
		htmlMatches := htmlImgRegex.FindAllStringSubmatch(line, -1)
		for _, m := range htmlMatches {
			if len(m) >= 2 {
				imgPath := strings.TrimSpace(m[1])
				// URL-decode percent-encoded paths
				if decoded, err := url.PathUnescape(imgPath); err == nil {
					imgPath = decoded
				}
				images = append(images, imgPath)
			}
		}
	}

	return images, scanner.Err()
}

type unresolvedMarkdownLink struct {
	Source string
	Target string
}

// deadAnchorLink is a link whose file resolves but whose #fragment does not
// match any id the target page will actually publish.
type deadAnchorLink struct {
	Source string // chapter the link was written in
	Link   string // the link exactly as written, e.g. "guide.md#setup" or "#setup"
	Target string // chapter the fragment was looked up in
}

// findUnresolvedMarkdownLinks reports only the link-target half of the check.
// Callers that have not parsed the chapters' headings (build, doctor) cannot
// judge anchors, so passing no anchor map suppresses that half entirely.
func findUnresolvedMarkdownLinks(cfg *config.BookConfig) ([]unresolvedMarkdownLink, error) {
	unresolved, _, err := findMarkdownLinkIssues(cfg, nil)
	return unresolved, err
}

// findMarkdownLinkIssues checks every intra-book link in the chapter files:
// the path must name a chapter in the build graph, and a #fragment must match
// an id that chapter's page will carry.
//
// anchorsByChapter maps a normalized chapter path to the heading ids the
// renderer generates for it. Chapters missing from the map (because content
// validation could not parse them) are skipped rather than reported, so one
// unreadable file cannot manufacture a wall of dead-anchor warnings.
func findMarkdownLinkIssues(cfg *config.BookConfig, anchorsByChapter map[string]map[string]struct{}) ([]unresolvedMarkdownLink, []deadAnchorLink, error) {
	flatChapters := config.FlattenChapters(cfg.Chapters)
	if len(flatChapters) == 0 {
		return nil, nil, nil
	}

	// Scan each distinct chapter once: a file listed twice must not be read,
	// or reported on, twice.
	scans := make(map[string]markdownRefs, len(flatChapters))
	var order []config.ChapterDef
	for _, ch := range flatChapters {
		key := linkrewrite.NormalizePath(ch.File)
		if _, done := scans[key]; done {
			continue
		}
		refs, err := scanMarkdownRefs(cfg.ResolvePath(ch.File))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract markdown links from %s: %w", ch.File, err)
		}
		scans[key] = refs
		order = append(order, ch)
	}

	// An anchor can also come from raw HTML the author wrote by hand
	// (<a id="x">, <h2 id="x">), which the heading scan knows nothing about.
	anchors := make(map[string]map[string]struct{}, len(scans))
	for key, refs := range scans {
		set := make(map[string]struct{}, len(anchorsByChapter[key])+len(refs.AnchorIDs))
		for id := range anchorsByChapter[key] {
			set[id] = struct{}{}
		}
		for _, id := range refs.AnchorIDs {
			set[id] = struct{}{}
		}
		anchors[key] = set
	}

	var unresolved []unresolvedMarkdownLink
	var dead []deadAnchorLink
	for _, ch := range order {
		key := linkrewrite.NormalizePath(ch.File)
		currentDir := filepath.Dir(key)
		for _, link := range scans[key].Links {
			targetPath := key
			if link.Target != "" {
				targetPath = linkrewrite.NormalizePath(filepath.Join(currentDir, link.Target))
				if _, ok := scans[targetPath]; !ok {
					unresolved = append(unresolved, unresolvedMarkdownLink{
						Source: ch.File,
						Target: link.Target,
					})
					continue
				}
			}
			if link.Fragment == "" {
				continue
			}
			// Only chapters whose headings were collected can be judged.
			if _, known := anchorsByChapter[targetPath]; !known {
				continue
			}
			if _, ok := anchors[targetPath][link.Fragment]; ok {
				continue
			}
			dead = append(dead, deadAnchorLink{
				Source: ch.File,
				Link:   link.Raw,
				Target: targetPath,
			})
		}
	}

	return unresolved, dead, nil
}

// markdownLinkRef is one link that points inside the book.
type markdownLinkRef struct {
	// Target is the chapter path as written, or "" when the link is a
	// same-page anchor such as [see](#setup).
	Target string
	// Fragment is the anchor id without the leading "#", or "" when absent.
	Fragment string
	// Raw is the link as it appears in the source, for diagnostics.
	Raw string
}

// markdownRefs is everything one pass over a chapter file collects: the links
// it makes and the anchors it defines.
type markdownRefs struct {
	Links []markdownLinkRef
	// AnchorIDs are ids declared in raw HTML (id=/name= attributes). Heading
	// ids are generated by the renderer and come from the parser instead.
	AnchorIDs []string
}

// scanMarkdownRefs reads a chapter file once and collects its intra-book links
// and its hand-written HTML anchors, skipping fenced code and inline code so
// examples in a book about Markdown are not mistaken for real references.
func scanMarkdownRefs(filePath string) (markdownRefs, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return markdownRefs{}, fmt.Errorf("failed to open markdown link file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	var refs markdownRefs
	var fences fenceTracker
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Examples inside code fences are not real references.
		line := scannableLine(scanner.Text(), &fences)
		if line == "" {
			continue
		}

		matches := mdLinkPattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			if len(m) < 6 {
				continue
			}
			if m[0] > 0 && line[m[0]-1] == '!' {
				continue
			}
			if ref, ok := parseMarkdownLinkRef(line[m[4]:m[5]]); ok {
				refs.Links = append(refs.Links, ref)
			}
		}

		htmlMatches := htmlLinkHrefPattern.FindAllStringSubmatch(line, -1)
		for _, m := range htmlMatches {
			if len(m) < 2 {
				continue
			}
			if ref, ok := parseMarkdownLinkRef(m[1]); ok {
				refs.Links = append(refs.Links, ref)
			}
		}

		for _, m := range htmlAnchorIDPattern.FindAllStringSubmatch(line, -1) {
			if len(m) >= 2 {
				refs.AnchorIDs = append(refs.AnchorIDs, m[1])
			}
		}
	}

	return refs, scanner.Err()
}

// parseMarkdownLinkRef splits a link destination into the chapter path and the
// anchor it points at, and reports whether the link is one mdpress can check.
// Links that leave the book — remote URLs, mailto:, absolute paths, non-.md
// files — are not.
func parseMarkdownLinkRef(raw string) (markdownLinkRef, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return markdownLinkRef{}, false
	}

	lower := strings.ToLower(trimmed)
	for _, prefix := range []string{"http://", "https://", "mailto:", "tel:", "javascript:", "data:"} {
		if strings.HasPrefix(lower, prefix) {
			return markdownLinkRef{}, false
		}
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return markdownLinkRef{}, false
	}

	target, fragment, _ := strings.Cut(trimmed, "#")
	// A fragment travels through the URL, so "#foo%20bar" and "#foo bar" name
	// the same anchor.
	if decoded, err := url.PathUnescape(fragment); err == nil {
		fragment = decoded
	}

	if target == "" {
		// A bare "#fragment" points at the current page; "#" alone (a
		// placeholder link) points nowhere and is not worth reporting.
		if fragment == "" {
			return markdownLinkRef{}, false
		}
		return markdownLinkRef{Fragment: fragment, Raw: trimmed}, true
	}
	if strings.ToLower(filepath.Ext(target)) != ".md" {
		return markdownLinkRef{}, false
	}
	return markdownLinkRef{Target: target, Fragment: fragment, Raw: trimmed}, true
}

// validateChapterContentAndSequence parses every chapter once. Besides the
// issues and warnings it reports, it hands back the heading ids each chapter
// will publish, keyed by normalized chapter path, so the link check can tell a
// live #fragment from a dead one without parsing the book a second time.
func validateChapterContentAndSequence(cfg *config.BookConfig) (issues []string, warnings []string, headingIDs map[string]map[string]struct{}, err error) {
	issues = validateChapterSequence(cfg.Chapters)
	parser := markdown.NewParser()
	headingIDs = make(map[string]map[string]struct{})

	for _, flat := range flattenChaptersWithDepth(cfg.Chapters) {
		filePath := cfg.ResolvePath(flat.Def.File)
		content, readErr := utils.ReadFile(filePath)
		if readErr != nil {
			return issues, warnings, headingIDs, fmt.Errorf("failed to read chapter file %s: %w", flat.Def.File, readErr)
		}

		htmlContent, headings, diagnostics, parseErr := parser.ParseWithDiagnostics(content)
		if parseErr != nil {
			return issues, warnings, headingIDs, fmt.Errorf("failed to parse chapter %s: %w", flat.Def.File, parseErr)
		}

		ids := make(map[string]struct{}, len(headings))
		for _, h := range headings {
			if h.ID != "" {
				ids[h.ID] = struct{}{}
			}
		}
		headingIDs[linkrewrite.NormalizePath(flat.Def.File)] = ids

		// The build silently drops a chapter that renders to nothing — no
		// page, no sidebar entry — and the only trace is a WARN in the build
		// log. A chapter truncated to 0 bytes by a bad merge therefore shipped
		// past `validate --strict`, which is the gate that exists so nobody
		// has to read build logs. Test emptiness exactly the way
		// cmd/chapter_pipeline.go does so the two cannot drift apart.
		if htmlContent == "" {
			issues = append(issues, fmt.Sprintf("Empty chapter %s: it has no content, so it produces no page and no navigation entry", flat.Def.File))
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
			switch {
			case strings.HasPrefix(diag.Rule, "mermaid-"):
				issues = append(issues, fmt.Sprintf("Mermaid issue in %s at %d:%d: %s", flat.Def.File, diag.Line, diag.Column, diag.Message))
			case diag.Rule == "unclosed-code-fence":
				// Silently swallows the rest of the chapter, so it belongs in
				// the report a user runs precisely to catch that.
				issues = append(issues, fmt.Sprintf("Unclosed code block in %s at %d:%d: %s", flat.Def.File, diag.Line, diag.Column, diag.Message))
			}
		}
		if markdown.NeedsMermaid(htmlContent) {
			if err := validateRenderedMermaidHTML(htmlContent); err != nil {
				issues = append(issues, fmt.Sprintf("Mermaid render check failed in %s: %v", flat.Def.File, err))
			}
		}
	}

	return issues, warnings, headingIDs, nil
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
					if !slices.Equal(expected, seq) {
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
	relaxedChineseTitleSequencePattern = regexp.MustCompile(`^\s*第\s*([一二三四五六七八九十百零〇两\d]+)\s*([章节篇部卷])(?:\s+|$)`)
	mdImagePattern                     = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	htmlImgSrcPattern                  = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
	mdLinkPattern                      = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	htmlLinkHrefPattern                = regexp.MustCompile(`<a[^>]+href=["']([^"']*)["']`)
	// htmlAnchorIDPattern finds anchors an author declared by hand in raw
	// HTML, which the renderer passes straight through to the page.
	htmlAnchorIDPattern = regexp.MustCompile(`<[a-zA-Z][^>]*?\b(?:id|name)\s*=\s*["']([^"']+)["']`)
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
	Warnings    int              `json:"warnings"`
	Strict      bool             `json:"strict"`
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

	// In strict mode warnings are failures, so the report has to say "failed"
	// too — a report that reads "passed" next to a non-zero exit code is worse
	// than no report at all.
	strictFailure := validateStrict && warned > 0

	if validateReportPath != "" {
		if err := writeValidationReport(validateReportPath, validationReport{
			Status:      validationStatus(hasError || strictFailure),
			TotalChecks: len(results),
			Passed:      passed,
			Failed:      failed,
			Warnings:    warned,
			Strict:      validateStrict,
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
			label := utils.Green("Result:")
			mark := " ✓"
			if strictFailure {
				label = utils.Red("Result:")
				mark = " (failed: --strict)"
			}
			fmt.Printf("  %s %d checks passed, %s warnings%s\n",
				label,
				passed,
				utils.Yellow(strconv.Itoa(warned)),
				mark,
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
	if strictFailure {
		return fmt.Errorf("validation found %d warning(s) (run without --strict to ignore)", warned)
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
		return os.WriteFile(absPath, data, 0o644)
	case ".md":
		return os.WriteFile(absPath, []byte(renderValidationMarkdown(report)), 0o644)
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
