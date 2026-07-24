# 页眉和页脚

页眉和页脚为 PDF 输出的每一页添加一致的页面信息。它们在 `book.yaml` 中配置，支持一小组占位符。

**注意：**页眉和页脚是仅限 PDF 的功能（Chromium 后端）。它们不会出现在 HTML、站点或 EPUB 输出中。

## 默认行为

开箱即用时，PDF 会得到：

- **页脚**：低调小字号的居中页码
- **页眉**：无

只有想改变这个默认样式时才需要配置。

## 基本配置

在 `book.yaml` 的 `style` 部分定义页眉和页脚。每个都有三个可选单元格 —— `left`、`center` 和 `right`：

```yaml
style:
  header:
    left: "{title}"
    center: ""
    right: "{page}"

  footer:
    center: "{page} / {pages}"
```

不需要的单元格可以省略。如果页眉/页脚的所有单元格都为空，则不会渲染。

## 关闭页眉和页脚

`output.header` 和 `output.footer` 布尔值是总开关：

```yaml
output:
  header: false   # 永不渲染页眉
  footer: false   # 永不渲染页脚（也会关闭默认页码）
```

两者默认均为 `true`。

## 封面页

封面是出血印刷到纸张边缘的整版画面，因此 mdPress 不会在封面上打印页眉和页脚。这无需任何配置：只要 `output.cover: true`（默认值），第一页就是干净的。

封面仍然算作第 1 页，所以目录页的页码显示为 `2`，目录里列出的页码也把封面计算在内。若希望页码从目录开始，请设置 `output.cover: false`。

## 支持的占位符

| 占位符 | 替换为 |
| --- | --- |
| `{page}` | 当前页码 |
| `{pages}` | 总页数 |
| `{title}` | `book.yaml` 中的书名 |
| `{author}` | 来自 `book.author` 的作者 |

旧版本脚手架配置使用的 Go 模板风格占位符也仍然接受：`{{.PageNum}}`（= `{page}`）、`{{.TotalPages}}`（= `{pages}`）、`{{.Book.Title}}`（= `{title}`）、`{{.Book.Author}}`（= `{author}`）。`{{.Chapter.Title}}` 出于兼容会被接受，但会展开为空 —— Chrome 打印模板没有逐章上下文。

其余内容一律作为字面文本处理（并进行 HTML 转义以保证安全），因此可以自由地把占位符和固定文本组合：

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "第 {page} 页，共 {pages} 页"
    right: "{title}"
```

## 示例

### 第 X 页，共 Y 页

```yaml
style:
  footer:
    center: "第 {page} 页，共 {pages} 页"
```

### 页眉书名、页脚页码

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "{page}"
```

### 草稿标记

```yaml
style:
  footer:
    center: "草稿 — {title} — {page}"
```

如果想要横跨页面的对角线草稿水印，请改用 `output.watermark`。

## 样式说明

页眉页脚文字使用固定的、刻意低调的样式（9px、柔和的灰色、系统字体栈）。可自定义的是内容，不是样式。如果页眉页脚显得拥挤，可通过 `output.margin_top` / `output.margin_bottom` 增大页边距（例如 `"20mm"`）。

## 故障排查

### 占位符在 PDF 中按字面显示

1. 检查拼写 —— 支持的占位符只有 `{page}`、`{pages}`、`{title}` 和 `{author}`（区分大小写）。
2. 确认你在生成 PDF 输出；页眉页脚不适用于其他格式。

### 没有出现页眉

默认页眉为空。只有当 `style.header` 的单元格设置了非空值（且 `output.header` 不为 `false`）时才会渲染页眉。

### 页眉/页脚与正文重叠

增大 PDF 页边距：

```yaml
output:
  margin_top: "20mm"
  margin_bottom: "20mm"
```

## 完整示例

```yaml
book:
  title: "mdPress Documentation"
  author: "mdPress Team"

chapters:
  - title: 前言
    file: chapters/01-introduction.md
  - title: 安装
    file: chapters/02-installation.md

style:
  theme: technical
  header:
    left: "{title}"
    right: "{page}"
  footer:
    center: "第 {page} 页，共 {pages} 页"

output:
  header: true
  footer: true
  margin_top: "18mm"
  margin_bottom: "18mm"
```

参见[自定义 CSS](./custom-css.md)和[内置主题](./builtin-themes.md)了解更多样式选项，以及[模板占位符参考](../reference/template-variables.md)查看完整占位符列表。
