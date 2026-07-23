# 构建插件：分步教程

我们要写的插件叫 `reading-time`：给每一章前面加上一行"3 min read"。它足够小，可
以一口气读完，同时覆盖了协议的全部要点——两个探测标志、逐章内容改写，以及来自
`book.yaml` 的插件配置。

这里用 Python 只是因为它短。**任何可执行文件都行**：shell 脚本、Go 二进制、编译
好的 Rust 程序。mdPress 只关心它能跑起来，并且在 stdin/stdout 上说 JSON。

## 第 1 步：准备一个项目

```bash
mkdir -p demo/chapters demo/plugins
cd demo
```

`book.yaml`：

```yaml
book:
  title: "Reading Time Demo"
  author: "You"

chapters:
  - title: "Introduction"
    file: chapters/01-intro.md
```

`chapters/01-intro.md`：

```markdown
# Introduction

Some words to count in this chapter.
```

先确认不带插件的构建是通的：

```bash
mdpress build --format html
```

> 章节条目是带 `title:` 和 `file:` 键的映射。裸字符串（`- chapters/01-intro.md`）
> 不是简写，而是一个 YAML 类型错误。

## 第 2 步：两个探测标志

插件在被要求干活之前，必须先回答两个标志。先只写这部分，创建
`plugins/reading-time`：

```python
#!/usr/bin/env python3
"""reading-time: prepends an estimated reading time to every chapter."""
import json
import sys


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] == "--mdpress-info":
            json.dump({"version": "1.0.0",
                       "description": "Prepends an estimated reading time to each chapter."},
                      sys.stdout)
            return
        if sys.argv[1] == "--mdpress-hooks":
            json.dump(["after_parse"], sys.stdout)
            return


main()
```

加上可执行位——没有可执行位的插件会被 mdPress 拒绝：

```bash
chmod +x plugins/reading-time
```

手工验证这两个标志：

```console
$ ./plugins/reading-time --mdpress-info
{"version": "1.0.0", "description": "Prepends an estimated reading time to each chapter."}
$ ./plugins/reading-time --mdpress-hooks
["after_parse"]
```

`--mdpress-hooks` 就是插件订阅阶段的方式。回答 `["after_parse"]` 意味着 mdPress
每章运行本程序一次，其他阶段一律不调用。**不**回答这个标志的插件会被订阅到全部七
个阶段，这几乎从来不是你想要的。

## 第 3 步：处理钩子

钩子触发时，mdPress 会以**不带任何参数**的方式运行同一个程序，向 stdin 写入一个
JSON 请求，再从 stdout 读取一个 JSON 响应。补上主体：

```python
#!/usr/bin/env python3
"""reading-time: prepends an estimated reading time to every chapter."""
import json
import re
import sys

TAGS = re.compile(r"<[^>]+>")


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] == "--mdpress-info":
            json.dump({"version": "1.0.0",
                       "description": "Prepends an estimated reading time to each chapter."},
                      sys.stdout)
            return
        if sys.argv[1] == "--mdpress-hooks":
            json.dump(["after_parse"], sys.stdout)
            return

    req = json.load(sys.stdin)

    wpm = req.get("config", {}).get("words_per_minute", 200)
    words = len(TAGS.sub(" ", req.get("content", "")).split())
    minutes = max(1, round(words / wpm))

    banner = f'<p class="reading-time">{minutes} min read</p>\n'
    json.dump({"content": banner + req.get("content", "")}, sys.stdout)


main()
```

有三点值得注意：

- **`content` 是 HTML。** 到 `after_parse` 时 Markdown 已经转换完了，所以才要那个
  去标签的正则，横幅也是 `<p>` 元素而不是 Markdown 段落。
- **`config` 直接来自 `book.yaml`。** 数字就是数字；永远要给默认值，因为键可能压
  根不存在。
- **响应会替换该章。** 返回 `{"content": ""}` 或 `{}` 表示"保持不变"——只读型插件
  就是这样报告成功的。

## 第 4 步：脱离 mdPress 测试插件

协议就是 stdin 上的普通 JSON，所以插件在 shell 里就能测：

```console
$ echo '{"phase":"after_parse","content":"<p>hello world</p>","chapter_index":0,
  "chapter_file":"chapters/01-intro.md","config":{"words_per_minute":200},"metadata":{}}' \
  | ./plugins/reading-time
{"content": "<p class=\"reading-time\">1 min read</p>\n<p>hello world</p>"}
```

