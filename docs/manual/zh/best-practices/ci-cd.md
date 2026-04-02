# CI/CD 集成

使用持续集成和持续部署自动化书籍构建。本指南涵盖 GitHub Actions、GitLab CI 和一般 CI/CD 最佳实践的设置。

## GitHub Actions 工作流

### 基础 PDF 构建和部署到 GitHub Pages

创建 `.github/workflows/build.yml`：

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
    env:
      MDPRESS_CACHE_DIR: .mdpress-cache

    steps:
      - name: Checkout code
        uses: actions/checkout@v6

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version: '1.26'
          cache: 'go'

      - name: Install mdPress
        run: go install github.com/yeasy/mdpress@latest

      - name: Cache mdPress cache
        uses: actions/cache@v4
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
        uses: actions/upload-artifact@v4
        with:
          name: book-pdf
          path: output.pdf
          retention-days: 30

      - name: Build HTML site
        run: mdpress build --format site

      - name: Deploy to GitHub Pages
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        uses: actions/upload-pages-artifact@v4
        with:
          path: '_book/'

      - name: Deploy Pages
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        id: deployment
        uses: actions/deploy-pages@v4
```

### 拉取请求预览

为拉取请求构建预览以验证更改：

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
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
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
        uses: actions/upload-artifact@v4
        with:
          name: pr-preview-${{ github.event.number }}
          path: pr-preview.html
```

### 矩阵构建（多种格式）

同时构建 PDF、HTML 和 ePub：

```yaml
name: Multi-Format Build

on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        format: [pdf, html, epub, site]
    env:
      MDPRESS_CACHE_DIR: .mdpress-cache

    steps:
      - uses: actions/checkout@v6

      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
          cache: 'go'

      - run: go install github.com/yeasy/mdpress@latest

      - uses: actions/cache@v4
        with:
          path: .mdpress-cache
          key: ${{ runner.os }}-mdpress-${{ matrix.format }}-${{ hashFiles('**/*.md') }}

      - name: Build ${{ matrix.format }}
        run: mdpress build --format ${{ matrix.format }}

      - name: Upload ${{ matrix.format }} artifact
        uses: actions/upload-artifact@v4
        with:
          name: book-${{ matrix.format }}
          path: |
            output.*
            _book/
```

## GitLab CI 设置

### 基础管道

创建 `.gitlab-ci.yml`：

```yaml
stages:
  - validate
  - build
  - deploy

variables:
  CACHE_COMPRESSION_LEVEL: fastest
  MDPRESS_CACHE_DIR: .mdpress-cache

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

### 保护分支构建

仅在推送到 main 或发布分支时构建：

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

## 环境设置

### 安装 Go 和 mdPress

**Linux (Ubuntu/Debian)：**
```bash
# 安装 Go
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 安装 mdPress
go install github.com/yeasy/mdpress@latest
```

**macOS：**
```bash
# 通过 Homebrew 安装 Go
brew install go

# 安装 mdPress
go install github.com/yeasy/mdpress@latest
```

**Docker：**
```dockerfile
FROM golang:1.26-alpine

# 安装 Chrome 用于 PDF 渲染
RUN apk add --no-cache chromium

# 安装 mdPress
RUN go install github.com/yeasy/mdpress@latest

WORKDIR /workspace
ENTRYPOINT ["mdpress"]
```

### 环境变量

```bash
# Chrome/Chromium 路径（如果未自动检测）
export MDPRESS_CHROME_PATH=/usr/bin/chromium

# 用于私有仓库构建的 GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx

# 缓存目录位置
export MDPRESS_CACHE_DIR=/tmp/mdpress-cache
```

### 安装 Chrome 用于 PDF 渲染

**Ubuntu/Debian：**
```bash
apt-get update
apt-get install -y chromium-browser
# 或：apt-get install -y google-chrome-stable
```

**Alpine (Docker)：**
```dockerfile
RUN apk add --no-cache chromium
```

**macOS：**
```bash
brew install chromium
# 或：brew install google-chrome
```

### CJK 字体支持

对于包含中文、日文或韩文字符的书籍，安装 CJK 字体：

**Ubuntu/Debian：**
```bash
apt-get install -y fonts-noto-cjk fonts-noto-cjk-extra
```

**Alpine (Docker)：**
```dockerfile
RUN apk add --no-cache font-noto-cjk
```

**macOS：**
```bash
# 已包含在系统字体中
# 或单独安装 Noto 字体：
brew install font-noto-sans-cjk
```

## 缓存 Go 模块

### GitHub Actions

```yaml
- uses: actions/setup-go@v6
  with:
    go-version: '1.26'
    cache: 'go'  # 自动缓存 Go 模块

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

## 构建和缓存 mdPress 缓存

跨 CI/CD 运行保留 mdPress 构建缓存以加速重建：

### GitHub Actions

```yaml
- uses: actions/cache@v4
  with:
    path: .mdpress-cache
    key: mdpress-${{ hashFiles('**/*.md', 'book.yaml') }}
    restore-keys: |
      mdpress-
```

这基于文件内容缓存。当 Markdown 文件更改时，缓存密钥更改并进行新的构建。

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

## 部署输出

### GitHub Pages 部署

自动部署 HTML 网站到 GitHub Pages：

```yaml
- name: Build site
  run: mdpress build --format site

- uses: actions/upload-pages-artifact@v4
  with:
    path: '_book/'

- uses: actions/deploy-pages@v4
  if: github.ref == 'refs/heads/main'
```

