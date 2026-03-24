# `mdpress doctor`

[English](doctor.md)

## 作用

输出当前运行环境信息，并检查目标项目是否具备最基本的 PDF / 项目可加载条件，适合在安装完成后或构建失败时快速排查问题。

## 语法

```bash
mdpress doctor [directory] [flags]
```

## 位置参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `[directory]` | 否 | 要检查的项目目录，省略时默认当前目录。 |

## 命令参数

`doctor` 支持一个专属报告参数，以及常见日志参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--report <path>` | 空 | 将检查结果写入 `.json` 或 `.md` 报告文件。 |
| `-v, --verbose` | 关闭 | 输出详细日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

## 检查内容

当前实现分两类输出：

环境信息：

- Go 运行平台信息
- Go 版本（建议 >= 1.25）
- 缓存目录状态

可用性检查：

- Chrome / Chromium 是否可用（用于 PDF 输出）
- Go 版本检查（>= 1.25）
- Git 是否可用（用于远程源构建）
- github.com 网络连通性
- 输出目录磁盘空间
- CJK 字体可用性（用于亚洲文本 PDF 渲染）
- PlantUML 可用性（当项目中检测到 PlantUML 图表时）
- 插件健康状态和可用性
- `book.yaml`、`SUMMARY.md` 和 `LANGS.md` 的存在性
- 项目是否可以成功加载
- Markdown 章节链接是否落在当前构建图谱内

## 示例

```bash
mdpress doctor
mdpress doctor /path/to/book
mdpress doctor ./examples/chapter01
```

## 注意事项

- `--verbose` 提供详细输出，包括精确的磁盘空间、Go 版本解析详情和每项检查的详细错误信息。
- PlantUML 可用性仅在项目中检测到 PlantUML 图表时才进行检查。
- 插件检查验证路径存在性和可执行性。
- 如果没有检测到 Chrome 或 Chromium，`doctor` 会明确提示 PDF 输出将失败。
- 如果目录没有 `book.yaml` 但存在 `SUMMARY.md`，当前实现会尝试按自动发现方式加载项目。
- `doctor` 不会修改任何文件。
- `--config` 虽然是全局参数，但当前 `doctor` 不会按这个参数切换到其他配置文件路径。
