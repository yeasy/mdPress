# `mdpress serve`

[中文说明](serve_zh.md)

## Purpose

Build a local preview site and start an HTTP server. `serve` is meant for iterative writing: when files change, mdpress rebuilds and refreshes the browser automatically.

## Syntax

```bash
mdpress serve [source] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[source]` | No | Input source. It can be omitted, a local directory, or a GitHub repository URL. If omitted, the current directory is used. |

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--host <addr>` | `127.0.0.1` | HTTP listen address. By default, `serve` only accepts local connections. Use an explicit address such as `0.0.0.0` to expose it on the LAN. |
| `--port <number>` | `9000` | HTTP server port. If not explicitly set, mdpress starts at `9000` and finds the next available port. |
| `--output <dir>` | `<project>/_book` | Output directory for the preview site. |
| `--open` | off | Open the browser automatically after startup. By default, mdpress does not open the browser. |
| `--summary <path>` | auto-detect | Explicit path to a `SUMMARY.md` file. Overrides chapters from `book.yaml` or auto-discovery. |
| `--config <path>` | `book.yaml` | Config file path for local projects. |
| `-v, --verbose` | off | Print detailed logs. |
| `-q, --quiet` | off | Print errors only. |

## Behavior

### What Gets Previewed

Before starting the server, `serve` builds a multi-page HTML preview site. Compared with `build --format site`, `serve` is the development-oriented path.

### Auto-Discovery Scope

When the target directory contains neither `book.yaml` nor `SUMMARY.md`, `serve` recursively scans Markdown files under that directory.

That means:

- Running `mdpress serve` inside a dedicated docs directory usually behaves as expected.
- Running it from a repository root may pull in `README.md`, `docs/`, `examples/`, `tests/`, and internal design notes.

If you only want to preview formal documentation, prefer one of these patterns:

```bash
mdpress serve ./docs
mdpress serve --config ./docs/book.yaml ./docs
```

Or provide an explicit `book.yaml` or `SUMMARY.md` in the target directory so chapter scope and order are constrained.

### Output Directory

If `--output` is omitted, the preview site is written to `_book/` under the project directory.

```bash
mdpress serve
mdpress serve --open
mdpress serve --output ./preview
```

If `--port` is not explicitly provided, `serve` starts at `9000` and increments until it finds a free port.

### Network Binding

By default, `serve` listens on `127.0.0.1`, so only the local machine can access the preview.

If you want to expose the preview to other machines or bind to a specific interface, pass `--host` explicitly:

```bash
mdpress serve --host 0.0.0.0
mdpress serve --host 192.168.1.10 --port 9000
```

### Config Loading

For local directories, config loading follows this order:

1. The file passed via `--config`
2. Default `book.yaml`
3. Auto-discovered `SUMMARY.md` or Markdown files

In auto-discovery mode, mdpress skips hidden directories, `node_modules`, `vendor`, and `_book`, but it does not automatically know which Markdown files are "real chapters" versus repository notes.

## Examples

```bash
mdpress serve
mdpress serve --open
mdpress serve --host 0.0.0.0
mdpress serve --port 9000
mdpress serve --port 3000
mdpress serve --output ./preview
mdpress serve /path/to/book
mdpress serve https://github.com/yeasy/agentic_ai_guide
```

## Notes

- `serve` currently accepts GitHub repository URLs, but it does not expose `--branch` or `--subdir`; remote preview uses the repository default branch and repository root.
- For remote repository inputs, if `--output` is not set, the current implementation writes preview artifacts to a temporary directory. Those files are typically removed when the process exits.
- For remote repository inputs, the current implementation prefers `book.yaml` inside the remote project. A local `--config` path does not override the remote config location.
- By default, `serve` only accepts local connections on `127.0.0.1`. Network exposure requires an explicit `--host`.
- The browser is not opened automatically unless `--open` is passed.
- The `serve` layout is optimized for website-style reading. It does not preserve PDF or print-style page margin semantics. If final page layout matters, also verify with `build --format pdf` or `build --format html`.
- `serve` is for local reading and iteration. It does not produce PDF or ePub output.
- If `--quiet` and `--verbose` are both set, the current implementation gives precedence to `--quiet`.

## FAQ

### 1. Why am I seeing many Markdown files I did not intend to preview?

This usually happens when `mdpress serve` is run from a repository root without `book.yaml` or `SUMMARY.md`. That triggers auto-discovery and treats most Markdown files as candidate chapters.

Preferred fixes:

- Switch to `mdpress serve ./docs`
- Or add `book.yaml` or `SUMMARY.md`

### 2. Why does the preview look like a mix of document margins and website sidebars?

Because multi-page preview and PDF/single-page HTML target different reading modes. `serve` is optimized for browser navigation, not print layout. In site mode, prioritize navigation, width, and readability rather than PDF fidelity.

### 3. What should I check if images do not appear?

Check these first:

- Whether the relative path in Markdown is relative to the current chapter file, not the repository root
- Whether the image file actually exists
- Whether you are previewing a remote repository with unavailable external assets

For remote images, `mdpress validate` and `mdpress doctor` are the first commands to use when debugging.
