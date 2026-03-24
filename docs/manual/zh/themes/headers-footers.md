# 页眉和页脚

页眉和页脚允许你向 PDF 输出中的每一页添加一致的品牌、页面信息和导航元素。它们在 `book.yaml` 中配置，并支持动态模板变量。

**注意：** 页眉和页脚是仅限 PDF 的功能。它们不会出现在 HTML 输出中。

## 基本配置

在 `book.yaml` 的 `style` 部分定义页眉和页脚：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}}"
    center: ""
    right: "{{.Date}}"
```

每个部分（左、中、右）都是可选的。省略你不需要的部分。

## 模板变量

页眉和页脚通过模板变量支持动态内容。这些在 PDF 生成期间被替换为实际值。

### 书籍变量

从 `book.yaml` 访问书籍级别的信息：

- `{{.Book.Title}}` - 书籍标题
- `{{.Book.Author}}` - 书籍作者
- `{{.Book.Subtitle}}` - 书籍副标题（如果定义）
- `{{.Book.Version}}` - 书籍版本（如果定义）
- `{{.Book.Description}}` - 书籍描述

示例：

```yaml
style:
  header:
    left: "{{.Book.Title}} v{{.Book.Version}}"
    right: "{{.Book.Author}}"
```

### 章节变量

访问当前章节的信息：

- `{{.Chapter.Title}}` - 当前章节的标题
- `{{.Chapter.Number}}` - 章节号
- `{{.Chapter.File}}` - 章节的源文件名

示例：

```yaml
style:
  header:
    left: "{{.Chapter.Title}}"
    center: ""
    right: "Page {{.PageNum}}"
```

### 页面变量

动态页面信息：

- `{{.PageNum}}` - 当前页码（整数）
- `{{.PageTotal}}` - 总页数
- `{{.TotalPages}}` - `.PageTotal` 的别名

创建"第 X 页，共 Y 页"页脚：

```yaml
style:
  footer:
    right: "Page {{.PageNum}} of {{.PageTotal}}"
```

### 日期变量

日期和时间信息：

- `{{.Date}}` - 当前日期，默认格式（YYYY-MM-DD）
- `{{.DateTime}}` - 当前日期和时间（YYYY-MM-DD HH:MM:SS）
- `{{.Year}}` - 当前年份
- `{{.Month}}` - 当前月份（1-12）
- `{{.Day}}` - 当月的当前日期

示例：

```yaml
style:
  footer:
    left: "Generated on {{.Date}}"
    right: "{{.Year}}"
```

### 部分变量

对于组织成部分的文档：

- `{{.Section}}` - 当前部分的名称（如果可用）
- `{{.Subsection}}` - 当前子部分的名称（如果可用）

## 完整配置示例

### 技术文档

```yaml
style:
  header:
    left: "{{.Book.Title}} - {{.Chapter.Title}}"
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}}"
    center: "Confidential"
    right: "{{.Date}}"
```

这创建了显示文档和章节标题的页眉，右侧有页码。页脚包括作者、机密性声明和日期。

### 用户手册

```yaml
style:
  header:
    left: "{{.Chapter.Title}}"
    right: "Page {{.PageNum}} of {{.PageTotal}}"

  footer:
    center: "{{.Book.Title}} v{{.Book.Version}}"
```

在标题中显示当前章节和页码，在页脚中显示书籍标题和版本。

### 公司报告

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Book.Version}}"

  footer:
    left: "{{.Book.Author}}"
    center: ""
    right: "{{.Year}}"
```

公司风格的格式，标题和版本在页眉中，作者和年份在页脚中。

### 学术文档

```yaml
style:
  header:
    left: "{{.Book.Author}}"
    right: "{{.Date}}"

  footer:
    left: "{{.Book.Title}}"
    center: "{{.PageNum}}"
    right: ""
```

学术格式，页眉中包含作者和日期，页脚中包含标题和居中的页码。

## 页眉和页脚样式

