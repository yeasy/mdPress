# 使用 mdpress doctor

`mdpress doctor` 命令检查你的系统环境和项目配置以识别潜在问题。使用它在问题出现前诊断构建问题。

## 快速开始

```bash
mdpress doctor

# 示例输出：
# Checking environment...
# ✓ Platform: Linux
# ✓ Go: 1.26.0
# ✓ Chrome/Chromium: /usr/bin/chromium
# ✓ CJK fonts: Noto Sans CJK SC
# ✓ PlantUML: not installed (optional)
# ✓ Cache directory: /tmp/mdpress-cache
# ✗ book.yaml: missing

# Doctor completed: 1 warning
```

## Doctor 检查的内容

### 环境检测

**平台和操作系统：**
```
✓ Platform: Linux (ubuntu-latest)
✓ OS version: 6.8.0
```

识别你的操作系统以提供系统特定的指导。

**Go 安装：**
```
✓ Go: 1.26.0 installed
✓ go binary: /usr/local/go/bin/go
```

mdPress 正常工作所需。如果缺失，doctor 建议安装。

**Chrome/Chromium：**
```
✓ Chrome: /usr/bin/chromium (version 120.0)
```

PDF 渲染所需。Doctor 自动检测或使用 `MDPRESS_CHROME_PATH` 环境变量。

**CJK 字体：**
```
✓ CJK fonts detected:
  - Noto Sans CJK SC (Chinese Simplified)
  - Noto Sans CJK JP (Japanese)
```

对于包含中文、日文或韩文文本的书籍是可选但推荐的。

**PlantUML：**
```
✗ PlantUML: not installed
  → Optional for diagram rendering
  → Install from: http://plantuml.com/download
```

用于图表支持的可选依赖。

### 项目配置

**配置文件：**
```
✓ book.yaml found
✓ Config valid (12 chapters, 24 images)
```

检查 `book.yaml` 并验证基本语法。

**章节文件：**
```
✓ All 12 chapter files exist
✓ Total chapters: 12 (4000 lines)
```

验证配置中引用的所有章节存在于磁盘上。

**图像资源：**
```
✓ 24 images found
  - PNG: 18 (5.2 MB total)
  - SVG: 4 (150 KB total)
  - JPEG: 2 (800 KB total)
```

列出所有嵌入的图像和总大小。

**链接验证：**
```
✓ Internal links checked
✓ No broken cross-references
```

验证章节间的相对链接。

### 缓存和构建就绪

**缓存目录：**
```
✓ Cache: /tmp/mdpress-cache (2.4 MB)
  - 12 cached chapters
  - Last updated: 2 hours ago
```

显示缓存位置、大小和新鲜度。

**构建就绪：**
```
✓ Ready to build
  Recommended format: pdf (Chrome available)
```

最终系统是否准备好构建的判断。

## 运行 Doctor

### 基础检查

```bash
mdpress doctor
```

检查当前目录的配置和环境。

### 检查特定目录

```bash
mdpress doctor /path/to/book
```

诊断不同目录中的项目。

### 生成报告

```bash
# JSON 格式（用于解析）
mdpress doctor --report report.json

# Markdown 格式（用于文档）
mdpress doctor --report report.md

# 纯文本（默认，输出到终端）
mdpress doctor
```

### 示例：生成 Markdown 报告

```bash
mdpress doctor --report DOCTOR_REPORT.md
```

报告包括：
- 系统信息
- 已安装的工具和版本
- 配置验证结果
- 修复建议

## 解释 Doctor 输出

### 绿色复选标记（✓）

一切工作正常：

```
✓ Platform: Linux
✓ Go: 1.26.0
✓ Chrome: /usr/bin/chromium
```

无需采取行动。

### 黄色警告（!）

可选依赖缺失，但不关键：

```
! PlantUML: not installed
  → This is optional
  → Diagram rendering will be skipped
  → To enable: install PlantUML from http://plantuml.com
```

PlantUML 是可选的。书籍不需要它就能构建，但图表不会呈现。

### 红色错误（✗）

阻止构建的关键问题：

```
✗ Chrome: not found
  → Required for PDF rendering
  → To fix: Install Chrome or set MDPRESS_CHROME_PATH
  → Install: https://www.google.com/chrome/
```

构建前必须修复错误。

## 常见 Doctor 发现和修复

### Chrome/Chromium 未找到

**发现：**
```
✗ Chrome: not found
  → Required for PDF rendering with --format pdf
```

**修复：**

```bash
# 安装 Chrome/Chromium
# Linux
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Windows
# 从 https://www.google.com/chrome/ 下载

# 然后设置路径（如果未自动检测）
export MDPRESS_CHROME_PATH=/path/to/chrome
mdpress build --format pdf
```

或使用 Typst 作为替代：

```bash
mdpress build --format typst
```

### 缺失 CJK 字体

**发现：**
```
! CJK fonts: not detected
  → Books with Chinese, Japanese, or Korean text need CJK fonts
  → Optional if your book is English-only
```

**修复（如需要）：**

```bash
# Ubuntu/Debian
sudo apt-get install fonts-noto-cjk

# macOS
brew install font-noto-sans-cjk

# Alpine
apk add font-noto-cjk
```

