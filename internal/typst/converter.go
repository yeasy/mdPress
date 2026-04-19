// Package typst provides a Markdown-to-Typst converter and Typst PDF generator.
package typst

import (
	"log/slog"
	"regexp"
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
	var codeBlockContent strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Handle code blocks: triple backticks
		if rest, ok := strings.CutPrefix(line, "```"); ok {
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				codeBlockLang = strings.TrimSpace(rest)
				codeBlockContent.Reset()
			} else {
				// End of code block
				inCodeBlock = false
				typstCode := c.convertCodeBlock(codeBlockContent.String(), codeBlockLang)
				result.WriteString(typstCode)
				result.WriteString("\n\n")
			}
			continue
		}

		// If we're inside a code block, accumulate the content
		if inCodeBlock {
			codeBlockContent.WriteString(line)
			codeBlockContent.WriteString("\n")
			continue
		}

		// Handle headings (require space after # per Markdown spec)
		if strings.HasPrefix(line, "#") {
			level := countLeadingChars(line, '#')
			if level <= 6 && len(line) > level && line[level] == ' ' {
				heading := strings.TrimSpace(line[level+1:])
				if heading != "" {
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
			result.WriteString("> ")
			result.WriteString(c.convertInline(content))
			result.WriteString("\n")
			continue
		}

		// Handle horizontal rules
		if isHorizontalRule(line) {
			result.WriteString("---\n\n")
			continue
		}

		// Handle empty lines
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
			continue
		}

		// Handle paragraphs
		result.WriteString(c.convertInline(line))
		result.WriteString("\n\n")
	}

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
		segment = c.convertImages(segment)
		segment = c.convertLinks(segment)
		segment = c.convertItalic(segment)
		segment = c.convertBold(segment)
		return segment
	})
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
func (c *MarkdownToTypstConverter) convertCodeBlock(content, lang string) string {
	content = strings.TrimRight(content, "\n")
	if lang == "" {
		return "```\n" + content + "\n```"
	}
	return "```" + lang + "\n" + content + "\n```"
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
