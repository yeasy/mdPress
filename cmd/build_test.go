package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/renderer"
)

// TestBuildCommand_Creation tests that the build command is properly created
func TestBuildCommand_Creation(t *testing.T) {
	if buildCmd == nil {
		t.Fatal("buildCmd should not be nil")
	}

	if buildCmd.Use != "build [source]" {
		t.Errorf("buildCmd.Use should be 'build [source]', got %q", buildCmd.Use)
	}

	if buildCmd.Short != "Build documents (PDF/HTML/site/ePub/Typst)" {
		t.Errorf("buildCmd.Short should be 'Build documents (PDF/HTML/site/ePub/Typst)', got %q", buildCmd.Short)
	}
}

// TestBuildCommand_FlagRegistration tests that all required flags are registered
func TestBuildCommand_FlagRegistration(t *testing.T) {
	flags := []string{
		"format",
		"branch",
		"subdir",
		"output",
		"summary",
		"allow-plugins",
	}

	for _, f := range flags {
		flag := buildCmd.Flags().Lookup(f)
		if flag == nil {
			t.Errorf("build command should have --%s flag", f)
		}
	}

	// Verify shorthands.
	if f := buildCmd.Flags().Lookup("format"); f != nil && f.Shorthand != "f" {
		t.Errorf("build --format should have shorthand -f, got %q", f.Shorthand)
	}
	if f := buildCmd.Flags().Lookup("output"); f != nil && f.Shorthand != "o" {
		t.Errorf("build --output should have shorthand -o, got %q", f.Shorthand)
	}
}

func TestExecuteBuild_UsesConfigBaseDirForMultilingualProject(t *testing.T) {
	defer suppressOutput(t)()

	origCfgFile := cfgFile
	origBuildFormat := buildFormat
	origBuildOutput := buildOutput
	origBuildSummary := buildSummary
	origBuildSubDir := buildSubDir
	origBuildBranch := buildBranch
	origQuiet := quiet
	origVerbose := verbose
	defer func() {
		cfgFile = origCfgFile
		buildFormat = origBuildFormat
		buildOutput = origBuildOutput
		buildSummary = origBuildSummary
		buildSubDir = origBuildSubDir
		buildBranch = origBuildBranch
		quiet = origQuiet
		verbose = origVerbose
	}()

	tmpDir := t.TempDir()
	workspaceDir := filepath.Join(tmpDir, "workspace")
	projectDir := filepath.Join(tmpDir, "projects", "multilingual-book")
	for _, dir := range []string{
		workspaceDir,
		filepath.Join(projectDir, "en"),
		filepath.Join(projectDir, "zh"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create directory %q: %v", dir, err)
		}
	}

	writeFile := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	writeFile(filepath.Join(projectDir, "book.yaml"), `book:
  title: "Language Root"
chapters:
  - title: "Overview"
    file: "README.md"
output:
  formats: ["html"]
`)
	writeFile(filepath.Join(projectDir, "LANGS.md"), `# Languages

- [English](en/)
- [中文](zh/)
`)
	writeFile(filepath.Join(projectDir, "README.md"), "# Overview\n\nRoot overview.\n")
	writeFile(filepath.Join(projectDir, "en", "book.yaml"), `book:
  title: "English Book"
chapters:
  - title: "Intro"
    file: "README.md"
output:
  formats: ["html"]
  filename: "book.html"
`)
	writeFile(filepath.Join(projectDir, "en", "README.md"), "# Intro\n\nEnglish content.\n")
	writeFile(filepath.Join(projectDir, "zh", "book.yaml"), `book:
  title: "中文书"
chapters:
  - title: "简介"
    file: "README.md"
output:
  formats: ["html"]
  filename: "book.html"
`)
	writeFile(filepath.Join(projectDir, "zh", "README.md"), "# 简介\n\n中文内容。\n")

	t.Chdir(workspaceDir)
	cfgFile = filepath.Join("..", "projects", "multilingual-book", "book.yaml")
	buildFormat = "html"
	buildOutput = ""
	buildSummary = ""
	buildSubDir = ""
	buildBranch = ""
	quiet = true
	verbose = false

	if err := executeBuild(context.Background(), ""); err != nil {
		t.Fatalf("executeBuild() returned error: %v", err)
	}

	for _, expected := range []string{
		filepath.Join(projectDir, "_mdpress_langs.html"),
		filepath.Join(projectDir, "en", "book.html"),
		filepath.Join(projectDir, "zh", "book.html"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected generated file %q: %v", expected, err)
		}
	}

	if _, err := os.Stat(filepath.Join(workspaceDir, "_mdpress_langs.html")); !os.IsNotExist(err) {
		t.Fatalf("landing page should not be written to workspace dir, stat err=%v", err)
	}
}

