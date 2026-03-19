# Dependencies

This document outlines the external dependencies used by the markdown parser module.

## Required Dependencies

### goldmark
- **Module**: `github.com/yuin/goldmark`
- **Purpose**: Core Markdown parsing engine
- **Why**: Industry-standard, fast, and extensible Markdown parser with excellent GFM support
- **Typical Version**: `v1.4.2` or later

```go
import "github.com/yuin/goldmark"
```

### goldmark-highlighting
- **Module**: `github.com/yuin/goldmark-highlighting/v2`
- **Purpose**: Syntax highlighting for code blocks
- **Why**: Integrates seamlessly with goldmark and provides multiple color themes
- **Typical Version**: `v2.0.0` or later

```go
import "github.com/yuin/goldmark-highlighting/v2"
```

## Standard Library Dependencies

The module uses only standard library packages:

```go
import (
    "bytes"              // Buffer operations
    "fmt"                // String formatting
    "io/ioutil"          // File I/O
    "path/filepath"      // Path utilities
    "regexp"             // Regular expressions
    "strings"            // String manipulation
    "sync"               // Concurrency primitives
)
```

## Installation

To install the required dependencies, add them to your `go.mod`:

```bash
go get github.com/yuin/goldmark
go get github.com/yuin/goldmark-highlighting/v2
```

Or in your `go.mod` file:

```
require (
    github.com/yuin/goldmark v1.4.2
    github.com/yuin/goldmark-highlighting/v2 v2.0.0
)
```

## Dependency Tree

```
mdpress/internal/markdown
├── github.com/yuin/goldmark
│   └── (no external dependencies)
├── github.com/yuin/goldmark-highlighting/v2
│   ├── github.com/yuin/goldmark (already included)
│   └── github.com/alecthomas/chroma (transitive)
└── Standard library
```

## Version Compatibility

| Dependency | Minimum | Tested | Status |
|-----------|---------|--------|--------|
| Go | 1.16 | 1.21 | Stable |
| goldmark | 1.4.0 | 1.4.2 | Stable |
| goldmark-highlighting | 2.0.0 | 2.0.0 | Stable |

## Optional Considerations

### Future Dependencies

If adding support for additional features, consider:

```go
// For LaTeX/MathJax support
github.com/yuin/goldmark-meta

// For diagram support
github.com/yuin/goldmark-emoji

// For YAML front matter
github.com/acobaugh/goldmark-frontmatter
```

## Transitive Dependencies

The markdown module brings in these transitive dependencies through goldmark-highlighting:

- `github.com/alecthomas/chroma` - Syntax highlighter

These are automatically included by the go toolchain.

## Build Constraints

The module has no specific build constraints and works on:
- Linux
- macOS
- Windows
- Other UNIX-like systems

## Security Considerations

### goldmark
- Well-maintained by the community
- No known security vulnerabilities
- Regular security updates
- Safe HTML rendering by default

### goldmark-highlighting
- Built on top of chroma
- Actively maintained
- No security vulnerabilities
- Properly escapes output

## Performance Notes

All dependencies are designed for performance:
- **goldmark**: Pure Go implementation, efficient memory usage
- **goldmark-highlighting**: Streaming highlight with minimal overhead
- **chroma**: Pre-compiled lexers, fast processing

## Future Updates

When updating dependencies:

1. Check compatibility notes in release logs
2. Run full test suite after updates
3. Monitor for breaking changes in goldmark APIs
4. Verify HTML output remains consistent

## Alternative Implementations

If you need different functionality, consider these alternatives:

| Need | Alternative | Trade-offs |
|------|-------------|-----------|
| Markdown only | `github.com/gomarkdown/markdown` | Less GFM support |
| Maximum performance | `github.com/tetratelabs/markdown` | Fewer extensions |
| Simplicity | `github.com/russross/blackfriday` | Older, less maintained |

## Contributing

When contributing to this module:
1. Don't add unnecessary dependencies
2. Use standard library when possible
3. Document any new dependencies
4. Ensure compatibility with Go 1.16+
5. Run dependency audit: `go mod audit`

## Dependency Audit

To check for known vulnerabilities:

```bash
# Check for vulnerabilities
go mod verify

# Update go.sum
go mod tidy

# Run vulnerability scan (Go 1.18+)
go list -json -m all | nancy sleuth
```

## License Compatibility

All dependencies use compatible open-source licenses:
- goldmark: MIT License
- goldmark-highlighting: MIT License
- chroma: MIT License

This module (mdpress) should maintain license compatibility.
