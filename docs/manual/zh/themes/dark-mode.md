# 深色模式

mdPress 包含内置的深色模式支持，具有自动系统检测、手动切换控制和持久化用户偏好设置。本指南说明了如何启用、自定义和扩展文档的深色模式。

## 启用深色模式

默认情况下，mdPress 中启用了深色模式。文档自动检测用户的系统偏好并应用相应的主题。

在 `book.yaml` 中显式启用深色模式：

```yaml
style:
  dark_mode: true
  dark_mode_default: system
```

## 深色模式偏好设置

使用 `dark_mode_default` 选项设置如何激活深色模式：

### 系统检测（默认）

```yaml
style:
  dark_mode_default: system
```

使用操作系统的主题偏好设置：
- 尊重 `prefers-color-scheme: dark` CSS 媒体查询
- 当用户更改其操作系统主题时自动切换
- 最易访问的选项，因为它与用户的系统设置一致

### 始终浅色

```yaml
style:
  dark_mode_default: light
```

为所有用户强制使用浅色模式。深色模式切换仍然可供希望手动切换的用户使用。

### 始终深色

```yaml
style:
  dark_mode_default: dark
```

默认强制使用深色模式。用户可以根据需要切换到浅色模式。

## 深色模式切换按钮

默认情况下，在顶部导航栏中显示深色模式切换按钮。用户可以单击它在浅色和深色模式之间切换。

```yaml
style:
  dark_mode_toggle: true
  dark_mode_toggle_position: top-right
```

### 位置选项

- `top-right` - 默认位置在右上角
- `top-left` - 左上角
- `bottom-right` - 右下角
- `header` - 在主标题/导航栏中

如果你只想使用系统检测，请禁用切换按钮：

```yaml
style:
  dark_mode_toggle: false
```

## Catppuccin 色板

mdPress 为深色模式使用 Catppuccin 色板，这是一组精心策划的柔和色彩，专为舒适的长期观看而设计。

### 浅色模式颜色 (Catppuccin Latte)

```css
--ctp-latte-rosewater: #f4dbd6;
--ctp-latte-flamingo: #f2cdcd;
--ctp-latte-pink: #f5c2e7;
--ctp-latte-mauve: #d6d7ee;
--ctp-latte-red: #e64553;
--ctp-latte-maroon: #d20f39;
--ctp-latte-peach: #fe640e;
--ctp-latte-yellow: #d79921;
--ctp-latte-green: #40a02f;
--ctp-latte-teal: #00b592;
--ctp-latte-sky: #04a5e5;
--ctp-latte-sapphire: #209fb5;
--ctp-latte-blue: #1e66f5;
--ctp-latte-lavender: #7287fd;
--ctp-latte-text: #4c4f69;
--ctp-latte-subtext1: #5c5f77;
--ctp-latte-subtext0: #6c6f85;
--ctp-latte-overlay2: #7c7f93;
--ctp-latte-overlay1: #8c8fa1;
--ctp-latte-overlay0: #9ca0b0;
--ctp-latte-surface2: #acb0be;
--ctp-latte-surface1: #bcc0cc;
--ctp-latte-surface0: #ccd0da;
--ctp-latte-base: #eff1f5;
--ctp-latte-mantle: #e6e9ef;
--ctp-latte-crust: #dce0e8;
```

### 深色模式颜色 (Catppuccin Mocha)

```css
--ctp-mocha-rosewater: #f5e0dc;
--ctp-mocha-flamingo: #f2cdcd;
--ctp-mocha-pink: #f5c2e7;
--ctp-mocha-mauve: #cba6f7;
--ctp-mocha-red: #f38ba8;
--ctp-mocha-maroon: #eba0ac;
--ctp-mocha-peach: #fab387;
--ctp-mocha-yellow: #f9e2af;
--ctp-mocha-green: #a6e3a1;
--ctp-mocha-teal: #94e2d5;
--ctp-mocha-sky: #89dceb;
--ctp-mocha-sapphire: #74c7ec;
--ctp-mocha-blue: #89b4fa;
--ctp-mocha-lavender: #b4befe;
--ctp-mocha-text: #cdd6f4;
--ctp-mocha-subtext1: #bac2de;
--ctp-mocha-subtext0: #a6adc8;
--ctp-mocha-overlay2: #9399b2;
--ctp-mocha-overlay1: #7f849c;
--ctp-mocha-overlay0: #6c7086;
--ctp-mocha-surface2: #585b70;
--ctp-mocha-surface1: #45475a;
--ctp-mocha-surface0: #313244;
--ctp-mocha-base: #1e1e2e;
--ctp-mocha-mantle: #181825;
--ctp-mocha-crust: #11111b;
```

## 深色模式的 CSS 自定义属性

使用 `prefers-color-scheme` 媒体查询应用深色模式样式：

```css
/* 浅色模式（默认） */
:root {
  --background: #ffffff;
  --text: #2c3e50;
  --border: #ecf0f1;
  --code-bg: #f4f4f4;
}

/* 深色模式 */
@media (prefers-color-scheme: dark) {
  :root {
    --background: #1e1e2e;
    --text: #cdd6f4;
    --border: #313244;
    --code-bg: #313244;
  }
}
```

在你的选择器中应用这些变量：

```css
.content {
  background: var(--background);
  color: var(--text);
}

.content code {
  background: var(--code-bg);
  border-color: var(--border);
}
```

## `html.dark` 类

当深色模式处于活动状态时，mdPress 将 `dark` 类添加到 `<html>` 元素。使用此来获得更精细的控制：

```css
/* 浅色模式 */
.content {
  background: white;
  color: #2c3e50;
}

/* 深色模式 - 使用 html.dark 选择器 */
html.dark .content {
  background: #1e1e2e;
  color: #cdd6f4;
}
```

