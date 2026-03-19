// progress.go provides progress output for build steps.
// It supports step counting, status markers, and colored output.
package utils

import (
	"fmt"
	"time"
)

// ProgressTracker tracks build progress.
type ProgressTracker struct {
	total   int       // Total number of steps.
	current int       // Current step index.
	start   time.Time // Build start time.
}

// NewProgressTracker creates a new progress tracker.
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total: total,
		start: time.Now(),
	}
}

// Start marks the beginning of a new step and prints the pending state.
func (p *ProgressTracker) Start(description string) {
	p.current++
	prefix := fmt.Sprintf("[%d/%d]", p.current, p.total)

	if colorEnabled {
		fmt.Printf("  %s%s %s%s ...%s", colorCyan, prefix, colorReset, description, "")
	} else {
		fmt.Printf("  %s %s ...", prefix, description)
	}
}

// Done marks the current step as completed.
func (p *ProgressTracker) Done() {
	if colorEnabled {
		fmt.Printf(" %s✓%s\n", colorGreen, colorReset)
	} else {
		fmt.Println(" ✓")
	}
}

// Fail marks the current step as failed.
func (p *ProgressTracker) Fail() {
	if colorEnabled {
		fmt.Printf(" %s✗%s\n", colorRed, colorReset)
	} else {
		fmt.Println(" ✗")
	}
}

// Skip marks the current step as skipped.
func (p *ProgressTracker) Skip(reason string) {
	if colorEnabled {
		fmt.Printf(" %s⊘ %s%s\n", colorYellow, reason, colorReset)
	} else {
		fmt.Printf(" ⊘ %s\n", reason)
	}
}

// DoneWithDetail marks the current step as completed with extra detail.
func (p *ProgressTracker) DoneWithDetail(detail string) {
	if colorEnabled {
		fmt.Printf(" %s✓%s %s%s%s\n", colorGreen, colorReset, colorDim, detail, colorReset)
	} else {
		fmt.Printf(" ✓ %s\n", detail)
	}
}

// Finish prints the build-complete summary.
func (p *ProgressTracker) Finish() {
	elapsed := time.Since(p.start).Round(time.Millisecond)
	fmt.Println()
	if colorEnabled {
		fmt.Printf("  %s%s✅ Build completed%s (elapsed %s)\n", colorBold, colorGreen, colorReset, elapsed)
	} else {
		fmt.Printf("  ✅ Build completed (elapsed %s)\n", elapsed)
	}
}

// FinishWithError prints the build-failed summary.
func (p *ProgressTracker) FinishWithError(err error) {
	elapsed := time.Since(p.start).Round(time.Millisecond)
	fmt.Println()
	if colorEnabled {
		fmt.Printf("  %s%s❌ Build failed%s (elapsed %s): %v\n", colorBold, colorRed, colorReset, elapsed, err)
	} else {
		fmt.Printf("  ❌ Build failed (elapsed %s): %v\n", elapsed, err)
	}
}
