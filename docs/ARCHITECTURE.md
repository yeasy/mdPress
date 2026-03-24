# mdPress Architecture

[中文说明](ARCHITECTURE_zh.md)

> Version: v0.6.0
> Updated: 2026-03-23

## 1. System Overview

mdPress is a CLI that turns Markdown books or long-form documentation into PDF, HTML, site, and ePub outputs. The overall architecture follows a pipeline model: content moves from source loading through parsing and post-processing into final output generation.

### 1.1 Core Pipeline

```
Source
  │
  ▼
Config Loading
  │  book.yaml / SUMMARY.md / auto-discovery
  ▼
Preprocessing
  │  variable expansion, language detection
  ▼
Markdown Parsing
  │  Goldmark AST -> HTML
  │  code highlighting, GFM extensions, footnotes
  ▼
Post-Processing
  │  image embedding / path resolution
  │  cross-reference resolution
  │  glossary highlighting
  │  GFM Alert / Mermaid conversion
  │  PlantUML local/server rendering
  ▼
Assembly
  │  cover + TOC + chapters -> full HTML
  │  theme CSS + custom CSS + print CSS
  ▼
Output
  ├─ PDF: Chromium headless -> printToPDF
  ├─ PDF (Typst): Typst CLI -> native PDF
  ├─ HTML: single-page document or multi-page site
  ├─ Site: multi-page static website
  └─ ePub: ZIP(OPF + NCX + XHTML)
```

### 1.2 Command Structure

```
mdpress (root)
  ├─ build       Build PDF/HTML/ePub/site outputs
  ├─ serve       Start the local preview server
  ├─ init        Initialize a project skeleton
  ├─ quickstart  Create a sample project and preview it
  ├─ validate    Validate project configuration
  ├─ doctor      Verify environment setup
  ├─ migrate     Migrate from GitBook/HonKit
  └─ themes      Inspect themes (list / show / preview)
```

## 2. Module Dependency Map

```mermaid
graph TD
    main[main.go] --> cmd[cmd/]

    cmd --> |build/serve| config[internal/config]
    cmd --> |build/serve| markdown[internal/markdown]
    cmd --> |build/serve| theme[internal/theme]
    cmd --> |build/serve| cover[internal/cover]
    cmd --> |build/serve| toc_mod[internal/toc]
    cmd --> |build/serve| crossref[internal/crossref]
    cmd --> |build/serve| glossary[internal/glossary]
    cmd --> |build/serve| i18n[internal/i18n]
    cmd --> |build/serve| variables[internal/variables]
    cmd --> |build/serve| renderer_mod[internal/renderer]
    cmd --> |build| pdf_mod[internal/pdf]
    cmd --> |build/serve| output[internal/output]
    cmd --> |build/serve| linkrewrite[internal/linkrewrite]
    cmd --> |build| typst[internal/typst]
    cmd --> |build/serve| plugin_mod[internal/plugin]
    cmd --> |build| source[internal/source]

    renderer_mod --> config
    renderer_mod --> theme

    variables --> config
    cover --> config

    config --> |ParseSummary| config

    cmd --> utils[pkg/utils]
    output --> utils

    subgraph External_Dependencies
        cobra[spf13/cobra]
        goldmark[yuin/goldmark]
        chromedp[chromedp/chromedp]
        yaml[gopkg.in/yaml.v3]
    end

    cmd --> cobra
    markdown --> goldmark
    pdf_mod --> chromedp
    config --> yaml
```

### 2.1 Layered Architecture

```mermaid
graph TB
    subgraph CLI_Layer
        cmd_build[cmd/build.go]
        cmd_serve[cmd/serve.go]
        cmd_init[cmd/init_cmd.go]
        cmd_themes[cmd/themes.go]
    end

    subgraph Core_Processing
        config_pkg[config - config management]
        markdown_pkg[markdown - parser engine]
        renderer_pkg[renderer - HTML assembly]
    end

    subgraph Feature_Modules
        cover_pkg[cover - cover generation]
        toc_pkg[toc - table of contents]
        crossref_pkg[crossref - cross references]
        glossary_pkg[glossary - glossary]
        i18n_pkg[i18n - multilingual support]
        variables_pkg[variables - template variables]
        theme_pkg[theme - theme system]
    end

    subgraph Output_Layer
        pdf_pkg[pdf - PDF generation]
        html_out[output/html - HTML output]
        epub_out[output/epub - ePub output]
        site_out[output/site - static site output]
    end

    subgraph Infrastructure
        utils_pkg[pkg/utils - files and images]
    end

    CLI_Layer --> Core_Processing
    CLI_Layer --> Feature_Modules
    CLI_Layer --> Output_Layer
    Core_Processing --> Feature_Modules
    Output_Layer --> Infrastructure
```

