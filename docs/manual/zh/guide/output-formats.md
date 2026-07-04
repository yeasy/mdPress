# 输出格式

`mdpress build` 可以在一次运行中生成一种或多种格式。

## 格式

| 格式 | 你会得到什么 | 说明 |
| --- | --- | --- |
| `pdf` | `book.pdf` | 通过 Chromium 生成的默认 PDF 输出。遵循页面尺寸、页边距、目录、封面、页眉和页脚。 |
| `html` | `book.html` | 单个自包含的 HTML 文件。 |
| `site` | `book_site/` | 多页面静态站点，包含 `index.html`、章节页面、搜索和侧边栏导航。 |
| `epub` | `book.epub` | 面向电子阅读器的 EPUB 3 包。 |
| `typst` | `book-typst.pdf` | 备选 PDF 后端，需要 `PATH` 中存在 Typst。 |

## 构建多种格式

```bash
mdpress build --format pdf,html,epub
mdpress build --format all
```

`all` 会展开为 `pdf,html,site,epub,typst`，即构建全部 5 种格式。

## 输出路径

`--output` 设置基础路径，mdPress 会追加格式后缀：

- `book.pdf`
- `book.html`
- `book.epub`
- `book_site/`
- `book-typst.pdf`（typst 格式）

如果传入的是一个目录，文件会被写入该目录内。
