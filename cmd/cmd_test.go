// cmd_test.go Tests CLI integration behavior.
// Covers: --help, --version, invalid arguments, subcommand help, etc.
package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// suppressOutput temporarily redirects stdout and stderr to /dev/null.
// Returns a restore function that must be deferred.
func suppressOutput(t *testing.T) func() {
	t.Helper()
	origStdout := os.Stdout
	origStderr := os.Stderr
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	os.Stdout = devNull
	os.Stderr = devNull
	// Also suppress slog default logger
	origHandler := slog.Default().Handler()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	return func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		slog.SetDefault(slog.New(origHandler))
		devNull.Close()
	}
}

// TestRootCommand_Help tests root command --help output
func TestRootCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	output := out.String()

	// Verify help output contains key content
	checks := []struct {
		desc    string
		contain string
	}{
		{"tool name", "mdpress"},
		{"build subcommand", "build"},
		{"init subcommand", "init"},
		{"serve subcommand", "serve"},
		{"themes subcommand", "themes"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.contain) {
			t.Errorf("help output should contain %s (%q)", c.desc, c.contain)
		}
	}
}

// TestRootCommand_Version tests version number setup
func TestRootCommand_Version(t *testing.T) {
	// Verify rootCmd Version field is set correctly
	if rootCmd.Version != Version {
		t.Errorf("rootCmd.Version should be %q, got: %q", Version, rootCmd.Version)
	}
}

// TestBuildCommand_Help tests build subcommand help output
func TestBuildCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"build", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("build --help error: %v", err)
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
			t.Errorf("build help should contain %q", c)
		}
	}
}

// TestBuildCommand_HelpContainsExamples tests build help contains usage examples
func TestBuildCommand_HelpContainsExamples(t *testing.T) {
	rootCmd.SetArgs([]string{"build", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build --help error: %v", err)
	}
	output := out.String()

	if !strings.Contains(output, "mdpress build") {
		t.Error("build help should contain usage example 'mdpress build'")
	}
}

// TestInitCommand_Help tests init subcommand help
func TestInitCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"init", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("init --help error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "init") {
		t.Error("init help should contain 'init'")
	}
}

// TestServeCommand_Help tests serve subcommand help
func TestServeCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"serve", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("serve --help error: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "--port") {
		t.Error("serve help should contain --port option")
	}
	if !strings.Contains(output, "--host") {
		t.Error("serve help should contain --host option")
	}
	if !strings.Contains(output, "--open") {
		t.Error("serve help should contain --open option")
	}
}

// TestThemesCommand_Help tests themes subcommand help
func TestThemesCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"themes", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("themes --help error: %v", err)
	}
	output := out.String()

	if !strings.Contains(output, "list") {
		t.Error("themes help should contain 'list' subcommand")
	}
	if !strings.Contains(output, "show") {
		t.Error("themes help should contain 'show' subcommand")
	}
}

// TestDoctorCommand_Help tests doctor subcommand help
func TestDoctorCommand_Help(t *testing.T) {
	rootCmd.SetArgs([]string{"doctor", "--help"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("doctor --help error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "doctor") {
		t.Error("doctor help should contain 'doctor'")
	}
}

// TestInvalidSubcommand tests invalid subcommand
func TestInvalidSubcommand(t *testing.T) {
	rootCmd.SetArgs([]string{"nonexistent"})
	var errOut bytes.Buffer
	rootCmd.SetErr(&errOut)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("invalid subcommand should return an error")
	}
}

// TestPersistentFlags tests persistent flags
func TestPersistentFlags(t *testing.T) {
	// Verify --config persistent flag exists
	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("--config persistent flag should exist")
		return
	}
	if flag.DefValue != "book.yaml" {
		t.Errorf("--config default should be 'book.yaml', got %q", flag.DefValue)
	}

	// Verify --verbose persistent flag exists
	flag = rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("--verbose persistent flag should exist")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("--verbose default should be 'false', got %q", flag.DefValue)
	}

	flag = rootCmd.PersistentFlags().Lookup("cache-dir")
	if flag == nil {
		t.Fatal("--cache-dir persistent flag should exist")
	}
	if flag.DefValue != "" {
		t.Errorf("--cache-dir default should be empty, got %q", flag.DefValue)
	}

	flag = rootCmd.PersistentFlags().Lookup("no-cache")
	if flag == nil {
		t.Fatal("--no-cache persistent flag should exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("--no-cache default should be 'false', got %q", flag.DefValue)
	}
}

// TestBuildCommand_Flags tests build command flags
func TestBuildCommand_Flags(t *testing.T) {
	flag := buildCmd.Flags().Lookup("format")
	if flag == nil {
		t.Fatal("build should have --format flag")
		return
	}
	if flag.DefValue != "" {
		t.Errorf("--format default should be empty, got %q", flag.DefValue)
	}

	flag = buildCmd.Flags().Lookup("branch")
	if flag == nil {
		t.Error("build should have --branch flag")
	}

	flag = buildCmd.Flags().Lookup("subdir")
	if flag == nil {
		t.Error("build should have --subdir flag")
	}

	flag = buildCmd.Flags().Lookup("output")
	if flag == nil {
		t.Error("build should have --output flag")
	}
}

