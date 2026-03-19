package source

import (
	"strings"
	"testing"
)

// TestIsGitHubURL 测试 GitHub URL 识别（表驱动测试）
func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantRes bool
	}{
		// 标准格式
		{
			name:    "标准 HTTPS URL",
			input:   "https://github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "标准 HTTP URL",
			input:   "http://github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "无协议的 GitHub URL",
			input:   "github.com/owner/repo",
			wantRes: true,
		},
		// .git 后缀
		{
			name:    "带 .git 后缀",
			input:   "https://github.com/owner/repo.git",
			wantRes: true,
		},
		{
			name:    "带 .git 后缀（无协议）",
			input:   "github.com/owner/repo.git",
			wantRes: true,
		},
		// www 前缀
		{
			name:    "带 www 前缀",
			input:   "https://www.github.com/owner/repo",
			wantRes: true,
		},
		{
			name:    "www 无协议",
			input:   "www.github.com/owner/repo",
			wantRes: true,
		},
		// 带路径的 URL
		{
			name:    "带路径的 URL",
			input:   "https://github.com/owner/repo/tree/main",
			wantRes: true,
		},
		{
			name:    "带 issues 路径",
			input:   "https://github.com/owner/repo/issues/123",
			wantRes: true,
		},
		// 无效或非 GitHub URL
		{
			name:    "非 GitHub 域名",
			input:   "https://gitlab.com/owner/repo",
			wantRes: false,
		},
		{
			name:    "局部路径",
			input:   "/home/user/project",
			wantRes: false,
		},
		{
			name:    "相对路径",
			input:   "./project",
			wantRes: false,
		},
		{
			name:    "纯文本名称",
			input:   "owner/repo",
			wantRes: false,
		},
		{
			name:    "空字符串",
			input:   "",
			wantRes: false,
		},
		{
			name:    "仅有 owner 无 repo",
			input:   "https://github.com/owner",
			wantRes: false,
		},
		{
			name:    "Windows 路径",
			input:   "C:\\Users\\project",
			wantRes: false,
		},
		// 特殊字符测试
		{
			name:    "URL 带查询参数",
			input:   "https://github.com/owner/repo?tab=readme",
			wantRes: false,
		},
		{
			name:    "URL 带哈希锚点",
			input:   "https://github.com/owner/repo#readme",
			wantRes: false,
		},
		{
			name:    "带连字符的仓库名",
			input:   "https://github.com/owner/my-cool-repo",
			wantRes: true,
		},
		{
			name:    "带下划线的仓库名",
			input:   "https://github.com/owner/my_repo",
			wantRes: true,
		},
		{
			name:    "带数字的仓库名",
			input:   "https://github.com/owner/repo123",
			wantRes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubURL(tt.input)
			if got != tt.wantRes {
				t.Errorf("isGitHubURL(%q) = %v, want %v", tt.input, got, tt.wantRes)
			}
		})
	}
}

// TestParseGitHubURL 测试从 GitHub URL 提取 owner 和 repo
func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "标准 HTTPS URL",
			input:     "https://github.com/golang/go",
			wantOwner: "golang",
			wantRepo:  "go",
		},
		{
			name:      "HTTP URL",
			input:     "http://github.com/kubernetes/kubernetes",
			wantOwner: "kubernetes",
			wantRepo:  "kubernetes",
		},
		{
			name:      "无协议 URL",
			input:     "github.com/rust-lang/rust",
			wantOwner: "rust-lang",
			wantRepo:  "rust",
		},
		{
			name:      "带 .git 后缀",
			input:     "https://github.com/torvalds/linux.git",
			wantOwner: "torvalds",
			wantRepo:  "linux",
		},
		{
			name:      "www 前缀",
			input:     "https://www.github.com/python/cpython",
			wantOwner: "python",
			wantRepo:  "cpython",
		},
		{
			name:      "带路径",
			input:     "https://github.com/nodejs/node/tree/main",
			wantOwner: "nodejs",
			wantRepo:  "node",
		},
		{
			name:      "带查询参数",
			input:     "https://github.com/django/django?tab=readme",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "带哈希锚点",
			input:     "https://github.com/vuejs/vue#readme",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "带连字符和下划线",
			input:     "https://github.com/my-org/my_repo-v2",
			wantOwner: "my-org",
			wantRepo:  "my_repo-v2",
		},
		{
			name:      "非 GitHub URL",
			input:     "https://gitlab.com/owner/repo",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "无效格式",
			input:     "https://github.com/onlyowner",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "空字符串",
			input:     "",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo := parseGitHubURL(tt.input)
			if gotOwner != tt.wantOwner || gotRepo != tt.wantRepo {
				t.Errorf("parseGitHubURL(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotOwner, gotRepo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

// TestDetect 测试源类型自动检测
func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     Options
		wantType string
		wantErr  bool
	}{
		// GitHub URL 检测
		{
			name:     "GitHub HTTPS URL",
			input:    "https://github.com/golang/go",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL 无协议",
			input:    "github.com/python/cpython",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL 带 .git",
			input:    "https://github.com/torvalds/linux.git",
			opts:     Options{},
			wantType: "github",
			wantErr:  false,
		},
		{
			name:     "GitHub URL 带分支选项",
			input:    "https://github.com/nodejs/node",
			opts:     Options{Branch: "main"},
			wantType: "github",
			wantErr:  false,
		},
		// 本地路径检测
		{
			name:     "本地绝对路径",
			input:    "/home/user/project",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "本地相对路径",
			input:    "./project",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "本地路径（当前目录）",
			input:    ".",
			opts:     Options{},
			wantType: "local",
			wantErr:  false,
		},
		{
			name:     "本地路径 with SubDir",
			input:    "/home/user/project",
			opts:     Options{SubDir: "docs"},
			wantType: "local",
			wantErr:  false,
		},
		// 错误情况
		{
			name:     "空字符串",
			input:    "",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "仅空格",
			input:    "   ",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "制表符和空格",
			input:    "\t \n",
			opts:     Options{},
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Detect(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Detect(%q) unexpected error: %v", tt.input, err)
				return
			}

			if src == nil {
				t.Errorf("Detect(%q) returned nil source", tt.input)
				return
			}

			if src.Type() != tt.wantType {
				t.Errorf("Detect(%q).Type() = %q, want %q", tt.input, src.Type(), tt.wantType)
			}
		})
	}
}

// TestDetectEmptyInput 测试空输入处理
func TestDetectEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"spaces only", "   "},
		{"tab only", "\t"},
		{"newline only", "\n"},
		{"mixed whitespace", " \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := Detect(tt.input, Options{})

			if err == nil {
				t.Errorf("Detect(%q) should return error for empty/whitespace input", tt.input)
			}

			if src != nil {
				t.Errorf("Detect(%q) should return nil source for empty/whitespace input", tt.input)
			}

			// 验证错误消息包含有意义的内容
			if err != nil && !strings.Contains(err.Error(), "空") {
				t.Logf("Error message: %v", err)
			}
		})
	}
}
