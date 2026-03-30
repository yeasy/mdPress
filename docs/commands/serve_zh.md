# `mdpress serve`

[English](serve.md)

## 作用

构建本地预览站点并启动 HTTP 服务。适合在写作过程中持续预览内容，文件变化后会自动重建并刷新页面。

## 语法

```bash
mdpress serve [source] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[source]` | 否 | 输入源。可以省略、可以是本地目录，也可以是 GitHub 仓库 URL。省略时默认使用当前目录。 |

## 命令参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--host <addr>` | `127.0.0.1` | HTTP 监听地址。默认只接受本机访问；如果要暴露给局域网或绑定到特定网卡地址，需要显式传入，例如 `0.0.0.0`。 |
| `--port <number>` | `9000` | HTTP 服务端口。未显式指定时，会从 `9000` 开始自动寻找可用端口。 |
| `--output <dir>` | `<project>/_book` | 预览站点输出目录。 |
| `--open` | 关闭 | 启动后自动打开浏览器。默认不会自动打开浏览器。 |
| `--summary <path>` | 自动检测 | 显式指定 `SUMMARY.md` 文件路径。会覆盖 `book.yaml` 中的章节定义或自动发现结果。 |
| `--config <path>` | `book.yaml` | 本地项目使用的配置文件路径。 |
| `-v, --verbose` | 关闭 | 输出详细日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |
| `--cache-dir <path>` | 系统默认 | 自定义构建缓存目录。 |
| `--no-cache` | 关闭 | 禁用所有构建缓存。 |

## 用法说明

### 预览内容

`serve` 启动前会先生成一份多页 HTML 预览站点，然后启动本地服务。和 `build --format site` 相比，`serve` 更适合开发和预览。

### 自动发现范围

当目标目录中既没有 `book.yaml`，也没有 `SUMMARY.md` 时，`serve` 会递归扫描该目录下的 Markdown 文件。

这意味着：

- 在纯文档目录里执行 `mdpress serve`，通常效果符合预期
- 在代码仓库根目录直接执行 `mdpress serve`，可能会把 `README.md`、`docs/`、`examples/`、`tests/`、内部设计文档等一起纳入预览

如果你只想预览正式文档，建议优先使用下面两种方式之一：

```bash
mdpress serve ./docs
mdpress serve --config ./docs/book.yaml ./docs
```

或者在目标目录中显式提供 `book.yaml` / `SUMMARY.md`，限制章节范围和顺序。

### 输出目录

如果没有传 `--output`，默认输出到项目目录下的 `_book/`。

```bash
mdpress serve
mdpress serve --open
mdpress serve --output ./preview
```

如果没有显式传 `--port`，`serve` 会从 `9000` 开始向上尝试，直到找到一个可用端口。

### 网络监听

默认情况下，`serve` 监听在 `127.0.0.1`，因此只有本机可以访问预览页面。

如果你确实要暴露给其他机器，或者绑定到某个明确的地址，需要显式传入 `--host`：

```bash
mdpress serve --host 0.0.0.0
mdpress serve --host 192.168.1.10 --port 9000
```

### 配置加载

本地目录下的配置加载顺序是：

1. `--config` 指定的文件
2. 默认 `book.yaml`
3. 自动发现 `SUMMARY.md` 或 Markdown 文件

自动发现模式下会跳过隐藏目录、`node_modules`、`vendor`、`_book`，但不会自动识别“哪些 Markdown 只是仓库说明、哪些才是正式章节”。

## 示例

```bash
mdpress serve
mdpress serve --open
mdpress serve --host 0.0.0.0
mdpress serve --port 9000
mdpress serve --port 3000
mdpress serve --output ./preview
mdpress serve /path/to/book
mdpress serve https://github.com/yeasy/agentic_ai_guide
```

## 注意事项

- `serve` 当前支持 GitHub 仓库 URL，但没有提供 `--branch` 和 `--subdir` 参数；远程预览默认使用仓库默认分支和仓库根目录。
- 对远程仓库输入，如果没有显式指定 `--output`，当前实现会把预览产物写到临时目录；进程退出后，这些文件通常也会被清理。
- 对远程仓库输入，当前实现优先读取远程项目中的 `book.yaml`，本地 `--config` 不会覆盖远程配置路径。
- 默认只监听 `127.0.0.1`，不会自动暴露给局域网；如需对外访问，必须显式传入 `--host`。
- 默认不会自动打开浏览器；只有显式传入 `--open` 才会打开。
- `serve` 的站点布局适合“网站式阅读”，不会保留 PDF/打印场景下的页面边距语义；如果你关心最终排版，请同时用 `build --format pdf` 或 `build --format html` 验证。
- `serve` 主要用于本地阅读体验，不会生成 PDF 或 ePub。
- 如果同时传 `--quiet` 和 `--verbose`，当前实现以 `--quiet` 为准。

## 常见问题

### 1. 为什么页面里出现了很多我不想预览的 Markdown？

通常是因为你在仓库根目录运行了 `mdpress serve`，而该目录没有 `book.yaml` 或 `SUMMARY.md`。此时会触发自动发现，把仓库里的大多数 Markdown 都当成候选章节。

优先解决方式：

- 改成 `mdpress serve ./docs`
- 或者补一份 `book.yaml` / `SUMMARY.md`

### 2. 为什么预览布局看起来像“文档页边距”和“网站侧边栏”混在一起？

多页预览站点和 PDF/单页 HTML 的布局目标不同。`serve` 侧重网站式浏览，而不是打印排版。站点模式应以导航、宽度和可读性为主，不应直接拿来判断 PDF 最终效果。

### 3. 图片不显示怎么办？

优先检查：

- Markdown 里的相对路径是否相对当前章节文件而不是仓库根目录
- 图片文件是否真实存在
- 是否在预览远程仓库且网络资源不可用

如果是远程图片，建议优先结合 `mdpress validate` 和 `mdpress doctor` 一起排查。
