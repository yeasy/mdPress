package utils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
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

// ImgSrcRegex matches HTML <img> tags and captures the src attribute.
// Exported so that other packages (e.g. internal/output/epub) can reuse
// the same compiled pattern instead of declaring a duplicate.
var ImgSrcRegex = regexp.MustCompile(`<img\s+([^>]*\s+)?src=["']([^"']+)["']([^>]*)>`)

// HTMLTagPattern matches any HTML tag. Useful for stripping tags to obtain
// plain text.  Shared across cmd and internal/output packages.
var HTMLTagPattern = regexp.MustCompile(`<[^>]*>`)

// StripHTMLTags removes all HTML tags from s, returning plain text.
func StripHTMLTags(s string) string {
	return HTMLTagPattern.ReplaceAllString(s, "")
}

const (
	// MaxImageSize is the maximum allowed size for a downloaded image (50 MB).
	MaxImageSize = 50 * 1024 * 1024
	// imageDownloadTimeout is the timeout for downloading a single image.
	imageDownloadTimeout = 30 * time.Second
	// imageDownloadRetryDelay is the delay before retrying a failed image download.
	imageDownloadRetryDelay = 1 * time.Second
)

// imageHTTPClient is a shared HTTP client for image downloads, enabling TCP
// connection reuse across multiple downloads from the same host.
// CheckRedirect validates each redirect target against SSRF rules to prevent
// an attacker from redirecting to internal/cloud-metadata endpoints.
var imageHTTPClient = &http.Client{
	Timeout: imageDownloadTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if ssrfCheckEnabled.Load() {
			if err := checkURLNotPrivate(req.URL); err != nil {
				return fmt.Errorf("redirect blocked by SSRF check: %w", err)
			}
		}
		return nil
	},
}

// ImageExtensionMap maps MIME types to file extensions.
// This is used by multiple packages (epub, etc.) to ensure consistent MIME type handling.
var ImageExtensionMap = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
	"image/bmp":     ".bmp",
	"image/tiff":    ".tiff",
	"image/x-icon":  ".ico",
}

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

	// SSRF prevention: block requests to private/internal network addresses.
	if ssrfCheckEnabled.Load() {
		if err := checkURLNotPrivate(parsedURL); err != nil {
			return "", fmt.Errorf("blocked image download from %q: %w", urlStr, err)
		}
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
	req, err := http.NewRequestWithContext(context.Background(), "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for image download: %w", err)
	}
	resp, err := imageHTTPClient.Do(req)
	if err != nil {
		// Close response body if there was an error but resp is not nil.
		if resp != nil {
			resp.Body.Close() //nolint:errcheck
		}
		// Retry once for transient network errors.
		time.Sleep(imageDownloadRetryDelay)
		// Create a fresh context for the retry instead of reusing the original one,
		// which may have consumed its timeout or been canceled during the first attempt.
		retryCtx, retryCancel := context.WithTimeout(context.Background(), imageDownloadTimeout)
		defer retryCancel() // cancel after response body is fully consumed
		req, err := http.NewRequestWithContext(retryCtx, "GET", urlStr, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request for image download (retry): %w", err)
		}
		resp, err = imageHTTPClient.Do(req)
		if err != nil {
			// Close response body if there was an error on retry but resp is not nil.
			if resp != nil {
				resp.Body.Close() //nolint:errcheck
			}
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
	renamed := false
	defer func() {
		tmpFile.Close() //nolint:errcheck
		if !renamed {
			_ = os.Remove(tmpPath)
		}
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
		// Let defer clean up tmpPath; the winner's file at destPath is valid.
		if !CacheDisabled() {
			if info, statErr := os.Stat(destPath); statErr == nil && !info.IsDir() && info.Size() > 0 {
				return destPath, nil
			}
		}
		return "", fmt.Errorf("failed to move cached image into place: %w", err)
	}
	renamed = true

	return destPath, nil
}

// ImageToBase64 converts an image file to a base64 data URI.
func ImageToBase64(path string) (string, error) {
	// Check file size before reading to prevent OOM on very large images.
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat image file: %w", err)
	}
	if fi.Size() > MaxImageSize {
		return "", fmt.Errorf("image %q exceeds maximum size (%d bytes)", path, MaxImageSize)
	}

	// Read the image file.
	data, err := ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Detect the MIME type from file content first, then fall back to extension.
	mimeType := DetectImageMIME(path, data)

	// For SVG images containing CJK characters, inject CJK font declarations
	// so the text renders correctly when embedded as a data URI in PDF output
	// (where SVG content is isolated from external CSS).
	if mimeType == "image/svg+xml" {
		data = injectSVGCJKFonts(data)
	}

	// Encode as base64.
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Build the data URI.
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
	return dataURI, nil
}

