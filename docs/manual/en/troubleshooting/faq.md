# Frequently Asked Questions

## Installation and Setup

### Can I use mdPress without Go installed?

**Q:** Do I need to install Go to use mdPress?

**A:** No. Pre-built binaries are available for Linux, macOS, and Windows. Download from:
- GitHub Releases: https://github.com/yeasy/mdpress/releases
- Homebrew (macOS): `brew install mdpress`
- Manually: Extract binary and add to PATH

If you want to build from source, you'll need Go 1.21+.

### Does mdPress need internet?

**Q:** Does mdPress require an internet connection to build books?

**A:** No. mdPress is fully offline. All processing happens locally on your machine. The only exception is:
- Building from GitHub repository URLs (requires network to clone)
- For local directories, no network is needed

To build offline:

```bash
# Clone repository once
git clone https://github.com/user/book-repo
cd book-repo

# Then build offline
mdpress build --format pdf
```

### Can I use mdPress on Docker?

**Q:** How do I use mdPress in a Docker container?

**A:** Use the official Docker image or create your own:

```dockerfile
FROM golang:1.21-alpine
RUN apk add --no-cache chromium font-noto-cjk
RUN go install github.com/yeasy/mdpress@latest
WORKDIR /workspace
ENTRYPOINT ["mdpress"]
```

Build and use:

```bash
docker build -t mdpress:latest .
docker run -v "$(pwd):/workspace" mdpress:latest build --format pdf
```

## Configuration and Styling

### Can I use custom fonts?

**Q:** Can I apply custom fonts to my PDF?

**A:** Yes, via custom CSS:

```yaml
style:
  custom_css: |
    @font-face {
      font-family: 'MyFont';
      src: url('assets/myfont.ttf') format('truetype');
    }
    body {
      font-family: 'MyFont', sans-serif;
    }
```

Place font files in `assets/` and reference in CSS. For PDF output, use TrueType (.ttf) or OpenType (.otf) fonts.

### How do I add a cover image?

**Q:** How do I create a book cover with an image?

**A:** Set the cover image in `book.yaml`:

```yaml
book:
  title: "My Book"
  cover:
    image: "assets/cover.png"
    background: "#ffffff"  # optional background color

output:
  cover: true
```

Image requirements:
- Format: PNG, JPEG, or PDF
- Recommended size: 1200×1600 pixels (3:4 aspect ratio)
- Maximum: 5 MB

The background color is used if image doesn't fully cover the page.

### How do I customize headers and footers?

**Q:** Can I add custom text to page headers and footers?

**A:** Yes, use template variables:

```yaml
style:
  header:
    left: "{{.Book.Title}}"
    center: "{{.Chapter.Title}}"
    right: "{{.PageNum}}"
  footer:
    left: "©{{.Year}} {{.Book.Author}}"
    center: ""
    right: "Page {{.PageNum}}"
```

Available variables:
- `{{.Book.Title}}` - Book title
- `{{.Book.Author}}` - Book author
- `{{.Book.Version}}` - Book version
- `{{.Chapter.Title}}` - Current chapter title
- `{{.PageNum}}` - Current page number
- `{{.Date}}` - Build date
- Custom variables via plugins

See [template-variables.md](../reference/template-variables.md) for complete reference.

### Can I use Markdown in headers/footers?

**Q:** Can I use Markdown formatting in header/footer text?

**A:** No, only plain text or template variables. For complex headers, use custom CSS:

```yaml
style:
  custom_css: |
    @page {
      @top-center {
        content: "Chapter " var(--chapter-num);
        font-size: 12pt;
        font-weight: bold;
      }
    }
```

## Output Formats

### What's the difference between "html" and "site" format?

**Q:** Should I use `--format html` or `--format site`?

**A:** They produce different outputs:

| Format | Output | Use Case |
|--------|--------|----------|
| `html` | Single HTML file | Simple reading, offline sharing |
| `site` | Multi-page website | Online hosting, search engines, navigation |

```bash
# Single file: good for email, download
mdpress build --format html
# Output: output.html (1-5 MB)

# Website: good for GitHub Pages, personal servers
mdpress build --format site
# Output: _book/ directory (navigable website)
```

Choose `site` if you'll host online. Choose `html` if you want a single shareable file.

### Can I deploy to GitHub Pages?

**Q:** How do I deploy my book to GitHub Pages?

**A:** Yes. Build with `--format site` and push to `gh-pages` branch:

