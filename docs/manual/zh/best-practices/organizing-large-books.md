# 组织大型书籍

构建包含数百页的多章节书籍时，良好的结构组织可以简化维护工作，帮助 mdPress 高效构建。本指南介绍了管理大型文档和书籍的策略。

## 使用 SUMMARY.md 进行章节分组

对于复杂书籍，可以使用 `SUMMARY.md` 文件定义具有分层结构的章节：

```markdown
# Summary

- [Introduction](intro.md)
- [Part One: Fundamentals](part1.md)
  - [Chapter 1: Basics](ch01.md)
  - [Chapter 2: Installation](ch02.md)
- [Part Two: Advanced Topics](part2.md)
  - [Chapter 3: Architecture](ch03.md)
  - [Chapter 4: Optimization](ch04.md)
- [Appendix](appendix.md)
```

当 `book.yaml` 中省略了 `chapters` 配置时，mdPress 会自动读取此结构。每个缩进级别都创建嵌套分段，允许你将章节组织成逻辑部分。

## 替代方案：在 book.yaml 中明确定义章节

对于大型书籍，可以在 `book.yaml` 中明确定义具有嵌套分段的章节：

```yaml
book:
  title: "Advanced Guide"
  author: "Your Name"

chapters:
  - title: "Introduction"
    file: "intro.md"
  - title: "Part One: Fundamentals"
    file: "part1.md"
    sections:
      - title: "Chapter 1: Basics"
        file: "chapters/ch01.md"
      - title: "Chapter 2: Installation"
        file: "chapters/ch02.md"
  - title: "Part Two: Advanced Topics"
    file: "part2.md"
    sections:
      - title: "Chapter 3: Architecture"
        file: "chapters/ch03.md"
      - title: "Chapter 4: Optimization"
        file: "chapters/ch04.md"
```

部分标题的 `file` 字段成为该部分的介绍。分段可以嵌套多个级别。

## 文件命名规范

采用一致的命名约定便于导航：

- **顺序编号**：`ch01.md`、`ch02.md` 等
- **功能性命名**：`installation.md`、`troubleshooting.md`、`api-reference.md`
- **基于部分的分组**：`intro.md`、`part1-basics.md`、`part2-advanced.md`
- **描述性命名**：使用连字符，不用空格：`getting-started.md` 而不是 `Getting Started.md`

50章书籍的结构示例：

```
docs/
├── book.yaml
├── SUMMARY.md
├── intro.md
├── part1/
│   ├── ch01.md
│   ├── ch02.md
│   └── ch03.md
├── part2/
│   ├── ch04.md
│   ├── ch05.md
│   └── ch06.md
├── appendices/
│   ├── glossary.md
│   └── references.md
└── assets/
    ├── diagrams/
    └── screenshots/
```

## 在子目录中使用 README.md

通过创建包含 README.md 文件的子目录来组织大型书籍，这些文件作为部分介绍：

```
my-book/
├── book.yaml
├── SUMMARY.md
├── part1/
│   ├── README.md          # "Part One: Basics" 介绍
│   ├── ch01-intro.md
│   ├── ch02-setup.md
│   └── assets/
│       └── diagrams/
├── part2/
│   ├── README.md          # "Part Two: Advanced" 介绍
│   ├── ch03-architecture.md
│   ├── ch04-performance.md
│   └── assets/
│       └── diagrams/
└── appendices/
    ├── README.md          # "Appendices" 介绍
    ├── glossary.md
    └── references.md
```

在 `SUMMARY.md` 中：

```markdown
# Summary

- [Introduction](intro.md)
- [Part One: Basics](part1/README.md)
  - [Chapter 1: Introduction](part1/ch01-intro.md)
  - [Chapter 2: Setup](part1/ch02-setup.md)
- [Part Two: Advanced](part2/README.md)
  - [Chapter 3: Architecture](part2/ch03-architecture.md)
  - [Chapter 4: Performance](part2/ch04-performance.md)
- [Appendices](appendices/README.md)
  - [Glossary](appendices/glossary.md)
  - [References](appendices/references.md)
```

## 拆分长章节

避免单个章节超过 10,000 字（大约 40-50 页）。将其拆分成重点分段：

**之前（一个 50 页的章节）：**
```
api-guide.md (50 pages)
  - API Overview
  - Authentication
  - Endpoints
  - Error Handling
  - Rate Limiting
  - Examples
```

**之后（五个重点章节）：**
```
api-overview.md (10 pages)
api-authentication.md (10 pages)
api-endpoints.md (12 pages)
api-errors.md (8 pages)
api-examples.md (10 pages)
```

使用交叉引用（参见 [Authentication](api-authentication.md)）来连接相关章节。

## 资源目录中的图像管理

创建专用的 `assets/` 目录树，其镜像章节组织：

```
assets/
├── diagrams/
│   ├── architecture.png
│   ├── flow-chart.svg
│   └── deployment.png
├── screenshots/
│   ├── interface-main.png
│   ├── interface-settings.png
│   └── interface-profile.png
└── icons/
    ├── checkmark.svg
    ├── warning.svg
    └── info.svg
```

