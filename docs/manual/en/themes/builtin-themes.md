# Built-in Themes

mdPress comes with three professionally designed built-in themes that cover common documentation use cases. Each theme has distinct visual characteristics and is optimized for specific content types.

## Available Themes

### Technical Theme

The Technical theme is the default. It is designed for API documentation, technical guides, and code-heavy content.

**Characteristics:**
- Navy ink palette: `#12344D` headings, `#1C5A9E` links and accent
- Clean, sans-serif typography optimized for readability
- Code blocks with `github` syntax highlighting on a subtle tinted background
- Hairline table borders with a tinted header row and zebra striping
- Deep-navy full-bleed default cover

**Best for:**
- API reference documentation
- Programming tutorials
- Technical specifications
- Command-line tool documentation

```yaml
style:
  theme: technical
```

### Elegant Theme

The Elegant theme is designed for narrative documentation, books, and formal publications.

**Characteristics:**
- Warm serif palette: `#3E2723` body text on a warm `#FFFBF0` background, bronze `#A87B3B` accent
- Beautiful serif font stack for body text
- Warm hairline borders and generous line height
- `github` code highlighting on a warm-tinted background
- Deep warm-brown default cover

**Best for:**
- Books and longer-form documentation
- User manuals with narrative sections
- Academic or formal documentation
- Publications requiring elegant presentation

```yaml
style:
  theme: elegant
```

### Minimal Theme

The Minimal theme is a quiet monochrome design that prioritizes content clarity and print friendliness.

**Characteristics:**
- Near-black ink: `#000000` text and headings, `#1A1A1A` accent
- Grayscale `bw` code highlighting — no color in code blocks
- Minimal decorative elements, generous whitespace
- Print-optimized with no unnecessary colors
- Light default cover with dark ink

**Best for:**
- Print publications
- Accessibility-critical documentation
- Professional black-and-white output
- Government or formal documentation

```yaml
style:
  theme: minimal
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

Each theme ships its own code highlighting style (`github` for technical and elegant, `bw` for minimal). Leaving `style.code_theme` empty inherits the theme's style; set it explicitly to override:

```yaml
style:
  theme: minimal
  code_theme: github   # override minimal's grayscale code style
```

## Theme-Aware Covers

When `book.cover.image` and `book.cover.background` are not set, the default cover follows the theme: deep navy for `technical`, deep warm brown for `elegant`, and a light cover with dark ink for `minimal`. Setting `cover.background` to a light color (including `white` or named/`rgb()` colors) automatically switches the cover text to a dark ink.

## Custom Themes

You can define your own theme as a YAML file. There are two ways to use one:

1. **Project theme directory** — a file at `themes/<name>.yaml` (or `.yml`) in your project defines theme `<name>`, or overrides the built-in theme of the same name. With `themes/corporate.yaml` in place, select it in `book.yaml`:

   ```yaml
   style:
     theme: corporate
   ```

2. **Direct file path** — point `style.theme` at a YAML theme file (relative to `book.yaml`):

   ```yaml
   style:
     theme: mytheme.yaml
   ```

The theme file schema matches the built-in theme fields. The repository ships `themes/technical.yaml` as a complete example:

```yaml
name: mytheme
page_size: A4
font_family: "'Georgia', 'Times New Roman', serif"
font_size: 12
code_theme: github
line_height: 1.75
colors:
  text: "#1F2933"
  background: "#FFFFFF"
  heading: "#12344D"
  link: "#1C5A9E"
  code_bg: "#F5F7F9"
  code_text: "#1F2933"
  accent: "#1C5A9E"
  border: "#E4E7EB"
margins:
  top: 20.0
  bottom: 20.0
  left: 20.0
  right: 20.0
```

## Theme Management Commands

mdPress provides CLI commands to explore and preview themes. They derive their output from the live theme palettes, so what they print is what a build uses.

### List Available Themes

```bash
mdpress themes list
```

Lists each theme with its description, key colors (heading / link / accent / background), and page properties.

### Show Theme Details

```bash
mdpress themes show technical
```

Prints the theme's description, typography (font family, size, line height, code theme), full color palette, and a sample `book.yaml` snippet.

### Preview All Themes

```bash
mdpress themes preview
# Output: themes-preview.html
```

Generates a self-contained HTML file that showcases every built-in theme applied to sample content, rendered with the exact stylesheet the build pipeline uses. Use `-o, --output <path>` to write it elsewhere.

## Customizing Built-in Themes

Beyond full custom themes, you can layer CSS overrides on top of any theme with `style.custom_css`, which points to a CSS file:

```yaml
style:
  theme: technical
  custom_css: styles/overrides.css
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
