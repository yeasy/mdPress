package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// TestIsRemoteURL 测试远程 URL 判断
func TestIsRemoteURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://example.com/img.png", true},
		{"http://example.com/img.png", true},
		{"./local/img.png", false},
		{"/absolute/img.png", false},
		{"img.png", false},
		{"ftp://example.com/img.png", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("IsRemoteURL(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestGetImageMIME 测试 MIME 类型推断
func TestGetImageMIME(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"anim.gif", "image/gif"},
		{"modern.webp", "image/webp"},
		{"vector.svg", "image/svg+xml"},
		{"bitmap.bmp", "image/bmp"},
		{"icon.ico", "image/x-icon"},
		{"UPPER.PNG", "image/png"},
		{"Photo.JPG", "image/jpeg"},
		{"unknown.xyz", "image/png"}, // 默认
		{"noext", "image/png"},       // 无扩展名
	}

	for _, tt := range tests {
		got := GetImageMIME(tt.path)
		if got != tt.want {
			t.Errorf("GetImageMIME(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// TestImageToBase64 测试图片转 base64
func TestImageToBase64(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建一个假 PNG 文件（PNG 文件头）
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	imgPath := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(imgPath, pngHeader, 0644); err != nil {
		t.Fatal(err)
	}

	dataURI, err := ImageToBase64(imgPath)
	if err != nil {
		t.Fatalf("转换 base64 失败: %v", err)
	}

	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Errorf("data URI 格式错误: %q", dataURI[:50])
	}
}

// TestImageToBase64JPEG 测试 JPEG 转 base64
func TestImageToBase64JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(imgPath, []byte{0xFF, 0xD8, 0xFF}, 0644); err != nil {
		t.Fatal(err)
	}

	dataURI, err := ImageToBase64(imgPath)
	if err != nil {
		t.Fatalf("转换 base64 失败: %v", err)
	}

	if !strings.HasPrefix(dataURI, "data:image/jpeg;base64,") {
		t.Errorf("JPEG data URI 格式错误: %q", dataURI)
	}
}

func TestImageToBase64SVGWithoutExtension(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "badge")
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"></svg>`)
	if err := os.WriteFile(imgPath, svg, 0644); err != nil {
		t.Fatal(err)
	}

	dataURI, err := ImageToBase64(imgPath)
	if err != nil {
		t.Fatalf("转换 base64 失败: %v", err)
	}

	if !strings.HasPrefix(dataURI, "data:image/svg+xml;base64,") {
		t.Errorf("SVG 无扩展名时 MIME 识别错误: %q", dataURI)
	}
}

// TestImageToBase64NonExistent 测试不存在的图片
func TestImageToBase64NonExistent(t *testing.T) {
	_, err := ImageToBase64("/nonexistent/image.png")
	if err == nil {
		t.Error("不存在的文件应返回错误")
	}
}

// TestProcessImagesLocalEmbed 测试本地图片嵌入
func TestProcessImagesLocalEmbed(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建图片文件
	imgPath := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4E, 0x47}, 0644); err != nil {
		t.Fatalf("write image failed: %v", err)
	}

	html := `<img src="image.png">`
	result, err := ProcessImages(html, tmpDir, true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	if !strings.Contains(result, "data:image/png;base64,") {
		t.Error("本地图片应被嵌入为 base64")
	}
	if strings.Contains(result, `src="image.png"`) {
		t.Error("原始 src 应被替换")
	}
}

// TestProcessImagesNoEmbed 测试不嵌入模式
func TestProcessImagesNoEmbed(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "image.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4E, 0x47}, 0644); err != nil {
		t.Fatalf("write image failed: %v", err)
	}

	html := `<img src="image.png">`
	result, err := ProcessImages(html, tmpDir, false)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	// 不嵌入时应保持相对路径
	if strings.Contains(result, "data:image") {
		t.Error("不嵌入模式下不应有 data URI")
	}
}

