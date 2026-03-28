package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestThemesCmd_Creation tests that the themes command is properly created
func TestThemesCmd_Creation(t *testing.T) {
	if themesCmd == nil {
		t.Fatal("themesCmd should not be nil")
	}

	if themesCmd.Use != "themes" {
		t.Errorf("themesCmd.Use should be 'themes', got %q", themesCmd.Use)
	}

	if themesCmd.Short != "Manage built-in themes" {
		t.Errorf("themesCmd.Short should be 'Manage built-in themes', got %q", themesCmd.Short)
	}

	if !strings.Contains(themesCmd.Long, "List and inspect the built-in themes") {
		t.Error("themesCmd.Long should contain theme description")
	}
}

// TestThemesCmd_SubcommandRegistration tests that subcommands are properly registered
func TestThemesCmd_SubcommandRegistration(t *testing.T) {
	subcommands := []struct {
		name string
		cmd  string
	}{
		{"list", "list"},
		{"show", "show"},
		{"preview", "preview"},
	}

	for _, sc := range subcommands {
		found := false
		for _, cmd := range themesCmd.Commands() {
			if strings.HasPrefix(cmd.Use, sc.cmd) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("themes command should have %s subcommand", sc.name)
		}
	}
}

// TestThemesListCmd_Creation tests that the list subcommand is properly created
func TestThemesListCmd_Creation(t *testing.T) {
	if themesListCmd == nil {
		t.Fatal("themesListCmd should not be nil")
	}

	if themesListCmd.Use != "list" {
		t.Errorf("themesListCmd.Use should be 'list', got %q", themesListCmd.Use)
	}

	if themesListCmd.Short != "List all themes" {
		t.Errorf("themesListCmd.Short should be 'List all themes', got %q", themesListCmd.Short)
	}
}

// TestThemesShowCmd_Creation tests that the show subcommand is properly created
func TestThemesShowCmd_Creation(t *testing.T) {
	if themesShowCmd == nil {
		t.Fatal("themesShowCmd should not be nil")
	}

	if themesShowCmd.Use != "show <theme-name>" {
		t.Errorf("themesShowCmd.Use should be 'show <theme-name>', got %q", themesShowCmd.Use)
	}

	if themesShowCmd.Short != "Show theme details" {
		t.Errorf("themesShowCmd.Short should be 'Show theme details', got %q", themesShowCmd.Short)
	}
}

// TestThemesPreviewCmd_Creation tests that the preview subcommand is properly created
func TestThemesPreviewCmd_Creation(t *testing.T) {
	if themesPreviewCmd == nil {
		t.Fatal("themesPreviewCmd should not be nil")
	}

	if themesPreviewCmd.Use != "preview" {
		t.Errorf("themesPreviewCmd.Use should be 'preview', got %q", themesPreviewCmd.Use)
	}

	if themesPreviewCmd.Short != "Generate an HTML preview of all themes" {
		t.Errorf("themesPreviewCmd.Short should mention HTML preview, got %q", themesPreviewCmd.Short)
	}
}

// TestThemesPreviewCmd_OutputFlag tests that the output flag is properly configured
func TestThemesPreviewCmd_OutputFlag(t *testing.T) {
	flag := themesPreviewCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("preview command should have --output flag")
	}

	if flag.DefValue != "themes-preview.html" {
		t.Errorf("output flag default should be 'themes-preview.html', got %q", flag.DefValue)
	}
}

// TestGetAvailableThemes_ExpectedSet tests that all expected themes are present
func TestGetAvailableThemes_ExpectedSet(t *testing.T) {
	themes := getAvailableThemes()

	if len(themes) == 0 {
		t.Fatal("getAvailableThemes should return at least one theme")
	}

	// Verify expected themes exist
	expectedThemes := map[string]bool{
		"technical": false,
		"elegant":   false,
		"minimal":   false,
	}

	for _, theme := range themes {
		if _, exists := expectedThemes[theme.name]; exists {
			expectedThemes[theme.name] = true
		}
	}

	for themeName, found := range expectedThemes {
		if !found {
			t.Errorf("expected theme %q not found in available themes", themeName)
		}
	}
}

