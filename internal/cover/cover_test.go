package cover

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// mustTheme fetches a built-in theme by name, failing the test on error.
func mustTheme(t *testing.T, name string) *theme.Theme {
	t.Helper()
	thm, err := theme.NewThemeManager().Get(name)
	if err != nil {
		t.Fatalf("failed to load built-in theme %q: %v", name, err)
	}
	return thm
}

// TestNewCoverGenerator verifies cover generator creation.
func TestNewCoverGenerator(t *testing.T) {
	meta := config.BookMeta{Title: "Test Book"}
	gen := NewCoverGenerator(meta, nil)
	if gen == nil {
		t.Fatal("NewCoverGenerator returned nil")
	}
}

// TestRenderHTMLBasic verifies basic cover rendering.
func TestRenderHTMLBasic(t *testing.T) {
	meta := config.BookMeta{
		Title:   "My Book",
		Author:  "Author",
		Version: "1.0.0",
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "My Book") {
		t.Error("cover should include the title")
	}
	if !strings.Contains(html, "Author") {
		t.Error("cover should include the author")
	}
	if !strings.Contains(html, "1.0.0") {
		t.Error("cover should include the version")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("cover should be a complete HTML document")
	}
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should contain the cover-page class")
	}
}

// TestRenderHTMLWithSubtitle verifies subtitle rendering.
func TestRenderHTMLWithSubtitle(t *testing.T) {
	meta := config.BookMeta{
		Title:    "Main Title",
		Subtitle: "Subtitle Content",
		Author:   "Author",
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Subtitle Content") {
		t.Error("cover should include the subtitle")
	}
	if !strings.Contains(html, "cover-subtitle") {
		t.Error("cover should include the subtitle class")
	}
}

// TestRenderHTMLWithBackground verifies background color rendering.
func TestRenderHTMLWithBackground(t *testing.T) {
	meta := config.BookMeta{
		Title: "Test",
		Cover: config.CoverMeta{Background: "#1a1a2e"},
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "#1a1a2e") {
		t.Error("cover should include the configured background color")
	}
}

// TestRenderHTMLWithCoverImage verifies cover image rendering.
func TestRenderHTMLWithCoverImage(t *testing.T) {
	meta := config.BookMeta{
		Title: "Test",
		Cover: config.CoverMeta{Image: "cover.png"},
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "cover.png") {
		t.Error("cover should include the configured cover image")
	}
	if !strings.Contains(html, "background-image") {
		t.Error("cover should include background image styles")
	}
}

// TestRenderHTMLDefaultNavyCover verifies the default navy cover with light
// text when no theme and no cover background/image are configured.
func TestRenderHTMLDefaultNavyCover(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "background-color: #102a43") {
		t.Error("default cover should use the deep navy background")
	}
	if !strings.Contains(html, "color: #f6f8fc") {
		t.Error("default cover should use near-white text on the navy background")
	}
	if strings.Contains(html, "color: #14304a") {
		t.Error("default cover should not use the light-background ink")
	}
}

// TestRenderHTMLThemeDefaults verifies that the default cover adapts to the
// active theme when no cover background/image is configured.
func TestRenderHTMLThemeDefaults(t *testing.T) {
	tests := []struct {
		name        string
		thm         *theme.Theme
		wantBg      string
		wantInk     string
		wantFont    string
		wantDivider string
	}{
		{
			name:        "nil theme keeps navy",
			thm:         nil,
			wantBg:      "background-color: #102a43",
			wantInk:     "color: #f6f8fc",
			wantFont:    `font-family: -apple-system, BlinkMacSystemFont, "Segoe UI"`,
			wantDivider: "rgba(255, 255, 255, 0.5)",
		},
		{
			name:        "technical keeps navy",
			thm:         mustTheme(t, "technical"),
			wantBg:      "background-color: #102a43",
			wantInk:     "color: #f6f8fc",
			wantFont:    "font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC'",
			wantDivider: "rgba(255, 255, 255, 0.5)",
		},
		{
			name:        "elegant gets warm serif cover",
			thm:         mustTheme(t, "elegant"),
			wantBg:      "background-color: #33261D",
			wantInk:     "color: #F5EDDF",
			wantFont:    "font-family: 'Songti SC'",
			wantDivider: "rgba(255, 255, 255, 0.5)",
		},
		{
			name:        "minimal gets light cover with near-black ink",
			thm:         mustTheme(t, "minimal"),
			wantBg:      "background-color: #FAFAFA",
			wantInk:     "color: #111111",
			wantFont:    "font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC'",
			wantDivider: "background-color: #D4D4D4",
		},
		{
			name: "unknown custom theme falls back to navy",
			thm: &theme.Theme{
				Name:       "corporate",
				FontFamily: "'Custom Font', sans-serif",
			},
			wantBg:      "background-color: #102a43",
			wantInk:     "color: #f6f8fc",
			wantFont:    "font-family: 'Custom Font', sans-serif",
			wantDivider: "rgba(255, 255, 255, 0.5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := config.BookMeta{Title: "Test"}
			html := NewCoverGenerator(meta, tt.thm).RenderHTML()

			if !strings.Contains(html, tt.wantBg) {
				t.Errorf("cover should contain %q", tt.wantBg)
			}
			if !strings.Contains(html, tt.wantInk) {
				t.Errorf("cover should contain text ink %q", tt.wantInk)
			}
			if !strings.Contains(html, tt.wantFont) {
				t.Errorf("cover should contain font stack %q", tt.wantFont)
			}
			if !strings.Contains(html, tt.wantDivider) {
				t.Errorf("cover should contain divider style %q", tt.wantDivider)
			}
		})
	}
}

