// Package plugin - loader.go loads plugins declared in book.yaml and returns an
// initialized Manager ready for use throughout the build pipeline.
package plugin

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
)

// LoadPlugins reads the plugin declarations from cfg.Plugins, creates an
// ExternalPlugin for each entry, registers them with a new Manager, and
// initializes the whole set.  Returns an empty (no-op) Manager when no plugins
// are configured.  Plugins are registered and executed in declaration order.
func LoadPlugins(cfg *config.BookConfig) (*Manager, error) {
	mgr := NewManager()

	if cfg == nil || len(cfg.Plugins) == 0 {
		return mgr, nil
	}

	for _, pc := range cfg.Plugins {
		if pc.Name == "" {
			return nil, errors.New("plugin config is missing the required 'name' field")
		}
		if pc.Path == "" {
			return nil, fmt.Errorf("plugin %q is missing the required 'path' field", pc.Name)
		}

		// Resolve the path relative to the directory that contains book.yaml.
		resolvedPath := cfg.ResolvePath(pc.Path)

		// Reject relative plugin paths that resolve outside the project directory.
		// Only enforce containment for relative paths (absolute paths are explicit).
		if !filepath.IsAbs(pc.Path) {
			absPlugin, absErr := filepath.Abs(resolvedPath)
			if absErr == nil {
				absBase, baseErr := filepath.Abs(cfg.ResolvePath("."))
				if baseErr == nil && !strings.HasPrefix(absPlugin, absBase+string(filepath.Separator)) && absPlugin != absBase {
					return nil, fmt.Errorf("plugin %q path resolves outside project directory: %s", pc.Name, resolvedPath)
				}
			}
		}

		ep, err := NewExternalPlugin(pc.Name, resolvedPath, pc.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %q: %w", pc.Name, err)
		}

		mgr.Register(ep)
	}

	if err := mgr.InitAll(cfg); err != nil {
		return nil, fmt.Errorf("plugin initialisation failed: %w", err)
	}

	return mgr, nil
}

// MustLoadPlugins is like LoadPlugins but never fails the build.  Any loading
// error is passed to warnFn (if non-nil) and an empty Manager is returned.
func MustLoadPlugins(cfg *config.BookConfig, warnFn func(msg string)) *Manager {
	mgr, err := LoadPlugins(cfg)
	if err != nil {
		if warnFn != nil {
			warnFn(fmt.Sprintf("plugin loading failed (continuing without plugins): %v", err))
		}
		return NewManager()
	}
	return mgr
}
