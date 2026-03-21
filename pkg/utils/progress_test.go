package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewProgressTracker(t *testing.T) {
	tracker := NewProgressTracker(5)
	if tracker == nil {
		t.Fatal("NewProgressTracker returned nil")
	}
	if tracker.total != 5 {
		t.Errorf("Expected total=5, got %d", tracker.total)
	}
	if tracker.current != 0 {
		t.Errorf("Expected current=0, got %d", tracker.current)
	}
	if tracker.start.IsZero() {
		t.Error("Expected start time to be set")
	}
}

func TestProgressTracker_Start(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(3)
	tracker.Start("Step 1")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "[1/3]") {
		t.Errorf("Expected [1/3] in output, got: %s", output)
	}
	if !strings.Contains(output, "Step 1") {
		t.Errorf("Expected 'Step 1' in output, got: %s", output)
	}
	if tracker.current != 1 {
		t.Errorf("Expected current=1 after Start, got %d", tracker.current)
	}
}

func TestProgressTracker_MultipleStarts(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(3)
	tracker.Start("Step 1")
	tracker.Start("Step 2")
	tracker.Start("Step 3")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "[1/3]") {
		t.Errorf("Expected [1/3] in output")
	}
	if !strings.Contains(output, "[2/3]") {
		t.Errorf("Expected [2/3] in output")
	}
	if !strings.Contains(output, "[3/3]") {
		t.Errorf("Expected [3/3] in output")
	}
	if tracker.current != 3 {
		t.Errorf("Expected current=3, got %d", tracker.current)
	}
}

func TestProgressTracker_Done(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(1)
	tracker.Start("Build")
	tracker.Done()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain either the checkmark or just "✓"
	if !strings.Contains(output, "✓") && !strings.Contains(output, "√") {
		t.Errorf("Expected success marker in output, got: %s", output)
	}
}

func TestProgressTracker_Fail(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(1)
	tracker.Start("Build")
	tracker.Fail()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain the cross mark
	if !strings.Contains(output, "✗") && !strings.Contains(output, "×") {
		t.Errorf("Expected failure marker in output, got: %s", output)
	}
}

func TestProgressTracker_Skip(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(1)
	tracker.Start("Build")
	tracker.Skip("Already built")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain skip marker and reason
	if !strings.Contains(output, "Already built") {
		t.Errorf("Expected skip reason in output, got: %s", output)
	}
}

func TestProgressTracker_DoneWithDetail(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(1)
	tracker.Start("Compile")
	tracker.DoneWithDetail("5 files compiled")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "5 files compiled") {
		t.Errorf("Expected detail message in output, got: %s", output)
	}
	if !strings.Contains(output, "✓") && !strings.Contains(output, "√") {
		t.Errorf("Expected success marker in output")
	}
}

func TestProgressTracker_Finish(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(2)
	tracker.Start("Step 1")
	tracker.Done()
	tracker.Start("Step 2")
	tracker.Done()
	tracker.Finish()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Build completed") {
		t.Errorf("Expected 'Build completed' in output, got: %s", output)
	}
	// Should contain elapsed time indicator
	if !strings.Contains(output, "elapsed") {
		t.Errorf("Expected 'elapsed' in output, got: %s", output)
	}
}

func TestProgressTracker_FinishWithError(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(1)
	tracker.Start("Build")
	tracker.Fail()
	tracker.FinishWithError(fmt.Errorf("compilation error"))

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Build failed") {
		t.Errorf("Expected 'Build failed' in output, got: %s", output)
	}
	if !strings.Contains(output, "compilation error") {
		t.Errorf("Expected error message in output, got: %s", output)
	}
	if !strings.Contains(output, "elapsed") {
		t.Errorf("Expected 'elapsed' in output")
	}
}

func TestProgressTracker_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		testFunc  func(*ProgressTracker)
		shouldErr bool
	}{
		{
			name:  "zero total steps",
			total: 0,
			testFunc: func(pt *ProgressTracker) {
				pt.Start("Test")
			},
			shouldErr: false, // Should not error, just show [1/0]
		},
		{
			name:  "single step",
			total: 1,
			testFunc: func(pt *ProgressTracker) {
				pt.Start("Only step")
				pt.Done()
			},
			shouldErr: false,
		},
		{
			name:  "large number of steps",
			total: 1000,
			testFunc: func(pt *ProgressTracker) {
				for i := 0; i < 10; i++ {
					pt.Start(fmt.Sprintf("Step %d", i+1))
					pt.Done()
				}
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Suppress output
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			tracker := NewProgressTracker(tt.total)
			tt.testFunc(tracker)

			w.Close()
			os.Stdout = oldStdout

			// Just verify no panic occurred
		})
	}
}

func TestProgressTracker_Timing(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(2)
	tracker.Start("Step 1")
	tracker.Done()

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	tracker.Start("Step 2")
	tracker.Done()
	tracker.Finish()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should have captured elapsed time
	if !strings.Contains(output, "elapsed") {
		t.Errorf("Expected elapsed time in output, got: %s", output)
	}

	if tracker.current != 2 {
		t.Errorf("Expected current=2, got %d", tracker.current)
	}
}

func TestProgressTracker_ConcurrentCalls(t *testing.T) {
	// This is a stress test - in practice you shouldn't use the same tracker
	// from multiple goroutines without synchronization
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(5)

	var wg sync.WaitGroup
	// Create a sequence of operations
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Note: This is not thread-safe without external synchronization
			// Just testing that it doesn't panic
			tracker.Start(fmt.Sprintf("Step %d", index))
			time.Sleep(1 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	tracker.Finish()

	w.Close()
	os.Stdout = oldStdout

	// Just verify we got output
	var buf bytes.Buffer
	io.Copy(&buf, r)
}

func TestProgressTracker_ColorOutput(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save original color state
	originalColors := colorEnabled

	// Test with colors enabled
	SetColorEnabled(true)
	tracker := NewProgressTracker(1)
	tracker.Start("Test")
	tracker.Done()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	outputWithColor := buf.String()

	// Restore
	SetColorEnabled(originalColors)

	// Output should contain the message
	if !strings.Contains(outputWithColor, "Test") {
		t.Errorf("Expected 'Test' in color output")
	}
}

func TestProgressTracker_NoColorOutput(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Save original color state
	originalColors := colorEnabled

	// Test with colors disabled
	SetColorEnabled(false)
	tracker := NewProgressTracker(1)
	tracker.Start("Test")
	tracker.Done()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	outputNoColor := buf.String()

	// Restore
	SetColorEnabled(originalColors)

	// Output should contain the message
	if !strings.Contains(outputNoColor, "Test") {
		t.Errorf("Expected 'Test' in no-color output")
	}
}

func TestProgressTracker_WorkflowSequence(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tracker := NewProgressTracker(4)

	tracker.Start("Parse")
	tracker.Done()

	tracker.Start("Validate")
	tracker.DoneWithDetail("20 warnings")

	tracker.Start("Transform")
	tracker.Skip("Not needed")

	tracker.Start("Output")
	tracker.Done()

	tracker.Finish()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify all steps appear
	if !strings.Contains(output, "[1/4]") {
		t.Error("Expected [1/4]")
	}
	if !strings.Contains(output, "[2/4]") {
		t.Error("Expected [2/4]")
	}
	if !strings.Contains(output, "[3/4]") {
		t.Error("Expected [3/4]")
	}
	if !strings.Contains(output, "[4/4]") {
		t.Error("Expected [4/4]")
	}
	if !strings.Contains(output, "20 warnings") {
		t.Error("Expected detail message")
	}
}
