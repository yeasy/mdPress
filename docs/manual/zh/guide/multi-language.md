# 多语言书籍

多语言项目就是一个包含 `LANGS.md` 以及每种语言一个子目录的目录。**每个语言目录都是一个普通的、自包含的 mdPress 项目。** mdPress 会逐个构建它们，并在旁边写一个简单的着陆页。

不存在 `multiLanguage`、`languages`、`defaultLanguage`、`languageSwitcher` 之类的配置项。`LANGS.md` 就是全部机制。

## 目录布局

```
book/
├── LANGS.md          <- 让这个目录成为多语言项目的唯一标志
├── en/
│   ├── book.yaml
│   ├── README.md
│   └── guide.md
└── zh/
    ├── book.yaml
    ├── README.md
    └── guide.md
```

**不要**在根目录放 `book.yaml`。mdPress 会先加载根配置再去看 `LANGS.md`，而一个自身没有章节的根 `book.yaml` 会让构建失败：

```
Error: failed to load config: config validation failed: at least one chapter is required
```

## LANGS.md

```markdown
# Languages

- [English](en/)
- [中文](zh/)
```

解析规则（与实现完全一致）：

- 任何包含 Markdown 链接 `[名称](目录)` 的行都会成为一种语言。以 `#` 开头的行和空行会被跳过，所以那个标题只是装饰。
- 链接文本是语言切换器中显示的名称。
- **链接目标必须是语言目录。** 结尾的斜杠可有可无。绝对路径和 `..` 会被拒绝。
- 如果指向文件（`[English](en/README.md)`），构建会失败：`stat …/en/README.md/<Title>_site: not a directory`。
- 列表符号无所谓，`-` 和 `*` 都可以。

## 每种语言的配置

每个语言目录都按与独立项目相同的规则被发现：先 `book.yaml`，再 `book.json`，再 `SUMMARY.md`，最后是对目录下 `.md` 文件的零配置发现。

语言目录的 `book.yaml` 使用 mdPress 的正常 schema——是 `book:` 块，而不是顶层的 `title:`/`language:`：

`en/book.yaml`：

```yaml
book:
  title: "My Docs"
  author: "Your Name"
  description: "Complete guide to my project"
  language: "en-US"

chapters:
  - title: "Intro"
    file: "README.md"
  - title: "Guide"
    file: "guide.md"
```

`zh/book.yaml`：

```yaml
book:
  title: "我的文档"
  author: "你的名字"
  description: "我的项目的完整指南"
  language: "zh-CN"

chapters:
  - title: "简介"
    file: "README.md"
  - title: "指南"
    file: "guide.md"
```

把 `title:` 写在顶层而不是 `book:` 下面，会得到 unknown key 警告，并且书名变成 "Untitled Book"。

`book.language` 接受完整的 locale 标签，例如 `en-US` 或 `zh-CN`，它决定界面文案与封面。省略时 mdPress 会按目录名猜测（`en/` → 英文，`zh/` → 中文）。

其余配置——`style:`、`output:`、`plugins:`——都是按语言独立的。两种语言可以用不同主题。

## 构建

```bash
cd book
mdpress build --format site
```

mdPress 会逐个构建每种语言。产物落在**各自的语言目录内部**：

```
book/
├── _mdpress_langs.html          <- 列出各语言的着陆页
├── en/
│   └── My Docs_site/            <- 名字取自书名，或 output.filename
│       └── index.html
└── zh/
    └── 我的文档_site/
        └── index.html
```

文件类格式同理——`mdpress build --format pdf` 会生成 `en/My Docs.pdf` 与 `zh/我的文档.pdf`。在某个语言的 `book.yaml` 里设置 `output.filename` 可以控制这个名字：

```yaml
output:
  filename: "en-docs"     # -> en/en-docs.pdf 和 en/en-docs_site/
```

生成的每个站点页面顶部还会注入一条语言切换栏，链接到其他语言和着陆页。

### 整项目构建的已知限制

整项目构建适合本地预览，但**目前还不适合**直接拿去部署：

- 着陆页叫 `_mdpress_langs.html` 而不是 `index.html`，Web 服务器在站点根路径上并不会提供它。
- 站点目录名取自书**标题**，因此标题里的空格或中文会进入 URL 路径（`en/My%20Docs_site/`）。
- 着陆页与切换栏里的链接没有做百分号编码，所以上述标题会产生打不开的链接。
- `--output ./dist` **不会**创建 `dist/`。它会在项目旁边生成 `dist-en_site/`、`dist-zh_site/` 和 `dist-index.html`，并且各语言自己的 `output.filename` 会被忽略。

要得到可部署的产物，请用下面的按语言构建方式。

### 构建单一语言（部署推荐做法）

没有 `--lang` 参数。把每个语言目录当作独立项目构建，并自己指定输出路径。**给 `--output` 加上结尾斜杠**，这样 mdPress 才会把它当目录而不是文件基名：

```bash
mdpress build ./en --format site --output ./dist/en/
mdpress build ./zh --format site --output ./dist/zh/
```

这会得到正是你想部署的布局：

```
dist/
├── en/
│   ├── index.html
│   ├── guide.html
│   └── search-index.json
└── zh/
    ├── index.html
    ├── guide.html
    └── search-index.json
```

不加结尾斜杠时，`--output ./dist/en` 会被当作文件基名，站点会落到 `dist/en_site/`。

这也是只重建某一种语言而不动其他语言的方式。

### 预览单一语言

`mdpress serve` 没有多语言模式。请把它指向某个语言目录：

```bash
mdpress serve ./en
```

## 跨语言链接

跨语言链接就是普通相对链接，mdPress **不会**改写它们；从 `en/guide.md` 指向 `../zh/guide.md` 的链接落在英文书的构建图之外，`mdpress doctor` 会报告：

```
⚠ Detected 1 Markdown link(s) outside the build graph
    - ../zh/guide.md (from guide.md)
```

要让链接在部署后的站点里可用，请直接写部署后的 URL：

```markdown
[English version](/en/guide.html)
```

## 部署

用上面的按语言构建方式，`dist/` 就是一棵普通的静态站点目录树：

```
https://docs.example.com/en/
https://docs.example.com/zh/
```

如果需要根路径跳转到默认语言，请自己加一个 `dist/index.html`——mdPress 目前还生成不了可用的那一个。

每种语言也可以单独部署到各自的域名或存储桶，因为每个 `dist/<lang>/` 都是自包含的。

## 排查

| 现象 | 原因 |
| --- | --- |
| 根目录报 `at least one chapter is required` | 项目根目录存在 `book.yaml`。删掉它；配置应放在各语言目录里。 |
| `stat …/en/README.md/…: not a directory` | `LANGS.md` 的某一项指向了文件。请改为指向目录：`[English](en/)`。 |
| `no language definitions found in LANGS.md` | `LANGS.md` 里没有任何一行包含 Markdown 链接。 |
| 书名显示为 "Untitled Book" | 语言目录的 `book.yaml` 用了顶层 `title:` 而不是 `book: { title: … }`。 |
| 站点界面语言不对 | 在该语言的 `book.yaml` 里设置 `book.language`（`en-US`、`zh-CN` 等）。 |
| `build --output ./dist` 之后没有 `dist/` | 整项目模式下这是预期行为。请按上面的方式逐语言构建。 |
