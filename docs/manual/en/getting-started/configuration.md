# Configuration

Configure mdPress behavior and appearance through `book.yaml`, `book.json`, or `SUMMARY.md`. Learn what options are available and how to use them.

## Configuration Files

mdPress supports multiple configuration file formats. It searches for them in this order:

1. `book.yaml` (recommended)
2. `book.json`
3. `book.toml`

Only one is needed. Create the file in your project root:

```bash
# Create an empty book.yaml
touch book.yaml
```

## Book Metadata

Configure your documentation's basic information:

```yaml
book:
  title: My Documentation
  author: John Smith
  description: A comprehensive guide to using my product
  language: en
  direction: ltr
```

In `book.json`:

```json
{
  "book": {
    "title": "My Documentation",
    "author": "John Smith",
    "description": "A comprehensive guide",
    "language": "en",
    "direction": "ltr"
  }
}
```

Available fields:
- `title` - Shown in browser tab and headers
- `author` - Document metadata
- `description` - Used in search engines and social media
- `language` - Language code (en, fr, es, etc.)
- `direction` - Text direction: `ltr` (left-to-right) or `rtl` (right-to-left)

## Chapters and Structure

Define chapter sources in configuration. This is optional if you use `SUMMARY.md`:

```yaml
chapters:
  - path: README.md
  - path: chapters/chapter1.md
  - path: chapters/chapter2.md
    chapters:
      - path: chapters/chapter2/section1.md
      - path: chapters/chapter2/section2.md
  - path: chapters/chapter3.md
```

Or use `SUMMARY.md` instead (recommended for most projects). If both exist, `SUMMARY.md` takes precedence.

## Style Configuration

Control appearance with style options:

```yaml
style:
  theme: light
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: 16
  code_theme: monokai
  line_height: 1.6
  margins:
    top: 20mm
    bottom: 20mm
    left: 20mm
    right: 20mm
  header:
    left: Chapter {chapter_num}
    center: My Documentation
    right: Page {page_num}
  footer:
    left: ""
    center: ""
    right: "© 2026 Company Name"
  custom_css: custom.css
```

### Theme Options

Available themes:
- `light` - Clean, light background (default)
- `dark` - Dark background for low-light viewing
- `auto` - Follow system preference

```yaml
style:
  theme: light
```

### Page Size

Set PDF page dimensions:
- `A4` - Standard (210 × 297 mm)
- `A5` - Half A4
- `Letter` - US Letter (8.5 × 11 in)
- `Custom` - With width and height

```yaml
style:
  page_size: A4

# Or custom:
style:
  page_size: Custom
  page_width: 200mm
  page_height: 280mm
```

### Font Configuration

Choose fonts for body text and code:

```yaml
style:
  font_family: "Georgia, serif"
  font_size: 14
  code_font: "Monaco, monospace"
  code_font_size: 13
```

Use system fonts or web-safe fonts. For embedded fonts, place `.ttf` or `.woff2` files in assets and reference:

```yaml
style:
  font_family: "CustomFont"
  custom_fonts:
    - name: CustomFont
      path: assets/fonts/custom.ttf
```

### Code Highlighting Theme

Syntax highlighting for code blocks:

- `monokai` - Dark with colorful syntax
- `atom-one-dark` - Atom editor dark theme
- `atom-one-light` - Atom editor light theme
- `dracula` - Popular dark theme
- `gruvbox` - Retro groove colors
- `solarized-dark` - Solarized dark variant
- `solarized-light` - Solarized light variant
- `github` - GitHub default theme
- `nord` - Arctic, north-bluish theme

```yaml
style:
  code_theme: monokai
```

### Line Height and Margins

Control spacing and layout:

```yaml
style:
  line_height: 1.6        # 1.0-2.0, affects readability
  margins:
    top: 25mm
    bottom: 25mm
    left: 20mm
    right: 20mm
```

### Headers and Footers

Add text to page headers and footers (PDF only):

```yaml
style:
  header:
    left: Chapter {chapter_num}
    center: {title}
    right: ""
  footer:
    left: ""
    center: Page {page_num}
    right: {date}
```

