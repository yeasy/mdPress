package katex

import (
	"strings"
	"sync"
	"testing"
)

func TestRenderProducesVisualAndMathML(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		tex     string
		display bool
	}{
		{"inline", `E = mc^2`, false},
		{"display", `\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}`, true},
		{"matrix", `\begin{pmatrix} a & b \\ c & d \end{pmatrix}`, true},
	} {
		markup, latexErr := r.Render(tc.tex, tc.display)
		if latexErr != nil {
			t.Errorf("%s: unexpected LaTeX error: %v", tc.name, latexErr)
		}
		if !strings.Contains(markup, `class="katex`) {
			t.Errorf("%s: no KaTeX markup: %s", tc.name, markup)
		}
		// The MathML twin is what assistive technology reads.
		if !strings.Contains(markup, "<math") {
			t.Errorf("%s: no MathML in output: %s", tc.name, markup)
		}
		if tc.display && !strings.Contains(markup, `display="block"`) {
			t.Errorf("%s: display formula not marked as block: %s", tc.name, markup)
		}
	}
}

// TestRenderReportsBadLatex: a typo in a formula must not fail the build — the
// browser-side renderer has always drawn KaTeX's inline error instead — but it
// must be reported so the author hears about it.
func TestRenderReportsBadLatex(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	markup, latexErr := r.Render(`\frac{1}{`, false)
	if latexErr == nil {
		t.Error("malformed LaTeX should be reported")
	}
	if markup == "" {
		t.Error("malformed LaTeX should still produce KaTeX's error markup, not nothing")
	}
}

func TestRenderMathSpans(t *testing.T) {
	in := `<p>Inline <span class="math math-inline">$E = mc^2$</span> here.</p>` +
		`<p><span class="math math-display">$$a^2 + b^2 = c^2$$</span></p>`
	out, errs := RenderMathSpans(in)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if strings.Contains(out, `class="math math-`) {
		t.Errorf("math spans were not replaced: %s", out)
	}
	if n := strings.Count(out, `class="katex`); n < 2 {
		t.Errorf("expected both formulas rendered, found %d: %s", n, out)
	}
	if !strings.Contains(out, "<p>Inline ") || !strings.Contains(out, " here.</p>") {
		t.Errorf("surrounding markup was disturbed: %s", out)
	}
}

// TestRenderMathSpansUnescapesSource: the pipeline HTML-escapes the formula
// before it reaches here, so `a < b` arrives as `a &lt; b`. Feeding that to
// KaTeX verbatim renders the entity text instead of the operator.
func TestRenderMathSpansUnescapesSource(t *testing.T) {
	out, errs := RenderMathSpans(`<span class="math math-inline">$a &lt; b$</span>`)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if strings.Contains(out, "&amp;lt;") {
		t.Errorf("escaped source reached KaTeX: %s", out)
	}
	if !strings.Contains(out, "<mo>&lt;</mo>") {
		t.Errorf("expected a less-than operator in the MathML: %s", out)
	}
}

// TestRenderMathSpansWithoutMathIsFree: a book with no formulas must not pay
// for an interpreter.
func TestRenderMathSpansWithoutMathIsFree(t *testing.T) {
	in := `<p>No math here at all.</p>`
	out, errs := RenderMathSpans(in)
	if out != in || errs != nil {
		t.Errorf("fragment without math was altered: %q, %v", out, errs)
	}
}

// TestRenderMathSpansIsConcurrencySafe: chapters render in parallel, and a
// goja runtime is single-threaded.
func TestRenderMathSpansIsConcurrencySafe(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, errs := RenderMathSpans(`<span class="math math-inline">$x^2$</span>`)
			if len(errs) != 0 || !strings.Contains(out, "katex") {
				t.Errorf("concurrent render failed: %q %v", out, errs)
			}
		}()
	}
	wg.Wait()
}
