package output

import (
	"context"
	"encoding/json"
	"fmt"
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

// crossDepthProbePNG is a minimal valid 1x1 PNG used as a loadable content image.
var crossDepthProbePNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
	0x0C, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
	0x00, 0x05, 0xFE, 0x02, 0xFE, 0x0D, 0xEF, 0x46, 0xB8, 0x00, 0x00, 0x00,
	0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
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

// TestSiteSPACrossDepthImageResolves guards that a content image loads after an
// SPA navigation between pages at different directory depths. It is easy to
// assume the swap resolves the incoming page's relative image URLs against the
// previous page's base (the URL innerHTML is assigned under), which would 404 a
// "assets/pic.png" navigated to from a deeper page. In practice the browser
// defers the image load until after finalizeNavigation's pushState updates the
// base, so the image resolves against the target page and loads — this test
// pins that behavior (both directions, and under subdirectory hosting) so a
// future change to the swap ordering cannot silently break cross-depth images.
func TestSiteSPACrossDepthImageResolves(t *testing.T) {
	ctx := newBrowser(t)

	// clickJS clicks the sidebar link ending in %s, waits for the swapped-in
	// page's #probe image to settle, and reports whether it loaded. __spaMarker
	// survives a client-side swap but not a full reload, so it also proves the
	// swap path — not a fallback full navigation — is what ran.
	clickJS := func(linkSuffix string) string {
		return `(async function() {
      window.__spaMarker = 'alive';
      var link = document.querySelector('.sidebar a[href$="` + linkSuffix + `"]');
      if (!link) return JSON.stringify({ err: 'no-link' });
      link.click();
      var img = null;
      for (var i = 0; i < 100; i++) {
        img = document.getElementById('probe');
        if (img && img.complete && (img.naturalWidth > 0 || img.currentSrc)) break;
        await new Promise(function(r) { setTimeout(r, 50); });
      }
      if (!img) return JSON.stringify({ err: 'no-img' });
      return JSON.stringify({ natural: img.naturalWidth, currentSrc: img.currentSrc, spa: window.__spaMarker === 'alive' });
    })()`
	}

	// probe drives one navigation and returns the settled image state.
	type imageState struct {
		Natural    int    `json:"natural"`
		CurrentSrc string `json:"currentSrc"`
		SPA        bool   `json:"spa"`
		Err        string `json:"err"`
	}
	probe := func(t *testing.T, startURL, linkSuffix string) imageState {
		t.Helper()
		var raw string
		if err := chromedp.Run(ctx,
			chromedp.Navigate(startURL),
			chromedp.WaitReady(".sidebar", chromedp.ByQuery),
			chromedp.Sleep(500*time.Millisecond),
			chromedp.Evaluate(clickJS(linkSuffix), &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
				return p.WithAwaitPromise(true)
			}),
		); err != nil {
			t.Fatalf("navigation failed: %v", err)
		}
		var res imageState
		if err := json.Unmarshal([]byte(raw), &res); err != nil {
			t.Fatalf("decode probe %q: %v", raw, err)
		}
		if res.Err != "" {
			t.Fatalf("probe error: %s", res.Err)
		}
		if !res.SPA {
			t.Fatal("navigation fell back to a full reload; the SPA swap path was not exercised")
		}
		return res
	}

	// buildSite writes a three-chapter book with a real root-level asset and one
	// chapter that carries the #probe image at imgSrc.
	buildSite := func(t *testing.T, imgChapter SiteChapter) string {
		t.Helper()
		dir := t.TempDir()
		gen := NewSiteGenerator(SiteMeta{Title: "Depth", Language: "en-US"})
		gen.AddChapter(SiteChapter{Title: "Home", Filename: "home.html", Content: "<h1>Home</h1><p>home body.</p>"})
		gen.AddChapter(imgChapter)
		gen.AddChapter(SiteChapter{Title: "Deep", Filename: "part/deep.html", Content: "<h1>Deep</h1><p>deep body.</p>"})
		if err := gen.Generate(dir); err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "assets", "pic.png"), crossDepthProbePNG, 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	t.Run("deeper_to_shallower", func(t *testing.T) {
		dir := buildSite(t, SiteChapter{Title: "Root Image", Filename: "rootimg.html",
			Content: `<h1>Root Image</h1><img id="probe" src="assets/pic.png" alt="p">`})
		srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
		defer srv.Close()
		// Start deep (base .../part/), navigate to the root-depth image page.
		res := probe(t, srv.URL+"/part/deep.html", "rootimg.html")
		if strings.Contains(res.CurrentSrc, "/part/") {
			t.Errorf("image fetched from %q (the previous page's base), not root depth", res.CurrentSrc)
		}
		if res.Natural == 0 {
			t.Errorf("image did not load (naturalWidth=0, currentSrc=%q)", res.CurrentSrc)
		}
	})

	t.Run("shallower_to_deeper_subdir_hosting", func(t *testing.T) {
		dir := buildSite(t, SiteChapter{Title: "Deep Image", Filename: "part/deepimg.html",
			Content: `<h1>Deep Image</h1><img id="probe" src="../assets/pic.png" alt="p">`})
		// Serve under /repo/ to mimic GitHub Pages project hosting.
		mux := http.NewServeMux()
		mux.Handle("/repo/", http.StripPrefix("/repo/", http.FileServer(http.Dir(dir))))
		srv := httptest.NewServer(mux)
		defer srv.Close()
		// Start at the root page, navigate to the deep image page.
		res := probe(t, srv.URL+"/repo/home.html", "deepimg.html")
		if !strings.Contains(res.CurrentSrc, "/repo/assets/") {
			t.Errorf("image fetched from %q, want it under /repo/assets/ (the extra ../ was not clamped)", res.CurrentSrc)
		}
		if res.Natural == 0 {
			t.Errorf("image did not load (naturalWidth=0, currentSrc=%q)", res.CurrentSrc)
		}
	})
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

// TestSiteEscapeAndMobileSidebarStateTrap pins two coupled sidebar-state
// defects. Pressing Escape to close the search overlay used to fall through to
// the document-level handler, which collapsed the sidebar and persisted that
// choice to localStorage; on a phone that persisted flag then pinned the
// drawer off-screen forever, because body.sidebar-collapsed's
// translateX(-100%) outranks .sidebar.open — the reader got the dark backdrop
// with no sidebar and no way out.
func TestSiteEscapeAndMobileSidebarStateTrap(t *testing.T) {
	ctx := newBrowser(t)
	dir := browserTestSite(t)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	const escProbe = `(async function() {
      try { localStorage.clear(); } catch (e) {}
      window.openSearch();
      var input = document.getElementById('search-input');
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
      await new Promise(function(r) { setTimeout(r, 150); });
      var ls = null;
      try { ls = localStorage.getItem('mdpress-sidebar-collapsed'); } catch (e) {}
      return JSON.stringify({
        collapsed: document.body.classList.contains('sidebar-collapsed'),
        persisted: ls
      });
    })()`
	var raw string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/intro.html"),
		chromedp.WaitReady("#search-input", chromedp.ByQuery),
		chromedp.Evaluate(escProbe, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	); err != nil {
		t.Fatalf("escape probe failed: %v", err)
	}
	var esc struct {
		Collapsed bool    `json:"collapsed"`
		Persisted *string `json:"persisted"`
	}
	if err := json.Unmarshal([]byte(raw), &esc); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if esc.Collapsed || (esc.Persisted != nil && *esc.Persisted == "1") {
		t.Errorf("closing search with Escape collapsed the sidebar (collapsed=%v persisted=%v)", esc.Collapsed, esc.Persisted)
	}

	// Mobile: with the trap state pre-seeded, opening the drawer must clear it.
	const mobileProbe = `(async function() {
      try { localStorage.setItem('mdpress-sidebar-collapsed', '1'); } catch (e) {}
      document.body.classList.add('sidebar-collapsed');
      document.querySelector('.sidebar-toggle').click();
      await new Promise(function(r) { setTimeout(r, 350); });
      var sb = document.querySelector('.sidebar');
      var x = new DOMMatrixReadOnly(getComputedStyle(sb).transform).m41;
      return JSON.stringify({ open: sb.classList.contains('open'), x: x,
        collapsed: document.body.classList.contains('sidebar-collapsed') });
    })()`
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(375, 812),
		chromedp.Navigate(srv.URL+"/intro.html"),
		chromedp.WaitReady(".sidebar-toggle", chromedp.ByQuery),
		chromedp.Evaluate(mobileProbe, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
		chromedp.EmulateViewport(0, 0),
	); err != nil {
		t.Fatalf("mobile probe failed: %v", err)
	}
	var mob struct {
		Open      bool    `json:"open"`
		X         float64 `json:"x"`
		Collapsed bool    `json:"collapsed"`
	}
	if err := json.Unmarshal([]byte(raw), &mob); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if !mob.Open {
		t.Fatal("tapping the toggle did not mark the drawer open")
	}
	if mob.X < 0 {
		t.Errorf("drawer is open but still translated off-screen (x=%v, sidebar-collapsed=%v)", mob.X, mob.Collapsed)
	}
}

// TestSiteProseMeasureIsCapped pins the reading measure on wide screens. The
// content column grows to 1200px at 1920px viewports, and prose used to grow
// with it — ~158 characters per line, double the readable range — while the
// print-point theme size rendered body text at 14.7px. Prose now holds ~76ch;
// tables, code blocks and figures keep the full column, which is what
// actually benefits from the width.
func TestSiteProseMeasureIsCapped(t *testing.T) {
	ctx := newBrowser(t)
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Measure", Language: "en-US"})
	long := strings.Repeat("Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod. ", 8)
	gen.AddChapter(SiteChapter{
		Title: "One", Filename: "one.html",
		Content: `<h1>One</h1><p id="probe">` + long + `</p><pre><code>` + strings.Repeat("wide ", 60) + `</code></pre>`,
	})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1920, 1000),
		chromedp.Navigate(srv.URL+"/one.html"),
		chromedp.WaitReady("#probe", chromedp.ByQuery),
		chromedp.Evaluate(`JSON.stringify({
			p: document.getElementById('probe').getBoundingClientRect().width,
			pre: document.querySelector('.content pre').getBoundingClientRect().width
		})`, &raw),
		chromedp.EmulateViewport(0, 0),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	var r struct{ P, Pre float64 }
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if r.P > 850 {
		t.Errorf("prose measure is uncapped on a wide viewport: %.0fpx", r.P)
	}
	if r.Pre <= r.P+50 {
		t.Errorf("code blocks should keep the full column width (pre=%.0fpx, prose=%.0fpx)", r.Pre, r.P)
	}
}

