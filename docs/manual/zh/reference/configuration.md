# 配置参考

`book.yaml` 所有字段和选项的完整参考。

## 配置文件结构

`book.yaml` 文件分为四个主要分段：

```yaml
book:        # 书籍元数据（标题、作者等）
chapters:    # 章节定义和结构
style:       # 视觉样式和主题
output:      # 输出格式设置
plugins:     # 构建插件
```

## book 分段

书籍元数据和常规信息。

### book.title

**类型：** string
**默认值：** "Untitled Book"
**必需：** Yes

书籍标题，用于 PDF 元数据、页眉和输出。

```yaml
book:
  title: "Advanced Python Programming"
```

### book.subtitle

**类型：** string
**默认值：** ""
**必需：** No

可选副标题，在封面和元数据中显示。

```yaml
book:
  subtitle: "A Deep Dive into Modern Python"
```

### book.author

**类型：** string
**默认值：** ""
**必需：** No

书籍作者姓名，用于 PDF 元数据和页脚。

```yaml
book:
  author: "Jane Doe"
  # 或多个作者：
  author: "Jane Doe, John Smith"
```

### book.version

**类型：** string
**默认值：** "1.0.0"
**必需：** No

遵循语义版本控制的版本字符串（major.minor.patch）。

```yaml
book:
  version: "2.1.0"
```

### book.language

**类型：** string
**默认值：** "zh-CN"
**必需：** No

语言代码（ISO 639-1 格式）。影响字体选择和文字方向。

```yaml
book:
  language: "en"      # 英文
  language: "zh-CN"   # 中文（简体）
  language: "ja"      # 日文
  language: "ko"      # 韩文
```

### book.description

**类型：** string
**默认值：** ""
**必需：** No

用于元数据和 HTML meta 标签的书籍描述。

```yaml
book:
  description: "Learn advanced Python techniques and best practices"
```

### book.cover

**类型：** object
**默认值：** `{}`
**必需：** No

PDF 和其他格式的封面配置。

#### book.cover.image

封面图像文件的路径。

```yaml
book:
  cover:
    image: "assets/cover.png"
```

要求：
- 格式：PNG、JPEG 或 PDF
- 推荐大小：1200×1600 像素（3:4 宽高比）
- 最大：5 MB

#### book.cover.background

**类型：** string
**默认值：** "#ffffff"

如果图像不填满页面，则使用背景颜色。

```yaml
book:
  cover:
    image: "assets/cover.png"
    background: "#1a1a2e"
```

## chapters 分段

章节结构和文件引用。

### chapters

**类型：** chapter 定义的数组
**默认值：** 从 SUMMARY.md 自动检测
**必需：** Yes（如果没有 SUMMARY.md）

具有可选嵌套分段的章节列表。

```yaml
chapters:
  - title: "Chapter 1: Introduction"
    file: "ch01.md"
  - title: "Chapter 2: Basics"
    file: "ch02.md"
    sections:
      - title: "2.1: Setup"
        file: "ch02-setup.md"
      - title: "2.2: Configuration"
        file: "ch02-config.md"
```

### chapters[].title

**类型：** string
**必需：** Yes

章节或分段标题，在目录和导航中显示。

### chapters[].file

**类型：** string
**必需：** Yes

Markdown 文件的路径，相对于 `book.yaml` 位置。

```yaml
chapters:
  - title: "Introduction"
    file: "intro.md"           # 同一目录
  - title: "Chapter 1"
    file: "chapters/ch01.md"   # 子目录
```

### chapters[].sections

**类型：** array
**默认值：** `[]`
**必需：** No

嵌套的章节或分段，允许多级层次结构。

```yaml
chapters:
  - title: "Part One"
    file: "part1.md"
    sections:
      - title: "Chapter 1"
        file: "ch01.md"
      - title: "Chapter 2"
        file: "ch02.md"
        sections:
          - title: "Section 2.1"
            file: "ch02-sec1.md"
```

## style 分段

视觉样式、主题、字体和外观。

### style.theme

**类型：** string
**默认值：** "technical"

内置主题或自定义主题的路径。

可用主题：
- `technical` - 专业、代码友好（默认）
- `elegant` - 精致、基于衬线字体
- `minimal` - 清洁、极简

```yaml
style:
  theme: "technical"
  # 或自定义主题路径：
  theme: "./themes/custom"
```

### style.page_size

**类型：** string
**默认值：** "A4"

PDF 输出的纸张大小。

```yaml
style:
  page_size: "A4"      # 210 × 297 mm（默认）
  page_size: "A5"      # 148 × 210 mm（较小）
  page_size: "Letter"  # 8.5 × 11 英寸
  page_size: "Legal"   # 8.5 × 14 英寸
  page_size: "B5"      # 176 × 250 mm
```

### style.font_family

**类型：** string
**默认值：** System sans-serif stack
**示例：** "Noto Sans CJK SC, -apple-system, BlinkMacSystemFont, sans-serif"

