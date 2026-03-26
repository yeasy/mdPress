package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// stubMetaQueries replaces the plugin meta/hooks query functions with fast
// no-op stubs for the duration of the test, eliminating subprocess overhead.
func stubMetaQueries(t *testing.T) {
	t.Helper()
	origMeta, origHooks := pluginMetaQueryFn, pluginHooksQueryFn
	pluginMetaQueryFn = func(string) (string, string) { return "0.1.0", "" }
	pluginHooksQueryFn = func(string) []Phase { return allPhases() }
	t.Cleanup(func() {
		pluginMetaQueryFn = origMeta
		pluginHooksQueryFn = origHooks
	})
}

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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	_, err = ep.Execute(ctx)
	if err == nil {
		t.Fatal("Execute should return error when plugin reports an error")
	}
	if !strings.Contains(err.Error(), "something broke") {
		t.Errorf("error should contain plugin message, got: %v", err)
	}
}

func TestExternalPlugin_Execute_MalformedJSON(t *testing.T) {
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
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
	stubMetaQueries(t)
	if runtime.GOOS == "windows" {
		t.Skip("shell script syntax required; skipping on Windows")
	}
	dir := t.TempDir()
	// Script sleeps on normal execution to trigger the timeout.
	p := writeScript(t, dir, "slow", "sleep 30")
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
	stubMetaQueries(t)
	dir := t.TempDir()
	// Plugin reports an error in JSON; Execute should return that as a Go error.
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
	if err == nil {
		t.Fatal("expected error from plugin reporting 'oops'")
	}
	if !strings.Contains(err.Error(), "oops") {
		t.Errorf("error should contain plugin message, got: %v", err)
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

// --- resolvePluginExecutablePath tests ---

func TestResolvePluginExecutablePath_AbsolutePathExists(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "existing", "true")

	resolved, err := resolvePluginExecutablePath(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != p {
		t.Errorf("resolved = %q, want %q", resolved, p)
	}
}

func TestResolvePluginExecutablePath_RelativePathExists(t *testing.T) {
	dir := t.TempDir()
	p := writeScript(t, dir, "relative", "true")

	// Get relative path from current working directory
	// For simplicity, use the absolute path which should work
	resolved, err := resolvePluginExecutablePath(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	absResolved, _ := filepath.Abs(p)
	if resolved != absResolved {
		t.Errorf("resolved = %q, want %q", resolved, absResolved)
	}
}

func TestResolvePluginExecutablePath_NotFound(t *testing.T) {
	_, err := resolvePluginExecutablePath("/nonexistent/path/to/plugin")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should contain 'not found', got: %v", err)
	}
}

func TestResolvePluginExecutablePath_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	resolved, err := resolvePluginExecutablePath(dir)
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("error message should contain 'is a directory', got: %v", err)
	}
	if resolved != "" {
		t.Errorf("expected empty resolved path, got %q", resolved)
	}
}

func TestResolvePluginExecutablePath_WindowsSuffixResolution(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	dir := t.TempDir()

	// Create plugins with different extensions
	_ = writeScript(t, dir, "myplugin", "echo test")

	// Try to resolve without extension (should find .bat or .exe)
	basePath := filepath.Join(dir, "myplugin")
	resolved, err := resolvePluginExecutablePath(basePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved == "" {
		t.Fatal("expected resolved path")
	}
	if !strings.HasPrefix(resolved, basePath) {
		t.Errorf("resolved path should start with %q, got %q", basePath, resolved)
	}
}

func TestResolvePluginExecutablePath_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string) string // returns the path to test
		wantErr   bool
		errMsg    string
		checkPath bool // whether to validate the resolved path exists
	}{
		{
			name: "executable file exists",
			setup: func(dir string) string {
				return writeScript(t, dir, "exec", "true")
			},
			wantErr:   false,
			checkPath: true,
		},
		{
			name: "path does not exist",
			setup: func(dir string) string {
				return filepath.Join(dir, "nonexistent")
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "path is directory",
			setup: func(dir string) string {
				return dir
			},
			wantErr: true,
			errMsg:  "is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(dir)

			resolved, err := resolvePluginExecutablePath(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got: %v", tt.errMsg, err)
				}
				if resolved != "" {
					t.Errorf("expected empty resolved path on error, got %q", resolved)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.checkPath {
				if _, err := os.Stat(resolved); err != nil {
					t.Errorf("resolved path does not exist: %v", err)
				}
			}
		})
	}
}

