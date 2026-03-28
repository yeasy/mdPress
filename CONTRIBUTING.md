# Contributing Guide

[中文版](CONTRIBUTING_zh.md)

Thank you for your interest in mdPress! We welcome all forms of contribution.

## How to Contribute

### Reporting Bugs

If you find a bug, please file an issue on [GitHub Issues](https://github.com/yeasy/mdpress/issues) with:

- OS and version
- Go version (`go version`)
- Chrome/Chromium version
- mdPress version (`mdpress --version`)
- Steps to reproduce (ideally with a minimal `book.yaml` and Markdown files)
- Expected vs. actual behavior
- Relevant logs (use `--verbose` for detailed output)

### Feature Requests

Feature suggestions are welcome. Please describe in the issue:

- The problem you want to solve
- Your proposed solution
- Possible alternatives
- Which version this feature fits into (see [ROADMAP](docs/ROADMAP.md))

### Submitting Code

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Install pre-commit hooks: `make hooks`
4. Write code and add tests
5. Commit changes: `git commit -m "feat: add new feature"`
   The pre-commit hook automatically runs gofmt, go vet, golangci-lint, build, and fast tests.
6. Push the branch: `git push origin feature/my-feature`
7. Open a Pull Request describing your changes and motivation

## Development Setup

### Prerequisites

- Go 1.25 or later
- Chrome or Chromium browser (for PDF generation tests)
- GNU Make
- (Optional) [golangci-lint](https://golangci-lint.run/) for linting

### Setup Steps

```bash
# 1. Clone the repo
git clone https://github.com/yeasy/mdpress.git
cd mdpress

# 2. Build
make build

# 3. Run tests
make test

# 4. Run example to verify build
make example
```

### Common Development Commands

```bash
make build      # Build binary to bin/mdpress
make test       # Run all tests (with race detection)
make check      # Run fmt + lint + build + fast tests (pre-commit gate)
make lint       # Static analysis (go vet + golangci-lint)
make fmt        # Format code (gofmt)
make coverage   # Generate test coverage report (coverage.html)
make clean      # Clean build artifacts
make example    # Build example PDF using examples/
```

## Code Conventions

### Code Style

- Follow standard Go code style, use `gofmt` for formatting
- Line width should not exceed 120 characters (recommended)
- All exported functions, types, and methods must have documentation comments
- Comments should be in English for consistency

### Project Structure

| Directory | Purpose | Notes |
|-----------|---------|-------|
| `cmd/` | CLI commands (cobra) | Name files by function, no business logic |
| `internal/` | Internal packages, not exported | Each sub-package has a single responsibility |
| `pkg/utils/` | Shared utility functions | Pure utility functions with no business dependencies |
| `tests/` | Integration and e2e tests | Unit tests go in `_test.go` files within packages |
| `themes/` | Theme YAML configs | New themes must also update `internal/theme/builtin.go` |
| `examples/` | Example project files | Used for documentation and `make example` |

### Error Handling

- Use `fmt.Errorf("xxx: %w", err)` to wrap errors, preserving the call chain
- Return meaningful error messages to help users identify issues
- Avoid `panic` except for unrecoverable errors during initialization

### Logging

- Use standard library `log/slog` for logging
- `--verbose` mode outputs Debug level logs
- Normal mode only outputs Info level and above

## Commit Message Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Common types:

| type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `style` | Code formatting (no logic changes) |
| `refactor` | Refactoring |
| `test` | Test-related changes |
| `chore` | Build/toolchain changes |
| `perf` | Performance improvements |

Example:

```
feat(config): add SUMMARY.md auto-discovery support

When chapters are not defined in book.yaml, automatically look for
SUMMARY.md in the same directory and parse chapter structure.

Closes #42
```

## Testing Requirements

### Unit Tests

- New features must include corresponding unit tests
- Test files go in the corresponding package directory with `_test.go` suffix
- Use Go's standard testing framework (`testing` package)
- Test function naming: `TestXxx_Description` (e.g., `TestParseConfig_EmptyChapters`)

### Test Coverage

- Core packages (config, markdown, renderer, toc, crossref) target ≥ 80% coverage
- New code should not lower existing coverage levels
- Run `make coverage` to view the coverage report

### Integration Tests

- Features involving multiple modules should have integration tests in `tests/`
- Integration tests should use test data in `tests/golden/testdata/`

### End-to-End Tests

- Features involving CLI commands and file I/O should have e2e tests
- PDF generation tests can use build tags (e.g., `//go:build e2e`) to skip when Chromium is not available in CI

### Golden Tests

Golden tests are snapshot-based regression tests that capture the expected output of HTML generation. They help catch unintended changes to rendering output.

- **Run golden tests**: `go test ./tests/golden/...`
- **Update golden files**: `go test ./tests/golden/... -update`
- **When to update**: Only update golden files after intentional changes to HTML output (styling, structure, features). Golden files are stored in `tests/golden/testdata/golden/`
- **How they work**: Tests render markdown input to HTML, normalize volatile fields (dates), and compare against previously captured golden files. On first run, golden files are created and the test is skipped for manual review.

### Regression Testing with Real Samples

- Changes to GitBook compatibility, chapter links, TOC parsing, or multi-language features should be regression-tested against at least one real book sample
- Recommended samples: `docker_practice` (deep SUMMARY.md, images, chapter cross-links) and `learning_pickleball` (LANGS.md, bilingual directory behavior)
- Document any edge cases or limitations in README or ROADMAP rather than leaving them implicit in code

## PR Review Process

1. CI automatically runs tests and lint when a PR is created
2. At least one maintainer must approve the Code Review
3. All CI checks must pass
4. PR description should include: changes made, motivation, and testing approach
5. Breaking changes must be explicitly noted in the PR description

## Documentation Contributions

- Project documentation is written in Markdown
- `README.md` and `README_zh.md` are maintained separately for English and Chinese
- New features must also update the feature list and CLI commands in both README files
- Architecture changes must also update `docs/ARCHITECTURE.md` and `docs/ARCHITECTURE_zh.md`

## License

By contributing to mdPress, you agree that your contributions will be licensed under the [MIT License](LICENSE).
