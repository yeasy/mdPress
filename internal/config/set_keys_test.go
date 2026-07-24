package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsSetDistinguishesDefaultsFromUserValues is the guard for the whole
// class of bugs set_keys.go exists to kill: a field holding exactly the value
// DefaultConfig put there is not "configured", even though its value is
// identical to one the user could have typed.
func TestIsSetDistinguishesDefaultsFromUserValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.yaml")
	// version and language are written with exactly the DefaultConfig values.
	if err := os.WriteFile(path, []byte(`book:
  title: Pinned
  version: "1.0.0"
  language: en-US
style:
  margin:
    top: 7
chapters:
  - title: One
    file: one.md
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "one.md"), []byte("# One\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, key := range []string{"book.version", "book.language", "style.margin", "style.margin.top", "book.title"} {
		if !cfg.IsSet(key) {
			t.Errorf("IsSet(%q) = false, want true: the key is present in book.yaml", key)
		}
	}
	for _, key := range []string{"style.page_size", "style.margin.bottom", "book.author", "output.filename"} {
		if cfg.IsSet(key) {
			t.Errorf("IsSet(%q) = true, want false: the value came from DefaultConfig", key)
		}
	}
}

// TestIsSetOnUnloadedConfig makes sure the nil map is safe: configs built in
// memory (tests, zero-config discovery) have no recorded keys at all.
func TestIsSetOnUnloadedConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.IsSet("book.version") {
		t.Error("a config nobody loaded from disk reports book.version as configured")
	}
	cfg.markSet("book.version")
	if !cfg.IsSet("book.version") {
		t.Error("markSet did not record the key")
	}
}
