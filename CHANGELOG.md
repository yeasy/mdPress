# Changelog

All notable changes to this project will be documented in this file. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.6.6] - 2026-03-27

### Fixed

- **Duplicate chapter title in PDF**: Sub-chapters using h2–h6 in Markdown had their heading duplicated because the deduplication logic only matched h1 tags; now matches any heading level
- **SUMMARY.md parsing picks up inline links**: Navigation prose lines (e.g., `* \`A轨\`：从 [第一章](…) → …`) were incorrectly parsed as chapter entries; now only list items with a direct link are accepted

### Added

- **Title mismatch warning**: New `book-title-mismatch` rule warns when a SUMMARY.md title differs from the file's first heading; SUMMARY title takes precedence in rendered output

---

## [0.6.5] - 2026-03-27

### Security

- **Plugin path traversal enforced**: Relative plugin paths resolving outside the project directory are now rejected instead of only logging a warning
- **Mermaid XSS defense-in-depth**: Strip `<script>` tags and event handler attributes from Mermaid diagram content after HTML unescaping, protecting against XSS if Mermaid JS fails to load
- **Typst output capture bounded**: Limit captured stdout/stderr from `typst compile` to 1 MB, preventing OOM from malicious documents
- **Upgrade redirect SSRF protection**: Validate HTTP redirect targets during upgrade downloads to prevent SSRF via DNS poisoning
- **Git command timeouts**: Add 5-second timeout to `git describe` and `git config` calls in auto-discovery to prevent indefinite blocking

### Fixed

- **GitBook plugin migration errors**: Skip GitBook npm plugins during `book.json` migration instead of generating invalid configs with missing paths
- **Search index heading snippets**: Compute rune slice once per chapter instead of per heading, fixing O(N×M) allocation for chapters with many headings
- **Inline SVG badge height**: Fix badge height mismatch between inline SVG and `<img>` rendering

### Changed

- Remove tautological tests that produced false coverage

---

## [0.6.4] - 2026-03-26

### Fixed

- **PDF images not rendering**: Strip `loading="lazy"` from images before PDF generation, as Chrome's PrintToPDF skips lazy-loaded off-viewport images
- **Mermaid diagrams missing text**: Remove HTML re-escaping in `processMermaid` that broke arrows (`-->`), quotes, and `<br/>` tags; rely on Mermaid's `securityLevel:'strict'` (DOMPurify) for XSS protection
- **Mermaid digits/Latin chars missing in PDF**: Add Latin fonts (`-apple-system`, `Helvetica`, `Arial`) before CJK fonts in Mermaid SVG CSS rules for PDF rendering
- **SVG badge CJK text missing in PDF**: Inline CJK-containing SVGs into HTML (instead of `<img src="data:...">`), inject `CJK-Embedded` @font-face via `<style>` inside SVG, and remove `textLength` constraints
- **Badge images stacked vertically**: Change `img` CSS from unconditional `display:block` to only apply block display for standalone images (sole child of paragraph)
- **mdPress docs injected into PDF**: Filter `CHANGELOG.md`, `CONTRIBUTING.md`, `LICENSE.md` from auto-discovery and init scanning
- **Cover version defaults to 1.0.0**: Read `version` field from `book.json`, add `git describe --tags` as fallback
- **Duplicate "Built with mdPress" on cover**: Remove inline brand footer from cover HTML template; fix typo "Build with" → "Built with" in PDF footer

### Changed

- Bump parsed chapter cache version to v2 (invalidates stale caches with old Mermaid escaping)

---

## [0.6.3] - 2026-03-25

### Security

- **SSRF prevention for PlantUML**: Validates PlantUML server URLs against private/loopback IPs via DNS resolution
- **Mermaid XSS fix**: Re-escapes HTML entities after unescaping in Mermaid code blocks
- **EPUB path traversal prevention**: Rejects absolute image paths and validates relative paths stay within source directory
- **Tar path traversal prevention**: Skips tar entries containing `..` during upgrade extraction
- **Template injection prevention**: Strips `{{` and `}}` from Typst metadata and dimension fields
- **Config field validation**: Validates `font_family`, `font_size`, and `code_theme` against injection patterns
- **Custom CSS size limit**: Limits custom CSS file reads to 1 MB to prevent memory exhaustion
- **URL scheme validation**: `openBrowser` only allows `http` and `https` schemes

### Fixed

