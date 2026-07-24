package cmd

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// interruptChildEnv marks the re-executed copy of this test binary that plays
// the part of a running mdpress command.
const interruptChildEnv = "MDPRESS_TEST_REPEAT_INTERRUPT_CHILD"

// TestRepeatInterruptExitsImmediately pins the second-Ctrl+C contract.
//
// signal.NotifyContext turns the first SIGINT into a context cancellation and
// replaces Go's default "die on signal" behavior for every signal after it. A
// user watching a stage that had not yet reached a cancellation point pressed
// Ctrl+C again, and again, and nothing happened — the only way out was another
// terminal and `kill -9`. The second signal must end the process.
//
// os.Exit cannot be observed in-process, so the test re-executes itself: the
// child installs the same handlers Execute() does, then blocks in a stage that
// ignores cancellation, which is exactly the situation the first press cannot
// resolve.
func TestRepeatInterruptExitsImmediately(t *testing.T) {
	if os.Getenv(interruptChildEnv) == "1" {
		runInterruptChild()
		return
	}
	if runtime.GOOS == "windows" {
		t.Skip("SIGINT delivery to a child process is POSIX-specific")
	}

	// CommandContext, so a failing assertion below cannot leave the child alive
	// for the rest of the package's tests.
	childCtx, cancelChild := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelChild()
	cmd := exec.CommandContext(childCtx, os.Args[0], "-test.run=TestRepeatInterruptExitsImmediately", "-test.timeout=90s")
	cmd.Env = append(os.Environ(), interruptChildEnv+"=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	// Wait for the child to confirm both handlers are installed; signaling
	// before that would race with signal.Notify and test nothing.
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if scanner.Text() == "ready" {
				ready <- "ready"
				return
			}
		}
		close(ready)
	}()
	select {
	case _, ok := <-ready:
		if !ok {
			t.Fatal("child exited before it was ready")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("child never signaled readiness")
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	// The child is deliberately stuck in a stage that never looks at the
	// context, so the first signal cannot get it out. Give it a moment to prove
	// that, then press again.
	select {
	case err := <-waitErr:
		t.Fatalf("child exited (%v) on the first signal; the test is not exercising the repeat path", err)
	case <-time.After(500 * time.Millisecond):
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-waitErr:
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("child exited with %v, want a non-zero interrupt status", err)
		}
		if got := exitErr.ExitCode(); got != interruptExitCode {
			t.Errorf("child exit code = %d, want %d (128 + SIGINT)", got, interruptExitCode)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the second interrupt did not end the process: Ctrl+C is still a one-way door")
	}
}

// runInterruptChild mimics Execute(): first signal cancels the command's
// context, and a stage that does not watch it keeps running.
func runInterruptChild() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopRepeatWatch := watchForRepeatInterrupt()
	defer stopRepeatWatch()

	os.Stdout.WriteString("ready\n") //nolint:errcheck
	<-ctx.Done()
	// Stand in for a render that does not reach a cancellation point: without
	// the repeat-interrupt watcher, this is where the user is stranded.
	time.Sleep(60 * time.Second)
}

// TestRepeatInterruptWatcherStopsCleanly checks that a command which finishes
// normally unregisters its handler, and that stopping twice is harmless — the
// stop runs from a defer in Execute, where a panic on a double close would turn
// a successful build into a crash.
func TestRepeatInterruptWatcherStopsCleanly(t *testing.T) {
	stopRepeatWatch := watchForRepeatInterrupt()
	stopRepeatWatch()
	stopRepeatWatch()
}
