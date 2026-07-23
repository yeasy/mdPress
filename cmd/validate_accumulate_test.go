package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runValidateForReport runs executeValidate against dir and returns the parsed
// JSON report, which is the machine-readable view of the printed check lines.
func runValidateForReport(t *testing.T, dir string) validationReport {
	t.Helper()
	reportPath := filepath.Join(t.TempDir(), "report.json")
	defer suppressOutput(t)()
	defer withValidateReportPath(t, reportPath)()

	if err := executeValidate(context.Background(), dir); err == nil {
		t.Fatal("expected validation to fail")
	}

	data, err := os.ReadFile(reportPath) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report validationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	return report
}

func failedMessages(report validationReport) []string {
	var msgs []string
	for _, r := range report.Results {
		if !r.OK {
			msgs = append(msgs, r.Message)
		}
	}
	return msgs
}

// A book.yaml with several independent problems used to surface only the first
// one, so fixing it took one `mdpress validate` run per problem.
func TestExecuteValidate_ReportsAllConfigProblemsInOneRun(t *testing.T) {
	dir := t.TempDir()
	yaml := `book:
  title: "Broken Book"
chapters:
  - title: "A"
    file: "a.md"
  - title: "B"
    file: "b.md"
style:
  theme: "nonexistent"
  page_size: "A9"
output:
  toc_max_depth: 12
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	msgs := failedMessages(runValidateForReport(t, dir))
	if len(msgs) != 5 {
		t.Fatalf("expected 5 failed checks, got %d: %v", len(msgs), msgs)
	}
	for _, want := range []string{"a.md", "b.md", "nonexistent", "A9", "toc_max_depth"} {
		found := false
		for _, m := range msgs {
			if strings.Contains(m, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no failed check mentions %q: %v", want, msgs)
		}
	}
}

// Four missing chapter files should be listed together, not one per run.
func TestExecuteValidate_ReportsAllMissingChaptersInOneRun(t *testing.T) {
	dir := t.TempDir()
	yaml := `book:
  title: "Missing Chapters"
chapters:
  - title: "A"
    file: "a.md"
  - title: "B"
    file: "b.md"
  - title: "C"
    file: "c.md"
  - title: "D"
    file: "d.md"
`
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write book.yaml: %v", err)
	}

	msgs := failedMessages(runValidateForReport(t, dir))
	if len(msgs) != 4 {
		t.Fatalf("expected 4 failed checks, got %d: %v", len(msgs), msgs)
	}
}
