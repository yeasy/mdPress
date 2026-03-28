// boundary_test.go Boundary tests for the source package.
// Covers invalid URLs, permission issues, special paths, and other edge cases.
package source

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDetect_InvalidGitHubURLs tests error handling for invalid GitHub URLs.
func TestDetect_InvalidGitHubURLs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLocal bool // true if URL should fall through to LocalSource
		wantError bool // true if URL should produce an error
	}{
		// These GitHub-like URLs are not matched by the regex pattern and
		// fall through to LocalSource rather than returning an error.
		{"domain only", "https://github.com", true, false},
		{"owner only", "https://github.com/owner", true, false},
		{"with query params", "https://github.com/owner/repo?tab=readme", true, false},
		{"with anchor", "https://github.com/owner/repo#readme", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})
			if tt.wantError {
				if err == nil {
					t.Errorf("Detect(%q) should return an error, got source %T", tt.input, src)
				}
				return
			}
			if err != nil {
				t.Errorf("Detect(%q) returned unexpected error: %v", tt.input, err)
				return
			}
			if src == nil {
				t.Fatalf("Detect(%q) returned nil source with no error", tt.input)
			}
			_, isLocal := src.(*LocalSource)
			if tt.wantLocal && !isLocal {
				t.Errorf("Detect(%q) should fall through to LocalSource, got %T", tt.input, src)
			}
			if !tt.wantLocal && isLocal {
				t.Errorf("Detect(%q) should be recognized as GitHub URL, got LocalSource", tt.input)
			}
		})
	}
}

// TestLocalSource_PermissionDenied tests handling of directories without read permission.
func TestLocalSource_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-permission semantics required; skipping on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("cannot test permission restrictions as root")
	}

	// Create directory without read permission
	tempDir := t.TempDir()
	noReadDir := filepath.Join(tempDir, "noperm")
	if err := os.MkdirAll(noReadDir, 0000); err != nil {
		t.Fatalf("failed to create no-permission directory: %v", err)
	}
	defer func() {
		if err := os.Chmod(noReadDir, 0755); err != nil {
			t.Logf("failed to restore directory permissions: %v", err)
		}
	}()

	src := NewLocalSource(noReadDir, Options{})
	result, err := src.Prepare()

	// Prepare should either succeed (returns path) or return an error.
	// Must not panic or return both error and non-empty result.
	if err != nil && result != "" {
		t.Errorf("Prepare returned both error and result: result=%q, err=%v", result, err)
	}
}

// TestLocalSource_SymlinkPath tests symlink path handling.
func TestLocalSource_SymlinkPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create actual directory
	realDir := filepath.Join(tempDir, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("failed to create real directory: %v", err)
	}

	// Create symlink
	linkDir := filepath.Join(tempDir, "link")
	err := os.Symlink(realDir, linkDir)
	if err != nil {
		t.Skip("cannot create symlink")
	}

	src := NewLocalSource(linkDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Fatalf("symlink path should work: %v", err)
	}
	if result == "" {
		t.Error("symlink path should return non-empty result")
	}
}

// TestLocalSource_SubDirPermission tests subdirectory permission issues.
func TestLocalSource_SubDirPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-permission semantics required; skipping on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("cannot test permission restrictions as root")
	}

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "restricted")
	if err := os.MkdirAll(subDir, 0000); err != nil {
		t.Fatalf("failed to create restricted subdirectory: %v", err)
	}
	defer func() {
		if err := os.Chmod(subDir, 0755); err != nil {
			t.Logf("failed to restore subdirectory permissions: %v", err)
		}
	}()

	src := NewLocalSource(tempDir, Options{SubDir: "restricted"})
	result, err := src.Prepare()

	// Prepare should either succeed or return an error, not both.
	if err != nil && result != "" {
		t.Errorf("Prepare returned both error and result: result=%q, err=%v", result, err)
	}
}

// TestLocalSource_DeepNestedPath tests deeply nested directory paths.
func TestLocalSource_DeepNestedPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create deeply nested directory
	deepPath := filepath.Join(tempDir, "a", "b", "c", "d", "e", "f")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("failed to create deeply nested directory: %v", err)
	}

	src := NewLocalSource(tempDir, Options{SubDir: "a/b/c/d/e/f"})
	result, err := src.Prepare()
	if err != nil {
		t.Fatalf("deeply nested directory should work: %v", err)
	}
	if !strings.HasSuffix(result, filepath.Join("a", "b", "c", "d", "e", "f")) {
		t.Errorf("path should contain full nested path, got: %q", result)
	}
}

// TestLocalSource_SpecialCharsInPath tests paths containing special characters.
func TestLocalSource_SpecialCharsInPath(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory with spaces and CJK characters
	specialDir := filepath.Join(tempDir, "my docs")
	err := os.MkdirAll(specialDir, 0755)
	if err != nil {
		t.Skip("cannot create directory with special characters")
	}

	src := NewLocalSource(specialDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Fatalf("special character path should work: %v", err)
	}
	if result == "" {
		t.Error("should return non-empty path")
	}
}

// TestLocalSource_EmptyDirectory tests using an empty directory as source.
func TestLocalSource_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("failed to create empty directory: %v", err)
	}

	src := NewLocalSource(emptyDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Fatalf("empty directory should work as source: %v", err)
	}
	if result == "" {
		t.Error("empty directory should return a path")
	}
}

