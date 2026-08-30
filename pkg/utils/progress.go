// progress.go provides progress output for build steps.
// It supports step counting, status markers, and colored output.
package utils

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressTracker tracks build progress.
type ProgressTracker struct {
	mu      sync.Mutex
	total   int       // Total number of steps.
	current int       // Current step index.
	start   time.Time // Build start time.
	silent  bool      // Suppress step output (quiet mode).
	pending string    // Description of the step whose line is still open.
}

// pendingLine coordinates the one place two writers share a terminal line: a
// step prints "[2/5] Parsing ..." with no newline and finishes it later, and
// a log record emitted in between used to land in the middle of it. The log
// handler declares the interruption; the tracker then reprints the whole step
// line instead of appending a stray "✓" to the end of a warning.
var pendingLine struct {
	open        atomic.Bool
	interrupted atomic.Bool
}

// InterruptPendingLine reports whether a progress line is currently open, and
// records the interruption so its owner reprints the line. The caller is
// expected to start its own output on a fresh line when this returns true.
func InterruptPendingLine() bool {
	if !pendingLine.open.Load() {
		return false
	}
	pendingLine.interrupted.Store(true)
	return true
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total: total,
		start: time.Now(),
	}
}

// SetSilent suppresses the per-step output. --quiet documents itself as "only
// output errors", but the five-step progress and the completion banner were
// printed regardless, so piping a quiet build still produced a screenful.
// The final "Generated <format> → <path>" summary is printed by the caller and
// deliberately survives quiet mode: it is the one line a script wants.
func (p *ProgressTracker) SetSilent(silent bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.silent = silent
}

// quiet reports whether step output is suppressed.
func (p *ProgressTracker) quiet() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.silent
}

// Start marks the beginning of a new step and prints the pending state.
func (p *ProgressTracker) Start(description string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	if p.silent {
		return
	}
	prefix := fmt.Sprintf("[%d/%d]", p.current, p.total)
	p.pending = description
	pendingLine.open.Store(true)
	pendingLine.interrupted.Store(false)

	if colorEnabled.Load() {
		fmt.Printf("  %s%s %s%s ...%s", colorCyan, prefix, colorReset, description, "")
	} else {
		fmt.Printf("  %s %s ...", prefix, description)
	}
}

// closePending ends the open step line and reports whether a log record broke
// it; when one did, the caller reprints the whole "[n/m] description ..."
// prefix so the completion mark lands on a coherent line.
func (p *ProgressTracker) closePending() (reprint string, interrupted bool) {
	pendingLine.open.Store(false)
	if !pendingLine.interrupted.Swap(false) {
		return "", false
	}
	return fmt.Sprintf("[%d/%d]", p.current, p.total) + " " + p.pending, true
}

// reprintPrefix restores the step line after an interruption.
func (p *ProgressTracker) reprintPrefix(line string) {
	if colorEnabled.Load() {
		i := len(fmt.Sprintf("[%d/%d]", p.current, p.total))
		fmt.Printf("  %s%s %s%s ...", colorCyan, line[:i], colorReset, line[i+1:])
		return
	}
	fmt.Printf("  %s ...", line)
}

// Done marks the current step as completed.
func (p *ProgressTracker) Done() {
	if p.quiet() {
		return
	}
	if line, interrupted := p.closePending(); interrupted {
		p.reprintPrefix(line)
	}
	if colorEnabled.Load() {
		fmt.Printf(" %s✓%s\n", colorGreen, colorReset)
	} else {
		fmt.Println(" ✓")
	}
}

// Fail marks the current step as failed.
func (p *ProgressTracker) Fail() {
	if p.quiet() {
		return
	}
	if line, interrupted := p.closePending(); interrupted {
		p.reprintPrefix(line)
	}
	if colorEnabled.Load() {
		fmt.Printf(" %s✗%s\n", colorRed, colorReset)
	} else {
		fmt.Println(" ✗")
	}
}

// Skip marks the current step as skipped.
func (p *ProgressTracker) Skip(reason string) {
	if p.quiet() {
		return
	}
	if line, interrupted := p.closePending(); interrupted {
		p.reprintPrefix(line)
	}
	if colorEnabled.Load() {
		fmt.Printf(" %s⊘ %s%s\n", colorYellow, reason, colorReset)
	} else {
		fmt.Printf(" ⊘ %s\n", reason)
	}
}

// DoneWithDetail marks the current step as completed with extra detail.
func (p *ProgressTracker) DoneWithDetail(detail string) {
	if p.quiet() {
		return
	}
	if line, interrupted := p.closePending(); interrupted {
		p.reprintPrefix(line)
	}
	if colorEnabled.Load() {
		fmt.Printf(" %s✓%s %s%s%s\n", colorGreen, colorReset, colorDim, detail, colorReset)
	} else {
		fmt.Printf(" ✓ %s\n", detail)
	}
}

// Finish prints the build-complete summary.
func (p *ProgressTracker) Finish() {
	if p.quiet() {
		return
	}
	elapsed := time.Since(p.start).Round(time.Millisecond)
	fmt.Println()
	if colorEnabled.Load() {
		fmt.Printf("  %s%s✅ Build completed%s (elapsed %s)\n", colorBold, colorGreen, colorReset, elapsed)
	} else {
		fmt.Printf("  ✅ Build completed (elapsed %s)\n", elapsed)
	}
}

// FinishWithError prints the build-failed summary.
func (p *ProgressTracker) FinishWithError(err error) {
	elapsed := time.Since(p.start).Round(time.Millisecond)
	fmt.Println()
	if colorEnabled.Load() {
		fmt.Printf("  %s%s❌ Build failed%s (elapsed %s): %v\n", colorBold, colorRed, colorReset, elapsed, err)
	} else {
		fmt.Printf("  ❌ Build failed (elapsed %s): %v\n", elapsed, err)
	}
}
