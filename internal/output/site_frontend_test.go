package output

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// siteAssetRefPattern finds the shared stylesheet and script a generated page
// links to.
var siteAssetRefPattern = regexp.MustCompile(`(?:<link rel="stylesheet" href="|<script src=")([^"]+)"`)

// siteSourceBundle is everything a generated page is made of: the markup
// template plus the stylesheet and the script that now live in assets/ instead
// of being inlined into every page. Source-level assertions have to search this
// rather than sitePageTemplate alone.
var siteSourceBundle = sitePageTemplate + siteStylesheet("") + siteScriptJS

// siteCSSSource is the site stylesheet with comments stripped, for tests that
// parse declarations out of it.
func siteCSSSource() string {
	return styleSection("<style>" + siteStylesheet("") + "</style>")
}

// readSiteBundle returns a generated page concatenated with the shared
// stylesheet and script it links to, so a test can assert on markup, CSS and
// behavior in one string the way it could when all three were inlined.
func readSiteBundle(t *testing.T, dir, name string) string {
	t.Helper()
	page := readSiteFile(t, dir, name)
	pageDir := filepath.Dir(filepath.Join(dir, filepath.FromSlash(name)))
	var b strings.Builder
	b.WriteString(page)
	for _, m := range siteAssetRefPattern.FindAllStringSubmatch(page, -1) {
		data, err := os.ReadFile(filepath.Join(pageDir, filepath.FromSlash(m[1])))
		if err != nil {
			t.Fatalf("read linked asset %s of %s failed: %v", m[1], name, err)
		}
		b.WriteString("\n")
		b.Write(data)
	}
	return b.String()
}

// TestSidebarShowsDeeplyNestedChapters guards the sidebar against the depth cap
// that used to silently hide everything below the second level: the pages were
// generated, listed in sitemap.xml and indexed for search, but no page linked
// to them.
func TestSidebarShowsDeeplyNestedChapters(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: "en-US"})
	chapters := []SiteChapter{
		{Title: "Part 1", Filename: "part1.html", Children: []SiteChapter{
			{Title: "Chapter", Filename: "chapter.html", Depth: 1, Children: []SiteChapter{
				{Title: "Section", Filename: "section.html", Depth: 2, Children: []SiteChapter{
					{Title: "Subsection", Filename: "subsection.html", Depth: 3},
				}},
			}},
		}},
	}

	for _, active := range []string{"part1.html", "subsection.html"} {
		sidebar := gen.buildSidebar(chapters, active)
		for _, want := range []string{"chapter.html", "section.html", "subsection.html"} {
			if !strings.Contains(sidebar, `data-file="`+want+`"`) {
				t.Errorf("sidebar rendered for %s omits %s:\n%s", active, want, sidebar)
			}
		}
	}

	// The whole branch leading to the active page must be expanded, otherwise
	// the deep entries are in the markup but never revealed.
	sidebar := gen.buildSidebar(chapters, "subsection.html")
	for _, group := range []string{"part1.html", "chapter.html", "section.html"} {
		if !strings.Contains(sidebar, `class="nav-group expanded" data-group-file="`+group+`"`) {
			t.Errorf("ancestor group %s of the active page should be expanded:\n%s", group, sidebar)
		}
	}
}

// TestSidebarStopsRecursingOnCyclicChapters checks the depth limit still ends
// the walk, so a malformed chapter tree cannot hang the build.
func TestSidebarStopsRecursingOnCyclicChapters(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: "en-US"})
	deepest := SiteChapter{Title: "Leaf", Filename: "leaf.html"}
	ch := deepest
	for i := 0; i < maxSidebarChapterDepth+5; i++ {
		ch = SiteChapter{Title: "Level", Filename: "level.html", Children: []SiteChapter{ch}}
	}
	sidebar := gen.buildSidebar([]SiteChapter{ch}, "leaf.html")
	if got := strings.Count(sidebar, `data-file="level.html"`); got > maxSidebarChapterDepth {
		t.Errorf("sidebar recursed %d levels, past the %d-level guard", got, maxSidebarChapterDepth)
	}
}

