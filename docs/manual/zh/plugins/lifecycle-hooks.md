# 生命周期钩子详解

mdPress 提供六个生命周期钩子，允许插件拦截和修改构建过程。本指南说明了何时触发每个钩子、可用的数据以及每个钩子的实际用例。

## 钩子序列

构建过程遵循此顺序，钩子在关键点触发：

```
开始构建
  ↓
[before_build] - 插件可以初始化、验证设置
  ↓
解析内容
  ↓
[after_parse] - 插件可以修改解析的内容
  ↓
[before_render] - 插件可以为呈现做准备
  ↓
渲染到输出
  ↓
[after_render] - 插件可以后处理输出
  ↓
[after_build] - 构建完成，最终任务
  ↓
结束构建
```

对于 serve 模式（实时开发）：

```
[before_serve] - 服务器初始化
  ↓
监视并重新构建
  ├─ 文件更改触发重新构建
  ├─ 所有构建钩子执行
  └─ 重复
  ↓
[after_serve] - 服务器停止
  ↓
关闭
```

## before_build

在构建过程的最开始触发，在任何内容被处理之前。

### 用例

- 验证配置和依赖项
- 初始化外部资源（数据库、API）
- 设置日志记录或监控
- 清理之前的构建工件
- 准备构建范围的状态

### 可用数据

钩子上下文包括：

```json
{
  "context": {
    "phase": "before_build",
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0"
    },
    "config": {
      "theme": "technical",
      "output_format": "html"
    },
    "output_path": "build/",
    "chapters": [
      "chapters/01-intro.md",
      "chapters/02-guide.md"
    ]
  }
}
```

### 示例：验证外部 API

```python
#!/usr/bin/env python3
import json
import sys
import requests

def validate_api(config):
    try:
        # 检查外部 API 是否可达
        response = requests.get(config.get("api_url"), timeout=5)
        if response.status_code != 200:
            return {
                "status": "error",
                "action": "stop",
                "errors": ["API returned status code {}".format(response.status_code)]
            }

        return {
            "status": "success",
            "action": "continue",
            "data": {"api_validated": True}
        }
    except Exception as e:
        return {
            "status": "error",
            "action": "stop",
            "errors": ["API validation failed: {}".format(str(e))]
        }

# 在 main() 中
if request.get("action") == "execute_hook" and request.get("hook") == "before_build":
    response = validate_api(request.get("config", {}))
```

### 示例：清理构建目录

```bash
#!/bin/bash

# 从 stdin 读取 JSON
read -r request

# 清理构建目录
rm -rf build/html build/pdf build/epub

# 以成功响应
echo '{"status": "success", "action": "continue"}'
```

## after_parse

在每个章节的 markdown 被解析为 AST（抽象语法树）后触发，但在呈现前。

### 用例

- 修改或转换解析的内容
- 提取和处理元数据
- 验证内容结构
- 生成补充内容
- 扩展自定义语法

### 可用数据

钩子上下文包括解析的内容：

```json
{
  "context": {
    "phase": "parse",
    "content": "# Chapter Title\n\nContent here...",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html",
    "metadata": {
      "title": "Introduction",
      "author": "John Doe"
    }
  }
}
```

### 示例：扩展自定义宏

```python
#!/usr/bin/env python3
import json
import sys
import re
from datetime import datetime

def expand_macros(content):
    # 扩展 {{date}} 宏
    content = re.sub(
        r'{{date}}',
        datetime.now().strftime('%Y-%m-%d'),
        content
    )

    # 扩展 {{updated}} 宏为当前时间戳
    content = re.sub(
        r'{{updated}}',
        datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        content
    )

    return content

request = json.load(sys.stdin)
if request.get("hook") == "after_parse":
    context = request.get("context", {})
    original = context.get("content", "")
    modified = expand_macros(original)

    response = {
        "status": "success",
        "action": "continue",
        "modified_content": modified,
        "content_type": "markdown"
    }

    json.dump(response, sys.stdout)
```

### 示例：提取和验证链接

```python
#!/usr/bin/env python3
import json
import sys
import re

def extract_links(content):
    pattern = r'\[([^\]]+)\]\(([^\)]+)\)'
    matches = re.findall(pattern, content)
    return matches

request = json.load(sys.stdin)
if request.get("hook") == "after_parse":
    content = request.get("context", {}).get("content", "")
    links = extract_links(content)

    response = {
        "status": "success",
        "action": "continue",
        "metadata": {
            "links": links,
            "link_count": len(links)
        }
    }

    json.dump(response, sys.stdout)
```

