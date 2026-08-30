package output

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/yeasy/mdpress/internal/pdf"
)

// newOfflineBrowser returns a headless Chrome that cannot reach anything but
// the loopback interface, and that will not execute a single line of page
// JavaScript.
//
// Both restrictions matter. With the CDN reachable, a page that still linked
// jsDelivr would render math and look identical to one rendering from the
// packaged copy; blackholing DNS removes that ambiguity. And with scripting
// available, a page could still be rendering its formulas in the reader —
// which is what most e-readers cannot do. Turning scripting off makes the test
// a statement about the book rather than about Chrome.
//
// scriptEnabled=false only stops the page's own scripts; CDP evaluation runs
// in a separate world and keeps working, which is how the probes below read
// the DOM.
func newOfflineBrowser(t *testing.T) context.Context {
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
			chromedp.Flag("blink-settings", "scriptEnabled=false"),
		)...)
	t.Cleanup(cancelAlloc)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	t.Cleanup(cancelBrowser)
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 90*time.Second)
	t.Cleanup(cancelTimeout)
	// Chrome being installed is not the same as Chrome being able to start;
	// see newBrowser in site_browser_test.go for the sandboxed-CI case.
	if err := chromedp.Run(timeoutCtx); err != nil {
		t.Skipf("Chrome is installed but would not start here: %v", err)
	}
	return timeoutCtx
}

// scriptCanary is served next to the unpacked book. If its script ran, the
// paragraph says "yes" and the global is defined — meaning scripting was still
// on and the formula assertions below would prove nothing.
const scriptCanary = `<!DOCTYPE html><html><body><p id="canary">no</p>` +
	`<script>window.__scriptsRan = true; document.getElementById('canary').textContent = 'yes';</script>` +
	`</body></html>`

// TestEpubMathRendersWithoutJavaScript is the proof behind pre-rendering: the
// formulas are finished markup in the book, so they appear with no network and
// no scripting at all — the conditions an e-reader actually offers. A reading
// system resolves a chapter's relative references against the container, which
// is what the file server below reproduces.
func TestEpubMathRendersWithoutJavaScript(t *testing.T) {
	ctx := newOfflineBrowser(t)

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "math.epub")
	gen := NewEpubGenerator(EpubMeta{Title: "Math Book", Language: "en-US"})
	gen.AddChapter(EpubChapter{
		Title:    "Math",
		Filename: "math.xhtml",
		HTML: `<p>Inline <span class="math math-inline">$E = mc^2$</span> energy.</p>` +
			`<p><span class="math math-display">$$\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}$$</span></p>` +
			`<p><span class="math math-display">$$\begin{pmatrix} a &amp; b \\ c &amp; d \end{pmatrix}$$</span></p>`,
	})
	if err := gen.Generate(epubPath); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Unpack the container so the chapter's relative hrefs resolve the way a
	// reading system resolves them.
	root := filepath.Join(dir, "unpacked")
	for name, data := range epubEntries(t, epubPath) {
		dest := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	served := filepath.Join(root, "OEBPS")
	// The canary lives beside the book, not in it: the generator strips
	// <script> from chapters, so there is no way to smuggle one into the page
	// under test.
	if err := os.WriteFile(filepath.Join(served, "canary.html"), []byte(scriptCanary), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(served)))
	defer srv.Close()

	var canary string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/canary.html"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`String(typeof window.__scriptsRan) + "/" + document.getElementById('canary').textContent`, &canary),
	); err != nil {
		t.Fatalf("canary probe failed: %v", err)
	}
	if canary != "undefined/no" {
		t.Fatalf("page JavaScript still executes (canary = %q); the rest of this test would prove nothing", canary)
	}

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/math.xhtml"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`(function() {
			var formulas = document.querySelectorAll('.katex');
			var laidOut = 0;
			for (var i = 0; i < formulas.length; i++) {
				var box = formulas[i].getBoundingClientRect();
				// A formula whose stylesheet never arrived still has a box, so
				// require a plausible one: KaTeX sizes glyphs in ems.
				if (box.width > 8 && box.height > 8) { laidOut++; }
			}
			return JSON.stringify({
				rendered: formulas.length,
				laidOut: laidOut,
				// The MathML twin is what a screen reader speaks.
				mathml: document.querySelectorAll('.katex-mathml math').length,
				// The twin must not be shown next to the visual rendering: the
				// packaged stylesheet clips it to a 1px box.
				mathmlClipped: getComputedStyle(document.querySelector('.katex-mathml')).clip !== 'auto',
				// A formula that never rendered would leave the raw source.
				leftoverSource: document.body.textContent.indexOf('\\\\frac') >= 0,
				scripts: document.querySelectorAll('script').length
			});
		})()`, &raw),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	for _, want := range []string{
		`"rendered":3`,
		`"laidOut":3`,
		`"mathml":3`,
		`"mathmlClipped":true`,
		`"leftoverSource":false`,
		`"scripts":0`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %s in the rendered page, got %s", want, raw)
		}
	}
}
