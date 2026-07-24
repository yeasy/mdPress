package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// This file takes the running head and folio back off a book's cover.
//
// A cover is artwork printed to the edge of the sheet, but Chrome draws the
// print header and footer from the PrintToPDF parameters rather than from the
// page box, so they land on page one like they land on every other page: a book
// whose style.header is "{title}" / "{author}" and whose footer is
// "Page {page} of {pages}" printed gray running text and "Page 1 of 17" on top
// of its navy cover. Nothing in the document can prevent that. `@page :first {
// margin: 0 }` only moves the page box, and Chrome's header/footer template is
// rendered as an isolated one-page document per sheet into which it injects
// only pageNumber, totalPages, title, date and url as plain text — there is no
// class, no counter and no selector that can tell the template it is being
// drawn on page one.
//
// So the header and footer are removed afterwards, from the printed file.
// Chrome appends them to each page's content stream as the final balanced
// "q … Q" group, byte-for-byte the same on every page apart from the folio's
// glyphs, and — unlike document content — outside any marked-content sequence,
// so dropping the group takes nothing out of the structure tree. Recognizing it
// is therefore a matter of finding the same trailing group on page one and page
// two; a document where the two do not agree is left exactly as Chrome printed
// it, because a wrongly identified group would delete real cover artwork, which
// is far worse than the blemish this removes.

// StripCoverHeaderFooter removes the print header and footer Chrome drew on
// page one of data, which is only correct when page one is a cover.
//
// The document is returned unchanged, with an error explaining why, whenever
// the running head cannot be identified beyond doubt.
func StripCoverHeaderFooter(data []byte) ([]byte, error) {
	offsets, trailer, startxref, ok := resolveXref(data)
	if !ok {
		return data, errors.New("pdf cover: cross-reference table is not readable")
	}
	if _, encrypted := dictRawValue(trailer, "Encrypt"); encrypted {
		return data, errors.New("pdf cover: encrypted documents are not modified")
	}
	rootNum, rootGen, ok := dictObjectRef(trailer, "Root")
	if !ok {
		return data, errors.New("pdf cover: trailer has no /Root reference")
	}
	catalog, ok := indirectDictBody(data, offsets, rootNum, rootGen)
	if !ok {
		return data, errors.New("pdf cover: document catalog is not readable")
	}
	pageObjects, err := orderedPageObjects(data, offsets, catalog)
	if err != nil {
		return data, err
	}
	// The running head is recognized by appearing identically on two pages, so
	// a one-page document offers nothing to compare against. A one-page book is
	// a cover with nothing behind it anyway.
	if len(pageObjects) < 2 {
		return data, errors.New("pdf cover: a single-page document has no second page to compare against")
	}

	cover, err := readPage(data, offsets, pageObjects[0])
	if err != nil {
		return data, err
	}
	next, err := readPage(data, offsets, pageObjects[1])
	if err != nil {
		return data, err
	}
	// Two pages drawn from the same bytes make the comparison below vacuous —
	// every group matches itself — so the cover's own artwork would qualify as
	// the running head. Chrome gives each page its own stream, so this only
	// happens in a document mdPress did not print.
	if cover.contentsNum == next.contentsNum || bytes.Equal(cover.content, next.content) {
		return data, errors.New("pdf cover: the first two pages are drawn from identical content")
	}

	trimmed, err := stripRunningHead(cover.content, next.content)
	if err != nil {
		return data, err
	}
	return replacePageContent(data, trailer, startxref, cover, trimmed)
}

// printedPage is the part of a printed page this file rewrites: the page
// object itself and the single content stream Chrome gave it.
type printedPage struct {
	objNum      int
	dict        []byte // the bytes between "<<" and ">>" of the page object
	contentsNum int
	content     []byte // the decoded content stream
}

// readPage loads a page object and inflates its content stream.
func readPage(data []byte, offsets map[int]int, objNum int) (printedPage, error) {
	dict, ok := indirectDictBody(data, offsets, objNum, 0)
	if !ok {
		return printedPage{}, fmt.Errorf("pdf cover: page object %d is not readable", objNum)
	}
	// A page may hold an array of content streams. Chrome writes exactly one,
	// and splitting a running head across several is not a shape this code has
	// been able to check, so anything else is refused.
	contentsNum, contentsGen, ok := dictObjectRef(dict, "Contents")
	if !ok {
		return printedPage{}, fmt.Errorf("pdf cover: page object %d has no single /Contents stream", objNum)
	}
	streamDict, raw, ok := indirectStream(data, offsets, contentsNum, contentsGen)
	if !ok {
		return printedPage{}, fmt.Errorf("pdf cover: content stream %d is not readable", contentsNum)
	}
	filter, _ := dictRawValue(streamDict, "Filter")
	content, err := decodeContentStream(raw, filter)
	if err != nil {
		return printedPage{}, fmt.Errorf("pdf cover: content stream %d: %w", contentsNum, err)
	}
	return printedPage{objNum: objNum, dict: dict, contentsNum: contentsNum, content: content}, nil
}

