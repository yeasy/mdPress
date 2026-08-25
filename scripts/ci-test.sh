#!/usr/bin/env bash
# ci-test.sh - full local pre-push check for mdPress.
#
# This is a DEVELOPER convenience script, not the CI pipeline. No workflow
# invokes it; GitHub Actions runs the equivalent steps as separate jobs in
# .github/workflows/ci.yml, and coverage is enforced by Codecov (see
# codecov.yml) against the pull request's base commit rather than by the fixed
# threshold below. The name once implied otherwise, which made the 80% line
# below look like an enforced gate while coverage could regress on any green PR.
#
# The threshold here is a local guard rail: it runs the full (non-short) suite,
# so Chrome- and typst-dependent tests execute and the number is higher than the
# one CI measures.
#
# Override the threshold with COVERAGE_THRESHOLD=<n> ./scripts/ci-test.sh
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
THRESHOLD="${COVERAGE_THRESHOLD:-80}"
# awk rather than bc: bc is absent from a stock macOS-on-ARM and several slim
# Linux images, so the comparison used to fail before it could be evaluated.
if awk -v c="$COVERAGE" -v t="$THRESHOLD" 'BEGIN { exit !(c < t) }'; then
    echo "FAIL: coverage ${COVERAGE}% is below the local threshold ${THRESHOLD}%"
    exit 1
fi
echo "Coverage ${COVERAGE}% meets the local threshold ${THRESHOLD}%"

echo ""
echo "=== All local pre-push checks passed ==="
