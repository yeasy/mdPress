# Template Variables for Headers and Footers

Template variables are placeholders you can use in page headers and footers. They are replaced with actual values during PDF rendering.

## Variable Reference

### Book Information

Variables for book metadata:

#### {{.Book.Title}}

The book title from `book.yaml`.

```yaml
style:
  header:
    left: "{{.Book.Title}}"
```

Output: "Advanced Python Guide"

#### {{.Book.Subtitle}}

The book subtitle from `book.yaml`.

```yaml
style:
  footer:
    center: "{{.Book.Subtitle}}"
```

Output: "From Basics to Expert Level"

#### {{.Book.Author}}

The book author from `book.yaml`.

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
```

Output: "© 2024 Jane Doe"

#### {{.Book.Version}}

The book version from `book.yaml`.

```yaml
style:
  header:
    right: "v{{.Book.Version}}"
```

Output: "v3.2.1"

### Chapter Information

Variables for current chapter context:

#### {{.Chapter.Title}}

The title of the current chapter.

```yaml
style:
  header:
    right: "{{.Chapter.Title}}"
```

Output: "Chapter 3: Advanced Techniques"

#### {{.Chapter.Number}}

The chapter number (1-based index).

```yaml
style:
  header:
    center: "Chapter {{.Chapter.Number}}"
```

Output: "Chapter 5"

#### {{.Chapter.File}}

The filename of the current chapter.

```yaml
style:
  footer:
    right: "{{.Chapter.File}}"
```

Output: "ch05-concurrency.md"

### Page Information

Variables for current page context:

#### {{.PageNum}}

The current page number.

```yaml
style:
  footer:
    center: "{{.PageNum}}"
```

Output: "47"

#### {{.TotalPages}}

The total number of pages in the PDF.

```yaml
style:
  footer:
    right: "{{.PageNum}} / {{.TotalPages}}"
```

Output: "47 / 250"

### Date and Time

Variables for dates and times:

#### {{.Date}}

The build date in default format (YYYY-MM-DD).

```yaml
style:
  footer:
    right: "{{.Date}}"
```

Output: "2024-03-23"

#### {{.DateTime}}

The build date and time (YYYY-MM-DD HH:MM:SS).

```yaml
style:
  footer:
    right: "{{.DateTime}}"
```

Output: "2024-03-23 14:30:45"

#### {{.Year}}

The current year.

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
```

Output: "© 2024 Jane Doe"

#### {{.Month}}

The current month number (01-12).

```yaml
style:
  header:
    center: "{{.Month}}/{{.Day}}/{{.Year}}"
```

Output: "03/23/2024"

#### {{.Day}}

The current day of month (01-31).

## Real-World Examples

### Professional Technical Documentation

```yaml
style:
  header:
    left: "{{.Book.Title}} (v{{.Book.Version}})"
    center: "{{.Chapter.Title}}"
    right: ""
  footer:
    left: "{{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "© {{.Year}} Acme Corp"
```

Output:
```
Advanced Python Guide (v3.2.1)     Chapter 3: Functions
Acme Corporation                      23                    © 2024 Acme Corp
```

### Academic Paper

```yaml
style:
  header:
    left: "{{.Book.Author}}"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    left: "Draft"
    center: "{{.PageNum}}"
    right: ""
```

Output:
```
Jane Doe        Advanced Python Guide        2024-03-23
Draft                               5
```

### Book Publication

```yaml
style:
  header:
    left: "{{.Chapter.Number}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: ""
```

Output:
```
5                                  Chapter 3: Functions
                               23
```

### Internal Document

```yaml
style:
  header:
    left: "CONFIDENTIAL"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    left: "Document Version {{.Book.Version}}"
    center: "Page {{.PageNum}} of {{.TotalPages}}"
    right: "INTERNAL USE ONLY"
```

Output:
```
CONFIDENTIAL    Advanced Python Guide    2024-03-23
Document Version 3.2.1    Page 23 of 250    INTERNAL USE ONLY
```

### Minimal Header/Footer

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: ""
```

Output:
```
Advanced Python Guide               Chapter 3: Functions
                               23
```

## Formatting Examples

### Date Formats

The default `{{.Date}}` format is YYYY-MM-DD. For other formats, use CSS or custom styling:

```yaml
style:
  footer:
    right: "{{.Date}}"
    # Output: 2024-03-23

  footer:
    right: "Build: {{.DateTime}}"
    # Output: Build: 2024-03-23 14:30:45
```

### Version Formatting

```yaml
style:
  header:
    right: "v{{.Book.Version}}"
    # Output: v3.2.1

  header:
    right: "Version {{.Book.Version}} ({{.Date}})"
    # Output: Version 3.2.1 (2024-03-23)
