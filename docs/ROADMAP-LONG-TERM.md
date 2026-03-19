# mdPress 长期路线图

> 更新日期：2026-03-18
> 由战略规划委员会讨论产出（产品经理、架构师、程序员、测试四个视角）
> 当前状态：v0.2.0 已发布，v0.3.0 开发中

---

## 一、项目定位与长期愿景

mdPress 的长期目标是成为**开源 Markdown 出版工具链的事实标准**——一个从写作到多格式发布、从个人笔记到企业级文档的完整解决方案。

核心竞争力演进路径：

- **v0.x 阶段**：功能追赶（对标 GitBook/mdBook），建立核心用户群
- **v1.0 阶段**：稳定可靠，成为生产级工具
- **v1.x+ 阶段**：平台化与生态化，构建护城河

---

## 二、v0.4.0 — 性能与排版（目标：2026-11）

**主题：更快、更美、更轻量**

### 2.1 Typst 后端替代 chromedp

#### 产品经理

Typst 后端解决两个用户痛点：CI/CD 环境中 Chromium 依赖过重（镜像体积 400MB+），以及无头浏览器的启动延迟（冷启动 3-5 秒）。这不是功能缺失，而是部署体验的质变。目标用户是 CI/CD 重度使用者和容器化部署场景。建议作为可选后端提供，不强制替换 Chromium——两个后端长期共存，用户按场景选择。

#### 架构师

**技术可行性**：Typst 是 Rust 编写的排版系统，CLI 可通过 `os/exec` 调用。核心挑战在于 CSS 主题体系到 Typst 样式的映射。

**迁移路径设计**：

1. 定义 `PDFBackend` 接口，抽象 Chromium 和 Typst 两种实现
2. Chromium 后端保持为默认（`--backend chromium`）
3. Typst 后端作为替代（`--backend typst`）
4. 共享同一套中间表示（章节 HTML 或 AST），后端各自消费

**架构改动**：在 `internal/pdf` 中引入 `Backend` 接口，`ChromiumBackend` 封装现有逻辑，`TypstBackend` 新增。对上层调用透明。

```go
type PDFBackend interface {
    Name() string
    Generate(ctx context.Context, input *PDFInput, output string) error
    Available() bool  // 检测运行环境是否可用
}
```

**性能对比预估**：

| 维度 | Chromium | Typst |
|---|---|---|
| 冷启动 | 3-5 秒 | <0.5 秒 |
| 100 页 PDF | ~30 秒 | ~5 秒（预估） |
| 500 页 PDF | ~120 秒 | ~20 秒（预估） |
| Docker 镜像增量 | +400MB | +50MB |
| CJK 支持 | 优秀 | 优秀（需嵌入字体） |
| CSS 兼容性 | 完整 | 需适配层 |

**风险点**：CSS → Typst 的映射覆盖率难以 100%，初期版本只能支持核心主题（technical、elegant、minimal），自定义 CSS 需要提供 Typst 等价写法的文档。

#### 程序员

**工作量分解**：

- `PDFBackend` 接口抽取 + Chromium 适配：1 周
- Typst `.typ` 模板开发（3 个核心主题）：2 周
- Markdown/HTML → Typst 内容转换器：1.5 周
- CJK 字体嵌入与测试：0.5 周
- 合计：**5 周**

**技术选型建议**：不建议将 Typst 编译为 Go 库（Rust FFI 复杂），直接通过 `os/exec` 调用 Typst CLI。类似当前 chromedp 的模式，但更轻量。Typst 的安装可通过 `mdpress doctor` 检测并提示。

**关键实现细节**：需要开发一个 Markdown AST → Typst markup 的转换器，而非先转 HTML 再转 Typst。这样能保留更多语义信息，排版质量更高。

#### 测试

**测试策略**：

- 视觉回归测试：对同一本书分别用 Chromium 和 Typst 生成 PDF，逐页像素对比，容差阈值 5%
- CJK 排版专项：中日韩混排、标点避头尾、行间距验证
- 边界场景：超长表格、嵌套列表、代码块跨页、数学公式渲染
- 性能基准：10/100/500/1000 页书籍的构建时间对比
- 平台兼容：macOS/Linux/Windows 三平台 Typst CLI 调用验证

---

### 2.2 增量编译

#### 产品经理

增量编译是大型书籍作者的核心体验提升。当前 500 页书籍修改一个章节需要 120 秒全量重建，目标是降到 10 秒以内。这直接影响 `build` 命令的使用体验——`serve` 命令已有 `IncrementalBuilder` 雏形，但 `build` 还是全量构建。

