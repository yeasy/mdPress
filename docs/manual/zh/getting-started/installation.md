# 安装

mdPress 是一个快速、现代的文档和书籍生成工具，使用 Go 语言编写。本指南介绍如何在你的系统上安装 mdPress。

## 系统要求

安装 mdPress 之前，请确保你的系统满足以下要求：

- **Go 1.26 或更高版本** —— 构建和运行 mdPress 所需
- **Chrome 或 Chromium 浏览器** —— PDF 生成所需
- **Typst**（可选）—— 用于高级 PDF 样式和排版功能
- **PlantUML**（可选）—— 用于渲染文档中的图表语法

### 支持的平台

mdPress 支持以下平台：
- macOS（Intel 和 Apple Silicon）
- Linux（x86_64 和 arm64）
- Windows（x86_64 和 arm64）

## 安装方式

### 使用 go install

最快的安装方式是使用 `go install`：

```bash
go install github.com/yeasy/mdpress@latest
```

这将下载最新版本并将二进制文件安装到你的 `$GOPATH/bin` 目录（通常是 `~/go/bin`）。

验证安装：

```bash
mdpress --version
```

### 从源代码构建

如果你偏好从源代码构建或想使用开发版功能：

```bash
git clone https://github.com/yeasy/mdpress.git
cd mdpress
go build -o mdpress .
```

将二进制文件复制到 PATH 中的某个位置，或直接使用它：

```bash
./mdpress --version
```

## 验证安装

安装完成后，验证 mdPress 正常工作：

```bash
mdpress --help
```

你应该能看到带有可用命令的帮助信息。检查版本：

```bash
mdpress --version
```

## 环境变量

mdPress 支持多个环境变量配置：

### GITHUB_TOKEN

用于访问私有仓库或避免 API 速率限制的 GitHub token。如果计划从 GitHub 获取内容，请设置此变量：

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

### MDPRESS_CHROME_PATH

指定 Chrome 或 Chromium 可执行文件的路径。mdPress 会自动在标准位置查找 Chrome/Chromium，但如果你的安装位置非标准，请使用此变量：

```bash
export MDPRESS_CHROME_PATH=/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome
```

在 Linux 上：

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
```

在 Windows 上：

```bash
set MDPRESS_CHROME_PATH=C:\Program Files\Google\Chrome\Application\chrome.exe
```

### MDPRESS_CACHE_DIR

mdPress 存储缓存内容和临时文件的目录。默认为系统临时目录下的 `mdpress-cache` 子目录（例如 `/tmp/mdpress-cache`）。在 CI/CD 环境中非常有用：

```bash
export MDPRESS_CACHE_DIR=/var/cache/mdpress
```

### MDPRESS_DISABLE_CACHE

禁用所有缓存（在开发或测试时很有用）。设置为任何值：

```bash
export MDPRESS_DISABLE_CACHE=1
mdpress build  # 将绕过缓存
```

## 可选依赖

### 安装 Typst

Typst 提供高级 PDF 格式化和样式功能。访问 https://github.com/typst/typst 为你的平台安装。

在 macOS 上使用 Homebrew：

```bash
brew install typst
```

在 Linux 上：

```bash
# 从 https://github.com/typst/typst/releases 下载
# 或使用包管理器
apt-get install typst  # Debian/Ubuntu（如果可用）
```

在 Windows 上：

```bash
choco install typst  # 使用 Chocolatey
```

### 安装 PlantUML

如果在文档中使用图表语法，需要 PlantUML。它需要 Java：

```bash
# macOS
brew install plantuml

# Linux
apt-get install plantuml  # Debian/Ubuntu

# Windows - 从 https://plantuml.com/download 下载
```

## 故障排除

### 命令未找到：mdpress

确保 `$GOPATH/bin` 在你的 PATH 中：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

将此添加到你的 shell 配置文件（`~/.bashrc`、`~/.zshrc` 或 `~/.profile`）以使其永久生效。

### 未找到 Chrome/Chromium

显式设置 Chrome 路径：

```bash
export MDPRESS_CHROME_PATH=/path/to/chrome
mdpress build --format pdf
```

### Linux 上的权限被拒绝

在某些 Linux 系统上，你可能需要允许 mdPress 运行：

```bash
chmod +x $(which mdpress)
```

### PDF 生成缓慢

PDF 生成是 I/O 密集型操作。如果速度太慢：

1. 确保磁盘空间充足
2. 尝试禁用缓存：`export MDPRESS_DISABLE_CACHE=1`
3. 使用 `MDPRESS_CACHE_DIR` 指向更快的存储设备
4. 检查你的 Chrome/Chromium 路径是否正确
