# 配置参考

本页说明 `book.yaml`。路径相对于配置文件所在位置。如果 `chapters` 为空，则改用 `SUMMARY.md`。

## 默认值

- `book.title`：`Untitled Book`
- `book.language`：`zh-CN`
- `style.theme`：`technical`
- `style.page_size`：`A4`
- `style.code_theme`：为空 —— 继承主题的代码配色（technical/elegant 为 `github`，minimal 为 `bw`）；显式设置的值优先
- `output.filename`：默认为 `output.pdf`，但除非你显式覆盖，否则构建时会使用书籍标题或目录名
- `output.toc`、`output.cover`、`output.header`、`output.footer`：启用
- `output.toc_max_depth`：`2`
- `output.generate_bookmarks`：`true`

## `book`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `title` | string | 必需。 |
| `subtitle` | string | 可选。 |
| `author` | string | 可选。 |
| `version` | string | 可选。 |
| `language` | string | BCP 47 标签，例如 `en-US` 或 `zh-CN`。 |
| `description` | string | 用于 EPUB 和 HTML 的元数据。 |
| `cover.image` | string | 封面图像路径。 |
| `cover.background` | string | 未使用封面图像时的背景颜色。浅色背景（包括 `white` 以及颜色名/`rgb()` 形式）会自动使用深色封面文字。 |

当 `cover.image` 和 `cover.background` 均未设置时，默认封面跟随主题：`technical` 为深海军蓝，`elegant` 为深暖棕色，`minimal` 为浅色配深色文字。

## `chapters`

`chapters` 是章节定义的有序列表：

```yaml
chapters:
  - title: Introduction
    file: README.md
  - title: Part One
    file: part-1.md
    sections:
      - title: Setup
        file: part-1/setup.md
```

- `title` 是读者在导航中看到的文字。
- `file` 是必需的，且必须指向一个 Markdown 文件。
- `sections` 在当前条目下嵌套更多章节。
- `SUMMARY.md` 用 Markdown 链接和缩进来表达同样的树状结构。

## `style`

- `theme`：内置主题（`technical`、`elegant`、`minimal`）、项目内 `themes/<name>.yaml`（或 `.yml`）定义的自定义主题名，或指向 YAML 主题文件的路径（如 `theme: mytheme.yaml`，相对于 `book.yaml`）。`themes/<name>.yaml` 文件也可以覆盖同名的内置主题。主题文件的字段说明见[内置主题](../themes/builtin-themes.md)。
- `page_size`：`A4`、`A5`、`Letter`、`Legal` 或 `B5`。
- `font_family`、`font_size`、`code_theme` 和 `line_height` 控制排版。`code_theme` 留空时继承主题的代码配色；站点/HTML 输出中代码高亮会自动匹配深色模式的对应样式（如 `github` → `github-dark`）。
- `margin` 设置上、下、左、右页边距，单位为毫米。
- `header` 和 `footer` 各接受 `left`/`center`/`right` 字符串，用于 PDF 页眉/页脚。支持的占位符：`{page}`、`{pages}`、`{title}`（旧式的 `{{.PageNum}}`、`{{.TotalPages}}`、`{{.Book.Title}}` 也接受）。默认页脚为居中页码，默认没有页眉；`output.header`/`output.footer` 布尔开关控制启停。
- `custom_css` 指向额外的 CSS 文件。

## `output`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `filename` | string | 生成输出的基础文件名。 |
| `formats` | list[string] | 支持的值：`pdf`、`html`、`site`、`epub`、`typst`。 |
| `toc` | bool | 包含目录。 |
| `toc_max_depth` | int | 范围 `1` 到 `6`。 |
| `cover` | bool | 包含封面页。 |
| `header` | bool | 启用页眉。 |
| `footer` | bool | 启用页脚。 |
| `pdf_timeout` | int | PDF 生成超时，单位秒。 |
| `watermark` | string | 水印文本。 |
| `watermark_opacity` | float | 不透明度，`0.0` 到 `1.0`。 |
| `margin_top` | string | PDF 或 Typst 的页边距覆盖，例如 `20mm`。 |
| `margin_bottom` | string | PDF 或 Typst 的页边距覆盖，例如 `20mm`。 |
| `margin_left` | string | PDF 或 Typst 的页边距覆盖，例如 `20mm`。 |
| `margin_right` | string | PDF 或 Typst 的页边距覆盖，例如 `20mm`。 |
| `generate_bookmarks` | bool | 生成 PDF 书签。 |
| `site_url` | string | 部署站点的公开基础 URL（例如 `https://user.github.io/repo`）。设置后，`site` 格式会生成符合规范的 `sitemap.xml`（绝对 `<loc>` 加 `<lastmod>`）；不设置则不生成 sitemap。 |
| `edit_base` | string | “编辑此页”链接的基础 URL（例如 `https://github.com/user/repo/edit/main/`）。设置后每个站点章节都会带一个编辑链接。 |
| `tagged_pdf` | bool | 生成可访问的带标签 PDF（默认 `true`）。设为 `false` 可显著减小文件体积，代价是丢失无障碍标签。 |

`style.margin` 与 `output.margin_*` 字段是相互独立的。用 `style.margin` 做通用布局设置，用 `output.margin_*` 做 PDF 或 Typst 的覆盖。

## `plugins`

插件是按声明顺序运行的外部可执行文件。

```yaml
plugins:
  - name: word-count
    path: ./examples/plugins/word-count
    config:
      warn_threshold: 500
```

- `name` 是插件标识符。
- `path` 指向可执行文件，相对于 `book.yaml`。
- `config` 以 JSON 数据形式传递给插件。

> **安全提示：** 插件是在你机器上运行的可执行文件，会在构建和预览期间被执行（包括探测阶段）。只对你信任的项目执行 build/serve。从 v0.7.12 起，远程来源默认拒绝运行插件，除非传入 `--allow-plugins`。详见 [插件概述](../plugins/overview.md)。

## 发现说明

当 `book.yaml` 不存在时，为兼容 GitBook 会支持 `book.json`，但本页聚焦于 `book.yaml`。
