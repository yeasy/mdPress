# mdPress Product Roadmap

[中文说明](ROADMAP_zh.md)

> Updated: 2026-03-29
> Maintainer: mdPress product team

---

## Version Overview

```
v0.1.0 ██████████████████████████████████████████ released
v0.2.0 ██████████████████████████████████████████ released (2026-03-19)
v0.3.0 ██████████████████████████████████████████ released (2026-03-19)
v0.3.1 ██████████████████████████████████████████ released (2026-03-19)
v0.4.0 ██████████████████████████████████████████ released (2026-03-19)
v0.4.1 ██████████████████████████████████████████ released (2026-03-19)
v0.4.2 ██████████████████████████████████████████ released (2026-03-19)
v0.4.3 ██████████████████████████████████████████ released (2026-03-19)
v0.5.0 ██████████████████████████████████████████ released (2026-03-20)
v0.5.1 ██████████████████████████████████████████ released (2026-03-21)
v0.5.2 ██████████████████████████████████████████ released (2026-03-22)
v0.5.3 ██████████████████████████████████████████ released (2026-03-23)
v0.5.4 ██████████████████████████████████████████ released (2026-03-23)
v0.6.0 ██████████████████████████████████████████ released (2026-03-23)
v0.6.1 ██████████████████████████████████████████ released (2026-03-24)
v0.6.2 ██████████████████████████████████████████ released (2026-03-25)
v0.6.3 ██████████████████████████████████████████ released (2026-03-25)
v0.6.4 ██████████████████████████████████████████ released (2026-03-26)
v0.7.0 ██████████████████████████████████████████ released (2026-03-28)
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

**Release date**: 2026-03-19
**Theme**: broader output support and extensibility

v0.3.0 expanded mdPress into a true multi-format publishing platform with a plugin architecture and first-class support for math and diagrams.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| ePub 3 output | P0 | Standard ePub 3 books with cover, TOC, and metadata |
| Plugin system | P0 | Full plugin lifecycle with external process loading and hook registration |
| KaTeX math support | P1 | Inline (`$...$`) and block (`$$...$$`) LaTeX math via KaTeX |
| Mermaid diagram support | P1 | Native Mermaid rendering in all HTML-based outputs; automatic Mermaid syntax checks in `validate` |
| GitBook migration tool | P1 | `mdpress migrate` converts GitBook projects to mdPress format |
| Multi-format `all` shorthand | P1 | `--format all` builds PDF, HTML, site, and ePub in one command |
| GitHub Actions template | P2 | Pre-built workflow for automated book builds in CI |

### v0.3.1 Patch (2026-03-19)

| Fix | Description |
| --- | --- |
| CJK PDF font embedding | Inject `@font-face` rules with `file://` URLs so Chrome can embed CJK glyphs |
| TOC depth control | `output.toc_max_depth` config to limit heading depth in table of contents |
| Smart output filename | Derive output filename from book title instead of defaulting to `output.pdf` |
| Regexp performance | Promote regexp compilation to package level in crossref, glossary, markdown, and image processing |
| Git author fallback | Fall back to `git config user.name` when no author is specified |

---

## v0.4.0 - Typst Backend And Incremental Builds

**Release date**: 2026-03-19
**Theme**: performance and zero-dependency PDF

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Typst backend | P2 | Use Typst instead of Chromium for true zero-external-dependency PDF |
| Incremental builds | P2 | Rebuild only changed chapters |
| Parallel builds | P2 | Use multiple CPU cores for chapter parsing and rendering |
| PDF watermarking | P2 | Text watermarks with opacity control |
| Custom PDF margins | P2 | Per-side margin configuration in multiple units |
| PDF bookmarks | P2 | Auto-generated from heading hierarchy for better navigation |
| Build cache | P2 | File-hash-based cache to avoid redundant work |
| PlantUML support | P2 | Auto-detect and render PlantUML diagrams in code blocks |

### Typst Backend Direction

| Dimension | Chromium backend | Typst backend |
| --- | --- | --- |
| External dependency | Requires Chrome or Chromium | None, if bundled or invoked via Typst CLI |
| Layout quality | Excellent web-standard rendering | Excellent professional typesetting |
| CJK support | Strong | Strong |
| Build speed | Moderate due to browser startup | Faster native compilation path |
| Binary size | Small without Chromium | Likely larger if Typst is bundled |
| CSS compatibility | Full | Requires adaptation into Typst styling |