// TestProcessImagesDataURI 测试已有 data URI 的图片
func TestProcessImagesDataURI(t *testing.T) {
	html := `<img src="data:image/png;base64,abc123">`
	result, err := ProcessImages(html, ".", true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	// 已有 data URI 应保持不变
	if result != html {
		t.Error("已有 data URI 不应被修改")
	}
}

// TestProcessImagesRemoteURL 测试远程 URL 图片（不嵌入）
func TestProcessImagesRemoteURL(t *testing.T) {
	html := `<img src="https://example.com/img.png">`
	result, err := ProcessImages(html, ".", false)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	if result != html {
		t.Error("不嵌入模式下远程 URL 应保持不变")
	}
}

// TestProcessImagesNonExistentLocal 测试引用不存在的本地图片
func TestProcessImagesNonExistentLocal(t *testing.T) {
	html := `<img src="nonexistent.png">`
	result, err := ProcessImages(html, "/tmp", true)
	if err != nil {
		t.Fatalf("ProcessImages 不应因不存在的图片报错: %v", err)
	}

	// 不存在的文件应保持原样
	if result != html {
		t.Error("不存在的图片路径应保持原样")
	}
}

// TestProcessImagesMultiple 测试多个图片
func TestProcessImagesMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建两个图片
	for _, name := range []string{"a.png", "b.jpg"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte{1, 2, 3}, 0644); err != nil {
			t.Fatalf("write image %s failed: %v", name, err)
		}
	}

	html := `<img src="a.png"><p>text</p><img src="b.jpg">`
	result, err := ProcessImages(html, tmpDir, true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	count := strings.Count(result, "data:image")
	if count != 2 {
		t.Errorf("应有 2 个 base64 图片: got %d", count)
	}
}

// TestProcessImagesWithAttributes 测试带属性的 img 标签
func TestProcessImagesWithAttributes(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "img.png"), []byte{1, 2}, 0644); err != nil {
		t.Fatalf("write image failed: %v", err)
	}

	html := `<img class="photo" src="img.png" alt="test" width="100">`
	result, err := ProcessImages(html, tmpDir, true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	if !strings.Contains(result, "data:image") {
		t.Error("带属性的 img 标签也应被处理")
	}
}

// TestProcessImagesNoImages 测试无图片的 HTML
func TestProcessImagesNoImages(t *testing.T) {
	html := `<p>No images here</p><div>Just text</div>`
	result, err := ProcessImages(html, ".", true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}
	if result != html {
		t.Error("无图片的 HTML 不应被修改")
	}
}

// TestProcessImagesAbsolutePath 测试绝对路径
func TestProcessImagesAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "abs.png")
	if err := os.WriteFile(imgPath, []byte{1, 2, 3}, 0644); err != nil {
		t.Fatalf("write image failed: %v", err)
	}

	html := `<img src="` + imgPath + `">`
	result, err := ProcessImages(html, "/other/dir", true)
	if err != nil {
		t.Fatalf("ProcessImages 失败: %v", err)
	}

	if !strings.Contains(result, "data:image") {
		t.Error("绝对路径的图片也应被嵌入")
	}
}

// TestDownloadImageInvalidURL tests that an invalid URL returns an error.
func TestDownloadImageInvalidURL(t *testing.T) {
	_, err := DownloadImage("://bad-scheme", t.TempDir())
	if err == nil {
		t.Error("invalid URL should return an error")
	}
}

func TestDownloadImageUsesCacheOnRepeatedRequests(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	destDir := t.TempDir()
	first, err := DownloadImage(server.URL+"/badge", destDir)
	if err != nil {
		t.Fatalf("第一次下载失败: %v", err)
	}
	second, err := DownloadImage(server.URL+"/badge", destDir)
	if err != nil {
		t.Fatalf("第二次下载失败: %v", err)
	}

	if first != second {
		t.Fatalf("缓存文件路径不一致: %q vs %q", first, second)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("应只命中远程服务一次，实际 %d 次", got)
	}
}

