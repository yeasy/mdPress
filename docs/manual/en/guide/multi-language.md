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
- An entry may point at the directory (`[English](en/)`) or at a file inside it (`[English](en/README.md)`); a file resolves to its directory.
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

mdPress builds each language in turn into **one output tree**:

```
book/
└── _book/                   <- the whole deployable site
    ├── index.html           <- language switcher (the site root)
    ├── en/
    │   └── index.html
    └── zh/
        └── index.html
```

Pass `--output` to build somewhere else. It names the root of the whole tree, so `./dist` and `./dist/` are equivalent:

```bash
mdpress build --format site --output ./dist
# -> dist/index.html, dist/en/index.html, dist/zh/index.html
```

A path that looks like a file names the directory it would have lived in — `--output ./dist/book.html` builds into `dist/book/` — because a multi-language build produces a tree, not a single file.

File formats sit beside each language's site — `mdpress build --format pdf` writes `_book/en/<name>.pdf`. Set `output.filename` in a language's `book.yaml` to control that name:

```yaml
output:
  filename: "en-docs"     # -> _book/en/en-docs.pdf
```

Each generated site page also gets a language-switcher bar injected at the top, linking to the other languages and to the site root.

Directory names come from `LANGS.md`, not from the book title, so a title containing spaces or CJK characters never becomes part of a URL path. The switcher's links are percent-encoded.

A root `book.yaml` alongside `LANGS.md` is allowed and is the place for shared metadata; its chapters are not required, because each language directory has its own.

### Building one language on its own

There is no `--lang` flag, but a language directory is a complete project, so you can build one by pointing mdPress at it:

```bash
mdpress build ./en --format site --output ./dist/en
```

Use this when you deploy each language separately. For a single deployable tree covering every language, prefer the whole-project build above.

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

The output tree is a normal static site, ready to upload as-is:

```
https://docs.example.com/          <- language switcher
https://docs.example.com/en/
https://docs.example.com/zh/
```

The switcher at the root is a real `index.html`, so a web server serves it without extra configuration. Each `<lang>/` directory is also self-contained, so a language can be deployed on its own to a separate domain or bucket.

## Troubleshooting

| Symptom | Cause |
| --- | --- |
| `at least one chapter is required` at the root | A root `book.yaml` without `LANGS.md` beside it. With `LANGS.md` present a root `book.yaml` is allowed and holds shared metadata. |
| `no language definitions found in LANGS.md` | No line in `LANGS.md` contains a Markdown link. |
| Book is titled "Untitled Book" | The language `book.yaml` uses top-level `title:` instead of `book: { title: … }`. |
| Site UI is in the wrong language | Set `book.language` (`en-US`, `zh-CN`, …) in that language's `book.yaml`. |
| Output landed somewhere unexpected | `--output` names the root of the whole tree. `./dist` and `./dist/` build into `dist/`; a file-like path names its directory, so `./dist/book.html` builds into `dist/book/`. |
