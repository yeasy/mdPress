# Reddit r/commandline

## Title

mdpress — a single-binary CLI to turn Markdown files into PDF, HTML, ePub, and static sites

## Post

I made a CLI tool called [**mdpress**](https://github.com/yeasy/mdpress) for publishing Markdown books. It's a single Go binary — no Node.js, no LaTeX, no Python dependencies.

### Quick taste

```bash
# Install
brew tap yeasy/tap && brew install mdpress

# Create a sample book and preview it immediately
mdpress quickstart mybook
# opens browser at localhost:3456 with live reload

# Build all formats at once
mdpress build mybook --format pdf,html,epub

# Build from an existing folder of .md files (zero config)
mdpress build ./my-markdown-folder --format html

# Build directly from a GitHub repo
mdpress build https://github.com/user/repo --format pdf

# Validate your project setup
mdpress validate ./mybook

# Check your environment
mdpress doctor

# Explore built-in themes
mdpress themes list
mdpress themes preview technical
```

### CLI commands

| Command | What it does |
|---|---|
| `mdpress build [source]` | Build PDF, HTML, site, or ePub |
| `mdpress serve [source]` | Live preview with hot-reload |
| `mdpress quickstart [name]` | Scaffold a sample project |
| `mdpress init [dir]` | Auto-generate `book.yaml` from existing files |
| `mdpress validate [dir]` | Check config and file structure |
| `mdpress doctor [dir]` | Verify environment (Chrome, fonts, etc.) |
| `mdpress themes list\|show\|preview` | Browse built-in themes |

### What I like about the workflow

**Zero-config start**: Just point it at a folder with `.md` files. It auto-discovers chapters and builds. No config file needed unless you want to customize.

**`serve` is great for writing**: File watcher + WebSocket = your browser refreshes when you save. The output is a three-column GitBook-style site with sidebar, dark mode toggle, code copy buttons, and search.

**Multiple formats, one command**: `--format pdf,html,epub` generates all three in one pass.

**GitBook migration**: If you have an existing `SUMMARY.md` from GitBook or HonKit, mdpress picks it up automatically. No rewriting needed.

### Example `book.yaml` (optional)

```yaml
book:
  title: "My Technical Book"
  author: "Your Name"
  language: en
chapters:
  - title: "Getting Started"
    file: "chapters/getting-started.md"
  - title: "Advanced Topics"
    file: "chapters/advanced.md"
    sections:
      - title: "Deep Dive"
        file: "chapters/deep-dive.md"
style:
  theme: technical
output:
  format: [pdf, html]
  toc: true
  cover: true
```

It's MIT-licensed and written in Go. Feedback welcome.

GitHub: https://github.com/yeasy/mdpress
