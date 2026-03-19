# `mdpress themes`

[English](themes.md)

## 作用

查看 `mdpress` 内置主题。当前主题相关命令包括：

- `mdpress themes list`
- `mdpress themes show <theme-name>`
- `mdpress themes preview`

## 可用主题

当前内置主题有：

| 主题 | 说明 |
| --- | --- |
| `technical` | 面向技术书和文档的专业风格 |
| `elegant` | 更适合随笔、学术和文学类排版 |
| `minimal` | 极简、高对比、偏阅读效率 |

## `mdpress themes list`

### 语法

```bash
mdpress themes list
```

### 作用

列出所有内置主题，包括名称、描述、主要配色和特性。

### 示例

```bash
mdpress themes list
```

## `mdpress themes show`

### 语法

```bash
mdpress themes show <theme-name>
```

### 参数

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `<theme-name>` | 是 | 主题名，例如 `technical`、`elegant`、`minimal`。 |

### 作用

展示单个主题的详情，包括：

- 描述
- 作者、版本、许可证
- 特性列表
- 颜色配置
- `book.yaml` 里的示例配置方式

### 示例

```bash
mdpress themes show technical
mdpress themes show elegant
mdpress themes show minimal
```

## `mdpress themes preview`

### 语法

```bash
mdpress themes preview [flags]
```

### 参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-o, --output <path>` | `themes-preview.html` | 生成的 HTML 预览页输出路径。 |
| `-v, --verbose` | 关闭 | 输出详细日志。 |
| `-q, --quiet` | 关闭 | 只输出错误。 |

### 作用

生成一个自包含的 HTML 页面，把所有内置主题应用到同一份示例内容上。它适合做主题对比、设计评审，以及为文档或发布说明准备截图。

### 示例

```bash
mdpress themes preview
mdpress themes preview --output ./artifacts/themes.html
```

## 注意事项

- 输入不存在的主题名时，命令会报错并提示先运行 `mdpress themes list`。
- `themes` 只负责查看内置主题信息，不会修改项目文件。
- `--config` 虽然是全局参数，但当前 `themes` 不会使用它。
