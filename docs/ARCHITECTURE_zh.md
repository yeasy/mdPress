# mdPress 架构设计文档

[English](ARCHITECTURE.md)

> 版本: v0.7.5
> 更新日期: 2026-04-06

## 1. 系统架构总览

mdPress 是一个将 Markdown 格式图书转换为 PDF/HTML/ePub 的命令行工具。整体架构遵循**管道（Pipeline）模式**，数据从输入源经过多个处理阶段，最终输出为目标格式。

### 1.1 核心 Pipeline

```
输入源 (Source)
  │
  ▼
配置加载 (Config)
  │  book.yaml / SUMMARY.md / 自动发现
  ▼
预处理 (Preprocessing)
  │  变量展开、多语言检测
  ▼
Markdown 解析 (Parser)
  │  Goldmark AST → HTML
  │  代码高亮、GFM 扩展、脚注
  ▼
后处理 (PostProcessing)
  │  图片处理（base64 嵌入/路径解析）
  │  交叉引用解析（{{ref:id}} → 编号）
  │  术语表高亮（tooltip 注释）
  │  GFM Alert / Mermaid / PlantUML 转换
  ▼
组装 (Assembly)
  │  封面 + 目录 + 章节 → 完整 HTML
  │  主题 CSS + 自定义 CSS + 打印 CSS
  ▼
输出 (Output)
  ├─ PDF (Chromium):  Chromium headless → printToPDF
  ├─ PDF (Typst): Typst 命令行 → 原生 PDF
  ├─ HTML: 单页 HTML 文档
  ├─ Site: 多页静态站点
  └─ ePub 3: ZIP(OPF + NCX + XHTML + Navigation Document)
```

### 1.2 命令结构

```
mdpress (root)
  ├─ build       构建 PDF/HTML/ePub/Site 输出
  ├─ serve       启动本地预览服务器
  ├─ init        初始化项目骨架
  ├─ quickstart  创建示例项目并立即构建
  ├─ validate    验证项目配置
  ├─ doctor      验证环境设置
  ├─ migrate     从 GitBook/HonKit 迁移
  ├─ upgrade     自升级到最新版本
  ├─ completion  生成 shell 补全脚本
  ├─ version     打印版本信息
  └─ themes      查看主题（list / show / preview）
```

## 2. 模块依赖关系图

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

    subgraph 外部依赖
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

### 2.1 分层架构

```mermaid
graph TB
    subgraph CLI层
        cmd_build[cmd/build.go]
        cmd_serve[cmd/serve.go]
        cmd_init[cmd/init_cmd.go]
        cmd_themes[cmd/themes.go]
    end

    subgraph 核心处理层
        config_pkg[config - 配置管理]
        markdown_pkg[markdown - 解析引擎]
        renderer_pkg[renderer - HTML组装]
    end

    subgraph 功能模块层
        cover_pkg[cover - 封面生成]
        toc_pkg[toc - 目录生成]
        crossref_pkg[crossref - 交叉引用]
        glossary_pkg[glossary - 术语表]
        i18n_pkg[i18n - 多语言]
        variables_pkg[variables - 变量替换]
        theme_pkg[theme - 主题系统]
    end

    subgraph 输出层
        pdf_pkg[pdf - PDF生成]
        html_out[output/html - HTML输出]
        epub_out[output/epub - ePub输出]
        site_out[output/site - 静态站点]
    end

    subgraph 基础设施层
        utils_pkg[pkg/utils - 文件/图片工具]
    end

    CLI层 --> 核心处理层
    CLI层 --> 功能模块层
    CLI层 --> 输出层
    核心处理层 --> 功能模块层
    输出层 --> 基础设施层
```

## 3. 各模块职责和接口定义

### 3.1 cmd/ — CLI 命令层

