# Markdown Parser Module

This module provides a production-grade Markdown parser for the mdpress project, a Markdown-to-PDF book converter.

## Overview

The `markdown` package wraps [goldmark](https://github.com/yuin/goldmark) with carefully selected extensions to provide:

- **GFM Extensions**: Tables, strikethrough, task lists, and autolinks
- **Code Syntax Highlighting**: Via [goldmark-highlighting](https://github.com/yuin/goldmark-highlighting)
- **Footnotes**: Full footnote support
- **Custom Heading IDs**: Automatic ID generation for cross-referencing
- **Heading Collection**: Automatic gathering of heading information for TOC generation
- **Cross-Reference Links**: Support for internal document references

## Features

### Supported Markdown Extensions

1. **Tables** (GFM)
   - Standard GitHub Flavored Markdown tables
   - Column alignment support

2. **Strikethrough** (GFM)
   - `~~text~~` syntax for strikethrough

3. **Task Lists** (GFM)
   - `- [ ] unchecked` and `- [x] checked` syntax

4. **Autolinks** (GFM)
   - `<https://example.com>` and `<user@example.com>` formats

5. **Footnotes**
   - `[^1]` references with `[^1]: content` definitions

6. **Code Highlighting**
   - Syntax highlighting for code blocks
   - Multiple theme support (github, monokai, dracula, etc.)

7. **Custom Heading IDs**
   - Automatic ID generation for headings
   - Support for manual ID specification via attributes

## Installation

This module is part of the mdpress project and requires:

```bash
go get github.com/yuin/goldmark
go get github.com/yuin/goldmark-highlighting/v2
```

## Usage

### Basic Parsing

```go
package main

import (
    "fmt"
    "github.com/yeasy/mdpress/internal/markdown"
)

func main() {
    parser := markdown.NewParser()

    source := []byte(`# Hello World
This is **bold** text.`)

    html, headings, err := parser.Parse(source)
    if err != nil {
        panic(err)
    }

    fmt.Println("HTML:", html)
    fmt.Println("Headings:", headings)
}
```

### With Custom Code Theme

```go
parser := markdown.NewParser(
    markdown.WithCodeTheme("monokai"),
)

html, headings, err := parser.Parse(source)
```

### Collecting Headings for TOC

```go
parser := markdown.NewParser()
html, headings, _ := parser.Parse(source)

// headings is []HeadingInfo with Level, Text, and ID
for _, h := range headings {
    fmt.Printf("%d. %s -> #%s\n", h.Level, h.Text, h.ID)
}
```

### Changing Code Theme After Creation

```go
parser := markdown.NewParser()
parser.SetCodeTheme("github")
html, _, _ := parser.Parse(source)
```

## API Reference

### Parser

The main parser struct that holds the goldmark instance and manages parsing state.

```go
type Parser struct {
    // ... private fields
}
```

### NewParser

Creates and returns a new Markdown parser instance.

```go
func NewParser(opts ...ParserOption) *Parser
```

**Parameters:**
- `opts`: Variable number of ParserOption functions for customization

**Returns:**
- `*Parser`: Initialized parser instance

**Example:**
```go
parser := markdown.NewParser(
    markdown.WithCodeTheme("github"),
)
```

### Parse

Parses Markdown source code and returns HTML string and heading information.

```go
func (p *Parser) Parse(source []byte) (string, []HeadingInfo, error)
```

**Parameters:**
- `source`: Markdown source code as byte slice

**Returns:**
- `string`: Generated HTML content
- `[]HeadingInfo`: Slice of collected heading information
- `error`: Any parsing errors

**Example:**
```go
html, headings, err := parser.Parse([]byte("# Title\nContent"))
if err != nil {
    log.Fatal(err)
}
```

### SetCodeTheme

Sets the code syntax highlighting theme.

```go
func (p *Parser) SetCodeTheme(theme string)
```

**Parameters:**
- `theme`: Theme name (e.g., "github", "monokai", "dracula")

**Note:** Reinitializes the parser with the new theme. Invalid theme names will fall back to the default style gracefully.

**Supported Themes:**
- `github` (default)
- `monokai`
- `dracula`
- `solarized-dark`
- `solarized-light`
- And others supported by goldmark-highlighting

### GetHeadings

Retrieves all collected heading information from the last parse.

```go
func (p *Parser) GetHeadings() []HeadingInfo
```

**Returns:**
- `[]HeadingInfo`: Thread-safe copy of collected headings

### HeadingInfo

Structure containing information about a heading.

```go
type HeadingInfo struct {
    Level int    // Heading level (1-6)
    Text  string // Heading text content
    ID    string // Custom heading ID for cross-referencing
}
```

### ParserOption

Functional option type for customizing parser behavior.

```go
type ParserOption func(*Parser)
```

**Built-in Options:**
- `WithCodeTheme(theme string)`: Set code highlighting theme
- `WithExtensions(exts ...goldmark.Extender)`: Add custom extensions
- `WithParserOptions(opts ...parser.Option)`: Set goldmark parser options

## Examples

### Complete Document Parsing

```go
package main

import (
    "fmt"
    "github.com/yeasy/mdpress/internal/markdown"
)

func main() {
    parser := markdown.NewParser(
        markdown.WithCodeTheme("github"),
    )

    md := []byte(`# Go Tutorial

## Chapter 1

Here's a code example:

\`\`\`go
func main() {
    fmt.Println("Hello")
}
\`\`\`

| Feature | Support |
|---------|---------|
| Tables  | Yes     |
| Code    | Yes     |

- [x] Implemented
- [ ] TODO
`)

    html, headings, err := parser.Parse(md)
    if err != nil {
        panic(err)
    }

    // Print TOC
    for _, h := range headings {
        fmt.Printf("%s# %s\n",
            repeatString("  ", h.Level-1), h.Text)
    }
}

