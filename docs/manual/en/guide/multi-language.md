# Multi-Language Books

mdPress supports creating documentation in multiple languages with automatic language switching and separate builds for each language version. This guide explains how to set up and manage multi-language documentation.

## Multi-Language Structure

Multi-language books use a specific directory structure and configuration files to organize content by language.

### Directory Layout

Organize your multi-language documentation like this:

```
book/
├── LANGS.md
├── book.yaml
├── en/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── chapter-1.md
│   ├── chapter-2.md
│   └── assets/
├── zh/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── chapter-1.md
│   ├── chapter-2.md
│   └── assets/
└── ja/
    ├── book.yaml
    ├── SUMMARY.md
    ├── chapter-1.md
    ├── chapter-2.md
    └── assets/
```

Each language has its own directory with complete documentation content and configuration.

## LANGS.md Configuration

The `LANGS.md` file defines all available languages in your documentation.

### LANGS.md Format

Create `LANGS.md` in your root directory:

```markdown
# Languages

- [English](en/README.md)
- [中文](zh/README.md)
- [日本語](ja/README.md)
```

Or with more detailed format:

```markdown
# Languages

* [English - English Documentation](en/)
* [中文（简体）- Simplified Chinese](zh/)
* [繁體中文 - Traditional Chinese](zh-tw/)
* [日本語 - Japanese](ja/)
```

### Language Structure

The format is Markdown with:
- A main heading (H1) "Languages"
- A list of links to each language
- Link text shown to users
- Link target pointing to the language directory or README

## Per-Language Configuration

Each language needs its own `book.yaml` file with language-specific settings.

### Language-Specific book.yaml

In `en/book.yaml`:

```yaml
title: "My Project Documentation"
description: "Complete guide to my project"
language: "en"
authors:
  - "Your Name"

# Language-specific metadata
site:
  baseUrl: "https://docs.example.com/en/"

search:
  languages: ["en"]
```

In `zh/book.yaml`:

```yaml
title: "我的项目文档"
description: "我的项目的完整指南"
language: "zh"
authors:
  - "你的名字"

site:
  baseUrl: "https://docs.example.com/zh/"

search:
  languages: ["zh"]
```

In `ja/book.yaml`:

```yaml
title: "マイプロジェクトドキュメント"
description: "マイプロジェクトの完全ガイド"
language: "ja"
authors:
  - "あなたの名前"

site:
  baseUrl: "https://docs.example.com/ja/"

search:
  languages: ["ja"]
```

### Required Language Settings

Each language's `book.yaml` should have:

- `language`: ISO 639-1 language code (en, zh, ja, fr, de, etc.)
- `title`: Title in the target language
- `description`: Description in the target language
- `search.languages`: Match the language code

## Root book.yaml Configuration

The root `book.yaml` defines global settings for the entire multi-language project.

### Root Configuration

In your root `book.yaml`:

```yaml
title: "Multi-Language Documentation"
description: "Documentation in multiple languages"

# Multi-language configuration
multiLanguage: true
languages:
  - code: "en"
    name: "English"
    region: "US"
  - code: "zh"
    name: "中文"
    region: "CN"
  - code: "ja"
    name: "日本語"
    region: "JP"

# Default language for the root
defaultLanguage: "en"

# Language switcher configuration
languageSwitcher:
  enabled: true
  position: "top-right"
```

### Language Codes

Use standard ISO 639-1 language codes:

- `en`: English
- `zh`: Chinese (Simplified)
- `zh-tw`: Chinese (Traditional)
- `ja`: Japanese
- `ko`: Korean
- `fr`: French
- `de`: German
- `es`: Spanish
- `ru`: Russian
- `ar`: Arabic
- `pt`: Portuguese
- `it`: Italian

## Building Multi-Language Documentation

Build all languages or specific languages with command-line flags.

### Build All Languages

```bash
mdpress build
```

Builds all languages specified in `LANGS.md` and creates:

```
dist/
├── en/
│   ├── index.html
│   ├── chapter-1/
│   └── ...
├── zh/
│   ├── index.html
│   ├── chapter-1/
│   └── ...
└── ja/
    ├── index.html
    ├── chapter-1/
    └── ...
```

### Build Specific Languages

```bash
mdpress build --lang en,ja
```

Builds only English and Japanese, skipping Chinese.

### Build with Output Directory

```bash
mdpress build --output ./dist
```

Outputs all language versions to the `./dist` directory with the structure above.

## Per-Language Building

Build individual languages separately if needed.

### Building a Single Language

```bash
cd en
mdpress build
```

Or from the root:

```bash
mdpress build --lang en --output ./dist/en
```

This is useful for deploying individual language versions to separate servers or domains.

### Language-Specific Output

When building a single language, the structure is simpler:

```
dist/
├── index.html
├── chapter-1/
│   └── index.html
├── chapter-2/
│   └── index.html
└── assets/
```

### Incremental Builds

For large multi-language projects, rebuild only changed languages:

```bash
mdpress build --lang zh
```

Only rebuilds the Chinese version, leaving other languages untouched.

## Language Switching

Users can switch between languages in the documentation interface.

### Language Switcher UI

The language switcher appears in the site interface (position configurable in `book.yaml`):

- Typically located in the header or sidebar
- Shows all available languages with native names
- Clicking switches to the selected language
- Current language is highlighted

### Navigation After Language Switch

When switching languages:

1. User clicks the language switcher
2. Browser navigates to the same page in the new language
3. If the page doesn't exist in the new language, navigates to the homepage
4. Language preference is saved in browser local storage

### Deep Linking Between Languages

Link from one language to the corresponding page in another:

In English documentation:
```markdown
[日本語版](../ja/chapter-1.md)
```

In Japanese documentation:
```markdown
[English Version](../en/chapter-1.md)
```

mdPress automatically rewrites these links based on the language context.

## Managing Content Across Languages

### Synchronizing Updates

When you update English documentation, you need to update translations:

1. Update source content in `en/chapter-1.md`
2. Update the corresponding file in `zh/chapter-1.md`
3. Update the corresponding file in `ja/chapter-1.md`
4. Rebuild the documentation

### Maintaining Consistency

Use the same file names across all languages:

```
en/chapter-1.md
zh/chapter-1.md
ja/chapter-1.md
```

This makes it easy to identify corresponding sections.

### Partial Language Support

You can have incomplete language versions. For example:

- English: 100% complete
- Japanese: 80% complete
- Chinese: 50% complete

The language switcher still shows all languages. Users who click an unavailable language are directed to the homepage in that language.

### Translation Workflow

Recommended workflow for managing translations:

1. Create English documentation
2. Commit and publish English version
3. Create language directories with structure copied from English
4. Translate section by section
5. Commit translated sections
6. Rebuild to include all available languages

## Example Multi-Language Setup

### Complete Example

Here's a complete example with three languages:

Create the directory structure:

```bash
mkdir -p book/{en,zh,ja}
```

Create `book/LANGS.md`:

```markdown
# Languages

- [English](en/)
- [中文](zh/)
- [日本語](ja/)
```

Create `book/book.yaml`:

```yaml
title: "Multi-Language Docs"
multiLanguage: true
languages:
  - code: "en"
    name: "English"
  - code: "zh"
    name: "中文"
  - code: "ja"
    name: "日本語"
defaultLanguage: "en"
```

Create `book/en/book.yaml`:

```yaml
title: "Documentation"
language: "en"
```

Create `book/en/SUMMARY.md`:

```markdown
# Summary

- [Introduction](./README.md)
- [Getting Started](./getting-started.md)
- [Configuration](./configuration.md)
```

Create `book/en/README.md`:

```markdown
# Welcome

Welcome to the documentation.
```

Then copy the structure to `zh/` and `ja/` directories and translate the content.

Build all languages:

```bash
cd book
mdpress build --output ../dist
```

## Deployment

### Language-Based Routing

Deploy all languages to the same domain with path-based routing:

```
https://docs.example.com/en/
https://docs.example.com/zh/
https://docs.example.com/ja/
```

Configure your web server or CDN to route requests appropriately.

### Separate Domains

Deploy each language to a separate domain:

```
https://en.docs.example.com/
https://zh.docs.example.com/
https://ja.docs.example.com/
```

Update the `baseUrl` in each language's `book.yaml` accordingly.

### Redirects and Localization

For the root domain, redirect users to their preferred language:

```
https://docs.example.com/ → https://docs.example.com/en/ (for English users)
https://docs.example.com/ → https://docs.example.com/zh/ (for Chinese users)
```

Implement this using:
- Web server configuration
- Service worker redirect
- JavaScript redirect based on browser language

## Troubleshooting

### Language Not Building

Verify:
1. The language directory exists
2. `book.yaml` exists in the language directory
3. `SUMMARY.md` exists in the language directory
4. Language code in `book.yaml` matches `LANGS.md`

### Broken Links Between Languages

Check:
1. File names match across language directories
2. Relative paths in links account for the language prefix
3. All referenced files exist in the target language

### Language Switcher Not Appearing

Verify in `book.yaml`:
- `multiLanguage: true` is set
- `languageSwitcher.enabled: true` is set
- Multiple languages are listed in `languages`

### Inconsistent Content

When content diverges between language versions, document the differences:

```markdown
> Note: This feature is only available in English documentation.
> Please refer to the [English version](../en/chapter-1.md#feature-name).
```

Keep translation changes documented for reference.
