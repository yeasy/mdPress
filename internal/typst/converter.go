// Package typst provides a Markdown-to-Typst converter and Typst PDF generator.
package typst

import (
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Package-level regexp patterns for performance optimization.
var (
	imagePattern          = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	boldAsteriskPattern   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	boldUnderscorePattern = regexp.MustCompile(`__([^_]+)__`)
)

// MarkdownToTypstConverter converts Markdown syntax to Typst syntax.
// Typst syntax is quite close to Markdown, with some key differences:
//   - `# Heading` → `= Heading`
//   - `**bold**` → `*bold*`
//   - `*italic*` → `_italic_`
//   - “ `code` “ → “ `code` “
//   - ```code blocks``` → ``` ```code blocks``` ``` (block syntax)
//   - `![alt](img)` → `#image("img")`
//   - `[text](url)` → `#link("url")[text]`
type MarkdownToTypstConverter struct {
	// We'll process the markdown line by line, handling block-level
	// elements and inline formatting.
}

// Convert takes Markdown text and converts it to Typst markup.
func (c *MarkdownToTypstConverter) Convert(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	var inCodeBlock bool
	var codeBlockLang string
	var codeFenceLen int
	var codeBlockContent strings.Builder

	// A paragraph is accumulated across its soft-wrapped source lines and
	// converted as one unit. Converting each source line on its own splits an
	// inline span at the wrap: `*emphasis that\ncontinues*` became `*emphasis
	// that` + `continues*`, two lines each carrying one unbalanced `*`, and
	// Typst rejected the file with "unclosed delimiter" so no PDF was written.
	// A single Markdown newline inside a paragraph is a space, so the lines are
	// joined with one.
	var paragraph []string
	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		result.WriteString(c.convertInline(strings.Join(paragraph, " ")))
		result.WriteString("\n\n")
		paragraph = paragraph[:0]
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Inside a code block, only a fence of backticks at least as long as
		// the opening one and carrying no info string closes it (CommonMark).
		// Tracking the opening length keeps an inner ``` fence from prematurely
		// closing a book that documents a fenced block with a longer ````
		// outer fence — otherwise the language tag and body were mis-parsed and
		// Typst failed with "unclosed raw text".
		if inCodeBlock {
			if n := leadingBacktickRun(line); n >= codeFenceLen && strings.TrimRight(line[n:], " \t") == "" {
				inCodeBlock = false
				typstCode := c.convertCodeBlock(codeBlockContent.String(), codeBlockLang)
				result.WriteString(typstCode)
				result.WriteString("\n\n")
				continue
			}
			codeBlockContent.WriteString(line)
			codeBlockContent.WriteString("\n")
			continue
		}

		// Start of a code block: a run of three or more backticks at the start
		// of the line. The full run length is recorded (not assumed to be 3) so
		// a ```` documentation fence is closed by a matching ```` and its
		// language tag is parsed correctly.
		if n := leadingBacktickRun(line); n >= 3 {
			flushParagraph()
			inCodeBlock = true
			codeFenceLen = n
			codeBlockLang = strings.TrimSpace(line[n:])
			codeBlockContent.Reset()
			continue
		}

		// Handle headings (require space after # per Markdown spec)
		if strings.HasPrefix(line, "#") {
			level := countLeadingChars(line, '#')
			if level <= 6 && len(line) > level && line[level] == ' ' {
				heading := strings.TrimSpace(line[level+1:])
				if heading != "" {
					flushParagraph()
					// Convert # to =, ## to ==, etc.
					typstLevel := strings.Repeat("=", level)
					result.WriteString(typstLevel)
					result.WriteString(" ")
					result.WriteString(c.convertInline(heading))
					result.WriteString("\n\n")
				}
				continue
			}
			// Not a valid heading (e.g., #hashtag); fall through to paragraph handling
		}

		// Handle lists (unordered)
		trimmedForList := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmedForList, "- ") || strings.HasPrefix(trimmedForList, "* ") {
			flushParagraph()
			depth := countLeadingSpaces(line) / 2
			item := strings.TrimSpace(trimmedForList[2:])
			result.WriteString(strings.Repeat("  ", depth))
			result.WriteString("- ")
			result.WriteString(c.convertInline(item))
			result.WriteString("\n")
			continue
		}

		// Handle ordered lists
		if isOrderedListItem(line) {
			flushParagraph()
			depth := countLeadingSpaces(line) / 2
			item := extractListItemContent(line)
			result.WriteString(strings.Repeat("  ", depth))
			result.WriteString("+ ")
			result.WriteString(c.convertInline(item))
			result.WriteString("\n")
			continue
		}

		// Handle blockquotes
		if content, ok := strings.CutPrefix(line, "> "); ok {
			flushParagraph()
			result.WriteString("> ")
			result.WriteString(c.convertInline(content))
			result.WriteString("\n")
			continue
		}

		// Handle horizontal rules
		if isHorizontalRule(line) {
			flushParagraph()
			result.WriteString("---\n\n")
			continue
		}

		// Handle empty lines: a blank line ends the current paragraph, and is
		// otherwise kept as the separator that follows a list item or blockquote.
		if strings.TrimSpace(line) == "" {
			flushParagraph()
			result.WriteString("\n")
			continue
		}

		// Accumulate a paragraph line; flushed at the next blank or block-level
		// line, or at end of input.
		paragraph = append(paragraph, line)
	}

	flushParagraph()

	// Flush an unclosed code block at EOF so content is preserved.
	if inCodeBlock {
		typstCode := c.convertCodeBlock(codeBlockContent.String(), codeBlockLang)
		result.WriteString(typstCode)
		result.WriteString("\n\n")
		slog.Warn("Typst converter: unclosed code block at end of input, preserved content",
			slog.String("lang", codeBlockLang))
	}

	return strings.TrimSpace(result.String())
}

