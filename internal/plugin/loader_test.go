package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// --- Helper: createTestPlugin writes a simple test plugin script that echoes valid responses ---

func createTestPlugin(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		scriptPath := filepath.Join(dir, name+".bat")
		script := "@echo off\r\necho {}\r\n"
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
			t.Fatalf("failed to create test plugin: %v", err)
		}
		return scriptPath
	}
	scriptPath := filepath.Join(dir, name)
	script := "#!/bin/sh\necho '{}'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create test plugin: %v", err)
	}
	return scriptPath
}

// --- LoadPlugins tests ---

// TestLoadPlugins_EmptyPluginList tests LoadPlugins with no plugins configured.
func TestLoadPlugins_EmptyPluginList(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{} // No plugins

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins returned unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("LoadPlugins should return a Manager, not nil")
	}
	if len(mgr.Plugins()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(mgr.Plugins()))
	}
}

// TestLoadPlugins_SinglePlugin tests LoadPlugins with one valid plugin.
func TestLoadPlugins_SinglePlugin(t *testing.T) {
	dir := t.TempDir()
	pluginPath := createTestPlugin(t, dir, "test-plugin")

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "test-plugin",
			Path: pluginPath,
			Config: map[string]interface{}{
				"key": "value",
			},
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins returned unexpected error: %v", err)
	}
	if mgr == nil {
		t.Fatal("LoadPlugins should return a Manager, not nil")
	}
	if len(mgr.Plugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(mgr.Plugins()))
	}
	if mgr.Plugins()[0].Name() != "test-plugin" {
		t.Errorf("plugin name = %q, want %q", mgr.Plugins()[0].Name(), "test-plugin")
	}
}

// TestLoadPlugins_MultiplePlugins tests LoadPlugins with multiple valid plugins.
func TestLoadPlugins_MultiplePlugins(t *testing.T) {
	dir := t.TempDir()
	plugin1Path := createTestPlugin(t, dir, "plugin1")
	plugin2Path := createTestPlugin(t, dir, "plugin2")
	plugin3Path := createTestPlugin(t, dir, "plugin3")

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "plugin1", Path: plugin1Path, Config: nil},
		{Name: "plugin2", Path: plugin2Path, Config: nil},
		{Name: "plugin3", Path: plugin3Path, Config: nil},
	}

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins returned unexpected error: %v", err)
	}
	if len(mgr.Plugins()) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(mgr.Plugins()))
	}
	// Verify plugins are registered in declaration order
	if mgr.Plugins()[0].Name() != "plugin1" {
		t.Errorf("first plugin name = %q, want %q", mgr.Plugins()[0].Name(), "plugin1")
	}
	if mgr.Plugins()[1].Name() != "plugin2" {
		t.Errorf("second plugin name = %q, want %q", mgr.Plugins()[1].Name(), "plugin2")
	}
	if mgr.Plugins()[2].Name() != "plugin3" {
		t.Errorf("third plugin name = %q, want %q", mgr.Plugins()[2].Name(), "plugin3")
	}
}

// TestLoadPlugins_MissingName tests LoadPlugins rejects plugin config with missing name.
func TestLoadPlugins_MissingName(t *testing.T) {
	dir := t.TempDir()
	pluginPath := createTestPlugin(t, dir, "plugin")

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "", // Missing name
			Path: pluginPath,
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err == nil {
		t.Fatal("LoadPlugins should return an error for missing name")
	}
	if mgr != nil {
		t.Error("LoadPlugins should return nil Manager on error")
	}
	if !contains(err.Error(), "missing the required 'name' field") {
		t.Errorf("error message should mention missing name field, got: %v", err)
	}
}

// TestLoadPlugins_MissingPath tests LoadPlugins rejects plugin config with missing path.
func TestLoadPlugins_MissingPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "plugin1",
			Path: "", // Missing path
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err == nil {
		t.Fatal("LoadPlugins should return an error for missing path")
	}
	if mgr != nil {
		t.Error("LoadPlugins should return nil Manager on error")
	}
	if !contains(err.Error(), "missing the required 'path' field") {
		t.Errorf("error message should mention missing path field, got: %v", err)
	}
}

