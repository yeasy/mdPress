package pdf

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// This file reads back the page layout Chrome just produced, so that a printed
// table of contents can carry the page numbers a reader needs. Chrome records
// every HTML anchor it printed as a PDF named destination pointing at a page
// object; walking the page tree turns those page objects into the page numbers
// printed in the folio.
//
// It reuses the small PDF object reader in metadata.go and, like it, only
// understands the classic cross-reference tables Chrome writes. Anything it
// cannot read is reported as an error so the caller can fall back to a table of
// contents without page numbers rather than fail the build.

// maxPageTreeDepth bounds the page-tree walk. Real documents nest a handful of
// levels; a deeper tree means the file is malformed or hostile.
const maxPageTreeDepth = 64

// namedDestinationPages maps each named destination in a printed PDF (the
// fragment ids of the anchors in the source HTML) to its 1-based page number.
func namedDestinationPages(data []byte) (map[string]int, error) {
	offsets, trailer, _, ok := resolveXref(data)
	if !ok {
		return nil, errors.New("pdf: cross-reference table is not readable")
	}
	rootNum, rootGen, ok := dictObjectRef(trailer, "Root")
	if !ok {
		return nil, errors.New("pdf: trailer has no /Root reference")
	}
	catalog, ok := indirectDictBody(data, offsets, rootNum, rootGen)
	if !ok {
		return nil, errors.New("pdf: document catalog is not readable")
	}

	pageOfObject, err := pageNumbersByObject(data, offsets, catalog)
	if err != nil {
		return nil, err
	}

	destsBody, ok := resolveDictValue(data, offsets, catalog, "Dests")
	if !ok {
		return nil, errors.New("pdf: document catalog has no readable /Dests dictionary")
	}
	entries, ok := parsePDFDict(destsBody)
	if !ok {
		return nil, errors.New("pdf: /Dests is not a dictionary")
	}

	pages := make(map[string]int, len(entries))
	for _, entry := range entries {
		target, ok := destinationPageObject(entry.value)
		if !ok {
			continue
		}
		page, ok := pageOfObject[target]
		if !ok {
			continue
		}
		name := decodePDFName(entry.key)
		pages[name] = page
		// Chrome percent-encodes a non-ASCII fragment before writing it as a
		// PDF name, so a Chinese heading arrives here as "%E6%A6%82%E8%BF%B0"
		// while the HTML anchor it came from reads "概述". Record both spellings
		// so the lookup succeeds either way.
		if decoded, err := url.PathUnescape(name); err == nil && decoded != name {
			pages[decoded] = page
		}
	}
	if len(pages) == 0 {
		return nil, errors.New("pdf: no named destination resolved to a page")
	}
	return pages, nil
}

// pageNumbersByObject walks the catalog's page tree and returns the 1-based
// page number of every page object, keyed by object number.
func pageNumbersByObject(data []byte, offsets map[int]int, catalog []byte) (map[int]int, error) {
	ordered, err := orderedPageObjects(data, offsets, catalog)
	if err != nil {
		return nil, err
	}
	numbers := make(map[int]int, len(ordered))
	for i, num := range ordered {
		numbers[num] = i + 1
	}
	return numbers, nil
}

// orderedPageObjects returns the object numbers of the document's pages in
// reading order.
func orderedPageObjects(data []byte, offsets map[int]int, catalog []byte) ([]int, error) {
	pagesNum, pagesGen, ok := dictObjectRef(catalog, "Pages")
	if !ok {
		return nil, errors.New("pdf: document catalog has no /Pages reference")
	}
	var ordered []int
	visited := make(map[int]bool)
	if !collectPageObjects(data, offsets, pagesNum, pagesGen, 0, visited, &ordered) {
		return nil, errors.New("pdf: page tree is not readable")
	}
	if len(ordered) == 0 {
		return nil, errors.New("pdf: page tree contains no pages")
	}
	return ordered, nil
}

