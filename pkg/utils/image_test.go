package utils

import (
	"os"
	"path/filepath"
	"strings"
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

// TestDownloadImageInvalidURL 测试无效 URL 下载
func TestDownloadImageInvalidURL(t *testing.T) {
	_, err := DownloadImage("not-a-url", t.TempDir())
	if err == nil {
		t.Error("无效 URL 应返回错误")
	}
}
