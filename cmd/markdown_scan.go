// markdown_scan.go provides code-aware line scanning for the Markdown checks
// that run outside the parser (validate's link and image extraction).
//
// Those checks read files line by line with regexes. Without fence tracking
// they treat every example inside a ``` block as a real reference, so any book
// that documents Markdown — including mdPress's own manual — fails validation
// with a wall of errors for links that were never meant to resolve.
package cmd

import (
	"regexp"
	"strings"
)

// fencePattern matches a fenced code block delimiter: three or more backticks
// or tildes, indented by at most three spaces, per CommonMark.
var fencePattern = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")

// fenceTracker follows fenced-code-block state across the lines of one file.
// The zero value is ready to use and starts outside any block.
type fenceTracker struct {
	// marker is the run of characters that opened the current block, or ""
	// when outside one.
	marker string
	// openedAt is the 1-based line number of the opening fence, or 0 when
	// outside a block.
	openedAt int
	// line counts the lines fed to InCode so an unterminated fence can be
	// reported at its opening position.
	line int
}

// InCode reports whether line belongs to a fenced code block, and advances the
// tracker. Fence delimiters themselves count as code, so a caller can simply
// skip every line for which this returns true.
func (f *fenceTracker) InCode(line string) bool {
	f.line++
	m := fencePattern.FindStringSubmatch(line)
	if m == nil {
		return f.marker != ""
	}
	delimiter, info := m[1], m[2]

	if f.marker == "" {
		// An opening fence may carry an info string (```go).
		f.marker = delimiter
		f.openedAt = f.line
		return true
	}
	// A closing fence must use the same character, be at least as long, and
	// carry no info string; otherwise it is content inside the block.
	if delimiter[0] == f.marker[0] &&
		len(delimiter) >= len(f.marker) &&
		strings.TrimSpace(info) == "" {
		f.marker = ""
		f.openedAt = 0
	}
	return true
}

// inlineCodePattern matches an inline code span, including the multi-backtick
// forms used to quote text that itself contains backticks.
var inlineCodePattern = regexp.MustCompile("`+[^`]*`+")

// stripInlineCode blanks out inline code spans so a path mentioned as
// `see ./missing.md` is not reported as a broken reference. Spans are replaced
// with spaces rather than removed, so byte offsets into the line stay valid
// for callers that use FindAllStringSubmatchIndex.
func stripInlineCode(line string) string {
	return inlineCodePattern.ReplaceAllStringFunc(line, func(span string) string {
		return strings.Repeat(" ", len(span))
	})
}

// Open reports the line number of the fence that is still open, or 0 when the
// tracker is outside a block. Callers check this after the last line: an
// unterminated fence swallows the remainder of the document into a code block,
// which is silent, easy to introduce, and hard to spot in the source.
func (f *fenceTracker) Open() int {
	return f.openedAt
}

// scannableLine returns the part of a Markdown line that reference checks
// should look at: "" when the line is inside a code fence, otherwise the line
// with inline code spans blanked out.
func scannableLine(line string, fences *fenceTracker) string {
	if fences.InCode(line) {
		return ""
	}
	return stripInlineCode(line)
}
