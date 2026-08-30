package katex

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

// Rendering math at build time, in Go, is what lets a book carry finished
// formulas instead of the machinery to produce them. KaTeX is JavaScript, so a
// JavaScript interpreter runs it once during the build; the reader then needs
// no scripting at all, which matters because most e-readers do not offer any.
// Loading the library costs about 40 ms and a formula about 1–15 ms.

// mathSpanPattern matches the math markers the Markdown pipeline emits:
//
//	<span class="math math-display">$$…$$</span>
//	<span class="math math-inline">$…$</span>
//
// The body is HTML-escaped at that point (see internal/markdown/math.go), so
// it is unescaped again before reaching KaTeX.
var mathSpanPattern = regexp.MustCompile(`(?s)<span class="math math-(display|inline)">(.*?)</span>`)

// Renderer converts LaTeX to markup. It is not safe for concurrent use; call
// RenderMathSpans instead, which keeps a pool.
type Renderer struct {
	vm     *goja.Runtime
	render goja.Callable
}

// NewRenderer loads KaTeX into a fresh JavaScript runtime.
func NewRenderer() (*Renderer, error) {
	src, err := assets.ReadFile("assets/katex.min.js")
	if err != nil {
		return nil, fmt.Errorf("read vendored katex.min.js: %w", err)
	}
	vm := goja.New()
	if _, err := vm.RunString(string(src)); err != nil {
		return nil, fmt.Errorf("load KaTeX: %w", err)
	}
	global := vm.Get("katex")
	if global == nil {
		return nil, fmt.Errorf("KaTeX did not define a global after loading")
	}
	fn, ok := goja.AssertFunction(global.ToObject(vm).Get("renderToString"))
	if !ok {
		return nil, fmt.Errorf("katex.renderToString is not callable")
	}
	return &Renderer{vm: vm, render: fn}, nil
}

// Render returns the markup for one formula.
//
// Output is KaTeX's default "htmlAndMathml": styled spans for the visual
// rendering, plus a MathML twin the stylesheet hides from sight and leaves to
// assistive technology. That pairing is why Stylesheet and the fonts still
// ship — they are what the visual half is drawn with.
//
// A malformed formula is rendered as KaTeX's own inline error rather than
// failing the build, matching what the browser-side renderer has always done;
// the second return value reports it so a caller can warn. Detecting that
// needs the strict pass: with throwOnError off KaTeX swallows the parse error
// and hands back error markup, with nothing in the return value to distinguish
// it from a formula that rendered. The lenient pass only runs for the formulas
// that actually failed.
func (r *Renderer) Render(tex string, displayMode bool) (string, error) {
	markup, err := r.renderWith(tex, displayMode, true)
	if err == nil {
		return markup, nil
	}
	fallback, fallbackErr := r.renderWith(tex, displayMode, false)
	if fallbackErr != nil {
		// KaTeX declined to render even its own error message; nothing
		// useful can be emitted for this formula.
		return "", err
	}
	return fallback, err
}

// renderWith calls katex.renderToString once.
func (r *Renderer) renderWith(tex string, displayMode, throwOnError bool) (string, error) {
	opts := r.vm.NewObject()
	for key, value := range map[string]any{
		"displayMode":  displayMode,
		"output":       "htmlAndMathml",
		"throwOnError": throwOnError,
		// KaTeX's default strict mode reports questionable-but-renderable
		// input through console.warn. It guards the call, so nothing breaks
		// in an interpreter that has no console — but relying on that guard
		// is a needless dependency on KaTeX's internals.
		"strict": false,
	} {
		if err := opts.Set(key, value); err != nil {
			return "", err
		}
	}
	res, err := r.render(goja.Undefined(), r.vm.ToValue(tex), opts)
	if err != nil {
		return "", katexError(err)
	}
	return res.String(), nil
}

// katexError unwraps a thrown KaTeX ParseError into its message, so a warning
// reads "Undefined control sequence: \fracc" instead of a goja stack trace.
func katexError(err error) error {
	var exception *goja.Exception
	if errors.As(err, &exception) {
		msg := exception.Value().String()
		return fmt.Errorf("%s", strings.TrimPrefix(msg, "ParseError: "))
	}
	return err
}

// rendererPool keeps warmed interpreters around: the chapter pipeline renders
// concurrently, a goja runtime is single-threaded, and reloading the library
// per chapter would cost more than the formulas do.
var rendererPool = sync.Pool{
	New: func() any {
		r, err := NewRenderer()
		if err != nil {
			return err
		}
		return r
	},
}

// RenderMathSpans replaces every math span in an HTML fragment with finished
// KaTeX markup, and reports the formulas KaTeX rejected.
//
// Fragments containing no math are returned untouched without ever starting an
// interpreter, so a book without formulas pays nothing.
func RenderMathSpans(fragment string) (string, []error) {
	if !strings.Contains(fragment, `class="math math-`) {
		return fragment, nil
	}
	pooled := rendererPool.Get()
	if err, isErr := pooled.(error); isErr {
		return fragment, []error{err}
	}
	renderer := pooled.(*Renderer)
	defer rendererPool.Put(renderer)

	var errs []error
	out := mathSpanPattern.ReplaceAllStringFunc(fragment, func(span string) string {
		m := mathSpanPattern.FindStringSubmatch(span)
		display := m[1] == "display"
		tex := strings.TrimSpace(html.UnescapeString(m[2]))
		// Strip the delimiters the pipeline wrote around the formula.
		if display {
			tex = strings.TrimSuffix(strings.TrimPrefix(tex, "$$"), "$$")
		} else {
			tex = strings.TrimSuffix(strings.TrimPrefix(tex, "$"), "$")
		}
		markup, latexErr := renderer.Render(strings.TrimSpace(tex), display)
		if latexErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", strings.TrimSpace(tex), latexErr))
		}
		if markup == "" {
			return span
		}
		return markup
	})
	return out, errs
}
