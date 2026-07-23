# Installation

mdPress is a Go CLI. Install it with Homebrew, Docker, a pre-built binary, `go install`, or build it from source.

## Requirements

- Go 1.26 or newer if you want to build from source.
- Chrome or Chromium for PDF output.
- Typst if you want to try `mdpress build --format typst`.
- PlantUML and Java only if you use `plantuml` code blocks.

## Homebrew (macOS)

```bash
brew tap yeasy/tap
brew install --cask mdpress
```

The cask clears the macOS quarantine flag automatically, so Gatekeeper will not block it.

## Docker

```bash
# Minimal image (~15 MB) — no Chromium, so pick a format that does not need it
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/book" \
  ghcr.io/yeasy/mdpress build --format site

# Full image (~300 MB) — bundles Chromium, so PDF works
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/book" \
  ghcr.io/yeasy/mdpress:full build --format pdf
```

`build` defaults to PDF, which the minimal image cannot produce. Use `site`, `html`, or `epub` there, or switch to the `:full` tag.

Both images run as an in-image `mdpress` user whose UID does not exist on your host, so without `--user` the container either cannot write into the mounted directory at all or leaves the generated files owned by an unrelated UID. `--user "$(id -u):$(id -g)"` makes the outputs yours. On Docker Desktop for macOS and Windows the mapping is handled for you and `--user` can be omitted.

## Download Binary

Download a pre-built binary for your platform from [GitHub Releases](https://github.com/yeasy/mdpress/releases).

Supported platforms: macOS (amd64 / arm64), Linux (amd64 / arm64), Windows (amd64 / arm64).

> **macOS Gatekeeper note:** binaries are not notarized yet. If you download the binary directly and macOS blocks it, clear the quarantine flag once:
>
> ```bash
> xattr -d com.apple.quarantine ./mdpress
> ```

## Go Install

```bash
go install github.com/yeasy/mdpress@latest
mdpress --version
```

`go install` places the binary in `$GOPATH/bin` unless `GOBIN` is set. If `mdpress` is not found, add that directory to your `PATH`.

## Build From Source

```bash
git clone https://github.com/yeasy/mdpress.git
cd mdpress
go build -o mdpress .
./mdpress --version
```

## Environment Variables

- `GITHUB_TOKEN` for private GitHub sources and rate limits.
- `MDPRESS_CHROME_PATH` to point at a specific Chrome or Chromium binary.
- `MDPRESS_CACHE_DIR` to move the cache.
- `MDPRESS_DISABLE_CACHE` to disable caching.
- `PLANTUML_JAR` to use a local PlantUML JAR with Java.

## Check The Install

```bash
mdpress --help
mdpress doctor
```

## Upgrade

Update an existing install to the latest release in place:

```bash
mdpress upgrade
```

(Homebrew users can also run `brew upgrade --cask mdpress`.)
