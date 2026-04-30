package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNewGitHubSource tests GitHub source creation
func TestNewGitHubSource(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
		opts  Options
	}{
		{
			name:  "basic GitHub source",
			owner: "golang",
			repo:  "go",
			opts:  Options{},
		},
		{
			name:  "with branch option",
			owner: "python",
			repo:  "cpython",
			opts:  Options{Branch: "main"},
		},
		{
			name:  "with subdirectory option",
			owner: "nodejs",
			repo:  "node",
			opts:  Options{SubDir: "docs"},
		},
		{
			name:  "with multiple options",
			owner: "kubernetes",
			repo:  "kubernetes",
			opts:  Options{Branch: "release-1.29", SubDir: "docs"},
		},
		{
			name:  "special characters in owner",
			owner: "my-org",
			repo:  "repo_name",
			opts:  Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGitHubSource(tt.owner, tt.repo, tt.opts)

			if src == nil {
				t.Fatal("NewGitHubSource returned nil")
			}

			if src.owner != tt.owner {
				t.Errorf("owner = %q, want %q", src.owner, tt.owner)
			}

			if src.repo != tt.repo {
				t.Errorf("repo = %q, want %q", src.repo, tt.repo)
			}

			if src.opts != tt.opts {
				t.Errorf("opts = %v, want %v", src.opts, tt.opts)
			}

			// tempDir should be empty initially
			if src.tempDir != "" {
				t.Errorf("tempDir should be empty initially, got %q", src.tempDir)
			}
		})
	}
}

// TestGitHubSourceType tests GitHub source type
func TestGitHubSourceType(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})
	if src.Type() != "github" {
		t.Errorf("Type() = %q, want %q", src.Type(), "github")
	}
}

// TestGitHubSourceRepoName tests repository full name format
func TestGitHubSourceRepoName(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		wantName string
	}{
		{
			name:     "basic format",
			owner:    "golang",
			repo:     "go",
			wantName: "golang/go",
		},
		{
			name:     "special characters",
			owner:    "my-org",
			repo:     "my_repo-v2",
			wantName: "my-org/my_repo-v2",
		},
		{
			name:     "numbers",
			owner:    "org123",
			repo:     "repo456",
			wantName: "org123/repo456",
		},
		{
			name:     "uppercase letters",
			owner:    "MyOrg",
			repo:     "MyRepo",
			wantName: "MyOrg/MyRepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGitHubSource(tt.owner, tt.repo, Options{})
			repoName := src.RepoName()

			if repoName != tt.wantName {
				t.Errorf("RepoName() = %q, want %q", repoName, tt.wantName)
			}
		})
	}
}

// TestGitHubSourceCleanupNoTempDir tests cleanup when tempDir is unset
func TestGitHubSourceCleanupNoTempDir(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})

	// Prepare is not called, so tempDir should be empty
	err := src.Cleanup()

	// Should safely return nil without attempting to remove a non-existent directory
	if err != nil {
		t.Errorf("Cleanup() should return nil when tempDir is empty, got %v", err)
	}

	if src.tempDir != "" {
		t.Errorf("tempDir should remain empty after Cleanup(), got %q", src.tempDir)
	}
}

// TestGitHubSourceCleanupMultipleCalls tests safety of multiple cleanup calls
func TestGitHubSourceCleanupMultipleCalls(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})

	// Create temp directory to simulate Prepare result
	tempDir := t.TempDir()
	src.tempDir = tempDir

	// Verify directory exists
	if _, err := os.Stat(tempDir); err != nil {
		t.Fatalf("Test setup failed: %v", err)
	}

	// First cleanup should succeed
	err := src.Cleanup()
	if err != nil {
		t.Errorf("First Cleanup() failed: %v", err)
	}

	// Verify tempDir was cleared
	if src.tempDir != "" {
		t.Errorf("tempDir should be empty after Cleanup(), got %q", src.tempDir)
	}

	// Second cleanup should safely return nil (directory already removed)
	err = src.Cleanup()
	if err != nil {
		t.Errorf("Second Cleanup() should safely return nil, got %v", err)
	}
}

// TestGitHubSourceFields tests GitHub source field access
func TestGitHubSourceFields(t *testing.T) {
	opts := Options{Branch: "dev", SubDir: "src"}
	src := NewGitHubSource("test-owner", "test-repo", opts)

	// Verify fields via public methods
	if src.Type() != "github" {
		t.Error("Type() failed")
	}

	if src.RepoName() != "test-owner/test-repo" {
		t.Error("RepoName() failed")
	}

	// opts should be saved
	if src.opts.Branch != "dev" {
		t.Errorf("Branch option not saved, got %q", src.opts.Branch)
	}

	if src.opts.SubDir != "src" {
		t.Errorf("SubDir option not saved, got %q", src.opts.SubDir)
	}
}

