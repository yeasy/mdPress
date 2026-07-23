package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// writeConfigShowProject creates a minimal project and returns its directory.
func writeConfigShowProject(t *testing.T, bookYAML string) string {
	t.Helper()
	dir := t.TempDir()
	if bookYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte(bookYAML), 0o600); err != nil {
			t.Fatalf("failed to write book.yaml: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "intro.md"), []byte("# Intro\n\nHello.\n"), 0o600); err != nil {
		t.Fatalf("failed to write intro.md: %v", err)
	}
	return dir
}

// runConfigShow renders the report for dir in the requested format.
func runConfigShow(t *testing.T, dir, format string) string {
	t.Helper()
	origCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = origCfgFile })
	cfgFile = defaultConfigName

	var buf bytes.Buffer
	if err := executeConfigShow(context.Background(), dir, format, &buf); err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	return buf.String()
}

const configShowBookYAML = `book:
  title: "My Demo Book"
  author: "A. Writer"
  language: zh-CN
style:
  theme: elegant
  code_theme: dracula
  page_size: A5
output:
  formats: [pdf, site]
  toc_max_depth: 3
chapters:
  - title: "Intro"
    file: intro.md
`

// TestConfigShowJSONReportsConfiguredValues covers the core complaint: none of
// the settings a user writes in book.yaml could be read back from the CLI, so
// "I set it and nothing happened" had no first debugging step.
func TestConfigShowJSONReportsConfiguredValues(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)

	var report map[string]any
	if err := json.Unmarshal([]byte(runConfigShow(t, dir, "json")), &report); err != nil {
		t.Fatalf("config show --format json produced invalid JSON: %v", err)
	}

	style, _ := report["style"].(map[string]any)
	if got := style["theme"]; got != "elegant" {
		t.Errorf("style.theme = %v, want elegant (the value in book.yaml)", got)
	}
	if got := style["page_size"]; got != "A5" {
		t.Errorf("style.page_size = %v, want A5", got)
	}
	if got := style["code_theme"]; got != "dracula" {
		t.Errorf("style.code_theme = %v, want dracula", got)
	}

	book, _ := report["book"].(map[string]any)
	if got := book["title"]; got != "My Demo Book" {
		t.Errorf("book.title = %v, want My Demo Book", got)
	}
	if got := book["language"]; got != "zh-CN" {
		t.Errorf("book.language = %v, want zh-CN", got)
	}

	output, _ := report["output"].(map[string]any)
	if got := output["toc_max_depth"]; got != float64(3) {
		t.Errorf("output.toc_max_depth = %v, want 3", got)
	}
	if got := output["filename"]; got != "" {
		t.Errorf("output.filename = %v, want the empty (derive from title) value", got)
	}
}

// TestConfigShowReportsResolvedValues covers the computed half: the theme a
// build would load and the files each format would be written to are not in
// book.yaml at all, and were previously visible nowhere.
func TestConfigShowReportsResolvedValues(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)

	var report map[string]any
	if err := json.Unmarshal([]byte(runConfigShow(t, dir, "json")), &report); err != nil {
		t.Fatalf("config show --format json produced invalid JSON: %v", err)
	}
	resolved, ok := report["resolved"].(map[string]any)
	if !ok {
		t.Fatalf("report has no resolved section: %v", report)
	}

	if got, want := resolved["config_file"], filepath.Join(dir, "book.yaml"); got != want {
		t.Errorf("resolved.config_file = %v, want %v", got, want)
	}
	if got := resolved["discovered"]; got != false {
		t.Errorf("resolved.discovered = %v, want false for a project with a book.yaml", got)
	}
	if got := resolved["chapter_count"]; got != float64(1) {
		t.Errorf("resolved.chapter_count = %v, want 1", got)
	}

	theme, _ := resolved["theme"].(map[string]any)
	if got := theme["name"]; got != "elegant" {
		t.Errorf("resolved.theme.name = %v, want elegant", got)
	}
	if got := theme["source"]; got != "built-in" {
		t.Errorf("resolved.theme.source = %v, want built-in", got)
	}
	// style.code_theme overrides the theme's own; this is the value the
	// Markdown parser is actually given.
	if got := theme["code_theme"]; got != "dracula" {
		t.Errorf("resolved.theme.code_theme = %v, want dracula", got)
	}

	artifacts, _ := resolved["artifacts"].(map[string]any)
	pdf, _ := artifacts["pdf"].(string)
	if !strings.HasSuffix(pdf, "My-Demo-Book.pdf") {
		t.Errorf("resolved.artifacts.pdf = %q, want the title-derived My-Demo-Book.pdf", pdf)
	}
	site, _ := artifacts["site"].(string)
	if !strings.HasSuffix(site, filepath.Join("_book", "index.html")) {
		t.Errorf("resolved.artifacts.site = %q, want _book/index.html", site)
	}
	if _, unexpected := artifacts["epub"]; unexpected {
		t.Errorf("resolved.artifacts lists epub, which output.formats does not request: %v", artifacts)
	}
}