// TestSiteSearchJSKeepsHelpers is a source change-detector, NOT behavioral
// coverage. It only asserts that siteScriptJS still contains the query-parsing
// and matching helpers by name and wires them into doSearch; it cannot tell
// whether search actually works. A regression that keeps every substring below
// but breaks the logic — e.g. gutting the per-character CJK fallback inside
// termMatches so an unspaced 数据库索引 query matches nothing — passes this test.
// The real behavior (quoted phrases and unspaced CJK queries) is verified by the
// Chrome-driven TestSiteSearchInBrowser. Read a green run here, in a Chrome-less
// environment, as "the search source was not reverted", never as "search works".
func TestSiteSearchJSKeepsHelpers(t *testing.T) {
	js := siteScriptJS
	if !strings.Contains(js, "function parseSearchTerms(raw)") {
		t.Error("search should parse the query into terms, keeping quoted runs whole")
	}
	if !strings.Contains(js, `.replace(/"/g, '')`) {
		t.Error("stray quote characters must be stripped from a term, not searched for")
	}
	if !strings.Contains(js, "function termMatches(haystack, term)") {
		t.Error("search should match a term through termMatches, which handles CJK")
	}
	if !strings.Contains(js, "var terms = parseSearchTerms(qLower);") {
		t.Error("doSearch should build its terms with parseSearchTerms")
	}
	if !strings.Contains(js, "if (!termMatches(h, terms[t])) { return false; }") {
		t.Error("doSearch should test each term with termMatches, not a raw indexOf")
	}
	if strings.Contains(js, `var terms = qLower.split(/\s+/)`) {
		t.Error("the whitespace-only tokenizer should be gone; it broke quoted phrases")
	}
}

// TestSiteSearchIndexFailureJSKeepsErrorPath is a source change-detector, NOT
// behavioral coverage. It only asserts that loadIndex's failure-handling
// substrings are still present in siteScriptJS; it never performs a failed index
// load, so it cannot confirm the failure is actually surfaced. The behavior — a
// file:// preview reporting the index as unavailable instead of silently
// answering "No results" — is verified by the Chrome-driven
// TestSiteSearchOverFileProtocol. A green run here, in a Chrome-less
// environment, means only that the error-path source was not reverted.
func TestSiteSearchIndexFailureJSKeepsErrorPath(t *testing.T) {
	js := siteScriptJS
	if strings.Contains(js, "searchIndex = [];") {
		t.Error("a failed index load must not resolve as an empty index")
	}
	if !strings.Contains(js, "searchIndexError = err;") || !strings.Contains(js, "throw err;") {
		t.Error("loadIndex should remember and rethrow the failure so doSearch can report it")
	}
	if !strings.Contains(js, "__ui.searchUnavailable + hint") {
		t.Error("the failure path should render the search-unavailable message")
	}
	if !strings.Contains(js, "<code>mdpress serve</code>") {
		t.Error("a file:// page should be told how to serve the site over http")
	}
}

// TestSiteDarkModeCoversEveryHeadingLevel guards against a repeat of h5/h6
// keeping the light theme's near-black heading color on the dark background,
// which measured 1.27:1 — invisible, and never seen by an author working in
// light mode. The theme CSS colors h1..h6, so the dark overrides must too.
func TestSiteDarkModeCoversEveryHeadingLevel(t *testing.T) {
	rules := parseCSSRules(siteCSSSource())
	for _, level := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		selector := "html.dark .content " + level
		var found bool
		for _, rule := range rules {
			if strings.Contains(rule.AtRule, "print") || !ruleHasSelector(rule, selector) {
				continue
			}
			if cssColorPattern.MatchString(rule.Body) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no dark-mode color for %q; it would inherit the light theme's heading color", selector)
		}
	}
}