- **Search broken on subpages**: Search index was fetched with relative path, causing 404 on pages in subdirectories
- **Search result links wrong**: Search result hrefs were relative, causing incorrect navigation from subpages
- **Off-by-one bounds check**: Fixed `len(matches) < 3` to `< 4` in EPUB and image regex submatch access
- **Unchecked type assertion**: PlantUML cache `sync.Map` load now uses comma-ok pattern
- **Ignored JS exception**: PDF font loading exception is now logged instead of silently discarded
- **Ignored filepath.Abs error**: Doctor command now handles `filepath.Abs` failure gracefully
- **GitHub tempdir leak**: `Prepare()` now cleans up temp directory on validation failure
- **CrossRef priority mismatch**: `GetAllReferences` now matches `Resolve` priority order

### Changed

- **Search redesigned as right-side panel**: Search opens as a GitBook-style right panel instead of a modal overlay
- **Search auto-close on mouse leave**: Panel closes when mouse leaves (unless input is focused)
- **Defensive slice copy**: `Plugins()` now returns a copy to prevent external mutation
- **Goroutine panic recovery**: Image prefetch goroutines now recover from panics
- **Debounce context check**: File watcher callbacks check context cancellation before rebuilding
- **HTTP client timeout**: Doctor network check now has a 5-second timeout
- **Hex color support**: `isLightColor` now handles 4-char and 8-char hex codes

### Performance

- **Package-level MIME map**: Image MIME type map allocated once instead of per-call
- **strings.Builder**: Theme preview HTML generation uses Builder instead of concatenation
- **Shared line splitting**: Diagnostics splits source lines once instead of twice
- **Pre-compiled regexes**: Config validation regexes compiled at package level

### Documentation

- Updated ROADMAP and ARCHITECTURE for v0.6.2 release
- Added missing `upgrade`, `completion`, `version` commands to architecture diagram
- Fixed ePub version from "2.0" to "3" in Chinese architecture doc
- Normalized CHANGELOG heading from `Docs` to `Documentation`

### Tests

- Converted `t.Logf` to `t.Errorf`/`t.Fatalf` across 42 test files
- Added real assertions to previously assertion-less tests
- Replaced `t.Logf` + `return` with `t.Skipf` for proper skip reporting

---

## [0.6.2] - 2026-03-25

### Security

- **WebSocket origin validation**: Preview server now validates the `Origin` header against the request `Host`, preventing cross-origin WebSocket hijacking
- **Upgrade URL domain validation**: Binary downloads now verify the URL points to `github.com` or `*.githubusercontent.com` before fetching
- **Absolute image path rejection**: `resolveLocalImagePath` rejects absolute paths to prevent reading arbitrary local files
- **Theme CSS injection prevention**: Theme color and font values from user YAML are validated against unsafe characters before CSS output
- **CSS color pattern tightened**: `rgb()`/`hsl()` patterns now restrict parenthesized content to digits, commas, spaces, dots, and percent signs
- **Security headers**: Preview server now sets `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`

### Fixed

- **Chapter cache key mismatch**: Cache key computation now uses the same fallback logic as the parser, preventing stale cache hits when the code theme defaults to "github"
- **PlantUML encoding**: Replaced zlib with raw deflate and implemented the correct PlantUML custom 6-bit encoding alphabet
- **Heading ID race condition**: Eliminated shared mutable state in `headingIDTransformer`; each `Transform` call now uses a local `usedIDs` map
- **Path traversal via book.json**: Added `safeJoin()` helper that rejects absolute paths and paths escaping the project directory
- **Typst font size fallback**: `fmt.Sscanf` parse failures now fall back to `12pt` instead of producing invisible `0.0pt` text
- **Glossary double-wrapping**: Overlapping terms no longer create nested `<span>` tags
- **UTF-8 title capitalization**: `fileNameToTitle` and `inferBookTitle` now use `[]rune` + `unicode.ToUpper` for multi-byte first characters
- **Double-parse in validate**: `validateChapterContentAndSequence` now captures HTML from `ParseWithDiagnostics` instead of discarding it and re-parsing
- **Panic on empty headings**: Fixed index-before-bounds-check in e2e test that would panic when `headings` slice was empty
- **Sidebar title matching**: Changed case-sensitive `==` to `strings.EqualFold` for sidebar chapter deduplication
- **404 redirect method**: Changed from 307 (preserves POST) to 302 (converts to GET) for non-existent page redirects
- **Typst template injection**: User content with `{{ }}` no longer panics or injects code via `text/template`

### Changed

