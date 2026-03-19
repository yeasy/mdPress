# `mdpress themes`

[中文说明](themes_zh.md)

## Purpose

Inspect the built-in `mdpress` themes. The current theme-related commands are:

- `mdpress themes list`
- `mdpress themes show <theme-name>`
- `mdpress themes preview`

## Available Themes

Current built-in themes:

| Theme | Description |
| --- | --- |
| `technical` | A professional style for technical books and documentation |
| `elegant` | Better suited for essays, academic writing, and literary layouts |
| `minimal` | Minimal, high-contrast, reading-efficiency oriented |

## `mdpress themes list`

### Syntax

```bash
mdpress themes list
```

### Purpose

List all built-in themes, including their names, descriptions, primary colors, and main characteristics.

### Example

```bash
mdpress themes list
```

## `mdpress themes show`

### Syntax

```bash
mdpress themes show <theme-name>
```

### Arguments

| Argument | Required | Description |
| --- | --- | --- |
| `<theme-name>` | Yes | Theme name, such as `technical`, `elegant`, or `minimal`. |

### Purpose

Show detailed information about one theme, including:

- Description
- Author, version, and license
- Feature list
- Color configuration
- Example `book.yaml` usage

### Examples

```bash
mdpress themes show technical
mdpress themes show elegant
mdpress themes show minimal
```

## `mdpress themes preview`

### Syntax

```bash
mdpress themes preview [flags]
```

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-o, --output <path>` | `themes-preview.html` | Output path for the generated HTML preview. |
| `-v, --verbose` | off | Print detailed logs. |
| `-q, --quiet` | off | Print errors only. |

### Purpose

Generate a self-contained HTML page that applies each built-in theme to the same sample content. This is useful for design review, theme comparison, and screenshot generation for docs or release notes.

### Examples

```bash
mdpress themes preview
mdpress themes preview --output ./artifacts/themes.html
```

## Notes

- If the theme name does not exist, the command returns an error and suggests running `mdpress themes list` first.
- `themes` only inspects built-in theme information and does not modify project files.
- `--config` appears in global flags, but `themes` does not use it.
