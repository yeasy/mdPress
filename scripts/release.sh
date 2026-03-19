#!/usr/bin/env bash
# release.sh - Cross-platform release builder for mdPress
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
DIST_DIR="dist"
MODULE="github.com/yeasy/mdpress"
LDFLAGS="-s -w -X ${MODULE}/cmd.Version=${VERSION} -X ${MODULE}/cmd.BuildTime=${BUILD_TIME}"

echo "Building mdPress ${VERSION} (${BUILD_TIME})"
echo ""

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    output_name="mdpress-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "  Building ${GOOS}/${GOARCH}..."
    GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "${DIST_DIR}/${output_name}" .

    # Create tarball (or zip for Windows)
    pushd "${DIST_DIR}" > /dev/null
    if [ "$GOOS" = "windows" ]; then
        zip -q "${output_name%.exe}.zip" "${output_name}"
    else
        tar czf "${output_name}.tar.gz" "${output_name}"
    fi
    popd > /dev/null
done

echo ""
echo "Release artifacts in ${DIST_DIR}/:"
ls -lh "${DIST_DIR}/"
echo ""
echo "Done."
