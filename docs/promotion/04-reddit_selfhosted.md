# Reddit r/selfhosted

## Title

mdpress — self-host your own book publishing pipeline: Markdown to PDF/HTML/ePub, single binary, no cloud dependencies

## Post

If you write documentation or technical books in Markdown and want to keep the entire publishing pipeline local, [**mdpress**](https://github.com/yeasy/mdpress) might be useful.

### What it is

A single Go binary that converts Markdown files into:

- **PDF** — print-ready, with cover page, TOC, page numbers, headers/footers
- **HTML** — self-contained single-page document
- **Site** — multi-page static website (GitBook-style, with sidebar, dark mode, search)
- **ePub** — for e-readers

No cloud service required. Everything runs locally on your machine.

### Why it matters for self-hosting

**No external dependencies to manage.** Install with `brew install mdpress` (or `go install`). No Node.js runtime, no LaTeX installation, no Python environment. The only optional dependency is Chrome/Chromium for PDF output — and you probably already have that.

**No SaaS lock-in.** GitBook went commercial. Read the Docs is great but requires uploading your content. mdpress runs entirely on your machine or your own server.

**CI/CD friendly.** Add it to GitHub Actions, GitLab CI, or any runner:

```yaml
# GitHub Actions example
- name: Install mdpress
  run: |
    brew tap yeasy/tap
    brew install mdpress

- name: Build book
  run: mdpress build ./docs --format pdf,html,site

- name: Deploy site to GitHub Pages
  uses: peaceiris/actions-gh-pages@v3
  with:
    publish_dir: ./docs/output/site
```

Or build from a GitHub repo directly:

```bash
mdpress build https://github.com/user/repo --format site
```

**Serve locally for writing and review:**

```bash
mdpress serve ./my-book
# Opens browser at localhost:3456
# Auto-reloads when you edit .md files
```

### Migration from GitBook/HonKit

If you have existing books using `SUMMARY.md` (the GitBook chapter definition format), mdpress reads it natively. Point mdpress at your existing repo and it works.

### Output examples

The site output gives you a three-column layout with sidebar navigation, dark mode toggle, code copy buttons, and full-text search — all static HTML that you can serve from nginx, Caddy, or any static file server.

### Quick start

```bash
brew tap yeasy/tap && brew install mdpress
mdpress quickstart mybook
mdpress serve mybook        # preview
mdpress build mybook --format pdf,html,site,epub  # build everything
```

MIT licensed. Written in Go.

GitHub: https://github.com/yeasy/mdpress
