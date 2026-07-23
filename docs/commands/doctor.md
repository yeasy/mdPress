# `mdpress doctor`

[中文说明](doctor_zh.md)

## Purpose

Report the current runtime environment and check whether the target project meets the most basic PDF and project-loadability conditions. `doctor` is useful right after installation or when troubleshooting a failed build.

## Syntax

```bash
mdpress doctor [directory] [flags]
```

## Positional Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `[directory]` | No | Project directory to inspect. If omitted, the current directory is used. |

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-o, --report <path>` | empty | Write the check results to a `.json` or `.md` report file. |
| `--strict` | off | Exit with a non-zero status when any error-level check fails. Useful as a CI gate. |
| `-v, --verbose` | off | Print detailed logs. |
| `-q, --quiet` | off | Print errors only. |

## What Gets Checked

The current implementation has two categories of output:

Environment information:

- Go runtime platform information
- Go version (>= 1.26 recommended)
- Cache directory status

Readiness checks:

- Chrome/Chromium availability (for PDF output)
- Typst CLI availability (a missing Chrome is only reported as an error when Typst is also unavailable)
- Go version check (>= 1.26)
- Git availability (for remote source builds)
- Network connectivity to github.com
- Disk space in output directory
- CJK font availability (for Asian text PDF rendering)
- PlantUML availability (if PlantUML diagrams are detected in the project)
- Plugin health and availability
- Presence of `book.yaml`, `SUMMARY.md`, and `LANGS.md`
- Whether the project can be loaded
- Whether Markdown chapter links stay inside the build graph

## Examples

```bash
mdpress doctor
mdpress doctor /path/to/book
mdpress doctor --strict
mdpress doctor --report doctor-report.json
```

## Notes

- **Without `--strict`, `doctor` always exits 0**, even when it prints `✗` lines. A CI step that runs plain `mdpress doctor` gates on nothing.
- `--verbose` provides detailed output including exact disk space measurements, Go version parse details, and verbose error messages for each check.
- PlantUML availability is only checked if PlantUML diagrams are detected in the project.
- Plugin checks validate both path existence and executability.
- If Chrome or Chromium is not detected, `doctor` clearly reports that PDF output will fail.
- If a directory has no `book.yaml` but does have `SUMMARY.md`, the current implementation attempts to load the project through auto-discovery behavior.
- `doctor` does not modify any files.
- `--config` appears in global flags, but `doctor` currently does not switch to a different config path based on it.
