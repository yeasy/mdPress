# CI/CD Integration

Automate book builds using continuous integration and continuous deployment. This guide covers setup for GitHub Actions, GitLab CI, and general CI/CD best practices.

## GitHub Actions Workflow

### Basic PDF Build and Deploy to GitHub Pages

Create `.github/workflows/build.yml`:

```yaml
name: Build and Deploy Book

on:
  push:
    branches:
      - main
      - develop
  pull_request:
    branches:
      - main

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pages: write
      id-token: write

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'
          cache: 'go'

      - name: Install mdPress
        run: go install github.com/yeasy/mdpress@latest

      - name: Cache mdPress cache
        uses: actions/cache@v3
        with:
          path: .mdpress-cache
          key: ${{ runner.os }}-mdpress-${{ hashFiles('**/*.md', 'book.yaml') }}
          restore-keys: |
            ${{ runner.os }}-mdpress-

      - name: Validate config
        run: mdpress validate

      - name: Build PDF
        run: mdpress build --format pdf

      - name: Upload PDF artifact
        uses: actions/upload-artifact@v3
        with:
          name: book-pdf
          path: output.pdf
          retention-days: 30

      - name: Build HTML site
        run: mdpress build --format site

      - name: Deploy to GitHub Pages
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        uses: actions/upload-pages-artifact@v2
        with:
          path: '_book/'

      - name: Deploy Pages
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        id: deployment
        uses: actions/deploy-pages@v2
```

### Pull Request Preview

Build previews for pull requests to validate changes:

```yaml
name: PR Preview Build

on:
  pull_request:
    branches:
      - main

jobs:
  preview:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
          cache: 'go'

      - name: Install mdPress
        run: go install github.com/yeasy/mdpress@latest

      - name: Build HTML preview
        run: mdpress build --format html --output pr-preview.html

      - name: Comment on PR
        uses: actions/github-script@v6
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '📚 Book preview: [Download HTML](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})'
            })

      - name: Upload preview
        uses: actions/upload-artifact@v3
        with:
          name: pr-preview-${{ github.event.number }}
          path: pr-preview.html
```

### Matrix Build (Multiple Formats)

Build PDF, HTML, and ePub simultaneously:

```yaml
name: Multi-Format Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        format: [pdf, html, epub, site]

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
          cache: 'go'

      - run: go install github.com/yeasy/mdpress@latest

      - uses: actions/cache@v3
        with:
          path: .mdpress-cache
          key: ${{ runner.os }}-mdpress-${{ matrix.format }}-${{ hashFiles('**/*.md') }}

      - name: Build ${{ matrix.format }}
        run: mdpress build --format ${{ matrix.format }}

      - name: Upload ${{ matrix.format }} artifact
        uses: actions/upload-artifact@v3
        with:
          name: book-${{ matrix.format }}
          path: |
            output.*
            _book/
```

## GitLab CI Setup

### Basic Pipeline

Create `.gitlab-ci.yml`:

```yaml
stages:
  - validate
  - build
  - deploy

variables:
  CACHE_COMPRESSION_LEVEL: fastest

before_script:
  - apt-get update && apt-get install -y golang-go
  - go install github.com/yeasy/mdpress@latest

cache:
  key: "$CI_COMMIT_REF_SLUG-mdpress"
  paths:
    - .mdpress-cache/
    - .cache/go-build/

validate:
  stage: validate
  script:
    - mdpress validate
  only:
    - merge_requests
    - main

build-pdf:
  stage: build
  script:
    - mdpress build --format pdf
  artifacts:
    paths:
      - output.pdf
    expire_in: 30 days
  only:
    - main
    - develop

build-site:
  stage: build
  script:
    - mdpress build --format site
  artifacts:
    paths:
      - _book/
    expire_in: 7 days
  only:
    - main

pages:
  stage: deploy
  script:
    - mdpress build --format site
    - mkdir -p public
    - mv _book/* public/
  artifacts:
    paths:
      - public
  only:
    - main
```

### Protected Branch Builds

Build only when pushing to main or release branches:

```yaml
build-protected:
  stage: build
  script:
    - mdpress build --format pdf --output book-$CI_COMMIT_TAG.pdf
  artifacts:
    paths:
      - "*.pdf"
    expire_in: 1 year
  only:
    - tags
    - main@myorg/my-project
```

## Environment Setup

### Install Go and mdPress

**Linux (Ubuntu/Debian):**
```bash
# Install Go
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install mdPress
go install github.com/yeasy/mdpress@latest
```