#### 架构师

**缓存设计**：

```
.mdpress-cache/
├── manifest.json          # 文件哈希清单
├── chapters/
│   ├── ch01.html          # 章节级缓存
│   ├── ch02.html
│   └── ...
├── assets/
│   ├── images.json        # 图片哈希映射
│   └── ...
└── metadata.json          # 构建配置快照
```

**依赖图分析**：

- 直接依赖：章节文件本身的内容变化（SHA-256 哈希）
- 间接依赖：图片资源变化、CSS/主题变化、配置变化
- 全局依赖：SUMMARY.md 结构变化 → 目录重建、交叉引用变化 → 关联章节重建

**失效策略**：

| 变化类型 | 失效范围 |
|---|---|
| 单章节内容修改 | 该章节 + 目录页 |
| 图片资源修改 | 引用该图片的章节 |
| CSS/主题修改 | 全量重建 |
| book.yaml 修改 | 全量重建 |
| SUMMARY.md 结构修改 | 全量重建 |
| 交叉引用目标修改 | 引用源章节 + 目标章节 |

**预估加速比**：单章节修改场景下，500 页书籍从 120 秒降至 5-10 秒（加速比 12x-24x）。关键路径是目录重建和 PDF 合并。

#### 程序员

**实现路径**：

1. 复用 `IncrementalBuilder` 的文件哈希逻辑（`serve` 已实现）
2. 为 `build` 命令增加 `--cache` 标志（默认开启，`--no-cache` 跳过）
3. 实现章节级 HTML 缓存的序列化/反序列化
4. 实现 PDF 页面合并（用 `pdfcpu` 或类似库将缓存页和新页合并）

**难点**：PDF 页码的全局重算。即使只改了一个章节，后续章节的页码可能变化（因为改动章节的页数可能增减）。解决方案：目录页码延迟渲染——先生成无页码的 PDF，再二次扫描注入页码。

**工作量**：2.5 周

#### 测试

**正确性验证**：增量构建的输出必须与全量构建**逐字节一致**（PDF 除外，因为 PDF 包含时间戳等元数据，使用视觉比对）。需要构建一个自动化测试框架：对同一本书先全量构建 A，再修改某章节增量构建 B，最后全量构建 C，验证 B == C。

**性能基准**：

| 书籍规模 | 全量构建 | 增量（改 1 章） | 加速比 |
|---|---|---|---|
| 10 页 | baseline | baseline | 目标 2x |
| 100 页 | baseline | baseline | 目标 5x |
| 500 页 | baseline | baseline | 目标 15x |
| 1000 页 | baseline | baseline | 目标 20x |

---

### 2.3 PDF 排版提升

#### 产品经理

企业用户有三类需求：品牌化（页眉页脚模板带 logo）、安全性（水印防泄露）、灵活性（自定义页边距和分栏）。这些都是从"能用"到"商用"的关键跨越。

#### 架构师

基于现有 Chromium 后端的 CSS `@page` 规则扩展，改动集中在主题系统层面：

- **页眉页脚模板**：扩展 `book.yaml` 的 `output.pdf.header/footer` 配置，支持 HTML 模板（可嵌入 logo 图片、页码变量、章节标题变量）
- **水印**：CSS `::after` 伪元素叠加半透明旋转文字，或 HTML 绝对定位层
- **自定义页边距**：`@page { margin: top right bottom left }` 从 `book.yaml` 读取
- **分栏排版**：CSS `column-count` / `column-gap`，适用于词典、参考手册等场景

```yaml
output:
  pdf:
    margin:
      top: "25mm"
      bottom: "25mm"
      left: "20mm"
      right: "20mm"
    watermark:
      text: "CONFIDENTIAL"
      opacity: 0.1
      angle: -45
    columns: 2          # 分栏数
    column_gap: "15mm"
    header:
      template: "{{book.title}} | {{chapter.title}}"
      logo: "assets/logo.png"
    footer:
      template: "第 {{page}} 页 / 共 {{pages}} 页"
```

#### 程序员

改动集中在 `internal/renderer` 和 `internal/theme`，新增 CSS 生成逻辑。1.5 周可完成。水印的主要工作在 CSS 调试——不同页面尺寸下的旋转角度和位置需要精确计算。

#### 测试