页眉和页脚使用默认样式，但你可以通过 PDF 生成选项自定义其外观。字体大小、字体家族和颜色从你的主题继承。

### 控制外观

```yaml
pdf:
  header_height: 0.5in
  footer_height: 0.5in
  header_font_size: 10
  footer_font_size: 10
  margins:
    top: 1in
    bottom: 1in
```

调整边距以为页眉和页脚提供足够的空间：

```yaml
pdf:
  margins:
    top: 1.2in        # 为页眉留出额外空间
    bottom: 1.2in     # 为页脚留出额外空间
    left: 1in
    right: 1in
```

## 高级模式

### 交替页眉

在奇数页和偶数页上显示不同的页眉（对书籍有用）：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
```

打印双面时，左页显示书籍标题，右页显示章节标题。

### 部分分隔符

在页眉中使用章节号来表示结构：

```yaml
style:
  header:
    left: "Chapter {{.Chapter.Number}}: {{.Chapter.Title}}"
    right: "{{.PageNum}}"
```

### 品牌信息

在页脚中包含公司信息：

```yaml
style:
  footer:
    left: "© 2026 Acme Corporation"
    center: "Internal Use Only"
    right: "{{.Date}}"
```

### 版本控制

跟踪文档版本：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "Version {{.Book.Version}}"
    right: "Generated {{.DateTime}}"
```

这对跟踪 PDF 的生成时间很有用。

## 条件内容

虽然模板变量不支持完整的条件语句，但你可以解决这个问题：

在需要两个变量时使用分隔符：

```yaml
style:
  footer:
    left: "{{.Book.Author}} — {{.Book.Title}}"
```

或为不同目的创建多个页脚变体：

```yaml
# 用于草稿
style:
  footer:
    center: "DRAFT — {{.Date}}"

# 用于最终版本（注释掉草稿版本）
# style:
#   footer:
#     center: "Final Release"
```

## 故障排除

### 变量未被替换

如果变量如 `{{.Book.Title}}` 在 PDF 中按字面意思出现：

1. 检查你是否在生成 PDF（页眉/页脚仅 PDF）
2. 验证变量名是否完全匹配（区分大小写）
3. 确保变量在你的 `book.yaml` 中定义

### 页眉/页脚与内容重叠

如果页眉或页脚与主要内容重叠：

1. 在 PDF 配置中增加边距值：
   ```yaml
   pdf:
     margins:
       top: 1.2in
       bottom: 1.2in
   ```

2. 如果配置了，减少页眉/页脚高度：
   ```yaml
   pdf:
     header_height: 0.4in
     footer_height: 0.4in
   ```

### 章节中缺少变量

如果 `{{.Chapter.Title}}` 为空：

1. 确保每个 markdown 文件都有一个 H1 标题
2. 验证该章节在 `book.yaml` 下的 `chapters` 中正确列出
3. 检查 markdown 文件不是空的

## 完整示例

这是一个配置了页眉和页脚的完整 `book.yaml`：

```yaml
book:
  title: "mdPress Documentation"
  author: "mdPress Team"
  version: "1.0"
  description: "Complete guide to mdPress"

chapters:
  - chapters/01-introduction.md
  - chapters/02-installation.md
  - chapters/03-usage.md

style:
  theme: technical

  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.PageNum}}"

  footer:
    left: "{{.Book.Author}} — v{{.Book.Version}}"
    center: ""
    right: "{{.Date}}"

pdf:
  output: build/mdpress-guide.pdf
  margins:
    top: 1.2in
    bottom: 1.2in
    left: 1in
    right: 1in
```

当你生成 PDF 时，每一页都将包括：
- **页眉**：左侧"mdPress Documentation"，右侧页码
- **页脚**：左侧"mdPress Team — v1.0"，右侧今天的日期

参见 [自定义 CSS](./custom-css.md) 和 [内置主题](./builtin-themes.md) 了解更多样式选项。
