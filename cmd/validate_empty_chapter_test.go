package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// TestValidateReportsEmptyChapter pins the gap that let a chapter truncated to
// 0 bytes ship: the build drops it (no page, no sidebar entry) with only a WARN
// in the log, while `validate --strict` reported "all checks passed" and exited
// 0, so the CI gate added for exactly this case never fired.
func TestValidateReportsEmptyChapter(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantIssue bool
	}{
		{"zero byte chapter", "", true},
		{"whitespace only chapter", "   \n\n\t\n", true},
		{"chapter with content is fine", "# One\n\nbody\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tmpDir, "one.md"), []byte(tt.body), 0o644); err != nil {
				t.Fatalf("write chapter: %v", err)
			}

			cfg := &config.BookConfig{}
			cfg.SetBaseDir(tmpDir)
			cfg.Book.Title = "Test"
			cfg.Chapters = []config.ChapterDef{{Title: "One", File: "one.md"}}

			issues, _, _, err := validateChapterContentAndSequence(cfg)
			if err != nil {
				t.Fatalf("validateChapterContentAndSequence: %v", err)
			}

			found := false
			for _, issue := range issues {
				if strings.Contains(issue, "Empty chapter one.md") {
					found = true
				}
			}
			if found != tt.wantIssue {
				t.Errorf("empty-chapter issue reported = %v, want %v (issues: %v)", found, tt.wantIssue, issues)
			}
		})
	}
}