然后重建：

```bash
mdpress build --format pdf --no-cache
```

### 缺失 book.yaml

**发现：**
```
✗ book.yaml: not found
  → Configuration file required
  → Create one with: mdpress init
```

**修复：**

```bash
# 生成模板
mdpress init

# 或手动创建
cat > book.yaml << 'EOF'
book:
  title: "My Book"
  author: "Your Name"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"
EOF

# 然后构建
mdpress build
```

### 破损的章节引用

**发现：**
```
✗ Config validation failed
  Chapter 1 references missing file: chapters/ch01.md
```

**修复：**

```bash
# 检查存在的文件
ls -la

# 更新 book.yaml 为正确路径
vim book.yaml

# 验证修复
mdpress validate
```

### 破损的图像引用

**发现：**
```
! Image not found: ../assets/diagram.png
  → Referenced in: chapters/ch02.md
```

**修复：**

```bash
# 检查图像是否存在
ls -la assets/

# 如果缺失，添加图像：
cp /path/to/diagram.png assets/

# 如果路径错误，更新 Markdown：
# 更改：![Diagram](diagram.png)
# 为：![Diagram](../assets/diagram.png)

# 验证
mdpress validate
```

## 在 CI/CD 中使用 Doctor

### GitHub Actions

```yaml
- name: Run doctor check
  run: mdpress doctor

- name: Generate doctor report
  if: always()
  run: mdpress doctor --report doctor-report.md

- name: Upload report as artifact
  uses: actions/upload-artifact@v4
  with:
    name: doctor-report
    path: doctor-report.md
```

### GitLab CI

```yaml
doctor:
  stage: validate
  script:
    - mdpress doctor
    - mdpress doctor --report doctor-report.md
  artifacts:
    paths:
      - doctor-report.md
    expire_in: 30 days
  allow_failure: true  # 警告不会导致构建失败
```

### 构建前验证

```bash
#!/bin/bash
set -e

echo "Running doctor check..."
mdpress doctor

# 如果 doctor 失败，停止构建
if [ $? -ne 0 ]; then
  echo "Doctor check failed. Please fix issues above."
  exit 1
fi

echo "Doctor check passed. Building..."
mdpress build --format pdf
```

## Doctor 报告字段

使用 `--report report.json` 时：

```json
{
  "platform": "Linux",
  "osVersion": "6.8.0",
  "go": {
    "installed": true,
    "version": "1.26.0",
    "path": "/usr/local/go/bin/go"
  },
  "chrome": {
    "found": true,
    "path": "/usr/bin/chromium",
    "version": "120.0.0.0"
  },
  "cjkFonts": ["Noto Sans CJK SC", "Noto Sans CJK JP"],
  "plantUml": {
    "installed": false,
    "recommended": false
  },
  "project": {
    "configValid": true,
    "chaptersCount": 12,
    "imagesCount": 24,
    "totalImageSize": "6.2 MB"
  },
  "recommendations": [
    "Install PlantUML for diagram support (optional)"
  ]
}
```

以编程方式解析：

```bash
# 获取 Chrome 版本
mdpress doctor --report report.json && jq '.chrome.version' report.json

# 检查项目是否就绪
mdpress doctor --report report.json && jq '.project.configValid' report.json

# 列出建议
mdpress doctor --report report.json && jq '.recommendations[]' report.json
```

## 持续监控

### 健康检查脚本

```bash
#!/bin/bash

echo "=== mdPress Health Check ==="
echo ""

echo "1. Running doctor..."
mdpress doctor --report /tmp/doctor.json

if ! jq empty /tmp/doctor.json 2>/dev/null; then
  echo "✗ Doctor check failed"
  exit 1
fi

echo ""
echo "2. Validating configuration..."
if ! mdpress validate; then
  echo "✗ Validation failed"
  exit 1
fi

echo ""
echo "3. Test build (HTML format, fastest)..."
if ! mdpress build --format html --output /tmp/test.html; then
  echo "✗ Build failed"
  exit 1
fi

echo ""
echo "✓ All checks passed!"
echo "Ready to build PDF: mdpress build --format pdf"
```

定期运行：

```bash
# 每周健康检查
0 0 * * 0 /path/to/health-check.sh
```

## Doctor 的局限性

Doctor 检查**环境**和**配置**，但不进行：

- Markdown 语法验证（使用 `mdpress validate`）
- 检查语法或拼写错误
- 验证图像质量或分辨率
- 检测构建期间的性能问题

为获得全面验证：

```bash
# 完整验证
mdpress validate      # 配置和结构
mdpress doctor        # 环境就绪
mdpress build --format html  # 尝试实际构建
```

## Doctor 后的后续步骤

如果 doctor 报告全部绿色：

```bash
# 尝试构建
mdpress serve       # 实时重载预览
mdpress build       # 构建 PDF

# 如果构建失败，尽管 doctor 报告绿色：
mdpress build --verbose  # 启用调试输出
```

如果 doctor 报告警告：

- **可选功能**：构建有效，但某些功能不可用
- **错误**：在尝试构建前修复

参见 [common-issues.md](common-issues.md) 了解每种错误类型的详细修复说明。
