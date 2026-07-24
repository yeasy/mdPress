package output

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/yeasy/mdpress/internal/pdf"
)

// The search assertions elsewhere in this package read the generated JavaScript
// as text, which cannot tell whether the code actually behaves. These tests run
// a generated site in the same headless Chrome the PDF backend already needs,
// and drive the real search box. They skip when Chrome is absent.

// searchProbeJS drives the real search UI for one query and reports what the
// reader would see.
const searchProbeJS = `(async function() {
  var input = document.getElementById('search-input');
  var status = document.getElementById('search-status');
  var results = document.getElementById('search-results');
  window.openSearch();
  input.value = %s;
  input.dispatchEvent(new Event('input', { bubbles: true }));
  await new Promise(function(r) { setTimeout(r, 600); });
  return JSON.stringify({
    status: status.textContent,
    titles: Array.prototype.map.call(results.querySelectorAll('.search-result-title'), function(e) { return e.textContent; }),
    empty: results.textContent
  });
})()`

type searchProbe struct {
	Status string   `json:"status"`
	Titles []string `json:"titles"`
	Empty  string   `json:"empty"`
}

// browserTestSite generates the fixture book used by the browser tests.
func browserTestSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Search Test", Language: "en-US"})
	gen.AddChapter(SiteChapter{
		Title:    "Intro",
		Filename: "intro.html",
		Content: `<h1>Intro</h1><p>This chapter contains the unique word zorbulax ` +
			`and a phrase "quick brown fox".</p><h2>Sub heading one</h2><p>Body text.</p>`,
	})
	gen.AddChapter(SiteChapter{
		Title:    "中文章节",
		Filename: "cjk.html",
		Content:  `<h1>中文章节</h1><p>这是一个中文章节，包含关键词 数据库 和 索引。</p>`,
	})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	return dir
}

// newBrowser returns a headless Chrome context, skipping the test when no
// browser is installed.
func newBrowser(t *testing.T) context.Context {
	t.Helper()
	if testing.Short() {
		t.Skip("browser test skipped in -short mode")
	}
	if err := pdf.CheckChromiumAvailable(); err != nil {
		t.Skipf("Chrome/Chromium not available: %v", err)
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("allow-file-access-from-files", false),
		)...)
	t.Cleanup(cancelAlloc)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	t.Cleanup(cancelBrowser)
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, 90*time.Second)
	t.Cleanup(cancelTimeout)

	// Chrome being installed is not the same as Chrome being able to start.
	// A sandboxed CI environment — goreleaser's release runner among them —
	// has the binary but crashpad aborts the launch on missing cpufreq sysfs
	// files. That is an environment limitation, not a site regression, so
	// force the browser to start here (chromedp allocates it lazily on the
	// first Run) and skip on failure, rather than letting every individual
	// assertion below fail with "chrome failed to start". The probe runs on
	// timeoutCtx — the same context the test uses and whose cancel is deferred
	// to t.Cleanup — because chromedp ties the browser's lifetime to the
	// context of that first Run; a shorter-lived one would be canceled out
	// from under the test. A real launch means the failures that follow are
	// real.
	if err := chromedp.Run(timeoutCtx); err != nil {
		t.Skipf("Chrome is installed but would not start here: %v", err)
	}

	return timeoutCtx
}

// runSearch loads pageURL and types query into the site's own search box.
func runSearch(t *testing.T, ctx context.Context, pageURL, query string) searchProbe {
	t.Helper()
	quoted, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	js := strings.Replace(searchProbeJS, "%s", string(quoted), 1)
	var raw string
	err = chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("#search-input", chromedp.ByQuery),
		chromedp.Evaluate(js, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	)
	if err != nil {
		t.Fatalf("search for %q failed: %v", query, err)
	}
	var probe searchProbe
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		t.Fatalf("decode probe %q: %v", raw, err)
	}
	return probe
}

// TestSiteSearchInBrowser exercises the query syntaxes the manual documents:
// a quoted phrase used to return nothing because the quote characters went
// into the substring search, and a CJK query typed the natural way — with no
// spaces — matched only when it happened to appear as one contiguous run.
func TestSiteSearchInBrowser(t *testing.T) {
	ctx := newBrowser(t)
	srv := httptest.NewServer(http.FileServer(http.Dir(browserTestSite(t))))
	defer srv.Close()

	cases := []struct {
		query string
		want  string // expected chapter title, "" for no results
	}{
		{"zorbulax", "Intro"},
		{`"zorbulax"`, "Intro"},
		{`"quick brown fox"`, "Intro"},
		{`"Sub heading"`, "Intro"},
		{"数据库", "中文章节"},
		{"数据库索引", "中文章节"},
		{"数据库 索引", "中文章节"},
		{"nosuchwordanywhere", ""},
		{`"fox brown quick"`, ""},
	}
	for _, tc := range cases {
		probe := runSearch(t, ctx, srv.URL+"/intro.html", tc.query)
		if tc.want == "" {
			if len(probe.Titles) != 0 {
				t.Errorf("search %q should find nothing, got %v", tc.query, probe.Titles)
			}
			continue
		}
		var found bool
		for _, title := range probe.Titles {
			if title == tc.want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("search %q: want a result titled %q, got %v (status %q)",
				tc.query, tc.want, probe.Titles, probe.Status)
		}
	}
}

