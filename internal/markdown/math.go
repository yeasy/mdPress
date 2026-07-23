// math.go implements pre/post processing for math formulas.
//
// Problem: goldmark follows CommonMark spec where `_` inside words may be
// treated as emphasis delimiters, so `$x_1^2$` becomes `$x<em>1</em>^2$`,
// breaking the formula structure.
//
// Solution: Before goldmark processes the Markdown source, replace $$...$$ and
// $...$ with placeholder tokens (e.g. MDPMATHBLOCK000000) that contain no
// Markdown special characters. After goldmark renders HTML, replace the
// placeholders back with HTML span elements that KaTeX auto-render can find.
package markdown

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

const (
	// mathBlockFmt is the format string for display math placeholder tokens.
	mathBlockFmt = "MDPMATHBLOCK%06d"
	// mathInlineFmt is the format string for inline math placeholder tokens.
	mathInlineFmt = "MDPMATHINLINE%06d"
)

// blockMathPattern matches display math delimited by $$...$$ (multiline).
// Non-greedy to avoid spanning multiple formula blocks.
var blockMathPattern = regexp.MustCompile(`(?s)\$\$(.*?)\$\$`)

// inlineMathPattern matches inline math delimited by $...$.
// Constraints:
//   - Does not overlap with $$ (block math is handled first)
//   - Content must not contain newlines
//   - Content must not start or end with a space (avoids matching currency like $5)
var inlineMathPattern = regexp.MustCompile(`\$([^$\n\t ][^$\n]*?[^$\n\t ]|[^$\n\t ])\$`)

// mathPreprocessor stores extracted math formulas so they can be restored after
// goldmark has finished processing the Markdown source.
type mathPreprocessor struct {
	blocks  []string // display math formula contents (without $$ delimiters)
	inlines []string // inline math formula contents (without $ delimiters)
}

// newMathPreprocessor creates a new math preprocessor.
func newMathPreprocessor() *mathPreprocessor {
	return &mathPreprocessor{}
}

// preprocess replaces $$...$$ and $...$ in the Markdown source with safe
// placeholder tokens. The placeholders only contain alphanumeric characters so
// they cannot trigger any Markdown syntax rules inside goldmark.
//
// Placeholder formats:
//
//	MDPMATHBLOCK000000   for display math
//	MDPMATHINLINE000000  for inline math
func (m *mathPreprocessor) preprocess(md string) string {
	// Split the source into fenced-code and non-fenced segments so that math
	// substitution is never applied inside fenced code blocks (``` or ~~~).
	// Code samples routinely contain '$' characters (shell "$HOME", "$$" for a
	// PID, Makefile/awk "$$", etc.) that must not be mangled into math spans.
	//
	// Block math ($$...$$) can legitimately span multiple lines, so it is
	// applied to each non-fenced segment as a whole. Inline math ($...$) is
	// applied per line while skipping inline code spans (`...`).
	return ProcessOutsideCode(md, m.substituteMath)
}

// ProcessOutsideCode applies fn to every part of a Markdown document that is
// not inside a fenced code block, leaving fenced content byte-for-byte intact.
//
// Any transformation of Markdown source has to respect this boundary: a book
// that documents a tool will show that tool's own syntax inside fences, and
// rewriting it there corrupts the very thing the page is trying to display.
// (Inline `code` spans are a separate, narrower concern — see
// ProcessOutsideCodeSpans.)
func ProcessOutsideCode(md string, fn func(string) string) string {
	var out strings.Builder
	lines := splitLinesKeepEndings(md)

	var nonFenced strings.Builder // accumulates consecutive non-fenced lines
	flush := func() {
		if nonFenced.Len() == 0 {
			return
		}
		out.WriteString(fn(nonFenced.String()))
		nonFenced.Reset()
	}

	inFence := false
	var fenceChar byte // '`' or '~'
	var fenceLen int   // length of the opening fence marker

	for _, line := range lines {
		if !inFence {
			if ch, n, ok := mathFenceMarker(line); ok {
				// Opening fence: flush pending non-fenced content, then emit the
				// fence line verbatim and enter fenced state.
				flush()
				out.WriteString(line)
				inFence = true
				fenceChar = ch
				fenceLen = n
				continue
			}
			nonFenced.WriteString(line)
			continue
		}

		// Inside a fenced block: emit lines verbatim. A fence closes only on a
		// matching marker of the same char and at least the opening length,
		// with no info string after it.
		out.WriteString(line)
		if ch, n, ok := mathFenceMarker(line); ok && ch == fenceChar && n >= fenceLen && !mathFenceHasInfo(line) {
			inFence = false
		}
	}
	flush()
	return out.String()
}

// ProcessOutsideCodeSpans applies fn outside backtick-delimited inline code
// spans, leaving their contents unchanged.
func ProcessOutsideCodeSpans(text string, fn func(string) string) string {
	return mathProcessOutsideCodeSpans(text, fn)
}