// TestConfigShowYAMLMatchesJSON verifies the two encodings describe the same
// document, so a jq recipe and a human reading the YAML never disagree.
func TestConfigShowYAMLMatchesJSON(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)

	var fromYAML map[string]any
	if err := yaml.Unmarshal([]byte(runConfigShow(t, dir, "yaml")), &fromYAML); err != nil {
		t.Fatalf("config show produced invalid YAML: %v", err)
	}
	var fromJSON map[string]any
	if err := json.Unmarshal([]byte(runConfigShow(t, dir, "json")), &fromJSON); err != nil {
		t.Fatalf("config show --format json produced invalid JSON: %v", err)
	}

	for _, key := range []string{"book", "chapters", "style", "output", "markdown", "resolved"} {
		if _, ok := fromYAML[key]; !ok {
			t.Errorf("YAML output is missing the %q section", key)
		}
		if _, ok := fromJSON[key]; !ok {
			t.Errorf("JSON output is missing the %q section", key)
		}
	}
	if len(fromYAML) != len(fromJSON) {
		t.Errorf("YAML has %d top-level keys, JSON has %d; the encodings must match", len(fromYAML), len(fromJSON))
	}
}

// TestConfigShowReportsCustomThemeFile verifies the theme source names the
// file a project-local theme comes from, which is the difference between "my
// theme override works" and "mdpress silently used the built-in".
func TestConfigShowReportsCustomThemeFile(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)
	themeDir := filepath.Join(dir, "themes")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatalf("failed to create themes dir: %v", err)
	}
	themeFile := filepath.Join(themeDir, "elegant.yaml")
	custom := "name: elegant\npage_size: A4\nfont_size: 13\nline_height: 1.5\ncode_theme: monokai\ncolors:\n  text: \"#111111\"\n  background: \"#ffffff\"\n"
	if err := os.WriteFile(themeFile, []byte(custom), 0o600); err != nil {
		t.Fatalf("failed to write theme file: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal([]byte(runConfigShow(t, dir, "json")), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	resolved, _ := report["resolved"].(map[string]any)
	theme, _ := resolved["theme"].(map[string]any)
	if got := theme["source"]; got != themeFile {
		t.Errorf("resolved.theme.source = %v, want the project theme file %s", got, themeFile)
	}
}

// TestConfigShowZeroConfig verifies a project without book.yaml reports the
// configuration auto-discovery inferred, and says that it did so.
func TestConfigShowZeroConfig(t *testing.T) {
	dir := writeConfigShowProject(t, "")

	var report map[string]any
	if err := json.Unmarshal([]byte(runConfigShow(t, dir, "json")), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	resolved, _ := report["resolved"].(map[string]any)
	if got := resolved["discovered"]; got != true {
		t.Errorf("resolved.discovered = %v, want true when there is no book.yaml", got)
	}
	if got := resolved["config_file"]; got != "" {
		t.Errorf("resolved.config_file = %v, want empty in zero-config mode", got)
	}
	if got := resolved["chapter_count"]; got != float64(1) {
		t.Errorf("resolved.chapter_count = %v, want the one discovered chapter", got)
	}
}

// TestConfigShowRejectsUnknownFormat keeps a typo'd --format from silently
// falling back to some default encoding.
func TestConfigShowRejectsUnknownFormat(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)
	origCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = origCfgFile })
	cfgFile = defaultConfigName

	var buf bytes.Buffer
	err := executeConfigShow(context.Background(), dir, "xml", &buf)
	if err == nil {
		t.Fatal("config show --format xml should fail")
	}
	if !strings.Contains(err.Error(), "yaml") || !strings.Contains(err.Error(), "json") {
		t.Errorf("error should list the supported encodings, got %q", err)
	}
}

// TestConfigShowMissingExplicitConfig verifies a mistyped --config is an error
// rather than a report about a different project found by discovery.
func TestConfigShowMissingExplicitConfig(t *testing.T) {
	dir := writeConfigShowProject(t, configShowBookYAML)
	origCfgFile := cfgFile
	t.Cleanup(func() { cfgFile = origCfgFile })
	cfgFile = "does-not-exist.yaml"

	var buf bytes.Buffer
	if err := executeConfigShow(context.Background(), dir, "yaml", &buf); err == nil {
		t.Fatalf("config show with a missing --config should fail, got output: %s", buf.String())
	}
}
