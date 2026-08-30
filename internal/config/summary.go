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
	"io"
	"os"
	"regexp"
	"strings"
)

// listItemLinkPattern matches a list item whose primary content is a single
// Markdown link: "* [Title](path)" or "- [Title](path)" or "+ [Title](path)".
// Lines where the link is embedded in prose (e.g., "* `A轨`：从 [第一章](…)")
// do NOT match, preventing navigation paragraphs from being parsed as chapters.
var listItemLinkPattern = regexp.MustCompile(`^[*+\-]\s+\[([^\]]+)\]\(([^)]+)\)\s*$`)

// isSummaryTitle reports whether a level-1 SUMMARY.md heading is the file's
// conventional self-title rather than a part label. Other h1 labels are kept:
// mdBook books use "# Part One" headings for numbered parts.
func isSummaryTitle(label string) bool {
	switch strings.ToLower(label) {
	case "summary", "table of contents", "contents", "toc":
		return true
	case "目录", "目次", "차례", "목차":
		return true
	}
	return false
}

// ParseSummary parses chapter definitions from SUMMARY.md.
// Nesting is expressed with indentation: two spaces or one tab per level.
func ParseSummary(path string) ([]ChapterDef, error) {
	// Limit file size to guard against malformed or malicious inputs.
	// Check size via os.Stat before reading to avoid loading large files into memory.
	const maxSummarySize = 10 * 1024 * 1024 // 10MB
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat SUMMARY.md: %w", err)
	}
	if info.Size() > int64(maxSummarySize) {
		return nil, fmt.Errorf("SUMMARY.md is too large (%d bytes; max allowed is %d bytes)", info.Size(), maxSummarySize)
	}

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
	// Group label from the most recent heading, pending attachment to the next
	// chapter parsed.
	pendingSection := ""

	// Belt-and-suspenders: also limit the reader to guard against TOCTOU
	// races where the file could grow between Stat and Open.
	scanner := bufio.NewScanner(io.LimitReader(f, int64(maxSummarySize)+1))
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// A heading groups the chapters that follow it. GitBook and HonKit use
		// "## Part I" this way, and dropping them collapsed a structured book
		// into one flat, unscannable sidebar. The label is held here and
		// attached to the next chapter parsed.
		if strings.HasPrefix(trimmed, "#") {
			label := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			// "# Summary" (or a localized equivalent) is the document's own
			// title — the GitBook and mdBook convention — not a part label;
			// treating it as one stamped "Summary" over the first group of
			// chapters in the sidebar, the ePub nav and the PDF contents.
			if label != "" && (!strings.HasPrefix(trimmed, "# ") || !isSummaryTitle(label)) {
				pendingSection = label
			}
			continue
		}

		// Parse indentation depth.
		indent := countIndent(line)

		// Only accept list items whose primary content is a single link.
		// This skips prose lines that happen to contain inline links
		// (e.g., navigation guides like "* `A轨`：从 [第一章](…) → …").
		matches := listItemLinkPattern.FindStringSubmatch(trimmed)
		if len(matches) < 3 {
			// Lines with a list marker but no direct link may be navigation
			// prose or a formatting issue — skip silently.
			continue
		}

		title := strings.TrimSpace(matches[1])
		file := strings.TrimSpace(matches[2])
		section := pendingSection
		pendingSection = ""

		// Skip anchor-only links such as #introduction. Keep the pending label
		// so it lands on the next real chapter instead of being lost here.
		if strings.HasPrefix(file, "#") {
			pendingSection = section
			continue
		}

		ch := ChapterDef{
			Title:   title,
			File:    file,
			Section: section,
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

	// Detect silent truncation from LimitReader: if we consumed exactly the
	// limit, the file may have grown between Stat and Open (TOCTOU).
	var probe [1]byte
	if _, readErr := io.ReadFull(io.LimitReader(f, 1), probe[:]); readErr == nil {
		return nil, fmt.Errorf("SUMMARY.md exceeds size limit (%d bytes)", maxSummarySize)
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
