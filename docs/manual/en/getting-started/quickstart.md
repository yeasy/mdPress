# Quick Start

Create a sample project with:

```bash
mdpress quickstart my-book
cd my-book
mdpress build
mdpress serve --open
```

`quickstart` creates `book.yaml`, `README.md`, `preface.md`, `chapter01/README.md`, `chapter02/README.md`, `chapter03/README.md`, `images/README.md`, and `images/cover.svg`.

The generated config uses the project name as the PDF filename base and is ready for editing immediately.

`mdpress quickstart` defaults to `my-book`. It refuses to write into a non-empty directory.

## Next Steps

- Edit `book.yaml` for title, author, theme, and output settings.
- Replace the sample Markdown files with your own content.
- Run `mdpress validate` before building if you want a quick sanity check.
