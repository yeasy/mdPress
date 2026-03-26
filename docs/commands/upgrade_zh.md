# `mdpress upgrade`

[English](upgrade.md)

## 作用

检查 mdpress 的新版本并自动安装。`upgrade` 命令通过自动化版本检查和二进制替换，简化了 mdpress 安装的更新过程。

## 语法

```bash
mdpress upgrade [flags]
```

## 命令参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--check` | 关闭 | 仅检查更新，不进行安装。用于 CI/CD 流程或定期健康检查。 |
| `-v, --verbose` | 关闭 | 升级过程中输出详细日志。 |
| `-q, --quiet` | 关闭 | 仅输出错误。 |

## 用法说明

### 版本检查和下载

`upgrade` 命令执行以下步骤：

1. **获取最新发布**：通过 GitHub API 查询最新的 mdpress 发布版本
2. **比较版本**：使用语义化版本号比较最新版本和当前版本
3. **平台检测**：自动检测你的操作系统和架构（Linux、macOS、Windows；x86_64、arm64）
4. **下载二进制**：为你的平台下载相应的预编译二进制文件
5. **备份当前版本**：创建当前 mdpress 二进制的备份（扩展名为 `.backup`）
6. **替换二进制**：使用可执行权限安装新的二进制文件
7. **清理**：成功安装后删除备份

如果安装过程中出现错误，命令会自动尝试从备份中恢复。

### 版本检测

升级命令的版本检测方式：

- 使用语义化版本号（例如 `1.2.3`）进行比较
- 如果 GitHub 标签包含 `v` 前缀，会自动去除
- 按数字方式比较版本组件
- 识别你已在运行最新版本的情况

### 支持的平台

升级命令自动支持：

- **Linux**：x86_64（amd64）、ARM64（aarch64）
- **macOS**：x86_64（Intel）、ARM64（Apple Silicon）
- **Windows**：x86_64（amd64）、ARM64（aarch64）

如果最新发布版本中找不到你的平台对应的二进制文件，升级将失败并显示清晰的错误信息。

## 示例

```bash
# 检查更新但不安装
mdpress upgrade --check

# 安装最新版本（默认行为）
mdpress upgrade

# 使用详细输出进行故障排除
mdpress upgrade --verbose

# 验证升级是否完成
mdpress --version
```

## 环境变量

`upgrade` 命令尊重标准的 HTTP 环境变量：

| 变量 | 说明 |
| --- | --- |
| `HTTP_PROXY` | 用于发布下载的 HTTP 代理 URL |
| `HTTPS_PROXY` | 用于发布下载的 HTTPS 代理 URL |
| `NO_PROXY` | 以逗号分隔的不使用代理的域名 |

## 注意事项

- **备份创建**：替换二进制文件前，会创建一个以 `.backup` 为扩展名的备份。成功安装后，此备份会自动删除。
- **自动恢复**：安装失败时，命令会自动从备份中恢复。
- **手动恢复**：如果恢复失败，会显示备份位置，方便你手动恢复。
- **二进制位置**：升级会替换 `os.Executable()` 返回的位置的二进制文件，即正在运行的 mdpress 二进制文件的路径。
- **权限设置**：新二进制文件以可执行权限（`0755`）写入。
- **GitHub 速率限制**：GitHub API 对未认证请求允许每小时每个 IP 60 个请求。升级不应达到此限制。
- **网络连通性**：命令需要互联网接入以检查和下载新版本。
- **安装权限**：在 Linux/macOS 上，如果二进制文件安装在系统目录中（例如 `/usr/local/bin/`），你可能需要提升权限来替换它。

## 常见问题和解决方案

### "你的平台找不到二进制文件"

最新的 GitHub 发布版本中没有你的操作系统/架构组合的预编译二进制文件时出现此错误。

**解决方案：**
- 检查你的平台是否[受支持](#支持的平台)
- 从源代码构建：`go install github.com/yeasy/mdpress@latest`
- 等待包含你平台的下一个发布版本

### "无法确定当前可执行文件路径"

系统无法确定 mdpress 二进制文件的位置时出现此错误。

**解决方案：**
- 直接运行 mdpress（不要通过符号链接或包装脚本）
- 运行 `which mdpress` 验证二进制文件位置
- 尝试使用完整路径：`/usr/local/bin/mdpress upgrade`

### "无法创建备份"或"无法安装二进制文件"

安装失败，可能是由于文件权限问题。

**解决方案：**
- 检查文件权限：`ls -l $(which mdpress)`
- 尝试提升权限：`sudo mdpress upgrade`
- 检查磁盘空间：`df -h`
- 手动从备份恢复：`mv mdpress.backup mdpress`

### "无法获取最新发布"且代理错误

网络错误，可能是由于代理或防火墙。

**解决方案：**
- 如果需要，配置 HTTP 代理：
  ```bash
  export HTTPS_PROXY=https://proxy.example.com:8080
  mdpress upgrade
  ```
- 使用详细输出尝试：`mdpress upgrade --verbose`
- 测试网络连通性：`curl -I https://api.github.com`

### 升级失败后留下备份文件

存在 `.backup` 文件但新二进制文件没有正确安装。

**解决方案：**
```bash
# 检查当前状态
ls -la $(which mdpress)*

# 如果新二进制文件能工作，删除备份
rm $(which mdpress).backup

# 如果新二进制文件损坏，从备份恢复
mv $(which mdpress).backup $(which mdpress)
chmod +x $(which mdpress)
mdpress --version
```

## 相关命令

- [build](build.md) - 从 Markdown 构建输出
- [doctor](doctor.md) - 检查系统就绪情况
- [COMMANDS_zh.md](../COMMANDS_zh.md) - 命令概述
