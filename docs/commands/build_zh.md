# `mdpress build`

[English](build.md)

## 作用

从本地目录或 GitHub 仓库构建发布产物。支持 `pdf`、`html`、`site`、`epub`、`typst` 五种输出格式，也支持一次构建多种格式。

## 语法

```bash
mdpress build [source] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[source]` | 否 | 输入源。可以省略、可以是本地目录，也可以是 GitHub 仓库 URL。省略时默认使用当前目录。 |

## 命令参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-f, --format <list>` | 配置值或 `pdf` | 输出格式，逗号分隔（如 `pdf,html,epub`）或 `all` 表示所有格式。 |
| `--branch <name>` | 仓库默认分支 | GitHub 仓库分支名，仅对远程仓库输入生效。 |
| `--subdir <path>` | 仓库根目录 | 指定仓库内的子目录，仅对远程仓库输入生效。 |
| `-o, --output <path>` | `output.filename` / site 为 `_book/` | 输出文件路径、输出目录或文件名基名。 |
| `--summary <path>` | 自动检测 | 显式指定 `SUMMARY.md` 文件路径。会覆盖 `book.yaml` 中的章节定义或自动发现结果。 |
| `--allow-plugins` | 关闭 | 执行远程项目 `book.yaml` 中声明的插件（任意代码）。本地项目始终执行插件。 |
| `--config <path>` | `book.yaml` | 本地构建时使用的配置文件路径。 |
| `-v, --verbose` | 关闭 | 输出详细日志和逐条警告。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |
| `--cache-dir <path>` | 系统默认 | 自定义构建缓存目录。 |
| `--no-cache` | 关闭 | 禁用所有构建缓存。 |

## 用法说明

### 输入解析

`build` 的配置加载优先级如下：

1. `book.yaml`
2. `book.json`（GitBook 兼容）
3. `SUMMARY.md`
4. 自动扫描 `.md` 文件

如果你没有提供 `[source]`，命令默认从当前目录工作。

如果当前目录是一个大型代码仓库，而不是单独的文档目录，建议不要直接在仓库根目录依赖自动发现。更稳妥的方式是：

```bash
mdpress build ./docs --format html
mdpress build --config ./docs/book.yaml ./docs --format pdf,html
```

### `--format` 的行为

- 传入 `--format` 时，命令行值会覆盖配置文件中的 `output.formats`
- 不传 `--format` 时，优先使用 `output.formats`
- 如果两者都没有设置，默认构建 `pdf`
- `--format all` 展开为 `pdf,html,site,epub`。`typst` 被有意排除：它依赖可选的 Typst CLI，且产物与 `pdf` 相同，需要时请用 `--format typst` 显式指定。

### `--output` 的行为

不传 `--output` 时，文件类格式会写入项目目录（文件名取 `output.filename` 或书名），`site` 会写入项目目录下的 `_book/` —— 与 `mdpress serve` 使用相同的位置。多语言项目（含 `LANGS.md`）仍保留按语言划分的 `<lang>_site/` 目录。

传 `--output` 时有两种用法：

1. 传一个目录 —— 已存在的目录，或任何以斜杠结尾的路径

```bash
mdpress build --output ./dist
mdpress build --output ./dist/
```

文件类格式会写入该目录（例如 `./dist/<书名>.pdf`、`./dist/<书名>.html`）。`site` 的页面会直接写入该目录（就地写入，目录中已有的旧文件不会被清理）。

2. 传一个文件路径或文件名基名

```bash
mdpress build --format pdf,html,site --output ./release/manual.pdf
```

无法解析为目录的路径会被当作“基准路径”处理：

- `pdf` 会得到 `./release/manual.pdf`
- `html` 会得到 `./release/manual.html`
- `site` 会得到 `./release/manual_site/`

### 站点输出的安全机制

站点有两条写入路径，只有其中一条会清理旧页面：

