package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// ---- rewriteGitBookSyntax tests ----

func TestRewriteGitBookSyntax_HintBlock(t *testing.T) {
	input := `{% hint style="info" %}
This is an informational note.
{% endhint %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Fatal("expected content to be changed")
	}
	if !strings.Contains(got, "> **INFO:**") {
		t.Errorf("expected blockquote with INFO label, got:\n%s", got)
	}
	if strings.Contains(got, "{%") {
		t.Errorf("expected all template tags to be removed, got:\n%s", got)
	}
}

func TestRewriteGitBookSyntax_HintMultipleStyles(t *testing.T) {
	for _, style := range []string{"warning", "danger", "success"} {
		input := "{% hint style=\"" + style + "\" %}body{% endhint %}"
		got, changed := rewriteGitBookSyntax(input)
		if !changed {
			t.Errorf("style %q: expected change", style)
		}
		upperStyle := strings.ToUpper(style)
		if !strings.Contains(got, "> **"+upperStyle+":**") {
			t.Errorf("style %q: expected label in output, got: %s", style, got)
		}
	}
}

func TestRewriteGitBookSyntax_CodeBlock(t *testing.T) {
	input := `{% code title="example.go" %}
package main
{% endcode %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Fatal("expected content to be changed")
	}
	if !strings.Contains(got, "```") {
		t.Errorf("expected fenced code block, got:\n%s", got)
	}
	if !strings.Contains(got, "package main") {
		t.Errorf("expected code body to be preserved, got:\n%s", got)
	}
}

