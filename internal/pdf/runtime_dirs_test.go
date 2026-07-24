package pdf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mkdirAged creates dir under base and back-dates it by age.
func mkdirAged(t *testing.T, base, name string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(filepath.Join(dir, "user-data"), 0o755); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(dir, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestPruneStaleChromiumRuntimeDirs covers the sweep in isolation: only run-*
// directories old enough to belong to a dead build go away.
func TestPruneStaleChromiumRuntimeDirs(t *testing.T) {
	base := t.TempDir()
	stale := mkdirAged(t, base, "run-dead", staleChromiumRuntimeAge+time.Hour)
	recent := mkdirAged(t, base, "run-live", time.Minute)
	unrelated := mkdirAged(t, base, "parsed-chapters", 30*24*time.Hour)

	pruneStaleChromiumRuntimeDirs(base, time.Now())

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale run dir should have been removed, stat err = %v", err)
	}
	for _, keep := range []string{recent, unrelated} {
		if _, err := os.Stat(keep); err != nil {
			t.Errorf("%s should have been kept: %v", filepath.Base(keep), err)
		}
	}
}

// TestPrepareChromiumRuntimeDirsPrunesStaleRuns is the regression guard: a PDF
// build has to collect the runtime directories left behind when an earlier build
// was killed hard (OOM, kill -9, canceled CI job). Nothing else ever did, so they
// accumulated for the life of the machine.
func TestPrepareChromiumRuntimeDirsPrunesStaleRuns(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("MDPRESS_CACHE_DIR", cacheDir)
	t.Setenv("MDPRESS_DISABLE_CACHE", "")

	runtimeBase := filepath.Join(cacheDir, "chrome-runtime")
	orphan := mkdirAged(t, runtimeBase, "run-orphan", staleChromiumRuntimeAge+time.Hour)

	dirs, err := prepareChromiumRuntimeDirs()
	if err != nil {
		t.Fatalf("prepareChromiumRuntimeDirs() = %v", err)
	}
	defer dirs.cleanup()

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("an abandoned run dir should not survive the next build, stat err = %v", err)
	}
	if _, err := os.Stat(dirs.userData); err != nil {
		t.Errorf("the new run's user-data dir should exist: %v", err)
	}
}
