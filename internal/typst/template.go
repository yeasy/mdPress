package typst

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
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
}

// TypstTemplate defines the base Typst document template.
const typstTemplateStr = `#set page(
  paper: "{{ .PageWidth }}-x-{{ .PageHeight }}",
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
  font: ({{ .FontFamily }}),
  size: {{ .FontSize }},
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
// It also strips Go template delimiters to prevent text/template injection.
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
	"{{", "",
	"}}", "",
)

func sanitizeTypstText(s string) string {
	return typstTextReplacer.Replace(s)
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
	// FontFamily needs Typst text escaping, not dimension validation.
	data.FontFamily = sanitizeTypstText(data.FontFamily)

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
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Typst file: %w", err)
	}
	return nil
}

// getPageDimensions returns page width and height in mm as Typst format strings.
func getPageDimensions(pageSize string) (width, height string) {
	switch strings.ToUpper(pageSize) {
	case "A4":
		return "210mm", "297mm"
	case "A5":
		return "148mm", "210mm"
	case "B5":
		return "176mm", "250mm"
	case "LETTER":
		return "216mm", "279mm"
	case "LEGAL":
		return "216mm", "356mm"
	default:
		// Default to A4
		return "210mm", "297mm"
	}
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

// currentDate returns the current date as a formatted string.
func currentDate() string {
	return time.Now().Format("2006-01-02")
}

// prepareTypstContent prepares the Typst content by escaping special characters if needed.
// This is a placeholder for future enhancements (e.g., escaping Typst-specific syntax).
func prepareTypstContent(content string) string {
	// For now, return as-is. In the future, we might need to escape certain
	// Typst syntax characters.
	return content
}

// makeTypstFont converts a CSS font family string to a Typst font list.
// The input is sanitized to prevent Typst code injection.
func makeTypstFont(cssFontFamily string) string {
	if cssFontFamily == "" {
		return `"Segoe UI", "Helvetica", sans-serif`
	}
	return sanitizeTypstValue(cssFontFamily)
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
	if err := os.MkdirAll(typstDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create Typst directory: %w", err)
	}
	return typstDir, nil
}
