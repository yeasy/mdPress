# CLI Commands Reference

Complete reference for all mdPress commands, flags, and options.

## Global Flags

These flags work with most commands:

```bash
mdpress [global-flags] <command> [command-flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config <path>` | `book.yaml` | Path to configuration file |
| `--cache-dir <path>` | OS default | Override cache directory location |
| `--no-cache` | off | Disable all caching; forces full rebuild |
| `--summary <path>` | auto-detect | Path to SUMMARY.md file |
| `-v, --verbose` | off | Enable verbose output and debug logging |
| `-q, --quiet` | off | Print errors only; suppress info messages |
| `--help` | — | Show command help and exit |
| `--version` | — | Show version and exit |

### Examples

```bash
# Use custom config file
mdpress build --config docs/book.yaml

# Disable cache for full rebuild
mdpress build --no-cache --format pdf

# Enable verbose output
mdpress build --verbose

# Quiet mode (suppress non-errors)
mdpress build --quiet
```

## build

Build book output in specified format.

```bash
mdpress build [source] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[source]` | Input directory or GitHub URL (default: current directory) |

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format <format>` | pdf | Output format: `pdf`, `html`, `epub`, `site`, `typst` |
| `--output <file>` | auto | Output filename or directory |
| `--config <path>` | book.yaml | Configuration file path |

### Examples

```bash
# Build PDF (default)
mdpress build

# Build multiple formats
mdpress build --format pdf --format html --format epub

# Build in HTML format (fastest)
mdpress build --format html

# Build site format for GitHub Pages
mdpress build --format site

# Build from different directory
mdpress build ./docs

# Build with custom output filename
mdpress build --format pdf --output my-book.pdf

# Build from GitHub repository
mdpress build https://github.com/user/book-repo

# Build from private GitHub repo (requires token)
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-repo

# Build with custom config
mdpress build --config docs/book.yaml

# Force full rebuild (skip cache)
mdpress build --no-cache --format pdf

# Enable verbose output
mdpress build --verbose
```

### Output Files

| Format | Output | Location |
|--------|--------|----------|
| `pdf` | Single PDF file | `./output.pdf` or custom filename |
| `html` | Single HTML file | `./output.html` |
| `site` | Website directory | `./_book/` |
| `epub` | E-book file | `./output.epub` |
| `typst` | Typst source | `./output.typ` |

## serve

Start local server with live preview and file watching.

```bash
mdpress serve [source] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[source]` | Input directory (default: current directory) |

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port <port>` | 9000 | HTTP server port |
| `--format <format>` | html | Preview format: `html`, `site` |
| `--config <path>` | book.yaml | Configuration file path |
| `--open` | off | Automatically open browser |

### Examples

```bash
# Start server on default port 9000
mdpress serve

# Start on custom port
mdpress serve --port 3000

# Open browser automatically
mdpress serve --open

# Preview site format
mdpress serve --format site

# Watch specific directory
mdpress serve ./docs

# Serve with custom config
mdpress serve --config docs/book.yaml --open
```

### Features

- **Live reload**: Browser auto-refreshes when files change
- **File watching**: Monitors Markdown, images, and configuration
- **Incremental build**: Only rebuilds changed chapters
- **Fast feedback**: HTML preview is fastest (1-2 seconds)

Access at: http://localhost:9000

## init

Generate configuration file from existing Markdown files.

```bash
mdpress init [directory] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[directory]` | Directory to scan for Markdown files (default: current) |

### Flags

None

### Examples

```bash
# Create book.yaml in current directory
mdpress init

# Scan specific directory
mdpress init ./docs

# Creates book.yaml with auto-discovered chapters
# List: README.md, ch01.md, ch02.md, ...
```

### Generated file

`book.yaml` with:
- Detected title (from directory or first heading)
- Automatically discovered chapter files
- Default style and output settings

Edit the generated file to customize.

## quickstart

Create a new sample project with template files.

```bash
mdpress quickstart [directory]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[directory]` | Directory for new project |

### Examples

```bash
# Create sample project
mdpress quickstart my-book
cd my-book

# Ready to build immediately
mdpress build
```

### Generated Files

