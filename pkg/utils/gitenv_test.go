package utils

import (
	"strings"
	"testing"
)

// TestSanitizedGitEnvDropsOverrides verifies that the repository-selecting git
// variables are removed while the rest of the environment survives.
func TestSanitizedGitEnvDropsOverrides(t *testing.T) {
	t.Setenv("GIT_DIR", "/somewhere/.git")
	t.Setenv("GIT_WORK_TREE", "/somewhere")
	t.Setenv("GIT_INDEX_FILE", "/somewhere/.git/index")
	t.Setenv("MDPRESS_KEEP_ME", "kept")

	var sawKeeper bool
	for _, kv := range SanitizedGitEnv() {
		name, _, _ := strings.Cut(kv, "=")
		for _, dropped := range gitEnvOverrides {
			if name == dropped {
				t.Errorf("%s survived sanitization", name)
			}
		}
		if kv == "MDPRESS_KEEP_ME=kept" {
			sawKeeper = true
		}
	}
	if !sawKeeper {
		t.Error("unrelated variable MDPRESS_KEEP_ME was dropped")
	}
}
