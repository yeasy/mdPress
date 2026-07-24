package pdf

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestPrintTimeoutErrorNamesTheSetting pins the wording. The point of the type
// is that "context canceled" gave the author nothing to search for: the message
// has to name output.pdf_timeout, the limit that expired, and a value to raise
// it to.
func TestPrintTimeoutErrorNamesTheSetting(t *testing.T) {
	tests := []struct {
		name  string
		limit time.Duration
		want  []string
	}{
		{
			name:  "default limit suggests double",
			limit: 120 * time.Second,
			want:  []string{"output.pdf_timeout", "2m0s", "output.pdf_timeout: 240"},
		},
		{
			name:  "suggestion stays inside the configured maximum",
			limit: 3000 * time.Second,
			want:  []string{"output.pdf_timeout: 3600"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := (&printTimeoutError{stage: "rendering the PDF in Chrome", limit: tt.limit}).Error()
			for _, want := range tt.want {
				if !strings.Contains(msg, want) {
					t.Errorf("message %q missing %q", msg, want)
				}
			}
			if strings.Contains(msg, "canceled") {
				t.Errorf("message %q must not read as an interruption", msg)
			}
		})
	}
}

// TestTimeoutErrorOnlyForDeadlines keeps Ctrl+C honest: a canceled build must
// not be reported as a timeout the user could have configured away.
func TestTimeoutErrorOnlyForDeadlines(t *testing.T) {
	g := NewGenerator(WithTimeout(time.Second))

	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := g.timeoutError(expired, "rendering the PDF in Chrome"); err == nil {
		t.Error("an expired deadline should produce a timeout error")
	} else {
		var timeoutErr *printTimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Errorf("got %T, want *printTimeoutError", err)
		}
	}

	canceled, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	if err := g.timeoutError(canceled, "rendering the PDF in Chrome"); err != nil {
		t.Errorf("a canceled context is not a timeout, got %v", err)
	}
	if err := g.timeoutError(context.Background(), "rendering the PDF in Chrome"); err != nil {
		t.Errorf("a live context is not a timeout, got %v", err)
	}
}

// TestGenerateReportsTimeoutWhenChromeNeverStarts drives the real generator with
// a stand-in browser that never speaks the DevTools protocol. Before the fix the
// startup deadline was attached to the Run that allocates the browser, which
// chromedp turns into a dead browser rather than a reported deadline, so the
// author saw "context canceled" with no hint that output.pdf_timeout existed.
func TestGenerateReportsTimeoutWhenChromeNeverStarts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in browser is a /bin/sh script")
	}
	dir := t.TempDir()
	fakeChrome := filepath.Join(dir, "fake-chrome")
	if err := os.WriteFile(fakeChrome, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MDPRESS_CHROME_PATH", fakeChrome)
	t.Setenv("CHROME_BIN", "")

	g := NewGenerator(WithTimeout(500 * time.Millisecond))
	err := g.Generate("<html><head></head><body>hello</body></html>", filepath.Join(dir, "out.pdf"))
	if err == nil {
		t.Fatal("expected an error when Chrome never comes up")
	}
	if !strings.Contains(err.Error(), "output.pdf_timeout") {
		t.Errorf("error should name the setting that caused it, got: %v", err)
	}
	if strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should not read as an interruption, got: %v", err)
	}
}
