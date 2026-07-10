# Output Formats

`mdpress build` can generate one or more formats in a single run.

## Formats

| Format | What You Get | Notes |
| --- | --- | --- |
| `pdf` | `book.pdf` | Default PDF output via Chromium. Honors page size, margins, TOC, cover, headers, and footers. |
| `html` | `book.html` | Single self-contained HTML file. |
| `site` | `_book/` | Multi-page static site with `index.html`, chapter pages, search, and sidebar navigation. |
| `epub` | `book.epub` | EPUB 3 package for e-readers. |
| `typst` | `book-typst.pdf` | Alternate PDF backend that requires Typst in `PATH`. |

## Building Multiple Formats

```bash
mdpress build --format pdf,html,epub
mdpress build --format all
```

`all` expands to `pdf,html,site,epub,typst`, building all 5 formats.

## Output Path

For file formats, `--output` sets the base path and mdPress adds the format suffix:

- `book.pdf`
- `book.html`
- `book.epub`
- `book-typst.pdf` (typst format)

If you pass a directory — an existing one, or any path with a trailing slash — the files are written inside that directory, and site pages are written directly into it (in place, without pruning existing files there). Any other path is treated as a filename base: `--output release/manual.html` produces `release/manual.html`, `release/manual.pdf`, and the site at `release/manual_site/`.

The `site` format produces a directory rather than a single file. By default it is
written to `_book/` under the project directory — the same location `mdpress serve`
uses, and the directory the deployment examples assume. The default site build is
staged in a temporary directory and atomically swapped into `_book/`, so pages left
over from renamed or removed chapters are pruned; a non-empty target that does not
look generated (no `index.html`/`search-index.json`) is refused as a safety measure.

**Multi-language exception:** projects with a `LANGS.md` keep their per-language
`<lang>_site/` directories (plus a language landing page) instead of a single `_book/`.

Remote GitHub builds (e.g. `mdpress build https://github.com/user/repo`) without
`--output` write their outputs to the current working directory: files as
`./<Title>.pdf` etc., and the site as `./_book/`.

After a successful build, one `✓ Generated <format> → <path>` line is printed per
format (even with `--quiet`).

## Site Options

The `site` output uses relative navigation links throughout, so it works on GitHub
Pages project sites (`https://user.github.io/repo/`) and even when opened directly
via `file://`. A `404.html` page is generated automatically.

Two `output` config fields extend the site:

- `site_url`: the public base URL of the deployed site. When set, a spec-compliant
  `sitemap.xml` (absolute `<loc>`, `<lastmod>`) is generated; without it no sitemap
  is written.
- `edit_base`: a base URL such as `https://github.com/user/repo/edit/main/`. When
  set, every chapter page gets an "Edit this page" link.

## PDF Options

By default the PDF footer is a centered page number and there is no header; customize
both via `style.header`/`style.footer` (see the
[configuration reference](../reference/configuration.md)). The new `output.tagged_pdf`
option defaults to `true` (accessible tagged PDF); set it to `false` for noticeably
smaller files.
