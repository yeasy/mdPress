package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// shrinkParsedChapterCacheBudget makes the cache budget small enough to test
// without writing half a gigabyte.
func shrinkParsedChapterCacheBudget(t *testing.T, maxBytes, checkBytes int64) {
	t.Helper()
	origMax, origCheck := parsedChapterCacheMaxBytes, parsedChapterCacheCheckBytes
	parsedChapterCacheMaxBytes, parsedChapterCacheCheckBytes = maxBytes, checkBytes
	parsedChapterCacheWritten.Store(0)
	t.Cleanup(func() {
		parsedChapterCacheMaxBytes, parsedChapterCacheCheckBytes = origMax, origCheck
		parsedChapterCacheWritten.Store(0)
	})
}

func parsedChapterCacheSize(t *testing.T, root string) (bytes int64, entries int) {
	t.Helper()
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // best-effort measurement
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		bytes += info.Size()
		entries++
		return nil
	})
	return bytes, entries
}

// TestParsedChapterCacheStaysWithinBudget pins the size cap. The cache key
// covers the chapter's content, so every save of a chapter mints a new entry
// and nothing replaces the superseded one: `mdpress serve` on a 20 MB chapter
// grew the cache ~24 MB per save (165 MB in seven saves), and the only
// eviction was a 14-day age cutoff that reclaims nothing during the session
// that creates the mess.
func TestParsedChapterCacheStaysWithinBudget(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("MDPRESS_CACHE_DIR", cacheDir)
	const budget = 64 << 10
	shrinkParsedChapterCacheBudget(t, budget, 4<<10)

	body := strings.Repeat("x", 4<<10)
	var lastContent string
	for i := range 60 {
		// Every iteration is a fresh "save" of the same chapter.
		lastContent = fmt.Sprintf("# Big\n\nsave %d\n", i)
		if err := storeParsedChapterCache("big.md", lastContent, "github", &cachedParsedChapter{HTML: body}); err != nil {
			t.Fatalf("store: %v", err)
		}
	}

	size, entries := parsedChapterCacheSize(t, parsedChaptersCacheDir())
	// One entry of slack: the budget is rechecked after a write, not before.
	if limit := int64(budget) + int64(len(body)) + 1024; size > limit {
		t.Errorf("cache grew to %d bytes in %d entries, budget is %d", size, entries, budget)
	}

	// Evicting the entry the current build just wrote would make the cache
	// useless exactly when it matters.
	if _, hit, err := loadParsedChapterCache("big.md", lastContent, "github"); err != nil || !hit {
		t.Errorf("most recent entry should survive eviction (hit=%v err=%v)", hit, err)
	}
}

// TestEnforceParsedChapterCacheBudgetEvictsLeastRecentlyUsed checks the
// eviction order: entries are touched on every cache hit, so the oldest
// modification time is the least recently used chapter.
func TestEnforceParsedChapterCacheBudgetEvictsLeastRecentlyUsed(t *testing.T) {
	root := t.TempDir()
	shard := filepath.Join(root, "ab")
	if err := os.MkdirAll(shard, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	paths := make([]string, 4)
	for i := range paths {
		p := filepath.Join(shard, fmt.Sprintf("entry%d.json", i))
		if err := os.WriteFile(p, []byte(strings.Repeat("y", 1000)), 0o644); err != nil {
			t.Fatal(err)
		}
		// entry0 is the oldest, entry3 the newest.
		mod := now.Add(time.Duration(i-len(paths)) * time.Hour)
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}

	if removed := enforceParsedChapterCacheBudget(root, 2500); removed != 2 {
		t.Fatalf("removed %d entries, want 2", removed)
	}
	for i, p := range paths {
		_, err := os.Stat(p)
		if wantGone := i < 2; wantGone != os.IsNotExist(err) {
			t.Errorf("entry%d: exists=%v, want gone=%v", i, err == nil, wantGone)
		}
	}

	// A cache already inside the budget must not be touched.
	if removed := enforceParsedChapterCacheBudget(root, 1<<20); removed != 0 {
		t.Errorf("a cache within budget should not be evicted, removed %d", removed)
	}
}
