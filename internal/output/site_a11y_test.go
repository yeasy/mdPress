package output

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// cssRule is one declaration block from a generated stylesheet.
type cssRule struct {
	AtRule   string // enclosing at-rule prelude, "" at the top level
	Selector string
	Body     string
}

// styleSection returns the contents of the template's first <style> element
// with comments removed, so brace counting sees only real CSS.
func styleSection(tmpl string) string {
	start := strings.Index(tmpl, "<style>")
	end := strings.Index(tmpl, "</style>")
	if start < 0 || end < start {
		return ""
	}
	css := tmpl[start+len("<style>") : end]
	for {
		open := strings.Index(css, "/*")
		if open < 0 {
			return css
		}
		closing := strings.Index(css[open:], "*/")
		if closing < 0 {
			return css[:open]
		}
		css = css[:open] + css[open+closing+2:]
	}
}

// parseCSSRules walks a stylesheet and returns its declaration blocks. It is
// deliberately naive (brace counting only), which is enough for the
// generator's own hand-written CSS.
func parseCSSRules(css string) []cssRule {
	var rules []cssRule
	var atRules []string
	var prelude strings.Builder
	for i := 0; i < len(css); i++ {
		switch css[i] {
		case '{':
			head := normalizeSelector(prelude.String())
			prelude.Reset()
			if strings.HasPrefix(head, "@") {
				atRules = append(atRules, head)
				continue
			}
			depth := 1
			start := i + 1
			j := start
			for ; j < len(css) && depth > 0; j++ {
				if css[j] == '{' {
					depth++
				} else if css[j] == '}' {
					depth--
				}
			}
			rules = append(rules, cssRule{
				AtRule:   strings.Join(atRules, " "),
				Selector: head,
				Body:     css[start : j-1],
			})
			i = j - 1
		case '}':
			prelude.Reset()
			if len(atRules) > 0 {
				atRules = atRules[:len(atRules)-1]
			}
		default:
			prelude.WriteByte(css[i])
		}
	}
	return rules
}

