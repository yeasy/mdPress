// Package source resolves and prepares book content sources.
// mdpress can read from local directories and GitHub repositories through the Source interface.
package source

import (
	"fmt"
	"regexp"
	"strings"
)

// Source is the shared interface implemented by all source providers.
type Source interface {
	// Prepare returns a local readable directory for the source content.
	Prepare() (string, error)

	// Cleanup releases temporary resources such as cloned repositories.
	Cleanup() error

	// Type returns the source type identifier, for example "local" or "github".
	Type() string
}

// Options configures source resolution.
type Options struct {
	Branch string // Branch override for remote repository sources.
	SubDir string // Subdirectory to use inside the source.
}

// githubURLPattern matches GitHub repository URLs.
var githubURLPattern = regexp.MustCompile(
	`^(?:https?://)?(?:www\.)?github\.com/([^/]+)/([^/\s?#]+)(?:\.git)?(?:/.*)?$`,
)

// Detect infers a source type from user input.
func Detect(input string, opts Options) (Source, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("input path cannot be empty")
	}

	// Handle GitHub repository URLs first.
	if isGitHubURL(input) {
		owner, repo := parseGitHubURL(input)
		if owner == "" || repo == "" {
			return nil, fmt.Errorf("failed to parse GitHub repository URL: %s", input)
		}
		return NewGitHubSource(owner, repo, opts), nil
	}

	// Everything else is treated as a local path.
	return NewLocalSource(input, opts), nil
}

// isGitHubURL reports whether the input looks like a GitHub repository URL.
func isGitHubURL(input string) bool {
	return githubURLPattern.MatchString(input)
}

// parseGitHubURL extracts owner and repo from a GitHub URL.
func parseGitHubURL(input string) (owner, repo string) {
	matches := githubURLPattern.FindStringSubmatch(input)
	if len(matches) < 3 {
		return "", ""
	}
	owner = matches[1]
	repo = strings.TrimSuffix(matches[2], ".git")
	return owner, repo
}
