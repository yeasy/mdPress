# Output Formats

`mdpress build` can generate one or more formats in a single run.

## Formats

| Format | What You Get | Notes |
| --- | --- | --- |
| `pdf` | `book.pdf` | Default PDF output via Chromium. Honors page size, margins, TOC, cover, headers, and footers. |
| `html` | `book.html` | Single self-contained HTML file. |
| `site` | `book_site/` | Multi-page static site with `index.html`, chapter pages, search, and sidebar navigation. |
| `epub` | `book.epub` | EPUB 3 package for e-readers. |
| `typst` | `book-typst.pdf` | Alternate PDF backend that requires Typst in `PATH`. |

## Building Multiple Formats

```bash
mdpress build --format pdf,html,epub
mdpress build --format all
```

`all` expands to `pdf,html,site,epub,typst`, building all 5 formats.

## Output Path

`--output` sets the base path. mdPress adds the format suffix:

- `book.pdf`
- `book.html`
- `book.epub`
- `book_site/`
- `book-typst.pdf` (typst format)

If you pass a directory, the files are written inside that directory.
