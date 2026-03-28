# 项目结构

理解 mdPress 项目布局有助于你有效地组织内容并使用所有可用功能。

## 目录结构

典型的 mdPress 项目看起来像这样：

```
my-documentation/
├── book.yaml                # 配置文件（可选）
├── README.md                # 登陆页 / 介绍
├── SUMMARY.md               # 目录结构
├── GLOSSARY.md              # 术语定义（可选）
├── LANGS.md                 # 多语言支持（可选）
├── assets/                  # 图像、图表、文件
│   ├── images/
│   │   ├── logo.png
│   │   └── diagram.svg
│   ├── diagrams/
│   └── downloads/
└── chapters/                # 按目录组织的内容
    ├── getting-started/
    │   ├── installation.md
    │   └── configuration.md
    ├── guide/
    │   ├── basic-usage.md
    │   └── advanced-tips.md
    └── reference/
        └── api.md
```

## 核心文件

### README.md

用户首次访问你的文档时显示的登陆页：

```markdown
# 项目标题

介绍文本、概览和导航指南。

## 关键部分

- [快速开始](chapters/getting-started.md)
- [用户指南](chapters/guide/index.md)
```

mdPress 在扫描书籍内容时会跳过顶层的 README.md（将其视为项目文档，而非书籍章节）。如需将其包含在书籍中，请在 SUMMARY.md 中显式列出。

### SUMMARY.md

定义目录和章节结构。这是最重要的文件：

```markdown
# 目录

- [介绍](README.md)
- [第1章](chapter1.md)
- [第2章](chapter2.md)
  - [部分 2.1](chapter2/section1.md)
  - [部分 2.2](chapter2/section2.md)
- [附录](appendix.md)
```

格式规则：
- 以 `-` 开头的每一行是一个章节条目
- 缩进创建嵌套（深度无限制）
- 链接格式：`[显示文本](file_path.md)`
- 路径相对于项目根目录

## 配置文件（可选）

为高级设置创建 `book.yaml`：

```yaml
book:
  title: 我的文档
  author: 作者名字
  description: 简短描述

style:
  theme: technical
```

mdPress 也接受 `book.json`（用于 GitBook 兼容）。如果没有配置文件，mdPress 使用合理的默认值（零配置模式）。

## 资源目录

在 `assets/` 中存储媒体和可下载文件：

```
assets/
├── images/          # PNG、JPG、SVG、GIF
├── diagrams/        # PlantUML 或 Mermaid
└── downloads/       # 可附加的文件
```

在 Markdown 中引用资源：

```markdown
![我的图像](assets/images/screenshot.png)

[下载 PDF](assets/downloads/guide.pdf)
```

mdPress 自动处理图像并为网页输出优化它们。

## 可选特殊文件

### GLOSSARY.md

定义整个文档中使用的术语：

```markdown
# 词汇表

## API
应用程序编程接口。定义软件组件如何通信。

## CLI
命令行界面。基于文本的用户界面。

## REST
表述性状态转移。API 架构风格。
```

当 GLOSSARY.md 中的术语出现在内容中时，它们会链接到定义。

### LANGS.md

支持多种语言（用于文档翻译）：

```markdown
# 语言

- [English](.)
- [Español](es/)
- [Français](fr/)
```

这创建了一个语言切换器。每种语言都有自己的子目录：

```
documentation/
├── README.md              # 英文
├── SUMMARY.md
├── es/
│   ├── README.md          # 西班牙文
│   └── SUMMARY.md
└── fr/
    ├── README.md          # 法文
    └── SUMMARY.md
```

## 章节组织

在嵌套目录中组织章节以清晰表达：

```
chapters/
├── 01-getting-started/
│   ├── index.md
│   ├── installation.md
│   └── first-project.md
├── 02-user-guide/
│   ├── index.md
│   ├── basic-concepts.md
│   ├── workflows/
│   │   ├── workflow-a.md
│   │   └── workflow-b.md
│   └── best-practices.md
└── 03-reference/
    ├── api.md
    └── cli.md
```

在 SUMMARY.md 中引用：

```markdown
- [快速开始](chapters/01-getting-started/index.md)
  - [安装](chapters/01-getting-started/installation.md)
  - [第一个项目](chapters/01-getting-started/first-project.md)
- [用户指南](chapters/02-user-guide/index.md)
  - [基本概念](chapters/02-user-guide/basic-concepts.md)
  - [工作流](chapters/02-user-guide/workflows/index.md)
    - [工作流 A](chapters/02-user-guide/workflows/workflow-a.md)
```

## 零配置自动发现

mdPress 可以在没有任何配置的情况下运行：

1. 将 Markdown 文件放在你的项目中
2. 使用你的结构创建 SUMMARY.md
3. 运行 `mdpress serve` 或 `mdpress build`

mdPress 自动：
- 将 README.md 作为登陆页
- 解析 SUMMARY.md 结构
- 在标准位置发现资源
- 应用默认样式和格式化

你仅在自定义默认行为时才需要配置文件。

## 最佳实践

### 文件命名

使用清晰、小写的文件名和连字符：

```
好的：  installation.md, getting-started.md, api-reference.md
不好的：Installation.md, Getting Started.md, API_REFERENCE.MD
```

### 目录深度

保持嵌套合理（通常 3 层）：

```
✓ chapters/section/subsection/page.md
✗ chapters/a/b/c/d/e/f/page.md （太深）
```

### 资源组织

分组相似的资源：

```
assets/
├── images/           # 所有图像
├── icons/            # UI 图标
├── screenshots/      # 应用截图
└── downloads/        # 可附加的文件
```

### 交叉引用

在章节之间链接：

```markdown
另请参阅：[安装指南](installation.md)

或使用完整路径：[配置](../getting-started/configuration.md)
```

### 部分锚点

链接到特定标题：

```markdown
[跳至 API 参考](#api-reference)

或者：[跳至另一个章节](#installation)
```
