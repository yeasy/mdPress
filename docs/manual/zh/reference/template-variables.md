# 页眉页脚模板令牌

`style.header` / `style.footer` 中的字符串支持一组占位令牌，在 PDF 渲染时被替换；字符串中的其他内容按字面文本处理（并做 HTML 转义以保证安全）。

## 令牌参考

| 令牌 | 替换为 | 示例输出 |
| --- | --- | --- |
| `{page}` | 当前页码 | `47` |
| `{pages}` | 总页数 | `250` |
| `{title}` | `book.yaml` 中的书名 | `Python 高级指南` |
| `{author}` | 来自 `book.author` 的作者 | `张三` |

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "第 {page} 页 / 共 {pages} 页"
```

## 旧式令牌

旧版 mdPress 脚手架生成的配置使用 Go 模板风格的令牌，它们仍然被接受：

| 旧式令牌 | 等价于 |
| --- | --- |
| `{{.PageNum}}` | `{page}` |
| `{{.TotalPages}}` | `{pages}` |
| `{{.Book.Title}}` | `{title}` |
| `{{.Book.Author}}` | `{author}` |
| `{{.Chapter.Title}}` | 展开为空（Chrome 打印模板没有按章上下文） |

新配置请优先使用简短的 `{page}` / `{pages}` / `{title}` / `{author}` 形式。

## 令牌与文本混排

令牌可以与固定文本自由组合：

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "{page} / {pages}"
    right: "{title}"
```

## 常用组合

### 简单页码（与内置默认一致）

```yaml
style:
  footer:
    center: "{page}"
```

### 书名 + 页数

```yaml
style:
  header:
    left: "{title}"
  footer:
    center: "第 {page} 页 / 共 {pages} 页"
```

### 正式文档

```yaml
style:
  header:
    left: "{title}"
    right: "内部资料"
  footer:
    left: "© 2026 Acme Corp"
    center: "{page}"
```

## 限制

- 令牌仅在 PDF 页眉页脚中生效；其他格式不渲染它们。
- 输出为纯文本——页眉页脚字符串中不支持 Markdown 或 HTML 标记（标记会被转义）。
- 没有日期、版本、章节标题等令牌，需要时请写成字面文本，例如 `left: "© 2026 — v3.2.1"`。无法识别的 `{{...}}` 令牌会被从页面上删除（并给出告警）而不是打印出来，因此它周围的文本会变得孤零零的。
- 页眉页脚的字号与颜色是固定的（小号、灰色），只有内容可配置。需要更多留白时可调整 `output.margin_top` / `output.margin_bottom`。

配置细节参见[页眉与页脚](../themes/headers-footers.md)。