## 3. Module Responsibilities And Interfaces

### 3.1 `cmd/` - CLI Command Layer

| File | Responsibility |
| --- | --- |
| `root.go` | Cobra root command and global flags such as `--config` and `--verbose` |
| `build.go` | Build pipeline orchestration: load config -> parse -> render -> output |
| `serve.go` | Build the preview site and start the HTTP server |
| `init_cmd.go` | Scan a directory and generate a `book.yaml` skeleton |
| `quickstart.go` | Create a sample project and make it previewable immediately |
| `validate.go` | Validate `book.yaml` and related project structure |
| `themes.go` | List and show built-in themes |

Key functions:

- `executeBuild()` as the main build orchestrator
- `executeServe()` as build-plus-server orchestration
- `flattenChapters()` for recursively flattening nested chapter definitions

### 3.2 `internal/config` - Config Management

Responsibility: load `book.yaml`, parse `SUMMARY.md`, and auto-detect `GLOSSARY.md` and `LANGS.md`.

Core types:

```go
type BookConfig struct {
    Book     BookMeta
    Chapters []ChapterDef
    Style    StyleConfig
    Output   OutputConfig
}

type ChapterDef struct {
    Title    string
    File     string
    Sections []ChapterDef
}
```

Key methods:

- `Load(path) -> (*BookConfig, error)` to load config and discover companion files
- `Validate() -> error` to check config completeness
- `ResolvePath(p) -> string` to resolve relative paths against the config directory
- `ParseSummary(path) -> ([]ChapterDef, error)` to parse GitBook-style `SUMMARY.md`

### 3.3 `internal/markdown` - Markdown Parsing Engine

Responsibility: convert Markdown to HTML using Goldmark with support for GFM, footnotes, syntax highlighting, and heading ID generation.

Core types:

```go
type Parser struct { ... }
type ParserOption func(*Parser)
type HeadingInfo struct {
    Level int
    Text  string
    ID    string
}
```

Key methods:

- `NewParser(opts...) -> *Parser`
- `Parse(content) -> (html, []HeadingInfo, error)`
- `PostProcess(html) -> string` for alert and Mermaid transformations

### 3.4 `internal/renderer` - HTML Assembler

Responsibility: combine cover, TOC, chapter HTML, and CSS into one complete HTML5 document.

Core types:

```go
type HTMLRenderer struct { ... }
type RenderParts struct {
    CoverHTML    string
    TOCHTML      string
    ChaptersHTML []ChapterHTML
    CustomCSS    string
}
```

Key method:

- `Render(parts) -> (string, error)`

### 3.5 `internal/pdf` - PDF Generator

Responsibility: convert HTML into PDF through a headless Chromium session.

Core types:

```go
type Generator struct { ... }
type GeneratorOption func(*Generator)
```

Key methods:

- `Generate(html, outputPath) -> error`
- `GenerateFromFile(htmlPath, outputPath) -> error`

### 3.6 `internal/output` - Output Generation

Responsibility: produce HTML documents, static sites, and ePub outputs.

| Component | Responsibility |
| --- | --- |
| `HTMLGenerator` | Single-page HTML output |
| `SiteGenerator` | GitBook-style multi-page static site |
| `EpubGenerator` | ePub output |

### 3.7 `internal/cover` - Cover Generation

Responsibility: generate a cover page from book metadata, with either a cover image or a solid-color background.

### 3.8 `internal/toc` - Table Of Contents

Responsibility: build a hierarchical TOC tree from flattened heading lists and render it as nested HTML.

Algorithm: use a stack to build parent-child relationships from heading levels.

### 3.9 `internal/crossref` - Cross References

Responsibility: register figures, tables, and sections; replace `{{ref:id}}` placeholders; add figure and table captions automatically.

Numbering rules:

- Figures and tables increment in appearance order
- Sections use hierarchical numbering such as `1.2.3`

### 3.10 `internal/glossary` - Glossary

Responsibility: parse `GLOSSARY.md`, highlight glossary terms in HTML, attach tooltips, and render a glossary page.

### 3.11 `internal/variables` - Variable Expansion

Responsibility: expand template variables such as `{{ book.title }}` before Markdown parsing.

### 3.12 `internal/theme` - Theme System

Responsibility: manage built-in and custom themes and produce CSS.

Built-in themes:

- `technical`
- `elegant`
- `minimal`

### 3.13 `internal/i18n` - Multilingual Support

Responsibility: parse `LANGS.md` and detect multilingual projects.

### 3.14 `internal/linkrewrite` - Link Rewriting

Responsibility: rewrite Markdown `.md` links in HTML content to appropriate targets based on the output format.

Core types:

```go
type Mode string   // ModeSingle or ModeSite
type Target struct {
    ChapterID    string
    PageFilename string
}
```

Key functions:

- `RewriteLinks(html, currentFile, targets, mode) -> string` — rewrite all `.md` href attributes
- `NormalizePath(path) -> string` — normalize chapter file paths for consistent lookups

In single-page mode (`ModeSingle`), `.md` links become `#chapter-id` anchors. In site mode (`ModeSite`), they become page filenames like `ch_001.html`. Unresolved links are annotated with `data-mdpress-link="unresolved-markdown"`.

### 3.15 `pkg/utils` - Utility Helpers

Responsibility: file I/O, image downloading and base64 embedding, path resolution, and HTML escaping.

### 3.16 `internal/typst` - Typst PDF Generator

Responsibility: generate PDF files using the Typst command-line tool as an alternative to Chromium.

Typst is a markup-based PDF engine. The `Generator` type:

- Accepts Markdown or raw Typst content
- Converts Markdown to Typst syntax using `MarkdownToTypstConverter`
- Renders a Typst template with document metadata, page size, margins, and fonts
- Invokes `typst compile` to produce the final PDF

Configuration options:

```go
type Generator struct {
    timeout, pageSize, margins, fontFamily, fontSize, lineHeight, language, author, title, version, date
}
```

Advantages over Chromium: faster compilation, native PDF output, no browser dependency.

### 3.17 `internal/plantuml` - PlantUML Diagram Renderer

Responsibility: detect and render PlantUML code blocks in HTML to SVG.

The `Renderer` type:

- Searches HTML for `language-plantuml` code blocks
- Encodes PlantUML syntax using deflate + base64 custom alphabet
- Fetches SVG from the PlantUML online server or renders locally via `plantuml` CLI / `PLANTUML_JAR`
- Caches rendered SVGs to avoid repeated network calls
- Wraps each SVG in a div for styling

Key method: `RenderHTML(html) -> (string, error)` replaces all PlantUML blocks with SVG output. Local rendering detects `plantuml` on PATH or uses `PLANTUML_JAR` env var.

### 3.18 `internal/server` - Development Server

Responsibility: provide file watching and live browser reloads for `mdpress serve`.

The `Server` type:

- Listens on a specified host and port
- Watches `.md`, `.yaml`, `.yml`, and `.css` files using `fsnotify`
- Injects live-reload JavaScript into HTML responses
- Maintains a WebSocket hub for push notifications to connected clients
- Debounces file changes (500ms) to avoid repeated rebuilds
- Supports CSS-only updates (reload stylesheets) vs. full page reloads
- Falls back to polling if `fsnotify` is unavailable

Reloads are triggered via WebSocket messages; the browser-side script listens for `{"type":"reload"}` and full-page navigation.

## 4. Data Flow

### 4.1 Build Command Flow