```

### Page Number Formatting

```yaml
style:
  footer:
    center: "{{.PageNum}}"
    # Output: 23

  footer:
    center: "{{.PageNum}} / {{.TotalPages}}"
    # Output: 23 / 250

  footer:
    center: "Page {{.PageNum}}"
    # Output: Page 23
```

### Conditional Text (Using Text Only)

Since variables only support simple substitution, use fixed text combined with variables:

```yaml
style:
  header:
    left: "Draft - {{.Date}}"
    # Output: Draft - 2024-03-23

  footer:
    left: "© {{.Year}} {{.Book.Author}} - All Rights Reserved"
    # Output: © 2024 Jane Doe - All Rights Reserved
```

## Common Header/Footer Combinations

### Combination 1: Title + Chapter + Page

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "{{.PageNum}}"
```

Useful for: General documentation, technical guides

### Combination 2: Simple Page Numbers

```yaml
style:
  header:
    left: ""
    center: ""
    right: ""
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: ""
```

Useful for: Clean, minimalist layouts

### Combination 3: Formal Document

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "v{{.Book.Version}}"
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "CONFIDENTIAL"
```

Useful for: Business, legal, formal documents

### Combination 4: Chapter-Based

```yaml
style:
  header:
    left: "Chapter {{.Chapter.Number}}"
    center: "{{.Chapter.Title}}"
    right: ""
  footer:
    left: "{{.Date}}"
    center: "{{.PageNum}}"
    right: "{{.Book.Version}}"
```

Useful for: Textbooks, multi-chapter books

### Combination 5: Minimal with Footer

```yaml
style:
  header:
    left: ""
    center: ""
    right: ""
  footer:
    left: "{{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "{{.Date}}"
```

Useful for: Short documents, reports

## Styling Headers and Footers

While variables provide the content, use CSS in `custom_css` for styling:

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "{{.Chapter.Title}}"
    right: "{{.PageNum}}"

  custom_css: |
    @page {
      @top-left {
        font-size: 10pt;
        font-weight: bold;
      }
      @top-center {
        font-size: 11pt;
        font-style: italic;
      }
      @bottom-center {
        font-size: 10pt;
        color: #666;
      }
    }
```

## Limitations

### Variable Support

Not all variables work in all contexts:
- Headers and footers: All variables supported
- Custom CSS: Use CSS variables instead (see below)
- Markdown content: Use literal text instead

### Unsupported Variables

These are not currently supported:
- {{.SectionNumber}} - Section numbering
- {{.WordCount}} - Word count of chapter
- {{.BuildTime}} - Total build duration
- {{.Environment}} - Environment variables

For these, use external tools or pre/post-processing.

### Plain Text Only

Variables output plain text only. For rich formatting:
- Use CSS styling (see above)
- Use multiple header/footer sections with different content
- Combine variables with fixed text

## Examples with Different Themes

### Technical Theme

```yaml
style:
  theme: "technical"
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @top-left {
        color: #1a1a2e;
        font-weight: bold;
      }
      @bottom-center {
        color: #666;
      }
    }
```

### Elegant Theme

```yaml
style:
  theme: "elegant"
  header:
    left: "{{.Book.Author}}"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @top-left, @top-right {
        font-style: italic;
        color: #333;
      }
    }
```

### Minimal Theme

```yaml
style:
  theme: "minimal"
  header:
    left: ""
    center: ""
    right: ""
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @bottom-center {
        font-size: 9pt;
      }
    }
```

## Tips and Best Practices

1. **Keep headers/footers brief**: Long text may overflow or break layout
2. **Use consistent variables across documents**: Aids recognition
3. **Include page numbers**: Always helpful for reference
4. **Add version/date for drafts**: Helps track document evolution
5. **Test output**: Preview in `mdpress serve` before final build
6. **Consider first/last pages**: Customize first page separately if needed

## Default Values

When variables are not available (e.g., no chapter title):
- `{{.Chapter.Title}}` → empty string
- `{{.PageNum}}` → 1 (first page)
- `{{.Date}}` → build date

## Advanced: CSS Variables

For more complex styling beyond simple text, use CSS variables:

```yaml
style:
  custom_css: |
    :root {
      --header-color: #1a1a2e;
      --footer-color: #666;
      --page-width: 210mm;
    }

    @page {
      @top-left {
        color: var(--header-color);
        font-weight: bold;
      }
      @bottom-center {
        color: var(--footer-color);
      }
    }
```

Combined with template variables for dynamic, stylized headers and footers.