// TestGitHubSourceCleanupWithInvalidTempDir tests cleanup with invalid temp directory
func TestGitHubSourceCleanupWithInvalidTempDir(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})

	// Set a temp directory path that no longer exists
	src.tempDir = "/nonexistent/temp/dir/that/should/not/exist"

	// Attempt to clean up a non-existent directory
	// Per implementation, os.RemoveAll returns nil for non-existent directories
	err := src.Cleanup()

	// Should return nil (os.RemoveAll behavior) or an appropriate error
	// Here we verify the implementation's behavior is reasonable
	if err != nil {
		t.Errorf("Cleanup() with invalid dir should not error: %v", err)
	}

	// tempDir should be cleared
	if src.tempDir != "" {
		t.Errorf("tempDir should be cleared after Cleanup(), got %q", src.tempDir)
	}
}

// TestGitHubSourceEdgeCases tests edge cases
func TestGitHubSourceEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{
			name:  "empty string owner",
			owner: "",
			repo:  "repo",
		},
		{
			name:  "empty string repo",
			owner: "owner",
			repo:  "",
		},
		{
			name:  "both empty",
			owner: "",
			repo:  "",
		},
		{
			name:  "spaces only",
			owner: "  ",
			repo:  "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NewGitHubSource should accept any string
			// (validation happens in Prepare, not during creation)
			src := NewGitHubSource(tt.owner, tt.repo, Options{})
			if src == nil {
				t.Fatal("NewGitHubSource should not return nil for any input")
			}

			repoName := src.RepoName()
			if repoName != tt.owner+"/"+tt.repo {
				t.Errorf("RepoName() = %q, want %q", repoName, tt.owner+"/"+tt.repo)
			}
		})
	}
}

// TestGitHubSourceTokenHintOnCloneFailure verifies that the error message
// suggests GITHUB_TOKEN when the token is not set and clone fails.
func TestGitHubSourceTokenHintOnCloneFailure(t *testing.T) {
	// Ensure GITHUB_TOKEN is not set for this test.
	t.Setenv("GITHUB_TOKEN", "")

	// Use a non-existent repo — the clone will fail because the repo
	// does not exist. The error should suggest setting GITHUB_TOKEN.
	src := NewGitHubSource("nonexistent-test-owner", "nonexistent-test-repo", Options{})

	_, err := src.Prepare()
	if err == nil {
		// If git somehow succeeds (unlikely), clean up and skip.
		src.Cleanup() //nolint:errcheck
		t.Skip("clone unexpectedly succeeded")
	}
	// The error should contain the GITHUB_TOKEN hint since it's not set.
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error should mention GITHUB_TOKEN hint, got: %v", err)
	}
}

// TestGitHubSourceTokenNotLeakedInRepoName verifies that RepoName()
// never includes the token.
func TestGitHubSourceTokenNotLeakedInRepoName(t *testing.T) {
	src := NewGitHubSource("myorg", "private-repo", Options{})
	name := src.RepoName()
	if name != "myorg/private-repo" {
		t.Errorf("RepoName() = %q, want %q", name, "myorg/private-repo")
	}
}

// TestSafeNameRegex tests the regex that validates owner and repo names.
func TestSafeNameRegex(t *testing.T) {
	valid := []string{"golang", "my-org", "repo_name", "repo.v2", "123"}
	for _, v := range valid {
		if !safeNameRegex.MatchString(v) {
			t.Errorf("safeNameRegex should accept %q", v)
		}
	}

	invalid := []string{
		"",
		"; rm -rf /",
		"$(whoami)",
		"`id`",
		"owner/repo",
		"name with spaces",
		"--upload-pack=evil",
		"hello\nworld",
		"../../../etc/passwd",
	}
	for _, v := range invalid {
		if safeNameRegex.MatchString(v) {
			t.Errorf("safeNameRegex should reject %q", v)
		}
	}
}

// TestBranchRegex tests the regex that validates branch names.
func TestBranchRegex(t *testing.T) {
	valid := []string{"main", "release-1.29", "feature/foo", "v2.0.0", "dev_branch"}
	for _, v := range valid {
		if !branchRegex.MatchString(v) {
			t.Errorf("branchRegex should accept %q", v)
		}
	}

	invalid := []string{
		"",
		"--upload-pack=evil",
		"-flag",
		".hidden",
		"; echo pwned",
		"$(cmd)",
		"branch with spaces",
	}
	for _, v := range invalid {
		if branchRegex.MatchString(v) {
			t.Errorf("branchRegex should reject %q", v)
		}
	}
}

