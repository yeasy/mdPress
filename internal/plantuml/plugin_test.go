package plantuml

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/plugin"
)

// TestNewPlugin tests that NewPlugin creates a plugin with default renderer settings.
func TestNewPlugin(t *testing.T) {
	p := NewPlugin()
	if p == nil {
		t.Fatal("NewPlugin returned nil")
	}
	if p.renderer == nil {
		t.Fatal("renderer should not be nil")
	}
}

// TestPluginName tests the Name method returns the expected plugin name.
func TestPluginName(t *testing.T) {
	p := NewPlugin()
	if p.Name() != "plantuml" {
		t.Errorf("Name() = %q, want %q", p.Name(), "plantuml")
	}
}

// TestPluginVersion tests the Version method returns a semantic version.
func TestPluginVersion(t *testing.T) {
	p := NewPlugin()
	if p.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want %q", p.Version(), "1.0.0")
	}
}

// TestPluginDescription tests the Description method returns a non-empty string.
func TestPluginDescription(t *testing.T) {
	p := NewPlugin()
	desc := p.Description()
	if desc == "" {
		t.Fatal("Description() returned empty string")
	}
	if desc != "Renders PlantUML diagrams in Markdown content" {
		t.Errorf("Description() = %q", desc)
	}
}

// TestPluginInit_NoConfig tests Init with a config that has no PlantUML plugin config.
func TestPluginInit_NoConfig(t *testing.T) {
	p := NewPlugin()
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{} // No plugins configured

	err := p.Init(cfg)
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}
	// Renderer should still be initialized with defaults
	if p.renderer == nil {
		t.Fatal("renderer should not be nil after Init")
	}
}

// TestPluginInit_WithServerURL tests Init with custom server URL.
func TestPluginInit_WithServerURL(t *testing.T) {
	p := NewPlugin()
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "plantuml",
			Path: "",
			Config: map[string]any{
				"server": "http://custom-plantuml.example.com/plantuml",
			},
		},
	}

	// The server URL uses a non-existent domain, so Init should return a validation error.
	err := p.Init(cfg)
	if err == nil {
		t.Fatal("expected Init to fail for non-resolvable server URL")
	}
	if !strings.Contains(err.Error(), "dns resolution failed") {
		t.Fatalf("expected DNS resolution error, got: %v", err)
	}
}

// TestPluginInit_WithLocalFlag tests Init with local rendering flag.
func TestPluginInit_WithLocalFlag(t *testing.T) {
	p := NewPlugin()
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "plantuml",
			Path: "",
			Config: map[string]any{
				"local": true,
			},
		},
	}

	err := p.Init(cfg)
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}
	if p.renderer == nil {
		t.Fatal("renderer should not be nil after Init")
	}
}

// TestPluginInit_WithBothOptions tests Init with both server and local options.
func TestPluginInit_WithBothOptions(t *testing.T) {
	p := NewPlugin()
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "plantuml",
			Path: "",
			Config: map[string]any{
				"server": "http://plantuml.example.com",
				"local":  false,
			},
		},
	}

	// The server URL uses a non-existent domain, so Init should return a validation error.
	err := p.Init(cfg)
	if err == nil {
		t.Fatal("expected Init to fail for non-resolvable server URL")
	}
}

// TestPluginInit_InvalidConfigTypes tests Init handles incorrect config value types gracefully.
func TestPluginInit_InvalidConfigTypes(t *testing.T) {
	p := NewPlugin()
	cfg := config.DefaultConfig()
	cfg.Plugins = []config.PluginConfig{
		{
			Name: "plantuml",
			Path: "",
			Config: map[string]any{
				// Use wrong types: server should be string, local should be bool.
				"server": 12345,
				"local":  "true",
			},
		},
	}

	err := p.Init(cfg)
	if err != nil {
		t.Fatalf("Init should ignore invalid types, got error: %v", err)
	}
	// Should use defaults when types are invalid
	if p.renderer == nil {
		t.Fatal("renderer should still be initialized")
	}
}

// TestPluginHooks tests the Hooks method returns the correct phase.
func TestPluginHooks(t *testing.T) {
	p := NewPlugin()
	hooks := p.Hooks()
	if len(hooks) != 1 {
		t.Fatalf("Hooks() returned %d phases, want 1", len(hooks))
	}
	if hooks[0] != plugin.PhaseAfterParse {
		t.Errorf("Hooks()[0] = %q, want %q", hooks[0], plugin.PhaseAfterParse)
	}
}

// TestPluginExecute_WrongPhase tests Execute returns nil when phase doesn't match.
func TestPluginExecute_WrongPhase(t *testing.T) {
	p := NewPlugin()
	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseBeforeBuild, // Not AfterParse
		Content:  "<p>Hello</p>",
		Metadata: map[string]any{},
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result != nil {
		t.Error("Execute should return nil for non-matching phase")
	}
}

// TestPluginExecute_CorrectPhase_NoPlantUML tests Execute with AfterParse phase but no PlantUML content.
func TestPluginExecute_CorrectPhase_NoPlantUML(t *testing.T) {
	p := NewPlugin()
	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseAfterParse,
		Content:  "<p>Just regular HTML</p>",
		Metadata: map[string]any{},
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return a HookResult even without PlantUML")
	}
	if result.Content != "<p>Just regular HTML</p>" {
		t.Errorf("content should be unchanged, got %q", result.Content)
	}
}

