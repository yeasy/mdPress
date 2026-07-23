# 使用 mdpress doctor

`mdpress doctor` 报告 mdPress 在你的环境中发现了什么，以及目标目录里的项目能否被加载。构建之前跑一次；在 CI 中请配合 `--strict` 作为门禁。

```bash
mdpress doctor                 # 检查当前目录
mdpress doctor /path/to/book   # 检查其他目录
```

## 真实输出

以下是在环境完好的机器上、对一个脚手架项目运行的完整输出：

```
  mdpress Environment Check
  ──────────────────────────────────────────────────

  ✓ Platform: darwin/arm64
  ✓ Go version: go1.26.5
  ✓ Runtime cache: /private/tmp/mdpress-cache
  ✓ Chromium/Chrome is available
  ✓ Typst is available: typst 0.15.0 (unknown commit)
  ✓ Go version 1.26.5 (>= 1.26)
  ✓ Git is available
  ✓ Network connectivity to github.com available
  ✓ Disk space available
  ✓ CJK fonts available: (system CJK fonts detected)
  ✓ PlantUML not needed (no diagrams detected)


  Project Check
  ──────────────────────────────────────────────────
  ✓ Detected book.yaml
  ⚠ SUMMARY.md not found
  ⚠ LANGS.md not found
  ✓ Config loads successfully: my-book (4 top-level chapters)
  ✓ Markdown chapter links resolve within the build graph
```

`SUMMARY.md not found` 与 `LANGS.md not found` 只是提示：用 `book.yaml` 配置的项目并不需要这两个文件。

## 各项检查的含义

| 检查项 | 含义 |
| --- | --- |
| Platform | 当前二进制的 `GOOS/GOARCH` |
| Go version | 编译 mdPress 所用的 Go 运行时版本（仅供参考） |
| Runtime cache | 解析缓存所在目录；`--no-cache` 关闭缓存时会打印警告 |
| Chromium/Chrome | 默认 PDF 后端。自动探测，或取自 `MDPRESS_CHROME_PATH` |
| Typst | `--format typst` 使用的备用 PDF 后端 |
| Go version >= 1.26 | 仅在通过 `go install` 安装时有意义 |
| Git | 从 GitHub URL 构建时需要 |
| Network connectivity | 针对 github.com 的可达性探测，用于远程源 |
| Disk space | 输出目录所在分区的可用空间 |
| CJK fonts | PDF 输出中的中日韩文字需要 |
| PlantUML | 项目中没有 PlantUML 围栏时只报告 "not needed" |
| Plugins | `book.yaml` 中 `plugins:` 的每一项都必须存在且可执行 |
| book.yaml / SUMMARY.md / LANGS.md | 项目中存在哪些文件 |
| Config loads | `config.Load`（或基于 `SUMMARY.md` 的自动发现）是否成功 |
| Markdown chapter links | 相对 `.md` 链接是否指向构建图内的文件 |

### PDF 后端检查

PDF 只需要 Chromium/Chrome **或** Typst 之一，因此只有两者都缺失才算错误：

- 缺 Chromium、有 Typst → 警告：`Chromium/Chrome is unavailable — use --format typst for PDF output instead`
- 两者都缺 → 错误：`No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)`

### 构建图之外的链接

```
  ⚠ Detected 1 Markdown link(s) outside the build graph
    - ../outside.md (from one.md)
```

链接目标不属于 mdPress 会构建的章节，因此在 site 和 HTML 产物里它会是一个死链。请把该文件加入章节，或修正链接。

## 退出码与 `--strict`

**不加 `--strict` 时，`mdpress doctor` 永远退出 0**，即使打印了 `✗` 行。只跑 `mdpress doctor` 的 CI 步骤实际上没有任何门禁作用。

```bash
mdpress doctor --strict
```

`--strict` 在任何 error 级检查失败时以非零状态退出，并打印：

```
Error: doctor found 1 error-level issue(s) (run without --strict to ignore)
```

error 级问题包括：完全没有 PDF 后端、`book.yaml` 加载失败、自动发现失败、磁盘空间不足、插件条目损坏。警告（缺少 CJK 字体、没有 `SUMMARY.md`、构建图外的链接）不影响退出码。

## 报告

```bash
mdpress doctor --report report.json
mdpress doctor --report report.md
```

由扩展名决定格式。其他扩展名会在检查已经打印完之后报错：

```
Error: failed to write doctor report: unsupported report extension: .txt (use .json or .md)
```

### JSON 报告

JSON 报告是一个扁平对象，键名为 `snake_case`。以下是上面那次健康运行的完整报告：

```json
{
  "platform": "darwin/arm64",
  "go_version": "go1.26.5",
  "cache_dir": "/private/tmp/mdpress-cache",
  "cache_disabled": false,
  "chromium_available": true,
  "typst_available": true,
  "typst_version": "typst 0.15.0 (unknown commit)",
  "cjk_fonts_available": true,
  "plantuml_available": false,
  "plantuml_needed": false,
  "go_version_check": "go1.26.5",
  "git_available": true,
  "network_available": true,
  "disk_space_gb": 62.70961380004883,
  "disk_space_ok": true,
  "plugins_valid": true,
  "book_yaml_found": true,
  "summary_found": false,
  "langs_found": false,
  "project_loadable": true,
  "project_title": "my-book",
  "top_level_chapters": 4
}
```