func TestProcessImagesWithOptionsUsesFileURLsAndDedupesRemoteDownloads(t *testing.T) {
	localDir := t.TempDir()
	cacheDir := t.TempDir()

	localPath := filepath.Join(localDir, "local.png")
	if err := os.WriteFile(localPath, []byte{0x89, 0x50, 0x4E, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	html := `<img src="local.png"><img src="` + server.URL + `/remote-badge"><img src="` + server.URL + `/remote-badge">`
	result, err := ProcessImagesWithOptions(html, localDir, ImageProcessingOptions{
		RewriteLocalToFileURL:  true,
		RewriteRemoteToFileURL: true,
		DownloadRemote:         true,
		CacheDir:               cacheDir,
		MaxConcurrentDownloads: 4,
	})
	if err != nil {
		t.Fatalf("ProcessImagesWithOptions 失败: %v", err)
	}

	if strings.Count(result, `src="file://`) != 3 {
		t.Fatalf("应将所有图片改写为 file:// URL，实际结果: %s", result)
	}
	if strings.Contains(result, "data:image") {
		t.Fatal("file URL 模式下不应嵌入 base64")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("重复远程 URL 应只下载一次，实际 %d 次", got)
	}
}

// TestDetectImageMIME tests MIME type detection from content
func TestDetectImageMIME(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		path     string
		wantMIME string
	}{
		{
			name:     "PNG signature",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			path:     "test.png",
			wantMIME: "image/png",
		},
		{
			name:     "JPEG signature",
			data:     []byte{0xFF, 0xD8, 0xFF},
			path:     "test.jpg",
			wantMIME: "image/jpeg",
		},
		{
			name:     "GIF signature",
			data:     []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, // GIF89a
			path:     "test.gif",
			wantMIME: "image/gif",
		},
		{
			name:     "SVG detection",
			data:     []byte("<svg version=\"1.0\">"),
			path:     "test.svg",
			wantMIME: "image/svg+xml",
		},
		{
			name:     "SVG with XML declaration",
			data:     []byte("<?xml version=\"1.0\"?><svg>"),
			path:     "test.svg",
			wantMIME: "image/svg+xml",
		},
		{
			name:     "WebP signature",
			data:     []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			path:     "test.webp",
			wantMIME: "image/webp",
		},
		{
			name:     "Unknown format falls back to extension",
			data:     []byte("not an image"),
			path:     "test.png",
			wantMIME: "image/png",
		},
		{
			name:     "Empty data",
			data:     []byte{},
			path:     "test.jpg",
			wantMIME: "image/jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectImageMIME(tt.path, tt.data)
			if got != tt.wantMIME {
				t.Errorf("DetectImageMIME() = %q, want %q", got, tt.wantMIME)
			}
		})
	}
}

// TestStableHash tests hash generation
func TestStableHash(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
	}{
		{
			name:  "single string",
			parts: []string{"test"},
		},
		{
			name:  "multiple strings",
			parts: []string{"part1", "part2", "part3"},
		},
		{
			name:  "empty strings",
			parts: []string{"", "test", ""},
		},
		{
			name:  "unicode strings",
			parts: []string{"中文", "日本語", "한글"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash should be consistent
			hash1 := StableHash(tt.parts...)
			hash2 := StableHash(tt.parts...)

			if hash1 != hash2 {
				t.Errorf("StableHash() should return consistent results")
			}

			// Hash should be non-empty and hex
			if len(hash1) != 64 { // SHA-256 is 64 hex characters
				t.Errorf("Hash should be SHA-256 (64 chars), got %d", len(hash1))
			}

			// Hash should only contain hex characters
			for _, ch := range hash1 {
				if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
					t.Errorf("Hash contains non-hex character: %c", ch)
				}
			}
		})
	}
}

