# Writing Content for mdPress

mdPress uses standard Markdown for content authoring, with support for GitHub Flavored Markdown (GFM) and additional extensions. This guide covers the syntax and features available for writing your documentation.

## Basic Markdown Syntax

Start with the fundamentals that every mdPress document supports.

### Headings

Use hash symbols to create headings from H1 to H6:

```markdown
# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6
```

Each heading automatically generates an ID for cross-references. The ID is derived from the heading text converted to lowercase with hyphens replacing spaces.

### Paragraphs and Line Breaks

Separate paragraphs with blank lines. A single line break within a paragraph does not create a new paragraph.

```markdown
This is the first paragraph.

This is the second paragraph.
```

For a line break without starting a new paragraph, end a line with two spaces or a backslash:

```markdown
Line one
Line two

Or use backslash:
Line one\
Line two
```

### Emphasis and Strong Text

```markdown
*italic* or _italic_
**bold** or __bold__
***bold italic*** or ___bold italic___
~~strikethrough~~
```

### Lists

#### Unordered Lists

```markdown
- Item 1
- Item 2
  - Nested item 2a
  - Nested item 2b
- Item 3
```

You can use `-`, `*`, or `+` interchangeably.

#### Ordered Lists

```markdown
1. First item
2. Second item
   1. Nested item 2a
   2. Nested item 2b
3. Third item
```

### Blockquotes

```markdown
> This is a blockquote.
> It can span multiple lines.
>
> And contain multiple paragraphs.
```

Nested blockquotes are supported:

```markdown
> Level 1 blockquote
>
> > Level 2 blockquote
```

### Links

Create links using the following syntax:

```markdown
[Link text](https://example.com)
[Link with title](https://example.com "Title")
```

Reference-style links:

```markdown
[Link text][ref]

[ref]: https://example.com
```

Autolinks (automatically converted in mdPress):

```markdown
https://example.com
user@example.com
```

### Code

Inline code uses backticks:

```markdown
Use the `const` keyword to declare constants.
```

Code blocks use triple backticks with optional language specification:

````markdown
```javascript
function hello() {
  console.log("Hello, world!");
}
```
````

## GitHub Flavored Markdown (GFM)

mdPress includes full GFM support for modern documentation needs.

### Tables

Create tables using pipes and hyphens:

```markdown
| Header 1 | Header 2 | Header 3 |
|----------|----------|----------|
| Cell 1   | Cell 2   | Cell 3   |
| Cell 4   | Cell 5   | Cell 6   |
```

Alignment is specified with colons:

```markdown
| Left | Center | Right |
|:-----|:------:|------:|
| L1   |   C1   |    R1 |
| L2   |   C2   |    R2 |
```

### Task Lists

Task lists are useful for documentation about checklist features:

```markdown
- [x] Completed task
- [ ] Incomplete task
- [x] Another completed task
```

### Strikethrough

Already mentioned in emphasis section, but critical for GFM:

```markdown
~~This text is struck through~~
```

## Code Blocks with Syntax Highlighting

mdPress supports syntax highlighting for over 100 programming languages using industry-standard themes.

### Supported Languages

Common languages include: bash, c, cpp, csharp, css, dart, elixir, elm, go, groovy, haskell, java, javascript, kotlin, lisp, lua, objective-c, perl, php, python, ruby, rust, scala, shell, sql, swift, typescript, xml, yaml, and many more.

### Syntax Highlighting Example

````markdown
```python
def fibonacci(n):
    """Calculate the nth Fibonacci number."""
    if n <= 1:
        return n
    return fibonacci(n - 1) + fibonacci(n - 2)

result = fibonacci(10)
print(f"Result: {result}")
```
````

### Line Highlighting

Highlight specific lines in your code blocks:

````markdown
```javascript {3,5-7}
function process(data) {
  const trimmed = data.trim();
  const processed = transform(trimmed);  // Line 3

  validate(processed);  // Line 5
  store(processed);     // Line 6
  return processed;     // Line 7
}
```
````

### Themes

mdPress includes multiple syntax highlighting themes. Light and dark variants are automatically applied based on user preference. Examples include: GitHub Light, GitHub Dark, Dracula, Nord, Solarized Light/Dark, and others.

## Images

Include images in your documentation with alt text for accessibility:

```markdown
![Alt text for image](./images/screenshot.png)
```

### Lazy Loading

Images are automatically lazy-loaded in mdPress. The renderer defers image loading until the image is about to enter the viewport, improving page load performance.

### Image URLs

Both relative and absolute URLs work:

```markdown
![Local image](./assets/diagram.svg)
![External image](https://example.com/image.png)
```

## Blockquotes for Notes and Warnings

Use blockquotes to highlight important information:

```markdown
> **Note:** This is an important note for users.

> **Warning:** Be careful with this configuration.

> **Tip:** Here's a helpful suggestion.
```

## Internal Links Between Chapters

Link to other chapters in your documentation:

```markdown
[See the configuration guide](./configuration.md)
[Jump to advanced usage](./advanced-usage.md#custom-plugins)
```

When building, mdPress automatically resolves these links based on your book structure. Relative paths work across your documentation hierarchy.

### Fragment Links

Link to specific sections using heading IDs:

```markdown
[See the installation section](#installation)
[Jump to advanced config](./configuration.md#advanced-configuration)
```

## Heading IDs for Cross-References

Every heading automatically gets a unique ID. The ID is generated by:

1. Converting the heading text to lowercase
2. Replacing spaces with hyphens
3. Removing special characters

### Example Heading IDs

```markdown
# Getting Started
```
becomes `#getting-started`

```markdown
### Advanced Configuration (v2.0)
```
becomes `#advanced-configuration-v20`

You can reference these IDs from anywhere:

```markdown
[See the getting started section](#getting-started)
```

Custom IDs can be specified in some mdPress configurations, but the auto-generated IDs work for all standard use cases.

## Best Practices

### Structure Your Content

- Use H1 for the chapter title (only one per file)
- Use H2 for major sections
- Use H3 and below for subsections
- Keep heading hierarchy logical (don't jump from H2 to H4)

### Write Clear Code Examples

- Always include the language identifier in code blocks
- Use realistic, complete examples
- Add comments to explain non-obvious code
- Show both the code and the expected output when helpful

### Use Emphasis Appropriately

- Use **bold** for UI elements and key terms
- Use *italic* for emphasis
- Use `code` for technical terms, file names, and commands
- Avoid combining multiple emphasis styles

### Create Helpful Links

- Use descriptive link text that indicates where the link goes
- Link to related sections within your documentation
- Include both adjacent chapters and cross-references to relevant content
- Use fragment links to point to specific subsections

### Make Tables Readable

- Keep tables concise with focused content
- Use alignment to improve readability
- Consider breaking large tables into multiple smaller ones
- Provide explanatory text before tables

## Troubleshooting

### Links Not Working

Verify that relative paths use `./` prefix and correct file names. mdPress resolves paths relative to the current file location.

### Code Highlighting Not Applied

Ensure you specify the language identifier after the opening backticks. Without it, the code block displays as plain text.

### Images Not Loading

Check that image paths are correct and relative to your content file. Image paths use the same resolution rules as document links.