| 文件 | 职责 |
|------|------|
| `root.go` | Cobra 根命令，全局 flag（`--config`, `--verbose`） |
| `build.go` | 构建命令：源解析、配置加载、格式分发 |
| `build_run.go` | 核心构建执行：渲染流水线、输出生成 |
| `build_orchestrator.go` | `buildOrchestrator` 并发章节处理 |
| `build_manifest.go` | 构建清单，用于增量/缓存构建 |
| `chapter_pipeline.go` | 并行章节解析与后处理流水线 |
| `chapter_cache.go` | 章节级构建缓存 |
| `format_builders.go` | `formatBuilderRegistry` 输出格式分发 |
| `serve.go` | 构建 HTML 站点并启动 HTTP 服务器 |
| `init_cmd.go` | 扫描目录，生成 book.yaml 骨架 |
| `quickstart.go` | 创建示例项目并立即构建预览 |
| `validate.go` | 验证 book.yaml 配置正确性 |
| `validate_mermaid.go` | 通过 Chromium 进行 Mermaid 图表服务端验证 |
| `themes.go` | 列出/展示内置主题 |
| `themes_preview.go` | 生成主题可视化预览图 |
| `doctor.go` | 验证环境配置（Chromium、字体等） |
| `migrate.go` | 从 GitBook/HonKit 配置迁移 |
| `upgrade.go` | 从 GitHub 自动升级到最新版本 |
| `navigation.go` | 站点输出的导航辅助（上一页/下一页） |
| `issues.go` | 项目问题收集与报告 |
| `completion.go` | Shell 补全脚本生成 |
| `version.go` | 打印版本和构建信息 |

**关键函数：**
- `executeBuild()` — 分发到 `executeBuildForConfig()` 处理每种语言
- `executeBuildForConfig()` — 运行完整的渲染和输出流水线
- `buildOrchestrator.ProcessChapters()` — 并发章节解析
- `executeServe()` — 构建站点 + 启动 HTTP server

### 3.2 internal/config — 配置管理

**职责：** 加载 book.yaml，解析 SUMMARY.md，自动检测 GLOSSARY.md / LANGS.md。

**核心类型：**
```go
type BookConfig struct {     // 顶层配置
    Book     BookMeta         // 书籍元数据
    Chapters []ChapterDef     // 章节定义（支持嵌套）
    Style    StyleConfig      // 样式配置
    Output   OutputConfig     // 输出配置
    Plugins  []PluginConfig   // 插件配置
}

type ChapterDef struct {     // 章节定义
    Title    string
    File     string
    Sections []ChapterDef     // 嵌套子章节
}
```

**关键方法：**
- `Load(path) → (*BookConfig, error)` — 加载配置，自动发现辅助文件
- `Validate() → error` — 校验配置完整性
- `ResolvePath(p) → string` — 基于配置目录解析相对路径
- `ParseSummary(path) → ([]ChapterDef, error)` — 解析 SUMMARY.md

### 3.3 internal/markdown — Markdown 解析引擎

**职责：** 基于 Goldmark 的 Markdown → HTML 转换，支持 GFM、脚注、代码高亮、heading ID 生成。

**核心类型：**
```go
type Parser struct { ... }           // Markdown 解析器
type ParserOption func(*Parser)      // 函数式选项
type HeadingInfo struct {            // 标题信息
    Level  int
    Text   string
    ID     string
    Line   int
    Column int
}
```

**关键方法：**
- `NewParser(opts...) → *Parser` — 创建解析器
- `Parse(content) → (html, []HeadingInfo, error)` — 解析 Markdown
- `postProcess(html) → string`（包级未导出函数）— GFM Alert / Mermaid 后处理

### 3.4 internal/renderer — HTML 组装器

**职责：** 将封面、目录、章节等部件组装成完整的 HTML5 文档。

**核心类型：**
```go
type HTMLRenderer struct { ... }
type RenderParts struct {
    CoverHTML    string
    TOCHTML      string
    ChaptersHTML []ChapterHTML
    CustomCSS    string
}
```

