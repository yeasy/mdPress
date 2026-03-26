# 常见问题解答

## 安装和设置

### 我能不安装 Go 的情况下使用 mdPress 吗？

**问：** 我需要安装 Go 才能使用 mdPress 吗？

**答：** 不需要。预编译的二进制文件适用于 Linux、macOS 和 Windows。从以下位置下载：
- GitHub Releases：https://github.com/yeasy/mdpress/releases
- Homebrew (macOS)：`brew install mdpress`
- 手动：提取二进制文件并添加到 PATH

如果要从源代码构建，你需要 Go 1.25+。

### mdPress 需要网络吗？

**问：** mdPress 需要互联网连接来构建书籍吗？

**答：** 不需要。mdPress 完全离线。所有处理都在你的机器上进行。唯一的例外是：
- 从 GitHub 仓库 URL 构建（需要网络来克隆）
- 对于本地目录，不需要网络

离线构建：

```bash
# 克隆仓库一次
git clone https://github.com/user/book-repo
cd book-repo

# 然后离线构建
mdpress build --format pdf
```

### 我能在 Docker 上使用 mdPress 吗？

**问：** 我如何在 Docker 容器中使用 mdPress？

**答：** 使用官方 Docker 镜像或创建你自己的：

```dockerfile
FROM golang:1.25-alpine
RUN apk add --no-cache chromium font-noto-cjk
RUN go install github.com/yeasy/mdpress@latest
WORKDIR /workspace
ENTRYPOINT ["mdpress"]
```

构建和使用：

```bash
docker build -t mdpress:latest .
docker run -v "$(pwd):/workspace" mdpress:latest build --format pdf
```

## 配置和样式

### 我能使用自定义字体吗？

**问：** 我能将自定义字体应用到我的 PDF 吗？

**答：** 是的，通过自定义 CSS：

```yaml
style:
  custom_css: |
    @font-face {
      font-family: 'MyFont';
      src: url('assets/myfont.ttf') format('truetype');
    }
    body {
      font-family: 'MyFont', sans-serif;
    }
```

将字体文件放在 `assets/` 中并在 CSS 中引用。对于 PDF 输出，使用 TrueType (.ttf) 或 OpenType (.otf) 字体。

### 我如何添加封面图像？

**问：** 我如何用图像创建书籍封面？

**答：** 在 `book.yaml` 中设置封面图像：

```yaml
book:
  title: "My Book"
  cover:
    image: "assets/cover.png"
    background: "#ffffff"  # 可选背景颜色

output:
  cover: true
```

图像要求：
- 格式：PNG、JPEG 或 PDF
- 推荐大小：1200×1600 像素（3:4 宽高比）
- 最大：5 MB

如果图像不完全覆盖页面，使用背景颜色。

### 我如何自定义页眉和页脚？

**问：** 我能添加自定义文本到页眉和页脚吗？

**答：** 是的，使用模板变量：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "{{.Chapter.Title}}"
    right: "{{.PageNum}}"
  footer:
    left: "©{{.Year}} {{.Book.Author}}"
    center: ""
    right: "Page {{.PageNum}}"
```

可用变量：
- `{{.Book.Title}}` - 书籍标题
- `{{.Book.Author}}` - 书籍作者
- `{{.Book.Version}}` - 书籍版本
- `{{.Chapter.Title}}` - 当前章节标题
- `{{.PageNum}}` - 当前页号
- `{{.Date}}` - 构建日期
- 通过插件的自定义变量

参见 [template-variables.md](../reference/template-variables.md) 了解完整引用。

### 我能在页眉/页脚中使用 Markdown 吗？

**问：** 我能在页眉/页脚文本中使用 Markdown 格式吗？

**答：** 不能，仅支持纯文本或模板变量。对于复杂的页眉，使用自定义 CSS：

```yaml
style:
  custom_css: |
    @page {
      @top-center {
        content: "Chapter " var(--chapter-num);
        font-size: 12pt;
        font-weight: bold;
      }
    }
```

## 输出格式

### "html" 和 "site" 格式有什么区别？

**问：** 我应该使用 `--format html` 还是 `--format site`？

**答：** 它们产生不同的输出：

| 格式 | 输出 | 用途 |
|--------|--------|----------|
| `html` | 单个 HTML 文件 | 简单阅读、离线共享 |
| `site` | 多页网站 | 在线托管、搜索引擎、导航 |

```bash
# 单个文件：适合电子邮件、下载
mdpress build --format html
# 输出：output.html (1-5 MB)

# 网站：适合 GitHub Pages、个人服务器
mdpress build --format site
# 输出：_book/ 目录（可导航网站）
```

如果将在线托管，选择 `site`。如果想要单个可共享文件，选择 `html`。

### 我能部署到 GitHub Pages 吗？

**问：** 我如何部署我的书籍到 GitHub Pages？

**答：** 是的。使用 `--format site` 构建并推送到 `gh-pages` 分支：

```bash
# 构建网站
mdpress build --format site