// TestGitHubSource_InvalidOwnerRepo tests invalid owner/repo combinations.
func TestGitHubSource_InvalidOwnerRepo(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{"empty owner", "", "repo"},
		{"empty repo", "owner", ""},
		{"both empty", "", ""},
		{"with spaces", "owner name", "repo name"},
		{"with special chars", "owner@", "repo!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGitHubSource(tt.owner, tt.repo, Options{})
			// Creation should not fail (validation deferred to Prepare)
			if src == nil {
				t.Fatal("NewGitHubSource should not return nil")
			}
			if src.Type() != "github" {
				t.Error("type should be github")
			}
		})
	}
}

// TestDetect_WhitespaceVariations tests various whitespace inputs.
func TestDetect_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"single space", " ", true},
		{"multiple spaces", "     ", true},
		{"tab", "\t", true},
		{"newline", "\n", true},
		{"mixed whitespace", " \t\n\r ", true},
		{"path with surrounding spaces", "  /tmp/test  ", false}, // should be trimmed and handled
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})
			if tt.wantErr {
				if err == nil {
					t.Errorf("Detect(%q) should return error", tt.input)
				}
				if src != nil {
					t.Errorf("Detect(%q) source should be nil on error", tt.input)
				}
			} else if err != nil {
				t.Errorf("Detect(%q) should not return error: %v", tt.input, err)
			}
		})
	}
}

// TestDetect_OptionsPassthrough tests that options are correctly passed to created sources.
func TestDetect_OptionsPassthrough(t *testing.T) {
	opts := Options{
		Branch: "develop",
		SubDir: "docs/zh",
	}

	// GitHub source
	src, err := Detect("https://github.com/test/repo", opts)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	ghSrc, ok := src.(*GitHubSource)
	if !ok {
		t.Fatal("should return GitHubSource")
	}
	if ghSrc.opts.Branch != "develop" {
		t.Errorf("Branch should be 'develop', got %q", ghSrc.opts.Branch)
	}
	if ghSrc.opts.SubDir != "docs/zh" {
		t.Errorf("SubDir should be 'docs/zh', got %q", ghSrc.opts.SubDir)
	}

	// Local source
	src2, err := Detect(t.TempDir(), opts)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	localSrc, ok := src2.(*LocalSource)
	if !ok {
		t.Fatal("should return LocalSource")
	}
	if localSrc.opts.SubDir != "docs/zh" {
		t.Errorf("SubDir should be 'docs/zh', got %q", localSrc.opts.SubDir)
	}
}

// TestLocalSource_PathTraversal tests path traversal attack scenarios.
func TestLocalSource_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "docs"), 0755); err != nil {
		t.Fatalf("failed to create docs directory: %v", err)
	}

	// Subdirectory uses .. to attempt path traversal
	src := NewLocalSource(tempDir, Options{SubDir: "../../../etc"})
	result, err := src.Prepare()
	if err == nil && result != "" {
		// If Prepare succeeded, verify the resolved path stays within tempDir.
		absTemp, absErr := filepath.Abs(tempDir)
		if absErr != nil {
			t.Fatalf("filepath.Abs(tempDir) failed: %v", absErr)
		}
		absResult, absErr := filepath.Abs(result)
		if absErr != nil {
			t.Fatalf("filepath.Abs(result) failed: %v", absErr)
		}
		if !strings.HasPrefix(absResult, absTemp+string(filepath.Separator)) && absResult != absTemp {
			t.Errorf("path traversal not blocked: SubDir=%q resolved to %q (outside %q)", "../../../etc", absResult, absTemp)
		}
	}
}

// TestGitHubURL_EdgeCases tests edge cases of GitHub URL regex matching.
func TestGitHubURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMatch bool
	}{
		{"uppercase GITHUB", "https://GITHUB.COM/owner/repo", false}, // regex is case-sensitive
		{"trailing slash", "https://github.com/owner/repo/", true},
		{"deep path", "https://github.com/owner/repo/tree/main/docs", true},
		{"github.io", "https://github.io/owner/repo", false},
		{"enterprise GitHub", "https://github.example.com/owner/repo", false},
		{"bare github.com", "github.com", false},
		{"github.com/", "github.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubURL(tt.input)
			if got != tt.wantMatch {
				t.Errorf("isGitHubURL(%q) = %v, want %v", tt.input, got, tt.wantMatch)
			}
		})
	}
}

// TestLocalSource_ConcurrentPrepare tests concurrent Prepare calls are safe.
func TestLocalSource_ConcurrentPrepare(t *testing.T) {
	tempDir := t.TempDir()
	src := NewLocalSource(tempDir, Options{})

	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := src.Prepare()
			errs <- err
		}()
	}
	for i := 0; i < 10; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent Prepare() failed: %v", err)
		}
	}
}

// TestGitHubSource_RepeatedCleanup tests that multiple Cleanup calls are safe.
func TestGitHubSource_RepeatedCleanup(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})

	// GitHubSource need not be concurrency-safe; verify repeated calls don't panic
	for i := 0; i < 10; i++ {
		if err := src.Cleanup(); err != nil {
			t.Fatalf("Cleanup call %d failed: %v", i+1, err)
		}
	}
}
