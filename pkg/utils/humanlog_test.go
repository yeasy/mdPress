package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureHandler runs fn with a HumanLogHandler writing to a temp file and
// returns what it wrote. Color is forced off so assertions see plain text.
func captureHandler(t *testing.T, level slog.Level, fn func(*slog.Logger)) string {
	t.Helper()
	prev := colorEnabled.Load()
	colorEnabled.Store(false)
	t.Cleanup(func() { colorEnabled.Store(prev) })

	f, err := os.Create(filepath.Join(t.TempDir(), "log"))
	if err != nil {
		t.Fatal(err)
	}
	fn(slog.New(NewHumanLogHandler(f, level)))
	f.Close()
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestHumanLogHandlerShapesTheSentence pins the folding of attributes into a
// readable line: one attribute names the subject, one carries the substance,
// the rest become a parenthetical — instead of the raw key=value TextHandler
// output that used to splice into the styled progress UI.
func TestHumanLogHandlerShapesTheSentence(t *testing.T) {
	out := captureHandler(t, slog.LevelWarn, func(l *slog.Logger) {
		l.Warn("document issue",
			slog.String("rule", "unresolved-markdown-link"),
			slog.String("file", "intro.md"),
			slog.String("detail", "link target is outside the build graph: ./nothere.md"))
	})
	want := "  ⚠ intro.md: document issue: link target is outside the build graph: ./nothere.md (rule: unresolved-markdown-link)\n"
	if out != want {
		t.Errorf("line = %q\nwant  %q", out, want)
	}
}

func TestHumanLogHandlerLevelsAndMarks(t *testing.T) {
	out := captureHandler(t, slog.LevelWarn, func(l *slog.Logger) {
		l.Debug("hidden")
		l.Info("hidden too")
		l.Warn("careful")
		l.Error("broken", slog.String("error", "it failed"))
	})
	if strings.Contains(out, "hidden") {
		t.Errorf("below-level records leaked: %q", out)
	}
	if !strings.Contains(out, "⚠ careful") {
		t.Errorf("warning mark missing: %q", out)
	}
	if !strings.Contains(out, "✗ broken: it failed") {
		t.Errorf("error mark or body missing: %q", out)
	}
}

func TestHumanLogHandlerWithAttrs(t *testing.T) {
	out := captureHandler(t, slog.LevelWarn, func(l *slog.Logger) {
		l.With(slog.String("file", "ch1.md")).Warn("something odd")
	})
	if !strings.Contains(out, "ch1.md: something odd") {
		t.Errorf("logger-level attrs should fold in: %q", out)
	}
}

// TestProgressReprintsInterruptedLine pins the coordination between a pending
// "[n/m] step ..." line and a log record landing while it is open: the record
// declares the interruption and the tracker reprints the whole step line, so
// the completion mark never dangles at the end of a warning.
func TestProgressReprintsInterruptedLine(t *testing.T) {
	tr := NewProgressTracker(3)
	tr.Start("Parsing chapters")
	if !InterruptPendingLine() {
		t.Fatal("a started step should report an open line")
	}
	line, interrupted := tr.closePending()
	if !interrupted {
		t.Fatal("closePending should observe the interruption")
	}
	if line != "[1/3] Parsing chapters" {
		t.Errorf("reprint line = %q", line)
	}
	// Once closed, nothing is open and nothing left to reprint.
	if InterruptPendingLine() {
		t.Error("no line should be open after closePending")
	}
	tr.Start("Next step")
	if l, i := tr.closePending(); i || l != "" {
		t.Errorf("uninterrupted step should not ask for a reprint (line=%q interrupted=%v)", l, i)
	}
}