- **替换式（默认的 `_book/`，以及不与其他格式共用目录的 `--output`）**：站点先构建到临时目录，再原子地替换到目标位置，因此目标目录不会出现半成品，改名或删除章节留下的旧页面会被清理。作为保护措施，如果目标目录非空且看起来不是生成产物（没有 `index.html`/`search-index.json`），会拒绝覆盖。
- **就地式（`--output <dir>` 且其他格式也写入同一目录）**：页面直接写入该目录，因为其他格式构建器会并发写同一目录。**不会做任何清理**——你已删除或改名的章节所对应的页面仍留在磁盘上，并会随其他产物一起被部署。如果这一点重要，请在构建前先删除该目录。

如果默认的 `_book/` 构建旁边还留有旧版 mdPress 生成的 `<name>_site/` 目录，会输出一条提示日志。

### 构建结果汇总

每次构建成功后，CLI 会为每种格式打印一行结果，路径为绝对路径，例如：

```
  ✓ Generated pdf   → /home/you/my-book/my-book.pdf
  ✓ Generated site  → /home/you/my-book/_book/index.html
```

`site` 那一行指向生成的 `index.html`，而不是目录本身。

这些行属于构建结果而非进度输出，即使指定 `--quiet` 也会打印。

## 示例

```bash
mdpress build
mdpress build --format html
mdpress build --format pdf,html,epub
mdpress build --format all --output ./dist
mdpress build --format site --output ./dist/book
mdpress build /path/to/book --format html
mdpress build https://github.com/yeasy/agentic_ai_guide
mdpress build https://github.com/yeasy/agentic_ai_guide --branch main --subdir docs
mdpress build --config ./configs/book.yaml --verbose
```

## 构建结果

| 格式 | 结果 |
| --- | --- |
| `pdf` | 单个 PDF 文件 |
| `html` | 自包含单页 HTML 文件 |
| `site` | 多页静态站点目录 |
| `epub` | 单个 ePub 文件 |
| `typst` | 通过 Typst CLI 生成的 PDF 文件（无需 Chromium） |

## 注意事项

- 生成 PDF 依赖 Chrome 或 Chromium；如果系统没有浏览器，PDF 构建会失败。
- Typst 输出依赖 Typst CLI；如果未安装，`--format typst` 构建会失败。
- 构建过程中会检查章节标题编号、Markdown 链接和 Mermaid 诊断，但很多问题是警告，不一定直接终止构建。
- 当项目根目录存在 `LANGS.md` 时，`build` 会按语言分别构建，并额外生成语言入口页。
- 对远程 GitHub 仓库输入，当前实现优先使用远程仓库中的 `book.yaml`。本地传入的 `--config` 不会覆盖远程项目内的配置文件路径。
- 对远程 GitHub 仓库输入，如果没有传 `--output`，产物会写入当前工作目录（文件为 `./<书名>.pdf` 等，站点为 `./_book/`）。
- 对远程 GitHub 仓库输入，除非传入 `--allow-plugins`，否则不会执行远程 `book.yaml` 中声明的插件（插件是任意可执行程序）。
- 如果同时传 `--quiet` 和 `--verbose`，当前实现以 `--quiet` 为准。

## 常见问题

### 1. 为什么生成的章节比预期多？

如果没有 `book.yaml` 或 `SUMMARY.md`，`build` 会递归扫描 Markdown。对代码仓库来说，这通常会把 `README.md`、`docs/`、`examples/`、测试数据等一起带进来。

解决方式：

- 指定更精确的目录，例如 `mdpress build ./docs`
- 或者补 `book.yaml` / `SUMMARY.md`

### 2. 为什么站点预览和 PDF 排版不完全一样？

这是预期行为。`site` / `serve` 是网站式阅读布局，`pdf` 和单页 `html` 更接近文档排版。判断最终排版质量时，应以 `build --format pdf` 或 `build --format html` 为准。
