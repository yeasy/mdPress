package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// FileExists reports whether a file or directory exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory when it does not already exist.
// MkdirAll is idempotent, so we call it directly to avoid a TOCTOU race.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", path, err)
	}
	return nil
}

// CacheRootDir returns the root directory used for mdpress runtime caches.
// MDPRESS_CACHE_DIR overrides the default location when set.
//
// The default is the per-user cache directory, not the system temp dir. A
// parsed-chapter cache entry's HTML is emitted into the published book
// verbatim, so a cache at a fixed path under a shared, world-writable /tmp let
// any other local user pre-create the tree and plant entries — arbitrary script
// in someone else's site, from a build that reports success, and it defeated
// markdown.allow_html: false, the setting whose entire purpose is publishing
// content you did not write.
func CacheRootDir() string {
	if override := strings.TrimSpace(os.Getenv("MDPRESS_CACHE_DIR")); override != "" {
		return override
	}
	if userCache, err := os.UserCacheDir(); err == nil && userCache != "" {
		return filepath.Join(userCache, "mdpress")
	}
	// No per-user cache directory (HOME/XDG_CACHE_HOME unset, as in some
	// container images). cacheRootUsable below is what keeps this fallback from
	// reintroducing the shared-temp-dir problem.
	return filepath.Join(os.TempDir(), "mdpress-cache")
}

// CacheDisabled reports whether runtime caches are disabled for this process.
//
// Besides the explicit MDPRESS_DISABLE_CACHE opt-out, caching is refused when
// the cache root is not a private directory belonging to this user: trusting a
// tree someone else can write to is the same as trusting their HTML.
func CacheDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MDPRESS_DISABLE_CACHE"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return !cacheRootUsable(CacheRootDir())
}

// cacheRootWarned records the roots already reported so a rejected cache root
// is explained once per process rather than once per chapter.
var cacheRootWarned sync.Map

// cacheRootUsable reports whether root is a directory this user owns and only
// this user can write to, creating it privately when it does not exist yet.
// Creating it here rather than on first write also closes the window where an
// attacker could win the race to create it.
func cacheRootUsable(root string) bool {
	if root == "" {
		return false
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		warnUnusableCacheRoot(root, "cannot be created: "+err.Error())
		return false
	}
	info, err := os.Stat(root)
	if err != nil {
		warnUnusableCacheRoot(root, "cannot be inspected: "+err.Error())
		return false
	}
	if !info.IsDir() {
		warnUnusableCacheRoot(root, "is not a directory")
		return false
	}
	if uid, ok := fileOwnerUID(info); ok {
		// os.Getuid reports -1 where there is no uid to compare against.
		if self := os.Getuid(); self >= 0 && uid != uint64(self) {
			warnUnusableCacheRoot(root, "is owned by another user")
			return false
		}
	}
	// Windows reports synthesized permission bits, so the mode says nothing
	// about who may write there; the ownership check above carries that platform.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
		warnUnusableCacheRoot(root, "is writable by other users")
		return false
	}
	return true
}

func warnUnusableCacheRoot(root, reason string) {
	if _, seen := cacheRootWarned.LoadOrStore(root, struct{}{}); seen {
		return
	}
	slog.Warn("Cache disabled: the cache directory is not private to this user",
		slog.String("dir", root), slog.String("reason", reason))
}

// fileOwnerUID returns the uid owning info. Unix carries it in the
// syscall.Stat_t behind FileInfo.Sys(); Windows reports no uid at all, so ok is
// false there. Reading it reflectively keeps this in one file instead of a pair
// of build-tagged ones for a single field.
func fileOwnerUID(info os.FileInfo) (uint64, bool) {
	sys := info.Sys()
	if sys == nil {
		return 0, false
	}
	value := reflect.Indirect(reflect.ValueOf(sys))
	if value.Kind() != reflect.Struct {
		return 0, false
	}
	field := value.FieldByName("Uid")
	if !field.IsValid() || !field.CanUint() {
		return 0, false
	}
	return field.Uint(), true
}

// maxReadFileSize is a general-purpose safety net for ReadFile.
// Individual callers should set tighter limits for their use case.
const maxReadFileSize = 100 * 1024 * 1024 // 100 MB

