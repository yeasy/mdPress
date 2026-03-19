// cmd_test.go 测试命令行接口的集成行为。
// 包括：--help、--version、无效参数、子命令帮助等场景。
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// TestRootCommand_Help 测试根命令 --help 输出
func TestRootCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("--help 返回错误（某些 cobra 版本正常）: %v", err)
	}

	output := out.String()

	// 验证帮助信息包含关键内容
	checks := []struct {
		desc    string
		contain string
	}{
		{"工具名称", "mdpress"},
		{"build 子命令", "build"},
		{"init 子命令", "init"},
		{"serve 子命令", "serve"},
		{"themes 子命令", "themes"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.contain) {
			t.Errorf("帮助输出应包含 %s (%q)", c.desc, c.contain)
		}
	}
}

// TestRootCommand_Version 测试版本号设置
func TestRootCommand_Version(t *testing.T) {
	// 验证 rootCmd 的 Version 字段已正确设置
	if rootCmd.Version != Version {
		t.Errorf("rootCmd.Version 应为 %q, 实际: %q", Version, rootCmd.Version)
	}
}

// TestBuildCommand_Help 测试 build 子命令帮助信息
func TestBuildCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"build", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("build --help 错误: %v", err)
	}

	output := out.String()

	checks := []string{
		"build",
		"--format",
		"--branch",
		"--subdir",
		"pdf",
		"html",
	}

	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Errorf("build 帮助应包含 %q", c)
		}
	}
}

// TestBuildCommand_HelpContainsExamples 测试 build 帮助包含使用示例
func TestBuildCommand_HelpContainsExamples(t *testing.T) {
	rootCmd.SetArgs([]string{"build", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Logf("build --help 错误: %v", err)
	}
	output := out.String()

	if !strings.Contains(output, "mdpress build") {
		t.Error("build 帮助应包含使用示例 'mdpress build'")
	}
}

// TestInitCommand_Help 测试 init 子命令帮助
func TestInitCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"init", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("init --help 错误: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "init") {
		t.Error("init 帮助应包含 'init'")
	}
}

// TestServeCommand_Help 测试 serve 子命令帮助
func TestServeCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"serve", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("serve --help 错误: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "--port") {
		t.Error("serve 帮助应包含 --port 选项")
	}
	if !strings.Contains(output, "--host") {
		t.Error("serve 帮助应包含 --host 选项")
	}
	if !strings.Contains(output, "--open") {
		t.Error("serve 帮助应包含 --open 选项")
	}
}

// TestThemesCommand_Help 测试 themes 子命令帮助
func TestThemesCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"themes", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Logf("themes --help 错误: %v", err)
	}
	output := out.String()

	if !strings.Contains(output, "list") {
		t.Error("themes 帮助应包含 'list' 子命令")
	}
	if !strings.Contains(output, "show") {
		t.Error("themes 帮助应包含 'show' 子命令")
	}
}

// TestDoctorCommand_Help 测试 doctor 子命令帮助
func TestDoctorCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"doctor", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Logf("doctor --help 错误: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "doctor") {
		t.Error("doctor 帮助应包含 'doctor'")
	}
}

// TestInvalidSubcommand 测试无效子命令
func TestInvalidSubcommand(t *testing.T) {
	rootCmd.SetArgs([]string{"nonexistent"})
	var errOut bytes.Buffer
	rootCmd.SetErr(&errOut)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("无效子命令应返回错误")
	}
}