字体族（CSS 风格），带回退。

```yaml
style:
  font_family: "Georgia, serif"
  font_family: "Noto Sans CJK SC, sans-serif"
  font_family: "'Courier New', monospace"
```

使用系统字体或 web 安全字体。对于自定义字体，使用 `custom_css`：

```yaml
style:
  font_family: "MyCustomFont, sans-serif"
  custom_css: |
    @font-face {
      font-family: 'MyCustomFont';
      src: url('assets/font.ttf') format('truetype');
    }
```

### style.font_size

**类型：** string
**默认值：** "12pt"

正文文本的基础字体大小。

```yaml
style:
  font_size: "11pt"
  font_size: "12pt"
  font_size: "13pt"
```

### style.code_theme

**类型：** string
**默认值：** "github"

代码块的语法高亮主题。

可用主题：`github`、`monokai`、`atom-one-dark`、`vs-code` 等。

```yaml
style:
  code_theme: "github"
```

### style.line_height

**类型：** float
**默认值：** 1.6

行间距倍数。

```yaml
style:
  line_height: 1.5      # 紧凑
  line_height: 1.6      # 默认
  line_height: 1.8      # 宽松
  line_height: 2.0      # 非常宽松
```

### style.margin

**类型：** object
**默认值：** `{top: 25, bottom: 25, left: 20, right: 20}`

页边距，单位毫米。

```yaml
style:
  margin:
    top: 25
    bottom: 25
    left: 20
    right: 20
```

或使用单独的设置（参见下面的输出分段）。

### style.header

**类型：** object
**默认值：** `{left: "{{.Book.Title}}", center: "", right: "{{.Chapter.Title}}"}`

带有模板变量的页眉文本。

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "Chapter {{.Chapter.Number}}"
    right: "{{.Chapter.Title}}"
```

参见 [template-variables.md](template-variables.md) 了解可用变量。

### style.footer

**类型：** object
**默认值：** `{left: "", center: "{{.PageNum}}", right: ""}`

带有模板变量的页脚文本。

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "Confidential"
```

### style.custom_css

**类型：** string
**默认值：** ""

用于样式的自定义 CSS（PDF 和 HTML 输出）。

```yaml
style:
  custom_css: |
    body {
      font-family: Georgia, serif;
      color: #333;
    }
    h1 {
      color: #1a1a2e;
      border-bottom: 2px solid #1a1a2e;
    }
    code {
      background: #f5f5f5;
      padding: 2px 4px;
    }
```

## output 分段

输出格式设置和渲染选项。

### output.formats

**类型：** array
**默认值：** `["pdf"]`

要生成的输出格式列表。

```yaml
output:
  formats:
    - pdf      # 需要 Chrome/Chromium
    - html     # 单个 HTML 文件
    - epub     # 电子书格式
    - site     # 多页网站
```

在一次构建中生成多种格式：

```bash
mdpress build --format pdf,html,epub
```

### output.filename

**类型：** string
**默认值：** 从标题自动生成
**示例：** "output.pdf"

输出文件名（对于单个格式）或前缀（对于多个）。

```yaml
output:
  filename: "my-book.pdf"
  filename: "documentation.html"
```

### output.toc

**类型：** boolean
**默认值：** true

生成目录。

```yaml
output:
  toc: true      # 包括目录
  toc: false     # 省略目录
```

### output.toc_max_depth

**类型：** integer
**默认值：** 2
**范围：** 1-6

要在目录中包含的最大标题级别。1 = 仅 H1，2 = H1+H2 等。

```yaml
output:
  toc_max_depth: 2  # H1 和 H2
  toc_max_depth: 3  # H1、H2、H3
```

### output.cover

**类型：** boolean
**默认值：** true

生成封面页。

```yaml
output:
  cover: true       # 包括封面
  cover: false      # 省略封面
```

### output.header

**类型：** boolean
**默认值：** true

在输出中包括页眉。

```yaml
output:
  header: true      # 包括页眉
  header: false     # 省略页眉
```

### output.footer

**类型：** boolean
**默认值：** true

在输出中包括页脚。

```yaml
output:
  footer: true      # 包括页脚
  footer: false     # 省略页脚
```

### output.pdf_timeout

**类型：** integer
**默认值：** 120
**单位：** 秒

PDF 渲染等待的最大时间。

```yaml
output:
  pdf_timeout: 120     # 2 分钟（默认）
  pdf_timeout: 300     # 5 分钟（大型书籍）
```

对复杂文档或缓慢系统增加。

### output.watermark

**类型：** string
**默认值：** ""

要在页面上覆盖的文本或图像。

```yaml
output:
  watermark: "DRAFT"
  watermark: "CONFIDENTIAL"
  watermark: "assets/watermark.png"
```

### output.watermark_opacity

