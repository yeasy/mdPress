# Performance Tips

mdPress is designed for speed, with built-in optimizations for large books. This guide covers performance tuning and best practices to maximize build speed and minimize resource usage.

## Understanding Build Performance

mdPress builds consist of three main phases:

1. **Parsing**: Converts Markdown to internal representation (parallelized across chapters)
2. **Processing**: Applies transformations and rendering (caching enabled)
3. **Output**: Generates PDF, HTML, or other formats (format-specific optimizations)

For large books (50+ chapters), parsing is typically the bottleneck. mdPress automatically parallelizes this phase using all available CPU cores.

## Caching Strategy

### How Build Cache Works

mdPress maintains a `.mdpress-cache/` directory (default location) that stores:

- **Chapter hashes**: MD5 checksums of chapter content
- **Compiled content**: Pre-processed Markdown for unchanged chapters
- **Metadata**: Build state and timestamps

On rebuild:

```
First build:    Parse all chapters → Cache all → Output formats
Second build:   Check hashes → Reuse cached chapters → Output formats
Changed files:  Re-parse changed chapters → Update cache → Output
```

This reduces rebuild time from minutes to seconds for large books with few changes.

### Enable Caching (Default Behavior)

Caching is enabled by default. No configuration needed:

```bash
# First build: compiles all chapters
mdpress build --format pdf
# Output: takes 2 minutes for 50 chapters

# Second build: reuses cache
mdpress build --format pdf
# Output: takes 5 seconds (no changes)
```

### View Cache Directory

```bash
# Default cache location
ls -la .mdpress-cache/

# Change cache directory
mdpress build --cache-dir /tmp/mdpress-cache --format pdf
```

The cache directory is safe to delete—it will be regenerated on the next build.

### Force Full Rebuild

When you need to rebuild everything (e.g., theme or configuration changes):

```bash
# Skip cache entirely
mdpress build --no-cache --format pdf

# Use case: After changing book.yaml style settings
mdpress build --no-cache --format pdf --output my-book.pdf
```

**When to use `--no-cache`**:
- After changing `style` settings in `book.yaml`
- After updating themes or custom CSS
- If builds appear stale or incorrect
- To ensure reproducible builds in CI/CD

**When NOT to use `--no-cache`**:
- During normal development (cache speeds up iteration)
- For single-chapter builds (caching overhead negligible)
- On every build in CI/CD (use strategically)

## Incremental Builds During Development

Use `mdpress serve` for the fastest feedback loop:

```bash
# Start live preview server
mdpress serve

# Edit chapters and save
# → Server automatically rebuilds affected chapters
# → Preview updates in browser (no manual rebuild)
```

Benefits:
- Only modified chapters are recompiled
- Cache is preserved between changes
- Live reload in browser provides instant feedback
- No CLI invocation needed after changes

Example workflow:

```bash
# Terminal 1: Start server
$ mdpress serve
[INFO] Serving at http://localhost:9000
[INFO] Watching for changes...

# Terminal 2: Edit chapter
$ vim chapters/ch03.md
# ... make changes, save

# Server automatically rebuilds (2 seconds)
# Browser shows updated content
```

## Parallel Chapter Processing

### Automatic Parallelization

mdPress automatically detects your system's CPU core count and processes chapters in parallel:

```bash
# On a 4-core system: processes 4 chapters simultaneously
# On a 16-core system: processes 16 chapters simultaneously
mdpress build --format pdf

# No configuration needed—scales automatically
```

Example performance scaling on different systems:

| System | Cores | 50 Chapters | Time |
|--------|-------|------------|------|
| Laptop | 4 | First build | 90 seconds |
| Laptop | 4 | Cached rebuild | 3 seconds |
| Server | 16 | First build | 25 seconds |
| Server | 16 | Cached rebuild | 2 seconds |

### Control Parallelism

Currently, mdPress doesn't expose parallelism configuration—it always uses all available cores. To limit parallelism, run mdPress in a container with limited CPU resources:

```bash
# Docker: limit to 2 cores
docker run --cpus=2 myimage mdpress build

# cgroups v2 (Linux): limit to 2 cores
systemd-run --scope -p CPUQuota=200% mdpress build
```

