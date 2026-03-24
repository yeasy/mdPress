# 内置主题

mdPress 提供三个专业设计的内置主题，覆盖常见的文档使用场景。每个主题都有独特的视觉特征，并针对特定的内容类型进行了优化。

## 可用主题

### Technical 主题

Technical 主题为 API 文档、技术指南和代码密集型内容而设计。

**特征：**
- 主色：#1A5490（专业蓝）
- 简洁的无衬线字体，优化了可读性
- 代码块是焦点，带有语法高亮
- 中性灰色背景，最小化干扰
- 宽边距用于注释和笔记
- 内联代码和参数使用等宽字体

**最适合：**
- API 参考文档
- 编程教程
- 技术规范
- 命令行工具文档

**视觉示例：**
```yaml
style:
  theme: technical
  code_theme: monokai
```

### Elegant 主题

Elegant 主题为叙述性文档、书籍和正式出版物而设计。

**特征：**
- 主色：#34495e（老练的深蓝灰）
- 正文使用优美的衬线字体
- 优雅的章节分隔符和装饰元素
- 充足的空白和垂直节奏
- 边注和侧边栏用于补充内容
- 针对打印优化的布局，精心考虑分页

**最适合：**
- 书籍和长篇幅文档
- 包含叙述部分的用户手册
- 学术或正式文档
- 需要优雅呈现的出版物

**视觉示例：**
```yaml
style:
  theme: elegant
  typography:
    body_font: georgia
```

### Minimal 主题

Minimal 主题优先考虑内容清晰度和打印友好性。

**特征：**
- 主色：#000（黑色）
- 纯黑文本在白色背景上
- 高对比度以获得最大可读性
- 最小的装饰元素
- 针对打印优化，无不必要的颜色
- 响应式和可访问设计
- 适合屏幕阅读器和辅助工具

**最适合：**
- 打印出版物
- 可访问性关键的文档
- 最小带宽使用
- 专业黑白输出
- 政府或正式文档

**视觉示例：**
```yaml
style:
  theme: minimal
  colors:
    background: "#ffffff"
    text: "#000000"
```

## 设置主题

在 `book.yaml` 配置文件的 `style` 部分指定你的主题：

```yaml
book:
  title: "My Documentation"
  author: "Your Name"

style:
  theme: technical
```

你也可以在不改变整个主题的情况下覆盖主题颜色：

```yaml
style:
  theme: elegant
  colors:
    primary: "#2c3e50"
    accent: "#e74c3c"
```

## 主题管理命令

mdPress 提供 CLI 命令来探索和预览主题：

### 列出可用主题

```bash
mdpress themes list
```

输出：
```
Available themes:
  • technical    - Professional blue theme for code-heavy docs
  • elegant      - Sophisticated serif theme for narrative content
  • minimal      - High-contrast black & white theme
```

### 显示主题详情

```bash
mdpress themes show technical
```

这显示关于主题的全面信息：

```
Theme: technical
Description: Professional blue theme optimized for technical documentation

Colors:
  Primary:    #1A5490
  Secondary:  #16a085
  Accent:     #e74c3c
  Background: #ffffff
  Text:       #2c3e50

Typography:
  Body Font:  -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
  Code Font:  "Courier New", monospace
  Font Size:  16px
  Line Height: 1.6

Code Highlighting:
  Theme: monokai
  Line Numbers: enabled
```

### 预览主题

```bash
mdpress themes preview technical
```

这生成一个示例文档，展示主题的实际效果，包括：
- 排版示例
- 代码块及语法高亮
- 标注框和提示
- 表格和列表
- 链接和内联格式

预览在你的默认浏览器中打开。

## 自定义内置主题

虽然你可以按原样使用内置主题，但你也可以通过添加 CSS 覆盖来自定义它们：

```yaml
style:
  theme: technical
  custom_css: |
    :root {
      --primary-color: #0056b3;
      --font-size-base: 18px;
    }

    .content h1 {
      font-size: 2.5em;
      text-transform: uppercase;
    }
```

参见 [自定义 CSS 指南](./custom-css.md) 了解详细的自定义选项。

## 主题切换

要在开发期间切换主题，更新 `book.yaml` 并重新构建：

```bash
# 编辑 book.yaml
vim book.yaml

# 用新主题重新构建
mdpress build

# 或使用实时重载进行监视
mdpress serve
```

当你改变主题时，实时服务器将自动刷新你的浏览器。

## 主题文件位置

对于高级自定义，主题文件存储在：

```
~/.mdpress/themes/
├── technical/
│   ├── style.css
│   ├── config.yaml
│   └── templates/
├── elegant/
│   ├── style.css
│   ├── config.yaml
│   └── templates/
└── minimal/
    ├── style.css
    ├── config.yaml
    └── templates/
```

你可以复制和修改主题以进行完全自定义。参见 [自定义 CSS 指南](./custom-css.md) 了解如何创建你自己的主题。