// convertInline processes inline Markdown formatting, skipping code spans.
func (c *MarkdownToTypstConverter) convertInline(text string) string {
	return c.processOutsideCodeSpans(text, func(segment string) string {
		// Convert images and links first: these emit Typst markup (#image,
		// #link) and carry URLs that must NOT be escaped as prose. Their
		// output is protected behind placeholder tokens so the subsequent
		// prose-escaping pass leaves the markup and URLs intact.
		segment = c.convertImages(segment)
		segment = c.convertLinks(segment)

		var tokens []string
		segment, tokens = protectTypstMarkup(segment, tokens)

		// Escape Typst control characters in the remaining prose. Bold/italic
		// markers (* and _) are intentionally left for the passes below.
		segment = escapeTypstProse(segment)

		segment = c.convertItalic(segment)
		segment = c.convertBold(segment)

		// Restore the protected #image()/#link() markup.
		segment = restoreTypstMarkup(segment, tokens)
		return segment
	})
}

// typstProseReplacer escapes Typst control characters that appear in ordinary
// prose and would otherwise break compilation ($, #, @, <, >, `). It
// deliberately does NOT touch '*' or '_' (used for bold/italic conversion) or
// brackets/parens.
//
// Backticks are escaped here even though matched code spans are extracted
// upstream: an UNMATCHED backtick run still reaches prose escaping (e.g. the
// manual's sentence "A ```plantuml block is published as a plain code block"),
// and a bare ``` run left verbatim opens a Typst raw block that swallows the
// rest of the document — the compile then fails with "unclosed raw text". A
// literal backtick in Typst is written "\`".
var typstProseReplacer = strings.NewReplacer(
	"$", "\\$",
	"#", "\\#",
	"@", "\\@",
	"<", "\\<",
	">", "\\>",
	"`", "\\`",
)