- **Triple config load eliminated**: `doctor` command now loads `config.Load()` once and passes the result to `checkDiskSpace` and `checkPlugins`
- **Dead code removed**: Unused `CacheStatistics`, no-op `convertCodeSpans`, and duplicate `fileExists` in migrate.go removed
- **Glossary regex hoisted**: `skipPattern` compiled once at package level instead of per-call
- **Search index optimization**: Replaced `len([]rune(...))` with `utf8.RuneCountInString()` for search snippet slicing
- **Font family formatting**: Fixed double-space issue in `quoteFontFamily` CSS output

### Documentation

- **validate command**: Documented the `--report` flag (was listed as "no dedicated flags")
- **completion command**: Corrected `--no-descriptions` support from `bash`/`zsh` to `bash`/`fish`
- **build command**: Noted that `--format all` expands to `pdf,html,site,epub` and excludes `typst`
- **upgrade command**: Removed fabricated exit codes section

---

## [0.6.1] - 2026-03-24

### Changed

- **Site generator refactor**: Split monolithic `site.go` (2,788 lines) into focused modules — `site.go` (core logic), `site_sidebar.go` (navigation), `site_ui.go` (i18n strings), and `site_template.go` (HTML/CSS/JS template)

### Security

- **Path traversal validation**: `validateFilename()` now rejects absolute paths and `..` components in chapter filenames, preventing writes outside the output directory
- **safeHTML trust boundary**: Added security documentation for template functions that bypass HTML escaping

### Tests

- **Parser integration tests**: 6 new test functions covering complex Markdown, edge cases, CJK content, nested blockquotes, and mixed HTML
- **Theme validation tests**: 11 new test functions covering structure integrity, color validation, CSS syntax, name uniqueness, and YAML loading

---

## [0.6.0] - 2026-03-23

### Added

- **`mdpress upgrade` command**: Self-upgrade from GitHub releases with platform detection, SHA-256 checksum verification, and `--check` dry-run mode (`cmd/upgrade.go`, 436 lines + 788-line test suite)
- **Enhanced `mdpress doctor`**: Six new environment checks — Go version (≥1.25), Git availability, network connectivity to github.com, disk space, CJK font detection, and plugin health; new `--verbose` flag for detailed output
- **Bilingual user manual**: Complete Chinese + English user manual (60+ Markdown files) built with mdPress itself, covering getting started, user guide, themes, plugins, best practices, troubleshooting, and CLI reference
- **`ParseVersionPart` utility**: Reusable version-string parser in `pkg/utils/file.go` for `doctor` and `upgrade` commands

### Changed

- **Path traversal hardening**: `LocalSource.Prepare()` now validates subdirectory paths against traversal attacks (rejects absolute paths and `..` components)
- **Cross-platform `HasLangsFile`**: Replaced string concatenation with `filepath.Join` in `internal/i18n/langs.go` for correct path handling on Windows
- **Documentation**: Added `upgrade` command to README command table (EN + ZH) and COMMANDS docs

### Fixed

- **Lint cleanup**: Removed 4 duplicate `suppressOutput` test helpers and unused imports across `cmd/*_test.go`; simplified `else { if }` to `else if` in `internal/plugin/external_test.go`
- **Chinese text truncation test**: Increased input from 131 to 191 runes so the ≥160-rune truncation path is actually exercised

### Tests

- **1,500+ new test lines** across 12 files:
  - `cmd/upgrade_test.go` (788 lines): full coverage of version comparison, asset selection, download, and in-place binary replacement
  - `cmd/doctor_test.go`: expanded tests for new Go/Git/network/disk/plugin checks
  - `cmd/cmd_test.go`, `cmd/themes_test.go`, `cmd/quickstart_test.go`, `cmd/validate_run_test.go`: additional edge-case coverage
  - `pkg/utils/file_test.go` (+183 lines), `pkg/utils/cjk_test.go` (+66 lines), `pkg/utils/image_test.go` (+215 lines): comprehensive utility function tests
  - `internal/plugin/external_test.go` (+496 lines): expanded plugin lifecycle and error-path tests

---

## [0.5.4] - 2026-03-23

### Added

