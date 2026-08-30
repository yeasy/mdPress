package cmd

import (
	"context"

	"github.com/yeasy/mdpress/internal/config"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildServeOutputHonorsLangsFile pins that the preview builds the same
// multi-language tree the build command ships. serve used to ignore LANGS.md
// and auto-discover the root as one ordinary book: both languages mixed into
// a single flat sidebar, the folder name as the title, and no language
// switcher — the preview and the shipped site were different programs.
func TestBuildServeOutputHonorsLangsFile(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("LANGS.md", "# Languages\n\n- [English](en/)\n- [中文](zh/)\n")
	write("en/book.yaml", "book:\n  title: Handbook\n  language: en\nchapters:\n  - file: README.md\n")
	write("en/README.md", "# Handbook\n\nEnglish home.\n")
	write("zh/book.yaml", "book:\n  title: 手册\n  language: zh\nchapters:\n  - file: README.md\n")
	write("zh/README.md", "# 手册\n\n中文首页。\n")

	cfg, err := config.Discover(context.Background(), root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if cfg.LangsFile == "" {
		t.Fatal("fixture did not register LANGS.md; the test would pass vacuously")
	}

	out := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	if err := buildServeOutput(context.Background(), cfg, out, logger); err != nil {
		t.Fatalf("buildServeOutput: %v", err)
	}

	landing, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("no landing page at the preview root: %v", err)
	}
	if !strings.Contains(string(landing), "English") || !strings.Contains(string(landing), "中文") {
		t.Errorf("landing page should offer both languages:\n%s", landing)
	}

	en, err := os.ReadFile(filepath.Join(out, "en", "index.html"))
	if err != nil {
		t.Fatalf("no en variant: %v", err)
	}
	if strings.Contains(string(en), "手册") {
		t.Error("the other language leaked into the en sidebar")
	}
	if !strings.Contains(string(en), "mdpress-lang-switcher") {
		t.Error("en page is missing the language switcher")
	}
	if _, err := os.Stat(filepath.Join(out, "zh", "index.html")); err != nil {
		t.Errorf("no zh variant: %v", err)
	}
}
