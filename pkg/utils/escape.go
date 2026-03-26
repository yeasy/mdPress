// Package utils provides shared utility functions.

package utils

import "strings"

// Pre-compiled replacers to avoid allocation on every call.
var (
	htmlReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	xmlReplacer = strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
)

// EscapeHTML escapes HTML special characters including single quotes.
// This is the canonical HTML escaping function used throughout mdpress.
func EscapeHTML(s string) string {
	return htmlReplacer.Replace(s)
}

// EscapeXML escapes XML special characters.
// Similar to EscapeHTML but uses &apos; for single quotes per the XML specification.
func EscapeXML(s string) string {
	return xmlReplacer.Replace(s)
}

// EscapeAttr escapes an HTML attribute value.
func EscapeAttr(s string) string {
	return EscapeHTML(s)
}
