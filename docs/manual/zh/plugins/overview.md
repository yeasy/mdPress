# 插件系统概述

mdPress 插件系统允许你扩展和自定义构建过程，而无需修改核心代码。插件是连接到文档生成管道的外部可执行文件，可实现自定义内容处理、验证和生成等功能。

## 插件能做什么

插件可以在构建管道中执行各种任务：

### 内容处理
- 在呈现前转换内容（例如，自定义语法、宏）
- 动态修改 markdown 或 HTML
- 注入内容或元数据

### 验证
- 检查链接和引用
- 验证代码示例
- 验证文档约定
- 检查元数据完整性

### 生成
- 生成目录
- 创建索引和词汇表
- 从规范生成图表
- 创建示例输出

### 分析
- 计数字和统计信息
- 跟踪变更和贡献
- 监控文档质量

### 集成
- 发布到外部服务
- 提交给搜索引擎
- 生成替代格式
- 后处理输出

### 自定义功能
- 添加特定领域的功能
- 实现自定义构建步骤
- 扩展模板化系统

## 何时使用插件

当你需要以下功能时使用插件：

1. **扩展核心功能**，无需修改 mdPress
2. **与外部工具集成**（linter、生成器、API）
3. **自动化构建过程中的重复任务**
4. **实现特定于文档的自定义业务逻辑**
5. **跨团队或项目共享可重用的构建扩展**

不要使用插件来：
- 简单的样式更改（使用 [自定义 CSS](../themes/custom-css.md)）
- 主题自定义（使用 [内置主题](../themes/builtin-themes.md)）
- 基本的内容组织（使用 markdown 和配置）

## 插件如何工作

mdPress 使用外部可执行模型，其中插件是通过 JSON 在 stdin/stdout 上通信的独立程序：

```
┌─────────────┐
│  mdPress    │
│  (Go)       │
└──────┬──────┘
       │ 发送 JSON
       ▼
┌──────────────────┐
│  插件（任何      │
│  语言）          │
│  (Python/Node.js)│
└──────┬───────────┘
       │ 以 JSON 响应
       ▼
┌─────────────┐
│  mdPress    │
│  继续       │
└─────────────┘
```

### 主要优势

- **语言无关**：用 Python、Node.js、Go、Ruby 或任何语言编写插件
- **安全**：插件作为单独的进程运行，无法直接访问 mdPress 内部
- **松耦合**：插件不依赖 mdPress API 版本
- **易于测试**：插件是独立的程序，可以隔离测试
- **易于调试**：标准 JSON 协议使调试简单明了

## 加载插件

插件在 `book.yaml` 的 `plugins` 部分定义：

```yaml
plugins:
  - name: word-counter
    command: ./plugins/word-counter.py
    config:
      min_words: 100

  - name: link-checker
    command: ./plugins/link-checker
    enabled: true

  - name: custom-syntax
    command: python3
    args:
      - ./plugins/syntax.py
```

### 插件配置选项

- `name`（必需）- 插件的唯一标识符
- `command`（必需）- 可执行文件的路径
- `args`（可选）- 传递给命令的参数
- `config`（可选）- 传递给插件的自定义配置
- `enabled`（可选）- 启用/禁用插件（默认：true）
- `hooks`（可选）- 要运行的特定钩子（默认：全部）

## 执行顺序

插件按照它们在 `book.yaml` 中定义的顺序执行。构建过程遵循此顺序：

```
1. mdPress 开始构建
2. before_build 钩子运行（按顺序）
3. 解析内容
4. after_parse 钩子运行（按顺序）
5. before_render 钩子运行（按顺序）
6. 渲染到输出格式
7. after_render 钩子运行（按顺序）
8. after_build 钩子运行（按顺序）
9. 构建完成
```

对于 serve 模式：
```
1. before_serve 钩子运行
2. 服务器启动（监视文件更改）
3. 文件更改时：触发重新构建钩子
4. after_serve 钩子运行（服务器停止时）
```

### 控制执行顺序

如果插件执行顺序很重要，请在 `book.yaml` 中适当地组织它们：

```yaml
plugins:
  # 这首先运行
  - name: preprocess
    command: ./plugins/preprocess.py

  # 这在 preprocess 之后运行
  - name: validate
    command: ./plugins/validate.py

  # 这最后运行
  - name: postprocess
    command: ./plugins/postprocess.py
```

## 插件生命周期

每个插件经历这些阶段：

1. **加载** - mdPress 读取插件配置
2. **初始化** - 插件可执行文件启动，读取 JSON，初始化
3. **执行钩子** - 插件按需要处理钩子
4. **清理** - 插件执行清理，发送最终响应

## 示例：简单字数计数器

这是一个演示该模型的最小字数计数插件：

```python
#!/usr/bin/env python3
import json
import sys

def count_words(text):
    return len(text.split())

def main():
    # 从 mdPress 读取插件请求
    request = json.load(sys.stdin)

    # 根据钩子处理
    response = {
        "name": "word-counter",
        "version": "1.0",
        "status": "success",
        "data": {
            "word_count": count_words(request['content']),
            "characters": len(request['content'])
        }
    }

    # 将响应发送回 mdPress
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

## 下一步

- [插件 API 参考](./api.md) - 完整的 API 规范
- [生命周期钩子](./lifecycle-hooks.md) - 详细的钩子文档
- [构建插件](./building-a-plugin.md) - 分步教程
