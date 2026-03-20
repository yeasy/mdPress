# mdPress Command Manual

[中文说明](COMMANDS_zh.md)

This document summarizes the main `mdpress` commands, global flags, and practical caveats. For command-specific behavior, use the linked subdocuments below.

## Command Hierarchy

```mermaid
flowchart TD
    root["mdpress"]

    root --> build["build [source]<br/>Build outputs"]
    root --> serve["serve [source]<br/>Live preview"]
    root --> init["init [directory]<br/>Generate book.yaml"]
    root --> quickstart["quickstart [directory]<br/>Sample project"]
    root --> validate["validate [directory]<br/>Check config"]
    root --> doctor["doctor [directory]<br/>Check environment"]
    root --> themes["themes<br/>Theme management"]
    root --> completion["completion &lt;shell&gt;<br/>Shell completions"]

    themes --> list["list"]
    themes --> show["show &lt;name&gt;"]
    themes --> preview["preview"]

    build --> pdf["--format pdf"]
    build --> html["--format html"]
    build --> site["--format site"]
    build --> epub["--format epub"]
```

## Command Matrix

| Command | Purpose | Doc |
| --- | --- | --- |
| `mdpress build [source]` | Build PDF, HTML, site, or ePub outputs | [build](commands/build.md) |
| `mdpress serve [source]` | Start the local preview server and watch for file changes | [serve](commands/serve.md) |
| `mdpress init [directory]` | Scan Markdown files and generate `book.yaml` | [init](commands/init.md) |
| `mdpress quickstart [directory]` | Create a sample project that can be built immediately | [quickstart](commands/quickstart.md) |
| `mdpress validate [directory]` | Validate config, chapter files, and referenced assets | [validate](commands/validate.md) |
| `mdpress doctor [directory]` | Check environment readiness and project buildability | [doctor](commands/doctor.md) |
| `mdpress themes list` | List built-in themes | [themes](commands/themes.md) |
| `mdpress themes show <theme-name>` | Show theme details and config hints | [themes](commands/themes.md) |
| `mdpress themes preview` | Generate an HTML preview of built-in themes | [themes](commands/themes.md) |
| `mdpress completion <shell>` | Generate shell completion scripts | [completion](commands/completion.md) |

## Global Flags

These flags appear in `--help` output for most commands.

| Flag | Default | Description |
| --- | --- | --- |
| `--config <path>` | `book.yaml` | Config file path. Mainly relevant for commands that load project config, such as `build`, `serve`, and `validate`. |
| `-v, --verbose` | off | Print more detailed logs and warning-by-warning output. |
| `-q, --quiet` | off | Print errors only. |

Notes:

- If `--quiet` and `--verbose` are both set, the current implementation gives precedence to `--quiet`.
- `--config` is a global flag, but not every command actually uses it. `doctor`, `themes`, and `completion` currently ignore it.

## Input Source Rules

mdPress mainly supports two kinds of input:

- Local directories: if `[source]` is omitted, the current directory is used.
- GitHub repository URLs: for example `https://github.com/yeasy/agentic_ai_guide`. For private repositories, set `GITHUB_TOKEN` (see below).

For local directories, config discovery usually follows this order:

1. `book.yaml`
2. `SUMMARY.md`
3. Automatic `.md` file discovery

## GitHub Authentication

To build from private repositories, set the `GITHUB_TOKEN` environment variable before running `mdpress build` or `mdpress serve`:

    export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
    mdpress build https://github.com/myorg/private-docs

The token is embedded in the clone URL and never logged. Any GitHub personal access token or fine-grained token with `contents:read` scope will work. When the token is not set and a clone fails, the error message will suggest setting it.

## Outputs And Defaults

- If `build` is called without `--format`, it first checks `output.formats`.
- If `output.formats` is also absent, the default output is `pdf`.
- The default output filename is derived from the book title (with filesystem-unsafe characters replaced). If the title is empty or "Untitled Book", the project directory name is used instead. You can override this with `output.filename`.
- `serve` writes preview output to `_book/` under the project directory by default.

## Output Configuration

| Setting | Default | Description |
| --- | --- | --- |
| `output.toc_max_depth` | `2` | Maximum heading level to include in the table of contents (1–6). For example, `2` includes h1 and h2; `3` also includes h3. |
| `output.pdf_timeout` | `120` | Maximum seconds to wait for Chromium to finish rendering a PDF page. Increase for very large books. |
| `MDPRESS_CHROME_PATH` (env) | auto-detect | Absolute path to a Chrome or Chromium binary. When set, mdPress skips auto-detection and uses this path directly. |

Example `book.yaml` snippet:

    output:
      toc_max_depth: 3
      pdf_timeout: 300

Example environment variable usage:

    MDPRESS_CHROME_PATH=/usr/bin/chromium mdpress build --format pdf

## Boundaries Of Auto-Discovery

Auto-discovery works well when one directory is clearly one book or one documentation set. It is not a good fit for a large repository root.

Typical risks:

- The repository root `README.md` may become chapter one.
- `docs/`, `examples/`, `tests/`, and internal design notes may all enter the chapter list.
- The build may succeed, but the resulting information architecture is often not what you actually want.

Recommended approach:

- Run commands in the real docs subdirectory, for example `mdpress serve ./docs`.
- Or provide an explicit `book.yaml` or `SUMMARY.md`.

## Troubleshooting

- For command boundaries and behavior, start with [serve](commands/serve.md) and [build](commands/build.md).

## Suggested Reading Order

- For a quick start, read [build](commands/build.md) and [serve](commands/serve.md) first.
- For integrating an existing repository, continue with [init](commands/init.md) and [validate](commands/validate.md).
- For environment troubleshooting, read [doctor](commands/doctor.md).
- For theming, read [themes](commands/themes.md).
