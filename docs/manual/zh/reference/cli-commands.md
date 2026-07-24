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

- `-f, --format` 接受逗号分隔的格式，例如 `pdf,html,epub`，或 `all`。
- `--branch` 和 `--subdir` 适用于 GitHub 源。
- `-o, --output` 设置输出目标：已存在的目录（或以斜杠结尾的路径）会接收各格式文件，站点页面直接写入该目录；其他路径作为文件名基名（`manual.pdf`、`manual.html`、`manual_site/`）。当 `site` 是唯一请求的格式时，该路径会被原样当作站点目录（`--format site -o ./dist` 生成 `dist/index.html`）。不传 `--output` 时，站点格式输出到项目目录下的 `_book/`。
- `--summary` 从指定的 `SUMMARY.md` 加载章节。
- `--allow-plugins` 执行远程项目 `book.yaml` 中声明的插件（插件是任意可执行程序，默认跳过；本地项目始终执行插件）。

### `serve`

```bash
mdpress serve [source] [flags]
```

启动实时预览服务器。

它接受当前目录、本地目录或 GitHub URL。

主要标志：

- `--host` 设置监听地址。
- `--port` 设置监听端口。
- `-o, --output` 设置预览输出目录（默认 `_book/`）。
- `--open` 自动打开浏览器。
- `--summary` 从指定的 `SUMMARY.md` 加载章节。
- `--branch` 和 `--subdir` 适用于 GitHub 源。
- `--allow-plugins` 执行远程项目 `book.yaml` 中声明的插件。

### `init`

```bash
mdpress init [directory] [-i]
```

扫描 Markdown 文件并生成 `book.yaml`。使用 `-i, --interactive` 可交互式回答标题、作者、语言和主题的提示。

### `quickstart`

```bash
mdpress quickstart [directory] [--force]
```

创建一个示例项目。`--force` 允许在非空目录中生成脚手架（已有文件绝不会被覆盖）。

### `validate`

```bash
mdpress validate [directory] [flags]
```

验证配置、被引用的文件、图像、章节链接以及页内锚点。它还会报告没有被任何章节列表收录的
Markdown 文件，以及被列为多个章节的文件。

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--report <path>` | — | 将校验报告写入 `.json` 或 `.md` 文件 |
| `--strict` | off | 只要出现任何告警（而不只是错误）就以非零状态退出 |

不加 `--strict` 时只有错误会让运行失败，因此告警 —— 重复的章节条目、无法识别的配置键 ——
仍然以 0 退出。把 validate 用作 CI 门禁时请加上 `--strict`：

```bash
mdpress validate --strict
```

### `config show`

```bash
mdpress config show [directory] [flags]
```

打印这个项目在构建时实际会使用的配置：应用默认值之后的 `book.yaml` 设置（没有 `book.yaml`
时则是自动发现推断出的设置），外加一个 `resolved` 小节，说明加载了哪个配置文件、主题来自
哪里、样式覆盖之后渲染器收到的排版参数，以及每种请求的格式将写入哪个文件。

遇到“我明明设置了却没生效”时就用这个命令。

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `-f, --format <yaml\|json>` | `yaml` | 输出编码 |

```bash
mdpress config show
mdpress config show ./my-book
mdpress config show --config release.yaml

# 脚本化
mdpress config show --format json | jq -r .style.theme
mdpress config show --format json | jq -r .resolved.artifacts.pdf
```

### `cache info` / `cache clear`

```bash
mdpress cache info
mdpress cache clear
```

mdPress 会缓存解析后的章节以及其他构建中间产物，使未改动的章节不必重新渲染。`cache info`
打印缓存位置、条目数和占用大小；`cache clear` 删除全部条目。两周未使用的条目会被自动清理，
所以这个命令用于立刻回收空间，或强制一次完全冷启动的重建。

缓存位置由 `--cache-dir` 或 `MDPRESS_CACHE_DIR` 决定；`--no-cache` 只让单次命令绕过缓存，
不会删除任何东西。

## doctor

检查环境与系统就绪情况。

```bash
mdpress doctor [directory] [flags]
```

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `-r, --report <path>` | — | 将诊断报告写入 `.json` 或 `.md` 文件 |
| `--strict` | off | 当任何错误级检查失败时以非零状态退出（可用作 CI 门禁） |

```bash
# 检查环境
mdpress doctor

# 生成 JSON 报告
mdpress doctor --report report.json

# 在 CI 中让错误级检查导致任务失败
mdpress doctor --strict

# 检查特定项目
mdpress doctor ./docs
```

检查项：平台/OS、Go 安装、Chrome/Chromium 与 Typst 可用性、CJK 字体、git、网络、磁盘空间、缓存目录、已声明的插件、配置有效性、章节和图像引用。

详见 [doctor.md](../troubleshooting/doctor.md)。

## upgrade

检查并安装 mdpress 的新版本。

```bash
mdpress upgrade [flags]
```

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--check` | off | 仅检查更新，不安装 |
| `--force` | off | 强制替换二进制，即使安装来源是 Homebrew/`go install` |
| `--skip-checksum` | off | 跳过对下载二进制的校验和验证（不推荐） |
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
# Bash（写入 ~/.bashrc 可永久生效）
source <(mdpress completion bash)

# Zsh（写入 ~/.zshrc 可永久生效）
source <(mdpress completion zsh)

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

| 标志 | 默认值 | 用途 |
| --- | --- | --- |
| `--json` | off | 以 JSON 打印构建信息 |

```bash
mdpress version --json | jq -r .version
```

该 JSON 对象包含 `version`、`commit`、`built_at`、`go_version`、`os` 和 `arch`。

## 环境变量

### MDPRESS_CHROME_PATH

Chrome 或 Chromium 二进制文件的路径：

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
mdpress build --format pdf
```

### MDPRESS_CACHE_DIR

构建缓存所在目录，等价于 `--cache-dir`。在 CI 中很有用 —— 缓存需要放在任务能够恢复的位置：

```bash
export MDPRESS_CACHE_DIR=.mdpress-cache
mdpress build --format site
```