**macOS:**
```bash
# Install Go via Homebrew
brew install go

# Install mdPress
go install github.com/yeasy/mdpress@latest
```

**Docker:**
```dockerfile
FROM golang:1.25-alpine

# Install Chrome for PDF rendering
RUN apk add --no-cache chromium

# Install mdPress
RUN go install github.com/yeasy/mdpress@latest

WORKDIR /workspace
ENTRYPOINT ["mdpress"]
```

### Environment Variables

```bash
# Chrome/Chromium path (if not auto-detected)
export MDPRESS_CHROME_PATH=/usr/bin/chromium

# GitHub token for private repo builds
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx

# Cache directory location
export MDPRESS_CACHE_DIR=/tmp/mdpress-cache
```

### Install Chrome for PDF Rendering

**Ubuntu/Debian:**
```bash
apt-get update
apt-get install -y chromium-browser
# or: apt-get install -y google-chrome-stable
```

**Alpine (Docker):**
```dockerfile
RUN apk add --no-cache chromium
```

**macOS:**
```bash
brew install chromium
# or: brew install google-chrome
```

### CJK Font Support

For books with Chinese, Japanese, or Korean characters, install CJK fonts:

**Ubuntu/Debian:**
```bash
apt-get install -y fonts-noto-cjk fonts-noto-cjk-extra
```

**Alpine (Docker):**
```dockerfile
RUN apk add --no-cache font-noto-cjk
```

**macOS:**
```bash
# Already included with system fonts
# Or install Noto fonts separately:
brew install font-noto-sans-cjk
```

## Caching Go Modules

### GitHub Actions

```yaml
- uses: actions/setup-go@v4
  with:
    go-version: '1.25'
    cache: 'go'  # Automatically caches Go modules

- name: Download modules
  run: go mod download
```

### GitLab CI

```yaml
before_script:
  - go env
  - export GOPATH="$CI_PROJECT_DIR/.go"
  - export PATH="$GOPATH/bin:$PATH"

cache:
  paths:
    - .go/pkg/mod/
```

## Building and Caching mdPress Cache

Preserve mdPress build cache across CI/CD runs to speed up rebuilds:

### GitHub Actions

```yaml
- uses: actions/cache@v3
  with:
    path: .mdpress-cache
    key: mdpress-${{ hashFiles('**/*.md', 'book.yaml') }}
    restore-keys: |
      mdpress-
```

This caches based on file contents. When Markdown files change, the cache key changes and a fresh build occurs.

### GitLab CI

```yaml
cache:
  paths:
    - .mdpress-cache/
  key:
    files:
      - '**/*.md'
      - 'book.yaml'
    prefix: ${CI_COMMIT_REF_SLUG}
```

## Deploying Output

### GitHub Pages Deployment

Automatic deployment of HTML site to GitHub Pages:

```yaml
- name: Build site
  run: mdpress build --format site

- uses: actions/upload-pages-artifact@v2
  with:
    path: '_book/'

- uses: actions/deploy-pages@v2
  if: github.ref == 'refs/heads/main'
```

Enable in repository settings:
1. Go to Settings → Pages
2. Set Source to "GitHub Actions"
3. Workflow deploys automatically on push to main

Access site at: `https://username.github.io/repo-name/`

### GitLab Pages Deployment

Automatic deployment of HTML site to GitLab Pages:

```yaml
pages:
  stage: deploy
  script:
    - mdpress build --format site
    - mkdir -p public
    - mv _book/* public/
  artifacts:
    paths:
      - public
  only:
    - main
```

Access site at: `https://username.gitlab.io/project-name/`

### AWS S3 Deployment

Deploy HTML site to S3:

```yaml
deploy-s3:
  stage: deploy
  script:
    - mdpress build --format site
    - aws s3 sync _book/ s3://my-docs-bucket/ --delete
  only:
    - main
  environment:
    name: production
    url: https://docs.example.com
```

### Custom Server Deployment

Deploy via SCP or rsync:

```yaml
deploy:
  stage: deploy
  script:
    - mdpress build --format site
    - scp -r _book/* deploy@docs.example.com:/var/www/docs/
  only:
    - main
  tags:
    - deploy  # Use specific runner with SSH access
```

## Generating PDF Artifacts

### Release PDF from Tags

Build and release PDF automatically when you create a git tag:

```yaml
release-pdf:
  stage: build
  script:
    - mdpress build --format pdf --output "book-$CI_COMMIT_TAG.pdf"
  artifacts:
    paths:
      - "*.pdf"
    expire_in: 1 year
  only:
    - tags
```

Workflow:
```bash
# Tag a release
git tag v1.0.0
git push origin v1.0.0

# CI/CD automatically:
# 1. Builds book-v1.0.0.pdf
# 2. Stores as artifact
# 3. Can be downloaded from CI/CD UI
```

### Release to GitHub Releases

Upload PDF to GitHub Releases:

```yaml
- name: Create Release
  if: startsWith(github.ref, 'refs/tags/')
  uses: softprops/action-gh-release@v1
  with:
    files: output.pdf
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Best Practices

### 1. Validate Before Build

Always validate configuration early:

```yaml
validate:
  script:
    - mdpress validate
  artifacts:
    reports:
      junit: validation-report.xml
```

### 2. Fail Fast on Pull Requests

Don't require full PDF builds for PR validation:

```yaml
pull_request:
  script:
    - mdpress validate
    - mdpress build --format html  # Fast validation
```

### 3. Use Semantic Versioning for Releases

Tag releases semantically:

```bash
# Major release
git tag v2.0.0

# Minor release (new features)
git tag v1.1.0

# Patch release (bug fixes)
git tag v1.0.1
```

Builds for tags automatically generate versioned artifacts.

### 4. Cache Strategically

```yaml
cache:
  key:
    # Different cache per branch and file changes
    files:
      - 'book.yaml'
      - '**/*.md'
    prefix: ${CI_COMMIT_REF_SLUG}
  paths:
    - .mdpress-cache/
```

This ensures cache invalidates when source files change.

### 5. Monitor Build Time

Track build times to catch performance regressions:

```yaml
- name: Build and time
  run: |
    time mdpress build --format pdf
    # Output example:
    # real    0m45.123s
    # user    3m12.456s
    # sys     0m8.901s
```

If builds slow down over time, check for:
- New large image files
- Addition of many new chapters
- Theme or CSS changes

### 6. Parallel Builds for Multiple Formats

Build different formats in parallel to save time:

```yaml
strategy:
  matrix:
    format: [pdf, html, epub, site]

steps:
  - run: mdpress build --format ${{ matrix.format }}
```

Reduces total build time significantly.

### 7. Clean Up Old Artifacts

Prevent disk space issues:

```yaml
artifacts:
  paths:
    - output.pdf
  expire_in: 30 days  # Auto-delete after 30 days
```

## Example: Complete Multi-Stage Pipeline

```yaml
name: Complete Book Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  release:
    types: [published]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress validate

  build-pr:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    needs: validate
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress build --format html
      - uses: actions/upload-artifact@v3
        with:
          name: pr-preview
          path: output.html

  build-release:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    needs: validate
    strategy:
      matrix:
        format: [pdf, html, epub]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go install github.com/yeasy/mdpress@latest
      - uses: actions/cache@v3
        with:
          path: .mdpress-cache
          key: mdpress-${{ matrix.format }}-${{ hashFiles('**/*.md') }}
      - run: mdpress build --format ${{ matrix.format }}
      - uses: actions/upload-artifact@v3
        with:
          name: book-${{ matrix.format }}
          path: output.*

  deploy:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    needs: build-release
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress build --format site
      - uses: actions/upload-pages-artifact@v2
        with:
          path: '_book/'
      - uses: actions/deploy-pages@v2
```

This pipeline:
- Validates all changes
- Builds PR previews (fast)
- Builds all formats on main (cached)
- Deploys site to GitHub Pages automatically

## Troubleshooting CI/CD

### Chrome Not Found Error

```
Error: Chrome binary not found
```

Solution: Install Chrome in CI environment:

```yaml
- name: Install Chrome
  run: |
    apt-get update
    apt-get install -y chromium-browser
```

### Cache Not Working

```
Key not found, creating cache
```

Ensure cache path exists:

```yaml
- run: mkdir -p .mdpress-cache
- uses: actions/cache@v3
  with:
    path: .mdpress-cache
```

### Build Timeouts

Increase timeout for large books:

```yaml
timeout-minutes: 30  # Increase from default 360
steps:
  - run: mdpress build --format pdf
```

### Out of Memory

Reduce parallelism by limiting resources:

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    # GitHub provides up to 7 GB RAM by default
    # For very large books, use a larger runner
    runs-on: ubuntu-latest-xl  # 14 GB RAM
```