CLI direction (implemented):

- `--format pdf` uses Chromium (default PDF backend)
- `--format typst` uses Typst as an alternative PDF backend

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

## v0.5.0 - PlantUML Local Rendering And Test Infrastructure

**Release date**: 2026-03-20
**Theme**: offline capability and test coverage foundation

v0.5.0 closes the most critical gap for enterprise users (offline PlantUML rendering), repairs the release CI pipeline, and establishes a golden test framework to prevent backend regressions.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| PlantUML local rendering | P1 | `renderLocal()` invokes local `plantuml` CLI or `plantuml.jar`; enabled via `plantuml.use_local: true` |
| Golden test framework | P2 | Snapshot-based regression tests in `tests/golden/`; regenerate with `-update` flag |
| Doctor PlantUML check | P1 | `mdpress doctor` detects local PlantUML availability and prints install hints |
| CI Node.js 24 upgrade | P0 | `goreleaser-action@v7`, `codecov-action@v5` ahead of 2026-06-02 deadline |
| Release CI Docker fix | P0 | Removed stale Docker Hub login step; images publish exclusively to GHCR |

### Test Coverage Milestones

| Package | v0.4.3 | v0.5.0 | Delta |
| --- | --- | --- | --- |
| `internal/plantuml` | 54.8% | 75%+ | +20 pp |
| `internal/source` | 41.2% | 62%+ | +21 pp |
| `cmd` | 47.7% | 60%+ | +12 pp |
| **Overall** | **62.3%** | **≥ 68%** | **+6 pp** |

---

## v0.5.1 - Sidebar Navigation and Context Fixes

**Release date**: 2026-03-21
**Theme**: UI refinement and bug fixes

v0.5.1 delivers accordion-style sidebar navigation improvements and fixes critical context propagation issues affecting PlantUML rendering, SUMMARY.md title handling, and migration workflows.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Accordion sidebar navigation | P1 | Expanding a chapter section automatically collapses sibling sections at the same level for cleaner GitBook-like navigation |
| Smoother sidebar transitions | P2 | CSS transitions upgraded to Material Design easing curves for better expand/collapse animations |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| PlantUML context propagation | P0 | Replaced `context.Background()` with caller-provided context, ensuring build timeout and cancellation signals propagate correctly |
| Generic SUMMARY.md title filtering | P0 | Titles like "在线阅读", "Read Online", and "Contents" now correctly recognized as generic navigation headings |
| book.json title priority | P1 | When `book.json` provides a title, it is no longer overwritten by README.md inference during auto-discovery |
| Missing `migrate` command in README | P1 | Added the `migrate` command to the "All Commands" table in both English and Chinese READMEs |
| GitLab CI lint version mismatch | P2 | Aligned golangci-lint from v2.1 to v2.11.3 to match GitHub Actions |
| Misspelling in completion command | P2 | Fixed `behaviour` → `behavior` in comment |

---

## v0.5.2 - Windows Compatibility and CI Fixes

**Release date**: 2026-03-22
**Theme**: Cross-platform robustness and CI reliability

v0.5.2 improves Windows support with plugin executable resolution via PATHEXT, fixes cross-platform test failures, and hardens the CI supply chain by upgrading all GitHub Actions to their latest major versions.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Windows plugin executable resolution | P1 | Plugin paths without extensions auto-resolve via PATHEXT (`.exe`, `.bat`, `.cmd`) |
| Search focus style and a11y traps | P2 | Improved keyboard accessibility with visible focus indicators |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Cross-platform test paths | P0 | Replaced hardcoded Unix paths with `t.TempDir()` in config tests, fixing Windows CI |
| Codecov action parameter | P1 | Corrected `file` to `files` for codecov-action@v5 |
| Dependabot config syntax | P1 | Fixed invalid `pull-requests.max-number` to `open-pull-requests-limit` |
| ePub test resource leak | P2 | Added missing `reader.Close()` in epub test |

---

## v0.5.3 - Code Block Fixes and Documentation

**Release date**: 2026-03-23
**Theme**: Bug fixes and documentation alignment

