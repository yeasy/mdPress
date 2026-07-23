// frontmatter.go strips the leading bytes that are not part of a document's
// prose: a UTF-8 byte order mark, and a YAML front matter block.
//
// Neither was handled, and both are common in Markdown that came from another
// tool. A BOM stopped the first "# Heading" from being recognized as a heading
// at all — the line rendered as literal text and the chapter lost its title.
// Front matter was rendered verbatim, so "title: …" and "description: …"
// appeared in the page body, the search index and the PDF.
package markdown

import (
	"bytes"
	"strings"
)

// utf8BOM is the byte order mark some editors and exporters prepend.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// FrontMatter holds the fields mdpress understands from a front matter block.
// Unrecognized keys are ignored rather than rejected: front matter is a shared
// convention, and a file carrying fields for another tool must still build.
type FrontMatter struct {
	Title       string
	Description string
}

// StripLeadingMetadata removes a UTF-8 BOM and a leading YAML front matter
// block, returning the remaining source and whatever metadata was recognized.
//
// Only a block that starts on the document's very first line counts, per the
// usual convention — a "---" later in the file is a thematic break.
func StripLeadingMetadata(source []byte) ([]byte, FrontMatter) {
	source = bytes.TrimPrefix(source, utf8BOM)

	body, block, found := splitFrontMatter(source)
	if !found {
		return source, FrontMatter{}
	}
	return body, parseFrontMatter(block)
}

// splitFrontMatter separates a leading "---" delimited block from the body.
func splitFrontMatter(source []byte) (body, block []byte, found bool) {
	const delimiter = "---"

	// The opening delimiter must be the first line, alone on that line.
	rest := source
	line, remainder, hasMore := bytes.Cut(rest, []byte("\n"))
	if !hasMore || strings.TrimRight(string(line), "\r") != delimiter {
		return source, nil, false
	}

	// Scan for the closing delimiter.
	var collected []byte
	for {
		line, remainder, hasMore = bytes.Cut(remainder, []byte("\n"))
		trimmed := strings.TrimRight(string(line), "\r")
		if trimmed == delimiter || trimmed == "..." {
			return remainder, collected, true
		}
		if !hasMore {
			// Unterminated: treat the whole thing as content rather than
			// swallowing the document.
			return source, nil, false
		}
		collected = append(collected, line...)
		collected = append(collected, '\n')
	}
}

// parseFrontMatter reads the handful of scalar keys mdpress uses. It is
// deliberately not a general YAML parse: the block belongs to the wider
// ecosystem, and failing on a key meant for another tool would be wrong.
func parseFrontMatter(block []byte) FrontMatter {
	var fm FrontMatter
	for _, raw := range strings.Split(string(block), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "title":
			fm.Title = value
		case "description":
			fm.Description = value
		}
	}
	return fm
}
