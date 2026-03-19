# mdpress 技术调研报告

> 调研日期：2026-03-18
> 调研范围：Typst 后端、WASM Playground、VS Code 插件、插件系统、增量编译、ePub 3

---

## 1. Typst 后端可行性

### 调研结论

**Typst 当前状态**：最新版本 0.14.2（2025年12月），仍处于 0.x 阶段，允许破坏性变更，但已有明确的 1.0 路线图。0.14 版本重点加强了可访问性和 PDF 功能，包含少量破坏性变更（标签/URL 不能为空、`pdf.embed` 改为 `pdf.attach` 等）。

**Go 调用 Typst 的方式对比**：

| 方式 | 优点 | 缺点 |
|------|------|------|
| CLI subprocess | 最稳定、无依赖、易跨平台 | 进程开销、IPC 开销、需单独部署二进制 |
| WASM | 无二进制依赖、轻量级 | 浏览器环境限制、Go 调用 WASM 不成熟 |
| FFI (cgo) | 零开销、进程内 | 跨平台困难、需编译 Rust、维护复杂 |

已有的 Go 集成包：`go-typst`（Dadido3）、`gotypst`（francescoalemanno）、`typst-go`（hiifong），均采用 CLI 子进程方式。

**性能对比**（Typst vs Chromedp）：

- Typst 生成 4 页 PDF：**356.5ms**（vs XeLaTeX 9.653s，快 27 倍）
- Typst 500 页文档比 speedata Publisher 快 28 倍
- Typst 0.12 多线程优化带来 2-3 倍性能提升，段落布局算法提升 6 倍
- Chromedp 内存占用 200-500MB+，性能波动大；Typst 轻量且稳定
- 结论：Typst 在文档生成场景中胜出 **5-10 倍**，但 Chromedp 适合复杂网页布局

**中文支持**：

- 内置 Noto Serif CJK SC / Noto Sans CJK SC 支持
- 社区字体推荐：霞鹜文楷 SC、Fandol 系列
- 支持 `lang: "zh"` / `region: "CN"` 属性
- 已知限制：SimSun、SimHei 等常用字体无法加粗或斜体（Issue #635）
- 社区包：`zh-kit`、`ctyp`、`jastylest-zh` 提供中文排版函数
- 排版质量评估：**8/10**

**字体管理**：优先级为 项目嵌入字体 > 服务器字体 > 系统字体。CLI 内置 Libertinus Serif、New Computer Modern、DejaVu Sans Mono。支持 `--font-path` 和 `TYPST_FONT_PATHS` 环境变量。

**Markdown → Typst 转换工具**：

| 工具 | 类型 | 成熟度 |
|------|------|--------|
| Pandoc 3.1.2+ | CLI，官方原生 Typst writer | 生产就绪 |
| cmarker | Typst 包，CommonMark 标准 | 实验阶段 |
| ConvertorPanda | 在线工具 | 可用 |

**Pandoc Typst Writer**：Haskell 实现（`Text.Pandoc.Writers.Typst` 模块），已从实验转向稳定支持，支持脚注、表格、引用等复杂结构。支持 `typst:prop` 属性控制输出格式。

### 推荐方案

采用 **CLI subprocess** 方式集成 Typst：

```
Markdown → Pandoc (markdown → typst) → Typst CLI (subprocess) → PDF
```

使用 `go-typst` 包装 CLI 调用。中文配置：

```typst
#set text(
  font: ("Noto Serif", (name: "Noto Serif CJK SC", covers: "cjk")),
  lang: "zh", region: "CN", size: 11pt
)
```

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| API 破坏性变更 | 中 | 版本锁定至 0.14.2，vendor 管理 CLI 二进制 |
| 中文字体样式限制 | 低 | 改用 Noto Serif CJK SC（支持字重 400/700） |
| 进程启动开销累计 | 低 | 批量编译合并任务 |
| 跨平台二进制分发 | 低 | CI/CD 多平台构建 + Docker |

### 参考链接

