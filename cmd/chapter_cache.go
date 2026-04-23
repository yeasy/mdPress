package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/pkg/utils"
)

const parsedChapterCacheVersion = "v2"

type cachedParsedChapter struct {
	HTML        string                 `json:"html"`
	Headings    []markdown.HeadingInfo `json:"headings"`
	Diagnostics []markdown.Diagnostic  `json:"diagnostics"`
}

func parsedChapterCachePath(chapterPath, expandedContent, codeTheme string) string {
	key := utils.StableHash(parsedChapterCacheVersion, chapterPath, codeTheme, expandedContent)
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
