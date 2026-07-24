#!/usr/bin/env bash
# bump-version.sh -- move every hand-maintained copy of the release version to a
# new one, in a single command.
#
# A bump used to be a checklist of files edited from memory. Missing cmd/root.go
# makes every source build report the previous version; missing README.md leaves
# a copy-pasteable download recipe pointing at an old release; missing the
# CHANGELOG stanza ships a release with no notes. None of those fail a build,
# and two of the last three releases went out with a wrong version string.
#
# Usage:
#   scripts/bump-version.sh 0.8.2          (or: make bump VERSION=0.8.2)
#
# Every rewrite is staged first and only written once all of them succeeded: a
# pattern this script can no longer find stops the whole bump with the file
# named, rather than leaving a half-bumped tree behind.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [ "$#" -ne 1 ]; then
	echo "usage: $0 <new-version>   e.g. $0 0.8.2" >&2
	exit 2
fi

new="${1#v}"
if ! printf '%s' "$new" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$'; then
	echo "ERROR: '$new' is not a semantic version like 0.8.2 or 0.9.0-rc1" >&2
	exit 2
fi

old="$(sed -n 's/^const defaultVersion = "\(.*\)"$/\1/p' cmd/root.go | head -n1)"
if [ -z "$old" ]; then
	echo "ERROR: cannot read defaultVersion from cmd/root.go" >&2
	exit 1
fi
if [ "$old" = "$new" ]; then
	echo "ERROR: the tree is already at $new" >&2
	exit 1
fi

date_today="$(date -u '+%Y-%m-%d')"
repo_url="https://github.com/yeasy/mdpress"

stagedir="$(mktemp -d)"
trap 'rm -rf "$stagedir"' EXIT
staged=()

# staged_path FILE -- where FILE's rewritten content lives while staged.
staged_path() {
	printf '%s/%s' "$stagedir" "$(printf '%s' "$1" | tr '/' '_')"
}

# require_changed FILE WHAT NEW-CONTENT -- accept a rewrite, or stop the bump.
require_changed() {
	local file="$1" what="$2" candidate="$3"
	if cmp -s "$file" "$candidate"; then
		echo "ERROR: $file: nothing to rewrite for $what (expected version $old)" >&2
		echo "       Fix the file by hand, or update scripts/bump-version.sh if its shape changed." >&2
		echo "       No files were modified." >&2
		exit 1
	fi
	local dest
	dest="$(staged_path "$file")"
	if [ "$candidate" != "$dest" ]; then
		mv "$candidate" "$dest"
	fi
	staged+=("$file")
	echo "  staged $file ($what)"
}

# current FILE -- the staged content if the file was already rewritten, else the
# file itself. Lets CHANGELOG.md go through two rewrites in sequence.
current() {
	local file="$1" dest
	dest="$(staged_path "$file")"
	if [ -f "$dest" ]; then
		printf '%s' "$dest"
	else
		printf '%s' "$file"
	fi
}

# edit FILE WHAT SED-EXPRESSION... -- stage a sed rewrite.
edit() {
	local file="$1" what="$2"
	shift 2
	if [ ! -f "$file" ]; then
		echo "ERROR: $file is missing; bump-version.sh needs updating. No files were modified." >&2
		exit 1
	fi
	local args=() expr
	for expr in "$@"; do
		args+=(-e "$expr")
	done
	local tmp
	tmp="$(mktemp "$stagedir/tmp.XXXXXX")"
	sed "${args[@]}" "$(current "$file")" >"$tmp"
	require_changed "$file" "$what" "$tmp"
}

echo ">>> Bumping $old -> $new"

edit cmd/root.go "defaultVersion" \
	"s/^const defaultVersion = \"$old\"\$/const defaultVersion = \"$new\"/"

edit cmd/version_test.go "defaultVersion assertion" \
	"s/defaultVersion != \"$old\"/defaultVersion != \"$new\"/" \
	"s/want $old\"/want $new\"/"

edit README.md "download recipe" "s/^VERSION=$old\$/VERSION=$new/"
edit README_zh.md "download recipe" "s/^VERSION=$old\$/VERSION=$new/"

edit docs/ARCHITECTURE.md "document header" \
	"s/^> Version: v$old\$/> Version: v$new/" \
	"s/^> Updated: .*\$/> Updated: $date_today/"
edit docs/ARCHITECTURE_zh.md "document header" \
	"s/^> 版本: v$old\$/> 版本: v$new/" \
	"s/^> 更新日期: .*\$/> 更新日期: $date_today/"

# CHANGELOG: add the stanza above the previous release, then move the compare
# links. Both are skipped by hand-bumps often enough to be worth automating.
tmp="$(mktemp "$stagedir/tmp.XXXXXX")"
awk -v new="$new" -v date="$date_today" '
	/^## \[/ && $0 !~ /^## \[Unreleased\]/ && !inserted {
		print "## [" new "] - " date
		print ""
		print "TODO: describe this release before tagging."
		print ""
		print "---"
		print ""
		inserted = 1
	}
	{ print }
' "$(current CHANGELOG.md)" >"$tmp"
require_changed CHANGELOG.md "new [$new] stanza" "$tmp"

tmp="$(mktemp "$stagedir/tmp.XXXXXX")"
awk -v new="$new" -v old="$old" -v url="$repo_url" '
	$0 ~ "^\\[Unreleased\\]: " {
		print "[Unreleased]: " url "/compare/v" new "...HEAD"
		next
	}
	$0 ~ "^\\[" old "\\]: " && !inserted {
		print "[" new "]: " url "/compare/v" old "...v" new
		inserted = 1
	}
	{ print }
' "$(current CHANGELOG.md)" >"$tmp"
require_changed CHANGELOG.md "compare links" "$tmp"

echo ">>> Writing ${#staged[@]} staged rewrites"
for file in "${staged[@]}"; do
	cat "$(staged_path "$file")" >"$file"
done

echo ">>> Verifying every location agrees on $new"
if command -v go >/dev/null 2>&1; then
	go test ./cmd/ -run 'TestRepoVersionsAreConsistent|TestChangelogLinksNewestRelease|TestDefaultVersionConstant' -count=1
else
	echo "    (go not found -- skipping the consistency test; run 'make test' before tagging)"
fi

cat <<EOF

>>> Bumped to $new. Still to do by hand:
      1. Fill in the [$new] stanza in CHANGELOG.md (it currently says TODO).
      2. Add the release entry to docs/ROADMAP.md and docs/ROADMAP_zh.md.
      3. make check
      4. git commit -am "Release v$new" && git tag v$new && git push --follow-tags
EOF
