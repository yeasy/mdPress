---
title: "mdpress: Why I Built a Single-Binary Markdown Book Publisher in Go"
published: true
description: "A Go CLI tool that converts Markdown books to PDF, HTML, ePub, and static sites — with zero runtime dependencies."
tags: go, markdown, opensource, publishing
cover_image: # TODO: add cover image URL
canonical_url: https://github.com/yeasy/mdpress
---

## The Problem

I maintain several open-source technical books written in Markdown. Over the years, I've cycled through multiple publishing tools, and each one came with friction:

**GitBook** was great — until it went fully commercial and dropped the open-source CLI. Existing books had to either migrate or stay frozen.

**HonKit** (the GitBook fork) kept things alive but development slowed. It still requires Node.js, and the npm dependency tree is heavy.

**mdBook** is excellent for Rust-ecosystem documentation, but it doesn't output PDF. If you need a printable book, you're on your own.

**Pandoc** can do everything, but producing a good-looking PDF requires LaTeX, which means installing a multi-gigabyte TeX distribution and learning its templating system.

What I actually wanted was simple: one tool that reads my existing Markdown files, respects my existing `SUMMARY.md` from GitBook, and outputs PDF, HTML, and ePub — without requiring me to install Node.js, LaTeX, or Python.

So I built **mdpress**.

## What mdpress Does

mdpress is a command-line tool written in Go. You give it a folder of Markdown files, and it produces:

- **PDF** — professional-quality, with cover page, table of contents, page numbers, headers/footers
- **HTML** — a single self-contained HTML file
- **Site** — a multi-page static website with three-column GitBook-style layout, dark mode, code copy buttons, and full-text search
- **ePub** — standard ePub for e-readers

It's a single binary. Install it, and it works.

```bash
brew tap yeasy/tap && brew install mdpress
```

## Quick Start

```bash
# Create a sample project
mdpress quickstart mybook

# Preview with live reload
mdpress serve mybook

# Build everything
mdpress build mybook --format pdf,html,site,epub
```

That's it. No config file required — mdpress auto-discovers `.md` files in the folder.

## Key Design Decisions

### Zero-config by default, configurable when needed

mdpress uses a three-tier config discovery chain:

1. If `book.yaml` exists, use it (full control over chapters, metadata, themes, output)
2. If `SUMMARY.md` exists (GitBook format), parse it for chapter structure
3. Otherwise, auto-scan all `.md` files in the directory

This means you can point mdpress at any folder of Markdown files and get output immediately, but you can also define a detailed `book.yaml` when you need precise control:

```yaml
book:
  title: "My Technical Book"
  author: "Author Name"
  language: en
chapters:
  - title: "Introduction"
    file: "intro.md"
  - title: "Architecture"
    file: "architecture.md"
    sections:
      - title: "Backend Design"
        file: "backend.md"
style:
  theme: technical
output:
  format: [pdf, html]
  toc: true
  cover: true
  page_size: A4
```

### GitBook compatibility as a first-class feature

Many existing Markdown books use `SUMMARY.md` for chapter organization and `GLOSSARY.md` for term definitions. mdpress reads both natively. If you're migrating from GitBook or HonKit, your existing structure works without changes.

### Single binary, minimal dependencies

The entire tool compiles to a single Go binary (~15 MB). The only optional external dependency is Chrome/Chromium — and only if you want PDF output. HTML, site, and ePub generation work with zero external dependencies.

## Architecture Overview

The internal architecture follows a pipeline model:

```
Source Loading → Config Discovery → Markdown Parsing → Post-Processing → Assembly → Output
```

The Go libraries powering the pipeline:

- **Goldmark** — CommonMark-compliant Markdown parser with GFM extensions
- **Chroma** — syntax highlighting for 100+ languages
- **chromedp** — headless Chrome DevTools Protocol for PDF rendering
- **Cobra** — CLI framework
- **fsnotify** — file watching for the `serve` command

The output layer uses a `FormatBuilderRegistry` pattern — each output format (PDF, HTML, Site, ePub) implements an `OutputFormat` interface and registers itself. Adding new formats requires no modifications to the core pipeline.

The source layer is similarly abstracted behind a `Source` interface. Today it supports local filesystem and GitHub repositories (build directly from a GitHub URL). GitLab and other providers can be added by implementing the same interface.

## How It Compares

| Feature | mdpress | GitBook (OSS) | mdBook | Pandoc |
|---|---|---|---|---|
| PDF output | Yes | Yes | No | Yes |
| HTML output | Yes | Yes | Yes | Yes |
| ePub output | Yes | Yes | No | Yes |
| Static site | Yes | Yes | Yes | No |
| Runtime deps | None (Go binary) | Node.js | None (Rust binary) | LaTeX for PDF |
| SUMMARY.md | Yes | Yes | Yes (similar) | No |
| Dark mode site | Yes | Yes | Yes | N/A |
| Zero-config | Yes | No | No | No |
| Live preview | Yes | Yes | Yes | No |
| Active development | Yes | Discontinued | Yes | Yes |
| License | MIT | Apache 2.0 | MPL 2.0 | GPL 2.0 |

## Features Worth Highlighting

**`mdpress serve`** starts a local server with WebSocket-based hot reload. Edit your Markdown, save, and the browser refreshes automatically. The served site has sidebar navigation, dark mode, and search.

**GitHub source support** lets you build directly from a repo URL:

```bash
mdpress build https://github.com/user/repo --format pdf
```

It supports `GITHUB_TOKEN` for private repositories.

**Cross-references and glossary** — define figure and table IDs, reference them with `{{ref:id}}`, and they get auto-numbered. Define terms in `GLOSSARY.md` and they're highlighted with tooltips throughout the book.

**Three built-in themes** — `technical`, `elegant`, and `minimal` — plus custom CSS support for full styling control.

**Variable expansion** — use `{{ book.title }}`, `{{ book.author }}`, and other template variables in your Markdown.

## What's Next

The [roadmap](https://github.com/yeasy/mdpress/blob/main/docs/ROADMAP.md) has some interesting items planned:

**v0.3.0** (target: August 2026) — Plugin system with lifecycle hooks, LaTeX math rendering, client-side search improvements, and custom font embedding.

**v0.4.0** (target: November 2026) — A Typst backend as an alternative to Chromium, which would make PDF generation truly zero-external-dependency. Also planned: incremental builds and parallel chapter processing for large books.

**v1.0.0** (target: Q1 2027) — API stability freeze, 90%+ test coverage, official theme and plugin registries.

## Try It

```bash
# Install
brew tap yeasy/tap && brew install mdpress
# or: go install github.com/yeasy/mdpress@latest

# Quick start
mdpress quickstart mybook
mdpress serve mybook
```

The project is MIT licensed and open for contributions. Whether it's bug reports, feature requests, theme contributions, or code — all welcome.

GitHub: [https://github.com/yeasy/mdpress](https://github.com/yeasy/mdpress)
