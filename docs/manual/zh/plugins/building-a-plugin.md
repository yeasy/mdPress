# 构建插件：分步教程

本教程通过从头开始创建一个完整的 mdPress 插件来讲解。我们将构建一个字数计数插件，分析文档统计并生成报告。

## 项目设置

### 创建插件目录

```bash
mkdir -p my-project/plugins
cd my-project
```

### 目录结构

```
my-project/
├── book.yaml
├── chapters/
│   └── 01-intro.md
└── plugins/
    ├── word-count.py
    └── word-count.json
```

## 步骤 1：定义插件元数据

创建一个描述你的插件的 JSON 文件：

**`plugins/word-count.json`:**

```json
{
  "name": "word-count",
  "version": "1.0.0",
  "description": "Analyzes and reports word count statistics",
  "author": "Your Name",
  "license": "MIT",
  "hooks": ["after_parse", "after_build"],
  "configuration": {
    "min_words": 100,
    "warn_long_chapters": true,
    "warn_threshold": 5000
  }
}
```

## 步骤 2：创建插件脚本

创建主插件可执行文件。我们将使用 Python，但你可以使用任何语言。

**`plugins/word-count.py`:**

```python
#!/usr/bin/env python3
"""
mdPress 字数计数插件
分析字数并生成统计报告
"""

import json
import sys
import os
from pathlib import Path

class WordCountPlugin:
    """跟踪和报告文档统计信息。"""

    def __init__(self):
        self.name = "word-count"
        self.version = "1.0.0"
        self.config = {}
        self.stats = {
            "total_words": 0,
            "total_characters": 0,
            "chapter_stats": [],
            "long_chapters": []
        }

    def init(self, config):
        """使用配置初始化插件。"""
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Analyzes and reports word count statistics",
            "status": "success",
            "capabilities": ["after_parse", "after_build"]
        }

    def count_words(self, content):
        """计数内容中的字数。"""
        # 移除 markdown 语法以获得更精确的计数
        words = content.split()
        return len(words)

    def count_characters(self, content):
        """计数字符（不包括空白）。"""
        return len(content.replace(" ", "").replace("\n", ""))

    def execute_after_parse(self, context):
        """解析后处理内容。"""
        content = context.get("content", "")
        chapter_file = context.get("chapter_file", "")
        chapter_index = context.get("chapter_index", 0)

        # 计数字和字符
        word_count = self.count_words(content)
        char_count = self.count_characters(content)

        # 更新总数
        self.stats["total_words"] += word_count
        self.stats["total_characters"] += char_count

        # 跟踪每章统计
        chapter_stat = {
            "file": chapter_file,
            "index": chapter_index,
            "words": word_count,
            "characters": char_count
        }
        self.stats["chapter_stats"].append(chapter_stat)

        # 检查长章节
        threshold = self.config.get("warn_threshold", 5000)
        if self.config.get("warn_long_chapters") and word_count > threshold:
            self.stats["long_chapters"].append({
                "file": chapter_file,
                "words": word_count
            })

        return {
            "status": "success",
            "action": "continue",
            "metadata": {
                "word_count": word_count,
                "character_count": char_count
            }
        }

    def execute_after_build(self, context):
        """构建完成后生成报告。"""
        output_path = context.get("output_path", "build")

        # 计算平均值
        num_chapters = len(self.stats["chapter_stats"])
        avg_words = (self.stats["total_words"] // num_chapters
                    if num_chapters > 0 else 0)

        report = {
            "timestamp": context.get("timestamp", ""),
            "book": context.get("book", {}).get("title", "Unknown"),
            "statistics": {
                "total_words": self.stats["total_words"],
                "total_characters": self.stats["total_characters"],
                "chapters": num_chapters,
                "average_words_per_chapter": avg_words
            },
            "chapters": self.stats["chapter_stats"]
        }

        # 为长章节添加警告
        if self.stats["long_chapters"]:
            report["warnings"] = {
                "long_chapters": self.stats["long_chapters"],
                "message": f"Found {len(self.stats['long_chapters'])} "
                          f"chapters exceeding {self.config.get('warn_threshold')} words"
            }

        # 将报告写入文件
        report_path = os.path.join(output_path, "word-count-report.json")
        os.makedirs(output_path, exist_ok=True)

        with open(report_path, "w") as f:
            json.dump(report, f, indent=2)

        return {
            "status": "success",
            "action": "continue",
            "data": {
                "report_generated": True,
                "report_path": report_path,
                "total_words": self.stats["total_words"],
                "total_chapters": num_chapters
            }
        }

    def execute_hook(self, hook, context):
        """路由到适当的钩子处理程序。"""
        if hook == "after_parse":
            return self.execute_after_parse(context)
        elif hook == "after_build":
            return self.execute_after_build(context)
        else:
            return {
                "status": "success",
                "action": "continue"
            }

    def cleanup(self):
        """关闭时清理。"""
        return {
            "status": "success"
        }

    def process_request(self, request):
        """处理来自 mdPress 的传入请求。"""
        action = request.get("action")

        try:
            if action == "init":
                return self.init(request.get("config", {}))
            elif action == "execute_hook":
                return self.execute_hook(
                    request.get("hook"),
                    request.get("context", {})
                )
            elif action == "cleanup":
                return self.cleanup()
            else:
                return {
                    "status": "error",
                    "errors": [f"Unknown action: {action}"]
                }
        except Exception as e:
            return {
                "status": "error",
                "action": "stop",
                "errors": [f"Plugin error: {str(e)}"]
            }

def main():
    """入口点。"""
    plugin = WordCountPlugin()

    # 从 mdPress 读取请求
    request = json.load(sys.stdin)

    # 处理并响应
    response = plugin.process_request(request)
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

## 步骤 3：使脚本可执行

```bash
chmod +x plugins/word-count.py
```

## 步骤 4：在 book.yaml 中注册插件

将插件添加到你的 `book.yaml`：

```yaml
book:
  title: "My Documentation"
  author: "Your Name"