**关键方法：**
- `Render(parts) → (string, error)` — 组装完整 HTML

### 3.5 internal/pdf — PDF 生成器

**职责：** 通过 Chromium headless 浏览器将 HTML 转换为 PDF。

**核心类型：**
```go
type Generator struct { ... }
type GeneratorOption func(*Generator)
```

**关键方法：**
- `Generate(html, outputPath) → error` — HTML 字符串 → PDF 文件
- `GenerateFromFile(htmlPath, outputPath) → error` — HTML 文件 → PDF 文件

### 3.6 internal/output — 输出格式生成

**职责：** 生成 HTML 静态站点和 ePub 电子书。

| 组件 | 职责 |
|------|------|
| `HTMLGenerator` | 单页 HTML 输出 |
| `SiteGenerator` | Gitbook 风格多页静态站点 |
| `EpubGenerator` | ePub 3 电子书 |

### 3.7 internal/cover — 封面生成

**职责：** 根据书籍元数据生成 HTML 封面页，支持封面图片或纯色背景。

### 3.8 internal/toc — 目录生成

**职责：** 从扁平的 heading 列表构建层级目录树，渲染为嵌套的 HTML 列表。

**算法：** 使用栈（stack）结构按 heading level 构建父子关系树。

### 3.9 internal/crossref — 交叉引用

**职责：** 注册图表/表格/章节引用，替换 `{{ref:id}}` 占位符，自动添加 figcaption / caption。

**编号规则：** 图表和表格按出现顺序递增编号；章节使用层级编号（如 1.2.3）。

### 3.10 internal/glossary — 术语表

**职责：** 解析 GLOSSARY.md，在 HTML 中高亮术语并添加 tooltip，渲染术语表页面。

### 3.11 internal/variables — 变量替换

**职责：** 在 Markdown 解析前展开 `{{ book.title }}` 等模板变量。

### 3.12 internal/theme — 主题系统

**职责：** 管理内置和自定义主题，提供 CSS 生成。

**内置主题：** technical（技术文档）、elegant（文艺风格）、minimal（极简设计）

### 3.13 internal/i18n — 多语言支持

**职责：** 解析 LANGS.md，检测多语言项目。

### 3.14 internal/linkrewrite — 链接重写

**职责：** 根据输出格式将 HTML 中的 Markdown `.md` 链接重写为对应目标。

**核心类型：**
```go
type Mode string   // ModeSingle 或 ModeSite
type Target struct {
    ChapterID    string    // 章节锚点 ID
    PageFilename string    // 站点模式下的页面文件名
}
```

**关键函数：**
- `RewriteLinks(html, currentFile, targets, mode) → string` — 重写所有 `.md` href 属性
- `NormalizePath(path) → string` — 规范化章节文件路径以确保查找一致性

在单页模式（`ModeSingle`）下，`.md` 链接变为 `#chapter-id` 锚点。在站点模式（`ModeSite`）下，变为 `ch_001.html` 等页面文件名。未解析的链接会被标注 `data-mdpress-link="unresolved-markdown"` 属性。

### 3.15 pkg/utils — 工具函数

**职责：** 文件 I/O、图片处理（下载/base64 嵌入/路径解析）、HTML 转义。

### 3.16 internal/typst — Typst PDF 生成

**职责：** 通过 Typst 命令行工具将 Markdown 或中间格式转换为原生 PDF。

Typst 是基于标记语言的 PDF 引擎。`Generator` 类型：

- 接受 Markdown 或原始 Typst 内容
- 使用 `MarkdownToTypstConverter` 将 Markdown 转换为 Typst 语法
- 使用文档元数据、页面尺寸、边距和字体渲染 Typst 模板
- 调用 `typst compile` 生成最终 PDF

配置选项：

```go
type Generator struct {
    timeout, pageSize, margins, fontFamily, fontSize, lineHeight, language, author, title, version, date
}
```