// TestGitHubSourcePrepareValidation tests that Prepare rejects invalid inputs
// without actually running git. This test requires git to be installed.
func TestGitHubSourcePrepareValidation(t *testing.T) {
	if _, err := os.Stat("/usr/bin/git"); err != nil {
		// Also check PATH
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("git not installed, skipping Prepare validation tests")
		}
	}

	tests := []struct {
		name    string
		owner   string
		repo    string
		branch  string
		subDir  string
		wantErr string
	}{
		{
			name:    "shell injection in owner",
			owner:   "; cat /etc/passwd",
			repo:    "repo",
			wantErr: "invalid repository owner",
		},
		{
			name:    "command substitution in repo",
			owner:   "owner",
			repo:    "$(whoami)",
			wantErr: "invalid repository name",
		},
		{
			name:    "git option injection in branch",
			owner:   "owner",
			repo:    "repo",
			branch:  "--upload-pack=evil",
			wantErr: "invalid branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGitHubSource(tt.owner, tt.repo, Options{
				Branch: tt.branch,
				SubDir: tt.subDir,
			})
			_, err := src.Prepare()
			if err == nil {
				t.Fatalf("Prepare() should fail for %s, got nil error", tt.name)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error should contain %q, got: %v", tt.wantErr, err)
			}
			// Verify no temp directory is left behind
			if src.tempDir != "" {
				t.Errorf("tempDir should be empty after validation failure, got %q", src.tempDir)
				os.RemoveAll(src.tempDir)
			}
		})
	}
}

// TestGitHubSourceSubDirValidation tests the subdirectory path traversal
// prevention in Prepare(). It simulates a cloned repo by pre-creating a temp
// directory and setting tempDir, then verifying that unsafe SubDir values
// are rejected.
func TestGitHubSourceSubDirValidation(t *testing.T) {
	tests := []struct {
		name    string
		subDir  string
		setup   func(t *testing.T, baseDir string) // optional: create subdirectory
		wantErr string
	}{
		{
			name:    "path traversal with ..",
			subDir:  "../../../etc/passwd",
			wantErr: "unsafe subdirectory path",
		},
		{
			name: "absolute path",
			subDir: func() string {
				if runtime.GOOS == "windows" {
					return `C:\Windows\System32`
				}
				return "/etc/passwd"
			}(),
			wantErr: "unsafe subdirectory path",
		},
		{
			name:    "relative path traversal",
			subDir:  "docs/../../..",
			wantErr: "unsafe subdirectory path",
		},
		{
			name:    "non-existent subdirectory",
			subDir:  "nonexistent",
			wantErr: "requested subdirectory does not exist",
		},
		{
			name:   "file instead of directory",
			subDir: "afile.txt",
			setup: func(t *testing.T, baseDir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(baseDir, "afile.txt"), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "requested subdirectory is not a directory",
		},
		{
			name:   "valid subdirectory",
			subDir: "docs",
			setup: func(t *testing.T, baseDir string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(baseDir, "docs"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "", // should succeed
		},
		{
			name:   "symlink escape",
			subDir: "escape",
			setup: func(t *testing.T, baseDir string) {
				t.Helper()
				if err := os.Symlink(os.TempDir(), filepath.Join(baseDir, "escape")); err != nil {
					t.Skip("cannot create symlinks on this OS")
				}
			},
			wantErr: "subdirectory escapes repository root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake "cloned repo" directory.
			baseDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(t, baseDir)
			}

			// Build a GitHubSource and inject the temp dir to skip cloning.
			// We use an invalid owner so that if our tempDir injection fails
			// the test will error at clone instead of silently passing.
			src := &GitHubSource{
				owner:   "test",
				repo:    "test",
				opts:    Options{SubDir: tt.subDir},
				tempDir: baseDir,
			}

			// Call the SubDir validation path directly by simulating post-clone.
			dir, err := src.validateSubDir()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				wantDir := filepath.Join(baseDir, tt.subDir)
				if evaled, evalErr := filepath.EvalSymlinks(wantDir); evalErr == nil {
					wantDir = evaled
				}
				if dir != wantDir {
					t.Errorf("dir = %q, want %q", dir, wantDir)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
		token string
		want  string
	}{
		{"empty token", "some output", "", "some output"},
		{"empty string", "", "secret", ""},
		{"no match", "clean output", "secret", "clean output"},
		{"single match", "https://secret@github.com/repo", "secret", "https://[REDACTED]@github.com/repo"},
		{"multiple matches", "secret and secret again", "secret", "[REDACTED] and [REDACTED] again"},
		{"token is entire string", "secret", "secret", "[REDACTED]"},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactToken(tt.input, tt.token)
			if got != tt.want {
				t.Errorf("redactToken(%q, %q) = %q, want %q", tt.input, tt.token, got, tt.want)
			}
		})
	}
}