// stripRunningHead removes the trailing header/footer group from cover, using
// next — the page printed after it — as the witness that identifies the group.
func stripRunningHead(cover, next []byte) ([]byte, error) {
	coverTokens, ok := scanContentStream(cover)
	if !ok {
		return nil, errors.New("pdf cover: the cover's content stream is not readable")
	}
	nextTokens, ok := scanContentStream(next)
	if !ok {
		return nil, errors.New("pdf cover: the second page's content stream is not readable")
	}

	coverGroup, ok := trailingGroup(coverTokens)
	if !ok {
		return nil, errors.New("pdf cover: the cover does not end with a self-contained drawing group")
	}
	nextGroup, ok := trailingGroup(nextTokens)
	if !ok {
		return nil, errors.New("pdf cover: the second page does not end with a self-contained drawing group")
	}

	head := coverTokens[coverGroup.first : coverGroup.last+1]
	witness := nextTokens[nextGroup.first : nextGroup.last+1]
	if !sameRunningHead(head, witness) {
		return nil, errors.New("pdf cover: the cover's last drawing group is not the page's running head")
	}
	if err := checkRemovable(head); err != nil {
		return nil, err
	}

	// Chrome closes marked-content sequences opened by the document content
	// from inside the running-head group. Dropping those EMC operators along
	// with the group would leave the sequence open, and a viewer reading the
	// tagged structure would then attribute the rest of the file to a content
	// element that ends on the cover.
	var out bytes.Buffer
	out.Write(bytes.TrimRight(cover[:coverTokens[coverGroup.first].start], " \t\r\n"))
	out.WriteByte('\n')
	for _, tok := range head {
		if tok.operator && tok.text == "EMC" {
			out.WriteString("EMC\n")
		}
	}
	return out.Bytes(), nil
}

// trailingGroup returns the balanced "q … Q" group that closes a content
// stream. It requires at least one group before it, so that a page whose whole
// drawing sits in one group can never be blanked.
func trailingGroup(tokens []csToken) (contentGroup, bool) {
	groups := topLevelGroups(tokens)
	if len(groups) < 2 {
		return contentGroup{}, false
	}
	last := groups[len(groups)-1]
	if last.last != len(tokens)-1 {
		return contentGroup{}, false
	}
	return last, true
}

// sameRunningHead reports whether two trailing groups are the same header and
// footer printed on two different sheets.
//
// The folio makes the two groups differ, so numbers and glyph strings are
// allowed to: page one reads "1" where page two reads "2", and the text that
// follows shifts by the width of the digit. Everything else — the operators, in
// order, and the resource names they refer to — has to match, which is a bar
// that page one's cover artwork and page two's contents cannot clear by
// accident. EMC is ignored on both sides because whether one lands inside the
// group depends on what the page's own content left open.
func sameRunningHead(a, b []csToken) bool {
	left := withoutEMC(a)
	right := withoutEMC(b)
	if len(left) != len(right) || len(left) == 0 {
		return false
	}
	for i := range left {
		if left[i].operator != right[i].operator {
			return false
		}
		if left[i].operator {
			if left[i].text != right[i].text {
				return false
			}
			continue
		}
		// Operands: only numbers and strings — the folio and the positions it
		// shifts — may differ.
		if left[i].text == right[i].text {
			continue
		}
		if !isVariableOperand(left[i].text) || !isVariableOperand(right[i].text) {
			return false
		}
	}
	return true
}

func withoutEMC(tokens []csToken) []csToken {
	out := make([]csToken, 0, len(tokens))
	for _, tok := range tokens {
		if tok.operator && tok.text == "EMC" {
			continue
		}
		out = append(out, tok)
	}
	return out
}

// isVariableOperand reports whether an operand is of a kind the folio changes
// from sheet to sheet: a number, a string, or an array of them (the operand of
// TJ).
func isVariableOperand(text string) bool {
	if text == "" {
		return false
	}
	switch text[0] {
	case '(', '<', '[':
		return true
	}
	_, err := strconv.ParseFloat(text, 64)
	return err == nil
}

// checkRemovable reports why a trailing group may not be deleted, or nil when
// every operator in it is one this code has reasoned about.
func checkRemovable(tokens []csToken) error {
	for _, tok := range tokens {
		if !tok.operator {
			continue
		}
		if !safeCoverOperators[tok.text] {
			return fmt.Errorf("pdf cover: the running head uses the %q operator, which is not safe to remove", tok.text)
		}
	}
	return nil
}

