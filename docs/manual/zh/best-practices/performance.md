# 性能优化技巧

mdPress 为速度而设计，为大型书籍提供内置优化。本指南涵盖性能调优和最佳实践，以最大化构建速度并最小化资源使用。

## 理解构建性能

mdPress 构建包含三个主要阶段：

1. **解析**：将 Markdown 转换为内部表示（跨章节并行化）
2. **处理**：应用转换和渲染（启用缓存）
3. **输出**：生成 PDF、HTML 或其他格式（格式特定优化）

对于大型书籍（50+ 章节），解析通常是瓶颈。mdPress 使用所有可用 CPU 核心自动并行化此阶段。

## 缓存策略

### 构建缓存如何工作

mdPress 维护一个 `.mdpress-cache/` 目录（默认位置），其中存储：

- **章节哈希**：章节内容的 MD5 校验和
- **已编译内容**：未变章节的预处理 Markdown
- **元数据**：构建状态和时间戳

在重建时：

```
首次构建：    解析所有章节 → 缓存所有 → 输出格式
第二次构建：   检查哈希 → 重用缓存的章节 → 输出格式
已更改文件：  重新解析已更改的章节 → 更新缓存 → 输出
```

这将大型书籍的重建时间从数分钟减少到数秒（只进行少量更改）。

### 启用缓存（默认行为）

缓存默认启用。无需配置：

```bash
# 首次构建：编译所有章节
mdpress build --format pdf
# 输出：对 50 章节耗时 2 分钟

# 第二次构建：重用缓存
mdpress build --format pdf
# 输出：耗时 5 秒（无更改）
```

### 查看缓存目录

```bash
# 默认缓存位置
ls -la .mdpress-cache/

# 更改缓存目录
mdpress build --cache-dir /tmp/mdpress-cache --format pdf
```

缓存目录安全删除，下次构建时将重新生成。

### 强制完全重建

当需要重建所有内容时（例如主题或配置更改）：

```bash
# 完全跳过缓存
mdpress build --no-cache --format pdf

# 用例：更改 book.yaml 样式设置后
mdpress build --no-cache --format pdf --output my-book.pdf
```

**何时使用 `--no-cache`**：
- 更改 `book.yaml` 中的 `style` 设置后
- 更新主题或自定义 CSS 后
- 构建似乎过时或不正确
- 确保 CI/CD 中的可重现构建

**何时不使用 `--no-cache`**：
- 在正常开发中（缓存加快迭代）
- 对于单章节构建（缓存开销可忽略）
- 不是 CI/CD 中的每次构建（战略使用）

## 开发期间的增量构建

使用 `mdpress serve` 获得最快的反馈循环：

```bash
# 启动实时预览服务器
mdpress serve

# 编辑章节并保存
# → 服务器自动重建受影响的章节
# → 浏览器预览更新（无需手动重建）
```

优点：
- 仅重新编译已修改的章节
- 跨更改保留缓存
- 浏览器中的实时重载提供即时反馈
- 保存后无需 CLI 调用

工作流示例：

```bash
# 终端 1：启动服务器
$ mdpress serve
[INFO] Serving at http://localhost:9000
[INFO] Watching for changes...

# 终端 2：编辑章节
$ vim chapters/ch03.md
# ... 进行更改，保存

# 服务器自动重建（2 秒）
# 浏览器显示更新的内容
```

## 并行章节处理

### 自动并行化

mdPress 自动检测系统的 CPU 核心数并并行处理章节：

```bash
# 在 4 核系统上：同时处理 4 个章节
# 在 16 核系统上：同时处理 16 个章节
mdpress build --format pdf

# 无需配置，自动扩展
```

不同系统上的性能扩展示例：

| 系统 | 核心 | 50 章节 | 时间 |
|--------|-------|------------|------|
| 笔记本 | 4 | 首次构建 | 90 秒 |
| 笔记本 | 4 | 缓存重建 | 3 秒 |
| 服务器 | 16 | 首次构建 | 25 秒 |
| 服务器 | 16 | 缓存重建 | 2 秒 |

### 控制并行性

目前，mdPress 不暴露并行性配置，总是使用所有可用核心。要限制并行性，在资源受限的容器中运行 mdPress：

```bash
# Docker：限制为 2 核
docker run --cpus=2 myimage mdpress build

# cgroups v2 (Linux)：限制为 2 核
systemd-run --scope -p CPUQuota=200% mdpress build
```

