// epub_xhtml.go turns chapter HTML into well-formed XHTML for EPUB packaging.
//
// EPUB reading systems parse chapter documents with a strict XML parser: a
// single unquoted attribute value or unbalanced tag makes the whole book
// unopenable. Chapter HTML can contain arbitrary author-written raw HTML, so
// the conversion is done by parsing the fragment into a DOM and re-serializing
// it under XHTML rules, rather than by patching the markup with regexes.
package output

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// epubScriptWarningOnce ensures the dropped-script warning is logged only once.
var epubScriptWarningOnce sync.Once

// epubVoidElements are HTML elements that never have content and must be
// written as self-closing tags in XHTML.
var epubVoidElements = map[string]bool{
	"area": true, "base": true, "basefont": true, "br": true, "col": true,
	"embed": true, "frame": true, "hr": true, "img": true, "input": true,
	"isindex": true, "keygen": true, "link": true, "meta": true, "param": true,
	"source": true, "track": true, "wbr": true,
}

// epubBooleanAttrs are HTML attributes that may be written without a value.
// XHTML requires attr="attr" instead.
var epubBooleanAttrs = map[string]bool{
	"checked": true, "disabled": true, "selected": true, "readonly": true,
	"multiple": true, "autofocus": true, "autoplay": true, "controls": true,
	"loop": true, "muted": true, "open": true, "required": true,
	"reversed": true, "hidden": true, "defer": true, "async": true,
	"novalidate": true, "ismap": true, "nomodule": true, "default": true,
}

