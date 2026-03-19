# 第二章 进阶用法 / Chapter 2: Advanced Usage

本章介绍 mdpress 的高级功能，帮助你打造专业级的 PDF 图书。

*This chapter covers advanced mdpress features for creating professional-grade PDF books.*

## 2.1 自定义主题 / Custom Themes

mdpress 内置三套主题 / mdpress comes with three built-in themes:

- **technical**：技术文档风格，适合编程书籍 / Technical docs style, ideal for programming books
- **elegant**：优雅文学风格，适合文学作品 / Elegant literary style, ideal for literary works
- **minimal**：极简风格，适合笔记和备忘录 / Minimalist style, ideal for notes and memos

在 `book.yaml` 中切换主题 / Switch themes in `book.yaml`:

```yaml
style:
  theme: "elegant"
```

### 创建自定义主题 / Creating Custom Themes

你也可以创建自己的主题文件（YAML 格式）/ You can create your own theme file (YAML format):

```yaml
name: my-theme
font_family: "Source Han Sans SC"
font_size: "11pt"
code_theme: "dracula"
colors:
  text: "#2d2d2d"
  heading: "#c0392b"
  link: "#2980b9"
```

## 2.2 交叉引用 / Cross References

mdpress 支持图表和章节的交叉引用。/ mdpress supports cross-referencing of figures, tables, and sections.

### 图片引用 / Figure References

```markdown
![图片标题 / Figure caption](image.png){#fig:architecture}

如 {{ref:fig:architecture}} 所示... / As shown in {{ref:fig:architecture}}...
```

### 表格引用 / Table References

```markdown
| 列1 / Col1 | 列2 / Col2 |
|-----|-----|
| a   | b   |
{#tab:comparison}

详见 {{ref:tab:comparison}}。/ See {{ref:tab:comparison}}.
```

## 2.3 多文件组织 / Multi-file Organization

对于大型图书项目，建议按章节组织文件 / For large book projects, organize files by chapter:

```
my-book/
├── book.yaml
├── cover.png
├── preface.md
├── part1/
│   ├── chapter01.md
│   ├── chapter02.md
│   └── images/
│       ├── fig01.png
│       └── fig02.png
├── part2/
│   ├── chapter03.md
│   └── chapter04.md
└── appendix/
    └── references.md
```

## 2.4 页眉页脚 / Headers & Footers

自定义页眉页脚支持模板变量 / Custom headers and footers support template variables:

| 变量 / Variable | 说明 / Description |
|------|------|
| `{{.Book.Title}}` | 书名 / Book title |
| `{{.Book.Author}}` | 作者 / Author |
| `{{.Chapter.Title}}` | 当前章节标题 / Current chapter title |
| `{{.PageNum}}` | 当前页码 / Current page number |

配置示例 / Configuration example:

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "第 {{.PageNum}} 页 / Page {{.PageNum}}"
```

## 2.5 脚注 / Footnotes[^1]

mdpress 完整支持 Markdown 脚注语法，脚注会自动编号并在页面底部或章节末尾显示。

*mdpress fully supports Markdown footnote syntax. Footnotes are automatically numbered and displayed at the bottom of the page or end of the chapter.*

[^1]: 这是一个脚注示例。/ This is a footnote example.

## 2.6 小结 / Summary

通过本章的学习，你已经掌握了 mdpress 的大部分高级功能。更多详细信息请参阅项目文档。

*After this chapter, you have mastered most of the advanced features of mdpress. For more details, refer to the project documentation.*
