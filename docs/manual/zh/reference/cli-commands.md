# CLI 命令参考

所有 mdPress 命令、标志和选项的完整参考。

## 全局标志

这些标志适用于大多数命令：

```bash
mdpress [global-flags] <command> [command-flags]
```

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--config <path>` | `book.yaml` | 配置文件路径 |
| `--cache-dir <path>` | OS 默认 | 覆盖缓存目录位置 |
| `--no-cache` | off | 禁用所有缓存；强制完全重建 |
| `-v, --verbose` | off | 启用详细输出和调试日志 |
| `-q, --quiet` | off | 仅打印错误；抑制信息消息 |
| `--help` | — | 显示命令帮助并退出 |
| `--version` | — | 显示版本并退出 |

### 示例

```bash
# 使用自定义配置文件
mdpress build --config docs/book.yaml

# 禁用缓存进行完全重建
mdpress build --no-cache --format pdf

# 启用详细输出
mdpress build --verbose

# 安静模式（抑制非错误消息）
mdpress build --quiet
```

## build

在指定格式中构建书籍输出。

```bash
mdpress build [source] [flags]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[source]` | 输入目录或 GitHub URL（默认：当前目录） |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--format <format>` | pdf | 输出格式：`pdf`、`html`、`site`、`epub`、`typst` |
| `--output <file>` | 自动 | 输出文件名或目录 |
| `--config <path>` | book.yaml | 配置文件路径 |
| `--branch <name>` | — | Git 分支名称（仅限 GitHub 源） |
| `--subdir <path>` | — | 源中的子目录 |
| `--summary <path>` | 自动检测 | SUMMARY.md 文件路径 |

### 示例

```bash
# 构建 PDF（默认）
mdpress build

# 构建多种格式
mdpress build --format pdf,html,epub

# 以 HTML 格式构建（最快）
mdpress build --format html

# 为 GitHub Pages 构建网站格式
mdpress build --format site

# 从不同目录构建
mdpress build ./docs

# 使用自定义输出文件名构建
mdpress build --format pdf --output my-book.pdf

# 从 GitHub 仓库构建
mdpress build https://github.com/user/book-repo

# 从私有 GitHub 仓库构建（需要 token）
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-repo

# 使用自定义配置构建
mdpress build --config docs/book.yaml

# 强制完全重建（跳过缓存）
mdpress build --no-cache --format pdf

# 启用详细输出
mdpress build --verbose
```

### 输出文件

| 格式 | 输出 | 位置 |
|--------|--------|----------|
| `pdf` | 单个 PDF 文件 | `./output.pdf` 或自定义文件名 |
| `html` | 单个 HTML 文件 | `./output.html` |
| `site` | 网站目录 | `./_book/` |
| `epub` | 电子书文件 | `./output.epub` |
| `typst` | 通过 Typst CLI 生成的 PDF | `./output-typst.pdf` |

## serve

启动带有实时预览和文件监视的本地服务器。

```bash
mdpress serve [source] [flags]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[source]` | 输入目录（默认：当前目录） |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--port <port>` | 9000 | HTTP 服务器端口 |
| `--host <address>` | 127.0.0.1 | HTTP 监听地址 |
| `--output <path>` | _book | 输出目录 |
| `--summary <path>` | 自动检测 | SUMMARY.md 文件路径 |
| `--config <path>` | book.yaml | 配置文件路径 |
| `--open` | off | 自动打开浏览器 |

### 示例

```bash
# 在默认端口 9000 启动服务器
mdpress serve

# 在自定义端口启动
mdpress serve --port 3000

# 自动打开浏览器
mdpress serve --open

# 预览网站
mdpress serve

# 监视特定目录
mdpress serve ./docs

# 使用自定义配置提供
mdpress serve --config docs/book.yaml --open
```

### 特性

- **实时重载**：文件更改时浏览器自动刷新
- **文件监视**：监控 Markdown、图像和配置
- **增量构建**：仅重建已更改的章节
- **快速反馈**：HTML 预览最快（1-2 秒）

在以下地址访问：http://127.0.0.1:9000

## init

从现有 Markdown 文件生成配置文件。

```bash
mdpress init [directory] [flags]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[directory]` | 扫描 Markdown 文件的目录（默认：当前） |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `-i, --interactive` | off | 启用交互式提示 |

### 示例

```bash
# 在当前目录创建 book.yaml
mdpress init

# 扫描特定目录
mdpress init ./docs

# 创建具有自动发现的章节的 book.yaml
# 列表：README.md、ch01.md、ch02.md、...
```

### 生成的文件

`book.yaml` 包括：
- 检测到的标题（从目录或第一个标题）
- 自动发现的章节文件
- 默认样式和输出设置

编辑生成的文件进行自定义。

## quickstart

创建带有模板文件的新样本项目。

```bash
mdpress quickstart [directory]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[directory]` | 新项目的目录 |

### 示例

```bash
# 创建示例项目
mdpress quickstart my-book
cd my-book

