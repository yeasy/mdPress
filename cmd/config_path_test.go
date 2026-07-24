package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigFlag sets the package-level --config value for one test.
func withConfigFlag(t *testing.T, value string) {
	t.Helper()
	prev := cfgFile
	cfgFile = value
	t.Cleanup(func() { cfgFile = prev })
}

func TestResolveConfigPath(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "release.yaml")
	if err := os.WriteFile(explicit, []byte("book:\n  title: T\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "book.yaml"), []byte("book:\n  title: W\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("default falls back to the source directory and allows discovery", func(t *testing.T) {
		withConfigFlag(t, defaultConfigName)
		path, allow := resolveConfigPath(dir)
		if path != filepath.Join(dir, defaultConfigName) {
			t.Errorf("path = %q", path)
		}
		if !allow {
			t.Error("discovery should be allowed for the implicit default")
		}
	})

	t.Run("explicit config wins over the source directory", func(t *testing.T) {
		withConfigFlag(t, explicit)
		path, allow := resolveConfigPath(dir)
		if path != explicit {
			t.Errorf("explicit --config was discarded: got %q, want %q", path, explicit)
		}
		if allow {
			t.Error("an explicit --config must not fall back to discovery")
		}
	})

	t.Run("relative explicit config resolves against the source directory", func(t *testing.T) {
		withConfigFlag(t, "release.yaml")
		path, _ := resolveConfigPath(dir)
		if path != explicit {
			t.Errorf("path = %q, want %q", path, explicit)
		}
	})

	t.Run("missing explicit config is reported, not silently discovered", func(t *testing.T) {
		withConfigFlag(t, "nope.yaml")
		path, allow := resolveConfigPath(dir)
		if allow {
			t.Error("a missing explicit --config must not fall back to discovery")
		}
		if err := errExplicitConfigMissing(path); err == nil {
			t.Error("expected an error for the missing explicit config")
		}
	})
}

// TestExplicitDefaultNameConfigIsHonored pins the bug where the flag's default
// value doubled as "not set": `--config book.yaml ./docs` was indistinguishable
// from omitting the flag, so it took the implicit branch and quietly built
// ./docs's auto-discovered book instead of the book.yaml the user pointed at.
func TestExplicitDefaultNameConfigIsHonored(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, defaultConfigName), []byte("book:\n  title: Root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	// Go through pflag rather than assigning cfgFile, so the "was the flag on
	// the command line?" state under test is the real one.
	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("--config flag is not registered on rootCmd")
	}
	prevValue, prevChanged := cfgFile, flag.Changed
	t.Cleanup(func() {
		cfgFile = prevValue
		flag.Changed = prevChanged
	})
	if err := rootCmd.PersistentFlags().Set("config", defaultConfigName); err != nil {
		t.Fatal(err)
	}

	path, allow := resolveConfigPath(source)
	if path != defaultConfigName {
		t.Errorf("path = %q, want the cwd's %q that --config named", path, defaultConfigName)
	}
	if allow {
		t.Error("an explicit --config must not fall back to auto-discovering the source directory")
	}
}
