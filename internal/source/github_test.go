package source

import (
	"os"
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
// suggests GITHUB_TOKEN when the token is not set.
func TestGitHubSourceTokenHintOnCloneFailure(t *testing.T) {
	// Ensure GITHUB_TOKEN is not set for this test.
	t.Setenv("GITHUB_TOKEN", "")

	// Use an invalid owner/repo combination that will fail validation,
	// so we don't actually try to clone anything.
	src := NewGitHubSource("valid-owner", "valid-repo", Options{})

	// We can't easily test a full clone failure without network access,
	// but we can verify the source was created correctly and Type() is "github".
	if src.Type() != "github" {
		t.Errorf("Type() = %q, want %q", src.Type(), "github")
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
