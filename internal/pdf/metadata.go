package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// DocumentMetadata is the document information mdPress records in a PDF's
// /Info dictionary. Empty fields are left out of the dictionary entirely.
type DocumentMetadata struct {
	Title    string
	Author   string
	Subject  string
	Keywords string
	Creator  string
}

// IsZero reports whether no metadata field is set.
func (m DocumentMetadata) IsZero() bool {
	return m.Title == "" && m.Author == "" && m.Subject == "" &&
		m.Keywords == "" && m.Creator == ""
}

// buildTime returns the timestamp to stamp into generated documents.
//
// SOURCE_DATE_EPOCH (https://reproducible-builds.org/specs/source-date-epoch/)
// is honored so that rebuilding the same sources produces byte-identical PDFs;
// distributions and CI pipelines rely on that to verify artifacts.
func buildTime() time.Time {
	if raw := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); raw != "" {
		if secs, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return time.Unix(secs, 0).UTC()
		}
	}
	return time.Now().UTC()
}

// pdfDate formats a timestamp as a PDF date string (PDF 32000-1 §7.9.4).
func pdfDate(t time.Time) string {
	return t.UTC().Format("D:20060102150405") + "+00'00'"
}

// pdfTextString encodes s as a PDF string object. Pure-ASCII values become
// readable literal strings; anything else is written as UTF-16BE with a byte
// order mark, the only encoding PDF viewers reliably decode for non-ASCII
// document information.
func pdfTextString(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			ascii = false
			break
		}
	}
	if ascii {
		r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
		return "(" + r.Replace(s) + ")"
	}
	var buf bytes.Buffer
	buf.WriteString("<FEFF")
	for _, u := range utf16.Encode([]rune(s)) {
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], u)
		fmt.Fprintf(&buf, "%02X%02X", b[0], b[1])
	}
	buf.WriteString(">")
	return buf.String()
}

// managedInfoKeys are the /Info entries mdPress owns. Everything else found in
// the original dictionary (notably /Producer, which names the rendering engine)
// is carried over untouched.
var managedInfoKeys = map[string]bool{
	"Title":        true,
	"Author":       true,
	"Subject":      true,
	"Keywords":     true,
	"Creator":      true,
	"CreationDate": true,
	"ModDate":      true,
}