// TestSiteSidebarScrollsToActiveOnLoad reproduces the fresh-load race: the
// nav-activation pass runs from the bottom-of-body script, which can execute
// before the external stylesheet has applied, so scrollIntoView computed the
// active item's position against unstyled layout and the sidebar stayed at
// scrollTop 0 with the highlight far below the fold. SPA navigation — running
// against settled layout — always scrolled correctly. The stylesheet is
// deliberately delayed here to force that ordering deterministically.
func TestSiteSidebarScrollsToActiveOnLoad(t *testing.T) {
	ctx := newBrowser(t)
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Deep Nav", Language: "en-US"})
	for i := 1; i <= 80; i++ {
		gen.AddChapter(SiteChapter{
			Title:    fmt.Sprintf("Chapter %02d", i),
			Filename: fmt.Sprintf("ch%02d.html", i),
			Content:  fmt.Sprintf("<h1>Chapter %02d</h1><p>Body.</p>", i),
		})
	}
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	files := http.FileServer(http.Dir(dir))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			time.Sleep(300 * time.Millisecond)
		}
		files.ServeHTTP(w, r)
	}))
	defer srv.Close()

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1280, 800),
		chromedp.Navigate(srv.URL+"/ch70.html"),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`new Promise(function(resolve) {
			function probe() {
				// Give the load handler a frame to run after the load event.
				setTimeout(function() {
					var sb = document.querySelector('.sidebar');
					var active = document.querySelector('.nav-item.active');
					var sbRect = sb.getBoundingClientRect();
					var aRect = active ? active.getBoundingClientRect() : null;
					resolve(JSON.stringify({
						scrollTop: sb.scrollTop,
						visible: aRect !== null && aRect.top >= sbRect.top && aRect.bottom <= sbRect.bottom
					}));
				}, 100);
			}
			if (document.readyState === 'complete') { probe(); }
			else { window.addEventListener('load', probe); }
		})`, &raw, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
		chromedp.EmulateViewport(0, 0),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	var r struct {
		ScrollTop float64
		Visible   bool
	}
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if r.ScrollTop == 0 || !r.Visible {
		t.Errorf("active sidebar item is not in view after initial load (scrollTop=%.0f, visible=%v)", r.ScrollTop, r.Visible)
	}
}

