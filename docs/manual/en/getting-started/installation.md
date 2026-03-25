# Installation

mdPress is a Go CLI. Install it with Go or build it from source.

## Requirements

- Go 1.25 or newer if you want to build from source.
- Chrome or Chromium for PDF output.
- Typst if you want to try `mdpress build --format typst`.
- PlantUML and Java only if you use `plantuml` code blocks.

## Install

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