```mermaid
sequenceDiagram
    participant User
    participant CLI as cmd/build
    participant Config as config
    participant Parser as markdown
    participant Proc as post-processing
    participant Renderer as renderer
    participant Out as output

    User->>CLI: mdpress build
    CLI->>Config: Load(book.yaml)
    Config-->>CLI: BookConfig

    loop each chapter
        CLI->>CLI: ReadFile(chapter.md)
        CLI->>CLI: variables.Expand()
        CLI->>Parser: Parse(markdown)
        Parser-->>CLI: HTML + Headings
        CLI->>Proc: ProcessImages()
        CLI->>Proc: crossref.ProcessHTML()
        CLI->>Proc: glossary.ProcessHTML()
    end

    CLI->>CLI: cover.RenderHTML()
    CLI->>CLI: toc.Generate() + RenderHTML()
    CLI->>Renderer: Render(parts)
    Renderer-->>CLI: Full HTML

    alt PDF
        CLI->>Out: pdf.Generate(html)
    end
    alt HTML
        CLI->>Out: html.Generate(html)
    end
    alt ePub
        CLI->>Out: epub.Generate(chapters)
    end
```

### 4.2 Chapter Processing Flow

```
chapter.md (raw Markdown)
  │
  ├─ variables.Expand()        -> replace {{book.title}} and similar variables
  │
  ├─ parser.Parse()            -> HTML + HeadingInfo[]
  │
  ├─ utils.ProcessImages()     -> resolve image paths / embed base64
  │
  ├─ crossref.RegisterSection()
  ├─ crossref.ProcessHTML()    -> replace {{ref:id}} with numbered links
  ├─ crossref.AddCaptions()    -> add figcaption / caption
  │
  └─ glossary.ProcessHTML()    -> term highlighting + tooltip
      │
      ▼
  ChapterHTML { Title, ID, Content }
```

### 4.3 Parallel Chapter Processing

Chapter parsing (`ChapterPipeline`) uses worker pools to process chapters concurrently:

- `computeMaxConcurrency()` determines worker count: uses `runtime.NumCPU()` (capped at 8) by default, or respects explicit `MaxConcurrency` config.
- `parseChaptersParallel()` distributes chapters to workers via job and result channels.
- Each worker runs its own `markdown.Parser` instance (goldmark state is not thread-safe).
- Results are collected in order, maintaining chapter sequence for TOC and assembly.
- First error halts all workers; panics are recovered and returned as errors.

### 4.4 Incremental Builds

Build manifest (`cmd/build_manifest.go`) enables fast incremental rebuilds via SHA-256 hashing:

- `LoadManifest()` reads cached chapter state from `build-manifest.json`.
- `ComputeChapterHash()` calculates SHA-256 of chapter file content.
- `BuildManifest.IsStale()` checks if app version, config, or CSS changed (invalidates entire cache if true).
- `GetEntry()` looks up cached HTML and headings for unchanged chapters.
- Chapters with matching hash skip parsing and reuse cached output.

Cache is stored in the project cache directory and survives across builds unless `mdpress_cache_dir` is disabled.

## 5. Implemented And Planned Architecture Extensions

> Sections 5.1 through 5.4 describe architecture that has been **implemented in v0.2.0**. Section 5.5 describes extension points reserved for v0.3.0.

### 5.1 Source Abstraction (Implemented)

The `Source` interface abstracts "where content comes from" so mdPress can read from local filesystems, GitHub repositories, and future providers.

Interface:

```go
// Source defines a unified abstraction for content sources
type Source interface {
    // Open opens and prepares the content source (e.g., clone repo, download files)
    Open(ctx context.Context) error

    // ReadFile reads the file content at the specified path
    ReadFile(path string) ([]byte, error)

    // ListFiles lists all files matching the pattern
    ListFiles(pattern string) ([]string, error)

    // Resolve resolves a relative path to an absolute path within the source
    Resolve(base, rel string) string

    // Close closes the source and cleans up temporary resources
    Close() error

    // Type returns the source type identifier
    Type() string
}
```

Class diagram:

```mermaid
classDiagram
    class Source {
        <<interface>>
        +Open(ctx) error
        +ReadFile(path) ([]byte, error)
        +ListFiles(pattern) ([]string, error)
        +Resolve(base, rel) string
        +Close() error
        +Type() string
    }

    class LocalSource {
        -rootDir string
        +Open(ctx) error
        +ReadFile(path) ([]byte, error)
    }

    class GitHubSource {
        -owner string
        -repo string
        -ref string
        -localDir string
        +Open(ctx) error
        +ReadFile(path) ([]byte, error)
    }

    class GitLabSource {
        -projectID string
        -ref string
        +Open(ctx) error
    }

    class URLSource {
        -baseURL string
        -cacheDir string
        +Open(ctx) error
    }

    Source <|.. LocalSource
    Source <|.. GitHubSource
    Source <|.. GitLabSource : future extension
    Source <|.. URLSource : future extension
```

