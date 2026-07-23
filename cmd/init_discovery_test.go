package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// initFixture builds a project that exercises everything `mdpress init` used
// to get wrong: a top-level README, chapters numbered past nine, a nested
// directory with its own README, a project doc, and a broken symlink.
func initFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "the-book")
	if err := os.MkdirAll(filepath.Join(dir, "guide"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("README.md", "# The Big Book\n\nWelcome.\n")
	for _, n := range []string{"1", "2", "3", "10", "11"} {
		write(n+"-chapter.md", "# Chapter "+n+"\n\nText.\n")
	}
	write("guide/README.md", "# Guide Index\n\nGuide.\n")
	write("guide/detail.md", "# Guide Detail\n\nDetail.\n")
	write("CHANGELOG.md", "# Changelog\n")
	if err := os.Symlink(filepath.Join(dir, "gone.md"), filepath.Join(dir, "dangling.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return dir
}

func chapterFiles(chapters []config.ChapterDef) []string {
	var out []string
	for _, ch := range config.FlattenChapters(chapters) {
		out = append(out, ch.File)
	}
	return out
}

func TestInitGeneratesUsableBookYAML(t *testing.T) {
	defer suppressOutput(t)()
	dir := initFixture(t)

	if err := executeInit(context.Background(), dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	cfg, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("the book.yaml init wrote does not load: %v", err)
	}

	// The top-level README is the book's first chapter, as it is under
	// zero-config discovery.
	if cfg.Chapters[0].File != "README.md" {
		t.Errorf("first chapter = %q, want README.md", cfg.Chapters[0].File)
	}

	// Numbered chapters read in the order their author numbered them.
	want := []string{
		"README.md", "1-chapter.md", "2-chapter.md", "3-chapter.md",
		"10-chapter.md", "11-chapter.md", "guide/README.md", "guide/detail.md",
	}
	if got := chapterFiles(cfg.Chapters); !equalStrings(got, want) {
		t.Errorf("chapter order = %v, want %v", got, want)
	}

	// The directory structure survives as a section tree.
	var guideSections []config.ChapterDef
	foundGuide := false
	for _, ch := range cfg.Chapters {
		if ch.File == "guide/README.md" {
			guideSections, foundGuide = ch.Sections, true
		}
	}
	if !foundGuide {
		t.Fatal("guide/README.md should be a top-level chapter")
	}
	if len(guideSections) != 1 || guideSections[0].File != "guide/detail.md" {
		t.Errorf("guide sections = %+v, want guide/detail.md nested under it", guideSections)
	}

	// A broken symlink is not a chapter; writing it made the very next
	// command init suggests fail.
	for _, f := range chapterFiles(cfg.Chapters) {
		if f == "dangling.md" {
			t.Error("book.yaml lists a dangling symlink as a chapter")
		}
	}

	// Build artifacts should not be committed.
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); err != nil {
		t.Errorf("init did not create .gitignore: %v", err)
	}
}

// init and a zero-config build of the same directory must agree on which files
// are chapters and in what order.
func TestInitChapterOrderMatchesZeroConfig(t *testing.T) {
	defer suppressOutput(t)()
	dir := initFixture(t)

	discovered, err := config.Discover(context.Background(), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if err := executeInit(context.Background(), dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	initialized, err := config.Load(filepath.Join(dir, "book.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got, want := chapterFiles(initialized.Chapters), chapterFiles(discovered.Chapters); !equalStrings(got, want) {
		t.Errorf("init chapters %v differ from zero-config chapters %v", got, want)
	}
}

// An existing .gitignore is never clobbered.
func TestInitKeepsExistingGitignore(t *testing.T) {
	defer suppressOutput(t)()
	dir := initFixture(t)
	const mine = "my-own-rules\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(mine), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if err := executeInit(context.Background(), dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore")) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(data) != mine {
		t.Errorf(".gitignore was overwritten: %q", string(data))
	}
}

// A directory whose files are all numbered must nest under the directory's own
// first file when it has no README.
func TestBuildChapterTreePromotesFirstFileWithoutReadme(t *testing.T) {
	nodes := buildChapterTree([]discoveredFile{
		{RelPath: "part1/2-b.md", Title: "B", Depth: 1},
		{RelPath: "part1/10-c.md", Title: "C", Depth: 1},
		{RelPath: "part1/1-a.md", Title: "A", Depth: 1},
	})
	if len(nodes) != 1 {
		t.Fatalf("expected one section, got %+v", nodes)
	}
	if nodes[0].File != "part1/1-a.md" {
		t.Errorf("section entry = %q, want part1/1-a.md", nodes[0].File)
	}
	want := []string{"part1/2-b.md", "part1/10-c.md"}
	var got []string
	for _, s := range nodes[0].Sections {
		got = append(got, s.File)
	}
	if !equalStrings(got, want) {
		t.Errorf("nested sections = %v, want %v", got, want)
	}
}

func TestInitReportsSkippedProjectDocs(t *testing.T) {
	dir := initFixture(t)
	skipped := skippedTopLevelDocs(dir)
	if !equalStrings(skipped, []string{"CHANGELOG.md"}) {
		t.Errorf("skippedTopLevelDocs() = %v, want [CHANGELOG.md]", skipped)
	}
	if strings.Contains(strings.Join(skipped, ","), "README") {
		t.Error("README.md is book content and must not be reported as skipped")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
