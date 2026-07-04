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

## 协议

当可执行文件支持相应标志时，mdPress 会用 `--mdpress-info` 和 `--mdpress-hooks` 探测它。在钩子执行期间，通过 stdin 发送 JSON，从 stdout 读取 JSON。写入 stderr 的任何内容都会被记录到日志中。

如果缺少这些辅助标志，mdPress 会回退到版本 `0.1.0`，并将该插件订阅到所有阶段。

## 钩子阶段

| 阶段 | 何时运行 |
| --- | --- |
| `before_build` | 配置加载之后、章节处理之前。 |
| `after_parse` | 某个章节被渲染为 HTML 之后。 |
| `before_render` | 最终 HTML 组装之前。 |
| `after_render` | HTML 文档组装完成之后。 |
| `after_build` | 所有输出文件写入之后。 |
| `before_serve` | 实时预览服务器启动之前。 |
| `after_serve` | 实时预览服务器关闭时。 |

## 钩子数据

每个钩子会收到一个 `HookContext`，其中包含：

- 当前配置
- 活动阶段
- 当前内容负载
- 章节索引和源文件
- 相关时的输出路径和格式
- 一个共享的 `Metadata` 映射，用于在各阶段之间传递状态

如果插件返回非空的 `content`，mdPress 会替换当前的内容负载。如果它返回 `stop: true`，同一阶段中后续的插件会被跳过。
