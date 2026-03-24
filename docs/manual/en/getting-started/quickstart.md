# Quick Start

Get a documentation site or book up and running in minutes with mdPress.

## Using the Quickstart Command

The fastest way to start is with the built-in quickstart template:

```bash
mdpress quickstart my-book
cd my-book
mdpress serve --open
```

This creates a ready-to-use project with example content and configuration. Your browser opens automatically to http://localhost:9000.

## Creating a Project Manually

If you prefer to start from scratch or understand the basics, here's the minimal setup:

### Step 1: Create a Project Directory

```bash
mkdir my-documentation
cd my-documentation
```

### Step 2: Create README.md

Create a `README.md` file with your introduction:

```markdown
# My Documentation

Welcome to my documentation site. This is the landing page that readers see first.
```

### Step 3: Create SUMMARY.md

The `SUMMARY.md` file defines your book's structure:

```markdown
# Summary

- [Introduction](README.md)
- [Getting Started](chapters/getting-started.md)
  - [Installation](chapters/installation.md)
  - [Configuration](chapters/configuration.md)
- [Advanced Topics](chapters/advanced.md)
- [FAQ](chapters/faq.md)
```

### Step 4: Create Chapter Files

Create the chapter files referenced in SUMMARY.md:

```bash
mkdir chapters
```

Then create `chapters/getting-started.md`:

```markdown
# Getting Started

This is the main getting started page.
```

Create `chapters/installation.md`:

```markdown
# Installation

Installation instructions go here.
```

Continue for other chapters.

### Step 5: Verify Your Project Structure

Your project should now look like this:

```
my-documentation/
├── README.md
├── SUMMARY.md
└── chapters/
    ├── getting-started.md
    ├── installation.md
    ├── configuration.md
    ├── advanced.md
    └── faq.md
```

## Running the Development Server

Start the development server with live reload:

```bash
mdpress serve
```

By default, your site is available at http://localhost:9000. The browser automatically reloads when you change files.

To open your browser automatically:

```bash
mdpress serve --open
```

To use a custom port:

```bash
mdpress serve --port 3000
```

## Building Your Documentation

When you're ready to deploy, build a static site:

```bash
mdpress build --format site
```

This creates an `_book` directory with your complete HTML site ready for hosting.

### Building to PDF

To generate a PDF version:

```bash
mdpress build --format pdf
```

This creates a single `book.pdf` file containing all your documentation.

### Building to EPUB

For e-book format:

```bash
mdpress build --format epub
```

Creates `book.epub` ready for e-readers.

### Building Multiple Formats

Build all formats at once:

```bash
mdpress build
```

This generates HTML site, PDF, and EPUB if available.

## Understanding Commands

### mdpress serve

The development command for writing and testing:

```bash
mdpress serve [OPTIONS]
```

Available options:
- `--open` - Open browser automatically
- `--port <PORT>` - Use custom port (default: 8080)
- `--watch` - Watch for file changes and rebuild (default: enabled)
- `--no-cache` - Disable caching during development

The server watches all source files and rebuilds instantly when you save changes.

### mdpress build

The production command for generating output:

```bash
mdpress build [OPTIONS]
```

Available options:
- `--format <FORMAT>` - Output format: `site`, `pdf`, `epub`, `html`
- `--output <PATH>` - Output directory (default: `_book`)
- `--minify` - Minify HTML and CSS for smaller file sizes
- `--no-cache` - Disable caching

## Next Steps

Now that your documentation is running, you can:

1. **Configure your project** - Create `book.yaml` for advanced settings (see Configuration reference)
2. **Add assets** - Put images and files in an `assets/` directory
3. **Create a glossary** - Add `GLOSSARY.md` for term definitions
4. **Customize styling** - Add custom CSS or choose themes in configuration

See the Configuration guide for more advanced options.