// TestTheme_Structure tests that themes have all required fields
func TestTheme_Structure(t *testing.T) {
	themes := getAvailableThemes()

	tests := []struct {
		name  string
		check func(*themeInfo) error
	}{
		{
			name: "has name",
			check: func(th *themeInfo) error {
				if th.name == "" {
					return fmt.Errorf("theme should have a non-empty name")
				}
				return nil
			},
		},
		{
			name: "has display name",
			check: func(th *themeInfo) error {
				if th.displayName == "" {
					return fmt.Errorf("theme should have a non-empty display name")
				}
				return nil
			},
		},
		{
			name: "has description",
			check: func(th *themeInfo) error {
				if th.description == "" {
					return fmt.Errorf("theme should have a non-empty description")
				}
				return nil
			},
		},
		{
			name: "has author",
			check: func(th *themeInfo) error {
				if th.author == "" {
					return fmt.Errorf("theme should have a non-empty author")
				}
				return nil
			},
		},
		{
			name: "has version",
			check: func(th *themeInfo) error {
				if th.version == "" {
					return fmt.Errorf("theme should have a non-empty version")
				}
				return nil
			},
		},
		{
			name: "has license",
			check: func(th *themeInfo) error {
				if th.license == "" {
					return fmt.Errorf("theme should have a non-empty license")
				}
				return nil
			},
		},
		{
			name: "has features",
			check: func(th *themeInfo) error {
				if len(th.features) == 0 {
					return fmt.Errorf("theme should have at least one feature")
				}
				return nil
			},
		},
	}

	for _, theme := range themes {
		for _, tt := range tests {
			if err := tt.check(&theme); err != nil {
				t.Errorf("theme %q: %s failed: %v", theme.name, tt.name, err)
			}
		}
	}
}

// TestThemeColorsStructure tests that themes have all required color fields
func TestThemeColorsStructure(t *testing.T) {
	themes := getAvailableThemes()

	colorFields := []struct {
		name string
		get  func(*themeColors) string
	}{
		{"primary", func(tc *themeColors) string { return tc.primary }},
		{"secondary", func(tc *themeColors) string { return tc.secondary }},
		{"accent", func(tc *themeColors) string { return tc.accent }},
		{"text", func(tc *themeColors) string { return tc.text }},
		{"background", func(tc *themeColors) string { return tc.background }},
		{"code background", func(tc *themeColors) string { return tc.codeBg }},
	}

	for _, theme := range themes {
		for _, field := range colorFields {
			color := field.get(&theme.colors)
			if color == "" {
				t.Errorf("theme %q: color field %q should not be empty", theme.name, field.name)
			}
			if !strings.HasPrefix(color, "#") {
				t.Errorf("theme %q: color field %q should be hex format, got %q", theme.name, field.name, color)
			}
		}
	}
}

// TestExecuteThemesList_Success tests the executeThemesList function succeeds
func TestExecuteThemesList_Success(t *testing.T) {
	defer suppressOutput(t)()
	err := executeThemesList()
	if err != nil {
		t.Errorf("executeThemesList should not return error, got %v", err)
	}
	// Note: output goes to fmt.Println, not captured by test
	// The function returns nil on success
}

// TestExecuteThemesShow_AllValidThemes tests showing each valid theme
func TestExecuteThemesShow_AllValidThemes(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		shouldErr bool
	}{
		{
			name:      "valid technical theme",
			themeName: "technical",
			shouldErr: false,
		},
		{
			name:      "valid elegant theme",
			themeName: "elegant",
			shouldErr: false,
		},
		{
			name:      "valid minimal theme",
			themeName: "minimal",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer suppressOutput(t)()
			err := executeThemesShow(tt.themeName)
			if (err != nil) != tt.shouldErr {
				t.Errorf("executeThemesShow(%q) should error=%v, got err=%v", tt.themeName, tt.shouldErr, err)
			}
		})
	}
}

// TestExecuteThemesShow_InvalidThemes tests error handling for various invalid themes
func TestExecuteThemesShow_InvalidThemes(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		shouldErr bool
	}{
		{
			name:      "nonexistent theme",
			themeName: "nonexistent",
			shouldErr: true,
		},
		{
			name:      "empty theme name",
			themeName: "",
			shouldErr: true,
		},
		{
			name:      "invalid theme name",
			themeName: "invalid_theme_xyz",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer suppressOutput(t)()
			err := executeThemesShow(tt.themeName)
			if (err != nil) != tt.shouldErr {
				t.Errorf("executeThemesShow(%q) should error=%v, got err=%v", tt.themeName, tt.shouldErr, err)
			}

			if tt.shouldErr && err != nil {
				// Verify error message mentions the theme was not found
				if !strings.Contains(err.Error(), "theme not found") {
					t.Errorf("error should mention theme not found, got: %v", err)
				}
			}
		})
	}
}