// ReadFile reads a file and returns clearer errors.
// It rejects files larger than 100 MB as a safety net against OOM.
//
// The size check uses Fstat on the open file descriptor to avoid a
// TOCTOU race between a separate Stat call and the subsequent read.
func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("file does not exist %q: %w", path, err)
		}
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file: %q", path)
	}
	if fi.Size() > maxReadFileSize {
		return nil, fmt.Errorf("file %q is too large (%d bytes, max %d)", path, fi.Size(), maxReadFileSize)
	}

	data, err := io.ReadAll(io.LimitReader(f, maxReadFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	if int64(len(data)) > maxReadFileSize {
		return nil, fmt.Errorf("file %q is too large (exceeded %d bytes during read)", path, maxReadFileSize)
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
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", path, err)
	}

	return nil
}

// CopyFile copies a file from src to dst.
func CopyFile(src, dst string) error {
	// Open the source file directly (avoids TOCTOU race from a separate stat check).
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

	// Reject files that exceed the general-purpose size limit to prevent
	// disk exhaustion from symlinks to special files (e.g. /dev/zero).
	if srcInfo.Size() > maxReadFileSize {
		return fmt.Errorf("source file %q is too large (%d bytes, max %d)", src, srcInfo.Size(), maxReadFileSize)
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

	// cleanup removes the partial destination file on any write-path error.
	cleanup := func() {
		dstFile.Close() //nolint:errcheck
		os.Remove(dst)  //nolint:errcheck
	}

	// Copy file content with LimitReader as belt-and-suspenders defense
	// against the file growing between the Stat check and the copy (TOCTOU).
	// Use maxReadFileSize+1 so we can detect if the file grew past the limit.
	n, err := io.Copy(dstFile, io.LimitReader(srcFile, maxReadFileSize+1))
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	if n > maxReadFileSize {
		cleanup()
		return fmt.Errorf("source file %q grew beyond size limit during copy (%d bytes, max %d)", src, n, maxReadFileSize)
	}

	// Close explicitly to catch flush errors (e.g. NFS write-back failures).
	if err := dstFile.Close(); err != nil {
		os.Remove(dst) //nolint:errcheck
		return fmt.Errorf("failed to close destination file %q: %w", dst, err)
	}

	// Preserve file permissions.
	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		os.Remove(dst) //nolint:errcheck
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
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(title)
		}
	}

	// Best-effort title extraction — log scan errors for debugging.
	if err := scanner.Err(); err != nil {
		slog.Debug("scanner error during title extraction", slog.Any("error", err))
	}
	return ""
}

// SafeJoin joins a base directory with an untrusted relative path and verifies
// the result stays within the base directory. It returns an error if the
// resolved path escapes the base via ".." or absolute-path tricks.
func SafeJoin(baseDir, untrusted string) (string, error) {
	// Clean and resolve the base directory (including symlinks).
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base directory: %w", err)
	}
	if evaled, evalErr := filepath.EvalSymlinks(absBase); evalErr == nil {
		absBase = evaled
	}
	// Join with the resolved base and clean.
	joined := filepath.Join(absBase, untrusted)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve joined path: %w", err)
	}
	// Resolve symlinks to prevent containment bypass. When the full path
	// does not exist, walk up to the nearest existing ancestor, resolve
	// its symlinks, then re-append the remaining tail.
	if evaled, evalErr := filepath.EvalSymlinks(absJoined); evalErr == nil {
		absJoined = evaled
	} else {
		absJoined = EvalSymlinksAncestor(absJoined)
	}
	// Ensure the result is inside baseDir.
	if !strings.HasPrefix(absJoined, absBase+string(filepath.Separator)) && absJoined != absBase {
		return "", fmt.Errorf("path %q escapes base directory %q", untrusted, absBase)
	}
	return absJoined, nil
}

// EvalSymlinksAncestor resolves symlinks on the nearest existing ancestor of
// path and re-appends the non-existent tail. This handles the case where
// filepath.EvalSymlinks fails because the leaf file does not exist yet, but
// an intermediate directory is a symlink pointing outside the base.
func EvalSymlinksAncestor(path string) string {
	dir := path
	var tail []string
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		tail = append(tail, filepath.Base(dir))
		dir = parent
		if evaled, err := filepath.EvalSymlinks(dir); err == nil {
			// Rebuild the path with the resolved ancestor.
			for i := len(tail) - 1; i >= 0; i-- {
				evaled = filepath.Join(evaled, tail[i])
			}
			return evaled
		}
	}
	return path
}

// ParseVersionPart parses a version number component (e.g., "25" from "1.25.0").
func ParseVersionPart(s string) (int, error) {
	// Strip any non-numeric suffix (e.g., "25rc1" -> parse as "25")
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("no numeric part found in %q", s)
	}
	return strconv.Atoi(s[:i])
}
