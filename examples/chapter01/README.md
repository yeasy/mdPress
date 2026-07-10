# Chapter 1: Quick Start

This chapter helps you get started with mdpress, from installation to building your first book.

## 1.1 Installation

### Install with Go

Ensure you have Go 1.26 or later installed, then run:

```bash
go install github.com/yeasy/mdpress@latest
```

### Build from Source

```bash
git clone https://github.com/yeasy/mdpress.git
cd mdpress
make build
```

### Prerequisites

mdpress uses Chromium for PDF rendering. Ensure the following is installed:

- **macOS**: `brew install chromium` or install Google Chrome
- **Ubuntu/Debian**: `apt install chromium-browser`
- **Windows**: Install Google Chrome

## 1.2 Initialize a Project

Run in your book directory:

```bash
mdpress init
```

This creates the following file structure:

```
my-book/
├── book.yaml          # Config file
├── .gitignore         # Ignores build artifacts
├── preface.md         # Preface
└── chapter01/
    └── README.md      # Chapter 1
```

## 1.3 Write Content

Edit Markdown files with your favorite editor. mdpress fully supports GitHub Flavored Markdown (GFM), including:

### Tables

| Feature | Supported | Description |
|---------|-----------|-------------|
| Tables | ✅ | GFM table syntax |
| Task lists | ✅ | `- [x]` syntax |
| Footnotes | ✅ | `[^1]` syntax |
| Code highlighting | ✅ | Multi-language support |

### Code Highlighting

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, mdpress!")
}
```

```python
def hello():
    """mdpress supports syntax highlighting for many languages"""
    print("Hello, mdpress!")
```

### Task Lists

- [x] Install mdpress
- [x] Create project structure
- [ ] Write content
- [ ] Build the book

## 1.4 Build the Book

When everything is ready, run:

```bash
mdpress build
```

This builds a PDF by default; the file is saved to the path specified in `output.filename` in the config file. Use `--format` to build other formats from the same sources:

```bash
mdpress build --format site       # static website in _book/
mdpress build --format html,epub  # standalone HTML and EPUB
```

Or specify a config file:

```bash
mdpress build --config path/to/book.yaml
```

## 1.5 Summary

Congratulations! You have successfully built your first book. Read Chapter 2 for more advanced features.