v0.5.3 fixes critical code block rendering issues in site output and updates project documentation.

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Invisible code block text | P0 | Chroma syntax highlighter injected inline `style="background-color:#fff"` on `<pre>` tags, making code blocks unreadable. Fix strips chroma's inline style during post-processing. |
| Site code block color scheme | P1 | Changed site output code blocks from dark theme to light theme (`#f6f8fa` background, `#24292e` text) matching the chroma "github" palette |

### Changed

| Change | Description |
| --- | --- |
| Documentation updates | Updated ARCHITECTURE docs version, fixed ROADMAP version ordering, added Typst format to README table |
| Removed stale NEXT-STEPS.md | Deleted outdated planning document that referenced v0.4.3 as latest |

---

## v0.5.4 - Site Enhancement and Bug Fixes

**Release date**: 2026-03-23
**Theme**: Rich site features and security hardening

v0.5.4 is a major feature release for the site output format, adding client-side full-text search, dark mode, breadcrumb navigation, page TOC, code copy buttons, and SEO optimization. It also fixes several bugs including ePub resource leaks, UTF-8 truncation issues, and a symlink-based path traversal vulnerability in the dev server.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Full-text search | P0 | Client-side search with Cmd/Ctrl+K shortcut, keyboard navigation, and result highlighting |
| Dark mode toggle | P0 | Three-way theme switcher (light/dark/system) with localStorage persistence |
| SEO meta tags | P1 | Auto-generated description and Open Graph tags per page |
| Sitemap generation | P1 | `sitemap.xml` for search engine indexing |
| Breadcrumb navigation | P1 | Page hierarchy trail on each site page |
| Page TOC sidebar | P1 | "On this page" sidebar with scroll-spy via IntersectionObserver |
| Code block copy button | P1 | Hover-to-reveal copy button with clipboard integration |
| Sidebar collapse | P2 | Desktop sidebar collapsible with persistent state |
| Lazy loading images | P2 | `loading="lazy"` on all `<img>` tags automatically |
| CJK heading IDs | P1 | Custom heading ID transformer that preserves Unicode letters |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| ePub zip writer resource leak | P1 | Added `defer w.Close()` for all error paths |
| UTF-8 truncation in SVG cover | P1 | Rune-based truncation replaces byte-based slicing |
| SVG XML attribute escaping | P2 | Added `"` and `'` entity escaping |
| Symlink path traversal | P1 | `filepath.EvalSymlinks()` added to serve path check |
| Scroll behavior regression | P2 | Fixed ternary always returning `'auto'` instead of `'smooth'` |
| Description meta truncation | P2 | Rune-based truncation for multi-byte character safety |

---

## v0.6.0 - Self-Upgrade, Doctor Enhancement, and User Manual

**Release date**: 2026-03-23
**Theme**: production readiness foundation

v0.6.0 bridges the gap between feature-driven development (v0.1–v0.5) and the production-ready v1.0.0 release.

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| `mdpress upgrade` command | P0 | Self-upgrade from GitHub releases with platform detection, SHA-256 checksum verification, and `--check` dry-run mode |
| Enhanced `mdpress doctor` | P0 | Six new environment checks: Go version (≥1.25), Git availability, network connectivity, disk space, CJK font detection, and plugin health; new `--verbose` flag |
| Bilingual user manual | P0 | Complete Chinese + English user manual (60+ Markdown files) built with mdPress itself |
| `ParseVersionPart` utility | P2 | Reusable version-string parser for `doctor` and `upgrade` commands |

### Improvements

| Improvement | Priority | Description |
| --- | --- | --- |
| Path traversal hardening | P1 | `LocalSource.Prepare()` validates subdirectory paths against traversal attacks |
| Cross-platform path handling | P1 | `filepath.Join` replaces string concatenation in `HasLangsFile` for Windows compatibility |
| Documentation updates | P2 | `upgrade` command added to README command table (EN + ZH) and COMMANDS docs |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Lint cleanup | P2 | Removed duplicate test helpers and unused imports across `cmd/*_test.go` |
| Chinese text truncation test | P2 | Increased input size so the ≥160-rune truncation path is exercised |

### Tests

