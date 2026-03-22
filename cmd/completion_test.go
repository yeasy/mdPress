package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompletionCmd_Creation tests that the completion command is properly created
func TestCompletionCmd_Creation(t *testing.T) {
	if completionCmd == nil {
		t.Fatal("completionCmd should not be nil")
	}

	if completionCmd.Use != "completion <shell>" {
		t.Errorf("completionCmd.Use should be 'completion <shell>', got %q", completionCmd.Use)
	}

	if completionCmd.Short != "Generate shell completion scripts" {
		t.Errorf("completionCmd.Short should be 'Generate shell completion scripts', got %q", completionCmd.Short)
	}

	if !strings.Contains(completionCmd.Long, "shell completion scripts") {
		t.Error("completionCmd.Long should contain 'shell completion scripts'")
	}
}

// TestCompletionCmd_SubcommandRegistration tests that all shell subcommands are properly registered
func TestCompletionCmd_SubcommandRegistration(t *testing.T) {
	subcommands := []struct {
		name string
		cmd  string
	}{
		{"bash", "bash"},
		{"zsh", "zsh"},
		{"fish", "fish"},
		{"powershell", "powershell"},
	}

	for _, sc := range subcommands {
		found := false
		for _, cmd := range completionCmd.Commands() {
			if strings.HasPrefix(cmd.Use, sc.cmd) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("completion command should have %s subcommand", sc.name)
		}
	}
}

// TestBashCompletionCmd_Creation tests that the bash subcommand is properly created
func TestBashCompletionCmd_Creation(t *testing.T) {
	if bashCompletionCmd == nil {
		t.Fatal("bashCompletionCmd should not be nil")
	}

	if bashCompletionCmd.Use != "bash" {
		t.Errorf("bashCompletionCmd.Use should be 'bash', got %q", bashCompletionCmd.Use)
	}

	if bashCompletionCmd.Short != "Generate bash completion script" {
		t.Errorf("bashCompletionCmd.Short should be 'Generate bash completion script', got %q", bashCompletionCmd.Short)
	}

	if !strings.Contains(bashCompletionCmd.Long, "bash") {
		t.Error("bashCompletionCmd.Long should mention bash")
	}
}

// TestZshCompletionCmd_Creation tests that the zsh subcommand is properly created
func TestZshCompletionCmd_Creation(t *testing.T) {
	if zshCompletionCmd == nil {
		t.Fatal("zshCompletionCmd should not be nil")
	}

	if zshCompletionCmd.Use != "zsh" {
		t.Errorf("zshCompletionCmd.Use should be 'zsh', got %q", zshCompletionCmd.Use)
	}

	if zshCompletionCmd.Short != "Generate zsh completion script" {
		t.Errorf("zshCompletionCmd.Short should be 'Generate zsh completion script', got %q", zshCompletionCmd.Short)
	}

	if !strings.Contains(zshCompletionCmd.Long, "zsh") {
		t.Error("zshCompletionCmd.Long should mention zsh")
	}
}

// TestFishCompletionCmd_Creation tests that the fish subcommand is properly created
func TestFishCompletionCmd_Creation(t *testing.T) {
	if fishCompletionCmd == nil {
		t.Fatal("fishCompletionCmd should not be nil")
	}

	if fishCompletionCmd.Use != "fish" {
		t.Errorf("fishCompletionCmd.Use should be 'fish', got %q", fishCompletionCmd.Use)
	}

	if fishCompletionCmd.Short != "Generate fish shell completion script" {
		t.Errorf("fishCompletionCmd.Short should be 'Generate fish shell completion script', got %q", fishCompletionCmd.Short)
	}

	if !strings.Contains(fishCompletionCmd.Long, "fish") {
		t.Error("fishCompletionCmd.Long should mention fish")
	}
}

// TestPowershellCompletionCmd_Creation tests that the powershell subcommand is properly created
func TestPowershellCompletionCmd_Creation(t *testing.T) {
	if powershellCompletionCmd == nil {
		t.Fatal("powershellCompletionCmd should not be nil")
	}

	if powershellCompletionCmd.Use != "powershell" {
		t.Errorf("powershellCompletionCmd.Use should be 'powershell', got %q", powershellCompletionCmd.Use)
	}

	if powershellCompletionCmd.Short != "Generate PowerShell completion script" {
		t.Errorf("powershellCompletionCmd.Short should be 'Generate PowerShell completion script', got %q", powershellCompletionCmd.Short)
	}

	if !strings.Contains(powershellCompletionCmd.Long, "PowerShell") {
		t.Error("powershellCompletionCmd.Long should mention PowerShell")
	}
}

// TestExecuteCompletion_Bash tests the executeCompletion function with bash shell
func TestExecuteCompletion_Bash(t *testing.T) {
	err := executeCompletion("bash")
	if err != nil {
		t.Errorf("executeCompletion(\"bash\") should not error, got %v", err)
	}
}

// TestExecuteCompletion_Zsh tests the executeCompletion function with zsh shell
func TestExecuteCompletion_Zsh(t *testing.T) {
	err := executeCompletion("zsh")
	if err != nil {
		t.Errorf("executeCompletion(\"zsh\") should not error, got %v", err)
	}
}

// TestExecuteCompletion_Fish tests the executeCompletion function with fish shell
func TestExecuteCompletion_Fish(t *testing.T) {
	err := executeCompletion("fish")
	if err != nil {
		t.Errorf("executeCompletion(\"fish\") should not error, got %v", err)
	}
}

// TestExecuteCompletion_Powershell tests the executeCompletion function with powershell shell
func TestExecuteCompletion_Powershell(t *testing.T) {
	err := executeCompletion("powershell")
	if err != nil {
		t.Errorf("executeCompletion(\"powershell\") should not error, got %v", err)
	}
}

