// declared_metadata_test.go covers the metadata a GitBook-style project
// declares in book.json and mdpress used to throw away, because "declared" was
// inferred from the value instead of from the key's presence.
package config

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// writeGitBookProject lays out the standard GitBook shape: book.json,
// SUMMARY.md, README.md and one chapter.
func writeGitBookProject(t *testing.T, bookJSON, readme string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"book.json":  bookJSON,
		"SUMMARY.md": "# Summary\n\n* [Intro](intro.md)\n",
		"README.md":  readme,
		"intro.md":   "# Intro\n\nBody.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestBookJSONVersionSurvivesGitTag pins the version bug: "1.0.0" is both the
// most common version string a book declares and DefaultConfig's value, so the
// old `== DefaultConfig().Book.Version` test read it as unset and stamped the
// enclosing repository's newest git tag on the cover instead.
func TestBookJSONVersionSurvivesGitTag(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := writeGitBookProject(t, `{"title": "Frozen Book", "version": "1.0.0"}`, "# Frozen Book\n\nEnglish body text.\n")

	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=t@example.com", "-c", "user.name=T", "commit", "--allow-empty", "-qm", "init"},
		{"tag", "v9.9.9"},
	} {
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v (%s)", args, err, out)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if cfg.Book.Version != "1.0.0" {
		t.Errorf("book version = %q, want the declared 1.0.0 (a git tag overwrote it)", cfg.Book.Version)
	}
}

// TestBookJSONVersionStillFallsBackToGitTag keeps the useful half of the
// behavior: a project that declares no version does get one inferred.
func TestBookJSONVersionStillFallsBackToGitTag(t *testing.T) {
	dir := writeGitBookProject(t, `{"title": "Loose Book"}`, "# Loose Book\n\n**v2.5.0**\n\nEnglish body text.\n")

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if cfg.Book.Version != "2.5.0" {
		t.Errorf("book version = %q, want 2.5.0 inferred from README", cfg.Book.Version)
	}
}

// TestBookJSONLanguageSurvivesContentSniff pins the language bug: the README
// sniff never returns "", so the `meta.Language != ""` guard was always true
// and book.json's declared language was discarded on every project that also
// had a SUMMARY.md — which is the standard GitBook layout.
func TestBookJSONLanguageSurvivesContentSniff(t *testing.T) {
	dir := writeGitBookProject(t,
		`{"title": "Handbook", "language": "ja"}`,
		"# ハンドブック\n\nこれは日本語の文書です。日本語のテキストがここにあります。\n")

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if cfg.Book.Language != "ja-JP" {
		t.Errorf("language = %q, want ja-JP as declared in book.json", cfg.Book.Language)
	}
}

// TestContentSniffStillSetsLanguageWhenUndeclared keeps zero-declaration
// projects working: nothing said what language this is, so the README decides.
func TestContentSniffStillSetsLanguageWhenUndeclared(t *testing.T) {
	dir := writeGitBookProject(t, `{"title": "手册"}`, "# 手册\n\n这是一本中文手册，内容全部使用中文撰写。\n")

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if cfg.Book.Language != "zh-CN" {
		t.Errorf("language = %q, want zh-CN sniffed from the README", cfg.Book.Language)
	}
}
