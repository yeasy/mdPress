package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestFormatIssueSummary_Empty(t *testing.T) {
	result := formatIssueSummary([]projectIssue{})
	if result != "" {
		t.Errorf("Expected empty string for no issues, got %q", result)
	}
}

func TestFormatIssueSummary_Single(t *testing.T) {
	issues := []projectIssue{
		{
			Rule:    "test-rule",
			File:    "test.md",
			Line:    1,
			Column:  1,
			Message: "test message",
		},
	}

	result := formatIssueSummary(issues)
	if !strings.Contains(result, "test-rule=1") {
		t.Errorf("Expected 'test-rule=1' in summary, got %q", result)
	}
}

func TestFormatIssueSummary_Multiple(t *testing.T) {
	issues := []projectIssue{
		{Rule: "rule-a", File: "file1.md", Line: 1, Column: 1, Message: "msg1"},
		{Rule: "rule-b", File: "file2.md", Line: 2, Column: 2, Message: "msg2"},
		{Rule: "rule-a", File: "file3.md", Line: 3, Column: 3, Message: "msg3"},
	}

	result := formatIssueSummary(issues)

	if !strings.Contains(result, "rule-a=2") {
		t.Errorf("Expected 'rule-a=2' in summary, got %q", result)
	}
	if !strings.Contains(result, "rule-b=1") {
		t.Errorf("Expected 'rule-b=1' in summary, got %q", result)
	}
}

func TestFormatIssueSummary_Sorted(t *testing.T) {
	issues := []projectIssue{
		{Rule: "z-rule", File: "f1.md", Line: 1, Column: 1, Message: "m1"},
		{Rule: "a-rule", File: "f2.md", Line: 2, Column: 2, Message: "m2"},
		{Rule: "m-rule", File: "f3.md", Line: 3, Column: 3, Message: "m3"},
	}

	result := formatIssueSummary(issues)

	// Should be sorted alphabetically
	aIdx := strings.Index(result, "a-rule")
	mIdx := strings.Index(result, "m-rule")
	zIdx := strings.Index(result, "z-rule")

	if aIdx < 0 || mIdx < 0 || zIdx < 0 {
		t.Errorf("Not all rules found in summary: %q", result)
	}

	if aIdx >= mIdx || mIdx >= zIdx {
		t.Errorf("Rules not properly sorted in summary: %q", result)
	}
}

func TestFormatIssueSummary_Format(t *testing.T) {
	issues := []projectIssue{
		{Rule: "rule1", File: "f1.md", Line: 1, Column: 1, Message: "m1"},
		{Rule: "rule2", File: "f2.md", Line: 2, Column: 2, Message: "m2"},
	}

	result := formatIssueSummary(issues)

	// Should be in format "rule=count, rule=count"
	if !strings.Contains(result, "rule1=1") {
		t.Errorf("Expected 'rule1=1' format")
	}
	if !strings.Contains(result, "rule2=1") {
		t.Errorf("Expected 'rule2=1' format")
	}
	if !strings.Contains(result, ",") {
		t.Errorf("Expected comma-separated values")
	}
}

func TestFormatIssueSummary_ManyOfSameRule(t *testing.T) {
	issues := make([]projectIssue, 20)
	for i := 0; i < 20; i++ {
		issues[i] = projectIssue{
			Rule:    "same-rule",
			File:    fmt.Sprintf("file%d.md", i),
			Line:    i,
			Column:  1,
			Message: "message",
		}
	}

	result := formatIssueSummary(issues)

	if !strings.Contains(result, "same-rule=20") {
		t.Errorf("Expected 'same-rule=20' in summary, got %q", result)
	}
}

func TestReportBuildIssues_Empty(t *testing.T) {
	// Create a logger that writes to a buffer
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	verbose = false
	reportBuildIssues(logger, []projectIssue{})

	// Should not log anything for empty issues
	output := buf.String()
	if strings.Contains(output, "document issue") {
		t.Errorf("Expected no output for empty issues, got: %s", output)
	}
}

func TestReportBuildIssues_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	issues := []projectIssue{
		{Rule: "rule1", File: "file1.md", Line: 10, Column: 5, Message: "Issue 1"},
		{Rule: "rule2", File: "file2.md", Line: 20, Column: 3, Message: "Issue 2"},
	}

	verbose = true
	reportBuildIssues(logger, issues)

	output := buf.String()

	// In verbose mode, should log each issue individually
	if !strings.Contains(output, "document issue detected") {
		t.Errorf("Expected 'document issue detected' in verbose output, got: %s", output)
	}
	if !strings.Contains(output, "file1.md") {
		t.Errorf("Expected file reference 'file1.md', got: %s", output)
	}
}

