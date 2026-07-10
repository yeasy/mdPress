# `mdpress build`

[中文说明](build_zh.md)

## Purpose

Build publishable outputs from a local directory or a GitHub repository. `build` supports `pdf`, `html`, `site`, `epub`, and `typst`, and it can generate multiple formats in a single run.

## Syntax

```bash
mdpress build [source] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[source]` | No | Input source. It can be omitted, a local directory, or a GitHub repository URL. If omitted, the current directory is used. |

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-f, --format <list>` | config value or `pdf` | Output formats, comma-separated (e.g., `pdf,html,epub`) or `all` for all formats. |
| `--branch <name>` | repository default branch | GitHub branch name. Only applies to remote repository inputs. |
| `--subdir <path>` | repository root | Subdirectory inside the repository. Only applies to remote repository inputs. |
| `-o, --output <path>` | `output.filename` / `_book/` for site | Output file path, output directory, or filename base. |
| `--summary <path>` | auto-detect | Explicit path to a `SUMMARY.md` file. Overrides chapters from `book.yaml` or auto-discovery. |
| `--allow-plugins` | off | Execute plugins declared by a remote project's `book.yaml` (arbitrary code). Local sources always run plugins. |
| `--config <path>` | `book.yaml` | Config file path for local builds. |
| `-v, --verbose` | off | Print detailed logs and warning-by-warning output. |
| `-q, --quiet` | off | Print errors only. |
| `--cache-dir <path>` | system default | Custom cache directory for build artifacts. |
| `--no-cache` | off | Disable all build caching. |

## Behavior

### Input Resolution

`build` loads project structure in this order:

1. `book.yaml`
2. `book.json` (GitBook compatibility)
3. `SUMMARY.md`
4. Automatic `.md` file discovery

If `[source]` is omitted, the current directory is used.

If the current directory is a large code repository instead of a dedicated docs directory, avoid relying on auto-discovery from the repository root. A safer approach is:

```bash
mdpress build ./docs --format html
mdpress build --config ./docs/book.yaml ./docs --format pdf,html
```

### `--format`

- When `--format` is provided, the CLI value overrides `output.formats` in config.
- When `--format` is omitted, `output.formats` is used first.
- If neither is set, the default output is `pdf`.
- `--format all` expands to `pdf,html,site,epub,typst`.

### `--output`

Without `--output`, file formats are written into the project directory (named after `output.filename` or the book title), and `site` is written to `_book/` under the project directory — the same location `mdpress serve` uses. Multi-language projects (with `LANGS.md`) keep their per-language `<lang>_site/` directories.

With `--output`, there are two patterns:

1. Pass a directory — an existing directory, or any path with a trailing slash

```bash
mdpress build --output ./dist
mdpress build --output ./dist/
```

File formats are written into that directory (e.g. `./dist/<Title>.pdf`, `./dist/<Title>.html`). The `site` pages are written directly into that directory (in place — stale files already there are not pruned).

2. Pass a file path or filename base

```bash
mdpress build --format pdf,html,site --output ./release/manual.pdf
```

A path that does not resolve to a directory is treated as a base path:

- `pdf` becomes `./release/manual.pdf`
- `html` becomes `./release/manual.html`
- `site` becomes `./release/manual_site/`

### Site output safety

The site is built into a temporary directory and then atomically swapped into place, so stale pages from renamed or removed chapters are pruned. As a safeguard, a non-empty target directory that does not look generated (no `index.html`/`search-index.json`) is refused instead of being overwritten. If a legacy `<name>_site/` directory from an older mdPress version sits next to a default `_book/` build, a hint is logged.

### Result summary

After every successful build, the CLI prints one line per format, for example:

```
  ✓ Generated pdf   → my-book.pdf
  ✓ Generated site  → _book/
```

These lines are a build result rather than progress output, so they are printed even with `--quiet`.

## Examples

```bash
mdpress build
mdpress build --format html
mdpress build --format pdf,html,epub
mdpress build --format all --output ./dist
mdpress build --format site --output ./dist/book
mdpress build /path/to/book --format html
mdpress build https://github.com/yeasy/agentic_ai_guide
mdpress build https://github.com/yeasy/agentic_ai_guide --branch main --subdir docs
mdpress build --config ./configs/book.yaml --verbose
```

## Outputs

| Format | Result |
| --- | --- |
| `pdf` | A single PDF file |
| `html` | A self-contained single-page HTML file |
| `site` | A multi-page static site directory |
| `epub` | A single ePub file |
| `typst` | A PDF file generated via the Typst CLI (Chromium-free) |

## Notes

- PDF generation requires Chrome or Chromium. If neither is available, PDF builds will fail.
- Typst output requires the Typst CLI. If it is not installed, `--format typst` builds will fail.
- During the build, mdpress checks heading numbering, Markdown links, and Mermaid diagnostics. Many of these are warnings and do not necessarily stop the build.
- When `LANGS.md` exists at the project root, `build` generates one output set per language and also creates a language landing page.
- For remote GitHub inputs, the current implementation prefers the remote repository's `book.yaml`. A local `--config` path does not override the remote project's config location.
- For remote GitHub inputs without `--output`, outputs are written to the current working directory (files as `./<Title>.pdf` etc., site as `./_book/`).
- For remote GitHub inputs, plugins declared in the remote `book.yaml` are not executed unless `--allow-plugins` is passed, since plugins are arbitrary executables.
- If `--quiet` and `--verbose` are both set, the current implementation gives precedence to `--quiet`.

## FAQ

### 1. Why are there more chapters than expected?

If neither `book.yaml` nor `SUMMARY.md` exists, `build` recursively scans Markdown files. In a code repository, that often pulls in `README.md`, `docs/`, `examples/`, and test fixtures.

Preferred fixes:

- Build a narrower directory, for example `mdpress build ./docs`
- Or add `book.yaml` or `SUMMARY.md`

### 2. Why does the site preview not match the PDF layout exactly?

That is expected. `site` and `serve` are optimized for website-style reading, while `pdf` and single-page `html` are optimized for document-style layout. When judging final layout quality, use `build --format pdf` or `build --format html`.