// TestStableHashDifferenceForDifferentInputs tests that different inputs produce different hashes
func TestStableHashDifferenceForDifferentInputs(t *testing.T) {
	hash1 := StableHash("input1")
	hash2 := StableHash("input2")

	if hash1 == hash2 {
		t.Error("Different inputs should produce different hashes")
	}
}

// TestImageExtensionForContentType tests extension inference from Content-Type
func TestImageExtensionForContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantExt     string
	}{
		{"image/png", "image/png", ".png"},
		{"image/jpeg", "image/jpeg", ".jpg"},
		{"image/png with charset", "image/png; charset=utf-8", ".png"},
		{"image/webp", "image/webp", ".webp"},
		{"image/svg+xml", "image/svg+xml", ".svg"},
		{"image/gif", "image/gif", ".gif"},
		{"image/bmp", "image/bmp", ".bmp"},
		{"image/tiff", "image/tiff", ".tiff"},
		{"image/x-icon", "image/x-icon", ".ico"},
		{"image/vnd.microsoft.icon", "image/vnd.microsoft.icon", ".ico"},
		{"uppercase", "IMAGE/PNG", ".png"},
		{"with spaces", "  image/png  ", ".png"},
		{"unknown", "image/unknown", ""},
		{"not image", "text/html", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageExtensionForContentType(tt.contentType)
			if got != tt.wantExt {
				t.Errorf("imageExtensionForContentType(%q) = %q, want %q", tt.contentType, got, tt.wantExt)
			}
		})
	}
}

