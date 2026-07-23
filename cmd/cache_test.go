package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeCacheEntry creates a parsed-chapter cache entry with the given age.
func writeCacheEntry(t *testing.T, root, shard, name string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, shard)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create shard dir: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(`{"html":"<p>x</p>"}`), 0o644); err != nil {
		t.Fatalf("write cache entry: %v", err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("set cache entry mtime: %v", err)
	}
	return path
}

// TestSweepParsedChapterCacheEvictsStaleEntries covers finding 1007: the parsed
// chapter cache gained an entry for every edit of every chapter and never lost
// one, growing without bound.
func TestSweepParsedChapterCacheEvictsStaleEntries(t *testing.T) {
	root := filepath.Join(t.TempDir(), "parsed-chapters")

	fresh := writeCacheEntry(t, root, "ab", "ab111.json", time.Hour)
	stale := writeCacheEntry(t, root, "cd", "cd222.json", 30*24*time.Hour)

	removed := sweepParsedChapterCache(root, parsedChapterCacheMaxAge, time.Now())
	if removed != 1 {
		t.Fatalf("sweep should have removed exactly the stale entry, removed %d", removed)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("a recently used entry must survive the sweep: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("a stale entry should have been evicted, stat err=%v", err)
	}
	// The emptied shard directory should not be left behind either.
	if _, err := os.Stat(filepath.Join(root, "cd")); !os.IsNotExist(err) {
		t.Errorf("an emptied shard directory should be removed, stat err=%v", err)
	}
}

// TestSweepParsedChapterCacheTolerantOfMissingRoot keeps the sweep off the
// critical path of a first-ever build.
func TestSweepParsedChapterCacheTolerantOfMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	if removed := sweepParsedChapterCache(root, parsedChapterCacheMaxAge, time.Now()); removed != 0 {
		t.Errorf("sweeping a missing cache should be a no-op, removed %d", removed)
	}
}

// TestParsedChapterCacheHitRefreshesMtime makes eviction mean "unused for two
// weeks" rather than "written two weeks ago", so an actively rebuilt book does
// not lose its cache.
func TestParsedChapterCacheHitRefreshesMtime(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MDPRESS_CACHE_DIR", root)
	t.Setenv("MDPRESS_DISABLE_CACHE", "")

	const content = "# Chapter\n\nBody.\n"
	if err := storeParsedChapterCache("ch.md", content, "github", &cachedParsedChapter{HTML: "<p>x</p>"}); err != nil {
		t.Fatalf("store cache entry: %v", err)
	}
	path := parsedChapterCachePath("ch.md", content, "github")
	old := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age cache entry: %v", err)
	}

	if _, ok, err := loadParsedChapterCache("ch.md", content, "github"); err != nil || !ok {
		t.Fatalf("expected a cache hit, ok=%v err=%v", ok, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache entry: %v", err)
	}
	if time.Since(info.ModTime()) > time.Minute {
		t.Errorf("a cache hit should refresh the entry's mtime, got %s", info.ModTime())
	}
}

// TestCacheInfoAndClear covers the other half of finding 1007: there was no
// CLI entry point at all for inspecting or reclaiming the cache.
func TestCacheInfoAndClear(t *testing.T) {
	restoreGlobalFlags(t)

	root := filepath.Join(t.TempDir(), "mdpress-cache")
	writeCacheEntry(t, filepath.Join(root, "parsed-chapters"), "ab", "ab111.json", time.Hour)
	writeCacheEntry(t, filepath.Join(root, "parsed-chapters"), "cd", "cd222.json", time.Hour)

	out, err := captureThemesStdout(t, func() error { return executeCacheInfo(root) })
	if err != nil {
		t.Fatalf("cache info returned error: %v", err)
	}
	if !strings.Contains(out, root) {
		t.Errorf("cache info should report the cache location, got:\n%s", out)
	}
	if !strings.Contains(out, "Entries:  2") {
		t.Errorf("cache info should report 2 entries, got:\n%s", out)
	}
	if !strings.Contains(out, "parsed-chapters") {
		t.Errorf("cache info should break the usage down per cache, got:\n%s", out)
	}

	if _, err := captureThemesStdout(t, func() error { return executeCacheClear(root) }); err != nil {
		t.Fatalf("cache clear returned error: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("cache clear should remove the cache directory, stat err=%v", err)
	}

	// Clearing an already-empty cache must stay a success.
	if _, err := captureThemesStdout(t, func() error { return executeCacheClear(root) }); err != nil {
		t.Fatalf("clearing an empty cache should succeed, got: %v", err)
	}
}

// TestCacheCommandRegistered makes sure the command is reachable from the CLI.
func TestCacheCommandRegistered(t *testing.T) {
	var found bool
	for _, c := range rootCmd.Commands() {
		if c.Name() == "cache" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("mdpress cache should be registered on the root command")
	}
	for _, sub := range []string{"info", "clear"} {
		var ok bool
		for _, c := range cacheCmd.Commands() {
			if c.Name() == sub {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("mdpress cache should have a %q sub-command", sub)
		}
	}
}

func TestFormatCacheSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{2048, "2.0 KB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{3 * 1024 * 1024 * 1024, "3.0 GB"},
	}
	for _, tt := range tests {
		if got := formatCacheSize(tt.bytes); got != tt.want {
			t.Errorf("formatCacheSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
