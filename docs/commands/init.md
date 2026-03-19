# `mdpress init`

[中文说明](init_zh.md)

## Purpose

Initialize an `mdpress` project in a directory. It has two modes:

- If the directory already contains Markdown files, scan the structure and generate `book.yaml`
- If the directory has no Markdown files, create a minimal starter template

## Syntax

```bash
mdpress init [directory] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[directory]` | No | Target directory. If omitted, the current directory is used. |

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-i, --interactive` | off | Use interactive prompts for title, author, language, and theme. |
| `-v, --verbose` | off | Print detailed logs. |
| `-q, --quiet` | off | Print errors only. |

## Behavior

### Scan An Existing Project

```bash
mdpress init ./my-book
```

The current implementation:

- Recursively scans `.md` files
- Uses the first H1 in each file as the chapter title
- Generates `book.yaml`
- Detects `SUMMARY.md`, `GLOSSARY.md`, and `LANGS.md`
- Auto-detects common cover filenames such as `cover.png`, `cover.jpg`, and `cover.svg`

### Initialize An Empty Directory

If the target directory has no Markdown files, the command creates a minimal project skeleton with:

- `book.yaml`
- `preface.md`
- `chapter01/README.md`

### Interactive Mode

```bash
mdpress init --interactive
mdpress init ./my-book -i
```

Interactive mode asks for:

- Title
- Author
- Language
- Theme

If the current terminal is not interactive, the command falls back to default values.

## Examples

```bash
mdpress init
mdpress init ./docs-book
mdpress init --interactive
mdpress init ./docs-book -i
```

## Notes

- If `book.yaml` already exists in the target directory, the command fails instead of overwriting it.
- When scanning an existing project, a top-level `README.md` is currently treated as project-level introduction and skipped as a chapter file; `README.md` files inside subdirectories are kept.
- If `SUMMARY.md` is detected, the generated `book.yaml` does not write a `chapters` list. Chapter structure is delegated to `SUMMARY.md` during builds.
- `--config` appears in global flags, but `init` does not use it to change the output path. It always writes `book.yaml` into the target directory.