// escapeTypstProse escapes Typst control characters in a plain-text prose
// segment. It runs after image/link markup has been protected so that Typst
// markup and URLs (which may legitimately contain '#', '@', etc.) are untouched.
func escapeTypstProse(text string) string {
	return typstProseReplacer.Replace(text)
}

// typstMarkupPattern matches the markup emitted by convertImages/convertLinks
// so it can be shielded from prose escaping. Only the URL-bearing prefix is
// protected: the whole #image("...") span, and the #link("...") prefix (its
// trailing [text] is left in place so that prose escaping and bold/italic
// conversion still apply to the visible link text).
var typstMarkupPattern = regexp.MustCompile(`#image\("[^"]*"\)|#link\("[^"]*"\)`)

// typstMarkupSentinel delimits protected markup placeholders. It uses control
// characters (\x00) that cannot appear in Markdown source, so it never
// collides with user content and is never itself escaped.
const typstMarkupSentinel = "\x00MDPRESSMARKUP\x00"

// protectTypstMarkup replaces already-converted Typst markup spans with
// sentinel placeholder tokens, appending each removed span to tokens. It
// returns the rewritten text and the updated token slice.
func protectTypstMarkup(text string, tokens []string) (string, []string) {
	out := typstMarkupPattern.ReplaceAllStringFunc(text, func(m string) string {
		idx := len(tokens)
		tokens = append(tokens, m)
		return typstMarkupSentinel + strconv.Itoa(idx) + typstMarkupSentinel
	})
	return out, tokens
}

// restoreTypstMarkup replaces sentinel placeholder tokens with their original
// Typst markup spans.
func restoreTypstMarkup(text string, tokens []string) string {
	for i, tok := range tokens {
		placeholder := typstMarkupSentinel + strconv.Itoa(i) + typstMarkupSentinel
		text = strings.Replace(text, placeholder, tok, 1)
	}
	return text
}

// processOutsideCodeSpans splits text on backtick-delimited code spans
// (any number of consecutive backticks), applies fn only to non-code regions,
// and preserves code span content unchanged. Unmatched backticks are
// treated as regular text and passed through fn.
func (c *MarkdownToTypstConverter) processOutsideCodeSpans(text string, fn func(string) string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		// Count consecutive backticks starting at position i.
		backtickLen := 0
		for i+backtickLen < len(text) && text[i+backtickLen] == '`' {
			backtickLen++
		}

		if backtickLen == 0 {
			// Not a backtick — accumulate regular text until next backtick or end.
			start := i
			for i < len(text) && text[i] != '`' {
				i++
			}
			result.WriteString(fn(text[start:i]))
			continue
		}

		// We found an opening backtick sequence of length backtickLen.
		// Look for the matching closing sequence.
		delimiter := text[i : i+backtickLen]
		closeIdx := strings.Index(text[i+backtickLen:], delimiter)
		if closeIdx == -1 {
			// No matching close — treat the rest as regular text.
			result.WriteString(fn(text[i:]))
			return result.String()
		}

		// Emit the code span verbatim (including backticks).
		spanEnd := i + backtickLen + closeIdx + backtickLen
		result.WriteString(text[i:spanEnd])
		i = spanEnd
	}
	return result.String()
}

// convertImages converts ![alt](url) to #image("url")
func (c *MarkdownToTypstConverter) convertImages(text string) string {
	// Pattern: ![alt text](image.png)
	return imagePattern.ReplaceAllString(text, `#image("$2")`)
}

// convertLinks converts [text](url) to #link("url")[text]
func (c *MarkdownToTypstConverter) convertLinks(text string) string {
	// Pattern: [link text](url)
	// Avoid matching image syntax by negative lookbehind
	// Note: Go's regexp doesn't support lookahead/lookbehind, so we use replaceLinks helper.
	return c.replaceLinks(text)
}

