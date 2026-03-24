# Built-in Themes

mdPress comes with three professionally designed built-in themes that cover common documentation use cases. Each theme has distinct visual characteristics and is optimized for specific content types.

## Available Themes

### Technical Theme

The Technical theme is designed for API documentation, technical guides, and code-heavy content.

**Characteristics:**
- Primary color: #1A5490 (professional blue)
- Clean, sans-serif typography optimized for readability
- Code blocks are the focal point with syntax highlighting
- Neutral gray backgrounds minimize distractions
- Wide margins for annotations and notes
- Monospace fonts for inline code and parameters

**Best for:**
- API reference documentation
- Programming tutorials
- Technical specifications
- Command-line tool documentation

**Visual example:**
```yaml
style:
  theme: technical
  code_theme: monokai
```

### Elegant Theme

The Elegant theme is designed for narrative documentation, books, and formal publications.

**Characteristics:**
- Primary color: #34495e (sophisticated dark blue-gray)
- Beautiful serif font for body text
- Elegant chapter dividers and decorative elements
- Generous whitespace and vertical rhythm
- Margin notes and sidebars for supplementary content
- Print-optimized layout with careful pagination

**Best for:**
- Books and longer-form documentation
- User manuals with narrative sections
- Academic or formal documentation
- Publications requiring elegant presentation

**Visual example:**
```yaml
style:
  theme: elegant
  typography:
    body_font: georgia
```

### Minimal Theme

The Minimal theme prioritizes content clarity and print friendliness.

**Characteristics:**
- Primary color: #000 (black)
- Pure black text on white background
- High contrast for maximum readability
- Minimal decorative elements
- Print-optimized with no unnecessary colors
- Responsive and accessible design
- Perfect for screen readers and accessibility tools

**Best for:**
- Print publications
- Accessibility-critical documentation
- Minimal bandwidth usage
- Professional black-and-white output
- Government or formal documentation

**Visual example:**
```yaml
style:
  theme: minimal
  colors:
    background: "#ffffff"
    text: "#000000"
```

## Setting a Theme

Specify your theme in the `book.yaml` configuration file under the `style` section:

```yaml
book:
  title: "My Documentation"
  author: "Your Name"

style:
  theme: technical
```

You can also override theme colors without changing the entire theme:

```yaml
style:
  theme: elegant
  colors:
    primary: "#2c3e50"
    accent: "#e74c3c"
```

## Theme Management Commands

mdPress provides CLI commands to explore and preview themes:

### List Available Themes

```bash
mdpress themes list
```

Output:
```
Available themes:
  • technical    - Professional blue theme for code-heavy docs
  • elegant      - Sophisticated serif theme for narrative content
  • minimal      - High-contrast black & white theme
```

### Show Theme Details

```bash
mdpress themes show technical
```

This displays comprehensive information about the theme:

```
Theme: technical
Description: Professional blue theme optimized for technical documentation

Colors:
  Primary:    #1A5490
  Secondary:  #16a085
  Accent:     #e74c3c
  Background: #ffffff
  Text:       #2c3e50

Typography:
  Body Font:  -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
  Code Font:  "Courier New", monospace
  Font Size:  16px
  Line Height: 1.6

Code Highlighting:
  Theme: monokai
  Line Numbers: enabled
```

### Preview a Theme

```bash
mdpress themes preview technical
```

This generates a sample document showing the theme in action, including:
- Typography samples
- Code blocks with syntax highlighting
- Callout boxes and admonitions
- Tables and lists
- Links and inline formatting

The preview opens in your default browser.

## Customizing Built-in Themes

While you can use built-in themes as-is, you can also customize them by adding CSS overrides:

```yaml
style:
  theme: technical
  custom_css: |
    :root {
      --primary-color: #0056b3;
      --font-size-base: 18px;
    }

    .content h1 {
      font-size: 2.5em;
      text-transform: uppercase;
    }
```

See the [Custom CSS guide](./custom-css.md) for detailed customization options.

## Theme Switching

To switch themes during development, update `book.yaml` and rebuild:

```bash
# Edit book.yaml
vim book.yaml

# Rebuild with new theme
mdpress build

# Or watch with live reload
mdpress serve
```

The live server will refresh your browser automatically when you change the theme.

## Theme Files Location

For advanced customization, theme files are stored in:

```
~/.mdpress/themes/
├── technical/
│   ├── style.css
│   ├── config.yaml
│   └── templates/
├── elegant/
│   ├── style.css
│   ├── config.yaml
│   └── templates/
└── minimal/
    ├── style.css
    ├── config.yaml
    └── templates/
```

You can copy and modify a theme for full customization. See the [Custom CSS guide](./custom-css.md) for creating your own theme.
