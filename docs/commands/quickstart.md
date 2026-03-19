# `mdpress quickstart`

[中文说明](quickstart_zh.md)

## Purpose

Create a complete sample project. `quickstart` is meant for first-time evaluation and for spinning up a demo repository quickly.

## Syntax

```bash
mdpress quickstart [directory] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[directory]` | No | Target directory. The default is `my-book`. |

## Flags

`quickstart` currently has no dedicated flags. It supports the standard logging flags:

| Flag | Default | Description |
| --- | --- | --- |
| `-v, --verbose` | off | Print detailed logs. |
| `-q, --quiet` | off | Print errors only. |

## What Gets Created

The current implementation generates:

- `book.yaml`
- `README.md`
- `preface.md`
- `chapter01/README.md`
- `chapter02/README.md`
- `chapter03/README.md`
- `images/README.md`
- `images/cover.svg`

After generation, the project can be built and previewed immediately.

## Examples

```bash
mdpress quickstart
mdpress quickstart my-book
mdpress quickstart ./examples/demo-book
```

## Recommended Next Commands

```bash
cd my-book
mdpress build --format html
mdpress serve
```

## Notes

- If the target directory already exists and is not empty, the command refuses to write, to avoid overwriting user files.
- If the target directory already exists but is empty, the current implementation allows writing.
- `quickstart` is for creating a demo project. It does not scan your existing Markdown content; use `mdpress init` for that.
- `--config` appears in global flags, but `quickstart` does not use it.
