# 配置

通过 `book.yaml`、`book.json` 或 `SUMMARY.md` 配置 mdPress 的行为和外观。学习可用的选项和如何使用它们。

## 配置文件

mdPress 支持多种配置文件格式。它按这个顺序搜索：

1. `book.yaml`（推荐）
2. `book.json`（GitBook 兼容）

只需其中一个。在项目根目录创建文件：

```bash
# 创建一个空的 book.yaml
touch book.yaml
```

## 书籍元数据

配置你的文档的基本信息：

```yaml
book:
  title: 我的文档
  author: 张三
  description: 我的产品使用综合指南
  language: en
```

在 `book.json` 中：

```json
{
  "book": {
    "title": "我的文档",
    "author": "张三",
    "description": "综合指南",
    "language": "en"
  }
}
```

可用字段：
- `title` —— 显示在浏览器标签和标题中
- `author` —— 文档元数据
- `description` —— 用于搜索引擎和社交媒体
- `language` —— 语言代码（en、fr、es 等）

## 章节和结构

在配置中定义章节来源。如果使用 `SUMMARY.md`，这是可选的：

```yaml
chapters:
  - file: README.md
  - file: chapters/chapter1.md
  - file: chapters/chapter2.md
    sections:
      - file: chapters/chapter2/section1.md
      - file: chapters/chapter2/section2.md
  - file: chapters/chapter3.md
```

或使用 `SUMMARY.md`（对大多数项目推荐）。如果两者都存在，`SUMMARY.md` 优先。

## 样式配置

用样式选项控制外观：

```yaml
style:
  theme: technical
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: "16pt"
  code_theme: monokai
  line_height: 1.6
  margin:
    top: "20mm"
    bottom: "20mm"
    left: "20mm"
    right: "20mm"
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"
  footer:
    left: ""
    center: ""
    right: "© 2026 公司名称"
  custom_css: custom.css
```

### 主题选项

可用主题：
- `technical` —— 适合技术文档（默认）
- `elegant` —— 优雅风格，适合长篇写作
- `minimal` —— 极简风格，适合团队手册

```yaml
style:
  theme: technical
```

### 页面大小

设置 PDF 页面尺寸：
- `A4` —— 标准（210 × 297 mm）
- `A5` —— A4 的一半
- `Letter` —— 美国信纸（8.5 × 11 英寸）
- `B5` —— B5 纸张尺寸
- `Legal` —— 美国法律纸（8.5 × 14 英寸）

```yaml
style:
  page_size: A4

# 支持的页面尺寸：A4、Letter、A5、B5、Legal
```

### 字体配置

为正文和代码选择字体：

```yaml
style:
  font_family: "Georgia, serif"
  font_size: "14pt"
```

使用系统字体或网络安全字体。

### 代码高亮主题

代码块的语法高亮：

- `monokai` —— 深色，彩色语法
- `atom-one-dark` —— Atom 编辑器深色主题
- `atom-one-light` —— Atom 编辑器浅色主题
- `dracula` —— 流行深色主题
- `gruvbox` —— 复古沟槽色
- `solarized-dark` —— Solarized 深色变体
- `solarized-light` —— Solarized 浅色变体
- `github` —— GitHub 默认主题
- `nord` —— 北极蓝调主题

```yaml
style:
  code_theme: monokai
```

### 行高和边距

控制间距和布局：

```yaml
style:
  line_height: 1.6        # 1.0-2.0，影响可读性
  margin:
    top: "25mm"
    bottom: "25mm"
    left: "20mm"
    right: "20mm"
```

### 页眉和页脚

将文本添加到页眉和页脚（仅限 PDF）：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: ""
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: "{{.Chapter.Title}}"
```

可用变量：
- `{{.Book.Title}}` —— 书籍标题
- `{{.Book.Author}}` —— 书籍作者
- `{{.Chapter.Title}}` —— 当前章节标题
- `{{.PageNum}}` —— 当前页码

### 自定义 CSS

添加自定义样式：

```yaml
style:
  custom_css: custom.css
```

在项目根目录创建 `custom.css`：

```css
body {
  font-family: "Georgia", serif;
  color: #333;
}

h1, h2, h3 {
  color: #0066cc;
}

code {
  background-color: #f5f5f5;
  padding: 2px 4px;
  border-radius: 3px;
}
```

## 输出配置

配置输出格式行为：

```yaml
output:
  formats:
    - site
    - pdf
    - epub
  toc: true
  toc_max_depth: 3
  cover: true
  watermark: ""
  pdf_timeout: 300
  generate_bookmarks: true
```

### 格式

列出要生成的输出格式：

```yaml
output:
  formats:
    - site      # HTML 静态网站
    - pdf       # 单个 PDF 文档
    - epub      # 电子书格式
```

### 目录

控制目录生成：

```yaml
output:
  toc: true                    # 生成目录
  toc_max_depth: 3                 # 包含到第 3 级的标题
```

### 封面页

启用/禁用封面生成：

```yaml
output:
  cover: true                  # 生成封面

book:
  cover:
    image: assets/cover.png    # 可选自定义封面图像
```

### 水印

向 PDF 输出添加水印（对草稿有用）：

```yaml
output:
  watermark: "草稿"
```

### PDF 超时

设置 PDF 生成超时（秒）：

```yaml
output:
  pdf_timeout: 300            # 5 分钟
```

对于渲染时间较长的大型文档，增加此值。

### 书签

包含 PDF 书签（大纲）：

```yaml
output:
  generate_bookmarks: true    # 启用 PDF 大纲/书签
```

书签允许读者通过 PDF 查看器导航。

## 插件配置

启用可选插件：

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
```

可用插件取决于你的 mdPress 安装。

## 完整示例

这是一个完整的 `book.yaml` 配置：

```yaml
book:
  title: mdPress 完全指南
  author: 张开发者
  description: 从基础到高级使用学习 mdPress
  language: en

style:
  theme: technical
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: "15pt"
  code_theme: monokai
  line_height: 1.6
  margin:
    top: "20mm"
    bottom: "20mm"
    left: "20mm"
    right: "20mm"
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"
  footer:
    left: ""
    center: "{{.Chapter.Title}}"
    right: "© 2026 张开发者"
  custom_css: custom.css

output:
  formats:
    - site
    - pdf
    - epub
  toc: true
  toc_max_depth: 3
  cover: true
  pdf_timeout: 300
  generate_bookmarks: true
```

## 配置继承

如果创建 `book.yaml` 但不指定所有选项，mdPress 会对缺失值使用默认值：

```yaml
# 最小配置 - 其他所有内容使用默认值
book:
  title: 我的项目
```

这完全没问题。仅在需要自定义行为时添加配置。

## 配置故障排除

### 更改未生效

重启开发服务器：

```bash
# 停止当前服务器 (Ctrl+C)
# 然后重启：
mdpress serve
```

### 无效的 YAML 语法

检查你的 `book.yaml` 是否有正确的缩进和格式：

```yaml
# ✓ 正确
style:
  theme: technical
  font_size: 16

# ✗ 错误 - 缩进不一致
style:
  theme: technical
    font_size: 16
```

使用 YAML 验证器或支持 YAML 的编辑器来捕获错误。

### 未找到配置

确保 `book.yaml` 在你的项目根目录，而不是子目录中：

```
✓ project/book.yaml
✗ project/config/book.yaml
```
