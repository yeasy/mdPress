package cmd

import (
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

	if buildCmd.Short != "Build documents (PDF/HTML/ePub)" {
		t.Errorf("buildCmd.Short should be 'Build documents (PDF/HTML/ePub)', got %q", buildCmd.Short)
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
	}

	for _, f := range flags {
		flag := buildCmd.Flags().Lookup(f)
		if flag == nil {
			t.Errorf("build command should have --%s flag", f)
		}
	}
}

// TestBuildCommand_FormatFlagDefaults tests that the format flag has correct defaults
func TestBuildCommand_FormatFlagDefaults(t *testing.T) {
	flag := buildCmd.Flags().Lookup("format")
	if flag == nil {
		t.Fatal("format flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("format default value should be empty, got %q", flag.DefValue)
	}
}

// TestBuildCommand_BranchFlagDefaults tests that the branch flag has correct defaults
func TestBuildCommand_BranchFlagDefaults(t *testing.T) {
	flag := buildCmd.Flags().Lookup("branch")
	if flag == nil {
		t.Fatal("branch flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("branch default value should be empty, got %q", flag.DefValue)
	}
}

// TestBuildCommand_SubdirFlagDefaults tests that the subdir flag has correct defaults
func TestBuildCommand_SubdirFlagDefaults(t *testing.T) {
	flag := buildCmd.Flags().Lookup("subdir")
	if flag == nil {
		t.Fatal("subdir flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("subdir default value should be empty, got %q", flag.DefValue)
	}
}

// TestBuildCommand_OutputFlagDefaults tests that the output flag has correct defaults
func TestBuildCommand_OutputFlagDefaults(t *testing.T) {
	flag := buildCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("output flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("output default value should be empty, got %q", flag.DefValue)
	}
}

// TestBuildCommand_SummaryFlagDefaults tests that the summary flag has correct defaults
func TestBuildCommand_SummaryFlagDefaults(t *testing.T) {
	flag := buildCmd.Flags().Lookup("summary")
	if flag == nil {
		t.Fatal("summary flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("summary default value should be empty, got %q", flag.DefValue)
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
	result := flattenChapters(nil)
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

	result := flattenChapters(chapters)
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

	result := flattenChapters(chapters)
	if len(result) < 4 {
		t.Errorf("flattening with sections should return at least 4 items, got %d", len(result))
	}
}

// TestGetPageDimensions_A4 tests A4 page dimension
func TestGetPageDimensions_A4(t *testing.T) {
	w, h := getPageDimensions("A4")
	if w != 210 || h != 297 {
		t.Errorf("A4 dimensions should be (210, 297), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_A5 tests A5 page dimension
func TestGetPageDimensions_A5(t *testing.T) {
	w, h := getPageDimensions("A5")
	if w != 148 || h != 210 {
		t.Errorf("A5 dimensions should be (148, 210), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_Letter tests Letter page dimension
func TestGetPageDimensions_Letter(t *testing.T) {
	w, h := getPageDimensions("LETTER")
	if w != 216 || h != 279 {
		t.Errorf("LETTER dimensions should be (216, 279), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_Legal tests Legal page dimension
func TestGetPageDimensions_Legal(t *testing.T) {
	w, h := getPageDimensions("LEGAL")
	if w != 216 || h != 356 {
		t.Errorf("LEGAL dimensions should be (216, 356), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_B5 tests B5 page dimension
func TestGetPageDimensions_B5(t *testing.T) {
	w, h := getPageDimensions("B5")
	if w != 176 || h != 250 {
		t.Errorf("B5 dimensions should be (176, 250), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_Lowercase tests lowercase size names
func TestGetPageDimensions_Lowercase(t *testing.T) {
	w, h := getPageDimensions("a4")
	if w != 210 || h != 297 {
		t.Errorf("lowercase a4 should work like A4, got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_Unknown tests unknown size defaults to A4
func TestGetPageDimensions_Unknown(t *testing.T) {
	w, h := getPageDimensions("unknown")
	if w != 210 || h != 297 {
		t.Errorf("unknown size should default to A4 (210, 297), got (%v, %v)", w, h)
	}
}

// TestGetPageDimensions_Empty tests empty string defaults to A4
func TestGetPageDimensions_Empty(t *testing.T) {
	w, h := getPageDimensions("")
	if w != 210 || h != 297 {
		t.Errorf("empty size should default to A4 (210, 297), got (%v, %v)", w, h)
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
	if result == "" {
		t.Error("should return content even with no targets")
	}
}

// TestRewriteMarkdownLinksInHTML_ValidTargets tests link rewriting with valid targets
func TestRewriteMarkdownLinksInHTML_ValidTargets(t *testing.T) {
	html := `<p>See <a href="chapter1.md">Chapter 1</a></p>`
	targets := map[string]string{
		"chapter1.md": "ch1",
	}

	result := rewriteMarkdownLinksInHTML(html, "current.md", targets)
	if result == "" {
		t.Error("should return processed content")
	}
}

// TestBuildVariables_Initialization tests that build variables are initialized
func TestBuildVariables_Initialization(t *testing.T) {
	_ = buildFormat
	_ = buildBranch
	_ = buildSubDir
	_ = buildOutput
	_ = buildSummary
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
	formatFlag := buildCmd.Flags().Lookup("format")
	if formatFlag == nil {
		t.Fatal("format flag should exist")
	}
	if formatFlag.Value.Type() != "string" {
		t.Errorf("format flag should be string type, got %q", formatFlag.Value.Type())
	}

	branchFlag := buildCmd.Flags().Lookup("branch")
	if branchFlag == nil {
		t.Fatal("branch flag should exist")
	}
	if branchFlag.Value.Type() != "string" {
		t.Errorf("branch flag should be string type, got %q", branchFlag.Value.Type())
	}

	subdirFlag := buildCmd.Flags().Lookup("subdir")
	if subdirFlag == nil {
		t.Fatal("subdir flag should exist")
	}
	if subdirFlag.Value.Type() != "string" {
		t.Errorf("subdir flag should be string type, got %q", subdirFlag.Value.Type())
	}

	outputFlag := buildCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("output flag should exist")
	}
	if outputFlag.Value.Type() != "string" {
		t.Errorf("output flag should be string type, got %q", outputFlag.Value.Type())
	}

	summaryFlag := buildCmd.Flags().Lookup("summary")
	if summaryFlag == nil {
		t.Fatal("summary flag should exist")
	}
	if summaryFlag.Value.Type() != "string" {
		t.Errorf("summary flag should be string type, got %q", summaryFlag.Value.Type())
	}
}
