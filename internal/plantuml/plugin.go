// Package plantuml - plugin.go registers PlantUML rendering as an mdpress plugin.
package plantuml

import (
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/plugin"
)

// Plugin implements the plugin.Plugin interface for PlantUML diagram rendering.
type Plugin struct {
	renderer *Renderer
}

// NewPlugin creates a new PlantUML plugin with default settings.
func NewPlugin() *Plugin {
	return &Plugin{
		renderer: NewRenderer("", false),
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "plantuml"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Renders PlantUML diagrams in Markdown content"
}

// Init initializes the plugin from configuration.
// Expects config options:
//   - server: Base URL for PlantUML server (optional, defaults to http://www.plantuml.com/plantuml)
//   - local: Boolean flag to use local plantuml command (optional, defaults to false)
func (p *Plugin) Init(cfg *config.BookConfig) error {
	serverURL := ""
	useLocal := false

	// Find the PlantUML plugin configuration if present
	for _, pc := range cfg.Plugins {
		if pc.Name == "plantuml" {
			if url, ok := pc.Config["server"]; ok {
				if s, ok := url.(string); ok {
					serverURL = s
				}
			}
			if local, ok := pc.Config["local"]; ok {
				if b, ok := local.(bool); ok {
					useLocal = b
				}
			}
			break
		}
	}

	p.renderer = NewRenderer(serverURL, useLocal)
	return nil
}

// Hooks returns the list of hook phases this plugin handles.
// PlantUML rendering happens after HTML parsing.
func (p *Plugin) Hooks() []plugin.Phase {
	return []plugin.Phase{plugin.PhaseAfterParse}
}

// Execute processes the HTML content to render PlantUML diagrams.
func (p *Plugin) Execute(hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	if hookCtx.Phase != plugin.PhaseAfterParse {
		return nil, nil
	}

	// Process the HTML content
	result, err := p.renderer.RenderHTML(hookCtx.Content)
	if err != nil {
		// Log the error but don't fail the build
		if hookCtx.Metadata == nil {
			hookCtx.Metadata = make(map[string]interface{})
		}
		hookCtx.Metadata["plantuml.error"] = err.Error()
		return &plugin.HookResult{Content: hookCtx.Content}, nil
	}

	return &plugin.HookResult{Content: result}, nil
}

// Cleanup performs any cleanup after the build finishes.
func (p *Plugin) Cleanup() error {
	// Nothing to clean up
	return nil
}

// EnableIfNeeded checks if any chapter contains PlantUML diagrams and returns
// true if the plugin should be auto-enabled.
func EnableIfNeeded(chapters []string) bool {
	for _, content := range chapters {
		if strings.Contains(content, "```plantuml") {
			return true
		}
	}
	return false
}
