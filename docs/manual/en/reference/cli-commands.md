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

- `-f, --format` accepts comma-separated formats such as `pdf,html,epub` or `all`.
- `--branch` and `--subdir` apply to GitHub sources.
- `-o, --output` sets the output target: an existing directory (or a path with a trailing slash) receives the files, and site pages are written into it directly; any other path acts as a filename base (`manual.pdf`, `manual.html`, `manual_site/`). Without `--output`, the site format goes to `_book/` under the project directory.
- `--summary` loads chapters from a specific `SUMMARY.md`.
- `--allow-plugins` executes plugins declared by a remote project's `book.yaml` (they are skipped by default because plugins are arbitrary executables; local projects always run plugins).

### `serve`

```bash
mdpress serve [source] [flags]
```

Start the live preview server.

It accepts the current directory, a local directory, or a GitHub URL.

Key flags:

- `--host` sets the listen address.
- `--port` sets the listen port.
- `-o, --output` sets the preview output directory (default `_book/`).
- `--open` opens the browser automatically.
- `--summary` loads chapters from a specific `SUMMARY.md`.
- `--branch` and `--subdir` apply to GitHub sources.
- `--allow-plugins` executes plugins declared by a remote project's `book.yaml`.

### `init`

```bash
mdpress init [directory] [-i]
```

Scan Markdown files and generate `book.yaml`. Use `-i, --interactive` to answer prompts for title, author, language, and theme.

### `quickstart`

```bash
mdpress quickstart [directory] [--force]
```

Create a sample project. `--force` allows scaffolding into a non-empty directory (existing files are never overwritten).

### `validate`

```bash
mdpress validate [directory] [--report path]
```

Validate config, referenced files, images, and chapter links. `--report` writes a `.json` or `.md` report.

## doctor

Check environment and system readiness.

```bash
mdpress doctor [directory] [flags]
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `-o, --report <path>` | — | Write diagnostic report to a `.json` or `.md` file |
| `--strict` | off | Exit non-zero when any error-level check fails (useful as a CI gate) |

```bash
# Check environment
mdpress doctor

# Generate a JSON report
mdpress doctor --report report.json

# Fail the CI job on error-level findings
mdpress doctor --strict

# Check a specific project
mdpress doctor ./docs
```

Checks: platform/OS, Go installation, Chrome/Chromium availability, CJK fonts, PlantUML, cache directory, config validity, chapter and image references.

See [doctor.md](../troubleshooting/doctor.md) for details.

## upgrade

Check for and install a new version of mdpress.

```bash
mdpress upgrade [flags]
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `--check` | off | Only check for updates, do not install |
| `--force` | off | Force binary replacement even for Homebrew/`go install` managed installs |
| `--skip-checksum` | off | Skip checksum verification of the downloaded binary (not recommended) |
| `-v, --verbose` | off | Enable verbose output |
| `-q, --quiet` | off | Print errors only |

```bash
# Check for updates without installing
mdpress upgrade --check

# Install the latest version
mdpress upgrade

# Verify the upgrade
mdpress --version
```

Features: automatic platform detection, backup and restore, semantic version comparison, progress feedback. Supports Linux, macOS, and Windows on x86_64 and ARM64.

## migrate

Convert a GitBook or HonKit project to mdPress.

```bash
mdpress migrate [directory]
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `--dry-run` | off | Preview changes without writing files |
| `--force` | off | Overwrite existing `book.yaml` instead of skipping |

```bash
# Migrate a GitBook project
mdpress migrate ./gitbook-project
```

Converts `book.json` to `book.yaml`, updates `SUMMARY.md`, maps plugin and theme settings to mdPress equivalents.

## themes

Manage themes and view theme information.

```bash
mdpress themes <subcommand>
```

### `themes list`

List available themes.

```bash
mdpress themes list
```

### `themes show`

Show theme details and configuration options.

```bash
mdpress themes show <theme-name>
```

### `themes preview`

Generate a preview of all built-in themes.

```bash
mdpress themes preview
# Output: themes-preview.html
```

Use `-o, --output <path>` to write the preview to a custom location:

```bash
mdpress themes preview --output custom-preview.html
```

## completion

Generate shell completion scripts.

```bash
mdpress completion <shell>
```

Supported shells: `bash`, `zsh`, `fish`, `powershell`.

```bash
# Bash (add to ~/.bashrc to make it permanent)
source <(mdpress completion bash)

# Zsh (add to ~/.zshrc to make it permanent)
source <(mdpress completion zsh)

# Fish
mdpress completion fish > ~/.config/fish/completions/mdpress.fish

# PowerShell
mdpress completion powershell >> $PROFILE
```

## version

Display mdPress version and build information.

```bash
mdpress version
mdpress --version
```

## Environment Variables

### MDPRESS_CHROME_PATH

Path to the Chrome or Chromium binary:

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
mdpress build --format pdf
```