```
my-book/
├── book.yaml         # Configuration
├── SUMMARY.md        # Chapter structure
├── README.md         # Introduction
├── chapter1.md       # Sample chapter
├── chapter2.md       # Sample chapter
├── assets/
│   └── cover.png     # Sample cover image
└── .gitignore        # Git configuration
```

Provides a working example to build on.

## validate

Check configuration and validate project structure.

```bash
mdpress validate [directory] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[directory]` | Project directory to validate (default: current) |

### Flags

None

### Examples

```bash
# Validate current project
mdpress validate

# Validate specific directory
mdpress validate ./docs

# Output example:
# ✓ Config file syntax OK
# ✓ All 12 chapters exist
# ✓ 24 images found
# ✓ Cross-references valid
# ✓ Validation successful!
```

### Checks

- Configuration file syntax
- Chapter file existence
- Image file existence
- Cross-reference validity
- Link format correctness
- GLOSSARY.md format (if present)
- LANGS.md format (if present)

Run before building to catch issues early.

## doctor

Check environment and system readiness.

```bash
mdpress doctor [directory] [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[directory]` | Project directory to check (default: current) |

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--report <format>` | text | Output format: `text`, `json`, `markdown` |

### Examples

```bash
# Check environment
mdpress doctor

# Generate JSON report
mdpress doctor --report json > report.json

# Generate Markdown report
mdpress doctor --report markdown > report.md

