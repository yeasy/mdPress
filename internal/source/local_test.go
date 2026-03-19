package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewLocalSource 测试本地源创建
func TestNewLocalSource(t *testing.T) {
	tests := []struct {
		name string
		path string
		opts Options
	}{
		{
			name: "简单路径",
			path: "/home/user/project",
			opts: Options{},
		},
		{
			name: "路径带子目录选项",
			path: "/home/user/project",
			opts: Options{SubDir: "docs"},
		},
		{
			name: "相对路径",
			path: "./project",
			opts: Options{},
		},
		{
			name: "当前目录",
			path: ".",
			opts: Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewLocalSource(tt.path, tt.opts)
			if src == nil {
				t.Fatal("NewLocalSource returned nil")
				return
			}

			if src.path != tt.path {
				t.Errorf("path = %q, want %q", src.path, tt.path)
			}

			if src.opts != tt.opts {
				t.Errorf("opts = %v, want %v", src.opts, tt.opts)
			}
		})
	}
}

// TestLocalSourceType 测试本地源类型
func TestLocalSourceType(t *testing.T) {
	src := NewLocalSource("/tmp/test", Options{})
	if src.Type() != "local" {
		t.Errorf("Type() = %q, want %q", src.Type(), "local")
	}
}

// TestLocalSourceCleanup 测试本地源清理（应安全返回 nil）
func TestLocalSourceCleanup(t *testing.T) {
	src := NewLocalSource("/tmp/test", Options{})
	err := src.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() should return nil for local source, got %v", err)
	}
}

// TestLocalSourcePrepare 测试本地源准备函数
func TestLocalSourcePrepare(t *testing.T) {
	// 使用临时目录进行测试
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() string // 返回要使用的路径
		opts    Options
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		// 成功情况
		{
			name: "现有目录",
			setup: func() string {
				return tempDir
			},
			opts:    Options{},
			wantErr: false,
			check: func(t *testing.T, result string) {
				if result == "" {
					t.Error("Prepare should return non-empty path for existing dir")
				}
				// 应该返回绝对路径
				if !filepath.IsAbs(result) {
					t.Errorf("path should be absolute, got %q", result)
				}
			},
		},
		{
			name: "相对路径的现有目录",
			setup: func() string {
				// 创建临时子目录
				subDir := filepath.Join(tempDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("mkdir subdir failed: %v", err)
				}
				return subDir
			},
			opts:    Options{},
			wantErr: false,
			check: func(t *testing.T, result string) {
				if !filepath.IsAbs(result) {
					t.Errorf("path should be absolute, got %q", result)
				}
			},
		},
		// 错误情况
		{
			name: "不存在的路径",
			setup: func() string {
				return "/nonexistent/path/that/should/not/exist"
			},
			opts:    Options{},
			wantErr: true,
			check:   nil,
		},
		{
			name: "文件而非目录",
			setup: func() string {
				filePath := filepath.Join(tempDir, "testfile.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("write test file failed: %v", err)
				}
				return filePath
			},
			opts:    Options{},
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			src := NewLocalSource(path, tt.opts)

			result, err := src.Prepare()

			if tt.wantErr {
				if err == nil {
					t.Error("Prepare() should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("Prepare() unexpected error: %v", err)
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestLocalSourcePrepareWithSubDir 测试带子目录选项的本地源准备
func TestLocalSourcePrepareWithSubDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setup     func() (baseDir, subDir string) // 返回基础路径和子目录
		opts      Options
		wantErr   bool
		checkPath func(t *testing.T, path string, subDir string)
	}{
		{
			name: "存在的子目录",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "docs"
				if err := os.MkdirAll(filepath.Join(baseDir, subDir), 0755); err != nil {
					t.Fatalf("mkdir docs failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:    Options{SubDir: "docs"},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				if !filepath.IsAbs(path) {
					t.Errorf("path should be absolute, got %q", path)
				}
				if !strings.HasSuffix(path, subDir) {
					t.Errorf("path should end with %q, got %q", subDir, path)
				}
			},
		},
		{
			name: "不存在的子目录",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "nonexistent_subdir"
				return baseDir, subDir
			},
			opts:      Options{SubDir: "nonexistent_subdir"},
			wantErr:   true,
			checkPath: nil,
		},
		{
			name: "子目录是文件而非目录",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "file.txt"
				if err := os.WriteFile(filepath.Join(baseDir, subDir), []byte("test"), 0644); err != nil {
					t.Fatalf("write file.txt failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:      Options{SubDir: "file.txt"},
			wantErr:   true,
			checkPath: nil,
		},
		{
			name: "嵌套子目录",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = "a/b/c"
				if err := os.MkdirAll(filepath.Join(baseDir, subDir), 0755); err != nil {
					t.Fatalf("mkdir nested subdir failed: %v", err)
				}
				return baseDir, subDir
			},
			opts:    Options{SubDir: "a/b/c"},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				if !strings.HasSuffix(path, filepath.Join("a", "b", "c")) {
					t.Errorf("path should contain nested dir, got %q", path)
				}
			},
		},
		{
			name: "空子目录名称（应被忽略）",
			setup: func() (baseDir, subDir string) {
				baseDir = tempDir
				subDir = ""
				return baseDir, subDir
			},
			opts:    Options{SubDir: ""},
			wantErr: false,
			checkPath: func(t *testing.T, path string, subDir string) {
				// 空子目录名称应返回基础目录本身
				absPath, _ := filepath.Abs(tempDir)
				if path != absPath {
					t.Errorf("path with empty SubDir should be base path, got %q, want %q", path, absPath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir, subDir := tt.setup()
			src := NewLocalSource(baseDir, tt.opts)

			result, err := src.Prepare()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Prepare() should return error for %s", tt.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Prepare() unexpected error: %v", err)
				return
			}

			if tt.checkPath != nil {
				tt.checkPath(t, result, subDir)
			}
		})
	}
}
