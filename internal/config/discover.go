// discover.go implements zero-config project discovery.
// When neither book.yaml nor SUMMARY.md exists, mdpress scans .md files,
// sorts them, and derives chapter metadata automatically.
package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/yeasy/mdpress/pkg/utils"
)

// Discover auto-discovers project configuration in a directory.
// Priority: book.yaml > book.json (GitBook compat) > SUMMARY.md > Markdown file scanning.
// The context is used for potentially long-running operations like git commands.
func Discover(ctx context.Context, dir string) (*BookConfig, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Priority 1: load book.yaml.
	bookYamlPath := filepath.Join(absDir, "book.yaml")
	if _, err := os.Stat(bookYamlPath); err == nil {
		return Load(bookYamlPath)
	}

	// Priority 2: load book.json (GitBook compatibility) for metadata.
	bookJSONPath := filepath.Join(absDir, "book.json")
	hasBookJSON := false
	var cfg *BookConfig
	var bookJSONErr error
	if _, err := os.Stat(bookJSONPath); err == nil {
		hasBookJSON = true
		cfg, bookJSONErr = LoadBookJSON(ctx, bookJSONPath)
		if bookJSONErr != nil {
			return nil, bookJSONErr
		}
	}

	// Priority 3: load SUMMARY.md (can coexist with book.json).
	summaryPath := filepath.Join(absDir, "SUMMARY.md")
	if _, err := os.Stat(summaryPath); err == nil {
		chapters, err := ParseSummary(summaryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SUMMARY.md: %w", err)
		}
		// If we have chapters from SUMMARY.md, use them
		if len(chapters) > 0 {
			if cfg == nil {
				cfg = DefaultConfig()
			}
			cfg.baseDir = absDir
			cfg.Chapters = chapters

			// Extract rich metadata from README.md.
			readmePath := filepath.Join(absDir, "README.md")
			meta := ExtractReadmeMetadata(ctx, readmePath)
			defaultTitle := DefaultConfig().Book.Title
			if cfg.Book.Title == "" || cfg.Book.Title == defaultTitle {
				if meta.Title != "" {
					cfg.Book.Title = meta.Title
				} else {
					cfg.Book.Title = filepath.Base(absDir)
				}
			}
			if meta.Version != "" {
				cfg.Book.Version = meta.Version
			}
			// Fallback: try git tag when version is still the default.
			if cfg.Book.Version == DefaultConfig().Book.Version {
				if tag := gitLatestTag(ctx, absDir); tag != "" {
					cfg.Book.Version = tag
				}
			}
			if meta.Language != "" {
				cfg.Book.Language = meta.Language
			}
			if meta.Author != "" {
				cfg.Book.Author = meta.Author
			} else if cfg.Book.Author == "" {
				cfg.Book.Author = gitConfigAuthor(ctx, absDir)
			}

			// Detect GLOSSARY.md.
			glossaryPath := filepath.Join(absDir, "GLOSSARY.md")
			if _, err := os.Stat(glossaryPath); err == nil {
				cfg.GlossaryFile = glossaryPath
			}

			// Detect LANGS.md.
			langsPath := filepath.Join(absDir, "LANGS.md")
			if _, err := os.Stat(langsPath); err == nil {
				cfg.LangsFile = langsPath
			}

			if err := cfg.Validate(); err != nil {
				return nil, fmt.Errorf("SUMMARY.md config validation failed: %w", err)
			}
			return cfg, nil
		}
		// If book.json doesn't exist and SUMMARY.md has no chapters, return error
		if !hasBookJSON {
			return nil, errors.New("SUMMARY.md contains no chapter definitions")
		}
	}

	// If book.json was found and we got this far, validate and return the config.
	if hasBookJSON {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("book.json config validation failed: %w", err)
		}
		return cfg, nil
	}

	// Priority 4: scan .md files directly.
	return autoDiscover(ctx, absDir)
}