- **Lazy loading for images**: `loading="lazy"` attribute automatically added to all `<img>` tags during post-processing for better page performance
- **SEO meta tags**: Auto-generated `<meta name="description">` and Open Graph tags (`og:title`, `og:description`, `og:type`) for each site page
- **Sitemap generation**: `sitemap.xml` automatically created during site builds for search engine indexing
- **Full-text search**: Client-side search with `search-index.json` generation, search modal (Cmd/Ctrl+K or `/`), keyboard navigation, and result highlighting
- **Dark mode toggle**: Three-way theme switcher (light/dark/system) with localStorage persistence and system preference detection
- **Breadcrumb navigation**: Breadcrumb trail showing page hierarchy on each site page
- **Right-side page TOC**: "On this page" table of contents sidebar with scroll-spy highlighting via IntersectionObserver
- **Sidebar collapse on desktop**: Desktop sidebar can be collapsed/expanded with state persisted in localStorage
- **Code block copy button**: Hover over any code block to reveal a "Copy" button with clipboard integration and visual feedback
- **Search results show chapter path**: Search results now display breadcrumb path (e.g., `Part 1 > Chapter 2`) above the title for better context
- **CJK heading IDs**: Custom heading ID transformer preserves Unicode letters, generating correct CJK-aware IDs

### Changed

- **Heading tracking via IntersectionObserver**: Replaced scroll-based heading detection with IntersectionObserver for better performance and accuracy
- **Sidebar heading depth**: Set `maxSidebarHeadingDepth` to 0, showing only chapter titles in sidebar navigation (no in-page headings)
- **Sidebar toggle behavior**: Click on expanded chapter title now collapses it (toggle behavior)
- **Resize handler debounced**: Window resize events now debounced with 200ms timeout
- **External script security**: Added `crossOrigin="anonymous"` and `referrerPolicy="no-referrer"` to dynamically loaded scripts and stylesheets

### Fixed

- **ePub zip writer resource leak**: Added `defer w.Close()` to ensure zip.Writer is always closed on all error paths
- **UTF-8 truncation in SVG cover**: Replaced byte-based string slicing with rune-based truncation to avoid splitting multi-byte characters
- **SVG XML attribute escaping**: Added missing `"` and `'` entity escaping in cover SVG generation
- **Symlink path traversal**: Added `filepath.EvalSymlinks()` to the serve path security check so symlinks pointing outside OutputDir are blocked
- **Scroll behavior on navigation**: Fixed `prefersReducedMotion` ternary always returning `'auto'` instead of `'smooth'`
- **Description truncation**: Fixed byte-based truncation that could split multi-byte UTF-8 characters in meta descriptions

---

## [0.5.3] - 2026-03-23

### Fixed

- **Invisible code block text in site/serve output**: Chroma syntax highlighter injected inline `style="background-color:#fff"` on `<pre>` elements, overriding the site CSS dark-themed text color and making code blocks unreadable (white text on white background). Fix strips chroma's inline style from `<pre>` tags during post-processing so each output format's CSS controls code block appearance.
- **Site code block color scheme**: Changed site output code blocks from dark theme (`#2d2d2d` background) to light theme (`#f6f8fa` background, `#24292e` text) matching the chroma "github" syntax highlighting palette.

### Changed

- **Documentation updates**: Updated ARCHITECTURE docs version to v0.5.2, fixed ROADMAP version section ordering, added Typst format to README "What You Get" table, replaced text comparison table with emoji icons.
- **Removed outdated NEXT-STEPS.md**: Deleted stale planning document that referenced v0.4.3 as latest.

---

## [0.5.2] - 2026-03-22

### Added

- **Search focus style and a11y focus traps**: Improved keyboard accessibility with visible focus indicators and proper focus trap management
- **Windows plugin executable resolution**: Plugin paths without file extensions now auto-resolve via `PATHEXT` on Windows (`.exe`, `.bat`, `.cmd`, `.com`)

### Changed

- **`context.Context` added to Discover API**: Config discovery functions now accept a context parameter for cancellation and timeout propagation
- **`ExtractTitleFromFile` moved to utils package**: Refactored to shared `pkg/utils` for reuse across packages
- **CI matrix simplified to Go 1.25 only**: Removed Go 1.23.x and 1.24.x from the test matrix (1.25 is the minimum supported version)
- **CI supply chain hardened**: Upgraded `actions/checkout` to v5, `codecov-action` to v5, CodeQL checkout to v5; fixed Dependabot configuration syntax

### Fixed

- **Mermaid validation nil guard**: Fixed potential nil pointer dereference in navigation during mermaid validation
- **Dependabot configuration syntax**: Corrected invalid `pull-requests.max-number` to `open-pull-requests-limit`
- **Codecov action parameter name**: Fixed `file` to `files` for `codecov-action@v5` compatibility
- **Cross-platform test paths**: Replaced hardcoded Unix paths (`/home/user/book`) with `t.TempDir()` and `filepath.Join` in config tests, fixing Windows CI failures
- **ePub test resource leak**: Added missing `defer reader.Close()` after `zip.OpenReader` in epub test

