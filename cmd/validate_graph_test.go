package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validateReportFor runs the validate command against dir and returns the
// JSON report it wrote. Tests assert on the report rather than on stdout: the
// report is the machine-readable contract CI consumes.
func validateReportFor(t *testing.T, dir string) (validationReport, error) {
	t.Helper()
	defer suppressOutput(t)()

	reportPath := filepath.Join(t.TempDir(), "report.json")
	defer withValidateReportPath(t, reportPath)()

	runErr := executeValidate(context.Background(), dir)

	data, err := os.ReadFile(reportPath) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read validation report: %v", err)
	}
	var report validationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse validation report: %v", err)
	}
	return report, runErr
}

// findResult returns the first result whose message contains substr.
func findResult(report validationReport, substr string) (validateResult, bool) {
	for _, r := range report.Results {
		if strings.Contains(r.Message, substr) {
			return r, true
		}
	}
	return validateResult{}, false
}

// writeProject writes a set of relative path -> contents into dir.
func writeProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestExecuteValidate_ReportsDuplicateChapterEntry(t *testing.T) {
	tmpDir := t.TempDir()
	writeProject(t, tmpDir, map[string]string{
		"book.yaml": `book:
  title: "Dup Book"
  author: "Tester"
chapters:
  - title: "One"
    file: "ch1.md"
  - title: "Two"
    file: "ch2.md"
  - title: "Two Again"
    file: "./ch2.md"
`,
		"ch1.md": "# One\n",
		"ch2.md": "# Two\n",
	})

	report, err := validateReportFor(t, tmpDir)
	if err != nil {
		t.Fatalf("a duplicate entry is a warning, not an error: %v", err)
	}

	got, ok := findResult(report, "Duplicate chapter entry")
	if !ok {
		t.Fatalf("expected a duplicate chapter diagnostic, got results: %+v", report.Results)
	}
	if !got.Warning {
		t.Errorf("duplicate chapter entry should be reported as a warning, got %+v", got)
	}
	if !strings.Contains(got.Message, "ch2.md") {
		t.Errorf("diagnostic should name the duplicated file, got %q", got.Message)
	}
}

func TestExecuteValidate_NoDuplicateWarningForDistinctFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeProject(t, tmpDir, map[string]string{
		"book.yaml": `book:
  title: "Clean Book"
  author: "Tester"
chapters:
  - title: "One"
    file: "ch1.md"
  - title: "Two"
    file: "sub/ch1.md"
`,
		"ch1.md":     "# One\n",
		"sub/ch1.md": "# Two\n",
	})

	report, err := validateReportFor(t, tmpDir)
	if err != nil {
		t.Fatalf("validate should pass: %v", err)
	}
	if got, ok := findResult(report, "Duplicate chapter entry"); ok {
		t.Errorf("same basename in different directories is not a duplicate, got %q", got.Message)
	}
}

func withValidateStrict(t *testing.T, strict bool) func() {
	t.Helper()
	old := validateStrict
	validateStrict = strict
	return func() { validateStrict = old }
}

// A warning-only project exits 0 by default, which is what makes `validate`
// safe to adopt, and exits non-zero under --strict, which is what makes it
// usable as a CI gate.
func TestExecuteValidate_StrictFailsOnWarningsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	writeProject(t, tmpDir, map[string]string{
		"book.yaml": `book:
  title: "Warn Book"
  author: "Tester"
chapters:
  - title: "One"
    file: "ch1.md"
  - title: "One Again"
    file: "ch1.md"
`,
		"ch1.md": "# One\n",
	})

	report, err := validateReportFor(t, tmpDir)
	if err != nil {
		t.Fatalf("without --strict warnings must not fail the run: %v", err)
	}
	if report.Warnings == 0 {
		t.Fatalf("fixture should produce at least one warning, got %+v", report.Results)
	}
	if report.Status != "passed" {
		t.Errorf("status without --strict should be passed, got %q", report.Status)
	}

	defer withValidateStrict(t, true)()
	strictReport, strictErr := validateReportFor(t, tmpDir)
	if strictErr == nil {
		t.Fatal("--strict should fail a run that produced warnings")
	}
	if strictReport.Status != "failed" {
		t.Errorf("status under --strict should be failed, got %q", strictReport.Status)
	}
	if !strictReport.Strict {
		t.Error("report should record that it ran under --strict")
	}
}

func TestExecuteValidate_StrictPassesWhenClean(t *testing.T) {
	defer withValidateStrict(t, true)()
	tmpDir := t.TempDir()
	writeProject(t, tmpDir, map[string]string{
		"book.yaml": `book:
  title: "Clean Book"
  author: "Tester"
chapters:
  - title: "One"
    file: "ch1.md"
`,
		"ch1.md": "# One\n",
	})

	if _, err := validateReportFor(t, tmpDir); err != nil {
		t.Errorf("--strict should not fail a warning-free project: %v", err)
	}
}

func TestValidateCmd_HasStrictFlag(t *testing.T) {
	if validateCmd.Flags().Lookup("strict") == nil {
		t.Fatal("validate should expose a --strict flag, like doctor")
	}
}
