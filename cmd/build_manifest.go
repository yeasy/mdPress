package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/yeasy/mdpress/pkg/utils"
)

const buildManifestVersion = "v1"
const buildManifestFilename = "build-manifest.json"

// manifestEntry represents cached metadata for a single chapter.
type manifestEntry struct {
	SHA256   string    `json:"sha256"`
	HTMLPath string    `json:"html_path"`
	Headings []string  `json:"headings"` // List of heading texts for TOC
	ModTime  time.Time `json:"mod_time"` // Chapter file modification time
}

// BuildManifest stores chapter compilation state for incremental builds.
type BuildManifest struct {
	Version  string                   `json:"version"`
	AppVer   string                   `json:"app_version"`
	ConfigSH string                   `json:"config_sha256"`
	CSSHash  string                   `json:"css_hash"`
	Chapters map[string]manifestEntry `json:"chapters"`
}

// LoadManifest loads the build manifest from the cache directory.
// Returns an empty manifest if the file doesn't exist.
func LoadManifest(cacheDir string) (*BuildManifest, error) {
	if utils.CacheDisabled() {
		return NewBuildManifest(""), nil
	}

	manifestPath := filepath.Join(cacheDir, buildManifestFilename)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewBuildManifest(""), nil
		}
		return nil, fmt.Errorf("read build manifest: %w", err)
	}

	var manifest BuildManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		slog.Debug("build manifest unmarshal failed, starting fresh",
			"path", manifestPath, "error", err)
		return NewBuildManifest(""), nil
	}

	return &manifest, nil
}

// SaveManifest writes the manifest to disk atomically.
func SaveManifest(cacheDir string, manifest *BuildManifest) error {
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

// ComputeChapterHash computes the SHA-256 hash of a chapter file.
func ComputeChapterHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// ComputeConfigHash computes the SHA-256 hash of the config file.
func ComputeConfigHash(filePath string) (string, error) {
	return ComputeChapterHash(filePath)
}

// ComputeCSSHash computes the SHA-256 hash of CSS content.
func ComputeCSSHash(content string) string {
	h := sha256.New()
	_, _ = io.WriteString(h, content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NewBuildManifest creates a fresh manifest with the current app version.
func NewBuildManifest(appVer string) *BuildManifest {
	if appVer == "" {
		appVer = Version // Use the global Version from root.go
	}
	return &BuildManifest{
		Version:  buildManifestVersion,
		AppVer:   appVer,
		Chapters: make(map[string]manifestEntry),
	}
}

// IsStale checks if the manifest should be invalidated due to:
// - Version change
// - Config file change
// - CSS/theme change
func (m *BuildManifest) IsStale(currentAppVer, currentConfigHash, currentCSSHash string) bool {
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

// UpdateEntry updates a manifest entry for a chapter.
func (m *BuildManifest) UpdateEntry(chapterPath, hash, htmlPath string, headingTexts []string, modTime time.Time) {
	m.Chapters[chapterPath] = manifestEntry{
		SHA256:   hash,
		HTMLPath: htmlPath,
		Headings: headingTexts,
		ModTime:  modTime,
	}
}

// GetEntry retrieves a manifest entry for a chapter.
func (m *BuildManifest) GetEntry(chapterPath string) (manifestEntry, bool) {
	entry, ok := m.Chapters[chapterPath]
	return entry, ok
}

// cacheStatistics tracks cache hit/miss counts.
type cacheStatistics struct {
	Total     int
	Hits      int
	Misses    int
	Timestamp time.Time
}

// newCacheStatistics creates a new stats tracker.
func newCacheStatistics() *cacheStatistics {
	return &cacheStatistics{
		Timestamp: time.Now(),
	}
}

// RecordHit increments hit counter.
func (s *cacheStatistics) RecordHit() {
	s.Total++
	s.Hits++
}

// RecordMiss increments miss counter.
func (s *cacheStatistics) RecordMiss() {
	s.Total++
	s.Misses++
}

// String returns a human-readable summary.
func (s *cacheStatistics) String() string {
	if s.Total == 0 {
		return "cache: no chapters processed"
	}
	percentage := int(float64(s.Hits) * 100 / float64(s.Total))
	return fmt.Sprintf("cache: %d/%d hits (%d%%)", s.Hits, s.Total, percentage)
}
