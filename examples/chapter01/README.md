# 第一章 快速开始 / Chapter 1: Quick Start

本章将帮助你快速上手 mdpress，从安装到生成第一本 PDF 图书。

*This chapter helps you get started with mdpress, from installation to generating your first PDF book.*

## 1.1 安装 / Installation

### 使用 Go 安装 / Install with Go

确保你已安装 Go 1.21 或更高版本，然后运行：

*Ensure you have Go 1.21 or later installed, then run:*

```bash
go install github.com/yeasy/mdpress@latest
```

### 从源码编译 / Build from Source

```bash
git clone https://github.com/yeasy/mdpress.git
cd mdpress
make build
```

### 前置依赖 / Prerequisites

mdpress 使用 Chromium 进行 PDF 渲染，请确保系统中已安装：

*mdpress uses Chromium for PDF rendering. Ensure the following is installed:*

- **macOS**: `brew install chromium` 或安装 Google Chrome / or install Google Chrome
- **Ubuntu/Debian**: `apt install chromium-browser`
- **Windows**: 安装 Google Chrome / Install Google Chrome

## 1.2 初始化项目 / Initialize a Project

在你的图书目录下运行 / Run in your book directory:

```bash
mdpress init
```

这将创建以下文件结构 / This creates the following file structure:

```
my-book/
├── book.yaml          # 配置文件 / Config file
├── preface.md         # 前言 / Preface
└── chapter01/
    └── README.md      # 第一章 / Chapter 1
```

## 1.3 编写内容 / Write Content

使用你喜欢的编辑器编辑 Markdown 文件。mdpress 完整支持 GitHub Flavored Markdown（GFM），包括：

*Edit Markdown files with your favorite editor. mdpress fully supports GFM, including:*

### 表格 / Tables

| 功能 / Feature | 支持 / Supported | 说明 / Description |
|------|---------|------|
| 表格 / Tables | ✅ | GFM 表格语法 / GFM table syntax |
| 任务列表 / Task lists | ✅ | `- [x]` 语法 / `- [x]` syntax |
| 脚注 / Footnotes | ✅ | `[^1]` 语法 / `[^1]` syntax |
| 代码高亮 / Code highlighting | ✅ | 多语言支持 / Multi-language support |

### 代码高亮 / Code Highlighting

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, mdpress!")
}
```

```python
def hello():
    """mdpress 支持多种编程语言的语法高亮 / mdpress supports syntax highlighting for many languages"""
    print("Hello, mdpress!")
```

### 任务列表 / Task Lists

- [x] 安装 mdpress / Install mdpress
- [x] 创建项目结构 / Create project structure
- [ ] 编写内容 / Write content
- [ ] 生成 PDF / Generate PDF

## 1.4 生成 PDF / Generate PDF

一切就绪后，运行构建命令 / When everything is ready, run:

```bash
mdpress build
```

或指定配置文件 / Or specify a config file:

```bash
mdpress build --config path/to/book.yaml
```

生成的 PDF 文件将保存到配置文件中 `output.filename` 指定的路径。

*The generated PDF will be saved to the path specified in `output.filename` in the config file.*

## 1.5 小结 / Summary

恭喜！你已经成功生成了第一本 PDF 图书。接下来，请阅读第二章了解更多进阶功能。

*Congratulations! You have successfully generated your first PDF book. Read Chapter 2 for more advanced features.*
