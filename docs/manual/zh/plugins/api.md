# 插件 API 参考

mdPress 插件就是**任意一个可执行文件**。没有 SDK，没有动态库，也没有插件清单
文件。mdPress 与插件之间的全部交流，就是运行它并通过 stdin/stdout 交换一个
JSON 文档。

本页是该协议的规范说明，与 `internal/plugin/external.go` 保持一致；参考实现是
[`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count)。

> **插件就是任意代码。** 在 `book.yaml` 里声明一个插件，意味着 mdPress 会在你的
> 机器上执行那个文件。构建别人的项目之前请先读[信任模型](#信任模型)。

## 声明插件

```yaml
plugins:
  - name: reading-time
    path: ./plugins/reading-time
    config:
      words_per_minute: 200
```

| 键 | 必填 | 含义 |
| --- | --- | --- |
| `name` | 是 | 出现在日志和 `mdpress doctor` 输出中的标识符。 |
| `path` | 是 | 可执行文件路径，**相对于 `book.yaml`**。 |
| `config` | 否 | 任意 YAML 映射，每次调用都原样传给插件。 |

这三个键就是全部 schema。`command:`、`hooks:`、`enabled:` 都不被识别——写
`command:` 会让该条目以 *"is missing the required 'path' field"* 失败，其他多余
的键会产生一条 *"unknown key in config file"* 警告并被忽略。插件自己选择要订阅
的阶段（见[能力探测](#能力探测)），`book.yaml` 无法对其过滤。

加载插件时强制执行的路径规则：

- 绝对路径直接拒绝：*"absolute paths are not allowed; use a path relative to the
  project directory"*。
- 解析后落在项目目录之外的相对路径（`../mdpress`，或指向树外的符号链接）同样
  被拒绝。
- 在非 Windows 系统上，文件必须带有可执行位。
- 在 Windows 上，不带扩展名的路径还会尝试 `PATHEXT` 中的后缀。

只要**任意一项**没通过这些检查，mdPress 就会打印一条警告，并在**完全不加载任何
插件**的情况下继续构建——插件加载是全有或全无的：

```
WARN plugin loading failed (continuing without plugins): failed to load
     plugin "missing": plugin executable not found at ".../does-not-exist"
```

插件按声明顺序运行。

## 能力探测

加载插件时，mdPress 会分别带一个标志运行它两次。两次调用的超时都是 **5 秒**，
stdout 上限为 1 MB。

### `--mdpress-info`

```console
$ ./plugins/reading-time --mdpress-info
{"version":"1.0.0","description":"Prepends an estimated reading time to each chapter."}
```

只有 `version` 和 `description` 会被读取，且都只是展示用，出现在
`mdpress doctor --verbose` 里：

```
✓ Plugin "reading-time" responds (version 1.0.0, 1 hook(s))
```

如果调用失败、超时或没有输出 JSON，mdPress 会回退到版本 `0.1.0` 和空描述。

### `--mdpress-hooks`

```console
$ ./plugins/reading-time --mdpress-hooks
["after_parse"]
```

一个由阶段名组成的 JSON 数组。插件只会在它列出的阶段被调用。如果调用失败或没有
输出 JSON 数组，mdPress 会把该插件订阅到**全部七个**阶段，于是它被执行的次数会
远超预期。

两个标志都不回答不会导致失败，但 `mdpress doctor` 会点名：

```
⚠ Plugin "doc-protocol" does not speak the mdpress plugin protocol
  (no valid --mdpress-info or --mdpress-hooks response)
```

## 钩子调用

每次触发钩子，mdPress 都会**新起一个进程**（不带任何参数），向它的 stdin 写入一
个 JSON 对象并关闭 stdin，然后从它的 stdout 读取一个 JSON 对象。插件处理完后应
当自行退出。

因为每次调用都是新进程，插件**无法在钩子之间用内存保存状态**。需要跨章节累积任
何东西，就写到文件里。

单次调用的超时是 **30 秒**，stdout 和 stderr 各自的上限是 10 MB。插件继承
mdPress 启动时所在的工作目录。

### 请求

```json
{
  "phase": "after_parse",
  "content": "<h1 id=\"introduction\">Introduction</h1>\n<p>Hello…</p>",
  "chapter_index": 0,
  "chapter_file": "chapters/01-intro.md",
  "output_path": "",
  "output_format": "",
  "config": { "words_per_minute": 200 },
  "metadata": {}
}
```

八个键始终存在。它们实际携带的内容：

| 字段 | 类型 | 取值 |
| --- | --- | --- |
| `phase` | string | 阶段名，例如 `after_parse`。 |
| `content` | string | 该阶段的负载——见[生命周期钩子](./lifecycle-hooks.md)。`before_build` 和 `after_build` 中为空。 |
| `chapter_index` | number | 从 0 开始的章节序号。只在 `after_parse` 有意义，其他阶段一律是 `0`。 |
| `chapter_file` | string | 章节源文件路径。只在 `after_parse` 设置。 |
| `output_path` | string | **目前始终为空。** 没有任何调用点填充它。 |
| `output_format` | string | **目前始终为空。** 没有任何调用点填充它。 |
| `config` | object | `book.yaml` 里的 `config:` 映射，或 `{}`。 |
| `metadata` | object | **目前始终为空。** 响应里没有任何写回它的途径。 |

请求里没有书名、作者、主题、章节列表和输出目录。需要项目元数据的插件得自己去读
`book.yaml`。

### 响应

```json
{
  "content": "<p class=\"reading-time\">3 min read</p>\n<h1>…</h1>",
  "stop": false,
  "error": ""
}
```

| 字段 | 类型 | 效果 |
| --- | --- | --- |
| `content` | string | 替换负载——**但仅在 `after_parse` 生效**。空字符串或省略该键表示"保持内容不变"。 |
| `stop` | bool | 跳过该阶段中后续的插件。 |
| `error` | string | 非空表示插件失败，见[失败处理](#失败处理)。 |

这三个字段就是响应的全部 schema。`status`、`action`、`modified_content`、
`content_type`、`metadata`、`data`、`warnings`、`errors` **都不属于本协议**：只
由这些键组成的响应能被正常解析，被读作"没有变化"，于是插件静默地什么也没做。

什么都不输出也是合法的，同样表示"没有变化"。

## 失败处理

插件失败永远只是一条**警告**。它不会让构建失败，也不会改变退出码。

| 情况 | mdPress 的处理 |
| --- | --- |
| 非零退出 | `WARN … plugin exited with error: exit status 1`——stderr 会附在消息后面。 |
| 响应里设置了 `"error"` | `WARN … plugin "name" reported error: <文本>` |
| 响应不是合法 JSON | `WARN … failed to parse plugin response`，附前 200 个字符。 |
| 超时（30 秒） | 进程被杀掉，并按上面的方式报告失败。 |

失败调用返回的内容会被丢弃；构建继续使用原内容，并继续执行其余插件。

**成功调用的 stderr 会被丢弃。** 它只在进程非零退出时才被展示，所以 stderr 是崩
溃报告通道，不是日志通道。想要可见的运行日志，插件应该自己写文件。

## 信任模型

插件是 mdPress 在 `build` 和 `serve` 期间执行的可执行文件——包括加载时用于
`--mdpress-info` / `--mdpress-hooks` 探测的那两次执行。因此打开一个项目就等于运
行了这个项目自带的代码。

- **本地来源始终会运行它们的插件。** 没有开关可以关掉它；不信任某个目录，就不要
  构建它。
- **远程来源默认拒绝运行插件**，除非你显式选择启用：

  ```bash
  mdpress build https://github.com/owner/repo --allow-plugins
  ```

  不加这个标志会得到 `Refusing to run N plugin(s) from a remote project; pass
  --allow-plugins to trust and execute them.`，构建会在不加载插件的情况下继续。
  `--allow-plugins` 在 `build` 和 `serve` 上都有，且**对本地项目没有任何影响**
  ——本地项目本来就被视为可信。

[声明插件](#声明插件)里的收敛规则（禁止绝对路径、不得跳出项目目录）限制的是
*哪个文件*会被运行，而不是*它能做什么*。插件以你的完整用户权限运行。

## 参考实现

[`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count)
是一个完整的 Go 插件，实现了两个探测标志和 `after_parse` 钩子：

```bash
go build -o plugins/word-count ./examples/plugins/word-count
```

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
```

分步教程见[编写插件](./building-a-plugin.md)，各阶段实际能做什么见[生命周期钩子](./lifecycle-hooks.md)。