---

## [0.5.1] - 2026-03-21

### Added

- **Accordion sidebar navigation**: Expanding a chapter section now automatically collapses sibling sections at the same level, providing a cleaner GitBook-like navigation experience in both `serve` and `build --format site/html` outputs
- **Smoother sidebar transitions**: Upgraded CSS transitions to Material Design easing curves (`cubic-bezier(0.4, 0, 0.2, 1)`) for expand/collapse animations

### Fixed

- **PlantUML context propagation**: Replaced `context.Background()` with caller-provided context in PlantUML renderer, ensuring build timeout and cancellation signals propagate correctly to HTTP requests and external processes
- **Generic SUMMARY.md title filtering**: Titles like "在线阅读", "Read Online", and "Contents" are now correctly recognized as generic navigation headings and skipped during book title inference
- **book.json title priority**: When `book.json` provides a title, it is no longer overwritten by README.md inference during auto-discovery
- **Missing `migrate` command in README**: Added the `migrate` command to the "All Commands" table in both English and Chinese READMEs
- **GitLab CI lint version mismatch**: Aligned golangci-lint from v2.1 to v2.11.3 to match GitHub Actions
- **Misspelling in completion command**: Fixed `behaviour` → `behavior` in comment

---

## [0.5.0] - 2026-03-20

### Added

- **PlantUML local rendering**: `renderLocal()` now invokes a local `plantuml` CLI (auto-detected via `PATH`) or `plantuml.jar` (configurable via `PLANTUML_JAR` env or `plantuml.jar_path` config), enabling offline and air-gapped environments; mode is selected by `plantuml.use_local: true` in `book.yaml`
- **Golden test framework**: `tests/golden/` infrastructure for snapshot-based regression testing of Markdown → HTML output; `go test ./tests/golden/... -update` regenerates fixtures; initial suite covers 12 Markdown feature combinations across both PDF backends
- **`mdpress doctor` PlantUML check**: Doctor command now detects local PlantUML availability (`plantuml` CLI or `PLANTUML_JAR`) and reports installation instructions when absent

### Changed

- **Default CodeTheme now "github"**: When neither `style.code_theme` in `book.yaml` nor the selected theme specifies a code theme, the fallback defaults to `"github"` instead of leaving it unset. Existing configurations with explicit `code_theme` values are unaffected. Impact: code blocks in Markdown are now highlighted with the GitHub color scheme by default.
- **HTML `lang` attribute now uses book language**: The generated HTML templates now include `lang="{{.Language}}"` with fallback to `"en"` if not set. The book language defaults to `"zh-CN"` via `book.yaml`, so CJK projects are unaffected. This affects PDF, HTML, and ePub output rendering. Impact: browsers and assistive technologies now correctly detect document language.
- **Monospace font stack now includes CJK fonts**: Code blocks and inline code now use a dedicated CJK-aware monospace font family (`ui-monospace, 'SF Mono', Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Noto Sans Mono CJK SC', monospace`) instead of the generic `'Courier New', monospace`. Code blocks also no longer have a background color by default (professional book style). Impact: CJK characters in code blocks render correctly on all platforms; code styling is now cleaner without the background.
- **CI: Node.js 24 actions upgrade**: Updated `goreleaser/goreleaser-action` to v7 (Node.js 24) ahead of the 2026-06-02 GitHub deprecation deadline for Node.js 20 runners; also pinned `codecov/codecov-action` to v5
- **CI: Release workflow Docker job fix**: Removed stale "Log in to Docker Hub" step that caused every release tag to show a red check; images are now exclusively published to GHCR via `docker/login-action` with `registry: ghcr.io`

### Tests

- **Test coverage improvement** from 62.3% to ≥ 68%:
  - `internal/plantuml`: raised from 54.8% to 75%+ with `renderLocal` path tests, mock HTTP server tests for `renderServer`, and `NeedsPlantuml` edge cases
  - `internal/source`: raised from 41.2% to 62%+ covering `LocalSource`, `GitHubSource` clone path, and `ListMarkdownFiles` edge cases
  - `cmd`: raised from 47.7% to 60%+ covering `rendererHeadingsToSiteHeadings`, `pdfChapterImageOptions`, and flag propagation
- **plantuml plugin lifecycle tests**: `Init`, `Execute`, `Cleanup`, and `EnableIfNeeded` now covered, closing zero-coverage gap in `internal/plantuml/plugin.go`

