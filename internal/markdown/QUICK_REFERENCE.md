# Quick Reference Guide

Fast lookup for common tasks with the markdown parser module.

## Installation

```bash
go get github.com/yuin/goldmark
go get github.com/yuin/goldmark-highlighting/v2
```

## Common Tasks

### Parse Markdown to HTML

```go
import "github.com/yeasy/mdpress/internal/markdown"

parser := markdown.NewParser()
html, headings, err := parser.Parse([]byte("# Title\nContent"))
```

### Get Headings for Table of Contents

```go
_, headings, _ := parser.Parse(source)

for _, h := range headings {
    fmt.Printf("Level %d: %s (id: %s)\n", h.Level, h.Text, h.ID)
}
```

### Change Code Highlighting Theme

```go
parser.SetCodeTheme("monokai")
// or during creation:
parser := markdown.NewParser(markdown.WithCodeTheme("monokai"))
```

### Process Multiple Files

```go
processor := markdown.NewDocumentProcessor(4) // 4 concurrent workers
results := processor.ProcessFiles([]string{"file1.md", "file2.md"})

for _, result := range results {
    if result.Error != nil {
        log.Printf("Error: %v\n", result.Error)
        continue
    }
    fmt.Println(result.HTML)
}
```

### Generate Table of Contents

```go
// HTML format
tocGen := markdown.NewTableOfContentsGenerator(3, true) // max 3 levels
html, headings, _ := parser.Parse(source)
toc := tocGen.Generate(headings)

// Markdown format
tocGen := markdown.NewTableOfContentsGenerator(3, false)
toc := tocGen.Generate(headings)
```

### Combine Multiple Markdown Files

```go
builder := markdown.NewDocumentBuilder()
builder.AddFile("intro.md")
builder.AddFile("chapter1.md")
builder.AddFile("chapter2.md")

html, headings, err := builder.BuildWithTOC()
```

### Export to Complete HTML Document

```go
opts := markdown.ExportOptions{
    DocumentTitle: "My Book",
    Author: "John Doe",
    IncludeCSS: true,
}
complete := markdown.ExportHTML(html, opts)
```

## API Reference

### Parser

```go
// Create parser
parser := markdown.NewParser()
parser := markdown.NewParser(markdown.WithCodeTheme("github"))

// Parse markdown
html, headings, err := parser.Parse(source)

// Change theme
parser.SetCodeTheme("monokai")

// Get headings
headings := parser.GetHeadings()
```

### HeadingInfo

```go
type HeadingInfo struct {
    Level int    // 1-6
    Text  string // Heading text
    ID    string // Generated ID for linking
}
```

### DocumentProcessor

```go
processor := markdown.NewDocumentProcessor(concurrency)
result := processor.ProcessFile(path)
results := processor.ProcessFiles(paths)
processor.ClearCache()
```

### TableOfContentsGenerator

```go
gen := markdown.NewTableOfContentsGenerator(maxLevel, htmlFormat)
toc := gen.Generate(headings)
```

### DocumentBuilder

```go
builder := markdown.NewDocumentBuilder()
builder.AddFile(path)
html, headings, err := builder.Build()
html, headings, err := builder.BuildWithTOC()
```

### ExportHTML

```go
html := markdown.ExportHTML(content, markdown.ExportOptions{
    DocumentTitle: "Title",
    Author: "Author",
    IncludeCSS: true,
    CustomCSS: "/* CSS */",
})
```

## Supported Markdown Features

| Feature | Syntax | Status |
|---------|--------|--------|
| Headings | `# Title` | ✅ |
| Bold | `**text**` or `__text__` | ✅ |
| Italic | `*text*` or `_text_` | ✅ |
| Code | `` `code` `` | ✅ |
| Code blocks | ` ```language ... ``` ` | ✅ |
| Links | `[text](url)` | ✅ |
| Auto links | `<url>` | ✅ |
| Images | `![alt](url)` | ✅ |
| Lists | `- item` or `1. item` | ✅ |
| Blockquotes | `> quote` | ✅ |
| Tables | GFM tables | ✅ |
| Strikethrough | `~~text~~` | ✅ |
| Task lists | `- [x] done` | ✅ |
| Footnotes | `[^1]` | ✅ |

