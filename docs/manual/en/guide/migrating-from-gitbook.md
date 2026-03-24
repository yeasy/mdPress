# Migrating from GitBook

mdPress provides a migration tool to help you convert documentation from GitBook or HonKit to the mdPress format. This guide explains the migration process and what gets converted.

## Overview

The mdPress migration command automatically converts:

- `book.json` configuration to `book.yaml`
- GitBook template syntax to standard Markdown and HTML
- SUMMARY.md structure (compatible in both systems)
- Assets and media files
- Plugins configuration to equivalent mdPress features

## Running the Migration

Use the `mdpress migrate` command to convert your GitBook documentation.

### Basic Migration

```bash
mdpress migrate ./gitbook-project
```

This reads your GitBook project and creates a new mdPress project in the current directory with converted files.

### Output Directory

Specify where to create the migrated project:

```bash
mdpress migrate ./gitbook-project --output ./mdpress-project
```

Creates the mdPress project in `./mdpress-project` with all converted content.

### Dry Run Mode

Preview the migration without making changes:

```bash
mdpress migrate ./gitbook-project --dry-run
```

Shows what would be converted, what might lose information, and what requires manual attention. No files are created in dry-run mode.

### Verbose Output

Get detailed information about each step of the migration:

```bash
mdpress migrate ./gitbook-project --verbose
```

Shows which files are being converted, what transformations are applied, and any warnings about potential issues.

### Complete Migration Examples

Typical migration commands:

```bash
# Preview the migration first
mdpress migrate ./my-gitbook --dry-run

# Then perform the actual migration
mdpress migrate ./my-gitbook --output ./my-mdpress

# Verbose output to see what's happening
mdpress migrate ./my-gitbook --output ./my-mdpress --verbose
```

## What Gets Converted

The migration process handles several key conversion tasks.

### Configuration: book.json to book.yaml

GitBook `book.json`:

```json
{
  "title": "My Documentation",
  "description": "Complete guide",
  "author": "John Doe",
  "output": "docs",
  "plugins": [
    "search",
    "highlight",
    "mathjax"
  ],
  "pluginsConfig": {
    "mathjax": {
      "delimiters": ["$$", "$$"]
    }
  }
}
```

Converts to mdPress `book.yaml`:

```yaml
title: "My Documentation"
description: "Complete guide"
authors:
  - "John Doe"

# Plugins converted to mdPress features
search:
  enabled: true

markdown:
  highlight: true
  math: true

# Additional mdPress configuration
site:
  baseUrl: "/docs/"
```

### Template Tags to Markdown

GitBook template syntax like `{% hint %}` blocks are converted to standard Markdown blockquotes.

#### Hint Block Conversion

GitBook `{% hint %} ... {% endhint %}`:

```markdown
{% hint style="warning" %}
This is a warning message.
{% endhint %}
```

Converts to mdPress blockquote:

```markdown
> **Warning:** This is a warning message.
```

#### Info Block Conversion

GitBook info block:

```markdown
{% hint style="info" %}
This is important information.
{% endhint %}
```

Converts to:

```markdown
> **Info:** This is important information.
```

#### Success Block Conversion

GitBook success block:

```markdown
{% hint style="success" %}
Operation completed successfully.
{% endhint %}
```

Converts to:

```markdown
> **Success:** Operation completed successfully.
```

#### Danger Block Conversion

GitBook danger block:

```markdown
{% hint style="danger" %}
Dangerous operation ahead.
{% endhint %}
```

Converts to:

```markdown
> **Danger:** Dangerous operation ahead.
```

### Code Block Conversion

GitBook `{% code %}` blocks convert to standard fenced code blocks.

#### Code Block Syntax

GitBook syntax:

```
{% code title="example.js" %}
function hello() {
  console.log("Hello, world!");
}
{% endcode %}
```

Converts to mdPress:

````markdown
```javascript
// File: example.js
function hello() {
  console.log("Hello, world!");
}
```
````

#### Code Block with Language

GitBook:

```
{% code language="python" %}
def hello():
    print("Hello, world!")
{% endcode %}
```

Converts to:

````markdown
```python
def hello():
    print("Hello, world!")
```
````

### Tab Block Conversion

GitBook tabs:

```
{% tabs %}
{% tab title="JavaScript" %}
console.log("Hello");
{% endtab %}
{% tab title="Python" %}
print("Hello")
{% endtab %}
{% endtabs %}
```

Converts to mdPress Markdown with headings:

```markdown
### JavaScript
console.log("Hello");

### Python
print("Hello")
```

## SUMMARY.md Compatibility

SUMMARY.md has compatible syntax between GitBook and mdPress, so minimal conversion is needed.

### SUMMARY.md Format

Both systems support the same basic format:

```markdown
# Summary

- [Introduction](README.md)
- [Getting Started](getting-started.md)
  - [Installation](getting-started/installation.md)
  - [Configuration](getting-started/configuration.md)
- [Advanced](advanced/README.md)
  - [API Reference](advanced/api.md)
  - [Plugins](advanced/plugins.md)
- [FAQ](faq.md)
```

This structure works identically in both GitBook and mdPress.

### SUMMARY.md Links

Both systems use relative file paths:

```markdown
- [Chapter Title](path/to/file.md)
```

mdPress automatically resolves these links, so no conversion is necessary.

## Asset File Handling

