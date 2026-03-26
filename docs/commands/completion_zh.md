# `mdpress completion`

[English](completion.md)

## 作用

为 Shell 生成自动补全脚本。当前支持：

- `bash`
- `zsh`
- `fish`
- `powershell`

## 语法

```bash
mdpress completion <shell>
```

## 参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `<shell>` | 是 | Shell 类型，可选 `bash`、`zsh`、`fish`、`powershell`。 |

## 常见用法

### Bash

```bash
mdpress completion bash
source <(mdpress completion bash)
```

### Zsh

```bash
mdpress completion zsh
source <(mdpress completion zsh)
```

## 子命令参数

`bash` 和 `fish` 补全子命令当前支持：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--no-descriptions` | 关闭 | 关闭补全项描述。 |

## 注意事项

- 补全脚本输出到标准输出，通常需要重定向到文件，或通过 `source <(...)` 立即载入。
- `bash` 的帮助信息明确要求系统安装 `bash-completion`。
- `zsh` 如果还没有启用补全，需要先在环境里执行 `autoload -U compinit; compinit`。
- `--config` 虽然是全局参数，但当前 `completion` 不会使用它。
