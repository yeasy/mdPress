# 为 mdPress 编写内容

mdPress 使用标准 Markdown 进行内容编写，支持 GitHub Flavored Markdown (GFM) 和额外的扩展功能。本指南涵盖了编写文档时可用的语法和功能。

## 基本 Markdown 语法

从每个 mdPress 文档都支持的基础知识开始。

### 标题

使用哈希符号创建 H1 至 H6 的标题：

```markdown
# 标题 1
## 标题 2
### 标题 3
#### 标题 4
##### 标题 5
###### 标题 6
```

每个标题自动生成一个 ID 用于交叉引用。ID 由标题文本转换为小写并用连字符替换空格生成。

### 段落和换行

用空白行分隔段落。段落内的单个换行符不会创建新段落。

```markdown
这是第一段。

这是第二段。
```

要创建不开始新段落的换行符，在行尾添加两个空格或反斜杠：

```markdown
第一行
第二行

或使用反斜杠：
第一行\
第二行
```

### 强调和加粗文本

```markdown
*斜体* 或 _斜体_
**加粗** 或 __加粗__
***加粗斜体*** 或 ___加粗斜体___
~~删除线~~
```

### 列表

#### 无序列表

```markdown
- 项目 1
- 项目 2
  - 嵌套项目 2a
  - 嵌套项目 2b
- 项目 3
```

你可以互换使用 `-`、`*` 或 `+`。

#### 有序列表

```markdown
1. 第一项
2. 第二项
   1. 嵌套项目 2a
   2. 嵌套项目 2b
3. 第三项
```

### 引用块

```markdown
> 这是一个引用块。
> 它可以跨越多行。
>
> 并且可以包含多个段落。
```

支持嵌套引用块：

```markdown
> 第 1 级引用块
>
> > 第 2 级引用块
```

### 链接

使用以下语法创建链接：

```markdown
[链接文本](https://example.com)
[带标题的链接](https://example.com "标题")
```

引用样式链接：

```markdown
[链接文本][ref]

[ref]: https://example.com
```

自动链接（在 mdPress 中自动转换）：

```markdown
https://example.com
user@example.com
```

### 代码

内联代码使用反引号：

```markdown
使用 `const` 关键字声明常量。
```

代码块使用三个反引号，可选语言指定：

````markdown
```javascript
function hello() {
  console.log("Hello, world!");
}
```
````

## GitHub Flavored Markdown (GFM)

mdPress 包含对现代文档需求的完整 GFM 支持。

### 表格

使用管道和连字符创建表格：

```markdown
| 标题 1 | 标题 2 | 标题 3 |
|--------|--------|--------|
| 单元格 1 | 单元格 2 | 单元格 3 |
| 单元格 4 | 单元格 5 | 单元格 6 |
```

对齐使用冒号指定：

```markdown
| 左对齐 | 居中 | 右对齐 |
|:------|:----:|-------:|
| L1    |  C1  |     R1 |
| L2    |  C2  |     R2 |
```

### 任务列表

任务列表对于文档中的检查列表功能很有用：

```markdown
- [x] 已完成任务
- [ ] 未完成任务
- [x] 另一个已完成任务
```

### 删除线

在强调部分已提及，但对 GFM 至关重要：

```markdown
~~这段文本被删除了~~
```

## 代码块与语法高亮

mdPress 使用行业标准主题支持 100+ 种编程语言的语法高亮。

### 支持的语言

常见语言包括：bash、c、cpp、csharp、css、dart、elixir、elm、go、groovy、haskell、java、javascript、kotlin、lisp、lua、objective-c、perl、php、python、ruby、rust、scala、shell、sql、swift、typescript、xml、yaml 等。

### 语法高亮示例

````markdown
```python
def fibonacci(n):
    """计算第 n 个斐波那契数。"""
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)

result = fibonacci(10)
print(f"Result: {result}")
```
````

### 行高亮

在代码块中突出显示特定行：

````markdown
```javascript {3,5-7}
function process(data) {
  const trimmed = data.trim();
  const processed = transform(trimmed);  // 第 3 行

  validate(processed);  // 第 5 行
  store(processed);     // 第 6 行
  return processed;     // 第 7 行
}
```
````

### 主题

mdPress 包含多个语法高亮主题。浅色和深色变体会根据用户偏好自动应用。示例包括：GitHub Light、GitHub Dark、Dracula、Nord、Solarized Light/Dark 等。

## 图像

在文档中包含图像并添加替代文本以提高可访问性：

```markdown
![图像的替代文本](./images/screenshot.png)
```

### 延迟加载

图像在 mdPress 中自动延迟加载。渲染器会延迟图像加载直到图像即将进入视口，从而提高页面加载性能。

### 图像 URL

相对和绝对 URL 都有效：

```markdown
![本地图像](./assets/diagram.svg)
![外部图像](https://example.com/image.png)
```

## 用于注释和警告的引用块

使用引用块突出显示重要信息：

```markdown
> **注意：** 这是用户的重要注释。

> **警告：** 小心此配置。

> **提示：** 这是一个有用的建议。
```

## 章节间的内部链接

链接到文档中的其他章节：

```markdown
[查看配置指南](./configuration.md)
[跳转到高级用法](./advanced-usage.md#custom-plugins)
```

构建时，mdPress 会根据你的书籍结构自动解析这些链接。相对路径在文档层次结构中有效。

### 片段链接

使用标题 ID 链接到特定部分：

```markdown
[查看安装部分](#installation)
[跳转到高级配置](./configuration.md#advanced-configuration)
```

## 标题 ID 用于交叉引用

每个标题自动获得唯一 ID。ID 通过以下方式生成：

1. 将标题文本转换为小写
2. 用连字符替换空格
3. 删除特殊字符

### 标题 ID 示例

```markdown
# 开始使用
```
变为 `#getting-started`

```markdown
### 高级配置 (v2.0)
```
变为 `#advanced-configuration-v20`

你可以从任何地方引用这些 ID：

```markdown
[查看开始使用部分](#getting-started)
```

自定义 ID 可以在某些 mdPress 配置中指定，但自动生成的 ID 适用于所有标准用例。

## 最佳实践

### 组织内容结构

- 为章节标题使用 H1（每个文件仅一个）
- 为主要部分使用 H2
- 为小节使用 H3 及以下
- 保持标题层次逻辑（不要从 H2 跳到 H4）

### 编写清晰的代码示例

- 始终在代码块中包含语言标识符
- 使用真实、完整的示例
- 添加注释来解释不明显的代码
- 在有帮助时同时展示代码和预期输出

### 适当使用强调

- 为 UI 元素和关键术语使用 **加粗**
- 为强调使用 *斜体*
- 为技术术语、文件名和命令使用 `代码`
- 避免组合多个强调样式

### 创建有用的链接

- 使用描述链接指向位置的链接文本
- 链接到文档内的相关部分
- 包括相邻章节和相关内容的交叉引用
- 使用片段链接指向特定小节

### 使表格可读

- 保持表格简洁且内容集中
- 使用对齐来改进可读性
- 考虑将大表格分成多个较小的表格
- 在表格前提供解释文本

## 故障排除

### 链接不工作

验证相对路径使用 `./` 前缀和正确的文件名。mdPress 相对于当前文件位置解析路径。

### 未应用代码高亮

确保在开始反引号后指定语言标识符。不指定则代码块显示为纯文本。

### 图像未加载

检查图像路径是否正确且相对于你的内容文件。图像路径使用与文档链接相同的解析规则。
