// boundary_test.go 补充 source 包的边界测试。
// 覆盖无效 URL、权限问题、特殊路径等边界场景。
package source

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDetect_InvalidGitHubURLs 测试无效 GitHub URL 的错误处理
func TestDetect_InvalidGitHubURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"仅域名", "https://github.com"},
		{"仅 owner", "https://github.com/owner"},
		{"带查询参数", "https://github.com/owner/repo?tab=readme"},
		{"带锚点", "https://github.com/owner/repo#readme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})
			// 如果识别为 GitHub URL 但解析失败，应返回错误
			// 如果不识别为 GitHub URL，应返回 local 类型
			if err != nil {
				t.Logf("正确返回错误: %v", err)
				return
			}
			if src != nil {
				t.Logf("类型: %s（作为本地路径处理）", src.Type())
			}
		})
	}
}

// TestLocalSource_PermissionDenied 测试无权限目录的处理
func TestLocalSource_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-permission semantics required; skipping on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root 用户无法测试权限限制")
	}

	// 创建无读取权限的目录
	tempDir := t.TempDir()
	noReadDir := filepath.Join(tempDir, "noperm")
	if err := os.MkdirAll(noReadDir, 0000); err != nil {
		t.Fatalf("创建无权限目录失败: %v", err)
	}
	defer func() {
		if err := os.Chmod(noReadDir, 0755); err != nil {
			t.Logf("恢复目录权限失败: %v", err)
		}
	}()

	src := NewLocalSource(noReadDir, Options{})
	result, err := src.Prepare()

	// 路径存在但可能因权限返回目录（Prepare 只检查 Stat）
	// 只要不 panic 即可
	t.Logf("无权限目录: result=%q, err=%v", result, err)
}

// TestLocalSource_SymlinkPath 测试符号链接路径
func TestLocalSource_SymlinkPath(t *testing.T) {
	tempDir := t.TempDir()

	// 创建实际目录
	realDir := filepath.Join(tempDir, "real")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatalf("创建实际目录失败: %v", err)
	}

	// 创建符号链接
	linkDir := filepath.Join(tempDir, "link")
	err := os.Symlink(realDir, linkDir)
	if err != nil {
		t.Skip("无法创建符号链接")
	}

	src := NewLocalSource(linkDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Errorf("符号链接路径应可正常使用: %v", err)
	}
	if result == "" {
		t.Error("符号链接路径应返回非空结果")
	}
}

// TestLocalSource_SubDirPermission 测试子目录权限问题
func TestLocalSource_SubDirPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file-permission semantics required; skipping on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root 用户无法测试权限限制")
	}

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "restricted")
	if err := os.MkdirAll(subDir, 0000); err != nil {
		t.Fatalf("创建受限子目录失败: %v", err)
	}
	defer func() {
		if err := os.Chmod(subDir, 0755); err != nil {
			t.Logf("恢复子目录权限失败: %v", err)
		}
	}()

	src := NewLocalSource(tempDir, Options{SubDir: "restricted"})
	result, err := src.Prepare()

	// 子目录存在但可能无法访问
	t.Logf("受限子目录: result=%q, err=%v", result, err)
}

// TestLocalSource_DeepNestedPath 测试深层嵌套路径
func TestLocalSource_DeepNestedPath(t *testing.T) {
	tempDir := t.TempDir()

	// 创建深层嵌套目录
	deepPath := filepath.Join(tempDir, "a", "b", "c", "d", "e", "f")
	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatalf("创建深层嵌套目录失败: %v", err)
	}

	src := NewLocalSource(tempDir, Options{SubDir: "a/b/c/d/e/f"})
	result, err := src.Prepare()
	if err != nil {
		t.Errorf("深层嵌套目录应可正常使用: %v", err)
	}
	if !strings.HasSuffix(result, filepath.Join("a", "b", "c", "d", "e", "f")) {
		t.Errorf("路径应包含完整嵌套路径, 实际: %q", result)
	}
}

// TestLocalSource_SpecialCharsInPath 测试路径中包含特殊字符
func TestLocalSource_SpecialCharsInPath(t *testing.T) {
	tempDir := t.TempDir()

	// 创建包含空格和中文的目录
	specialDir := filepath.Join(tempDir, "我的 文档")
	err := os.MkdirAll(specialDir, 0755)
	if err != nil {
		t.Skip("无法创建特殊字符目录")
	}

	src := NewLocalSource(specialDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Errorf("特殊字符路径应可正常使用: %v", err)
	}
	if result == "" {
		t.Error("应返回非空路径")
	}
}

// TestLocalSource_EmptyDirectory 测试空目录作为源
func TestLocalSource_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("创建空目录失败: %v", err)
	}

	src := NewLocalSource(emptyDir, Options{})
	result, err := src.Prepare()
	if err != nil {
		t.Errorf("空目录应可作为源: %v", err)
	}
	if result == "" {
		t.Error("空目录应返回路径")
	}
}

