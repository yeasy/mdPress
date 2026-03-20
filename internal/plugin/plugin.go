// Package plugin defines the plugin system interface for mdpress.
//
// Plugins are external executables that communicate with mdpress over JSON
// stdin/stdout.  They can intercept any of the seven build lifecycle phases
// to transform content, perform validation, or trigger side-effects such as
// uploading artifacts or sending notifications.
//
// The Manager handles plugin registration, ordering, and hook dispatch.
// Use plugin.LoadPlugins to create a Manager from a BookConfig, or call
// plugin.NewManager and Register plugins manually in tests.
package plugin

import (
	"context"
	"errors"

	"github.com/yeasy/mdpress/internal/config"
)

// Phase identifies a stage in the build pipeline at which plugins are invoked.
type Phase string

const (
	// PhaseBeforeBuild fires after configuration is loaded, before chapter processing begins.
	// Plugins can inspect or modify the configuration and perform environment checks.
	PhaseBeforeBuild Phase = "before_build"

	// PhaseAfterParse fires after each chapter's Markdown has been converted to HTML.
	// Plugins can modify the rendered HTML (e.g. inject custom elements, check links).
	PhaseAfterParse Phase = "after_parse"

	// PhaseBeforeRender fires before the per-format HTML assembly step.
	// Plugins can modify cover, TOC, or per-chapter HTML before the final document
	// is assembled.
	PhaseBeforeRender Phase = "before_render"

	// PhaseAfterRender fires after the HTML document has been fully assembled.
	// Plugins can post-process the complete HTML (e.g. inject SEO tags, watermarks).
	PhaseAfterRender Phase = "after_render"

	// PhaseAfterBuild fires after all output formats have been written to disk.
	// Plugins can run post-build tasks such as CDN uploads or build notifications.
	PhaseAfterBuild Phase = "after_build"

	// PhaseBeforeServe fires before the local preview server starts listening.
	// Plugins can run pre-serve checks or warm caches.
	PhaseBeforeServe Phase = "before_serve"

	// PhaseAfterServe fires after the local preview server shuts down.
	// Plugins can perform cleanup tasks.
	PhaseAfterServe Phase = "after_serve"
)

// HookContext carries all data available to a plugin at the point of invocation.
// Plugins may read any field and may modify Content to influence the build output.
type HookContext struct {
	// Context is a standard Go context used for timeout and cancellation.
	Context context.Context

	// Config is the current build configuration.  Plugins may read but should
	// not modify it after the BeforeBuild phase.
	Config *config.BookConfig

	// Phase is the lifecycle phase that triggered this invocation.
	Phase Phase

	// Content is the text being processed.
	// In AfterParse it holds the chapter HTML; in BeforeRender / AfterRender it
	// holds the cover HTML (as a representative payload).
	// A plugin that returns a non-empty Content replaces this value.
	Content string

	// ChapterIndex is the zero-based index of the chapter being processed.
	// Set to -1 in non-chapter contexts.
	ChapterIndex int

	// ChapterFile is the source file path of the chapter (AfterParse only).
	ChapterFile string

	// OutputPath is the path of the generated output file (AfterBuild only).
	OutputPath string

	// OutputFormat is the output format name (AfterBuild only).
	OutputFormat string

	// Metadata is a shared key-value map for passing data between phases or
	// between consecutive plugins.
	// No sync protection is needed: each HookContext instance gets its own map,
	// and RunHook is always called sequentially from the build pipeline.
	Metadata map[string]interface{}
}

// HookResult is returned by a plugin's Execute method.
type HookResult struct {
	// Content replaces HookContext.Content when non-empty.
	Content string

	// Stop instructs the Manager to skip all subsequent plugins for this phase.
	Stop bool
}

// Plugin is the interface that all mdpress plugins must implement.
type Plugin interface {
	// Name returns the unique plugin identifier (lowercase, hyphen-separated).
	Name() string

	// Version returns the semantic version of the plugin (e.g. "1.0.0").
	Version() string

	// Description returns a short human-readable description of the plugin.
	Description() string

	// Init is called once before the build starts.  It receives the complete
	// BookConfig so the plugin can validate its configuration.
	Init(cfg *config.BookConfig) error

	// Hooks returns the list of phases this plugin wants to handle.
	// The Manager only calls Execute for phases listed here.
	Hooks() []Phase

	// Execute is invoked at each phase listed by Hooks.
	// It receives a HookContext with the current build state and returns a
	// HookResult (or nil) plus any error.
	Execute(hookCtx *HookContext) (*HookResult, error)

	// Cleanup is called after the build finishes, regardless of success.
	Cleanup() error
}

// Manager registers plugins and dispatches hook calls to them in order.
type Manager struct {
	plugins []Plugin
}

// NewManager creates an empty Manager.
func NewManager() *Manager {
	return &Manager{
		plugins: make([]Plugin, 0),
	}
}

// Register appends p to the manager's plugin list.
// Plugins are executed in registration order.
func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, p)
}

// InitAll calls Init on every registered plugin.
// Returns the first error encountered, wrapped in a PluginError.
func (m *Manager) InitAll(cfg *config.BookConfig) error {
	for _, p := range m.plugins {
		if err := p.Init(cfg); err != nil {
			return &PluginError{
				PluginName: p.Name(),
				Phase:      "",
				Err:        err,
			}
		}
	}
	return nil
}

// RunHook dispatches the hook to every plugin that listed hookCtx.Phase in its
// Hooks list.  Plugins are called in registration order.  If a plugin sets
// HookResult.Stop the remaining plugins are skipped.
// The first plugin error is returned wrapped in a PluginError.
func (m *Manager) RunHook(hookCtx *HookContext) error {
	for _, p := range m.plugins {
		if !m.pluginHandlesPhase(p, hookCtx.Phase) {
			continue
		}
		result, err := p.Execute(hookCtx)
		if err != nil {
			return &PluginError{
				PluginName: p.Name(),
				Phase:      string(hookCtx.Phase),
				Err:        err,
			}
		}
		if result != nil {
			if result.Content != "" {
				hookCtx.Content = result.Content
			}
			if result.Stop {
				break
			}
		}
	}
	return nil
}

// CleanupAll calls Cleanup on every registered plugin.
// Errors do not prevent subsequent plugins from being cleaned up.
// Returns all errors encountered combined with errors.Join, or nil if no errors.
func (m *Manager) CleanupAll() error {
	var errs []error
	for _, p := range m.plugins {
		if err := p.Cleanup(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Plugins returns a snapshot of the registered plugin list.
func (m *Manager) Plugins() []Plugin {
	return m.plugins
}

// pluginHandlesPhase reports whether p listed phase in its Hooks list.
func (m *Manager) pluginHandlesPhase(p Plugin, phase Phase) bool {
	for _, h := range p.Hooks() {
		if h == phase {
			return true
		}
	}
	return false
}

// PluginError wraps an error returned by a plugin, adding the plugin name and
// the hook phase for diagnostic purposes.
type PluginError struct {
	PluginName string
	Phase      string
	Err        error
}

// Error implements the error interface.
func (e *PluginError) Error() string {
	if e.Phase != "" {
		return "plugin " + e.PluginName + " failed at phase " + e.Phase + ": " + e.Err.Error()
	}
	return "plugin " + e.PluginName + " failed during init: " + e.Err.Error()
}

// Unwrap supports errors.Is / errors.As unwrapping.
func (e *PluginError) Unwrap() error {
	return e.Err
}
