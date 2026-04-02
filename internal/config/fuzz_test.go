package config

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzParseSummary fuzzes the SUMMARY.md parser with arbitrary input.
func FuzzParseSummary(f *testing.F) {
	seeds := []string{
		"# Summary\n\n* [Preface](preface.md)\n* [Chapter 1](ch01.md)\n  * [Section 1.1](ch01-1.md)",
		"",
		"# 目录\n\n* [前言](README.md)",
		"* [Link](file.md)\n\t* [Nested](nested.md)",
		"---\n* [After rule](file.md)",
		"* [No file]()",
		"random text without links",
		"* [Mixed](file.md)\n  * [Tabs\t](tab.md)",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Write to a temp file then parse.
		tmpDir := t.TempDir()
		summaryPath := filepath.Join(tmpDir, "SUMMARY.md")
		if err := os.WriteFile(summaryPath, []byte(data), 0o644); err != nil {
			t.Fatalf("write SUMMARY.md failed: %v", err)
		}

		// The parser must handle arbitrary input gracefully (no panics).
		// Errors are expected and intentionally discarded.
		result, err := ParseSummary(summaryPath)
		_ = err    // fuzz: errors are expected
		_ = result // fuzz: result is not validated
	})
}
