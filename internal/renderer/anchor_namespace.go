// anchor_namespace.go makes heading anchors unique across a whole book.
//
// Heading ids are minted per document, so they are only unique within one
// chapter. That is fine for the multi-page site, where each chapter is its own
// page, but the standalone HTML concatenates every chapter into one document:
// the ids collide, the browser resolves each "#examples" link to whichever
// chapter happens to come first, and the resulting page is invalid HTML.
//
// mdPress's own manual produced 195 duplicated ids, "notes" appearing twelve
// times — so eleven of every twelve sidebar links under that name went to the
// wrong chapter.
package renderer

import (
	"fmt"
	"regexp"
)

// headingIDPattern matches the id attribute of a heading tag.
var headingIDPattern = regexp.MustCompile(`(?i)(<h[1-6]\b[^>]*\sid=")([^"]+)(")`)

// inPageHrefPattern matches a same-document link.
var inPageHrefPattern = regexp.MustCompile(`(?i)(href=")#([^"]+)(")`)

// anchorNamespacer rewrites heading ids and same-document links so that every
// anchor in the assembled document is unique.
type anchorNamespacer struct {
	// used tracks every id handed out across the whole book.
	used map[string]bool
}

func newAnchorNamespacer() *anchorNamespacer {
	return &anchorNamespacer{used: map[string]bool{}}
}

// Reserve claims an id verbatim (used for chapter ids, which the pipeline has
// already de-duplicated across the book).
func (a *anchorNamespacer) Reserve(id string) {
	if id != "" {
		a.used[id] = true
	}
}

// Rewrite returns chapter HTML whose heading ids are unique document-wide,
// along with the mapping from the chapter's original ids to the new ones so
// the sidebar and page TOC can be rewritten to match.
//
// Ids that are already unique keep their original spelling, so the common case
// produces the same anchors as before and existing external links still work.
func (a *anchorNamespacer) Rewrite(chapterID, html string) (string, map[string]string) {
	mapping := map[string]string{}

	rewritten := headingIDPattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := headingIDPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		original := parts[2]
		unique, ok := mapping[original]
		if !ok {
			unique = a.claim(chapterID, original)
			mapping[original] = unique
		}
		return parts[1] + unique + parts[3]
	})

	// Links inside the chapter must follow its own headings.
	rewritten = inPageHrefPattern.ReplaceAllStringFunc(rewritten, func(match string) string {
		parts := inPageHrefPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		if target, ok := mapping[parts[2]]; ok {
			return parts[1] + "#" + target + parts[3]
		}
		return match
	})

	return rewritten, mapping
}

// claim returns an unused id for original, preferring the original spelling and
// falling back to a chapter-qualified form.
func (a *anchorNamespacer) claim(chapterID, original string) string {
	if !a.used[original] {
		a.used[original] = true
		return original
	}
	if chapterID != "" {
		qualified := chapterID + "--" + original
		if !a.used[qualified] {
			a.used[qualified] = true
			return qualified
		}
		for n := 2; ; n++ {
			candidate := fmt.Sprintf("%s-%d", qualified, n)
			if !a.used[candidate] {
				a.used[candidate] = true
				return candidate
			}
		}
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", original, n)
		if !a.used[candidate] {
			a.used[candidate] = true
			return candidate
		}
	}
}

// remapHeadings rewrites a navigation tree's ids through mapping, so sidebar
// and TOC entries point at the ids the content actually carries.
func remapHeadings(headings []NavHeading, mapping map[string]string) []NavHeading {
	if len(headings) == 0 {
		return headings
	}
	out := make([]NavHeading, len(headings))
	for i, h := range headings {
		out[i] = h
		if mapped, ok := mapping[h.ID]; ok {
			out[i].ID = mapped
		}
		out[i].Children = remapHeadings(h.Children, mapping)
	}
	return out
}
