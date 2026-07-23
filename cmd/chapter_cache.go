package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/pkg/utils"
)

// parsedChapterCacheVersion invalidates cached chapters when the cache's own
// format changes. It does not cover renderer changes — see the key below.
const parsedChapterCacheVersion = "v2"

type cachedParsedChapter struct {
	HTML        string                 `json:"html"`
	Headings    []markdown.HeadingInfo `json:"headings"`
	Diagnostics []markdown.Diagnostic  `json:"diagnostics"`
}

// rendererFingerprint identifies the binary that produced a cached chapter.
//
// The cache is keyed on chapter content, so anything that changes how content
// is rendered must also change the key — otherwise an unchanged chapter keeps
// serving HTML from the previous binary and every rendering fix stays
// invisible. The version alone is not enough: it does not move between builds
// from source, which is exactly when the renderer changes most. The
// executable's size and modification time cover that, and are stable for an
// installed release, so caching still works across runs.
var rendererFingerprint = sync.OnceValue(func() string {
	parts := []string{Version}
	if exe, err := os.Executable(); err == nil {
		if info, statErr := os.Stat(exe); statErr == nil {
			parts = append(parts,
				strconv.FormatInt(info.Size(), 10),
				strconv.FormatInt(info.ModTime().UnixNano(), 10))
		}
	}
	return strings.Join(parts, ":")
})

// parsedChapterCacheMaxAge is how long an unused cache entry is kept.
//
// The cache is keyed on chapter content, so every edit of every chapter adds a
// permanent entry and nothing ever replaced one. Left alone it grew without
// bound — hundreds of megabytes and thousands of files on a machine that had
// only ever built a handful of books. Two weeks is well past the point where a
// stale entry could still be hit by a rebuild.
const parsedChapterCacheMaxAge = 14 * 24 * time.Hour

// parsedChaptersCacheDir is the subdirectory holding parsed-chapter entries.
func parsedChaptersCacheDir() string {
	return filepath.Join(utils.CacheRootDir(), "parsed-chapters")
}

func parsedChapterCachePath(chapterPath, expandedContent, codeTheme string) string {
	key := utils.StableHash(parsedChapterCacheVersion, rendererFingerprint(), chapterPath, codeTheme, expandedContent)
	return filepath.Join(parsedChaptersCacheDir(), key[:2], key+".json")
}

// sweepParsedChapterCache deletes entries not used within
// parsedChapterCacheMaxAge, plus any temp files a crashed write left behind.
// It is best-effort: a cache that cannot be pruned must never fail a build.
func sweepParsedChapterCache(root string, maxAge time.Duration, now time.Time) (removed int) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	cutoff := now.Add(-maxAge)
	for _, shard := range entries {
		if !shard.IsDir() {
			continue
		}
		shardDir := filepath.Join(root, shard.Name())
		files, err := os.ReadDir(shardDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			info, err := f.Info()
			if err != nil || info.IsDir() || info.ModTime().After(cutoff) {
				continue
			}
			if err := os.Remove(filepath.Join(shardDir, f.Name())); err == nil {
				removed++
			}
		}
		// Drop the shard directory once it is empty; keeping 256 empty
		// directories around forever is its own small mess.
		if remaining, err := os.ReadDir(shardDir); err == nil && len(remaining) == 0 {
			_ = os.Remove(shardDir)
		}
	}
	return removed
}

// sweepParsedChapterCacheOnce prunes the cache at most once per process, on
// first use, so long-lived `mdpress serve` sessions do not rescan every build.
var sweepParsedChapterCacheOnce = sync.OnceFunc(func() {
	if utils.CacheDisabled() {
		return
	}
	if removed := sweepParsedChapterCache(parsedChaptersCacheDir(), parsedChapterCacheMaxAge, time.Now()); removed > 0 {
		slog.Debug("pruned stale parsed chapter cache entries", "count", removed)
	}
})

func loadParsedChapterCache(chapterPath, expandedContent, codeTheme string) (*cachedParsedChapter, bool, error) {
	if utils.CacheDisabled() {
		return nil, false, nil
	}
	sweepParsedChapterCacheOnce()
	cachePath := parsedChapterCachePath(chapterPath, expandedContent, codeTheme)
	data, err := utils.ReadFile(cachePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read parsed chapter cache: %w", err)
	}

	var cached cachedParsedChapter
	if err := json.Unmarshal(data, &cached); err != nil {
		slog.Debug("parsed chapter cache unmarshal failed, treating as cache miss",
			"path", cachePath, "error", err)
		return nil, false, nil
	}
	// Touch on hit so the sweep above evicts by "unused for two weeks" rather
	// than "written two weeks ago" — an actively rebuilt book keeps its cache.
	now := time.Now()
	if err := os.Chtimes(cachePath, now, now); err != nil {
		slog.Debug("could not refresh parsed chapter cache mtime", "path", cachePath, "error", err)
	}
	return &cached, true, nil
}

func storeParsedChapterCache(chapterPath, expandedContent, codeTheme string, cached *cachedParsedChapter) error {
	if utils.CacheDisabled() {
		return nil
	}
	cachePath := parsedChapterCachePath(chapterPath, expandedContent, codeTheme)
	cacheDir := filepath.Dir(cachePath)
	if err := utils.EnsureDir(cacheDir); err != nil {
		return fmt.Errorf("failed to ensure cache directory exists: %w", err)
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshal parsed chapter cache: %w", err)
	}
	// Use atomic write (tmp file + rename) to prevent a crash or power loss
	// from leaving a corrupted cache file on disk.
	tmpFile, err := os.CreateTemp(cacheDir, "cache-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up the temp file if rename did not consume it.
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("write parsed chapter cache: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp cache file: %w", err)
	}
	if err := os.Rename(tmpPath, cachePath); err != nil {
		return fmt.Errorf("rename temp cache file: %w", err)
	}
	return nil
}
