package utils

import (
	"os"
	"slices"
	"strings"
)

// gitEnvOverrides lists the git environment variables that make git resolve
// against a repository chosen by the environment rather than by the command's
// own "-C <dir>" flag or working directory. A caller such as a git pre-commit
// hook exports GIT_DIR, which silently overrides both.
var gitEnvOverrides = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_COMMON_DIR",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

// SanitizedGitEnv returns the process environment without the variables that
// would redirect git away from the directory the command selects itself.
// Use it as the base environment for any git subprocess.
func SanitizedGitEnv() []string {
	env := os.Environ()
	out := env[:0:0]
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(gitEnvOverrides, name) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