// SetDocumentInfo rewrites the /Info dictionary of an in-memory PDF so it
// carries mdPress's document metadata and a deterministic timestamp.
//
// The rewrite is done in place whenever the new dictionary fits inside the
// bytes the old one occupied — that keeps every cross-reference offset valid
// and, because the replaced bytes included Chrome's wall-clock timestamps,
// makes the output reproducible. When the new dictionary is larger the old
// object is emptied and a replacement is appended as a standard incremental
// update.
//
// Any PDF this function cannot confidently parse is returned unchanged along
// with an error, so a failure degrades to "metadata not set" rather than a
// corrupt file.
func SetDocumentInfo(data []byte, meta DocumentMetadata, ts time.Time) ([]byte, error) {
	offsets, trailer, startxref, ok := resolveXref(data)
	if !ok {
		return data, errors.New("pdf metadata: cross-reference table not understood")
	}
	if _, found := dictRawValue(trailer, "Encrypt"); found {
		return data, errors.New("pdf metadata: encrypted documents are not modified")
	}
	infoNum, infoGen, ok := dictObjectRef(trailer, "Info")
	if !ok {
		return data, errors.New("pdf metadata: trailer has no /Info reference")
	}
	infoOffset, ok := offsets[infoNum]
	if !ok || infoOffset <= 0 || infoOffset >= len(data) {
		return data, fmt.Errorf("pdf metadata: no offset for /Info object %d", infoNum)
	}
	objStart, objEnd, dictBody, ok := readIndirectDict(data, infoOffset, infoNum, infoGen)
	if !ok {
		return data, fmt.Errorf("pdf metadata: /Info object %d is not a dictionary", infoNum)
	}

	entries, ok := parsePDFDict(dictBody)
	if !ok {
		return data, fmt.Errorf("pdf metadata: cannot parse /Info object %d", infoNum)
	}
	newDict := buildInfoDict(entries, meta, ts)

	replacement := fmt.Sprintf("%d %d obj\n%s\nendobj", infoNum, infoGen, newDict)
	span := objEnd - objStart
	if len(replacement) <= span {
		out := make([]byte, len(data))
		copy(out, data)
		copy(out[objStart:], replacement)
		// Trailing spaces sit between two indirect objects, where PDF allows
		// arbitrary whitespace.
		for i := objStart + len(replacement); i < objEnd; i++ {
			out[i] = ' '
		}
		return out, nil
	}

	// The replacement does not fit. Empty the original object so its
	// wall-clock timestamps no longer appear in the file, then append the real
	// dictionary as a new object in an incremental update.
	emptied := fmt.Sprintf("%d %d obj\n<<>>\nendobj", infoNum, infoGen)
	if len(emptied) > span {
		return data, fmt.Errorf("pdf metadata: /Info object %d is too small to rewrite", infoNum)
	}
	rootRefNum, rootRefGen, ok := dictObjectRef(trailer, "Root")
	if !ok {
		return data, errors.New("pdf metadata: trailer has no /Root reference")
	}
	size, ok := dictInt(trailer, "Size")
	if !ok {
		return data, errors.New("pdf metadata: trailer has no /Size")
	}

	out := make([]byte, len(data))
	copy(out, data)
	copy(out[objStart:], emptied)
	for i := objStart + len(emptied); i < objEnd; i++ {
		out[i] = ' '
	}
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}

	newNum := size
	newOffset := len(out)
	out = append(out, fmt.Sprintf("%d 0 obj\n%s\nendobj\n", newNum, newDict)...)
	xrefOffset := len(out)
	var update strings.Builder
	update.WriteString("xref\n")
	fmt.Fprintf(&update, "%d 1\n", newNum)
	fmt.Fprintf(&update, "%010d %05d n \n", newOffset, 0)
	update.WriteString("trailer\n<< /Size ")
	fmt.Fprintf(&update, "%d /Root %d %d R /Info %d 0 R /Prev %d",
		newNum+1, rootRefNum, rootRefGen, newNum, startxref)
	if id, found := dictRawValue(trailer, "ID"); found {
		update.WriteString(" /ID ")
		update.Write(id)
	}
	update.WriteString(" >>\nstartxref\n")
	fmt.Fprintf(&update, "%d\n%%%%EOF\n", xrefOffset)
	out = append(out, update.String()...)
	return out, nil
}

// buildInfoDict serializes the new /Info dictionary: mdPress's own values
// first, then every preserved entry from the original dictionary in a stable
// order so that repeated builds produce identical bytes.
func buildInfoDict(original []pdfDictEntry, meta DocumentMetadata, ts time.Time) string {
	var b strings.Builder
	b.WriteString("<<")
	write := func(key, value string) {
		if value == "" {
			return
		}
		b.WriteString("/" + key + " " + pdfTextString(value) + "\n")
	}
	write("Title", meta.Title)
	write("Author", meta.Author)
	write("Subject", meta.Subject)
	write("Keywords", meta.Keywords)
	write("Creator", meta.Creator)
	b.WriteString("/CreationDate " + pdfTextString(pdfDate(ts)) + "\n")
	b.WriteString("/ModDate " + pdfTextString(pdfDate(ts)) + "\n")

	preserved := make([]pdfDictEntry, 0, len(original))
	for _, e := range original {
		if managedInfoKeys[e.key] {
			continue
		}
		preserved = append(preserved, e)
	}
	sort.Slice(preserved, func(i, j int) bool { return preserved[i].key < preserved[j].key })
	for _, e := range preserved {
		b.WriteString("/" + e.key + " ")
		b.Write(e.value)
		b.WriteString("\n")
	}
	b.WriteString(">>")
	return b.String()
}

// ---- minimal PDF syntax helpers -------------------------------------------
//
// These parse just enough of the classic (uncompressed) cross-reference table
// and dictionary syntax that Chrome and Typst emit. Anything unexpected makes
// them report failure, and the caller then leaves the document untouched.

type pdfDictEntry struct {
	key   string
	value []byte
}

