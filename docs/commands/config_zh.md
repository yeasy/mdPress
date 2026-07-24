# `mdpress config`

[English](config.md)

## 用途

查看 mdPress 为一个项目实际解析出的配置，让“我明明设置了却没生效”从靠猜变成可以回答。

## 语法

```bash
mdpress config show [directory] [flags]
```

## 子命令

| 子命令 | 说明 |
| --- | --- |
| `show` | 打印构建时实际生效的配置 |

## 位置参数

| 参数 | 必需 | 说明 |
| --- | --- | --- |
| `[directory]` | 否 | 项目目录。省略时使用当前目录。 |

## 参数

| 参数 | 简写 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--format` | `-f` | `yaml` | 输出编码：`yaml` 或 `json` |

`config show` 同样支持全局参数，其中 `--config <path>` 可用于查看 `book.yaml` 之外的配置文件。

## 会打印什么

两部分：

1. 应用默认值之后的 `book.yaml` 设置 —— 没有 `book.yaml` 时则是自动发现推断出的设置。你在
   这里能看到 `output.filename` 其实是空的而不是 `output.pdf`，或者 `book.language` 解析
   成了 `en-US`。
2. 一个 `resolved` 小节，里面是 mdPress 计算出来（而非直接读取）的值：加载了哪个配置文件、
   它是否来自自动发现、基准目录、章节数、glossary 与 `LANGS.md` 的路径、主题的名字和来源
   （内置还是项目里的 `themes/<name>.yaml`）、样式覆盖之后渲染器收到的排版参数，以及每种
   请求的输出格式将写入哪个文件。

## 示例

```bash
mdpress config show
mdpress config show ./my-book
mdpress config show --config release.yaml
mdpress config show --format json
```

脚本化：

```bash
mdpress config show --format json | jq -r .style.theme
mdpress config show --format json | jq -r .resolved.artifacts.pdf
```

## 注意事项

- 该命令加载配置的方式与构建完全一致，因此配置错误在这里出现的形式与 `build` 时相同。
- 它是只读的：不会写入任何文件，也不会渲染任何章节。