// autoDiscover scans Markdown files and generates config automatically.
func autoDiscover(ctx context.Context, dir string) (*BookConfig, error) {
	cfg := DefaultConfig()
	cfg.baseDir = dir

	// Scan all Markdown files.
	mdFiles, err := findMarkdownFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to find markdown files: %w", err)
	}

	if len(mdFiles) == 0 {
		return nil, &DiscoverError{Dir: dir, Msg: "no .md files found in directory"}
	}

	// Split top-level README.md from other files.
	var readmeFile string
	var otherFiles []string
	for _, f := range mdFiles {
		relPath, err := filepath.Rel(dir, f)
		if err != nil {
			slog.Warn("failed to compute relative path", slog.String("dir", dir), slog.String("file", f), slog.Any("error", err))
			relPath = f // fallback to absolute path
		}
		baseName := strings.ToLower(filepath.Base(f))
		if baseName == "readme.md" && filepath.Dir(relPath) == "." {
			readmeFile = relPath
		} else {
			// Skip special project files and common documentation files
			// that should not appear as book chapters.
			if baseName == "summary.md" || baseName == "glossary.md" || baseName == "langs.md" ||
				baseName == "changelog.md" || baseName == "contributing.md" || baseName == "license.md" {
				continue
			}
			otherFiles = append(otherFiles, relPath)
		}
	}

	// Sort files in lexical order.
	sort.Strings(otherFiles)

	// Use top-level README.md as the first chapter when present.
	if readmeFile != "" {
		title := utils.ExtractTitleFromFile(filepath.Join(dir, readmeFile))
		if title == "" {
			title = "Preface"
		}
		cfg.Chapters = append(cfg.Chapters, ChapterDef{
			Title: title,
			File:  readmeFile,
		})

		// Reuse the README title as the book title.
		cfg.Book.Title = title
	}

	// Add the remaining chapters.
	for _, f := range otherFiles {
		title := utils.ExtractTitleFromFile(filepath.Join(dir, f))
		if title == "" {
			// Fall back to the file name.
			title = fileNameToTitle(f)
		}
		cfg.Chapters = append(cfg.Chapters, ChapterDef{
			Title: title,
			File:  f,
		})
	}

	// If README.md did not define a title, fall back to the first chapter title.
	if cfg.Book.Title == "Untitled Book" && len(cfg.Chapters) > 0 {
		firstTitle := utils.ExtractTitleFromFile(filepath.Join(dir, cfg.Chapters[0].File))
		if firstTitle != "" {
			cfg.Book.Title = firstTitle
		}
	}

	// Auto-detect GLOSSARY.md.
	glossaryPath := filepath.Join(dir, "GLOSSARY.md")
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("auto-discovered config validation failed: %w", err)
	}
	return cfg, nil
}

// findMarkdownFiles recursively finds Markdown files.
func findMarkdownFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("error during directory walk", slog.String("path", path), slog.Any("error", err))
			return nil // Skip inaccessible files.
		}

		// Skip hidden and dependency directories.
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_book" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect only .md files.
		if strings.ToLower(filepath.Ext(path)) == ".md" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// fileNameToTitle converts a file path into a readable title.
func fileNameToTitle(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Replace common separators with spaces.
	name = fileNameReplacer.Replace(name)

	// Uppercase the first rune (safe for multi-byte UTF-8).
	runes := []rune(name)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
		name = string(runes)
	}

	return name
}

// gitLatestTag returns the latest semver-like git tag (e.g. "v1.7.0" → "1.7.0"),
// or an empty string when git is unavailable or no tags exist.
func gitLatestTag(ctx context.Context, dir string) string {
	// Use describe to find the most recent tag reachable from HEAD.
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return ""
	}
	tag := strings.TrimSpace(string(out))
	return strings.TrimPrefix(tag, "v")
}

// gitConfigAuthor returns the git user.name configured in the given directory,
// or an empty string when git is unavailable or no name is set.
// The context is used to allow cancellation of the git command.
func gitConfigAuthor(ctx context.Context, dir string) string {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// DiscoverError describes auto-discovery failures.
type DiscoverError struct {
	Dir string
	Msg string
}

func (e *DiscoverError) Error() string {
	return e.Msg + ": " + e.Dir
}

// ReadmeMetadata holds metadata extracted from a project README.md.
type ReadmeMetadata struct {
	Title    string // Book title (may differ from H1 heading).
	Version  string // e.g. "1.6.5"
	Author   string // Detected author name or GitHub username.
	Language string // e.g. "zh-CN", "en-US"
}

// Magic number constants for metadata extraction and detection.
const (
	maxReadmeLineCount    = 200 // Maximum lines to scan from README.md
	minCJKTitleLength     = 4   // Minimum character length for CJK titles
	cjkDetectionThreshold = 0.2 // Ratio threshold for CJK language detection
)

// Patterns for extracting metadata from README.md.
var (
	// versionBoldPattern matches **vX.Y.Z** or **X.Y.Z**.
	versionBoldPattern = regexp.MustCompile(`\*\*v?([\d]+\.[\d]+(?:\.[\d]+)?)\*\*`)
	// githubUserPattern extracts username from GitHub URLs.
	githubUserPattern = regexp.MustCompile(`github\.com/([a-zA-Z0-9_-]+)/`)
	// authorPattern matches explicit author lines.
	authorPattern = regexp.MustCompile(`(?:作者|[Aa]uthor)[：:]\s*(.+)`)
	// badgeTitlePattern extracts titles from badge URLs (e.g. badge/([^-\]]+?)[-\]]).
	badgeTitlePattern = regexp.MustCompile(`badge/([^-\]]+?)[-\]]`)

	// fileNameReplacer converts underscores and hyphens to spaces.
	fileNameReplacer = strings.NewReplacer("_", " ", "-", " ")
)

// ExtractReadmeMetadata reads a README.md and extracts book metadata.
// It tries to find a meaningful title (beyond just the H1), version, language, and author.
// Exported so that cmd/init_cmd.go can also use it.
// The context is used for potentially long-running operations like git commands.
func ExtractReadmeMetadata(ctx context.Context, path string) ReadmeMetadata {
	f, err := os.Open(path)
	if err != nil {
		return ReadmeMetadata{}
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("failed to close README file", slog.String("path", path), slog.Any("error", err))
		}
	}()

	var meta ReadmeMetadata
	var h1Title string
	var allText strings.Builder
	var githubUser string
	lineCount := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineCount++
		if lineCount > maxReadmeLineCount {
			break // Only scan the first 200 lines.
		}
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		allText.WriteString(trimmed)
		allText.WriteString("\n")

		// Extract H1 title.
		if h1Title == "" && strings.HasPrefix(trimmed, "# ") {
			h1Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		}

		// Extract version from **vX.Y.Z** pattern.
		if meta.Version == "" {
			if matches := versionBoldPattern.FindStringSubmatch(trimmed); len(matches) >= 2 {
				meta.Version = matches[1]
			}
		}

		// Extract GitHub username from repo URLs.
		if githubUser == "" {
			if matches := githubUserPattern.FindStringSubmatch(trimmed); len(matches) >= 2 {
				githubUser = matches[1]
			}
		}

		// Extract explicit author line.
		if meta.Author == "" {
			if matches := authorPattern.FindStringSubmatch(trimmed); len(matches) >= 2 {
				meta.Author = strings.TrimSpace(matches[1])
			}
		}
	}

	// Check for scanner errors; log them for best-effort metadata extraction.
	if err := scanner.Err(); err != nil {
		// Error occurred during scanning, but we continue with the metadata extracted so far.
		// This is best-effort metadata extraction from README.
		slog.Warn("error reading README file during metadata extraction", slog.String("path", path), slog.Any("error", err))
	}

	content := allText.String()

	// Determine book title: try to find a meaningful title that is NOT just
	// a generic heading like "Preface" or "Introduction".
	meta.Title = inferBookTitle(h1Title, content, filepath.Dir(path))

	// Detect language from content.
	meta.Language = detectContentLanguage(content)

	// Fallback author to GitHub username.
	if meta.Author == "" && githubUser != "" {
		meta.Author = githubUser
	}

	// Last-resort fallback: try git config user.name in the book directory.
	if meta.Author == "" {
		meta.Author = gitConfigAuthor(ctx, filepath.Dir(path))
	}

	return meta
}