// TestGitHubSource_InvalidOwnerRepo 测试无效的 owner/repo 组合
func TestGitHubSource_InvalidOwnerRepo(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{"空 owner", "", "repo"},
		{"空 repo", "owner", ""},
		{"两者都空", "", ""},
		{"含空格", "owner name", "repo name"},
		{"含特殊符号", "owner@", "repo!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := NewGitHubSource(tt.owner, tt.repo, Options{})
			// 创建不应失败（验证推迟到 Prepare）
			if src == nil {
				t.Error("NewGitHubSource 不应返回 nil")
			}
			if src.Type() != "github" {
				t.Error("类型应为 github")
			}
		})
	}
}

// TestDetect_WhitespaceVariations 测试各种空白字符输入
func TestDetect_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"空字符串", "", true},
		{"单个空格", " ", true},
		{"多个空格", "     ", true},
		{"制表符", "\t", true},
		{"换行符", "\n", true},
		{"混合空白", " \t\n\r ", true},
		{"前后有空格的路径", "  /tmp/test  ", false}, // 应被 trim 后正常处理
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})
			if tt.wantErr {
				if err == nil {
					t.Errorf("Detect(%q) 应返回错误", tt.input)
				}
				if src != nil {
					t.Errorf("Detect(%q) 错误时 source 应为 nil", tt.input)
				}
			} else if err != nil {
				t.Errorf("Detect(%q) 不应返回错误: %v", tt.input, err)
			}
		})
	}
}

// TestDetect_OptionsPassthrough 测试选项正确传递给创建的源
func TestDetect_OptionsPassthrough(t *testing.T) {
	opts := Options{
		Branch: "develop",
		SubDir: "docs/zh",
	}

	// GitHub 源
	src, err := Detect("https://github.com/test/repo", opts)
	if err != nil {
		t.Fatalf("Detect 失败: %v", err)
	}
	ghSrc, ok := src.(*GitHubSource)
	if !ok {
		t.Fatal("应返回 GitHubSource")
	}
	if ghSrc.opts.Branch != "develop" {
		t.Errorf("Branch 应为 'develop', 实际 %q", ghSrc.opts.Branch)
	}
	if ghSrc.opts.SubDir != "docs/zh" {
		t.Errorf("SubDir 应为 'docs/zh', 实际 %q", ghSrc.opts.SubDir)
	}

	// 本地源
	src2, err := Detect(t.TempDir(), opts)
	if err != nil {
		t.Fatalf("Detect 失败: %v", err)
	}
	localSrc, ok := src2.(*LocalSource)
	if !ok {
		t.Fatal("应返回 LocalSource")
	}
	if localSrc.opts.SubDir != "docs/zh" {
		t.Errorf("SubDir 应为 'docs/zh', 实际 %q", localSrc.opts.SubDir)
	}
}

// TestLocalSource_PathTraversal 测试路径穿越攻击场景
func TestLocalSource_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "docs"), 0755); err != nil {
		t.Fatalf("创建 docs 目录失败: %v", err)
	}

	// 子目录使用 .. 尝试路径穿越
	src := NewLocalSource(tempDir, Options{SubDir: "../../../etc"})
	_, err := src.Prepare()
	// 此处不检查是否报错，重要的是不会导致 panic 或安全问题
	t.Logf("路径穿越测试: err=%v", err)
}

// TestGitHubURL_EdgeCases 测试 GitHub URL 正则匹配的边界情况
func TestGitHubURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMatch bool
	}{
		{"大写 GITHUB", "https://GITHUB.COM/owner/repo", false}, // 正则未忽略大小写
		{"尾部斜杠", "https://github.com/owner/repo/", true},
		{"多层路径", "https://github.com/owner/repo/tree/main/docs", true},
		{"github.io", "https://github.io/owner/repo", false},
		{"企业 GitHub", "https://github.example.com/owner/repo", false},
		{"仅 github.com", "github.com", false},
		{"github.com/", "github.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubURL(tt.input)
			if got != tt.wantMatch {
				t.Errorf("isGitHubURL(%q) = %v, 期望 %v", tt.input, got, tt.wantMatch)
			}
		})
	}
}

// TestLocalSource_ConcurrentPrepare 测试并发调用 Prepare 的安全性
func TestLocalSource_ConcurrentPrepare(t *testing.T) {
	tempDir := t.TempDir()
	src := NewLocalSource(tempDir, Options{})

	// 并发调用 Prepare 不应 panic
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = src.Prepare()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestGitHubSource_RepeatedCleanup 测试多次调用 Cleanup 的安全性
func TestGitHubSource_RepeatedCleanup(t *testing.T) {
	src := NewGitHubSource("owner", "repo", Options{})

	// GitHubSource 不需要并发安全；验证连续多次调用不会 panic
	for i := 0; i < 10; i++ {
		if err := src.Cleanup(); err != nil {
			t.Fatalf("Cleanup 第 %d 次调用失败: %v", i+1, err)
		}
	}
}
