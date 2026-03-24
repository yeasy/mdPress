# Headers and Footers

Headers and footers allow you to add consistent branding, page information, and navigation elements to every page in your PDF output. They're configured in `book.yaml` and support dynamic template variables.

**Note:** Headers and footers are a PDF-only feature. They do not appear in HTML output.

## Basic Configuration

Define headers and footers in the `style` section of `book.yaml`:

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}}"
    center: ""
    right: "{{.Date}}"
```

Each section (left, center, right) is optional. Omit sections you don't need.

## Template Variables

Headers and footers support dynamic content through template variables. These are replaced with actual values during PDF generation.

### Book Variables

Access book-level information from `book.yaml`:

- `{{.Book.Title}}` - The book title
- `{{.Book.Author}}` - The book author
- `{{.Book.Subtitle}}` - The book subtitle (if defined)
- `{{.Book.Version}}` - The book version (if defined)
- `{{.Book.Description}}` - The book description

Example:

```yaml
style:
  header:
    left: "{{.Book.Title}} v{{.Book.Version}}"
    right: "{{.Book.Author}}"
```

### Chapter Variables

Access the current chapter information:

- `{{.Chapter.Title}}` - The title of the current chapter
- `{{.Chapter.Number}}` - The chapter number
- `{{.Chapter.File}}` - The source filename of the chapter

Example:

```yaml
style:
  header:
    left: "{{.Chapter.Title}}"
    center: ""
    right: "Page {{.PageNum}}"
```

### Page Variables

Dynamic page information:

- `{{.PageNum}}` - Current page number (integer)
- `{{.PageTotal}}` - Total number of pages
- `{{.TotalPages}}` - Alias for `.PageTotal`

Create "page X of Y" footers:

```yaml
style:
  footer:
    right: "Page {{.PageNum}} of {{.PageTotal}}"
```

### Date Variables

Date and time information:

- `{{.Date}}` - Current date in default format (YYYY-MM-DD)
- `{{.DateTime}}` - Current date and time (YYYY-MM-DD HH:MM:SS)
- `{{.Year}}` - Current year
- `{{.Month}}` - Current month (1-12)
- `{{.Day}}` - Current day of month

Example:

```yaml
style:
  footer:
    left: "Generated on {{.Date}}"
    right: "{{.Year}}"
```

### Section Variables

For documentation organized into sections:

- `{{.Section}}` - Name of current section (if available)
- `{{.Subsection}}` - Name of current subsection (if available)

## Complete Configuration Examples

### Technical Documentation

```yaml
style:
  header:
    left: "{{.Book.Title}} - {{.Chapter.Title}}"
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}}"
    center: "Confidential"
    right: "{{.Date}}"
```

This creates headers showing the document and chapter title, with page numbers on the right. Footers include author, a confidentiality notice, and the date.

### User Manual

```yaml
style:
  header:
    left: "{{.Chapter.Title}}"
    right: "Page {{.PageNum}} of {{.PageTotal}}"

  footer:
    center: "{{.Book.Title}} v{{.Book.Version}}"
```

Displays the current chapter in the header with page numbering, and the book title with version in the footer.

### Corporate Report

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Book.Version}}"

  footer:
    left: "{{.Book.Author}}"
    center: ""
    right: "{{.Year}}"
```

Corporate-style formatting with title and version in the header, author and year in the footer.

### Academic Document

```yaml
style:
  header:
    left: "{{.Book.Author}}"
    right: "{{.Date}}"

  footer:
    left: "{{.Book.Title}}"
    center: "{{.PageNum}}"
    right: ""
```

Academic format with author and date in header, title and centered page numbers in footer.

## Styling Headers and Footers

Headers and footers use default styling, but you can customize their appearance through PDF generation options. The font size, font family, and colors inherit from your theme.

### Controlling Appearance

```yaml
pdf:
  header_height: 0.5in
  footer_height: 0.5in
  header_font_size: 10
  footer_font_size: 10
  margins:
    top: 1in
    bottom: 1in
```

Adjust margins to provide enough space for headers and footers:

```yaml
pdf:
  margins:
    top: 1.2in        # Extra space for header
    bottom: 1.2in     # Extra space for footer
    left: 1in
    right: 1in
```

## Advanced Patterns

### Alternating Headers

Show different headers on odd and even pages (useful for books):

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
```

When printed double-sided, the left page shows the book title, right page shows the chapter title.

### Section Dividers

Use chapter numbers in headers to indicate structure:

```yaml
style:
  header:
    left: "Chapter {{.Chapter.Number}}: {{.Chapter.Title}}"
    right: "{{.PageNum}}"
```

### Branding Information

Include company information in footers:

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "Internal Use Only"
    right: "{{.Date}}"
```

### Version Control

Track document versions:

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "Version {{.Book.Version}}"
    right: "Generated {{.DateTime}}"
```

This is useful for tracking when PDFs were generated.

## Conditional Content

While template variables don't support full conditionals, you can work around this:

Use a separator when both variables are needed:

```yaml
style:
  footer:
    left: "{{.Book.Author}} — {{.Book.Title}}"
```

Or create multiple variations of the footer for different purposes:

```yaml
# For drafts
style:
  footer:
    center: "DRAFT — {{.Date}}"

# For final release (comment out the draft version)
# style:
#   footer:
#     center: "Final Release"
```

## Troubleshooting

### Variables Not Replaced

If variables like `{{.Book.Title}}` appear literally in the PDF:

1. Check that you're generating PDF (headers/footers are PDF-only)
2. Verify the variable name matches exactly (case-sensitive)
3. Ensure the variable is defined in your `book.yaml`

### Headers/Footers Overlapping Content

If headers or footers overlap the main content:

1. Increase margin values in your PDF configuration:
   ```yaml
   pdf:
     margins:
       top: 1.2in
       bottom: 1.2in
   ```

2. Reduce header/footer height if configured:
   ```yaml
   pdf:
     header_height: 0.4in
     footer_height: 0.4in
   ```

### Missing Variables in Chapter

If `{{.Chapter.Title}}` is blank:

1. Ensure each markdown file has an H1 heading
2. Verify the chapter is properly listed in `book.yaml` under `chapters`
3. Check that the markdown file isn't empty

## Complete Example

Here's a complete `book.yaml` with headers and footers configured:

```yaml
book:
  title: "mdPress Documentation"
  author: "mdPress Team"
  version: "1.0"
  description: "Complete guide to mdPress"

chapters:
  - chapters/01-introduction.md
  - chapters/02-installation.md
  - chapters/03-usage.md

style:
  theme: technical

  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}} — v{{.Book.Version}}"
    center: ""
    right: "{{.Date}}"

pdf:
  output: build/mdpress-guide.pdf
  margins:
    top: 1.2in
    bottom: 1.2in
    left: 1in
    right: 1in
```

When you generate the PDF, every page will include:
- **Header**: "mdPress Documentation" on the left, page numbers on the right
- **Footer**: "mdPress Team — v1.0" on the left, today's date on the right

See [Custom CSS](./custom-css.md) and [Built-in Themes](./builtin-themes.md) for more styling options.
