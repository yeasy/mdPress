# Chapter 1: Quick Start

This chapter helps you get started with mdpress, from installation to generating your first PDF book.

## 1.1 Installation

### Install with Go

Ensure you have Go 1.25 or later installed, then run:

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
- [ ] Generate PDF

## 1.4 Generate PDF

When everything is ready, run:

```bash
mdpress build
```

Or specify a config file:

```bash
mdpress build --config path/to/book.yaml
```

The generated PDF will be saved to the path specified in `output.filename` in the config file.

## 1.5 Summary

Congratulations! You have successfully generated your first PDF book. Read Chapter 2 for more advanced features.