相比 Chromium 的优势：编译更快、原生 PDF 输出、无浏览器依赖。

### 3.17 internal/plantuml — PlantUML 处理

**职责：** 处理 PlantUML 图表，支持本地 `plantuml` CLI 命令或通过 `PLANTUML_JAR` 环境变量指定本地 JAR 文件，以及远程 PlantUML 服务生成图像。

`Renderer` 类型：

- 在 HTML 中搜索 `language-plantuml` 代码块
- 使用 deflate + 自定义 base64 字母表编码 PlantUML 语法
- 从 PlantUML 在线服务器获取 SVG 或通过本地 `plantuml` CLI / `PLANTUML_JAR` 渲染
- 缓存已渲染的 SVG 以避免重复网络请求
- 将每个 SVG 包装在 div 中以便样式设置

关键方法：`RenderHTML(ctx, html) -> (string, error)` 将所有 PlantUML 代码块替换为 SVG 输出。本地渲染会检测 PATH 中的 `plantuml` 或使用 `PLANTUML_JAR` 环境变量。

### 3.18 internal/server — 开发服务器

**职责：** 为 `serve` 命令提供完整的开发服务器架构，支持文件监听和浏览器自动刷新。

**核心组件：**

```go
type Server struct {
    Host      string
    Port      int
    WatchDir  string
    OutputDir string
    AutoOpen  bool
    BuildFunc func() error
    clients   map[*wsClient]struct{}
    clientsMu sync.RWMutex
}

type wsClient struct {
    conn    *websocket.Conn
    writeMu sync.Mutex
}
```

**关键功能：**
- 初始构建（复用现有 site 构建流程）
- 使用 fsnotify 监听 `.md`、`.yaml`、`.yml` 和 `.css` 文件变更
- 注入 WebSocket 客户端脚本到生成的 HTML 页面
- 维护 WebSocket 连接池，向已连接的客户端推送通知
- 文件变更防抖（500ms），避免重复构建
- 支持仅 CSS 更新（重新加载样式表）与全页面刷新
- 当 fsnotify 不可用时回退到轮询模式

重新加载通过 WebSocket 消息触发；浏览器端脚本监听 `{"type":"reload"}` 并执行全页面导航。

## 4. 数据流说明

### 4.1 Build 命令数据流

```mermaid
sequenceDiagram
    participant User
    participant CLI as cmd/build
    participant Config as config
    participant Parser as markdown
    participant Proc as 后处理模块
    participant Renderer as renderer
    participant Out as 输出模块

    User->>CLI: mdpress build
    CLI->>Config: Load(book.yaml)
    Config-->>CLI: BookConfig

    loop 每个章节
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
    Renderer-->>CLI: 完整 HTML

    alt PDF 输出
        CLI->>Out: pdf.Generate(html)
    end
    alt HTML 输出
        CLI->>Out: html.Generate(html)
    end
    alt ePub 输出
        CLI->>Out: epub.Generate(chapters)
    end
```

### 4.2 章节处理数据流

```
chapter.md (原始 Markdown)
  │
  ├─ variables.Expand()        → 替换 {{book.title}} 等变量
  │
  ├─ parser.Parse()            → HTML + HeadingInfo[]
  │
  ├─ utils.ProcessImages()     → 图片路径解析/base64嵌入
  │
  ├─ crossref.RegisterSection()→ 注册标题到引用表
  ├─ crossref.ProcessHTML()    → 替换 {{ref:id}} 为编号链接
  ├─ crossref.AddCaptions()    → 添加 figcaption/caption
  │
  └─ glossary.ProcessHTML()    → 术语高亮 + tooltip
      │
      ▼
  ChapterHTML { Title, ID, Content }
```

### 4.3 并行章节处理

章节解析（`chapterPipeline`）使用 worker pool 并发处理多个章节：

