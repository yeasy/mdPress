// epub_cover.go draws the default EPUB cover image.
//
// A book without a cover-image item shows up as a blank tile in Apple Books,
// Kobo and every other library UI. Rendering a raster cover would drag in an
// image toolchain; SVG is an EPUB 3 core media type, so the same title,
// subtitle and author the title page uses can be drawn directly.
package output

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/yeasy/mdpress/pkg/utils"
)

// Cover geometry, in SVG user units. The 1:1.6 ratio is the common ebook
// thumbnail shape, so readers letterbox it as little as possible.
const (
	coverWidth   = 1600
	coverHeight  = 2560
	coverPadding = 170
)

// synthesizedCoverAsset renders the default cover for books that configure no
// cover image of their own.
func (g *EpubGenerator) synthesizedCoverAsset() *epubAsset {
	return &epubAsset{
		ID:          "cover-image",
		Filename:    "assets/cover.svg",
		MediaType:   "image/svg+xml",
		Data:        []byte(g.renderCoverSVG()),
		Synthesized: true,
	}
}

func (g *EpubGenerator) renderCoverSVG() string {
	bg := epubCoverBackground(g.meta.CoverBackground)
	// Same adaptive ink choice as the generated title page.
	title, secondary, rule := "#f6f8fc", "rgba(255,255,255,0.82)", "rgba(255,255,255,0.35)"
	if epubIsLightColor(bg) {
		title, secondary, rule = "#14304a", "#475569", "rgba(20,48,74,0.30)"
	}

	titleLines := wrapCoverText(defaultString(g.meta.Title, "Untitled"), 15, 4)
	subtitleLines := wrapCoverText(g.meta.Subtitle, 30, 2)

	var b strings.Builder
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" preserveAspectRatio="xMidYMid meet" role="img">
`, coverWidth, coverHeight, coverWidth, coverHeight)
	fmt.Fprintf(&b, "  <title>%s</title>\n", utils.EscapeXML(g.meta.Title))
	fmt.Fprintf(&b, "  <rect width=\"%d\" height=\"%d\" fill=\"%s\"/>\n", coverWidth, coverHeight, utils.EscapeXML(bg))
	fmt.Fprintf(&b, "  <rect x=\"%d\" y=\"%d\" width=\"180\" height=\"12\" fill=\"%s\"/>\n",
		coverPadding, coverPadding+40, utils.EscapeXML(secondary))

	// The title block is anchored a little above the optical center so a long
	// title grows downward into empty space instead of colliding with the byline.
	y := 900
	const titleSize, titleLead = 132, 168
	for _, line := range titleLines {
		fmt.Fprintf(&b, "  <text x=\"%d\" y=\"%d\" fill=\"%s\" font-family=\"%s\" font-size=\"%d\" font-weight=\"700\">%s</text>\n",
			coverPadding, y, utils.EscapeXML(title), coverSansStack, titleSize, utils.EscapeXML(line))
		y += titleLead
	}

	if len(subtitleLines) > 0 {
		y += 100
		fmt.Fprintf(&b, "  <rect x=\"%d\" y=\"%d\" width=\"420\" height=\"4\" fill=\"%s\"/>\n",
			coverPadding, y-110, utils.EscapeXML(rule))
		for _, line := range subtitleLines {
			fmt.Fprintf(&b, "  <text x=\"%d\" y=\"%d\" fill=\"%s\" font-family=\"%s\" font-size=\"64\">%s</text>\n",
				coverPadding, y, utils.EscapeXML(secondary), coverSansStack, utils.EscapeXML(line))
			y += 86
		}
	}

	footer := coverHeight - coverPadding - 90
	if author := strings.TrimSpace(g.meta.Author); author != "" {
		fmt.Fprintf(&b, "  <text x=\"%d\" y=\"%d\" fill=\"%s\" font-family=\"%s\" font-size=\"62\">%s</text>\n",
			coverPadding, footer, utils.EscapeXML(title), coverSansStack, utils.EscapeXML(truncateCoverText(author, 34)))
	}
	if version := strings.TrimSpace(g.meta.Version); version != "" {
		fmt.Fprintf(&b, "  <text x=\"%d\" y=\"%d\" fill=\"%s\" font-family=\"%s\" font-size=\"46\">Version %s</text>\n",
			coverPadding, footer+80, utils.EscapeXML(secondary), coverSansStack, utils.EscapeXML(truncateCoverText(version, 24)))
	}

	b.WriteString("</svg>\n")
	return b.String()
}

// coverSansStack is quoted for an SVG attribute, so it uses no double quotes.
const coverSansStack = "'Noto Sans', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Segoe UI', Helvetica, Arial, sans-serif"

func defaultString(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// wrapCoverText breaks text into at most maxLines lines of roughly maxCols
// columns. SVG has no text wrapping, so the line breaks have to be decided
// here; CJK characters are counted as two columns because they render about
// twice as wide as Latin ones.
func wrapCoverText(text string, maxCols, maxLines int) []string {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" || maxLines <= 0 {
		return nil
	}

	var lines []string
	var line []rune
	width := 0
	lastSpace := -1

	flush := func(upTo int) {
		out := strings.TrimSpace(string(line[:upTo]))
		if out != "" {
			lines = append(lines, out)
		}
		rest := line[upTo:]
		// The space a line was broken at belongs to neither line; leaving it in
		// would count against the next line's width and force an early break.
		for len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
		line = append([]rune{}, rest...)
		width = 0
		for _, r := range line {
			width += runeCols(r)
		}
		lastSpace = -1
	}

	for _, r := range text {
		if r == ' ' {
			lastSpace = len(line)
		}
		line = append(line, r)
		width += runeCols(r)
		if width <= maxCols {
			continue
		}
		if lastSpace > 0 {
			flush(lastSpace)
		} else {
			// A single unbreakable run (a long word, or CJK, which has no
			// spaces at all): break it where it overflows.
			flush(len(line) - 1)
		}
		if len(lines) == maxLines {
			// Text remains but there is no room for it.
			lines[maxLines-1] = truncateCoverText(lines[maxLines-1], maxCols-1) + "…"
			return lines
		}
	}
	if rest := strings.TrimSpace(string(line)); rest != "" {
		lines = append(lines, rest)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = truncateCoverText(lines[maxLines-1], maxCols-1) + "…"
	}
	return lines
}

func truncateCoverText(text string, maxCols int) string {
	width := 0
	for i, r := range text {
		width += runeCols(r)
		if width > maxCols {
			return strings.TrimSpace(text[:i])
		}
	}
	return text
}

func runeCols(r rune) int {
	if r > 0x2E7F && !unicode.IsSpace(r) {
		return 2
	}
	return 1
}
