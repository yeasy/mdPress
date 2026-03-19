# `mdpress quickstart`

[English](quickstart.md)

## 作用

创建一个完整的示例项目，适合第一次体验 `mdpress` 或快速搭建演示仓库。

## 语法

```bash
mdpress quickstart [directory] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[directory]` | 否 | 目标目录，默认是 `my-book`。 |

## 命令参数

`quickstart` 当前没有专属参数，支持常见全局日志参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-v, --verbose` | 关闭 | 输出详细日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

## 会创建什么

当前实现会生成：

- `book.yaml`
- `README.md`
- `preface.md`
- `chapter01/README.md`
- `chapter02/README.md`
- `chapter03/README.md`
- `images/README.md`
- `images/cover.svg`

生成完成后，项目可以直接执行构建和预览。

## 示例

```bash
mdpress quickstart
mdpress quickstart my-book
mdpress quickstart ./examples/demo-book
```

## 推荐后续命令

```bash
cd my-book
mdpress build --format html
mdpress serve
```

## 注意事项

- 如果目标目录已存在且非空，命令会拒绝写入，避免覆盖用户文件。
- 如果目标目录已存在但为空，当前实现允许写入。
- `quickstart` 用于创建演示项目，不会扫描你现有的 Markdown 内容；要接入已有目录请使用 `mdpress init`。
- `--config` 虽然会出现在全局参数里，但当前 `quickstart` 不会使用它。
