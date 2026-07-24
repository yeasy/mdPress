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

- `book` —— 元数据、封面设置，以及站点品牌（`favicon`、`logo`、`copyright`）。
- `chapters` —— 阅读顺序，支持嵌套的 `sections`，另有 `section` 用于标注侧边栏分组。
- `style` —— 主题、字体、页面尺寸、页边距，以及页眉/页脚模板。
- `output` —— 生成的格式、PDF 设置，以及站点选项（`site_url`、`edit_base`、`footer_html`、`show_theme_badge`）。
- `markdown` —— 解析行为（`allow_html`）。
- `variables` —— 你自己的 `{{ key }}` 模板取值。
- `plugins` —— 外部可执行文件。

运行 `mdpress config show` 可以看到解析后的结果 —— 应用了全部默认值，还包括加载了哪个配置
文件以及主题来自哪里。全部键的说明见[配置参考](../reference/configuration.md)。

## 章节来源

- 如果存在 `chapters`，mdPress 使用其中列出的文件。
- 如果 `chapters` 为空或省略，则改为解析 `SUMMARY.md`。
- 路径相对于 `book.yaml`。

## 说明

- 存在 `LANGS.md` 时启用多语言构建。
- 会自动检测 `GLOSSARY.md`。
- `static/` 目录会被原样复制到站点根目录 —— `CNAME`、`.nojekyll` 之类的文件应该放在那里。
- `mdpress init` 可以从现有的 Markdown 文件生成初始的 `book.yaml`。
- `mdpress validate --strict` 会因为无法识别的配置键之类的告警而失败，因此可用作 CI 门禁。