PDF 视觉回归测试矩阵：3 个主题 × 5 种页面尺寸 × 有/无水印 × 有/无分栏 = 60 种组合。建议用 `pdf2image` + `pixelmatch` 自动化。

---

### 2.4 PlantUML 支持

#### 产品经理

PlantUML 在企业级技术文档中使用广泛（尤其是 UML 类图、时序图）。与 Mermaid 互补而非替代——Mermaid 侧重简洁的流程图和状态图，PlantUML 侧重复杂的 UML 图。建议作为可选插件，不增加核心依赖。

#### 架构师

**三种实现方案对比**：

| 方案 | 优点 | 缺点 |
|---|---|---|
| 本地 JAR | 离线可用，渲染速度快 | 需要 JRE，增加依赖 |
| 在线渲染 (plantuml.com) | 零依赖 | 需联网，隐私风险，速度不稳定 |
| 内嵌 Go 实现 | 无外部依赖 | 不存在成熟的 Go PlantUML 库 |

**推荐方案**：优先支持在线渲染（默认），本地 JAR 作为可选配置。通过插件系统实现，在 `AfterParse` 钩子中将 `plantuml` 代码块转为 SVG 图片。

```yaml
plugins:
  - name: plantuml
    options:
      renderer: online   # 或 local
      jar_path: /opt/plantuml.jar  # 本地模式
      server: https://www.plantuml.com/plantuml  # 在线模式
```

#### 程序员

在线渲染方案约 1.5 周。核心工作：PlantUML 文本 → 编码 → HTTP 请求 → SVG 响应 → 注入 HTML。本地 JAR 方案额外 0.5 周（Java 检测 + JAR 调用）。建议先只做在线方案，本地 JAR 按需迭代。

#### 测试

需要覆盖：类图、时序图、用例图、活动图、组件图等 10+ 种图表类型。在线渲染需要 mock HTTP 服务用于 CI 测试（避免依赖外部服务）。

---

### 2.5 构建缓存

#### 产品经理

构建缓存与增量编译互为补充。增量编译是同一项目内的加速，构建缓存是跨构建会话的加速（如 CI/CD 中可以缓存 `.mdpress-cache/` 目录）。

#### 架构师

**缓存策略**：

- 缓存键：`SHA256(文件内容 + 构建配置 + 主题版本 + mdpress 版本)`
- 缓存粒度：章节级 HTML + 资源级（图片 base64、Mermaid SVG）
- 存储位置：项目本地 `.mdpress-cache/`，可通过 `--cache-dir` 自定义

**失效机制**：

- mdpress 版本升级 → 全量失效（避免兼容性问题）
- 主题变更 → 全量失效
- 缓存条目超过 30 天未使用 → 自动清理
- `mdpress build --no-cache` → 跳过缓存
- `mdpress clean` → 清除缓存

#### 程序员

与增量编译共享基础设施，额外工作量约 1 周。核心是 manifest.json 的读写和版本兼容性校验。

#### 测试

缓存一致性测试：有缓存与无缓存的构建结果必须一致。缓存失效测试：模拟版本升级、主题变更等场景，验证缓存正确失效。

---

### 2.6 v0.4.0 里程碑时间线

```
2026-09-01  PDFBackend 接口定义 + Typst POC 完成
2026-09-15  增量编译设计评审 + 原型实现
2026-10-01  Typst 核心主题适配完成
2026-10-15  增量编译 + 构建缓存集成
2026-10-30  PDF 排版提升 + PlantUML 插件
2026-11-15  集成测试 + 性能基准测试
2026-11-30  v0.4.0 发布
```

---

## 三、v1.0.0 — 稳定发布（目标：2027-Q1）

**主题：生产就绪、长期支持**

### 3.1 API 冻结标准

#### 产品经理

v1.0.0 的核心承诺是稳定性。API 冻结意味着用户可以放心地在生产流水线中使用 mdPress，不必担心升级破坏现有配置。这是从"尝鲜工具"到"生产级基础设施"的关键转变。

#### 架构师

**需要冻结的接口清单**：

| 类别 | 具体项 | 兼容性承诺 |
|---|---|---|
| CLI 命令 | `build`, `serve`, `init`, `validate`, `themes` | 命令名和核心 flag 不变 |
| CLI flags | `--format`, `--output`, `--config`, `--backend`, `--theme` | 语义不变，可新增不可删除 |
| book.yaml | `book.*`, `chapters.*`, `style.*`, `output.*` | 向后兼容，新增字段用默认值 |
| 插件接口 | `Plugin` 接口、`HookContext`、`Phase` 常量 | v1.0 后只新增不修改 |
| 输出格式 | PDF、HTML、Site、ePub | 输出行为不回退 |