把插件接进构建之前先这么做一遍。在 mdPress 里插件失败永远只是一条警告，坏掉的插
件在构建输出里很容易被忽略。

## 第 5 步：在 `book.yaml` 中注册

```yaml
book:
  title: "Reading Time Demo"
  author: "You"

chapters:
  - title: "Introduction"
    file: chapters/01-intro.md

plugins:
  - name: reading-time
    path: ./plugins/reading-time
    config:
      words_per_minute: 200
```

`path` 必须相对于 `book.yaml`，而且必须落在项目目录内。绝对路径会被拒绝。

## 第 6 步：用 `doctor` 验证

```console
$ mdpress doctor --verbose
  ✓ Plugin "reading-time" responds (version 1.0.0, 1 hook(s))
  ✓ All 1 plugin(s) are valid
```

`doctor` 会真的去跑那两个探测标志，所以这一行能证明握手是通的。不加 `--verbose`
就只有汇总行。两个标志都不回答的插件会被报告为：

```
⚠ Plugin "reading-time" does not speak the mdpress plugin protocol
  (no valid --mdpress-info or --mdpress-hooks response)
```

## 第 7 步：构建

```console
$ mdpress build --format html
  ✅ Build completed (elapsed 123ms)
  ✓ Generated html  → .../Reading-Time-Demo.html

$ grep -o '<p class="reading-time">[^<]*</p>' Reading-Time-Demo.html
<p class="reading-time">1 min read</p>
```

`mdpress serve` 会在首次渲染和每次文件变更后跑同一个钩子，所以你编辑时横幅始终是
最新的。

## 调试

### 构建过程完全没提我的插件

这是正常情况——成功的插件是安静的。想确认它到底跑没跑，让它写个文件：

```python
with open("/tmp/reading-time.log", "a") as fh:
    fh.write(req.get("chapter_file", "?") + "\n")
```

**别用 stderr 干这件事。** mdPress 只在插件非零退出时才展示 stderr；运行成功时它
会被丢弃。

### 插件跑了，但什么都没变

按顺序检查三件事：

1. **阶段。** `after_parse` 是唯一会采用返回 `content` 的阶段。从
   `before_build`、`before_render`、`after_render`、`after_build` 返回的内容都会
   被丢弃。见[生命周期钩子](./lifecycle-hooks.md)。
2. **字段名。** 响应的键叫 `content`。`modified_content`、`output`、`body` 之类都
   会被忽略，而被忽略的响应会被读作"没有变化"。
3. **多余的 stdout。** 响应必须是 stdout 上*唯一*的东西。一句多余的 `print()` 就
   会让响应无法解析，报成 `failed to parse plugin response`。

### 只有一部分插件运行了

插件加载是全有或全无的。一个坏条目会让这次构建的所有插件都不加载：

```
WARN plugin loading failed (continuing without plugins): failed to load
     plugin "missing": plugin executable not found at ".../does-not-exist"
```

也检查一下前面的插件是不是返回了 `"stop": true`——它会跳过在它之后声明的、订阅了
同一阶段的所有插件。

### 我的插件需要知道输出路径

拿不到。`output_path` 和 `output_format` 虽然在请求里，但始终为空。需要什么就通过
自己的 `config:` 块传进来。

## 分发插件

没有插件仓库，也没有安装命令。分发方式就是"把可执行文件和 `book.yaml` 片段一起
发出去"：

```
reading-time/
├── README.md          # 它做什么，外加可以直接复制的 book.yaml 块
├── LICENSE
└── reading-time       # 可执行文件
```

如果插件是编译型的，请连源码和构建命令一起提供——不同操作系统或架构的用户需要自
己编译：

```bash
go build -o plugins/word-count ./examples/plugins/word-count
```

请在 README 里直说：安装这个插件意味着允许 mdPress 在每一次 build 和 serve 时执行
它。

## 后续阅读

- [插件 API 参考](./api.md) —— 精确的请求/响应 schema 与信任模型。
- [生命周期钩子](./lifecycle-hooks.md) —— 每个阶段能做什么、不能做什么。
- [`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count)
  —— 同一套协议的 Go 版本。