```bash
# Build site
mdpress build --format site

# Create/switch to gh-pages branch
git checkout --orphan gh-pages

# Move _book to root
mv _book/* .

# Commit and push
git add -A
git commit -m "Deploy book"
git push origin gh-pages
```

Or use GitHub Actions (recommended):

```yaml
deploy:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: go install github.com/yeasy/mdpress@latest
    - run: mdpress build --format site
    - uses: actions/upload-pages-artifact@v2
      with:
        path: '_book/'
    - uses: actions/deploy-pages@v2
```

Site appears at: `https://username.github.io/repo-name/`

### What output formats are supported?

**Q:** What are all the output formats mdPress supports?

**A:**
- `pdf` - Portable PDF (default, requires Chrome or Typst)
- `html` - Single HTML file
- `site` - Multi-page website
- `epub` - E-book format (Kindle, Apple Books, etc.)
- `typst` - Typst source (for Typst typesetting system)

```bash
# All formats in one command
mdpress build --format pdf --format html --format epub

# Check supported formats
mdpress build --help
```

### Can I password-protect my PDF?

**Q:** Can I add password protection to PDF output?

**A:** Not directly in mdPress. Password protection is typically handled by:

1. **External tools** (after building):
   ```bash
   # Linux/macOS
   qpdf --encrypt user-password owner-password 128 -- output.pdf output-protected.pdf

   # Windows
   pdftk output.pdf output output-protected.pdf user_pw "password" owner_pw "ownerpassword"
   ```

2. **CI/CD pipeline**:
   ```yaml
   - run: mdpress build --format pdf
   - run: qpdf --encrypt mypassword mypassword 128 -- output.pdf secure.pdf
   ```

3. **Cloud services**: Upload PDF to cloud storage with access controls

## Content and Structure

### How do I organize chapters?

**Q:** How should I structure chapters in a large book?

**A:** Use nested sections in `book.yaml`:

```yaml
chapters:
  - title: "Part 1: Basics"
    file: "part1.md"
    sections:
      - title: "Chapter 1"
        file: "ch01.md"
      - title: "Chapter 2"
        file: "ch02.md"
```

Or use `SUMMARY.md`:

```markdown
# Summary
- [Part 1](part1.md)
  - [Chapter 1](ch01.md)
  - [Chapter 2](ch02.md)
```

See [organizing-large-books.md](../best-practices/organizing-large-books.md) for detailed strategies.

### Can I use Markdown includes?

**Q:** Can I include one Markdown file inside another?

**A:** Not natively. Work around using:

1. **Separate chapter files** (recommended):
   ```yaml
   chapters:
     - title: "Section A"
       file: "section-a.md"
     - title: "Section B"
       file: "section-b.md"
   ```

2. **Pre-processing script**:
   ```bash
   # Concatenate chapters before building
   cat chapter1.md chapter2.md > combined.md
   mdpress build
   ```

3. **Build tool integration** (Makefile):
   ```makefile
   build:
   	cat intro.md part1/*.md part2/*.md > full-book.md
   	mdpress build
   ```

### How do I add footnotes?

**Q:** How do I add footnotes or endnotes?

**A:** Use standard Markdown footnote syntax:

```markdown
This is a sentence with a footnote[^1].

[^1]: Here's the footnote content.
```

For PDF output, footnotes appear at the bottom of the page. For HTML, they become links to endnotes.

### Can I use LaTeX/math equations?

**Q:** Can I include mathematical equations?

**A:** Limited support. Use these formats:

**Inline math** (HTML only, with custom CSS):
```markdown
E = mc²
```

**Display equations** (as images):
```markdown
![Equation](assets/equation.png)
```

**LaTeX in HTML** (with MathJax plugin):
```html
<script src="https://polyfill.io/v3/polyfill.min.js?features=es6"></script>
<script id="MathJax-script" async src="https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js"></script>
```

For complex math, render to images and embed.

## Advanced Topics

### Can I use plugins?

**Q:** Can I extend mdPress with plugins?

**A:** Yes. Plugins run during the build:

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
```

Create a plugin (simple example):

```go
// plugins/my-plugin/main.go
package main

import (
  "fmt"
  "os"
)

func main() {
  // Read input
  input, _ := os.ReadFile("/dev/stdin")

  // Process
  output := processMarkdown(input)

  // Write output
  fmt.Println(output)
}
```

See plugin documentation for detailed API.

### Can I use custom themes?

**Q:** Can I create custom themes?

**A:** Yes. Themes are CSS files:

```bash
# List built-in themes
mdpress themes list