- [Typst 官方网站](https://typst.app/)
- [Typst GitHub](https://github.com/typst/typst)
- [go-typst](https://github.com/Dadido3/go-typst)
- [Pandoc Typst Writer](https://pandoc.org/typst-property-output.html)
- [cmarker Typst 包](https://typst.app/universe/package/cmarker/)
- [Typst vs XeLaTeX 性能对比](https://slhck.info/software/2025/10/25/typst-pdf-generation-xelatex-alternative.html)
- [Zerodha 大规模 PDF 生成案例](https://zerodha.tech/blog/1-5-million-pdfs-in-25-minutes/)
- [zh-kit 社区包](https://typst.app/universe/package/zh-kit/)
- [CJK 字体样式问题 #635](https://github.com/typst/typst/issues/635)

---

## 2. WASM Playground 技术方案

### 调研结论

**Go 编译为 WASM 的现状**：

| 维度 | 标准 Go | TinyGo |
|------|---------|--------|
| 文件大小 | 2.88-3.1 MiB（Hello World） | 100-500KB，最优 31KB (gzip) |
| 性能 | 比原生慢 10-15% | 略慢于标准 Go |
| 标准库 | 大部分支持，单线程限制 | 子集支持 |
| goroutine | 阻塞式（无真正并行） | 同样受限 |

Go 1.24 新增 `go:wasmexport` 指令，改进 WASM 导出机制。GC 性能提升约 10%。

**goldmark 在 WASM 中的可运行性**：可以运行。已有实例项目 [render-md](https://github.com/milinddethe15/render-md) 使用 Go + WASM + goldmark 实现客户端 Markdown 编辑。但更优方案是 [markdown-wasm](https://github.com/rsms/markdown-wasm)（基于 C 的 md4c，仅 31KB gzipped）。

**类似项目参考**：

| 项目 | 技术栈 | 特点 |
|------|--------|------|
| Typst Web (typst.app) | Rust + WASM + Leptos | 完整编辑器、实时编译 |
| markdown-wasm | C + WASM | 31KB gz，高性能 |
| wasm-typst-studio-rs | Rust + Leptos + Tauri | 双端应用 |
| TeXlyre | React + WASM + Yjs | 实时协作 |

**WASM 性能限制**：桌面浏览器比原生慢 10-15%；移动浏览器慢 3-4 倍；500KB+ 在移动网络下加载明显。

### 推荐方案

**分阶段实施**：

**阶段一（MVP）**：JavaScript + markdown-wasm
- 31KB 体积，最快原型速度
- 验证 Playground 产品概念

**阶段二（正式版）**：Go + TinyGo（如果团队已有 Go 基础）
- 复用 goldmark 生态，目标体积 300-400KB gzipped
- 使用 Web Workers 避免 UI 阻塞

**最优方案**（如果愿意引入 Rust）：Rust + Leptos + markdown-wasm
- 总体积约 170KB gzipped，1 秒内加载
- Typst 已验证此架构可行

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| Go WASM 体积过大 | 高 | TinyGo + wasm-opt + gzip；或使用 JS/Rust 方案 |
| 移动设备性能差 | 高 | 渐进式功能，移动端降级 |
| goroutine 阻塞 UI | 高 | Web Workers 处理计算 |
| 标准库不兼容 | 中 | goldmark 本身兼容，扩展需逐个测试 |
| 首次加载延迟 | 中 | Service Worker 缓存 + Brotli 压缩 |

### 参考链接

- [Go 官方 Wiki: WebAssembly](https://go.dev/wiki/WebAssembly)
- [Go 1.24 WASM 支持](https://cloud.google.com/blog/products/application-development/go-1-24-expands-support-for-wasm)
- [TinyGo 优化指南](https://tinygo.org/docs/guides/optimizing-binaries/)
- [Fermyon: Shrink TinyGo WASM by 60%](https://www.fermyon.com/blog/optimizing-tinygo-wasm)
- [markdown-wasm](https://github.com/rsms/markdown-wasm)
- [render-md (Go+goldmark+WASM)](https://github.com/milinddethe15/render-md)
- [typst.ts](https://github.com/Myriad-Dreamin/typst.ts)
- [wasm-typst-studio-rs](https://github.com/automataIA/wasm-typst-studio-rs)

---

## 3. VS Code 插件最佳实践

### 调研结论

**书籍编辑插件生态**：

- **Learn Authoring Pack**：微软官方文档编辑套件（Linting、拼写检查、图片压缩）
- **Foam**：模块化 PKM，充分利用 VS Code 生态的 1000+ 扩展
- **Dendron**：层级笔记，大部分功能集成在扩展内部
- **Quarto**：支持多格式文档和书籍，代码执行 + 并排预览
- **mdBook**：VS Code 上无专门官方插件，用户通过通用 Markdown 工具 + CLI 使用

**LSP vs WebView 方案对比**：

| 维度 | LSP | WebView |
|------|-----|---------|
| 适用场景 | 自动完成、诊断、跳转定义、符号重命名 | 自定义 UI、富交互编辑 |
| 性能 | 独立进程，不阻塞编辑器 | 有性能和可访问性开销 |
| 跨编辑器 | 可复用于 Vim、Neovim 等 | VS Code 专属 |
| 安全性 | 进程隔离，较高 | 需严格配置 CSP |

最佳实践：LSP 处理语言智能，WebView 提供可视化编辑，两者结合使用。

**实时预览实现**：

- Markdown Preview Enhanced 架构：PreviewProvider → NotebooksManager → Notebook → markdown-it 渲染
- 性能关键：使用 `getState()`/`setState()` 而非 `retainContextWhenHidden`；大文件使用虚拟滚动；增量 DOM 更新

### 推荐方案

**短期（原型）**：基于 `CustomTextEditorProvider` + WebView 预览面板

```
扩展核心
├─ CustomTextEditorProvider（利用 VS Code TextDocument）
├─ 并排预览 WebView（markdown-it 渲染）
└─ 命令面板集成（导出 PDF/HTML）
```

不实现 LSP，先验证编辑体验。WebView 配置：`enableScripts: true`、`retainContextWhenHidden: false`。

**中期**：增加简化版 LSP 支持跨文件链接、符号导航。

**长期**：参考 Foam 的模块化特性系统（feature 系统），而非 Dendron 的"万能扩展"模式。

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| Webview UI Toolkit 已弃用（2025年1月） | 中 | 使用 Svelte/React + 手写 CSS |
| WebView 大文件卡顿 | 中 | 虚拟滚动 + 增量 DOM 更新 |
| retainContextWhenHidden 内存泄漏 | 中 | 默认禁用，使用 getState/setState |
| 安全隐患（WebView 内容注入） | 低 | 严格 CSP + 白名单本地资源 |

### 参考链接

- [VS Code Custom Editor API](https://code.visualstudio.com/api/extension-guides/custom-editors)
- [VS Code Webview API](https://code.visualstudio.com/api/extension-guides/webview)
- [VS Code LSP 指南](https://code.visualstudio.com/api/language-extensions/language-server-extension-guide)
- [Foam](https://foambubble.github.io/foam/)
- [Dendron](https://marketplace.visualstudio.com/items?itemName=dendron.dendron)
- [Markdown Preview Enhanced](https://github.com/shd101wyy/vscode-markdown-preview-enhanced)
- [Learn Authoring Pack](https://learn.microsoft.com/en-us/contribute/content/how-to-write-docs-auth-pack)

---

## 4. 插件系统设计参考

### 调研结论

**mdBook 插件机制**：
- stdin/stdout JSON 协议：语言无关，任何语言均可实现
- 两阶段调用：先检查兼容性（返回 0），再传入完整书籍结构处理
- Preprocessor（渲染前修改内容）与 Renderer（生成最终格式）分离
- 通过 `book.toml` 配置文件注册插件

**Gatsby 插件系统**：
- 丰富生命周期：`onPreInit → sourceNodes → onCreateNode → onPreBuild → onPostBuild`
- 基于 Redux 状态管理 + `api-runner` 顺序执行
- 三类 API：`gatsby-node.js`、`gatsby-browser.js`、`gatsby-ssr.js`

**Hugo Modules**：
- 基于 Go Modules（`go.mod`）声明依赖
- 文件系统联合挂载：自动将 static/content/layouts/data/assets/i18n 挂载到统一文件系统
- 任何 Hugo 项目都可作为模块

**Obsidian 插件 API**：
- 发布-订阅事件系统，`registerEvent()` 自动处理清理
- TypeScript 开发，esbuild 编译为 main.js
- 活跃社区生态

**Go 插件方案对比**：

| 方案 | 优点 | 缺点 |
|------|------|------|
| Go builtin plugin | 同进程高速 | 版本兼容严格、跨平台差、无法热重载 |
| RPC 子进程 (HashiCorp go-plugin) | 安全隔离、热重载、跨平台 | 多进程开销（50-100μs） |
| stdin/stdout JSON (mdBook 风格) | 语言无关、安全隔离、易分发 | 进程启动开销 |

### 推荐方案

**采用类 mdBook 的 stdin/stdout JSON 协议**：

```
mdpress (main)
├─ preprocessor subprocess → stdin/stdout (JSON)
├─ renderer subprocess → stdin/stdout (JSON)
└─ plugin registry (book.toml)
```

核心理由：
1. **语言无关**：插件可用 Go/Python/Rust/JS 等任何语言编写
2. **安全隔离**：插件崩溃不影响主程序
3. **易于分发**：独立二进制，无版本兼容性问题
4. **已验证**：mdBook 已证明此方案的可行性

**实现建议**：
- 定义 JSON Schema（含 context、config、book 三部分）
- 编写 Go/Python/Node.js SDK 简化开发
- 支持 semver 版本检查
- 生命周期：`Load Config → Discover Plugins → Execute Preprocessors → Prepare Book → Execute Renderers → Output`

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| JSON 协议向后兼容 | 中 | JSON Schema + 可选字段策略 + 版本号管理 |
| 子进程错误调试难 | 中 | 完整日志系统 + 插件超时控制 |
| 大型书籍多进程开销 | 低 | 进程池 + 缓存 + 允许禁用不需要的插件 |
| 安全隐患 | 中 | 只加载信任插件，文档明确安全提示 |

### 参考链接

- [mdBook Preprocessors](https://rust-lang.github.io/mdBook/for_developers/preprocessors.html)
- [mdBook Renderers](https://rust-lang.github.io/mdBook/format/configuration/renderers.html)
- [mdBook 插件案例研究](https://eli.thegreenplace.net/2025/plugins-case-study-mdbook-preprocessors/)
- [Gatsby Lifecycle APIs](https://www.gatsbyjs.com/docs/conceptual/gatsby-lifecycle-apis/)
- [Hugo Modules](https://gohugo.io/hugo-modules/)
- [Obsidian Plugin API](https://github.com/obsidianmd/obsidian-api)
- [Go 插件系统设计](https://eli.thegreenplace.net/2021/plugins-in-go/)
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)

---

## 5. 增量编译技术

### 调研结论

**Hugo 的增量编译**：
- Watch 模式采用**部分渲染策略**：只重新渲染当前页面、首页和最近访问的前 10 个页面
- 通过 fsnotify 监控文件变化，支持 `--poll` 参数配置轮询间隔
- 局限：每次构建仍需读取整个源代码树，真正的增量编译依赖图追踪尚未实现

**Vite 的缓存策略**：
- 内容哈希 + ETag：文件名含哈希（`index-G34XebCm.js`），安全缓存 1 年
- HMR：仅对修改模块失效，实现 <50ms 热更新
- 分层缓存：哈希资源缓存 1 年，非哈希资源 1 小时；稳定依赖单独 chunk

**Turbopack 的做法**：
- 增量计算：函数级缓存 + 延迟打包（按需）
- HMR 时间 <10ms（Vite 约 50ms）
- 文件变化自动触发增量图更新

**文件级依赖追踪**：
- DAG 依赖图：通过 `-MMD` 自动生成依赖或运行时监控文件访问
- Rust 编译器方案：查询依赖图 + **红绿标记算法**进行变更检测 + 写时复制策略

**Content-Addressable Storage (CAS)**：
- Bazel：SHA256 哈希为键，双层结构（Action Cache + CAS），支持远程缓存（gRPC/REST）
- Nix：固定长度哈希路径（`/nix/store/hash-name/`），完全可复现构建
- Buildbarn：远程执行协议支持（Bazel、Buck2、BuildStream）

### 推荐方案

**三阶段实施**：

**阶段一：基础缓存层**（优先实现，预期 3-5x 加速）
```
1. 文件哈希追踪：计算源文件内容哈希，存储在 .cache/file_hashes.json
2. 文件级依赖图：解析时记录 markdown → includes → dependencies
3. 反向依赖映射：文件改变时快速查找所有受影响文件
```

**阶段二：内容哈希缓存**（预期额外 2-3x 加速）
```
1. 产物内容哈希：哈希不变时直接复用
2. AST 缓存：缓存 Markdown AST 中间表示
3. LRU 淘汰策略：.cache/outputs/{hash}.html
```

**阶段三：分布式缓存**（团队加速 5-10x）
```
1. CAS 存储：SHA256 哈希为键
2. 远程后端：开发环境用 SQLite，CI 用 S3
3. 团队共享中间产物
```

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| 缓存一致性 | 中 | `--clean` 强制重建 + 单元测试验证 |
| 缓存磁盘爆炸 | 中 | 配置化大小限制（如 500MB）+ 定期清理 |
| 并发修改检测失效 | 低 | 文件事件合并（100ms 窗口）+ 原子操作 |
| 跨平台文件系统差异 | 低 | 使用 Go fsnotify 库 + 多平台测试 |
| 缓存版本不兼容 | 低 | 缓存文件加版本号，不匹配时自动清除 |

### 参考链接

- [Hugo server 文档](https://gohugo.io/commands/hugo_server/)
- [Hugo 增量编译讨论 #1643](https://github.com/gohugoio/hugo/issues/1643)
- [Vite HMR API](https://vite.dev/guide/api-hmr)
- [Vite vs Turbopack vs RSpack 2025 对比](https://drcodes.com/posts/vite-vs-turbopack-vs-rspack-2025-javascript-bundler-guide/)
- [Rust 增量编译详解](https://rustc-dev-guide.rust-lang.org/queries/incremental-compilation-in-detail.html)
- [Bazel 远程缓存](https://bazel.build/remote/caching)
- [Nix + Bazel 可复现构建](https://www.tweag.io/blog/2018-03-15-bazel-nix/)
- [Gradle 增量构建](https://docs.gradle.org/current/userguide/incremental_build.html)
- [Makefile 正确的增量构建](https://www.evanjones.ca/makefile-dependencies.html)

---

## 6. ePub 3 最佳实践

### 调研结论

**ePub 3.3 规范**（2025年5月 W3C 推荐标准）：
- 可访问性首次集成为核心部分（不再独立规范）
- 与 ePub 3.2 完全兼容
- 基于 HTML5，支持音视频、JavaScript 交互、MathML
- 完整支持 CJK、RTL 等多语言系统

**ePub 3 vs ePub 2 关键差异**：

| 特性 | ePub 2 | ePub 3 |
|------|--------|--------|
| 技术基础 | 旧 HTML | HTML5 |
| 多媒体 | 不支持 | 音视频、动画 |
| 交互 | 不支持脚本 | JavaScript |
| 数学公式 | 不支持 | MathML |
| 可访问性 | 基础 | 完整 WCAG + TTS |
| CJK | 基本 | 完整支持 |

**Go 语言 ePub 库**：

| 库 | 功能 | 状态 |
|------|------|------|
| go-shiori/go-epub | 创建生成，EPUB 3.0 + 2.0 兼容 | 活跃（2026年1月更新） |
| bmaupin/go-epub | 创建生成 | 已弃用 |
| ArcadiaLin/go-epub | 读取解析 | 维护中 |

推荐：**go-shiori/go-epub**，支持多媒体（AddVideo/AddAudio），完整文件夹组织。

**EPUBCheck 验证**：最新版 5.0.1，完全支持 EPUB 3.3。常见错误：

| 错误 | 原因 | 修复 |
|------|------|------|
| PKG-026 | 字体媒体类型错误 | 改为 `font/sfnt` |
| 元数据缺失 | 缺 title/author/language | 补充必需元数据 |
| 无效 XHTML | 标签未闭合 | XML 验证器 |
| MIMETYPE 缺失 | ZIP 首项非 mimetype | 确保 uncompressed mimetype |

**平台兼容性**：

| 特性 | Apple Books | Kindle | Kobo |
|------|-------------|--------|------|
| EPUB 3 | 完全支持 | 不原生支持（自动转 AZW3） | 支持（Kepub 更优） |
| CSS3 | 完整 | 有限 | 常用特性 |
| JavaScript | 受限支持 | 不支持 | 不支持 |
| 嵌入字体 | 支持 | 限制 | 需放 Fonts 文件夹 |

**中文电子书注意**：
- UTF-8 编码保存
- 推荐 Source Han Sans CJK / Noto Serif CJK 字体
- 嵌入字体声明为 `font/sfnt`（非 `application/x-font-ttf`，否则 Apple Books 拒收）
- Kindle 上复杂中文排版需充分测试

### 推荐方案

使用 **go-shiori/go-epub** 生成 EPUB 3.0，同时保持 EPUB 2.0 目录兼容性（最大化阅读器支持）。

验证流程：
```
1. 生成 EPUB 3.0
2. EPUBCheck 5.0.1 验证
3. Apple Books / Kindle 预览器测试
4. 在线验证器二次确认
5. 重点检查字体（PKG-026 是最常见拒收原因）
```

中文字体策略：
```
嵌入 Noto Serif CJK SC（正文）+ Noto Sans CJK SC（标题）
→ 声明为 font/sfnt
→ 提供 fallback 到系统字体
```

### 技术风险

| 风险 | 等级 | 缓解方案 |
|------|------|---------|
| 字体 PKG-026 导致平台拒收 | 高 | 严格使用 `font/sfnt` 媒体类型 |
| Kindle 不支持 EPUB | 中 | 通过 KDP 上传让 Amazon 自动转换 |
| go-epub 功能不完整 | 中 | 必要时 fork 自定义增强 |
| 中文排版在 Kindle 上异常 | 中 | 充分测试 + 简化 CSS |
| JavaScript 在多数阅读器不支持 | 低 | 交互功能做渐进增强 |

### 参考链接

- [W3C EPUB 3.3 规范](https://www.w3.org/TR/epub-33/)
- [go-shiori/go-epub](https://github.com/go-shiori/go-epub)
- [EPUBCheck 官方文档](https://www.w3.org/publishing/epubcheck/docs/messages/)
- [EPUBCheck 运行指南](https://www.w3.org/publishing/epubcheck/docs/running/)
- [Apple Books ePUB 支持](https://www.helpandmanual.com/help/hm_ref_formats_epub.html)
- [Kindle ePUB 兼容性](https://www.automateed.com/does-kindle-take-epub)
- [Kobo 中文字体配置](https://amigotechnotes.wordpress.com/2018/11/21/epub-chinese-text-and-chinese-fonts-in-kobo-ereaders/)

---

## 总结与优先级建议

| 方向 | 推荐方案 | 优先级 | 实施难度 |
|------|---------|--------|---------|
| Typst 后端 | CLI subprocess + Pandoc Typst writer | P1 | 中 |
| 增量编译 | 文件哈希 + 依赖图（三阶段） | P1 | 中 |
| 插件系统 | 类 mdBook stdin/stdout JSON 协议 | P2 | 中 |
| ePub 3 | go-shiori/go-epub + EPUBCheck 验证 | P2 | 低 |
| VS Code 插件 | CustomTextEditorProvider + WebView | P3 | 中 |
| WASM Playground | 阶段一 JS+markdown-wasm，阶段二 Go+TinyGo | P3 | 高 |

---

## 7. 业界最新最佳实践（2025-2026 补充调研）

### 7.1 构建工具增量编译标杆

#### Astro Content Layer API（Astro 5.0+）

Astro 5.0 引入 Content Layer API，实现类型安全的增量内容更新：
- 支持从任何源（CMS、API、本地文件）加载内容，只更新变化条目
- Markdown 编译速度提升 **5 倍**，MDX 编译提升 **2 倍**，内存减少 **25-50%**
- Astro 5.10.0 引入实验性实时内容集合，支持运行时数据获取

**对 mdpress 的启示**：设计增量更新的数据源接口，避免全量重编译；实现细粒度缓存追踪。

#### Turbopack（Next.js 15/16 默认打包器）

Turbopack 2025 年里程碑式进展：
- Next.js 16（2025年10月）成为默认打包器；16.1 文件系统缓存稳定
- 编译速度快 **2-5 倍**，Fast Refresh 快 **5-10 倍**，内存减少 **25-35%**
- vercel.com 大型应用：服务器启动快 **76.7%**，代码更新快 **96.3%**
- 核心创新——**Value Cells 架构**：每个值单元是细粒度执行单位，函数调用和依赖关系懒加载追踪，比传统 memoization 更细粒度
- 手动内存管理（无 GC 尖峰）+ 多线程并行 + 惰性编译

**对 mdpress 的启示**：实现细粒度执行追踪和缓存；支持惰性/按需编译；设计并行化编译架构。

#### Rspack 1.x（Webpack 兼容 Rust 构建工具）

- 相比 Webpack 性能提升 **5-10 倍**
- 实际案例（Mews）：启动从 3 分钟降至 **10 秒**，构建减少 **80%**
- 1000 个 React 组件基准：比 Webpack 快 **20 倍**，比 Vite 快 **10 倍**
- 增量策略：影响范围追踪 + Rebuilder 依赖追踪 + HMR 优化
- 已被 TikTok、Discord、Microsoft、Amazon 生产使用

#### Farm 1.0（Vite 兼容 Rust 构建工具）

- 模块级磁盘持久缓存（默认启用）
- HMR 时间 **10ms**，比 Vite 快 **6 倍**
- 懒加载编译：任意大小项目预览 **1 秒内**
- 热启动时间减少 **80%**

#### VitePress（文档构建工具标杆）

- 亚秒级冷启动 + 即时 HMR（<100ms）
- 按需编译：只编译正在服务的页面
- VitePress 1.6.4（2025年8月）

#### Docusaurus 3.8+

- `@docusaurus/faster` 框架：Rspack 持久缓存 + SWC 转译 + Lightning CSS
- 但大型站点仍有长构建时间和高内存问题，增量编译不是默认行为
- 相比 VitePress 有架构劣势

**参考链接**：
- [Astro Content Layer Deep Dive](https://astro.build/blog/content-layer-deep-dive/)
- [Turbopack Incremental Computation](https://nextjs.org/blog/turbopack-incremental-computation)
- [Next.js 16](https://nextjs.org/blog/next-16)
- [Rspack 官网](https://rspack.rs/)
- [Farm 官网](https://www.farmfe.org/)
- [VitePress](https://vitepress.dev/)
- [Docusaurus 3.8](https://docusaurus.io/blog/releases/3.8)

---

### 7.2 Go 生态最新工具和库

#### Go 1.24 WASM 重大突破（2025年2月）

- `go:wasmexport` 指令：将 Go 函数导出给 WASM 主机调用
- 支持 WASI reactor/library 模式（无需主程序即可运行）
- 放宽 `go:wasmimport` 参数类型限制
- Map 性能提升：大 Map 访问 **~30%**，预分配赋值 **~35%**，迭代 **~10-60%**
- 新 Swiss Tables Map 实现

**生产案例**：Dagger Cloud UI 完全从 React 迁移到 Go WASM，获得更一致的体验和更低的内存占用。

#### Go PDF 生态

| 库 | 特点 | 2025 状态 | 推荐度 |
|------|------|---------|-------|
| **Maroto v2** | 组件树架构、声明式构建、比 v1 快 2 倍+ | 活跃 | 最推荐 |
| **UniPDF** | 功能完整（修改+提取） | 商业维护 | 商用推荐 |
| **gopdf** | 简洁易用 | 活跃 | 轻量场景 |
| **go-typst** | CLI 包装，多格式输出，支持 Docker | 活跃 | Typst 集成首选 |

Maroto v2 主要改进：组件树架构（一切皆组件）、两阶段运行时（声明 + 生成分离）、单元测试友好。

#### Goldmark 扩展生态（2025-2026）

| 扩展 | 功能 | 来源 |
|------|------|------|
| goldmark-highlighting | 语法高亮（Chroma） | yuin |
| goldmark-frontmatter | YAML/TOML 前置元数据 | go.abhg.dev |
| goldmark-toc | 目录生成 | go.abhg.dev |
| goldmark-markdown | 渲染回 Markdown | teekennedy |

#### Templ 模板引擎

- HTML UI 语言，编译为高效 Go 代码
- IDE 自动完成和类型安全
- 支持 HTMX 实现响应式交互
- 比 html/template 更快，代码可读性高

#### gopls LSP 2025 进展

- 支持 Model Context Protocol (MCP)，与 AI 助手集成
- 完整 LSP 特性：导航、完成、诊断、重构

**参考链接**：
- [Go 1.24 WASM 支持](https://cloud.google.com/blog/products/application-development/go-1-24-expands-support-for-wasm)
- [Go wasmexport](https://go.dev/blog/wasmexport)
- [Dagger 用 Go WASM 替换 React](https://dagger.io/blog/replaced-react-with-go/)
- [Maroto v2](https://pkg.go.dev/github.com/johnfercher/maroto/v2)
- [Goldmark](https://github.com/yuin/goldmark)
- [goldmark-frontmatter](https://pkg.go.dev/go.abhg.dev/goldmark/frontmatter)
- [goldmark-toc](https://pkg.go.dev/go.abhg.dev/goldmark/toc)
- [Templ](https://templ.guide/)
- [gopls MCP 支持](https://go.dev/gopls/features/mcp)

---

### 7.3 插件系统设计模式最新演进

#### WASM 插件框架成为行业趋势

**Extism**（已达 1.0 稳定版）：
- 轻量级跨运行时 WASM 插件框架
- PDK 支持读取主机输入、返回数据、HTTP 调用
- 语言无关：任何编译到 WASM 的语言均可

**Zed 编辑器的 WASM 插件系统**：
- Rust 编写 → 编译为 wasm32-wasip1 → Wasmtime 加载
- 故障被沙箱隔离，支持无需重启的热重载
- WIT/wit_bindgen 处理类型转换，体验接近原生 Rust
- 单一 WASM 二进制跨平台发布

**Fermyon Spin 3.0**：
- 支持 WebAssembly Component Model 标准
- WIT（WebAssembly Interface Types）跨语言互操作
- OCI 兼容注册表发布组件

**Lapce 编辑器**：
- 基于 WASI 的插件系统
- 插件负责启动 LSP 并转发消息
- 计划推出 Plugin Server Protocol (PSP)

#### WASM vs 子进程插件对比（2025-2026 共识）

| 维度 | WASM | 子进程 |
|------|------|--------|
| 安全性 | 沙箱级隔离，能力白名单 | 操作系统级隔离 |
| 分发 | 单一跨平台二进制 | 需多架构编译 |
| 启动 | 毫秒级 | 秒级 |
| I/O | WASI 仍在演进 | 全面支持 |
| 可靠性 | 故障不波及主进程 | 进程隔离 |

行业动向：Helm 从子进程迁移到 WASM；Zed、Lapce 原生 WASM；WASI 3.0 预期 2026 年闭合 I/O 差距。

#### Deno 权限模型

- 默认沙箱：无文件系统、网络、环境变量访问
- 权限集合（Permission Sets）：deno.json 声明多套权限配置
- 2025 创新：云部署轻量 Linux microVM 执行不可信代码

#### Astro Integration 系统

- 灵感来自 Rollup/Vite 插件（低学习成本）
- `astro add` 命令自动化设置
- 支持框架集成、SSR 适配器、工具集成
- Starlight 文档工具支持自定义插件体系

#### Eleventy 3.0 插件

- 异步插件支持（3.0 新增）
- 唯一性控制 + 命名空间机制
- 错误隔离包装

#### WebAssembly Component Model 进展

- WASI Preview 3 已稳定，Component Model 广泛采用
- WIT 消除语言边界：Rust + Python + JS 可组合为统一组件
- WASI 0.3 预期 2025 发布，1.0 预期 2026-2027

#### 对 mdpress 的推荐架构

```
mdpress 核心（Go）
├─ 内置插件（Go interface 模式）— Phase 1
├─ WASM 扩展（Wazero/Extism 沙箱）— Phase 2
├─ 子进程插件（stdin/stdout JSON，兼容性）— Phase 1
└─ LSP/语言服务（外部工具）— Phase 3
```

**Wazero**（零依赖 Go WASM 运行时）：
- go-plugin WASM 版（v0.9.0）已升级到 wazero v1.7.0
- 插件编译：`GOOS=wasip1 GOARCH=wasm go build`
- 自动从 Protocol Buffers 生成插件 SDK
- 安全保证：panic 不影响主进程，默认无文件系统/网络访问

**参考链接**：
- [Extism](https://extism.org/)
- [Zed 扩展开发](https://zed.dev/blog/zed-decoded-extensions)
- [Fermyon Spin 3.0](https://www.fermyon.com/blog/introducing-spin-v3)
- [Lapce 插件系统](https://docs.lapce.dev/)
- [Astro 集成指南](https://docs.astro.build/en/guides/integrations-guide/)
- [Eleventy 3.0 插件](https://www.11ty.dev/docs/plugins/)
- [WebAssembly Component Model](https://component-model.bytecodealliance.org/)
- [Wazero](https://wazero.io/)
- [go-plugin WASM 版](https://github.com/knqyf263/go-plugin)
- [Deno 安全文档](https://docs.deno.com/runtime/fundamentals/security/)