## before_render

在章节从解析的内容呈现到输出格式（HTML、PDF 等）之前触发。

### 用例

- 为特定输出格式准备内容
- 应用格式特定的转换
- 注入呈现提示或元数据
- 优化输出格式的内容

### 可用数据

与 `after_parse` 相同，加上呈现上下文：

```json
{
  "context": {
    "phase": "render",
    "content": "# Chapter Title\n\nContent...",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html"
  }
}
```

### 示例：为 PDF 修改内容

```python
#!/usr/bin/env python3
import json
import sys

request = json.load(sys.stdin)

if request.get("hook") == "before_render":
    context = request.get("context", {})
    content = context.get("content", "")
    output_format = context.get("output_format", "")

    # 对于 PDF 输出，在 H1 标题前添加分页符
    if output_format == "pdf":
        modified = content.replace(
            "# ",
            "\n---\n# "  # 标题前的分页符
        )
    else:
        modified = content

    response = {
        "status": "success",
        "action": "continue",
        "modified_content": modified,
        "content_type": "markdown"
    }

    json.dump(response, sys.stdout)
```

## after_render

在章节被渲染到最终输出格式（HTML、PDF 等）后触发。

### 用例

- 后处理生成的输出
- 添加或修改输出文件
- 生成替代格式
- 缩小或优化输出
- 从渲染的内容中提取信息

### 可用数据

上下文包括渲染的输出：

```json
{
  "context": {
    "phase": "render",
    "content": "<h1>Chapter Title</h1>\n<p>Content...</p>",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html",
    "content_type": "html"
  }
}
```

### 示例：缩小 HTML

```python
#!/usr/bin/env python3
import json
import sys
import re

def minify_html(html):
    # 移除注释
    html = re.sub(r'<!--.*?-->', '', html, flags=re.DOTALL)

    # 移除标签之间不必要的空白
    html = re.sub(r'>\s+<', '><', html)

    # 移除行的开头/结尾空白
    html = '\n'.join(line.strip() for line in html.split('\n'))

    return html

request = json.load(sys.stdin)

if request.get("hook") == "after_render":
    context = request.get("context", {})

    if context.get("output_format") == "html":
        original = context.get("content", "")
        minified = minify_html(original)

        response = {
            "status": "success",
            "action": "continue",
            "modified_content": minified,
            "content_type": "html",
            "metadata": {
                "original_size": len(original),
                "minified_size": len(minified),
                "compression_ratio": f"{len(minified) / len(original) * 100:.1f}%"
            }
        }
    else:
        response = {
            "status": "success",
            "action": "continue"
        }

    json.dump(response, sys.stdout)
```

### 示例：生成 AMP 版本

```python
#!/usr/bin/env python3
import json
import sys
import re
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_render":
    context = request.get("context", {})

    if context.get("output_format") == "html":
        html = context.get("content", "")
        output_path = context.get("output_path", "")

        # 转换为 AMP 友好的 HTML
        amp_html = html
        amp_html = re.sub(r'<img ', '<amp-img ', amp_html)
        amp_html = re.sub(r'<iframe ', '<amp-iframe ', amp_html)

        # 写入 AMP 版本
        amp_path = output_path.replace('.html', '.amp.html')
        os.makedirs(os.path.dirname(amp_path), exist_ok=True)
        with open(amp_path, 'w') as f:
            f.write(amp_html)

        response = {
            "status": "success",
            "action": "continue",
            "data": {
                "amp_generated": True,
                "amp_path": amp_path
            }
        }
    else:
        response = {"status": "success", "action": "continue"}

    json.dump(response, sys.stdout)
```

## after_build

在所有章节被处理且整个构建完成后触发一次。

### 用例

- 构建后验证和验证
- 生成构建报告和统计信息
- 将文档上传到服务器
- 生成搜索索引
- 创建存档或备份
- 通知外部服务

### 可用数据

钩子上下文包括构建范围的信息：

```json
{
  "context": {
    "phase": "after_build",
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0"
    },
    "output_path": "build/",
    "output_format": "html",
    "chapters_processed": 5,
    "build_duration_ms": 2500
  }
}
```

### 示例：生成构建报告

