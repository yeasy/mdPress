// Package i18n provides multi-language book support.
// It parses language definitions from LANGS.md, where each language maps to a subdirectory.
//
// LANGS.md 格式：
//
//	# Languages
//
//	* [English](en/)
//	* [中文](zh/)
package i18n

import (
	"bufio"
	"fmt"
	"os"
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
var linkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// ParseLangsFile parses language definitions from LANGS.md.
func ParseLangsFile(path string) ([]LangDef, error) {
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
		// Trim any trailing slash from the directory.
		dir = strings.TrimRight(dir, "/")

		if name != "" && dir != "" {
			langs = append(langs, LangDef{Name: name, Dir: dir})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read LANGS.md: %w", err)
	}

	if len(langs) == 0 {
		return nil, fmt.Errorf("no language definitions found in LANGS.md")
	}

	return langs, nil
}

// HasLangsFile reports whether LANGS.md exists in a directory.
func HasLangsFile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "LANGS.md"))
	return err == nil
}
