package typst

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TypstTemplateData holds the data needed to render a Typst document.
type TypstTemplateData struct {
	Title        string
	Subtitle     string
	Author       string
	Date         string
	Version      string
	Language     string
	Content      string // The Typst-formatted body content
	PageWidth    string // e.g., "210mm" for A4
	PageHeight   string // e.g., "297mm"
	MarginTop    string
	MarginRight  string
	MarginBottom string
	MarginLeft   string
	FontFamily   string
	FontSize     string
	LineHeight   float64
	// Description becomes the PDF /Subject entry.
	Description string
	// BuildTime is the timestamp recorded as the PDF creation date. The zero
	// value leaves Typst to stamp the wall clock.
	BuildTime time.Time
	// FontLine is the fully-rendered Typst "font: (...)" line (including a
	// trailing newline) or empty to omit it. It is computed from FontFamily
	// by renderTypstDocument so the template never interpolates a raw CSS
	// font stack (which is invalid Typst).
	FontLine string
	// DocumentLine is the fully-rendered Typst "#set document(...)" call
	// (including a trailing newline), or empty when there is no metadata to
	// record. Typst maps it onto the PDF document information dictionary.
	DocumentLine string
}

// TypstTemplate defines the base Typst document template.
const typstTemplateStr = `{{ .DocumentLine }}#set page(
  width: {{ .PageWidth }},
  height: {{ .PageHeight }},
  margin: (
    top: {{ .MarginTop }},
    bottom: {{ .MarginBottom }},
    left: {{ .MarginLeft }},
    right: {{ .MarginRight }},
  ),
  header: [],
  footer: [],
)

#set text(
{{ .FontLine }}  size: {{ .FontSize }},
  lang: "{{ .Language }}",
)

#set par(leading: {{ .LineHeight }}em)

// Heading styles
#set heading(numbering: "1.1.1")

// Code block styling
#show raw.where(block: true): block.with(
  fill: rgb("#f5f5f5"),
  inset: 8pt,
  radius: 4pt,
)

#show raw.where(block: false): box.with(
  fill: rgb("#f5f5f5"),
  inset: 2pt,
  radius: 2pt,
)

// Title and metadata
#align(center)[
  = {{ .Title }}

  {{ if .Subtitle }}#emph[{{ .Subtitle }}]{{ end }}

  {{ if .Author }}_by {{ .Author }}_{{ end }}

  {{ if .Version }}Version {{ .Version }}{{ end }}

  {{ if .Date }}_{{ .Date }}_{{ end }}
]

#pagebreak()

// Table of contents
#outline(
  title: "Contents",
  depth: 3,
)

#pagebreak()

// Body content
__MDPRESS_CONTENT_PLACEHOLDER__
`

// typstTextReplacer escapes Typst control characters in user-supplied metadata
// to prevent code injection via fields like Title or Author.
// It also strips Go template delimiters to prevent text/template injection,
// and replaces newlines with spaces to prevent breaking heading syntax.
var typstTextReplacer = strings.NewReplacer(
	"\\", "\\\\",
	"#", "\\#",
	"$", "\\$",
	"=", "\\=",
	"@", "\\@",
	"<", "\\<",
	">", "\\>",
	"_", "\\_",
	"*", "\\*",
	"\n", " ",
	"\r", " ",
	"{{", "",
	"}}", "",
)

func sanitizeTypstText(s string) string {
	return typstTextReplacer.Replace(s)
}

// typstStringLiteralReplacer escapes a value for use inside a Typst double-quoted
// string. Markup escaping (sanitizeTypstText) must not be used there: inside a
// string literal a "\#" would show up verbatim in the PDF metadata.
var typstStringLiteralReplacer = strings.NewReplacer(
	"\\", "\\\\",
	`"`, `\"`,
	"\n", " ",
	"\r", " ",
	"{{", "",
	"}}", "",
)

// typstStringLiteral quotes s as a Typst string literal.
func typstStringLiteral(s string) string {
	return `"` + typstStringLiteralReplacer.Replace(s) + `"`
}

// buildDocumentLine renders the "#set document(...)" call that gives the PDF its
// document information. Typst otherwise writes an empty title and an author-less
// dictionary, so a generated book shows up untitled in library software.
//
// The build timestamp is passed explicitly rather than left to Typst's wall
// clock so that rebuilding the same sources yields an identical PDF.
func buildDocumentLine(data TypstTemplateData) string {
	var fields []string
	if data.Title != "" {
		fields = append(fields, "title: "+typstStringLiteral(data.Title))
	}
	if data.Author != "" {
		fields = append(fields, "author: ("+typstStringLiteral(data.Author)+",)")
	}
	if data.Description != "" {
		fields = append(fields, "description: "+typstStringLiteral(data.Description))
	}
	if !data.BuildTime.IsZero() {
		t := data.BuildTime.UTC()
		fields = append(fields, fmt.Sprintf(
			"date: datetime(year: %d, month: %d, day: %d, hour: %d, minute: %d, second: %d)",
			t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second()))
	}
	if len(fields) == 0 {
		return ""
	}
	return "#set document(" + strings.Join(fields, ", ") + ")\n"
}

