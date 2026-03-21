package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSummary creates a minimal SUMMARY.md and the chapter files it references.
func writeSummary(t *testing.T, dir string) {
	t.Helper()
	summary := "# Summary\n\n* [Chapter One](ch1.md)\n* [Chapter Two](ch2.md)\n"
	if err := os.WriteFile(filepath.Join(dir, "SUMMARY.md"), []byte(summary), 0644); err != nil {
		t.Fatalf("write SUMMARY.md: %v", err)
	}
	for _, f := range []string{"ch1.md", "ch2.md"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("# "+f), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
}

// TestLoadBookJSON_BasicFields verifies that title, author, description,
// and language are mapped correctly into BookConfig.
func TestLoadBookJSON_BasicFields(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	content := `{
		"title": "My GitBook",
		"author": "Alice",
		"description": "A test book",
		"language": "en"
	}`
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}

	if cfg.Book.Title != "My GitBook" {
		t.Errorf("Title = %q, want %q", cfg.Book.Title, "My GitBook")
	}
	if cfg.Book.Author != "Alice" {
		t.Errorf("Author = %q, want %q", cfg.Book.Author, "Alice")
	}
	if cfg.Book.Description != "A test book" {
		t.Errorf("Description = %q, want %q", cfg.Book.Description, "A test book")
	}
	if cfg.Book.Language != "en-US" {
		t.Errorf("Language = %q, want %q", cfg.Book.Language, "en-US")
	}
}

// TestLoadBookJSON_AuthorArray ensures that a JSON array of authors is joined
// into a single comma-separated string.
func TestLoadBookJSON_AuthorArray(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	content := `{"title":"T","author":["Alice","Bob"]}`
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}
	if cfg.Book.Author != "Alice, Bob" {
		t.Errorf("Author = %q, want %q", cfg.Book.Author, "Alice, Bob")
	}
}

// TestLoadBookJSON_LanguageNormalization checks that short language codes are
// expanded to BCP 47 tags.
func TestLoadBookJSON_LanguageNormalization(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"en", "en-US"},
		{"en-us", "en-US"},
		{"en-gb", "en-GB"},
		{"zh", "zh-CN"},
		{"zh-hans", "zh-CN"},
		{"zh-cn", "zh-CN"},
		{"zh-tw", "zh-TW"},
		{"zh-hant", "zh-TW"},
		{"ja", "ja-JP"},
		{"ko", "ko-KR"},
		{"fr", "fr-FR"},
		{"de", "de-DE"},
		{"es", "es-ES"},
		{"pt", "pt-BR"},
		{"pt-br", "pt-BR"},
		{"ru", "ru-RU"},
		{"unknown-lang", "unknown-lang"}, // preserved as-is
	}

	for _, tc := range cases {
		got := normalizeLanguage(tc.input)
		if got != tc.want {
			t.Errorf("normalizeLanguage(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestLoadBookJSON_Plugins verifies plugin name mapping and disabled-plugin filtering.
func TestLoadBookJSON_Plugins(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	content := `{
		"title": "T",
		"plugins": ["search", "-sharing", "highlight"],
		"pluginsConfig": {
			"highlight": {"theme": "monokai"}
		}
	}`
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}

	// "-sharing" must be filtered out.
	if len(cfg.Plugins) != 2 {
		t.Fatalf("Plugins len = %d, want 2", len(cfg.Plugins))
	}
	if cfg.Plugins[0].Name != "search" {
		t.Errorf("Plugins[0].Name = %q, want %q", cfg.Plugins[0].Name, "search")
	}
	if cfg.Plugins[1].Name != "highlight" {
		t.Errorf("Plugins[1].Name = %q, want %q", cfg.Plugins[1].Name, "highlight")
	}
	// Plugin config should be wired through.
	if cfg.Plugins[1].Config["theme"] != "monokai" {
		t.Errorf("highlight config theme = %v, want %q", cfg.Plugins[1].Config["theme"], "monokai")
	}
}

// TestLoadBookJSON_LoadsChaptersFromSummary confirms that SUMMARY.md in the
// same directory is parsed and populates cfg.Chapters.
func TestLoadBookJSON_LoadsChaptersFromSummary(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	content := `{"title": "Book With Summary"}`
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}
	if len(cfg.Chapters) != 2 {
		t.Errorf("Chapters len = %d, want 2", len(cfg.Chapters))
	}
}

// TestLoadBookJSON_CustomSummaryPath checks that structure.summary is honored.
func TestLoadBookJSON_CustomSummaryPath(t *testing.T) {
	dir := t.TempDir()

	// Write a custom-named summary file.
	summary := "# Summary\n\n* [Intro](intro.md)\n"
	if err := os.WriteFile(filepath.Join(dir, "CONTENTS.md"), []byte(summary), 0644); err != nil {
		t.Fatalf("write CONTENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "intro.md"), []byte("# Intro"), 0644); err != nil {
		t.Fatalf("write intro.md: %v", err)
	}

	content := `{
		"title": "Custom Summary",
		"structure": {"summary": "CONTENTS.md"}
	}`
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}
	if len(cfg.Chapters) != 1 {
		t.Errorf("Chapters len = %d, want 1", len(cfg.Chapters))
	}
}

// TestLoadBookJSON_GlossaryDetected confirms GLOSSARY.md auto-detection.
func TestLoadBookJSON_GlossaryDetected(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "GLOSSARY.md"), []byte("# Terms\n"), 0644); err != nil {
		t.Fatalf("write GLOSSARY.md: %v", err)
	}

	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(`{"title":"T"}`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}
	if cfg.GlossaryFile == "" {
		t.Error("GlossaryFile should be set when GLOSSARY.md exists")
	}
}

