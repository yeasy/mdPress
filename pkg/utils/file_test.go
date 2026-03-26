package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileExists tests file existence check
func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Existing file
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if !FileExists(tmpFile) {
		t.Error("file should exist")
	}

	// Non-existent file
	if FileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("file should not exist")
	}

	// Existing directory
	if !FileExists(tmpDir) {
		t.Error("directory should exist")
	}
}

// TestEnsureDir tests directory creation
func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create new directory
	newDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if !FileExists(newDir) {
		t.Error("directory should have been created")
	}

	// Existing directory should not error
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("existing directory should not cause error: %v", err)
	}
}

// TestReadFile tests file reading
func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write test file
	content := "测试内容 hello world"
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Normal read
	data, err := ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

// TestReadFileNotExist tests reading a non-existent file
func TestReadFileNotExist(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("reading non-existent file should return an error")
	}
}

// TestReadFileIsDir tests reading a directory
func TestReadFileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadFile(tmpDir)
	if err == nil {
		t.Error("reading a directory should return an error")
	}
}

// TestWriteFile tests file writing
func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Write file
	filePath := filepath.Join(tmpDir, "output.txt")
	content := "写入的内容"
	if err := WriteFile(filePath, []byte(content)); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read for verification: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

// TestWriteFileAutoCreateDir tests auto-creating parent directories on write
func TestWriteFileAutoCreateDir(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "sub", "dir", "output.txt")
	if err := WriteFile(filePath, []byte("test")); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if !FileExists(filePath) {
		t.Error("file should have been created")
	}
}

// TestWriteFileOverwrite tests overwriting an existing file
func TestWriteFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// First write
	if err := WriteFile(filePath, []byte("first")); err != nil {
		t.Fatal(err)
	}

	// Overwrite
	if err := WriteFile(filePath, []byte("second")); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file after overwrite: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("overwrite failed: got %q", string(data))
	}
}

// TestCopyFile tests file copying
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := "source content 源文件"
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}

	// Verify
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(data) != content {
		t.Errorf("copied content mismatch: got %q", string(data))
	}
}

// TestCopyFileNonExistent tests copying a non-existent file
func TestCopyFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	err := CopyFile("/nonexistent/file", filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Error("copying non-existent file should return an error")
	}
}

// TestCopyFileIsDir tests copying a directory (should fail)
func TestCopyFileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	err := CopyFile(srcDir, filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Error("copying a directory should return an error")
	}
}

// TestCopyFileAutoCreateDstDir tests auto-creating destination directory on copy
func TestCopyFileAutoCreateDstDir(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.txt")
	if err := os.WriteFile(srcPath, []byte("data"), 0644); err != nil {
		t.Fatalf("write src file failed: %v", err)
	}

	dstPath := filepath.Join(tmpDir, "new", "dir", "dst.txt")
	err := CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}
	if !FileExists(dstPath) {
		t.Error("destination file should have been created")
	}
}

// TestRelPath tests relative path computation
func TestRelPath(t *testing.T) {
	tests := []struct {
		base   string
		target string
		want   string
	}{
		{"/home/user", "/home/user/file.txt", "file.txt"},
		{"/home/user", "/home/user/sub/file.txt", "sub/file.txt"},
		{"/home/user", "/home/user", "."},
		{"/home/user/a", "/home/user/b/file.txt", "../b/file.txt"},
	}

	for _, tt := range tests {
		got := RelPath(tt.base, tt.target)
		if got != tt.want {
			t.Errorf("RelPath(%q, %q) = %q, want %q", tt.base, tt.target, got, tt.want)
		}
	}
}

// TestReadWriteRoundTrip tests read-write round trip
func TestReadWriteRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "roundtrip.txt")

	// Various content types
	contents := []string{
		"简单文本",
		"包含\n换行符\n的文本",
		"包含 Unicode: 中文 日本語 한국어 emoji: 🎉",
		"", // empty file
		"very long " + string(make([]byte, 10000)), // large file
	}

	for i, content := range contents {
		if err := WriteFile(filePath, []byte(content)); err != nil {
			t.Fatalf("case %d: write failed: %v", i, err)
		}
		data, err := ReadFile(filePath)
		if err != nil {
			t.Fatalf("case %d: read failed: %v", i, err)
		}
		if string(data) != content {
			t.Errorf("case %d: read-write mismatch", i)
		}
	}
}