// indirectStream returns the dictionary body and the raw (still encoded) bytes
// of a stream object.
func indirectStream(data []byte, offsets map[int]int, num, gen int) (dict, raw []byte, ok bool) {
	off, found := offsets[num]
	if !found || off <= 0 || off >= len(data) {
		return nil, nil, false
	}
	header := fmt.Sprintf("%d %d obj", num, gen)
	if off+len(header) > len(data) || string(data[off:off+len(header)]) != header {
		return nil, nil, false
	}
	i := skipPDFSpace(data, off+len(header))
	if i+1 >= len(data) || data[i] != '<' || data[i+1] != '<' {
		return nil, nil, false
	}
	dictEnd, tokOK := scanPDFToken(data, i)
	if !tokOK {
		return nil, nil, false
	}
	dict = data[i+2 : dictEnd-2]
	i = skipPDFSpace(data, dictEnd)
	const kw = "stream"
	if i+len(kw) > len(data) || string(data[i:i+len(kw)]) != kw {
		return nil, nil, false
	}
	i += len(kw)
	// The keyword is followed by CRLF or LF — never by CR alone (PDF 32000-1
	// §7.3.8.1).
	if i < len(data) && data[i] == '\r' {
		i++
	}
	if i >= len(data) || data[i] != '\n' {
		return nil, nil, false
	}
	i++
	// An indirect /Length would need another object lookup; Chrome writes it
	// directly, and a stream whose length this code has to guess at is not one
	// it should be rewriting.
	length, lengthOK := dictInt(dict, "Length")
	if !lengthOK || length < 0 || i+length > len(data) {
		return nil, nil, false
	}
	raw = data[i : i+length]
	after := skipPDFSpace(data, i+length)
	const end = "endstream"
	if after+len(end) > len(data) || string(data[after:after+len(end)]) != end {
		return nil, nil, false
	}
	return dict, raw, true
}

// replacePageContent appends an incremental update that gives page a freshly
// compressed content stream.
//
// The update rewrites the page object in place of the file's existing one and
// adds the new stream beside it; every other object, and so every named
// destination, outline entry and structure element pointing at this page, keeps
// working because the page keeps its object number.
func replacePageContent(data, trailer []byte, startxref int, page printedPage, content []byte) ([]byte, error) {
	size, ok := dictInt(trailer, "Size")
	if !ok {
		return data, errors.New("pdf cover: trailer has no /Size")
	}
	rootNum, rootGen, ok := dictObjectRef(trailer, "Root")
	if !ok {
		return data, errors.New("pdf cover: trailer has no /Root reference")
	}
	entries, ok := parsePDFDict(page.dict)
	if !ok {
		return data, fmt.Errorf("pdf cover: cannot parse page object %d", page.objNum)
	}
	compressed, err := encodeContentStream(content)
	if err != nil {
		return data, fmt.Errorf("pdf cover: %w", err)
	}

	streamNum := size
	var pageDict strings.Builder
	pageDict.WriteString("<<")
	for _, entry := range entries {
		pageDict.WriteString("/" + entry.key + " ")
		if entry.key == "Contents" {
			fmt.Fprintf(&pageDict, "%d 0 R", streamNum)
		} else {
			pageDict.Write(entry.value)
		}
		pageDict.WriteString("\n")
	}
	pageDict.WriteString(">>")

	out := make([]byte, len(data), len(data)+len(compressed)+len(pageDict.String())+512)
	copy(out, data)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}

	pageOffset := len(out)
	out = append(out, fmt.Sprintf("%d 0 obj\n%s\nendobj\n", page.objNum, pageDict.String())...)
	streamOffset := len(out)
	out = append(out, fmt.Sprintf("%d 0 obj\n<</Filter /FlateDecode\n/Length %d>> stream\n", streamNum, len(compressed))...)
	out = append(out, compressed...)
	out = append(out, "\nendstream\nendobj\n"...)

	type xrefEntry struct {
		num    int
		offset int
	}
	added := []xrefEntry{{num: page.objNum, offset: pageOffset}, {num: streamNum, offset: streamOffset}}
	sort.Slice(added, func(i, j int) bool { return added[i].num < added[j].num })

	xrefOffset := len(out)
	var update strings.Builder
	update.WriteString("xref\n")
	// Object 0, the head of the free list, is repeated here even though nothing
	// in this section changes it: a cross-reference section whose first
	// subsection does not start at object 0 makes strict readers (pypdf among
	// them) decide the producer numbered its objects from one and renumber
	// everything they load to compensate.
	update.WriteString("0 1\n0000000000 65535 f \n")
	for _, e := range added {
		fmt.Fprintf(&update, "%d 1\n%010d %05d n \n", e.num, e.offset, 0)
	}
	fmt.Fprintf(&update, "trailer\n<< /Size %d /Root %d %d R", streamNum+1, rootNum, rootGen)
	if infoNum, infoGen, found := dictObjectRef(trailer, "Info"); found {
		fmt.Fprintf(&update, " /Info %d %d R", infoNum, infoGen)
	}
	fmt.Fprintf(&update, " /Prev %d", startxref)
	if id, found := dictRawValue(trailer, "ID"); found {
		update.WriteString(" /ID ")
		update.Write(bytes.TrimSpace(id))
	}
	update.WriteString(" >>\nstartxref\n")
	fmt.Fprintf(&update, "%d\n%%%%EOF\n", xrefOffset)
	out = append(out, update.String()...)
	return out, nil
}
