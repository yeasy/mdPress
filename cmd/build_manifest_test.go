package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeChapterHash(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-chapter-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "# Test Chapter\n\nSome content here."
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	hash, err := ComputeChapterHash(tmpFile.Name())
	if err != nil {
		t.Fatalf("ComputeChapterHash failed: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Verify it's a valid hex string (64 chars for SHA-256)
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}

	// Verify same content produces same hash
	hash2, err := ComputeChapterHash(tmpFile.Name())
	if err != nil {
		t.Fatalf("second ComputeChapterHash failed: %v", err)
	}
	if hash != hash2 {
		t.Error("same file produced different hashes")
	}
}

func TestComputeConfigHash(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	config := "title: Test Book\nauthor: Test Author"
	if _, err := tmpFile.WriteString(config); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	hash, err := ComputeConfigHash(tmpFile.Name())
	if err != nil {
		t.Fatalf("ComputeConfigHash failed: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}
}

func TestComputeCSSHash(t *testing.T) {
	css := "body { color: black; }"
	hash := ComputeCSSHash(css)

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}

	// Verify consistency
	hash2 := ComputeCSSHash(css)
	if hash != hash2 {
		t.Error("same CSS produced different hashes")
	}

	// Verify different CSS produces different hash
	hash3 := ComputeCSSHash("body { color: red; }")
	if hash == hash3 {
		t.Error("different CSS produced same hash")
	}
}

func TestManifestLoadSaveRoundTrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-manifest-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a manifest
	manifest := NewBuildManifest("1.0.0")
	manifest.ConfigSH = "config-hash-123"
	manifest.CSSHash = "css-hash-456"

	modTime := time.Now().UTC()
	manifest.UpdateEntry("ch01.md", "hash1", "/path/ch01.html",
		[]string{"Heading 1", "Heading 2"}, modTime)
	manifest.UpdateEntry("ch02.md", "hash2", "/path/ch02.html",
		[]string{"Heading 3"}, modTime)

	// Save it
	if err := SaveManifest(tmpDir, manifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	// Load it back
	loaded, err := LoadManifest(tmpDir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	// Verify contents
	if loaded.Version != buildManifestVersion {
		t.Errorf("version mismatch: expected %q, got %q", buildManifestVersion, loaded.Version)
	}
	if loaded.AppVer != "1.0.0" {
		t.Errorf("app version mismatch: expected 1.0.0, got %q", loaded.AppVer)
	}
	if loaded.ConfigSH != "config-hash-123" {
		t.Errorf("config hash mismatch: expected config-hash-123, got %q", loaded.ConfigSH)
	}
	if loaded.CSSHash != "css-hash-456" {
		t.Errorf("CSS hash mismatch: expected css-hash-456, got %q", loaded.CSSHash)
	}

	if len(loaded.Chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(loaded.Chapters))
	}

	// Verify chapter 1
	entry1, ok := loaded.GetEntry("ch01.md")
	if !ok {
		t.Error("ch01.md not found in manifest")
	} else {
		if entry1.SHA256 != "hash1" {
			t.Errorf("ch01 hash mismatch: expected hash1, got %q", entry1.SHA256)
		}
		if len(entry1.Headings) != 2 {
			t.Errorf("ch01 expected 2 headings, got %d", len(entry1.Headings))
		}
	}

	// Verify chapter 2
	entry2, ok := loaded.GetEntry("ch02.md")
	if !ok {
		t.Error("ch02.md not found in manifest")
	} else {
		if entry2.SHA256 != "hash2" {
			t.Errorf("ch02 hash mismatch: expected hash2, got %q", entry2.SHA256)
		}
	}
}

func TestManifestIsStale(t *testing.T) {
	tests := []struct {
		name             string
		manifest         *BuildManifest
		appVer           string
		configHash       string
		cssHash          string
		expectStale      bool
		description      string
	}{
		{
			name:        "nil manifest",
			manifest:    nil,
			expectStale: true,
			description: "nil manifest should be stale",
		},
		{
			name:        "no chapters",
			manifest:    NewBuildManifest("1.0.0"),
			appVer:      "1.0.0",
			configHash:  "",
			cssHash:     "",
			expectStale: false,
			description: "fresh manifest with no chapters should not be stale",
		},
		{
			name:        "version mismatch",
			manifest:    NewBuildManifest("1.0.0"),
			appVer:      "2.0.0",
			expectStale: true,
			description: "version mismatch should be stale",
		},
		{
			name:        "config hash mismatch",
			manifest:    &BuildManifest{Version: buildManifestVersion, AppVer: "1.0.0", ConfigSH: "old-hash", Chapters: map[string]ManifestEntry{}},
			appVer:      "1.0.0",
			configHash:  "new-hash",
			cssHash:     "",
			expectStale: true,
			description: "config hash change should be stale",
		},
		{
			name:        "css hash mismatch",
			manifest:    &BuildManifest{Version: buildManifestVersion, AppVer: "1.0.0", ConfigSH: "same-hash", CSSHash: "old-css", Chapters: map[string]ManifestEntry{}},
			appVer:      "1.0.0",
			configHash:  "same-hash",
			cssHash:     "new-css",
			expectStale: true,
			description: "CSS hash change should be stale",
		},
		{
			name:        "all match",
			manifest:    &BuildManifest{Version: buildManifestVersion, AppVer: "1.0.0", ConfigSH: "hash1", CSSHash: "hash2", Chapters: map[string]ManifestEntry{"ch01.md": {SHA256: "h1"}}},
			appVer:      "1.0.0",
			configHash:  "hash1",
			cssHash:     "hash2",
			expectStale: false,
			description: "matching manifest should not be stale",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.manifest.IsStale(tc.appVer, tc.configHash, tc.cssHash)
			if result != tc.expectStale {
				t.Errorf("%s: expected stale=%v, got %v", tc.description, tc.expectStale, result)
			}
		})
	}
}

func TestCacheStatistics(t *testing.T) {
	stats := NewCacheStatistics()

	if stats.Total != 0 {
		t.Errorf("expected 0 total, got %d", stats.Total)
	}

	stats.RecordHit()
	if stats.Total != 1 || stats.Hits != 1 || stats.Misses != 0 {
		t.Errorf("after hit: total=%d, hits=%d, misses=%d", stats.Total, stats.Hits, stats.Misses)
	}

	stats.RecordMiss()
	if stats.Total != 2 || stats.Hits != 1 || stats.Misses != 1 {
		t.Errorf("after miss: total=%d, hits=%d, misses=%d", stats.Total, stats.Hits, stats.Misses)
	}

	stats.RecordHit()
	str := stats.String()
	if str == "" {
		t.Error("expected non-empty stats string")
	}
	if !contains(str, "2/3") {
		t.Errorf("expected '2/3' in stats string, got: %s", str)
	}
}

func TestManifestUpdateEntry(t *testing.T) {
	manifest := NewBuildManifest("1.0.0")
	modTime := time.Now().UTC()

	manifest.UpdateEntry("chapter.md", "somehash", "/output/chapter.html",
		[]string{"Heading 1", "Heading 2"}, modTime)

	entry, ok := manifest.GetEntry("chapter.md")
	if !ok {
		t.Fatal("entry not found after update")
	}

	if entry.SHA256 != "somehash" {
		t.Errorf("hash mismatch: expected somehash, got %q", entry.SHA256)
	}
	if entry.HTMLPath != "/output/chapter.html" {
		t.Errorf("html path mismatch: expected /output/chapter.html, got %q", entry.HTMLPath)
	}
	if len(entry.Headings) != 2 {
		t.Errorf("expected 2 headings, got %d", len(entry.Headings))
	}
}

func TestManifestFileCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-manifest-file-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewBuildManifest("1.0.0")
	manifest.ConfigSH = "test-config"
	manifest.UpdateEntry("test.md", "testhash", "/out/test.html", []string{}, time.Now())

	if err := SaveManifest(tmpDir, manifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	manifestPath := filepath.Join(tmpDir, buildManifestFilename)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file not created: %v", err)
	}

	// Verify it's valid JSON by loading it
	loaded, err := LoadManifest(tmpDir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	if loaded.AppVer != "1.0.0" {
		t.Errorf("loaded app version mismatch: expected 1.0.0, got %q", loaded.AppVer)
	}
}

func TestComputeHashDifferentFiles(t *testing.T) {
	tmpFile1, err := os.CreateTemp("", "test-ch1-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile1.Name())

	tmpFile2, err := os.CreateTemp("", "test-ch2-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile2.Name())

	// Write different content
	tmpFile1.WriteString("# Chapter 1\nContent 1")
	tmpFile1.Close()

	tmpFile2.WriteString("# Chapter 2\nContent 2")
	tmpFile2.Close()

	hash1, err := ComputeChapterHash(tmpFile1.Name())
	if err != nil {
		t.Fatalf("ComputeChapterHash failed: %v", err)
	}

	hash2, err := ComputeChapterHash(tmpFile2.Name())
	if err != nil {
		t.Fatalf("ComputeChapterHash failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("different files produced same hash")
	}
}

func TestManifestEmptyChaptersMap(t *testing.T) {
	manifest := NewBuildManifest("1.0.0")

	if len(manifest.Chapters) != 0 {
		t.Errorf("expected empty chapters map, got %d entries", len(manifest.Chapters))
	}

	// Try to get non-existent entry
	_, ok := manifest.GetEntry("nonexistent.md")
	if ok {
		t.Error("expected false for non-existent entry")
	}
}

func TestManifestVersionMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-manifest-version-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewBuildManifest("1.0.0")
	manifest.Version = "wrong-version"
	manifest.UpdateEntry("ch01.md", "hash1", "/path/ch01.html", []string{}, time.Now())

	if err := SaveManifest(tmpDir, manifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	loaded, err := LoadManifest(tmpDir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	// Version should be loaded as-is
	if loaded.Version != "wrong-version" {
		t.Errorf("expected wrong-version, got %q", loaded.Version)
	}
}

func TestCacheStatisticsPercentage(t *testing.T) {
	stats := NewCacheStatistics()

	// Record 7 hits and 3 misses
	for i := 0; i < 7; i++ {
		stats.RecordHit()
	}
	for i := 0; i < 3; i++ {
		stats.RecordMiss()
	}

	if stats.Total != 10 {
		t.Errorf("expected 10 total, got %d", stats.Total)
	}
	if stats.Hits != 7 {
		t.Errorf("expected 7 hits, got %d", stats.Hits)
	}
	if stats.Misses != 3 {
		t.Errorf("expected 3 misses, got %d", stats.Misses)
	}

	str := stats.String()
	if !contains(str, "70%") {
		t.Errorf("expected 70%% in stats, got: %s", str)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