// inferBookTitle tries to find the real book title, not just the README H1.
// Strategy: check for a badge with Chinese book title → check SUMMARY.md first line →
// use H1 if it's not a generic heading → fallback to directory name.
func inferBookTitle(h1Title, content, dir string) string {
	// 1. Look for a Chinese book title in badge URLs (e.g. Docker%20%E6%8A%80%E6%9C%AF...).
	// These badges often contain the official book title, URL-encoded.
	for _, match := range badgeTitlePattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		candidate := match[1]
		// URL-decode to handle %E6%8A%80 style encoding.
		if decoded, err := url.PathUnescape(candidate); err == nil {
			candidate = decoded
		}
		candidate = strings.ReplaceAll(candidate, "+", " ")
		candidate = strings.TrimSpace(candidate)
		// Filter: must contain CJK characters (to find actual book titles, not "Stars" etc.)
		if utils.ContainsCJK(candidate) && len([]rune(candidate)) >= minCJKTitleLength {
			return candidate
		}
	}

	// 2. Check if SUMMARY.md has a top-level title.
	summaryPath := filepath.Join(dir, "SUMMARY.md")
	genericSummaryTitles := map[string]bool{
		"目录": true, "table of contents": true, "summary": true,
		"在线阅读": true, "read online": true, "contents": true,
	}
	if summaryTitle := utils.ExtractTitleFromFile(summaryPath); summaryTitle != "" && !genericSummaryTitles[strings.ToLower(summaryTitle)] {
		return summaryTitle
	}

	// 3. Use H1 if it's not a generic heading.
	genericH1s := map[string]bool{
		"前言": true, "preface": true, "readme": true, "introduction": true,
		"简介": true, "概述": true, "overview": true,
	}
	if h1Title != "" && !genericH1s[strings.ToLower(h1Title)] {
		return h1Title
	}

	// 4. Fallback to project directory name (cleaned up).
	dirName := filepath.Base(dir)
	dirName = strings.ReplaceAll(dirName, "_", " ")
	dirName = strings.ReplaceAll(dirName, "-", " ")
	dr := []rune(dirName)
	if len(dr) > 0 {
		dr[0] = unicode.ToUpper(dr[0])
		dirName = string(dr)
	}
	return dirName
}

// detectContentLanguage detects the primary language of content by CJK character ratio.
func detectContentLanguage(content string) string {
	if len(content) == 0 {
		return "en-US"
	}
	cjkCount := 0
	totalCount := 0
	for _, r := range content {
		if unicode.IsLetter(r) {
			totalCount++
			if utils.IsCJKRune(r) {
				cjkCount++
			}
		}
	}
	if totalCount == 0 {
		return "en-US"
	}
	ratio := float64(cjkCount) / float64(totalCount)
	if ratio > cjkDetectionThreshold {
		return "zh-CN" // Predominantly CJK content.
	}
	return "en-US"
}