// TestPersistentFlags 测试持久标志
func TestPersistentFlags(t *testing.T) {
	// 验证 --config 持久标志存在
	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("应存在 --config 持久标志")
		return
	}
	if flag.DefValue != "book.yaml" {
		t.Errorf("--config 默认值应为 'book.yaml', 实际 %q", flag.DefValue)
	}

	// 验证 --verbose 持久标志存在
	flag = rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("应存在 --verbose 持久标志")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("--verbose 默认值应为 'false', 实际 %q", flag.DefValue)
	}

	flag = rootCmd.PersistentFlags().Lookup("cache-dir")
	if flag == nil {
		t.Fatal("应存在 --cache-dir 持久标志")
	}
	if flag.DefValue != "" {
		t.Errorf("--cache-dir 默认值应为空, 实际 %q", flag.DefValue)
	}

	flag = rootCmd.PersistentFlags().Lookup("no-cache")
	if flag == nil {
		t.Fatal("应存在 --no-cache 持久标志")
	}
	if flag.DefValue != "false" {
		t.Errorf("--no-cache 默认值应为 'false', 实际 %q", flag.DefValue)
	}
}

// TestBuildCommand_Flags 测试 build 命令的标志
func TestBuildCommand_Flags(t *testing.T) {
	flag := buildCmd.Flags().Lookup("format")
	if flag == nil {
		t.Fatal("build 应有 --format 标志")
		return
	}
	if flag.DefValue != "" {
		t.Errorf("--format 默认值应为空, 实际 %q", flag.DefValue)
	}

	flag = buildCmd.Flags().Lookup("branch")
	if flag == nil {
		t.Error("build 应有 --branch 标志")
	}

	flag = buildCmd.Flags().Lookup("subdir")
	if flag == nil {
		t.Error("build 应有 --subdir 标志")
	}

	flag = buildCmd.Flags().Lookup("output")
	if flag == nil {
		t.Error("build 应有 --output 标志")
	}
}

// TestServeCommand_Flags 测试 serve 命令的标志
func TestServeCommand_Flags(t *testing.T) {
	flag := serveCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatal("serve 应有 --port 标志")
		return
	}
	if flag.DefValue != "9000" {
		t.Errorf("--port 默认值应为 9000, 实际 %q", flag.DefValue)
	}

	flag = serveCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("serve 应有 --output 标志")
		return
	}

	flag = serveCmd.Flags().Lookup("host")
	if flag == nil {
		t.Fatal("serve 应有 --host 标志")
		return
	}
	if flag.DefValue != "127.0.0.1" {
		t.Errorf("--host 默认值应为 127.0.0.1, 实际 %q", flag.DefValue)
	}

	flag = serveCmd.Flags().Lookup("open")
	if flag == nil {
		t.Fatal("serve 应有 --open 标志")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("--open 默认值应为 false, 实际 %q", flag.DefValue)
	}
}

// TestThemesShowCommand_ArgsValidation 测试 themes show 的参数验证
func TestThemesShowCommand_ArgsValidation(t *testing.T) {
	rootCmd.SetArgs([]string{"themes", "show"})
	var errOut bytes.Buffer
	rootCmd.SetErr(&errOut)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("themes show 没有参数应返回错误")
	}
}