## Image Optimization

### Automatic Optimizations

mdPress automatically optimizes images for output:

- **HTML/site output**: Images are embedded with lazy loading (reduces initial page load)
- **PDF output**: Images are embedded at optimal resolution for screen or print
- **ePub output**: Images are embedded with device-optimized sizing

### Manual Image Optimization

Optimize images before adding to your book:

```bash
# PNG optimization (lossless)
optipng -o2 diagram.png

# JPEG optimization
jpegoptim --max=85 screenshot.jpg

# SVG optimization (preferred for diagrams)
# SVG scales infinitely and has small file size
# Use for flowcharts, architecture diagrams, icons
```

Recommended image sizes:

| Type | Format | Max Size | Use Case |
|------|--------|----------|----------|
| Diagrams | SVG | 50 KB | Flowcharts, architectures, system diagrams |
| Screenshots | PNG | 500 KB | User interface demonstrations |
| Photos | JPEG | 300 KB | Documentation photos, covers |
| Icons | SVG | 10 KB | Inline icons, callouts |

### Image Storage Best Practices

```
assets/
├── diagrams/
│   ├── architecture.svg      (prefer SVG)
│   ├── flow-chart.svg
│   └── deployment.svg
├── screenshots/
│   ├── interface-main.png    (PNG, under 500 KB)
│   └── setup-wizard.png
└── photos/
    └── team-photo.jpg        (JPEG, under 2 MB total)
```

Total asset directory should stay under 50 MB for reasonably fast builds.

## Output Format Performance

### PDF vs. HTML vs. ePub

**PDF output** (default, `--format pdf`):
- Requires Chrome/Chromium or Typst
- Slower (10-30 seconds for 50 chapters)
- Produces single portable file
- Best for printing and distribution

**HTML output** (`--format html`):
- Fastest (5-10 seconds for 50 chapters)
- Produces single HTML file
- Good for basic documentation
- No external dependencies

**Site output** (`--format site`):
- Medium speed (8-15 seconds for 50 chapters)
- Produces multi-page website
- Good for search engines and navigation
- Best for online documentation

**ePub output** (`--format epub`):
- Medium speed (8-15 seconds for 50 chapters)
- Produces portable e-book file
- Good for e-readers (Kindle, Apple Books)
- Requires formatting validation

### When to Use Each Format

For development and validation:

```bash
# Fastest feedback: use serve for live preview
mdpress serve

# Then switch to final format for final builds
mdpress build --format pdf       # For distribution
mdpress build --format site      # For online documentation
```

## Typst Backend for Lightweight PDF

The Typst backend provides an alternative to Chrome/Chromium that's faster and lighter:

```bash
# Install Typst first
# See https://typst.app for installation

# Build with Typst
mdpress build --format typst

# Output: produces native PDF
# Advantages: faster, no Chromium needed, professional output
```

Performance comparison:

| Backend | Speed | Size | Dependencies | Quality |
|---------|-------|------|--------------|---------|
| Chrome/Chromium | 30 seconds | 5 MB output | Browser engine | Professional |
| Typst | 15 seconds | 4 MB output | Typst CLI | Professional |

Use Typst for:
- Systems without Chrome installed
- Resource-constrained environments (CI/CD runners)
- Faster PDF generation
- Minimal dependencies

## Cache Directory Management

### Default Cache Location

```bash
# Linux/macOS (OS temp directory)
/tmp/mdpress-cache

# Windows
%TEMP%\mdpress-cache

# Override via environment variable
export MDPRESS_CACHE_DIR=/path/to/custom/cache
```

Override with `--cache-dir`:

```bash
mdpress build --cache-dir /tmp/mdpress-cache --format pdf
```

### Cache Size

The cache grows with book size. Typical sizes:

- 10 chapters: 2-5 MB
- 50 chapters: 10-20 MB
- 100 chapters: 20-40 MB

### Clean Cache

