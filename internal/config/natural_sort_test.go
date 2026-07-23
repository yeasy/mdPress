package config

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestNaturalCompareOrdersNumbersByValue(t *testing.T) {
	got := []string{"11-scale.md", "2-install.md", "10-deploy.md", "1-intro.md", "3-use.md"}
	slices.SortFunc(got, NaturalCompare)
	want := []string{"1-intro.md", "2-install.md", "3-use.md", "10-deploy.md", "11-scale.md"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("natural sort = %v, want %v", got, want)
		}
	}
}

func TestNaturalCompareEdgeCases(t *testing.T) {
	cases := []struct {
		a, b string
		less bool
	}{
		{"a", "b", true},
		{"chapter2", "chapter10", true},
		{"chapter10", "chapter2", false},
		{"02-a", "2-a", true}, // equal value; the shorter form sorts first
		{"a/1.md", "a/2.md", true},
		{"a2b3", "a2b10", true}, // multiple digit runs
		{"", "a", true},
		{"file", "file2", true},
	}
	for _, c := range cases {
		if got := NaturalLess(c.a, c.b); got != c.less {
			t.Errorf("NaturalLess(%q, %q) = %v, want %v", c.a, c.b, got, c.less)
		}
	}
	if NaturalCompare("same", "same") != 0 {
		t.Error("identical strings should compare equal")
	}
}

// Zero-config discovery reads chapters in the order their author numbered
// them; lexical order silently put chapter 10 ahead of chapter 2.
func TestAutoDiscoverOrdersNumberedChaptersNaturally(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"1", "2", "3", "10", "11"} {
		if err := os.WriteFile(filepath.Join(dir, n+"-chapter.md"), []byte("# Chapter "+n+"\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	want := []string{"1-chapter.md", "2-chapter.md", "3-chapter.md", "10-chapter.md", "11-chapter.md"}
	for i, ch := range cfg.Chapters {
		if ch.File != want[i] {
			t.Fatalf("chapter %d = %q, want %q (full order: %+v)", i, ch.File, want[i], cfg.Chapters)
		}
	}
}

// A dangling symlink is not readable content; listing it as a chapter made
// every subsequent build fail on a file that never existed.
func TestAutoDiscoverSkipsDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone.md"), filepath.Join(dir, "dangling.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	cfg, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	for _, ch := range cfg.Chapters {
		if ch.File == "dangling.md" {
			t.Error("a dangling symlink was discovered as a chapter")
		}
	}
}
