# Headers and Footers

Headers and footers add consistent page information to every page of your PDF output. They are configured in `book.yaml` and support a small set of placeholder tokens.

**Note:** Headers and footers are a PDF-only feature (Chromium backend). They do not appear in HTML, site, or EPUB output.

## Defaults

Out of the box, PDFs get:

- **Footer**: a centered page number in subtle small print
- **Header**: none

You only need configuration to change this.

## Basic Configuration

Define headers and footers in the `style` section of `book.yaml`. Each has three optional cells — `left`, `center`, and `right`:

```yaml
style:
  header:
    left: "{title}"
    center: ""
    right: "{page}"

  footer:
    center: "{page} / {pages}"
```

Omit cells you don't need. If every cell of a header/footer is empty, it is not rendered.

## Switching Headers and Footers Off

The `output.header` and `output.footer` booleans act as on/off switches:

```yaml
output:
  header: false   # never render a header
  footer: false   # never render a footer (also disables the default page number)
```

Both default to `true`.

## The Cover Page

A cover is artwork printed to the edge of the sheet, so mdPress prints no header and no footer on it. Nothing needs configuring: with `output.cover: true` (the default), page one comes out clean.

The cover still counts as page one, so the first page of the table of contents reads `2`, and the page numbers printed in the table of contents count it too. Set `output.cover: false` if you want the folio to start at the table of contents instead.

## Supported Tokens

| Token | Replaced with |
| --- | --- |
| `{page}` | Current page number |
| `{pages}` | Total number of pages |
| `{title}` | The book title from `book.yaml` |
| `{author}` | The author from `book.author` |

Legacy Go-template-style tokens from older scaffolded configs are also accepted: `{{.PageNum}}` (= `{page}`), `{{.TotalPages}}` (= `{pages}`), `{{.Book.Title}}` (= `{title}`), and `{{.Book.Author}}` (= `{author}`). `{{.Chapter.Title}}` is accepted for compatibility but expands to nothing — Chrome print templates have no per-chapter context.

Everything else is treated as literal text (HTML-escaped for safety), so you can freely combine tokens with fixed text:

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "Page {page} of {pages}"
    right: "{title}"
```

## Examples

### Page X of Y

```yaml
style:
  footer:
    center: "Page {page} of {pages}"
```

### Title in the Header, Page Number in the Footer

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "{page}"
```

### Draft Marker

```yaml
style:
  footer:
    center: "DRAFT — {title} — {page}"
```

For a diagonal DRAFT overlay across the page instead, use `output.watermark`.

## Styling

Header and footer text uses a fixed, deliberately subtle style (9px, muted gray, matching the system font stack). The content is customizable; the styling is not. If headers or footers feel cramped, increase the page margins via `output.margin_top` / `output.margin_bottom` (e.g. `"20mm"`).

## Troubleshooting

### Tokens Appear Literally in the PDF

1. Check the spelling — supported tokens are exactly `{page}`, `{pages}`, `{title}`, and `{author}` (case-sensitive).
2. Check that you are generating PDF output; headers/footers do not apply to other formats.

### No Header Appears

The default header is empty. A header renders only when you set `style.header` cells to non-empty values (and `output.header` is not `false`).

### Header/Footer Overlaps Content

Increase the PDF margins:

```yaml
output:
  margin_top: "20mm"
  margin_bottom: "20mm"
```

## Complete Example

```yaml
book:
  title: "mdPress Documentation"
  author: "mdPress Team"

chapters:
  - title: Introduction
    file: chapters/01-introduction.md
  - title: Installation
    file: chapters/02-installation.md

style:
  theme: technical
  header:
    left: "{title}"
    right: "{page}"
  footer:
    center: "Page {page} of {pages}"

output:
  header: true
  footer: true
  margin_top: "18mm"
  margin_bottom: "18mm"
```

See [Custom CSS](./custom-css.md) and [Built-in Themes](./builtin-themes.md) for more styling options, and the [template token reference](../reference/template-variables.md) for the full token list.