**版本兼容策略**：遵循语义化版本（SemVer）。v1.x.y 中 x 不变时保证向后兼容。`book.yaml` 支持 `version` 字段，mdPress 自动检测并兼容旧版配置。

**不冻结的部分**：内部 Go API（`internal/` 包）、主题 CSS 细节、构建性能特征。

#### 程序员

需要完成的稳定化工作：

1. 对所有 CLI flag 编写集成测试（确保不会意外删除）
2. 对 `book.yaml` 编写 schema 校验（JSON Schema 或等价方案）
3. 对插件接口编写兼容性测试套件
4. 编写 CHANGELOG 自动生成工具

工作量：2 周

#### 测试

引入**契约测试**：为每个公开接口编写契约断言。CI 中如果检测到契约变更但未升级主版本号，自动阻止合并。

---

### 3.2 90% 测试覆盖率路径

#### 产品经理

测试覆盖率是用户信心的基础。90% 不是目标而是底线——核心路径（build pipeline、配置解析、输出生成）应该接近 100%。

#### 架构师

**当前覆盖率评估与目标**：

| 模块 | 当前估计 | v1.0 目标 |
|---|---|---|
| `internal/config` | ~70% | 95% |
| `internal/markdown` | ~65% | 95% |
| `internal/renderer` | ~50% | 90% |
| `internal/pdf` | ~40% | 85%（Chromium 集成测试受限） |
| `internal/output` | ~45% | 90% |
| `internal/plugin` | ~30% | 90% |
| `cmd/` | ~35% | 85% |
| `pkg/utils` | ~60% | 95% |
| **整体** | **~50%** | **90%** |

**策略**：

- 单元测试：所有纯逻辑函数（解析、转换、格式化）
- 集成测试：端到端 build pipeline 的 golden file 测试
- Mock 测试：PDF 生成使用 `MockGenerator`（已有），减少对 Chromium 的测试依赖
- 属性测试：配置解析的 fuzzing 测试

#### 程序员

按模块分配，每个模块补充测试约 1 周，总计约 6-8 周的测试编写工作。建议分散到 v0.4.0 → v1.0.0 的整个周期中，每个 PR 要求增量覆盖率不低于 80%。

#### 测试

**自动化测试演进**：

1. 单元测试：Go 原生 `testing`，表驱动测试风格
2. 集成测试：`testscript` 包模拟 CLI 调用
3. E2E 测试：构建真实书籍项目，校验所有输出格式
4. 视觉回归测试：PDF/HTML 截图比对（Percy 或自建方案）
5. 性能回归测试：每次发布前运行基准测试，禁止性能劣化超 10%

**CI 集成**：GitHub Actions 中增加覆盖率门禁，PR 合并要求覆盖率不降低。

---

### 3.3 VS Code 插件

#### 产品经理

VS Code 是 Markdown 写作者的主要编辑器。插件的核心价值不是替代 CLI，而是提供集成体验：实时预览、配置自动补全、错误高亮、一键构建。目标用户是非 CLI 原教旨用户。

**功能范围（MVP）**：

1. 侧边栏实时预览（WebView，调用 `mdpress serve` 的 WebSocket 端点）
2. `book.yaml` 的 schema 校验和自动补全
3. SUMMARY.md 的可视化章节树
4. 一键构建（触发 `mdpress build`，输出面板展示日志）
5. 问题诊断（调用 `mdpress validate`，映射到编辑器诊断信息）

#### 架构师

**实现方案**：不做 LSP（过重），采用轻量方案：

- WebView 面板展示 `serve` 预览，通过 WebSocket 同步
- `book.yaml` 补全通过 JSON Schema + VS Code 内置 YAML 支持
- 命令面板集成 `mdpress build/serve/validate` 命令
- TreeView 展示 SUMMARY.md 的章节结构

**独立仓库**：`mdpress-vscode`，TypeScript 项目，独立发布到 VS Code Marketplace。

#### 程序员

TypeScript + VS Code Extension API，工作量约 4 周（含 WebView 集成和测试）。核心依赖是 mdPress CLI 已安装——插件本身不嵌入 mdPress，只是调用 CLI。

#### 测试