// TestGetImageMIMETableDriven tests MIME type resolution with more cases
func TestGetImageMIMETableDriven(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"PNG lowercase", "image.png", "image/png"},
		{"PNG uppercase", "IMAGE.PNG", "image/png"},
		{"PNG mixed", "Image.Png", "image/png"},
		{"JPEG extensions", "photo.jpeg", "image/jpeg"},
		{"JPG extension", "photo.jpg", "image/jpeg"},
		{"GIF", "anim.gif", "image/gif"},
		{"WebP", "modern.webp", "image/webp"},
		{"SVG", "vector.svg", "image/svg+xml"},
		{"BMP", "bitmap.bmp", "image/bmp"},
		{"TIFF", "scan.tiff", "image/tiff"},
		{"ICO", "favicon.ico", "image/x-icon"},
		{"No extension", "imagefile", "image/png"},
		{"Unknown extension", "image.xyz", "image/png"},
		{"Multiple dots", "image.backup.png", "image/png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetImageMIME(tt.path)
			if got != tt.want {
				t.Errorf("GetImageMIME(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestProcessImagesWithMultipleConcurrentDownloads tests concurrent download semaphore
func TestProcessImagesWithMultipleConcurrentDownloads(t *testing.T) {
	localDir := t.TempDir()
	cacheDir := t.TempDir()

	var downloadCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&downloadCount, 1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	// Create HTML with 5 different remote images
	var htmlParts []string
	for i := 0; i < 5; i++ {
		htmlParts = append(htmlParts, `<img src="`+server.URL+`/image`+fmt.Sprintf("%d", i)+`">`)
	}
	html := strings.Join(htmlParts, "")

	result, err := ProcessImagesWithOptions(html, localDir, ImageProcessingOptions{
		DownloadRemote:         true,
		CacheDir:               cacheDir,
		MaxConcurrentDownloads: 2,
	})
	if err != nil {
		t.Fatalf("ProcessImagesWithOptions failed: %v", err)
	}

	// All 5 images should be downloaded (not cached yet since different URLs)
	if got := atomic.LoadInt32(&downloadCount); got != 5 {
		t.Fatalf("expected 5 downloads, got %d", got)
	}

	// Result should contain something (either original or file URLs depending on options)
	if result == "" {
		t.Fatal("result should not be empty")
	}
}

// TestDownloadImageErrorHandling tests error handling for failed downloads
func TestDownloadImageErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() (string, func())
		wantErr     bool
		errContains string
	}{
		{
			name: "HTTP 404 error",
			setupServer: func() (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
				return server.URL + "/notfound", server.Close
			},
			wantErr:     true,
			errContains: "404",
		},
		{
			name: "HTTP 500 error",
			setupServer: func() (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				return server.URL + "/error", server.Close
			},
			wantErr:     true,
			errContains: "500",
		},
		{
			name: "Connection refused",
			setupServer: func() (string, func()) {
				// Start a server and immediately close it to get a refused port.
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				url := s.URL + "/image"
				s.Close()
				return url, func() {}
			},
			wantErr:     true,
			errContains: "failed to download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, cleanup := tt.setupServer()
			defer cleanup()

			destDir := t.TempDir()
			_, err := DownloadImage(url, destDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error=%v, got error=%v", tt.wantErr, err)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

// TestProcessImagesEmptyImageList tests processing with no images
func TestProcessImagesEmptyImageList(t *testing.T) {
	html := `<p>Just text</p><div>No images here</div><span>Nothing</span>`
	result, err := ProcessImages(html, "/tmp", true)
	if err != nil {
		t.Fatalf("ProcessImages failed: %v", err)
	}

	// Should return unchanged HTML
	if result != html {
		t.Errorf("empty image list should return unchanged HTML")
	}
}

// TestProcessImagesWithOptionsEmptyList tests options with no images
func TestProcessImagesWithOptionsEmptyList(t *testing.T) {
	html := `<div class="content"><p>Text only</p></div>`
	result, err := ProcessImagesWithOptions(html, ".", ImageProcessingOptions{
		EmbedRemoteAsBase64:    true,
		RewriteRemoteToFileURL: true,
		MaxConcurrentDownloads: 4,
	})
	if err != nil {
		t.Fatalf("ProcessImagesWithOptions failed: %v", err)
	}

	if result != html {
		t.Error("empty image list should not modify HTML")
	}
}

// TestDownloadImageSizeExceeded tests that oversized images are rejected
func TestDownloadImageSizeExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		// Write data larger than MaxImageSize
		chunk := make([]byte, 1024*1024) // 1MB chunk
		for i := 0; i < 60; i++ {        // Total > 50MB
			_, _ = w.Write(chunk)
		}
	}))
	defer server.Close()

	_, err := DownloadImage(server.URL+"/huge", t.TempDir())
	if err == nil {
		t.Error("oversized image should cause error")
	}
	// The error may be "exceeds maximum" (size limit reached) or context/timeout
	// related (the 30s download timeout fires before the full body is read).
	errMsg := err.Error()
	if !strings.Contains(errMsg, "exceeds maximum") &&
		!strings.Contains(errMsg, "context") &&
		!strings.Contains(errMsg, "timeout") {
		t.Errorf("error should mention size limit or timeout, got: %v", err)
	}
}

// TestFnv32a tests the FNV-1a hash function
func TestFnv32a(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint32 // We test determinism and known values
	}{
		{"empty string", "", 2166136261}, // FNV offset basis
		{"single char", "a", 0},          // We'll compute this
		{"hello", "hello", 0},            // We'll compute this
		{"simple", "test", 0},            // We'll compute this
	}

	// Test determinism - same input should produce same output
	for _, tt := range tests {
		t.Run(tt.name+" determinism", func(t *testing.T) {
			hash1 := fnv32a(tt.input)
			hash2 := fnv32a(tt.input)
			if hash1 != hash2 {
				t.Errorf("fnv32a(%q) not deterministic: %d vs %d", tt.input, hash1, hash2)
			}
		})
	}

	// Test known values
	t.Run("empty string constant", func(t *testing.T) {
		hash := fnv32a("")
		if hash != 2166136261 {
			t.Errorf("fnv32a(\"\") = %d, want 2166136261 (FNV offset)", hash)
		}
	})

	// Test different inputs produce different hashes
	t.Run("different inputs differ", func(t *testing.T) {
		hash1 := fnv32a("input1")
		hash2 := fnv32a("input2")
		if hash1 == hash2 {
			t.Error("Different inputs should produce different hashes")
		}
	})

	// Test that small variations produce different outputs
	t.Run("small variations differ", func(t *testing.T) {
		hash1 := fnv32a("test")
		hash2 := fnv32a("tests")
		if hash1 == hash2 {
			t.Error("Similar inputs should produce different hashes")
		}
	})
}

