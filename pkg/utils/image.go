package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// fnv32a computes a simple FNV-1a hash to generate unique identifiers from strings.
func fnv32a(s string) uint32 {
	const offset32 = uint32(2166136261)
	const prime32 = uint32(16777619)
	h := offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

// MaxImageSize is the maximum allowed size for a downloaded image (50 MB).
const MaxImageSize = 50 * 1024 * 1024

// IsRemoteURL reports whether a path is an HTTP(S) URL.
func IsRemoteURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// ImageProcessingOptions controls how image references are rewritten.
type ImageProcessingOptions struct {
	EmbedLocalAsBase64     bool
	EmbedRemoteAsBase64    bool
	RewriteLocalToFileURL  bool
	RewriteRemoteToFileURL bool
	DownloadRemote         bool
	CacheDir               string
	MaxConcurrentDownloads int
	Logger                 *slog.Logger
}

// DownloadImage downloads an image from a URL and returns the local path.
func DownloadImage(urlStr string, destDir string) (string, error) {
	// Validate the URL first.
	if !IsRemoteURL(urlStr) {
		return "", fmt.Errorf("invalid URL: %q", urlStr)
	}

	// Parse the URL.
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %q: %w", urlStr, err)
	}

	// Ensure the destination directory exists.
	if err := EnsureDir(destDir); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Derive a unique file name from the URL to avoid collisions.
	fileName := filepath.Base(parsedURL.Path)
	if fileName == "" || fileName == "/" || fileName == "." {
		// Use a hash of the full URL to generate a unique filename.
		fileName = fmt.Sprintf("image-%x.png", fnv32a(urlStr))
	} else {
		// Prefix with a URL hash to avoid collisions when multiple URLs share the same basename.
		ext := filepath.Ext(fileName)
		base := strings.TrimSuffix(fileName, ext)
		fileName = fmt.Sprintf("%s-%x%s", base, fnv32a(urlStr), ext)
	}

	// Build the destination path.
	destPath := filepath.Join(destDir, fileName)
	if !CacheDisabled() {
		if info, err := os.Stat(destPath); err == nil && !info.IsDir() && info.Size() > 0 {
			return destPath, nil
		}
		if matches, err := filepath.Glob(filepath.Join(destDir, strings.TrimSuffix(fileName, filepath.Ext(fileName))+".*")); err == nil {
			for _, match := range matches {
				if info, statErr := os.Stat(match); statErr == nil && !info.IsDir() && info.Size() > 0 {
					return match, nil
				}
			}
		}
	}

	// Download with a timeout so unresponsive servers do not hang forever.
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		// Retry once for transient network errors.
		time.Sleep(1 * time.Second)
		resp, err = client.Get(urlStr)
		if err != nil {
			return "", fmt.Errorf("failed to download image %q (after retry): %w", urlStr, err)
		}
	}
	defer resp.Body.Close() //nolint:errcheck

	// Require a successful HTTP status.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image download failed with HTTP %d %s for %q", resp.StatusCode, http.StatusText(resp.StatusCode), urlStr)
	}

	if filepath.Ext(fileName) == "" {
		if ext := imageExtensionForContentType(resp.Header.Get("Content-Type")); ext != "" {
			fileName += ext
			destPath = filepath.Join(destDir, fileName)
			if !CacheDisabled() {
				if info, err := os.Stat(destPath); err == nil && !info.IsDir() && info.Size() > 0 {
					return destPath, nil
				}
			}
		}
	}

	// Create the destination file.
	tmpFile, err := os.CreateTemp(destDir, "mdpress-image-*")
	if err != nil {
		return "", fmt.Errorf("failed to create file %q: %w", destPath, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close() //nolint:errcheck
		_ = os.Remove(tmpPath)
	}()

	// Copy the response body into the file with a size limit to prevent disk exhaustion.
	limitedReader := io.LimitReader(resp.Body, MaxImageSize+1)
	written, err := io.Copy(tmpFile, limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	if written > MaxImageSize {
		return "", fmt.Errorf("image exceeds maximum allowed size of %d bytes: %q", MaxImageSize, urlStr)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize image download: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		// Another process may have won the race and written the same cache file.
		if !CacheDisabled() {
			if info, statErr := os.Stat(destPath); statErr == nil && !info.IsDir() && info.Size() > 0 {
				return destPath, nil
			}
		}
		return "", fmt.Errorf("failed to move cached image into place: %w", err)
	}

	return destPath, nil
}

// ImageToBase64 converts an image file to a base64 data URI.
func ImageToBase64(path string) (string, error) {
	// Read the image file.
	data, err := ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Detect the MIME type from file content first, then fall back to extension.
	mimeType := DetectImageMIME(path, data)

	// Encode as base64.
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Build the data URI.
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
	return dataURI, nil
}

// GetImageMIME returns a MIME type from the file extension.
func GetImageMIME(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		".ico":  "image/x-icon",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// Default to PNG when the extension is unknown.
	return "image/png"
}

// DetectImageMIME determines the MIME type from content when possible.
// This is more reliable than extension-only detection for remote assets
// whose URLs do not include a file suffix, such as Shields badges.
func DetectImageMIME(path string, data []byte) string {
	if looksLikeSVG(data) {
		return "image/svg+xml"
	}

	if len(data) > 0 {
		sniffLen := len(data)
		if sniffLen > 512 {
			sniffLen = 512
		}
		mimeType := http.DetectContentType(data[:sniffLen])
		if strings.HasPrefix(mimeType, "image/") {
			return mimeType
		}
	}

	return GetImageMIME(path)
}

func looksLikeSVG(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return bytes.HasPrefix(trimmed, []byte("<svg")) ||
		bytes.HasPrefix(trimmed, []byte("<?xml")) && bytes.Contains(trimmed, []byte("<svg"))
}

// ProcessImages resolves image paths in HTML and optionally embeds them as base64.
// An optional logger can be provided to record warnings when image processing fails.
// Pass nil to silently skip failures (legacy behavior).
func ProcessImages(htmlContent string, baseDir string, embedAsBase64 bool, logger ...*slog.Logger) (string, error) {
	var log *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		log = logger[0]
	}
	return ProcessImagesWithOptions(htmlContent, baseDir, ImageProcessingOptions{
		EmbedLocalAsBase64:     embedAsBase64,
		EmbedRemoteAsBase64:    embedAsBase64,
		DownloadRemote:         embedAsBase64,
		CacheDir:               filepath.Join(CacheRootDir(), "images"),
		MaxConcurrentDownloads: 4,
		Logger:                 log,
	})
}