// TestSiteSearchOverFileProtocol covers previewing a built book by opening it
// off disk. The search index cannot be fetched from a file:// page, and the
// failure used to be swallowed into an empty index, so every query answered
// "No results" — indistinguishable from content that was never indexed.
func TestSiteSearchOverFileProtocol(t *testing.T) {
	ctx := newBrowser(t)
	dir := browserTestSite(t)
	pageURL := "file://" + filepath.ToSlash(filepath.Join(dir, "intro.html"))

	probe := runSearch(t, ctx, pageURL, "zorbulax")
	if strings.Contains(probe.Empty, "No results") {
		t.Errorf("file:// search reported no results instead of an unavailable index: %q", probe.Empty)
	}
	if !strings.Contains(probe.Empty, "mdpress serve") {
		t.Errorf("file:// search should say how to serve the site over http, got %q", probe.Empty)
	}
}

// TestSitePageLoadsSharedAssets checks that moving the stylesheet and the
// script out of every page into cacheable assets/ files did not break the page:
// the CSS has to apply and the script has to run, over http and from disk.
func TestSitePageLoadsSharedAssets(t *testing.T) {
	ctx := newBrowser(t)
	dir := browserTestSite(t)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	// The page itself must be small — the whole point of the split.
	page, err := os.ReadFile(filepath.Join(dir, "intro.html"))
	if err != nil {
		t.Fatalf("read intro.html: %v", err)
	}
	if len(page) > 40_000 {
		t.Errorf("intro.html is %d bytes; the shared assets should not be inlined", len(page))
	}

	const probeJS = `JSON.stringify({
      sidebar: getComputedStyle(document.querySelector('.sidebar')).position,
      search: typeof window.openSearch,
      active: (document.querySelector('.nav-item.active') || {}).textContent || ''
    })`
	for _, pageURL := range []string{
		srv.URL + "/intro.html",
		"file://" + filepath.ToSlash(filepath.Join(dir, "intro.html")),
	} {
		var raw string
		if err := chromedp.Run(ctx,
			chromedp.Navigate(pageURL),
			chromedp.WaitReady(".sidebar", chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Evaluate(probeJS, &raw),
		); err != nil {
			t.Fatalf("load %s: %v", pageURL, err)
		}
		var state struct {
			Sidebar string `json:"sidebar"`
			Search  string `json:"search"`
			Active  string `json:"active"`
		}
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			t.Fatalf("decode probe %q: %v", raw, err)
		}
		if state.Sidebar != "fixed" {
			t.Errorf("%s: stylesheet did not apply (sidebar position %q)", pageURL, state.Sidebar)
		}
		if state.Search != "function" {
			t.Errorf("%s: shared script did not run (openSearch is %q)", pageURL, state.Search)
		}
		if state.Active != "Intro" {
			t.Errorf("%s: script did not receive the page's active file (active nav %q)", pageURL, state.Active)
		}
	}
}

// TestSiteDarkModeHeadingContrast measures what a reader actually sees: h5 and
// h6 used to keep the light theme's near-black heading color on the dark
// background, 1.27:1, effectively invisible.
func TestSiteDarkModeHeadingContrast(t *testing.T) {
	ctx := newBrowser(t)
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Headings", Language: "en-US"})
	// Every theme colors all six heading levels (internal/theme.ToCSS), which
	// is exactly what the dark overrides have to answer for. Without it the
	// headings just inherit the body color and the bug is invisible.
	gen.SetCSS("h1, h2, h3, h4, h5, h6 {\n  color: #12344D;\n  font-weight: 600;\n}\n")
	gen.AddChapter(SiteChapter{
		Title:    "Levels",
		Filename: "levels.html",
		Content: "<h1>One</h1><h2>Two</h2><h3>Three</h3>" +
			"<h4>Four</h4><h5>Five</h5><h6>Six</h6>",
	})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	const probeJS = `(function() {
      document.documentElement.classList.add('dark');
      var out = {};
      ['h1','h2','h3','h4','h5','h6'].forEach(function(tag) {
        var el = document.querySelector('.content ' + tag);
        out[tag] = el ? getComputedStyle(el).color : '';
      });
      out.bg = getComputedStyle(document.body).backgroundColor;
      return JSON.stringify(out);
    })()`
	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/levels.html"),
		chromedp.WaitReady(".content h6", chromedp.ByQuery),
		chromedp.Evaluate(probeJS, &raw),
	); err != nil {
		t.Fatalf("load levels.html: %v", err)
	}
	var colors map[string]string
	if err := json.Unmarshal([]byte(raw), &colors); err != nil {
		t.Fatalf("decode probe %q: %v", raw, err)
	}
	bg := parseRGB(t, colors["bg"])
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		fg := parseRGB(t, colors[tag])
		if ratio := contrastRatio(fg, bg); ratio < 4.5 {
			t.Errorf("dark mode %s: %s on %s is %.2f:1, below the WCAG AA minimum of 4.5:1",
				tag, colors[tag], colors["bg"], ratio)
		}
	}
}

// parseRGB converts a computed "rgb(r, g, b)" color into the hex form the
// contrast helpers in site_a11y_test.go expect.
func parseRGB(t *testing.T, css string) string {
	t.Helper()
	open := strings.Index(css, "(")
	closeIdx := strings.Index(css, ")")
	if open < 0 || closeIdx < open {
		t.Fatalf("unexpected computed color %q", css)
	}
	parts := strings.Split(css[open+1:closeIdx], ",")
	if len(parts) < 3 {
		t.Fatalf("unexpected computed color %q", css)
	}
	hex := "#"
	for _, p := range parts[:3] {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			t.Fatalf("unexpected computed color %q: %v", css, err)
		}
		hex += string("0123456789abcdef"[v>>4]) + string("0123456789abcdef"[v&0xf])
	}
	return hex
}
