# 自定义 CSS

除了内置主题之外，mdPress 还允许通过自定义 CSS 进行广泛的自定义。你可以覆盖文档中任何元素的颜色、排版、间距和样式。

## 添加自定义 CSS

在 `book.yaml` 中使用 `style.custom_css` 选项指定自定义 CSS：

```yaml
style:
  theme: technical
  custom_css: |
    :root {
      --primary-color: #0056b3;
      --secondary-color: #6c757d;
    }

    .content h1 {
      color: var(--primary-color);
      border-bottom: 3px solid var(--primary-color);
      padding-bottom: 0.5em;
    }
```

对于较大的 CSS 文件，引用外部文件：

```yaml
style:
  theme: technical
  custom_css_file: ./styles/custom.css
```

## CSS 变量（自定义属性）

mdPress 主题使用 CSS 自定义属性便于自定义。在你的自定义 CSS 中覆盖它们：

### 颜色变量

```css
:root {
  /* 主品牌颜色 */
  --primary-color: #1A5490;

  /* 强调的次要颜色 */
  --secondary-color: #16a085;

  /* 高亮和行动号召的强调 */
  --accent-color: #e74c3c;

  /* 背景颜色 */
  --background-color: #ffffff;
  --background-alt: #f5f5f5;

  /* 文本颜色 */
  --text-color: #2c3e50;
  --text-light: #7f8c8d;
  --text-lighter: #bdc3c7;

  /* 边框和分隔线 */
  --border-color: #ecf0f1;
  --border-dark: #bdc3c7;

  /* 代码块颜色 */
  --code-background: #f4f4f4;
  --code-text: #c7254e;

  /* 标注颜色 */
  --info-color: #3498db;
  --warning-color: #f39c12;
  --danger-color: #e74c3c;
  --success-color: #27ae60;
}
```

覆盖示例：

```css
:root {
  --primary-color: #6f42c1;      /* 改为紫色 */
  --accent-color: #fd7e14;       /* 改为橙色 */
  --text-color: #1a1a1a;         /* 更深的文本 */
}
```

### 排版变量

```css
:root {
  /* 字体家族 */
  --font-family-base: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-family-mono: "Courier New", monospace;
  --font-family-serif: georgia, "Times New Roman", serif;

  /* 字体大小 */
  --font-size-base: 16px;
  --font-size-h1: 2.5em;
  --font-size-h2: 2em;
  --font-size-h3: 1.5em;
  --font-size-h4: 1.25em;
  --font-size-h5: 1.1em;
  --font-size-h6: 1em;
  --font-size-small: 0.875em;
  --font-size-code: 0.9em;

  /* 行高 */
  --line-height-base: 1.6;
  --line-height-heading: 1.2;
  --line-height-code: 1.5;

  /* 字母间距 */
  --letter-spacing-normal: normal;
  --letter-spacing-heading: -0.02em;
}
```

覆盖示例：

```css
:root {
  --font-family-base: "Inter", "Helvetica Neue", sans-serif;
  --font-size-base: 18px;
  --line-height-base: 1.7;
}
```

### 间距变量

```css
:root {
  --spacing-unit: 8px;
  --spacing-xs: calc(var(--spacing-unit) * 0.5);
  --spacing-sm: var(--spacing-unit);
  --spacing-md: calc(var(--spacing-unit) * 2);
  --spacing-lg: calc(var(--spacing-unit) * 3);
  --spacing-xl: calc(var(--spacing-unit) * 4);
  --spacing-2xl: calc(var(--spacing-unit) * 6);

  /* 内容填充 */
  --content-padding: 2em;
  --content-max-width: 900px;
}
```

## 针对特定元素

### 内容区域

```css
.content {
  /* 主内容容器 */
  background: var(--background-color);
  color: var(--text-color);
  font-family: var(--font-family-base);
}

.content p {
  margin-bottom: 1em;
  line-height: var(--line-height-base);
}

.content strong {
  font-weight: 600;
  color: var(--primary-color);
}
```

### 标题

```css
.content h1,
.content h2,
.content h3 {
  color: var(--primary-color);
  font-family: var(--font-family-base);
  font-weight: 700;
  line-height: var(--line-height-heading);
  margin-top: 1.5em;
  margin-bottom: 0.5em;
}

.content h1 {
  font-size: var(--font-size-h1);
  border-bottom: 2px solid var(--primary-color);
  padding-bottom: 0.3em;
}

.content h2 {
  font-size: var(--font-size-h2);
  margin-top: 2em;
}

.content h3 {
  font-size: var(--font-size-h3);
}
```

### 代码块

```css
.content code {
  background: var(--code-background);
  color: var(--code-text);
  padding: 0.2em 0.4em;
  border-radius: 3px;
  font-family: var(--font-family-mono);
  font-size: var(--font-size-code);
}

.content pre {
  background: var(--code-background);
  border: 1px solid var(--border-color);
  border-radius: 5px;
  padding: 1em;
  overflow-x: auto;
  line-height: var(--line-height-code);
}

.content pre code {
  background: none;
  color: inherit;
  padding: 0;
}
```

### 列表

