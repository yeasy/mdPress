package typst

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// TypstTemplateData holds the data needed to render a Typst document.
type TypstTemplateData struct {
	Title       string
	Subtitle    string
	Author      string
	Date        string
	Version     string
	Language    string
	Content     string // The Typst-formatted body content
	PageWidth   string // e.g., "210mm" for A4
	PageHeight  string // e.g., "297mm"
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
{{ .Content }}
`

// RenderTypstDocument renders the template with the provided data.
func RenderTypstDocument(data TypstTemplateData) (string, error) {
	tmpl, err := template.New("typst").Parse(typstTemplateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse Typst template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render Typst template: %w", err)
	}

	return buf.String(), nil
}

// WriteTypstFile writes Typst content to a file.
func WriteTypstFile(filePath string, content string) error {
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write Typst file: %w", err)
	}
	return nil
}

// GetPageDimensions returns page width and height in mm as Typst format strings.
func GetPageDimensions(pageSize string) (width, height string) {
	switch pageSize {
	case "A4":
		return "210mm", "297mm"
	case "A5":
		return "148mm", "210mm"
	case "Letter":
		return "216mm", "279mm"
	case "Legal":
		return "216mm", "356mm"
	default:
		// Default to A4
		return "210mm", "297mm"
	}
}

// ConvertMarginToTypst converts a margin value (e.g., "20mm", "1in") to a Typst string.
// If the value is empty, returns a default value.
func ConvertMarginToTypst(margin string, defaultVal string) string {
	if margin == "" {
		return defaultVal
	}
	// Validate that it ends with a unit (mm, cm, in, pt, etc.)
	// For now, assume it's already in a valid Typst format
	return margin
}

// CurrentDate returns the current date as a formatted string.
func CurrentDate() string {
	return time.Now().Format("2006-01-02")
}

// PrepareTypstContent prepares the Typst content by escaping special characters if needed.
// This is a placeholder for future enhancements (e.g., escaping Typst-specific syntax).
func PrepareTypstContent(content string) string {
	// For now, return as-is. In the future, we might need to escape certain
	// Typst syntax characters.
	return content
}

// MakeTypstFont converts a CSS font family string to a Typst font list.
func MakeTypstFont(cssFontFamily string) string {
	if cssFontFamily == "" {
		return `"Segoe UI", "Helvetica", sans-serif`
	}
	// For now, return the CSS font family as-is.
	// A more sophisticated implementation might convert common web fonts
	// to their Typst equivalents.
	return cssFontFamily
}

// MakeTypstFontSize converts a CSS font size (e.g., "12pt", "14px") to Typst format.
func MakeTypstFontSize(cssFontSize string) string {
	if cssFontSize == "" {
		return "12pt"
	}
	// Simple conversion: assume it's already in a compatible format (pt)
	// If it's in px, convert: 1px ≈ 0.75pt
	if len(cssFontSize) > 2 && cssFontSize[len(cssFontSize)-2:] == "px" {
		// Parse and convert
		sizeStr := cssFontSize[:len(cssFontSize)-2]
		var sizeVal float64
		_, _ = fmt.Sscanf(sizeStr, "%f", &sizeVal)
		return fmt.Sprintf("%.1fpt", sizeVal*0.75)
	}
	return cssFontSize
}

// CreateTypstDir ensures the Typst working directory exists.
func CreateTypstDir(baseDir string) (string, error) {
	typstDir := filepath.Join(baseDir, ".typst")
	if err := os.MkdirAll(typstDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create Typst directory: %w", err)
	}
	return typstDir, nil
}