# 创建/切换到 gh-pages 分支
git checkout --orphan gh-pages

# 将 _book 移到根目录
mv _book/* .

# 提交并推送
git add -A
git commit -m "Deploy book"
git push origin gh-pages
```

或使用 GitHub Actions（推荐）：

```yaml
deploy:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: go install github.com/yeasy/mdpress@latest
    - run: mdpress build --format site
    - uses: actions/upload-pages-artifact@v2
      with:
        path: '_book/'
    - uses: actions/deploy-pages@v2
```

网站出现在：`https://username.github.io/repo-name/`

### 支持什么输出格式？

**问：** mdPress 支持哪些输出格式？

**答：**
- `pdf` - 便携式 PDF（默认，需要 Chrome 或 Typst）
- `html` - 单个 HTML 文件
- `site` - 多页网站
- `epub` - 电子书格式（Kindle、Apple Books 等）
- `typst` - Typst 源代码（用于 Typst 排版系统）

```bash
# 一个命令中的所有格式
mdpress build --format pdf --format html --format epub

# 检查支持的格式
mdpress build --help
```

### 我能密码保护我的 PDF 吗？

**问：** 我能将密码保护添加到 PDF 输出吗？

**答：** 不能直接在 mdPress 中。密码保护通常由以下方式处理：

1. **外部工具**（构建后）：
   ```bash
   # Linux/macOS
   qpdf --encrypt user-password owner-password 128 -- output.pdf output-protected.pdf

   # Windows
   pdftk output.pdf output output-protected.pdf user_pw "password" owner_pw "ownerpassword"
   ```

2. **CI/CD 管道**：
   ```yaml
   - run: mdpress build --format pdf
   - run: qpdf --encrypt mypassword mypassword 128 -- output.pdf secure.pdf
   ```

3. **云服务**：将 PDF 上传到具有访问控制的云存储

## 内容和结构

### 我应该如何组织章节？

**问：** 我应该如何在大型书籍中组织章节？

**答：** 在 `book.yaml` 中使用嵌套分段：

```yaml
chapters:
  - title: "Part 1: Basics"
    file: "part1.md"
    sections:
      - title: "Chapter 1"
        file: "ch01.md"
      - title: "Chapter 2"
        file: "ch02.md"
```

或使用 `SUMMARY.md`：

```markdown
# Summary
- [Part 1](part1.md)
  - [Chapter 1](ch01.md)
  - [Chapter 2](ch02.md)
```

参见 [organizing-large-books.md](../best-practices/organizing-large-books.md) 了解详细策略。

### 我能使用 Markdown 包含吗？

**问：** 我能在另一个 Markdown 文件内包含一个吗？

**答：** 不能原生支持。使用以下方法：

1. **单独的章节文件**（推荐）：
   ```yaml
   chapters:
     - title: "Section A"
       file: "section-a.md"
     - title: "Section B"
       file: "section-b.md"
   ```

2. **预处理脚本**：
   ```bash
   # 构建前连接章节
   cat chapter1.md chapter2.md > combined.md
   mdpress build
   ```

3. **构建工具集成**（Makefile）：
   ```makefile
   build:
   	cat intro.md part1/*.md part2/*.md > full-book.md
   	mdpress build
   ```

### 我如何添加脚注？

**问：** 我如何添加脚注或尾注？

**答：** 使用标准 Markdown 脚注语法：

```markdown
This is a sentence with a footnote[^1].

[^1]: Here's the footnote content.
```

对于 PDF 输出，脚注出现在页面底部。对于 HTML，它们变成尾注链接。

### 我能使用 LaTeX/数学方程吗？

**问：** 我能包含数学方程吗？

**答：** 有限的支持。使用这些格式：

**内联数学**（仅 HTML，使用自定义 CSS）：
```markdown
E = mc²
```

**显示方程**（作为图像）：
```markdown
![Equation](assets/equation.png)
```

**HTML 中的 LaTeX**（使用 MathJax 插件）：
```html
<script src="https://polyfill.io/v3/polyfill.min.js?features=es6"></script>
<script id="MathJax-script" async src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script>
```

对于复杂的数学，呈现为图像并嵌入。

## 高级主题

### 我能使用插件吗？

**问：** 我能使用插件扩展 mdPress 吗？

**答：** 是的。插件在构建期间运行：

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
```

创建一个插件（简单示例）：

```go
// plugins/my-plugin/main.go
package main

import (
  "fmt"
  "os"
)

func main() {
  // 读取输入
  input, _ := os.ReadFile("/dev/stdin")

  // 处理
  output := processMarkdown(input)

  // 写输出
  fmt.Println(output)
}
```

参见插件文档了解详细 API。

### 我能使用自定义主题吗？

**问：** 我能创建自定义主题吗？

**答：** 是的。主题是 CSS 文件：

```bash
# 列出内置主题
mdpress themes list

# 显示主题详细信息
mdpress themes show technical

# 创建自定义主题
mkdir -p themes/my-theme
cat > themes/my-theme/style.css << 'EOF'
body {
  font-family: Georgia, serif;
  color: #333;
}
EOF

# 在 book.yaml 中使用
style:
  theme: "./themes/my-theme"
```

### 我能与版本控制集成吗？

**问：** 我如何在 git 中跟踪 PDF 版本？

**答：** 使用 Git LFS（大文件存储）处理二进制文件：

```bash
# 安装 Git LFS
brew install git-lfs

# 跟踪 PDF
git lfs install
git lfs track "*.pdf"
git add .gitattributes

# 现在正常提交 PDF
git add output.pdf
git commit -m "Release v1.0.0"
```

或从 git 排除：

```bash
# .gitignore
output.pdf
output.html
_book/
.mdpress-cache/
```

改为在 CI/CD 中构建 PDF。

### 我能从 GitHub 仓库构建吗？

**问：** 我能从 GitHub 仓库构建书籍吗？

**答：** 是的：

```bash
# 公开仓库
mdpress build https://github.com/user/book-repo

# 私有仓库（需要 GITHUB_TOKEN）
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-book
```

要求：
- 仓库必须包含 `book.yaml` 或 `SUMMARY.md`
- 对于私有仓库，提供具有 `contents:read` 权限的 `GITHUB_TOKEN`

## 故障排除

### 我在哪里找到日志？

**问：** 我如何查看详细的构建日志？

**答：** 使用详细模式：

```bash
mdpress build --verbose --format pdf

# 输出包括：
# [DEBUG] Loading config from book.yaml
# [DEBUG] Found 12 chapters
# [DEBUG] Parsing chapter 1... (125ms)
# [DEBUG] Parsing chapter 2... (98ms)
# ...
```

### 我在哪里报告错误？

**问：** 我应该在哪里报告错误或请求功能？

**答：** GitHub Issues：
- 问题：https://github.com/yeasy/mdpress/issues
- 讨论：https://github.com/yeasy/mdpress/discussions

包括：
- mdPress 版本：`mdpress --version`
- 系统信息：`mdpress doctor`
- 最小重现步骤
- 使用 `--verbose` 标志的错误输出

### 我如何获得帮助？

**问：** 我在哪里可以获得使用 mdPress 的帮助？

**答：**
1. **文档**：https://github.com/yeasy/mdpress
2. **GitHub Issues**：https://github.com/yeasy/mdpress/issues
3. **GitHub Discussions**：https://github.com/yeasy/mdpress/discussions
4. **运行 doctor**：`mdpress doctor`（诊断许多问题）
5. **验证**：`mdpress validate`（检查配置错误）

## 性能和优化

### 我如何加快构建？

**问：** 我的构建很缓慢。我如何使它们更快？

**答：** 参见 [performance.md](../best-practices/performance.md) 了解详细的优化策略。

快速检查清单：
1. 启用缓存（默认）：`mdpress build --format pdf`
2. 开发时使用 HTML：`mdpress serve --format html`
3. 优化图像：保持每个不超过 500 KB
4. 使用 Typst 获得更快的 PDF：`mdpress build --format typst`

典型时间：
- 小型书籍（10 章）：30 秒
- 大型书籍（50 章）：1-2 分钟（启用缓存）

### 我能从一个仓库构建多本书吗？

**问：** 我能在一个仓库中维护多本书吗？

**答：** 是的。创建带有各自 `book.yaml` 的单独目录：

```
repo/
├── book1/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── ch01.md
│   └── ch02.md
├── book2/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── ch01.md
│   └── ch02.md
```

构建每个：

```bash
mdpress build book1/
mdpress build book2/
```

或在 CI/CD 中：

```yaml
strategy:
  matrix:
    book: [book1, book2]

steps:
  - run: mdpress build ${{ matrix.book }}
```

## 许可和分发

### 我能商业使用 mdPress 吗？

**问：** 我能为商业项目使用 mdPress 吗？

**答：** 是的。mdPress 是 MIT 许可证下的开源，允许商业使用。

### 我能出售用 mdPress 创建的书吗？

**问：** 我能出售由 mdPress 生成的 PDF 吗？

**答：** 是的，但：
- 该工具本身是免费的开源
- 你拥有你创建的输出
- 你可以出售生成的书籍
- 你必须尊重你包含的内容的版权

## 更多问题？

如果你的问题在这里没有得到回答：

1. 查看完整文档
2. 搜索 GitHub Issues
3. 在 GitHub Discussions 提问
4. 运行 `mdpress doctor` 和 `mdpress validate` 获取诊断

mdPress 社区友好且乐于帮助！
