package plugin

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The plugin's own lifetime in these tests. It has to be comfortably longer
// than the hook timeout plus pluginOutputDrainDelay so that a build which
// waits for the orphan instead of the timeout is unmistakable.
const orphanLifetime = "20"

// TestExternalPlugin_Execute_TimeoutIgnoredOrphanedChild pins the fix for a
// hook that the documented timeout did not bound: exec.CommandContext SIGKILLs
// only the plugin, and a process it left behind kept the inherited output pipe
// open, so Wait blocked for as long as the orphan lived. `sleep 90` in a plugin
// froze the build for 90 s against a 30 s cap; a plugin that daemonized froze
// it forever.
func TestExternalPlugin_Execute_TimeoutIgnoredOrphanedChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the script relies on Bourne shell child-process semantics")
	}
	stubMetaQueries(t)
	dir := t.TempDir()
	// The normal shape of a shell plugin: the work runs as a child process.
	p := writeScript(t, dir, "hanger", "sleep "+orphanLifetime+"\n")

	ep, err := NewExternalPlugin("hanger", p, nil)
	if err != nil {
		t.Fatal(err)
	}
	ep.timeout = 300 * time.Millisecond

	start := time.Now()
	_, err = ep.Execute(&HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterBuild,
		Metadata: map[string]any{},
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a plugin that never exits should fail the hook, not succeed")
	}
	// "signal: killed" reads like a crash or an OOM; it names neither the cap
	// the author has to raise nor the sleep they have to remove.
	if !strings.Contains(err.Error(), "timed out after") {
		t.Errorf("error should name the timeout, got: %v", err)
	}
	if limit := 10 * time.Second; elapsed > limit {
		t.Errorf("hook took %s: the timeout is bounded by the orphaned child, not by ep.timeout", elapsed)
	}
}

// TestExternalPlugin_Execute_SucceedsWithBackgroundedChild covers the other
// half: a plugin that backgrounds work and exits 0 (a notification, an upload)
// used to hold the build hostage for the whole life of that background job and
// then report success. The plugin did its part, so the hook still succeeds —
// it just no longer waits.
func TestExternalPlugin_Execute_SucceedsWithBackgroundedChild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the script relies on Bourne shell child-process semantics")
	}
	stubMetaQueries(t)
	dir := t.TempDir()
	p := writeScript(t, dir, "backgrounder", "sleep "+orphanLifetime+" &\necho '{\"content\":\"replaced\"}'\n")

	ep, err := NewExternalPlugin("backgrounder", p, nil)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	res, err := ep.Execute(&HookContext{
		Context:  context.Background(),
		Phase:    PhaseAfterBuild,
		Metadata: map[string]any{},
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("a plugin that exits 0 should succeed even with a background child: %v", err)
	}
	if res.Content != "replaced" {
		t.Errorf("plugin response was lost: %+v", res)
	}
	if limit := 10 * time.Second; elapsed > limit {
		t.Errorf("hook took %s: the build waited for the backgrounded child", elapsed)
	}
}
