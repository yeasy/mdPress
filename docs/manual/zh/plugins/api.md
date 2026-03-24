# 插件 API 参考

本文档指定了完整的插件 API，包括插件必须实现的接口和 mdPress 与插件之间交换的数据结构。

## 插件接口

每个 mdPress 插件都必须实现这些核心方法：

### Init

初始化插件。mdPress 启动时调用一次。

**请求：**
```json
{
  "action": "init",
  "config": {
    "min_words": 100,
    "custom_option": "value"
  }
}
```

**响应：**
```json
{
  "name": "word-counter",
  "version": "1.0.0",
  "description": "Counts words in documentation",
  "status": "success",
  "capabilities": ["after_parse", "after_build"],
  "errors": []
}
```

### Execute

在构建过程中执行钩子。使用不同的钩子上下文多次调用。

**请求：**
```json
{
  "action": "execute_hook",
  "hook": "after_parse",
  "context": {
    "content": "# Chapter Title\n\nSome content...",
    "phase": "parse",
    "chapter_index": 0,
    "chapter_file": "chapters/intro.md",
    "output_path": "build/intro.html",
    "output_format": "html"
  }
}
```

**响应：**
```json
{
  "status": "success",
  "action": "continue",
  "modified_content": "# Chapter Title\n\nModified content...",
  "metadata": {
    "word_count": 42
  },
  "errors": []
}
```

### Cleanup

清理资源。mdPress 关闭时调用。

**请求：**
```json
{
  "action": "cleanup"
}
```

**响应：**
```json
{
  "status": "success",
  "errors": []
}
```

## 插件元数据

插件必须提供描述自己的元数据：

### 必需字段

```json
{
  "name": "unique-plugin-name",
  "version": "1.0.0",
  "description": "Brief description of what the plugin does"
}
```

### 可选字段

```json
{
  "name": "plugin-name",
  "version": "1.0.0",
  "description": "What this plugin does",
  "author": "Your Name",
  "license": "MIT",
  "homepage": "https://github.com/...",
  "documentation": "https://docs.example.com/plugin"
}
```

## 钩子上下文

HookContext 包含有关当前处理阶段的所有信息：

### 上下文字段

```typescript
{
  "context": {
    // 当前正在处理的内容
    "content": "# Chapter Title\n\nContent here",

    // 文档的元数据
    "metadata": {
      "title": "Chapter Title",
      "author": "Author Name",
      "date": "2026-03-23",
      "custom": {}
    },

    // 构建阶段标识符
    "phase": "parse",  // 或 "render"、"serve" 等

    // 当前章节的索引（0 开始）
    "chapter_index": 0,

    // 源 markdown 文件路径
    "chapter_file": "chapters/01-intro.md",

    // 输出文件路径
    "output_path": "build/html/intro.html",

    // 目标输出格式
    "output_format": "html",  // 或 "pdf"、"epub"

    // 书籍配置
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0",
      "description": "Description"
    },

    // 构建配置
    "config": {
      "theme": "technical",
      "language": "en"
    }
  }
}
```

### 可用元数据

`metadata` 对象包含：

```json
{
  "metadata": {
    "title": "Chapter Title",
    "author": "Original Author",
    "date": "2026-03-23",
    "tags": ["tag1", "tag2"],
    "custom": {
      "custom_field": "custom_value"
    }
  }
}
```

## 钩子响应操作

处理后，插件可以指定 mdPress 应该做什么：

### Continue（默认）

继续到下一个插件或下一个阶段：

```json
{
  "status": "success",
  "action": "continue",
  "modified_content": "Updated content"
}
```

### Stop

停止处理并使构建失败：

```json
{
  "status": "error",
  "action": "stop",
  "errors": ["Critical error message"]
}
```

### Skip

跳过此钩子的剩余插件：

```json
{
  "status": "success",
  "action": "skip",
  "reason": "Conditions not met for further processing"
}
```

## 返回值

所有响应必须包括：

### Status

```json
{
  "status": "success"  // 或 "error" 或 "warning"
}
```

### Errors 数组

```json
{
  "errors": [
    "Error message 1",
    "Error message 2"
  ]
}
```

### 修改的内容

处理内容时，返回修改：

```json
{
  "modified_content": "Updated markdown or HTML",
  "content_type": "markdown"  // 或 "html"
}
```

### 自定义元数据

添加或修改元数据：

```json
{
  "metadata": {
    "word_count": 1250,
    "reading_time": "5 min",
    "custom_field": "value"
  }
}
```

### 数据输出

