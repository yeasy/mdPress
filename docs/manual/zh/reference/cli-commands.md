# CLI 命令参考

本页汇总当前 CLI 提供的命令和标志。

## 全局标志

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--config <path>` | `book.yaml` | 本地项目的配置文件。 |
| `--cache-dir <path>` | OS 默认 | 覆盖缓存目录。 |
| `--no-cache` | off | 禁用运行时缓存。 |
| `-v, --verbose` | off | 显示详细日志。 |
| `-q, --quiet` | off | 仅打印错误。 |
| `--version` | - | 打印版本号。 |

`--config` 用于本地项目。当你传入 GitHub URL 作为源时，mdPress 使用抓取到的仓库内部的配置文件。

## 核心命令

### `build`

```bash
mdpress build [source] [flags]
```

从当前目录、本地目录或 GitHub URL 构建文档。

主要标志：

- `--format` 接受逗号分隔的格式，例如 `pdf,html,epub`，或 `all`。
- `--branch` 和 `--subdir` 适用于 GitHub 源。
- `--output` 设置输出基础路径。
- `--summary` 从指定的 `SUMMARY.md` 加载章节。

### `serve`

```bash
mdpress serve [source] [flags]
```

启动实时预览服务器。

它接受当前目录、本地目录或 GitHub URL。

主要标志：

- `--host` 设置监听地址。
- `--port` 设置监听端口。
- `--output` 设置预览输出目录。
- `--open` 自动打开浏览器。
- `--summary` 从指定的 `SUMMARY.md` 加载章节。

### `init`

```bash
mdpress init [directory] [-i]
```

扫描 Markdown 文件并生成 `book.yaml`。使用 `-i, --interactive` 可交互式回答标题、作者、语言和主题的提示。

### `quickstart`

```bash
mdpress quickstart [directory]
```

创建一个示例项目。该命令没有专用标志。

### `validate`

```bash
mdpress validate [directory] [--report path]
```

验证配置、被引用的文件、图像和章节链接。`--report` 写入 `.json` 或 `.md` 报告。

## doctor

检查环境与系统就绪情况。

```bash
mdpress doctor [directory] [flags]
```

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--report <path>` | — | 将诊断报告写入 `.json` 或 `.md` 文件 |

```bash
# 检查环境
mdpress doctor

# 生成 JSON 报告
mdpress doctor --report report.json

# 检查特定项目
mdpress doctor ./docs
```

检查项：平台/OS、Go 安装、Chrome/Chromium 可用性、CJK 字体、PlantUML、缓存目录、配置有效性、章节和图像引用。

详见 [doctor.md](../troubleshooting/doctor.md)。

## upgrade

检查并安装 mdpress 的新版本。

```bash
mdpress upgrade [flags]
```

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--check` | off | 仅检查更新，不安装 |
| `-v, --verbose` | off | 启用详细输出 |
| `-q, --quiet` | off | 仅打印错误 |

```bash
# 检查更新但不安装
mdpress upgrade --check

# 安装最新版本
mdpress upgrade

# 验证升级结果
mdpress --version
```

特性：自动平台检测、备份与恢复、语义化版本比较、进度反馈。支持 Linux、macOS 和 Windows 上的 x86_64 与 ARM64。

## migrate

将 GitBook 或 HonKit 项目转换为 mdPress。

```bash
mdpress migrate [directory]
```

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--dry-run` | off | 预览变更而不写入文件 |
| `--force` | off | 覆盖已有的 `book.yaml` 而非跳过 |

```bash
# 迁移 GitBook 项目
mdpress migrate ./gitbook-project
```

将 `book.json` 转换为 `book.yaml`，更新 `SUMMARY.md`，并把插件和主题设置映射为 mdPress 的等价物。

## themes

管理主题并查看主题信息。

```bash
mdpress themes <subcommand>
```

### `themes list`

列出可用主题。

```bash
mdpress themes list
```

### `themes show`

显示主题详情和配置选项。

```bash
mdpress themes show <theme-name>
```

### `themes preview`

生成所有内置主题的预览。

```bash
mdpress themes preview
# 输出：themes-preview.html
```

使用 `-o, --output <path>` 将预览写入自定义位置：

```bash
mdpress themes preview --output custom-preview.html
```

## completion

生成 shell 补全脚本。

```bash
mdpress completion <shell>
```

支持的 shell：`bash`、`zsh`、`fish`、`powershell`。

```bash
# Bash
mdpress completion bash > mdpress-completion.bash
source mdpress-completion.bash

# Zsh
mdpress completion zsh > ~/.zfunc/_mdpress

# Fish
mdpress completion fish > ~/.config/fish/completions/mdpress.fish

# PowerShell
mdpress completion powershell >> $PROFILE
```

## version

显示 mdPress 版本和构建信息。

```bash
mdpress version
mdpress --version
```

## 环境变量

### MDPRESS_CHROME_PATH

Chrome 或 Chromium 二进制文件的路径：

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
mdpress build --format pdf
```