## 图像优化

### 自动优化

mdPress 自动为输出优化图像：

- **HTML/网站输出**：嵌入图像并使用延迟加载（减少初始页面加载）
- **PDF 输出**：以屏幕或打印的最优分辨率嵌入图像
- **ePub 输出**：嵌入设备优化的图像大小

### 手动图像优化

在将图像添加到书籍之前进行优化：

```bash
# PNG 优化（无损）
optipng -o2 diagram.png

# JPEG 优化
jpegoptim --max=85 screenshot.jpg

# SVG 优化（首选用于图表）
# SVG 可无限缩放，文件大小小
# 用于流程图、架构图、图标
```

推荐的图像大小：

| 类型 | 格式 | 最大大小 | 用途 |
|------|--------|----------|----------|
| 图表 | SVG | 50 KB | 流程图、架构图、系统图 |
| 截图 | PNG | 500 KB | 用户界面演示 |
| 照片 | JPEG | 300 KB | 文档照片、封面 |
| 图标 | SVG | 10 KB | 内联图标、标注 |

### 图像存储最佳实践

```
assets/
├── diagrams/
│   ├── architecture.svg      (首选 SVG)
│   ├── flow-chart.svg
│   └── deployment.svg
├── screenshots/
│   ├── interface-main.png    (PNG，不超过 500 KB)
│   └── setup-wizard.png
└── photos/
    └── team-photo.jpg        (JPEG，总计不超过 2 MB)
```

总资源目录应保持在 50 MB 以内以实现合理的快速构建。

## 输出格式性能

### PDF vs. HTML vs. ePub

**PDF 输出**（默认，`--format pdf`）：
- 需要 Chrome/Chromium 或 Typst
- 较慢（50 章节耗时 10-30 秒）
- 生成单个可移植文件
- 最适合打印和分发

**HTML 输出**（`--format html`）：
- 最快（50 章节耗时 5-10 秒）
- 生成单个 HTML 文件
- 适合基础文档
- 无外部依赖

**网站输出**（`--format site`）：
- 中等速度（50 章节耗时 8-15 秒）
- 生成多页网站
- 适合搜索引擎和导航
- 最适合在线文档

**ePub 输出**（`--format epub`）：
- 中等速度（50 章节耗时 8-15 秒）
- 生成可移植的电子书文件
- 适合电子阅读器（Kindle、Apple Books）
- 需要格式化验证

### 何时使用每种格式

在开发和验证期间：

```bash
# 最快的反馈：使用 HTML 输出
mdpress serve --format html

# 然后切换到最终格式进行最终构建
mdpress build --format pdf       # 用于分发
mdpress build --format site      # 用于在线文档
```

## 用于轻量级 PDF 的 Typst 后端

Typst 后端提供了 Chrome/Chromium 的替代品，更快更轻：

```bash
# 首先安装 Typst
# 参见 https://typst.app 获取安装说明

# 使用 Typst 构建
mdpress build --format typst

# 输出：生成本地 PDF
# 优点：更快、无需 Chromium、专业输出
```

性能比较：

| 后端 | 速度 | 大小 | 依赖 | 质量 |
|---------|-------|------|--------------|---------|
| Chrome/Chromium | 30 秒 | 5 MB 输出 | 浏览器引擎 | 专业 |
| Typst | 15 秒 | 4 MB 输出 | Typst CLI | 专业 |

在以下情况下使用 Typst：
- 未安装 Chrome 的系统
- 资源受限的环境（CI/CD 运行器）
- 更快的 PDF 生成
- 最小依赖

## 缓存目录管理

### 默认缓存位置

```bash
# Linux/macOS
~/.cache/mdpress/

# Windows
%USERPROFILE%\AppData\Local\mdpress\

# macOS (Homebrew)
~/Library/Caches/mdpress/
```

使用 `--cache-dir` 覆盖：

```bash
mdpress build --cache-dir /tmp/mdpress-cache --format pdf
```

### 缓存大小

缓存随书籍大小增长。典型大小：

- 10 章节：2-5 MB
- 50 章节：10-20 MB
- 100 章节：20-40 MB

### 清理缓存