### Fixed

- **`renderLocal` error path**: Replaced silent no-op stub with actionable error message guiding users to install PlantUML or set `PLANTUML_JAR`

---

## [0.4.3] - 2026-03-19

### Fixed

- **Glossary regex performance**: Pre-compile term-matching regular expressions outside the highlight loop, avoiding repeated `regexp.MustCompile` calls per term
- **README Go version**: Corrected minimum Go version from 1.24+ to 1.25+ in both English and Chinese README
- **README Docker image refs**: Updated Docker examples from `yeasy/mdpress` to `ghcr.io/yeasy/mdpress`

### Changed

- **Release CI**: Removed Docker Hub login dependency; images now publish exclusively to GitHub Container Registry (GHCR). Upgraded to `actions/checkout@v5`, `actions/setup-go@v6`, and Go 1.25.0

### Documentation

- **COMMANDS.md**: Added missing `completion` command to English command hierarchy and matrix
- **COMMANDS_zh.md**: Added missing `--cache-dir` and `--no-cache` global flags to Chinese docs
- **COMMANDS.md + COMMANDS_zh.md**: Added `--summary` build/serve flag documentation to both languages

---

## [0.4.2] - 2026-03-19

### Changed

- **Parallel pre-commit checks**: `make check` now runs lint, build, and test concurrently via `make -j3`, reducing wall-clock time significantly
- **Faster plugin timeout test**: Rewritten `TestExternalPlugin_Execute_Timeout` script to respond instantly to `--mdpress-info`/`--mdpress-hooks` queries, eliminating ~10s of query timeouts during test setup
- **Faster image download tests**: Replaced DNS-dependent invalid host test with closed httptest server (instant connection refused), eliminating ~10s DNS resolution timeout
- **Go source formatting**: Applied `gofmt` to 8 files with minor alignment fixes

---

## [0.4.1] - 2026-03-19

### Fixed

- **Typst heading off-by-one**: `# h1` now correctly maps to `= h1` instead of `== h1` in Typst backend
- **Typst list indent**: Unordered list items with leading whitespace are now correctly extracted
- **PlantUML response limit**: Server responses capped at 10 MB to prevent excessive memory usage
- **Typst input sanitization**: `sanitizeTypstValue()` prevents code injection via margin, font, and font-size config fields
- **Parallel worker panic recovery**: Worker goroutines now catch panics and convert them to errors instead of hanging the build
- **Landing page atomic write**: Multilingual landing page uses temp-file-then-rename to prevent partial writes
- **CSS load logging**: Custom CSS load failures now log the file path; successful loads emit a Debug message
- **Plugin cleanup error handling**: `CleanupAll()` now uses `errors.Join` to preserve all plugin cleanup errors instead of only the last one
- **Example config accuracy**: Fixed incorrect field names and values in `examples/book.yaml`

### Changed

- **Typst regex performance**: Promoted 3 hot-path `regexp.MustCompile` calls in converter to package-level variables
- **CI pipeline**: Upgraded to Go 1.25, actions/checkout v5, actions/setup-go v6, golangci-lint-action v9
- **Dockerfile**: Updated builder stage from Go 1.24 to Go 1.25

### Documentation

- **Architecture docs**: Updated ARCHITECTURE.md (EN + ZH) to v0.4.0 with Typst, PlantUML, and Server module sections; added parallel build and incremental build data flow diagrams
- **COMMANDS.md**: Fixed incorrect flag descriptions and removed ghost command entry
- **Typst behavior differences**: Documented Typst vs Chromium backend differences in CHANGELOG
- **Plugin metadata**: Documented that `HookContext.Metadata` map requires no sync protection (serial access only)
- **Go version badge**: Updated README badge to Go 1.25
- **Removed internal planning docs**: Removed 5 outdated internal documents (~3,400 lines)

### Tests

- **920+ new test lines** across 11 files, closing all 15 tracked test gaps (TG-1 through TG-15):
  - CJK character detection, chapter cache invalidation, image concurrent download
  - Quickstart ReadDir, Typst generator CLI and options, root flag parsing (new file)
  - Site flattenChapters, Typst replaceLinks and unclosed code blocks
  - isFenceClose boundary cases, convertImages/convertBold, deriveLanguageOutputOverride
- Test-to-code ratio improved from 1.11:1 to 1.15:1

---

## [0.4.0] - 2026-03-19

### Added

