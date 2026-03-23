package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileExists reports whether a file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory when it does not already exist.
func EnsureDir(path string) error {
	if FileExists(path) {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}
	return nil
}

// CacheRootDir returns the root directory used for mdpress runtime caches.
// MDPRESS_CACHE_DIR overrides the default location when set.
func CacheRootDir() string {
	if override := strings.TrimSpace(os.Getenv("MDPRESS_CACHE_DIR")); override != "" {
		return override
	}
	return filepath.Join(os.TempDir(), "mdpress-cache")
}

// CacheDisabled reports whether runtime caches are disabled for this process.
func CacheDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MDPRESS_DISABLE_CACHE"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ReadFile reads a file and returns clearer errors.
func ReadFile(path string) ([]byte, error) {
	// Ensure the file exists first.
	if !FileExists(path) {
		return nil, fmt.Errorf("file does not exist: %q", path)
	}

	// Reject directories explicitly.
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %q", path)
	}

	// Read the file content.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	return data, nil
}

// WriteFile writes file content and creates parent directories when needed.
func WriteFile(path string, data []byte) error {
	// Resolve the parent directory.
	dir := filepath.Dir(path)

	// Ensure the parent directory exists.
	if err := EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Write the file content.
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}

	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	// Ensure the source file exists.
	if !FileExists(src) {
		return fmt.Errorf("source file does not exist: %q", src)
	}

	// Open the source file.
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %w", src, err)
	}
	defer srcFile.Close() //nolint:errcheck

	// Read source metadata.
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Reject directory sources.
	if srcInfo.IsDir() {
		return fmt.Errorf("source path is a directory: %q", src)
	}

	// Ensure the destination directory exists.
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create the destination file.
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %w", dst, err)
	}
	defer dstFile.Close() //nolint:errcheck

	// Copy file content.
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Preserve file permissions.
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set destination file mode: %w", err)
	}

	return nil
}

// RelPath computes the target path relative to the base path.
func RelPath(basePath, targetPath string) string {
	// Normalize both paths first.
	basePath = filepath.Clean(basePath)
	targetPath = filepath.Clean(targetPath)

	// Return the current directory when the paths are identical.
	if basePath == targetPath {
		return "."
	}

	// Compute the relative path.
	relPath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		// Fall back to the normalized target path when Rel fails.
		return targetPath
	}

	// Use forward slashes on Windows as well.
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	return relPath
}

// ExtractTitleFromFile scans a Markdown file and returns the first H1 heading.
// For performance, scanning stops after 50 lines.
func ExtractTitleFromFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	lineCount := 0
	const maxLines = 50

	for scanner.Scan() {
		lineCount++
		if lineCount > maxLines {
			break // Stop scanning after 50 lines for performance.
		}

		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	// Check for scanner errors; silently ignore them for best-effort title extraction.
	if err := scanner.Err(); err != nil {
		// Error occurred during scanning, but we continue with empty result.
		// This is best-effort title extraction.
		_ = err
	}
	return ""
}
