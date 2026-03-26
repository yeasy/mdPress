package source

import (
	"strings"
	"testing"
)

// TestIsGitHubURL tests GitHub URL detection (table-driven)
func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantRes bool
	}{
		// Standard formats
		{
			name:    "standard HTTPS URL",
			input:   "https://github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "standard HTTP URL",
			input:   "http://github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "GitHub URL without protocol",
			input:   "github.com/owner/repo",
			wantRes: true,
		},
		// .git suffix
		{
			name:    "with .git suffix",
			input:   "https://github.com/owner/repo.git",
			wantRes: true,
		},
		{
			name:    "with .git suffix (no protocol)",
			input:   "github.com/owner/repo.git",
			wantRes: true,
		},
		// www prefix
		{
			name:    "with www prefix",
			input:   "https://www.github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "www without protocol",
			input:   "www.github.com/owner/repo",
			wantRes: true,
		},
		// URL with path
		{
			name:    "URL with path",
			input:   "https://github.com/owner/repo/tree/main",
			wantRes: true,
		},
		{
			name:    "with issues path",
			input:   "https://github.com/owner/repo/issues/123",
			wantRes: true,
		},
		// Invalid or non-GitHub URLs
		{
			name:    "non-GitHub domain",
			input:   "https://gitlab.com/owner/repo",
			wantRes: false,
		},
		{
			name:    "local path",
			input:   "/home/user/project",
			wantRes: false,
		},
		{
			name:    "relative path",
			input:   "./project",
			wantRes: false,
		},
		{
			name:    "plain text name",
			input:   "owner/repo",
			wantRes: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantRes: false,
		},
		{
			name:    "owner only without repo",
			input:   "https://github.com/owner",
			wantRes: false,
		},
		{
			name:    "Windows path",
			input:   "C:\\Users\\project",
			wantRes: false,
		},
		// Special character tests
		{
			name:    "URL with query params",
			input:   "https://github.com/owner/repo?tab=readme",
			wantRes: false,
		},
		{
			name:    "URL with hash anchor",
			input:   "https://github.com/owner/repo#readme",
			wantRes: false,
		},
		{
			name:    "repo name with hyphens",
			input:   "https://github.com/owner/my-cool-repo",
			wantRes: true,
		},
		{
			name:    "repo name with underscores",
			input:   "https://github.com/owner/my_repo",
			wantRes: true,
		},
		{
			name:    "repo name with numbers",
			input:   "https://github.com/owner/repo123",
			wantRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubURL(tt.input)
			if got != tt.wantRes {
				t.Errorf("isGitHubURL(%q) = %v, want %v", tt.input, got, tt.wantRes)
			}
		})
	}
}

// TestParseGitHubURL tests extracting owner and repo from GitHub URL
func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "standard HTTPS URL",
			input:     "https://github.com/golang/go",
			wantOwner: "golang",
			wantRepo:  "go",
		},
		{
			name:      "HTTP URL",
			input:     "http://github.com/kubernetes/kubernetes",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
		},
		{
			name:      "URL without protocol",
			input:     "github.com/rust-lang/rust",
			wantOwner: "rust-lang",
			wantRepo:  "rust",
		},
		{
			name:      "with .git suffix",
			input:     "https://github.com/torvalds/linux.git",
			wantOwner: "torvalds",
			wantRepo:  "linux",
		},
		{
			name:      "www prefix",
			input:     "https://www.github.com/python/cpython",
			wantOwner: "python",
			wantRepo:  "cpython",
		},
		{
			name:      "with path",
			input:     "https://github.com/nodejs/node/tree/main",
			wantOwner: "nodejs",
			wantRepo:  "node",
		},
		{
			name:      "with query params",
			input:     "https://github.com/django/django?tab=readme",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "with hash anchor",
			input:     "https://github.com/vuejs/vue#readme",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "with hyphens and underscores",
			input:     "https://github.com/my-org/my_repo-v2",
			wantOwner: "my-org",
			wantRepo:  "my_repo-v2",
		},
		{
			name:      "non-GitHub URL",
			input:     "https://gitlab.com/owner/repo",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "invalid format",
			input:     "https://github.com/onlyowner",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "empty string",
			input:     "",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo := parseGitHubURL(tt.input)
			if gotOwner != tt.wantOwner || gotRepo != tt.wantRepo {
				t.Errorf("parseGitHubURL(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotOwner, gotRepo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

// TestDetect tests automatic source type detection
func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     Options
		wantType string
		wantErr  bool
	}{
		// GitHub URL detection
		{
			name:     "GitHub HTTPS URL",
			input:    "https://github.com/golang/go",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL without protocol",
			input:    "github.com/python/cpython",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL with .git",
			input:    "https://github.com/torvalds/linux.git",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL with branch option",
			input:    "https://github.com/nodejs/node",
			opts:     Options{Branch: "main"},
			wantType: "github",
			wantErr:  false,
		},
		// Local path detection
		{
			name:     "local absolute path",
			input:    "/home/user/project",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "local relative path",
			input:    "./project",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "local path (current directory)",
			input:    ".",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "local path with SubDir",
			input:    "/home/user/project",
			opts:     Options{SubDir: "docs"},
			wantType: "local",
			wantErr:  false,
		},
		// Error cases
		{
			name:     "empty string",
			input:    "",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "spaces only",
			input:    "   ",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "tabs and spaces",
			input:    "\t \n",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Detect(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Detect(%q) unexpected error: %v", tt.input, err)
				return
			}

			if src == nil {
				t.Errorf("Detect(%q) returned nil source", tt.input)
				return
			}

			if src.Type() != tt.wantType {
				t.Errorf("Detect(%q).Type() = %q, want %q", tt.input, src.Type(), tt.wantType)
			}
		})
	}
}

