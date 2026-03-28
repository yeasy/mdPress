// github.go implements GitHub repository sources.
package source

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// gitCloneTimeout is the maximum time allowed for a git clone operation.
	gitCloneTimeout = 5 * time.Minute
)

// Pre-compiled regexps for input validation.
var (
	safeNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	branchRegex   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
)

// GitHubSource clones content from a GitHub repository.
type GitHubSource struct {
	owner   string // Repository owner.
	repo    string // Repository name.
	opts    Options
	tempDir string // Temporary clone directory.
}

// NewGitHubSource creates a GitHub source.
func NewGitHubSource(owner, repo string, opts Options) *GitHubSource {
	return &GitHubSource{
		owner: owner,
		repo:  repo,
		opts:  opts,
	}
}

// Prepare clones the GitHub repository into a temporary directory.
func (s *GitHubSource) Prepare() (string, error) {
	// Ensure git is installed.
	if _, err := exec.LookPath("git"); err != nil {
		return "", errors.New("git command not found; please install git first")
	}

	// Create the temporary directory.
	tempDir, err := os.MkdirTemp("", "mdpress-github-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	s.tempDir = tempDir

	// Validate owner and repo names to avoid command injection.
	if !safeNameRegex.MatchString(s.owner) {
		os.RemoveAll(s.tempDir)
		s.tempDir = ""
		return "", fmt.Errorf("invalid repository owner: %q", s.owner)
	}
	if !safeNameRegex.MatchString(s.repo) {
		os.RemoveAll(s.tempDir)
		s.tempDir = ""
		return "", fmt.Errorf("invalid repository name: %q", s.repo)
	}

	// Build the clone URL.
	// When GITHUB_TOKEN is set, embed it in the URL for authenticated access
	// to private repositories. The token is never logged.
	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", s.owner, s.repo)
	logURL := cloneURL // safe URL for logging (no token)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneURL = fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, s.owner, s.repo)
		slog.Info("Cloning GitHub repository (authenticated)", slog.String("url", logURL))
	} else {
		slog.Info("Cloning GitHub repository", slog.String("url", logURL))
	}

	// Build the git clone command.
	args := []string{"clone", "--depth", "1"}
	if s.opts.Branch != "" {
		// Validate the branch name to avoid command injection.
		if !branchRegex.MatchString(s.opts.Branch) {
			os.RemoveAll(s.tempDir)
			s.tempDir = ""
			return "", fmt.Errorf("invalid branch name: %q", s.opts.Branch)
		}
		args = append(args, "--branch", s.opts.Branch)
	}
	args = append(args, cloneURL, tempDir)

	// Create a context with a timeout for git clone.
	ctx, cancel := context.WithTimeout(context.Background(), gitCloneTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Clean up the clone directory on failure.
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			slog.Warn("Failed to clean up temporary directory after clone failure", slog.String("dir", tempDir), slog.String("error", rmErr.Error()))
		}
		s.tempDir = ""
		hint := ""
		if os.Getenv("GITHUB_TOKEN") == "" {
			hint = "; for private repositories, set the GITHUB_TOKEN environment variable"
		}
		return "", fmt.Errorf("failed to clone repository: %w (URL: %s%s)", err, logURL, hint)
	}

	slog.Info("Repository clone completed", slog.String("dir", tempDir))

	// Check for Git LFS usage and warn if LFS files may not be fully fetched.
	gitattrsPath := filepath.Join(tempDir, ".gitattributes")
	if fi, statErr := os.Stat(gitattrsPath); statErr == nil && fi.Size() < 1<<20 {
		data, err := os.ReadFile(gitattrsPath)
		if err == nil && strings.Contains(string(data), "filter=lfs") {
			slog.Warn("Repository uses Git LFS. Large files (images, binaries) may not be fully fetched with shallow clone. " +
				"If images are missing, clone the repository locally with 'git lfs pull' and use a local path instead.")
		}
	}

	// Apply an optional subdirectory.
	targetDir := tempDir
	if s.opts.SubDir != "" {
		// Prevent path traversal through the subdirectory.
		cleanSubDir := filepath.Clean(s.opts.SubDir)
		if filepath.IsAbs(cleanSubDir) || strings.HasPrefix(cleanSubDir, "..") {
			if rmErr := os.RemoveAll(tempDir); rmErr != nil {
				slog.Warn("Failed to clean up temporary directory", slog.String("dir", tempDir), slog.String("error", rmErr.Error()))
			}
			s.tempDir = ""
			return "", fmt.Errorf("unsafe subdirectory path: %q", s.opts.SubDir)
		}
		targetDir = filepath.Join(tempDir, cleanSubDir)
		info, err := os.Stat(targetDir)
		if err != nil {
			if rmErr := os.RemoveAll(tempDir); rmErr != nil {
				slog.Warn("Failed to clean up temporary directory", slog.String("dir", tempDir), slog.String("error", rmErr.Error()))
			}
			s.tempDir = ""
			return "", fmt.Errorf("requested subdirectory does not exist in the repository: %s", s.opts.SubDir)
		}
		if !info.IsDir() {
			if rmErr := os.RemoveAll(tempDir); rmErr != nil {
				slog.Warn("Failed to clean up temporary directory", slog.String("dir", tempDir), slog.String("error", rmErr.Error()))
			}
			s.tempDir = ""
			return "", fmt.Errorf("requested subdirectory is not a directory: %s", s.opts.SubDir)
		}
	}

	return targetDir, nil
}

// Cleanup removes the temporary clone directory.
func (s *GitHubSource) Cleanup() error {
	if s.tempDir != "" {
		slog.Debug("Cleaning temporary directory", slog.String("dir", s.tempDir))
		if err := os.RemoveAll(s.tempDir); err != nil {
			return fmt.Errorf("failed to clean temporary directory: %w", err)
		}
		s.tempDir = ""
	}
	return nil
}

// Type returns the source type.
func (s *GitHubSource) Type() string {
	return "github"
}

// RepoName returns the full repository name.
func (s *GitHubSource) RepoName() string {
	return s.owner + "/" + s.repo
}