// TestSiteWideTablesScrollOnMobile: cells carried word-break:break-word, so on
// a phone a table wider than the viewport crushed its columns and broke
// identifiers character by character instead of scrolling. Tables are now
// wrapped in an overflow container; the long token must stay on one line and
// the page itself must not overflow sideways.
func TestSiteWideTablesScrollOnMobile(t *testing.T) {
	ctx := newBrowser(t)
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Tables", Language: "en-US"})
	gen.AddChapter(SiteChapter{
		Title: "Wide", Filename: "wide.html",
		Content: `<h1>Wide</h1><table><tr><th>Setting</th><th>Description</th></tr>` +
			`<tr><td id="tok">EnableExperimentalCrossReferenceResolutionStrategy</td>` +
			`<td>Turns on the very long experimental resolver flag documented here.</td></tr></table>`,
	})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	var raw string
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(375, 812, chromedp.EmulateScale(1)),
		chromedp.Navigate(srv.URL+"/wide.html"),
		chromedp.WaitReady("#tok", chromedp.ByQuery),
		chromedp.Evaluate(`(function() {
			var td = document.getElementById('tok');
			var wrap = td.closest('.table-scroll');
			// Count the line boxes the identifier occupies: an unbroken token
			// is exactly one; the old word-break:break-word shredded it into
			// several. (The cell's own height is useless here — the row is as
			// tall as its tallest sibling cell.)
			var range = document.createRange();
			range.selectNodeContents(td);
			var lines = range.getClientRects().length;
			return JSON.stringify({
				wrapped: wrap !== null,
				scrolls: wrap !== null && wrap.scrollWidth > wrap.clientWidth + 1,
				shredded: lines > 1,
				pageOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1
			});
		})()`, &raw),
		chromedp.EmulateViewport(0, 0),
	); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	var r struct{ Wrapped, Scrolls, Shredded, PageOverflow bool }
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if !r.Wrapped || !r.Scrolls {
		t.Errorf("wide table should scroll inside a wrapper (wrapped=%v, scrolls=%v)", r.Wrapped, r.Scrolls)
	}
	if r.Shredded {
		t.Error("long identifier in a table cell is still broken character by character")
	}
	if r.PageOverflow {
		t.Error("wide table overflows the page instead of its own container")
	}
}