# 立即可以构建
mdpress build
```

### 生成的文件

```
my-book/
├── book.yaml         # 配置
├── SUMMARY.md        # 章节结构
├── README.md         # 介绍
├── chapter1.md       # 示例章节
├── chapter2.md       # 示例章节
├── assets/
│   └── cover.png     # 示例封面图像
└── .gitignore        # Git 配置
```

提供一个可以构建的工作示例。

## validate

检查配置并验证项目结构。

```bash
mdpress validate [directory] [flags]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[directory]` | 要验证的项目目录（默认：当前） |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--report <path>` | — | 将验证报告写入 .json 或 .md 文件 |

### 示例

```bash
# 验证当前项目
mdpress validate

# 验证特定目录
mdpress validate ./docs

# 输出示例：
# ✓ Config file syntax OK
# ✓ All 12 chapters exist
# ✓ 24 images found
# ✓ Cross-references valid
# ✓ Validation successful!
```

### 检查

- 配置文件语法
- 章节文件存在
- 图像文件存在
- 交叉引用有效性
- 链接格式正确性
- GLOSSARY.md 格式（如存在）
- LANGS.md 格式（如存在）

在构建前运行以尽早捕捉问题。

## doctor

检查环境和系统就绪情况。

```bash
mdpress doctor [directory] [flags]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[directory]` | 要检查的项目目录（默认：当前） |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--report <path>` | — | 将诊断报告写入 .json 或 .md 文件 |

### 示例

```bash
# 检查环境
mdpress doctor

# 生成 JSON 报告
mdpress doctor --report report.json

# 生成 Markdown 报告
mdpress doctor --report report.md

# 检查特定项目
mdpress doctor ./docs
```

### 检查

- 平台和 OS 版本
- Go 安装
- Chrome/Chromium 可用性
- CJK 字体安装
- PlantUML 安装
- 缓存目录状态
- 配置有效性
- 章节文件引用
- 图像文件引用

参见 [doctor.md](../troubleshooting/doctor.md) 了解详细信息。

## upgrade

检查并安装 mdpress 的新版本。

```bash
mdpress upgrade [flags]
```

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--check` | off | 仅检查更新，不进行安装 |
| `-v, --verbose` | off | 启用详细输出 |
| `-q, --quiet` | off | 仅打印错误 |

### 示例

```bash
# 检查更新但不安装
mdpress upgrade --check

# 安装最新版本（默认行为）
mdpress upgrade

# 使用详细输出进行安装
mdpress upgrade --verbose

# 验证升级是否完成
mdpress --version
```

### 特性

- **自动平台检测**：自动识别你的操作系统和架构并下载对应的二进制文件
- **备份和恢复**：安装前创建备份，出错时自动恢复
- **版本比较**：使用语义化版本号检测新版本
- **进度反馈**：显示下载进度和完成状态

支持的平台：
- **Linux**：x86_64、ARM64
- **macOS**：x86_64、ARM64（Apple Silicon）
- **Windows**：x86_64、ARM64

### 环境变量

- `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY`：如需要，配置 HTTP 代理

### 常见问题

