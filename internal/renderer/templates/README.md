# Renderer Templates

This directory is reserved for embedded HTML templates.

Currently, templates are defined as string constants in `template.go` and `html_standalone.go`.
A future refactoring (tracked in the roadmap) will migrate these to standalone `.html` files
using Go's `embed` package for better maintainability and theme customization.

## Planned Structure

```
templates/
├── pdf.html           # PDF rendering template
├── standalone.html    # Single-page HTML template
├── site/
│   ├── index.html     # Site index page
│   └── chapter.html   # Site chapter page
└── components/
    ├── cover.html     # Cover page component
    ├── toc.html       # Table of contents component
    └── nav.html       # Navigation component
```