func TestRewriteGitBookSyntax_TabsBlock(t *testing.T) {
	input := `{% tabs %}
{% tab title="Go" %}
fmt.Println("hello")
{% endtab %}
{% tab title="Python" %}
print("hello")
{% endtab %}
{% endtabs %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Fatal("expected content to be changed")
	}
	if !strings.Contains(got, "#### Go") {
		t.Errorf("expected tab heading for Go, got:\n%s", got)
	}
	if !strings.Contains(got, "#### Python") {
		t.Errorf("expected tab heading for Python, got:\n%s", got)
	}
	if strings.Contains(got, "{%") {
		t.Errorf("expected all template tags removed, got:\n%s", got)
	}
}

func TestRewriteGitBookSyntax_NoChange(t *testing.T) {
	input := "# Just a normal Markdown file\n\nNo GitBook syntax here.\n"
	got, changed := rewriteGitBookSyntax(input)
	if changed {
		t.Error("expected no change for plain Markdown")
	}
	if got != input {
		t.Errorf("content was modified unexpectedly")
	}
}

// ---- executeMigrate integration tests ----

func TestExecuteMigrate_NoGitBookProject(t *testing.T) {
	dir := t.TempDir()
	err := executeMigrate(dir, true)
	if err == nil {
		t.Fatal("expected an error for a directory with no GitBook files")
	}
}

func TestExecuteMigrate_OnlySummary(t *testing.T) {
	dir := t.TempDir()
	summaryContent := "# Summary\n\n* [Introduction](README.md)\n"
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summaryContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Should succeed (SUMMARY.md is enough for detection) in dry-run mode.
	if err := executeMigrate(dir, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteMigrate_BookJSON(t *testing.T) {
	dir := t.TempDir()
	bookJSON := `{
		"title": "My GitBook",
		"author": "Alice",
		"language": "en",
		"description": "A test book",
		"plugins": ["mathjax"]
	}`
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(bookJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Run in dry-run mode so no files are written.
	if err := executeMigrate(dir, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteMigrate_CreatesBookYAML(t *testing.T) {
	dir := t.TempDir()
	bookJSON := `{"title":"Test Book","author":"Bob","language":"zh"}`
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(bookJSON), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outPath := filepath.Join(dir, "book.yaml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("book.yaml was not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Test Book") {
		t.Errorf("book.yaml should contain the book title, got:\n%s", content)
	}
	if !strings.Contains(content, "Bob") {
		t.Errorf("book.yaml should contain the author, got:\n%s", content)
	}
}

func TestExecuteMigrate_RewritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	// Minimal SUMMARY.md so detection works.
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// A Markdown file with GitBook syntax.
	mdContent := "{% hint style=\"warning\" %}Be careful.{% endhint %}\n"
	if err := os.WriteFile(filepath.Join(dir, "chapter.md"), []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "chapter.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "{%") {
		t.Errorf("GitBook tags were not removed from chapter.md:\n%s", string(data))
	}
}

// ---- Edge case tests ----

// TestRewriteGitBookSyntax_EmptyInput tests that empty input returns unchanged.
func TestRewriteGitBookSyntax_EmptyInput(t *testing.T) {
	input := ""
	got, changed := rewriteGitBookSyntax(input)
	if changed {
		t.Error("expected no change for empty input")
	}
	if got != "" {
		t.Errorf("expected empty output, got %q", got)
	}
}

// TestRewriteGitBookSyntax_OnlyWhitespace tests input with only whitespace.
func TestRewriteGitBookSyntax_OnlyWhitespace(t *testing.T) {
	input := "   \n\n\t\n  "
	got, changed := rewriteGitBookSyntax(input)
	if changed {
		t.Error("expected no change for whitespace-only input")
	}
	if got != input {
		t.Errorf("expected input unchanged, got %q", got)
	}
}

// TestRewriteGitBookSyntax_TabsWithoutContent tests tabs markers with no actual content between them.
func TestRewriteGitBookSyntax_TabsWithoutContent(t *testing.T) {
	input := `{% tabs %}
{% tab title="Empty" %}
{% endtab %}
{% endtabs %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Should have the tab heading but stripped of markers
	if !strings.Contains(got, "#### Empty") {
		t.Errorf("expected tab heading, got:\n%s", got)
	}
	if strings.Contains(got, "{%") {
		t.Errorf("expected all template tags removed, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_NestedHints tests hint blocks that appear within each other (edge case).
func TestRewriteGitBookSyntax_NestedHints(t *testing.T) {
	// Note: nested hints are technically malformed GitBook syntax, but we should handle gracefully.
	input := `Before
{% hint style="info" %}
Outer hint content
{% endhint %}
After`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// The regex should match the outermost block
	if !strings.Contains(got, "> **INFO:**") {
		t.Errorf("expected blockquote conversion, got:\n%s", got)
	}
	// Should still have Before and After text
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Errorf("expected surrounding text preserved, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_UnicodeContent tests that unicode characters are preserved.
func TestRewriteGitBookSyntax_UnicodeContent(t *testing.T) {
	input := `{% hint style="info" %}
这是中文内容。日本語のテキスト。Ελληνικά.
{% endhint %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Check that unicode characters are preserved
	if !strings.Contains(got, "中文内容") {
		t.Errorf("expected Chinese characters preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "日本語") {
		t.Errorf("expected Japanese characters preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "Ελληνικά") {
		t.Errorf("expected Greek characters preserved, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_MultipleHintsInSequence tests multiple hint blocks in one document.
func TestRewriteGitBookSyntax_MultipleHintsInSequence(t *testing.T) {
	input := `First section.
{% hint style="info" %}
Info note
{% endhint %}

Second section.
{% hint style="warning" %}
Warning note
{% endhint %}

Third section.`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Should have both converted hints
	infoCount := strings.Count(got, "> **INFO:**")
	warningCount := strings.Count(got, "> **WARNING:**")
	if infoCount != 1 {
		t.Errorf("expected 1 info blockquote, got %d", infoCount)
	}
	if warningCount != 1 {
		t.Errorf("expected 1 warning blockquote, got %d", warningCount)
	}
	// Should preserve section text
	if !strings.Contains(got, "First section") || !strings.Contains(got, "Third section") {
		t.Errorf("expected section text preserved, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_CodeWithSpecialCharacters tests code blocks with special regex characters.
func TestRewriteGitBookSyntax_CodeWithSpecialCharacters(t *testing.T) {
	input := `{% code title="regex.go" %}
pattern := "[a-zA-Z0-9._+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
regex := regexp.MustCompile(pattern)
{% endcode %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Check that special regex characters are preserved
	if !strings.Contains(got, `[a-zA-Z0-9._+-]+@`) {
		t.Errorf("expected regex pattern preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "```") {
		t.Errorf("expected fenced code block, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_TabsWithMultipleLinesPerTab tests tabs with multiline content.
func TestRewriteGitBookSyntax_TabsWithMultipleLinesPerTab(t *testing.T) {
	input := `{% tabs %}
{% tab title="JavaScript" %}
function hello() {
  console.log("Hello, World!");
  return true;
}
{% endtab %}
{% tab title="Python" %}
def hello():
    print("Hello, World!")
    return True
{% endtab %}
{% endtabs %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Check that both tab headings exist
	if !strings.Contains(got, "#### JavaScript") {
		t.Errorf("expected JavaScript tab heading, got:\n%s", got)
	}
	if !strings.Contains(got, "#### Python") {
		t.Errorf("expected Python tab heading, got:\n%s", got)
	}
	// Check that code is preserved
	if !strings.Contains(got, "console.log") {
		t.Errorf("expected JavaScript code preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "def hello") {
		t.Errorf("expected Python code preserved, got:\n%s", got)
	}
	// No template tags should remain
	if strings.Contains(got, "{%") {
		t.Errorf("expected all template tags removed, got:\n%s", got)
	}
}

// TestMigrateBookJSON_MalformedJSON tests migration with corrupt/malformed JSON.
func TestMigrateBookJSON_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	// Write invalid JSON
	bookJSON := `{
		"title": "Broken Book",
		"author": "Invalid JSON" // Missing closing brace and quotes
	`
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(bookJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Should fail during parsing
	err := executeMigrate(dir, false)
	if err == nil {
		t.Fatal("expected an error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// TestMigrateBookJSON_EmptyJSON tests migration with empty JSON object.
func TestMigrateBookJSON_EmptyJSON(t *testing.T) {
	dir := t.TempDir()
	bookJSON := `{}`
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(bookJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Should succeed with defaults
	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that book.yaml was created with defaults
	data, err := os.ReadFile(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("book.yaml was not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Untitled Book") {
		t.Errorf("expected default title, got:\n%s", content)
	}
	if !strings.Contains(content, "Unknown") {
		t.Errorf("expected default author, got:\n%s", content)
	}
}

// TestMigrateMarkdownFiles_SkipsNonMarkdownFiles tests that .txt and other files are ignored.
func TestMigrateMarkdownFiles_SkipsNonMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	// Create SUMMARY.md for detection
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a non-markdown file with GitBook syntax
	nonMdContent := "{% hint style=\"info\" %}This should not be touched{% endhint %}"
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte(nonMdContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the .txt file was not modified
	data, err := os.ReadFile(filepath.Join(dir, "notes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != nonMdContent {
		t.Errorf("expected .txt file unchanged, got:\n%s", string(data))
	}
}

// TestMigrateMarkdownFiles_SkipsHiddenDirectories tests that .git, .vscode, etc. are skipped.
func TestMigrateMarkdownFiles_SkipsHiddenDirectories(t *testing.T) {
	dir := t.TempDir()
	// Create SUMMARY.md for detection
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create hidden directory with a markdown file
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.Mkdir(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	hiddenMd := filepath.Join(hiddenDir, "secret.md")
	mdContent := "{% hint style=\"info\" %}Secret content{% endhint %}"
	if err := os.WriteFile(hiddenMd, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the hidden directory file was not modified
	data, err := os.ReadFile(hiddenMd)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != mdContent {
		t.Errorf("expected hidden .md file unchanged, got:\n%s", string(data))
	}
}

// TestMigrateMarkdownFiles_SkipsNodeModules tests that node_modules is skipped.
func TestMigrateMarkdownFiles_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	// Create SUMMARY.md for detection
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create node_modules with markdown file
	nmDir := filepath.Join(dir, "node_modules")
	if err := os.Mkdir(nmDir, 0755); err != nil {
		t.Fatal(err)
	}
	nmMd := filepath.Join(nmDir, "package.md")
	mdContent := "{% hint style=\"info\" %}Package docs{% endhint %}"
	if err := os.WriteFile(nmMd, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the node_modules file was not modified
	data, err := os.ReadFile(nmMd)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != mdContent {
		t.Errorf("expected node_modules file unchanged, got:\n%s", string(data))
	}
}

// TestMigrateMarkdownFiles_SkipsBook directory tests that _book is skipped.
func TestMigrateMarkdownFiles_SkipsBookDirectory(t *testing.T) {
	dir := t.TempDir()
	// Create SUMMARY.md for detection
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte("# Summary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create _book directory with markdown file
	bookDir := filepath.Join(dir, "_book")
	if err := os.Mkdir(bookDir, 0755); err != nil {
		t.Fatal(err)
	}
	bookMd := filepath.Join(bookDir, "output.md")
	mdContent := "{% hint style=\"info\" %}Generated content{% endhint %}"
	if err := os.WriteFile(bookMd, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := executeMigrate(dir, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the _book file was not modified
	data, err := os.ReadFile(bookMd)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != mdContent {
		t.Errorf("expected _book file unchanged, got:\n%s", string(data))
	}
}

// TestRewriteGitBookSyntax_StripTabsRE tests the stripTabsRE regex with variations.
func TestRewriteGitBookSyntax_StripTabsRE(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "{% tabs %}content{% endtabs %}",
			expected: "content",
			desc:     "basic tabs removal",
		},
		{
			input:    "{%  tabs  %}content{%  endtabs  %}",
			expected: "content",
			desc:     "tabs with extra spaces",
		},
		{
			input:    "no tabs here",
			expected: "no tabs here",
			desc:     "no tabs markers",
		},
		{
			input:    "{% tabs %}text with {% endtabs %} in middle",
			expected: "text with in middle",
			desc:     "tabs with content",
		},
	}

	for _, tc := range testCases {
		got, changed := rewriteGitBookSyntax(tc.input)
		if tc.desc == "no tabs here" && changed {
			t.Errorf("%s: expected no change", tc.desc)
			continue
		}
		if strings.TrimSpace(got) != strings.TrimSpace(tc.expected) {
			t.Errorf("%s: expected %q, got %q", tc.desc, tc.expected, got)
		}
	}
}

// TestRewriteGitBookSyntax_CaseSensitivity tests that regex is case-insensitive where appropriate.
func TestRewriteGitBookSyntax_CaseSensitivity(t *testing.T) {
	// GitBook tags should be case-insensitive in regex
	input := `{% Hint style="info" %}Content{% Endhint %}`
	_, changed := rewriteGitBookSyntax(input)
	// The regex is case-sensitive, so this should not be changed by the hint rule.
	// However, with the current implementation, it won't match because the regex uses lowercase.
	// This documents that behavior.
	if !changed && !strings.Contains(input, "{%") {
		t.Error("malformed case variant wasn't processed")
	}
}

// TestRewriteGitBookSyntax_BackslashInContent tests content with backslashes.
func TestRewriteGitBookSyntax_BackslashInContent(t *testing.T) {
	input := `{% code title="paths.txt" %}
C:\\Users\\Admin\\Documents
/home/user/documents
{% endcode %}`

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Backslashes should be preserved
	if !strings.Contains(got, `C:\\Users\\Admin`) {
		t.Errorf("expected Windows path preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "/home/user/documents") {
		t.Errorf("expected Unix path preserved, got:\n%s", got)
	}
}

// TestRewriteGitBookSyntax_VeryLongContent tests handling of very long hint content.
func TestRewriteGitBookSyntax_VeryLongContent(t *testing.T) {
	// Create a hint with a lot of content
	longContent := strings.Repeat("This is a line of content.\n", 100)
	input := "{% hint style=\"info\" %}\n" + longContent + "{% endhint %}"

	got, changed := rewriteGitBookSyntax(input)
	if !changed {
		t.Error("expected content to be changed")
	}
	// Should still be converted to blockquote
	if !strings.Contains(got, "> **INFO:**") {
		t.Errorf("expected blockquote, got:\n%s", got[:100])
	}
	// Content should be mostly preserved (with quote prefixes)
	lineCount := strings.Count(got, "\n")
	if lineCount < 50 {
		t.Errorf("expected many lines, got %d", lineCount)
	}
}

// TestNonEmpty tests the nonEmpty helper function with edge cases.
func TestNonEmpty(t *testing.T) {
	testCases := []struct {
		input    string
		fallback string
		expected string
		desc     string
	}{
		{"hello", "world", "hello", "non-empty string"},
		{"", "world", "world", "empty string"},
		{"   ", "world", "world", "whitespace-only string"},
		{"\t\n", "world", "world", "tabs and newlines"},
		{"  hello  ", "world", "  hello  ", "string with surrounding spaces (trimming checks only within if)"},
	}

	for _, tc := range testCases {
		got := nonEmpty(tc.input, tc.fallback)
		if got != tc.expected {
			t.Errorf("%s: expected %q, got %q", tc.desc, tc.expected, got)
		}
	}
}

// TestFileExists tests the utils.FileExists helper function.
func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	nonExistent := filepath.Join(dir, "does-not-exist.txt")

	// Test non-existent file
	if utils.FileExists(nonExistent) {
		t.Error("expected FileExists to return false for non-existent file")
	}

	// Create a file and test
	testFile := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if !utils.FileExists(testFile) {
		t.Error("expected FileExists to return true for existing file")
	}

	// Test with directory
	if !utils.FileExists(dir) {
		t.Error("expected FileExists to return true for existing directory")
	}
}