参见 [upgrade 故障排除](../../../commands/upgrade_zh.md#常见问题和解决方案) 了解解决方案。

## migrate

将 GitBook 或 HonKit 项目转换为 mdPress。

```bash
mdpress migrate [directory]
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `[directory]` | GitBook/HonKit 项目目录 |

### 标志

| 标志 | 默认值 | 描述 |
|------|---------|-------------|
| `--dry-run` | off | 预览变更而不写入任何文件 |
| `--force` | off | 覆盖已有的 book.yaml 而非跳过 |

### 示例

```bash
# 迁移现有 GitBook 项目
mdpress migrate ./gitbook-project

# 转换：
# - book.json → book.yaml
# - SUMMARY.md → 更新为 mdPress 格式
# - Markdown 文件 → mdPress 兼容
```

### 转换

- `book.json` → `book.yaml`（带属性映射）
- `SUMMARY.md` → 兼容格式
- 插件配置 → mdPress 插件
- 主题设置 → CSS 覆盖
- 输出选项 → mdPress 输出配置

## themes

管理主题并查看主题信息。

```bash
mdpress themes <subcommand> [flags]
```

### 子命令

#### list

列出可用的主题。

```bash
mdpress themes list

# 输出：
# Built-in themes:
#   - technical (default)
#   - elegant
#   - minimal
```

#### show

显示主题详细信息和配置选项。

```bash
mdpress themes show <theme-name>

# 示例：
mdpress themes show technical
mdpress themes show elegant

# 输出包括：
# Theme: technical
# Description: Professional technical documentation theme
# Colors: ...
# Fonts: ...
# CSS variables: ...
```

#### preview

生成所有内置主题的预览。

```bash
mdpress themes preview

# 输出：preview-themes.html
# 在浏览器中查看以比较主题
```

### 示例

```bash
# 列出所有主题
mdpress themes list

# 显示技术主题详细信息
mdpress themes show technical

# 生成预览 HTML
mdpress themes preview

# 在 book.yaml 中使用主题
# style:
#   theme: "elegant"
```

## completion

生成 shell 完成脚本。

```bash
mdpress completion <shell>
```

### 参数

| 参数 | 描述 |
|----------|-------------|
| `<shell>` | Shell 类型：`bash`、`zsh`、`fish`、`powershell` |

### 示例

```bash
# Bash 完成
mdpress completion bash > mdpress-completion.bash
source mdpress-completion.bash

# Zsh 完成
mdpress completion zsh > ~/.zfunc/_mdpress

# Fish 完成
mdpress completion fish > ~/.config/fish/completions/mdpress.fish

# PowerShell 完成
mdpress completion powershell >> $PROFILE

# 然后按 Tab 自动完成命令
mdpress build<Tab>  # 建议 --format、--config 等
```

### 添加到 Shell 配置

**Bash (~/.bashrc 或 ~/.bash_profile)：**
```bash
source /path/to/mdpress-completion.bash
```

**Zsh (~/.zshrc)：**
```bash
fpath=(~/.zfunc $fpath)
autoload -Uz compinit && compinit
```

**Fish (~/.config/fish/config.fish)：**
```fish
# 完成从 ~/.config/fish/completions/ 自动加载
```

**PowerShell ($PROFILE)：**
```powershell
# 完成自动加载
```

## 版本和帮助

### version

显示 mdPress 版本和构建信息。

```bash
mdpress version
# 或
mdpress --version

# 输出：
# mdpress version 0.7.5
# Go version: go1.26.1
# Platform: linux/amd64
```

### help

显示一般帮助或命令特定帮助。

```bash
# 一般帮助
mdpress help
mdpress --help
mdpress -h

# 命令帮助
mdpress build --help
mdpress serve --help

# 输出：用法、标志、示例
```

## 环境变量

### MDPRESS_CHROME_PATH

Chrome 或 Chromium 二进制文件的路径：

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
mdpress build --format pdf
```

### GITHUB_TOKEN

用于私有仓库的 GitHub 个人访问 token：

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-repo
```

需要 `contents:read` 范围。

### MDPRESS_CACHE_DIR

覆盖缓存目录位置：

```bash
export MDPRESS_CACHE_DIR=/path/to/custom/cache
mdpress build --format pdf
```

默认位置为操作系统临时目录下的 `mdpress-cache` 子目录（例如 `/tmp/mdpress-cache`）。也可以通过 `--cache-dir` 标志覆盖。

## 退出代码

| 代码 | 含义 |
|------|---------|
| 0 | 成功 |
| 1 | 配置错误或验证失败 |
| 2 | 运行时错误（如 PDF 缺少 Chrome） |
| 3 | 文件或权限错误 |

示例：

```bash
mdpress build --format pdf
echo $?  # 打印退出代码
# 0 = 成功，非零 = 错误
```

在脚本中使用：

```bash
#!/bin/bash
mdpress build --format pdf
if [ $? -ne 0 ]; then
  echo "Build failed!"
  exit 1
fi
echo "Build succeeded!"
```

## 命令示例

### 快速预览

```bash
# 最快的预览（HTML、实时重载）
mdpress serve --open
```

### 开发工作流

```bash
# 在浏览器中启动实时预览
mdpress serve --port 3000 --open

# 在另一个终端，对 Markdown 文件进行更改
# → 浏览器自动刷新

# 准备就绪时，构建 PDF
mdpress build --format pdf
```

### CI/CD 集成

```bash
# 构建前验证
mdpress validate
if [ $? -ne 0 ]; then exit 1; fi

# 构建所有格式
mdpress build --format pdf
mdpress build --format site

# 部署网站
cp -r _book/* /var/www/docs/
```

### 批量处理

```bash
# 构建多个项目
for project in docs1 docs2 docs3; do
  mdpress build "$project" --format pdf
done
```

### 自定义输出名称

```bash
# 版本特定的 PDF
mdpress build --format pdf --output "book-v1.0.pdf"

# 带时间戳的输出
mdpress build --format html --output "snapshot-$(date +%Y%m%d).html"
```

## 故障排除命令问题

### 命令未找到

```bash
# 检查 mdPress 是否安装
mdpress --version

# 如果未找到，安装：
go install github.com/yeasy/mdpress@latest

# 或使用完整路径运行二进制文件
/path/to/mdpress build
```

### 权限被拒绝

```bash
# 使二进制文件可执行
chmod +x mdpress

# 然后运行
./mdpress build
```

### 未知命令

```bash
# 显示所有可用命令
mdpress --help

# 显示命令特定帮助
mdpress build --help
```

更多关于配置和变量的信息，参见 [configuration.md](configuration.md) 和 [template-variables.md](template-variables.md)。
