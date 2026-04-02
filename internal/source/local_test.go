package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewLocalSource tests local source creation
func TestNewLocalSource(t *testing.T) {
	tests := []struct {
		name string
		path string
		opts Options
	}{
		{
			name: "simple path",
			path: "/home/user/project",
			opts: Options{},
		},
		{
			name: "path with subdirectory option",
			path: "/home/user/project",
			opts: Options{SubDir: "docs"},
		},
		{
			name: "relative path",
			path: "./project",
			opts: Options{},
		},
		{
			name: "current directory",
			path: ".",
			opts: Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewLocalSource(tt.path, tt.opts)
			if src == nil {
				t.Fatal("NewLocalSource returned nil")
				return
			}

			if src.path != tt.path {
				t.Errorf("path = %q, want %q", src.path, tt.path)
			}

			if src.opts != tt.opts {
				t.Errorf("opts = %v, want %v", src.opts, tt.opts)
			}
		})
	}
}

// TestLocalSourceType tests local source type
func TestLocalSourceType(t *testing.T) {
	src := NewLocalSource(t.TempDir(), Options{})
	if src.Type() != "local" {
		t.Errorf("Type() = %q, want %q", src.Type(), "local")
	}
}

// TestLocalSourceCleanup tests local source cleanup (should safely return nil)
func TestLocalSourceCleanup(t *testing.T) {
	src := NewLocalSource(t.TempDir(), Options{})
	err := src.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() should return nil for local source, got %v", err)
	}
}

// TestLocalSourcePrepare tests local source Prepare function
func TestLocalSourcePrepare(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string // returns the path to use
		opts    Options
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		// Success cases
		{
			name: "existing directory",
			setup: func() string {
				return tempDir
			},
			opts:    Options{},
			wantErr: false,
			check: func(t *testing.T, result string) {
				if result == "" {
					t.Error("Prepare should return non-empty path for existing dir")
				}
				// Should return an absolute path
				if !filepath.IsAbs(result) {
					t.Errorf("path should be absolute, got %q", result)
				}
			},
		},
		{
			name: "existing directory with relative path",
			setup: func() string {
				// Create temp subdirectory
				subDir := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(subDir, 0o755); err != nil {
					t.Fatalf("mkdir subdir failed: %v", err)
				}
				return subDir
			},
			opts:    Options{},
			wantErr: false,
			check: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("path should be absolute, got %q", result)
				}
			},
		},
		// Error cases
		{
			name: "non-existent path",
			setup: func() string {
				return "/nonexistent/path/that/should/not/exist"
			},
			opts:    Options{},
			wantErr: true,
			check:   nil,
		},
		{
			name: "file instead of directory",
			setup: func() string {
				filePath := filepath.Join(tempDir, "testfile.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
					t.Fatalf("write test file failed: %v", err)
				}
				return filePath
			},
			opts:    Options{},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			src := NewLocalSource(path, tt.opts)

			result, err := src.Prepare()

			if tt.wantErr {
				if err == nil {
					t.Error("Prepare() should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("Prepare() unexpected error: %v", err)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestLocalSourcePrepareWithSubDir tests local source Prepare with subdirectory option
func TestLocalSourcePrepareWithSubDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func() (baseDir, subDir string) // returns base path and subdirectory
		opts      Options
		wantErr   bool
		checkPath func(t *testing.T, path string, subDir string)
	}{
		{
			name: "existing subdirectory",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "docs"
				if err := os.MkdirAll(filepath.Join(baseDir, subDir), 0o755); err != nil {
					t.Fatalf("mkdir docs failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:    Options{SubDir: "docs"},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				if !filepath.IsAbs(path) {
					t.Errorf("path should be absolute, got %q", path)
				}
				if !strings.HasSuffix(path, subDir) {
					t.Errorf("path should end with %q, got %q", subDir, path)
				}
			},
		},
		{
			name: "non-existent subdirectory",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "nonexistent_subdir"
				return baseDir, subDir
			},
			opts:      Options{SubDir: "nonexistent_subdir"},
			wantErr:   true,
			checkPath: nil,
		},
		{
			name: "subdirectory is a file instead of directory",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "file.txt"
				if err := os.WriteFile(filepath.Join(baseDir, subDir), []byte("test"), 0o644); err != nil {
					t.Fatalf("write file.txt failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:      Options{SubDir: "file.txt"},
			wantErr:   true,
			checkPath: nil,
		},
		{
			name: "nested subdirectory",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "a/b/c"
				if err := os.MkdirAll(filepath.Join(baseDir, subDir), 0o755); err != nil {
					t.Fatalf("mkdir nested subdir failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:    Options{SubDir: "a/b/c"},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				if !strings.HasSuffix(path, filepath.Join("a", "b", "c")) {
					t.Errorf("path should contain nested dir, got %q", path)
				}
			},
		},
		{
			name: "empty subdirectory name (should be ignored)",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = ""
				return baseDir, subDir
			},
			opts:    Options{SubDir: ""},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				// Empty subdirectory name should return the base directory itself
				absPath, _ := filepath.Abs(tempDir)
				if path != absPath {
					t.Errorf("path with empty SubDir should be base path, got %q, want %q", path, absPath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir, subDir := tt.setup()
			src := NewLocalSource(baseDir, tt.opts)

			result, err := src.Prepare()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Prepare() should return error for %s", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Prepare() unexpected error: %v", err)
				return
			}

			if tt.checkPath != nil {
				tt.checkPath(t, result, subDir)
			}
		})
	}
}
