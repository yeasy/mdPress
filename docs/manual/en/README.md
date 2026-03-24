# mdPress User Manual

> Modern Markdown publishing, from source to book.

mdPress is a command-line tool that transforms Markdown files into beautifully formatted books. It produces multi-page websites, PDFs, single-page HTML, and ePub — all from the same source.

## Why mdPress?

Writing should be simple. You focus on content in Markdown; mdPress handles the rest — layout, navigation, search, syntax highlighting, math rendering, diagrams, and more.

mdPress works out of the box with zero configuration. Point it at a directory of Markdown files and it builds a complete book. When you need more control, a single `book.yaml` gives you full power over structure, styling, and output.

## Key Features

- **Multiple output formats** — Site, PDF, HTML, ePub, and Typst in one command
- **Zero-config mode** — Auto-discovers and orders Markdown files
- **Live preview** — Hot-reloading development server
- **Full-text search** — Built-in client-side search for site output
- **Dark mode** — Automatic light/dark theme with system detection
- **Math & diagrams** — KaTeX math, Mermaid and PlantUML diagrams
- **CJK support** — First-class Chinese, Japanese, and Korean typography
- **Plugin system** — Extend the build pipeline with custom hooks
- **GitBook migration** — One-command migration from GitBook/HonKit
- **Multi-language** — Build the same book in multiple languages
- **Incremental builds** — Hash-based caching for fast rebuilds

## Quick Example

```bash
# Install
go install github.com/yeasy/mdpress@latest

# Create a new project
mdpress quickstart my-book
cd my-book

# Preview in browser
mdpress serve --open

# Build all formats
mdpress build --format all
```

## How This Manual Is Organized

- **Getting Started** — Installation, your first project, and configuration basics
- **User Guide** — Writing content, Markdown extensions, output formats, and more
- **Themes & Customization** — Built-in themes, custom CSS, dark mode, headers/footers
- **Plugin Development** — Building plugins to extend the build pipeline
- **Best Practices** — Organizing large books, performance, CI/CD
- **Troubleshooting** — Common issues, environment checks, FAQ
- **Reference** — Complete CLI and configuration reference
