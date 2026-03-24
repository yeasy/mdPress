# 从 GitBook 迁移

mdPress 提供迁移工具来帮助你将文档从 GitBook 或 HonKit 转换为 mdPress 格式。本指南解释了迁移过程以及转换的内容。

## 概述

mdPress 迁移命令自动转换：

- `book.json` 配置为 `book.yaml`
- GitBook 模板语法为标准 Markdown 和 HTML
- SUMMARY.md 结构（两个系统中兼容）
- 资源和媒体文件
- 插件配置为等效的 mdPress 功能

## 运行迁移

使用 `mdpress migrate` 命令转换你的 GitBook 文档。

### 基本迁移

```bash
mdpress migrate ./gitbook-project
```

这读取你的 GitBook 项目并在当前目录中使用转换的文件创建新的 mdPress 项目。

### 输出目录

指定在何处创建迁移的项目：

```bash
mdpress migrate ./gitbook-project --output ./mdpress-project
```

在 `./mdpress-project` 中创建 mdPress 项目，包含所有转换的内容。

### 模拟运行模式

预览迁移而不进行更改：

```bash
mdpress migrate ./gitbook-project --dry-run
```

显示将转换的内容、可能丢失信息的内容以及需要手动注意的内容。在模拟运行模式下不创建文件。

### 详细输出

获取迁移每个步骤的详细信息：

```bash
mdpress migrate ./gitbook-project --verbose
```

显示正在转换的文件、应用的转换以及可能问题的任何警告。

### 完整迁移示例

典型的迁移命令：

```bash
# 首先预览迁移
mdpress migrate ./my-gitbook --dry-run

# 然后执行实际迁移
mdpress migrate ./my-gitbook --output ./my-mdpress

# 详细输出以查看发生了什么
mdpress migrate ./my-gitbook --output ./my-mdpress --verbose
```

## 转换的内容

迁移过程处理几个关键转换任务。

### 配置：book.json 为 book.yaml

GitBook `book.json`：

```json
{
  "title": "My Documentation",
  "description": "Complete guide",
  "author": "John Doe",
  "output": "docs",
  "plugins": [
    "search",
    "highlight",
    "mathjax"
  ],
  "pluginsConfig": {
    "mathjax": {
      "delimiters": ["$$", "$$"]
    }
  }
}
```

转换为 mdPress `book.yaml`：

```yaml
title: "My Documentation"
description: "Complete guide"
authors:
  - "John Doe"

# Plugins converted to mdPress features
search:
  enabled: true

markdown:
  highlight: true
  math: true

# Additional mdPress configuration
site:
  baseUrl: "/docs/"
```

### 模板标签为 Markdown

像 `{% hint %}` 块这样的 GitBook 模板语法转换为标准 Markdown 引用块。

#### 提示块转换

GitBook `{% hint %} ... {% endhint %}`：

```markdown
{% hint style="warning" %}
This is a warning message.
{% endhint %}
```

转换为 mdPress 引用块：

```markdown
> **Warning:** This is a warning message.
```

#### 信息块转换

GitBook 信息块：

```markdown
{% hint style="info" %}
This is important information.
{% endhint %}
```

转换为：

```markdown
> **Info:** This is important information.
```

#### 成功块转换

GitBook 成功块：

```markdown
{% hint style="success" %}
Operation completed successfully.
{% endhint %}
```

转换为：

```markdown
> **Success:** Operation completed successfully.
```

#### 危险块转换

GitBook 危险块：

```markdown
{% hint style="danger" %}
Dangerous operation ahead.
{% endhint %}
```

转换为：

```markdown
> **Danger:** Dangerous operation ahead.
```

### 代码块转换

GitBook `{% code %}` 块转换为标准围栏代码块。

#### 代码块语法

GitBook 语法：

```
{% code title="example.js" %}
function hello() {
  console.log("Hello, world!");
}
{% endcode %}
```

转换为 mdPress：

````markdown
```javascript
// File: example.js
function hello() {
  console.log("Hello, world!");
}
```
````

#### 带语言的代码块

GitBook：

```
{% code language="python" %}
def hello():
    print("Hello, world!")
{% endcode %}
```

转换为：

````markdown
```python
def hello():
    print("Hello, world!")
```
````

### 选项卡块转换

GitBook 选项卡：

```
{% tabs %}
{% tab title="JavaScript" %}
console.log("Hello");
{% endtab %}
{% tab title="Python" %}
print("Hello")
{% endtab %}
{% endtabs %}
```

转换为带标题的 mdPress Markdown：

```markdown
### JavaScript
console.log("Hello");

### Python
print("Hello")
```

## SUMMARY.md 兼容性

SUMMARY.md 在 GitBook 和 mdPress 之间具有兼容的语法，因此需要最少的转换。

### SUMMARY.md 格式

两个系统支持相同的基本格式：

```markdown
# Summary

- [Introduction](README.md)
- [Getting Started](getting-started.md)
  - [Installation](getting-started/installation.md)
  - [Configuration](getting-started/configuration.md)
- [Advanced](advanced/README.md)
  - [API Reference](advanced/api.md)
  - [Plugins](advanced/plugins.md)
- [FAQ](faq.md)
```

此结构在 GitBook 和 mdPress 中的工作方式完全相同。

### SUMMARY.md 链接

两个系统使用相对文件路径：

```markdown
- [Chapter Title](path/to/file.md)
```

mdPress 自动解析这些链接，因此无需转换。

## 资源文件处理

图像、样式表和其他资源被复制到你的 mdPress 项目。