// TestLoadBookJSON_MissingFile checks that a helpful error is returned when
// book.json does not exist.
func TestLoadBookJSON_MissingFile(t *testing.T) {
	_, err := LoadBookJSON("/nonexistent/path/book.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// TestLoadBookJSON_InvalidJSON ensures malformed JSON returns a parse error.
func TestLoadBookJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(`{invalid json`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}
	_, err := LoadBookJSON(jsonPath)
	if err == nil {
		t.Error("expected parse error for invalid JSON, got nil")
	}
}

// TestLoadBookJSON_DefaultsPreserved checks that unset fields retain mdPress defaults.
func TestLoadBookJSON_DefaultsPreserved(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	jsonPath := filepath.Join(dir, "book.json")
	if err := os.WriteFile(jsonPath, []byte(`{"title":"Minimal"}`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := LoadBookJSON(jsonPath)
	if err != nil {
		t.Fatalf("LoadBookJSON error: %v", err)
	}
	if cfg.Style.PageSize != "A4" {
		t.Errorf("PageSize = %q, want default %q", cfg.Style.PageSize, "A4")
	}
	if cfg.Output.Filename != "output.pdf" {
		t.Errorf("Filename = %q, want default %q", cfg.Output.Filename, "output.pdf")
	}
	if !cfg.Output.TOC {
		t.Error("TOC should default to true")
	}
}

// TestDiscoverPrefersBookYAMLOverBookJSON ensures book.yaml takes priority.
func TestDiscoverPrefersBookYAMLOverBookJSON(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	// book.yaml with a distinct title.
	yamlContent := `
book:
  title: "From YAML"
chapters:
  - title: "Ch1"
    file: "ch1.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	// book.json with a different title that must NOT win.
	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(`{"title":"From JSON"}`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if cfg.Book.Title != "From YAML" {
		t.Errorf("Title = %q, want %q (book.yaml must take priority)", cfg.Book.Title, "From YAML")
	}
}

// TestDiscoverUsesBookJSON verifies that book.json is used when book.yaml is absent.
func TestDiscoverUsesBookJSON(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(`{"title":"GitBook Title"}`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if cfg.Book.Title != "GitBook Title" {
		t.Errorf("Title = %q, want %q", cfg.Book.Title, "GitBook Title")
	}
}

// TestDiscoverBookJSONBeforeSummary ensures book.json has higher priority than SUMMARY.md alone.
func TestDiscoverBookJSONBeforeSummary(t *testing.T) {
	dir := t.TempDir()
	writeSummary(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "book.json"), []byte(`{"title":"JSON Wins"}`), 0644); err != nil {
		t.Fatalf("write book.json: %v", err)
	}

	cfg, err := Discover(dir)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if cfg.Book.Title != "JSON Wins" {
		t.Errorf("Title = %q, want %q", cfg.Book.Title, "JSON Wins")
	}
}