// TestServeCommand_Flags tests serve command flags
func TestServeCommand_Flags(t *testing.T) {
	flag := serveCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatal("serve should have --port flag")
		return
	}
	if flag.DefValue != "9000" {
		t.Errorf("--port default should be 9000, got %q", flag.DefValue)
	}

	flag = serveCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("serve should have --output flag")
		return
	}

	flag = serveCmd.Flags().Lookup("host")
	if flag == nil {
		t.Fatal("serve should have --host flag")
		return
	}
	if flag.DefValue != "127.0.0.1" {
		t.Errorf("--host default should be 127.0.0.1, got %q", flag.DefValue)
	}

	flag = serveCmd.Flags().Lookup("open")
	if flag == nil {
		t.Fatal("serve should have --open flag")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("--open default should be false, got %q", flag.DefValue)
	}
}

// TestThemesShowCommand_ArgsValidation tests themes show argument validation
func TestThemesShowCommand_ArgsValidation(t *testing.T) {
	rootCmd.SetArgs([]string{"themes", "show"})
	var errOut bytes.Buffer
	rootCmd.SetErr(&errOut)

	err := rootCmd.Execute()
	if err == nil {
		t.Error("themes show without arguments should return an error")
	}
}

// TestFlattenChapters tests the chapter flattening function
func TestFlattenChapters(t *testing.T) {
	tests := []struct {
		name    string
		input   []config.ChapterDef
		wantLen int
	}{
		{
			name:    "empty list",
			input:   nil,
			wantLen: 0,
		},
		{
			name: "no nesting",
			input: []config.ChapterDef{
				{Title: "Ch1", File: "ch1.md"},
				{Title: "Ch2", File: "ch2.md"},
			},
			wantLen: 2,
		},
		{
			name: "single-level nesting",
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
			name: "multi-level nesting",
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
				t.Errorf("flattenChapters returned %d, expected %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestGetPageDimensions tests page size conversion
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
				t.Errorf("getPageDimensions(%q) = (%v, %v), expected (%v, %v)",
					tt.size, w, h, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}

// TestGetAvailableThemes tests the available themes list
func TestGetAvailableThemes(t *testing.T) {
	themes := getAvailableThemes()

	if len(themes) == 0 {
		t.Error("should have at least one available theme")
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
			t.Error("theme name should not be empty")
		}
		if thm.DisplayName == "" {
			t.Errorf("theme %q display name should not be empty", thm.Name)
		}
		if thm.Description == "" {
			t.Errorf("theme %q description should not be empty", thm.Name)
		}
		if len(thm.Features) == 0 {
			t.Errorf("theme %q should have a features list", thm.Name)
		}
		if thm.Colors.Primary == "" {
			t.Errorf("theme %q should have a primary color", thm.Name)
		}
	}

	for name, found := range requiredThemes {
		if !found {
			t.Errorf("missing required theme: %q", name)
		}
	}
}

// TestExecuteThemesShow_ValidTheme tests showing a valid theme
func TestExecuteThemesShow_ValidTheme(t *testing.T) {
	defer suppressOutput(t)()
	err := executeThemesShow("technical")
	if err != nil {
		t.Errorf("showing technical theme should not error: %v", err)
	}
}

// TestExecuteThemesShow_InvalidTheme tests showing an invalid theme
func TestExecuteThemesShow_InvalidTheme(t *testing.T) {
	defer suppressOutput(t)()
	err := executeThemesShow("nonexistent_theme")
	if err == nil {
		t.Error("showing a non-existent theme should error")
	}
	if !strings.Contains(err.Error(), "theme not found") {
		t.Errorf("error message should contain 'theme not found', got: %q", err.Error())
	}
}

// TestExecuteThemesList tests listing themes
func TestExecuteThemesList(t *testing.T) {
	defer suppressOutput(t)()
	err := executeThemesList()
	if err != nil {
		t.Errorf("listing themes should not error: %v", err)
	}
}

// TestInferTitleFromPath tests inferring title from path
func TestInferTitleFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple file", "preface.md", "Preface"},
		{"subdirectory README", "chapter01/README.md", "Chapter01"},
		{"nested path", "part1/intro.md", "Part1 - intro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferTitleFromPath(tt.path)
			if got != tt.want {
				t.Errorf("inferTitleFromPath(%q) = %q, expected %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestCountChapterDefs tests the chapter counting function
func TestCountChapterDefs(t *testing.T) {
	tests := []struct {
		name  string
		input []config.ChapterDef
		want  int
	}{
		{"empty list", nil, 0},
		{"two top-level", []config.ChapterDef{{Title: "A"}, {Title: "B"}}, 2},
		{
			"with nesting",
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
				t.Errorf("countChapterDefs = %d, expected %d", got, tt.want)
			}
		})
	}
}
