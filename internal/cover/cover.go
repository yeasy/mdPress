// Package cover generates and renders book cover pages.
// It builds a styled HTML cover from book metadata such as title, author, and version.
package cover

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/theme"
	"github.com/yeasy/mdpress/pkg/utils"
)

// luminanceThreshold is the perceived luminance cutoff for distinguishing
// light from dark colors (~73% brightness on a 0-255 scale, ITU-R BT.601).
const luminanceThreshold = 186

// defaultCoverBg is the deep navy used for the default cover background when
// the book does not configure book.cover.background or book.cover.image and
// the active theme has no cover palette of its own (technical, custom, or nil
// themes). It gives the out-of-the-box cover a premium, professionally-typeset
// look.
const defaultCoverBg = "#102a43"

// defaultCoverFontFamily is the cover font stack used when no theme is set
// (or the theme does not define a font family).
const defaultCoverFontFamily = `-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "WenQuanYi Micro Hei", sans-serif`

// cssColorPattern matches safe CSS color values (hex, rgb, rgba, hsl, hsla, named colors).
var cssColorPattern = regexp.MustCompile(`^(?i)(?:#[0-9a-f]{3,8}|(?:rgb|rgba|hsl|hsla)\([\d\s,%.]+\)|[a-z]{1,30})$`)

// cssFontFamilySafe rejects font-family values containing characters that
// could break out of a CSS property declaration (mirrors theme validation).
var cssFontFamilySafe = regexp.MustCompile(`^[^;{}<>\\]*$`)

// CoverGenerator builds the HTML cover page.
type CoverGenerator struct {
	meta config.BookMeta
	// thm is the active document theme; may be nil, in which case the
	// generator falls back to its built-in defaults.
	thm *theme.Theme
}

// NewCoverGenerator creates a new cover generator from book metadata and the
// active theme (nil theme falls back to built-in defaults).
func NewCoverGenerator(meta config.BookMeta, thm *theme.Theme) *CoverGenerator {
	return &CoverGenerator{
		meta: meta,
		thm:  thm,
	}
}

// coverDefaults returns the theme-derived cover styling: the default
// background color (used when neither cover.background nor cover.image is
// configured), the text ink for dark backgrounds, and the ink for light
// backgrounds. Each built-in theme gets a cover that harmonizes with its
// interior pages; custom and nil themes fall back to the navy default.
func (cg *CoverGenerator) coverDefaults() (defaultBg, darkInk, lightInk string) {
	name := ""
	if cg.thm != nil {
		name = cg.thm.Name
	}
	switch name {
	case "elegant":
		// Deep warm brown-burgundy with cream ink, matching elegant's warm
		// cream serif pages (#FFFBF0 / #3E2723).
		return "#33261D", "#F5EDDF", "#3E2723"
	case "minimal":
		// Light neutral background with near-black ink for the clean,
		// whitespace-heavy minimal look.
		return "#FAFAFA", "#F6F8FC", "#111111"
	default:
		// technical, custom, or nil theme: the deep navy publication cover.
		return defaultCoverBg, "#f6f8fc", "#14304a"
	}
}

// coverFontFamily returns the font stack for the cover page. It prefers the
// active theme's font family (so elegant covers are serif like their pages)
// and falls back to the built-in sans stack.
func (cg *CoverGenerator) coverFontFamily() string {
	if cg.thm != nil && cg.thm.FontFamily != "" && cssFontFamilySafe.MatchString(cg.thm.FontFamily) {
		return cg.thm.FontFamily
	}
	return defaultCoverFontFamily
}

// RenderHTML returns a self-contained HTML cover page.
func (cg *CoverGenerator) RenderHTML() string {
	var buf strings.Builder

	// Write the HTML document head.
	buf.WriteString(`<!DOCTYPE html>` + "\n")
	buf.WriteString(`<html lang="en">` + "\n")
	buf.WriteString(`<head>` + "\n")
	buf.WriteString(`  <meta charset="UTF-8">` + "\n")
	buf.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n")
	fmt.Fprintf(&buf, `  <title>%s</title>`+"\n", utils.EscapeHTML(cg.meta.Title))
	buf.WriteString(cg.renderStyles())
	buf.WriteString(`</head>` + "\n")
	buf.WriteString(`<body>` + "\n")
	buf.WriteString(cg.renderCoverContent())
	buf.WriteString(`</body>` + "\n")
	buf.WriteString(`</html>` + "\n")

	return buf.String()
}