// TestBuildCommand_FlagDefaults tests that build command flags have correct defaults
func TestBuildCommand_FlagDefaults(t *testing.T) {
	for _, name := range []string{"format", "branch", "subdir", "output", "summary"} {
		t.Run(name, func(t *testing.T) {
			f := buildCmd.Flags().Lookup(name)
			if f == nil {
				t.Fatalf("%s flag should exist", name)
				return // unreachable, but satisfies staticcheck SA5011
			}
			if f.DefValue != "" {
				t.Errorf("%s default value should be empty, got %q", name, f.DefValue)
			}
		})
	}
}

// TestBuildCommand_LongDescription tests that build command has comprehensive documentation
func TestBuildCommand_LongDescription(t *testing.T) {
	if buildCmd.Long == "" {
		t.Error("buildCmd.Long should not be empty")
	}

	requiredPhrases := []string{
		"Build high-quality documents",
		"Local directory",
		"GitHub repository",
		"pdf",
		"html",
		"site",
		"epub",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(buildCmd.Long, phrase) {
			t.Errorf("buildCmd.Long should contain %q", phrase)
		}
	}
}

// TestBuildCommand_HasRunE tests that build command has a RunE function
func TestBuildCommand_HasRunE(t *testing.T) {
	if buildCmd.RunE == nil {
		t.Fatal("buildCmd should have a RunE function")
	}
}

// TestFlattenChapters_EmptyList tests flattening of empty chapter list
func TestFlattenChapters_EmptyList(t *testing.T) {
	result := config.FlattenChapters(nil)
	if len(result) != 0 {
		t.Errorf("flattening empty list should return empty, got %d", len(result))
	}
}

// TestFlattenChapters_SingleLevel tests flattening of single-level chapters
func TestFlattenChapters_SingleLevel(t *testing.T) {
	chapters := []config.ChapterDef{
		{Title: "Chapter 1", File: "ch1.md"},
		{Title: "Chapter 2", File: "ch2.md"},
	}

	result := config.FlattenChapters(chapters)
	if len(result) != 2 {
		t.Errorf("flattening 2 chapters should return 2, got %d", len(result))
	}
}

// TestFlattenChapters_WithSections tests flattening with nested sections
func TestFlattenChapters_WithSections(t *testing.T) {
	chapters := []config.ChapterDef{
		{
			Title: "Part 1",
			File:  "part1.md",
			Sections: []config.ChapterDef{
				{Title: "Section 1.1", File: "sec1_1.md"},
				{Title: "Section 1.2", File: "sec1_2.md"},
			},
		},
		{Title: "Part 2", File: "part2.md"},
	}

	result := config.FlattenChapters(chapters)
	if len(result) < 4 {
		t.Errorf("flattening with sections should return at least 4 items, got %d", len(result))
	}
}

// TestRewriteChapterLinks_EmptyChapters tests rewriting with empty chapter list
func TestRewriteChapterLinks_EmptyChapters(t *testing.T) {
	chapters := []renderer.ChapterHTML{}
	files := []string{}

	result := rewriteChapterLinks(chapters, files)
	if len(result) != 0 {
		t.Errorf("rewriting empty chapters should return empty, got %d", len(result))
	}
}

// TestRewriteChapterLinks_MismatchedLengths tests rewriting with mismatched lengths
func TestRewriteChapterLinks_MismatchedLengths(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{ID: "ch1", Content: "<p>Chapter 1</p>"},
		{ID: "ch2", Content: "<p>Chapter 2</p>"},
	}
	files := []string{"ch1.md"}

	result := rewriteChapterLinks(chapters, files)
	if len(result) != 2 {
		t.Errorf("mismatched lengths should return original chapters, got %d", len(result))
	}
}

// TestRewriteChapterLinks_ValidChapters tests rewriting with valid chapters
func TestRewriteChapterLinks_ValidChapters(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{ID: "ch1", Content: "<p>Chapter 1</p>"},
		{ID: "ch2", Content: "<p>Chapter 2</p>"},
	}
	files := []string{"ch1.md", "ch2.md"}

	result := rewriteChapterLinks(chapters, files)
	if len(result) != 2 {
		t.Errorf("should process 2 chapters, got %d", len(result))
	}

	if result[0].ID != "ch1" {
		t.Errorf("first chapter ID should be ch1, got %q", result[0].ID)
	}

	if result[1].ID != "ch2" {
		t.Errorf("second chapter ID should be ch2, got %q", result[1].ID)
	}
}

// TestRewriteChapterLinks_EmptyChapterID tests rewriting with empty chapter ID
func TestRewriteChapterLinks_EmptyChapterID(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{ID: "", Content: "<p>Chapter 1</p>"},
	}
	files := []string{"ch1.md"}

	result := rewriteChapterLinks(chapters, files)
	if len(result) != 1 {
		t.Errorf("should return 1 chapter, got %d", len(result))
	}
}

// TestRewriteChapterLinks_EmptyFileName tests rewriting with empty file name
func TestRewriteChapterLinks_EmptyFileName(t *testing.T) {
	chapters := []renderer.ChapterHTML{
		{ID: "ch1", Content: "<p>Chapter 1</p>"},
	}
	files := []string{""}

	result := rewriteChapterLinks(chapters, files)
	if len(result) != 1 {
		t.Errorf("should return 1 chapter, got %d", len(result))
	}
}

