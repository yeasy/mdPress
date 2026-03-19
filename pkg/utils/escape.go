// Package utils provides shared utility functions.

package utils

import "strings"

// EscapeHTML escapes HTML special characters including single quotes.
// This is the canonical HTML escaping function used throughout mdpress.
func EscapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// EscapeXML escapes XML special characters.
// Similar to EscapeHTML but uses &apos; for single quotes per the XML specification.
func EscapeXML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}

// EscapeAttr escapes an HTML attribute value.
func EscapeAttr(s string) string {
	return EscapeHTML(s)
}
