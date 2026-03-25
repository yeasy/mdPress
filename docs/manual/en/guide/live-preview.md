# Live Preview

`mdpress serve` builds the site preview, starts a local HTTP server, and reloads the browser when source files change. It accepts a local directory or a GitHub URL; when no source is passed, it uses the current directory.

## Start It

```bash
mdpress serve
```

By default it listens on `127.0.0.1:9000`, writes preview output to `_book/`, and does not open the browser automatically.

## Useful Flags

- `--open` opens the browser after startup.
- `--host 0.0.0.0` exposes the server on your network.
- `--port 3000` picks a specific port. If you do not set it, mdPress starts at `9000` and uses the first free port.
- `--output ./preview` writes the generated site to a different directory.
- `--summary SUMMARY.md` forces chapter order from a specific summary file.

Example:

```bash
mdpress serve --host 0.0.0.0 --port 3000 --open
```

## What Triggers A Rebuild

- Markdown source files
- `book.yaml`
- `SUMMARY.md`
- referenced assets such as images and styles

## Notes

- `serve` is a browser preview path. Use `mdpress build --format pdf` or `mdpress build --format html` if you need final pagination checks.
- If you run from a repo root without `book.yaml` or `SUMMARY.md`, mdPress falls back to auto-discovery and may pick up more Markdown files than you intended.
