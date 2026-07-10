# 内置主题

mdPress 提供三个专业设计的内置主题，覆盖常见的文档使用场景。每个主题都有独特的视觉特征，并针对特定的内容类型进行了优化。

## 可用主题

### Technical 主题

Technical 主题是默认主题，为 API 文档、技术指南和代码密集型内容而设计。

**特征：**
- 海军蓝墨色调：标题 `#12344D`，链接与强调色 `#1C5A9E`
- 简洁的无衬线字体，优化了可读性
- 代码块使用 `github` 语法高亮，配浅色底
- 发丝级表格边框，着色表头与斑马纹
- 默认封面为深海军蓝整版底色

**最适合：**
- API 参考文档
- 编程教程
- 技术规范
- 命令行工具文档

```yaml
style:
  theme: technical
```

### Elegant 主题

Elegant 主题为叙述性文档、书籍和正式出版物而设计。

**特征：**
- 暖色衬线色调：正文 `#3E2723`，暖白背景 `#FFFBF0`，青铜强调色 `#A87B3B`
- 正文使用优美的衬线字体栈
- 暖色发丝级边框，行距充裕
- `github` 代码高亮，配暖色调底色
- 默认封面为深暖棕色

**最适合：**
- 书籍和长篇幅文档
- 包含叙述部分的用户手册
- 学术或正式文档
- 需要优雅呈现的出版物

```yaml
style:
  theme: elegant
```

### Minimal 主题

Minimal 主题是安静的单色设计，优先考虑内容清晰度和打印友好性。

**特征：**
- 近黑墨色：文本与标题 `#000000`，强调色 `#1A1A1A`
- 灰度 `bw` 代码高亮——代码块不使用彩色
- 装饰元素极少，留白充分
- 针对打印优化，无不必要的颜色
- 默认封面为浅色配深色文字

**最适合：**
- 打印出版物
- 可访问性关键的文档
- 专业黑白输出
- 政府或正式文档

```yaml
style:
  theme: minimal
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

每个主题自带代码高亮配色（technical 和 elegant 为 `github`，minimal 为 `bw`）。`style.code_theme` 留空即继承主题的配色；显式设置可覆盖：

```yaml
style:
  theme: minimal
  code_theme: github   # 覆盖 minimal 的灰度代码配色
```

## 主题感知封面

当 `book.cover.image` 和 `book.cover.background` 均未设置时，默认封面跟随主题：`technical` 为深海军蓝，`elegant` 为深暖棕色，`minimal` 为浅色配深色文字。把 `cover.background` 设为浅色（包括 `white` 或颜色名/`rgb()` 形式）时，封面文字会自动切换为深色。

## 自定义主题

你可以用一个 YAML 文件定义自己的主题，有两种使用方式：

1. **项目主题目录** —— 项目内 `themes/<name>.yaml`（或 `.yml`）文件定义主题 `<name>`，也可以覆盖同名的内置主题。放好 `themes/corporate.yaml` 后，在 `book.yaml` 中选择它：

   ```yaml
   style:
     theme: corporate
   ```

2. **直接文件路径** —— 让 `style.theme` 指向一个 YAML 主题文件（相对于 `book.yaml`）：

   ```yaml
   style:
     theme: mytheme.yaml
   ```

主题文件的字段与内置主题一致。仓库中的 `themes/technical.yaml` 是一份完整示例：

```yaml
name: mytheme
page_size: A4
font_family: "'Georgia', 'Times New Roman', serif"
font_size: 12
code_theme: github
line_height: 1.75
colors:
  text: "#1F2933"
  background: "#FFFFFF"
  heading: "#12344D"
  link: "#1C5A9E"
  code_bg: "#F5F7F9"
  code_text: "#1F2933"
  accent: "#1C5A9E"
  border: "#E4E7EB"
margins:
  top: 20.0
  bottom: 20.0
  left: 20.0
  right: 20.0
```

## 主题管理命令

mdPress 提供 CLI 命令来探索和预览主题。这些命令的输出直接来自实际的主题调色板，打印的内容与构建使用的一致。

### 列出可用主题

```bash
mdpress themes list
```

列出每个主题的描述、关键颜色（标题 / 链接 / 强调色 / 背景）和页面属性。

### 显示主题详情

```bash
mdpress themes show technical
```

打印主题的描述、排版（字体族、字号、行高、代码配色）、完整调色板和一段 `book.yaml` 示例。

### 预览所有主题

```bash
mdpress themes preview
# 输出：themes-preview.html
```

生成一个自包含的 HTML 文件，用与构建管线完全相同的样式表展示所有内置主题的示例内容。可用 `-o, --output <path>` 指定其他输出位置。

## 自定义内置主题

除了完整的自定义主题，你也可以通过 `style.custom_css`（指向一个 CSS 文件）在任意主题之上叠加 CSS 覆盖：

```yaml
style:
  theme: technical
  custom_css: styles/overrides.css
```

参见[自定义 CSS 指南](./custom-css.md)了解详细的自定义选项。

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
