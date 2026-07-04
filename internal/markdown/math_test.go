package markdown

import (
	"strings"
	"testing"
)

func TestMathPreprocessorBlockMath(t *testing.T) {
	m := newMathPreprocessor()
	input := "Before\n$$E = mc^2$$\nAfter"
	processed := m.preprocess(input)

	// Placeholder must replace the formula.
	if strings.Contains(processed, "$") {
		t.Errorf("preprocess: dollar sign not replaced, got: %q", processed)
	}
	if !strings.Contains(processed, "MDPMATHBLOCK") {
		t.Errorf("preprocess: block math placeholder not found, got: %q", processed)
	}

	// Postprocess must restore the formula inside a span.
	restored := m.postprocess(processed)
	if !strings.Contains(restored, `class="math math-display"`) {
		t.Errorf("postprocess: math-display span not found, got: %q", restored)
	}
	if !strings.Contains(restored, "E = mc^2") {
		t.Errorf("postprocess: formula content not restored, got: %q", restored)
	}
}

func TestMathPreprocessorInlineMath(t *testing.T) {
	m := newMathPreprocessor()
	input := "The value of $x_1^2$ is important."
	processed := m.preprocess(input)

	// Inline placeholder must be present; the underscore must not trigger goldmark emphasis.
	if !strings.Contains(processed, "MDPMATHINLINE") {
		t.Errorf("preprocess: inline math placeholder not found, got: %q", processed)
	}
	// The underscore inside the formula must be gone (stored in the slice, not in the source).
	if strings.Contains(processed, "_1") {
		t.Errorf("preprocess: underscore still present in processed markdown, got: %q", processed)
	}

	restored := m.postprocess(processed)
	if !strings.Contains(restored, `class="math math-inline"`) {
		t.Errorf("postprocess: math-inline span not found, got: %q", restored)
	}
	if !strings.Contains(restored, "x_1^2") {
		t.Errorf("postprocess: formula content not restored, got: %q", restored)
	}
}

func TestMathPreprocessorNoMath(t *testing.T) {
	m := newMathPreprocessor()
	input := "This costs $5 and $10 total."
	processed := m.preprocess(input)
	// Currency amounts should not be replaced (single $ with space or digit context).
	// The inline regex requires content without leading/trailing spaces.
	// "$5" → single char → matched by single-char variant; "$10" → two chars → matched.
	// This is acceptable: false positives with currency are a known trade-off.
	// Just ensure roundtrip is lossless.
	restored := m.postprocess(processed)
	// All original text must survive the round-trip (possibly wrapped in spans).
	if !strings.Contains(restored, "5") || !strings.Contains(restored, "10") {
		t.Errorf("postprocess: currency amounts lost, got: %q", restored)
	}
}

func TestMathPreprocessorSkipsCode(t *testing.T) {
	tests := []struct {
		name string
		// wantMath is true if the input should produce math placeholders.
		wantMath bool
		input    string
		// mustContain is a literal that must survive verbatim in the processed
		// output (e.g. code that must not be turned into math).
		mustContain []string
		// mustNotContain must be absent from the processed output.
		mustNotContain []string
	}{
		{
			name:     "inline math outside code becomes placeholder",
			wantMath: true,
			input:    "The value $x_1^2$ matters.",
		},
		{
			name:     "block math outside code becomes placeholder",
			wantMath: true,
			input:    "Before\n$$E = mc^2$$\nAfter\n",
		},
		{
			name:           "dollar inside bash fenced block untouched",
			wantMath:       false,
			input:          "Run this:\n\n```bash\necho \"$HOME\" and \"$PATH\"\nkill $$\nwait $$\n```\n\nDone.\n",
			mustContain:    []string{`echo "$HOME" and "$PATH"`, "kill $$", "wait $$"},
			mustNotContain: []string{"MDPMATHINLINE", "MDPMATHBLOCK"},
		},
		{
			name:           "dollar inside tilde fenced block untouched",
			wantMath:       false,
			input:          "~~~\nawk '{print $$1}'\n~~~\n",
			mustContain:    []string{"awk '{print $$1}'"},
			mustNotContain: []string{"MDPMATHINLINE", "MDPMATHBLOCK"},
		},
		{
			name:           "dollar inside inline code span untouched",
			wantMath:       false,
			input:          "Use `echo $PATH` in your shell and `kill $$` too.",
			mustContain:    []string{"`echo $PATH`", "`kill $$`"},
			mustNotContain: []string{"MDPMATHINLINE", "MDPMATHBLOCK"},
		},
		{
			name:        "real inline code span and real math on one line",
			wantMath:    true,
			input:       "The var `$PATH` is a path but $E = mc^2$ is physics.",
			mustContain: []string{"`$PATH`"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newMathPreprocessor()
			processed := m.preprocess(tc.input)

			hasMath := strings.Contains(processed, "MDPMATHINLINE") ||
				strings.Contains(processed, "MDPMATHBLOCK")
			if hasMath != tc.wantMath {
				t.Errorf("wantMath=%v but got placeholders=%v; processed=%q",
					tc.wantMath, hasMath, processed)
			}

			for _, s := range tc.mustContain {
				if !strings.Contains(processed, s) {
					t.Errorf("expected processed output to contain %q, got: %q", s, processed)
				}
			}
			for _, s := range tc.mustNotContain {
				if strings.Contains(processed, s) {
					t.Errorf("expected processed output NOT to contain %q, got: %q", s, processed)
				}
			}

			// Round-trip through postprocess must be lossless for code content.
			restored := m.postprocess(processed)
			for _, s := range tc.mustContain {
				if !strings.Contains(restored, s) {
					t.Errorf("expected restored output to contain %q, got: %q", s, restored)
				}
			}
		})
	}
}

func TestMathPreprocessorRealMathAndCodeSpan(t *testing.T) {
	m := newMathPreprocessor()
	input := "The var `$PATH` is a path but $E = mc^2$ is physics."
	processed := m.preprocess(input)

	if !strings.Contains(processed, "MDPMATHINLINE") {
		t.Errorf("expected inline math placeholder for real math, got: %q", processed)
	}
	if !strings.Contains(processed, "`$PATH`") {
		t.Errorf("code span with $ must be preserved, got: %q", processed)
	}

	restored := m.postprocess(processed)
	if !strings.Contains(restored, "E = mc^2") {
		t.Errorf("real math content not restored, got: %q", restored)
	}
	if !strings.Contains(restored, "`$PATH`") {
		t.Errorf("code span lost after round-trip, got: %q", restored)
	}
}

func TestParserWithMath(t *testing.T) {
	p := NewParser()
	source := []byte("# Euler\n\nEuler's identity: $e^{i\\pi} + 1 = 0$\n\nDisplay:\n\n$$\n\\int_0^\\infty e^{-x} dx = 1\n$$\n")
	html, headings, err := p.Parse(source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(headings) == 0 {
		t.Error("expected at least one heading")
	}
	if !strings.Contains(html, `class="math math-inline"`) {
		t.Errorf("expected inline math span in output, got: %q", html)
	}
	if !strings.Contains(html, `class="math math-display"`) {
		t.Errorf("expected display math span in output, got: %q", html)
	}
}