// TestExecuteCompletion_InvalidShell tests the executeCompletion function with an invalid shell
func TestExecuteCompletion_InvalidShell(t *testing.T) {
	tests := []struct {
		name  string
		shell string
	}{
		{
			name:  "nonexistent shell",
			shell: "nonexistent",
		},
		{
			name:  "empty shell",
			shell: "",
		},
		{
			name:  "invalid shell name",
			shell: "ksh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executeCompletion(tt.shell)
			if err == nil {
				t.Errorf("executeCompletion(%q) should error for invalid shell", tt.shell)
			}

			if !strings.Contains(err.Error(), "unsupported shell") {
				t.Errorf("error should mention unsupported shell, got: %v", err)
			}
		})
	}
}

// TestCompletionCmd_ExactArgsValidation tests that completion command requires exactly 1 argument
func TestCompletionCmd_ExactArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "one valid argument",
			args:    []string{"bash"},
			wantErr: false,
		},
		{
			name:    "two arguments",
			args:    []string{"bash", "extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completionCmd.SetArgs(tt.args)
			var out bytes.Buffer
			completionCmd.SetOut(&out)

			err := completionCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("completionCmd.Execute() with args %v should error=%v, got err=%v", tt.args, tt.wantErr, err)
			}
		})
	}
}

// TestBashCompletionCmd_HasNoDescriptionsFlag tests that bash completion has --no-descriptions flag
func TestBashCompletionCmd_HasNoDescriptionsFlag(t *testing.T) {
	flag := bashCompletionCmd.Flags().Lookup("no-descriptions")
	if flag == nil {
		t.Fatal("bash completion command should have --no-descriptions flag")
	}

	if flag.DefValue != "false" {
		t.Errorf("--no-descriptions flag default should be 'false', got %q", flag.DefValue)
	}
}

// TestFishCompletionCmd_HasNoDescriptionsFlag tests that fish completion has --no-descriptions flag
func TestFishCompletionCmd_HasNoDescriptionsFlag(t *testing.T) {
	flag := fishCompletionCmd.Flags().Lookup("no-descriptions")
	if flag == nil {
		t.Fatal("fish completion command should have --no-descriptions flag")
	}

	if flag.DefValue != "false" {
		t.Errorf("--no-descriptions flag default should be 'false', got %q", flag.DefValue)
	}
}

// TestZshCompletionCmd_NoDescriptionsFlag tests that zsh completion does not have --no-descriptions flag
func TestZshCompletionCmd_NoDescriptionsFlag(t *testing.T) {
	flag := zshCompletionCmd.Flags().Lookup("no-descriptions")
	if flag != nil {
		t.Error("zsh completion command should not have --no-descriptions flag (zsh doesn't support it)")
	}
}

// TestPowershellCompletionCmd_NoDescriptionsFlag tests that powershell completion does not have --no-descriptions flag
func TestPowershellCompletionCmd_NoDescriptionsFlag(t *testing.T) {
	flag := powershellCompletionCmd.Flags().Lookup("no-descriptions")
	if flag != nil {
		t.Error("powershell completion command should not have --no-descriptions flag (powershell doesn't support it)")
	}
}

// TestCompletionCmd_HelpOutput tests that completion command help output is correct
func TestCompletionCmd_HelpOutput(t *testing.T) {
	completionCmd.SetArgs([]string{"--help"})
	var out bytes.Buffer
	completionCmd.SetOut(&out)

	err := completionCmd.Execute()
	if err != nil && !strings.Contains(err.Error(), "help") {
		// Some cobra versions return an error for --help, some don't
		// Both are acceptable
	}

	output := out.String()
	checks := []string{
		"bash",
		"zsh",
		"fish",
		"powershell",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("completion help should mention %q shell", check)
		}
	}
}

// TestCompletionCmd_SupportedShellsDocumented tests that all supported shells are documented in Long help
func TestCompletionCmd_SupportedShellsDocumented(t *testing.T) {
	requiredShells := map[string]bool{
		"bash":       false,
		"zsh":        false,
		"fish":       false,
		"powershell": false,
	}

	for shell := range requiredShells {
		if strings.Contains(completionCmd.Long, shell) {
			requiredShells[shell] = true
		}
	}

	for shell, found := range requiredShells {
		if !found {
			t.Errorf("completion command Long help should document %q shell", shell)
		}
	}
}

// TestExecuteCompletion_AllValidShells tests executeCompletion with all valid shells
func TestExecuteCompletion_AllValidShells(t *testing.T) {
	validShells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range validShells {
		t.Run(shell, func(t *testing.T) {
			err := executeCompletion(shell)
			if err != nil {
				t.Errorf("executeCompletion(%q) should not error, got %v", shell, err)
			}
		})
	}
}

// TestCompletionCmd_RunE tests that the completion command's RunE function calls executeCompletion
func TestCompletionCmd_RunE(t *testing.T) {
	// Test with a valid shell argument
	completionCmd.SetArgs([]string{"bash"})
	var out bytes.Buffer
	completionCmd.SetOut(&out)

	err := completionCmd.Execute()
	if err != nil {
		t.Errorf("completionCmd with bash should not error, got %v", err)
	}
}

// TestCompletionCmd_CaseInsensitiveShellDetection tests if shell detection is case-sensitive (as per implementation)
func TestCompletionCmd_CaseInsensitiveShellDetection(t *testing.T) {
	// The current implementation uses lowercase string matching via switch statement
	// Testing that uppercase shells don't match (as expected from the implementation)
	err := executeCompletion("BASH")
	if err == nil {
		t.Error("executeCompletion with uppercase 'BASH' should error")
	}

	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("error should mention unsupported shell, got: %v", err)
	}
}
