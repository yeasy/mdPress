# 多语言书籍

mdPress 支持使用自动语言切换和每种语言版本的单独构建来创建多语言文档。本指南解释了如何设置和管理多语言文档。

## 多语言结构

多语言书籍使用特定的目录结构和配置文件来按语言组织内容。

### 目录布局

像这样组织多语言文档：

```
book/
├── LANGS.md
├── book.yaml
├── en/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── chapter-1.md
│   ├── chapter-2.md
│   └── assets/
├── zh/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── chapter-1.md
│   ├── chapter-2.md
│   └── assets/
└── ja/
    ├── book.yaml
    ├── SUMMARY.md
    ├── chapter-1.md
    ├── chapter-2.md
    └── assets/
```

每种语言都有其完整的文档内容和配置的目录。

## LANGS.md 配置

`LANGS.md` 文件定义文档中的所有可用语言。

### LANGS.md 格式

在根目录中创建 `LANGS.md`：

```markdown
# Languages

- [English](en/README.md)
- [中文](zh/README.md)
- [日本語](ja/README.md)
```

或更详细的格式：

```markdown
# Languages

* [English - English Documentation](en/)
* [中文（简体）- Simplified Chinese](zh/)
* [繁體中文 - Traditional Chinese](zh-tw/)
* [日本語 - Japanese](ja/)
```

### 语言结构

格式为 Markdown，包含：
- 主标题 (H1)："Languages"
- 每种语言的链接列表
- 显示给用户的链接文本
- 链接目标指向语言目录或 README

## 每种语言的配置

每种语言需要其自己带有语言特定设置的 `book.yaml` 文件。

### 特定语言的 book.yaml

在 `en/book.yaml` 中：

```yaml
title: "My Project Documentation"
description: "Complete guide to my project"
language: "en"
authors:
  - "Your Name"

# Language-specific metadata
site:
  baseUrl: "https://docs.example.com/en/"

search:
  languages: ["en"]
```

在 `zh/book.yaml` 中：

```yaml
title: "我的项目文档"
description: "我的项目的完整指南"
language: "zh"
authors:
  - "你的名字"

site:
  baseUrl: "https://docs.example.com/zh/"

search:
  languages: ["zh"]
```

在 `ja/book.yaml` 中：

```yaml
title: "マイプロジェクトドキュメント"
description: "マイプロジェクトの完全ガイド"
language: "ja"
authors:
  - "あなたの名前"

site:
  baseUrl: "https://docs.example.com/ja/"

search:
  languages: ["ja"]
```

### 必需的语言设置

每种语言的 `book.yaml` 应该有：

- `language`：ISO 639-1 语言代码（en、zh、ja、fr、de 等）
- `title`：目标语言的标题
- `description`：目标语言的描述
- `search.languages`：与语言代码匹配

## 根 book.yaml 配置

根 `book.yaml` 定义整个多语言项目的全局设置。

### 根配置

在你的根 `book.yaml` 中：

```yaml
title: "Multi-Language Documentation"
description: "Documentation in multiple languages"

# Multi-language configuration
multiLanguage: true
languages:
  - code: "en"
    name: "English"
    region: "US"
  - code: "zh"
    name: "中文"
    region: "CN"
  - code: "ja"
    name: "日本語"
    region: "JP"

# Default language for the root
defaultLanguage: "en"

# Language switcher configuration
languageSwitcher:
  enabled: true
  position: "top-right"
```

### 语言代码

使用标准 ISO 639-1 语言代码：

- `en`：英文
- `zh`：中文（简体）
- `zh-tw`：中文（繁体）
- `ja`：日文
- `ko`：韩文
- `fr`：法文
- `de`：德文
- `es`：西班牙文
- `ru`：俄文
- `ar`：阿拉伯文
- `pt`：葡萄牙文
- `it`：意大利文

## 构建多语言文档

使用命令行标志构建所有语言或特定语言。

### 构建所有语言

```bash
mdpress build
```

构建 `LANGS.md` 中指定的所有语言并创建：

```
dist/
├── en/
│   ├── index.html
│   ├── chapter-1/
│   └── ...
├── zh/
│   ├── index.html
│   ├── chapter-1/
│   └── ...
└── ja/
    ├── index.html
    ├── chapter-1/
    └── ...
```

### 构建特定语言

```bash
mdpress build --lang en,ja
```

仅构建英文和日文，跳过中文。

### 使用输出目录构建

```bash
mdpress build --output ./dist
```

将所有语言版本输出到 `./dist` 目录，结构如上。

## 每种语言的构建

如果需要，单独构建各个语言。

### 构建单一语言

```bash
cd en
mdpress build
```

或从根目录：

```bash
mdpress build --lang en --output ./dist/en
```

这对于将各个语言版本部署到单独的服务器或域很有用。

### 特定语言的输出

构建单一语言时，结构更简单：

```
dist/
├── index.html
├── chapter-1/
│   └── index.html
├── chapter-2/
│   └── index.html
└── assets/
```