VS Code 扩展测试框架 `@vscode/test-electron`，模拟编辑器环境。需覆盖 Windows/macOS/Linux 三平台。

---

### 3.4 插件注册中心

#### 产品经理

插件注册中心是生态建设的基础设施。v0.3.0 的插件系统解决"能扩展"，注册中心解决"好发现、好安装"。

**两种方案对比**：

| 方案 | 优点 | 缺点 |
|---|---|---|
| 类似 npm registry（自建） | 完全掌控、搜索体验好 | 运维成本高、需要自建基础设施 |
| GitHub-based（约定式） | 零运维、社区天然支持 | 搜索体验差、元数据分散 |

**建议**：v1.0.0 先做 **GitHub-based** 方案（低成本启动），长期视社区规模决定是否自建。

#### 架构师

**GitHub-based 方案设计**：

- 插件仓库命名约定：`mdpress-plugin-{name}`
- 插件元数据：仓库根目录的 `plugin.yaml`
- 索引仓库：`mdpress/plugin-registry`，自动抓取符合命名约定的仓库信息
- 安装方式：`mdpress plugin install {name}` → 克隆到 `~/.mdpress/plugins/`

```yaml
# plugin.yaml
name: katex
version: 1.0.0
description: "KaTeX math formula rendering"
author: "mdpress-team"
hooks: [after_parse]
compatibility: ">=0.3.0"
```

**长期演进**：当插件数量超过 50 个时，考虑自建 registry API，提供搜索、版本管理、下载统计等功能。

#### 程序员

GitHub-based 方案实现约 2 周：`mdpress plugin install/list/remove` 命令 + 插件下载/缓存逻辑。索引仓库的自动化可用 GitHub Actions 每日抓取。

#### 测试

插件安装/卸载/升级的 E2E 测试。需要 mock GitHub API 用于 CI。安全性测试：验证插件不能访问 mdPress 核心进程的文件系统（沙箱隔离）。

---

### 3.5 CLI 国际化

#### 产品经理

优先级最低的 v1.0.0 特性。主要受益者是中文和日文用户社区。建议只做中英双语，其他语言由社区贡献。

#### 架构师

**i18n 框架选择**：

| 方案 | 优点 | 缺点 |
|---|---|---|
| `go-i18n` | 成熟、功能完整、支持复数 | 引入外部依赖 |
| 自建 `embed` + JSON | 零依赖、简单 | 需自行处理复数和格式化 |
| `internal/i18n` 扩展 | 已有基础 | 需评估当前实现的扩展性 |

**建议**：扩展现有 `internal/i18n` 模块，使用 Go 1.16+ 的 `embed` 包嵌入翻译文件。保持零外部依赖。

#### 程序员

1 周。全量提取 CLI 输出字符串 → 创建 `locales/en.json` 和 `locales/zh.json` → 在所有输出点调用 `i18n.T("key")`。

#### 测试

每种语言的完整输出回归测试。确保切换语言不影响非文本行为。

---

### 3.6 v1.0.0 里程碑时间线

```
2026-12-01  API 冻结草案发布，征求社区反馈
2026-12-15  测试覆盖率达到 80%
2027-01-01  VS Code 插件 alpha 版发布
2027-01-15  插件注册中心上线（GitHub-based）
2027-02-01  测试覆盖率达到 90%，API 正式冻结
2027-02-15  CLI 国际化完成
2027-03-01  v1.0.0 RC1 发布
2027-03-15  v1.0.0 正式发布
```

---

## 四、v1.x+ — 长期愿景

### 4.1 在线 Playground

#### 产品经理

Playground 是获客漏斗的顶端——零安装体验 mdPress。对标 Typst 的 app.typst.app 和 Overleaf 的在线编辑器。核心价值是让用户在 30 秒内体验到 mdPress 的能力。

#### 架构师

**两条技术路径**：

**路径 A：Go → WebAssembly**

- 将 mdPress 核心编译为 WASM，浏览器内运行
- 优点：纯前端，无服务器成本
- 限制：PDF 输出不可能（Chromium 无法在 WASM 中运行），只能预览 HTML/Site
- 适合：轻量预览 Playground

**路径 B：云端服务**

- 服务端运行 mdPress CLI，前端展示结果
- 优点：完整功能，支持 PDF 输出
- 限制：需要服务器基础设施，安全隔离（用户输入可能包含恶意内容）
- 适合：完整功能 Playground

**建议**：先走路径 A（WASM + HTML 预览），作为 MVP 快速上线。长期如果有商业化需求，再考虑路径 B。