Available variables:
- `{title}` - Book title
- `{author}` - Book author
- `{page_num}` - Current page number
- `{total_pages}` - Total page count
- `{chapter_num}` - Current chapter number
- `{date}` - Build date
- `{time}` - Build time

### Custom CSS

Add custom styling:

```yaml
style:
  custom_css: custom.css
```

Create `custom.css` in your project root:

```css
body {
  font-family: "Georgia", serif;
  color: #333;
}

h1, h2, h3 {
  color: #0066cc;
}

code {
  background-color: #f5f5f5;
  padding: 2px 4px;
  border-radius: 3px;
}
```

## Output Configuration

Configure output format behavior:

```yaml
output:
  formats:
    - site
    - pdf
    - epub
  toc: true
  toc_depth: 3
  cover: true
  watermark: ""
  pdf_timeout: 300
  bookmarks: true
  minify: false
```

### Formats

List output formats to generate:

```yaml
output:
  formats:
    - site      # HTML static site
    - pdf       # Single PDF document
    - epub      # E-book format
```

### Table of Contents

Control TOC generation:

```yaml
output:
  toc: true                    # Generate table of contents
  toc_depth: 3                 # Include headings up to level 3
```

### Cover Page

Enable/disable cover generation:

```yaml
output:
  cover: true                  # Generate cover page
  cover_image: assets/cover.png  # Optional custom cover image
```

### Watermark

Add watermark to PDF output (useful for drafts):

```yaml
output:
  watermark: "DRAFT"
```

### PDF Timeout

Set PDF generation timeout in seconds:

```yaml
output:
  pdf_timeout: 300            # 5 minutes
```

Increase this for large documents that take longer to render.

### Bookmarks

Include PDF bookmarks (outline):

```yaml
output:
  bookmarks: true             # Enable PDF outline/bookmarks
```

Bookmarks allow readers to navigate via PDF viewers.

### Minification

Minify HTML and CSS for smaller output:

```yaml
output:
  minify: true
```

## Plugin Configuration

Enable optional plugins:

```yaml
plugins:
  - mermaid                    # Diagram support
  - mathjax                    # Math equations
  - highlighting               # Code highlighting
```

Available plugins depend on your mdPress installation.

## Complete Example

Here's a complete `book.yaml` configuration:

```yaml
book:
  title: Complete Guide to mdPress
  author: Jane Developer
  description: Learn mdPress from basic to advanced usage
  language: en

style:
  theme: light
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: 15
  code_theme: monokai
  line_height: 1.6
  margins:
    top: 20mm
    bottom: 20mm
    left: 20mm
    right: 20mm
  header:
    left: Chapter {chapter_num}
    center: ""
    right: Page {page_num}
  footer:
    left: ""
    center: {title}
    right: "© 2026 Jane Developer"
  custom_css: custom.css

output:
  formats:
    - site
    - pdf
    - epub
  toc: true
  toc_depth: 3
  cover: true
  pdf_timeout: 300
  bookmarks: true
  minify: false

plugins:
  - mermaid
```

## Configuration Inheritance

If you create `book.yaml` without specifying all options, mdPress uses defaults for missing values:

```yaml
# Minimal config - uses defaults for everything else
book:
  title: My Project
```

This is perfectly fine. Add configuration only when you need to customize behavior.

## Troubleshooting Configuration

### Changes not taking effect

Restart the development server:

```bash
# Stop current server (Ctrl+C)
# Then restart:
mdpress serve
```

### Invalid YAML syntax

Check your `book.yaml` for proper indentation and formatting:

```yaml
# ✓ Correct
style:
  theme: light
  font_size: 16

# ✗ Wrong - inconsistent indentation
style:
  theme: light
    font_size: 16
```

Use a YAML validator or editor with YAML support to catch errors.

### Configuration not found

Ensure `book.yaml` is in your project root, not in a subdirectory:

```
✓ project/book.yaml
✗ project/config/book.yaml
```
