# 输出格式

mdPress 可以以多种格式生成你的文档，每种格式都针对不同的使用场景进行了优化。你可以生成单一格式或使用单一命令同时生成多种格式。

## 网站输出

网站格式生成一个具有客户端导航和高级功能的现代多页面静态网站。

### 网站功能

网站输出包括：

- **多页面结构**：每一章变成独立的 HTML 页面以实现快速加载
- **SPA 导航**：单页面应用程序风格的导航，无需完整页面重新加载
- **全文搜索**：跨所有内容的客户端搜索，带键盘快捷方式 (Cmd/Ctrl+K)
- **深色模式**：自动深色/浅色主题切换，用户偏好持久化
- **响应式设计**：移动友好的布局，适应所有屏幕尺寸
- **目录**：自动侧边栏，带章节层次结构和当前页面突出显示
- **社交分享**：内置社交平台分享的元数据

### 构建网站

使用默认配置生成网站：

```bash
mdpress build --format site
```

指定输出目录：

```bash
mdpress build --format site --output ./public
```

输出目录结构将为：

```
public/
├── index.html
├── chapter-1/
│   └── index.html
├── chapter-2/
│   └── index.html
├── assets/
│   ├── style.css
│   ├── script.js
│   └── images/
├── search-index.json
└── sitemap.xml
```

### SPA 导航

网站使用单页面应用程序架构。当用户在章节之间导航时：

1. 仅内容改变，页面结构不改变
2. 不发生完整页面重新加载
3. 滚动位置自动管理
4. 浏览器历史记录得以维护，以便后退/前进导航

这提供了类似于本地应用程序的快速、流畅的体验。

### 搜索功能

网站包含由搜索索引提供支持的客户端全文搜索。用户可以：

- 按 Cmd+K (Mac) 或 Ctrl+K (Windows/Linux) 打开搜索对话框
- 在所有章节和部分中搜索
- 点击结果跳转到相关部分
- 在内容中看到突出显示的搜索词

搜索索引在构建过程中自动生成。

### 深色模式

用户可以在浅色和深色主题之间切换。偏好自动保存在浏览器本地存储中，并在会话间持久化。你可以在 `book.yaml` 配置中自定义主题颜色。

## PDF 输出

生成适合打印和分享的高质量 PDF 文档。

### PDF 生成方法

mdPress 支持两个 PDF 生成引擎：

#### 基于 Chromium 的 PDF

使用无头 Chromium 的默认方法：

```bash
mdpress build --format pdf
```

功能：
- 高保真渲染，匹配浏览器显示
- 支持 CSS 打印样式
- 快速生成
- 适用于大多数文档

#### 基于 Typst 的 PDF

使用 Typst 排版引擎的替代方法：

```bash
mdpress build --format typst
```

功能：
- 专业排版
- 更好的布局控制
- 更小的文件大小
- 在大型文档上性能更好

### PDF 配置

在 `book.yaml` 中配置 PDF 输出：

```yaml
style:
  page_size: "A4"        # Page size
output:
  margin_top: "20mm"      # Top margin
  margin_bottom: "20mm"   # Bottom margin
  margin_left: "15mm"     # Left margin
  margin_right: "15mm"    # Right margin
  header_template: "<span class='title'></span>"
  footer_template: "<span class='page'></span> of <span class='pageCount'></span>"
```

### PDF 功能

mdPress 生成的 PDF 包括：

- **书签**：目录创建可点击的 PDF 书签用于导航
- **水印**：可选的水印文本，用于草稿版本或机密文档
- **页眉和页脚**：自动页码和自定义文本
- **嵌入字体**：专业字体渲染
- **超链接**：内部链接和外部 URL 保持可点击状态
- **图像和图表**：对图像、Mermaid 和 PlantUML 图表的完整支持

### 添加水印

在 `book.yaml` 中配置水印：

```yaml
output:
  watermark: "DRAFT"
  watermark_opacity: 0.3
```

## HTML 输出

生成带有所有内容和样式嵌入的单个自包含 HTML 文件。

### 创建单文件 HTML

```bash
mdpress build --format html
```

输出是包含以下内容的单个 `index.html` 文件：

- 所有内容和结构
- 嵌入的 CSS 样式
- 用于交互性的嵌入式 JavaScript
- Base64 编码的图像
- 无外部依赖

### 单文件 HTML 的用途

单文件 HTML 适用于：

- 通过电子邮件分享文档
- 将文档作为完整单元进行存档
- 离线分发
- 从浏览器打印到纸张或 PDF
- 需要零外部依赖的系统

### HTML 功能

单文件 HTML 输出包括：

- 侧边栏中的完整导航
- 搜索功能（数据嵌入在文件中）
- 深色模式支持
- 响应式设计
- 网站格式的所有交互功能

### 文件大小考虑

