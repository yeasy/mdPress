package pdf

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/chromedp/chromedp"
)

// Chrome is an external process tree, not a goroutine: nothing in the Go
// runtime tears it down when mdpress exits. The normal cancel path is safe —
// chromedp starts the browser with exec.CommandContext, so canceling the render
// context makes os/exec SIGKILL it — but os.Exit runs no deferred cancels at
// all. Without this registry, a user who pressed Ctrl+C a second time to give
// up on a slow render would exit the CLI and leave a headless Chrome behind,
// still laying out a book nobody is waiting for.
var liveBrowsers struct {
	mu   sync.Mutex
	proc map[int]*os.Process
}

// trackBrowser registers the Chrome process behind an allocated chromedp
// context and returns a function that unregisters it. It returns nil when the
// context has no process to track (a remote allocator, or a browser that never
// started), so callers can defer the result unconditionally after a nil check.
func trackBrowser(browserCtx context.Context) func() {
	c := chromedp.FromContext(browserCtx)
	if c == nil || c.Browser == nil {
		return nil
	}
	p := c.Browser.Process()
	if p == nil {
		return nil
	}
	liveBrowsers.mu.Lock()
	defer liveBrowsers.mu.Unlock()
	if liveBrowsers.proc == nil {
		liveBrowsers.proc = make(map[int]*os.Process)
	}
	liveBrowsers.proc[p.Pid] = p
	return func() {
		liveBrowsers.mu.Lock()
		defer liveBrowsers.mu.Unlock()
		delete(liveBrowsers.proc, p.Pid)
	}
}

// KillRunningBrowsers SIGKILLs every Chrome process this package started and
// has not yet reaped. Call it immediately before os.Exit on a path that skips
// deferred cleanup; it is a no-op when no render is in flight.
func KillRunningBrowsers() {
	liveBrowsers.mu.Lock()
	procs := make([]*os.Process, 0, len(liveBrowsers.proc))
	for _, p := range liveBrowsers.proc {
		procs = append(procs, p)
	}
	liveBrowsers.proc = nil
	liveBrowsers.mu.Unlock()

	for _, p := range procs {
		// SIGKILL, not SIGTERM: the caller is on its way out and cannot wait for
		// Chrome to shut down politely. Killing the browser process closes the
		// IPC channels its renderer children watch, and they exit on their own.
		if err := p.Kill(); err != nil {
			slog.Debug("Failed to kill Chrome on exit", slog.Int("pid", p.Pid), slog.Any("error", err))
		}
	}
}
