# 生命周期钩子详解

协议里有七个阶段名。**其中五个会在 `mdpress build` 期间触发，一个会在
`mdpress serve` 期间触发，还有两个虽有定义但从不派发。** 而真正能改变输出的，只
有一个阶段。

本页写清楚每个阶段到底给你什么，免得靠试错去发现答案。线上格式见
[插件 API 参考](./api.md)。

## 哪些会触发，各自能做什么

| 阶段 | `build` 中触发 | `serve` 中触发 | `content` 负载 | 返回的 `content` 是否生效 |
| --- | --- | --- | --- | --- |
| `before_build` | 一次 | 否 | 空 | 否 |
| `after_parse` | 每章一次 | 每次重建、每章一次 | 该章渲染后的 HTML | **是** |
| `before_render` | 一次 | 否 | 封面 HTML | 否 |
| `after_render` | 一次 | 否 | 目录 HTML | 否 |
| `after_build` | 一次 | 否 | 空 | 否 |
| `before_serve` | 从不 | 从不 | — | — |
| `after_serve` | 从不 | 从不 | — | — |

动手之前有两条结论值得先记住：

1. **`after_parse` 是唯一能修改书籍内容的阶段。** 在其他任何阶段，你在 `content`
   里返回的值会被读取然后丢弃。想改变读者看到的东西，就在 `after_parse` 里改。
2. **`before_serve` 和 `after_serve` 从不运行。** 这两个名字在协议里有定义，
   `--mdpress-hooks` 也接受它们，但没有任何代码路径会派发。只订阅这两个阶段的插
   件会被正常加载，在 `mdpress doctor` 里显示为有效，然后永远不会被执行。

其余阶段作为*副作用*钩子仍然有用：进程确实会跑起来，所以它可以写文件、调用 API
或者大声报错。它只是没法把内容交回来。

## 构建时序

```
mdpress build
  │
  ├─ 加载 book.yaml、主题、插件      ← 插件在这里被探测（也就是被执行！）
  │
  ├─ [before_build]                 content: ""
  │
  ├─ 把每一章解析成 HTML
  │    └─ [after_parse]  每章一次     content: 章节 HTML  → 可替换
  │
  ├─ 生成封面 + 目录
  ├─ [before_render]                content: 封面 HTML
  ├─ 组装单页 HTML
  ├─ [after_render]                 content: 目录 HTML
  │
  ├─ 写出所有请求的格式（html、pdf、epub、site、typst…）
  │
  └─ [after_build]                  content: ""
```

注意 `before_render` / `after_render` / `after_build` 是**每次构建触发一次**，而
不是每种输出格式一次——`--format html,pdf,epub` 也只会各派发一次，而且从请求里
根本看不出这次请求了哪些格式。

## 预览时序

```
mdpress serve
  │
  ├─ 加载 book.yaml、主题、插件      ← 插件在这里被探测（也就是被执行！）
  ├─ 首次渲染
  │    └─ [after_parse]  每章一次
  │
  └─ 每次文件变更
       └─ [after_parse]  每章一次
```

`mdpress serve` 只会重新跑章节流水线，所以 `after_parse` 是它唯一派发的钩子。在
`after_build` 里干活的插件会表现为"开发模式下不生效"——原因就在这里。

## `before_build`

一次性触发，发生在 `book.yaml` 与主题加载完成之后、任何章节被解析之前。

请求：`phase` 是 `before_build`；`content`、`chapter_file`、`output_path` 和
`output_format` 都为空；`config` 是你的设置。

适合：前置检查（API key 在不在？转换器装了没？）、清理插件自己的临时目录、记录构
建开始时间。

不适合：任何需要改变构建行为的事。没有配置写回通道——mdPress 早已加载完配置，而
你返回的 `content` 会被忽略。

```bash
#!/bin/sh
# preflight —— 没有 API token 就拒绝构建。
case "$1" in
  --mdpress-info)  echo '{"version":"1.0.0","description":"Checks build prerequisites."}'; exit 0 ;;
  --mdpress-hooks) echo '["before_build"]'; exit 0 ;;
esac

cat > /dev/null   # 把请求读空；这里用不上

if [ -z "$DOCS_API_TOKEN" ]; then
  echo '{"error":"DOCS_API_TOKEN is not set"}'
else
  echo '{}'
fi
```