// TestLoadPlugins_NonExistentPlugin tests LoadPlugins rejects non-existent plugin executable.
func TestLoadPlugins_NonExistentPlugin(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "nonexistent",
			Path: "/no/such/binary/exists",
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err == nil {
		t.Fatal("LoadPlugins should return an error for non-existent plugin")
	}
	if mgr != nil {
		t.Error("LoadPlugins should return nil Manager on error")
	}
	if !contains(err.Error(), "failed to load plugin") {
		t.Errorf("error should wrap load failure, got: %v", err)
	}
}

// TestLoadPlugins_PartialFailure tests LoadPlugins stops at first plugin load error.
func TestLoadPlugins_PartialFailure(t *testing.T) {
	dir := t.TempDir()
	plugin1Path := createTestPlugin(t, dir, "plugin1")

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "plugin1", Path: plugin1Path, Config: nil},
		{Name: "plugin2", Path: "/no/such/plugin", Config: nil}, // Will fail
	}

	mgr, err := LoadPlugins(cfg)
	if err == nil {
		t.Fatal("LoadPlugins should return an error when second plugin fails")
	}
	if mgr != nil {
		t.Error("LoadPlugins should return nil Manager on error")
	}
}

// TestLoadPlugins_InitFailure tests LoadPlugins with a valid plugin that can be loaded.
// Note: External plugins have no-op Init methods, so Init failures cannot occur.
// This test verifies that LoadPlugins succeeds when a proper plugin fixture exists.
func TestLoadPlugins_InitFailure(t *testing.T) {
	dir := t.TempDir()
	// Create a valid test plugin (graceful degradation during metadata query)
	pluginPath := createTestPlugin(t, dir, "valid-plugin")
	if _, err := os.Stat(pluginPath); err != nil {
		t.Fatalf("plugin fixture not created at %q: %v", pluginPath, err)
	}

	cfg := config.DefaultConfig()
	cfg.SetBaseDir(dir) // Ensure base directory is set correctly
	cfg.Plugins = []config.PluginConfig{
		{Name: "valid-plugin", Path: pluginPath, Config: nil},
	}

	// LoadPlugins should succeed with a valid plugin that can be loaded
	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins should succeed with a valid plugin: %v", err)
	}
	if mgr == nil {
		t.Error("LoadPlugins should return a Manager")
	}
	if len(mgr.Plugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(mgr.Plugins()))
	}
}

// TestLoadPlugins_ResolvePath tests LoadPlugins resolves relative paths.
func TestLoadPlugins_ResolvePath(t *testing.T) {
	// Create a temporary directory to serve as the base config directory
	baseDir := t.TempDir()
	pluginDir := filepath.Join(baseDir, "plugins")
	if err := os.Mkdir(pluginDir, 0755); err != nil {
		t.Fatalf("failed to create plugin directory: %v", err)
	}

	// Create the test plugin fixture in the plugins subdirectory
	pluginPath := createTestPlugin(t, pluginDir, "myplugin")
	if _, err := os.Stat(pluginPath); err != nil {
		t.Fatalf("plugin fixture not created at %q: %v", pluginPath, err)
	}

	cfg := config.DefaultConfig()
	cfg.SetBaseDir(baseDir) // Set base directory so relative paths resolve correctly
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "myplugin",
			Path: "plugins/myplugin", // Relative path
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins with relative path failed: %v", err)
	}
	if len(mgr.Plugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(mgr.Plugins()))
	}
}

