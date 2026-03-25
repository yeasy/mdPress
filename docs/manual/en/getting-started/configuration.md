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

- `book` for metadata and cover settings.
- `chapters` for the reading order. Nested `sections` are supported.
- `style` for theme, fonts, page size, margins, and header/footer templates.
- `output` for generated formats and PDF settings.
- `plugins` for external executables.

## Chapter Sources

- If `chapters` is present, mdPress uses those files.
- If `chapters` is empty or omitted, `SUMMARY.md` is parsed instead.
- Paths are relative to `book.yaml`.

## Notes

- `LANGS.md` enables multi-language builds when present.
- `GLOSSARY.md` is detected automatically.
- `mdpress init` can generate a starter `book.yaml` from existing Markdown files.