// replaceLinks is a helper to replace [text](url) with #link("url")[text]
// This avoids the complexity of lookahead/lookbehind in Go's regexp.
func (c *MarkdownToTypstConverter) replaceLinks(text string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		// Find next [
		openBracket := strings.Index(text[i:], "[")
		if openBracket == -1 {
			result.WriteString(text[i:])
			break
		}
		// Check if this is an image (preceded by !)
		if openBracket > 0 && text[i+openBracket-1] == '!' {
			// Skip image links, already handled
			result.WriteString(text[i : i+openBracket+1])
			i += openBracket + 1
			continue
		}

		result.WriteString(text[i : i+openBracket])
		i += openBracket

		// Find matching ]
		closeBracket := strings.Index(text[i+1:], "]")
		if closeBracket == -1 {
			result.WriteString(text[i:])
			break
		}
		closeBracket++ // Adjust for the +1 offset

		// Bounds check: ensure i+closeBracket is within bounds
		if i+closeBracket >= len(text) {
			result.WriteString(text[i:])
			break
		}

		linkText := text[i+1 : i+closeBracket]

		// Check if followed by (url)
		if i+closeBracket+1 < len(text) && text[i+closeBracket+1] == '(' {
			closeParenIdx := strings.Index(text[i+closeBracket+2:], ")")
			if closeParenIdx != -1 {
				url := text[i+closeBracket+2 : i+closeBracket+2+closeParenIdx]
				result.WriteString(`#link("`)
				result.WriteString(url)
				result.WriteString(`")[`)
				result.WriteString(linkText)
				result.WriteString("]")
				i += closeBracket + 2 + closeParenIdx + 1
				continue
			}
		}

		// Not a valid link, output as-is
		result.WriteString(text[i : i+closeBracket+1])
		i += closeBracket + 1
	}
	return result.String()
}

// convertBold converts **text** or __text__ to *text*
func (c *MarkdownToTypstConverter) convertBold(text string) string {
	// Replace **text** with *text*
	text = boldAsteriskPattern.ReplaceAllString(text, `*$1*`)

	// Replace __text__ with *text*
	text = boldUnderscorePattern.ReplaceAllString(text, `*$1*`)

	return text
}

// convertItalic converts *text* or _text_ to _text_ (Typst italic).
// Must be called BEFORE convertBold so that single *text* is distinguished
// from **text** (bold). Single asterisks are converted to underscores;
// double asterisks are left for convertBold to handle.
func (c *MarkdownToTypstConverter) convertItalic(text string) string {
	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '*' {
			// Handle triple asterisks (bold+italic): ***text*** → **_text_**
			// convertBold will later convert **_text_** → *_text_*
			if i+2 < len(text) && text[i+1] == '*' && text[i+2] == '*' {
				closeIdx := strings.Index(text[i+3:], "***")
				if closeIdx > 0 {
					content := text[i+3 : i+3+closeIdx]
					result.WriteString("**_")
					result.WriteString(content)
					result.WriteString("_**")
					i += 3 + closeIdx + 3
					continue
				}
			}
			// Skip double asterisks (bold markers handled by convertBold)
			if i+1 < len(text) && text[i+1] == '*' {
				result.WriteByte('*')
				result.WriteByte('*')
				i += 2
				continue
			}
			// Single asterisk: find closing single *
			closeIdx := strings.Index(text[i+1:], "*")
			if closeIdx != -1 && closeIdx > 0 {
				// Ensure closing * is not part of **
				absClose := i + 1 + closeIdx
				if absClose+1 < len(text) && text[absClose+1] == '*' {
					// This * is part of **, skip it
					result.WriteByte(text[i])
					i++
					continue
				}
				content := text[i+1 : absClose]
				result.WriteByte('_')
				result.WriteString(content)
				result.WriteByte('_')
				i = absClose + 1
			} else {
				result.WriteByte(text[i])
				i++
			}
		} else if text[i] == '_' {
			// Handle _text_ (underscore italic)
			// Skip double underscores (bold markers handled by convertBold)
			if i+1 < len(text) && text[i+1] == '_' {
				result.WriteByte('_')
				result.WriteByte('_')
				i += 2
				continue
			}
			// Single underscore: find closing _
			closeIdx := strings.Index(text[i+1:], "_")
			if closeIdx != -1 && closeIdx > 0 {
				absClose := i + 1 + closeIdx
				// Ensure closing _ is not part of __
				if absClose+1 < len(text) && text[absClose+1] == '_' {
					result.WriteByte(text[i])
					i++
					continue
				}
				content := text[i+1 : absClose]
				result.WriteByte('_')
				result.WriteString(content)
				result.WriteByte('_')
				i = absClose + 1
			} else {
				result.WriteByte(text[i])
				i++
			}
		} else {
			result.WriteByte(text[i])
			i++
		}
	}

	return result.String()
}

