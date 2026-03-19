# `mdpress init`

[English](init.md)

## 作用

为一个目录初始化 `mdpress` 项目。它有两种工作模式：

- 目录里已有 Markdown 文件：扫描结构并生成 `book.yaml`
- 目录里没有 Markdown 文件：创建一个最小可用模板

## 语法

```bash
mdpress init [directory] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[directory]` | 否 | 目标目录，省略时默认当前目录。 |

## 命令参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-i, --interactive` | 关闭 | 进入交互模式，询问标题、作者、语言和主题。 |
| `-v, --verbose` | 关闭 | 输出详细日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

## 用法说明

### 扫描已有项目

```bash
mdpress init ./my-book
```

当前实现会：

- 递归扫描 `.md` 文件
- 读取文件中的第一个 H1 作为章节标题
- 生成 `book.yaml`
- 检测 `SUMMARY.md`、`GLOSSARY.md`、`LANGS.md`
- 自动检测常见封面文件名，例如 `cover.png`、`cover.jpg`、`cover.svg`

### 空目录初始化

如果目标目录没有 Markdown 文件，命令会创建一个最小项目骨架，包括：

- `book.yaml`
- `preface.md`
- `chapter01/README.md`

### 交互模式

```bash
mdpress init --interactive
mdpress init ./my-book -i
```

交互模式会询问：

- 标题
- 作者
- 语言
- 主题

如果当前终端不是交互式输入，命令会回退到默认值。

## 示例

```bash
mdpress init
mdpress init ./docs-book
mdpress init --interactive
mdpress init ./docs-book -i
```

## 注意事项

- 如果目标目录已经存在 `book.yaml`，命令会直接报错，不会覆盖。
- 扫描已有项目时，顶层 `README.md` 当前会被当作项目说明而不是章节文件跳过；子目录里的 `README.md` 会保留。
- 如果检测到 `SUMMARY.md`，生成的 `book.yaml` 不会写入 `chapters` 列表，而是交给 `SUMMARY.md` 在构建时决定目录结构。
- `--config` 虽然是全局参数，但当前 `init` 不会根据它改写输出文件名，始终生成目标目录下的 `book.yaml`。