# Check specific project
mdpress doctor ./docs
```

### Checks

- Platform and OS version
- Go installation
- Chrome/Chromium availability
- CJK font installation
- PlantUML installation
- Cache directory status
- Configuration validity
- Chapter file references
- Image file references

See [doctor.md](../troubleshooting/doctor.md) for detailed information.

## upgrade

Check for and install newer versions of mdpress.

```bash
mdpress upgrade [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | off | Only check for updates without installing |
| `-v, --verbose` | off | Enable verbose output |
| `-q, --quiet` | off | Print errors only |

### Examples

```bash
# Check for updates without installing
mdpress upgrade --check

# Install latest version (default behavior)
mdpress upgrade

# Install with verbose output
mdpress upgrade --verbose

# Verify upgrade completed
mdpress --version
```

### Features

- **Automatic platform detection**: Finds the right binary for your OS and architecture
- **Backup and restore**: Creates a backup before installation; restores on error
- **Version comparison**: Uses semantic versioning to detect newer versions
- **Progress feedback**: Shows download progress and completion status

Supports:
- **Linux**: x86_64, ARM64
- **macOS**: x86_64, ARM64 (Apple Silicon)
- **Windows**: x86_64, ARM64

### Environment Variables

- `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`: Configure HTTP proxies if needed

### Common Issues

See [upgrade troubleshooting](../commands/upgrade.md#common-issues-and-solutions) for solutions.

## migrate

Convert GitBook or HonKit project to mdPress.

```bash
mdpress migrate [directory]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `[directory]` | GitBook/HonKit project directory |

### Examples

```bash
# Migrate existing GitBook project
mdpress migrate ./gitbook-project

# Converts:
# - book.json → book.yaml
# - SUMMARY.md → updated for mdPress
# - Markdown files → mdPress compatible
```

### Conversions

- `book.json` → `book.yaml` (with property mapping)
- `SUMMARY.md` → compatible format
- Plugin configurations → mdPress plugins
- Theme settings → CSS overrides
- Output options → mdPress output config

## themes

Manage themes and view theme information.

```bash
mdpress themes <subcommand> [flags]
```

### Subcommands

#### list

List available themes.

```bash
mdpress themes list

# Output:
# Built-in themes:
#   - technical (default)
#   - elegant
#   - minimal
```

#### show

Show theme details and configuration options.

```bash
mdpress themes show <theme-name>

# Examples:
mdpress themes show technical
mdpress themes show elegant

# Output includes:
# Theme: technical
# Description: Professional technical documentation theme
# Colors: ...
# Fonts: ...
# CSS variables: ...
```

#### preview

Generate preview of all built-in themes.

```bash
mdpress themes preview

# Output: preview-themes.html
# View in browser to compare themes
```

### Examples

```bash
# List all themes
mdpress themes list

# Show technical theme details
mdpress themes show technical

# Generate preview HTML
mdpress themes preview

# Use theme in book.yaml
# style:
#   theme: "elegant"
```

## completion

Generate shell completion scripts.

```bash
mdpress completion <shell>
```

### Arguments

| Argument | Description |
|----------|-------------|
| `<shell>` | Shell type: `bash`, `zsh`, `fish`, `powershell` |

### Examples

```bash
# Bash completion
mdpress completion bash > mdpress-completion.bash
source mdpress-completion.bash

# Zsh completion
mdpress completion zsh > ~/.zfunc/_mdpress

# Fish completion
mdpress completion fish > ~/.config/fish/completions/mdpress.fish

# PowerShell completion
mdpress completion powershell >> $PROFILE

# Then press Tab to autocomplete commands
mdpress build<Tab>  # Suggests --format, --config, etc.
```

### Add to Shell Config

**Bash (~/.bashrc or ~/.bash_profile):**
```bash
source /path/to/mdpress-completion.bash
```

**Zsh (~/.zshrc):**
```bash
fpath=(~/.zfunc $fpath)
autoload -Uz compinit && compinit
```

**Fish (~/.config/fish/config.fish):**
```fish
# Completions auto-loaded from ~/.config/fish/completions/
```

**PowerShell ($PROFILE):**
```powershell
# Completions auto-loaded
```

## Version and Help

### version

Show mdPress version and build information.

```bash
mdpress version
# or
mdpress --version
# or
mdpress -v

# Output:
# mdPress version 1.2.3
# Go version: go1.21.0
# Platform: linux/amd64
```

### help

Show general help or command-specific help.

```bash
# General help
mdpress help
mdpress --help
mdpress -h

# Command help
mdpress build --help
mdpress serve --help

# Output: usage, flags, examples
```

## Environment Variables

### MDPRESS_CHROME_PATH

Path to Chrome or Chromium binary:

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
mdpress build --format pdf
```

### GITHUB_TOKEN

GitHub personal access token for private repositories:

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-repo
```

Requires `contents:read` scope.

### XDG_CACHE_HOME

Override cache directory location (Linux/macOS):

```bash
export XDG_CACHE_HOME=/tmp/cache
mdpress build --format pdf
```

Default locations:
- Linux: `~/.cache/mdpress/`
- macOS: `~/Library/Caches/mdpress/`
- Windows: `%USERPROFILE%\AppData\Local\mdpress\`

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration error or validation failure |
| 2 | Runtime error (e.g., missing Chrome for PDF) |
| 3 | File or permission error |

Example:

```bash
mdpress build --format pdf
echo $?  # Prints exit code
# 0 = success, non-zero = error
```

Use in scripts:

```bash
#!/bin/bash
mdpress build --format pdf
if [ $? -ne 0 ]; then
  echo "Build failed!"
  exit 1
fi
echo "Build succeeded!"
```

## Command Examples

### Quick Preview

```bash
# Fastest preview (HTML, live reload)
mdpress serve --open
```

### Development Workflow

```bash
# Start live preview in browser
mdpress serve --port 3000 --open

# In another terminal, make changes to Markdown files
# → Browser auto-refreshes

# When ready, build PDF
mdpress build --format pdf
```

### CI/CD Integration

```bash
# Validate before building
mdpress validate
if [ $? -ne 0 ]; then exit 1; fi

# Build all formats
mdpress build --format pdf
mdpress build --format site

# Deploy site
cp -r _book/* /var/www/docs/
```

### Batch Processing

```bash
# Build multiple projects
for project in docs1 docs2 docs3; do
  mdpress build "$project" --format pdf
done
```

### Custom Output Names

```bash
# Version-specific PDF
mdpress build --format pdf --output "book-v1.0.pdf"

# Timestamped output
mdpress build --format html --output "snapshot-$(date +%Y%m%d).html"
```

## Troubleshooting Command Issues

### Command not found

```bash
# Check if mdPress is installed
mdpress --version

# If not found, install:
go install github.com/yeasy/mdpress@latest

# Or use full path to binary
/path/to/mdpress build
```

### Permission denied

```bash
# Make binary executable
chmod +x mdpress

# Then run
./mdpress build
```

### Unknown command

```bash
# Show all available commands
mdpress --help

# Show command-specific help
mdpress build --help
```

For more information on configuration and variables, see [configuration.md](configuration.md) and [template-variables.md](template-variables.md).
