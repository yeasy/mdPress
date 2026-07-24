package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
)

// This file reads and rewrites the content stream of a single printed page.
//
// It exists for one job: taking Chrome's running head and folio back off a
// full-bleed cover (see coverpage.go). Content streams are a different syntax
// from the object syntax in metadata.go — a flat postfix sequence of operands
// followed by an operator — so they need their own scanner, but the operand
// lexer is the same one, reused from there.

// maxContentStreamSize bounds how much a page's content stream may inflate to.
// A page of a book is tens of kilobytes; anything past this is either not a
// document mdPress produced or a decompression bomb, and neither is worth
// spending memory on for a cosmetic rewrite.
const maxContentStreamSize = 64 << 20

// csToken is one token of a content stream: either an operand (a number, a
// name, a string, an array, a dictionary) or the operator that consumes the
// operands before it.
type csToken struct {
	text     string
	start    int // byte offset of the token in the stream
	end      int // byte offset just past the token
	operator bool
}

// scanContentStream splits a page content stream into tokens.
//
// It reports failure rather than guessing on anything it does not fully
// understand, because every caller's fallback — leave the page as Chrome
// printed it — is safe, and a mis-scan would silently delete drawing
// operations from a reader's document.
func scanContentStream(s []byte) ([]csToken, bool) {
	var tokens []csToken
	i := skipPDFSpace(s, 0)
	for i < len(s) {
		end, ok := scanPDFToken(s, i)
		if !ok || end <= i {
			return nil, false
		}
		text := string(s[i:end])
		// Operators are the only bare keywords in a content stream; every
		// operand starts with a digit, a sign, a dot, '/', '(', '<' or '['.
		// The quote operators (' and ") are the two that do not start with a
		// letter.
		first := s[i]
		isOperator := (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') ||
			first == '\'' || first == '"'
		// An inline image puts raw image bytes between ID and EI, and those
		// bytes are not tokens: scanning on would read "q" and "Q" out of pixel
		// data and mislead the caller about where a drawing group ends. Chrome
		// draws images as XObjects and never emits BI, so refusing the whole
		// stream costs nothing and keeps the group boundaries trustworthy.
		if isOperator && text == "BI" {
			return nil, false
		}
		tokens = append(tokens, csToken{text: text, start: i, end: end, operator: isOperator})
		i = skipPDFSpace(s, end)
	}
	return tokens, true
}

// safeCoverOperators are the drawing operators a running head and folio may be
// built from. The list is deliberately an allowlist: the group is only removed
// from a page when every operator in it appears here, so an operator this code
// has not reasoned about — an image (Do), an inline image (BI), a shading (sh)
// or a marked-content opener (BDC/BMC), which would leave a dangling structure
// element behind — aborts the rewrite instead of being deleted blind.
var safeCoverOperators = map[string]bool{
	// Graphics state and clipping.
	"q": true, "Q": true, "cm": true, "gs": true, "w": true, "d": true,
	"i": true, "j": true, "J": true, "M": true, "ri": true,
	"W": true, "W*": true,
	// Paths (a header may draw a rule under itself).
	"m": true, "l": true, "c": true, "v": true, "y": true, "h": true, "re": true,
	"n": true, "f": true, "F": true, "f*": true, "S": true, "s": true,
	"B": true, "B*": true, "b": true, "b*": true,
	// Color.
	"g": true, "G": true, "rg": true, "RG": true, "k": true, "K": true,
	"cs": true, "CS": true, "sc": true, "scn": true, "SC": true, "SCN": true,
	// Text.
	"BT": true, "ET": true, "Tc": true, "Tw": true, "Tz": true, "TL": true,
	"Tf": true, "Tr": true, "Ts": true, "Td": true, "TD": true, "Tm": true,
	"T*": true, "Tj": true, "TJ": true, "'": true, `"`: true,
	// Chrome closes the document content's last marked-content sequence from
	// inside the running-head group, so EMC has to be tolerated here — and
	// re-emitted when the group is dropped, or the sequence never closes.
	"EMC": true,
}

// contentGroup is a balanced "q … Q" group at the top level of a content
// stream, given as the half-open token range [first, last].
type contentGroup struct {
	first int // index of the "q" token
	last  int // index of the "Q" token
}

// topLevelGroups returns the balanced "q … Q" groups at nesting depth zero, in
// order. An unbalanced stream yields no groups.
func topLevelGroups(tokens []csToken) []contentGroup {
	var groups []contentGroup
	depth, start := 0, -1
	for i, tok := range tokens {
		if !tok.operator {
			continue
		}
		switch tok.text {
		case "q":
			if depth == 0 {
				start = i
			}
			depth++
		case "Q":
			depth--
			if depth < 0 {
				return nil
			}
			if depth == 0 && start >= 0 {
				groups = append(groups, contentGroup{first: start, last: i})
				start = -1
			}
		}
	}
	if depth != 0 {
		return nil
	}
	return groups
}

// decodeContentStream inflates a Flate-compressed content stream. A stream
// with no filter is returned as-is; any other filter is refused, because this
// package only ever rewrites what Chrome itself wrote.
func decodeContentStream(raw []byte, filter []byte) ([]byte, error) {
	switch string(bytes.TrimSpace(filter)) {
	case "":
		return raw, nil
	case "/FlateDecode":
	default:
		return nil, errors.New("pdf: content stream filter is not /FlateDecode")
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close() //nolint:errcheck // read-only reader
	out, err := io.ReadAll(io.LimitReader(zr, maxContentStreamSize+1))
	if err != nil {
		return nil, err
	}
	if len(out) > maxContentStreamSize {
		return nil, errors.New("pdf: content stream is implausibly large")
	}
	return out, nil
}

// encodeContentStream deflates a rewritten content stream. Go's flate encoder
// is deterministic, so a rebuilt page still produces byte-identical output for
// identical input — which SOURCE_DATE_EPOCH builds depend on.
func encodeContentStream(s []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(s); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
