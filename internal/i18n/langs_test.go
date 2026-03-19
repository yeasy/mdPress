package i18n

import (
	"os"
	"path/filepath"
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
	os.WriteFile(path, []byte(content), 0644)

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
	os.WriteFile(path, []byte(content), 0644)

	langs, err := ParseLangsFile(path)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if langs[0].Dir != "en" {
		t.Errorf("dir should not have trailing slash: got %q", langs[0].Dir)
	}
}

func TestParseLangsFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "LANGS.md")
	os.WriteFile(path, []byte("# Languages\n"), 0644)

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

	if HasLangsFile(dir) {
		t.Error("should return false when no LANGS.md")
	}

	os.WriteFile(filepath.Join(dir, "LANGS.md"), []byte("test"), 0644)
	if !HasLangsFile(dir) {
		t.Error("should return true when LANGS.md exists")
	}
}