// svgFontFamilyPattern matches font-family attributes in SVG elements.
var svgFontFamilyPattern = regexp.MustCompile(`(font-family=")([^"]*)(")`)

// cjkFontFallback is the CJK font stack appended to SVG font-family attributes
// when CJK characters are detected.  These are common system CJK fonts that
// cover Chinese, Japanese, and Korean.
const cjkFontFallback = `,'PingFang SC','Hiragino Sans GB','Microsoft YaHei','Noto Sans SC','Noto Sans CJK SC','Source Han Sans SC','WenQuanYi Micro Hei'`

// injectSVGCJKFonts adds CJK font families to font-family attributes in SVG
// content when the SVG contains CJK characters.  This ensures CJK text renders
// correctly when the SVG is embedded as a data URI in PDF output, where the SVG
// is isolated from external CSS.
func injectSVGCJKFonts(data []byte) []byte {
	s := string(data)
	if !ContainsCJK(s) {
		return data
	}
	// Append CJK fonts to every font-family attribute in the SVG.
	result := svgFontFamilyPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := svgFontFamilyPattern.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		existing := parts[2]
		// Avoid duplicate injection.
		if strings.Contains(existing, "PingFang SC") || strings.Contains(existing, "Noto Sans SC") {
			return match
		}
		return parts[1] + existing + cjkFontFallback + parts[3]
	})
	return []byte(result)
}

// injectSVGStyleForCJK inserts a <style> element after the opening <svg> tag
// that overrides font-family on all text/tspan elements to include CJK-Embedded.
func injectSVGStyleForCJK(svg string) string {
	// Find the opening <svg tag (skip any XML declaration or DOCTYPE).
	svgIdx := strings.Index(svg, "<svg")
	if svgIdx < 0 {
		return svg
	}
	idx := strings.Index(svg[svgIdx:], ">")
	if idx < 0 {
		return svg
	}
	insertPos := svgIdx + idx + 1
	style := `<style>text,tspan{font-family:"CJK-Embedded",Verdana,Geneva,"DejaVu Sans",sans-serif !important}</style>`
	return svg[:insertPos] + style + svg[insertPos:]
}

// svgTextLengthPattern matches textLength attributes in SVG text elements.
var svgTextLengthPattern = regexp.MustCompile(` textLength="[^"]*"`)

// imgWidthPattern extracts width from an <img> tag's attributes.
var imgWidthPattern = regexp.MustCompile(`width=["'](\d+)["']`)

// svgStartPattern matches the opening <svg tag to inject attributes.
var svgStartPattern = regexp.MustCompile(`<svg\b`)

// tryInlineCJKSVG reads an SVG file and, if it contains CJK characters,
// returns the SVG content inlined directly into the HTML (replacing the <img>
// tag).  Inlining allows the SVG to inherit the page's CJK @font-face rules
// which are inaccessible when the SVG is isolated inside <img src="data:...">.
// Returns "" if the file is not an SVG or contains no CJK text.
func tryInlineCJKSVG(path string, originalImgTag string) string {
	data, err := ReadFile(path)
	if err != nil || !looksLikeSVG(data) {
		return ""
	}
	content := string(data)
	if !ContainsCJK(content) {
		return ""
	}
	// Remove textLength constraints that cause CJK text to be compressed
	// (shields.io calculates textLength for Verdana, which is too narrow for CJK).
	// Inject a <style> inside the SVG to override font-family on text elements
	// with "CJK-Embedded" (the @font-face declared by the PDF generator).
	// Using <style> inside the SVG is more reliable than modifying font-family
	// attributes, as Chrome's PDF backend may not resolve @font-face names
	// from SVG element attributes.
	content = svgTextLengthPattern.ReplaceAllString(content, "")
	content = injectSVGStyleForCJK(content)

	// Extract the <svg>...</svg> portion (skip <?xml ...?> declaration).
	svgIdx := strings.Index(content, "<svg")
	if svgIdx < 0 {
		return ""
	}
	svg := content[svgIdx:]

	// Transfer width from <img> to <svg> for consistent sizing.
	if wm := imgWidthPattern.FindStringSubmatch(originalImgTag); len(wm) >= 2 {
		w := wm[1]
		// Only add width if the SVG doesn't already have one.
		if !strings.Contains(svg[:min(len(svg), 200)], "width=") {
			svg = svgStartPattern.ReplaceAllString(svg, `<svg width="`+w+`"`)
		}
	}
	// Ensure the SVG renders inline (like the <img> it replaces) rather than
	// as a block element which would break inline layouts such as badge rows.
	svg = svgStartPattern.ReplaceAllString(svg, `<svg style="display:inline"`)
	return svg
}

// mimeTypesByExt maps image file extensions to their MIME types.
var mimeTypesByExt = map[string]string{
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".ico":  "image/x-icon",
}