chapters:
  - chapters/01-intro.md

plugins:
  - name: word-count
    command: ./plugins/word-count.py
    config:
      min_words: 100
      warn_long_chapters: true
      warn_threshold: 5000
```

## 步骤 5：创建示例内容

创建一个测试章节：

**`chapters/01-intro.md`:**

```markdown
# Introduction

This is a sample chapter for testing the word-count plugin.
It contains enough text to demonstrate the plugin functionality.

## Features

The word-count plugin analyzes your documentation and provides:

- Total word count across all chapters
- Per-chapter statistics
- Character count
- Warnings for unusually long chapters
- JSON report output

This helps you understand the scope and complexity of your documentation.
```

## 步骤 6：测试插件

构建你的文档：

```bash
mdpress build
```

你应该看到插件执行并生成报告。检查输出：

```bash
cat build/word-count-report.json
```

输出：

```json
{
  "timestamp": "2026-03-23",
  "book": "My Documentation",
  "statistics": {
    "total_words": 87,
    "total_characters": 652,
    "chapters": 1,
    "average_words_per_chapter": 87
  },
  "chapters": [
    {
      "file": "chapters/01-intro.md",
      "index": 0,
      "words": 87,
      "characters": 652
    }
  ],
  "warnings": []
}
```

## 步骤 7：添加更多功能

### 在构建期间记录输出

增强插件以提供控制台反馈：

```python
def execute_after_parse(self, context):
    content = context.get("content", "")
    chapter_file = context.get("chapter_file", "")
    word_count = self.count_words(content)

    # 记录进度到 stderr（在构建期间可见）
    print(f"[word-count] Processing {chapter_file}: {word_count} words",
          file=sys.stderr)

    # ... 其余实现
```

### 添加阅读时间估计

```python
def estimate_reading_time(self, word_count, words_per_minute=200):
    """估计阅读时间（以分钟为单位）。"""
    return max(1, round(word_count / words_per_minute))

def execute_after_parse(self, context):
    # ... 现有代码 ...

    reading_time = self.estimate_reading_time(word_count)

    return {
        "status": "success",
        "action": "continue",
        "metadata": {
            "word_count": word_count,
            "reading_time_minutes": reading_time
        }
    }
```

## 步骤 8：调试提示

### 启用详细日志记录

修改你的插件以输出调试信息：

```python
import sys

def log_debug(msg):
    """将调试消息记录到 stderr。"""
    print(f"[word-count DEBUG] {msg}", file=sys.stderr)

# 在钩子中：
log_debug(f"Processing: {context.get('chapter_file')}")
log_debug(f"Word count: {word_count}")
```

使用可见的 stderr 运行：

```bash
mdpress build 2>&1 | tee build.log
```

### 独立测试插件

创建一个测试脚本来验证插件行为：

**`test-plugin.py`:**

```python
#!/usr/bin/env python3
import json
import subprocess