func normalizeSelector(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var cssColorPattern = regexp.MustCompile(`(?:^|[;{\s])color:\s*(#[0-9a-fA-F]{3,8})`)

// ruleColor returns the `color` value declared by the rule with the given
// selector. The second result is false when no such rule (or no such
// declaration) exists, so a renamed selector fails the test instead of
// silently skipping it.
func ruleColor(rules []cssRule, selector string) (string, bool) {
	for _, rule := range rules {
		if rule.Selector != selector || strings.Contains(rule.AtRule, "print") {
			continue
		}
		if m := cssColorPattern.FindStringSubmatch(rule.Body); m != nil {
			return m[1], true
		}
	}
	return "", false
}

// relativeLuminance implements the WCAG 2.1 definition for an sRGB hex colour.
func relativeLuminance(hex string) float64 {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	channel := func(offset int) float64 {
		v, err := strconv.ParseUint(hex[offset:offset+2], 16, 8)
		if err != nil {
			return 0
		}
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(0) + 0.7152*channel(2) + 0.0722*channel(4)
}

func contrastRatio(fg, bg string) float64 {
	l1, l2 := relativeLuminance(fg), relativeLuminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func TestContrastRatioMatchesWCAGReference(t *testing.T) {
	// Black on white is the WCAG maximum, 21:1.
	if got := contrastRatio("#000000", "#ffffff"); math.Abs(got-21) > 0.01 {
		t.Errorf("contrastRatio(black, white) = %.2f, want 21", got)
	}
	// #767676 on white is the canonical "just passes AA" grey.
	if got := contrastRatio("#767676", "#ffffff"); got < 4.5 || got > 4.6 {
		t.Errorf("contrastRatio(#767676, white) = %.2f, want ~4.54", got)
	}
}

// TestSiteMutedTextMeetsAA checks every de-emphasised body-size text token in
// the generated site CSS against the WCAG AA 4.5:1 threshold, in both colour
// schemes. These are breadcrumbs, page footers, the search panel and the
// sidebar metadata: none of them is large text, so none of them qualifies for
// the relaxed 3:1 bar.
func TestSiteMutedTextMeetsAA(t *testing.T) {
	rules := parseCSSRules(styleSection(sitePageTemplate))

	const (
		lightPage    = "#ffffff"
		lightSidebar = "#fafafa"
		darkPage     = "#1e1e2e"
	)

	cases := []struct {
		selector   string
		background string
	}{
		{".sidebar-subtitle", lightSidebar},
		{".sidebar-author", lightSidebar},
		{".sidebar-description", lightSidebar},
		{".bc-sep", lightPage},
		{".page-breadcrumb", lightPage},
		{".page-toc-nav a", lightPage},
		{".page-meta", lightPage},
		{".build-meta", lightPage},
		{".page-nav .nav-label", lightPage},
		{".search-result-path", lightPage},
		{".search-empty", lightPage},
		{".search-footer", lightPage},
		{"html.dark .bc-sep", darkPage},
		{"html.dark .page-meta, html.dark .build-meta", darkPage},
		{"html.dark .page-toc-nav a", darkPage},
		{"html.dark .search-result-path", darkPage},
		{"html.dark .search-empty", darkPage},
		{"html.dark .search-status", darkPage},
	}

	for _, tc := range cases {
		fg, ok := ruleColor(rules, tc.selector)
		if !ok {
			t.Errorf("no color declaration found for %q; update the selector list if it was renamed", tc.selector)
			continue
		}
		if ratio := contrastRatio(fg, tc.background); ratio < 4.5 {
			t.Errorf("%s: %s on %s is %.2f:1, below the WCAG AA minimum of 4.5:1",
				tc.selector, fg, tc.background, ratio)
		}
	}
}

func TestSitePrintStylesKeepSyntaxHighlighting(t *testing.T) {
	var printCSS string
	for _, rule := range parseCSSRules(styleSection(sitePageTemplate)) {
		if strings.Contains(rule.AtRule, "print") {
			printCSS += rule.Selector + "{" + rule.Body + "}\n"
		}
	}
	if printCSS == "" {
		t.Fatal("no @media print rules found")
	}
	if strings.Contains(printCSS, "\n*{") || strings.HasPrefix(printCSS, "*{") {
		t.Error("a blanket `* { color: black !important }` flattens every printed code block to monochrome")
	}
	if !strings.Contains(printCSS, "*:not(pre):not(pre *){") {
		t.Error("the print colour reset should exempt code blocks")
	}
	if strings.Contains(printCSS, "a::after{") {
		t.Error("printing the href of every relative link buries the text in paths; restrict it to external links")
	}
	if !strings.Contains(printCSS, `a[href^="http"]::after{`) {
		t.Error("external link hrefs should still be printed")
	}
}

func TestSiteMobileHeaderStaysCompact(t *testing.T) {
	var mobileCSS string
	for _, rule := range parseCSSRules(styleSection(sitePageTemplate)) {
		if strings.Contains(rule.AtRule, "max-width: 768px") {
			mobileCSS += rule.Selector + "{" + rule.Body + "}\n"
		}
	}
	if mobileCSS == "" {
		t.Fatal("no phone-width media query found")
	}
	// Without these the breadcrumb wraps onto extra lines and grows the sticky
	// header until it covers a quarter of the viewport.
	if !strings.Contains(mobileCSS, "text-overflow: ellipsis") {
		t.Error("phone breadcrumbs should truncate rather than wrap")
	}
	if !strings.Contains(mobileCSS, "min-width: 0") {
		t.Error("the breadcrumb needs min-width:0 for truncation to take effect inside a flex header")
	}
}

func TestSiteAnchorOffsetTracksHeaderHeight(t *testing.T) {
	if !strings.Contains(sitePageTemplate, "scroll-margin-top: var(--header-h, 64px)") {
		t.Error("anchor scroll offset should follow the measured header height, not a fixed 64px")
	}
	if !strings.Contains(sitePageTemplate, "setProperty('--header-h'") {
		t.Error("the page script should measure the header and publish --header-h")
	}
	if !strings.Contains(sitePageTemplate, "ResizeObserver") {
		t.Error("--header-h should be re-measured when the header resizes")
	}
}

func TestSiteCodeBlocksShowLanguageLabel(t *testing.T) {
	if !strings.Contains(sitePageTemplate, ".code-wrapper[data-lang]::before") {
		t.Error("site code blocks should render a language label like the standalone HTML build does")
	}
	if !strings.Contains(sitePageTemplate, "content: attr(data-lang)") {
		t.Error("the language label should come from the data-lang attribute")
	}
	if !strings.Contains(sitePageTemplate, "wrapper.setAttribute('data-lang', lang)") {
		t.Error("addCopyButtons should copy the fence language onto the wrapper")
	}
}