当你需要不同的样式超越简单的颜色交换时，这种方法很有用：

```css
html.dark .content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
}

html.light .content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}
```

## LocalStorage 持久化

用户深色模式偏好自动保存到浏览器 localStorage，键为 `mdpress-theme-preference`。

值可以是：
- `"light"` - 用户偏好浅色模式
- `"dark"` - 用户偏好深色模式
- `"system"` - 用户偏好系统默认值（或未设置）

### 检查用户偏好

JavaScript 可以检查保存的偏好：

```javascript
const preference = localStorage.getItem('mdpress-theme-preference');
console.log(preference); // "light"、"dark" 或 null
```

### 以编程方式设置主题

如果需要通过 JavaScript 设置主题：

```javascript
// 设置为深色模式
localStorage.setItem('mdpress-theme-preference', 'dark');
document.documentElement.classList.add('dark');

// 设置为浅色模式
localStorage.setItem('mdpress-theme-preference', 'light');
document.documentElement.classList.remove('dark');

// 重置为系统偏好
localStorage.removeItem('mdpress-theme-preference');
```

## 自定义深色模式颜色

在你的自定义 CSS 中覆盖 Catppuccin 颜色：

```yaml
style:
  theme: technical
  custom_css: |
    :root {
      --primary-color: #1A5490;
    }

    @media (prefers-color-scheme: dark) {
      :root {
        --primary-color: #89dceb;
        --background: #181825;
        --text: #cdd6f4;
      }
    }

    @media (prefers-color-scheme: light) {
      :root {
        --primary-color: #1A5490;
        --background: #ffffff;
        --text: #2c3e50;
      }
    }
```

### 深色模式主题变体

为每个主题创建单独的色彩方案：

```css
/* Technical 主题 - 浅色 */
:root[data-theme="technical"] {
  --primary-color: #1A5490;
  --code-bg: #f4f4f4;
}

/* Technical 主题 - 深色 */
@media (prefers-color-scheme: dark) {
  :root[data-theme="technical"] {
    --primary-color: #89dceb;
    --code-bg: #313244;
  }
}

/* Elegant 主题 - 浅色 */
:root[data-theme="elegant"] {
  --primary-color: #34495e;
  --serif-font: georgia;
}

/* Elegant 主题 - 深色 */
@media (prefers-color-scheme: dark) {
  :root[data-theme="elegant"] {
    --primary-color: #cba6f7;
  }
}
```

## 深色模式中的语法高亮

代码高亮主题自动适应深色模式：

```yaml
style:
  code_theme: monokai
  code_theme_dark: monokai-pro
```

### 可用的代码主题

浅色模式：
- `monokai`
- `solarized-light`
- `atom-one-light`
- `github-light`

深色模式：
- `monokai`（默认）
- `monokai-pro`
- `dracula`
- `nord`
- `solarized-dark`
- `atom-one-dark`

### 自定义语法高亮

覆盖深色模式的代码高亮颜色：

```css
@media (prefers-color-scheme: dark) {
  .hljs {
    background: #11111b;
    color: #cdd6f4;
  }

  .hljs-string {
    color: #a6e3a1;
  }

  .hljs-number,
  .hljs-literal {
    color: #89dceb;
  }

  .hljs-attr {
    color: #f9e2af;
  }

  .hljs-keyword {
    color: #f5c2e7;
  }

  .hljs-comment {
    color: #6c7086;
  }
}
```

## 测试深色模式

### 使用浏览器开发工具

1. 在浏览器中打开你的文档
2. 打开开发工具（F12 或 Cmd+Shift+I）
3. 按 Cmd+Shift+P（Linux/Windows 上为 Ctrl+Shift+P）
4. 搜索"模拟 CSS 媒体特性 prefers-color-scheme"
5. 选择"prefers-color-scheme: dark"或"prefers-color-scheme: light"

### 使用 mdPress Serve

开发服务器尊重你系统的深色模式设置：

```bash
mdpress serve
```

在顶部导航中切换深色模式进行测试。

## 可访问性考虑

深色模式改善了对光敏感用户的可访问性并减少了眼睛疲劳。最佳实践：

1. **对比度比率**：确保两种模式中正文至少有 4.5:1 的对比度
2. **颜色不是唯一的**：不要仅依赖颜色来传达意义
3. **减少运动**：在过渡中尊重 `prefers-reduced-motion`
4. **测试两种模式**：在浅色和深色模式下测试你的文档

对比度安全的深色模式颜色示例：

```css
@media (prefers-color-scheme: dark) {
  :root {
    --text: #e5e7eb;           /* 在深色背景上对比度良好 */
    --text-secondary: #d1d5db; /* 略深一点的辅助文本 */
    --background: #111827;     /* 纯深色背景 */
  }
}
```

## 完整示例

这是一个完整的深色模式配置：

```yaml
book:
  title: "Documentation"
  author: "Team"

style:
  theme: technical
  dark_mode: true
  dark_mode_default: system
  dark_mode_toggle: true
  dark_mode_toggle_position: top-right
  code_theme: atom-one-light
  code_theme_dark: atom-one-dark

  custom_css: |
    :root {
      --primary-color: #1A5490;
      --background: #ffffff;
      --text: #2c3e50;
    }

    @media (prefers-color-scheme: dark) {
      :root {
        --primary-color: #89dceb;
        --background: #1e1e2e;
        --text: #cdd6f4;
      }
    }

    .content {
      background: var(--background);
      color: var(--text);
    }

    html.dark .content code {
      background: #313244;
      color: #f9e2af;
    }
```

参见 [内置主题](./builtin-themes.md) 和 [自定义 CSS](./custom-css.md) 了解更多自定义选项。