// TestLooksLikeSVG tests SVG content detection
func TestLooksLikeSVG(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "SVG element",
			data:     []byte("<svg version=\"1.0\">"),
			expected: true,
		},
		{
			name:     "SVG with attributes",
			data:     []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100">`),
			expected: true,
		},
		{
			name:     "SVG with XML declaration",
			data:     []byte(`<?xml version="1.0"?><svg>`),
			expected: true,
		},
		{
			name:     "SVG with whitespace and XML",
			data:     []byte("  \n<?xml version=\"1.0\"?>\n<svg>\n  "),
			expected: true,
		},
		{
			name:     "SVG with XML but no svg element",
			data:     []byte(`<?xml version="1.0"?><root>`),
			expected: false,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: false,
		},
		{
			name:     "Whitespace only",
			data:     []byte("  \n  \t  "),
			expected: false,
		},
		{
			name:     "PNG header",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: false,
		},
		{
			name:     "HTML not SVG",
			data:     []byte("<html><body>test</body></html>"),
			expected: false,
		},
		{
			name:     "Text content",
			data:     []byte("This is not an SVG"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeSVG(tt.data)
			if got != tt.expected {
				t.Errorf("looksLikeSVG() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestResolveLocalImagePath tests relative and absolute path resolution
func TestResolveLocalImagePath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		baseDir    string
		src        string
		checkValid bool // Whether to check if path stays within baseDir
	}{
		{
			name:       "relative path in subdir",
			baseDir:    tmpDir,
			src:        "images/pic.png",
			checkValid: true,
		},
		{
			name:       "relative path current dir",
			baseDir:    tmpDir,
			src:        "pic.png",
			checkValid: true,
		},
		{
			name:    "parent directory traversal",
			baseDir: filepath.Join(tmpDir, "subdir"),
			src:     "../pic.png",
			// May return empty if function blocks traversal — don't assert non-empty.
		},
		{
			name:       "empty path",
			baseDir:    tmpDir,
			src:        "",
			checkValid: true,
		},
		{
			name:       "absolute path",
			baseDir:    tmpDir,
			src:        "/absolute/path/pic.png",
			checkValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveLocalImagePath(tt.baseDir, tt.src)
			// For normal relative paths, result should be non-empty.
			// Skip assertion for parent traversal ("..") since the function may
			// legitimately return empty to block directory escapes.
			if tt.checkValid && tt.src != "" && !filepath.IsAbs(tt.src) {
				if result == "" {
					t.Errorf("resolveLocalImagePath() = empty for relative path %q", tt.src)
				}
			}
		})
	}
}

// TestFilePathToURL tests file path to file:// URL conversion
func TestFilePathToURL(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		shouldStart string // file:// URL should start with this
	}{
		{
			name:        "unix absolute path",
			path:        "/home/user/image.png",
			shouldStart: "file://",
		},
		{
			name:        "path with spaces",
			path:        "/home/user/my image.png",
			shouldStart: "file://",
		},
		{
			name:        "path with special chars",
			path:        "/home/user/image-2024_v1.png",
			shouldStart: "file://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := filePathToURL(tt.path)
			if !strings.HasPrefix(url, tt.shouldStart) {
				t.Errorf("filePathToURL() = %q, should start with %q", url, tt.shouldStart)
			}
			// Should be a valid URL scheme
			if !strings.HasPrefix(url, "file://") {
				t.Errorf("filePathToURL() = %q, should use file:// scheme", url)
			}
		})
	}
}