返回任意数据以用于日志记录或外部使用：

```json
{
  "data": {
    "processed_items": 5,
    "warnings": 2,
    "custom_metric": 42
  }
}
```

## 完整响应示例

这是显示所有可能字段的完整响应：

```json
{
  "name": "example-plugin",
  "status": "success",
  "action": "continue",
  "modified_content": "# Updated Content\n\nModified by plugin",
  "content_type": "markdown",
  "metadata": {
    "word_count": 42,
    "processed": true,
    "plugin_version": "1.0.0"
  },
  "data": {
    "links_checked": 15,
    "broken_links": 0,
    "processing_time_ms": 125
  },
  "errors": [],
  "warnings": []
}
```

## 错误处理

### 验证错误

为验证问题返回 `status: error`：

```json
{
  "status": "error",
  "action": "stop",
  "errors": [
    "Line 5: Invalid syntax in code block",
    "Line 12: Missing required metadata field"
  ]
}
```

### 警告

为非关键问题返回 `status: warning`：

```json
{
  "status": "warning",
  "action": "continue",
  "warnings": [
    "External link timeout (retrying): https://example.com",
    "Missing author name in chapter metadata"
  ],
  "modified_content": "..."
}
```

### 异常

在意外错误时响应：

```json
{
  "status": "error",
  "action": "stop",
  "errors": [
    "Unexpected error: Database connection failed",
    "Stack trace details (optional)"
  ]
}
```

## 协议详情

### JSON 编码

所有通信使用 UTF-8 编码的 JSON：

```python
import json
import sys

# 从 mdPress 读取
request = json.load(sys.stdin)

# 处理请求...

# 发送响应
json.dump(response, sys.stdout)
```

### 单个请求/响应

每个钩子执行遵循单个请求/响应模式：

1. mdPress 在 stdin 上发送 JSON
2. 插件读取并处理
3. 插件在 stdout 上发送响应
4. 连接关闭

### 超时处理

mdPress 等待插件响应并设置超时（默认：30 秒）。长时间运行的操作应该：

1. 发送中间响应
2. 优雅地实现超时
3. 在元数据中报告进度

长时间运行操作的示例：

```json
{
  "status": "success",
  "action": "continue",
  "metadata": {
    "progress": "Processing 42 of 100 items",
    "estimated_remaining": "2 seconds"
  }
}
```

## 配置架构

插件在 `init` 响应中定义其配置架构：

```json
{
  "name": "advanced-plugin",
  "configuration_schema": {
    "properties": {
      "enabled": {
        "type": "boolean",
        "default": true,
        "description": "Enable this plugin"
      },
      "output_format": {
        "type": "string",
        "default": "json",
        "enum": ["json", "csv", "xml"],
        "description": "Output format"
      },
      "max_items": {
        "type": "integer",
        "default": 100,
        "description": "Maximum items to process"
      }
    },
    "required": ["enabled"]
  }
}
```

用户在 `book.yaml` 中提供配置：

```yaml
plugins:
  - name: advanced-plugin
    command: ./plugins/advanced.py
    config:
      enabled: true
      output_format: csv
      max_items: 500
```

## 完整 API 示例

这是实现完整 API 的完整示例插件：

```python
#!/usr/bin/env python3
import json
import sys

class DocumentPlugin:
    def __init__(self):
        self.name = "full-example"
        self.version = "1.0.0"
        self.config = {}

    def init(self, config):
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Complete API example plugin",
            "status": "success"
        }

    def execute_hook(self, hook, context):
        if hook == "after_parse":
            return self.handle_after_parse(context)
        elif hook == "after_render":
            return self.handle_after_render(context)
        else:
            return {
                "status": "success",
                "action": "continue"
            }

    def handle_after_parse(self, context):
        content = context.get("content", "")
        word_count = len(content.split())

        return {
            "status": "success",
            "action": "continue",
            "metadata": {
                "word_count": word_count
            },
            "data": {
                "chapter": context.get("chapter_file"),
                "processed": True
            }
        }

    def handle_after_render(self, context):
        return {
            "status": "success",
            "action": "continue"
        }

    def cleanup(self):
        return {
            "status": "success"
        }

    def process_request(self, request):
        action = request.get("action")

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

def main():
    plugin = DocumentPlugin()
    request = json.load(sys.stdin)
    response = plugin.process_request(request)
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

参见 [生命周期钩子](./lifecycle-hooks.md) 了解详细的钩子文档，参见 [构建插件](./building-a-plugin.md) 了解教程。
