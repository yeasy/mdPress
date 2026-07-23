// Package i18n provides multi-language book support.
// It parses language definitions from LANGS.md, where each language maps to a subdirectory.
//
// LANGS.md format:
//
//	# Languages
//
//	* [English](en/)
//	* [Chinese](zh/)
package i18n

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// LangDef describes one language variant.
type LangDef struct {
	Name string // Display name, for example "English".
	Dir  string // Subdirectory path, for example "en" or "zh".
}

// linkPattern matches Markdown links.
// NOTE: intentionally duplicated in internal/config/summary.go.
// Consolidating would add an unnatural dependency between i18n and config
// (neither package imports the other or pkg/utils for regex patterns).
var linkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// maxLangsFileSize is the maximum size for LANGS.md to prevent resource exhaustion.
const maxLangsFileSize = 1 * 1024 * 1024 // 1 MB

// ParseLangsFile parses language definitions from LANGS.md.
// Entries are resolved relative to the directory holding LANGS.md.
func ParseLangsFile(path string) ([]LangDef, error) {
	root := filepath.Dir(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open LANGS.md: %w", err)
	}
	if info.Size() > maxLangsFileSize {
		return nil, fmt.Errorf("LANGS.md exceeds maximum size of %d bytes", maxLangsFileSize)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open LANGS.md: %w", err)
	}
	defer f.Close() //nolint:errcheck

	var langs []LangDef
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := linkPattern.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		name := strings.TrimSpace(matches[1])
		dir := strings.TrimSpace(matches[2])
		// Trim any trailing path separator (both forward and back slash for Windows).
		dir = strings.TrimRight(dir, "/\\")

		if name != "" && dir != "" {
			// Reject absolute paths and path traversal attempts.
			if filepath.IsAbs(dir) || strings.Contains(dir, "..") {
				return nil, fmt.Errorf("LANGS.md: directory %q contains path traversal or absolute path", dir)
			}
			resolved, err := languageDir(root, dir)
			if err != nil {
				return nil, err
			}
			langs = append(langs, LangDef{Name: name, Dir: resolved})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read LANGS.md: %w", err)
	}

	if len(langs) == 0 {
		return nil, errors.New("no language definitions found in LANGS.md")
	}

	return langs, nil
}

// languageDir turns one LANGS.md link target into the language subdirectory
// the rest of the build expects.
//
// The manual has long shown "[English](en/README.md)" as an example, and a
// link to the language's landing page is the natural thing to write anyway.
// Taken literally that made the build try to treat a file as an output
// directory, so point-at-a-file entries are folded up to their directory.
// Only that rewrite is validated against the filesystem: a plain entry naming
// a directory that does not exist yet is left alone, so the caller still
// reports the missing language directory with its own message.
func languageDir(root, dir string) (string, error) {
	pointsAtFile := strings.EqualFold(filepath.Ext(dir), ".md")
	if !pointsAtFile {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err == nil && !info.IsDir() {
			pointsAtFile = true
		}
	}
	if !pointsAtFile {
		return dir, nil
	}

	parent := path.Dir(filepath.ToSlash(dir))
	if parent != "." && parent != "/" {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(parent))); err == nil && info.IsDir() {
			return parent, nil
		}
	}
	hint := "en"
	if parent != "." && parent != "/" {
		hint = parent
	}
	return "", fmt.Errorf("LANGS.md: entry %q must point at a language directory, e.g. [Name](%s/)", dir, hint)
}

// hasLangsFile reports whether LANGS.md exists in a directory.
func hasLangsFile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "LANGS.md"))
	return err == nil
}
