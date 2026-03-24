# Configuration Reference

Complete reference for all `book.yaml` fields and options.

## Configuration File Structure

The `book.yaml` file is organized into four main sections:

```yaml
book:        # Book metadata (title, author, etc.)
chapters:    # Chapter definitions and structure
style:       # Visual styling and themes
output:      # Output format settings
plugins:     # Build plugins
```

## book Section

Book metadata and general information.

### book.title

**Type:** string
**Default:** "Untitled Book"
**Required:** Yes

The book title, used in PDF metadata, headers, and outputs.

```yaml
book:
  title: "Advanced Python Programming"
```

### book.subtitle

**Type:** string
**Default:** ""
**Required:** No

Optional subtitle, displayed in covers and metadata.

```yaml
book:
  subtitle: "A Deep Dive into Modern Python"
```

### book.author

**Type:** string
**Default:** ""
**Required:** No

Book author name(s), used in PDF metadata and footers.

```yaml
book:
  author: "Jane Doe"
  # or multiple authors:
  author: "Jane Doe, John Smith"
```

### book.version

**Type:** string
**Default:** "1.0.0"
**Required:** No

Version string following semantic versioning (major.minor.patch).

```yaml
book:
  version: "2.1.0"
```

### book.language

**Type:** string
**Default:** "zh-CN"
**Required:** No

Language code (ISO 639-1 format). Affects font selection and text direction.

```yaml
book:
  language: "en"      # English
  language: "zh-CN"   # Chinese (Simplified)
  language: "ja"      # Japanese
  language: "ko"      # Korean
```

### book.description

**Type:** string
**Default:** ""
**Required:** No

Book description for metadata and HTML meta tags.

```yaml
book:
  description: "Learn advanced Python techniques and best practices"
```

### book.cover

**Type:** object
**Default:** `{}`
**Required:** No

Cover configuration for PDF and other formats.

#### book.cover.image

Path to cover image file.

```yaml
book:
  cover:
    image: "assets/cover.png"
```

Requirements:
- Format: PNG, JPEG, or PDF
- Recommended size: 1200×1600 pixels (3:4 aspect ratio)
- Maximum: 5 MB

#### book.cover.background

**Type:** string
**Default:** "#ffffff"

Background color if image doesn't fill page.

```yaml
book:
  cover:
    image: "assets/cover.png"
    background: "#1a1a2e"
```

## chapters Section

Chapter structure and file references.

### chapters

**Type:** array of chapter definitions
**Default:** auto-detected from SUMMARY.md
**Required:** Yes (if SUMMARY.md absent)

List of chapters with optional nested sections.

```yaml
chapters:
  - title: "Chapter 1: Introduction"
    file: "ch01.md"
  - title: "Chapter 2: Basics"
    file: "ch02.md"
    sections:
      - title: "2.1: Setup"
        file: "ch02-setup.md"
      - title: "2.2: Configuration"
        file: "ch02-config.md"
```

### chapters[].title

**Type:** string
**Required:** Yes

Chapter or section title, displayed in TOC and navigation.

### chapters[].file

**Type:** string
**Required:** Yes

Path to Markdown file, relative to `book.yaml` location.

```yaml
chapters:
  - title: "Introduction"
    file: "intro.md"           # Same directory
  - title: "Chapter 1"
    file: "chapters/ch01.md"   # Subdirectory
```

### chapters[].sections

**Type:** array
**Default:** `[]`
**Required:** No

Nested chapters or sections, allowing multiple levels of hierarchy.

```yaml
chapters:
  - title: "Part One"
    file: "part1.md"
    sections:
      - title: "Chapter 1"
        file: "ch01.md"
      - title: "Chapter 2"
        file: "ch02.md"
        sections:
          - title: "Section 2.1"
            file: "ch02-sec1.md"
```

## style Section

Visual styling, themes, fonts, and appearance.

### style.theme

**Type:** string
**Default:** "technical"

Built-in theme or path to custom theme.

Available themes:
- `technical` - Professional, code-friendly (default)
- `elegant` - Refined, serif-based
- `minimal` - Clean, minimalist

```yaml
style:
  theme: "technical"
  # or custom theme path:
  theme: "./themes/custom"
```

### style.page_size

**Type:** string
**Default:** "A4"

Paper size for PDF output.

```yaml
style:
  page_size: "A4"      # 210 × 297 mm (default)
  page_size: "A5"      # 148 × 210 mm (smaller)
  page_size: "Letter"  # 8.5 × 11 inches
  page_size: "Legal"   # 8.5 × 14 inches
  page_size: "B5"      # 176 × 250 mm
```

### style.font_family

**Type:** string
**Default:** System sans-serif stack
**Example:** "Noto Sans CJK SC, -apple-system, BlinkMacSystemFont, sans-serif"