// ruleHasSelector reports whether a parsed rule applies to selector.
func ruleHasSelector(rule cssRule, selector string) bool {
	for _, s := range strings.Split(rule.Selector, ",") {
		if strings.TrimSpace(s) == selector {
			return true
		}
	}
	return false
}

// TestSiteSharedAssetsAreFilesNotInlined covers the size blow-up: every page
// used to carry its own copy of the ~105 KB stylesheet and script, so a
// 30-chapter book shipped 3 MB of duplicated, uncacheable boilerplate.
func TestSiteSharedAssetsAreFilesNotInlined(t *testing.T) {
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{Title: "Test Book", Language: "en-US"})
	gen.SetCSS(".custom-theme-marker { color: #123456; }")
	gen.AddChapter(SiteChapter{Title: "Chapter 1", Filename: "ch1.html", Content: "<h1>Chapter 1</h1>"})
	gen.AddChapter(SiteChapter{Title: "Chapter 2", Filename: "part/ch2.html", Content: "<h1>Chapter 2</h1>"})
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, siteAssetDir))
	if err != nil {
		t.Fatalf("read asset dir failed: %v", err)
	}
	var css, js string
	for _, e := range entries {
		switch filepath.Ext(e.Name()) {
		case ".css":
			css = e.Name()
		case ".js":
			js = e.Name()
		}
	}
	if css == "" || js == "" {
		t.Fatalf("expected a stylesheet and a script in %s, got %v", siteAssetDir, entries)
	}

	for _, page := range []string{"index.html", "ch1.html", "part/ch2.html"} {
		html := readSiteFile(t, dir, page)
		if strings.Contains(html, ".nav-children-inner {") {
			t.Errorf("%s still inlines the stylesheet", page)
		}
		if strings.Contains(html, "function parseSearchTerms(") {
			t.Errorf("%s still inlines the site script", page)
		}
		if len(html) > 40_000 {
			t.Errorf("%s is %d bytes; the shared assets should no longer be part of it", page, len(html))
		}
		// The link has to resolve from the page's own directory, so the site
		// keeps working from a subdirectory and from file://.
		prefix := strings.Repeat("../", strings.Count(page, "/"))
		if !strings.Contains(html, `<link rel="stylesheet" href="`+prefix+siteAssetDir+"/"+css+`">`) {
			t.Errorf("%s does not link the shared stylesheet at the right depth", page)
		}
		if !strings.Contains(html, `<script src="`+prefix+siteAssetDir+"/"+js+`">`) {
			t.Errorf("%s does not link the shared script at the right depth", page)
		}
	}

	// The theme CSS a user configured has to survive the move out of the page.
	sheet, err := os.ReadFile(filepath.Join(dir, siteAssetDir, css))
	if err != nil {
		t.Fatalf("read stylesheet failed: %v", err)
	}
	if !strings.Contains(string(sheet), ".custom-theme-marker") {
		t.Error("the configured theme CSS should be part of the shared stylesheet")
	}
}

// TestSiteSharedAssetNamesTrackContent checks the content hash in the asset
// names, which is what lets a host cache them forever without a reader ever
// getting the previous build's script against this build's markup.
func TestSiteSharedAssetNamesTrackContent(t *testing.T) {
	name := func(t *testing.T, css string) string {
		t.Helper()
		dir := t.TempDir()
		shared, err := writeSharedSiteAssets(dir, css)
		if err != nil {
			t.Fatalf("writeSharedSiteAssets failed: %v", err)
		}
		return shared.stylesheet
	}
	first := name(t, "body { color: #111; }")
	if first != name(t, "body { color: #111; }") {
		t.Error("the same stylesheet should keep the same name across builds")
	}
	if first == name(t, "body { color: #222; }") {
		t.Error("a changed stylesheet must get a new name, or caches serve the old one")
	}
}