func isPDFWhitespace(c byte) bool {
	return c == 0 || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

func isPDFDelimiter(c byte) bool {
	switch c {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

func skipPDFSpace(b []byte, i int) int {
	for i < len(b) {
		if isPDFWhitespace(b[i]) {
			i++
			continue
		}
		if b[i] == '%' { // comment runs to end of line
			for i < len(b) && b[i] != '\n' && b[i] != '\r' {
				i++
			}
			continue
		}
		return i
	}
	return i
}

// scanPDFToken returns the end index of the object starting at i.
func scanPDFToken(b []byte, i int) (int, bool) {
	if i >= len(b) {
		return 0, false
	}
	switch {
	case b[i] == '(':
		depth := 0
		for ; i < len(b); i++ {
			switch b[i] {
			case '\\':
				i++
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return i + 1, true
				}
			}
		}
		return 0, false
	case b[i] == '<' && i+1 < len(b) && b[i+1] == '<':
		depth := 0
		for i < len(b) {
			switch {
			case b[i] == '<' && i+1 < len(b) && b[i+1] == '<':
				depth++
				i += 2
			case b[i] == '>' && i+1 < len(b) && b[i+1] == '>':
				depth--
				i += 2
				if depth == 0 {
					return i, true
				}
			case b[i] == '(':
				end, ok := scanPDFToken(b, i)
				if !ok {
					return 0, false
				}
				i = end
			default:
				i++
			}
		}
		return 0, false
	case b[i] == '<':
		for ; i < len(b); i++ {
			if b[i] == '>' {
				return i + 1, true
			}
		}
		return 0, false
	case b[i] == '[':
		depth := 0
		for i < len(b) {
			switch b[i] {
			case '[':
				depth++
				i++
			case ']':
				depth--
				i++
				if depth == 0 {
					return i, true
				}
			case '(', '<':
				end, ok := scanPDFToken(b, i)
				if !ok {
					return 0, false
				}
				i = end
			default:
				i++
			}
		}
		return 0, false
	default:
		start := i
		if b[i] == '/' {
			i++
		}
		for i < len(b) && !isPDFWhitespace(b[i]) && !isPDFDelimiter(b[i]) {
			i++
		}
		if i == start {
			return 0, false
		}
		return i, true
	}
}

// parsePDFDict splits the body of a dictionary (the bytes between "<<" and
// ">>") into ordered key/value pairs with the raw value bytes preserved.
func parsePDFDict(body []byte) ([]pdfDictEntry, bool) {
	var entries []pdfDictEntry
	i := skipPDFSpace(body, 0)
	for i < len(body) {
		if body[i] != '/' {
			return nil, false
		}
		keyEnd, ok := scanPDFToken(body, i)
		if !ok {
			return nil, false
		}
		key := string(body[i+1 : keyEnd])
		i = skipPDFSpace(body, keyEnd)
		valEnd, ok := scanPDFToken(body, i)
		if !ok {
			return nil, false
		}
		value := body[i:valEnd]
		next := skipPDFSpace(body, valEnd)
		// An indirect reference is three tokens ("12 0 R"); keep them together.
		if isPDFInteger(value) {
			if genEnd, ok := scanPDFToken(body, next); ok && isPDFInteger(body[next:genEnd]) {
				after := skipPDFSpace(body, genEnd)
				if rEnd, ok := scanPDFToken(body, after); ok && string(body[after:rEnd]) == "R" {
					value = body[i:rEnd]
					next = skipPDFSpace(body, rEnd)
				}
			}
		}
		entries = append(entries, pdfDictEntry{key: key, value: value})
		i = next
	}
	return entries, true
}

func isPDFInteger(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// dictRawValue returns the raw bytes of key's value in a dictionary body.
func dictRawValue(body []byte, key string) ([]byte, bool) {
	entries, ok := parsePDFDict(body)
	if !ok {
		return nil, false
	}
	for _, e := range entries {
		if e.key == key {
			return e.value, true
		}
	}
	return nil, false
}

// dictObjectRef returns the object and generation numbers of an indirect
// reference stored under key.
func dictObjectRef(body []byte, key string) (num, gen int, ok bool) {
	raw, found := dictRawValue(body, key)
	if !found {
		return 0, 0, false
	}
	fields := strings.Fields(string(raw))
	if len(fields) != 3 || fields[2] != "R" {
		return 0, 0, false
	}
	num, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, false
	}
	gen, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, false
	}
	return num, gen, true
}

func dictInt(body []byte, key string) (int, bool) {
	raw, found := dictRawValue(body, key)
	if !found {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return 0, false
	}
	return n, true
}

// readIndirectDict validates the object header at off and returns the byte
// span of the whole object plus the body of its dictionary.
func readIndirectDict(data []byte, off, num, gen int) (objStart, objEnd int, dictBody []byte, ok bool) {
	header := fmt.Sprintf("%d %d obj", num, gen)
	if off+len(header) > len(data) || string(data[off:off+len(header)]) != header {
		return 0, 0, nil, false
	}
	i := skipPDFSpace(data, off+len(header))
	if i+1 >= len(data) || data[i] != '<' || data[i+1] != '<' {
		return 0, 0, nil, false
	}
	dictEnd, ok := scanPDFToken(data, i)
	if !ok {
		return 0, 0, nil, false
	}
	rest := skipPDFSpace(data, dictEnd)
	const endobj = "endobj"
	if rest+len(endobj) > len(data) || string(data[rest:rest+len(endobj)]) != endobj {
		return 0, 0, nil, false
	}
	return off, rest + len(endobj), data[i+2 : dictEnd-2], true
}

// resolveXref walks the classic cross-reference chain from the last startxref
// and returns every in-use object offset, the newest trailer dictionary body,
// and the offset of the newest cross-reference section.
func resolveXref(data []byte) (offsets map[int]int, trailer []byte, startxref int, ok bool) {
	idx := bytes.LastIndex(data, []byte("startxref"))
	if idx < 0 {
		return nil, nil, 0, false
	}
	i := skipPDFSpace(data, idx+len("startxref"))
	end, tokOK := scanPDFToken(data, i)
	if !tokOK {
		return nil, nil, 0, false
	}
	off, err := strconv.Atoi(string(data[i:end]))
	if err != nil || off <= 0 || off >= len(data) {
		return nil, nil, 0, false
	}

	offsets = make(map[int]int)
	startxref = off
	// Bound the walk so a malformed /Prev cycle cannot spin forever.
	for depth := 0; depth < 64; depth++ {
		sectionOffsets, sectionTrailer, sectionOK := parseClassicXrefSection(data, off)
		if !sectionOK {
			return nil, nil, 0, false
		}
		for num, o := range sectionOffsets {
			if _, seen := offsets[num]; !seen {
				offsets[num] = o
			}
		}
		if trailer == nil {
			trailer = sectionTrailer
		}
		prev, hasPrev := dictInt(sectionTrailer, "Prev")
		if !hasPrev || prev <= 0 || prev >= len(data) || prev == off {
			return offsets, trailer, startxref, true
		}
		off = prev
	}
	return nil, nil, 0, false
}

// parseClassicXrefSection parses one "xref ... trailer <<...>>" section.
func parseClassicXrefSection(data []byte, off int) (map[int]int, []byte, bool) {
	i := skipPDFSpace(data, off)
	const kw = "xref"
	if i+len(kw) > len(data) || string(data[i:i+len(kw)]) != kw {
		return nil, nil, false
	}
	i += len(kw)
	offsets := make(map[int]int)
	for {
		i = skipPDFSpace(data, i)
		if i >= len(data) {
			return nil, nil, false
		}
		if bytes.HasPrefix(data[i:], []byte("trailer")) {
			i = skipPDFSpace(data, i+len("trailer"))
			if i+1 >= len(data) || data[i] != '<' || data[i+1] != '<' {
				return nil, nil, false
			}
			end, ok := scanPDFToken(data, i)
			if !ok {
				return nil, nil, false
			}
			return offsets, data[i+2 : end-2], true
		}
		start, next, ok := readXrefInt(data, i)
		if !ok {
			return nil, nil, false
		}
		count, next, ok := readXrefInt(data, next)
		if !ok {
			return nil, nil, false
		}
		i = next
		for n := 0; n < count; n++ {
			var objOff, gen int
			objOff, i, ok = readXrefInt(data, i)
			if !ok {
				return nil, nil, false
			}
			gen, i, ok = readXrefInt(data, i)
			if !ok {
				return nil, nil, false
			}
			_ = gen
			i = skipPDFSpace(data, i)
			if i >= len(data) {
				return nil, nil, false
			}
			kind := data[i]
			i++
			if kind == 'n' {
				offsets[start+n] = objOff
			} else if kind != 'f' {
				return nil, nil, false
			}
		}
	}
}

func readXrefInt(data []byte, i int) (value, next int, ok bool) {
	i = skipPDFSpace(data, i)
	start := i
	for i < len(data) && data[i] >= '0' && data[i] <= '9' {
		i++
	}
	if i == start {
		return 0, 0, false
	}
	v, err := strconv.Atoi(string(data[start:i]))
	if err != nil {
		return 0, 0, false
	}
	return v, i, true
}
