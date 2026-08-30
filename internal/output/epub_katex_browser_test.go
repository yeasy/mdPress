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

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/yeasy/mdpress/internal/pdf"
)

// newOfflineBrowser returns a headless Chrome that cannot reach anything but
// the loopback interface.
//
// This is the whole point of the test below: with the CDN reachable, a page
// that still linked jsDelivr would render math and look identical to one
// rendering from the packaged copy. Blackholing DNS removes that ambiguity —
// if formulas render here, they rendered from files inside the book.
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

// TestEpubPackagedKaTeXRendersOffline is the proof behind the packaging: the
// files are not merely present in the archive, they actually render the book's
// formulas with no network available. A reading system resolves a chapter's
// relative references against the container, which is what the file server
// below reproduces.
func TestEpubPackagedKaTeXRendersOffline(t *testing.T) {
	ctx := newOfflineBrowser(t)

	dir := t.TempDir()
	epubPath := filepath.Join(dir, "math.epub")
	gen := NewEpubGenerator(EpubMeta{Title: "Math Book", Language: "en-US"})
	gen.AddChapter(EpubChapter{
		Title:    "Math",
		Filename: "math.xhtml",
		HTML: `<p>Inline <span class="math math-inline">$E = mc^2$</span> energy.</p>` +
			`<p><span class="math math-display">$$\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}$$</span></p>`,
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
	srv := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(root, "OEBPS"))))
	defer srv.Close()

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/math.xhtml"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`new Promise(function(resolve) {
			var start = Date.now();
			(function check() {
				var rendered = document.querySelectorAll('.katex').length;
				if (rendered > 0 || Date.now() - start > 8000) {
					resolve(JSON.stringify({
						rendered: rendered,
						// KaTeX emits a MathML twin next to the visual output;
						// its presence means the library really ran.
						mathml: document.querySelectorAll('.katex-mathml math').length,
						// A formula that failed to render leaves the raw source.
						leftoverSource: document.body.textContent.indexOf('\\\\frac') >= 0
					}));
					return;
				}
				setTimeout(check, 100);
			})();
		})`, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if !strings.Contains(raw, `"rendered":2`) {
		t.Errorf("expected both formulas rendered by the packaged KaTeX, got %s", raw)
	}
	if strings.Contains(raw, `"leftoverSource":true`) {
		t.Errorf("a formula was left as raw LaTeX source: %s", raw)
	}
	if strings.Contains(raw, `"mathml":0`) {
		t.Errorf("KaTeX produced no MathML, so it did not really run: %s", raw)
	}
}
