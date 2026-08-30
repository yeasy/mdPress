package renderer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/yeasy/mdpress/internal/pdf"
)

// newOfflineStandaloneBrowser returns a headless Chrome that cannot resolve
// anything but the loopback interface.
//
// This is what makes the test below mean something. With the CDN reachable, a
// portable file that still linked jsDelivr would render math and look
// identical to one rendering from its own bytes. Blackholing DNS removes the
// ambiguity: if formulas are laid out here, they came out of the file.
func newOfflineStandaloneBrowser(t *testing.T) context.Context {
	t.Helper()
	if testing.Short() {
		t.Skip("browser test skipped in -short mode")
	}
	if err := pdf.CheckChromiumAvailable(); err != nil {
		t.Skipf("Chrome/Chromium not available: %v", err)
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("host-resolver-rules", "MAP * ~NOTFOUND, EXCLUDE 127.0.0.1"),
		)...)
	t.Cleanup(cancelAlloc)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	t.Cleanup(cancelBrowser)
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 90*time.Second)
	t.Cleanup(cancelTimeout)
	// Chrome being installed is not the same as Chrome being able to start.
	if err := chromedp.Run(timeoutCtx); err != nil {
		t.Skipf("Chrome is installed but would not start here: %v", err)
	}
	return timeoutCtx
}

// TestStandaloneMathRendersOffline is the proof behind the change: a portable
// HTML file opened over file://, the way a reader who was mailed one opens it,
// with no network at all, shows typeset formulas rather than LaTeX source.
func TestStandaloneMathRendersOffline(t *testing.T) {
	ctx := newOfflineStandaloneBrowser(t)

	html := renderStandalone(t, `<p>Inline <span class="math math-inline">$E = mc^2$</span> energy.</p>`+
		`<p><span class="math math-display">$$\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}$$</span></p>`+
		`<p>Also <span class="math math-inline">$\frac{a}{b}$</span> inline.</p>`)

	path := filepath.Join(t.TempDir(), "portable.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate("file://"+path),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`(function() {
			// document.fonts.ready settles once the embedded WOFF2 faces are
			// decoded, so the boxes measured below are the real laid-out ones.
			return document.fonts.ready.then(function() {
				var nodes = document.querySelectorAll('.katex');
				var laidOut = 0;
				for (var i = 0; i < nodes.length; i++) {
					var box = nodes[i].getBoundingClientRect();
					if (box.width > 0 && box.height > 0) laidOut++;
				}
				var main = document.getElementById('main-content');
				return JSON.stringify({
					rendered: nodes.length,
					laidOut: laidOut,
					// KaTeX emits a MathML twin; its presence means real
					// KaTeX markup, not something that merely looks like it.
					mathml: document.querySelectorAll('.katex-mathml math').length,
					display: document.querySelectorAll('.katex-display').length,
					// innerText is what the reader actually sees; the MathML
					// annotation carrying the source is clipped out of it.
					visibleSource: main.innerText.indexOf('\\frac') >= 0 ||
						main.innerText.indexOf('\\int') >= 0,
					// A failure notice would mean something still tried the network.
					assetError: document.querySelectorAll('.asset-error').length
				});
			});
		})()`, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if !strings.Contains(raw, `"rendered":3`) {
		t.Errorf("expected all three formulas rendered from the file itself, got %s", raw)
	}
	if !strings.Contains(raw, `"laidOut":3`) {
		t.Errorf("a formula rendered but has a zero-size box, so its fonts did not load: %s", raw)
	}
	if strings.Contains(raw, `"mathml":0`) {
		t.Errorf("no MathML twin, so this is not real KaTeX output: %s", raw)
	}
	if !strings.Contains(raw, `"display":1`) {
		t.Errorf("the display formula was not laid out in display mode: %s", raw)
	}
	if strings.Contains(raw, `"visibleSource":true`) {
		t.Errorf("the reader is being shown raw LaTeX source: %s", raw)
	}
	if !strings.Contains(raw, `"assetError":0`) {
		t.Errorf("the page raised an asset-failure notice, so it still wanted the network: %s", raw)
	}
}
