# 快速开始

在几分钟内使用 mdPress 创建文档网站或书籍。

## 使用快速开始命令

最快的方式是使用内置快速开始模板：

```bash
mdpress quickstart my-book
cd my-book
mdpress serve --open
```

这会创建一个可用的项目，包含示例内容和配置。你的浏览器会自动打开 http://localhost:9000。

在首次构建之前，建议验证你的环境是否就绪：

```bash
mdpress doctor
```

这会检查所有必要的依赖是否已安装（如用于 PDF 输出的 Chrome/Chromium 等）并报告任何潜在问题。

## 手动创建项目

如果你偏好从零开始或想了解基础知识，这是最小化设置：

### 步骤 1：创建项目目录

```bash
mkdir my-documentation
cd my-documentation
```

### 步骤 2：创建 README.md

创建一个 `README.md` 文件作为介绍：

```markdown
# 我的文档

欢迎来到我的文档网站。这是读者首先看到的登陆页面。
```

### 步骤 3：创建 SUMMARY.md

`SUMMARY.md` 文件定义你的书籍结构：

```markdown
# 目录

- [介绍](README.md)
- [快速开始](chapters/getting-started.md)
  - [安装](chapters/installation.md)
  - [配置](chapters/configuration.md)
- [高级主题](chapters/advanced.md)
- [常见问题](chapters/faq.md)
```

### 步骤 4：创建章节文件

创建 SUMMARY.md 中引用的章节文件：

```bash
mkdir chapters
```

然后创建 `chapters/getting-started.md`：

```markdown
# 快速开始

这是主要的快速开始页面。
```

创建 `chapters/installation.md`：

```markdown
# 安装

安装说明放在这里。
```

继续创建其他章节。

### 步骤 5：验证项目结构

你的项目现在应该看起来像这样：

```
my-documentation/
├── README.md
├── SUMMARY.md
└── chapters/
    ├── getting-started.md
    ├── installation.md
    ├── configuration.md
    ├── advanced.md
    └── faq.md
```

## 运行开发服务器

使用实时重载启动开发服务器：

```bash
mdpress serve
```

默认情况下，你的网站在 http://localhost:9000 可用。当你修改文件时，浏览器会自动重新加载。

自动打开浏览器：

```bash
mdpress serve --open
```

使用自定义端口：

```bash
mdpress serve --port 3000
```

## 构建你的文档

当准备好部署时，构建一个静态网站：

```bash
mdpress build --format site
```

这会创建一个 `_book` 目录，包含准备好部署的完整 HTML 网站。

### 构建为 PDF

生成 PDF 版本：

```bash
mdpress build --format pdf
```

这会创建一个包含所有文档的单个 `book.pdf` 文件。

### 构建为 EPUB

电子书格式：

```bash
mdpress build --format epub
```

生成可在电子阅读器上使用的 `book.epub`。

### 构建多种格式

一次性构建所有格式：

```bash
mdpress build
```

这会生成 HTML 网站、PDF 和 EPUB（如果可用）。

## 理解命令

### mdpress serve

用于编写和测试的开发命令：

```bash
mdpress serve [OPTIONS]
```

可用选项：
- `--open` —— 自动打开浏览器
- `--port <PORT>` —— 使用自定义端口（默认：9000）
- `--watch` —— 监视文件变化并重建（默认：启用）
- `--no-cache` —— 在开发时禁用缓存

服务器监视所有源文件，当你保存变化时立即重建。

### mdpress build

用于生成输出的生产命令：

```bash
mdpress build [OPTIONS]
```

可用选项：
- `--format <FORMAT>` —— 输出格式：`site`、`pdf`、`epub`、`html`
- `--output <PATH>` —— 输出目录（默认：`_book`）
- `--minify` —— 最小化 HTML 和 CSS 以减小文件大小
- `--no-cache` —— 禁用缓存

## 后续步骤

现在你的文档已运行，你可以：

1. **配置你的项目** —— 为高级设置创建 `book.yaml`（参见配置参考）
2. **添加资源** —— 将图像和文件放在 `assets/` 目录中
3. **创建词汇表** —— 为术语定义添加 `GLOSSARY.md`
4. **自定义样式** —— 在配置中添加自定义 CSS 或选择主题

详见配置指南了解更多高级选项。
