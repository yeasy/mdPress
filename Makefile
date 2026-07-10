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
# gofmt verification + lint + build + fast tests.
# Does NOT rewrite files: it fails if anything is unformatted so the hook
# blocks the commit. Run 'make fmt' to fix, then re-stage.
# Invoked by .githooks/pre-commit.

.PHONY: check
check:
	@echo ">>> [check] gofmt"
	@UNFMT=$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*' -not -path './.cache/*')); \
	if [ -n "$$UNFMT" ]; then \
		echo "Files need formatting:"; echo "$$UNFMT"; \
		echo "Run 'make fmt' and re-stage the changes (git add), then commit again."; exit 1; \
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

# Removes build artifacts and generated output anywhere in the tree:
# bin/, dist/, coverage files, tmp/, output/, and every _book/, _book.old/,
# _output/, *_site/, and mdpress-serve-*.tmp directory (build/serve output).
# The repo-local Go caches under .cache/ are intentionally kept so incremental
# builds stay fast; use 'make clean-cache' to remove them.
.PHONY: clean
clean:
	@echo ">>> Cleaning build artifacts and generated output..."
	$(GO) clean
	rm -rf bin/ dist/ tmp/ output/ coverage.txt coverage.html
	find . \( -path ./.git -o -path ./.cache \) -prune -o -type d \
		\( -name '_book' -o -name '_book.old' -o -name '_output' -o -name '*_site' -o -name 'mdpress-serve-*.tmp' \) \
		-prune -exec rm -rf {} +
	@echo ">>> Done. Go caches under .cache/ were kept (run 'make clean-cache' to remove them)."

# Removes the repo-local Go module/build caches created by running targets
# with CACHE_DIR=$(CURDIR)/.cache (they can grow to gigabytes). The module
# cache is written read-only, so restore write permission before deleting.
.PHONY: clean-cache
clean-cache:
	@echo ">>> Removing repo-local Go caches (.cache/)..."
	@if [ -d .cache ]; then chmod -R u+w .cache 2>/dev/null || true; rm -rf .cache; fi
	@echo ">>> Done."

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
	@echo "  make check      Pre-commit quality gate (gofmt check + lint + build + test)"
	@echo "  make hooks      Install pre-commit git hooks"
	@echo "  make clean      Remove build artifacts and generated output (_book, *_site, tmp, coverage)"
	@echo "  make clean-cache Remove the repo-local Go caches under .cache/"
	@echo "  make docker     Build both Docker images (minimal + full)"
	@echo "  make example    Run the example project"
	@echo "  make help       Show this help message"
