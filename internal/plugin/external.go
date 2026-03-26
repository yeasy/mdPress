// Package plugin - external.go implements a plugin backed by an external executable.
//
// External plugins are standalone programs that communicate with mdpress via JSON
// over stdin/stdout.  Each hook invocation starts the process, writes a JSON
// request to stdin, and reads a JSON response from stdout.  Any output written
// to stderr is captured and surfaced as debug/warning log messages.
package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yeasy/mdpress/internal/config"
)

const (
	// Plugin execution timeout for hook processing.
	defaultPluginTimeout = 30 * time.Second
	// Plugin metadata query timeout (--mdpress-info).
	pluginMetaQueryTimeout = 5 * time.Second
	// Plugin hooks query timeout (--mdpress-hooks).
	pluginHooksQueryTimeout = 5 * time.Second
)

// pluginMetaQueryFn and pluginHooksQueryFn are the functions used to query
// plugin metadata and supported hooks.  They are variables so tests can
// replace them with fast stubs to avoid subprocess overhead.
var pluginMetaQueryFn = queryPluginMeta
var pluginHooksQueryFn = queryPluginHooks

// ExternalPluginRequest is the JSON body sent to the external plugin process.
// It is serialized and written to the plugin's stdin on every hook invocation.
type ExternalPluginRequest struct {
	// Phase is the name of the current hook phase.
	Phase string `json:"phase"`
	// Content is the text being processed (Markdown or HTML, depending on phase).
	Content string `json:"content"`
	// ChapterIndex is the zero-based chapter index (-1 for non-chapter contexts).
	ChapterIndex int `json:"chapter_index"`
	// ChapterFile is the source path of the current chapter.
	ChapterFile string `json:"chapter_file"`
	// OutputPath is the output file path (valid in AfterBuild phase only).
	OutputPath string `json:"output_path"`
	// OutputFormat is the output format name (valid in AfterBuild phase only).
	OutputFormat string `json:"output_format"`
	// Config holds the plugin-specific settings from book.yaml plugins[n].config.
	Config map[string]any `json:"config"`
	// Metadata is a shared key-value store for inter-plugin and inter-phase communication.
	Metadata map[string]any `json:"metadata"`
}

// ExternalPluginResponse is the JSON body read from the external plugin process.
type ExternalPluginResponse struct {
	// Content is the modified text.  An empty string means "keep the original".
	Content string `json:"content"`
	// Stop instructs the manager to skip subsequent plugins for this phase.
	Stop bool `json:"stop"`
	// Error is an optional error message reported by the plugin.
	Error string `json:"error,omitempty"`
}

// ExternalPlugin represents a plugin implemented as an external executable.
// For each hook invocation it spawns a new process, sends a JSON request via
// stdin, and reads the JSON response from stdout.
type ExternalPlugin struct {
	// name is the plugin identifier used in logs and error messages.
	name string
	// version is the semantic version string.
	version string
	// description is a short human-readable description of the plugin.
	description string
	// execPath is the absolute path to the plugin executable.
	execPath string
	// pluginConfig holds the configuration values from book.yaml.
	pluginConfig map[string]any
	// hooks lists the hook phases this plugin handles.
	hooks []Phase
	// timeout is the maximum time allowed for a single hook invocation.
	timeout time.Duration
}

// NewExternalPlugin creates a new ExternalPlugin.
// execPath is resolved to an absolute path; relative paths are based on the
// current working directory at the time of the call.
func NewExternalPlugin(name, execPath string, pluginCfg map[string]any) (*ExternalPlugin, error) {
	resolvedPath, err := resolvePluginExecutablePath(execPath)
	if err != nil {
		return nil, err
	}

	if pluginCfg == nil {
		pluginCfg = make(map[string]any)
	}

	// Query the plugin for its metadata and supported hooks.
	// Falls back to safe defaults if the plugin does not support the flags.
	version, description := pluginMetaQueryFn(resolvedPath)

	return &ExternalPlugin{
		name:         name,
		version:      version,
		description:  description,
		execPath:     resolvedPath,
		pluginConfig: pluginCfg,
		hooks:        pluginHooksQueryFn(resolvedPath),
		timeout:      defaultPluginTimeout,
	}, nil
}

// resolvePluginExecutablePath resolves the configured plugin path to an existing
// executable. On Windows, it also tries common executable suffixes when the
// configured path omits one, so paths like "plugins/myplugin" can resolve to
// "plugins/myplugin.bat" or "plugins/myplugin.exe".
func resolvePluginExecutablePath(execPath string) (string, error) {
	absPath, err := filepath.Abs(execPath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve plugin path %q: %w", execPath, err)
	}

	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("plugin path %q is a directory, expected an executable", absPath)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
			return "", fmt.Errorf("plugin %q is not executable (missing execute permission)", absPath)
		}
		return absPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("plugin executable not found at %q: %w", absPath, err)
	}

	if runtime.GOOS != "windows" || filepath.Ext(absPath) != "" {
		return "", fmt.Errorf("plugin executable not found at %q: %w", absPath, os.ErrNotExist)
	}

	if resolved := resolveWindowsExecutableSuffix(absPath); resolved != "" {
		return resolved, nil
	}

	return "", fmt.Errorf("plugin executable not found at %q: %w", absPath, os.ErrNotExist)
}

