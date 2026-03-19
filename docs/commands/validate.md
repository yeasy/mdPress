# `mdpress validate`

[中文说明](validate_zh.md)

## Purpose

Validate project configuration and referenced assets so you can catch obvious problems before a real build.

## Syntax

```bash
mdpress validate [directory] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[directory]` | No | Target directory. If omitted, the current directory is used. |

## Flags

`validate` currently has no dedicated flags and mainly uses global flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--config <path>` | `book.yaml` | Config file path used when validating the current directory. |
| `-v, --verbose` | off | Print more detailed logs. |
| `-q, --quiet` | off | Print errors only. |

## What Gets Validated

The current implementation checks:

- Whether the config file exists and parses successfully
- Whether `book.title` is present
- Whether the chapter list is empty
- Whether chapter files exist
- Whether the cover image exists
- Whether `custom_css` exists
- Whether locally referenced images in Markdown exist
- Heading numbering and Mermaid-related diagnostics
- Whether Markdown chapter links point to files inside the current build graph

## Config Resolution Rules

### Without A Directory Argument

```bash
mdpress validate
mdpress validate --config ./configs/book.yaml
```

In this mode, the path passed with `--config` takes precedence.

### With A Directory Argument

```bash
mdpress validate /path/to/book
```

In this mode, the current implementation first looks for `/path/to/book/book.yaml`. If that file does not exist, it falls back to auto-discovered `SUMMARY.md` or Markdown files.

## Examples

```bash
mdpress validate
mdpress validate --config ./book.dev.yaml
mdpress validate /path/to/book
mdpress validate ./examples/chapter01
```

## Notes

- Missing `book.author` is currently reported as a warning and does not fail validation by itself.
- If `[directory]` is provided, the current implementation does not prioritize some other path passed via `--config`; it checks the target directory's `book.yaml` first.
- `validate` is good at catching explicit configuration problems, but it is not a full replacement for a real build. PDF dependencies and rendering issues should still be verified with `build` or `doctor`.