- **Typst PDF backend**: Alternative PDF generator via `--format typst`, enabling PDF generation without Chromium dependency; proof-of-concept for multi-backend architecture
- **Parallel chapter parsing**: Multi-core Markdown parsing with automatic worker pool; `MaxConcurrency` auto-detects `runtime.NumCPU()` and caps at 8 to prevent memory issues
- **Parallel format output**: Non-PDF formats (HTML, Site, ePub) now build concurrently via `errgroup`, speeding up multi-format builds
- **Build manifest system**: SHA-256 hash tracking for chapters, configuration, and stylesheets enabling incremental builds and reproducible output; tracks chapter content and metadata in `build-manifest.json`
- **PDF watermarks**: `output.watermark` and `output.watermark_opacity` configuration options (0.0-1.0, default 0.1) for document classification (e.g., "DRAFT", "CONFIDENTIAL")
- **Custom PDF margins**: `output.margin_top`, `output.margin_bottom`, `output.margin_left`, `output.margin_right` with support for multiple units (mm, cm, in, pt, px)
- **PDF bookmarks**: `output.generate_bookmarks` flag (default true) creates clickable PDF outline from heading hierarchy via Chrome's `GenerateDocumentOutline`; enables document navigation in readers
- **PDF branded footer**: Every PDF page displays a centered "Build with mdPress" footer with clickable link to GitHub project page, replacing generic watermarks
- **Expanded test coverage**: New test suites for `config` (488 lines), `pdf` (78+134 lines), `source` (177 lines), `utils/cjk` (106 lines), `utils/image` (193 lines), and `chapter_pipeline` (229 lines); 1200+ new test lines

### Fixed

- **site.go flattenChapters data loss**: Nested chapters now correctly preserve `Depth` and `Headings` fields when flattened, fixing missing sidebar indentation and in-page TOC navigation in site output
- **diagnostics.go isFenceClose logic error**: Rewritten closing fence detection to correctly accept extended closing fences (more fence chars than opening) and trailing whitespace per CommonMark spec, fixing false-positive "mermaid-unclosed-fence" diagnostics

### Changed

- **PDF default margins**: Bottom margin increased from 0mm to 10mm to accommodate branded footer; configurable via `output.margin_*` settings
- **build_run.go orchestration**: Enhanced to support manifest-based incremental builds and parallel format output dispatch
- **ROADMAP updates**: v0.3.0 and v0.3.1 marked as released; v0.4.0 milestones updated with completed features
- **Typst backend differences**: The Typst PDF backend (`--format typst`) produces native PDF without Chromium; margin units and code block rendering may differ from the Chromium backend; CJK support in Typst requires system fonts; only `_italic_` syntax is supported (not `*italic*`)

---

## [0.3.1] - 2026-03-19

### Added

- **Docker support**: Dual-image strategy with minimal (~15 MB, no PDF) and full (~300 MB, with Chromium) images; CI/CD auto-builds to Docker Hub and GHCR (`Dockerfile`, `.github/workflows/release.yml`)
- **TOC depth control**: `toc_max_depth` configuration option (default 2) limits heading levels included in the Table of Contents, reducing TOC from 90+ pages to ~12 pages for large books like docker\_practice
- **Setext heading diagnostic**: `heading-too-long` build diagnostic detects headings over 80 characters and warns about possible Setext heading misinterpretation (paragraph followed by `---` without blank line)
- **Chapter parse caching**: Incremental build acceleration via chapter-level parse caching, skipping re-parse of unchanged chapters
- **CJK zero-config PDF**: Automatic CJK font stack selection, HTML `lang` attribute, and font detection — Chinese/Japanese/Korean books produce correct PDFs without manual font configuration
- **PDF image optimization**: Dual-pass image processing with `file://` URL support for reliable local image embedding
- **ExternalPlugin test coverage**: 19 test functions covering construction, execution, timeouts, stderr capture, JSON edge cases, and metadata queries (`internal/plugin/external_test.go`)
- **README Docker installation**: Docker quick-start commands added to both English and Chinese READMEs
- **CI Docker smoke test**: Docker build verification added to the regular CI pipeline
- **CJK font detection**: Auto-detect CJK (Chinese/Japanese/Korean) characters in book content and warn if no CJK fonts are installed before PDF generation; `mdpress doctor` now checks CJK font availability (`pkg/utils/cjk.go`, `internal/pdf/generator.go`, `cmd/doctor.go`)

### Fixed

