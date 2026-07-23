# 输出格式

`mdpress build` 可以在一次运行中生成一种或多种格式。

## 格式

| 格式 | 你会得到什么 | 说明 |
| --- | --- | --- |
| `pdf` | `book.pdf` | 通过 Chromium 生成的默认 PDF 输出。遵循页面尺寸、页边距、目录、封面、页眉和页脚。 |
| `html` | `book.html` | 单个 HTML 文件，图片、样式和脚本都已内联。公式和 Mermaid 图在阅读时从 CDN 加载。 |
| `site` | `_book/` | 多页面静态站点，包含 `index.html`、章节页面、搜索和侧边栏导航。 |
| `epub` | `book.epub` | 面向电子阅读器的 EPUB 3 包。 |
| `typst` | `book-typst.pdf` | 备选 PDF 后端，需要 `PATH` 中存在 Typst。 |

## 构建多种格式

```bash
mdpress build --format pdf,html,epub
mdpress build --format all
```

`all` 会展开为 `pdf,html,site,epub`，不包含 `typst`：它依赖可选的 Typst CLI，且产物与 `pdf` 相同，否则在未安装 Typst 的机器上 `--format all` 必然失败。需要时请用 `--format typst` 显式指定。

## 输出路径

对于文件类格式，`--output` 设置基础路径，mdPress 会追加格式后缀：

- `book.pdf`
- `book.html`
- `book.epub`
- `book-typst.pdf`（typst 格式）

如果传入的是一个目录——已存在的目录，或任何以斜杠结尾的路径——文件会被写入该目录内，站点页面也会直接写入其中（就地写入，不会清理目录中已有的文件）。其他路径会被当作文件名基名处理：`--output release/manual.html` 会生成 `release/manual.html`、`release/manual.pdf`，站点则在 `release/manual_site/`。

当 `site` 是**唯一**请求的格式时，`--output <路径>` 就是站点目录本身——没有 `_site` 后缀，也不去猜这个路径是否已经存在：

```bash
mdpress build --format site --output ./dist   # -> dist/index.html
```

只有在 `site` 与文件类格式一起请求时才会出现 `_site` 后缀，因为那时两者共用同一个基名。

`site` 格式产出的是一个目录而非单个文件。默认写入项目目录下的 `_book/`——与
`mdpress serve` 使用的位置相同，也是各部署示例默认假设的目录。默认站点构建会先在
临时目录中完成，再原子地替换 `_book/`，因此改名或删除章节留下的旧页面会被清理；
如果目标目录非空且看起来不是生成产物（没有 `index.html`/`search-index.json`），
出于安全考虑会拒绝覆盖。

**多语言例外：**含 `LANGS.md` 的项目会在输出根目录下为每种语言构建一棵树——
`<输出目录>/en/`、`<输出目录>/zh/`——语言切换页位于 `<输出目录>/index.html`。
不传 `--output` 时，根目录就是 `_book/`。

远程 GitHub 构建（例如 `mdpress build https://github.com/user/repo`）在不传
`--output` 时会把产物写入当前工作目录：文件为 `./<书名>.pdf` 等，站点为 `./_book/`。

构建成功后，每种格式会打印一行 `✓ Generated <format> → <path>`（`--quiet` 下也会输出）。

## 站点选项

`site` 输出全程使用相对导航链接，因此可以部署在 GitHub Pages 项目站点
（`https://user.github.io/repo/`），甚至可以直接通过 `file://` 打开。
构建时会自动生成 `404.html` 页面。

两个 `output` 配置字段可扩展站点功能：

- `site_url`：部署站点的公开基础 URL。设置后会生成符合规范的 `sitemap.xml`
  （绝对 `<loc>` 加 `<lastmod>`）；不设置则不生成 sitemap。
- `edit_base`：形如 `https://github.com/user/repo/edit/main/` 的基础 URL。
  设置后每个章节页面都会带一个“编辑此页”链接。

## PDF 选项

默认情况下 PDF 页脚是居中页码，且没有页眉；可通过 `style.header`/`style.footer`
自定义（见[配置参考](../reference/configuration.md)）。新的 `output.tagged_pdf`
选项默认 `true`（生成可访问的带标签 PDF）；设为 `false` 可显著减小文件体积。
