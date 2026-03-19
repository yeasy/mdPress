# `mdpress build`

[English](build.md)

## 作用

从本地目录或 GitHub 仓库构建发布产物。支持 `pdf`、`html`、`site`、`epub` 四种输出格式，也支持一次构建多种格式。

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
| `--format <list>` | 配置值或 `pdf` | 输出格式，逗号分隔，例如 `pdf,html,epub`。 |
| `--branch <name>` | 仓库默认分支 | GitHub 仓库分支名，仅对远程仓库输入生效。 |
| `--subdir <path>` | 仓库根目录 | 指定仓库内的子目录，仅对远程仓库输入生效。 |
| `--output <path>` | `output.filename` | 输出文件路径、输出目录或文件名前缀。 |
| `--summary <path>` | 自动检测 | 显式指定 `SUMMARY.md` 文件路径。会覆盖 `book.yaml` 中的章节定义或自动发现结果。 |
| `--config <path>` | `book.yaml` | 本地构建时使用的配置文件路径。 |
| `-v, --verbose` | 关闭 | 输出详细日志和逐条警告。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

## 用法说明

### 输入解析

`build` 的配置加载优先级如下：

1. `book.yaml`
2. `SUMMARY.md`
3. 自动扫描 `.md` 文件

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

### `--output` 的行为

`--output` 有三种常见用法：

1. 传一个现有目录

```bash
mdpress build --output ./dist
```

结果会写成类似 `./dist/output.pdf`、`./dist/output.html`。

2. 传一个文件名前缀

```bash
mdpress build --format pdf,html --output ./dist/book
```

结果会写成：

- `./dist/book.pdf`
- `./dist/book.html`
- `./dist/book_site/`，如果同时构建 `site`

3. 传一个带扩展名的路径

```bash
mdpress build --format pdf --output ./release/manual.pdf
```

当前实现会把它当作“基准路径”处理。也就是说：

- `pdf` 会得到 `./release/manual.pdf`
- `html` 会得到 `./release/manual.html`
- `site` 会得到 `./release/manual_site/`

## 示例

```bash
mdpress build
mdpress build --format html
mdpress build --format pdf,html,epub
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

## 注意事项

- 生成 PDF 依赖 Chrome 或 Chromium；如果系统没有浏览器，PDF 构建会失败。
- 构建过程中会检查章节标题编号、Markdown 链接和 Mermaid 诊断，但很多问题是警告，不一定直接终止构建。
- 当项目根目录存在 `LANGS.md` 时，`build` 会按语言分别构建，并额外生成语言入口页。
- 对远程 GitHub 仓库输入，当前实现优先使用远程仓库中的 `book.yaml`。本地传入的 `--config` 不会覆盖远程项目内的配置文件路径。
- 如果同时传 `--quiet` 和 `--verbose`，当前实现以 `--quiet` 为准。

## 常见问题

### 1. 为什么生成的章节比预期多？

如果没有 `book.yaml` 或 `SUMMARY.md`，`build` 会递归扫描 Markdown。对代码仓库来说，这通常会把 `README.md`、`docs/`、`examples/`、测试数据等一起带进来。

解决方式：

- 指定更精确的目录，例如 `mdpress build ./docs`
- 或者补 `book.yaml` / `SUMMARY.md`

### 2. 为什么站点预览和 PDF 排版不完全一样？

这是预期行为。`site` / `serve` 是网站式阅读布局，`pdf` 和单页 `html` 更接近文档排版。判断最终排版质量时，应以 `build --format pdf` 或 `build --format html` 为准。
