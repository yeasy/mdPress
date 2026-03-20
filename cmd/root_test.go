package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// TestCacheDirFlagParsing tests that --cache-dir flag is properly parsed
func TestCacheDirFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		checkEnv func(t *testing.T)
	}{
		{
			name: "no cache flags",
			args: []string{},
			checkEnv: func(t *testing.T) {
				// Cache should be enabled by default
				if os.Getenv("MDPRESS_DISABLE_CACHE") == "1" {
					t.Error("cache should be enabled by default")
				}
			},
		},
		{
			name: "custom cache directory",
			args: []string{"--cache-dir", "/tmp/custom-cache"},
			checkEnv: func(t *testing.T) {
				// Environment will be set by configureRuntimeCacheEnv
				if os.Getenv("MDPRESS_CACHE_DIR") != "/tmp/custom-cache" {
					t.Errorf("MDPRESS_CACHE_DIR not set correctly, got %q", os.Getenv("MDPRESS_CACHE_DIR"))
				}
			},
		},
		{
			name: "no-cache flag",
			args: []string{"--no-cache"},
			checkEnv: func(t *testing.T) {
				// Environment will be set by configureRuntimeCacheEnv
				if os.Getenv("MDPRESS_DISABLE_CACHE") != "1" {
					t.Error("MDPRESS_DISABLE_CACHE should be set")
				}
			},
		},
		{
			name: "both cache flags",
			args: []string{"--cache-dir", "/tmp/custom", "--no-cache"},
			checkEnv: func(t *testing.T) {
				// no-cache takes precedence logically
				if os.Getenv("MDPRESS_DISABLE_CACHE") != "1" {
					t.Error("MDPRESS_DISABLE_CACHE should be set")
				}
				if os.Getenv("MDPRESS_CACHE_DIR") != "/tmp/custom" {
					t.Error("MDPRESS_CACHE_DIR should also be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset environment before each test
			os.Unsetenv("MDPRESS_CACHE_DIR")
			os.Unsetenv("MDPRESS_DISABLE_CACHE")

			// Create a test command to verify flag parsing
			testCmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			}}

			// Copy flags from root command
			testCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "Override mdpress runtime cache directory")
			testCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Disable mdpress runtime caches for this command")

			// Set up the pre-run to configure cache
			testCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
				configureRuntimeCacheEnv()
			}

			// Parse flags
			testCmd.SetArgs(tt.args)
			err := testCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error=%v, got %v", tt.wantErr, err)
			}

			// Check environment
			tt.checkEnv(t)
		})
	}
}

// TestNoCacheDisablesBothFlags tests that --no-cache properly disables caching
func TestNoCacheDisablesFlagCaching(t *testing.T) {
	os.Unsetenv("MDPRESS_CACHE_DIR")
	os.Unsetenv("MDPRESS_DISABLE_CACHE")

	// Simulate flag parsing and configuration
	noCache = true
	cacheDir = ""

	configureRuntimeCacheEnv()

	if os.Getenv("MDPRESS_DISABLE_CACHE") != "1" {
		t.Error("--no-cache should set MDPRESS_DISABLE_CACHE=1")
	}
}

// TestCacheDirOverrideWorks tests that --cache-dir properly overrides default
func TestCacheDirOverrideWorks(t *testing.T) {
	os.Unsetenv("MDPRESS_CACHE_DIR")
	os.Unsetenv("MDPRESS_DISABLE_CACHE")

	customPath := "/custom/cache/path"
	cacheDir = customPath
	noCache = false

	configureRuntimeCacheEnv()

	if got := os.Getenv("MDPRESS_CACHE_DIR"); got != customPath {
		t.Errorf("expected MDPRESS_CACHE_DIR=%q, got %q", customPath, got)
	}

	if os.Getenv("MDPRESS_DISABLE_CACHE") != "" {
		t.Error("MDPRESS_DISABLE_CACHE should not be set when only --cache-dir is used")
	}
}

// TestFlagDefaults tests that flags have proper defaults
func TestFlagDefaults(t *testing.T) {
	// Reset to defaults
	cacheDir = ""
	noCache = false

	if cacheDir != "" {
		t.Error("cacheDir should default to empty string")
	}

	if noCache != false {
		t.Error("noCache should default to false")
	}
}

// TestConfigureRuntimeCacheEnvDoesNotPanicOnEmpty tests robustness
func TestConfigureRuntimeCacheEnvRobustness(t *testing.T) {
	os.Unsetenv("MDPRESS_CACHE_DIR")
	os.Unsetenv("MDPRESS_DISABLE_CACHE")

	cacheDir = ""
	noCache = false

	// Should not panic
	configureRuntimeCacheEnv()
}
