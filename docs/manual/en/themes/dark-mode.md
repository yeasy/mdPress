# Dark Mode

mdPress includes built-in dark mode support with automatic system detection, manual toggle controls, and persistent user preferences. This guide explains how to enable, customize, and extend dark mode for your documentation.

## Enabling Dark Mode

Dark mode is enabled by default in mdPress. The documentation automatically detects the user's system preference and applies the appropriate theme.

Enable dark mode explicitly in `book.yaml`:

```yaml
style:
  dark_mode: true
  dark_mode_default: system
```

## Dark Mode Preferences

Set how dark mode is activated with the `dark_mode_default` option:

### System Detection (Default)

```yaml
style:
  dark_mode_default: system
```

Uses the operating system's theme preference:
- Respects `prefers-color-scheme: dark` CSS media query
- Automatically switches when the user changes their OS theme
- Most accessible option as it aligns with user's system settings

### Always Light

```yaml
style:
  dark_mode_default: light
```

Forces light mode for all users. The dark mode toggle is still available for users who prefer to switch manually.

### Always Dark

```yaml
style:
  dark_mode_default: dark
```

Forces dark mode by default. Users can toggle to light mode if preferred.

## The Dark Mode Toggle

A dark mode toggle button appears in the top navigation bar by default. Users can click it to switch between light and dark modes.

```yaml
style:
  dark_mode_toggle: true
  dark_mode_toggle_position: top-right
```

### Positioning Options

- `top-right` - Default position in the top-right corner
- `top-left` - Top-left corner
- `bottom-right` - Bottom-right corner
- `header` - In the main header/navbar

Disable the toggle if you want to use system detection only:

```yaml
style:
  dark_mode_toggle: false
```

## Catppuccin Color Palette

mdPress uses the Catppuccin color palette for dark mode, a carefully curated set of pastel colors designed for comfortable long-term viewing.

### Light Mode Colors (Catppuccin Latte)

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

### Dark Mode Colors (Catppuccin Mocha)

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

## CSS Custom Properties for Dark Mode

Use the `prefers-color-scheme` media query to apply dark mode styles:

```css
/* Light mode (default) */
:root {
  --background: #ffffff;
  --text: #2c3e50;
  --border: #ecf0f1;
  --code-bg: #f4f4f4;
}

/* Dark mode */
@media (prefers-color-scheme: dark) {
  :root {
    --background: #1e1e2e;
    --text: #cdd6f4;
    --border: #313244;
    --code-bg: #313244;
  }
}
```

Apply these variables in your selectors:

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

## The `html.dark` Class

When dark mode is active, mdPress adds the `dark` class to the `<html>` element. Use this for more granular control:

```css
/* Light mode */
.content {
  background: white;
  color: #2c3e50;
}

/* Dark mode - using html.dark selector */
html.dark .content {
  background: #1e1e2e;
  color: #cdd6f4;
}
```

This approach is useful when you need different styling that goes beyond simple color swaps:

```css
html.dark .content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
}

html.light .content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}
```

## LocalStorage Persistence

User dark mode preferences are automatically saved to browser localStorage under the key `mdpress-theme-preference`.

The value can be:
- `"light"` - User prefers light mode
- `"dark"` - User prefers dark mode
- `"system"` - User prefers system default (or not set)

### Checking User Preference

JavaScript can check the saved preference:

```javascript
const preference = localStorage.getItem('mdpress-theme-preference');
console.log(preference); // "light", "dark", or null
```

### Programmatically Setting Theme

If you need to set the theme via JavaScript:

```javascript
// Set to dark mode
localStorage.setItem('mdpress-theme-preference', 'dark');
document.documentElement.classList.add('dark');

// Set to light mode
localStorage.setItem('mdpress-theme-preference', 'light');
document.documentElement.classList.remove('dark');

// Reset to system preference
localStorage.removeItem('mdpress-theme-preference');
```

## Customizing Dark Mode Colors

Override Catppuccin colors in your custom CSS:

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

### Dark Mode Theme Variants

Create separate color schemes for each theme:

```css
/* Technical theme - light */
:root[data-theme="technical"] {
  --primary-color: #1A5490;
  --code-bg: #f4f4f4;
}

/* Technical theme - dark */
@media (prefers-color-scheme: dark) {
  :root[data-theme="technical"] {
    --primary-color: #89dceb;
    --code-bg: #313244;
  }
}

/* Elegant theme - light */
:root[data-theme="elegant"] {
  --primary-color: #34495e;
  --serif-font: georgia;
}

/* Elegant theme - dark */
@media (prefers-color-scheme: dark) {
  :root[data-theme="elegant"] {
    --primary-color: #cba6f7;
  }
}
```

## Syntax Highlighting in Dark Mode

Code highlighting themes automatically adapt to dark mode:

```yaml
style:
  code_theme: monokai
  code_theme_dark: monokai-pro
```

### Available Code Themes

Light mode:
- `monokai`
- `solarized-light`
- `atom-one-light`
- `github-light`

Dark mode:
- `monokai` (default)
- `monokai-pro`
- `dracula`
- `nord`
- `solarized-dark`
- `atom-one-dark`

### Custom Syntax Highlighting

Override code highlight colors for dark mode:

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

## Testing Dark Mode

### Using Browser DevTools

1. Open your documentation in a browser
2. Open DevTools (F12 or Cmd+Shift+I)
3. Press Cmd+Shift+P (or Ctrl+Shift+P on Linux/Windows)
4. Search for "Emulate CSS media feature prefers-color-scheme"
5. Select "prefers-color-scheme: dark" or "prefers-color-scheme: light"

### Using mdPress Serve

The development server respects your system's dark mode setting:

```bash
mdpress serve
```

Toggle dark mode in the top navigation to test.

## Accessibility Considerations

Dark mode improves accessibility for users with light sensitivity and reduces eye strain. Best practices:

1. **Contrast Ratios**: Ensure at least 4.5:1 contrast for body text in both modes
2. **Color Not Alone**: Don't rely solely on color to convey meaning
3. **Reduced Motion**: Respect `prefers-reduced-motion` in transitions
4. **Test Both Modes**: Test your documentation in both light and dark modes

Example contrast-safe dark mode colors:

```css
@media (prefers-color-scheme: dark) {
  :root {
    --text: #e5e7eb;           /* Good contrast on dark backgrounds */
    --text-secondary: #d1d5db; /* Slightly darker for secondary text */
    --background: #111827;     /* Pure dark background */
  }
}
```

## Complete Example

Here's a complete dark mode configuration:

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

See [Built-in Themes](./builtin-themes.md) and [Custom CSS](./custom-css.md) for more customization options.