**WASM 方案技术要点**：

- 使用 `GOOS=js GOARCH=wasm` 编译
- 剥离 chromedp、fsnotify 等不兼容依赖
- 前端使用 Monaco Editor 或 CodeMirror 作为编辑器
- 实时渲染 Markdown → HTML 预览

#### 程序员

WASM 路径：2-3 个月（含前端编辑器）。核心难点是 Go WASM 二进制的体积优化——初始可能 20MB+，需要通过 TreeShaking 和压缩降到 5MB 以内。

#### 测试

安全性测试（XSS 防护、资源限制）、并发测试（多用户同时编辑）、浏览器兼容性测试（Chrome/Firefox/Safari）。

---

### 4.2 AI 集成可能性

#### 产品经理

AI 集成不是核心功能，而是锦上添花。三个方向的优先级：

1. **内容摘要生成**（高）：为每个章节自动生成摘要，用于 SEO 和目录页
2. **自动目录优化**（中）：分析章节结构，建议更合理的组织方式
3. **智能排版建议**（低）：检测排版问题（过长段落、缺少配图、标题层级跳跃）

#### 架构师

**实现方案**：通过插件系统集成，不引入核心依赖。AI 功能作为可选插件，用户自行配置 API Key。

```yaml
plugins:
  - name: ai-assistant
    options:
      provider: openai  # 或 anthropic、ollama
      api_key: ${OPENAI_API_KEY}
      features:
        - summary          # 章节摘要
        - toc_suggest      # 目录优化建议
        - lint             # 排版建议
```

**架构要点**：

- AI 调用必须异步且可跳过（网络不可用时优雅降级）
- 生成的内容作为建议而非自动应用
- 支持本地模型（Ollama）以满足隐私需求

#### 程序员

作为插件开发，工作量 2-3 周。核心是 API 调用封装和结果格式化。建议优先支持 OpenAI API（用户量最大），后续按需增加 Anthropic 和 Ollama。

#### 测试

AI 输出不确定性要求测试策略不同：验证接口调用正确性而非输出内容。Mock API 响应用于 CI。

---

### 4.3 协作编辑

#### 产品经理

协作编辑是一个独立产品级别的功能。短期内不建议投入，但可以设计预留：

- 近期：支持 Git 工作流（多人通过 Git 协作，mdPress 做构建）
- 中期：基于 WebSocket 的简单协作（锁机制，同一时间一人编辑一章节）
- 远期：OT/CRDT 的实时协作（对标 Google Docs）

#### 架构师

近期方案（Git 工作流）不需要 mdPress 本身的改动，只需要文档和最佳实践指南。中期的章节锁方案可以基于 Playground 的后端服务扩展。远期的 CRDT 方案是一个全新的技术栈，建议独立评估。

#### 程序员

近期：0 工作量（纯文档）。中期：4-6 周（WebSocket 服务端 + 简单编辑器）。远期：6-12 个月（全新项目）。

#### 测试

并发编辑的冲突测试、数据一致性测试、网络中断恢复测试。

---

### 4.4 商业化路径

#### 产品经理

mdPress 的开源定位不变，但可以探索可持续的商业化路径：

**模式一：SaaS 版本**

- 在线 Playground 的付费版（更大文件、更多格式、自定义域名）
- 类似 GitBook 的托管方案
- 月费 $9-29，面向个人和小团队

**模式二：企业版功能**

- SSO/SAML 集成
- 审计日志
- 自定义品牌白标
- SLA 保障
- 年费 $499-2999/团队

**模式三：增值服务**

- 官方主题/模板商店（分成模式）
- 插件市场（分成模式）
- 技术支持和咨询服务

**建议**：v1.0.0 后先尝试模式一（SaaS），验证市场需求。企业版按需开发。

#### 架构师

商业化对架构的影响：

- SaaS 版本需要多租户隔离、用量计量、支付集成
- 企业版需要配置化的权限系统
- 这些都不应影响开源核心的架构纯粹性

建议在开源核心之上，商业功能作为独立层叠加，避免核心代码的复杂度膨胀。

---

### 4.5 生态系统建设

#### 产品经理

生态三板斧：插件市场、主题市场、模板库。

**插件市场**：v1.0.0 的 GitHub-based 注册中心演进为带搜索和评分的 Web 界面。

**主题市场**：参考 Hugo Themes，提供主题预览、一键安装。初期 10-15 个官方主题，后续开放社区贡献。