// TestRewriteMarkdownLinksInHTML_EmptyContent tests link rewriting with empty content
func TestRewriteMarkdownLinksInHTML_EmptyContent(t *testing.T) {
	targets := map[string]string{
		"chapter1.md": "ch1",
	}

	result := rewriteMarkdownLinksInHTML("", "current.md", targets)
	if result != "" {
		t.Errorf("empty content should return empty, got %q", result)
	}
}

// TestRewriteMarkdownLinksInHTML_NoTargets tests link rewriting with no targets
func TestRewriteMarkdownLinksInHTML_NoTargets(t *testing.T) {
	html := "<p>Some content</p>"
	targets := map[string]string{}

	result := rewriteMarkdownLinksInHTML(html, "current.md", targets)
	if result != html {
		t.Errorf("with no targets, content should be unchanged, got %q", result)
	}
}

// TestRewriteMarkdownLinksInHTML_ValidTargets tests link rewriting with valid targets
func TestRewriteMarkdownLinksInHTML_ValidTargets(t *testing.T) {
	html := `<p>See <a href="chapter1.md">Chapter 1</a></p>`
	targets := map[string]string{
		"chapter1.md": "ch1",
	}

	result := rewriteMarkdownLinksInHTML(html, "current.md", targets)
	if !strings.Contains(result, `href="#ch1"`) {
		t.Errorf("chapter1.md should be rewritten to #ch1, got %q", result)
	}
	if strings.Contains(result, `href="chapter1.md"`) {
		t.Error("original chapter1.md link should no longer appear")
	}
}

// TestBuildCommand_ExamplesInHelp tests that build help contains examples
func TestBuildCommand_ExamplesInHelp(t *testing.T) {
	if !strings.Contains(buildCmd.Long, "mdpress build") {
		t.Error("build help should contain example 'mdpress build'")
	}

	if !strings.Contains(buildCmd.Long, "--format") {
		t.Error("build help should mention --format flag")
	}
}

// TestBuildCommand_SupportsMultipleFormats tests that build help mentions format options
func TestBuildCommand_SupportsMultipleFormats(t *testing.T) {
	formats := []string{"pdf", "html", "site", "epub"}

	for _, format := range formats {
		if !strings.Contains(buildCmd.Long, format) {
			t.Errorf("build help should mention %q format", format)
		}
	}
}

// TestNewBuildCmd ensures newBuildCmd creates proper cobra command
func TestNewBuildCmd(t *testing.T) {
	if buildCmd == nil {
		t.Fatal("buildCmd must not be nil")
	}

	if buildCmd.Use == "" {
		t.Error("buildCmd.Use should not be empty")
	}

	if buildCmd.Short == "" {
		t.Error("buildCmd.Short should not be empty")
	}

	if buildCmd.RunE == nil {
		t.Error("buildCmd.RunE should not be nil")
	}
}

// TestBuildFlagTypes ensures all flags have correct types
func TestBuildFlagTypes(t *testing.T) {
	for _, name := range []string{"format", "branch", "subdir", "output", "summary"} {
		t.Run(name, func(t *testing.T) {
			f := buildCmd.Flags().Lookup(name)
			if f == nil {
				t.Fatalf("%s flag should exist", name)
				return
			}
			if f.Value.Type() != "string" {
				t.Errorf("%s flag should be string type, got %q", name, f.Value.Type())
			}
		})
	}
}

// TestResolveSiteOutputDir verifies the single-language "site" output directory:
// default "_book" under the project; an existing directory (or trailing
// separator) is used verbatim; a file-ish base becomes "<base>_site" so the
// other formats can share the same base path.
func TestResolveSiteOutputDir(t *testing.T) {
	existingDir := t.TempDir()
	missing := filepath.Join(existingDir, "missing")
	sep := string(os.PathSeparator)

	tests := []struct {
		name     string
		baseDir  string
		override string
		want     string
	}{
		{"default is _book under project", "/proj", "", filepath.Join("/proj", "_book")},
		{"existing directory used verbatim", "/proj", existingDir, existingDir},
		{"trailing separator forces directory", "/proj", missing + sep, missing},
		{"file path with extension becomes <base>_site", "/proj", filepath.Join(missing, "manual.html"), filepath.Join(missing, "manual_site")},
		{"pdf file path becomes <base>_site", "/proj", filepath.Join(missing, "manual.pdf"), filepath.Join(missing, "manual_site")},
		{"extension-less base becomes <base>_site", "/proj", filepath.Join(missing, "book"), filepath.Join(missing, "book_site")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveSiteOutputDir(tt.baseDir, tt.override); got != tt.want {
				t.Errorf("resolveSiteOutputDir(%q, %q) = %q, want %q", tt.baseDir, tt.override, got, tt.want)
			}
		})
	}
}
