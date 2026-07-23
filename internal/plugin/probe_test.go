package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeProbeScript writes an executable POSIX shell script and returns its
// path. It is separate from writeScript because these fixtures need to
// dispatch on the metadata flag rather than echo a fixed body.
func writeProbeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write script %q: %v", name, err)
	}
	return path
}

// TestProbeRejectsNonPlugin covers finding 1018: `mdpress doctor` reported
// "all plugins are valid" for programs that could not answer a single mdpress
// query, and the very next build warned about each of them.
func TestProbeRejectsNonPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixtures are POSIX shell scripts")
	}
	dir := t.TempDir()
	path := writeProbeScript(t, dir, "hello.sh", "#!/bin/sh\necho hello\n")

	result, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe of an existing executable should not error: %v", err)
	}
	if result.SpeaksProtocol {
		t.Error("a program that answers neither metadata query does not speak the plugin protocol")
	}
	if result.InfoErr == nil {
		t.Error("Probe should report why --mdpress-info was unusable")
	}
	if result.HooksErr == nil {
		t.Error("Probe should report why --mdpress-hooks was unusable")
	}
}

// TestProbeAcceptsRealPlugin is the positive case: a plugin that answers the
// handshake must be recognized, with its metadata carried through.
func TestProbeAcceptsRealPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixtures are POSIX shell scripts")
	}
	dir := t.TempDir()
	path := writeProbeScript(t, dir, "good.sh", `#!/bin/sh
case "$1" in
  --mdpress-info) echo '{"version":"1.2.3","description":"demo"}' ;;
  --mdpress-hooks) echo '["after_parse","after_build"]' ;;
esac
`)

	result, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if !result.SpeaksProtocol {
		t.Fatalf("a plugin answering both queries should be recognized: info=%v hooks=%v", result.InfoErr, result.HooksErr)
	}
	if result.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", result.Version)
	}
	if result.Description != "demo" {
		t.Errorf("Description = %q, want demo", result.Description)
	}
	if len(result.Hooks) != 2 || result.Hooks[0] != PhaseAfterParse {
		t.Errorf("Hooks = %v, want [after_parse after_build]", result.Hooks)
	}
}

// TestProbeAcceptsPartialProtocol keeps older plugins working: answering only
// --mdpress-info is enough to be a plugin, since --mdpress-hooks has a
// documented all-phases fallback.
func TestProbeAcceptsPartialProtocol(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixtures are POSIX shell scripts")
	}
	dir := t.TempDir()
	path := writeProbeScript(t, dir, "info-only.sh", `#!/bin/sh
case "$1" in
  --mdpress-info) echo '{"version":"0.9.0"}' ;;
  *) exit 1 ;;
esac
`)

	result, err := Probe(path)
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if !result.SpeaksProtocol {
		t.Error("answering --mdpress-info alone is enough to be a plugin")
	}
	if result.HooksErr == nil {
		t.Error("Probe should still record that --mdpress-hooks failed")
	}
}

// TestProbeMissingExecutable reports a path that resolves to nothing.
func TestProbeMissingExecutable(t *testing.T) {
	if _, err := Probe(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("probing a nonexistent path should fail")
	}
}
