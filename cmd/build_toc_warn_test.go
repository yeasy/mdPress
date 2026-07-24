package cmd

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// TestWarnTOCSettingsIgnored covers the silence around output.toc and
// output.toc_max_depth: only the printed PDF contents is built from them, so
// an author who set "toc: false" to drop the contents panel from their site
// rebuilt, got a byte-identical site, and was told nothing.
func TestWarnTOCSettingsIgnored(t *testing.T) {
	tests := []struct {
		name         string
		toc          bool
		maxDepth     int
		formats      []string
		wantWarn     bool
		wantSettings string
	}{
		{name: "defaults never warn", toc: true, maxDepth: 2, formats: []string{"html", "site"}},
		{name: "pdf honors both", toc: false, maxDepth: 1, formats: []string{"pdf"}},
		{
			name: "toc false on a site build", toc: false, maxDepth: 2,
			formats: []string{"site"}, wantWarn: true, wantSettings: "output.toc",
		},
		{
			name: "custom depth on a standalone html build", toc: true, maxDepth: 1,
			formats: []string{"html"}, wantWarn: true, wantSettings: "output.toc_max_depth",
		},
		{
			name: "pdf alongside a format that ignores them", toc: false, maxDepth: 1,
			formats: []string{"pdf", "site"}, wantWarn: true, wantSettings: "output.toc, output.toc_max_depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.BookConfig{}
			cfg.Output.TOC = tt.toc
			cfg.Output.TOCMaxDepth = tt.maxDepth

			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
			warnTOCSettingsIgnored(cfg, tt.formats, logger)

			warned := strings.Contains(logs.String(), "apply to the PDF only")
			if warned != tt.wantWarn {
				t.Fatalf("warned = %v, want %v (log: %q)", warned, tt.wantWarn, logs.String())
			}
			if tt.wantWarn && !strings.Contains(logs.String(), tt.wantSettings) {
				t.Errorf("warning should name %q, got %q", tt.wantSettings, logs.String())
			}
			if tt.wantWarn && strings.Contains(logs.String(), "pdf") {
				t.Errorf("the PDF honors both settings and must not be listed: %q", logs.String())
			}
		})
	}
}