Current implementations:

- `LocalSource` (implemented)
- `GitHubSource` (implemented, supports `GITHUB_TOKEN` for private repos)
- `GitLabSource` (future extension)
- `URLSource` (future extension)

Integration:

- `source.Detect()` selects a `Source` implementation based on the input URL or path
- `cmd/build.go` and `cmd/serve.go` use `Source` for project retrieval
- `LocalSource` wraps filesystem behavior so existing flows remain compatible

### 5.2 Config Discovery Chain (Implemented)

Priority:

```
1. Chapters explicitly defined in book.yaml      ← Highest priority
   │ (if chapters is empty)
   ▼
2. SUMMARY.md in the same directory              ← GitBook compatible
   │ (if SUMMARY.md does not exist)
   ▼
3. Automatic scanning of *.md files              ← Zero-config experience
   by directory structure + filename order
   excludes: README.md, SUMMARY.md, GLOSSARY.md, LANGS.md
```

Design:

```go
// ConfigDiscovery represents the config discovery chain
type ConfigDiscovery struct {
    discoverers []Discoverer
}

// Discoverer represents a single discovery strategy
type Discoverer interface {
    // Name returns the strategy name
    Name() string
    // Discover attempts to discover chapter configuration
    // Returns nil to indicate not found, passing control to next strategy
    Discover(baseDir string) ([]ChapterDef, error)
}
```

Implemented strategies:

- `YAMLDiscoverer` — loads `book.yaml`
- `SummaryDiscoverer` — parses `SUMMARY.md`
- `AutoDiscoverer` — scans Markdown files by directory structure

### 5.3 Output Format Abstraction (Implemented)

All output formats are normalized behind the `FormatBuilder` interface (in `cmd/format_builders.go`) with a `FormatBuilderRegistry` for registration and dispatch. The build pipeline uses the registry instead of switch-case logic.

Interface:

```go
type OutputFormat interface {
    Name() string
    Generate(ctx context.Context, req *RenderRequest, outputPath string) error
    Validate(cfg *config.OutputConfig) error
}

type RenderRequest struct {
    FullHTML string
    Chapters []ChapterContent
    CSS      string
    Meta     DocumentMeta
}
```

Mappings:

| Implementation | Existing Code |
| --- | --- |
| `PDFOutput` | `internal/pdf.Generator` |
| `HTMLOutput` | `internal/output.HTMLGenerator` |
| `SiteOutput` | `internal/output.SiteGenerator` |
| `EpubOutput` | `internal/output.EpubGenerator` |

The `FormatBuilderRegistry` is created at startup with all built-in builders pre-registered:

```go
type FormatBuilderRegistry struct {
    builders map[string]FormatBuilder
}
```

### 5.4 Server Module (Implemented)

The development server provides file watching and browser reload for `serve`.

Architecture diagram:

```mermaid
graph LR
    subgraph serve command
        Watcher[fsnotify file watching]
        Builder[incremental builder]
        Server[HTTP Server]
        WS[WebSocket Hub]
    end

    FS[file system *.md] -->|file changes| Watcher
    Watcher -->|debounce 500ms| Builder
    Builder -->|rebuild| Server
    Builder -->|notify| WS
    WS -->|reload message| Browser[browser]
    Browser -->|HTTP request| Server
    Browser -->|WS connection| WS
```

Key components:

```go
// DevServer is the development server
type DevServer struct {
    config    *config.BookConfig
    outputDir string
    port      int
    watcher   *fsnotify.Watcher
    hub       *WSHub
    builder   *IncrementalBuilder
}

// WSHub manages WebSocket connections
type WSHub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan []byte
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
}

// IncrementalBuilder handles incremental builds
type IncrementalBuilder struct {
    source    Source
    cache     map[string]*BuildCache  // file path → build cache
    debounce  time.Duration
}
```

Expansion from current behavior:

1. Reuse the existing site build for the initial render
2. Watch `.md`, `.yaml`, and `.css` files with `fsnotify`
3. Inject WebSocket client code into generated HTML
4. Trigger rebuilds and browser refresh messages when files change

### 5.5 Plugin Extension Points

