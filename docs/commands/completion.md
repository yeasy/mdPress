# `mdpress completion`

[中文说明](completion_zh.md)

## Purpose

Generate shell completion scripts. The current supported shells are:

- `bash`
- `zsh`
- `fish`
- `powershell`

## Syntax

```bash
mdpress completion <shell>
```

## Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `<shell>` | Yes | Shell type. Valid values are `bash`, `zsh`, `fish`, and `powershell`. |

## Common Usage

### Bash

```bash
mdpress completion bash
source <(mdpress completion bash)
```

### Zsh

```bash
mdpress completion zsh
source <(mdpress completion zsh)
```

## Subcommand Flags

Completion subcommands `bash` and `fish` currently support:

| Flag | Default | Description |
| --- | --- | --- |
| `--no-descriptions` | off | Disable completion item descriptions. |

## Notes

- Completion scripts are written to stdout, so they are usually redirected into a file or loaded immediately with `source <(...)`.
- The `bash` help output explicitly requires the system package `bash-completion`.
- If zsh completion is not enabled yet, run `autoload -U compinit; compinit` first.
- `--config` appears in global flags, but `completion` does not use it.