// collectPageObjects appends the object numbers of the leaf pages under the
// node at num, in reading order.
func collectPageObjects(data []byte, offsets map[int]int, num, gen, depth int, visited map[int]bool, out *[]int) bool {
	if depth > maxPageTreeDepth || visited[num] {
		return false
	}
	visited[num] = true

	body, ok := indirectDictBody(data, offsets, num, gen)
	if !ok {
		return false
	}
	nodeType, _ := dictRawValue(body, "Type")
	if string(nodeType) == "/Page" {
		*out = append(*out, num)
		return true
	}
	kids, ok := dictRawValue(body, "Kids")
	if !ok {
		return false
	}
	for _, kid := range objectRefsInArray(kids) {
		if !collectPageObjects(data, offsets, kid.num, kid.gen, depth+1, visited, out) {
			return false
		}
	}
	return true
}

// destinationPageObject returns the object number of the page a destination
// value points at. Destinations are written either as the explicit array
// "[page /XYZ …]" or as a dictionary whose /D holds that array.
func destinationPageObject(value []byte) (int, bool) {
	trimmed := strings.TrimSpace(string(value))
	if strings.HasPrefix(trimmed, "<<") && strings.HasSuffix(trimmed, ">>") {
		inner, ok := dictRawValue([]byte(trimmed[2:len(trimmed)-2]), "D")
		if !ok {
			return 0, false
		}
		trimmed = strings.TrimSpace(string(inner))
	}
	refs := objectRefsInArray([]byte(trimmed))
	if len(refs) == 0 {
		return 0, false
	}
	return refs[0].num, true
}

// objectRef is an indirect reference "num gen R".
type objectRef struct {
	num int
	gen int
}

// objectRefsInArray returns the indirect references found in a PDF array,
// ignoring any other array element.
func objectRefsInArray(value []byte) []objectRef {
	fields := strings.Fields(strings.Trim(strings.TrimSpace(string(value)), "[]"))
	var refs []objectRef
	for i := 0; i+2 < len(fields); i++ {
		if fields[i+2] != "R" {
			continue
		}
		num, err := strconv.Atoi(fields[i])
		if err != nil {
			continue
		}
		gen, err := strconv.Atoi(fields[i+1])
		if err != nil {
			continue
		}
		refs = append(refs, objectRef{num: num, gen: gen})
		i += 2
	}
	return refs
}

// indirectDictBody returns the dictionary body of object num.
func indirectDictBody(data []byte, offsets map[int]int, num, gen int) ([]byte, bool) {
	off, ok := offsets[num]
	if !ok || off <= 0 || off >= len(data) {
		return nil, false
	}
	_, _, body, ok := readIndirectDict(data, off, num, gen)
	return body, ok
}

// resolveDictValue returns the dictionary body stored under key, following an
// indirect reference when the value is one.
func resolveDictValue(data []byte, offsets map[int]int, body []byte, key string) ([]byte, bool) {
	raw, ok := dictRawValue(body, key)
	if !ok {
		return nil, false
	}
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "<<") && strings.HasSuffix(trimmed, ">>") {
		return []byte(trimmed[2 : len(trimmed)-2]), true
	}
	num, gen, ok := dictObjectRef(body, key)
	if !ok {
		return nil, false
	}
	return indirectDictBody(data, offsets, num, gen)
}

// decodePDFName undoes the "#xx" escaping PDF names use for bytes that cannot
// appear literally. Chrome writes non-ASCII anchor ids that way, so a Chinese
// heading's destination only matches its HTML id after decoding.
func decodePDFName(name string) string {
	if !strings.Contains(name, "#") {
		return name
	}
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		if name[i] == '#' && i+2 < len(name) {
			if v, err := strconv.ParseUint(name[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(v))
				i += 2
				continue
			}
		}
		b.WriteByte(name[i])
	}
	return b.String()
}
