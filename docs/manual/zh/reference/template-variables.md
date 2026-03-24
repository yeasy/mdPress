# 页眉和页脚的模板变量

模板变量是可以在页眉和页脚中使用的占位符。在 PDF 渲染期间，它们被替换为实际值。

## 变量参考

### 书籍信息

用于书籍元数据的变量：

#### {{.Book.Title}}

来自 `book.yaml` 的书籍标题。

```yaml
style:
  header:
    left: "{{.Book.Title}}"
```

输出："Advanced Python Guide"

#### {{.Book.Subtitle}}

来自 `book.yaml` 的书籍副标题。

```yaml
style:
  footer:
    center: "{{.Book.Subtitle}}"
```

输出："From Basics to Expert Level"

#### {{.Book.Author}}

来自 `book.yaml` 的书籍作者。

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
```

输出："© 2024 Jane Doe"

#### {{.Book.Version}}

来自 `book.yaml` 的书籍版本。

```yaml
style:
  header:
    right: "v{{.Book.Version}}"
```

输出："v3.2.1"

### 章节信息

当前章节上下文的变量：

#### {{.Chapter.Title}}

当前章节的标题。

```yaml
style:
  header:
    right: "{{.Chapter.Title}}"
```

输出："Chapter 3: Advanced Techniques"

#### {{.Chapter.Number}}

章节编号（基于 1 的索引）。

```yaml
style:
  header:
    center: "Chapter {{.Chapter.Number}}"
```

输出："Chapter 5"

#### {{.Chapter.File}}

当前章节的文件名。

```yaml
style:
  footer:
    right: "{{.Chapter.File}}"
```

输出："ch05-concurrency.md"

### 页面信息

当前页面上下文的变量：

#### {{.PageNum}}

当前页号。

```yaml
style:
  footer:
    center: "{{.PageNum}}"
```

输出："47"

#### {{.TotalPages}}

PDF 中的总页数。

```yaml
style:
  footer:
    right: "{{.PageNum}} / {{.TotalPages}}"
```

输出："47 / 250"

### 日期和时间

日期和时间的变量：

#### {{.Date}}

默认格式的构建日期（YYYY-MM-DD）。

```yaml
style:
  footer:
    right: "{{.Date}}"
```

输出："2024-03-23"

#### {{.DateTime}}

构建日期和时间（YYYY-MM-DD HH:MM:SS）。

```yaml
style:
  footer:
    right: "{{.DateTime}}"
```

输出："2024-03-23 14:30:45"

#### {{.Year}}

当前年份。

```yaml
style:
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
```

输出："© 2024 Jane Doe"

#### {{.Month}}

当前月份号（01-12）。

```yaml
style:
  header:
    center: "{{.Month}}/{{.Day}}/{{.Year}}"
```

输出："03/23/2024"

#### {{.Day}}

月份的当前日（01-31）。

## 真实示例

### 专业技术文档

```yaml
style:
  header:
    left: "{{.Book.Title}} (v{{.Book.Version}})"
    center: "{{.Chapter.Title}}"
    right: ""
  footer:
    left: "{{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "© {{.Year}} Acme Corp"
```

输出：
```
Advanced Python Guide (v3.2.1)     Chapter 3: Functions
Acme Corporation                      23                    © 2024 Acme Corp
```

### 学术论文

```yaml
style:
  header:
    left: "{{.Book.Author}}"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    left: "Draft"
    center: "{{.PageNum}}"
    right: ""
```

输出：
```
Jane Doe        Advanced Python Guide        2024-03-23
Draft                               5
```

### 书籍出版

```yaml
style:
  header:
    left: "{{.Chapter.Number}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: ""
```

输出：
```
5                                  Chapter 3: Functions
                               23
```

### 内部文档

```yaml
style:
  header:
    left: "CONFIDENTIAL"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    left: "Document Version {{.Book.Version}}"
    center: "Page {{.PageNum}} of {{.TotalPages}}"
    right: "INTERNAL USE ONLY"
```

输出：
```
CONFIDENTIAL    Advanced Python Guide    2024-03-23
Document Version 3.2.1    Page 23 of 250    INTERNAL USE ONLY
```

### 最少页眉/页脚

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "{{.Chapter.Title}}"
  footer:
    left: ""
    center: "{{.PageNum}}"
    right: ""
```

输出：
```
Advanced Python Guide               Chapter 3: Functions
                               23
```

## 格式化示例

### 日期格式

默认的 `{{.Date}}` 格式是 YYYY-MM-DD。对于其他格式，使用 CSS 或自定义样式：

```yaml
style:
  footer:
    right: "{{.Date}}"
    # 输出：2024-03-23

  footer:
    right: "Build: {{.DateTime}}"
    # 输出：Build: 2024-03-23 14:30:45
```

### 版本格式化

```yaml
style:
  header:
    right: "v{{.Book.Version}}"
    # 输出：v3.2.1

  header:
    right: "Version {{.Book.Version}} ({{.Date}})"
    # 输出：Version 3.2.1 (2024-03-23)
```

