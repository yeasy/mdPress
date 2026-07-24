# `mdpress config`

[中文说明](config_zh.md)

## Purpose

Show the configuration mdPress actually resolves for a project, so "I set it and nothing
happened" is a question you can answer instead of guess at.

## Syntax

```bash
mdpress config show [directory] [flags]
```

## Subcommands

| Subcommand | Description |
| --- | --- |
| `show` | Print the effective configuration a build would use |

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[directory]` | No | Project directory. If omitted, the current directory is used. |

## Flags

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--format` | `-f` | `yaml` | Output encoding: `yaml` or `json` |

`config show` also uses the global flags, notably `--config <path>` to inspect a config
file other than `book.yaml`.

## What Gets Printed

Two things:

1. The `book.yaml` settings after defaults have been applied — or, when there is no
   `book.yaml`, the settings auto-discovery inferred. This is where you see that
   `output.filename` is empty rather than `output.pdf`, or that `book.language` resolved to
   `en-US`.
2. A `resolved` section with values mdPress computes rather than reads: which config file
   was loaded and whether it was discovered, the base directory, the chapter count, the
   glossary and `LANGS.md` paths, the theme's name and source (built-in or a project
   `themes/<name>.yaml`), the typography renderers receive after style overrides, and the
   file each requested output format would be written to.

## Examples

```bash
mdpress config show
mdpress config show ./my-book
mdpress config show --config release.yaml
mdpress config show --format json
```

Scripting:

```bash
mdpress config show --format json | jq -r .style.theme
mdpress config show --format json | jq -r .resolved.artifacts.pdf
```

## Notes

- The command loads config exactly the way a build does, so a config error surfaces here
  the same way it would during `build`.
- It is read-only: nothing is written, and no chapters are rendered.
