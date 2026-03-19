package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