- `computeMaxConcurrency()` 确定 worker 数量：默认使用 `runtime.NumCPU()`（上限为 8），或遵循配置中的明确 `MaxConcurrency` 设置。
- `parseChaptersParallel()` 通过 job 和 result 通道将章节分配给 worker 处理。
- 每个 worker 运行自己的 `markdown.Parser` 实例（Goldmark 的状态非线程安全）。
- 结果按顺序收集，确保章节序列保持一致，用于目录和组装。
- 第一个错误会停止所有 worker；panic 会被捕获并转换为错误返回。

### 4.4 增量构建

构建清单（`cmd/build_manifest.go`）通过 SHA-256 哈希使快速增量重构成为可能：

- `loadManifest()` 从 `build-manifest.json` 读取缓存的章节状态。
- `computeChapterHash()` 计算章节文件内容的 SHA-256 哈希值。
- `buildManifest.IsStale()` 检查应用版本、配置或 CSS 是否改变（如果改变则整个缓存失效）。
- `buildManifest.GetEntry()` 查找未修改章节的缓存 HTML 和标题。
- 哈希值匹配的章节跳过解析并复用缓存输出。

缓存存储在项目缓存目录中，除非禁用 `MDPRESS_CACHE_DIR`，否则在构建之间保留。

## 5. 已实现与计划中的架构扩展

> 5.1 至 5.4 描述的架构已**实现**。5.5 描述已可用的插件扩展点。

### 5.1 Source 抽象层（已实现）

**目标：** 将"内容从哪里来"抽象为 Source 接口，使 mdPress 能从本地文件系统、GitHub 仓库等多种来源读取内容。

**接口定义：** 见 `internal/source/source.go`

```go
// Source 定义内容来源的统一抽象。
// Prepare 返回包含内容的本地目录路径。
type Source interface {
    Prepare() (string, error)
    Cleanup() error
    Type() string
}
```

**类图：**

```mermaid
classDiagram
    class Source {
        <<interface>>
        +Prepare() (string, error)
        +Cleanup() error
        +Type() string
    }

    class LocalSource {
        -path string
        -opts Options
        +Prepare() (string, error)
        +Cleanup() error
        +Type() string
    }

    class GitHubSource {
        -owner string
        -repo string
        -tempDir string
        -opts Options
        +Prepare() (string, error)
        +Cleanup() error
        +Type() string
    }

    Source <|.. LocalSource
    Source <|.. GitHubSource
```

**当前实现：**

- `LocalSource`（已实现）
- `GitHubSource`（已实现，支持 `GITHUB_TOKEN` 访问私有仓库）
- `GitLabSource`（未来扩展）
- `URLSource`（未来扩展）

**与现有代码的集成：**
- `source.Detect()` 根据输入 URL 或路径自动选择 Source 实现
- `cmd/build.go` 和 `cmd/serve.go` 使用 Source 进行项目获取
- LocalSource 封装现有的文件系统操作，保持向后兼容

### 5.2 Config 发现链（已实现）

**目标：** 实现 `book.yaml → book.json → SUMMARY.md → 自动发现` 的优先级配置发现链。

**发现优先级：**

```
1. book.yaml 中显式定义 chapters      ← 最高优先级
   │ (如果 chapters 为空)
   ▼
2. 同目录下的 book.json                ← GitBook JSON 格式兼容
   │ (如果 book.json 不存在)
   ▼
3. 同目录下的 SUMMARY.md              ← GitBook 兼容
   │ (如果 SUMMARY.md 不存在)
   ▼
4. 自动扫描 *.md 文件                  ← 零配置体验
   按目录结构 + 文件名排序
   如果存在 README.md 则作为第一章
   排除: SUMMARY.md, GLOSSARY.md, LANGS.md
```

**设计方案：**

`internal/config/discover.go` 中的 `Discover()` 函数实现了基于优先级的发现链：

```go
func Discover(ctx context.Context, dir string) (*BookConfig, error)
```