// substituteMath applies block-math and inline-math substitution to a chunk of
// non-fenced Markdown source. Block math is handled first (it may span multiple
// lines); inline math is then applied line-by-line, only outside inline code
// spans.
func (m *mathPreprocessor) substituteMath(src string) string {
	// Handle block math ($$...$$) first to avoid the inline pattern matching
	// against the inner $ of a $$ delimiter.
	src = blockMathPattern.ReplaceAllStringFunc(src, func(match string) string {
		sub := blockMathPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx := len(m.blocks)
		m.blocks = append(m.blocks, sub[1])
		return fmt.Sprintf(mathBlockFmt, idx)
	})

	// Handle inline math ($...$), skipping inline code spans so that '$'
	// characters inside `...` are left untouched.
	return mathProcessOutsideCodeSpans(src, func(segment string) string {
		return inlineMathPattern.ReplaceAllStringFunc(segment, func(match string) string {
			sub := inlineMathPattern.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			idx := len(m.inlines)
			m.inlines = append(m.inlines, sub[1])
			return fmt.Sprintf(mathInlineFmt, idx)
		})
	})
}

// splitLinesKeepEndings splits s into lines, keeping each line's trailing "\n"
// (the final line has no newline unless the source ends with one). This lets
// preprocess reassemble the source without altering line endings.
func splitLinesKeepEndings(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// mathFenceMarker reports whether line begins a fenced-code delimiter (a run of
// at least three '`' or '~', optionally preceded by up to three spaces of
// indentation). It returns the fence character, the marker length, and true.
func mathFenceMarker(line string) (byte, int, bool) {
	i := 0
	// Allow up to three leading spaces of indentation (CommonMark).
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return 0, 0, false
	}
	ch := line[i]
	n := 0
	for i+n < len(line) && line[i+n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	return ch, n, true
}

// mathFenceHasInfo reports whether the fence line carries a non-empty info
// string after its marker (e.g. "```bash"). A closing fence must have no info
// string, so this is used to reject candidate closers that carry one.
func mathFenceHasInfo(line string) bool {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	ch := line[i]
	for i < len(line) && line[i] == ch {
		i++
	}
	rest := strings.TrimRight(line[i:], "\r\n")
	return strings.TrimSpace(rest) != ""
}

// mathProcessOutsideCodeSpans splits text on backtick-delimited inline code
// spans (any number of consecutive backticks), applies fn only to non-code
// regions, and preserves code span content unchanged. Unmatched backticks are
// treated as regular text and passed through fn. This mirrors the approach used
// in internal/typst so that '$' inside `...` is never treated as math.
func mathProcessOutsideCodeSpans(text string, fn func(string) string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		backtickLen := 0
		for i+backtickLen < len(text) && text[i+backtickLen] == '`' {
			backtickLen++
		}

		if backtickLen == 0 {
			start := i
			for i < len(text) && text[i] != '`' {
				i++
			}
			result.WriteString(fn(text[start:i]))
			continue
		}

		delimiter := text[i : i+backtickLen]
		closeIdx := strings.Index(text[i+backtickLen:], delimiter)
		if closeIdx == -1 {
			// No matching close — treat the rest as regular text.
			result.WriteString(fn(text[i:]))
			return result.String()
		}

		spanEnd := i + backtickLen + closeIdx + backtickLen
		result.WriteString(text[i:spanEnd])
		i = spanEnd
	}
	return result.String()
}

// postprocess restores placeholders to HTML elements that KaTeX auto-render
// can recognize and render.
//
// Display math placeholder → <span class="math math-display">$$...$$</span>
// Inline math placeholder  → <span class="math math-inline">$...$</span>
//
// KaTeX auto-render is configured with $$ and $ delimiters, so it scans the
// text nodes inside these spans and renders the LaTeX.
func (m *mathPreprocessor) postprocess(htmlStr string) string {
	for i, content := range m.blocks {
		placeholder := fmt.Sprintf(mathBlockFmt, i)
		// HTML-escape the formula content to prevent XSS injection.
		// KaTeX auto-render reads textContent from DOM nodes, so the
		// browser-decoded text still contains the original LaTeX.
		escaped := html.EscapeString(content)
		replacement := `<span class="math math-display">$$` + escaped + `$$</span>`
		htmlStr = strings.ReplaceAll(htmlStr, placeholder, replacement)
	}
	for i, content := range m.inlines {
		placeholder := fmt.Sprintf(mathInlineFmt, i)
		escaped := html.EscapeString(content)
		replacement := `<span class="math math-inline">$` + escaped + `$</span>`
		htmlStr = strings.ReplaceAll(htmlStr, placeholder, replacement)
	}
	return htmlStr
}