// TestFlattenChapters 测试章节展开函数
func TestFlattenChapters(t *testing.T) {
	tests := []struct {
		name    string
		input   []config.ChapterDef
		wantLen int
	}{
		{
			name:    "空列表",
			input:   nil,
			wantLen: 0,
		},
		{
			name: "无嵌套",
			input: []config.ChapterDef{
				{Title: "Ch1", File: "ch1.md"},
				{Title: "Ch2", File: "ch2.md"},
			},
			wantLen: 2,
		},
		{
			name: "单层嵌套",
			input: []config.ChapterDef{
				{
					Title: "Ch1", File: "ch1.md",
					Sections: []config.ChapterDef{
						{Title: "Sec1.1", File: "sec1_1.md"},
						{Title: "Sec1.2", File: "sec1_2.md"},
					},
				},
			},
			wantLen: 3,
		},
		{
			name: "多层嵌套",
			input: []config.ChapterDef{
				{
					Title: "Part1", File: "p1.md",
					Sections: []config.ChapterDef{
						{
							Title: "Ch1", File: "ch1.md",
							Sections: []config.ChapterDef{
								{Title: "Sec1.1", File: "s1.md"},
							},
						},
					},
				},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenChapters(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("flattenChapters 返回 %d 个, 期望 %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestGetPageDimensions 测试页面尺寸转换
func TestGetPageDimensions(t *testing.T) {
	tests := []struct {
		size       string
		wantWidth  float64
		wantHeight float64
	}{
		{"A4", 210, 297},
		{"a4", 210, 297},
		{"A5", 148, 210},
		{"LETTER", 216, 279},
		{"LEGAL", 216, 356},
		{"B5", 176, 250},
		{"unknown", 210, 297},
		{"", 210, 297},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			w, h := getPageDimensions(tt.size)
			if w != tt.wantWidth || h != tt.wantHeight {
				t.Errorf("getPageDimensions(%q) = (%v, %v), 期望 (%v, %v)",
					tt.size, w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

// TestGetAvailableThemes 测试可用主题列表
func TestGetAvailableThemes(t *testing.T) {
	themes := getAvailableThemes()

	if len(themes) == 0 {
		t.Error("应至少有一个可用主题")
	}

	requiredThemes := map[string]bool{
		"technical": false,
		"elegant":   false,
		"minimal":   false,
	}

	for _, thm := range themes {
		if _, ok := requiredThemes[thm.Name]; ok {
			requiredThemes[thm.Name] = true
		}
		if thm.Name == "" {
			t.Error("主题名称不应为空")
		}
		if thm.DisplayName == "" {
			t.Errorf("主题 %q 的显示名不应为空", thm.Name)
		}
		if thm.Description == "" {
			t.Errorf("主题 %q 的描述不应为空", thm.Name)
		}
		if len(thm.Features) == 0 {
			t.Errorf("主题 %q 应有特性列表", thm.Name)
		}
		if thm.Colors.Primary == "" {
			t.Errorf("主题 %q 应有主色", thm.Name)
		}
	}

	for name, found := range requiredThemes {
		if !found {
			t.Errorf("缺少必须的主题: %q", name)
		}
	}
}

// TestExecuteThemesShow_ValidTheme 测试显示有效主题
func TestExecuteThemesShow_ValidTheme(t *testing.T) {
	err := executeThemesShow("technical")
	if err != nil {
		t.Errorf("显示 technical 主题不应报错: %v", err)
	}
}

// TestExecuteThemesShow_InvalidTheme 测试显示无效主题
func TestExecuteThemesShow_InvalidTheme(t *testing.T) {
	err := executeThemesShow("nonexistent_theme")
	if err == nil {
		t.Error("显示不存在的主题应报错")
	}
	if !strings.Contains(err.Error(), "theme not found") {
		t.Errorf("错误消息应包含 'theme not found', 实际: %q", err.Error())
	}
}

// TestExecuteThemesList 测试列出主题
func TestExecuteThemesList(t *testing.T) {
	err := executeThemesList()
	if err != nil {
		t.Errorf("列出主题不应报错: %v", err)
	}
}

// TestInferTitleFromPath 测试从路径推断标题
func TestInferTitleFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"简单文件", "preface.md", "Preface"},
		{"子目录 README", "chapter01/README.md", "Chapter01"},
		{"嵌套路径", "part1/intro.md", "Part1 - intro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferTitleFromPath(tt.path)
			if got != tt.want {
				t.Errorf("inferTitleFromPath(%q) = %q, 期望 %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestCountChapterDefs 测试章节计数函数
func TestCountChapterDefs(t *testing.T) {
	tests := []struct {
		name  string
		input []config.ChapterDef
		want  int
	}{
		{"空列表", nil, 0},
		{"两个顶级", []config.ChapterDef{{Title: "A"}, {Title: "B"}}, 2},
		{
			"带嵌套",
			[]config.ChapterDef{
				{Title: "A", Sections: []config.ChapterDef{{Title: "A.1"}, {Title: "A.2"}}},
				{Title: "B"},
			},
			4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countChapterDefs(tt.input)
			if got != tt.want {
				t.Errorf("countChapterDefs = %d, 期望 %d", got, tt.want)
			}
		})
	}
}