- 1,500+ new test lines across 12 files
- Full coverage of `upgrade` command: version comparison, asset selection, download, and binary replacement
- Expanded `doctor`, `cmd`, `themes`, `quickstart`, `validate` tests
- Comprehensive utility function tests for `file`, `cjk`, `image` packages
- Expanded plugin lifecycle and error-path tests

---

## v0.6.1 - Bug Fixes and Documentation

**Release date**: 2026-03-24
**Theme**: quality and consistency

v0.6.1 is a patch release that fixes bugs discovered in v0.6.0 and improves documentation coverage.

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Typst timeout configuration | P0 | `compileToPDF()` now uses the configured `g.timeout` instead of a hardcoded 120-second value |
| Format validation for Typst | P0 | Added `typst` to `validFormats` so `book.yaml` accepts `typst` as an output format |
| Git branch name validation | P1 | Branch name regex now requires leading alphanumeric character to prevent CLI flag injection |
| ePub cleanup error handling | P1 | `os.Remove` errors on failed builds are now logged instead of silently ignored |
| Error message consistency | P2 | Replaced remaining Chinese error messages with English in `parser.go` and `crossref.go` |
| GoReleaser Homebrew URL case | P2 | Corrected `mdPress` to `mdpress` in the Homebrew Cask verified URL |
| Gosec exclusion ordering | P2 | Sorted gosec rule exclusions numerically in `.golangci.yml` |

### Documentation

| Change | Description |
| --- | --- |
| Doctor command docs | Updated `doctor.md` and `doctor_zh.md` with all v0.6.0 environment checks |
| User manual links in README | Added links to the bilingual user manual in both READMEs |
| `version` command in COMMANDS | Added the `version` command to the command matrix and hierarchy diagram |
| ROADMAP update | Added v0.6.0 and v0.6.1 release notes |
| `.gitignore` enhancement | Added `.agent/`, `.env`, and credential file patterns |

---

## v0.6.2 - Security Hardening And Bug Fixes

**Release date**: 2026-03-25
**Theme**: security and correctness

v0.6.2 is a security-focused release that hardens the preview server, upgrade command, image handling, and theme CSS against injection and traversal attacks. It also fixes numerous parser, cache, and rendering bugs.

### Security Fixes

