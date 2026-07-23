# Using mdpress doctor

`mdpress doctor` reports what mdPress found in your environment and whether the project in the target directory can be loaded. Use it before a build, and as a CI gate with `--strict`.

```bash
mdpress doctor                 # check the current directory
mdpress doctor /path/to/book   # check another directory
```

## Real Output

This is the full output on a healthy machine and a scaffolded project:

```
  mdpress Environment Check
  ──────────────────────────────────────────────────

  ✓ Platform: darwin/arm64
  ✓ Go version: go1.26.5
  ✓ Runtime cache: /private/tmp/mdpress-cache
  ✓ Chromium/Chrome is available
  ✓ Typst is available: typst 0.15.0 (unknown commit)
  ✓ Go version 1.26.5 (>= 1.26)
  ✓ Git is available
  ✓ Network connectivity to github.com available
  ✓ Disk space available
  ✓ CJK fonts available: (system CJK fonts detected)
  ✓ PlantUML not needed (no diagrams detected)


  Project Check
  ──────────────────────────────────────────────────
  ✓ Detected book.yaml
  ⚠ SUMMARY.md not found
  ⚠ LANGS.md not found
  ✓ Config loads successfully: my-book (4 top-level chapters)
  ✓ Markdown chapter links resolve within the build graph
```

`SUMMARY.md not found` and `LANGS.md not found` are informational: a project configured with `book.yaml` does not need either file.

## What Each Check Means

| Check | Meaning |
| --- | --- |
| Platform | `GOOS/GOARCH` of the running binary |
| Go version | The Go runtime mdPress was compiled with (informational) |
| Runtime cache | Where the parse cache lives, or a warning when `--no-cache` disabled it |
| Chromium/Chrome | The default PDF backend. Auto-detected, or taken from `MDPRESS_CHROME_PATH` |
| Typst | The alternative PDF backend for `--format typst` |
| Go version >= 1.26 | Only relevant when installing via `go install` |
| Git | Needed for building from a GitHub URL |
| Network connectivity | A reachability probe against github.com, for remote sources |
| Disk space | Free space in the output directory |
| CJK fonts | Needed for Chinese/Japanese/Korean text in PDF output |
| PlantUML | Reported only as "not needed" when the project contains no PlantUML fences |
| Plugins | Each entry under `plugins:` in `book.yaml` must exist and be executable |
| book.yaml / SUMMARY.md / LANGS.md | Which project files are present |
| Config loads | Whether `config.Load` (or auto-discovery from `SUMMARY.md`) succeeds |
| Markdown chapter links | Whether relative `.md` links point at files inside the build graph |

### The PDF backend check

PDF needs *either* Chromium/Chrome *or* Typst, so a missing backend is only an error when both are gone:

- Chromium missing, Typst present → warning: `Chromium/Chrome is unavailable — use --format typst for PDF output instead`
- Both missing → error: `No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)`

### Links outside the build graph

```
  ⚠ Detected 1 Markdown link(s) outside the build graph
    - ../outside.md (from one.md)
```

The link target is not one of the chapters mdPress will build, so it will be a dead link in the site and HTML output. Either add the file as a chapter or fix the link.

## Exit Codes And `--strict`

**Without `--strict`, `mdpress doctor` always exits 0**, even when it prints `✗` lines. A CI step that runs plain `mdpress doctor` gates on nothing.

```bash
mdpress doctor --strict
```

`--strict` exits non-zero when any error-level check fails, printing:

```
Error: doctor found 1 error-level issue(s) (run without --strict to ignore)
```

Error-level findings are: no PDF backend at all, a `book.yaml` that fails to load, auto-discovery failure, low disk space, and broken plugin entries. Warnings (missing CJK fonts, absent `SUMMARY.md`, links outside the build graph) never affect the exit code.

## Reports

```bash
mdpress doctor --report report.json
mdpress doctor --report report.md
```

The extension picks the format. Anything else fails after the checks have already printed:

```
Error: failed to write doctor report: unsupported report extension: .txt (use .json or .md)
```

### JSON report

The JSON report is a flat object with `snake_case` keys. This is a complete report from the healthy run above:

```json
{
  "platform": "darwin/arm64",
  "go_version": "go1.26.5",
  "cache_dir": "/private/tmp/mdpress-cache",
  "cache_disabled": false,
  "chromium_available": true,
  "typst_available": true,
  "typst_version": "typst 0.15.0 (unknown commit)",
  "cjk_fonts_available": true,
  "plantuml_available": false,
  "plantuml_needed": false,
  "go_version_check": "go1.26.5",
  "git_available": true,
  "network_available": true,
  "disk_space_gb": 62.70961380004883,
  "disk_space_ok": true,
  "plugins_valid": true,
  "book_yaml_found": true,
  "summary_found": false,
  "langs_found": false,
  "project_loadable": true,
  "project_title": "my-book",
  "top_level_chapters": 4
}
```