**发现优先级：**
1. `book.yaml` / `book.json` — 加载显式配置
2. `SUMMARY.md` — 解析 GitBook 兼容的目录文件
3. 自动发现 — 扫描目录下所有 .md 文件，按路径排序

### 5.3 输出格式抽象（已实现）

所有输出格式通过 `formatBuilder` 接口（位于 `cmd/format_builders.go`）和 `formatBuilderRegistry` 进行注册和分发。构建流水线使用注册表替代 switch-case 逻辑。

**接口定义：** 见 `cmd/format_builders.go`

```go
// formatBuilder 定义输出格式的统一接口
type formatBuilder interface {
    // Name 返回格式名称（如 "pdf", "html", "site", "epub"）
    Name() string
    // Build 在给定基础路径生成输出文件
    Build(ctx *buildContext, baseName string) error
}
```

**实现映射：**

| 接口实现 | 对应现有代码 |
|---------|------------|
| `pdfBuilder` | `internal/pdf.Generator` |
| `htmlBuilder` | `internal/output.HTMLGenerator` |
| `siteBuilder` | `internal/output.SiteGenerator` |
| `epubBuilder` | `internal/output.EpubGenerator` |
| `typstBuilder` | `internal/typst.Generator` |

`formatBuilderRegistry` 在启动时创建，所有内置构建器预先注册：

```go
type formatBuilderRegistry struct {
    builders map[string]formatBuilder
}
```

### 5.4 Server 模块（已实现）

**目标：** 为 `serve` 命令设计完整的开发服务器架构，支持文件监听和浏览器自动刷新。

**架构设计：**

```mermaid
graph LR
    subgraph serve 命令
        Watcher[fsnotify 文件监听]
        Builder[增量构建器]
        Server[HTTP Server]
        WS[WebSocket Hub]
    end

    FS[文件系统 *.md] -->|变更事件| Watcher
    Watcher -->|防抖 500ms| Builder
    Builder -->|重新构建| Server
    Builder -->|通知| WS
    WS -->|reload 消息| Browser[浏览器]
    Browser -->|HTTP 请求| Server
    Browser -->|WS 连接| WS
```

**核心组件：**

```go
// Server 开发服务器，支持实时重载
type Server struct {
    Host      string
    Port      int
    WatchDir  string
    OutputDir string
    AutoOpen  bool
    BuildFunc func() error

    clients   map[*wsClient]struct{}
    clientsMu sync.RWMutex
    logger    *slog.Logger
    upgrader  websocket.Upgrader

    // 文件变更重建的防抖状态
    debounceTimer *time.Timer
    debounceMu    sync.Mutex
}

// wsClient 封装单个 WebSocket 连接及其写入锁
type wsClient struct {
    conn    *websocket.Conn
    writeMu sync.Mutex
}
```

**实现功能：**
1. 初始构建（复用现有 site 构建流程）
2. 使用 fsnotify 监听 .md / .yaml / .css 文件变更
3. 通过中间件注入 WebSocket 客户端脚本到生成的 HTML 页面
4. 防抖处理文件变更事件并触发重建
5. 向所有已连接的 WebSocket 客户端广播刷新消息

### 5.5 插件系统预留

**目标：** 提供 Plugin 接口和生命周期 Hook 点，使外部插件可接入构建流水线。

**接口定义：** 见 `internal/plugin/plugin.go`

**生命周期 Hook 点：**

```mermaid
graph TD
    ConfigLoaded[ConfigLoaded - 配置加载后]
    BeforeParse[BeforeParse - Markdown解析前]
    AfterParse[AfterParse - Markdown解析后]
    BeforeRender[BeforeRender - HTML组装前]
    AfterRender[AfterRender - HTML组装后]
    BeforeOutput[BeforeOutput - 输出前]
    AfterOutput[AfterOutput - 输出后]

    ConfigLoaded --> BeforeParse
    BeforeParse --> AfterParse
    AfterParse --> BeforeRender
    BeforeRender --> AfterRender
    AfterRender --> BeforeOutput
    BeforeOutput --> AfterOutput
```

