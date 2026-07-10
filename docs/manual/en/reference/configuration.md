# Configuration Reference

This page documents `book.yaml`. Paths are relative to the config file. If `chapters` is empty, `SUMMARY.md` is used instead.

## Defaults

- `book.title`: `Untitled Book`
- `book.language`: `zh-CN`
- `style.theme`: `technical`
- `style.page_size`: `A4`
- `style.code_theme`: empty — inherits the theme's code style (`github` for technical/elegant, `bw` for minimal); set an explicit value to override
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
| `cover.background` | string | Background color when no cover image is used. Light backgrounds (including `white` and named/`rgb()` colors) automatically get dark cover text. |

When neither `cover.image` nor `cover.background` is set, the default cover follows the theme: deep navy for `technical`, deep warm brown for `elegant`, and light with dark ink for `minimal`.

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

- `theme`: a built-in theme (`technical`, `elegant`, `minimal`), the name of a custom theme defined at `themes/<name>.yaml` (or `.yml`) in the project, or a path to a YAML theme file (e.g. `theme: mytheme.yaml`, relative to `book.yaml`). A `themes/<name>.yaml` file also overrides the built-in theme of the same name. See the theme file schema in [Built-in Themes](../themes/builtin-themes.md).
- `page_size`: `A4`, `A5`, `Letter`, `Legal`, or `B5`.
- `font_family`, `font_size`, `code_theme`, and `line_height` control typography. Leaving `code_theme` empty inherits the theme's code style; code highlighting automatically gets a dark-mode counterpart (e.g. `github` → `github-dark`) in site/HTML output.
- `margin` sets top, bottom, left, and right margins in millimeters.
- `header` and `footer` each take `left`/`center`/`right` strings for the PDF page header/footer. Supported tokens: `{page}`, `{pages}`, `{title}` (the legacy `{{.PageNum}}`, `{{.TotalPages}}`, `{{.Book.Title}}` forms are also accepted). By default the footer is a centered page number and there is no header; the `output.header`/`output.footer` booleans switch them off/on.
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
| `site_url` | string | Public base URL of the deployed site (e.g. `https://user.github.io/repo`). When set, a spec-compliant `sitemap.xml` with absolute `<loc>` and `<lastmod>` entries is generated for the `site` format; without it no sitemap is written. |
| `edit_base` | string | Base URL for "Edit this page" links (e.g. `https://github.com/user/repo/edit/main/`). When set, each site chapter gets an edit link. |
| `tagged_pdf` | bool | Generate an accessible tagged PDF (default `true`). Set `false` for noticeably smaller files at the cost of accessibility tagging. |

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

> **Security note:** plugins are executables that run on your machine during build and serve (including at probe time). Only build/serve projects you trust. Since v0.7.12, remote sources refuse to run plugins unless `--allow-plugins` is passed. See the [plugin overview](../plugins/overview.md).

## Discovery Note

`book.json` is supported for GitBook compatibility when `book.yaml` is absent, but this page focuses on `book.yaml`.