**模板库**：预置书籍模板（技术手册、API 文档、学术论文、小说等），用户通过 `mdpress init --template technical-manual` 快速开始。

#### 架构师

主题和模板的分发机制与插件一致，共享注册中心基础设施。主题是 CSS + 配置的包，模板是 book.yaml + 目录结构 + 示例内容的包。

---

### 4.6 与其他工具链集成

#### 产品经理

降低用户迁移成本，扩大内容源：

| 集成方向 | 用户价值 | 实现复杂度 |
|---|---|---|
| Notion → mdPress | 高（Notion 用户基数大） | 高（Notion API + 格式转换） |
| Obsidian → mdPress | 高（知识管理 → 出版） | 中（本地 Markdown，主要处理 Wikilink） |
| Confluence → mdPress | 中（企业用户） | 高（API + HTML → Markdown） |
| mdBook → mdPress | 中（Rust 社区） | 低（格式接近） |

#### 架构师

通过 `Source` 接口扩展实现。Notion 和 Confluence 作为新的 Source 实现，处理 API 调用和格式转换。Obsidian 主要是 Markdown 方言兼容（Wikilink `[[]]` → 标准链接）。

```go
// 未来 Source 实现
type NotionSource struct { ... }     // Notion API → Markdown
type ObsidianSource struct { ... }   // Wikilink 转换
type ConfluenceSource struct { ... } // Confluence API → Markdown
```

#### 程序员

每个集成约 2-3 周。建议按用户需求排序：Obsidian 优先（实现简单、用户需求大），Notion 其次。

---

### 4.7 性能极限：万章节书籍

#### 产品经理

极端场景的支持代表产品成熟度。目标：1000+ 章节的书籍（如大型 API 文档、法律法规汇编）能在 5 分钟内构建完成。

#### 架构师

**瓶颈分析**：

1. 内存：1000 章节 × 每章 100KB HTML ≈ 100MB，可控
2. Chromium PDF 渲染：串行渲染是瓶颈，需要并行化
3. 目录生成：O(n) 复杂度，不是瓶颈
4. 交叉引用解析：当前 O(n²)，需优化为 O(n log n)

**优化方案**：

- 并行章节处理：`sync.Pool` + worker goroutine 池
- PDF 分块渲染：每 50 章节渲染一个 PDF 分块，最后合并
- 内存映射：大文件使用 `mmap` 减少内存拷贝
- 流式输出：ePub 不在内存中组装完整 ZIP，流式写入

#### 程序员

并行化改造约 3 周。核心改动在 `cmd/build_orchestrator.go` 的章节处理循环。需要注意章节间的顺序依赖（交叉引用、术语表）。

#### 测试

生成 1000/5000/10000 章节的合成书籍项目，运行性能基准测试。监控内存峰值、CPU 利用率、构建时间。

---

### 4.8 无障碍（Accessibility）

#### 产品经理

WCAG 2.1 AA 合规是公共部门和教育机构的采购要求。HTML 和 ePub 输出的无障碍支持直接扩大可触达的用户群。

#### 架构师

**无障碍改进清单**：

| 输出格式 | 改进项 |
|---|---|
| HTML/Site | 语义化标签、ARIA 属性、键盘导航、高对比度模式、跳转链接 |
| ePub | EPUB Accessibility 1.0 合规、alt text 检查、阅读顺序标记 |
| PDF | PDF/UA 合规（Tagged PDF）、书签结构、替代文本 |

**实现方式**：

- 在 Markdown → HTML 转换时注入语义化属性
- 主题 CSS 增加高对比度变体
- `mdpress validate` 增加无障碍检查（图片缺少 alt、标题层级跳跃等）

#### 程序员

HTML 无障碍：2 周。ePub 无障碍：1 周。PDF/UA：依赖 Typst 后端（Chromium 生成 Tagged PDF 较困难），3 周。

#### 测试

使用 axe-core 自动化扫描 HTML 输出。使用 DAISY ACE 检查 ePub 无障碍合规。手动屏幕阅读器测试（VoiceOver、NVDA）。

---

## 五、完整版本路线图总览

