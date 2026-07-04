# 实时预览

`mdpress serve` 会构建站点预览、启动本地 HTTP 服务器，并在源文件变化时重新加载浏览器。它接受本地目录或 GitHub URL；不传入源时使用当前目录。

## 启动

```bash
mdpress serve
```

默认情况下，它监听 `127.0.0.1:9000`，将预览输出写入 `_book/`，并且不会自动打开浏览器。

## 常用标志

- `--open` 在启动后打开浏览器。
- `--host 0.0.0.0` 将服务器暴露到你的网络中。
- `--port 3000` 指定端口。若不设置，mdPress 从 `9000` 开始，使用第一个可用端口。
- `--output ./preview` 将生成的站点写入其他目录。
- `--summary SUMMARY.md` 强制从指定的 summary 文件确定章节顺序。

示例：

```bash
mdpress serve --host 0.0.0.0 --port 3000 --open
```

预览 GitHub 仓库时，可以用 `--branch` 选择分支：

```bash
mdpress serve https://github.com/yeasy/agentic_ai_guide --branch main
```

## 什么会触发重建

- Markdown 源文件
- `book.yaml`
- `SUMMARY.md`
- 引用的资源，例如图像和样式

## 说明

- `serve` 是浏览器预览路径。如果需要最终的分页检查，请使用 `mdpress build --format pdf` 或 `mdpress build --format html`。
- 如果在没有 `book.yaml` 或 `SUMMARY.md` 的仓库根目录运行，mdPress 会回退到自动发现，可能会收录比你预期更多的 Markdown 文件。