Goal: reserve plugin lifecycle hooks in v0.2 even if the full plugin system ships later.

Lifecycle hooks:

```mermaid
graph TD
    ConfigLoaded[ConfigLoaded - after config loading]
    BeforeParse[BeforeParse - before Markdown parsing]
    AfterParse[AfterParse - after Markdown parsing]
    BeforeRender[BeforeRender - before HTML assembly]
    AfterRender[AfterRender - after HTML assembly]
    BeforeOutput[BeforeOutput - before output]
    AfterOutput[AfterOutput - after output]

    ConfigLoaded --> BeforeParse
    BeforeParse --> AfterParse
    AfterParse --> BeforeRender
    BeforeRender --> AfterRender
    AfterRender --> BeforeOutput
    BeforeOutput --> AfterOutput
```

Capability matrix:

| Hook | Typical Use | Example Plugins |
| --- | --- | --- |
| `ConfigLoaded` | Inject defaults or environment-derived config | Environment variable injection |
| `BeforeParse` | Preprocess Markdown, include directives, custom syntax | Custom syntax plugins, include directives |
| `AfterParse` | Transform generated HTML | Automatic link checking |
| `BeforeRender` | Modify `RenderParts` | Custom cover plugins |
| `AfterRender` | Inject SEO tags or watermarks | SEO plugins, watermark plugins |
| `BeforeOutput` | Intercept output destination or output metadata | Output path customization |
| `AfterOutput` | Upload artifacts or send notifications | CDN upload, notification plugins |

## 6. Refactoring Notes

### 6.1 Refactors Already Completed

#### 6.1.1 New Interface Files

- `internal/source/source.go` for `Source` and `LocalSource`
- `internal/output/output.go` for `OutputFormat`, `Registry`, and `RenderRequest`
- `internal/plugin/plugin.go` for plugin interfaces and hook placeholders

#### 6.1.2 Interface Design Principles

1. Backward compatibility: new interfaces wrap existing behavior
2. Incremental migration: `cmd/build.go` can move gradually
3. Minimal interfaces: each interface exposes only required methods
4. Context propagation: long-running work should accept `context.Context`

### 6.2 Refactors Completed In v0.2.0

The following refactors from the original plan have been completed:

#### 6.2.1 Build Pipeline Split (Completed)

`BuildOrchestrator` (`cmd/build_orchestrator.go`) and `ChapterPipeline` (`cmd/chapter_pipeline.go`) now encapsulate the shared build workflow. Both `build` and `serve` delegate to these types:

```go
type BuildOrchestrator struct {
    Config *config.BookConfig
    Theme  *theme.Theme
    Parser *markdown.Parser
    Gloss  *glossary.Glossary
    Logger *slog.Logger
}

func (o *BuildOrchestrator) ProcessChapters() (*ChapterPipelineResult, error)
func (o *BuildOrchestrator) LoadCustomCSS() string
```

#### 6.2.2 Duplicate Logic Removal (Completed)

`ChapterPipeline` eliminated approximately 135 lines of duplicate chapter processing code between `build` and `serve`.

#### 6.2.3 Hard-Coded Values Extraction (Completed)

| Original Location | Change |
| --- | --- |
| PDF timeout | Moved to `OutputConfig.PDFTimeout` (default 120s) |
| Chrome path | Supports `MDPRESS_CHROME_PATH` environment variable |
| Mermaid CDN URL | Centralized in `pkg/utils/constants.go` as `MermaidCDNURL` |

#### 6.2.4 Error Handling (Completed)

- `renderer.NewHTMLRenderer()` and `NewStandaloneHTMLRenderer()` now return `(*Type, error)` instead of calling `panic`
- `pkg/utils/escape.go` provides centralized `EscapeHTML()`, `EscapeXML()`, and `EscapeAttr()`

#### 6.2.5 Testability (Completed)

- `ServeOptions` struct replaces global variables for serve configuration
- `internal/pdf/mock.go` provides `MockGenerator` for testing without Chromium
- `server.go` uses an independent `http.ServeMux`

### 6.3 Remaining Refactoring Opportunities

- CI: add Windows to the test matrix
- `source/github.go`: add `GitLabSource` for broader Git hosting support
- Consider extracting `IncrementalBuilder` for file-hash-based rebuild caching (v0.4.0)