// xmlNamePattern matches names XML accepts for elements and attributes. Markup
// carrying framework syntax (`@click`, `:class`, `{{x}}`) is not representable
// in XML and is dropped rather than shipped in a broken document.
var xmlNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]*(?::[A-Za-z_][A-Za-z0-9._-]*)?$`)

// normalizeHTMLForXHTML converts chapter HTML into well-formed XHTML.
//
// The fragment is parsed with a real HTML parser and re-serialized, which
// quotes attribute values, balances tags, self-closes void elements, resolves
// named entities that XML does not define (&nbsp; and friends), and drops
// markup XML cannot represent. Mermaid blocks are also switched to <pre> here
// because EPUB has no Mermaid runtime and readers collapse the whitespace of
// the <div> the site output uses.
func normalizeHTMLForXHTML(fragment string) string {
	if strings.TrimSpace(fragment) == "" {
		return fragment
	}

	context := &html.Node{Type: html.ElementNode, DataAtom: atom.Body, Data: "body"}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), context)
	if err != nil {
		// Parsing an in-memory fragment has no failure mode in practice; if it
		// ever does, keep the original markup so no content is lost. The
		// well-formedness guard in Generate reports the resulting document.
		slog.Warn("Failed to parse chapter HTML for EPUB; writing it unchanged", slog.Any("error", err))
		return fragment
	}

	var b strings.Builder
	for _, n := range nodes {
		rewriteMermaidBlocks(n)
		writeXHTMLNode(&b, n)
	}
	return b.String()
}

// rewriteMermaidBlocks converts Mermaid containers into <pre> elements.
// Mermaid diagrams are rendered by a browser script that EPUB readers do not
// run, so the diagram source is what the reader sees; inside a <div> its line
// breaks collapse and it arrives as one unreadable paragraph.
func rewriteMermaidBlocks(n *html.Node) {
	if n.Type == html.ElementNode && n.Namespace == "" && n.Data == "div" && hasClass(n, "mermaid") {
		n.Data = "pre"
		n.DataAtom = atom.Pre
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewriteMermaidBlocks(c)
	}
}

// hasClass reports whether the element carries the given class name.
func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Namespace != "" || !strings.EqualFold(a.Key, "class") {
			continue
		}
		for _, f := range strings.Fields(a.Val) {
			if f == class {
				return true
			}
		}
	}
	return false
}

func writeXHTMLNode(b *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		if parent := n.Parent; parent != nil && parent.Type == html.ElementNode &&
			parent.Namespace == "" && parent.Data == "style" {
			// CSS is wrapped in a CDATA section marked with CSS comments so it
			// survives both XML and HTML parsing intact.
			b.WriteString("/*<![CDATA[*/")
			b.WriteString(stripXMLInvalidChars(strings.ReplaceAll(n.Data, "]]>", "]] >")))
			b.WriteString("/*]]>*/")
			return
		}
		b.WriteString(escapeXHTMLText(n.Data))
	case html.CommentNode:
		// "--" cannot appear inside an XML comment.
		comment := strings.ReplaceAll(stripXMLInvalidChars(n.Data), "--", "- -")
		b.WriteString("<!--" + strings.TrimSuffix(comment, "-") + "-->")
	case html.ElementNode:
		writeXHTMLElement(b, n)
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeXHTMLNode(b, c)
		}
	}
	// Doctype and raw nodes are dropped: wrapXHTML emits the document prolog.
}

func writeXHTMLElement(b *strings.Builder, n *html.Node) {
	name := n.Data
	if n.Namespace == "" {
		name = strings.ToLower(name)
	}

	// EPUB manifests must declare scripted documents, and no reading system is
	// required to run scripts, so an author's <script> is dead weight that only
	// risks rejection. Its content is never displayed either way.
	if n.Namespace == "" && name == "script" {
		epubScriptWarningOnce.Do(func() {
			slog.Warn("Removed <script> from ePub chapter: EPUB readers are not required to run scripts.")
		})
		return
	}

	if !xmlNamePattern.MatchString(name) {
		// Unrepresentable element: keep the children, drop the wrapper.
		writeXHTMLChildren(b, n)
		return
	}

	b.WriteString("<" + name)
	writeXHTMLAttrs(b, n)
	// Foreign content only validates with its namespace declared, and the HTML
	// parser records the namespace out-of-band rather than as an attribute.
	if n.Namespace != "" && n.Parent != nil && n.Parent.Namespace != n.Namespace && !hasAttr(n, "xmlns") {
		switch n.Namespace {
		case "svg":
			b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
		case "math":
			b.WriteString(` xmlns="http://www.w3.org/1998/Math/MathML"`)
		}
	}

	if n.FirstChild == nil && (epubVoidElements[name] || n.Namespace != "") {
		b.WriteString(" />")
		return
	}
	b.WriteString(">")
	writeXHTMLChildren(b, n)
	b.WriteString("</" + name + ">")
}

func writeXHTMLChildren(b *strings.Builder, n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		writeXHTMLNode(b, c)
	}
}

func writeXHTMLAttrs(b *strings.Builder, n *html.Node) {
	seen := make(map[string]bool, len(n.Attr))
	for _, a := range n.Attr {
		name := a.Key
		if a.Namespace != "" {
			name = a.Namespace + ":" + a.Key
		}
		if !xmlNamePattern.MatchString(name) {
			continue
		}
		// XML forbids repeating an attribute; HTML tolerates it.
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true

		value := a.Val
		if value == "" && epubBooleanAttrs[strings.ToLower(a.Key)] {
			value = strings.ToLower(a.Key)
		}
		fmt.Fprintf(b, " %s=\"%s\"", name, escapeXHTMLAttr(value))
	}
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return true
		}
	}
	return false
}

var (
	xhtmlTextEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	xhtmlAttrEscaper = strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;",
		"\n", "&#10;", "\r", "&#13;", "\t", "&#9;",
	)
)

func escapeXHTMLText(s string) string {
	return xhtmlTextEscaper.Replace(stripXMLInvalidChars(s))
}

func escapeXHTMLAttr(s string) string {
	return xhtmlAttrEscaper.Replace(stripXMLInvalidChars(s))
}

// stripXMLInvalidChars removes control characters that XML 1.0 does not allow
// in character data. They can reach here from a stray byte in a source file and
// would make the whole document unparseable.
func stripXMLInvalidChars(s string) string {
	if strings.IndexFunc(s, isXMLInvalidRune) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isXMLInvalidRune(r) {
			return -1
		}
		return r
	}, s)
}

func isXMLInvalidRune(r rune) bool {
	switch {
	case r == '\t' || r == '\n' || r == '\r':
		return false
	case r < 0x20:
		return true
	case r >= 0x7F && r <= 0x9F:
		// C1 controls; only valid in XML 1.1, which EPUB does not use.
		return true
	case r >= 0xD800 && r <= 0xDFFF:
		return true
	case r == 0xFFFE || r == 0xFFFF:
		return true
	default:
		return false
	}
}

// validateXHTML reports whether a generated document parses as XML. Strict
// reading systems refuse a book outright when one chapter is malformed, and a
// build that "succeeded" only to produce an unopenable file is worse than a
// build that fails, so this runs before the document is written to the archive.
func validateXHTML(name, document string) error {
	decoder := xml.NewDecoder(strings.NewReader(document))
	// Chapter documents legitimately reference no external DTD; entities other
	// than the five XML built-ins are already resolved by the serializer.
	decoder.Strict = true
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s is not well-formed XML: %w", name, err)
		}
	}
}