Font family (CSS-style), with fallbacks.

```yaml
style:
  font_family: "Georgia, serif"
  font_family: "Noto Sans CJK SC, sans-serif"
  font_family: "'Courier New', monospace"
```

Use system fonts or web-safe fonts. For custom fonts, use `custom_css`:

```yaml
style:
  font_family: "MyCustomFont, sans-serif"
  custom_css: |
    @font-face {
      font-family: 'MyCustomFont';
      src: url('assets/font.ttf') format('truetype');
    }
```

### style.font_size

**Type:** string
**Default:** "12pt"

Base font size for body text.

```yaml
style:
  font_size: "11pt"
  font_size: "12pt"
  font_size: "13pt"
```

### style.code_theme

**Type:** string
**Default:** "github"

Syntax highlighting theme for code blocks.

Available themes: `github`, `monokai`, `atom-one-dark`, `vs-code`, etc.

```yaml
style:
  code_theme: "github"
```

### style.line_height

**Type:** float
**Default:** 1.6

Line spacing multiplier.

```yaml
style:
  line_height: 1.5      # Compact
  line_height: 1.6      # Default
  line_height: 1.8      # Loose
  line_height: 2.0      # Very loose
```

### style.margin

**Type:** object
**Default:** `{top: 25, bottom: 25, left: 20, right: 20}`

Page margins in millimeters.

```yaml
style:
  margin:
    top: 25
    bottom: 25
    left: 20
    right: 20
```

Or use individual settings (see output section below).

### style.header

**Type:** object
**Default:** `{left: "{{.Book.Title}}", center: "", right: "{{.Chapter.Title}}"}`

Header text with template variables.

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "Chapter {{.Chapter.Number}}"
    right: "{{.Chapter.Title}}"
```

See [template-variables.md](template-variables.md) for available variables.

### style.footer

**Type:** object
**Default:** `{left: "", center: "{{.PageNum}}", right: ""}`

Footer text with template variables.

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "Confidential"
```

### style.custom_css

**Type:** string
**Default:** ""

Custom CSS for styling (PDF and HTML output).

```yaml
style:
  custom_css: |
    body {
      font-family: Georgia, serif;
      color: #333;
    }
    h1 {
      color: #1a1a2e;
      border-bottom: 2px solid #1a1a2e;
    }
    code {
      background: #f5f5f5;
      padding: 2px 4px;
    }
```

## output Section

Output format settings and rendering options.

### output.formats

**Type:** array
**Default:** `["pdf"]`

List of output formats to generate.

```yaml
output:
  formats:
    - pdf      # Requires Chrome/Chromium
    - html     # Single HTML file
    - epub     # E-book format
    - site     # Multi-page website
```

Generate multiple formats in one build:

```bash
mdpress build --format pdf --format html --format epub
```

### output.filename

**Type:** string
**Default:** auto-generated from title
**Example:** "output.pdf"

Output filename (for single formats) or prefix (for multiple).

```yaml
output:
  filename: "my-book.pdf"
  filename: "documentation.html"
```

### output.toc

**Type:** boolean
**Default:** true

Generate table of contents.

```yaml
output:
  toc: true      # Include TOC
  toc: false     # Omit TOC
```

### output.toc_max_depth

**Type:** integer
**Default:** 2
**Range:** 1-6

Maximum heading level to include in TOC. 1 = H1 only, 2 = H1+H2, etc.

```yaml
output:
  toc_max_depth: 2  # H1 and H2
  toc_max_depth: 3  # H1, H2, H3
```

### output.cover

**Type:** boolean
**Default:** true

Generate cover page.

```yaml
output:
  cover: true       # Include cover
  cover: false      # Omit cover
```

### output.header

**Type:** boolean
**Default:** true

Include page headers in output.

```yaml
output:
  header: true      # Include headers
  header: false     # Omit headers
```

### output.footer

**Type:** boolean
**Default:** true

Include page footers in output.

```yaml
output:
  footer: true      # Include footers
  footer: false     # Omit footers
```

### output.pdf_timeout

**Type:** integer
**Default:** 120
**Unit:** seconds

Maximum time to wait for PDF rendering per page.

```yaml
output:
  pdf_timeout: 120     # 2 minutes (default)
  pdf_timeout: 300     # 5 minutes (large books)
```

Increase for complex documents or slow systems.

### output.watermark

**Type:** string
**Default:** ""

Text or image to overlay on pages.

```yaml
output:
  watermark: "DRAFT"
  watermark: "CONFIDENTIAL"
  watermark: "assets/watermark.png"
```

### output.watermark_opacity

**Type:** float
**Default:** 0.1
**Range:** 0.0 - 1.0

Watermark transparency (0 = transparent, 1 = opaque).

