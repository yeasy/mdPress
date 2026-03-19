# Integration Guide

This document provides detailed guidance on using the advanced integration features of the markdown parser module.

## Overview

The integration module provides higher-level abstractions for common document processing tasks:

- **DocumentProcessor**: Batch processing of multiple files with caching
- **TableOfContentsGenerator**: Automatic TOC generation in HTML or Markdown format
- **DocumentBuilder**: Combine multiple Markdown files into a single document
- **ExportOptions**: Export to complete HTML documents with styling

## DocumentProcessor

Efficiently process multiple Markdown files with optional caching and concurrent processing.

### Basic Usage

```go
processor := markdown.NewDocumentProcessor(4) // 4 concurrent workers

result := processor.ProcessFile("chapter1.md")
if result.Error != nil {
    log.Fatal(result.Error)
}

fmt.Println("HTML:", result.HTML)
fmt.Println("Headings:", result.Headings)
```

### Batch Processing

```go
processor := markdown.NewDocumentProcessor(4)

files := []string{
    "chapter1.md",
    "chapter2.md",
    "chapter3.md",
}

results := processor.ProcessFiles(files)

for _, result := range results {
    if result.Error != nil {
        fmt.Printf("Error processing %s: %v\n", result.FilePath, result.Error)
        continue
    }

    fmt.Printf("Processed %s: %d headings\n",
        result.FilePath, len(result.Headings))
}
```

### Using Cache

The processor automatically caches results to avoid reprocessing files.

```go
processor := markdown.NewDocumentProcessor(4)

// First call - reads from disk
result1 := processor.ProcessFile("chapter.md")

// Second call - returns cached result
result2 := processor.ProcessFile("chapter.md")

// Clear cache when needed
processor.ClearCache()
```

## TableOfContentsGenerator

Automatically generate table of contents from heading information.

### HTML Format TOC

```go
generator := markdown.NewTableOfContentsGenerator(3, true) // HTML format, max 3 levels

parser := markdown.NewParser()
_, headings, _ := parser.Parse(source)

toc := generator.Generate(headings)
fmt.Println(toc)
// Output:
// <nav class="toc">
// <ul>
// <li><a href="#chapter-1">Chapter 1</a></li>
// <ul>
// <li><a href="#section-1-1">Section 1.1</a></li>
// </ul>
// </ul>
// </nav>
```

### Markdown Format TOC

```go
generator := markdown.NewTableOfContentsGenerator(3, false) // Markdown format

toc := generator.Generate(headings)
// Output:
// - [Chapter 1](#chapter-1)
//   - [Section 1.1](#section-1-1)
```

### Controlling TOC Depth

```go
// Only show up to level 2 headings
shallowTOC := markdown.NewTableOfContentsGenerator(2, true)

// Show all levels (1-6)
deepTOC := markdown.NewTableOfContentsGenerator(6, true)
```

## DocumentBuilder

Build complete books from multiple Markdown files.

### Simple Document Combining

```go
builder := markdown.NewDocumentBuilder()

// Add files in order
builder.AddFile("intro.md")
builder.AddFile("chapter1.md")
builder.AddFile("chapter2.md")
builder.AddFile("conclusion.md")

// Build the document
html, headings, err := builder.Build()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated %d headings\n", len(headings))
```

### Building with Table of Contents

```go
builder := markdown.NewDocumentBuilder()

builder.AddFile("chapter1.md")
builder.AddFile("chapter2.md")

// Automatically generates and includes TOC
html, headings, err := builder.BuildWithTOC()
if err != nil {
    log.Fatal(err)
}

// HTML now includes:
// <div class="document">
//   <div class="toc-section">
//     <!-- Auto-generated TOC -->
//   </div>
//   <div class="content-section">
//     <!-- Combined content -->
//   </div>
// </div>
```

### Handling ID Conflicts

When building multiple files, the DocumentBuilder automatically adjusts heading IDs to prevent conflicts:

```go
// chapter1.md has: # Introduction (generates id: "introduction")
// chapter2.md also has: # Introduction (generates id: "chapter2-introduction")

builder.AddFile("chapter1.md")
builder.AddFile("chapter2.md")

html, headings, _ := builder.Build()

// Headings will have unique IDs across all files
for _, h := range headings {
    fmt.Printf("%s -> #%s\n", h.Text, h.ID)
}
// Output:
// Introduction -> #introduction
// Introduction -> #chapter2-introduction
```

## HTML Export

Export to complete, styled HTML documents.

### Basic Export

```go
html, _, _ := parser.Parse(source)

// Wrap in complete HTML document
complete := markdown.ExportHTML(html, markdown.ExportOptions{
    DocumentTitle: "My Book",
    Author: "John Doe",
    IncludeCSS: true,
})

err := ioutil.WriteFile("output.html", []byte(complete), 0644)
```

### Custom Styling

```go
opts := markdown.ExportOptions{
    DocumentTitle: "My Book",
    IncludeCSS: true,
    CustomCSS: `
    body {
        background-color: #f5f5f5;
    }
    h1 {
        color: #2c3e50;
    }
    code {
        font-size: 0.95em;
    }
    `,
}

complete := markdown.ExportHTML(html, opts)
```

### Export Complete Document Flow

```go
// Process files
processor := markdown.NewDocumentProcessor(4)
files := findMarkdownFiles("./chapters")
results := processor.ProcessFiles(files)

// Combine them
var htmlContent strings.Builder
for _, result := range results {
    htmlContent.WriteString(result.HTML)
    htmlContent.WriteString("<hr />\n")
}

// Generate TOC
var allHeadings []markdown.HeadingInfo
for _, result := range results {
    allHeadings = append(allHeadings, result.Headings...)
}

tocGenerator := markdown.NewTableOfContentsGenerator(3, true)
toc := tocGenerator.Generate(allHeadings)

// Assemble final document
finalHTML := `<div class="document">
    <div class="toc-section">` + toc + `</div>
    <div class="content-section">` + htmlContent.String() + `</div>
</div>`

// Export as complete HTML
output := markdown.ExportHTML(finalHTML, markdown.ExportOptions{
    DocumentTitle: "Book Title",
    Author: "Author Name",
    IncludeCSS: true,
})

ioutil.WriteFile("book.html", []byte(output), 0644)
```

## Complete Example: Building a Book

Here's a complete example of building a book from multiple chapters:

```go
package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "github.com/yeasy/mdpress/internal/markdown"
)

func main() {
    // 1. Find all markdown files
    files, _ := filepath.Glob("chapters/*.md")
    if len(files) == 0 {
        log.Fatal("No markdown files found")
    }

    // 2. Create and configure builder
    builder := markdown.NewDocumentBuilder()
    for _, file := range files {
        builder.AddFile(file)
    }

    // 3. Build document with TOC
    html, headings, err := builder.BuildWithTOC()
    if err != nil {
        log.Fatal(err)
    }

    // 4. Export as complete HTML
    exportOpts := markdown.ExportOptions{
        DocumentTitle: "My Book",
        Author: "John Doe",
        IncludeCSS: true,
        CustomCSS: customStyles(),
    }

    complete := markdown.ExportHTML(html, exportOpts)

    // 5. Write output
    err = ioutil.WriteFile("output.html", []byte(complete), 0644)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Book built successfully!\n")
    fmt.Printf("Generated %d headings\n", len(headings))
    fmt.Println("Output: output.html")
}

func customStyles() string {
    return `
    .document {
        font-family: "Georgia", serif;
    }

    .toc-section {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
        border-radius: 8px;
    }

    .toc-section a {
        color: #fff;
    }
    `
}
```

## Performance Considerations

### Concurrency Tuning

```go
// For I/O bound work, use more workers
processor := markdown.NewDocumentProcessor(8)

// For CPU bound work, match your core count
// Note: Markdown parsing is mostly I/O bound (file reading)
processor := markdown.NewDocumentProcessor(4)
```

