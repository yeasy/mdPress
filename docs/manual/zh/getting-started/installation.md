# 安装

mdPress 是一个 Go CLI 工具。你可以通过 Homebrew、Docker、`go install`、下载预编译二进制文件，或从源代码构建来安装它。

## 系统要求

- 从源代码构建时需要 Go 1.26 或更高版本。
- PDF 输出需要 Chrome 或 Chromium。
- 如果想尝试 `mdpress build --format typst`，需要安装 Typst。

## 安装

### Homebrew（macOS）

```bash
brew tap yeasy/tap
brew install --cask mdpress
mdpress --version
```

Homebrew cask 会自动移除 macOS 隔离标记，因此 Gatekeeper 不会拦截。

### go install

```bash
go install github.com/yeasy/mdpress@latest
mdpress --version
```

除非设置了 `GOBIN`，否则 `go install` 会将二进制文件放入 `$GOPATH/bin`。如果找不到 `mdpress`，请将该目录加入你的 `PATH`。

### Docker

```bash
# 精简镜像（~15 MB）——不含 Chromium，请选择不依赖它的格式
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/book" \
  ghcr.io/yeasy/mdpress build --format site

# 完整镜像（~300 MB）——内置 Chromium，可生成 PDF
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/book" \
  ghcr.io/yeasy/mdpress:full build --format pdf
```

`build` 默认输出 PDF，而精简镜像生成不了 PDF。在精简镜像里请使用 `site`、`html` 或 `epub`，或改用 `:full` 标签。

两个镜像都以镜像内的 `mdpress` 用户运行，该 UID 在宿主机上并不存在。因此不加 `--user` 时，容器要么根本无法写入挂载目录，要么生成的文件属于一个无关的 UID。`--user "$(id -u):$(id -g)"` 可以让产物归你所有。macOS 与 Windows 上的 Docker Desktop 已代为处理该映射，可以省略 `--user`。

### 直接下载 Binary

从 [GitHub Releases](https://github.com/yeasy/mdpress/releases) 下载对应平台的预编译 binary。支持平台：macOS（amd64 / arm64）、Linux（amd64 / arm64）、Windows（amd64 / arm64）。

> **macOS Gatekeeper 提示：** 二进制目前尚未公证。通过 Homebrew cask 安装会自动移除隔离标记；如果你直接下载二进制且被 macOS 阻止，手动清除一次即可：
>
> ```bash
> xattr -d com.apple.quarantine ./mdpress
> ```

## 从源代码构建

```bash
git clone https://github.com/yeasy/mdpress.git
cd mdpress
go build -o mdpress .
./mdpress --version
```

## 升级

如果你之前用二进制方式安装（如直接下载或某些包管理器），可以用内置命令自更新：

```bash
mdpress upgrade          # 检查并安装最新版本
mdpress upgrade --check  # 仅检查是否有新版本，不安装
```

Homebrew 或 `go install` 管理的安装会分别通过 `brew upgrade` 或再次运行 `go install` 来更新。

## 环境变量

- `GITHUB_TOKEN` —— 用于私有 GitHub 源以及规避 API 速率限制。
- `MDPRESS_CHROME_PATH` —— 指向特定的 Chrome 或 Chromium 可执行文件。
- `MDPRESS_CACHE_DIR` —— 移动缓存目录。
- `MDPRESS_DISABLE_CACHE` —— 禁用缓存（设置为 `1`、`true`、`yes` 或 `on`）。

## 验证安装

```bash
mdpress --help
mdpress doctor
```
