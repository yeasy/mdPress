package cmd

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/i18n"
)

// TestMultilingualBuildInfersLanguageFromDirectory pins the dead-code bug in
// the language-directory guess: the "did the user set a language?" test was
// `langCfg.Book.Language == ""`, which no loaded config can satisfy because
// Discover unmarshals over DefaultConfig's "en-US". So a zh/ directory with a
// book.yaml that omits `language:` published Chinese pages as
// <html lang="en-US"> with an English search UI — worse than the same project
// with no book.yaml at all, which sniffs the content and gets it right.
func TestMultilingualBuildInfersLanguageFromDirectory(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write("LANGS.md", "# Languages\n\n* [English](en/)\n* [中文](zh/)\n* [Français](fr/)\n")
	write("en/README.md", "# Guide\n\nEnglish content.\n")
	write("en/book.yaml", "book:\n  title: Guide\nchapters:\n  - title: Intro\n    file: README.md\n")
	// No `language:` key: the zh/ directory name has to supply it.
	write("zh/README.md", "# 指南\n\n中文内容在这里。\n")
	write("zh/book.yaml", "book:\n  title: 指南\nchapters:\n  - title: 简介\n    file: README.md\n")
	// An explicit language must still win over the directory name.
	write("fr/README.md", "# Guide\n\nContenu.\n")
	write("fr/book.yaml", "book:\n  title: Guide FR\n  language: fr-CA\nchapters:\n  - title: Intro\n    file: README.md\n")

	langs := []i18n.LangDef{
		{Name: "English", Dir: "en"},
		{Name: "中文", Dir: "zh"},
		{Name: "Français", Dir: "fr"},
	}
	out := filepath.Join(t.TempDir(), "site")
	prevQuiet := quiet
	quiet = true
	t.Cleanup(func() { quiet = prevQuiet })
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	if err := executeMultilingualBuild(context.Background(), dir, langs, []string{"html"}, out, logger); err != nil {
		t.Fatalf("executeMultilingualBuild: %v", err)
	}

	for _, tc := range []struct{ langDir, want string }{
		{"zh", `lang="zh-CN"`},
		{"en", `lang="en-US"`},
		{"fr", `lang="fr-CA"`},
	} {
		matches, err := filepath.Glob(filepath.Join(out, tc.langDir, "*.html"))
		if err != nil || len(matches) == 0 {
			t.Fatalf("no HTML generated for %s (err=%v)", tc.langDir, err)
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), tc.want) {
			t.Errorf("%s output does not carry %s", tc.langDir, tc.want)
		}
	}
}