// TestRenderHTMLLightBackgroundsGetDarkText verifies that configured light
// backgrounds — hex, named, and rgb() forms — yield dark text instead of the
// near-invisible near-white default.
func TestRenderHTMLLightBackgroundsGetDarkText(t *testing.T) {
	backgrounds := []string{
		"white",
		"ivory",
		"rgb(255, 255, 255)",
		"rgba(255, 255, 255, 0.9)",
		"#ffffff",
		"#FFF",
	}

	for _, bg := range backgrounds {
		t.Run(bg, func(t *testing.T) {
			meta := config.BookMeta{
				Title: "Test",
				Cover: config.CoverMeta{Background: bg},
			}
			html := NewCoverGenerator(meta, nil).RenderHTML()

			if !strings.Contains(html, "color: #14304a") {
				t.Errorf("background %q should yield dark text ink", bg)
			}
			if strings.Contains(html, "color: #f6f8fc") {
				t.Errorf("background %q should not yield near-white text", bg)
			}
		})
	}
}

// TestRenderHTMLDarkBackgroundsGetLightText verifies that configured dark
// backgrounds keep light text.
func TestRenderHTMLDarkBackgroundsGetLightText(t *testing.T) {
	backgrounds := []string{
		"black",
		"navy",
		"rgb(16, 42, 67)",
		"#1a1a2e",
	}

	for _, bg := range backgrounds {
		t.Run(bg, func(t *testing.T) {
			meta := config.BookMeta{
				Title: "Test",
				Cover: config.CoverMeta{Background: bg},
			}
			html := NewCoverGenerator(meta, nil).RenderHTML()

			if !strings.Contains(html, "color: #f6f8fc") {
				t.Errorf("background %q should yield near-white text ink", bg)
			}
		})
	}
}

// TestRenderHTMLEmptyTitle verifies empty-title handling.
func TestRenderHTMLEmptyTitle(t *testing.T) {
	meta := config.BookMeta{Title: "", Author: "Author"}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	// Rendering should still succeed.
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should still render when the title is empty")
	}
}