func repeatString(s string, count int) string {
    result := ""
    for i := 0; i < count; i++ {
        result += s
    }
    return result
}
```

### Multi-Document Processing

```go
parser := markdown.NewParser()

documents := [][]byte{
    []byte("# Document 1\nContent..."),
    []byte("# Document 2\nContent..."),
}

for _, doc := range documents {
    html, headings, _ := parser.Parse(doc)
    // Process html and headings
}
```

## Implementation Details

### Heading ID Generation

IDs are automatically generated from heading text by:
1. Converting to lowercase
2. Removing special characters
3. Replacing spaces with hyphens
4. Trimming leading/trailing hyphens

Example: "Hello World" → "hello-world"

### Heading Collection

The parser uses a custom AST transformer to collect heading information during parsing. This is done in a thread-safe manner using mutexes.

### Extensions Architecture

The module uses goldmark's extensibility properly:
- Built-in GFM extensions for standard features
- Custom transformers for heading ID generation
- Cross-reference link resolver for internal references

## Thread Safety

- The `Parser` instance is thread-safe for concurrent parsing operations
- Heading collection uses internal synchronization
- Recommended: Create one Parser instance and reuse it

## Performance

Benchmarks on typical documents:
- Simple documents: ~1-2ms
- Complex documents with multiple features: ~5-10ms

For optimal performance:
- Reuse Parser instances
- Parse in parallel for multiple documents
- Consider caching results for frequently parsed content

## Error Handling

The Parse method returns detailed errors:
- Syntax errors in the markdown (rare with goldmark)
- Rendering errors (usually configuration issues)

```go
if html, headings, err := parser.Parse(source); err != nil {
    fmt.Printf("Parse error: %v\n", err)
    // Handle error
}
```

## Testing

The module includes comprehensive tests:
- Unit tests for basic functionality
- Integration tests for extension support
- Benchmark tests for performance measurement

Run tests with:
```bash
go test ./internal/markdown -v
```

## Future Enhancements

Potential improvements for future versions:
- Custom renderer for PDF-specific formatting
- Math equation support (KaTeX/MathJax)
- Diagram support (Mermaid)
- Custom CSS class insertion for styling
- Metadata/front-matter extraction

## Troubleshooting

### Issue: Code blocks not highlighted
- **Solution**: Ensure language tag is specified in code fence (e.g., ` ```go`)

### Issue: Table not rendering
- **Solution**: Ensure table format follows GFM specification with proper separators

### Issue: Heading IDs have conflicts
- **Solution**: The parser automatically appends numbers to duplicate IDs (e.g., "hello-world-1")

## Contributing

When extending this module:
1. Maintain backward compatibility
2. Add tests for new features
3. Follow the existing code style
4. Update this documentation
5. Ensure thread safety for concurrent operations

## License

This module is part of the mdpress project.
