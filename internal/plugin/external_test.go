package plugin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// writeScript creates a temporary executable script that writes the given
// body to stdout when run.  On Unix it creates a shell script; on Windows
// it creates a .bat file with translated commands.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		p := filepath.Join(dir, name+".bat")
		winBody := toWindowsBatch(body)
		if err := os.WriteFile(p, []byte("@echo off\r\n"+winBody+"\r\n"), 0755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

// toWindowsBatch translates common Unix shell idioms to Windows batch equivalents.
func toWindowsBatch(body string) string {
	// Replace Unix command separator ";" with Windows "&"
	body = strings.ReplaceAll(body, "; ", "& ")
	// Replace "exit 1" with "exit /b 1" (batch requires /b to exit script, not cmd.exe)
	body = strings.ReplaceAll(body, "exit 1", "exit /b 1")
	// Replace bare "true" (Unix noop) with "rem noop"
	if body == "true" {
		return "rem noop"
	}
	// Strip single quotes from echo commands (Windows echo doesn't use them)
	if strings.HasPrefix(body, "echo '") && strings.HasSuffix(body, "'") {
		body = "echo " + body[6:len(body)-1]
	}
	// Handle echo with single quotes followed by other commands (e.g. "echo '...' >&2& ...")
	body = strings.ReplaceAll(body, "echo '", "echo ")
	body = strings.ReplaceAll(body, "' >&2", " 1>&2")
	body = strings.ReplaceAll(body, "' >", " >")
	return body
}

// --- NewExternalPlugin tests ---

func TestNewExternalPlugin_NonExistent(t *testing.T) {
	_, err := NewExternalPlugin("ghost", "/no/such/binary", nil)
	if err == nil {
		t.Fatal("expected error for non-existent executable")
	}
}

func TestNewExternalPlugin_Directory(t *testing.T) {
	dir := t.TempDir()
	_, err := NewExternalPlugin("dir-plugin", dir, nil)
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestNewExternalPlugin_NilConfig(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "plug", "echo '{}'")
	ep, err := NewExternalPlugin("test", p, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.pluginConfig == nil {
		t.Error("expected pluginConfig to be initialized, got nil")
	}
}

func TestNewExternalPlugin_BasicProperties(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "plug", "echo '{}'")
	ep, err := NewExternalPlugin("my-plug", p, map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Name() != "my-plug" {
		t.Errorf("Name() = %q, want %q", ep.Name(), "my-plug")
	}
	// ep.Description() may be empty if --mdpress-info is not supported
	if err := ep.Init(nil); err != nil {
		t.Errorf("Init should be no-op, got error: %v", err)
	}
	if err := ep.Cleanup(); err != nil {
		t.Errorf("Cleanup should be no-op, got error: %v", err)
	}
}

// --- Execute tests ---

func TestExternalPlugin_Execute_EmptyOutput(t *testing.T) {
	dir := t.TempDir()
	// Plugin that produces no output → keep original content.
	p := writeScript(t, dir, "noop", "true")
	ep, err := NewExternalPlugin("noop", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Content:  "original",
		Metadata: map[string]interface{}{},
	}
	result, err := ep.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Content != "" {
		t.Errorf("expected empty Content for no-output plugin, got %q", result.Content)
	}
}

func TestExternalPlugin_Execute_ContentReplacement(t *testing.T) {
	dir := t.TempDir()
	resp, _ := json.Marshal(ExternalPluginResponse{Content: "modified"})
	p := writeScript(t, dir, "replacer", "echo '"+string(resp)+"'")
	ep, err := NewExternalPlugin("replacer", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Content:  "original",
		Metadata: map[string]interface{}{},
	}
	result, err := ep.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Content != "modified" {
		t.Errorf("Content = %q, want %q", result.Content, "modified")
	}
}

func TestExternalPlugin_Execute_StopPropagation(t *testing.T) {
	dir := t.TempDir()
	resp, _ := json.Marshal(ExternalPluginResponse{Stop: true})
	p := writeScript(t, dir, "stopper", "echo '"+string(resp)+"'")
	ep, err := NewExternalPlugin("stopper", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	result, err := ep.Execute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Stop {
		t.Error("expected Stop=true")
	}
}

func TestExternalPlugin_Execute_ErrorResponse(t *testing.T) {
	dir := t.TempDir()
	resp, _ := json.Marshal(ExternalPluginResponse{Error: "something broke"})
	p := writeScript(t, dir, "errplugin", "echo '"+string(resp)+"'")
	ep, err := NewExternalPlugin("errplugin", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	result, err := ep.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute should not return Go error for JSON error field, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// Error should be recorded in Metadata.
	if v, ok := ctx.Metadata["errplugin.error"]; !ok || v != "something broke" {
		t.Errorf("expected error in metadata, got %v", ctx.Metadata)
	}
}

func TestExternalPlugin_Execute_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "badjson", "echo 'NOT JSON'")
	ep, err := NewExternalPlugin("badjson", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	_, err = ep.Execute(ctx)
	if err == nil {
		t.Fatal("expected error for malformed JSON output")
	}
}

func TestExternalPlugin_Execute_ProcessFailure(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "failing", "exit 1")
	ep, err := NewExternalPlugin("failing", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	_, err = ep.Execute(ctx)
	if err == nil {
		t.Fatal("expected error when plugin process exits non-zero")
	}
}

func TestExternalPlugin_Execute_Stderr(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "stderrplugin", "echo 'debug info' >&2; exit 1")
	ep, err := NewExternalPlugin("stderrplugin", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	_, err = ep.Execute(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should include stderr output.
	if got := err.Error(); got == "" {
		t.Error("error message is empty")
	}
}

func TestExternalPlugin_Execute_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script syntax required; skipping on Windows")
	}
	dir := t.TempDir()
	// Script responds quickly to --mdpress-info/--mdpress-hooks (so NewExternalPlugin
	// does not block on the 5s query timeout) but sleeps on normal execution.
	script := `case "$1" in --mdpress-*) echo '{}';; *) sleep 30;; esac`
	p := writeScript(t, dir, "slow", script)
	ep, err := NewExternalPlugin("slow", p, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Override the default timeout to something very short for testing.
	ep.timeout = 100 * time.Millisecond

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: map[string]interface{}{},
	}
	_, err = ep.Execute(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExternalPlugin_Execute_NilMetadata(t *testing.T) {
	dir := t.TempDir()
	resp, _ := json.Marshal(ExternalPluginResponse{Error: "oops"})
	p := writeScript(t, dir, "nilmeta", "echo '"+string(resp)+"'")
	ep, err := NewExternalPlugin("nilmeta", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterParse,
		Metadata: nil, // nil metadata should be handled gracefully
	}
	_, err = ep.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if ctx.Metadata == nil {
		t.Error("expected Metadata to be initialized")
	}
}

// --- Helper function tests ---

func TestAllPhases(t *testing.T) {
	phases := allPhases()
	if len(phases) != 7 {
		t.Errorf("expected 7 phases, got %d", len(phases))
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 5, "this ..."},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// --- queryPluginMeta / queryPluginHooks tests ---

func TestQueryPluginMeta_ValidResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script syntax required; skipping on Windows")
	}
	dir := t.TempDir()
	meta, _ := json.Marshal(map[string]string{"version": "2.0.0", "description": "test plugin"})
	// The script checks for --mdpress-info flag
	script := `
if [ "$1" = "--mdpress-info" ]; then
    echo '` + string(meta) + `'
else
    echo '{}'
fi
`
	p := writeScript(t, dir, "metaplug", script)
	ver, desc := queryPluginMeta(p)
	if ver != "2.0.0" {
		t.Errorf("version = %q, want %q", ver, "2.0.0")
	}
	if desc != "test plugin" {
		t.Errorf("description = %q, want %q", desc, "test plugin")
	}
}

func TestQueryPluginMeta_Fallback(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "badmeta", "exit 1")
	ver, desc := queryPluginMeta(p)
	if ver != "0.1.0" {
		t.Errorf("expected fallback version %q, got %q", "0.1.0", ver)
	}
	if desc != "" {
		t.Errorf("expected empty description, got %q", desc)
	}
}

func TestQueryPluginHooks_ValidResponse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script syntax required; skipping on Windows")
	}
	dir := t.TempDir()
	hooks, _ := json.Marshal([]string{"after_parse", "after_build"})
	script := `
if [ "$1" = "--mdpress-hooks" ]; then
    echo '` + string(hooks) + `'
else
    echo '[]'
fi
`
	p := writeScript(t, dir, "hookplug", script)
	phases := queryPluginHooks(p)
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(phases))
	}
	if phases[0] != PhaseAfterParse {
		t.Errorf("phases[0] = %q, want %q", phases[0], PhaseAfterParse)
	}
	if phases[1] != PhaseAfterBuild {
		t.Errorf("phases[1] = %q, want %q", phases[1], PhaseAfterBuild)
	}
}

func TestQueryPluginHooks_Fallback(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "badhooks", "exit 1")
	phases := queryPluginHooks(p)
	if len(phases) != 7 {
		t.Errorf("expected 7 fallback phases, got %d", len(phases))
	}
}
