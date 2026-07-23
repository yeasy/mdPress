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

func parsedChapterCachePath(chapterPath, expandedContent, codeTheme string) string {
	key := utils.StableHash(parsedChapterCacheVersion, rendererFingerprint(), chapterPath, codeTheme, expandedContent)
	return filepath.Join(utils.CacheRootDir(), "parsed-chapters", key[:2], key+".json")
}

func loadParsedChapterCache(chapterPath, expandedContent, codeTheme string) (*cachedParsedChapter, bool, error) {
	if utils.CacheDisabled() {
		return nil, false, nil
	}
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
