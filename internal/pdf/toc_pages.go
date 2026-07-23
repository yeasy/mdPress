package pdf

import (
	"html"
	"regexp"
	"strconv"

	"github.com/yeasy/mdpress/internal/toc"
)

// tocPageSlotPattern matches an empty page-number slot emitted by the TOC
// generator and captures the anchor id it belongs to.
var tocPageSlotPattern = regexp.MustCompile(
	`(<span[^>]*\s` + regexp.QuoteMeta(toc.PageSlotAttr) + `="([^"]*)"[^>]*>)</span>`)

// hasTOCPageSlots reports whether the document carries page-number slots that a
// second print pass could fill in.
func hasTOCPageSlots(htmlContent string) bool {
	return tocPageSlotPattern.MatchString(htmlContent)
}

// fillTOCPageNumbers writes the resolved page numbers into the table of
// contents and returns the updated document along with how many slots were
// filled. Slots whose anchor did not make it into the PDF are left empty so the
// entry simply shows no number instead of a wrong one.
func fillTOCPageNumbers(htmlContent string, pages map[string]int) (string, int) {
	filled := 0
	result := tocPageSlotPattern.ReplaceAllStringFunc(htmlContent, func(match string) string {
		groups := tocPageSlotPattern.FindStringSubmatch(match)
		if len(groups) != 3 {
			return match
		}
		page, ok := pages[html.UnescapeString(groups[2])]
		if !ok || page <= 0 {
			return match
		}
		filled++
		return groups[1] + strconv.Itoa(page) + "</span>"
	})
	return result, filled
}