上报 `error` 只会产生一条警告，**不会**中止构建——见
[失败处理](./api.md#失败处理)。

## `after_parse`

每章触发一次，就在该章的 Markdown 转成 HTML 之后、标题/图/表被登记进交叉引用之
前。所以你注入的内容会像手写内容一样参与编号。

请求：`content` 是该章的 HTML，`chapter_file` 是它的源文件路径，
`chapter_index` 是它在扁平化章节列表中从 0 开始的序号。

内容类工作都在这个阶段做：注入横幅、改写元素、统计字数、校验结构。

```python
#!/usr/bin/env python3
"""reading-time —— 为每一章加上预计阅读时间。"""
import json
import re
import sys

TAGS = re.compile(r"<[^>]+>")

if len(sys.argv) > 1:
    if sys.argv[1] == "--mdpress-info":
        json.dump({"version": "1.0.0",
                   "description": "Prepends an estimated reading time."}, sys.stdout)
        sys.exit(0)
    if sys.argv[1] == "--mdpress-hooks":
        json.dump(["after_parse"], sys.stdout)
        sys.exit(0)

req = json.load(sys.stdin)

wpm = req.get("config", {}).get("words_per_minute", 200)
words = len(TAGS.sub(" ", req.get("content", "")).split())
minutes = max(1, round(words / wpm))

banner = f'<p class="reading-time">{minutes} min read</p>\n'
json.dump({"content": banner + req.get("content", "")}, sys.stdout)
```

记住 `content` 是 **HTML，不是 Markdown**——转换早就完成了。去搜 `## ` 或者
`[text](link)` 的插件什么也搜不到。

返回 `"content": ""`（或干脆省略该键）表示保持该章不变，只读型插件就该这么写。

### 中断插件链

设置 `"stop": true` 可以跳过在你之后声明的、订阅了同一阶段的所有插件：

```json
{"content": "<p>…</p>", "stop": true}
```

该章会保留你返回的内容，后续插件对这一章不再运行。`stop` 不影响其他阶段，也不影
响其他章节。

## `before_render`

一次性触发，发生在所有章节解析完成之后、单页 HTML 文档组装之前。

请求：`content` 是**封面 HTML**——既不是某一章，也不是整篇文档。
`chapter_file`、`output_path` 和 `output_format` 都为空。

封面只是作为一个代表性负载供你查看。返回改过的封面没有任何效果，组装时用的仍是
mdPress 自己生成的封面。

适合：必须在全部内容就绪之后、输出文件写出之前发生的副作用。

## `after_render`

一次性触发，发生在单页 HTML 文档组装完成之后、各输出格式写出之前。

请求：`content` 是**目录 HTML**。和 `before_render` 一样，它只供查看，返回值会被
丢弃。

适合：提取文档大纲、检查每一章是否都进了目录。

## `after_build`

一次性触发，发生在所有请求的格式都已写入磁盘之后。

请求：`content`、`chapter_file`、`output_path`、`output_format` 全部为空。
**这个阶段不会告诉你输出去了哪里。** 插件需要路径的话，要么自己读 `book.yaml` 的
`output.filename`，要么通过自己的 `config:` 块传进来：

```yaml
plugins:
  - name: publish
    path: ./plugins/publish
    config:
      artifact: dist/book.pdf
```

适合：上传产物、发送通知、在一个你本来就知道的路径旁生成报告。

```python
#!/usr/bin/env python3
"""notify —— 构建结束后把产物复制到某处。"""
import json
import shutil
import sys

if len(sys.argv) > 1:
    if sys.argv[1] == "--mdpress-info":
        json.dump({"version": "1.0.0",
                   "description": "Copies the built artifact to a drop directory."}, sys.stdout)
        sys.exit(0)
    if sys.argv[1] == "--mdpress-hooks":
        json.dump(["after_build"], sys.stdout)
        sys.exit(0)

req = json.load(sys.stdin)
cfg = req.get("config", {})

try:
    shutil.copy(cfg["artifact"], cfg["drop_dir"])
except Exception as exc:                       # 会显示为一条构建警告
    json.dump({"error": f"publish failed: {exc}"}, sys.stdout)
else:
    json.dump({}, sys.stdout)
```

## `before_serve` 与 `after_serve`

协议中有定义，`--mdpress-hooks` 也接受，但**从不派发**。今天的 `mdpress serve`
没有任何地方会调用它们。

这里专门列出来，是为了让你不必花一个下午去调试一个本来就不会运行的插件。确实需要
响应 serve 的话，用 `after_parse`——它在首次渲染和每次重建时都会触发；但要记得它
是每章触发一次，"每次重建只做一次"的工作得自己加个判断。

## 如何选择阶段

| 你想…… | 用 |
| --- | --- |
| 改变读者看到的内容 | `after_parse` |
| 校验内容并给出警告 | `after_parse` |
| 在干活之前检查前置条件 | `before_build` |
| 查看封面或目录 | `before_render` / `after_render` |
| 构建完成后发布、通知或归档 | `after_build` |
| 响应实时预览的重建 | `after_parse` |

只订阅你真正会用的阶段。`--mdpress-hooks` 调用失败的插件会被订阅到全部七个阶段，
意味着每章一次进程启动、外加每次构建多出四次，全是白跑。

请求与响应的精确 schema 见[插件 API 参考](./api.md)，完整演练见
[编写插件](./building-a-plugin.md)。
