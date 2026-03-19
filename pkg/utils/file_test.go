package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileExists 测试文件存在检查
func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// 存在的文件
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if !FileExists(tmpFile) {
		t.Error("文件应存在")
	}

	// 不存在的文件
	if FileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("文件不应存在")
	}

	// 存在的目录
	if !FileExists(tmpDir) {
		t.Error("目录应存在")
	}
}

// TestEnsureDir 测试目录创建
func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建新目录
	newDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if !FileExists(newDir) {
		t.Error("目录应已创建")
	}

	// 已存在的目录不应报错
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("已存在的目录不应报错: %v", err)
	}
}

// TestReadFile 测试文件读取
func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 写入测试文件
	content := "测试内容 hello world"
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 正常读取
	data, err := ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容错误: got %q, want %q", string(data), content)
	}
}

// TestReadFileNotExist 测试读取不存在的文件
func TestReadFileNotExist(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("读取不存在的文件应返回错误")
	}
}

// TestReadFileIsDir 测试读取目录
func TestReadFileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadFile(tmpDir)
	if err == nil {
		t.Error("读取目录应返回错误")
	}
}

// TestWriteFile 测试文件写入
func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 写入文件
	filePath := filepath.Join(tmpDir, "output.txt")
	content := "写入的内容"
	if err := WriteFile(filePath, []byte(content)); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	// 验证内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("读取验证失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容错误: got %q", string(data))
	}
}

// TestWriteFileAutoCreateDir 测试写入时自动创建父目录
func TestWriteFileAutoCreateDir(t *testing.T) {
	tmpDir := t.TempDir()

	filePath := filepath.Join(tmpDir, "sub", "dir", "output.txt")
	if err := WriteFile(filePath, []byte("test")); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}

	if !FileExists(filePath) {
		t.Error("文件应已创建")
	}
}

// TestWriteFileOverwrite 测试覆盖写入
func TestWriteFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// 第一次写入
	if err := WriteFile(filePath, []byte("first")); err != nil {
		t.Fatal(err)
	}

	// 覆盖写入
	if err := WriteFile(filePath, []byte("second")); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filePath)
	if string(data) != "second" {
		t.Errorf("覆盖写入失败: got %q", string(data))
	}
}

// TestCopyFile 测试文件复制
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建源文件
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := "source content 源文件"
	if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 复制
	dstPath := filepath.Join(tmpDir, "dest.txt")
	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("复制文件失败: %v", err)
	}

	// 验证
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("复制内容错误: got %q", string(data))
	}
}

// TestCopyFileNonExistent 测试复制不存在的文件
func TestCopyFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	err := CopyFile("/nonexistent/file", filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Error("复制不存在的文件应返回错误")
	}
}

// TestCopyFileIsDir 测试复制目录（应失败）
func TestCopyFileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	os.Mkdir(srcDir, 0755)

	err := CopyFile(srcDir, filepath.Join(tmpDir, "dst"))
	if err == nil {
		t.Error("复制目录应返回错误")
	}
}

// TestCopyFileAutoCreateDstDir 测试复制时自动创建目标目录
func TestCopyFileAutoCreateDstDir(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.txt")
	os.WriteFile(srcPath, []byte("data"), 0644)

	dstPath := filepath.Join(tmpDir, "new", "dir", "dst.txt")
	err := CopyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("复制文件失败: %v", err)
	}
	if !FileExists(dstPath) {
		t.Error("目标文件应已创建")
	}
}

// TestRelPath 测试相对路径计算
func TestRelPath(t *testing.T) {
	tests := []struct {
		base   string
		target string
		want   string
	}{
		{"/home/user", "/home/user/file.txt", "file.txt"},
		{"/home/user", "/home/user/sub/file.txt", "sub/file.txt"},
		{"/home/user", "/home/user", "."},
		{"/home/user/a", "/home/user/b/file.txt", "../b/file.txt"},
	}

	for _, tt := range tests {
		got := RelPath(tt.base, tt.target)
		if got != tt.want {
			t.Errorf("RelPath(%q, %q) = %q, want %q", tt.base, tt.target, got, tt.want)
		}
	}
}

// TestReadWriteRoundTrip 测试读写往返
func TestReadWriteRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "roundtrip.txt")

	// 各种内容
	contents := []string{
		"简单文本",
		"包含\n换行符\n的文本",
		"包含 Unicode: 中文 日本語 한국어 emoji: 🎉",
		"", // 空文件
		"very long " + string(make([]byte, 10000)), // 大文件
	}

	for i, content := range contents {
		if err := WriteFile(filePath, []byte(content)); err != nil {
			t.Fatalf("case %d: 写入失败: %v", i, err)
		}
		data, err := ReadFile(filePath)
		if err != nil {
			t.Fatalf("case %d: 读取失败: %v", i, err)
		}
		if string(data) != content {
			t.Errorf("case %d: 读写不一致", i)
		}
	}
}
