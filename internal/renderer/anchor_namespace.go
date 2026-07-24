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
	"strconv"

	"github.com/yeasy/mdpress/internal/toc"
)

// headingIDPattern matches the id attribute of a heading tag.
var headingIDPattern = regexp.MustCompile(`(?i)(<h([1-6])\b[^>]*\sid=")([^"]+)(")`)

// inPageHrefPattern matches a same-document link.
var inPageHrefPattern = regexp.MustCompile(`(?i)(href=")#([^"]+)(")`)

// documentAnchor is one destination the assembled document offers: either a
// chapter's own <div id> or a heading inside it. They are recorded in document
// order because a table of contents that was generated from the pre-namespacing
// ids can only be remapped by walking the two lists in step.
type documentAnchor struct {
	original string // the id the TOC generator was handed
	unique   string // the id the assembled document actually carries
	level    int    // heading level, so the TOC's depth filter can be replayed
}

// anchorNamespacer rewrites heading ids and same-document links so that every
// anchor in the assembled document is unique.
type anchorNamespacer struct {
	// used tracks every id handed out across the whole book.
	used map[string]bool
	// anchors records every destination in document order, for RemapTOC.
	anchors []documentAnchor
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

// MarkChapter records the chapter's own anchor — the id on its <div>, which is
// what a table of contents links a chapter title to — at the position the
// chapter occupies in the document. Call it immediately before Rewrite for the
// same chapter.
func (a *anchorNamespacer) MarkChapter(id string) {
	if id == "" {
		return
	}
	a.anchors = append(a.anchors, documentAnchor{original: id, unique: id, level: 1})
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
		if len(parts) != 5 {
			return match
		}
		original := parts[3]
		unique, ok := mapping[original]
		if !ok {
			unique = a.claim(chapterID, original)
			mapping[original] = unique
			level, err := strconv.Atoi(parts[2])
			if err != nil {
				level = 1
			}
			a.anchors = append(a.anchors, documentAnchor{original: original, unique: unique, level: level})
		}
		return parts[1] + unique + parts[4]
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

// tocEntryPattern matches one rendered table-of-contents entry: the link and
// everything up to its closing tag, which is where the page-number slot sits.
// Entries never nest inside each other — children live in a <ul> that follows
// the </a> — so the non-greedy body cannot swallow a sibling.
var tocEntryPattern = regexp.MustCompile(`(?s)(<a href="#)([^"]*)(">.*?</a>)`)

// tocPageSlotIDPattern matches the anchor id the page-number slot is keyed by.
// It repeats the link target, so it has to be remapped with it or the printed
// page number would still be looked up under the colliding id.
var tocPageSlotIDPattern = regexp.MustCompile(`(\s` + regexp.QuoteMeta(toc.PageSlotAttr) + `=")([^"]*)(")`)

// RemapTOC rewrites a table of contents that was generated from the per-chapter
// heading ids so it points at the ids the assembled document now carries.
//
// Without it the printed TOC lies: two chapters that both contain "## Overview"
// produce one "#overview" destination, so the second entry prints the first
// chapter's page number and links there too — while the PDF bookmarks, which
// Chrome derives from the headings themselves, show the right page. The
// document contradicts itself and nothing in the build output says so.
//
// The TOC is a depth-filtered subsequence of the document's anchors in order,
// so entries are matched by walking both lists with a forward-only cursor.
// maxDepth replays the filter cmd applied when generating the TOC; pass 0 when
// no filter was applied. An entry that cannot be matched is left untouched.
func (a *anchorNamespacer) RemapTOC(tocHTML string, maxDepth int) string {
	if tocHTML == "" || len(a.anchors) == 0 {
		return tocHTML
	}

	candidates := a.anchors
	if maxDepth > 0 && maxDepth < 6 {
		filtered := make([]documentAnchor, 0, len(candidates))
		for _, anchor := range candidates {
			if anchor.level <= maxDepth {
				filtered = append(filtered, anchor)
			}
		}
		candidates = filtered
	}

	cursor := 0
	return tocEntryPattern.ReplaceAllStringFunc(tocHTML, func(entry string) string {
		parts := tocEntryPattern.FindStringSubmatch(entry)
		if len(parts) != 4 {
			return entry
		}
		original := parts[2]
		unique := original
		for i := cursor; i < len(candidates); i++ {
			if candidates[i].original == original {
				unique = candidates[i].unique
				cursor = i + 1
				break
			}
		}
		if unique == original {
			return entry
		}
		body := tocPageSlotIDPattern.ReplaceAllStringFunc(parts[3], func(slot string) string {
			slotParts := tocPageSlotIDPattern.FindStringSubmatch(slot)
			if len(slotParts) != 4 {
				return slot
			}
			return slotParts[1] + unique + slotParts[3]
		})
		return parts[1] + unique + body
	})
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
