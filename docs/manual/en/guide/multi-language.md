# Multi-Language Books

A multi-language project is a directory containing a `LANGS.md` file plus one subdirectory per language. **Each language directory is an ordinary, self-contained mdPress project.** mdPress builds each one in turn and writes a small landing page next to them.

There is no `multiLanguage`, `languages`, `defaultLanguage`, or `languageSwitcher` configuration. `LANGS.md` is the entire mechanism.

## Directory Layout

```
book/
├── LANGS.md          <- the only thing that makes this a multi-language project
├── en/
│   ├── book.yaml
│   ├── README.md
│   └── guide.md
└── zh/
    ├── book.yaml
    ├── README.md
    └── guide.md
```

Do **not** put a `book.yaml` at the root. mdPress loads the root config before it looks at `LANGS.md`, and a root `book.yaml` with no chapters of its own fails the build:

```
Error: failed to load config: config validation failed: at least one chapter is required
```

## LANGS.md

```markdown
# Languages

- [English](en/)
- [中文](zh/)
```

Parsing rules, exactly as implemented:

- Any line containing a Markdown link `[Name](dir)` becomes a language. Lines starting with `#` and blank lines are skipped, so the heading is decoration.
- The link text is the display name shown in the language switcher.
- **The link target must be the language directory.** A trailing slash is optional. Absolute paths and `..` are rejected.
- Point an entry at a file (`[English](en/README.md)`) and the build fails with `stat …/en/README.md/<Title>_site: not a directory`.
- Bullet style does not matter; `-` and `*` both work.

## Per-Language Configuration

Each language directory is discovered with the same rules as a standalone project: `book.yaml`, then `book.json`, then `SUMMARY.md`, then zero-config discovery of the `.md` files present.

A language `book.yaml` uses the normal mdPress schema — a `book:` block, not top-level `title:`/`language:` keys:

`en/book.yaml`:

```yaml
book:
  title: "My Docs"
  author: "Your Name"
  description: "Complete guide to my project"
  language: "en-US"

chapters:
  - title: "Intro"
    file: "README.md"
  - title: "Guide"
    file: "guide.md"
```

`zh/book.yaml`:

```yaml
book:
  title: "我的文档"
  author: "你的名字"
  description: "我的项目的完整指南"
  language: "zh-CN"

chapters:
  - title: "简介"
    file: "README.md"
  - title: "指南"
    file: "guide.md"
```

Writing `title:` at the top level instead of under `book:` produces an "unknown key" warning and a book called "Untitled Book".

`book.language` takes a full locale tag such as `en-US` or `zh-CN`; it drives the UI strings and cover. If you omit it, mdPress guesses from the directory name (`en/` → English, `zh/` → Chinese).

Everything else — `style:`, `output:`, `plugins:` — is per language and independent. Two languages can use different themes.

## Building

```bash
cd book
mdpress build --format site
```

mdPress builds each language in turn. The outputs land **inside each language directory**:

```
book/
├── _mdpress_langs.html          <- landing page listing the languages
├── en/
│   └── My Docs_site/            <- named after the book title, or output.filename
│       └── index.html
└── zh/
    └── 我的文档_site/
        └── index.html
```

File formats behave the same way — `mdpress build --format pdf` writes `en/My Docs.pdf` and `zh/我的文档.pdf`. Set `output.filename` in a language's `book.yaml` to control that name:

```yaml
output:
  filename: "en-docs"     # -> en/en-docs.pdf and en/en-docs_site/
```

Each generated site page also gets a language-switcher bar injected at the top, linking to the other languages and to the landing page.

### Known limitations of the whole-project build

The whole-project build is convenient for local preview but is **not** a good source for deployment yet:

- The landing page is called `_mdpress_langs.html`, not `index.html`, so it is not what a web server serves at the site root.
- Site directory names come from the book **title**, so a title with spaces or CJK characters becomes part of the URL path (`en/My%20Docs_site/`).
- Links in the landing page and switcher are not percent-encoded, so those same titles produce links that do not resolve.
- `--output ./dist` does **not** create `dist/`. It produces sibling paths `dist-en_site/`, `dist-zh_site/` and `dist-index.html` next to the project, and each language's own `output.filename` is ignored.

Use the per-language builds below to produce something deployable.

### Building One Language (recommended for deployment)

There is no `--lang` flag. Build each language directory as its own project and choose the output path yourself. **Give `--output` a trailing slash** so mdPress treats it as a directory rather than a filename base:

```bash
mdpress build ./en --format site --output ./dist/en/
mdpress build ./zh --format site --output ./dist/zh/
```

This produces exactly the layout you want to deploy:

```
dist/
├── en/
│   ├── index.html
│   ├── guide.html
│   └── search-index.json
└── zh/
    ├── index.html
    ├── guide.html
    └── search-index.json
```

Without the trailing slash, `--output ./dist/en` is read as a filename base and the site lands in `dist/en_site/` instead.

This is also how you rebuild a single language without touching the others.

### Previewing One Language

`mdpress serve` has no multi-language mode. Point it at one language directory:

```bash
mdpress serve ./en
```

## Linking Between Languages

Cross-language links are ordinary relative links and are **not** rewritten by mdPress; a link from `en/guide.md` to `../zh/guide.md` points outside the English book's build graph, and `mdpress doctor` will report it:

```
⚠ Detected 1 Markdown link(s) outside the build graph
    - ../zh/guide.md (from guide.md)
```

For a link that works in the deployed site, write the deployed URL instead:

```markdown
[中文版](/zh/guide.html)
```

## Deployment

With the per-language build above, `dist/` is a normal static site tree:

```
https://docs.example.com/en/
https://docs.example.com/zh/
```

Add your own `dist/index.html` to redirect the root to a default language — mdPress does not generate a usable one yet.

Each language can also be deployed separately, to its own domain or bucket, since each `dist/<lang>/` is self-contained.

## Troubleshooting

| Symptom | Cause |
| --- | --- |
| `at least one chapter is required` at the root | There is a `book.yaml` at the project root. Delete it; per-language config lives in the language directories. |
| `stat …/en/README.md/…: not a directory` | A `LANGS.md` entry points at a file. Point it at the directory: `[English](en/)`. |
| `no language definitions found in LANGS.md` | No line in `LANGS.md` contains a Markdown link. |
| Book is titled "Untitled Book" | The language `book.yaml` uses top-level `title:` instead of `book: { title: … }`. |
| Site UI is in the wrong language | Set `book.language` (`en-US`, `zh-CN`, …) in that language's `book.yaml`. |
| `dist/` does not exist after `build --output ./dist` | Expected in whole-project mode. Build each language separately as shown above. |