// GetImageMIME returns a MIME type from the file extension.
func GetImageMIME(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Handle .jpeg as an alias for .jpg
	if ext == ".jpeg" {
		return "image/jpeg"
	}

	if mimeType, ok := mimeTypesByExt[ext]; ok {
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
		(bytes.HasPrefix(trimmed, []byte("<?xml")) && bytes.Contains(trimmed, []byte("<svg")))
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
	if options.CacheDir == "" {
		options.CacheDir = filepath.Join(CacheRootDir(), "images")
	}
	if CacheDisabled() {
		options.CacheDir = filepath.Join(os.TempDir(), "mdpress-nocache-images")
	}
	if options.MaxConcurrentDownloads <= 0 {
		options.MaxConcurrentDownloads = 4
	} else if options.MaxConcurrentDownloads > 32 {
		options.MaxConcurrentDownloads = 32
	}
	if options.EmbedRemoteAsBase64 || options.RewriteRemoteToFileURL {
		options.DownloadRemote = true
	}

	matches := ImgSrcRegex.FindAllStringSubmatch(htmlContent, -1)
	remoteImages := prefetchRemoteImages(matches, options)

	result := ImgSrcRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := ImgSrcRegex.FindStringSubmatch(match)
		if len(submatches) < 4 {
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
				if inlined := tryInlineCJKSVG(localPath, match); inlined != "" {
					return inlined
				}
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
			if inlined := tryInlineCJKSVG(targetPath, match); inlined != "" {
				return inlined
			}
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
	if err := EnsureDir(options.CacheDir); err != nil {
		if options.Logger != nil {
			options.Logger.Debug("Failed to ensure cache directory exists", slog.String("dir", options.CacheDir), slog.String("error", err.Error()))
		}
		return nil
	}

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
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in image prefetch goroutine", slog.String("src", src), slog.Any("panic", r))
				}
			}()
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
		return "" // reject absolute paths to prevent reading arbitrary files
	}
	resolved := filepath.Clean(filepath.Join(baseDir, src))
	// Ensure the resolved path stays within baseDir.
	// Use EvalSymlinks to prevent symlink-based containment bypass.
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "" // cannot verify containment; reject
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "" // cannot verify containment; reject
	}
	// Resolve symlinks to prevent containment bypass. Only apply symlink
	// resolution when both paths can be resolved to keep them comparable.
	evaledResolved, errR := filepath.EvalSymlinks(absResolved)
	evaledBase, errB := filepath.EvalSymlinks(absBase)
	if errR == nil && errB == nil {
		absResolved = evaledResolved
		absBase = evaledBase
	}
	if !strings.HasPrefix(absResolved, absBase+string(filepath.Separator)) && absResolved != absBase {
		// Path escapes baseDir, return empty to prevent traversal
		return ""
	}
	return resolved
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
	// Handle alternate MIME type for ico
	if contentType == "image/vnd.microsoft.icon" {
		return ".ico"
	}
	// Look up extension from the shared map
	if ext, ok := ImageExtensionMap[contentType]; ok {
		return ext
	}
	return ""
}

// StableHash returns a stable SHA-256 hex digest for the given strings.
func StableHash(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		// Errors from io.WriteString on hash.Hash are always nil;
		// hash writes never fail.
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ssrfCheckEnabled controls whether DownloadImage blocks private/internal IPs.
// Tests that use local HTTP servers should set this to false via DisableSSRFCheck.
// Uses atomic operations for thread-safety during parallel tests.
var ssrfCheckEnabled atomic.Bool

func init() { ssrfCheckEnabled.Store(true) }

// DisableSSRFCheck disables the SSRF prevention check. Intended for testing only.
func DisableSSRFCheck() { ssrfCheckEnabled.Store(false) }

// EnableSSRFCheck re-enables the SSRF prevention check.
func EnableSSRFCheck() { ssrfCheckEnabled.Store(true) }

// checkURLNotPrivate checks that a URL's hostname does not resolve to a
// private, loopback, or link-local IP address (SSRF prevention).
func checkURLNotPrivate(u *url.URL) error {
	host := u.Hostname()
	if host == "" {
		return errors.New("empty hostname")
	}

	// Block known internal hostnames.
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") {
		return fmt.Errorf("requests to %q are not allowed", host)
	}

	// Resolve and check IP addresses (with timeout to prevent DNS hang).
	dnsCtx, dnsCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dnsCancel()
	ips, err := net.DefaultResolver.LookupHost(dnsCtx, host)
	if err != nil {
		// DNS resolution failure — block the request to prevent DNS rebinding attacks.
		return fmt.Errorf("dns resolution failed for %q: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		// Check IPv4-mapped IPv6 addresses (e.g., ::ffff:10.0.0.1)
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("requests to private/internal address %s are not allowed", ipStr)
		}
	}
	return nil
}