// TestPluginExecute_WithPlantUML tests Execute with PlantUML content.
func TestPluginExecute_WithPlantUML(t *testing.T) {
	// Create a mock PlantUML server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"></svg>`))
	}))
	defer mockServer.Close()

	p := NewPlugin()
	p.renderer = newRendererNoValidation(mockServer.URL, false)

	html := `<pre><code class="language-plantuml">Alice -> Bob: Hello</code></pre>`
	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseAfterParse,
		Content:  html,
		Metadata: map[string]any{},
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return a HookResult")
	}
	// Verify the result contains a diagram div
	if result.Content == "" {
		t.Error("result Content should not be empty")
	}
}

// TestPluginExecute_RenderError tests Execute handles renderer errors gracefully.
func TestPluginExecute_RenderError(t *testing.T) {
	// Create a mock server that fails
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error"))
	}))
	defer mockServer.Close()

	p := NewPlugin()
	p.renderer = newRendererNoValidation(mockServer.URL, false)

	html := `<pre><code class="language-plantuml">Alice -> Bob</code></pre>`
	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseAfterParse,
		Content:  html,
		Metadata: make(map[string]any),
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute should not return Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return a HookResult even on error")
	}
	// Original content should be returned
	if result.Content == "" {
		t.Error("result Content should not be empty")
	}
	// Error should be stored in metadata
	if _, ok := ctx.Metadata["plantuml.error"]; !ok {
		t.Error("error should be stored in metadata")
	}
}

// TestPluginExecute_NilMetadata tests Execute initializes nil Metadata.
func TestPluginExecute_NilMetadata(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	p := NewPlugin()
	p.renderer = newRendererNoValidation(mockServer.URL, false)

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseAfterParse,
		Content:  `<pre><code class="language-plantuml">Alice -> Bob</code></pre>`,
		Metadata: nil,
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return a HookResult")
	}
	// Metadata should be initialized even if it was nil
	if ctx.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

// TestPluginCleanup tests Cleanup returns nil without errors.
func TestPluginCleanup(t *testing.T) {
	p := NewPlugin()
	err := p.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup returned unexpected error: %v", err)
	}
}

// TestEnableIfNeeded_Empty tests enableIfNeeded with empty chapter list.
func TestEnableIfNeeded_Empty(t *testing.T) {
	chapters := []string{}
	if enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should return false for empty chapters")
	}
}

// TestEnableIfNeeded_NoPlantUML tests enableIfNeeded with chapters that have no PlantUML.
func TestEnableIfNeeded_NoPlantUML(t *testing.T) {
	chapters := []string{
		"# Chapter 1\nSome content",
		"# Chapter 2\nMore content\n```python\ncode\n```",
	}
	if enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should return false when no PlantUML blocks found")
	}
}

// TestEnableIfNeeded_WithPlantUML tests enableIfNeeded detects PlantUML blocks.
func TestEnableIfNeeded_WithPlantUML(t *testing.T) {
	chapters := []string{
		"# Chapter 1\nSome content",
		"# Chapter 2\n```plantuml\nAlice -> Bob\n```",
	}
	if !enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should return true when PlantUML block is found")
	}
}

// TestEnableIfNeeded_FirstChapter tests enableIfNeeded when first chapter has PlantUML.
func TestEnableIfNeeded_FirstChapter(t *testing.T) {
	chapters := []string{
		"```plantuml\nAlice -> Bob\n```",
		"# Chapter 2\nMore content",
	}
	if !enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should return true for PlantUML in first chapter")
	}
}

// TestEnableIfNeeded_MultipleBlocks tests enableIfNeeded with multiple PlantUML blocks.
func TestEnableIfNeeded_MultipleBlocks(t *testing.T) {
	chapters := []string{
		"```plantuml\nAlice -> Bob\n```\nContent\n```plantuml\nC -> D\n```",
	}
	if !enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should return true with multiple PlantUML blocks")
	}
}

// TestEnableIfNeeded_SimilarButNotPlantUML tests enableIfNeeded doesn't match partial strings.
func TestEnableIfNeeded_SimilarButNotPlantUML(t *testing.T) {
	chapters := []string{
		"Here's some plantuml documentation",
		"Use code blocks like ```python or ```bash, but not plantuml",
	}
	// These should not match the exact ```plantuml marker
	if enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should not match 'plantuml' in general text or partial markers")
	}
}

// TestEnableIfNeeded_CaseSensitive tests enableIfNeeded is case-sensitive.
func TestEnableIfNeeded_CaseSensitive(t *testing.T) {
	chapters := []string{
		"```PLANTUML\nAlice -> Bob\n```",
		"```PlantUML\nAlice -> Bob\n```",
	}
	if enableIfNeeded(chapters) {
		t.Error("enableIfNeeded should be case-sensitive and only match lowercase ```plantuml")
	}
}

// TestPluginLifecycle tests the complete plugin lifecycle.
func TestPluginLifecycle(t *testing.T) {
	p := NewPlugin()

	// Test Name, Version, Description
	if p.Name() == "" || p.Version() == "" || p.Description() == "" {
		t.Fatal("Name, Version, and Description must not be empty")
	}

	// Test Init
	cfg := config.DefaultConfig()
	if err := p.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test Hooks returns AfterParse
	hooks := p.Hooks()
	if len(hooks) == 0 {
		t.Fatal("Hooks should not be empty")
	}

	// Test Execute with correct phase
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<svg></svg>`))
	}))
	defer mockServer.Close()

	p.renderer = newRendererNoValidation(mockServer.URL, false)
	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Phase:    plugin.PhaseAfterParse,
		Content:  `<pre><code class="language-plantuml">A -> B</code></pre>`,
		Metadata: map[string]any{},
	}

	result, err := p.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Execute should return a HookResult")
	}

	// Test Cleanup
	if err := p.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}
