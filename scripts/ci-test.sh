#!/usr/bin/env bash
# ci-test.sh - CI test runner for mdPress
# Runs unit tests, integration tests, and coverage checks.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "=== Running go vet ==="
go vet ./...

echo ""
echo "=== Running unit tests with race detection ==="
go test -race -count=1 -timeout 120s ./internal/... ./pkg/... ./cmd/...

echo ""
echo "=== Running integration tests ==="
go test -race -count=1 -timeout 300s -tags integration ./tests/...

echo ""
echo "=== Generating coverage report ==="
go test -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...
go tool cover -func=coverage.out | tail -1

echo ""
echo "=== Checking coverage threshold ==="
COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | tr -d '%')
THRESHOLD=80
if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "WARNING: Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%"
    exit 1
else
    echo "Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%"
fi

echo ""
echo "=== All CI checks passed ==="