```
2026-03  v0.2.0 ████████████████████████████████████████ 已发布
         核心：HTML 输出、GitHub 源、SUMMARY.md、serve 预览

2026-08  v0.3.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 开发中
         核心：ePub、插件系统、SPA 站点、KaTeX、Mermaid、迁移工具

2026-11  v0.4.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 计划中
         核心：Typst 后端、增量编译、PDF 排版、PlantUML、构建缓存

2027-Q1  v1.0.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 计划中
         核心：API 冻结、90% 测试覆盖、VS Code 插件、插件注册中心

2027-Q2  v1.1.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 愿景
         核心：在线 Playground（WASM 版）、Obsidian 集成、主题市场

2027-Q3  v1.2.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 愿景
         核心：无障碍合规、Notion 集成、性能极限优化

2027-Q4  v1.3.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 愿景
         核心：AI 辅助插件、协作编辑（章节锁模式）

2028+    v2.0.0 ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 远期愿景
         核心：SaaS 版本、企业版、实时协作、DOCX 输出
```

---

## 六、优先级总矩阵

| 版本 | 特性 | 用户价值 | 复杂度 | 优先级 |
|---|---|---|---|---|
| **v0.4.0** | Typst 后端 | 中 | 高 | P1 |
| **v0.4.0** | 增量编译 | 高 | 中 | P0 |
| **v0.4.0** | PDF 排版提升 | 中 | 低 | P1 |
| **v0.4.0** | PlantUML 插件 | 中 | 中 | P2 |
| **v0.4.0** | 构建缓存 | 中 | 中 | P1 |
| **v1.0.0** | API 冻结 | 高 | 中 | P0 |
| **v1.0.0** | 90% 测试覆盖 | 高 | 高 | P0 |
| **v1.0.0** | VS Code 插件 | 中 | 中 | P1 |
| **v1.0.0** | 插件注册中心 | 中 | 中 | P1 |
| **v1.0.0** | CLI 国际化 | 低 | 低 | P2 |
| **v1.1.0** | 在线 Playground | 高 | 高 | P1 |
| **v1.1.0** | Obsidian 集成 | 高 | 中 | P1 |
| **v1.1.0** | 主题市场 | 中 | 中 | P2 |
| **v1.2.0** | 无障碍合规 | 中 | 中 | P1 |
| **v1.2.0** | Notion 集成 | 高 | 高 | P2 |
| **v1.2.0** | 万章节性能优化 | 低 | 高 | P2 |
| **v1.3.0** | AI 辅助插件 | 中 | 中 | P2 |
| **v1.3.0** | 协作编辑（章节锁） | 中 | 高 | P2 |
| **v2.0.0** | SaaS 版本 | 高 | 极高 | 待评估 |
| **v2.0.0** | 企业版 | 中 | 高 | 待评估 |

---

## 七、技术债务管理

#### 架构师视角

v0.x 阶段允许积累技术债务以换取开发速度，但必须在 v1.0.0 之前清偿。

**当前已知技术债务**：

| 债务 | 风险等级 | 计划清偿版本 |
|---|---|---|
| `build.go` 仍有部分逻辑未迁移到 `BuildOrchestrator` | 中 | v0.4.0 |
| 交叉引用解析 O(n²) 复杂度 | 低（当前书籍规模下不可感知） | v1.0.0 |
| 主题 CSS 缺少单元测试 | 中 | v0.4.0 |
| Windows CI 测试缺失 | 高 | v0.4.0 |
| 错误消息未完全国际化 | 低 | v1.0.0 |
| `internal/pdf` 与 chromedp 耦合度高 | 中 | v0.4.0（PDFBackend 接口化） |

**技术债务管控原则**：

1. 每个版本预留 15% 的工期用于偿还技术债务
2. 新增特性不得引入超过"中"风险等级的技术债务
3. v1.0.0 之前所有"高"风险债务必须清零

---

## 八、决策原则

1. **稳定性优先于新特性**：v1.0.0 的核心承诺是稳定性，不引入大型新特性
2. **可选优先于必选**：新功能尽可能作为可选项（插件、后端选择），不增加默认依赖
3. **渐进增强**：功能分阶段交付（如 Typst 先支持核心主题，再逐步覆盖自定义样式）
4. **社区驱动**：长期愿景的优先级由社区反馈和 GitHub Issues 投票决定
5. **开源核心不妥协**：商业化功能不影响开源核心的完整性和可用性
6. **测试先行**：新功能开发前先定义验收标准和测试用例
7. **向后兼容**：v1.0.0 后的任何变更必须保证 book.yaml 和 CLI 的向后兼容

---

> 本文档将随项目进展持续更新。欢迎通过 [GitHub Issues](https://github.com/yeasy/mdpress/issues) 参与讨论和贡献。
