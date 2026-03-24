# Installation

mdPress is a fast, modern documentation and book generation tool written in Go. This guide covers installing mdPress on your system.

## System Requirements

Before installing mdPress, ensure your system meets these requirements:

- **Go 1.25 or later** - Required for building and running mdPress
- **Chrome or Chromium browser** - Required for PDF generation
- **Typst** (optional) - For advanced PDF styling and typography features
- **PlantUML** (optional) - For rendering diagram syntax in your documentation

### Supported Platforms

mdPress runs on:
- macOS (Intel and Apple Silicon)
- Linux (x86_64 and arm64)
- Windows (x86_64 and arm64)

## Installation Methods

### Using go install

The quickest way to install mdPress is with `go install`:

```bash
go install github.com/mdpress/mdpress@latest
```

This downloads the latest release and installs the binary to your `$GOPATH/bin` directory (usually `~/go/bin`).

Verify the installation:

```bash
mdpress --version
```

### Building from Source

If you prefer to build from source or want to use development features:

```bash
git clone https://github.com/mdpress/mdpress.git
cd mdpress
go build -o mdpress ./cmd/mdpress
```

Copy the binary to a location in your PATH, or use it directly:

```bash
./mdpress --version
```

## Verifying Installation

After installation, verify mdPress is working:

```bash
mdpress --help
```

You should see the help output with available commands. Check your version:

```bash
mdpress --version
```

## Environment Variables

mdPress respects several environment variables for configuration:

### GITHUB_TOKEN

GitHub API token for accessing private repositories or avoiding rate limits. Set this if you plan to fetch content from GitHub:

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

### MDPRESS_CHROME_PATH

Explicit path to your Chrome or Chromium executable. mdPress automatically finds Chrome/Chromium in standard locations, but use this if your installation is in a non-standard location:

```bash
export MDPRESS_CHROME_PATH=/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome
```

On Linux:

```bash
export MDPRESS_CHROME_PATH=/usr/bin/chromium
```

On Windows:

```bash
set MDPRESS_CHROME_PATH=C:\Program Files\Google\Chrome\Application\chrome.exe
```

### MDPRESS_CACHE_DIR

Directory where mdPress stores cached content and temporary files. Default is `~/.mdpress/cache`. Useful for CI/CD environments:

```bash
export MDPRESS_CACHE_DIR=/var/cache/mdpress
```

### MDPRESS_DISABLE_CACHE

Disable all caching (useful for development or testing). Set to any value:

```bash
export MDPRESS_DISABLE_CACHE=1
mdpress build  # Cache will be bypassed
```

## Optional Dependencies

### Installing Typst

Typst provides advanced PDF formatting and styling capabilities. Visit https://github.com/typst/typst to install for your platform.

On macOS with Homebrew:

```bash
brew install typst
```

On Linux:

```bash
# Download from https://github.com/typst/typst/releases
# Or use your package manager
apt-get install typst  # Debian/Ubuntu (if available)
```

On Windows:

```bash
choco install typst  # Using Chocolatey
```

### Installing PlantUML

PlantUML is needed if you use diagram syntax in your documentation. It requires Java:

```bash
# macOS
brew install plantuml

# Linux
apt-get install plantuml  # Debian/Ubuntu

# Windows - Download from https://plantuml.com/download
```

## Troubleshooting

### Command not found: mdpress

Ensure `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Add this to your shell profile (`~/.bashrc`, `~/.zshrc`, or `~/.profile`) to make it permanent.

### Chrome/Chromium not found

Set the Chrome path explicitly:

```bash
export MDPRESS_CHROME_PATH=/path/to/chrome
mdpress build --format pdf
```

### Permission denied on Linux

On some Linux systems, you may need to allow mdPress to run:

```bash
chmod +x $(which mdpress)
```

### Slow PDF generation

PDF generation is I/O intensive. If it's slow:

1. Ensure you have adequate disk space
2. Try disabling cache: `export MDPRESS_DISABLE_CACHE=1`
3. Use `MDPRESS_CACHE_DIR` to point to a faster storage device
4. Check your Chrome/Chromium path is correct
