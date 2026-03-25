# CLI Commands Reference

This page summarizes the commands and flags exposed by the current CLI.

## Global Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `--config <path>` | `book.yaml` | Config file for local projects. |
| `--cache-dir <path>` | OS default | Override the cache directory. |
| `--no-cache` | off | Disable runtime caches. |
| `-v, --verbose` | off | Show verbose logs. |
| `-q, --quiet` | off | Print errors only. |
| `--version` | - | Print the version number. |

`--config` is used for local projects. When you pass a GitHub URL as the source, mdPress uses the config file inside the fetched repository.

## Core Commands

### `build`

```bash
mdpress build [source] [flags]
```

Build documents from the current directory, a local directory, or a GitHub URL.

Key flags:

- `--format` accepts comma-separated formats such as `pdf,html,epub` or `all`.
- `--branch` and `--subdir` apply to GitHub sources.
- `--output` sets the output base path.
- `--summary` loads chapters from a specific `SUMMARY.md`.

### `serve`

```bash
mdpress serve [source] [flags]
```

Start the live preview server.

It accepts the current directory, a local directory, or a GitHub URL.

Key flags:

- `--host` sets the listen address.
- `--port` sets the listen port.
- `--output` sets the preview output directory.
- `--open` opens the browser automatically.
- `--summary` loads chapters from a specific `SUMMARY.md`.

### `init`

```bash
mdpress init [directory] [-i]
```

Scan Markdown files and generate `book.yaml`. Use `-i, --interactive` to answer prompts for title, author, language, and theme.

### `quickstart`

```bash
mdpress quickstart [directory]
```

Create a sample project. This command has no dedicated flags.

### `validate`

```bash
mdpress validate [directory] [--report path]
```

Validate config, referenced files, images, and chapter links. `--report` writes a `.json` or `.md` report.

## Other Commands

- `mdpress doctor`
- `mdpress themes`
- `mdpress migrate`
- `mdpress upgrade`
- `mdpress completion`
- `mdpress version`