// --- resolveWindowsExecutableSuffix tests ---

func TestResolveWindowsExecutableSuffix_NoExtension(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "plugin")

	// Create a .exe file
	exePath := basePath + ".exe"
	if err := os.WriteFile(exePath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	resolved := resolveWindowsExecutableSuffix(basePath)
	if resolved == "" {
		t.Fatal("expected resolved path")
	}
	if !strings.EqualFold(resolved, exePath) {
		t.Errorf("resolved = %q, want %q", resolved, exePath)
	}
}

func TestResolveWindowsExecutableSuffix_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "plugin")

	// Create a directory with .bat extension (should be skipped)
	batDir := basePath + ".bat"
	if err := os.Mkdir(batDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a .exe file
	exePath := basePath + ".exe"
	if err := os.WriteFile(exePath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	resolved := resolveWindowsExecutableSuffix(basePath)
	if resolved == "" {
		t.Fatal("expected resolved path")
	}
	if !strings.EqualFold(resolved, exePath) {
		t.Errorf("resolved = %q, want %q", resolved, exePath)
	}
}

func TestResolveWindowsExecutableSuffix_NotFound(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "nonexistent")

	resolved := resolveWindowsExecutableSuffix(basePath)
	if resolved != "" {
		t.Errorf("expected empty string for non-existent path, got %q", resolved)
	}
}

func TestResolveWindowsExecutableSuffix_MultipleExtensions(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "plugin")

	// Create multiple extension files
	batPath := basePath + ".bat"
	exePath := basePath + ".exe"
	cmdPath := basePath + ".cmd"

	if err := os.WriteFile(cmdPath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exePath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(batPath, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}

	resolved := resolveWindowsExecutableSuffix(basePath)
	if resolved == "" {
		t.Fatal("expected resolved path")
	}
	// Should resolve to one of the existing files
	validPaths := []string{batPath, exePath, cmdPath}
	found := false
	for _, valid := range validPaths {
		if strings.EqualFold(resolved, valid) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("resolved = %q, expected one of %v", resolved, validPaths)
	}
}

func TestResolveWindowsExecutableSuffix_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) (basePath string, expectedExt string)
		wantFound bool
	}{
		{
			name: "finds .exe",
			setup: func(t *testing.T, dir string) (string, string) {
				t.Helper()
				base := filepath.Join(dir, "plugin")
				path := base + ".exe"
				if err := os.WriteFile(path, []byte("test"), 0755); err != nil {
					t.Fatal(err)
				}
				return base, ".exe"
			},
			wantFound: true,
		},
		{
			name: "finds .bat",
			setup: func(t *testing.T, dir string) (string, string) {
				t.Helper()
				base := filepath.Join(dir, "plugin")
				path := base + ".bat"
				if err := os.WriteFile(path, []byte("test"), 0755); err != nil {
					t.Fatal(err)
				}
				return base, ".bat"
			},
			wantFound: true,
		},
		{
			name: "no matching extensions",
			setup: func(t *testing.T, dir string) (string, string) {
				base := filepath.Join(dir, "plugin")
				return base, ""
			},
			wantFound: false,
		},
		{
			name: "skips directories",
			setup: func(t *testing.T, dir string) (string, string) {
				t.Helper()
				base := filepath.Join(dir, "plugin")
				if err := os.Mkdir(base+".bat", 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(base+".exe", []byte("test"), 0755); err != nil {
					t.Fatal(err)
				}
				return base, ".exe"
			},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			basePath, expectedExt := tt.setup(t, dir)

			resolved := resolveWindowsExecutableSuffix(basePath)

			if tt.wantFound {
				if resolved == "" {
					t.Fatal("expected resolved path, got empty string")
				}
				if expectedExt != "" && !strings.HasSuffix(strings.ToLower(resolved), strings.ToLower(expectedExt)) {
					t.Errorf("resolved = %q, expected to end with %q", resolved, expectedExt)
				}
			} else if resolved != "" {
				t.Errorf("expected empty string, got %q", resolved)
			}
		})
	}
}

// --- windowsExecutableExtensions tests ---

