# 配置

mdPress 首先读取项目根目录下的 `book.yaml`。如果 `book.yaml` 不存在，发现流程会依次回退到 `book.json`（兼容 GitBook）、`SUMMARY.md`，最后是自动发现的 Markdown 文件。

## 最小示例

```yaml
book:
  title: My Book
  author: Jane Doe
  language: en-US

chapters:
  - title: Introduction
    file: README.md
  - title: Getting Started
    file: chapters/getting-started.md

style:
  theme: technical
  page_size: A4
  custom_css: custom.css

output:
  filename: my-book.pdf
  formats: [pdf, html]
```

## 可配置的内容

- `book` —— 元数据与封面设置。
- `chapters` —— 阅读顺序，支持嵌套的 `sections`。
- `style` —— 主题、字体、页面尺寸、页边距，以及页眉/页脚模板。
- `output` —— 生成的格式与 PDF 设置。
- `plugins` —— 外部可执行文件。

## 章节来源

- 如果存在 `chapters`，mdPress 使用其中列出的文件。
- 如果 `chapters` 为空或省略，则改为解析 `SUMMARY.md`。
- 路径相对于 `book.yaml`。

## 说明

- 存在 `LANGS.md` 时启用多语言构建。
- 会自动检测 `GLOSSARY.md`。
- `mdpress init` 可以从现有的 Markdown 文件生成初始的 `book.yaml`。
