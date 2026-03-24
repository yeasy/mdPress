# Organizing Large Books

When building multi-chapter books with hundreds of pages, a well-organized structure makes maintenance easier and helps mdPress build efficiently. This guide covers strategies for managing large documentation and books.

## Chapter Grouping with SUMMARY.md

For complex books, use a `SUMMARY.md` file to define hierarchical chapter structures with sections:

```markdown
# Summary

- [Introduction](intro.md)
- [Part One: Fundamentals](part1.md)
  - [Chapter 1: Basics](ch01.md)
  - [Chapter 2: Installation](ch02.md)
- [Part Two: Advanced Topics](part2.md)
  - [Chapter 3: Architecture](ch03.md)
  - [Chapter 4: Optimization](ch04.md)
- [Appendix](appendix.md)
```

mdPress reads this structure automatically when `chapters` is omitted from `book.yaml`. Each indentation level creates nested sections, allowing you to organize chapters into logical parts.

## Alternative: Explicit Chapters in book.yaml

For large books, you can define chapters explicitly in `book.yaml` with nested sections:

```yaml
book:
  title: "Advanced Guide"
  author: "Your Name"

chapters:
  - title: "Introduction"
    file: "intro.md"
  - title: "Part One: Fundamentals"
    file: "part1.md"
    sections:
      - title: "Chapter 1: Basics"
        file: "chapters/ch01.md"
      - title: "Chapter 2: Installation"
        file: "chapters/ch02.md"
  - title: "Part Two: Advanced Topics"
    file: "part2.md"
    sections:
      - title: "Chapter 3: Architecture"
        file: "chapters/ch03.md"
      - title: "Chapter 4: Optimization"
        file: "chapters/ch04.md"
```

The `file` field for part titles becomes the part introduction. Sections can be nested multiple levels deep.

## File Naming Conventions

Adopt consistent naming for easy navigation:

- **Sequential numbering**: `ch01.md`, `ch02.md`, etc.
- **Functional naming**: `installation.md`, `troubleshooting.md`, `api-reference.md`
- **Part-based grouping**: `intro.md`, `part1-basics.md`, `part2-advanced.md`
- **Descriptive names**: Use hyphens, not spaces: `getting-started.md` not `Getting Started.md`

Example structure for a 50-chapter book:

```
docs/
├── book.yaml
├── SUMMARY.md
├── intro.md
├── part1/
│   ├── ch01.md
│   ├── ch02.md
│   └── ch03.md
├── part2/
│   ├── ch04.md
│   ├── ch05.md
│   └── ch06.md
├── appendices/
│   ├── glossary.md
│   └── references.md
└── assets/
    ├── diagrams/
    └── screenshots/
```

## Using README.md in Subdirectories

Organize large books by creating subdirectories with README.md files that serve as part introductions:

```
my-book/
├── book.yaml
├── SUMMARY.md
├── part1/
│   ├── README.md          # "Part One: Basics" introduction
│   ├── ch01-intro.md
│   ├── ch02-setup.md
│   └── assets/
│       └── diagrams/
├── part2/
│   ├── README.md          # "Part Two: Advanced" introduction
│   ├── ch03-architecture.md
│   ├── ch04-performance.md
│   └── assets/
│       └── diagrams/
└── appendices/
    ├── README.md          # "Appendices" introduction
    ├── glossary.md
    └── references.md
```

In `SUMMARY.md`:

```markdown
# Summary

- [Introduction](intro.md)
- [Part One: Basics](part1/README.md)
  - [Chapter 1: Introduction](part1/ch01-intro.md)
  - [Chapter 2: Setup](part1/ch02-setup.md)
- [Part Two: Advanced](part2/README.md)
  - [Chapter 3: Architecture](part2/ch03-architecture.md)
  - [Chapter 4: Performance](part2/ch04-performance.md)
- [Appendices](appendices/README.md)
  - [Glossary](appendices/glossary.md)
  - [References](appendices/references.md)
```

## Splitting Long Chapters

Avoid single chapters exceeding 10,000 words (roughly 40-50 pages). Split into focused sections:

**Before (one 50-page chapter):**
```
api-guide.md (50 pages)
  - API Overview
  - Authentication
  - Endpoints
  - Error Handling
  - Rate Limiting
  - Examples
```

**After (five focused chapters):**
```
api-overview.md (10 pages)
api-authentication.md (10 pages)
api-endpoints.md (12 pages)
api-errors.md (8 pages)
api-examples.md (10 pages)
```

Use cross-references (see [Authentication](api-authentication.md)) to connect related chapters.

## Image Management in assets/

Create a dedicated `assets/` directory tree that mirrors chapter organization:

```
assets/
├── diagrams/
│   ├── architecture.png
│   ├── flow-chart.svg
│   └── deployment.png
├── screenshots/
│   ├── interface-main.png
│   ├── interface-settings.png
│   └── interface-profile.png
└── icons/
    ├── checkmark.svg
    ├── warning.svg
    └── info.svg
```