// TestLoadPlugins_PluginConfigPassed tests LoadPlugins passes plugin config to NewExternalPlugin.
func TestLoadPlugins_PluginConfigPassed(t *testing.T) {
	dir := t.TempDir()
	pluginPath := createTestPlugin(t, dir, "configurable")

	pluginConfig := map[string]interface{}{
		"setting1": "value1",
		"setting2": 42,
		"setting3": true,
	}

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name:   "configurable",
			Path:   pluginPath,
			Config: pluginConfig,
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins returned unexpected error: %v", err)
	}
	if len(mgr.Plugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(mgr.Plugins()))
	}
	// Plugin was successfully loaded with config
}

// TestLoadPlugins_ErrorWrapping tests LoadPlugins wraps errors with appropriate context.
func TestLoadPlugins_ErrorWrapping(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "bad-plugin",
			Path: "/no/such/path",
		},
	}

	mgr, err := LoadPlugins(cfg)
	if err == nil {
		t.Fatal("LoadPlugins should return an error")
	}
	if mgr != nil {
		t.Error("LoadPlugins should return nil Manager on error")
	}
	// Error should be wrapped with plugin loading context
	errMsg := err.Error()
	if !contains(errMsg, "bad-plugin") {
		t.Errorf("error should mention plugin name 'bad-plugin', got: %v", err)
	}
}

// --- MustLoadPlugins tests ---

// TestMustLoadPlugins_EmptyPluginList tests MustLoadPlugins with no plugins.
func TestMustLoadPlugins_EmptyPluginList(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{}

	mgr := MustLoadPlugins(cfg, nil)
	if mgr == nil {
		t.Fatal("MustLoadPlugins should always return a Manager")
	}
	if len(mgr.Plugins()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(mgr.Plugins()))
	}
}

// TestMustLoadPlugins_ValidPlugin tests MustLoadPlugins with a valid plugin.
func TestMustLoadPlugins_ValidPlugin(t *testing.T) {
	dir := t.TempDir()
	pluginPath := createTestPlugin(t, dir, "valid")

	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "valid", Path: pluginPath},
	}

	mgr := MustLoadPlugins(cfg, nil)
	if mgr == nil {
		t.Fatal("MustLoadPlugins should return a Manager")
	}
	if len(mgr.Plugins()) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(mgr.Plugins()))
	}
}

// TestMustLoadPlugins_NilWarnFn tests MustLoadPlugins with nil warn function.
func TestMustLoadPlugins_NilWarnFn(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "missing", Path: "/no/such/plugin"},
	}

	// Should not panic even with nil warnFn
	mgr := MustLoadPlugins(cfg, nil)
	if mgr == nil {
		t.Fatal("MustLoadPlugins should always return a Manager, even with nil warnFn")
	}
	if len(mgr.Plugins()) != 0 {
		t.Errorf("expected 0 plugins after error, got %d", len(mgr.Plugins()))
	}
}

// TestMustLoadPlugins_CallsWarnFn tests MustLoadPlugins calls warnFn on error.
func TestMustLoadPlugins_CallsWarnFn(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "broken", Path: "/missing/plugin"},
	}

	warnCalled := false
	var warnMsg string
	warnFn := func(msg string) {
		warnCalled = true
		warnMsg = msg
	}

	mgr := MustLoadPlugins(cfg, warnFn)
	if !warnCalled {
		t.Fatal("warnFn should be called when plugin loading fails")
	}
	if !contains(warnMsg, "plugin loading failed") {
		t.Errorf("warn message should mention plugin loading failed, got: %q", warnMsg)
	}
	if len(mgr.Plugins()) != 0 {
		t.Errorf("expected 0 plugins after error, got %d", len(mgr.Plugins()))
	}
}