Three more keys appear only when they have content:

| Key | Type | When it appears |
| --- | --- | --- |
| `plugin_count` | number | The project declares plugins |
| `warnings` | array of strings | Any warning or error message was recorded |
| `unresolved_markdown_links` | array of `{"Source", "Target"}` | Links point outside the build graph |

A project whose `book.yaml` does not load:

```json
{
  "book_yaml_found": true,
  "project_loadable": false,
  "warnings": [
    "Failed to load book.yaml: config validation failed: chapter validation failed: chapter 1 references a missing file: nope.md (paths are relative to book.yaml)"
  ]
}
```

A project with a link outside the build graph:

```json
{
  "unresolved_markdown_links": [
    {
      "Source": "one.md",
      "Target": "../outside.md"
    }
  ]
}
```

Note that `warnings` mixes warnings and error-level findings; the exit code from `--strict` is the only reliable way to tell them apart.

### Parsing with jq

```bash
mdpress doctor --report report.json

jq '.chromium_available' report.json        # can this machine produce a PDF with the default backend?
jq '.project_loadable' report.json          # does book.yaml load?
jq -r '.warnings[]? // empty' report.json   # list every warning, if any
jq -r '.unresolved_markdown_links[]? | "\(.Source) -> \(.Target)"' report.json
```

### Markdown report

`--report report.md` writes the same fields as a bullet list, suitable for uploading as a CI artifact:

```markdown
# mdpress Doctor Report

- Platform: darwin/arm64
- Go version: go1.26.5
- Go version check: go1.26.5
- Cache disabled: false
- Cache dir: /private/tmp/mdpress-cache
- Chromium available: true
- Typst available: true
- Typst version: typst 0.15.0 (unknown commit)
- CJK fonts available: true
- Git available: true
- Network connectivity: true
- Disk space available: 62.71 GB
- Disk space OK: true
- PlantUML needed: false
- PlantUML available: false
- Plugins valid: true
- book.yaml found: true
- SUMMARY.md found: false
- LANGS.md found: false
- Project loadable: true
- Project title: my-book
- Top-level chapters: 4
```

## Using Doctor In CI

Always pass `--strict` — without it the step can never fail.

### GitHub Actions

```yaml
- name: Environment and project readiness
  run: mdpress doctor --strict

- name: Upload doctor report
  if: always()
  run: mdpress doctor --report doctor-report.md
- uses: actions/upload-artifact@v4
  if: always()
  with:
    name: doctor-report
    path: doctor-report.md
```

### GitLab CI

```yaml
doctor:
  stage: validate
  script:
    - mdpress doctor --strict
    - mdpress doctor --report doctor-report.md
  artifacts:
    paths:
      - doctor-report.md
    expire_in: 30 days
```

### Pre-build script

```bash
#!/bin/bash
set -e

mdpress doctor --strict   # set -e stops here on an error-level finding
mdpress validate
mdpress build --format pdf
```

## Common Findings And Fixes

### No PDF backend available

```
✗ No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)
```

Install one of them, or build a format that needs neither:

```bash
# macOS
brew install chromium        # or: brew install typst
# Ubuntu/Debian
sudo apt-get install chromium-browser

# Chromium in a non-standard location
export MDPRESS_CHROME_PATH=/path/to/chrome

# Or skip the PDF backends entirely
mdpress build --format site,html,epub
```

### No CJK fonts detected

```
⚠ No CJK fonts detected — PDF output for Chinese/Japanese/Korean text may show blank squares
```

Harmless for an English-only book. Otherwise:

```bash
sudo apt-get install fonts-noto-cjk   # Ubuntu/Debian
apk add font-noto-cjk                 # Alpine
```

### Failed to load book.yaml

```
✗ Failed to load book.yaml: config validation failed: chapter validation failed: chapter 1 references a missing file: nope.md (paths are relative to book.yaml)
```

Chapter paths are resolved relative to `book.yaml`, not to your shell's working directory. `mdpress validate` reports every such problem in more detail.

### No directly buildable book.yaml or SUMMARY.md found

```
⚠ No directly buildable book.yaml or SUMMARY.md found in the target directory
```

You are pointing doctor at a directory that has neither. `mdpress init` generates a `book.yaml` from the Markdown files already there.

## What Doctor Does Not Check

Doctor inspects the environment and whether the config loads. It does not:

- validate Markdown or link targets in depth — that is `mdpress validate`
- verify images exist or are the right size
- run an actual build

A full pre-flight is all three:

```bash
mdpress doctor --strict
mdpress validate
mdpress build --format html
```

See [common-issues.md](common-issues.md) for fixes to build-time errors.