```bash
# 删除缓存目录（将在下次构建时重建）
rm -rf ~/.cache/mdpress/

# 或指定自定义位置
rm -rf /tmp/mdpress-cache/

# mdPress 将在下次构建时重新生成
mdpress build --format pdf
```

任何时候删除缓存都是安全的。

## CI/CD 性能

### GitHub Actions 示例

```yaml
name: Build Book
on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      # 缓存 Go 模块
      - uses: actions/cache@v3
        with:
          path: ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      # 缓存 mdPress 缓存目录
      - uses: actions/cache@v3
        with:
          path: .mdpress-cache
          key: ${{ runner.os }}-mdpress-${{ hashFiles('**/*.md') }}

      # 安装 mdPress
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go install github.com/yeasy/mdpress@latest

      # 构建书籍
      - run: mdpress build --format pdf

      # 上传工件
      - uses: actions/upload-artifact@v3
        with:
          name: pdf-output
          path: output.pdf
```

关键优化：
- 缓存 Go 模块以避免重新下载依赖
- 缓存 `.mdpress-cache/` 以重用已解析的章节
- 仅在文件更改时重建（使用 `hashFiles`）

### 构建时间目标

在 CI/CD 中的目标时间表：

- **小型书籍（1-10 章节）**：< 30 秒
- **中型书籍（11-50 章节）**：< 1 分钟
- **大型书籍（50+ 章节）**：1-3 分钟（启用缓存）

如果构建超过这些目标：
1. 验证缓存工作正常（检查 CI 日志中的缓存命中）
2. 检查图像大小（运行 `mdpress validate`）
3. 考虑拆分非常大的书籍
4. 对更快的输出使用 `--format html` 或 `--format typst`

## 性能分析和诊断

### 详细输出

启用详细的计时信息：

```bash
mdpress build --verbose --format pdf

# 输出包括：
# [INFO] Parsing chapters...
# [DEBUG] Chapter 1: 150ms
# [DEBUG] Chapter 2: 120ms
# [DEBUG] Chapter 3: 180ms
# ...
# [INFO] Total parsing: 4.2s
```

### 识别缓慢的章节

大型章节需要较长的解析时间。如果构建缓慢：

```bash
# 检查非常大的章节
find . -name "*.md" -size +1M

# 按行数排序
wc -l *.md | sort -n
```

## 性能检查清单

- [ ] 开发期间使用 `mdpress serve`（即时反馈）
- [ ] 启用缓存（默认，在重建时提供 10-100 倍加速）
- [ ] 开发期间使用 `--format html` 以加速输出
- [ ] 仅对最终构建切换到 `--format pdf`
- [ ] 优化图像（每个截图最多 500 KB，图表首选 SVG）
- [ ] 保持总资源不超过 50 MB
- [ ] 在 CI/CD 中使用 `--cache-dir` 并跨构建缓存
- [ ] 运行 `mdpress validate` 以尽早捕获破损链接
- [ ] 对于大型书籍（100+ 章节），考虑拆分成多个书籍
- [ ] 查看详细输出（`--verbose`）以识别瓶颈

## 常见性能问题

### 构建耗时 5+ 分钟

**可能原因**：缓存禁用或丢失

```bash
# 解决方案：启用缓存
mdpress build --format pdf
# 首次构建耗时，重建快速
```

### 构建之间性能不一致

**可能原因**：缓存目录在不同位置

```bash
# 解决方案：使用一致的缓存目录
mdpress build --cache-dir ~/.mdpress-cache --format pdf
```

### PDF 输出极其缓慢（30+ 秒）

**可能原因**：Chrome 渲染开销

```bash
# 解决方案 1：使用 Typst 代替
mdpress build --format typst

# 解决方案 2：开发时使用 HTML 输出
mdpress serve --format html
```

### 大型书籍上内存不足

**可能原因**：并行处理过多章节

```bash
# 解决方案：使用内存限制的容器运行
docker run --memory=2g myimage mdpress build --format pdf
```

## 总结

文档的最快工作流是：

1. **开发**：使用 `mdpress serve --format html`（最快反馈）
2. **验证**：使用 `mdpress validate`（尽早捕获问题）
3. **最终构建**：使用 `mdpress build --format pdf`（启用缓存）
4. **CI/CD**：跨构建缓存 `.mdpress-cache/`

通过适当的缓存和格式选择，mdPress 从单章节文档高效扩展到 200 页技术书籍。