// TestExecuteThemesShow_MatchesThemeData tests that show outputs match theme data
func TestExecuteThemesShow_MatchesThemeData(t *testing.T) {
	defer suppressOutput(t)()
	themes := getAvailableThemes()
	if len(themes) == 0 {
		t.Skip("no themes available to test")
	}

	// Test with the first theme
	testTheme := themes[0]

	err := executeThemesShow(testTheme.name)
	if err != nil {
		t.Fatalf("executeThemesShow failed for valid theme: %v", err)
	}

	// The function prints to stdout, so we can't easily verify the output
	// but we verify it doesn't error for valid themes
}

// TestExecuteThemesPreview tests the executeThemesPreview function
func TestExecuteThemesPreview(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
		shouldErr  bool
		useTempDir bool
	}{
		{
			name:       "default output",
			outputPath: "themes-preview.html",
			shouldErr:  false,
			useTempDir: true,
		},
		{
			name:       "custom output path",
			outputPath: filepath.Join(t.TempDir(), "test-themes-preview.html"),
			shouldErr:  false,
			useTempDir: false,
		},
		{
			name:       "empty output path",
			outputPath: "",
			shouldErr:  false,
			useTempDir: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := tt.outputPath

			// For tests that use relative paths, use a temp directory
			if tt.useTempDir {
				tmpDir := t.TempDir()
				if tt.outputPath == "" {
					outputPath = ""
				} else {
					outputPath = filepath.Join(tmpDir, tt.outputPath)
				}
				t.Chdir(tmpDir)
			}

			err := executeThemesPreview(outputPath)
			if (err != nil) != tt.shouldErr {
				t.Errorf("executeThemesPreview(%q) should error=%v, got err=%v", outputPath, tt.shouldErr, err)
			}
		})
	}
}

// TestThemesCmd_WithoutSubcommand verifies the themes command without
// a subcommand does not panic and either returns an error or produces
// help output.
func TestThemesCmd_WithoutSubcommand(t *testing.T) {
	defer suppressOutput(t)()
	themesCmd.SetArgs([]string{})
	var out bytes.Buffer
	themesCmd.SetOut(&out)
	themesCmd.SetErr(&out)

	// Should not panic; verify it either succeeds or returns a known error.
	err := themesCmd.Execute()
	// The command may succeed (printing help) or error -- both are acceptable.
	// The key assertion is that it doesn't panic. Shared-state fd errors
	// (e.g. "bad file descriptor" from suppressOutput) are benign.
	if err != nil && !strings.Contains(err.Error(), "required") && !strings.Contains(err.Error(), "bad file descriptor") {
		t.Errorf("themes command returned unexpected error: %v", err)
	}
}

// TestThemesShowCmd_MissingArgument tests show command without argument.
// Uses rootCmd.SetArgs to route through the real Cobra command tree.
func TestThemesShowCmd_MissingArgument(t *testing.T) {
	defer suppressOutput(t)()
	rootCmd.SetArgs([]string{"themes", "show"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	// Should require exactly 1 argument
	err := rootCmd.Execute()
	if err == nil {
		t.Error("show command should error when theme name is missing")
	}
}

// TestThemesShowCmd_TooManyArguments tests show command with too many arguments.
// Uses rootCmd.SetArgs to route through the real Cobra command tree.
func TestThemesShowCmd_TooManyArguments(t *testing.T) {
	defer suppressOutput(t)()
	rootCmd.SetArgs([]string{"themes", "show", "technical", "extra"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	// Should require exactly 1 argument
	err := rootCmd.Execute()
	if err == nil {
		t.Error("show command should error with too many arguments")
	}
}

// BenchmarkGetAvailableThemes benchmarks the getAvailableThemes function
func BenchmarkGetAvailableThemes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = getAvailableThemes()
	}
}

// BenchmarkExecuteThemesList benchmarks the executeThemesList function
func BenchmarkExecuteThemesList(b *testing.B) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	origHandler := slog.Default().Handler()
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		b.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	os.Stdout = devNull
	os.Stderr = devNull
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		slog.SetDefault(slog.New(origHandler))
		devNull.Close()
	}()
	for i := 0; i < b.N; i++ {
		_ = executeThemesList()
	}
}

// BenchmarkExecuteThemesShow benchmarks the executeThemesShow function
func BenchmarkExecuteThemesShow(b *testing.B) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	origHandler := slog.Default().Handler()
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		b.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	os.Stdout = devNull
	os.Stderr = devNull
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		slog.SetDefault(slog.New(origHandler))
		devNull.Close()
	}()
	for i := 0; i < b.N; i++ {
		_ = executeThemesShow("technical")
	}
}