// TestDetectEmptyInput tests empty input handling
func TestDetectEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"spaces only", "   "},
		{"tab only", "\t"},
		{"newline only", "\n"},
		{"mixed whitespace", " \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})

			if err == nil {
				t.Errorf("Detect(%q) should return error for empty/whitespace input", tt.input)
			}

			if src != nil {
				t.Errorf("Detect(%q) should return nil source for empty/whitespace input", tt.input)
			}

			// Verify error message contains meaningful content
			if err != nil && !strings.Contains(err.Error(), "empty") {
				t.Errorf("Error message should contain 'empty': %v", err)
			}
		})
	}
}

// TestGitHubURLEdgeCases tests GitHub URL edge cases
func TestGitHubURLEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantRes bool
	}{
		{
			name:    "GitHub URL with trailing slash",
			input:   "https://github.com/owner/repo/",
			wantRes: true,
		},
		{
			name:    "GitHub URL with .git/ trailing",
			input:   "https://github.com/owner/repo.git/",
			wantRes: true,
		},
		{
			name:    "Multiple slashes in path",
			input:   "https://github.com/owner/repo/path/to/file",
			wantRes: true,
		},
		{
			name:    "Empty owner/repo",
			input:   "https://github.com//repo",
			wantRes: false,
		},
		{
			name:    "Only owner",
			input:   "https://github.com/owner/",
			wantRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubURL(tt.input)
			if got != tt.wantRes {
				t.Errorf("isGitHubURL(%q) = %v, want %v", tt.input, got, tt.wantRes)
			}
		})
	}
}

// TestParseGitHubURLVariations tests GitHub URL variation parsing
func TestParseGitHubURLVariations(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "Owner with numbers",
			input:     "https://github.com/user123/repo456",
			wantOwner: "user123",
			wantRepo:  "repo456",
		},
		{
			name:      "Owner and repo with mixed case",
			input:     "https://github.com/MyOrg/MyRepo",
			wantOwner: "MyOrg",
			wantRepo:  "MyRepo",
		},
		{
			name:      "URL with fragment should fail",
			input:     "https://github.com/owner/repo#section",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "Repo name with dots and dashes",
			input:     "https://github.com/owner/my-repo.name",
			wantOwner: "owner",
			wantRepo:  "my-repo.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo := parseGitHubURL(tt.input)
			if gotOwner != tt.wantOwner || gotRepo != tt.wantRepo {
				t.Errorf("parseGitHubURL(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotOwner, gotRepo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

// TestDetectOptions tests source detection with options
func TestDetectOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     Options
		wantType string
	}{
		{
			name:     "Local with branch option (ignored)",
			input:    "/local/path",
			opts:     Options{Branch: "main"},
			wantType: "local",
		},
		{
			name:     "Local with subdir option",
			input:    "/local/path",
			opts:     Options{SubDir: "docs"},
			wantType: "local",
		},
		{
			name:     "GitHub with branch option",
			input:    "https://github.com/owner/repo",
			opts:     Options{Branch: "develop"},
			wantType: "github",
		},
		{
			name:     "GitHub with subdir option",
			input:    "https://github.com/owner/repo",
			opts:     Options{SubDir: "docs"},
			wantType: "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, tt.opts)
			if err != nil {
				t.Errorf("Detect(%q) unexpected error: %v", tt.input, err)
				return
			}

			if src.Type() != tt.wantType {
				t.Errorf("Detect(%q).Type() = %q, want %q", tt.input, src.Type(), tt.wantType)
			}
		})
	}
}

// TestDetectTrimsInput tests that Detect trims input whitespace
func TestDetectTrimsInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
	}{
		{
			name:     "Relative path with leading space",
			input:    "  ./project",
			wantType: "local",
		},
		{
			name:     "GitHub URL with trailing space",
			input:    "https://github.com/owner/repo  ",
			wantType: "github",
		},
		{
			name:     "Path with tabs",
			input:    "\t\t./project\t",
			wantType: "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})
			if err != nil {
				t.Errorf("Detect(%q) unexpected error: %v", tt.input, err)
				return
			}

			if src.Type() != tt.wantType {
				t.Errorf("Detect(%q).Type() = %q, want %q", tt.input, src.Type(), tt.wantType)
			}
		})
	}
}
