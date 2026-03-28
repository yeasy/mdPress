# Project Structure

Understanding mdPress project layout helps you organize content effectively and use all available features.

## Directory Structure

A typical mdPress project looks like this:

```
my-documentation/
├── book.yaml                # Configuration file (optional)
├── README.md                # Landing page / introduction
├── SUMMARY.md               # Table of contents structure
├── GLOSSARY.md              # Term definitions (optional)
├── LANGS.md                 # Multi-language support (optional)
├── assets/                  # Images, diagrams, files
│   ├── images/
│   │   ├── logo.png
│   │   └── diagram.svg
│   ├── diagrams/
│   └── downloads/
└── chapters/                # Content organized in directories
    ├── getting-started/
    │   ├── installation.md
    │   └── configuration.md
    ├── guide/
    │   ├── basic-usage.md
    │   └── advanced-tips.md
    └── reference/
        └── api.md
```

## Core Files

### README.md

The landing page shown when users first visit your documentation:

```markdown
# Project Title

Introduction text, overview, and navigation guidance.

## Key Sections

- [Getting Started](chapters/getting-started.md)
- [User Guide](chapters/guide/index.md)
```

mdPress skips top-level README.md when scanning for book content (it treats it as project documentation, not a chapter). To include it in your book, list it explicitly in SUMMARY.md.

### SUMMARY.md

Defines the table of contents and chapter structure. This is the most important file:

```markdown
# Summary

- [Introduction](README.md)
- [Chapter 1](chapter1.md)
- [Chapter 2](chapter2.md)
  - [Section 2.1](chapter2/section1.md)
  - [Section 2.2](chapter2/section2.md)
- [Appendix](appendix.md)
```

Format rules:
- Each line starting with `-` is a chapter entry
- Indentation creates nesting (up to any depth)
- Link format: `[Display Text](file_path.md)`
- Paths are relative to project root

## Configuration File (Optional)

Create `book.yaml` for advanced settings:

```yaml
book:
  title: My Documentation
  author: Author Name
  description: A brief description

style:
  theme: technical
```

mdPress also accepts `book.json` (for GitBook compatibility). If no config file exists, mdPress uses sensible defaults (zero-config mode).

## Asset Directory

Store media and downloadable files in `assets/`:

```
assets/
├── images/          # PNG, JPG, SVG, GIF
├── diagrams/        # PlantUML or Mermaid
└── downloads/       # Attachable files
```

Reference assets in your markdown:

```markdown
![My Image](assets/images/screenshot.png)

[Download PDF](assets/downloads/guide.pdf)
```

mdPress automatically processes images and optimizes them for web output.

## Optional Special Files

### GLOSSARY.md

Define terminology used throughout your documentation:

```markdown
# Glossary

## API
Application Programming Interface. Defines how software components communicate.

## CLI
Command-Line Interface. Text-based user interface.

## REST
Representational State Transfer. API architectural style.
```

When terms from GLOSSARY.md appear in your content, they're linked to their definitions.

### LANGS.md

Support multiple languages (for documentation translation):

```markdown
# Languages

- [English](.)
- [Español](es/)
- [Français](fr/)
```

This creates a language switcher. Each language gets its own subdirectory:

```
documentation/
├── README.md              # English
├── SUMMARY.md
├── es/
│   ├── README.md          # Spanish
│   └── SUMMARY.md
└── fr/
    ├── README.md          # French
    └── SUMMARY.md
```

## Chapter Organization

Organize chapters in nested directories for clarity:

```
chapters/
├── 01-getting-started/
│   ├── index.md
│   ├── installation.md
│   └── first-project.md
├── 02-user-guide/
│   ├── index.md
│   ├── basic-concepts.md
│   ├── workflows/
│   │   ├── workflow-a.md
│   │   └── workflow-b.md
│   └── best-practices.md
└── 03-reference/
    ├── api.md
    └── cli.md
```

Reference in SUMMARY.md:

```markdown
- [Getting Started](chapters/01-getting-started/index.md)
  - [Installation](chapters/01-getting-started/installation.md)
  - [First Project](chapters/01-getting-started/first-project.md)
- [User Guide](chapters/02-user-guide/index.md)
  - [Basic Concepts](chapters/02-user-guide/basic-concepts.md)
  - [Workflows](chapters/02-user-guide/workflows/index.md)
    - [Workflow A](chapters/02-user-guide/workflows/workflow-a.md)
```

## Zero-Config Auto-Discovery

mdPress can run without any configuration:

1. Place markdown files in your project
2. Create SUMMARY.md with your structure
3. Run `mdpress serve` or `mdpress build`

mdPress automatically:
- Finds README.md as the landing page
- Parses SUMMARY.md structure
- Discovers assets in standard locations
- Applies default styling and formatting

You only need configuration files when customizing behavior beyond defaults.

## Best Practices

### File Naming

Use clear, lowercase filenames with hyphens:

```
Good:  installation.md, getting-started.md, api-reference.md
Bad:   Installation.md, Getting Started.md, API_REFERENCE.MD
```

### Directory Depth

Keep nesting reasonable (3 levels is typical):

```
✓ chapters/section/subsection/page.md
✗ chapters/a/b/c/d/e/f/page.md (too deep)
```

### Asset Organization

Group similar assets:

```
assets/
├── images/           # All images
├── icons/            # UI icons
├── screenshots/      # Application screenshots
└── downloads/        # Attachable files
```

### Cross-References

Link between chapters:

```markdown
See also: [Installation Guide](installation.md)

Or with full path: [Configuration](../getting-started/configuration.md)
```

### Section Anchors

Link to specific headings:

```markdown
[Jump to API Reference](#api-reference)

Or: [Jump to another chapter](#installation)
```
