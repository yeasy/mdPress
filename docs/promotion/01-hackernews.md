# Show HN: mdpress – Convert Markdown books to PDF/HTML/ePub with a single binary

I built **mdpress**, a CLI tool written in Go that turns a folder of Markdown files into a professionally formatted PDF, a self-contained HTML page, a multi-page static site (GitBook-style), or an ePub — all from a single binary with zero runtime dependencies (no Node.js, no LaTeX, no Python).

The problem it solves: I maintain several open-source books written in Markdown. GitBook went commercial, HonKit is unmaintained, and mdBook doesn't output PDF. I wanted one tool that reads my existing `SUMMARY.md`, outputs multiple formats, and doesn't require me to install an entire Node.js ecosystem. So I wrote mdpress in Go.

Key features:

- **Single binary, zero dependencies** — `brew install mdpress` and you're done
- **Multi-format output** — PDF (via headless Chromium), single-page HTML, multi-page site, ePub
- **GitBook/HonKit compatible** — reads your existing `SUMMARY.md` and `GLOSSARY.md`
- **Zero-config mode** — point it at a folder of `.md` files and it auto-discovers chapters
- **`mdpress serve`** — live preview with file watching and WebSocket hot-reload
- **Three-column site output** — sidebar navigation, dark mode, code copy, full-text search
- **Built-in themes** — technical, elegant, minimal
- **GitHub source support** — build directly from a GitHub repo URL without cloning

Architecture: Goldmark for Markdown parsing, Chroma for syntax highlighting, chromedp for PDF generation. The whole thing compiles to ~15 MB.

Install: `brew tap yeasy/tap && brew install mdpress`

Quick start:
```
mdpress quickstart mybook
mdpress serve mybook
mdpress build mybook --format pdf,html,epub
```

GitHub: https://github.com/yeasy/mdpress

MIT licensed. Feedback and contributions welcome. Happy to answer any questions about the Go implementation or design decisions.