### 页号格式化

```yaml
style:
  footer:
    center: "{{.PageNum}}"
    # 输出：23

  footer:
    center: "{{.PageNum}} / {{.TotalPages}}"
    # 输出：23 / 250

  footer:
    center: "Page {{.PageNum}}"
    # 输出：Page 23
```

### 条件文本（仅使用文本）

由于变量只支持简单替换，将固定文本与变量组合：

```yaml
style:
  header:
    left: "Draft - {{.Date}}"
    # 输出：Draft - 2024-03-23

  footer:
    left: "© {{.Year}} {{.Book.Author}} - All Rights Reserved"
    # 输出：© 2024 Jane Doe - All Rights Reserved
```

## 常见页眉/页脚组合

### 组合 1：标题 + 章节 + 页号

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "{{.PageNum}}"
```

用途：一般文档、技术指南

### 组合 2：简单页号

```yaml
style:
  header:
    left: ""
    center: ""
    right: ""
  footer:
    center: "{{.PageNum}}"
```

用途：清洁、极简布局

### 组合 3：正式文档

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: ""
    right: "v{{.Book.Version}}"
  footer:
    left: "© {{.Year}} {{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "CONFIDENTIAL"
```

用途：业务、法律、正式文档

### 组合 4：基于章节

```yaml
style:
  header:
    left: "Chapter {{.Chapter.Number}}"
    center: "{{.Chapter.Title}}"
    right: ""
  footer:
    left: "{{.Date}}"
    center: "{{.PageNum}}"
    right: "{{.Book.Version}}"
```

用途：教科书、多章节书籍

### 组合 5：带页脚的最少

```yaml
style:
  header:
    left: ""
    center: ""
    right: ""
  footer:
    left: "{{.Book.Author}}"
    center: "{{.PageNum}}"
    right: "{{.Date}}"
```

用途：短文档、报告

## 样式页眉和页脚

虽然变量提供内容，但使用 `custom_css` 中的 CSS 进行样式设置：

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "{{.Chapter.Title}}"
    right: "{{.PageNum}}"

  custom_css: |
    @page {
      @top-left {
        font-size: 10pt;
        font-weight: bold;
      }
      @top-center {
        font-size: 11pt;
        font-style: italic;
      }
      @bottom-center {
        font-size: 10pt;
        color: #666;
      }
    }
```

## 限制

### 变量支持

并非所有变量都在所有上下文中工作：
- 页眉和页脚：支持所有变量
- 自定义 CSS：改用 CSS 变量（参见下文）
- Markdown 内容：改用文字文本

### 不支持的变量

这些目前不支持：
- {{.SectionNumber}} - 分段编号
- {{.WordCount}} - 章节字数
- {{.BuildTime}} - 总构建持续时间
- {{.Environment}} - 环境变量

对于这些，使用外部工具或预/后处理。

### 纯文本仅

变量仅输出纯文本。对于丰富的格式：
- 使用 CSS 样式（参见上文）
- 使用具有不同内容的多个页眉/页脚分段
- 将变量与固定文本组合

## 不同主题的示例

### 技术主题

```yaml
style:
  theme: "technical"
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @top-left {
        color: #1a1a2e;
        font-weight: bold;
      }
      @bottom-center {
        color: #666;
      }
    }
```

### 优雅主题

```yaml
style:
  theme: "elegant"
  header:
    left: "{{.Book.Author}}"
    center: "{{.Book.Title}}"
    right: "{{.Date}}"
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @top-left, @top-right {
        font-style: italic;
        color: #333;
      }
    }
```

### 极简主题

```yaml
style:
  theme: "minimal"
  header:
    left: ""
    center: ""
    right: ""
  footer:
    center: "{{.PageNum}}"
  custom_css: |
    @page {
      @bottom-center {
        font-size: 9pt;
      }
    }
```

## 提示和最佳实践

1. **保持页眉/页脚简洁**：长文本可能溢出或破坏布局
2. **在文档间使用一致的变量**：有助于识别
3. **始终包括页号**：对参考很有帮助
4. **为草稿添加版本/日期**：帮助跟踪文档演变
5. **测试输出**：在最终构建前在 `mdpress serve` 中预览
6. **考虑首/末页**：如需要可单独自定义首页

## 默认值

当变量不可用时（例如，无章节标题）：
- `{{.Chapter.Title}}` → 空字符串
- `{{.PageNum}}` → 1（第一页）
- `{{.Date}}` → 构建日期

## 高级：CSS 变量

对于超越简单文本的更复杂样式，使用 CSS 变量：

```yaml
style:
  custom_css: |
    :root {
      --header-color: #1a1a2e;
      --footer-color: #666;
      --page-width: 210mm;
    }

    @page {
      @top-left {
        color: var(--header-color);
        font-weight: bold;
      }
      @bottom-center {
        color: var(--footer-color);
      }
    }
```

与模板变量结合以获得动态、风格化的页眉和页脚。
