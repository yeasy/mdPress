package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/yeasy/mdpress/pkg/utils"
)

const buildManifestVersion = "v1"
const buildManifestFilename = "build-manifest.json"
const maxManifestSize = 10 * 1024 * 1024  // 10 MB
const maxHashFileSize = 100 * 1024 * 1024 // 100 MB

// manifestEntry represents cached metadata for a single chapter.
type manifestEntry struct {
	SHA256   string    `json:"sha256"`
	HTMLPath string    `json:"html_path"`
	Headings []string  `json:"headings"` // List of heading texts for TOC
	ModTime  time.Time `json:"mod_time"` // Chapter file modification time
}

// buildManifest stores chapter compilation state for incremental builds.
type buildManifest struct {
	Version  string                   `json:"version"`
	AppVer   string                   `json:"app_version"`
	ConfigSH string                   `json:"config_sha256"`
	CSSHash  string                   `json:"css_hash"`
	Chapters map[string]manifestEntry `json:"chapters"`
}

// loadManifest loads the build manifest from the cache directory.
// Returns an empty manifest if the file doesn't exist.
func loadManifest(cacheDir string) (*buildManifest, error) {
	if utils.CacheDisabled() {
		return newBuildManifest(""), nil
	}

	manifestPath := filepath.Join(cacheDir, buildManifestFilename)
	f, err := os.Open(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return newBuildManifest(""), nil
		}
		return nil, fmt.Errorf("open build manifest: %w", err)
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, maxManifestSize+1))
	if err != nil {
		return nil, fmt.Errorf("read build manifest: %w", err)
	}
	if int64(len(data)) > maxManifestSize {
		slog.Debug("build manifest too large, starting fresh",
			"path", manifestPath, "size", len(data))
		return newBuildManifest(""), nil
	}

	var manifest buildManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		slog.Debug("build manifest unmarshal failed, starting fresh",
			"path", manifestPath, "error", err)
		return newBuildManifest(""), nil
	}

	return &manifest, nil
}

// saveManifest writes the manifest to disk atomically.
func saveManifest(cacheDir string, manifest *buildManifest) error {
	if utils.CacheDisabled() {
		return nil
	}

	if err := utils.EnsureDir(cacheDir); err != nil {
		return fmt.Errorf("ensure cache dir: %w", err)
	}

	manifestPath := filepath.Join(cacheDir, buildManifestFilename)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// Atomic write using temp file + rename
	tmpFile, err := os.CreateTemp(cacheDir, "manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp manifest file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp manifest: %w", err)
	}
	if err := os.Rename(tmpPath, manifestPath); err != nil {
		return fmt.Errorf("rename manifest: %w", err)
	}

	return nil
}

// computeChapterHash computes the SHA-256 hash of a chapter file.
func computeChapterHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(file, maxHashFileSize+1))
	if err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	if n > maxHashFileSize {
		return "", fmt.Errorf("file too large for hashing: read %d bytes (max %d)", n, maxHashFileSize)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// computeCSSHash computes the SHA-256 hash of CSS content.
func computeCSSHash(content string) string {
	h := sha256.New()
	_, _ = io.WriteString(h, content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// newBuildManifest creates a fresh manifest with the current app version.
func newBuildManifest(appVer string) *buildManifest {
	if appVer == "" {
		appVer = Version // Use the global Version from root.go
	}
	return &buildManifest{
		Version:  buildManifestVersion,
		AppVer:   appVer,
		Chapters: make(map[string]manifestEntry),
	}
}

// IsStale checks if the manifest should be invalidated due to:
// - Version change
// - Config file change
// - CSS/theme change
func (m *buildManifest) IsStale(currentAppVer, currentConfigHash, currentCSSHash string) bool {
	if m == nil || m.Chapters == nil {
		return true
	}
	if m.Version != buildManifestVersion {
		return true
	}
	if m.AppVer != currentAppVer {
		return true
	}
	if m.ConfigSH != currentConfigHash {
		return true
	}
	if m.CSSHash != currentCSSHash {
		return true
	}
	return false
}