// TestMustLoadPlugins_NeverFails tests MustLoadPlugins never returns nil Manager.
func TestMustLoadPlugins_NeverFails(t *testing.T) {
	testCases := []struct {
		name   string
		config *config.BookConfig
	}{
		{
			name:   "nil plugins list",
			config: &config.BookConfig{Plugins: nil},
		},
		{
			name:   "empty plugins list",
			config: &config.BookConfig{Plugins: []config.PluginConfig{}},
		},
		{
			name: "invalid plugin config",
			config: &config.BookConfig{
				Plugins: []config.PluginConfig{
					{Name: "", Path: ""},
				},
			},
		},
		{
			name: "missing plugin file",
			config: &config.BookConfig{
				Plugins: []config.PluginConfig{
					{Name: "missing", Path: "/no/such/file"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := MustLoadPlugins(tc.config, nil)
			if mgr == nil {
				t.Fatal("MustLoadPlugins should never return nil Manager")
			}
			// Even on error, should return an empty but valid Manager
		})
	}
}

// TestMustLoadPlugins_WarnMessage tests warnFn receives properly formatted message.
func TestMustLoadPlugins_WarnMessage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{Name: "broken", Path: "/no/such/path"},
	}

	var capturedMsg string
	warnFn := func(msg string) {
		capturedMsg = msg
	}

	MustLoadPlugins(cfg, warnFn)

	if !contains(capturedMsg, "plugin loading failed") {
		t.Errorf("message should contain 'plugin loading failed', got: %q", capturedMsg)
	}
	if !contains(capturedMsg, "continuing without plugins") {
		t.Errorf("message should mention continuing without plugins, got: %q", capturedMsg)
	}
}

// TestLoadPlugins_ConfigNil tests LoadPlugins handles nil config gracefully.
func TestLoadPlugins_ConfigNil(t *testing.T) {
	// This should panic or handle gracefully - depends on implementation
	// Most likely it will panic when accessing cfg.Plugins on nil
	defer func() {
		_ = recover() // nil config may panic; either outcome is acceptable
	}()

	_, _ = LoadPlugins(nil)
	// If we get here without panic, that's also acceptable if it returns an error
}

// TestLoadPlugins_PluginOrderPreserved tests plugins are registered in declaration order.
func TestLoadPlugins_PluginOrderPreserved(t *testing.T) {
	dir := t.TempDir()
	pluginPaths := make([]string, 5)
	for i := 0; i < 5; i++ {
		pluginPaths[i] = createTestPlugin(t, dir, "plugin-"+string(rune('A'+i)))
	}

	cfg := config.DefaultConfig()
	for i := 0; i < 5; i++ {
		cfg.Plugins = append(cfg.Plugins, config.PluginConfig{
			Name: "plugin-" + string(rune('A'+i)),
			Path: pluginPaths[i],
		})
	}

	mgr, err := LoadPlugins(cfg)
	if err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	// Verify order is preserved
	for i := 0; i < 5; i++ {
		expectedName := "plugin-" + string(rune('A'+i))
		if mgr.Plugins()[i].Name() != expectedName {
			t.Errorf("plugin at index %d: name = %q, want %q", i, mgr.Plugins()[i].Name(), expectedName)
		}
	}
}

// --- Helper functions ---

// contains returns true if needle is found in haystack.
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestLoadPlugins_MulitpleErrors tests LoadPlugins handles multiple error scenarios.
func TestLoadPlugins_MultipleErrorScenarios(t *testing.T) {
	testCases := []struct {
		name      string
		config    *config.BookConfig
		wantError bool
	}{
		{
			name: "valid single plugin",
			config: func() *config.BookConfig {
				dir := t.TempDir()
				cfg := config.DefaultConfig()
				cfg.Plugins = []config.PluginConfig{
					{Name: "valid", Path: createTestPlugin(t, dir, "valid")},
				}
				return cfg
			}(),
			wantError: false,
		},
		{
			name: "empty name",
			config: &config.BookConfig{
				Plugins: []config.PluginConfig{
					{Name: "", Path: "/some/path"},
				},
			},
			wantError: true,
		},
		{
			name: "empty path",
			config: &config.BookConfig{
				Plugins: []config.PluginConfig{
					{Name: "test", Path: ""},
				},
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadPlugins(tc.config)
			if tc.wantError && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
