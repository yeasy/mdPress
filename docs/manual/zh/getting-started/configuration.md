# 配置

通过 `book.yaml`、`book.json` 或 `SUMMARY.md` 配置 mdPress 的行为和外观。学习可用的选项和如何使用它们。

## 配置文件

mdPress 支持多种配置文件格式。它按这个顺序搜索：

1. `book.yaml`（推荐）
2. `book.json`
3. `book.toml`

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
  direction: ltr
```

在 `book.json` 中：

```json
{
  "book": {
    "title": "我的文档",
    "author": "张三",
    "description": "综合指南",
    "language": "en",
    "direction": "ltr"
  }
}
```

可用字段：
- `title` —— 显示在浏览器标签和标题中
- `author` —— 文档元数据
- `description` —— 用于搜索引擎和社交媒体
- `language` —— 语言代码（en、fr、es 等）
- `direction` —— 文本方向：`ltr`（从左到右）或 `rtl`（从右到左）

## 章节和结构

在配置中定义章节来源。如果使用 `SUMMARY.md`，这是可选的：

```yaml
chapters:
  - path: README.md
  - path: chapters/chapter1.md
  - path: chapters/chapter2.md
    chapters:
      - path: chapters/chapter2/section1.md
      - path: chapters/chapter2/section2.md
  - path: chapters/chapter3.md
```

或使用 `SUMMARY.md`（对大多数项目推荐）。如果两者都存在，`SUMMARY.md` 优先。

## 样式配置

用样式选项控制外观：

```yaml
style:
  theme: light
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: 16
  code_theme: monokai
  line_height: 1.6
  margins:
    top: 20mm
    bottom: 20mm
    left: 20mm
    right: 20mm
  header:
    left: 第 {chapter_num} 章
    center: 我的文档
    right: 第 {page_num} 页
  footer:
    left: ""
    center: ""
    right: "© 2026 公司名称"
  custom_css: custom.css
```

### 主题选项

可用主题：
- `light` —— 清爽的浅色背景（默认）
- `dark` —— 深色背景，适合低光环境
- `auto` —— 跟随系统偏好设置

```yaml
style:
  theme: light
```

### 页面大小

设置 PDF 页面尺寸：
- `A4` —— 标准（210 × 297 mm）
- `A5` —— A4 的一半
- `Letter` —— 美国信纸（8.5 × 11 英寸）
- `Custom` —— 自定义宽度和高度

```yaml
style:
  page_size: A4

# 或自定义：
style:
  page_size: Custom
  page_width: 200mm
  page_height: 280mm
```

### 字体配置

为正文和代码选择字体：

```yaml
style:
  font_family: "Georgia, serif"
  font_size: 14
  code_font: "Monaco, monospace"
  code_font_size: 13
```

使用系统字体或网络安全字体。对于嵌入字体，在 assets 中放置 `.ttf` 或 `.woff2` 文件并参考：

```yaml
style:
  font_family: "CustomFont"
  custom_fonts:
    - name: CustomFont
      path: assets/fonts/custom.ttf
```

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
  margins:
    top: 25mm
    bottom: 25mm
    left: 20mm
    right: 20mm
```

### 页眉和页脚

将文本添加到页眉和页脚（仅限 PDF）：

```yaml
style:
  header:
    left: 第 {chapter_num} 章
    center: {title}
    right: ""
  footer:
    left: ""
    center: 第 {page_num} 页
    right: {date}
```

可用变量：
- `{title}` —— 书籍标题
- `{author}` —— 书籍作者
- `{page_num}` —— 当前页码
- `{total_pages}` —— 总页数
- `{chapter_num}` —— 当前章节号
- `{date}` —— 构建日期
- `{time}` —— 构建时间

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
  toc_depth: 3
  cover: true
  watermark: ""
  pdf_timeout: 300
  bookmarks: true
  minify: false
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
  toc_depth: 3                 # 包含到第 3 级的标题
```

### 封面页

启用/禁用封面生成：

```yaml
output:
  cover: true                  # 生成封面
  cover_image: assets/cover.png  # 可选自定义封面图像
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
  bookmarks: true             # 启用 PDF 大纲/书签
```

书签允许读者通过 PDF 查看器导航。

### 最小化

最小化 HTML 和 CSS 以减小输出大小：

```yaml
output:
  minify: true
```

## 插件配置

启用可选插件：

```yaml
plugins:
  - mermaid                    # 图表支持
  - mathjax                    # 数学方程
  - highlighting               # 代码高亮
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
  theme: light
  page_size: A4
  font_family: "Segoe UI, system-ui, sans-serif"
  font_size: 15
  code_theme: monokai
  line_height: 1.6
  margins:
    top: 20mm
    bottom: 20mm
    left: 20mm
    right: 20mm
  header:
    left: 第 {chapter_num} 章
    center: ""
    right: 第 {page_num} 页
  footer:
    left: ""
    center: {title}
    right: "© 2026 张开发者"
  custom_css: custom.css

output:
  formats:
    - site
    - pdf
    - epub
  toc: true
  toc_depth: 3
  cover: true
  pdf_timeout: 300
  bookmarks: true
  minify: false

plugins:
  - mermaid
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
  theme: light
  font_size: 16

# ✗ 错误 - 缩进不一致
style:
  theme: light
    font_size: 16
```

使用 YAML 验证器或支持 YAML 的编辑器来捕获错误。

### 未找到配置

确保 `book.yaml` 在你的项目根目录，而不是子目录中：

```
✓ project/book.yaml
✗ project/config/book.yaml
```