另有三个键只在有内容时才出现：

| 键 | 类型 | 出现条件 |
| --- | --- | --- |
| `plugin_count` | number | 项目声明了插件 |
| `warnings` | string 数组 | 记录了任何警告或错误信息 |
| `unresolved_markdown_links` | `{"Source", "Target"}` 数组 | 存在指向构建图之外的链接 |

`book.yaml` 无法加载的项目：

```json
{
  "book_yaml_found": true,
  "project_loadable": false,
  "warnings": [
    "Failed to load book.yaml: config validation failed: chapter validation failed: chapter 1 references a missing file: nope.md (paths are relative to book.yaml)"
  ]
}
```

存在构建图之外链接的项目：

```json
{
  "unresolved_markdown_links": [
    {
      "Source": "one.md",
      "Target": "../outside.md"
    }
  ]
}
```

注意 `warnings` 把警告和 error 级问题混在一起；要区分两者，唯一可靠的方式是看 `--strict` 的退出码。

### 用 jq 解析

```bash
mdpress doctor --report report.json

jq '.chromium_available' report.json        # 这台机器能用默认后端出 PDF 吗？
jq '.project_loadable' report.json          # book.yaml 能加载吗？
jq -r '.warnings[]? // empty' report.json   # 列出所有警告（若有）
jq -r '.unresolved_markdown_links[]? | "\(.Source) -> \(.Target)"' report.json
```

### Markdown 报告

`--report report.md` 把同样的字段写成一个列表，适合作为 CI 产物上传：

```markdown
# mdpress Doctor Report

- Platform: darwin/arm64
- Go version: go1.26.5
- Go version check: go1.26.5
- Cache disabled: false
- Cache dir: /private/tmp/mdpress-cache
- Chromium available: true
- Typst available: true
- Typst version: typst 0.15.0 (unknown commit)
- CJK fonts available: true
- Git available: true
- Network connectivity: true
- Disk space available: 62.71 GB
- Disk space OK: true
- PlantUML needed: false
- PlantUML available: false
- Plugins valid: true
- book.yaml found: true
- SUMMARY.md found: false
- LANGS.md found: false
- Project loadable: true
- Project title: my-book
- Top-level chapters: 4
```

## 在 CI 中使用

务必加上 `--strict`——否则这个步骤永远不会失败。

### GitHub Actions

```yaml
- name: Environment and project readiness
  run: mdpress doctor --strict

- name: Upload doctor report
  if: always()
  run: mdpress doctor --report doctor-report.md
- uses: actions/upload-artifact@v4
  if: always()
  with:
    name: doctor-report
    path: doctor-report.md
```

### GitLab CI

```yaml
doctor:
  stage: validate
  script:
    - mdpress doctor --strict
    - mdpress doctor --report doctor-report.md
  artifacts:
    paths:
      - doctor-report.md
    expire_in: 30 days
```

### 构建前脚本

```bash
#!/bin/bash
set -e

mdpress doctor --strict   # 出现 error 级问题时，set -e 会在这里终止
mdpress validate
mdpress build --format pdf
```

## 常见结果与修复

### 没有可用的 PDF 后端

```
✗ No PDF backend available: Chromium/Chrome and Typst are both missing (PDF output will fail)
```

安装其中之一，或改用不需要它们的格式：

```bash
# macOS
brew install chromium        # 或：brew install typst
# Ubuntu/Debian
sudo apt-get install chromium-browser

# Chromium 装在非标准路径时
export MDPRESS_CHROME_PATH=/path/to/chrome

# 或者干脆不用 PDF 后端
mdpress build --format site,html,epub
```

### 未检测到 CJK 字体

```
⚠ No CJK fonts detected — PDF output for Chinese/Japanese/Korean text may show blank squares
```

纯英文书可以忽略。否则：

```bash
sudo apt-get install fonts-noto-cjk   # Ubuntu/Debian
apk add font-noto-cjk                 # Alpine
```

### book.yaml 加载失败

```
✗ Failed to load book.yaml: config validation failed: chapter validation failed: chapter 1 references a missing file: nope.md (paths are relative to book.yaml)
```

章节路径相对 `book.yaml` 解析，而不是相对 shell 的当前目录。`mdpress validate` 会给出更详细的报告。

### 目标目录里没有可直接构建的 book.yaml 或 SUMMARY.md

```
⚠ No directly buildable book.yaml or SUMMARY.md found in the target directory
```

说明 doctor 指向的目录两者皆无。`mdpress init` 可以从目录里已有的 Markdown 文件生成一份 `book.yaml`。

## doctor 不检查什么

doctor 检查环境以及配置能否加载。它不会：

- 深入校验 Markdown 与链接目标——那是 `mdpress validate` 的职责
- 验证图片是否存在或尺寸是否合适
- 真的跑一次构建

完整的构建前检查是这三条：

```bash
mdpress doctor --strict
mdpress validate
mdpress build --format html
```

构建期错误的修复方法见 [common-issues.md](common-issues.md)。
