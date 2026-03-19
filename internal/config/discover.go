// discover.go implements zero-config project discovery.
// When neither book.yaml nor SUMMARY.md exists, mdpress scans .md files,
// sorts them, and derives chapter metadata automatically.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Discover auto-discovers project configuration in a directory.
// Priority: book.yaml > SUMMARY.md > Markdown file scanning.
func Discover(dir string) (*BookConfig, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Priority 1: load book.yaml.
	bookYamlPath := filepath.Join(absDir, "book.yaml")
	if _, err := os.Stat(bookYamlPath); err == nil {
		return Load(bookYamlPath)
	}

	// Priority 2: load SUMMARY.md.
	summaryPath := filepath.Join(absDir, "SUMMARY.md")
	if _, err := os.Stat(summaryPath); err == nil {
		return loadFromSummary(absDir, summaryPath)
	}

	// Priority 3: scan .md files directly.
	return autoDiscover(absDir)
}

// loadFromSummary builds config from SUMMARY.md.
func loadFromSummary(dir, summaryPath string) (*BookConfig, error) {
	chapters, err := ParseSummary(summaryPath)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	cfg.baseDir = dir
	cfg.Chapters = chapters

	if len(chapters) == 0 {
		return nil, fmt.Errorf("SUMMARY.md contains no chapter definitions")
	}

	// Try to derive the book title from README.md.
	readmePath := filepath.Join(dir, "README.md")
	if title := extractTitleFromFile(readmePath); title != "" {
		cfg.Book.Title = title
	}

	// Detect GLOSSARY.md.
	glossaryPath := filepath.Join(dir, "GLOSSARY.md")
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	// Detect LANGS.md.
	langsPath := filepath.Join(dir, "LANGS.md")
	if _, err := os.Stat(langsPath); err == nil {
		cfg.LangsFile = langsPath
	}

	return cfg, nil
}

// autoDiscover scans Markdown files and generates config automatically.
func autoDiscover(dir string) (*BookConfig, error) {
	cfg := DefaultConfig()
	cfg.baseDir = dir

	// Scan all Markdown files.
	mdFiles, err := findMarkdownFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(mdFiles) == 0 {
		return nil, &DiscoverError{Dir: dir, Msg: "no .md files found in directory"}
	}

	// Split top-level README.md from other files.
	var readmeFile string
	var otherFiles []string
	for _, f := range mdFiles {
		relPath, _ := filepath.Rel(dir, f)
		baseName := strings.ToLower(filepath.Base(f))
		if baseName == "readme.md" && filepath.Dir(relPath) == "." {
			readmeFile = relPath
		} else {
			// Skip special project files.
			if baseName == "summary.md" || baseName == "glossary.md" || baseName == "langs.md" {
				continue
			}
			otherFiles = append(otherFiles, relPath)
		}
	}

	// Sort files in lexical order.
	sort.Strings(otherFiles)

	// Use top-level README.md as the first chapter when present.
	if readmeFile != "" {
		title := extractTitleFromFile(filepath.Join(dir, readmeFile))
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
		title := extractTitleFromFile(filepath.Join(dir, f))
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
		firstTitle := extractTitleFromFile(filepath.Join(dir, cfg.Chapters[0].File))
		if firstTitle != "" {
			cfg.Book.Title = firstTitle
		}
	}

	// Auto-detect GLOSSARY.md.
	glossaryPath := filepath.Join(dir, "GLOSSARY.md")
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	return cfg, nil
}

// findMarkdownFiles recursively finds Markdown files.
func findMarkdownFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
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

// extractTitleFromFile returns the first H1 title from a Markdown file.
func extractTitleFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}
	return ""
}

// fileNameToTitle converts a file path into a readable title.
func fileNameToTitle(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Replace common separators with spaces.
	name = strings.NewReplacer(
		"_", " ",
		"-", " ",
	).Replace(name)

	// Uppercase the first letter.
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}

// DiscoverError describes auto-discovery failures.
type DiscoverError struct {
	Dir string
	Msg string
}

func (e *DiscoverError) Error() string {
	return e.Msg + ": " + e.Dir
}
