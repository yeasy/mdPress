package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/yeasy/mdpress/pkg/utils"
)

// defaultConfigName is the config file mdpress looks for when --config is not
// given. It is also the flag's default value, which is how an explicit
// --config is distinguished from the implicit one.
const defaultConfigName = "book.yaml"

// configExplicitlySet reports whether the user passed --config. Comparing
// against the flag default is enough here: passing --config book.yaml is
// indistinguishable from omitting it, and means the same thing either way.
func configExplicitlySet() bool {
	return cfgFile != "" && cfgFile != defaultConfigName
}

// resolveConfigPath decides which config file a command should load.
//
// An explicit --config always wins, including when a source directory or URL
// was also given: previously the source path silently replaced it with
// <source>/book.yaml, so `mdpress build --config release.yaml ./docs` built
// the wrong book and still exited 0. A relative --config is resolved against
// the current directory first (what the user typed), then against the source
// directory.
//
// Returns the path to load, and whether the caller may fall back to
// zero-config discovery when that path does not exist. Falling back is only
// allowed for the implicit default — an explicit --config that does not
// resolve is an error, not a hint to guess.
func resolveConfigPath(workDir string) (path string, allowDiscovery bool) {
	if !configExplicitlySet() {
		if workDir == "" {
			return cfgFile, true
		}
		return filepath.Join(workDir, defaultConfigName), true
	}

	if filepath.IsAbs(cfgFile) || utils.FileExists(cfgFile) {
		return cfgFile, false
	}
	if workDir != "" {
		if candidate := filepath.Join(workDir, cfgFile); utils.FileExists(candidate) {
			return candidate, false
		}
	}
	// Report the path the user typed, so the error names something they recognize.
	return cfgFile, false
}

// errExplicitConfigMissing builds the error for an explicit --config that does
// not exist. Silently discovering a different project instead would let a
// mistyped path produce a successful build of the wrong book.
func errExplicitConfigMissing(path string) error {
	return fmt.Errorf("config file not found: %s (--config was given explicitly; "+
		"remove it to auto-discover, or correct the path)", path)
}