# 创建测试请求
request = {
    "action": "init",
    "config": {
        "warn_threshold": 5000
    }
}

# 运行插件
result = subprocess.run(
    ["./plugins/word-count.py"],
    input=json.dumps(request),
    capture_output=True,
    text=True
)

# 检查响应
response = json.loads(result.stdout)
print("Response:", json.dumps(response, indent=2))

if result.stderr:
    print("Errors:", result.stderr)
```

运行测试：

```bash
python3 test-plugin.py
```

### 打印请求/响应

在你的插件中，保存请求/响应以供检查：

```python
def process_request(self, request):
    # 保存请求以进行调试
    with open("/tmp/mdpress-plugin-request.json", "w") as f:
        json.dump(request, f, indent=2)

    response = self._process(request)

    # 保存响应
    with open("/tmp/mdpress-plugin-response.json", "w") as f:
        json.dump(response, f, indent=2)

    return response
```

检查文件：

```bash
cat /tmp/mdpress-plugin-request.json
cat /tmp/mdpress-plugin-response.json
```

## 步骤 9：处理错误

使你的插件强大：

```python
def execute_after_parse(self, context):
    try:
        content = context.get("content", "")
        if not content:
            return {
                "status": "warning",
                "action": "continue",
                "warnings": ["Empty content in chapter"]
            }

        word_count = self.count_words(content)

        if word_count == 0:
            return {
                "status": "warning",
                "action": "continue",
                "warnings": ["No words found in chapter"]
            }

        # ... 正常处理 ...

    except Exception as e:
        return {
            "status": "error",
            "action": "continue",  # 不停止构建
            "errors": [f"Failed to process chapter: {str(e)}"]
        }
```

## 步骤 10：分发

### 打包你的插件

创建用于共享的标准结构：

```
word-count-plugin/
├── README.md
├── LICENSE
├── word-count.py
├── word-count.json
├── examples/
│   └── book.yaml
└── tests/
    └── test_word_count.py
```

### 记录安装

**`README.md`:**

```markdown
# Word Count Plugin

Analyzes word count statistics for mdPress documentation.

## Installation

1. Copy `word-count.py` to your `plugins/` directory
2. Add to `book.yaml`:

\`\`\`yaml
plugins:
  - name: word-count
    command: ./plugins/word-count.py
    config:
      warn_threshold: 5000
\`\`\`

3. Run `mdpress build`

## Output

Generates `word-count-report.json` in build output directory.
```

## 完整可工作的示例

这是完整的、已测试的实现：

**`plugins/word-count.py`** （完整）：

```python
#!/usr/bin/env python3
"""mdPress 字数计数插件"""

import json
import sys
import os

class WordCountPlugin:
    def __init__(self):
        self.name = "word-count"
        self.version = "1.0.0"
        self.config = {}
        self.stats = {
            "total_words": 0,
            "chapter_stats": []
        }

    def init(self, config):
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Analyzes word count statistics",
            "status": "success"
        }

    def count_words(self, content):
        return len(content.split())

    def execute_hook(self, hook, context):
        if hook == "after_parse":
            content = context.get("content", "")
            word_count = self.count_words(content)
            self.stats["total_words"] += word_count
            self.stats["chapter_stats"].append({
                "file": context.get("chapter_file"),
                "words": word_count
            })
            return {
                "status": "success",
                "action": "continue",
                "metadata": {"word_count": word_count}
            }

        elif hook == "after_build":
            report_path = os.path.join(
                context.get("output_path", "build"),
                "word-count-report.json"
            )
            os.makedirs(os.path.dirname(report_path) or ".", exist_ok=True)
            with open(report_path, "w") as f:
                json.dump(self.stats, f, indent=2)
            return {
                "status": "success",
                "action": "continue",
                "data": {"report_path": report_path}
            }

        return {"status": "success", "action": "continue"}

    def process_request(self, request):
        action = request.get("action")
        if action == "init":
            return self.init(request.get("config", {}))
        elif action == "execute_hook":
            return self.execute_hook(
                request.get("hook"),
                request.get("context", {})
            )
        return {"status": "success"}

if __name__ == "__main__":
    plugin = WordCountPlugin()
    request = json.load(sys.stdin)
    json.dump(plugin.process_request(request), sys.stdout)
```

参见 [插件 API 参考](./api.md) 和 [生命周期钩子](./lifecycle-hooks.md) 了解更多关于插件开发的详情。
