# mdPress Product Roadmap

[中文说明](ROADMAP_zh.md)

> Updated: 2026-03-18
> Maintainer: mdPress product team

---

## Version Overview

```
v0.1.0 ██████████████████████████████████████████ released
v0.2.0 ██████████████████████████████████████████ released (2026-03-19)
v0.3.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ planned (target: 2026-08)
v0.4.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ planned (target: 2026-11)
v1.0.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ planned (target: 2027-Q1)
```

---

## v0.1.0 - Foundational Markdown To PDF

**Release date**: 2026-03
**Theme**: core build pipeline

v0.1.0 established the base architecture of mdPress and shipped a complete Markdown-to-PDF workflow.

### Delivered Features

| Feature | Description |
| --- | --- |
| Markdown -> PDF conversion | Built on Chromium rendering for professional-quality output |
| Full GFM support | Tables, task lists, footnotes, strikethrough, and autolinks |
| `book.yaml` config system | Book metadata, chapters, styles, and output options |
| Auto-generated TOC | Built from heading hierarchy with page numbers and links |
| Cover page generation | Title, author, version, date, image, and background support |
| Syntax highlighting | Powered by Chroma with 100+ languages |
| Multi-chapter assembly | Nested chapter definitions supported |
| Theme system | Built-in `technical`, `elegant`, and `minimal` themes |
| Image handling | Local and remote images embedded as base64 when needed |
| Cross references | Figure and table IDs plus `{{ref:id}}` support |
| Headers and footers | Template variables for page number, chapter title, and book title |
| `mdpress init` | Project initialization with sample files |
| `mdpress themes` | Theme inspection commands |
| Multiple page sizes | A4, A5, Letter, Legal, and B5 |
| Custom CSS | External CSS overrides |
| `GLOSSARY.md` | Glossary highlighting and appendix generation |
| CI/CD integration | GitHub Actions workflow support |

---

## v0.2.0 - Multi-Format Output And Usability

**Release date**: 2026-03-19
**Theme**: lower the barrier to entry and expand output capability

v0.2.0 moves mdPress from a PDF-first generator toward a multi-format publishing platform, with a strong focus on usability and migration compatibility.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Single-page HTML output | P0 | `--format html` generates a self-contained HTML document |
| GitHub repository sources | P0 | Build directly from GitHub URLs without a local clone |
| `SUMMARY.md` compatibility | P0 | Support GitBook-style chapter definition files |
| Zero-config auto-discovery | P0 | Scan Markdown files automatically when `book.yaml` is absent |
| Live preview via `serve` | P0 | Local HTTP server with file watching and WebSocket reload |
| `site` output clarification | P0 | Sharpen the responsibility split between `html`, `serve`, and `site` |
| `doctor` command | P1 | Quick environment and project-readiness checks |

### Improvements

| Improvement | Description |
| --- | --- |
| `--format` | Allow `pdf`, `html`, and comma-separated multi-format builds |
| `--output` | Provide a unified output path or prefix for builds |
| Better errors | Attach actionable remediation hints to errors |
| CI/CD templates | Ship GitHub Actions and GitLab CI templates |

### Milestones

```
2026-04-01  feature development complete, internal testing starts ✓
2026-04-15  public beta ✓
2026-03-19  v0.2.0 release ✓
```

---

## v0.3.0 - ePub Output And Plugin System

**Target release**: 2026-08
**Theme**: broader output support and extensibility

### Planned Features

| Feature | Priority | Description |
| --- | --- | --- |
| ePub 3 output | P1 | Generate standard ePub 3 books with cover, TOC, and metadata |
| Plugin system | P1 | Support Markdown preprocessors and output post-process hooks |
| LaTeX math | P2 | Render inline and block math via KaTeX or MathJax |
| Mermaid diagrams | P2 | Native Mermaid handling for PDF and HTML |
| Better i18n | P1 | Improve language landing pages, switching, and metadata |
| HTML full-text search | P1 | Client-side search powered by lunr.js |
| Custom font embedding | P2 | Allow local TTF, OTF, and WOFF2 files |
| PDF bookmarks | P2 | Generate PDF outlines from chapter structure |

