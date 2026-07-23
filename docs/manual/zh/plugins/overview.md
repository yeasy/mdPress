# 插件概述

mdPress 插件是在 `book.yaml` 中声明的外部可执行文件，而不是进程内的 Go 插件。

> **安全警告：** `book.yaml` 中声明的插件是在你的机器上运行的可执行文件，会在构建（build）和预览（serve）过程中被执行——包括探测阶段（probe），mdPress 会用 `--mdpress-info` 和 `--mdpress-hooks` 调用每个可执行文件以查询其元数据。因此只对你信任的项目执行 build/serve。从 v0.7.12 起，远程来源（例如 GitHub URL）默认拒绝运行插件，除非显式传入 `--allow-plugins` 选项来选择启用。

## 配置插件

```yaml
plugins:
  - name: word-count
    path: ./examples/plugins/word-count
    config:
      warn_threshold: 500
```

- `name` 是插件标识符。
- `path` 指向可执行文件，相对于 `book.yaml`。
- `config` 以 JSON 形式透传给插件。

插件按声明顺序运行。

只要有任意一项加载失败，mdPress 会警告一次，然后在**完全不加载任何插件**的情况下继续构建。

## 协议

mdPress 会用 `--mdpress-info` 和 `--mdpress-hooks` 探测每个可执行文件。每次触发钩子时，都会重新启动一个进程：通过 stdin 发送一个 JSON 对象，从 stdout 读取一个 JSON 对象。

如果缺少这些辅助标志，mdPress 会回退到版本 `0.1.0`，并将该插件订阅到所有阶段。

stderr 只有在插件以非零状态退出时才会被展示出来。运行成功时 stderr 会被丢弃，因此它不是日志通道。

## 钩子阶段

| 阶段 | 何时运行 | 能否修改内容 |
| --- | --- | --- |
| `before_build` | 一次，配置加载之后、章节处理之前。 | 否 |
| `after_parse` | 每章一次，该章渲染为 HTML 之后。也是 `mdpress serve` 唯一会派发的钩子。 | **能** |
| `before_render` | 一次，最终 HTML 组装之前。负载是封面 HTML。 | 否 |
| `after_render` | 一次，HTML 文档组装完成之后。负载是目录 HTML。 | 否 |
| `after_build` | 一次，所有输出文件写入之后。 | 否 |
| `before_serve` | 协议中有定义，但**从不派发**。 | — |
| `after_serve` | 协议中有定义，但**从不派发**。 | — |

## 钩子数据

每个请求会带上阶段名、内容负载、章节索引与源文件（仅 `after_parse`），以及该插件自己的 `config` 块。`output_path`、`output_format` 和 `metadata` 虽然属于线上格式的一部分，但目前始终为空。

插件返回非空的 `content` 会替换内容负载——但**只在 `after_parse` 生效**，其余阶段一律丢弃。返回 `stop: true` 会跳过该阶段中后续的插件。返回非空的 `error` 只会产生一条构建警告，绝不会让构建失败。
