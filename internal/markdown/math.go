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
	// Handle block math ($$...$$) first to avoid the inline pattern matching
	// against the inner $ of a $$ delimiter.
	md = blockMathPattern.ReplaceAllStringFunc(md, func(match string) string {
		sub := blockMathPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx := len(m.blocks)
		m.blocks = append(m.blocks, sub[1])
		return fmt.Sprintf("MDPMATHBLOCK%06d", idx)
	})

	// Handle inline math ($...$).
	md = inlineMathPattern.ReplaceAllStringFunc(md, func(match string) string {
		sub := inlineMathPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx := len(m.inlines)
		m.inlines = append(m.inlines, sub[1])
		return fmt.Sprintf("MDPMATHINLINE%06d", idx)
	})
	return md
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
		placeholder := fmt.Sprintf("MDPMATHBLOCK%06d", i)
		// HTML-escape the formula content to prevent XSS injection.
		// KaTeX auto-render reads textContent from DOM nodes, so the
		// browser-decoded text still contains the original LaTeX.
		escaped := html.EscapeString(content)
		replacement := `<span class="math math-display">$$` + escaped + `$$</span>`
		htmlStr = strings.ReplaceAll(htmlStr, placeholder, replacement)
	}
	for i, content := range m.inlines {
		placeholder := fmt.Sprintf("MDPMATHINLINE%06d", i)
		escaped := html.EscapeString(content)
		replacement := `<span class="math math-inline">$` + escaped + `$</span>`
		htmlStr = strings.ReplaceAll(htmlStr, placeholder, replacement)
	}
	return htmlStr
}

// HasMath reports whether the Markdown source contains any math formula syntax.
// Used as a quick check to decide whether math processing is needed.
func HasMath(md string) bool {
	return strings.Contains(md, "$")
}
