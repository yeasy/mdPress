// katex_inline.go turns the vendored KaTeX distribution into something a
// single-file HTML document can carry: math rendered at build time, and a
// stylesheet whose fonts are data URIs rather than sibling files.
//
// The standalone format is advertised as portable — one file you can mail to
// someone. It used to fetch KaTeX from a CDN at view time, so a reader with no
// network got raw LaTeX source in the one format whose entire selling point is
// being self-contained. Rendering here, in Go, means the shipped document
// needs neither the network nor JavaScript to show a formula.
package renderer

import (
	"encoding/base64"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/dop251/goja"

	"github.com/yeasy/mdpress/internal/katex"
)

// mathSpanPattern matches the spans internal/markdown emits for math:
//
//	<span class="math math-display">$$…$$</span>
//	<span class="math math-inline">$…$</span>
//
// The formula between the delimiters is HTML-escaped by the Markdown
// pipeline, so it can never contain the "</span>" that ends the match and the
// lazy quantifier is safe.
var mathSpanPattern = regexp.MustCompile(`(?s)<span class="math math-(display|inline)">(.*?)</span>`)

// katexFontURL matches a font reference in the vendored stylesheet. Stylesheet()
// has already stripped the WOFF and TrueType sources, so only the WOFF2 files
// that actually ship are left to resolve.
var katexFontURL = regexp.MustCompile(`url\((fonts/[^)]+)\)`)

// katexVM is the JavaScript interpreter that renders math, built once and
// reused: parsing the 270 KB katex.min.js costs far more than rendering a
// formula, and a book has many formulas. goja values are not safe for
// concurrent use, so every call holds the mutex.
var (
	katexVMOnce sync.Once
	katexVM     *goja.Runtime
	katexRender goja.Callable
	katexVMErr  error
	katexVMMu   sync.Mutex
)

func loadKaTeXVM() (*goja.Runtime, goja.Callable, error) {
	katexVMOnce.Do(func() {
		assets, err := katex.Assets()
		if err != nil {
			katexVMErr = err
			return
		}
		var script []byte
		for _, a := range assets {
			if a.Path == "katex.min.js" {
				script = a.Data
				break
			}
		}
		if script == nil {
			katexVMErr = fmt.Errorf("vendored KaTeX distribution has no katex.min.js")
			return
		}
		vm := goja.New()
		if _, err := vm.RunString(string(script)); err != nil {
			katexVMErr = fmt.Errorf("evaluate vendored katex.min.js: %w", err)
			return
		}
		obj := vm.Get("katex")
		if obj == nil || goja.IsUndefined(obj) {
			katexVMErr = fmt.Errorf("vendored katex.min.js defined no katex global")
			return
		}
		fn, ok := goja.AssertFunction(obj.ToObject(vm).Get("renderToString"))
		if !ok {
			katexVMErr = fmt.Errorf("vendored katex.min.js exposes no renderToString")
			return
		}
		katexVM, katexRender = vm, fn
	})
	return katexVM, katexRender, katexVMErr
}

// renderMathSpans replaces every math span in fragment with finished KaTeX
// markup: the styled spans plus the hidden MathML twin, exactly what the
// browser-side auto-render used to produce. It returns the rewritten fragment
// and the formulas KaTeX rejected.
//
// A fragment with no math is returned untouched and never starts the
// interpreter, so a book without formulas pays nothing for this.
func renderMathSpans(fragment string) (string, []error) {
	if !strings.Contains(fragment, `class="math math-`) {
		return fragment, nil
	}
	vm, render, err := loadKaTeXVM()
	if err != nil {
		return fragment, []error{err}
	}

	katexVMMu.Lock()
	defer katexVMMu.Unlock()

	var failures []error
	out := mathSpanPattern.ReplaceAllStringFunc(fragment, func(match string) string {
		groups := mathSpanPattern.FindStringSubmatch(match)
		display := groups[1] == "display"
		delim := "$"
		if display {
			delim = "$$"
		}
		body := groups[2]
		if !strings.HasPrefix(body, delim) || !strings.HasSuffix(body, delim) ||
			len(body) < 2*len(delim) {
			// Not a shape this renderer produced; leave it for the reader
			// rather than guessing at the delimiters.
			return match
		}
		latex := html.UnescapeString(body[len(delim) : len(body)-len(delim)])

		// throwOnError distinguishes "KaTeX rendered this" from "KaTeX gave up
		// and drew the source in red". Ask for the strict answer first so the
		// build can report the bad formula, then fall back to the forgiving
		// render so the document still shows the reader where the problem is.
		rendered, err := callKaTeX(vm, render, latex, display, true)
		if err != nil {
			failures = append(failures, fmt.Errorf("KaTeX rejected %q: %w", latex, err))
			rendered, err = callKaTeX(vm, render, latex, display, false)
			if err != nil {
				return match
			}
		}
		return rendered
	})
	return out, failures
}

func callKaTeX(vm *goja.Runtime, render goja.Callable, latex string, display, throwOnError bool) (string, error) {
	opts := vm.NewObject()
	_ = opts.Set("displayMode", display)
	_ = opts.Set("output", "htmlAndMathml")
	_ = opts.Set("throwOnError", throwOnError)
	_ = opts.Set("strict", false)
	v, err := render(goja.Undefined(), vm.ToValue(latex), opts)
	if err != nil {
		return "", err
	}
	return v.String(), nil
}

// katexInlineStylesheet returns the vendored KaTeX stylesheet with every font
// embedded as a data URI, ready to drop into a <style> block.
func katexInlineStylesheet() (string, error) {
	css, err := katex.Stylesheet()
	if err != nil {
		return "", err
	}
	assets, err := katex.Assets()
	if err != nil {
		return "", err
	}
	return inlineKaTeXFonts(css, assets)
}

// inlineKaTeXFonts rewrites `url(fonts/KaTeX_*.woff2)` into
// `url(data:font/woff2;base64,…)` using the packaged assets.
//
// A reference that does not resolve is a build error, not a warning: the whole
// point of this format is that nothing is fetched later, so a font left as a
// relative path is a glyph that silently falls back to the system serif in
// every copy of the document that ever gets mailed anywhere.
func inlineKaTeXFonts(css string, assets []katex.Asset) (string, error) {
	byPath := make(map[string]katex.Asset, len(assets))
	for _, a := range assets {
		byPath[a.Path] = a
	}

	var missing []string
	out := katexFontURL.ReplaceAllStringFunc(css, func(match string) string {
		ref := katexFontURL.FindStringSubmatch(match)[1]
		asset, ok := byPath[ref]
		if !ok {
			missing = append(missing, ref)
			return match
		}
		mediaType := asset.MediaType
		if mediaType == "" {
			mediaType = "font/woff2"
		}
		return "url(data:" + mediaType + ";base64," +
			base64.StdEncoding.EncodeToString(asset.Data) + ")"
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("vendored KaTeX stylesheet references fonts that are not packaged: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}