```python
#!/usr/bin/env python3
import json
import sys
from datetime import datetime

request = json.load(sys.stdin)

if request.get("hook") == "after_build":
    context = request.get("context", {})

    report = {
        "timestamp": datetime.now().isoformat(),
        "book": context.get("book", {}).get("title"),
        "version": context.get("book", {}).get("version"),
        "format": context.get("output_format"),
        "chapters": context.get("chapters_processed", 0),
        "duration_ms": context.get("build_duration_ms", 0),
        "output_path": context.get("output_path")
    }

    with open("build/build-report.json", "w") as f:
        json.dump(report, f, indent=2)

    response = {
        "status": "success",
        "action": "continue",
        "data": {
            "report_generated": True,
            "report_path": "build/build-report.json"
        }
    }

    json.dump(response, sys.stdout)
```

### 示例：上传到服务器

```python
#!/usr/bin/env python3
import json
import sys
import requests
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_build":
    context = request.get("context", {})
    config = request.get("config", {})
    output_path = context.get("output_path")

    upload_url = config.get("upload_url")
    api_key = config.get("api_key")

    # 创建存档
    archive_path = "build/docs.tar.gz"
    os.system(f"tar -czf {archive_path} {output_path}")

    # 上传
    try:
        with open(archive_path, 'rb') as f:
            files = {'file': f}
            headers = {'Authorization': f'Bearer {api_key}'}
            response = requests.post(upload_url, files=files, headers=headers)

        if response.status_code == 200:
            return {
                "status": "success",
                "action": "continue",
                "data": {"uploaded": True}
            }
        else:
            return {
                "status": "error",
                "action": "continue",
                "warnings": [f"Upload failed: {response.status_code}"]
            }
    except Exception as e:
        return {
            "status": "warning",
            "action": "continue",
            "warnings": [f"Upload error: {str(e)}"]
        }
```

## before_serve

在开发服务器启动时触发一次，在监视文件更改之前。

### 用例

- 初始化开发模式功能
- 启动本地服务或数据库
- 设置实时重新加载配置
- 验证开发环境

### 可用数据

```json
{
  "context": {
    "phase": "serve",
    "server_host": "localhost",
    "server_port": 8000,
    "watch_paths": ["chapters/", "docs/"]
  }
}
```

### 示例：启动开发服务

```bash
#!/bin/bash

echo '{"status": "success", "action": "continue"}'

# 可以启动 Docker 容器、本地服务等
```

## after_serve

在开发服务器停止时触发一次。

### 用例

- 清理开发资源
- 停止本地服务
- 生成最终开发报告
- 存档开发工件

### 可用数据

```json
{
  "context": {
    "phase": "serve",
    "uptime_ms": 3600000,
    "files_watched": 45
  }
}
```

### 示例：清理开发服务

```python
#!/usr/bin/env python3
import json
import sys
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_serve":
    # 清理临时文件
    if os.path.exists(".dev-cache"):
        os.system("rm -rf .dev-cache")

    response = {
        "status": "success",
        "action": "continue",
        "data": {"cleanup_complete": True}
    }

    json.dump(response, sys.stdout)
```

## 钩子过滤

在 `book.yaml` 中指定插件应该接收哪些钩子：

```yaml
plugins:
  - name: build-plugin
    command: ./plugins/build.py
    hooks:
      - before_build
      - after_build
```

只有指定的钩子将被发送到插件，改进了性能。

## 完整生命周期示例

这是实现所有生命周期钩子的插件：

```python
#!/usr/bin/env python3
import json
import sys

class LifecyclePlugin:
    def __init__(self):
        self.stats = {
            "chapters_processed": 0,
            "links_found": 0,
            "errors": 0
        }

    def execute(self, hook, context):
        if hook == "before_build":
            print("Build starting...", file=sys.stderr)
            return {"status": "success", "action": "continue"}

        elif hook == "after_parse":
            self.stats["chapters_processed"] += 1
            return {"status": "success", "action": "continue"}

        elif hook == "before_render":
            return {"status": "success", "action": "continue"}

        elif hook == "after_render":
            return {"status": "success", "action": "continue"}

        elif hook == "after_build":
            print(f"Build complete: {self.stats}", file=sys.stderr)
            return {
                "status": "success",
                "action": "continue",
                "data": self.stats
            }

        elif hook == "before_serve":
            return {"status": "success", "action": "continue"}

        elif hook == "after_serve":
            return {"status": "success", "action": "continue"}

        return {"status": "success", "action": "continue"}

plugin = LifecyclePlugin()
request = json.load(sys.stdin)

if request.get("action") == "execute_hook":
    response = plugin.execute(
        request.get("hook"),
        request.get("context", {})
    )
    json.dump(response, sys.stdout)
```

参见 [插件 API](./api.md) 了解完整的 API 参考，参见 [构建插件](./building-a-plugin.md) 了解分步教程。