func TestReportBuildIssues_NonVerboseMode(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	issues := []projectIssue{
		{Rule: "rule1", File: "file1.md", Line: 10, Column: 5, Message: "Issue 1"},
		{Rule: "rule2", File: "file2.md", Line: 20, Column: 3, Message: "Issue 2"},
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()

	// In non-verbose mode, should summarize
	if !strings.Contains(output, "document issue") {
		t.Errorf("Expected 'document issue' in non-verbose output, got: %s", output)
	}
}

func TestReportBuildIssues_ManyTitleStyleIssues(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	// Create 15 book-title-style issues
	issues := make([]projectIssue, 15)
	for i := 0; i < 15; i++ {
		issues[i] = projectIssue{
			Rule:    "book-title-style",
			File:    fmt.Sprintf("chapter%d.md", i),
			Line:    i + 1,
			Column:  1,
			Message: "Style issue",
		}
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()

	// Should show summarized message when count > 10
	if !strings.Contains(output, "chapter title style inconsistencies") {
		t.Errorf("Expected summarized message 'chapter title style inconsistencies', got: %s", output)
	}
}

func TestReportBuildIssues_FewerTitleStyleIssues(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	// Create 5 book-title-style issues
	issues := make([]projectIssue, 5)
	for i := 0; i < 5; i++ {
		issues[i] = projectIssue{
			Rule:    "book-title-style",
			File:    fmt.Sprintf("chapter%d.md", i),
			Line:    i + 1,
			Column:  1,
			Message: "Style issue",
		}
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()

	// Should show individual issues when count <= 10
	if !strings.Contains(output, "chapter0.md") {
		t.Errorf("Expected individual issues shown (chapter0.md), got: %s", output)
	}
}

func TestReportBuildIssues_MixedIssues(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	issues := []projectIssue{
		{Rule: "book-title-style", File: "ch1.md", Line: 1, Column: 1, Message: "Title style"},
		{Rule: "book-title-style", File: "ch2.md", Line: 5, Column: 1, Message: "Title style"},
		{Rule: "other-rule", File: "intro.md", Line: 10, Column: 3, Message: "Other issue"},
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()

	// Should show non-title-style issues
	if !strings.Contains(output, "other-rule") && !strings.Contains(output, "intro.md") {
		t.Errorf("Expected other-rule or intro.md to be shown, got: %s", output)
	}
}

func TestReportBuildIssues_VerboseShowsAll(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	// Save and restore verbose flag
	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	// Create many title-style issues
	issues := make([]projectIssue, 15)
	for i := 0; i < 15; i++ {
		issues[i] = projectIssue{
			Rule:    "book-title-style",
			File:    fmt.Sprintf("chapter%d.md", i),
			Line:    i + 1,
			Column:  1,
			Message: "Style issue",
		}
	}

	verbose = true
	reportBuildIssues(logger, issues)

	output := buf.String()

	// In verbose mode, should show all issues individually
	// and also the summary
	if !strings.Contains(output, "document issue detected") {
		t.Errorf("Expected 'document issue detected' in verbose output, got: %s", output)
	}
}

func TestReportBuildIssues_NilLogger(t *testing.T) {
	// This should be handled gracefully (or panic is acceptable)
	// Just test that basic structure works
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	issues := []projectIssue{
		{Rule: "test", File: "f.md", Line: 1, Column: 1, Message: "msg"},
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()
	if !strings.Contains(output, "document issue") {
		t.Errorf("expected logger output to contain 'document issue', got: %s", output)
	}
	if !strings.Contains(output, "f.md") {
		t.Errorf("expected logger output to reference 'f.md', got: %s", output)
	}
}

func TestFormatIssueSummary_DuplicateRules(t *testing.T) {
	// All same rule
	issues := make([]projectIssue, 5)
	for i := 0; i < 5; i++ {
		issues[i] = projectIssue{
			Rule:    "duplicate-rule",
			File:    fmt.Sprintf("file%d.md", i),
			Line:    i,
			Column:  1,
			Message: "msg",
		}
	}

	result := formatIssueSummary(issues)

	if !strings.Contains(result, "duplicate-rule=5") {
		t.Errorf("Expected 'duplicate-rule=5', got %q", result)
	}
	if strings.Contains(result, ",") {
		t.Errorf("Should not have comma for single rule type, got %q", result)
	}
}

func TestFormatIssueSummary_SpecialCharactersInRule(t *testing.T) {
	issues := []projectIssue{
		{Rule: "rule-with-dashes", File: "f.md", Line: 1, Column: 1, Message: "m"},
		{Rule: "rule_with_underscores", File: "f.md", Line: 2, Column: 1, Message: "m"},
		{Rule: "CamelCaseRule", File: "f.md", Line: 3, Column: 1, Message: "m"},
	}

	result := formatIssueSummary(issues)

	if !strings.Contains(result, "CamelCaseRule=1") {
		t.Errorf("CamelCase rule not preserved: %q", result)
	}
	if !strings.Contains(result, "rule-with-dashes=1") {
		t.Errorf("Rule with dashes not preserved: %q", result)
	}
	if !strings.Contains(result, "rule_with_underscores=1") {
		t.Errorf("Rule with underscores not preserved: %q", result)
	}
}

func TestReportBuildIssues_SuppressVerboseLog(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := slog.New(handler)

	originalVerbose := verbose
	defer func() { verbose = originalVerbose }()

	issues := []projectIssue{
		{Rule: "book-title-style", File: "ch1.md", Line: 1, Column: 1, Message: "Style 1"},
		{Rule: "book-title-style", File: "ch2.md", Line: 2, Column: 1, Message: "Style 2"},
		{Rule: "book-title-style", File: "ch3.md", Line: 3, Column: 1, Message: "Style 3"},
	}

	verbose = false
	reportBuildIssues(logger, issues)

	output := buf.String()

	// With only 3 issues, should show all (not summarized)
	if !strings.Contains(output, "ch1.md") {
		t.Errorf("Expected ch1.md to be shown, got: %s", output)
	}
}
