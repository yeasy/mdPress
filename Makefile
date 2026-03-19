# mdpress Makefile
# Tooling for building high-quality output from Markdown books

BINARY_NAME=mdpress
MODULE=github.com/yeasy/mdpress
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.BuildTime=$(BUILD_TIME)"

GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOVET=$(GO) vet

# Default target
.PHONY: all
all: lint test build

# Build
.PHONY: build
build:
	@echo ">>> Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) .
	@echo ">>> Build complete: bin/$(BINARY_NAME)"

# Install to $GOPATH/bin
.PHONY: install
install:
	@echo ">>> Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) .

# Run tests
.PHONY: test
test:
	@echo ">>> Running tests..."
	$(GOTEST) -v -race ./...

# Test coverage
.PHONY: coverage
coverage:
	@echo ">>> Generating coverage report..."
	$(GOTEST) -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo ">>> Coverage report: coverage.html"

# Static checks
.PHONY: lint
lint:
	@echo ">>> Running static checks..."
	$(GOVET) ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "Tip: install golangci-lint for more complete checks"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo ">>> Formatting code..."
	$(GO) fmt ./...

# Pre-commit quality gate: fmt check + lint + build + fast tests
# This is the single target invoked by .githooks/pre-commit.
.PHONY: check
check:
	@echo ">>> [check] gofmt"
	@UNFMT=$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$UNFMT" ]; then \
		echo "Files need formatting:"; echo "$$UNFMT"; \
		echo "Run: make fmt"; exit 1; \
	fi
	@echo ">>> [check] go vet"
	$(GOVET) ./...
	@echo ">>> [check] golangci-lint"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping (install: https://golangci-lint.run)"; \
	fi
	@echo ">>> [check] go build"
	$(GOBUILD) ./...
	@echo ">>> [check] go test -short"
	$(GOTEST) -short -count=1 ./...
	@echo ">>> All checks passed."

# Install git hooks (pre-commit runs make check)
.PHONY: hooks
hooks:
	@echo ">>> Installing git hooks..."
	git config core.hooksPath .githooks
	@echo ">>> Done. Pre-commit hook will run 'make check' before each commit."
	@echo ">>> To skip once: git commit --no-verify"

# Clean build artifacts
.PHONY: clean
clean:
	@echo ">>> Cleaning..."
	$(GOCLEAN)
	rm -rf bin/ dist/ coverage.txt coverage.html

# Download dependencies
.PHONY: deps
deps:
	@echo ">>> Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Cross-compile for multiple platforms
.PHONY: release
release: clean
	@echo ">>> Cross-compiling..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-arm64.exe .
	@echo ">>> Cross-compile complete, output directory: dist/"

# Run the example project
.PHONY: example
example: build
	@echo ">>> Building PDF from example files..."
	cd examples && ../bin/$(BINARY_NAME) build

# Show help
.PHONY: help
help:
	@echo "mdpress - Build high-quality output from Markdown books"
	@echo ""
	@echo "Available targets:"
	@echo "  make build     - Build the project"
	@echo "  make install   - Install to GOPATH"
	@echo "  make test      - Run tests"
	@echo "  make coverage  - Generate a test coverage report"
	@echo "  make lint      - Run static checks"
	@echo "  make fmt       - Format code"
	@echo "  make check     - Run all pre-commit checks (fmt + lint + build + test)"
	@echo "  make hooks     - Install pre-commit git hooks (runs make check)"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make deps      - Download dependencies"
	@echo "  make release   - Cross-compile release binaries"
	@echo "  make example   - Run the example project"
	@echo "  make help      - Show this help message"