// ProcessImagesWithOptions resolves image paths in HTML with explicit control
// over embedding, remote downloads, and file URL rewriting.
func ProcessImagesWithOptions(htmlContent string, baseDir string, options ImageProcessingOptions) (string, error) {
	imgSrcRegex := regexp.MustCompile(`<img\s+([^>]*\s+)?src=["']([^"']+)["']([^>]*)>`)

	if options.CacheDir == "" {
		options.CacheDir = filepath.Join(CacheRootDir(), "images")
	}
	if CacheDisabled() {
		options.CacheDir = filepath.Join(os.TempDir(), "mdpress-nocache-images")
	}
	if options.MaxConcurrentDownloads <= 0 {
		options.MaxConcurrentDownloads = 4
	}
	if options.EmbedRemoteAsBase64 || options.RewriteRemoteToFileURL {
		options.DownloadRemote = true
	}

	matches := imgSrcRegex.FindAllStringSubmatch(htmlContent, -1)
	remoteImages := prefetchRemoteImages(matches, options)

	result := imgSrcRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := imgSrcRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		src := submatches[2]
		prefix := submatches[1]
		suffix := submatches[3]

		if strings.HasPrefix(src, "data:") {
			return match
		}

		if IsRemoteURL(src) {
			localPath, ok := remoteImages[src]
			if !ok || localPath == "" {
				return match
			}
			switch {
			case options.EmbedRemoteAsBase64:
				dataURI, err := ImageToBase64(localPath)
				if err != nil {
					if options.Logger != nil {
						options.Logger.Warn("Failed to convert remote image to base64", slog.String("src", src), slog.String("error", err.Error()))
					}
					return match
				}
				return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, dataURI, suffix)
			case options.RewriteRemoteToFileURL:
				return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, filePathToURL(localPath), suffix)
			default:
				return match
			}
		}

		targetPath := resolveLocalImagePath(baseDir, src)
		if !FileExists(targetPath) {
			return match
		}

		switch {
		case options.EmbedLocalAsBase64:
			dataURI, err := ImageToBase64(targetPath)
			if err != nil {
				return match
			}
			return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, dataURI, suffix)
		case options.RewriteLocalToFileURL:
			return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, filePathToURL(targetPath), suffix)
		default:
			newSrc := RelPath(baseDir, targetPath)
			return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, newSrc, suffix)
		}
	})

	return result, nil
}

func prefetchRemoteImages(matches [][]string, options ImageProcessingOptions) map[string]string {
	if !options.DownloadRemote {
		return nil
	}
	unique := make(map[string]struct{})
	for _, match := range matches {
		if len(match) >= 3 && IsRemoteURL(match[2]) {
			unique[match[2]] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return nil
	}
	_ = EnsureDir(options.CacheDir)

	results := make(map[string]string, len(unique))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, options.MaxConcurrentDownloads)

	// Capture the logger once before spawning goroutines so that all
	// goroutines use a consistent, safely-captured pointer.
	logger := options.Logger

	for src := range unique {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			localPath, err := DownloadImage(src, options.CacheDir)
			if err != nil {
				if logger != nil {
					logger.Warn("Failed to download remote image, keeping original URL", slog.String("src", src), slog.String("error", err.Error()))
				}
				return
			}
			mu.Lock()
			results[src] = localPath
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	return results
}

func resolveLocalImagePath(baseDir string, src string) string {
	if filepath.IsAbs(src) {
		return src
	}
	return filepath.Clean(filepath.Join(baseDir, src))
}

func filePathToURL(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absPath),
	}
	return u.String()
}

func imageExtensionForContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return ".ico"
	default:
		return ""
	}
}

// StableHash returns a stable SHA-256 hex digest for the given strings.
func StableHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
