package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

type projectIssue struct {
	Rule    string
	File    string
	Line    int
	Column  int
	Message string
}

func reportBuildIssues(logger *slog.Logger, issues []projectIssue) {
	if len(issues) == 0 {
		return
	}

	// Count occurrences per rule to decide whether to summarize.
	counts := make(map[string]int, len(issues))
	for _, issue := range issues {
		counts[issue.Rule]++
	}

	if verbose {
		// Verbose mode: show every issue individually.
		for _, issue := range issues {
			logger.Warn("document issue detected",
				slog.String("file", issue.File),
				slog.Int("line", issue.Line),
				slog.Int("column", issue.Column),
				slog.String("rule", issue.Rule),
				slog.String("detail", issue.Message))
		}
		logger.Warn("build completed with document issues",
			slog.Int("issues", len(issues)),
			slog.String("summary", formatIssueSummary(issues)))
		return
	}

	// Default mode: summarize book-title-style when count exceeds 10 to reduce noise.
	// Other issue types are still shown individually for easy location.
	titleStyleCount := counts["book-title-style"]
	if titleStyleCount > 10 {
		logger.Warn(fmt.Sprintf("%d chapter title style inconsistencies found, use --verbose for details", titleStyleCount))
	}

	// Show non-book-title-style issues (or book-title-style when count <= 10).
	for _, issue := range issues {
		if issue.Rule == "book-title-style" && titleStyleCount > 10 {
			continue
		}
		logger.Warn("document issue",
			slog.String("rule", issue.Rule),
			slog.String("file", issue.File),
			slog.String("detail", issue.Message))
	}
}

func formatIssueSummary(issues []projectIssue) string {
	if len(issues) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, issue := range issues {
		counts[issue.Rule]++
	}

	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}
