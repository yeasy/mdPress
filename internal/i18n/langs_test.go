package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLangsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "LANGS.md")
	content := `# Languages

* [English](en/)
* [中文](zh/)
* [日本語](ja/)
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	langs, err := ParseLangsFile(path)
	if err != nil {
		t.Fatalf("ParseLangsFile failed: %v", err)
	}
	if len(langs) != 3 {
		t.Fatalf("expected 3 languages, got %d", len(langs))
	}
	if langs[0].Name != "English" || langs[0].Dir != "en" {
		t.Errorf("first lang: got %+v", langs[0])
	}
	if langs[1].Name != "中文" || langs[1].Dir != "zh" {
		t.Errorf("second lang: got %+v", langs[1])
	}
}

func TestParseLangsFileNoTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "LANGS.md")
	content := "* [English](en)\n* [中文](zh)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	langs, err := ParseLangsFile(path)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if langs[0].Dir != "en" {
		t.Errorf("dir should not have trailing slash: got %q", langs[0].Dir)
	}
}

// The manual's own example links each language to its landing page, so an
// entry that names a file must resolve to the directory holding it.
func TestParseLangsFileEntryPointsAtLandingPage(t *testing.T) {
	dir := t.TempDir()
	for _, lang := range []string{"en", "zh"} {
		langDir := filepath.Join(dir, lang)
		if err := os.MkdirAll(langDir, 0o755); err != nil {
			t.Fatalf("mkdir %s failed: %v", langDir, err)
		}
		if err := os.WriteFile(filepath.Join(langDir, "README.md"), []byte("# Intro\n"), 0o644); err != nil {
			t.Fatalf("write README.md failed: %v", err)
		}
	}
	path := filepath.Join(dir, "LANGS.md")
	content := "* [English](en/README.md)\n* [中文](zh/index.md)\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	langs, err := ParseLangsFile(path)
	if err != nil {
		t.Fatalf("ParseLangsFile failed: %v", err)
	}
	if len(langs) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(langs))
	}
	if langs[0].Dir != "en" {
		t.Errorf("first lang dir = %q, want %q", langs[0].Dir, "en")
	}
	// The linked file need not exist; only its directory has to.
	if langs[1].Dir != "zh" {
		t.Errorf("second lang dir = %q, want %q", langs[1].Dir, "zh")
	}
}

func TestParseLangsFileEntryPointsAtExistingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	path := filepath.Join(dir, "LANGS.md")
	if err := os.WriteFile(path, []byte("* [English](en)\n"), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	_, err := ParseLangsFile(path)
	if err == nil {
		t.Fatal("entry naming a regular file should be rejected")
	}
	if !strings.Contains(err.Error(), "must point at a language directory") {
		t.Errorf("error should explain the fix, got: %v", err)
	}
}

func TestParseLangsFileEntryWithoutLanguageDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "LANGS.md")
	if err := os.WriteFile(path, []byte("* [English](README.md)\n"), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	_, err := ParseLangsFile(path)
	if err == nil {
		t.Fatal("entry pointing at a root-level file should be rejected")
	}
	if !strings.Contains(err.Error(), "must point at a language directory") {
		t.Errorf("error should explain the fix, got: %v", err)
	}
}

func TestParseLangsFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "LANGS.md")
	if err := os.WriteFile(path, []byte("# Languages\n"), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}

	_, err := ParseLangsFile(path)
	if err == nil {
		t.Error("empty LANGS.md should return error")
	}
}

func TestParseLangsFileNonExistent(t *testing.T) {
	_, err := ParseLangsFile("/nonexistent/LANGS.md")
	if err == nil {
		t.Error("should fail for non-existent file")
	}
}

func TestHasLangsFile(t *testing.T) {
	dir := t.TempDir()

	if hasLangsFile(dir) {
		t.Error("should return false when no LANGS.md")
	}

	if err := os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte("test"), 0o644); err != nil {
		t.Fatalf("write LANGS.md failed: %v", err)
	}
	if !hasLangsFile(dir) {
		t.Error("should return true when LANGS.md exists")
	}
}