```css
.content ul,
.content ol {
  margin-left: 2em;
  margin-bottom: 1em;
}

.content li {
  margin-bottom: 0.5em;
}

.content ul li {
  list-style-type: disc;
}

.content ol li {
  list-style-type: decimal;
}
```

### 表格

```css
.content table {
  width: 100%;
  border-collapse: collapse;
  margin: 1.5em 0;
}

.content thead {
  background: var(--background-alt);
}

.content th {
  border: 1px solid var(--border-color);
  padding: 0.75em;
  text-align: left;
  font-weight: 600;
  color: var(--primary-color);
}

.content td {
  border: 1px solid var(--border-color);
  padding: 0.75em;
}

.content tbody tr:nth-child(even) {
  background: var(--background-alt);
}
```

### 链接

```css
.content a {
  color: var(--primary-color);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s;
}

.content a:hover {
  border-bottom-color: var(--primary-color);
}

.content a:visited {
  color: #7b2cbf;
}
```

### 侧边栏导航

```css
.sidebar {
  background: var(--background-alt);
  border-right: 1px solid var(--border-color);
  padding: var(--content-padding);
}

.sidebar .nav-item {
  color: var(--text-color);
}

.sidebar .nav-item.active {
  background: var(--primary-color);
  color: white;
  border-radius: 4px;
}

.sidebar .nav-item a {
  color: inherit;
  text-decoration: none;
}
```

### 页面目录

```css
.page-toc {
  background: var(--background-alt);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: var(--spacing-md);
  margin: 2em 0;
}

.page-toc .toc-title {
  font-weight: 600;
  color: var(--primary-color);
  margin-bottom: 1em;
}

.page-toc ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.page-toc li {
  margin: 0.5em 0 0.5em var(--spacing-lg);
}

.page-toc a {
  color: var(--primary-color);
  text-decoration: none;
}
```

### 标注框

```css
.callout {
  border-left: 4px solid var(--primary-color);
  background: var(--background-alt);
  padding: var(--spacing-md);
  margin: 1.5em 0;
  border-radius: 2px;
}

.callout.info {
  border-left-color: var(--info-color);
}

.callout.warning {
  border-left-color: var(--warning-color);
}

.callout.danger {
  border-left-color: var(--danger-color);
}

.callout.success {
  border-left-color: var(--success-color);
}

.callout-title {
  font-weight: 600;
  color: var(--primary-color);
  margin-bottom: 0.5em;
}
```

## 响应式断点

mdPress 使用标准响应式断点。为移动、平板电脑和桌面布局定义媒体查询：

### 手机 (< 768px)

```css
@media (max-width: 767px) {
  .content {
    padding: var(--spacing-md);
    font-size: 15px;
  }

  .sidebar {
    display: none;
  }

  .page-toc {
    margin: 1em 0;
    padding: var(--spacing-sm);
  }

  .content h1 {
    font-size: 1.75em;
  }

  .content h2 {
    font-size: 1.5em;
  }
}
```

### 平板电脑 (768px - 900px)

```css
@media (min-width: 768px) and (max-width: 899px) {
  .content {
    max-width: 700px;
  }

  .sidebar {
    width: 200px;
  }

  .page-toc {
    font-size: 14px;
  }
}
```

### 桌面 (> 960px)

```css
@media (min-width: 960px) {
  .content {
    max-width: 900px;
  }

  .sidebar {
    width: 250px;
  }

  .right-sidebar {
    width: 280px;
  }
}
```

## 代码高亮主题自定义

自定义代码块的语法高亮颜色：

```css
/* 覆盖 Monokai 主题颜色 */
.hljs {
  background: #272822;
  color: #f8f8f2;
}

.hljs-string {
  color: #e6db74;
}

.hljs-number {
  color: #ae81ff;
}

.hljs-literal {
  color: #ae81ff;
}

.hljs-attr {
  color: #a6e22e;
}

.hljs-keyword {
  color: #f92672;
}

.hljs-function {
  color: #a6e22e;
}

.hljs-comment {
  color: #75715e;
}
```

## 完整示例

这是自定义 Technical 主题的完整示例：

```yaml
book:
  title: "API Documentation"
  author: "Development Team"

style:
  theme: technical
  custom_css: |
    :root {
      --primary-color: #0056b3;
      --accent-color: #fd7e14;
      --font-family-base: "Inter", sans-serif;
      --font-size-base: 17px;
    }

    .content {
      font-feature-settings: "kern" 1;
      text-rendering: optimizeLegibility;
    }

    .content h1 {
      background: linear-gradient(135deg, var(--primary-color) 0%, #004085 100%);
      color: white;
      padding: 0.5em 1em;
      border-radius: 4px;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .content code {
      background: #f8f9fa;
      color: #d63384;
      border: 1px solid #dee2e6;
    }

    .callout {
      background: linear-gradient(90deg, var(--background-alt) 0%, white 100%);
      box-shadow: 0 2px 4px rgba(0,0,0,0.05);
    }

    @media (max-width: 767px) {
      .content h1 {
        font-size: 1.5em;
      }
    }
```

参见 [内置主题](./builtin-themes.md) 了解默认主题变量和结构。
