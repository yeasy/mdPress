# Template Tokens for Headers and Footers

Header and footer strings in `style.header` / `style.footer` support a small set of placeholder tokens. They are replaced during PDF rendering; everything else in the string is treated as literal text (HTML-escaped for safety).

## Token Reference

| Token | Replaced with | Example output |
| --- | --- | --- |
| `{page}` | Current page number | `47` |
| `{pages}` | Total number of pages | `250` |
| `{title}` | The book title from `book.yaml` | `Advanced Python Guide` |

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "Page {page} of {pages}"
```

## Legacy Tokens

Configs scaffolded by older mdPress versions used Go-template-style tokens. They are still accepted:

| Legacy token | Equivalent |
| --- | --- |
| `{{.PageNum}}` | `{page}` |
| `{{.TotalPages}}` | `{pages}` |
| `{{.Book.Title}}` | `{title}` |
| `{{.Chapter.Title}}` | expands to nothing (no per-chapter context in Chrome print templates) |

Prefer the short `{page}` / `{pages}` / `{title}` forms in new configs.

## Combining Tokens with Text

Tokens can be freely mixed with fixed text:

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "{page} / {pages}"
    right: "{title}"
```

## Common Combinations

### Simple page numbers (matches the built-in default)

```yaml
style:
  footer:
    center: "{page}"
```

### Title + page count

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "Page {page} of {pages}"
```

### Formal document

```yaml
style:
  header:
    left: "{title}"
    right: "CONFIDENTIAL"
  footer:
    left: "© 2026 Acme Corp"
    center: "{page}"
```

## Limitations

- Tokens work only in PDF headers and footers; other formats do not render them.
- Output is plain text — no Markdown or HTML markup inside header/footer strings (markup is escaped).
- There are no tokens for author, dates, versions, or chapter titles. Write such values as literal text, e.g. `left: "© 2026 Jane Doe — v3.2.1"`.
- Header/footer typography is fixed (small, muted); only the content is configurable. Use `output.margin_top` / `output.margin_bottom` if you need more breathing room.

See [Headers and Footers](../themes/headers-footers.md) for configuration details.
