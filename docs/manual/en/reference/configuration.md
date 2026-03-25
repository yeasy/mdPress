# Configuration Reference

This page documents `book.yaml`. Paths are relative to the config file. If `chapters` is empty, `SUMMARY.md` is used instead.

## Defaults

- `book.title`: `Untitled Book`
- `book.language`: `zh-CN`
- `style.theme`: `technical`
- `style.page_size`: `A4`
- `style.code_theme`: `github`
- `output.filename`: `output.pdf` by default, but builds use the book title or directory name unless you override it
- `output.toc`, `output.cover`, `output.header`, `output.footer`: enabled
- `output.toc_max_depth`: `2`
- `output.generate_bookmarks`: `true`

## `book`

| Field | Type | Notes |
| --- | --- | --- |
| `title` | string | Required. |
| `subtitle` | string | Optional. |
| `author` | string | Optional. |
| `version` | string | Optional. |
| `language` | string | BCP 47 tag such as `en-US` or `zh-CN`. |
| `description` | string | Metadata for EPUB and HTML. |
| `cover.image` | string | Path to the cover image. |
| `cover.background` | string | Background color when no cover image is used. |

## `chapters`

`chapters` is an ordered list of chapter definitions:

```yaml
chapters:
  - title: Introduction
    file: README.md
  - title: Part One
    file: part-1.md
    sections:
      - title: Setup
        file: part-1/setup.md
```

- `title` is what readers see in navigation.
- `file` is required and must point to a Markdown file.
- `sections` nests more chapters under the current item.
- `SUMMARY.md` uses Markdown links and indentation to express the same tree.

## `style`

- `theme`: `technical`, `elegant`, or `minimal`.
- `page_size`: `A4`, `A5`, `Letter`, `Legal`, or `B5`.
- `font_family`, `font_size`, `code_theme`, and `line_height` control typography.
- `margin` sets top, bottom, left, and right margins in millimeters.
- `header` and `footer` accept template strings such as `{{.Book.Title}}` and `{{.PageNum}}`.
- `custom_css` points to an extra CSS file.

## `output`

| Field | Type | Notes |
| --- | --- | --- |
| `filename` | string | Base filename for generated output. |
| `formats` | list[string] | Supported values: `pdf`, `html`, `site`, `epub`, `typst`. |
| `toc` | bool | Include a table of contents. |
| `toc_max_depth` | int | Range `1` to `6`. |
| `cover` | bool | Include the cover page. |
| `header` | bool | Enable the page header. |
| `footer` | bool | Enable the page footer. |
| `pdf_timeout` | int | PDF generation timeout in seconds. |
| `watermark` | string | Watermark text. |
| `watermark_opacity` | float | Opacity from `0.0` to `1.0`. |
| `margin_top` | string | PDF or Typst margin override such as `20mm`. |
| `margin_bottom` | string | PDF or Typst margin override such as `20mm`. |
| `margin_left` | string | PDF or Typst margin override such as `20mm`. |
| `margin_right` | string | PDF or Typst margin override such as `20mm`. |
| `generate_bookmarks` | bool | Generate PDF bookmarks. |

`style.margin` and the `output.margin_*` fields are separate. Use `style.margin` for general layout settings and `output.margin_*` for PDF or Typst overrides.

## `plugins`

Plugins are external executables that run in declaration order.

```yaml
plugins:
  - name: word-count
    path: ./examples/plugins/word-count
    config:
      warn_threshold: 500
```

- `name` is the plugin identifier.
- `path` points to the executable, relative to `book.yaml`.
- `config` is passed to the plugin as JSON data.

## Discovery Note

`book.json` is supported for GitBook compatibility when `book.yaml` is absent, but this page focuses on `book.yaml`.
