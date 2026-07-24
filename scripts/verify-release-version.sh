#!/usr/bin/env bash
# verify-release-version.sh -- assert that a built mdpress artifact reports the
# version it was released as.
#
# v0.7.15 and v0.8.0 both shipped binaries that called themselves "<tag>+dirty"
# and nothing in the pipeline noticed, because no job ever ran a published
# artifact. This script is that missing gate: it unpacks each artifact, runs the
# binary, and compares `mdpress version --json` against the expected version.
#
# Usage:
#   scripts/verify-release-version.sh <expected-version> <artifact> [artifact...]
#
# <expected-version> is the release version without the leading "v" (0.8.2).
# <artifact> is a .tar.gz, a .zip, or an mdpress binary. Every artifact must be
# runnable on the host, so only pass artifacts for the current OS/arch.

set -euo pipefail

if [ "$#" -lt 2 ]; then
	echo "usage: $0 <expected-version> <artifact> [artifact...]" >&2
	exit 2
fi

expected="${1#v}"
shift

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

failures=0

# report_version runs the binary and echoes the version it claims.
report_version() {
	# Parse the JSON without depending on jq: release runners have it, minimal
	# containers do not.
	"$1" version --json | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

for artifact in "$@"; do
	if [ ! -f "$artifact" ]; then
		echo "ERROR: artifact not found: $artifact" >&2
		failures=$((failures + 1))
		continue
	fi

	unpacked="$workdir/$(basename "$artifact").d"
	mkdir -p "$unpacked"

	case "$artifact" in
	*.tar.gz | *.tgz)
		tar -xzf "$artifact" -C "$unpacked"
		;;
	*.zip)
		unzip -q -o "$artifact" -d "$unpacked"
		;;
	*)
		cp "$artifact" "$unpacked/mdpress"
		;;
	esac

	binary="$(find "$unpacked" -type f \( -name 'mdpress' -o -name 'mdpress.exe' \) -print -quit)"
	if [ -z "$binary" ]; then
		echo "FAIL $artifact: no mdpress binary inside" >&2
		failures=$((failures + 1))
		continue
	fi
	chmod +x "$binary"

	reported="$(report_version "$binary" || true)"
	if [ "$reported" = "$expected" ]; then
		echo "OK   $(basename "$artifact") reports $reported"
	else
		echo "FAIL $(basename "$artifact") reports '${reported:-<nothing>}', expected '$expected'" >&2
		failures=$((failures + 1))
	fi
done

if [ "$failures" -ne 0 ]; then
	echo "$failures artifact(s) do not report version $expected" >&2
	exit 1
fi
