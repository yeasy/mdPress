# 深色模式

mdPress 的 Web 输出（`site`、独立 `html` 和 `serve` 预览）内置深色模式支持：自动跟随系统、手动切换、并持久化用户偏好。深色模式始终可用——无需任何配置。

**注意：**深色模式仅作用于 Web 输出。PDF、Typst 和 EPUB 输出不受影响。

## 主题切换器

生成的站点在导航栏中提供三档主题切换器：

- **浅色** —— 强制浅色模式
- **系统** —— 跟随操作系统偏好（`prefers-color-scheme`）
- **深色** —— 强制深色模式

默认是**系统**：页面跟随操作系统主题，用户切换系统主题时自动更新。

## 工作原理

深色模式激活时，mdPress 会在 `<html>` 元素上添加 `dark` 类。所有深色样式都基于这个类，因此自定义 CSS 很容易挂接：

```css
/* 浅色模式 */
.chapter-content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 深色模式 */
html.dark .chapter-content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
}
```

页面绘制前有一段内联脚本会先应用已保存的偏好，因此深色模式用户在页面之间跳转时不会出现浅色闪烁。

## 偏好持久化

用户的选择保存在浏览器 localStorage 的 `mdpress-theme` 键下：

- `"light"` —— 用户选择了浅色模式
- `"dark"` —— 用户选择了深色模式
- 键不存在 —— 跟随系统偏好（默认）

可以直接用 JavaScript 读写：

```javascript
// 查看已保存的偏好
const preference = localStorage.getItem('mdpress-theme');
console.log(preference); // "light"、"dark" 或 null

// 强制深色模式
localStorage.setItem('mdpress-theme', 'dark');
document.documentElement.classList.add('dark');

// 恢复为跟随系统
localStorage.removeItem('mdpress-theme');
```

## 深色模式下的语法高亮

代码高亮基于 CSS 类实现，每个配置的代码配色都会自动获得深色模式的对应样式：

- 本身就是深色的配色保持自身：`monokai` → `monokai`、`dracula` → `dracula`
- 其余配色统一配对 `github-dark`（例如 `github` → `github-dark`、`bw` → `github-dark`）

没有单独的深色代码配色选项——配对是自动的，代码在两种模式下都保持可读，无需额外配置。表格（包括斑马纹）和代码块同样针对深色模式做了完整样式。

```yaml
style:
  code_theme: github   # 浅色模式：github；深色模式自动使用 github-dark
```

`style.code_theme` 留空即继承主题的代码配色（technical/elegant 为 `github`，minimal 为 `bw`）。

## 自定义深色模式颜色

通过 `style.custom_css`（指向一个 CSS 文件）在生成样式之上叠加覆盖：

```yaml
style:
  theme: technical
  custom_css: styles/overrides.css
```

```css
/* styles/overrides.css */

/* 通过 html.dark 类调整深色模式 */
html.dark {
  --md-accent: #7cb7ff;
}

/* 或直接跟随系统偏好 */
@media (prefers-color-scheme: dark) {
  .my-callout {
    border-color: #444;
  }
}
```

凡是希望页面内切换器能控制的样式，优先使用 `html.dark` 选择器；仅用 `prefers-color-scheme` 无法响应切换器。

## 测试深色模式

### 使用浏览器 DevTools

1. 在浏览器中打开你的文档
2. 打开 DevTools（F12 或 Cmd+Shift+I）
3. 按 Cmd+Shift+P（Linux/Windows 上为 Ctrl+Shift+P）
4. 搜索 "Emulate CSS media feature prefers-color-scheme"
5. 选择 "prefers-color-scheme: dark" 或 "prefers-color-scheme: light"

### 使用 mdPress Serve

开发服务器生成同样的站点输出，包括切换器：

```bash
mdpress serve
```

用导航栏中的主题切换器测试两种模式。

## 可访问性考虑

深色模式可以帮助对强光敏感的用户，并减轻眼部疲劳。自定义时的最佳实践：

1. **对比度**：确保两种模式下正文文本至少有 4.5:1 的对比度
2. **不止用颜色**：不要仅依赖颜色传达含义
3. **减少动效**：在过渡动画中尊重 `prefers-reduced-motion`
4. **两种模式都测试**：在浅色和深色模式下都检查你的文档

更多自定义选项参见[内置主题](./builtin-themes.md)和[自定义 CSS](./custom-css.md)。
