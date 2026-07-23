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
- 条目可以指向目录（`[English](en/)`），也可以指向目录里的文件（`[English](en/README.md)`）；指向文件时会解析到它所在的目录。
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

mdPress 会逐个构建每种语言，产物汇入**同一棵输出树**：

```
book/
└── _book/                   <- 整个可部署站点
    ├── index.html           <- 语言切换页（站点根）
    ├── en/
    │   └── index.html
    └── zh/
        └── index.html
```

用 `--output` 可以构建到别处。它指定的是整棵树的根，所以 `./dist` 和 `./dist/` 等价：

```bash
mdpress build --format site --output ./dist
# -> dist/index.html、dist/en/index.html、dist/zh/index.html
```

看起来像文件的路径会解析成它本该所在的目录——`--output ./dist/book.html` 会构建到 `dist/book/`——因为多语言构建产出的是一棵树，不是单个文件。

文件类格式与各语言站点并列——`mdpress build --format pdf` 会生成 `_book/en/<name>.pdf`。在某个语言的 `book.yaml` 里设置 `output.filename` 可以控制这个名字：

```yaml
output:
  filename: "en-docs"     # -> _book/en/en-docs.pdf
```

生成的每个站点页面顶部还会注入一条语言切换栏，链接到其他语言和站点根。

目录名取自 `LANGS.md` 而不是书名，所以标题里的空格或中文永远不会进入 URL 路径。切换栏里的链接都做了百分号编码。

`LANGS.md` 旁边可以有一个根 `book.yaml`，用来放共享元数据；它不需要 `chapters`，因为每个语言目录都有自己的。

### 构建单一语言

没有 `--lang` 参数，但语言目录本身就是一个完整项目，直接指向它即可：

```bash
mdpress build ./en --format site --output ./dist/en
```

需要各语言分开部署时用这种方式。想要一棵覆盖所有语言的可部署树，用上面的整项目构建。

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

输出树就是一棵普通的静态站点，可以直接上传：

```
https://docs.example.com/          <- 语言切换页
https://docs.example.com/en/
https://docs.example.com/zh/
```

根路径上的切换页是真正的 `index.html`，Web 服务器无需额外配置即可提供。每个 `<lang>/` 目录也是自包含的，可以单独部署到各自的域名或存储桶。

## 排查

| 现象 | 原因 |
| --- | --- |
| 根目录报 `at least one chapter is required` | 根目录有 `book.yaml` 但旁边没有 `LANGS.md`。有 `LANGS.md` 时根 `book.yaml` 是允许的，用来放共享元数据。 |
| `no language definitions found in LANGS.md` | `LANGS.md` 里没有任何一行包含 Markdown 链接。 |
| 书名显示为 "Untitled Book" | 语言目录的 `book.yaml` 用了顶层 `title:` 而不是 `book: { title: … }`。 |
| 站点界面语言不对 | 在该语言的 `book.yaml` 里设置 `book.language`（`en-US`、`zh-CN` 等）。 |
| 产物落在了意料之外的位置 | `--output` 指定的是整棵树的根。`./dist` 和 `./dist/` 都构建到 `dist/`；像文件的路径会解析成它所在的目录，`./dist/book.html` 构建到 `dist/book/`。 |
