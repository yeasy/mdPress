package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// captureVersionStdout runs the version command's Run func and returns stdout.
func captureVersionStdout(t *testing.T) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	versionCmd.Run(versionCmd, nil)

	w.Close()
	os.Stdout = oldStdout

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

// TestDefaultVersionConstant guards against stale hardcoded versions.
func TestDefaultVersionConstant(t *testing.T) {
	if defaultVersion != "0.7.15" {
		t.Errorf("defaultVersion = %q, want 0.7.15", defaultVersion)
	}
}