单文件 HTML 包含所有内容，可能导致更大的文件。对于包含许多图像或大型图表的文档，考虑：

- 压缩 HTML 文件
- 使用网站格式进行网络交付
- 将非常大的文档分成多个 HTML 文件

## ePub 输出

为 Kindle、Apple Books 和其他设备等电子阅读器生成 ePub 文件。

### 创建 ePub

```bash
mdpress build --format epub
```

输出是包含以下内容的 `index.epub` 文件：

- 标准 ePub 3.0 格式
- 作为单独文档的所有章节
- 带导航的目录
- 嵌入的字体和图像
- 对可重排文本的支持

### ePub 功能

ePub 文件支持：

- **可重排布局**：文本适应阅读器屏幕大小
- **目录**：可点击导航
- **书签**：阅读器可以保存进度
- **全文搜索**：内置在大多数电子阅读器中
- **代码块**：以等宽字体显示
- **数学方程**：在大多数电子阅读器中渲染为图像
- **图像和图表**：嵌入并显示

### ePub 配置

在 `book.yaml` 中配置 ePub 输出：

```yaml
book:
  title: "书名"
  author: "你的名字"
  language: "zh"
  cover:
    image: "./assets/cover.png"
```

### ePub 限制

某些功能在 ePub 格式中有限制：

- 交互式元素转换为静态内容
- 复杂的 CSS 布局可能不会完全相同地呈现
- 不支持 JavaScript 功能
- 某些图表类型可能呈现为图像

## 构建多种格式

使用单一命令生成多种输出格式。

### 构建所有格式

```bash
mdpress build --format all
```

这生成：网站、PDF、HTML、ePub 和 Typst。

### 构建特定格式

```bash
mdpress build --format pdf,html,epub
```

逗号分隔的格式名称仅生成指定的格式。

### 并行构建

构建多种格式时，mdPress：

1. 仅解析一次你的文档
2. 处理常见转换
3. 并行应用格式特定的渲染
4. 同时输出所有格式

这比分别构建每种格式更快。

### 构建多种格式

常见场景的示例命令：

```bash
# 网络和 PDF 用于分发
mdpress build --format site,pdf --output ./dist

# 所有格式用于全面分发
mdpress build --format all --output ./dist

# 仅 PDF 用于打印
mdpress build --format pdf --output ./print
```

### 输出组织

构建多种格式到同一目录时：

```
dist/
├── site/
│   ├── index.html
│   ├── chapter-1/
│   ├── assets/
│   └── search-index.json
├── index.html (单文件 HTML)
├── index.epub
├── index.pdf
└── build-report.json
```

## 输出自定义

### 自定义网站

修改 `book.yaml` 以自定义网站外观：

```yaml
style:
  theme: "technical"    # 或 "elegant" 或 "minimal"
  custom_css: "./assets/custom.css"  # 可选自定义 CSS
```

### 资源管理

在 `assets` 目录中放置静态资源：

```
book/
├── book.yaml
├── SUMMARY.md
├── chapter-1.md
├── chapter-2.md
└── assets/
    ├── logo.svg
    ├── favicon.ico
    └── images/
        ├── screenshot-1.png
        └── diagram.svg
```

资源会自动复制到输出目录，并在 `assets/` 路径中可用。

## 部署考虑

### 网站部署

网站格式适合网络部署：

```bash
# 为生产环境构建
mdpress build --format site --output ./public

# 部署到网络服务器
rsync -av public/ user@server:/var/www/docs/
```

### PDF 分发

PDF 适合电子邮件和下载：

```bash
# 使用特定名称生成
mdpress build --format pdf --output "./dist/Manual-v1.0.pdf"
```

### 静态托管

网站格式仅需要静态文件托管：

- GitHub Pages
- Netlify
- Vercel
- 任何静态主机（AWS S3、Cloudflare Pages 等）

无需服务器端处理。

## 故障排除

### PDF 生成失败

- 确保如果使用默认引擎，已安装 Chromium
- 检查可用磁盘空间
- 如果 Chromium 出现问题，尝试 Typst 引擎
- 查看错误消息以了解特定缺失的依赖

### 大文件大小

- 考虑压缩单文件 HTML 输出
- 分割非常大的文档
- 在包含之前优化图像
- 使用网站格式进行网络分发，而不是单个 HTML 文件

### 搜索不工作

- 验证搜索索引在构建输出中生成
- 检查浏览器控制台的 JavaScript 错误
- 确保在目标环境中启用了 JavaScript
- 对于单文件 HTML，在不同浏览器中测试

### 特定格式中的呈现问题

- 在不同输出格式中测试有问题的内容
- 某些功能（如交互元素）在所有格式中都不工作
- 为 PDF 输出简化复杂的 CSS
- 在 ePub 格式中为高级功能使用备选方案