# Show theme details
mdpress themes show technical

# Create custom theme
mkdir -p themes/my-theme
cat > themes/my-theme/style.css << 'EOF'
body {
  font-family: Georgia, serif;
  color: #333;
}
EOF

# Use in book.yaml
style:
  theme: "./themes/my-theme"
```

### Can I integrate with version control?

**Q:** How do I track PDF versions in git?

**A:** Git LFS (Large File Storage) for binaries:

```bash
# Install Git LFS
brew install git-lfs

# Track PDFs
git lfs install
git lfs track "*.pdf"
git add .gitattributes

# Now commit PDFs normally
git add output.pdf
git commit -m "Release v1.0.0"
```

Or exclude from git:

```bash
# .gitignore
output.pdf
output.html
_book/
.mdpress-cache/
```

Build PDFs in CI/CD instead of committing.

### Can I build from GitHub repositories?

**Q:** Can I build a book from a GitHub repository?

**A:** Yes:

```bash
# Public repository
mdpress build https://github.com/user/book-repo

# Private repository (requires GITHUB_TOKEN)
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
mdpress build https://github.com/org/private-book
```

Requirements:
- Repository must contain `book.yaml` or `SUMMARY.md`
- For private repos, provide `GITHUB_TOKEN` with `contents:read` permission

## Troubleshooting

### Where do I find logs?

**Q:** How do I see detailed build logs?

**A:** Use verbose mode:

```bash
mdpress build --verbose --format pdf

# Output includes:
# [DEBUG] Loading config from book.yaml
# [DEBUG] Found 12 chapters
# [DEBUG] Parsing chapter 1... (125ms)
# [DEBUG] Parsing chapter 2... (98ms)
# ...
```

### How do I report bugs?

**Q:** Where should I report bugs or request features?

**A:** GitHub Issues:
- Issues: https://github.com/yeasy/mdpress/issues
- Discussions: https://github.com/yeasy/mdpress/discussions

Include:
- mdPress version: `mdpress --version`
- System info: `mdpress doctor`
- Minimal reproduction steps
- Error output with `--verbose` flag

### How do I get help?

**Q:** Where can I get help using mdPress?

**A:**
1. **Documentation**: https://github.com/yeasy/mdpress
2. **GitHub Issues**: https://github.com/yeasy/mdpress/issues
3. **GitHub Discussions**: https://github.com/yeasy/mdpress/discussions
4. **Run doctor**: `mdpress doctor` (diagnoses many issues)
5. **Validate**: `mdpress validate` (checks for configuration errors)

## Performance and Optimization

### How can I speed up builds?

**Q:** My builds are slow. How do I make them faster?

**A:** See [performance.md](../best-practices/performance.md) for detailed optimization strategies.

Quick checklist:
1. Enable caching (default): `mdpress build --format pdf`
2. Use HTML for development: `mdpress serve --format html`
3. Optimize images: keep under 500 KB each
4. Use Typst for faster PDF: `mdpress build --format typst`

Typical times:
- Small book (10 chapters): 30 seconds
- Large book (50 chapters): 1-2 minutes with caching

### Can I build multiple books from one repository?

**Q:** Can I maintain multiple books in one repository?

**A:** Yes. Create separate directories with their own `book.yaml`:

```
repo/
├── book1/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── ch01.md
│   └── ch02.md
├── book2/
│   ├── book.yaml
│   ├── SUMMARY.md
│   ├── ch01.md
│   └── ch02.md
```

Build each:

```bash
mdpress build book1/
mdpress build book2/
```

Or in CI/CD:

```yaml
strategy:
  matrix:
    book: [book1, book2]

steps:
  - run: mdpress build ${{ matrix.book }}
```

## Licensing and Distribution

### Can I use mdPress commercially?

**Q:** Can I use mdPress for commercial projects?

**A:** Yes. mdPress is open source under the MIT license, allowing commercial use.

### Can I sell books created with mdPress?

**Q:** Can I sell PDFs generated by mdPress?

**A:** Yes, but:
- The tool itself is free and open source
- You own the output you create
- You can sell the resulting books
- You must respect copyright of content you include

## More Questions?

If your question isn't answered here:

1. Check the full documentation
2. Search GitHub Issues
3. Ask in GitHub Discussions
4. Run `mdpress doctor` and `mdpress validate` for diagnostics

The mdPress community is friendly and helpful!
