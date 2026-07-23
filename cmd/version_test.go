package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// captureVersionStdout runs the version command and returns stdout.
// The command moved from Run to RunE when --json was added, since encoding can
// fail and a failure must not look like success.
func captureVersionStdout(t *testing.T) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	runErr := versionCmd.RunE(versionCmd, nil)

	w.Close()
	os.Stdout = oldStdout
	if runErr != nil {
		t.Fatalf("version command failed: %v", runErr)
	}

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read captured output: %v", readErr)
	}
	return buf.String()
}

// TestVersionCommandOutput verifies the version command prints the version and,
// when set, the commit and build time.
func TestVersionCommandOutput(t *testing.T) {
	origVersion, origCommit, origBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = origVersion, origCommit, origBuildTime })

	Version = "9.9.9"
	Commit = "abcdef1"
	BuildTime = "2026-07-05T00:00:00Z"

	out := captureVersionStdout(t)
	if !strings.Contains(out, "mdpress version 9.9.9") {
		t.Errorf("version output missing version line: %q", out)
	}
	if !strings.Contains(out, "Commit abcdef1") {
		t.Errorf("version output missing commit line: %q", out)
	}
	if !strings.Contains(out, "Built at 2026-07-05T00:00:00Z") {
		t.Errorf("version output missing build time line: %q", out)
	}
}

// TestVersionCommandOmitsEmptyFields verifies commit/build time lines are
// suppressed when those values are empty/unknown (backward-compatible output).
func TestVersionCommandOmitsEmptyFields(t *testing.T) {
	origVersion, origCommit, origBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = origVersion, origCommit, origBuildTime })

	Version = "1.2.3"
	Commit = ""
	BuildTime = "unknown"

	out := captureVersionStdout(t)
	if !strings.Contains(out, "mdpress version 1.2.3") {
		t.Errorf("version output missing version line: %q", out)
	}
	if strings.Contains(out, "Commit") {
		t.Errorf("version output should omit commit line when empty: %q", out)
	}
	if strings.Contains(out, "Built at") {
		t.Errorf("version output should omit build time line when unknown: %q", out)
	}
}

// TestVersionJSONIsMachineReadable covers `mdpress version --json`, which used
// to fail with "unknown flag": a CI job that wanted the version had to scrape
// the human-readable text.
func TestVersionJSONIsMachineReadable(t *testing.T) {
	origVersion, origCommit, origBuildTime, origJSON := Version, Commit, BuildTime, versionJSON
	t.Cleanup(func() {
		Version, Commit, BuildTime, versionJSON = origVersion, origCommit, origBuildTime, origJSON
	})

	Version = "9.9.9"
	Commit = "abcdef1"
	BuildTime = "2026-07-05T00:00:00Z"
	versionJSON = true

	out := captureVersionStdout(t)

	var info map[string]any
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("version --json produced invalid JSON (%v): %q", err, out)
	}
	for key, want := range map[string]string{
		"version":  "9.9.9",
		"commit":   "abcdef1",
		"built_at": "2026-07-05T00:00:00Z",
	} {
		if got, ok := info[key]; !ok {
			t.Errorf("version --json is missing the %q key: %v", key, info)
		} else if got != want {
			t.Errorf("version --json %q = %v, want %v", key, got, want)
		}
	}
	if info["go_version"] == "" || info["os"] == "" || info["arch"] == "" {
		t.Errorf("version --json should report the build platform: %v", info)
	}
}

// TestVersionJSONKeepsUnstampedFieldsPresent verifies a source build still
// emits every key, so scripts do not have to special-case a missing one.
func TestVersionJSONKeepsUnstampedFieldsPresent(t *testing.T) {
	origVersion, origCommit, origBuildTime, origJSON := Version, Commit, BuildTime, versionJSON
	t.Cleanup(func() {
		Version, Commit, BuildTime, versionJSON = origVersion, origCommit, origBuildTime, origJSON
	})

	Version = "1.2.3"
	Commit = ""
	BuildTime = "unknown"
	versionJSON = true

	var info map[string]any
	if err := json.Unmarshal([]byte(captureVersionStdout(t)), &info); err != nil {
		t.Fatalf("version --json produced invalid JSON: %v", err)
	}
	for _, key := range []string{"version", "commit", "built_at"} {
		if _, ok := info[key]; !ok {
			t.Errorf("version --json dropped the %q key for an unstamped build: %v", key, info)
		}
	}
	if info["commit"] != "" || info["built_at"] != "" {
		t.Errorf("unstamped build should report empty commit/built_at, got %v", info)
	}
}

// TestVersionJSONFlagIsRegistered guards the flag itself: the command used to
// reject --json outright.
func TestVersionJSONFlagIsRegistered(t *testing.T) {
	if versionCmd.Flags().Lookup("json") == nil {
		t.Error("version command should accept --json")
	}
}

// TestDefaultVersionConstant guards against stale hardcoded versions.
func TestDefaultVersionConstant(t *testing.T) {
	if defaultVersion != "0.7.15" {
		t.Errorf("defaultVersion = %q, want 0.7.15", defaultVersion)
	}
}