```bash
# Remove cache directory (will be rebuilt on next build)
rm -rf /tmp/mdpress-cache

# If using a custom cache directory
rm -rf $MDPRESS_CACHE_DIR

# mdPress will regenerate on next build
mdpress build --format pdf
```

It's safe to delete the cache at any time.

## CI/CD Performance

### GitHub Actions Example

```yaml
name: Build Book
on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      # Cache Go modules
      - uses: actions/cache@v3
        with:
          path: ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      # Cache mdPress cache directory
      - uses: actions/cache@v3
        with:
          path: .mdpress-cache
          key: ${{ runner.os }}-mdpress-${{ hashFiles('**/*.md') }}

      # Install mdPress
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go install github.com/yeasy/mdpress@latest

      # Build book
      - run: mdpress build --format pdf

      # Upload artifact
      - uses: actions/upload-artifact@v3
        with:
          name: pdf-output
          path: output.pdf
```

Key optimizations:
- Cache Go modules to avoid re-downloading dependencies
- Cache `.mdpress-cache/` to reuse parsed chapters
- Only rebuild on file changes (using `hashFiles`)

### Build Time Targets

Aim for these timelines in CI/CD:

- **Small book (1-10 chapters)**: < 30 seconds
- **Medium book (11-50 chapters)**: < 1 minute
- **Large book (50+ chapters)**: 1-3 minutes with caching

If builds exceed these targets:
1. Verify caching is working (check CI logs for cache hits)
2. Check image sizes (run `mdpress validate`)
3. Consider splitting very large books
4. Use `--format html` or `--format typst` for faster output

## Profiling and Diagnostics

### Verbose Output

Enable detailed timing information:

```bash
mdpress build --verbose --format pdf

# Output includes:
# [INFO] Parsing chapters...
# [DEBUG] Chapter 1: 150ms
# [DEBUG] Chapter 2: 120ms
# [DEBUG] Chapter 3: 180ms
# ...
# [INFO] Total parsing: 4.2s
```

### Identify Slow Chapters

Large chapters take longer to parse. If build is slow:

```bash
# Check for very large chapters
find . -name "*.md" -size +1M

# Split large chapters (see organizing-large-books.md)
wc -l *.md | sort -n
```

## Performance Checklist

- [ ] Use `mdpress serve` during development (instant feedback)
- [ ] Enable caching (default, provides 10-100x speedup on rebuilds)
- [ ] Use `--format html` for fastest output during development
- [ ] Switch to `--format pdf` only for final builds
- [ ] Optimize images (max 500 KB per screenshot, prefer SVG for diagrams)
- [ ] Keep total assets under 50 MB
- [ ] Use `--cache-dir` in CI/CD and cache between builds
- [ ] Run `mdpress validate` to catch broken links early
- [ ] For large books (100+ chapters), consider splitting into multiple books
- [ ] Review verbose output (`--verbose`) to identify bottlenecks

## Common Performance Issues

### Build Takes 5+ Minutes

**Likely cause**: Cache disabled or missing

```bash
# Solution: enable caching
mdpress build --format pdf
# First build takes time, rebuilds are fast
```

### Inconsistent Performance Between Builds

**Likely cause**: Cache directory in different location

```bash
# Solution: use consistent cache directory
mdpress build --cache-dir ~/.mdpress-cache --format pdf
```

### PDF Output Very Slow (30+ seconds)

**Likely cause**: Chrome rendering overhead

```bash
# Solution 1: Use Typst instead
mdpress build --format typst

# Solution 2: Use serve for development
mdpress serve
```

### Out of Memory on Large Books

**Likely cause**: Processing too many chapters in parallel

```bash
# Solution: use Docker with memory limit
docker run --memory=2g myimage mdpress build --format pdf
```

## Summary

The fastest workflow for documentation is:

1. **Development**: Use `mdpress serve` (fastest feedback)
2. **Validation**: Use `mdpress validate` (catch issues early)
3. **Final build**: Use `mdpress build --format pdf` (with caching enabled)
4. **CI/CD**: Cache `.mdpress-cache/` between builds

With proper caching and format selection, mdPress builds scale efficiently from single-chapter documents to 200-page technical books.
