package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteValidationReportMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "validate-report.md")
	report := validationReport{
		Status:      "failed",
		TotalChecks: 2,
		Passed:      1,
		Failed:      1,
		Results: []validateResult{
			{OK: true, Message: "Config syntax is valid"},
			{OK: false, Message: "Markdown link target is outside the build graph"},
		},
	}

	if err := writeValidationReport(path, report); err != nil {
		t.Fatalf("writeValidationReport markdown failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read markdown report: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# mdpress Validation Report") {
		t.Fatalf("markdown report missing title: %s", content)
	}
	if !strings.Contains(content, "- [FAIL] Markdown link target is outside the build graph") {
		t.Fatalf("markdown report missing failure entry: %s", content)
	}
}

func TestWriteValidationReportJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "validate-report.json")
	report := validationReport{
		Status:      "passed",
		TotalChecks: 1,
		Passed:      1,
		Failed:      0,
		Results: []validateResult{
			{OK: true, Message: "Config syntax is valid"},
		},
	}

	if err := writeValidationReport(path, report); err != nil {
		t.Fatalf("writeValidationReport json failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read json report: %v", err)
	}

	var decoded validationReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode json report: %v", err)
	}
	if decoded.Status != "passed" || decoded.TotalChecks != 1 || len(decoded.Results) != 1 {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}
}

func TestWriteDoctorReportMarkdown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor-report.md")
	report := doctorReport{
		Platform:          "darwin/arm64",
		GoVersion:         "go1.24.2",
		ChromiumAvailable: true,
		SummaryFound:      true,
		ProjectLoadable:   true,
		ProjectTitle:      "Sample",
		Warnings:          []string{"LANGS.md not found"},
		UnresolvedMarkdown: []unresolvedMarkdownLink{
			{Source: "appendix/README.md", Target: "example_guidelines.md"},
		},
	}

	if err := writeDoctorReport(path, report); err != nil {
		t.Fatalf("writeDoctorReport markdown failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read doctor markdown report: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# mdpress Doctor Report") {
		t.Fatalf("doctor markdown report missing title: %s", content)
	}
	if !strings.Contains(content, "- example_guidelines.md (from appendix/README.md)") {
		t.Fatalf("doctor markdown report missing unresolved link entry: %s", content)
	}
}