// renderStyles generates the cover page CSS.
func (cg *CoverGenerator) renderStyles() string {
	var buf strings.Builder

	defaultBg, darkInk, lightInk := cg.coverDefaults()

	buf.WriteString(`  <style>` + "\n")

	// Reset styles and page layout.
	buf.WriteString(`    * {` + "\n")
	buf.WriteString(`      margin: 0;` + "\n")
	buf.WriteString(`      padding: 0;` + "\n")
	buf.WriteString(`      box-sizing: border-box;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Base html/body styling.
	buf.WriteString(`    html, body {` + "\n")
	buf.WriteString(`      width: 100%;` + "\n")
	buf.WriteString(`      height: 100%;` + "\n")
	fmt.Fprintf(&buf, "      font-family: %s;\n", cg.coverFontFamily())
	buf.WriteString(`      background-color: #ffffff;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Cover container styles.
	buf.WriteString(`    .cover-page {` + "\n")
	buf.WriteString(`      display: flex;` + "\n")
	buf.WriteString(`      align-items: center;` + "\n")
	buf.WriteString(`      justify-content: center;` + "\n")
	buf.WriteString(`      width: 100%;` + "\n")
	buf.WriteString(`      height: 100%;` + "\n")
	buf.WriteString(`      padding: 60px 40px;` + "\n")

	// Prefer a configured background color or image.
	customBg := strings.TrimSpace(cg.meta.Cover.Background)
	hasCustomBgColor := customBg != "" && cssColorPattern.MatchString(customBg)
	if hasCustomBgColor {
		fmt.Fprintf(&buf, `      background-color: %s;`+"\n", customBg)
		buf.WriteString(`      background-size: cover;` + "\n")
		buf.WriteString(`      background-position: center;` + "\n")
		buf.WriteString(`      background-attachment: fixed;` + "\n")
	} else if cg.meta.Cover.Image != "" {
		fmt.Fprintf(&buf, `      background-image: url('%s');`+"\n", escapeURL(cg.meta.Cover.Image))
		buf.WriteString(`      background-size: cover;` + "\n")
		buf.WriteString(`      background-position: center;` + "\n")
		buf.WriteString(`      background-attachment: fixed;` + "\n")
	} else {
		// Premium theme-matched default cover. Users can still override via
		// book.cover.background or book.cover.image.
		fmt.Fprintf(&buf, "      background-color: %s;\n", defaultBg)
	}

	buf.WriteString(`    }` + "\n\n")

	// Cover content layout.
	// Text color adapts: dark ink on light backgrounds, light ink on dark
	// backgrounds. Hex, named, and rgb()/rgba() colors are analyzed for
	// luminance; images and unparseable colors assume dark so light text
	// stays the safe default.
	var hasDarkBg bool
	switch {
	case hasCustomBgColor:
		hasDarkBg = !isLightColor(customBg)
	case cg.meta.Cover.Image != "":
		hasDarkBg = true
	default:
		hasDarkBg = !isLightColor(defaultBg)
	}
	textColor := darkInk
	if !hasDarkBg {
		textColor = lightInk
	}
	buf.WriteString(`    .cover-content {` + "\n")
	buf.WriteString(`      text-align: center;` + "\n")
	fmt.Fprintf(&buf, "      color: %s;\n", textColor)
	buf.WriteString(`      max-width: 800px;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Title styles — clean publication font sizing.
	buf.WriteString(`    .cover-title {` + "\n")
	buf.WriteString(`      font-size: 46px;` + "\n")
	buf.WriteString(`      font-weight: 700;` + "\n")
	buf.WriteString(`      margin-bottom: 18px;` + "\n")
	buf.WriteString(`      letter-spacing: 0.5px;` + "\n")
	buf.WriteString(`      line-height: 1.25;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Subtitle styles.
	buf.WriteString(`    .cover-subtitle {` + "\n")
	buf.WriteString(`      font-size: 21px;` + "\n")
	buf.WriteString(`      font-weight: 400;` + "\n")
	buf.WriteString(`      letter-spacing: 0.3px;` + "\n")
	buf.WriteString(`      margin-bottom: 8px;` + "\n")
	buf.WriteString(`      opacity: 0.85;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Divider.
	dividerColor := "#D4D4D4"
	if hasDarkBg {
		dividerColor = "rgba(255, 255, 255, 0.5)"
	}
	buf.WriteString(`    .cover-divider {` + "\n")
	buf.WriteString(`      width: 100px;` + "\n")
	buf.WriteString(`      height: 2px;` + "\n")
	fmt.Fprintf(&buf, "      background-color: %s;\n", dividerColor)
	buf.WriteString(`      margin: 30px auto;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Metadata container — fully opaque for readability.
	metaColor := "#555"
	if hasDarkBg {
		metaColor = "rgba(255,255,255,0.9)"
	}
	buf.WriteString(`    .cover-metadata {` + "\n")
	buf.WriteString(`      margin-top: 50px;` + "\n")
	buf.WriteString(`      font-size: 16px;` + "\n")
	fmt.Fprintf(&buf, "      color: %s;\n", metaColor)
	buf.WriteString(`    }` + "\n\n")

	// Metadata row.
	buf.WriteString(`    .cover-meta-item {` + "\n")
	buf.WriteString(`      margin: 10px 0;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Metadata label.
	buf.WriteString(`    .cover-meta-label {` + "\n")
	buf.WriteString(`      display: inline-block;` + "\n")
	buf.WriteString(`      font-weight: 600;` + "\n")
	buf.WriteString(`      margin-right: 10px;` + "\n")
	buf.WriteString(`      min-width: 80px;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Print-specific rules.
	buf.WriteString(`    @media print {` + "\n")
	buf.WriteString(`      html, body {` + "\n")
	buf.WriteString(`        width: 100%;` + "\n")
	buf.WriteString(`        height: 100%;` + "\n")
	buf.WriteString(`        margin: 0;` + "\n")
	buf.WriteString(`        padding: 0;` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`      .cover-page {` + "\n")
	buf.WriteString(`        page-break-after: always;` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`    }` + "\n\n")

	buf.WriteString(`  </style>` + "\n")

	return buf.String()
}

// coverLabels returns localized labels for the cover page metadata.
func coverLabels(lang string) (author, version, date string) {
	if strings.HasPrefix(lang, "zh") {
		return "作者", "版本", "日期"
	}
	if strings.HasPrefix(lang, "ja") {
		return "著者", "バージョン", "日付"
	}
	if strings.HasPrefix(lang, "ko") {
		return "저자", "버전", "날짜"
	}
	return "Author", "Version", "Date"
}

// renderCoverContent builds the cover page HTML structure.
func (cg *CoverGenerator) renderCoverContent() string {
	var buf strings.Builder

	labelAuthor, labelVersion, labelDate := coverLabels(cg.meta.Language)

	buf.WriteString(`  <div class="cover-page">` + "\n")
	buf.WriteString(`    <div class="cover-content">` + "\n")

	// Title — use <div> instead of <h1> so Chrome's GenerateDocumentOutline
	// does not create a PDF bookmark for the cover title. The chapter template
	// already produces an <h1> for the first chapter; when that chapter's title
	// matches the book title (common for README/preface), an <h1> here would
	// create a duplicate bookmark entry.
	if cg.meta.Title != "" {
		fmt.Fprintf(&buf, `      <div class="cover-title">%s</div>`+"\n", utils.EscapeHTML(cg.meta.Title))
	}

	// Subtitle — same rationale: keep out of the document outline.
	if cg.meta.Subtitle != "" {
		fmt.Fprintf(&buf, `      <div class="cover-subtitle">%s</div>`+"\n", utils.EscapeHTML(cg.meta.Subtitle))
	}

	// Divider
	buf.WriteString(`      <div class="cover-divider"></div>` + "\n")

	// Metadata
	buf.WriteString(`      <div class="cover-metadata">` + "\n")

	// Author
	if cg.meta.Author != "" {
		buf.WriteString(`        <div class="cover-meta-item">` + "\n")
		fmt.Fprintf(&buf, `          <span class="cover-meta-label">%s</span>`+"\n", labelAuthor)
		fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", utils.EscapeHTML(cg.meta.Author))
		buf.WriteString(`        </div>` + "\n")
	}

	// Version
	if cg.meta.Version != "" {
		buf.WriteString(`        <div class="cover-meta-item">` + "\n")
		fmt.Fprintf(&buf, `          <span class="cover-meta-label">%s</span>`+"\n", labelVersion)
		fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", utils.EscapeHTML(cg.meta.Version))
		buf.WriteString(`        </div>` + "\n")
	}

	// Date
	currentDate := time.Now().Format("2006-01-02")
	buf.WriteString(`        <div class="cover-meta-item">` + "\n")
	fmt.Fprintf(&buf, `          <span class="cover-meta-label">%s</span>`+"\n", labelDate)
	fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", currentDate)
	buf.WriteString(`        </div>` + "\n")

	buf.WriteString(`      </div>` + "\n")

	buf.WriteString(`    </div>` + "\n")
	buf.WriteString(`  </div>` + "\n")

	return buf.String()
}

// urlReplacer escapes URL-sensitive characters for CSS url() context.
// Pre-compiled to avoid allocation on every call.
var urlReplacer = strings.NewReplacer(
	`'`, "%27",
	`"`, "%22",
	`)`, "%29",
	`(`, "%28",
	`;`, "%3B",
	`#`, "%23",
	`?`, "%3F",
	"\n", "%0A",
	"\r", "%0D",
	`\`, "%5C",
)

// escapeURL escapes URL-sensitive characters for CSS url() context.
// It also rejects dangerous URI schemes (javascript:, vbscript:, data:).
func escapeURL(u string) string {
	lower := strings.ToLower(strings.TrimSpace(u))
	if strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "vbscript:") ||
		strings.HasPrefix(lower, "data:") {
		return ""
	}
	return urlReplacer.Replace(u)
}

// namedColorLight classifies common CSS named colors as perceptually light
// (true) or dark (false). Names absent from the map are assumed dark, which
// keeps light text as the safe default for unknown backgrounds.
var namedColorLight = map[string]bool{
	// Light backgrounds -> dark ink.
	"white": true, "ivory": true, "snow": true, "beige": true,
	"linen": true, "seashell": true, "floralwhite": true, "ghostwhite": true,
	"whitesmoke": true, "lightyellow": true, "lightgray": true,
	"lightgrey": true, "gainsboro": true, "aliceblue": true,
	"antiquewhite": true, "azure": true, "cornsilk": true, "honeydew": true,
	"lavenderblush": true, "lemonchiffon": true, "mintcream": true,
	"oldlace": true, "papayawhip": true, "wheat": true,
	// Dark anchors, documented for clarity (any unlisted name is also
	// treated as dark).
	"black": false, "navy": false, "maroon": false, "midnightblue": false,
	"darkblue": false, "darkslategray": false, "darkslategrey": false,
}

// IsLightColor reports whether the given CSS color is perceptually light.
// It understands hex colors (#rgb, #rgba, #rrggbb, #rrggbbaa), common named
// CSS colors, and numeric rgb()/rgba() forms. Unknown or unparseable formats
// are assumed dark so that light text remains the safer default. Alpha
// channels are ignored.
func IsLightColor(color string) bool {
	return isLightColor(color)
}

// isLightColor is the internal implementation behind IsLightColor.
func isLightColor(color string) bool {
	color = strings.TrimSpace(color)
	if strings.HasPrefix(color, "#") {
		return isLightHex(color[1:])
	}
	lower := strings.ToLower(color)
	if light, ok := namedColorLight[lower]; ok {
		return light
	}
	if r, g, b, ok := parseRGBFunc(lower); ok {
		return luminance(r, g, b) > luminanceThreshold
	}
	return false
}

// isLightHex reports whether a hex color body (without the leading '#') is
// perceptually light. Alpha channels are ignored.
func isLightHex(hex string) bool {
	// Expand shorthand (#rgb -> #rrggbb, #rgba -> #rrggbb).
	if len(hex) == 3 || len(hex) == 4 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	// Strip alpha channel from #rrggbbaa.
	if len(hex) == 8 {
		hex = hex[:6]
	}
	if len(hex) < 6 {
		return false
	}
	r := hexVal(hex[0])*16 + hexVal(hex[1])
	g := hexVal(hex[2])*16 + hexVal(hex[3])
	b := hexVal(hex[4])*16 + hexVal(hex[5])
	return luminance(float64(r), float64(g), float64(b)) > luminanceThreshold
}

// parseRGBFunc parses numeric rgb()/rgba() color functions. It accepts both
// the legacy comma syntax (rgb(255, 250, 240)) and the modern space syntax
// (rgb(255 250 240 / 0.5)); percentage components are scaled to the 0-255
// range. The alpha channel is ignored. The input must already be lowercase.
func parseRGBFunc(color string) (r, g, b float64, ok bool) {
	var body string
	switch {
	case strings.HasPrefix(color, "rgba(") && strings.HasSuffix(color, ")"):
		body = color[len("rgba(") : len(color)-1]
	case strings.HasPrefix(color, "rgb(") && strings.HasSuffix(color, ")"):
		body = color[len("rgb(") : len(color)-1]
	default:
		return 0, 0, 0, false
	}
	body = strings.NewReplacer(",", " ", "/", " ").Replace(body)
	fields := strings.Fields(body)
	if len(fields) < 3 {
		return 0, 0, 0, false
	}
	var channels [3]float64
	for i := 0; i < 3; i++ {
		f := fields[i]
		percent := strings.HasSuffix(f, "%")
		f = strings.TrimSuffix(f, "%")
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return 0, 0, 0, false
		}
		if percent {
			v = v * 255 / 100
		}
		channels[i] = v
	}
	return channels[0], channels[1], channels[2], true
}

// luminance computes perceived luminance (ITU-R BT.601) on a 0-255 scale:
// Y = 0.299R + 0.587G + 0.114B.
func luminance(r, g, b float64) float64 {
	return 0.299*r + 0.587*g + 0.114*b
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return 0
	}
}
