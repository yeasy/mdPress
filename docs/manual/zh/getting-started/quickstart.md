# 快速开始

用以下命令创建一个示例项目：

```bash
mdpress quickstart my-book
cd my-book
mdpress build
mdpress serve --open
```

`quickstart` 会创建 `book.yaml`、`README.md`、`preface.md`、`chapter01/README.md`、`chapter02/README.md`、`chapter03/README.md`、`images/README.md` 以及 `images/cover.svg`。

生成的配置以项目名作为 PDF 文件名的基础名，可以立即编辑。

`mdpress quickstart` 默认在 `my-book` 目录创建项目。它会拒绝写入非空目录。

## 后续步骤

- 编辑 `book.yaml` 设置标题、作者、主题和输出选项。
- 用你自己的内容替换示例 Markdown 文件。
- 构建前如果想快速检查一下，可以运行 `mdpress validate`。
