// version_consistency_test.go checks that every hand-maintained copy of the
// release version agrees with cmd.defaultVersion.
//
// A bump used to be nine hand-edited files with no gate: the only check
// (TestDefaultVersionConstant) compares one literal against another literal in
// the same package, so a stale README download recipe or a missing CHANGELOG
// stanza could not fail a build. Two of the last three releases shipped a wrong
// version string from exactly this class of omission. scripts/bump-version.sh
// rewrites all of these from one argument; this test is what catches a bump
// that was done by hand and missed one.
package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// versionLocation is one file that must spell out the current release version,
// together with the pattern that captures it.
type versionLocation struct {
	path    string
	pattern *regexp.Regexp
	what    string
}

func repoVersionLocations() []versionLocation {
	return []versionLocation{
		{
			path:    "CHANGELOG.md",
			pattern: regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+[^\]]*)\] - `),
			what:    "newest release heading",
		},
		{
			path:    "README.md",
			pattern: regexp.MustCompile(`(?m)^VERSION=(\S+)`),
			what:    "download recipe",
		},
		{
			path:    "README_zh.md",
			pattern: regexp.MustCompile(`(?m)^VERSION=(\S+)`),
			what:    "download recipe",
		},
		{
			path:    "docs/ARCHITECTURE.md",
			pattern: regexp.MustCompile(`(?m)^> Version: v(\S+)`),
			what:    "document header",
		},
		{
			path:    "docs/ARCHITECTURE_zh.md",
			pattern: regexp.MustCompile(`(?m)^> 版本: v(\S+)`),
			what:    "document header",
		},
	}
}

// TestRepoVersionsAreConsistent fails when a version bump touched some files but
// not others.
func TestRepoVersionsAreConsistent(t *testing.T) {
	for _, loc := range repoVersionLocations() {
		t.Run(loc.path, func(t *testing.T) {
			path := filepath.Join("..", filepath.FromSlash(loc.path))
			content, err := os.ReadFile(path) //nolint:gosec // fixed repo-relative path
			if err != nil {
				t.Fatalf("cannot read %s: %v", loc.path, err)
			}
			match := loc.pattern.FindSubmatch(content)
			if match == nil {
				t.Fatalf("%s: no version found for %s (pattern %s) -- did the file change shape? "+
					"scripts/bump-version.sh rewrites this location and must be updated too",
					loc.path, loc.what, loc.pattern)
			}
			if got := string(match[1]); got != defaultVersion {
				t.Errorf("%s: %s says %q, but cmd/root.go defaultVersion is %q -- "+
					"run 'make bump VERSION=%s' instead of editing versions by hand",
					loc.path, loc.what, got, defaultVersion, defaultVersion)
			}
		})
	}
}

// TestChangelogLinksNewestRelease guards the compare links at the bottom of
// CHANGELOG.md, which are easy to forget when a stanza is added by hand.
func TestChangelogLinksNewestRelease(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("cannot read CHANGELOG.md: %v", err)
	}

	linkRef := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(defaultVersion) + `\]: https://`)
	if !linkRef.Match(content) {
		t.Errorf("CHANGELOG.md has no [%s]: compare link", defaultVersion)
	}

	unreleased := regexp.MustCompile(`(?m)^\[Unreleased\]: \S+/compare/v(\S+)\.\.\.HEAD`)
	match := unreleased.FindSubmatch(content)
	if match == nil {
		t.Fatal("CHANGELOG.md has no [Unreleased]: compare link")
	}
	if got := string(match[1]); got != defaultVersion {
		t.Errorf("CHANGELOG.md [Unreleased] compares against v%s, want v%s", got, defaultVersion)
	}
}