### Plugin Preview

```yaml
plugins:
  - name: math
    options:
      renderer: katex
  - name: mermaid
  - name: custom-admonition
    path: ./plugins/admonition
```

Planned hook groups:

- `pre-parse`
- `post-parse`
- `post-build`

---

## v0.4.0 - Typst Backend And Incremental Builds

**Target release**: 2026-11
**Theme**: performance and zero-dependency PDF

### Planned Features

| Feature | Priority | Description |
| --- | --- | --- |
| Typst backend | P2 | Use Typst instead of Chromium for true zero-external-dependency PDF |
| Incremental builds | P2 | Rebuild only changed chapters |
| Parallel builds | P2 | Use multiple CPU cores for chapter parsing and rendering |
| PDF watermarking | P2 | Text and image watermarks |
| PDF encryption | P2 | Password protection and permission controls |
| Build cache | P2 | File-hash-based cache to avoid redundant work |

### Typst Backend Direction

| Dimension | Chromium backend | Typst backend |
| --- | --- | --- |
| External dependency | Requires Chrome or Chromium | None, if bundled or invoked via Typst CLI |
| Layout quality | Excellent web-standard rendering | Excellent professional typesetting |
| CJK support | Strong | Strong |
| Build speed | Moderate due to browser startup | Faster native compilation path |
| Binary size | Small without Chromium | Likely larger if Typst is bundled |
| CSS compatibility | Full | Requires adaptation into Typst styling |

Planned CLI direction:

- `--backend chromium` as the default
- `--backend typst` as the alternative

### Incremental Build Plan

```
.mdpress-cache/
├── manifest.json
├── ch01.html
├── ch02.html
└── ...
```

Expected flow:

1. Compute SHA-256 for every chapter
2. Compare hashes with `manifest.json`
3. Rebuild only changed chapters
4. Merge cached chapters with newly compiled chapters
5. Produce the final output

Expected payoff: rebuilding a 500-page book after editing one chapter should drop from roughly 120 seconds to under 10 seconds.

---

## v1.0.0 - Stable Release

**Target release**: 2027-Q1
**Theme**: production readiness and long-term support

### Stability Goals

| Goal | Description |
| --- | --- |
| API stability | Freeze CLI flags and `book.yaml` structure under semantic versioning |
| Test coverage | Reach at least 90% coverage for core packages |
| Documentation | Complete user manuals, bilingual docs, API docs, and migration guides |
| Performance baselines | Prevent regressions across releases |
| Platform validation | CI coverage across macOS, Linux, and Windows |
| Security review | Continuous dependency scanning and known-vulnerability control |

### Planned Features

| Feature | Description |
| --- | --- |
| `mdpress doctor` | Check Chrome version, fonts, and other environment readiness details |
| `mdpress upgrade` | Update the mdPress binary and themes |
| Official theme registry | Community-contributed theme distribution |
| Official plugin registry | Community-contributed plugin distribution |
| Full user manual | A manual built with mdPress itself |
| Migration tooling | Automated migration from GitBook, HonKit, and mdBook |

### LTS Policy

The first stable release is intended to become the first LTS version with:

- At least 12 months of bug-fix support
- At least 18 months of security-fix support
- Backward compatibility for config format
- Backward compatibility for CLI flags

---

## Longer-Term Ideas

These items are post-`v1.0.0` and will be prioritized by community demand:

| Feature | Description |
| --- | --- |
| GUI editor | Browser-based visual editor |
| Cloud build service | SaaS build service triggered from Git repositories |
| Collaborative editing | Real-time multi-user editing |
| PDF/A output | Archival-compliant PDF support |
| Print-focused output | Bleed, color management, and ICC profile support |
| DOCX output | Word document export |
| Template marketplace | Reusable design and layout presets |

---

## How To Contribute

mdPress is open source and welcomes contributions:

- Report bugs in [GitHub Issues](https://github.com/yeasy/mdpress/issues)
- Submit feature requests and note the version you are targeting
- Fork the repository and open pull requests
- Improve docs and translations
- Contribute themes

The roadmap is expected to evolve with community feedback and implementation progress.