**插件能力矩阵：**

| Hook 点 | 可做什么 | 示例插件 |
|---------|---------|---------|
| ConfigLoaded | 修改配置、注入默认值 | 环境变量注入插件 |
| BeforeParse | 预处理 Markdown 源码 | 自定义语法插件、Include 插件 |
| AfterParse | 修改解析后的 HTML | 自动链接检查插件 |
| BeforeRender | 修改 RenderParts | 自定义封面插件 |
| AfterRender | 修改最终 HTML | SEO 插件、水印插件 |
| BeforeOutput | 拦截/修改输出流程 | 输出路径自定义插件 |
| AfterOutput | 后处理动作 | 上传到 CDN 插件、通知插件 |

## 6. 重构建议与改进

### 6.1 已完成的重构

#### 6.1.1 新增接口定义文件

- **`internal/source/source.go`** — Source 接口 + LocalSource 实现
- **`internal/output/output.go`** — OutputFormat 接口 + Registry + RenderRequest
- **`internal/plugin/plugin.go`** — Plugin 接口 + HookContext + Manager（预留）

#### 6.1.2 接口设计原则

1. **向后兼容**：新接口封装现有实现，不破坏现有 API
2. **渐进式迁移**：cmd/build.go 可逐步切换到新接口，无需一次性重构
3. **最小接口原则**：每个接口只定义必要的方法
4. **Context 传递**：所有可能耗时的操作都接受 `context.Context`

### 6.2 v0.2.0 中已完成的重构

以下重构已从原计划中完成：

#### 6.2.1 构建 Pipeline 拆分（已完成）

`buildOrchestrator`（`cmd/build_orchestrator.go`）和 `chapterPipeline`（`cmd/chapter_pipeline.go`）现已封装共享的构建工作流。`build` 和 `serve` 两个命令都委托给这些类型：

```go
type buildOrchestrator struct {
    Config        *config.BookConfig
    Theme         *theme.Theme
    Parser        *markdown.Parser
    Gloss         *glossary.Glossary
    Logger        *slog.Logger
    PluginManager *plugin.Manager
}

func (o *buildOrchestrator) ProcessChapters(ctxOpts ...context.Context) (*chapterPipelineResult, error)
func (o *buildOrchestrator) LoadCustomCSS() string
```

#### 6.2.2 消除代码重复（已完成）

`chapterPipeline` 消除了 `build` 和 `serve` 之间约 135 行重复的章节处理代码。

#### 6.2.3 硬编码值提取（已完成）

| 原始位置 | 硬编码值 | 改进措施 |
|---------|---------|---------|
| PDF 超时 | 默认 2 分钟 | 移至 `OutputConfig.PDFTimeout`（默认 120s） |
| Chrome 路径 | 候选路径列表 | 支持 `MDPRESS_CHROME_PATH` 环境变量 |
| Mermaid CDN | CDN URL | 集中到 `pkg/utils/constants.go` 的 `MermaidCDNURL` |

#### 6.2.4 错误处理（已完成）

- `renderer.NewHTMLRenderer()` 和 `NewStandaloneHTMLRenderer()` 现返回 `(*Type, error)` 而非调用 `panic`
- `pkg/utils/escape.go` 提供集中式的 `EscapeHTML()`、`EscapeXML()` 和 `EscapeAttr()` 函数

#### 6.2.5 可测试性（已完成）

- `serveOptions` 结构体替代全局变量用于 serve 配置
- `internal/pdf/mock.go` 提供 `mockGenerator` 用于无需 Chromium 的测试
- `server.go` 使用独立的 `http.ServeMux`

### 6.3 剩余的重构机会

- `source/github.go`：添加 `GitLabSource` 以支持更广泛的 Git 托管平台