### Caching Strategy

```go
processor := markdown.NewDocumentProcessor(4)

// Reuse processor for multiple operations
// to benefit from caching
results1 := processor.ProcessFiles(files)
// Subsequent calls to ProcessFile will use cache
result := processor.ProcessFile("chapter1.md") // fast!

// Clear cache when you know files have changed
processor.ClearCache()
results2 := processor.ProcessFiles(files)
```

## Error Handling

### Processing Errors

```go
results := processor.ProcessFiles(files)

var errors []string
for _, result := range results {
    if result.Error != nil {
        errors = append(errors, fmt.Sprintf(
            "%s: %v",
            result.FilePath,
            result.Error,
        ))
    }
}

if len(errors) > 0 {
    fmt.Println("Processing errors:")
    for _, err := range errors {
        fmt.Println(" -", err)
    }
}
```

### File Not Found Handling

```go
result := processor.ProcessFile("missing.md")
if result.Error != nil {
    if os.IsNotExist(result.Error) {
        fmt.Printf("File not found: %s\n", result.FilePath)
    } else {
        fmt.Printf("Error: %v\n", result.Error)
    }
}
```

## CSS Classes and Styling

The generated HTML uses semantic class names for styling:

```html
<nav class="toc">
    <ul>
        <li><a href="#...">...</a></li>
    </ul>
</nav>

<div class="document">
    <div class="toc-section"><!-- TOC --></div>
    <div class="content-section"><!-- Content --></div>
</div>
```

Use these classes to customize the appearance:

```css
/* Style the table of contents */
nav.toc {
    background: #f8f9fa;
    border-left: 4px solid #667eea;
}

/* Style the main document layout */
.document {
    display: grid;
    grid-template-columns: 250px 1fr;
    gap: 30px;
}

/* Make responsive */
@media (max-width: 768px) {
    .document {
        grid-template-columns: 1fr;
    }

    .toc-section {
        order: -1; /* TOC appears first on mobile */
    }
}
```

## Troubleshooting

### Issue: Slow Performance

**Solution:**
- Increase concurrency in DocumentProcessor
- Check for large files (>5MB)
- Monitor memory usage for cache

### Issue: Duplicate IDs in Combined Documents

**Solution:**
- Use DocumentBuilder which automatically handles this
- Or manually adjust IDs before combining

### Issue: Missing Headings in TOC

**Solution:**
- Check `maxLevel` parameter in TableOfContentsGenerator
- Verify headings have been collected (not filtered out)
- Ensure heading IDs are valid (not empty)

## Best Practices

1. **Reuse Parser Instances**: Create once, reuse for multiple documents
2. **Enable Caching**: Use DocumentProcessor for batches of files
3. **Limit Concurrency**: Use 4-8 workers depending on system
4. **Include CSS**: Always export with CSS for proper formatting
5. **Test Output**: Validate generated HTML in browsers
6. **Document Structure**: Maintain consistent heading hierarchy
7. **ID Naming**: Use semantic file names for better ID prefixes

## Advanced Topics

### Custom Table of Contents Filtering

```go
// Filter headings before TOC generation
filtered := make([]markdown.HeadingInfo, 0)
for _, h := range allHeadings {
    if h.Level <= 2 { // Only show H1 and H2
        filtered = append(filtered, h)
    }
}

toc := generator.Generate(filtered)
```

### Multi-Language Support

```go
// Heading IDs work with any Unicode characters
// Language-specific heading text is preserved
parser := markdown.NewParser()

chinese := []byte(`# 中文标题
## 小章节`)

html, headings, _ := parser.Parse(chinese)
// IDs will be: "zhong-wen-biao-ti", "xiao-zhang-jie"
```

### External Document References

```go
// Build separate documents
book1 := buildBook("book1/chapters")
book2 := buildBook("book2/chapters")

// Link between them using IDs
crossRef := fmt.Sprintf(
    `<a href="book2.html#%s">See related topic</a>`,
    targetHeadingID,
)
```