在仓库设置中启用：
1. 转到 Settings → Pages
2. 将 Source 设置为 "GitHub Actions"
3. 工作流在推送到 main 时自动部署

在以下地址访问网站：`https://username.github.io/repo-name/`

### GitLab Pages 部署

自动部署 HTML 网站到 GitLab Pages：

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

在以下地址访问网站：`https://username.gitlab.io/project-name/`

### AWS S3 部署

部署 HTML 网站到 S3：

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

### 自定义服务器部署

通过 SCP 或 rsync 部署：

```yaml
deploy:
  stage: deploy
  script:
    - mdpress build --format site
    - scp -r _book/* deploy@docs.example.com:/var/www/docs/
  only:
    - main
  tags:
    - deploy  # 使用具有 SSH 访问权限的特定运行器
```

## 生成 PDF 工件

### 从标签发布 PDF

创建 git 标签时自动构建和发布 PDF：

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

工作流：
```bash
# 标记发布
git tag v1.0.0
git push origin v1.0.0

# CI/CD 自动：
# 1. 构建 book-v1.0.0.pdf
# 2. 存储为工件
# 3. 可从 CI/CD UI 下载
```

### 发布到 GitHub Releases

上传 PDF 到 GitHub Releases：

```yaml
- name: Create Release
  if: startsWith(github.ref, 'refs/tags/')
  uses: softprops/action-gh-release@v2
  with:
    files: output.pdf
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## 最佳实践

### 1. 构建前验证

始终早期验证配置：

```yaml
validate:
  script:
    - mdpress validate
  artifacts:
    reports:
      junit: validation-report.xml
```

### 2. 拉取请求上快速失败

不要求 PR 验证的完整 PDF 构建：

```yaml
pull_request:
  script:
    - mdpress validate
    - mdpress build --format html  # 快速验证
```

### 3. 对发布使用语义版本控制

语义标记发布：

```bash
# 主版本发布
git tag v2.0.0

# 次版本发布（新功能）
git tag v1.1.0

# 补丁版本发布（错误修复）
git tag v1.0.1
```

构建会自动为标签生成版本化工件。

### 4. 战略性缓存

```yaml
cache:
  key:
    # 不同缓存，按分支和文件更改
    files:
      - 'book.yaml'
      - '**/*.md'
    prefix: ${CI_COMMIT_REF_SLUG}
  paths:
    - .mdpress-cache/
```

这确保当源文件更改时缓存失效。

### 5. 监控构建时间

跟踪构建时间以捕捉性能回归：

```yaml
- name: Build and time
  run: |
    time mdpress build --format pdf
    # 输出示例：
    # real    0m45.123s
    # user    3m12.456s
    # sys     0m8.901s
```

如果构建随时间变慢，检查：
- 新的大型图像文件
- 许多新章节的添加
- 主题或 CSS 更改

### 6. 多格式的并行构建

以并行方式构建不同格式以节省时间：

```yaml
strategy:
  matrix:
    format: [pdf, html, epub, site]

steps:
  - run: mdpress build --format ${{ matrix.format }}
```

显著减少总构建时间。

### 7. 清理旧工件

防止磁盘空间问题：

```yaml
artifacts:
  paths:
    - output.pdf
  expire_in: 30 days  # 30 天后自动删除
```

## 示例：完整多阶段管道

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
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress validate

  build-pr:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    needs: validate
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress build --format html
      - uses: actions/upload-artifact@v4
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
    env:
      MDPRESS_CACHE_DIR: .mdpress-cache
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - run: go install github.com/yeasy/mdpress@latest
      - uses: actions/cache@v4
        with:
          path: .mdpress-cache
          key: mdpress-${{ matrix.format }}-${{ hashFiles('**/*.md') }}
      - run: mdpress build --format ${{ matrix.format }}
      - uses: actions/upload-artifact@v4
        with:
          name: book-${{ matrix.format }}
          path: output.*

  deploy:
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    needs: build-release
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.26'
      - run: go install github.com/yeasy/mdpress@latest
      - run: mdpress build --format site
      - uses: actions/upload-pages-artifact@v4
        with:
          path: '_book/'
      - uses: actions/deploy-pages@v4
```

此管道：
- 验证所有更改
- 构建 PR 预览（快速）
- 在 main 上构建所有格式（缓存）
- 自动部署网站到 GitHub Pages

## CI/CD 故障排除

### Chrome 未找到错误

```
Error: Chrome binary not found
```

解决方案：在 CI 环境中安装 Chrome：

```yaml
- name: Install Chrome
  run: |
    apt-get update
    apt-get install -y chromium-browser
```

### 缓存不工作

```
Key not found, creating cache
```

确保缓存路径存在：

```yaml
- run: mkdir -p .mdpress-cache
- uses: actions/cache@v4
  with:
    path: .mdpress-cache
```

### 构建超时

增加大型书籍的超时：

```yaml
timeout-minutes: 30  # 从默认 360 减少
steps:
  - run: mdpress build --format pdf
```

### 内存不足

通过限制资源减少并行性：

```yaml
jobs:
  build:
    # runs-on: ubuntu-latest  # 默认：7 GB RAM
    # 对于非常大的书籍，使用更大的运行器：
    runs-on: ubuntu-latest-xl  # 14 GB RAM
```
