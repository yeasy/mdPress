package cmd

import "testing"

func TestFenceTracker(t *testing.T) {
	lines := []string{
		"see [real](real.md)", // 0 content
		"```go",               // 1 fence open
		"[fake](fake.md)",     // 2 code
		"```",                 // 3 fence close
		"[real2](real2.md)",   // 4 content
		"~~~",                 // 5 tilde fence open
		"![img](nope.png)",    // 6 code
		"~~~",                 // 7 tilde close
		"after",               // 8 content
	}
	wantCode := []bool{false, true, true, true, false, true, true, true, false}

	var f fenceTracker
	for i, line := range lines {
		if got := f.InCode(line); got != wantCode[i] {
			t.Errorf("line %d (%q): InCode = %v, want %v", i, line, got, wantCode[i])
		}
	}
}

func TestFenceTracker_NestedAndLongerFences(t *testing.T) {
	// A ``` line inside a ```` block is content, not a close.
	lines := []string{"````md", "```go", "```", "````", "out"}
	wantCode := []bool{true, true, true, true, false}

	var f fenceTracker
	for i, line := range lines {
		if got := f.InCode(line); got != wantCode[i] {
			t.Errorf("line %d (%q): InCode = %v, want %v", i, line, got, wantCode[i])
		}
	}
}

func TestStripInlineCode(t *testing.T) {
	in := "text `[fake](fake.md)` and [real](real.md)"
	got := stripInlineCode(in)
	if len(got) != len(in) {
		t.Errorf("length changed: %d vs %d (offsets must stay valid)", len(got), len(in))
	}
	if want := "[real](real.md)"; !contains(got, want) {
		t.Errorf("real link was stripped: %q", got)
	}
	if contains(got, "fake.md") {
		t.Errorf("inline code span survived: %q", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