// sanitizeTemplateValue strips Go template delimiters from dimension/font values
// that are interpolated into the text/template but don't need Typst escaping.
func sanitizeTemplateValue(s string) string {
	s = strings.ReplaceAll(s, "{{", "")
	s = strings.ReplaceAll(s, "}}", "")
	return s
}

// dimensionPattern matches valid CSS/Typst dimension values like "10mm", "2.5cm", "1in".
var dimensionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?\s*(mm|cm|in|pt|em)$`)

// sanitizeDimension validates that a string looks like a dimension value.
// Returns a safe default if the input is not a valid dimension.
func sanitizeDimension(s, fallback string) string {
	s = sanitizeTemplateValue(strings.TrimSpace(s))
	if dimensionPattern.MatchString(s) {
		return s
	}
	return fallback
}

// contentPlaceholder is used instead of a Go template action for Content
// to prevent user-supplied Typst content containing "{{ }}" from being
// interpreted as template directives (which could panic or inject code).
const contentPlaceholder = "__MDPRESS_CONTENT_PLACEHOLDER__"

// renderTypstDocument renders the template with the provided data.
func renderTypstDocument(data TypstTemplateData) (string, error) {
	// Built from the raw values: the markup escaping below is for body text
	// and would leak backslashes into the PDF metadata.
	data.DocumentLine = buildDocumentLine(data)

	// Sanitize user-supplied metadata to prevent Typst injection.
	data.Title = sanitizeTypstText(data.Title)
	data.Subtitle = sanitizeTypstText(data.Subtitle)
	data.Author = sanitizeTypstText(data.Author)
	data.Version = sanitizeTypstText(data.Version)
	data.Date = sanitizeTypstText(data.Date)
	data.Language = sanitizeTypstText(data.Language)

	// Sanitize dimension fields against Typst code injection.
	data.PageWidth = sanitizeDimension(data.PageWidth, "210mm")
	data.PageHeight = sanitizeDimension(data.PageHeight, "297mm")
	data.MarginTop = sanitizeDimension(data.MarginTop, "25mm")
	data.MarginRight = sanitizeDimension(data.MarginRight, "25mm")
	data.MarginBottom = sanitizeDimension(data.MarginBottom, "25mm")
	data.MarginLeft = sanitizeDimension(data.MarginLeft, "25mm")
	data.FontSize = sanitizeDimension(data.FontSize, "11pt")
	// Convert the CSS font stack in FontFamily into a valid Typst font array
	// line. makeTypstFont drops generic families/keywords and quotes each
	// concrete name; an empty stack falls back to a sane default, so the
	// resulting line is always valid Typst.
	data.FontLine = "  font: (" + makeTypstFont(data.FontFamily) + "),\n"

	// Defense-in-depth: clamp LineHeight at point of use.
	if data.LineHeight < 0.5 || data.LineHeight > 5.0 {
		data.LineHeight = 1.6
	}

	// Save content before template execution — it is injected via string
	// replacement afterwards to avoid text/template interpreting any
	// "{{ }}" sequences in the Markdown-converted Typst body.
	content := data.Content
	data.Content = "" // not used by template

	tmpl, err := template.New("typst").Parse(typstTemplateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse Typst template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render Typst template: %w", err)
	}

	// Inject the body content safely outside the template engine.
	result := strings.Replace(buf.String(), contentPlaceholder, content, 1)
	return result, nil
}

// writeTypstFile writes Typst content to a file.
func writeTypstFile(filePath string, content string) error {
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write Typst file: %w", err)
	}
	return nil
}

// getPageDimensions returns page width and height in mm as Typst format strings.
func getPageDimensions(pageSize string) (width, height string) {
	d := utils.GetPageDimensions(pageSize)
	return d.WidthMM(), d.HeightMM()
}

// ConvertMarginToTypst converts a margin value (e.g., "20mm", "1in") to a Typst string.
// If the value is empty, returns a default value.
// The value is sanitized to prevent Typst code injection.
func ConvertMarginToTypst(margin string, defaultVal string) string {
	if margin == "" {
		return defaultVal
	}
	return sanitizeTypstValue(margin)
}

// sanitizeTypstValue removes characters that could be used for Typst code injection.
// Only allows alphanumeric characters, digits, dots, and common unit suffixes.
func sanitizeTypstValue(val string) string {
	var result strings.Builder
	for _, ch := range val {
		// Allow: letters, digits, dot, minus, space, comma, quotes (for font names)
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '-' ||
			ch == ' ' || ch == ',' || ch == '"' {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// buildTime returns the timestamp to stamp into generated documents.
//
// SOURCE_DATE_EPOCH (https://reproducible-builds.org/specs/source-date-epoch/)
// is honored so that rebuilding the same sources produces an identical PDF.
func buildTime() time.Time {
	if raw := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); raw != "" {
		if secs, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.Unix(secs, 0).UTC()
		}
	}
	return time.Now().UTC()
}

// currentDate returns the build date as a formatted string.
func currentDate() string {
	return buildTime().Format("2006-01-02")
}

// prepareTypstContent prepares the Typst content by escaping special characters if needed.
// This is a placeholder for future enhancements (e.g., escaping Typst-specific syntax).
func prepareTypstContent(content string) string {
	// For now, return as-is. In the future, we might need to escape certain
	// Typst syntax characters.
	return content
}

// typstGenericFontKeywords are CSS generic families / system keywords that are
// meaningless (and invalid) as Typst font names and must be dropped when
// converting a CSS font stack to a Typst font array.
var typstGenericFontKeywords = map[string]bool{
	"sans-serif":         true,
	"serif":              true,
	"monospace":          true,
	"cursive":            true,
	"fantasy":            true,
	"system-ui":          true,
	"ui-sans-serif":      true,
	"ui-serif":           true,
	"ui-monospace":       true,
	"ui-rounded":         true,
	"-apple-system":      true,
	"blinkmacsystemfont": true,
	"inherit":            true,
	"initial":            true,
	"unset":              true,
}

// typstDefaultFonts is the fallback Typst font array used when a CSS font stack
// yields no concrete font names (e.g. it consisted only of generic keywords).
const typstDefaultFonts = `"Segoe UI", "Helvetica Neue", "Arial"`

// makeTypstFont converts a CSS font-family stack (e.g.
// "-apple-system, BlinkMacSystemFont, 'PingFang SC', ..., sans-serif") into the
// contents of a Typst font array: a comma-separated list of double-quoted font
// names with CSS generic families and system keywords removed. The returned
// string does NOT include the surrounding parentheses; callers wrap it as
// needed. If no concrete font names remain, a sane default is returned.
//
// CSS single-quoted names are unquoted and re-emitted with Typst double quotes;
// bare identifiers are treated as font names. Each name is sanitized to prevent
// Typst code injection (double quotes and other control characters stripped).
func makeTypstFont(cssFontFamily string) string {
	names := parseTypstFontNames(cssFontFamily)
	if len(names) == 0 {
		return typstDefaultFonts
	}
	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, `"`+n+`"`)
	}
	return strings.Join(quoted, ", ")
}

// parseTypstFontNames splits a CSS font stack on commas, trims whitespace,
// strips surrounding single/double quotes, drops empty entries and generic CSS
// families/keywords, and sanitizes each remaining name for Typst.
func parseTypstFontNames(cssFontFamily string) []string {
	var names []string
	for _, part := range strings.Split(cssFontFamily, ",") {
		name := strings.TrimSpace(part)
		// Strip surrounding quotes (single or double).
		name = strings.Trim(name, `"'`)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if typstGenericFontKeywords[strings.ToLower(name)] {
			continue
		}
		// Sanitize to remove any residual quotes/control chars that could
		// break out of the Typst string literal.
		name = sanitizeTypstFontName(name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

// sanitizeTypstFontName removes characters that could break out of a Typst
// double-quoted string literal or inject code. It permits letters (including
// non-ASCII, for CJK font names), digits, spaces, hyphens, and dots.
func sanitizeTypstFontName(val string) string {
	var b strings.Builder
	for _, ch := range val {
		switch {
		case ch == '"' || ch == '\\' || ch == '\n' || ch == '\r':
			// drop quote/backslash/newlines that would break the literal
		case ch == '-' || ch == '.' || ch == ' ' || ch == '_':
			b.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z':
			b.WriteRune(ch)
		case ch > 127:
			// Allow non-ASCII (e.g. CJK) font names.
			b.WriteRune(ch)
		}
	}
	return strings.TrimSpace(b.String())
}

// makeTypstFontSize converts a CSS font size (e.g., "12pt", "14px") to Typst format.
// The output is sanitized to prevent Typst code injection.
func makeTypstFontSize(cssFontSize string) string {
	if cssFontSize == "" {
		return "12pt"
	}
	// Simple conversion: assume it's already in a compatible format (pt)
	// If it's in px, convert: 1px ≈ 0.75pt
	if len(cssFontSize) > 2 && cssFontSize[len(cssFontSize)-2:] == "px" {
		// Parse and convert
		sizeStr := cssFontSize[:len(cssFontSize)-2]
		var sizeVal float64
		if n, _ := fmt.Sscanf(sizeStr, "%f", &sizeVal); n == 1 && sizeVal > 0 {
			return fmt.Sprintf("%.1fpt", sizeVal*0.75)
		}
		return "12pt" // fallback for unparseable px values
	}
	return sanitizeTypstValue(cssFontSize)
}

// createTypstDir ensures the Typst working directory exists.
func createTypstDir(baseDir string) (string, error) {
	typstDir := filepath.Join(baseDir, ".typst")
	if err := os.MkdirAll(typstDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create Typst directory: %w", err)
	}
	return typstDir, nil
}