**类型：** float
**默认值：** 0.1
**范围：** 0.0 - 1.0

水印透明度（0 = 透明，1 = 不透明）。

```yaml
output:
  watermark: "DRAFT"
  watermark_opacity: 0.1      # 微妙（默认）
  watermark_opacity: 0.3      # 可见
  watermark_opacity: 0.5      # 非常可见
```

### output.margin_top

**类型：** string
**默认值：** "15mm"

顶页边距。单位：mm、cm、in

```yaml
output:
  margin_top: "15mm"
  margin_top: "1.5cm"
  margin_top: "0.6in"
```

### output.margin_bottom

**类型：** string
**默认值：** "15mm"

底页边距。

### output.margin_left

**类型：** string
**默认值：** "20mm"

左页边距。

### output.margin_right

**类型：** string
**默认值：** "20mm"

右页边距。

### output.generate_bookmarks

**类型：** boolean
**默认值：** true

从标题层次结构生成 PDF 书签。

```yaml
output:
  generate_bookmarks: true   # 包括书签
  generate_bookmarks: false  # 省略书签
```

在 PDF 读者中启用快速导航。

## plugins 分段

在构建期间执行的插件。

### plugins

**类型：** array
**默认值：** `[]`

带有配置的插件列表。

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500

  - name: link-checker
    path: ./plugins/link-checker
    config:
      check_external: false
```

### plugins[].name

**类型：** string
**必需：** Yes

唯一的插件标识符（小写、连字符分隔）。

### plugins[].path

**类型：** string
**必需：** Yes

插件可执行文件的路径，相对于 `book.yaml`。

### plugins[].config

**类型：** object
**默认值：** `{}`

传递给插件的任意键值配置。

```yaml
plugins:
  - name: custom-plugin
    path: ./plugins/custom
    config:
      setting1: value1
      setting2: ["array", "of", "values"]
      setting3:
        nested: object
```

## 完整示例

```yaml
book:
  title: "Advanced Python Guide"
  subtitle: "From Basics to Expert Level"
  author: "Jane Doe"
  version: "3.2.1"
  language: "en"
  description: "Comprehensive guide to advanced Python programming"
  cover:
    image: "assets/cover.png"
    background: "#1a1a2e"

chapters:
  - title: "Introduction"
    file: "intro.md"

  - title: "Part 1: Foundations"
    file: "part1.md"
    sections:
      - title: "Chapter 1: Python Basics"
        file: "ch01-basics.md"
      - title: "Chapter 2: Object-Oriented Programming"
        file: "ch02-oop.md"
      - title: "Chapter 3: Functional Programming"
        file: "ch03-functional.md"

  - title: "Part 2: Advanced Topics"
    file: "part2.md"
    sections:
      - title: "Chapter 4: Metaprogramming"
        file: "ch04-meta.md"
      - title: "Chapter 5: Performance Optimization"
        file: "ch05-performance.md"
      - title: "Chapter 6: Concurrency"
        file: "ch06-concurrency.md"

  - title: "Appendix"
    file: "appendix.md"

style:
  theme: "elegant"
  page_size: "A4"
  font_family: "Georgia, serif"
  font_size: "12pt"
  code_theme: "monokai"
  line_height: 1.6
  margin:
    top: 25
    bottom: 25
    left: 20
    right: 20
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: ""
  custom_css: |
    h1 {
      color: #1a1a2e;
      page-break-after: avoid;
    }
    code {
      background: #f5f5f5;
      padding: 2px 4px;
    }

output:
  filename: "python-guide.pdf"
  formats:
    - pdf
    - html
    - epub
  toc: true
  toc_max_depth: 3
  cover: true
  header: true
  footer: true
  pdf_timeout: 180
  watermark: ""
  watermark_opacity: 0.1
  margin_top: "20mm"
  margin_bottom: "20mm"
  margin_left: "25mm"
  margin_right: "25mm"
  generate_bookmarks: true

plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 10000
```

## 最少示例

用于快速测试：

```yaml
book:
  title: "My Book"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"
```

所有其他字段使用默认值。

## 使用环境变量

使用 `${VAR_NAME}` 在 YAML 中引用环境变量：

```yaml
book:
  author: "${AUTHOR_NAME}"
  version: "${VERSION}"

style:
  theme: "${THEME}"
```

然后在构建前设置：

```bash
export AUTHOR_NAME="John Doe"
export VERSION="1.0.0"
export THEME="elegant"
mdpress build
```

## 验证

所有配置都在以下情况下验证：
- 运行 `mdpress validate`
- 运行 `mdpress build`
- 运行 `mdpress serve`

错误由行号和修复建议报告。

更多信息，参见：
- [template-variables.md](template-variables.md) 了解页眉/页脚变量
- [cli-commands.md](cli-commands.md) 了解构建命令选项
- [../best-practices/organizing-large-books.md](../best-practices/organizing-large-books.md) 了解章节组织