// convertCodeBlock converts a code block to Typst syntax.
// In Typst, code blocks are marked with triple backticks and optional language.
//
// The fence length is chosen dynamically: a book that documents a fenced code
// block (e.g. a ```markdown example whose body contains a ```mermaid fence)
// carries a run of backticks inside the body. A fixed ``` fence would be closed
// early by that inner run, spilling the remainder into Typst markup and failing
// compilation with "unclosed delimiter". CommonMark and Typst both allow a
// longer fence, so open/close with a run at least one backtick longer than the
// longest run appearing in the content (never fewer than three).
func (c *MarkdownToTypstConverter) convertCodeBlock(content, lang string) string {
	content = strings.TrimRight(content, "\n")
	fenceLen := longestBacktickRun(content) + 1
	if fenceLen < 3 {
		fenceLen = 3
	}
	fence := strings.Repeat("`", fenceLen)
	return fence + lang + "\n" + content + "\n" + fence
}

// longestBacktickRun returns the length of the longest run of consecutive
// backtick characters in s (0 if there are none).
func longestBacktickRun(s string) int {
	longest, run := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return longest
}

// leadingBacktickRun returns the number of backtick characters at the very
// start of line (0 if the line does not begin with a backtick).
func leadingBacktickRun(line string) int {
	n := 0
	for n < len(line) && line[n] == '`' {
		n++
	}
	return n
}

// Helper functions

// countLeadingChars counts how many times a character appears at the start of a string.
func countLeadingChars(s string, ch rune) int {
	count := 0
	for _, c := range s {
		if c == ch {
			count++
		} else {
			break
		}
	}
	return count
}

// countLeadingSpaces counts leading whitespace.
func countLeadingSpaces(s string) int {
	count := 0
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if c == '\t' {
				count += 4
			} else {
				count++
			}
		} else {
			break
		}
	}
	return count
}

// isOrderedListItem checks if a line is an ordered list item (e.g., "1. Item").
// Supports both ASCII and fullwidth digits (e.g., "１. Item").
func isOrderedListItem(line string) bool {
	runes := []rune(strings.TrimSpace(line))
	// Find the first non-digit rune.
	i := 0
	for i < len(runes) && unicode.IsDigit(runes[i]) {
		i++
	}
	// Must have at least one digit followed by ". ".
	return i > 0 && i+1 < len(runes) && runes[i] == '.' && runes[i+1] == ' '
}

// extractListItemContent extracts the text from an ordered list item.
func extractListItemContent(line string) string {
	trimmed := strings.TrimSpace(line)
	idx := strings.Index(trimmed, ". ")
	if idx != -1 {
		return trimmed[idx+2:]
	}
	return trimmed
}

// isHorizontalRule checks if a line is a horizontal rule (---, ***, ___).
func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	// Check for --- or *** or ___
	chars := []rune(trimmed)
	firstChar := chars[0]
	if firstChar != '-' && firstChar != '*' && firstChar != '_' {
		return false
	}
	for _, ch := range chars {
		if ch != firstChar {
			return false
		}
	}
	return true
}
