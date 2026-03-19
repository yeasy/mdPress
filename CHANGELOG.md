# Changelog

All notable changes to this project will be documented in this file. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned

- LaTeX math support (KaTeX)
- Native Mermaid diagram support
- Plugin system
- Multilingual build improvements (i18n)

---

## [0.2.0] - 2026-03-19

### Added

- **Single-page HTML output**: `--format html` generates a self-contained HTML file for sharing and offline reading
- **Multi-page site output**: `--format site` produces a static site with sidebar navigation, one page per chapter
- **GitHub repository sources**: `mdpress build https://github.com/user/repo` clones and builds directly from GitHub, with `--branch` and `--subdir` support
- **SUMMARY.md compatibility**: Full GitBook-style SUMMARY.md support; `--summary` flag for explicit path selection
- **ePub output (experimental preview)**: `--format epub` generates ePub ebooks. Current support is experimental; full ePub 3 compliance is planned for v0.3.0
- **Multilingual builds**: When `LANGS.md` is present, build separate outputs per language directory with a landing page
- **Zero-config auto-discovery**: Run `mdpress build` in any directory with `.md` files — no `book.yaml` or `SUMMARY.md` required
- **Live preview server**: `mdpress serve` starts a local HTTP server with file watching and WebSocket auto-reload
- **CSS-only refresh**: When only CSS files change, the browser updates styles without a full page reload
- **Build error overlay**: When a build fails, the browser displays an error overlay with the failure message
- **`mdpress quickstart` command**: Create a sample project and start previewing immediately
- **`mdpress validate` command**: Validate `book.yaml` configuration and file references
- **`mdpress doctor` command**: Check Chromium/Chrome availability and project buildability
- **Multi-format builds**: `--format pdf,html` supports comma-separated format lists
- **Source abstraction layer**: `internal/source/` package with unified `Source` interface (LocalSource, GitHubSource)
- **OutputFormat interface**: `internal/output/output.go` with format registration and dispatch
- **Link rewrite package**: `internal/linkrewrite/` for consistent Markdown link rewriting across output formats
- **Git LFS detection**: Automatic detection and warning when building from repositories that use Git LFS
- **Plugin system placeholder**: `internal/plugin/plugin.go` defines Plugin interface and lifecycle hooks for v0.3.0

### Changed

- `--format` flag accepts `pdf`, `html`, `site`, `epub` and comma-separated combinations
- `build --format site` uses the full site generator instead of the simplified placeholder
- `build --output` controls output file path, directory, or filename prefix
- Multilingual outputs inject language switcher navigation
- All error messages include actionable remediation hints
- Remote image downloads retry once on transient network errors
- WebSocket protocol upgraded to JSON messages supporting reload, css-update, and build-error types
- `serve` builds to a temporary directory and atomically swaps on success; failed builds preserve the previous output
- fsnotify watcher automatically monitors newly created directories
- CONTRIBUTING.md updated to bilingual format

### Fixed

- Fixed TOC generation anomaly with deeply nested sub-chapters
- Fixed unhandled timeout in remote image base64 embedding
- Fixed GitBook-style `.md` link jumps in single-page HTML/PDF output
- Fixed GitBook-style `.md` link jumps in multi-page site output
- Added explicit annotation for Markdown links outside the build graph
- Fixed WebSocket debounce closure capturing stale event variable
- Fixed CSS-only detection race condition when multiple file types change rapidly

---

## [0.1.0] - 2026-03

### Added

- **Markdown to PDF conversion** via Chromium rendering engine (chromedp)
- **Full GFM support**: tables, task lists, footnotes, strikethrough, autolinks (via goldmark)
- **`book.yaml` config system**: book metadata, chapter list, style settings, output options
- **Auto-generated TOC**: built from heading hierarchy with page numbers and links
- **Cover page generation**: title, author, version, date, cover image, custom background color
- **Syntax highlighting**: powered by Chroma with 100+ language support (monokai, github, dracula, solarized-dark)
- **Multi-chapter assembly**: ordered chapter definitions with nested sub-chapters
- **Theme system**: built-in `technical`, `elegant`, and `minimal` themes
- **`mdpress themes list`** and **`mdpress themes show`** commands
- **Image handling**: local and remote images auto-embedded as base64
- **Cross-references**: figure/table IDs (`{#fig:id}`) and references (`{{ref:id}}`)
- **Headers and footers**: customizable with template variables (`{{.PageNum}}`, `{{.Book.Title}}`, `{{.Chapter.Title}}`)
- **Multiple page sizes**: A4, A5, Letter, Legal, B5
- **Custom CSS**: external CSS overrides via `style.custom_css`
- **GLOSSARY.md**: auto-highlight terms with tooltips and generate glossary appendix
- **Variable expansion**: `{{ book.title }}` and similar template variables in Markdown
- **`mdpress init`**: initialize book project with sample configuration
- **CI/CD integration**: GitHub Actions workflow support
- **Makefile**: build, test, lint, coverage, and release targets
- **Cross-compilation**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

---

[Unreleased]: https://github.com/yeasy/mdpress/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/yeasy/mdpress/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/yeasy/mdpress/releases/tag/v0.1.0