### 资源目录

GitBook 资源被复制到 mdPress 资源目录：

```
gitbook-project/
├── book.json
└── assets/
    ├── images/
    └── styles/

变为：

mdpress-project/
├── book.yaml
└── assets/
    ├── images/
    └── styles/
```

### 图像参考

图像引用在迁移期间自动更新：

GitBook：
```markdown
![Screenshot](assets/screenshot.png)
```

mdPress（仍然有效）：
```markdown
![Screenshot](./assets/screenshot.png)
```

## 手动调整

某些功能在迁移后需要手动审查或调整。

### 不支持的 GitBook 功能

这些功能在 mdPress 中没有直接等效：

- 高级主题定制
- 某些 GitBook 特定的插件
- 自定义 JavaScript 插件
- 复杂的布局配置

对于这些，迁移工具创建注释指示需要手动审查：

```markdown
<!-- TODO: Manual review needed - GitBook feature not auto-converted -->
```

### 审查插件转换

检查插件是否正确转换：

```yaml
# Original plugins configuration
plugins:
  - mathjax
  - mermaid
  - highlight

# May require manual verification in book.yaml
markdown:
  math: true
  diagrams: true
  highlight: true
```

### 迁移后测试链接

迁移后，验证：

1. 所有内部链接都有效
2. 跨章节链接正确解析
3. 导航结构正确
4. 资源路径正确

使用实时预览服务器：

```bash
mdpress serve
```

浏览整个文档并检查断开链接。

## 迁移报告

迁移工具生成转换内容的报告。

### 报告内容

迁移报告包括：

- 转换的文件
- 应用于每个文件的转换
- 潜在问题的警告
- 无法自动转换的功能
- 摘要统计

### 查看报告

迁移后，检查生成的 `MIGRATION_REPORT.md`：

```bash
cat MIGRATION_REPORT.md
```

报告显示：

```markdown
# Migration Report

## Summary
- Files processed: 42
- Lines converted: 1,200+
- Template blocks converted: 15
- Warnings: 3

## Converted Files
- README.md: 1 hint block converted
- getting-started.md: 2 code blocks converted
- ...

## Warnings
- Line 45 in api.md: Complex CSS not converted
- Line 120 in plugins.md: GitBook plugin syntax detected
- ...
```

### 模拟运行报告

模拟运行显示将转换的内容：

```bash
mdpress migrate ./gitbook-project --dry-run --verbose
```

输出指示：
- 将处理的文件数
- 将应用的转换类型
- 估计的成功率
- 任何可能的数据丢失风险

## 常见迁移问题

### 找不到模板语法

如果你的 GitBook 使用无法识别的自定义模板：

1. 检查迁移报告
2. 使用本指南中显示的模式手动转换
3. 在实时预览中测试

### 未转换复杂的 CSS

GitBook 中的自定义 CSS 将不会自动转换。选项：

1. 使用 mdPress CSS 定制
2. 简化 CSS 以使用标准 Markdown
3. 使用 mdPress 主题配置重新创建样式

### 插件功能丢失

添加功能的 GitBook 插件可能无法转换：

1. 检查 mdPress 是否具有等效功能
2. 改为使用 Markdown 扩展（KaTeX、Mermaid 等）
3. 如果重要，手动重新创建功能

### 指向外部 GitBook 的链接

如果你的文档链接到其他 GitBook 项目：

```markdown
[See other docs](https://docs.example.gitbook.io)
```

这些链接保持不变但现在指向外部资源。考虑：

1. 将文档合并到一个项目中
2. 改为在 mdPress 中使用交叉引用
3. 如果合适，保持外部链接

## 迁移后检查清单

运行迁移后，验证：

- [ ] 所有文件成功迁移
- [ ] `book.yaml` 配置完整
- [ ] SUMMARY.md 结构正确
- [ ] 运行 `mdpress serve` 并预览文档
- [ ] 点击主要章节和部分
- [ ] 验证所有图像正确显示
- [ ] 测试内部链接
- [ ] 检查代码块语法高亮
- [ ] 验证数学方程呈现（如使用）
- [ ] 测试搜索功能
- [ ] 构建到最终输出格式并验证

## 转换特定功能

### 选项卡为标题

将 GitBook 选项卡转换为 mdPress：

改为为每个选项使用 H3 标题：

```markdown
## Installation

### Using npm
```bash
npm install my-package
```
```

### Using yarn
```bash
yarn add my-package
```
```

### 标注块

将 GitBook 标注替换为引用块：

GitBook：
```
{% hint style="danger" %}
Don't do this!
{% endhint %}
```

mdPress：
```markdown
> **Danger:** Don't do this!
```

### 自定义属性

GitBook 属性如 `{#custom-id}` 已在 mdPress 中支持：

```markdown
# Chapter Title {#custom-id}
```

无需转换。

## 何时迁移 vs 重写

如果满足以下条件，考虑迁移：
- 你有广泛的文档
- 基本结构和内容很好
- 你想快速切换平台

如果满足以下条件，考虑重写：
- 文档需要重组
- 你正在显著更新内容
- 文档较小（50 页或更少）
- 你想从头开始利用 mdPress 特定功能

## 获取帮助

如果迁移没有产生预期的结果：

1. 查看 MIGRATION_REPORT.md
2. 检查警告中提到的特定部分
3. 在实时预览中测试各个转换
4. 参考编写内容指南了解 mdPress 语法
5. 手动调整特定的有问题的部分

迁移通常是成功的，但复杂的 GitBook 项目可能需要进行小的调整。
