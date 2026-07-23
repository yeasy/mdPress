package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A book.yaml with several independent mistakes should report all of them in
// one go; fail-fast validation meant one edit-build cycle per mistake.
func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.SetBaseDir(dir)
	cfg.Book.Title = "Broken"
	cfg.Chapters = []ChapterDef{{Title: "A", File: "a.md"}}
	cfg.Style.PageSize = "A9"
	cfg.Style.Theme = "nonexistent"
	cfg.Style.LineHeight = 99
	cfg.Output.TOCMaxDepth = 12
	cfg.Output.PDFTimeout = 2

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	problems := ValidationErrors(err)
	if len(problems) != 6 {
		t.Fatalf("expected 6 separate problems, got %d: %v", len(problems), problems)
	}

	joined := err.Error()
	for _, want := range []string{"a.md", "A9", "nonexistent", "line_height", "toc_max_depth", "pdf_timeout"} {
		if !strings.Contains(joined, want) {
			t.Errorf("validation error does not mention %q: %s", want, joined)
		}
	}
}

// Every missing chapter file should be named, including ones nested inside
// sections, so the author can create them all in a single pass.
func TestValidateReportsEveryMissingChapter(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.SetBaseDir(dir)
	cfg.Book.Title = "Broken"
	cfg.Chapters = []ChapterDef{
		{Title: "A", File: "a.md"},
		{Title: "B", File: "b.md", Sections: []ChapterDef{{Title: "B1", File: "b1.md"}}},
		{Title: "C", File: "c.md"},
	}

	problems := ValidationErrors(cfg.Validate())
	if len(problems) != 3 {
		t.Fatalf("expected 3 missing-file problems, got %d: %v", len(problems), problems)
	}
	for _, want := range []string{"a.md", "b.md", "c.md"} {
		found := false
		for _, p := range problems {
			if strings.Contains(p.Error(), want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no problem mentions missing chapter %q: %v", want, problems)
		}
	}
}

// Nested sections are only reachable once their parent chapter exists, so a
// present parent must not hide a missing child.
func TestValidateReportsMissingNestedSection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "parent.md"), []byte("# Parent\n"), 0o600); err != nil {
		t.Fatalf("write parent.md: %v", err)
	}
	cfg := DefaultConfig()
	cfg.SetBaseDir(dir)
	cfg.Book.Title = "Nested"
	cfg.Chapters = []ChapterDef{
		{Title: "Parent", File: "parent.md", Sections: []ChapterDef{{Title: "Child", File: "child.md"}}},
	}

	problems := ValidationErrors(cfg.Validate())
	if len(problems) != 1 || !strings.Contains(problems[0].Error(), "child.md") {
		t.Fatalf("expected the missing nested section to be reported, got %v", problems)
	}
}

// ValidationErrors is the seam cmd/validate.go uses to render one line per
// problem, so it has to see through Load's "config validation failed: %w".
func TestValidationErrorsSeesThroughLoadWrapper(t *testing.T) {
	dir := t.TempDir()
	yaml := `book:
  title: "Broken"
chapters:
  - title: "A"
    file: "a.md"
  - title: "B"
    file: "b.md"
style:
  page_size: "A9"
`
	path := filepath.Join(dir, "book.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	if got := len(ValidationErrors(err)); got != 3 {
		t.Fatalf("expected 3 problems through the Load wrapper, got %d: %v", got, ValidationErrors(err))
	}
}

// A single problem must stay a single problem, keeping the wrapper's context.
func TestValidationErrorsKeepsSingleErrorWrapped(t *testing.T) {
	dir := t.TempDir()
	yaml := `book:
  title: "Broken"
chapters:
  - title: "A"
    file: "a.md"
`
	path := filepath.Join(dir, "book.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	problems := ValidationErrors(err)
	if len(problems) != 1 {
		t.Fatalf("expected 1 problem, got %d: %v", len(problems), problems)
	}
	if !strings.Contains(problems[0].Error(), "config validation failed") {
		t.Errorf("single problem lost its wrapper context: %v", problems[0])
	}
}

func TestValidationErrorsNil(t *testing.T) {
	if got := ValidationErrors(nil); got != nil {
		t.Fatalf("expected nil for a nil error, got %v", got)
	}
}