| Fix | Priority | Description |
| --- | --- | --- |
| WebSocket origin validation | P0 | Preview server validates `Origin` header against `Host` to prevent cross-origin hijacking |
| Upgrade URL domain validation | P0 | Binary downloads verify the URL points to `github.com` or `*.githubusercontent.com` |
| Absolute image path rejection | P1 | `resolveLocalImagePath` rejects absolute paths to prevent reading arbitrary local files |
| Theme CSS injection prevention | P1 | Theme color and font values validated against unsafe characters before CSS output |
| CSS color pattern tightened | P1 | `rgb()`/`hsl()` patterns restrict content to safe characters |
| Security headers | P2 | Preview server sets `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Chapter cache key mismatch | P0 | Cache key uses same fallback logic as parser, preventing stale hits |
| PlantUML encoding | P0 | Correct raw deflate and PlantUML custom 6-bit encoding alphabet |
| Heading ID race condition | P0 | Each `Transform` call uses a local `usedIDs` map instead of shared state |
| Path traversal via book.json | P1 | `safeJoin()` rejects absolute and escaping paths |
| Typst font size fallback | P1 | Parse failures fall back to `12pt` instead of invisible `0.0pt` |
| Glossary double-wrapping | P1 | Overlapping terms no longer create nested `<span>` tags |
| UTF-8 title capitalization | P1 | Uses `[]rune` + `unicode.ToUpper` for multi-byte first characters |
| Typst template injection | P1 | User content with `{{ }}` no longer panics or injects code |

### Changed

| Change | Description |
| --- | --- |
| Triple config load eliminated | `doctor` loads config once and passes it through |
| Dead code removed | Unused `CacheStatistics`, no-op `convertCodeSpans`, duplicate `fileExists` removed |
| Glossary regex hoisted | `skipPattern` compiled once at package level |
| Search index optimization | `utf8.RuneCountInString()` replaces `len([]rune(...))` |

### Documentation

| Change | Description |
| --- | --- |
| validate command | Documented the `--report` flag |
| completion command | Corrected `--no-descriptions` support from `bash`/`zsh` to `bash`/`fish` |
| build command | Documented that `--format all` expands to `pdf,html,site,epub,typst` |
| upgrade command | Removed fabricated exit codes section |

---

## v0.6.3 - Security Hardening And Search Redesign

**Release date**: 2026-03-25
**Theme**: security and search UX

v0.6.3 hardens the codebase against SSRF, XSS, path traversal, and template injection attacks. It also redesigns the search UI as a right-side panel.

### Security Fixes

| Fix | Priority | Description |
| --- | --- | --- |
| SSRF prevention for PlantUML | P0 | Validates PlantUML server URLs against private/loopback IPs via DNS resolution |
| Mermaid XSS fix | P1 | Re-escapes HTML entities after unescaping in Mermaid code blocks |
| EPUB path traversal prevention | P1 | Rejects absolute image paths and validates relative paths stay within source directory |
| Tar path traversal prevention | P1 | Skips tar entries containing `..` during upgrade extraction |
| Template injection prevention | P1 | Strips `{{` and `}}` from Typst metadata and dimension fields |
| Config field validation | P1 | Validates `font_family`, `font_size`, and `code_theme` against injection patterns |
| Custom CSS size limit | P2 | Limits custom CSS file reads to 1 MB |
| URL scheme validation | P2 | `openBrowser` only allows `http` and `https` schemes |

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| Search broken on subpages | P0 | Search index fetched with absolute path instead of relative |
| Search result links wrong | P0 | Search result hrefs use absolute paths for correct navigation |
| Off-by-one bounds check | P1 | Fixed submatch access in EPUB and image regex |
| Unchecked type assertion | P1 | PlantUML cache uses comma-ok pattern |
| GitHub tempdir leak | P1 | `Prepare()` cleans up temp directory on validation failure |

### Changed

| Change | Description |
| --- | --- |
| Search redesigned as right-side panel | Search opens as a GitBook-style right panel instead of a modal overlay |
| Defensive slice copy | `Plugins()` returns a copy to prevent external mutation |
| Goroutine panic recovery | Image prefetch goroutines recover from panics |

---

## v0.6.4 - PDF Rendering Fixes

**Release date**: 2026-03-26
**Theme**: PDF image and diagram rendering

v0.6.4 fixes critical PDF rendering issues including missing images, broken Mermaid diagrams, invisible SVG badge text, and incorrect layout.

### Fixed Issues

| Fix | Priority | Description |
| --- | --- | --- |
| PDF images not rendering | P0 | Strip `loading="lazy"` from images before PDF generation |
| Mermaid diagrams missing text | P0 | Remove HTML re-escaping that broke arrows and tags in Mermaid |
| Mermaid digits/Latin missing in PDF | P1 | Add Latin fonts before CJK fonts in Mermaid SVG CSS rules |
| SVG badge CJK text missing in PDF | P1 | Inline CJK-containing SVGs with embedded font-face |
| Badge images stacked vertically | P1 | Block display only for standalone images |
| mdPress docs injected into PDF | P1 | Filter CHANGELOG.md, CONTRIBUTING.md, LICENSE.md from auto-discovery |
| Cover version defaults to 1.0.0 | P1 | Read version from book.json with git describe fallback |
| Duplicate branding on cover | P2 | Remove inline brand footer from cover template |

---

## v0.7.0 - Site UX Enhancements (Released 2026-03-28)

**Theme**: site output UX improvements

### Delivered Features

| Feature | Priority | Description |
| --- | --- | --- |
| Previous/Next navigation | P0 | Bottom-of-page buttons linking to the previous and next chapters for continuous reading |
| "Built with mdPress" branding | P1 | Subtle footer link in site output; localized as "使用 mdPress 构建" for Chinese books |
| Collapsible sidebar sections | P2 | Expand/collapse arrows for chapters with sub-pages in the sidebar |

### Deferred to Future Release

| Feature | Priority | Description |
| --- | --- | --- |
| Sidebar chapter grouping | P2 | Support `parts` in `book.yaml` to group chapters under collapsible section headers (e.g. "Part 1: Getting Started") |

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
| Official theme registry | Community-contributed theme distribution |
| Official plugin registry | Community-contributed plugin distribution |
| Migration tooling | Automated migration from mdBook (GitBook/HonKit migration already available via `mdpress migrate`) |

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