- **epub.go resource handling**: Replaced `defer Close()` with explicit `success` flag pattern — `zip.Writer.Close()` errors are now caught, partial `.epub` files are cleaned up on failure
- **migrate.go error handling**: `GetBool` errors properly returned with `fmt.Errorf` wrapping; `filepath.Rel` failures gracefully fall back to absolute paths
- **migrate.go regex compilation**: Promoted 3 regexps to package-level variables, avoiding per-call `regexp.MustCompile`
- **release.yml BUILD_TIME**: Replaced `head_commit.timestamp` (null on tag push) with reliable `$(date -u)` command
- **Context propagation**: Replaced `context.Background()` in the build pipeline with proper context threading from callers
- **WebSocket notification locking**: Snapshot client list under lock then iterate without holding the lock, preventing slow-client stalls
- **epub.go defer double-close**: Added `fileClosed` flag to prevent redundant `f.Close()` in error-path defer cleanup
- **server.go data race**: Moved `len(s.clients)` read inside the mutex in `handleWebSocket()` to prevent race detector warnings
- **server.go JSON escaping**: Replaced manual string replacement in `notifyBuildError()` with `json.Marshal()` for complete Unicode and control character escaping
- **Dockerfile non-root user**: Added `addgroup`/`adduser` and `USER mdpress` to both minimal and full images for container security best practices
- **validate.go chapter sequence**: Refactored `parseSequenceParts()` with shared `splitSequenceParts()` helper; improved Chinese title matching with `relaxedChineseTitleSequencePattern`
- **Title consistency false positives**: Chinese+Arabic mixed numbering styles (e.g. "第一章" + "1.1") no longer trigger style-mismatch warnings; common recurring titles ("本章小结", "简介") excluded from duplicate detection; directory-scoped dedup prevents cross-chapter false positives
- **Long line overflow in all outputs**: Fixed table cells, inline code, and code blocks across PDF, HTML, site, and ePub to properly wrap long text instead of truncating; changed `table-layout` from `fixed` to `auto`; added `overflow-wrap: anywhere` and `word-break` to tables, code elements, and content areas

### Changed

- **README features table**: Added Math/KaTeX and Plugin System rows to the feature comparison table
- **Promotion materials**: Replaced 7 platform-specific posts with 4 in-depth articles
- **site.go JS deduplication**: Extracted common CDN library loader helper for ensureKaTeX/ensureMermaid

---

## [0.3.0] - 2026-03-19

### Added

- **KaTeX math support**: Inline (`$...$`) and block (`$$...$$`) LaTeX math expressions rendered via KaTeX in HTML, site, and ePub output (`internal/markdown/math.go`)
- **Mermaid diagram support**: Native Mermaid diagram rendering in all HTML-based outputs; `mdpress validate --mermaid` checks diagram syntax (`cmd/validate_mermaid.go`)
- **Plugin system**: Full plugin lifecycle with external plugin loading and hook registration (`internal/plugin/external.go`, `internal/plugin/loader.go`)
- **GitBook migration tool**: `mdpress migrate` command converts GitBook projects to mdPress format (`cmd/migrate.go`)
- **GitHub Actions template**: Pre-built workflow for automated book builds in CI (`​.github/workflows/examples/mdpress-build.yml`)
- **Multi-format `all` shorthand**: `--format all` builds PDF, HTML, site, and ePub in one command

### Changed

- ePub output promoted from experimental preview to stable; improved ePub 3 structure, metadata, and stylesheet handling
- Plugin interface extended with external process and loader support
- `mdpress serve` and `mdpress build` share unified orchestrator and pipeline logic

### Fixed

- Various stability and edge-case fixes across markdown parser, renderer, TOC, and site generator

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

## [0.1.0] - 2026-03-15

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

[Unreleased]: https://github.com/yeasy/mdpress/compare/v0.6.3...HEAD
[0.6.3]: https://github.com/yeasy/mdpress/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/yeasy/mdpress/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/yeasy/mdpress/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/yeasy/mdpress/compare/v0.5.4...v0.6.0
[0.5.4]: https://github.com/yeasy/mdpress/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/yeasy/mdpress/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/yeasy/mdpress/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/yeasy/mdpress/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/yeasy/mdpress/compare/v0.4.3...v0.5.0
[0.4.3]: https://github.com/yeasy/mdpress/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/yeasy/mdpress/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/yeasy/mdpress/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/yeasy/mdpress/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/yeasy/mdpress/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/yeasy/mdpress/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/yeasy/mdpress/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/yeasy/mdpress/releases/tag/v0.1.0