```yaml
output:
  watermark: "DRAFT"
  watermark_opacity: 0.1      # Subtle (default)
  watermark_opacity: 0.3      # Visible
  watermark_opacity: 0.5      # Very visible
```

### output.margin_top

**Type:** string
**Default:** "15mm"

Top page margin. Units: mm, cm, in

```yaml
output:
  margin_top: "15mm"
  margin_top: "1.5cm"
  margin_top: "0.6in"
```

### output.margin_bottom

**Type:** string
**Default:** "15mm"

Bottom page margin.

### output.margin_left

**Type:** string
**Default:** "20mm"

Left page margin.

### output.margin_right

**Type:** string
**Default:** "20mm"

Right page margin.

### output.generate_bookmarks

**Type:** boolean
**Default:** true

Generate PDF bookmarks from heading hierarchy.

```yaml
output:
  generate_bookmarks: true   # Include bookmarks
  generate_bookmarks: false  # Omit bookmarks
```

Enables quick navigation in PDF readers.

## plugins Section

Plugins to execute during build.

### plugins

**Type:** array
**Default:** `[]`

List of plugins with configuration.

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500

  - name: link-checker
    path: ./plugins/link-checker
    config:
      check_external: false
```

### plugins[].name

**Type:** string
**Required:** Yes

Unique plugin identifier (lowercase, hyphen-separated).

### plugins[].path

**Type:** string
**Required:** Yes

Path to plugin executable, relative to `book.yaml`.

### plugins[].config

**Type:** object
**Default:** `{}`

Arbitrary key-value configuration passed to plugin.

```yaml
plugins:
  - name: custom-plugin
    path: ./plugins/custom
    config:
      setting1: value1
      setting2: ["array", "of", "values"]
      setting3:
        nested: object
```

## Complete Example

```yaml
book:
  title: "Advanced Python Guide"
  subtitle: "From Basics to Expert Level"
  author: "Jane Doe"
  version: "3.2.1"
  language: "en"
  description: "Comprehensive guide to advanced Python programming"
  cover:
    image: "assets/cover.png"
    background: "#1a1a2e"

chapters:
  - title: "Introduction"
    file: "intro.md"

  - title: "Part 1: Foundations"
    file: "part1.md"
    sections:
      - title: "Chapter 1: Python Basics"
        file: "ch01-basics.md"
      - title: "Chapter 2: Object-Oriented Programming"
        file: "ch02-oop.md"
      - title: "Chapter 3: Functional Programming"
        file: "ch03-functional.md"

  - title: "Part 2: Advanced Topics"
    file: "part2.md"
    sections:
      - title: "Chapter 4: Metaprogramming"
        file: "ch04-meta.md"
      - title: "Chapter 5: Performance Optimization"
        file: "ch05-performance.md"
      - title: "Chapter 6: Concurrency"
        file: "ch06-concurrency.md"

  - title: "Appendix"
    file: "appendix.md"

style:
  theme: "elegant"
  page_size: "A4"
  font_family: "Georgia, serif"
  font_size: "12pt"
  code_theme: "monokai"
  line_height: 1.6
  margin:
    top: 25
    bottom: 25
    left: 20
    right: 20
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: ""
  custom_css: |
    h1 {
      color: #1a1a2e;
      page-break-after: avoid;
    }
    code {
      background: #f5f5f5;
      padding: 2px 4px;
    }

output:
  filename: "python-guide.pdf"
  formats:
    - pdf
    - html
    - epub
  toc: true
  toc_max_depth: 3
  cover: true
  header: true
  footer: true
  pdf_timeout: 180
  watermark: ""
  watermark_opacity: 0.1
  margin_top: "20mm"
  margin_bottom: "20mm"
  margin_left: "25mm"
  margin_right: "25mm"
  generate_bookmarks: true

plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 10000
```

## Minimal Example

For quick testing:

```yaml
book:
  title: "My Book"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"
```

All other fields use defaults.

## Using Environment Variables

Reference environment variables in YAML using `${VAR_NAME}`:

```yaml
book:
  author: "${AUTHOR_NAME}"
  version: "${VERSION}"

style:
  theme: "${THEME}"
```

Then set before building:

```bash
export AUTHOR_NAME="John Doe"
export VERSION="1.0.0"
export THEME="elegant"
mdpress build
```

## Validation

All configuration is validated when:
- Running `mdpress validate`
- Running `mdpress build`
- Running `mdpress serve`

Errors are reported with line numbers and suggestions for fixes.

For more information on specific areas, see:
- [template-variables.md](template-variables.md) for header/footer variables
- [cli-commands.md](cli-commands.md) for build command options
- [../best-practices/organizing-large-books.md](../best-practices/organizing-large-books.md) for chapter organization
