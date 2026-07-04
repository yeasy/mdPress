#!/usr/bin/env bash
# release.sh - Cross-platform release builder for mdPress.
#
# GoReleaser is the single source of truth for releases (platforms, archive
# naming, in-archive binary name, checksums, packages, Homebrew cask). This
# script is a thin wrapper around a local GoReleaser snapshot build so the
# artifacts you produce locally match exactly what CI publishes. Do not
# reimplement the build matrix here -- edit .goreleaser.yml instead.
#
# Usage:
#   scripts/release.sh            # build a snapshot into ./dist
#   scripts/release.sh --help     # show goreleaser help
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

if ! command -v goreleaser >/dev/null 2>&1; then
    echo "error: goreleaser is not installed." >&2
    echo "Install it from https://goreleaser.com/install/ and retry." >&2
    exit 1
fi

if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    exec goreleaser release --help
fi

echo ">>> Building a local snapshot with GoReleaser (single source of truth)..."
echo ">>> This does NOT publish anything; artifacts land in ./dist/."
echo ""

# --snapshot: no git tag / publish required. --clean: wipe ./dist first.
goreleaser release --snapshot --clean "$@"

echo ""
echo "Release artifacts in dist/:"
ls -lh dist/
echo ""
echo "Done. To publish a real release, push a git tag; CI runs 'goreleaser release --clean'."