## Code Highlighting Themes

Common themes available:
- `github` (default)
- `monokai`
- `dracula`
- `solarized-dark`
- `solarized-light`
- `nord`
- `vim`
- `native`

## Options

### Parser Options

```go
// Set code theme
markdown.WithCodeTheme("monokai")

// Add custom extensions
markdown.WithExtensions(ext1, ext2)

// Set parser options
markdown.WithParserOptions(parserOpt)
```

## Type Signatures

```go
// Parser creation
func NewParser(opts ...ParserOption) *Parser

// Parsing
func (p *Parser) Parse(source []byte) (string, []HeadingInfo, error)

// Utilities
func (p *Parser) SetCodeTheme(theme string)
func (p *Parser) GetHeadings() []HeadingInfo

// Document processing
func NewDocumentProcessor(maxConcurrency int) *DocumentProcessor
func NewTableOfContentsGenerator(maxLevel int, htmlFormat bool) *TableOfContentsGenerator
func NewDocumentBuilder() *DocumentBuilder
func ExportHTML(html string, opts ExportOptions) string
```

## Common Patterns

### Single File Processing

```go
parser := markdown.NewParser()
html, headings, _ := parser.Parse(readFile("doc.md"))
```

### Batch Processing with Caching

```go
processor := markdown.NewDocumentProcessor(4)
results := processor.ProcessFiles(files)
// Reprocess uses cache automatically
```

### Building a Book

```go
builder := markdown.NewDocumentBuilder()
for _, chapter := range chapters {
    builder.AddFile(chapter)
}
html, _, _ := builder.BuildWithTOC()
markdown.ExportHTML(html, opts)
```

### Custom Styling

```go
opts := markdown.ExportOptions{
    IncludeCSS: true,
    CustomCSS: `
        body { font-family: Georgia; }
        h1 { color: navy; }
    `,
}
markdown.ExportHTML(html, opts)
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Code not highlighted | Add language to fence: ` ```go ` |
| Missing IDs | IDs auto-generated, check heading level |
| Slow processing | Increase concurrency in DocumentProcessor |
| Cache issues | Call `processor.ClearCache()` |
| Duplicate IDs | Use DocumentBuilder for multi-file |

## Examples

### Minimal Example
```go
parser := markdown.NewParser()
html, _, _ := parser.Parse([]byte("# Hello\nWorld"))
```

### Complete Example
```go
builder := markdown.NewDocumentBuilder()
builder.AddFile("ch1.md")
builder.AddFile("ch2.md")

html, headings, _ := builder.BuildWithTOC()

output := markdown.ExportHTML(html, markdown.ExportOptions{
    DocumentTitle: "Book",
    IncludeCSS: true,
})

ioutil.WriteFile("book.html", []byte(output), 0644)
```

## Performance Tips

1. **Reuse parsers**: Create once, use many times
2. **Enable caching**: Use DocumentProcessor
3. **Tune concurrency**: 4-8 workers for most cases
4. **Monitor memory**: Clear cache if processing many files
5. **Batch operations**: Process files in batches

## Error Handling

```go
html, headings, err := parser.Parse(source)
if err != nil {
    log.Fatalf("Parse error: %v", err)
}

result := processor.ProcessFile(path)
if result.Error != nil {
    log.Printf("File error: %v", result.Error)
}
```

## Thread Safety

- Parser is thread-safe for concurrent Parse calls
- DocumentProcessor handles concurrency safely
- No race conditions in normal usage
- Reuse instances for better performance

## Files to Know

- `parser.go` - Core parser implementation
- `extensions.go` - Custom extensions
- `integration.go` - High-level utilities
- `README.md` - Full documentation
- `INTEGRATION.md` - Advanced usage guide

## Resources

- **README.md** - Complete documentation
- **example_test.go** - Usage examples
- **parser_test.go** - Test cases with patterns
- **INTEGRATION.md** - Advanced patterns
- **goldmark docs** - https://github.com/yuin/goldmark

---

*Last Updated: 2026-03-18*
