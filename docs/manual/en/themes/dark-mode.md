# Dark Mode

mdPress includes built-in dark mode support in its web outputs (`site`, standalone `html`, and `serve` preview), with automatic system detection, a manual toggle, and a persistent user preference. Dark mode is always available — no configuration is needed to enable it.

**Note:** dark mode applies to web outputs only. PDF, Typst, and EPUB output are not affected.

## The Theme Switcher

Generated sites show a three-way theme switcher in the navigation bar:

- **Light** — force light mode
- **System** — follow the operating system preference (`prefers-color-scheme`)
- **Dark** — force dark mode

The default is **System**: the page follows the OS theme and switches automatically when the user changes it.

## How It Works

When dark mode is active, mdPress adds the `dark` class to the `<html>` element. All dark styling keys off this class, which makes it easy to hook into with custom CSS:

```css
/* Light mode */
.chapter-content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* Dark mode */
html.dark .chapter-content {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.5);
}
```

An inline script applies the saved preference before the page paints, so there is no light-mode flash when a dark-mode user navigates between pages.

## Preference Persistence

The user's choice is saved to browser localStorage under the key `mdpress-theme`:

- `"light"` — user chose light mode
- `"dark"` — user chose dark mode
- key absent — follow the system preference (default)

JavaScript can read or set it directly:

```javascript
// Check the saved preference
const preference = localStorage.getItem('mdpress-theme');
console.log(preference); // "light", "dark", or null

// Force dark mode
localStorage.setItem('mdpress-theme', 'dark');
document.documentElement.classList.add('dark');

// Reset to system preference
localStorage.removeItem('mdpress-theme');
```

## Syntax Highlighting in Dark Mode

Code highlighting is class-based, and every configured code theme automatically gets a dark-mode counterpart:

- Styles that are already dark keep themselves: `monokai` → `monokai`, `dracula` → `dracula`
- Everything else pairs with `github-dark` (e.g. `github` → `github-dark`, `bw` → `github-dark`)

There is no separate dark-mode code theme option — the pairing is automatic, so code stays readable in both modes without extra configuration. Tables (including zebra striping) and code blocks are fully styled for dark mode as well.

```yaml
style:
  code_theme: github   # light mode: github; dark mode: github-dark, automatically
```

Leaving `style.code_theme` empty inherits the theme's code style (`github` for technical/elegant, `bw` for minimal).

## Customizing Dark Mode Colors

Layer overrides on top of the generated styles with `style.custom_css`, which points to a CSS file:

```yaml
style:
  theme: technical
  custom_css: styles/overrides.css
```

```css
/* styles/overrides.css */

/* Tweak dark mode via the html.dark class */
html.dark {
  --md-accent: #7cb7ff;
}

/* Or follow the system preference directly */
@media (prefers-color-scheme: dark) {
  .my-callout {
    border-color: #444;
  }
}
```

Prefer the `html.dark` selector for anything the in-page toggle should control; `prefers-color-scheme` alone does not react to the toggle.

## Testing Dark Mode

### Using Browser DevTools

1. Open your documentation in a browser
2. Open DevTools (F12 or Cmd+Shift+I)
3. Press Cmd+Shift+P (or Ctrl+Shift+P on Linux/Windows)
4. Search for "Emulate CSS media feature prefers-color-scheme"
5. Select "prefers-color-scheme: dark" or "prefers-color-scheme: light"

### Using mdPress Serve

The development server produces the same site output, including the switcher:

```bash
mdpress serve
```

Use the theme switcher in the navigation bar to test both modes.

## Accessibility Considerations

Dark mode improves accessibility for users with light sensitivity and reduces eye strain. Best practices when customizing:

1. **Contrast Ratios**: Ensure at least 4.5:1 contrast for body text in both modes
2. **Color Not Alone**: Don't rely solely on color to convey meaning
3. **Reduced Motion**: Respect `prefers-reduced-motion` in transitions
4. **Test Both Modes**: Test your documentation in both light and dark modes

See [Built-in Themes](./builtin-themes.md) and [Custom CSS](./custom-css.md) for more customization options.