### 增量构建

对于大型多语言项目，仅重建更改的语言：

```bash
mdpress build --lang zh
```

仅重建中文版本，保持其他语言不变。

## 语言切换

用户可以在文档界面中切换语言。

### 语言切换器 UI

语言切换器出现在网站界面中（位置可在 `book.yaml` 中配置）：

- 通常位于标题或侧边栏中
- 显示所有可用语言的本地名称
- 点击切换到所选语言
- 突出显示当前语言

### 语言切换后的导航

切换语言时：

1. 用户点击语言切换器
2. 浏览器导航到新语言中的同一页面
3. 如果该页面在新语言中不存在，导航到主页
4. 语言偏好保存在浏览器本地存储中

### 语言间的深层链接

从一种语言链接到另一种语言中的相应页面：

在英文文档中：
```markdown
[日本語版](../ja/chapter-1.md)
```

在日文文档中：
```markdown
[English Version](../en/chapter-1.md)
```

mdPress 自动根据语言上下文重写这些链接。

## 跨语言管理内容

### 同步更新

更新英文文档时，需要更新翻译：

1. 在 `en/chapter-1.md` 中更新源内容
2. 在 `zh/chapter-1.md` 中更新相应文件
3. 在 `ja/chapter-1.md` 中更新相应文件
4. 重建文档

### 维护一致性

跨所有语言使用相同的文件名：

```
en/chapter-1.md
zh/chapter-1.md
ja/chapter-1.md
```

这使得识别相应的部分变得容易。

### 部分语言支持

你可以有不完整的语言版本。例如：

- 英文：100% 完整
- 日文：80% 完整
- 中文：50% 完整

语言切换器仍然显示所有语言。点击不可用语言的用户被定向到该语言中的主页。

### 翻译工作流

管理翻译的推荐工作流：

1. 创建英文文档
2. 提交并发布英文版本
3. 创建从英文复制结构的语言目录
4. 逐部分翻译
5. 提交翻译部分
6. 重建以包括所有可用语言

## 多语言设置示例

### 完整示例

这是一个包含三种语言的完整示例：

创建目录结构：

```bash
mkdir -p book/{en,zh,ja}
```

创建 `book/LANGS.md`：

```markdown
# Languages

- [English](en/)
- [中文](zh/)
- [日本語](ja/)
```

创建 `book/book.yaml`：

```yaml
title: "Multi-Language Docs"
multiLanguage: true
languages:
  - code: "en"
    name: "English"
  - code: "zh"
    name: "中文"
  - code: "ja"
    name: "日本語"
defaultLanguage: "en"
```

创建 `book/en/book.yaml`：

```yaml
title: "Documentation"
language: "en"
```

创建 `book/en/SUMMARY.md`：

```markdown
# Summary

- [Introduction](./README.md)
- [Getting Started](./getting-started.md)
- [Configuration](./configuration.md)
```

创建 `book/en/README.md`：

```markdown
# Welcome

Welcome to the documentation.
```

然后将结构复制到 `zh/` 和 `ja/` 目录并翻译内容。

构建所有语言：

```bash
cd book
mdpress build --output ../dist
```

## 部署

### 基于语言的路由

将所有语言部署到同一域，使用基于路径的路由：

```
https://docs.example.com/en/
https://docs.example.com/zh/
https://docs.example.com/ja/
```

配置你的网络服务器或 CDN 以适当地路由请求。

### 单独的域

将每种语言部署到单独的域：

```
https://en.docs.example.com/
https://zh.docs.example.com/
https://ja.docs.example.com/
```

在每种语言的 `book.yaml` 中相应地更新 `baseUrl`。

### 重定向和本地化

对于根域，将用户重定向到他们的首选语言：

```
https://docs.example.com/ → https://docs.example.com/en/ (对于英文用户)
https://docs.example.com/ → https://docs.example.com/zh/ (对于中文用户)
```

使用以下方法实现：
- 网络服务器配置
- Service Worker 重定向
- 基于浏览器语言的 JavaScript 重定向

## 故障排除

### 语言未构建

验证：
1. 语言目录存在
2. `book.yaml` 存在于语言目录中
3. `SUMMARY.md` 存在于语言目录中
4. `book.yaml` 中的语言代码与 `LANGS.md` 匹配

### 语言间的断开链接

检查：
1. 文件名跨语言目录匹配
2. 链接中的相对路径考虑语言前缀
3. 所有引用的文件存在于目标语言中

### 语言切换器未出现

在 `book.yaml` 中验证：
- `multiLanguage: true` 已设置
- `languageSwitcher.enabled: true` 已设置
- 多种语言列在 `languages` 中

### 内容不一致

当内容在语言版本之间分散时，记录差异：

```markdown
> 注意：此功能仅在英文文档中提供。
> 请参阅 [英文版本](../en/chapter-1.md#feature-name)。
```

保持翻译更改记录以供参考。