Images, stylesheets, and other assets are copied to your mdPress project.

### Asset Directory

GitBook assets are copied to the mdPress assets directory:

```
gitbook-project/
├── book.json
└── assets/
    ├── images/
    └── styles/

Becomes:

mdpress-project/
├── book.yaml
└── assets/
    ├── images/
    └── styles/
```

### Image References

Image references are automatically updated during migration:

GitBook:
```markdown
![Screenshot](assets/screenshot.png)
```

mdPress (unchanged, still works):
```markdown
![Screenshot](./assets/screenshot.png)
```

## Manual Adjustments

Some features require manual review or adjustment after migration.

### Unsupported GitBook Features

These features don't have direct equivalents in mdPress:

- Advanced theme customization
- Some GitBook-specific plugins
- Custom JavaScript plugins
- Complex layout configurations

For these, the migration tool creates comments indicating manual review needed:

```markdown
<!-- TODO: Manual review needed - GitBook feature not auto-converted -->
```

### Review Plugin Conversions

Check that plugins were converted correctly:

```yaml
# Original plugins configuration
plugins:
  - mathjax
  - mermaid
  - highlight

# May require manual verification in book.yaml
markdown:
  math: true
  diagrams: true
  highlight: true
```

### Test Links After Migration

After migration, verify that:

1. All internal links work correctly
2. Cross-chapter links resolve properly
3. Navigation structure is correct
4. Asset paths are correct

Use the live preview server:

```bash
mdpress serve
```

Navigate through the entire documentation and check for broken links.

## Migration Report

The migration tool generates a report of what was converted.

### Report Content

The migration report includes:

- Files converted
- Conversions applied to each file
- Warnings about potential issues
- Features that couldn't be auto-converted
- Summary statistics

### Viewing the Report

After migration, check the generated `MIGRATION_REPORT.md`:

```bash
cat MIGRATION_REPORT.md
```

The report shows:

```markdown
# Migration Report

## Summary
- Files processed: 42
- Lines converted: 1,200+
- Template blocks converted: 15
- Warnings: 3

## Converted Files
- README.md: 1 hint block converted
- getting-started.md: 2 code blocks converted
- ...

## Warnings
- Line 45 in api.md: Complex CSS not converted
- Line 120 in plugins.md: GitBook plugin syntax detected
- ...
```

### Dry Run Report

The dry run shows what would be converted:

```bash
mdpress migrate ./gitbook-project --dry-run --verbose
```

Output indicates:
- Number of files that would be processed
- Types of conversions that would be applied
- Estimated success rate
- Any potential data loss risks

## Common Migration Issues

### Template Syntax Not Found

If your GitBook uses custom templates that aren't recognized:

1. Check the migration report
2. Manually convert using the patterns shown in this guide
3. Test in the live preview

### Complex CSS Not Converted

Custom CSS in GitBook won't be automatically converted. Options:

1. Use mdPress CSS customization
2. Simplify the CSS to work with standard Markdown
3. Re-create styles using mdPress theme configuration

### Plugin Features Lost

GitBook plugins that add functionality may not convert:

1. Check if mdPress has equivalent features
2. Use Markdown extensions instead (KaTeX, Mermaid, etc.)
3. Manually recreate functionality if critical

### Links Pointing to External GitBook

If your documentation links to other GitBook projects:

```markdown
[See other docs](https://docs.example.gitbook.io)
```

These links remain unchanged but now point to external resources. Consider:

1. Consolidating documentation into one project
2. Using cross-references within mdPress instead
3. Keeping external links if appropriate

## Post-Migration Checklist

After running the migration, verify:

- [ ] All files migrated successfully
- [ ] `book.yaml` configuration is complete
- [ ] SUMMARY.md structure is correct
- [ ] Run `mdpress serve` and preview the documentation
- [ ] Click through major chapters and sections
- [ ] Verify all images display correctly
- [ ] Test internal links
- [ ] Check code block syntax highlighting
- [ ] Verify math equations render (if used)
- [ ] Test search functionality
- [ ] Build to final output formats and verify

## Converting Specific Features

### Tabs to Headings

Convert GitBook tabs to mdPress:

Instead of tabs, use H3 headings for each option:

```markdown
## Installation

### Using npm
```bash
npm install my-package
```
```

### Using yarn
```bash
yarn add my-package
```
```

### Callout Blocks

Replace GitBook callouts with blockquotes:

GitBook:
```
{% hint style="danger" %}
Don't do this!
{% endhint %}
```

mdPress:
```markdown
> **Danger:** Don't do this!
```

### Custom Attributes

GitBook attributes like `{#custom-id}` are already supported in mdPress:

```markdown
# Chapter Title {#custom-id}
```

No conversion needed.

## When to Migrate vs Rewrite

Consider migrating if:
- You have extensive documentation
- Basic structure and content is good
- You want to quickly switch platforms

Consider rewriting if:
- Documentation needs restructuring
- You're significantly updating content
- Documentation is small (50 pages or less)
- You want to take advantage of mdPress-specific features from scratch

## Getting Help

If the migration doesn't produce expected results:

1. Review the MIGRATION_REPORT.md
2. Check specific sections mentioned in warnings
3. Test individual conversions in the live preview
4. Refer to the Writing Content guide for mdPress syntax
5. Manually adjust specific problematic sections

Migrations are typically successful, but complex GitBook projects may require minor adjustments.