在章节中使用相对路径引用图像：

```markdown
# Installation

## System Requirements

![System Architecture](../assets/diagrams/architecture.png)

## Setup Steps

1. Download the installer
   ![Download Screen](../assets/screenshots/interface-main.png)
2. Configure options
3. Verify installation
```

对于大型书籍，按章节组织：

```
assets/
├── part1-basics/
│   ├── ch01-getting-started/
│   │   ├── step1.png
│   │   └── step2.png
│   └── ch02-installation/
│       ├── download.png
│       └── setup.png
└── part2-advanced/
    └── ch03-architecture/
        ├── diagram1.svg
        └── diagram2.svg
```

保持图像文件大小优化：
- 截图：每个不超过 500 KB
- 图表（SVG）：首选可扩展性
- 照片：总计不超过 2 MB

## 一致的标题层次结构

在章节内使用清晰的标题结构：

```markdown
# Chapter 1: Getting Started      (H1 - 章节标题)

## Section 1.1: Installation       (H2 - 主要部分)

### Step 1: Download              (H3 - 子部分)

#### Windows Installation         (H4 - 特定变体)

## Section 1.2: Configuration

### Basic Setup
### Advanced Options
```

规则：
- 以 H1（章节标题）开始每章
- 对主要部分使用 H2
- 对子部分保留 H3
- 避免跳过级别（不要从 H1 直接跳到 H3）
- 每章保持 H1 唯一（不重复）

在 `book.yaml` 中，设置目录深度以匹配你的结构：

```yaml
output:
  toc_max_depth: 3  # 在目录中包含 H1、H2 和 H3
```

## 章节间的交叉引用

使用相对文件路径在章节间链接（Markdown 语法）：

```markdown
有关身份验证的详细信息，请参见 [API Authentication](../api/authentication.md)。

有关分步说明，请参考 [Installation Guide](../getting-started/installation.md#step-1-download)。
```

对于 PDF 输出，mdPress 会自动将这些转换为 PDF 内部链接。对于 HTML 输出，确保路径是相对的和正确的。

交叉引用示例：

```markdown
# Advanced Configuration

如 [Basic Setup](../introduction/setup.md) 中所述，你应该首先安装
核心包。本章以这些基础为基础。

另请参见：
- [Performance Tuning](performance.md) 在此部分
- [Troubleshooting](../appendix/troubleshooting.md)
- [API Reference](../reference/api.md)
```

## 大型书籍的构建性能

### 并行处理

mdPress 自动使用多个 CPU 核心以并行方式解析章节。不需要配置，大型书籍的构建会自动受益于多核系统。

### 缓存策略

对于 50+ 章节的书籍，启用缓存以加速重建：

```bash
# 首次构建：完整编译
mdpress build --format pdf

# 后续构建：重用缓存的章节
mdpress build --format pdf

# 如果遇到过时的构建，强制完全重建
mdpress build --format pdf --no-cache
```

缓存基于文件哈希存储已编译的章节内容。对一个章节的更改不需要重新编译其他章节。

### 增量开发工作流

编写大型书籍时：

```bash
# 启动开发服务器进行实时预览
mdpress serve

# 此服务器监视文件更改并仅重建受影响的章节
# 开发期间不需要 --no-cache
```

## 示例：大型书籍结构

一个真实的 200 页技术书籍，包含 15 章：

```
technical-guide/
├── book.yaml
├── SUMMARY.md
├── intro.md
├── part1-foundations/
│   ├── README.md (部分介绍)
│   ├── ch01-overview.md
│   ├── ch02-installation.md
│   └── ch03-quick-start.md
├── part2-concepts/
│   ├── README.md (部分介绍)
│   ├── ch04-architecture.md
│   ├── ch05-data-model.md
│   ├── ch06-plugins.md
│   └── ch07-configuration.md
├── part3-advanced/
│   ├── README.md (部分介绍)
│   ├── ch08-performance.md
│   ├── ch09-scaling.md
│   ├── ch10-security.md
│   └── ch11-monitoring.md
├── part4-reference/
│   ├── README.md (部分介绍)
│   ├── ch12-api-reference.md
│   ├── ch13-cli-commands.md
│   └── ch14-configuration-reference.md
├── appendices/
│   ├── ch15-troubleshooting.md
│   ├── glossary.md
│   └── faq.md
└── assets/
    ├── diagrams/ (20 个 SVG 文件)
    ├── screenshots/ (30 个 PNG 文件)
    └── icons/ (10 个 SVG 文件)
```

使用镜像目录结构的 `SUMMARY.md` 和 `book.yaml` 中的显式引用，这个 200 页的书籍构建高效且组织清晰。

## 提示和技巧

- **使用一致的术语**：维护术语表（GLOSSARY.md）以记录技术术语
- **定期验证**：运行 `mdpress validate` 以尽早捕获破损的章节引用
- **经常预览**：在编写时使用 `mdpress serve` 以立即发现问题
- **保持章节重点**：每章目标是 5,000-10,000 字以便于阅读
- **记录你的结构**：在项目根目录添加 README，说明章节组织
