// summary.go parses chapter structure from SUMMARY.md.
// SUMMARY.md uses Markdown link lists to define chapter order in a GitBook-compatible format.
//
// Format example:
//
//	# Summary
//
//	* [Preface](preface.md)
//	* [Chapter 1](chapter01/README.md)
//	  * [Section 1.1](chapter01/section01.md)
//	* [Chapter 2](chapter02/README.md)
package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// linkPattern matches Markdown links: [title](path).
// NOTE: intentionally duplicated in internal/i18n/langs.go.
// Consolidating would add an unnatural dependency between config and i18n
// (neither package imports the other or pkg/utils for regex patterns).
var linkPattern = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// ParseSummary parses chapter definitions from SUMMARY.md.
// Nesting is expressed with indentation: two spaces or one tab per level.
func ParseSummary(path string) ([]ChapterDef, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SUMMARY.md: %w", err)
	}
	defer f.Close() //nolint:errcheck

	var chapters []ChapterDef
	// Track nesting with a stack of (indent, *[]ChapterDef).
	type stackFrame struct {
		indent int
		list   *[]ChapterDef
	}
	stack := []stackFrame{{indent: -1, list: &chapters}}

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Skip blank lines and headings.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Parse indentation depth.
		indent := countIndent(line)

		// Extract the Markdown link.
		matches := linkPattern.FindStringSubmatch(trimmed)
		if len(matches) < 3 {
			// Non-link, non-blank, non-heading lines that contain list markers
			// may indicate a formatting error in SUMMARY.md.
			if strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "+") {
				return nil, fmt.Errorf("SUMMARY.md line %d: list item has no Markdown link: %q", lineNum, trimmed)
			}
			continue // Skip non-link lines.
		}

		title := strings.TrimSpace(matches[1])
		file := strings.TrimSpace(matches[2])

		// Skip anchor-only links such as #introduction.
		if strings.HasPrefix(file, "#") {
			continue
		}

		ch := ChapterDef{
			Title: title,
			File:  file,
		}

		// Pop the stack until the parent indent is smaller.
		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		// Append to the current nesting level.
		parent := stack[len(stack)-1].list
		*parent = append(*parent, ch)

		// Push the current entry sections to support nested children.
		newEntry := &(*parent)[len(*parent)-1]
		stack = append(stack, stackFrame{indent: indent, list: &newEntry.Sections})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read SUMMARY.md: %w", err)
	}

	return chapters, nil
}

// countIndent counts leading indentation, treating tabs as two spaces.
func countIndent(line string) int {
	indent := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			indent++
		case '\t':
			indent += 2
		default:
			return indent
		}
	}
	return indent
}