Reference images with relative paths in your chapters:

```markdown
# Installation

## System Requirements

![System Architecture](../assets/diagrams/architecture.png)

## Setup Steps

1. Download the installer
   ![Download Screen](../assets/screenshots/interface-main.png)
2. Configure options
3. Verify installation
```

For large books, organize by chapter:

```
assets/
├── part1-basics/
│   ├── ch01-getting-started/
│   │   ├── step1.png
│   │   └── step2.png
│   └── ch02-installation/
│       ├── download.png
│       └── setup.png
└── part2-advanced/
    └── ch03-architecture/
        ├── diagram1.svg
        └── diagram2.svg
```

Keep image file sizes optimized:
- Screenshots: under 500 KB each
- Diagrams (SVG): preferred for scalability
- Photos: under 2 MB total

## Consistent Heading Hierarchy

Use a clear heading structure within chapters:

```markdown
# Chapter 1: Getting Started      (H1 - chapter title)

## Section 1.1: Installation       (H2 - main sections)

### Step 1: Download              (H3 - subsections)

#### Windows Installation         (H4 - specific variants)

## Section 1.2: Configuration

### Basic Setup
### Advanced Options
```

Rules:
- Start each chapter with H1 (the chapter title)
- Use H2 for major sections
- Reserve H3 for subsections
- Avoid skipping levels (don't jump from H1 to H3)
- Keep H1 unique per chapter (don't repeat)

In your `book.yaml`, set the table-of-contents depth to match your structure:

```yaml
output:
  toc_max_depth: 3  # Include H1, H2, and H3 in the TOC
```

## Cross-Referencing Between Chapters

Link between chapters using relative file paths (Markdown syntax):

```markdown
For details on authentication, see [API Authentication](../api/authentication.md).

For step-by-step instructions, refer to [Installation Guide](../getting-started/installation.md#step-1-download).
```

For PDF output, mdPress automatically converts these into PDF internal links. For HTML output, ensure paths are relative and correct.

Example cross-references:

```markdown
# Advanced Configuration

As discussed in [Basic Setup](../introduction/setup.md), you should first install
the core package. This chapter builds on those foundations.

See also:
- [Performance Tuning](performance.md) in this part
- [Troubleshooting](../appendix/troubleshooting.md)
- [API Reference](../reference/api.md)
```

## Build Performance for Large Books

### Parallel Processing

mdPress automatically uses multiple CPU cores to parse chapters in parallel. No configuration needed—builds of large books automatically benefit from multi-core systems.

### Caching Strategy

For books with 50+ chapters, enable caching to speed up rebuilds:

```bash
# First build: full compilation
mdpress build --format pdf

# Subsequent builds: reuse cached chapters
mdpress build --format pdf

# If you encounter stale builds, force full rebuild
mdpress build --format pdf --no-cache
```

The cache stores compiled chapter content based on file hashes. Changes to one chapter don't require recompiling others.

### Incremental Development Workflow

When writing large books:

```bash
# Start development server for live preview
mdpress serve

# This watches file changes and rebuilds only affected chapters
# No need for --no-cache during development
```

## Example: Large Book Structure

A real-world 200-page technical book with 15 chapters:

```
technical-guide/
├── book.yaml
├── SUMMARY.md
├── intro.md
├── part1-foundations/
│   ├── README.md (part intro)
│   ├── ch01-overview.md
│   ├── ch02-installation.md
│   └── ch03-quick-start.md
├── part2-concepts/
│   ├── README.md (part intro)
│   ├── ch04-architecture.md
│   ├── ch05-data-model.md
│   ├── ch06-plugins.md
│   └── ch07-configuration.md
├── part3-advanced/
│   ├── README.md (part intro)
│   ├── ch08-performance.md
│   ├── ch09-scaling.md
│   ├── ch10-security.md
│   └── ch11-monitoring.md
├── part4-reference/
│   ├── README.md (part intro)
│   ├── ch12-api-reference.md
│   ├── ch13-cli-commands.md
│   └── ch14-configuration-reference.md
├── appendices/
│   ├── ch15-troubleshooting.md
│   ├── glossary.md
│   └── faq.md
└── assets/
    ├── diagrams/ (20 SVG files)
    ├── screenshots/ (30 PNG files)
    └── icons/ (10 SVG files)
```

With a `SUMMARY.md` that mirrors the directory structure and explicit references in `book.yaml`, this 200-page book builds efficiently with clear organization.

## Tips and Tricks

- **Use consistent terminology**: Maintain a glossary (GLOSSARY.md) for technical terms
- **Validate regularly**: Run `mdpress validate` to catch broken chapter references early
- **Preview often**: Use `mdpress serve` while writing to catch issues immediately
- **Keep chapters focused**: Aim for 5,000-10,000 words per chapter for readability
- **Document your structure**: Add a README at the project root explaining chapter organization
