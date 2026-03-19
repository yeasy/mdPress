# Reddit r/golang

## Title

mdpress: a Go CLI that converts Markdown books to PDF/HTML/ePub — single binary, zero runtime dependencies

## Post

Hey r/golang,

I've been working on [**mdpress**](https://github.com/yeasy/mdpress), a Markdown-to-book publishing tool written entirely in Go. It converts a folder of Markdown files into PDF, single-page HTML, multi-page static sites, and ePub — all from one binary.

### Why I built it

I maintain several technical books in Markdown. The existing tools either require Node.js (GitBook, HonKit), don't produce PDF (mdBook), or need LaTeX (Pandoc for good-looking output). I wanted a single `go install` and done.

### Go stack and architecture

The core pipeline is straightforward:

```
Markdown → Goldmark (AST → HTML) → Post-processing → Output
```

Key libraries:

- **[yuin/goldmark](https://github.com/yuin/goldmark)** — Markdown parsing with GFM extensions, footnotes, syntax highlighting via Chroma
- **[chromedp/chromedp](https://github.com/chromedp/chromedp)** — headless Chrome for PDF rendering (only needed for PDF output; HTML/ePub/site work without Chrome)
- **[spf13/cobra](https://github.com/spf13/cobra)** — CLI framework
- **[gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)** — config parsing
- **[fsnotify](https://github.com/fsnotify/fsnotify)** — file watching for the `serve` command with WebSocket-based hot reload

The architecture follows a pipeline model with clean separation:

- `internal/source` — Source abstraction (local filesystem, GitHub repos)
- `internal/config` — Config discovery chain (book.yaml → SUMMARY.md → auto-scan)
- `internal/markdown` — Goldmark parser with extensions
- `internal/renderer` — HTML assembly (cover + TOC + chapters + CSS)
- `internal/output` — Format builders behind an `OutputFormat` interface with a registry
- `internal/pdf` — chromedp-based PDF generation with mock support for testing

One design decision I'm happy with: the `FormatBuilderRegistry` pattern. Adding a new output format just means implementing the `OutputFormat` interface and registering it — no switch-case modifications needed.

### Features

- Zero-config mode: auto-discovers `.md` files when no `book.yaml` exists
- `SUMMARY.md` compatibility for GitBook/HonKit migration
- Three built-in themes (technical, elegant, minimal)
- Cross-references, glossary, variable expansion
- GitHub source support — build from a repo URL without cloning
- `serve` with incremental builds and browser auto-refresh

### Install

```bash
brew tap yeasy/tap && brew install mdpress
# or
go install github.com/yeasy/mdpress@latest
```

### What's next

v0.3.0 (targeting August 2026) plans to add ePub 3 output improvements, a plugin system with lifecycle hooks, LaTeX math rendering, and client-side full-text search. v0.4.0 is exploring a Typst backend as an alternative to Chromium for truly zero-external-dependency PDF generation.

MIT licensed. PRs and issues welcome — especially around Go idioms, test coverage, and performance. The codebase is ~99% Go.

GitHub: https://github.com/yeasy/mdpress