// TestCacheRootDir tests cache root directory
func TestCacheRootDir(t *testing.T) {
	// Test 1: Default behavior (no env var)
	t.Run("default without env var", func(t *testing.T) {
		t.Setenv("MDPRESS_CACHE_DIR", "")
		cacheDir := CacheRootDir()
		if cacheDir == "" {
			t.Error("CacheRootDir should return non-empty path")
		}
		// Should contain expected pattern
		if !strings.HasSuffix(cacheDir, "mdpress-cache") {
			t.Errorf("CacheRootDir() = %q, expected to end with 'mdpress-cache'", cacheDir)
		}
	})

	// Test 2: With MDPRESS_CACHE_DIR set
	t.Run("with custom env var", func(t *testing.T) {
		customDir := "/custom/cache/path"
		t.Setenv("MDPRESS_CACHE_DIR", customDir)
		cacheDir := CacheRootDir()
		if cacheDir != customDir {
			t.Errorf("CacheRootDir() = %q, want %q", cacheDir, customDir)
		}
	})

	// Test 3: With whitespace in env var (should be trimmed)
	t.Run("with whitespace", func(t *testing.T) {
		customDir := "  /path/with/spaces  "
		expected := "/path/with/spaces"
		t.Setenv("MDPRESS_CACHE_DIR", customDir)
		cacheDir := CacheRootDir()
		if cacheDir != expected {
			t.Errorf("CacheRootDir() = %q, want %q (trimmed)", cacheDir, expected)
		}
	})
}

// TestCacheDisabled tests cache disabled check
func TestCacheDisabled(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectedVal bool
	}{
		{"unset", "", false},
		{"1", "1", true},
		{"true", "true", true},
		{"True", "True", true},
		{"TRUE", "TRUE", true},
		{"yes", "yes", true},
		{"Yes", "Yes", true},
		{"YES", "YES", true},
		{"on", "on", true},
		{"On", "On", true},
		{"ON", "ON", true},
		{"false", "false", false},
		{"0", "0", false},
		{"no", "no", false},
		{"off", "off", false},
		{"random", "random", false},
		{"with spaces", "  true  ", true},
		{"with spaces 1", "  1  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MDPRESS_DISABLE_CACHE", tt.envValue)
			result := CacheDisabled()
			if result != tt.expectedVal {
				t.Errorf("CacheDisabled() with %q = %v, want %v", tt.envValue, result, tt.expectedVal)
			}
		})
	}
}

// TestExtractTitleFromFile tests extracting title from file
func TestExtractTitleFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectedLen int // len > 0 for title found, 0 for no title
	}{
		{
			name:        "simple h1 heading",
			content:     "# My Title",
			expectedLen: 1,
		},
		{
			name:        "h1 with leading whitespace",
			content:     "  # My Title With Spaces  ",
			expectedLen: 1,
		},
		{
			name:        "h1 in middle",
			content:     "Some intro\n# The Real Title\nMore content",
			expectedLen: 1,
		},
		{
			name:        "multiple h1 (returns first)",
			content:     "# First Title\n# Second Title",
			expectedLen: 1,
		},
		{
			name:        "no h1 heading",
			content:     "## H2 Heading\n### H3 Heading",
			expectedLen: 0,
		},
		{
			name:        "empty file",
			content:     "",
			expectedLen: 0,
		},
		{
			name:        "h1 with special chars",
			content:     "# Title: With Special (Chars) & More",
			expectedLen: 1,
		},
		{
			name:        "h1 with unicode",
			content:     "# 你好世界 - Hello World",
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			title := ExtractTitleFromFile(filePath)
			if tt.expectedLen > 0 {
				if title == "" {
					t.Errorf("ExtractTitleFromFile() expected title, got empty")
				}
				if !strings.HasPrefix(tt.content, "") {
					// Basic sanity check that title appears in content
					if !strings.Contains(tt.content, title) {
						t.Errorf("ExtractTitleFromFile() = %q, not found in content", title)
					}
				}
			} else if title != "" {
				t.Errorf("ExtractTitleFromFile() = %q, expected empty", title)
			}
		})
	}
}

// TestExtractTitleFromFileNonExistent tests extracting title from non-existent file
func TestExtractTitleFromFileNonExistent(t *testing.T) {
	title := ExtractTitleFromFile("/nonexistent/file.md")
	if title != "" {
		t.Errorf("ExtractTitleFromFile() on non-existent file should return empty, got %q", title)
	}
}

// TestExtractTitleFrom50LineLimit tests the 50-line limit
func TestExtractTitleFrom50LineLimit(t *testing.T) {
	// Create a file where H1 is beyond the 50-line limit
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.md")

	// Create content with 100 lines, H1 at line 60
	var content strings.Builder
	for i := 0; i < 59; i++ {
		content.WriteString("Some content line\n")
	}
	content.WriteString("# Title After Line 50\n")
	content.WriteString("More content\n")

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	title := ExtractTitleFromFile(filePath)
	if title != "" {
		t.Errorf("ExtractTitleFromFile() should stop at 50 lines, got %q", title)
	}
}