// resolveWindowsExecutableSuffix tries executable suffixes commonly used on
// Windows when a plugin path is provided without an extension.
func resolveWindowsExecutableSuffix(absPath string) string {
	exts := windowsExecutableExtensions()
	for _, ext := range exts {
		candidate := absPath + ext
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return candidate
	}
	return ""
}

func windowsExecutableExtensions() []string {
	exts := strings.Split(os.Getenv("PATHEXT"), ";")
	result := make([]string, 0, len(exts))
	for _, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		result = append(result, ext)
	}
	if len(result) == 0 {
		return []string{".exe", ".bat", ".cmd", ".com"}
	}
	return result
}

// queryPluginMeta calls the plugin with --mdpress-info and parses the result.
// Expected stdout: {"version":"1.0.0","description":"..."}
// Returns safe defaults on any error.
func queryPluginMeta(execPath string) (version, description string) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginMetaQueryTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, execPath, "--mdpress-info").Output()
	if err != nil {
		slog.Debug("Failed to query plugin metadata", slog.String("path", execPath), slog.String("error", err.Error()))
		return "0.1.0", ""
	}

	var meta struct {
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &meta); err != nil {
		return "0.1.0", ""
	}
	if meta.Version == "" {
		meta.Version = "0.1.0"
	}
	return meta.Version, meta.Description
}

// queryPluginHooks calls the plugin with --mdpress-hooks and parses the result.
// Expected stdout: ["after_parse","after_build"]
// Returns all seven phases on any error so unknown plugins remain active everywhere.
func queryPluginHooks(execPath string) []Phase {
	ctx, cancel := context.WithTimeout(context.Background(), pluginHooksQueryTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, execPath, "--mdpress-hooks").Output()
	if err != nil {
		// Plugin does not support the flag; subscribe to all phases.
		return allPhases()
	}

	var hookNames []string
	if err := json.Unmarshal(bytes.TrimSpace(out), &hookNames); err != nil {
		return allPhases()
	}

	phases := make([]Phase, 0, len(hookNames))
	for _, n := range hookNames {
		phases = append(phases, Phase(strings.TrimSpace(n)))
	}
	return phases
}

// allPhases returns all seven defined hook phases.
func allPhases() []Phase {
	return []Phase{
		PhaseBeforeBuild, PhaseAfterParse, PhaseBeforeRender,
		PhaseAfterRender, PhaseAfterBuild, PhaseBeforeServe, PhaseAfterServe,
	}
}

// Name returns the plugin name.
func (p *ExternalPlugin) Name() string { return p.name }

// Version returns the plugin version.
func (p *ExternalPlugin) Version() string { return p.version }

// Description returns the plugin description.
func (p *ExternalPlugin) Description() string { return p.description }

// Init is a no-op for external plugins.
func (p *ExternalPlugin) Init(_ *config.BookConfig) error { return nil }

// Hooks returns the list of hook phases this plugin handles.
func (p *ExternalPlugin) Hooks() []Phase { return p.hooks }

// Execute runs the external plugin process for the given hook context.
// It writes a JSON request to stdin, reads the JSON response from stdout,
// and collects stderr for diagnostic purposes.
func (p *ExternalPlugin) Execute(hookCtx *HookContext) (*HookResult, error) {
	req := ExternalPluginRequest{
		Phase:        string(hookCtx.Phase),
		Content:      hookCtx.Content,
		ChapterIndex: hookCtx.ChapterIndex,
		ChapterFile:  hookCtx.ChapterFile,
		OutputPath:   hookCtx.OutputPath,
		OutputFormat: hookCtx.OutputFormat,
		Config:       p.pluginConfig,
		Metadata:     hookCtx.Metadata,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook request: %w", err)
	}

	ctx, cancel := context.WithTimeout(hookCtx.Context, p.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, p.execPath)
	cmd.Stdin = bytes.NewReader(reqJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("plugin exited with error: %w\nstderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("plugin exited with error: %w", err)
	}

	respBytes := bytes.TrimSpace(stdout.Bytes())
	if len(respBytes) == 0 {
		// No output from the plugin; keep the original content.
		return &HookResult{}, nil
	}

	var resp ExternalPluginResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %w (output: %s)", err, truncate(string(respBytes), 200))
	}

	// Treat plugin-reported errors as failures rather than silently storing them.
	if resp.Error != "" {
		return nil, fmt.Errorf("plugin %q reported error: %s", p.name, resp.Error)
	}

	return &HookResult{
		Content: resp.Content,
		Stop:    resp.Stop,
	}, nil
}

// Cleanup is a no-op for external plugins.
func (p *ExternalPlugin) Cleanup() error { return nil }

// truncate returns s truncated to at most maxLen bytes, with an ellipsis appended.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
