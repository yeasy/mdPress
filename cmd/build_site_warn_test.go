package cmd

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveSiteOutputDirWarnsOnlyWhenSiteRequested pins the fix for the
// warning every `mdpress build --format pdf -o book.pdf` used to print: it
// named a "book_site" directory that no site build was ever going to create,
// so users went looking for output that did not exist and CI jobs that treat
// stderr as a signal fired on every green build.
func TestResolveSiteOutputDirWarnsOnlyWhenSiteRequested(t *testing.T) {
	base := t.TempDir()
	output := filepath.Join(base, "dist", "manual.pdf")

	tests := []struct {
		name     string
		formats  []string
		wantWarn bool
	}{
		{"pdf only does not warn", []string{"pdf"}, false},
		{"html only does not warn", []string{"html"}, false},
		{"epub only does not warn", []string{"epub"}, false},
		{"pdf and html do not warn", []string{"pdf", "html"}, false},
		{"site alongside pdf warns", []string{"pdf", "site"}, true},
		{"site spelled oddly still warns", []string{"pdf", " Site "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))

			got := resolveSiteOutputDir(base, output, tt.formats, logger)
			if want := filepath.Join(base, "dist", "manual_site"); got != want {
				t.Errorf("resolveSiteOutputDir() = %q, want %q", got, want)
			}

			warned := strings.Contains(logs.String(), "site goes to a sibling directory")
			if warned != tt.wantWarn {
				t.Errorf("warning emitted = %v, want %v (log: %q)", warned, tt.wantWarn, logs.String())
			}
		})
	}
}
