package plugin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/plugin"
)

// ---- mock plugin for testing ----

type mockPlugin struct {
	name    string
	hooks   []plugin.Phase
	initErr error
	execFn  func(*plugin.HookContext) (*plugin.HookResult, error)
	cleaned bool
}

func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Version() string     { return "0.0.1" }
func (m *mockPlugin) Description() string { return "mock" }
func (m *mockPlugin) Init(_ *config.BookConfig) error {
	return m.initErr
}
func (m *mockPlugin) Hooks() []plugin.Phase { return m.hooks }
func (m *mockPlugin) Execute(ctx *plugin.HookContext) (*plugin.HookResult, error) {
	if m.execFn != nil {
		return m.execFn(ctx)
	}
	return &plugin.HookResult{}, nil
}
func (m *mockPlugin) Cleanup() error {
	m.cleaned = true
	return nil
}

// ---- Manager tests ----

func TestManager_Register_And_RunHook_Passthrough(t *testing.T) {
	mgr := plugin.NewManager()

	called := false
	p := &mockPlugin{
		name:  "noop",
		hooks: []plugin.Phase{plugin.PhaseAfterParse},
		execFn: func(ctx *plugin.HookContext) (*plugin.HookResult, error) {
			called = true
			return &plugin.HookResult{}, nil
		},
	}
	mgr.Register(p)

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Config:   config.DefaultConfig(),
		Phase:    plugin.PhaseAfterParse,
		Content:  "original",
		Metadata: map[string]any{},
	}
	if err := mgr.RunHook(ctx); err != nil {
		t.Fatalf("RunHook returned unexpected error: %v", err)
	}
	if !called {
		t.Error("plugin was not called for its registered phase")
	}
	// Content should be unchanged when the plugin returns empty Content.
	if ctx.Content != "original" {
		t.Errorf("content changed unexpectedly: got %q", ctx.Content)
	}
}

func TestManager_RunHook_ContentReplacement(t *testing.T) {
	mgr := plugin.NewManager()

	mgr.Register(&mockPlugin{
		name:  "replacer",
		hooks: []plugin.Phase{plugin.PhaseAfterParse},
		execFn: func(ctx *plugin.HookContext) (*plugin.HookResult, error) {
			return &plugin.HookResult{Content: "replaced"}, nil
		},
	})

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Config:   config.DefaultConfig(),
		Phase:    plugin.PhaseAfterParse,
		Content:  "original",
		Metadata: map[string]any{},
	}
	if err := mgr.RunHook(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Content != "replaced" {
		t.Errorf("expected %q, got %q", "replaced", ctx.Content)
	}
}

func TestManager_RunHook_StopPropagation(t *testing.T) {
	mgr := plugin.NewManager()

	secondCalled := false
	mgr.Register(&mockPlugin{
		name:  "stopper",
		hooks: []plugin.Phase{plugin.PhaseAfterParse},
		execFn: func(_ *plugin.HookContext) (*plugin.HookResult, error) {
			return &plugin.HookResult{Stop: true}, nil
		},
	})
	mgr.Register(&mockPlugin{
		name:  "should-not-run",
		hooks: []plugin.Phase{plugin.PhaseAfterParse},
		execFn: func(_ *plugin.HookContext) (*plugin.HookResult, error) {
			secondCalled = true
			return &plugin.HookResult{}, nil
		},
	})

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Config:   config.DefaultConfig(),
		Phase:    plugin.PhaseAfterParse,
		Metadata: map[string]any{},
	}
	if err := mgr.RunHook(ctx); err != nil {
		t.Fatal(err)
	}
	if secondCalled {
		t.Error("second plugin was called despite Stop=true from first plugin")
	}
}

func TestManager_RunHook_PhaseFilter(t *testing.T) {
	mgr := plugin.NewManager()

	called := false
	mgr.Register(&mockPlugin{
		name:  "only-after-build",
		hooks: []plugin.Phase{plugin.PhaseAfterBuild},
		execFn: func(_ *plugin.HookContext) (*plugin.HookResult, error) {
			called = true
			return &plugin.HookResult{}, nil
		},
	})

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Config:   config.DefaultConfig(),
		Phase:    plugin.PhaseAfterParse, // different phase
		Metadata: map[string]any{},
	}
	if err := mgr.RunHook(ctx); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("plugin was called for a phase it did not register")
	}
}

func TestManager_RunHook_PropagatesError(t *testing.T) {
	mgr := plugin.NewManager()

	sentinel := errors.New("boom")
	mgr.Register(&mockPlugin{
		name:  "failer",
		hooks: []plugin.Phase{plugin.PhaseBeforeBuild},
		execFn: func(_ *plugin.HookContext) (*plugin.HookResult, error) {
			return nil, sentinel
		},
	})

	ctx := &plugin.HookContext{
		Context:  context.Background(),
		Config:   config.DefaultConfig(),
		Phase:    plugin.PhaseBeforeBuild,
		Metadata: map[string]any{},
	}
	err := mgr.RunHook(ctx)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error chain does not contain sentinel: %v", err)
	}
}

func TestManager_InitAll_Error(t *testing.T) {
	mgr := plugin.NewManager()

	mgr.Register(&mockPlugin{
		name:    "bad-init",
		hooks:   []plugin.Phase{plugin.PhaseAfterParse},
		initErr: errors.New("init failed"),
	})

	// InitAll should surface the init error wrapped in a PluginError.
	// We pass a minimal config; InitAll itself does not call Validate.
	cfg := config.DefaultConfig()
	if err := mgr.InitAll(cfg); err == nil {
		t.Fatal("expected InitAll to return an error")
	}
}

func TestManager_CleanupAll(t *testing.T) {
	mgr := plugin.NewManager()

	p := &mockPlugin{name: "cleanup-test", hooks: []plugin.Phase{plugin.PhaseAfterBuild}}
	mgr.Register(p)

	if err := mgr.CleanupAll(); err != nil {
		t.Fatalf("CleanupAll returned unexpected error: %v", err)
	}
	if !p.cleaned {
		t.Error("plugin Cleanup was not called")
	}
}

func TestPhaseConstants(t *testing.T) {
	// Verify that all seven phase constants are distinct.
	phases := []plugin.Phase{
		plugin.PhaseBeforeBuild,
		plugin.PhaseAfterParse,
		plugin.PhaseBeforeRender,
		plugin.PhaseAfterRender,
		plugin.PhaseAfterBuild,
		plugin.PhaseBeforeServe,
		plugin.PhaseAfterServe,
	}
	seen := map[plugin.Phase]bool{}
	for _, p := range phases {
		if seen[p] {
			t.Errorf("duplicate phase value: %q", p)
		}
		seen[p] = true
	}
	if len(seen) != 7 {
		t.Errorf("expected 7 distinct phases, got %d", len(seen))
	}
}