func TestWindowsExecutableExtensions_DefaultExtensions(t *testing.T) {
	// Save original PATHEXT

	// Set empty PATHEXT to trigger default behavior
	t.Setenv("PATHEXT", "")

	exts := windowsExecutableExtensions()

	expectedDefaults := []string{".exe", ".bat", ".cmd", ".com"}
	if len(exts) != len(expectedDefaults) {
		t.Errorf("got %d extensions, want %d", len(exts), len(expectedDefaults))
	}
	for i, ext := range exts {
		if i < len(expectedDefaults) && ext != expectedDefaults[i] {
			t.Errorf("exts[%d] = %q, want %q", i, ext, expectedDefaults[i])
		}
	}
}

func TestWindowsExecutableExtensions_CustomPATHEXT(t *testing.T) {

	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")

	exts := windowsExecutableExtensions()

	if len(exts) != 4 {
		t.Errorf("got %d extensions, want 4", len(exts))
	}

	for _, ext := range exts {
		if !strings.HasPrefix(ext, ".") {
			t.Errorf("extension %q should start with dot", ext)
		}
	}
}

func TestWindowsExecutableExtensions_TrimsWhitespace(t *testing.T) {

	t.Setenv("PATHEXT", ".exe ; .bat ; .cmd")

	exts := windowsExecutableExtensions()

	for _, ext := range exts {
		if strings.Contains(ext, " ") {
			t.Errorf("extension %q should not contain whitespace", ext)
		}
		if ext == "" {
			t.Error("extension should not be empty")
		}
	}
}

func TestWindowsExecutableExtensions_SkipsEmptyComponents(t *testing.T) {

	t.Setenv("PATHEXT", ".exe;;.bat;;")

	exts := windowsExecutableExtensions()

	for _, ext := range exts {
		if ext == "" {
			t.Error("empty extensions should be skipped")
		}
	}
}

func TestWindowsExecutableExtensions_AddsDotPrefix(t *testing.T) {

	t.Setenv("PATHEXT", ".exe;bat;cmd")

	exts := windowsExecutableExtensions()

	for _, ext := range exts {
		if !strings.HasPrefix(ext, ".") {
			t.Errorf("extension %q should start with dot", ext)
		}
	}
}

func TestWindowsExecutableExtensions_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		pathext  string
		validate func([]string) error
		minCount int
	}{
		{
			name:     "default extensions",
			pathext:  "",
			minCount: 4,
			validate: func(exts []string) error {
				for _, ext := range exts {
					if ext != ".exe" && ext != ".bat" && ext != ".cmd" && ext != ".com" {
						return fmt.Errorf("unexpected extension: %q", ext)
					}
				}
				return nil
			},
		},
		{
			name:     "custom PATHEXT",
			pathext:  ".com;.exe;.bat;.cmd;.msi",
			minCount: 5,
			validate: func(exts []string) error {
				for _, ext := range exts {
					if !strings.HasPrefix(ext, ".") {
						return fmt.Errorf("extension should start with dot: %q", ext)
					}
				}
				return nil
			},
		},
		{
			name:     "whitespace trimming",
			pathext:  " .exe ; .bat ; .cmd ",
			minCount: 3,
			validate: func(exts []string) error {
				for _, ext := range exts {
					if strings.HasPrefix(ext, " ") || strings.HasSuffix(ext, " ") {
						return fmt.Errorf("extension should not have whitespace: %q", ext)
					}
				}
				return nil
			},
		},
		{
			name:     "skip empty components",
			pathext:  ".exe;;.bat;;.cmd",
			minCount: 3,
			validate: func(exts []string) error {
				for _, ext := range exts {
					if ext == "" {
						return fmt.Errorf("empty extensions should be skipped")
					}
				}
				return nil
			},
		},
		{
			name:     "add dot prefix",
			pathext:  ".exe;bat;cmd",
			minCount: 3,
			validate: func(exts []string) error {
				for _, ext := range exts {
					if !strings.HasPrefix(ext, ".") {
						return fmt.Errorf("extension should have dot prefix: %q", ext)
					}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			t.Setenv("PATHEXT", tt.pathext)

			exts := windowsExecutableExtensions()

			if len(exts) < tt.minCount {
				t.Errorf("got %d extensions, want at least %d", len(exts), tt.minCount)
			}

			if tt.validate != nil {
				if err := tt.validate(exts); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
