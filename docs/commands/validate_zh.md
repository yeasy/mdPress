# `mdpress validate`

[English](validate.md)

## 作用

校验项目配置和引用资源，帮助你在构建前发现明显问题。

## 语法

```bash
mdpress validate [directory] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[directory]` | 否 | 目标目录，省略时默认当前目录。 |

## 命令参数

| 参数 | 简写 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--report` | | （无） | 将校验报告写入 `.json` 或 `.md` 文件 |

`validate` 同时使用以下全局参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--config <path>` | `book.yaml` | 当前目录校验时使用的配置文件路径。 |
| `-v, --verbose` | 关闭 | 输出更详细的日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

## 校验内容

当前实现会检查：

- 配置文件是否存在、语法是否正确
- `book.title` 是否存在
- 章节列表是否为空
- 章节文件是否存在
- 封面图片是否存在
- `custom_css` 是否存在
- Markdown 中引用的本地图片是否存在
- 章节标题编号和 Mermaid 相关诊断
- Markdown 章节链接是否指向构建图谱内的文件

## 配置解析规则

### 不传目录

```bash
mdpress validate
mdpress validate --config ./configs/book.yaml
```

此时会优先使用 `--config` 指定的路径。

### 传入目录

```bash
mdpress validate /path/to/book
```

此时当前实现会优先查找 `/path/to/book/book.yaml`。如果没有，再尝试自动发现 `SUMMARY.md` 或 Markdown 文件。

## 示例

```bash
mdpress validate
mdpress validate --config ./book.dev.yaml
mdpress validate /path/to/book
mdpress validate ./examples/chapter01
```

## 注意事项

- `book.author` 缺失当前会作为警告显示，不会直接导致校验失败。
- 如果传入了 `[directory]`，当前实现不会优先使用 `--config` 指定的其他路径，而是先检查目标目录下的 `book.yaml`。
- `validate` 适合发现显式错误，但不能替代一次真实构建，尤其是 PDF 生成依赖和渲染类问题仍应通过 `build` 或 `doctor` 验证。
