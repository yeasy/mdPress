# Configuration

mdPress reads `book.yaml` in the project root first. If `book.yaml` is missing, discovery can fall back to `book.json` for GitBook compatibility, then `SUMMARY.md`, and finally auto-discovered Markdown files.

## Minimal Example

```yaml
book:
  title: My Book
  author: Jane Doe
  language: en-US

chapters:
  - title: Introduction
    file: README.md
  - title: Getting Started
    file: chapters/getting-started.md

style:
  theme: technical
  page_size: A4
  custom_css: custom.css

output:
  filename: my-book.pdf
  formats: [pdf, html]
```

## What To Configure

- `book` for metadata, cover settings, and site branding (`favicon`, `logo`, `copyright`).
- `chapters` for the reading order. Nested `sections` are supported, and `section` labels a sidebar group.
- `style` for theme, fonts, page size, margins, and header/footer templates.
- `output` for generated formats, PDF settings, and site options (`site_url`, `edit_base`, `footer_html`, `show_theme_badge`).
- `markdown` for parsing behavior (`allow_html`).
- `variables` for your own `{{ key }}` template values.
- `plugins` for external executables.

Run `mdpress config show` to see the resolved result — every default applied, plus which
config file was loaded and where the theme came from. See the
[Configuration Reference](../reference/configuration.md) for every key.

## Chapter Sources

- If `chapters` is present, mdPress uses those files.
- If `chapters` is empty or omitted, `SUMMARY.md` is parsed instead.
- Paths are relative to `book.yaml`.

## Notes

- `LANGS.md` enables multi-language builds when present.
- `GLOSSARY.md` is detected automatically.
- A `static/` directory is copied verbatim into the site root — that is where `CNAME`, `.nojekyll` and similar files belong.
- `mdpress init` can generate a starter `book.yaml` from existing Markdown files.
- `mdpress validate --strict` fails on warnings such as an unknown config key, which makes it usable as a CI gate.