// TestRenderHTMLEscaping verifies HTML escaping.
func TestRenderHTMLEscaping(t *testing.T) {
	meta := config.BookMeta{
		Title:  "<script>alert('xss')</script>",
		Author: `"injected"`,
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if strings.Contains(html, "<script>") {
		t.Error("HTML tags should be escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("escaped script tag should be present")
	}
	if !strings.Contains(html, "&quot;injected&quot;") {
		t.Error("quotes should be escaped")
	}
}

// TestRenderHTMLContainsDate verifies date rendering.
func TestRenderHTMLContainsDate(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Date") {
		t.Error("cover should contain the date label")
	}
	if !strings.Contains(html, "-") {
		t.Error("cover should contain an ISO-like date format")
	}
}

// TestRenderHTMLStructure verifies the HTML structure.
func TestRenderHTMLStructure(t *testing.T) {
	meta := config.BookMeta{Title: "Test", Author: "Author", Version: "1.0"}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	requiredTags := []string{
		"<!DOCTYPE html>", "<html", "</html>",
		"<head>", "</head>",
		"<body>", "</body>",
		"<style>", "</style>",
		"cover-page", "cover-content", "cover-title",
	}

	for _, tag := range requiredTags {
		if !strings.Contains(html, tag) {
			t.Errorf("cover HTML should contain %q", tag)
		}
	}
}

// TestEscapeHTMLCover verifies HTML escaping in the cover package.
func TestEscapeHTMLCover(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal text", "normal text"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{`he said "hi"`, "he said &quot;hi&quot;"},
		{"a & b", "a &amp; b"},
	}

	for _, tt := range tests {
		got := utils.EscapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestEscapeURL verifies URL escaping.
func TestEscapeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.png", "normal.png"},
		{"it's.png", "it%27s.png"},
		{"file with).png", "file with%29.png"},
		{`path"quote.png`, `path%22quote.png`},
		// Dangerous URI schemes must be rejected.
		{"javascript:alert(1)", ""},
		{"JavaScript:alert(1)", ""},
		{"vbscript:MsgBox", ""},
		{"data:text/html,<h1>hi</h1>", ""},
		// Whitespace-padded dangerous schemes must also be rejected.
		{"  javascript:alert(1)", ""},
		{"\tdata:text/html,x", ""},
		// Legitimate schemes pass through.
		{"https://example.com/img.png", "https://example.com/img.png"},
	}

	for _, tt := range tests {
		got := escapeURL(tt.input)
		if got != tt.want {
			t.Errorf("escapeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCoverLabelsLocalization verifies cover label localization.
func TestCoverLabelsLocalization(t *testing.T) {
	tests := []struct {
		lang                          string
		wantAuthor, wantVer, wantDate string
	}{
		{"en-US", "Author", "Version", "Date"},
		{"en", "Author", "Version", "Date"},
		{"", "Author", "Version", "Date"},
		{"zh-CN", "作者", "版本", "日期"},
		{"zh", "作者", "版本", "日期"},
		{"ja-JP", "著者", "バージョン", "日付"},
		{"ko-KR", "저자", "버전", "날짜"},
	}
	for _, tt := range tests {
		a, v, d := coverLabels(tt.lang)
		if a != tt.wantAuthor || v != tt.wantVer || d != tt.wantDate {
			t.Errorf("coverLabels(%q) = (%q,%q,%q), want (%q,%q,%q)",
				tt.lang, a, v, d, tt.wantAuthor, tt.wantVer, tt.wantDate)
		}
	}
}

// TestRenderHTMLChineseLabels verifies Chinese cover labels.
func TestRenderHTMLChineseLabels(t *testing.T) {
	meta := config.BookMeta{
		Title:    "测试书名",
		Author:   "张三",
		Version:  "1.0",
		Language: "zh-CN",
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "作者") {
		t.Error("Chinese cover should use '作者' label")
	}
	if !strings.Contains(html, "版本") {
		t.Error("Chinese cover should use '版本' label")
	}
	if !strings.Contains(html, "日期") {
		t.Error("Chinese cover should use '日期' label")
	}
	if strings.Contains(html, ">Author<") {
		t.Error("Chinese cover should not use English 'Author' label")
	}
}

// TestRenderHTMLEnglishLabels verifies English cover labels (default).
func TestRenderHTMLEnglishLabels(t *testing.T) {
	meta := config.BookMeta{
		Title:    "Test Book",
		Author:   "John",
		Version:  "1.0",
		Language: "en-US",
	}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	if !strings.Contains(html, "Author") {
		t.Error("English cover should use 'Author' label")
	}
	if !strings.Contains(html, "Version") {
		t.Error("English cover should use 'Version' label")
	}
	if !strings.Contains(html, "Date") {
		t.Error("English cover should use 'Date' label")
	}
}

// TestRenderHTMLNoAuthor verifies rendering without an author.
func TestRenderHTMLNoAuthor(t *testing.T) {
	meta := config.BookMeta{Title: "Test"}
	gen := NewCoverGenerator(meta, nil)
	html := gen.RenderHTML()

	// Rendering should succeed without an author block.
	if !strings.Contains(html, "cover-page") {
		t.Error("cover should render successfully")
	}
}

// TestIsLightColor verifies light/dark classification across hex, named, and
// rgb()/rgba() color forms, plus the assume-dark contract for unknown input.
func TestIsLightColor(t *testing.T) {
	tests := []struct {
		color string
		want  bool
	}{
		// Hex: long form, uppercase, shorthand, and alpha variants.
		{"#ffffff", true},
		{"#FFFFFF", true},
		{"#fff", true},
		{"#ffff", true},     // #rgba shorthand, alpha ignored
		{"#ffffffff", true}, // #rrggbbaa, alpha ignored
		{"#fafafa", true},
		{"#000000", false},
		{"#102a43", false},
		{"#1a1a2e", false},
		{"#33261D", false},
		{"  #ffffff  ", true}, // surrounding whitespace
		{"#ff", false},        // too short
		// Named colors: light table entries (case-insensitive).
		{"white", true},
		{"WHITE", true},
		{"ivory", true},
		{"snow", true},
		{"beige", true},
		{"linen", true},
		{"seashell", true},
		{"floralwhite", true},
		{"ghostwhite", true},
		{"whitesmoke", true},
		{"lightyellow", true},
		{"lightgray", true},
		{"lightgrey", true},
		{"gainsboro", true},
		// Named colors: dark entries and unlisted names.
		{"black", false},
		{"navy", false},
		{"maroon", false},
		{"rebeccapurple", false}, // unlisted named color -> dark
		// rgb()/rgba() numeric forms.
		{"rgb(255, 255, 255)", true},
		{"rgb(255,250,240)", true},
		{"RGB(255, 255, 255)", true},
		{"rgba(255, 255, 255, 0.9)", true},
		{"rgb(100%, 100%, 100%)", true},
		{"rgb(255 250 240 / 0.5)", true},
		{"rgb(16, 42, 67)", false},
		{"rgba(0, 0, 0, 1)", false},
		// Unparseable input stays dark (light text is the safe default).
		{"", false},
		{"not-a-color", false},
		{"hsl(0, 0%, 100%)", false},
		{"rgb()", false},
		{"rgb(a, b, c)", false},
	}

	for _, tt := range tests {
		if got := isLightColor(tt.color); got != tt.want {
			t.Errorf("isLightColor(%q) = %v, want %v", tt.color, got, tt.want)
		}
	}

	// The exported wrapper must agree with the internal implementation.
	if !IsLightColor("white") || IsLightColor("navy") {
		t.Error("IsLightColor should classify 'white' as light and 'navy' as dark")
	}
}
