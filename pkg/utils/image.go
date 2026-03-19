package utils

import (
	"bytes"
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

	// Create the destination file.
	file, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %q: %w", destPath, err)
	}
	defer file.Close() //nolint:errcheck

	// Copy the response body into the file with a size limit to prevent disk exhaustion.
	limitedReader := io.LimitReader(resp.Body, MaxImageSize+1)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	if written > MaxImageSize {
		_ = os.Remove(destPath)
		return "", fmt.Errorf("image exceeds maximum allowed size of %d bytes: %q", MaxImageSize, urlStr)
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
	// Match the src attribute of img tags.
	imgSrcRegex := regexp.MustCompile(`<img\s+([^>]*\s+)?src=["']([^"']+)["']([^>]*)>`)

	result := imgSrcRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		// Extract the src value.
		matches := imgSrcRegex.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		src := matches[2]
		prefix := matches[1]
		suffix := matches[3]

		// Keep existing data URIs unchanged.
		if strings.HasPrefix(src, "data:") {
			return match
		}

		// Handle remote URLs.
		if IsRemoteURL(src) {
			if embedAsBase64 {
				// Download the remote image and convert it to base64.
				tempDir := filepath.Join(baseDir, ".mdpress_temp")
				localPath, err := DownloadImage(src, tempDir)
				if err != nil {
					if log != nil {
						log.Warn("Failed to download remote image, keeping original URL", slog.String("src", src), slog.String("error", err.Error()))
					}
					return match
				}

				dataURI, err := ImageToBase64(localPath)
				if err != nil {
					if log != nil {
						log.Warn("Failed to convert image to base64", slog.String("src", src), slog.String("error", err.Error()))
					}
					return match
				}

				return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, dataURI, suffix)
			}
			// Leave remote URLs untouched when embedding is disabled.
			return match
		}

		// Resolve relative paths against baseDir.
		var targetPath string
		if filepath.IsAbs(src) {
			targetPath = src
		} else {
			targetPath = filepath.Clean(filepath.Join(baseDir, src))
		}

		// Keep missing local files unchanged.
		if !FileExists(targetPath) {
			return match
		}

		if embedAsBase64 {
			// Convert local files to base64 when requested.
			dataURI, err := ImageToBase64(targetPath)
			if err != nil {
				// Keep the original source when conversion fails.
				return match
			}
			return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, dataURI, suffix)
		}

		// Otherwise rewrite the source with a normalized relative path.
		newSrc := RelPath(baseDir, targetPath)
		return fmt.Sprintf(`<img %ssrc="%s"%s>`, prefix, newSrc, suffix)
	})

	return result, nil
}
