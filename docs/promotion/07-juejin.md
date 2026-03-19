# 掘金

## 标题

用 Go 写了一个 Markdown 书籍发布工具 mdpress：单二进制、零依赖、多格式输出

## 正文

### 背景

作为一个长期维护 Markdown 技术书籍的开发者，我在发布工具上踩过不少坑。GitBook 开源版停更后，社区虽然有 HonKit 接棒，但 Node.js 的依赖链管理一直让人头疼。mdBook 是 Rust 生态的优秀工具，但不支持 PDF。Pandoc 功能强大，但要生成高质量 PDF 需要安装数 GB 的 LaTeX 发行版。

我的理想工具应该是：**一个二进制文件，读进 Markdown，输出 PDF + HTML + ePub，不需要额外安装任何运行时。**

所以我用 Go 写了 [mdpress](https://github.com/yeasy/mdpress)。

### 核心能力

mdpress 是一个命令行工具，支持四种输出格式：

| 格式 | 说明 |
|---|---|
| PDF | 专业排版，支持封面、目录、页眉页脚、页码 |
| HTML | 单文件自包含 HTML 文档 |
| Site | 多页静态网站，三栏 GitBook 风格布局 |
| ePub | 标准 ePub 格式，适配电子阅读器 |

### 快速上手

```bash
# 安装
brew tap yeasy/tap && brew install mdpress

# 创建示例项目并预览
mdpress quickstart mybook
mdpress serve mybook

# 构建所有格式
mdpress build mybook --format pdf,html,site,epub
```

### 零配置设计

mdpress 采用三级配置发现机制：

1. **book.yaml** — 完整的配置文件，控制章节、元数据、主题、输出选项
2. **SUMMARY.md** — 兼容 GitBook 格式，自动解析章节结构
3. **自动扫描** — 没有任何配置文件？直接扫描文件夹中的 .md 文件

这意味着你可以直接 `mdpress build ./my-folder --format html`，无需编写任何配置。

### 架构设计

整体采用管道式架构：

```
源加载 → 配置发现 → Markdown 解析 → 后处理 → 组装 → 输出
```

**核心依赖库：**

| 库 | 用途 |
|---|---|
| [yuin/goldmark](https://github.com/yuin/goldmark) | Markdown 解析，支持 GFM 扩展、脚注 |
| [chromedp/chromedp](https://github.com/chromedp/chromedp) | 无头 Chrome，用于 PDF 渲染 |
| [spf13/cobra](https://github.com/spf13/cobra) | CLI 框架 |
| [alecthomas/chroma](https://github.com/alecthomas/chroma) | 代码语法高亮，支持 100+ 语言 |
| [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) | 文件监听，用于 serve 的热重载 |

**模块分层：**

```
CLI 层（cmd/）
  ├── BuildOrchestrator  — 构建流程编排
  └── ChapterPipeline    — 章节处理管道

核心处理层（internal/）
  ├── config     — 配置管理与发现
  ├── markdown   — Goldmark 解析引擎
  ├── renderer   — HTML 组装（封面 + 目录 + 章节 + CSS）
  ├── source     — 源抽象（本地文件系统、GitHub）
  └── ...

输出层（internal/output/）
  ├── PDFOutput   — chromedp PDF 生成
  ├── HTMLOutput  — 单页 HTML
  ├── SiteOutput  — 多页静态网站
  └── EpubOutput  — ePub 打包

基础设施（pkg/utils/）
  └── 文件 I/O、图片处理、路径解析
```

一个我比较满意的设计是 `FormatBuilderRegistry` 模式。所有输出格式实现统一的 `OutputFormat` 接口并注册到 Registry，新增格式不需要修改核心管道代码。

### 值得一提的功能

**GitBook 迁移兼容** — 如果你有现成的 `SUMMARY.md` 和 `GLOSSARY.md`，mdpress 直接读取，不需要做任何转换。

**`mdpress serve` 实时预览** — 基于 fsnotify 文件监听 + WebSocket 推送，编辑 Markdown 后浏览器自动刷新。输出的静态网站支持侧边栏导航、暗色模式、代码一键复制。

**GitHub 源支持** — 直接从 GitHub 仓库构建，无需 clone：

```bash
mdpress build https://github.com/user/repo --format pdf
```

支持 `GITHUB_TOKEN` 环境变量访问私有仓库。

**交叉引用与术语表** — 定义图表 ID，用 `{{ref:id}}` 引用并自动编号。在 `GLOSSARY.md` 中定义术语，正文中自动高亮并添加 tooltip。

### 与竞品对比

| 特性 | mdpress | GitBook (OSS) | mdBook | Pandoc |
|---|---|---|---|---|
| PDF 输出 | ✅ | ✅ | ❌ | ✅ |
| HTML 输出 | ✅ | ✅ | ✅ | ✅ |
| ePub 输出 | ✅ | ✅ | ❌ | ✅ |
| 静态网站 | ✅ | ✅ | ✅ | ❌ |
| 运行时依赖 | 无 | Node.js | 无 | LaTeX (PDF) |
| SUMMARY.md | ✅ | ✅ | 类似 | ❌ |
| 零配置 | ✅ | ❌ | ❌ | ❌ |
| 实时预览 | ✅ | ✅ | ✅ | ❌ |
| CJK 支持 | ✅ | ✅ | ✅ | 需配置 |

### 后续规划

- **v0.3.0**（2026 年 8 月）— 插件系统、LaTeX 数学公式、Mermaid 图表原生支持、自定义字体嵌入
- **v0.4.0**（2026 年 11 月）— Typst 后端（替代 Chromium，实现真正零外部依赖 PDF）、增量构建、并行编译
- **v1.0.0**（2027 年 Q1）— API 稳定、官方主题和插件注册中心

### 试一试

```bash
brew tap yeasy/tap && brew install mdpress
mdpress quickstart mybook
mdpress serve mybook
```

MIT 开源协议，欢迎 Star、Issue 和 PR。

GitHub: [https://github.com/yeasy/mdpress](https://github.com/yeasy/mdpress)
