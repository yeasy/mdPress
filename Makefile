# mdpress Makefile

BINARY_NAME := mdpress
MODULE      := github.com/yeasy/mdpress
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -ldflags "-s -w -X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.BuildTime=$(BUILD_TIME)"

GO      := go
GOTEST  := $(GO) test
GOBUILD := $(GO) build

# By default, use the system Go cache locations (~/go, ~/Library/Caches, etc.).
# Override CACHE_DIR to keep caches inside the workspace for CI or sandboxed
# environments where the home directory is not writable.
#   make check CACHE_DIR=$(CURDIR)/.cache
ifdef CACHE_DIR
GO_RUN_ENV  = GOPATH=$(CACHE_DIR)/go GOCACHE=$(CACHE_DIR)/go-build GOMODCACHE=$(CACHE_DIR)/gomod
LINT_RUN_ENV = $(GO_RUN_ENV) GOLANGCI_LINT_CACHE=$(CACHE_DIR)/golangci-lint
else
GO_RUN_ENV  =
LINT_RUN_ENV =
endif

# Docker
DOCKER_REPO ?= yeasy/mdpress

# ---------- Default target ----------

.PHONY: all
all: lint test build

# ---------- Build ----------

.PHONY: build
build:
	@echo ">>> Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GO_RUN_ENV) $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) .
	@echo ">>> Build complete: bin/$(BINARY_NAME)"

# Install to $GOPATH/bin
.PHONY: install
install:
	@echo ">>> Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) .

# ---------- Test ----------

.PHONY: test
test:
	@echo ">>> Running tests..."
	$(GO_RUN_ENV) $(GOTEST) -race -count=1 ./...

# Test coverage
.PHONY: coverage
coverage:
	@echo ">>> Generating coverage report..."
	$(GO_RUN_ENV) $(GOTEST) -coverprofile=coverage.txt -covermode=atomic ./...
	$(GO_RUN_ENV) $(GO) tool cover -html=coverage.txt -o coverage.html
	@echo ">>> Coverage report: coverage.html"

# ---------- Static checks ----------

.PHONY: lint
lint:
	@echo ">>> Running static checks..."
	$(GO_RUN_ENV) $(GO) vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		$(LINT_RUN_ENV) golangci-lint run ./...; \
	else \
		echo "Tip: install golangci-lint for more complete checks"; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo ">>> Formatting code..."
	$(GO_RUN_ENV) $(GO) fmt ./...

# ---------- Pre-commit quality gate ----------
# fmt check + lint + build + fast tests.
# Invoked by .githooks/pre-commit.

.PHONY: check
check: fmt
	@echo ">>> [check] gofmt"
	@UNFMT=$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*' -not -path './.cache/*')); \
	if [ -n "$$UNFMT" ]; then \
		echo "Files need formatting:"; echo "$$UNFMT"; \
		echo "Run: make fmt"; exit 1; \
	fi
	@echo ">>> [check] lint + build + test (parallel)"
	@$(MAKE) --no-print-directory -j3 _lint _build _test
	@echo ">>> All checks passed."

# Internal parallel targets for check (not meant to be called directly)
.PHONY: _lint _build _test
_lint:
	@echo ">>> [check] lint"
	@$(MAKE) --no-print-directory lint
_build:
	@echo ">>> [check] go build"
	@$(GO_RUN_ENV) $(GOBUILD) ./...
_test:
	@echo ">>> [check] go test -short"
	@$(GO_RUN_ENV) $(GOTEST) -short -count=1 -timeout 120s ./...

# Install git hooks (pre-commit runs make check)
.PHONY: hooks
hooks:
	@echo ">>> Installing git hooks..."
	git config core.hooksPath .githooks
	@echo ">>> Done. Pre-commit hook will run 'make check' before each commit."
	@echo ">>> To skip once: git commit --no-verify"

# ---------- Clean ----------

.PHONY: clean
clean:
	@echo ">>> Cleaning..."
	$(GO) clean
	rm -rf bin/ dist/ coverage.txt coverage.html

# ---------- Docker ----------

.PHONY: docker-minimal
docker-minimal:
	@echo ">>> Building minimal Docker image (no PDF support)..."
	docker build --target minimal \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_REPO):$(VERSION) \
		-t $(DOCKER_REPO):latest .
	@echo ">>> Image: $(DOCKER_REPO):$(VERSION) (minimal, ~15MB)"

.PHONY: docker-full
docker-full:
	@echo ">>> Building full Docker image (with Chromium for PDF)..."
	docker build --target full \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_REPO):$(VERSION)-full \
		-t $(DOCKER_REPO):full .
	@echo ">>> Image: $(DOCKER_REPO):$(VERSION)-full (with Chromium, ~300MB)"

.PHONY: docker
docker: docker-minimal docker-full

# ---------- Example ----------

.PHONY: example
example: build
	@echo ">>> Building PDF from example files..."
	cd examples && ../bin/$(BINARY_NAME) build

# ---------- Help ----------

.PHONY: help
help:
	@echo "mdpress — Build high-quality output from Markdown books"
	@echo ""
	@echo "Available targets:"
	@echo "  make build      Build the project"
	@echo "  make install    Install to GOPATH"
	@echo "  make test       Run tests with race detector"
	@echo "  make coverage   Generate a test coverage report"
	@echo "  make lint       Run static checks (vet + golangci-lint)"
	@echo "  make fmt        Format code"
	@echo "  make check      Pre-commit quality gate (fmt + lint + build + test)"
	@echo "  make hooks      Install pre-commit git hooks"
	@echo "  make clean      Remove build artifacts"
	@echo "  make docker     Build both Docker images (minimal + full)"
	@echo "  make example    Run the example project"
	@echo "  make help       Show this help message"
