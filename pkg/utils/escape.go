// Package utils provides shared utility functions.

package utils

import (
	"regexp"
	"strings"
)

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

// styleClosePattern matches "</style" (case-insensitive) which would break
// out of an inline <style> block if present in user-provided CSS.
var styleClosePattern = regexp.MustCompile(`(?i)</style`)

// cssImportPattern matches @import rules (case-insensitive) which could load
// external stylesheets and exfiltrate data via URL requests.
var cssImportPattern = regexp.MustCompile(`(?i)@import\b`)

// cssExpressionPattern matches expression() (legacy IE CSS expression) which
// could execute arbitrary JavaScript.
var cssExpressionPattern = regexp.MustCompile(`(?i)expression\s*\(`)

// cssExternalURLPattern matches url() references to external origins,
// including protocol-relative URLs (//host/...), which could exfiltrate
// data or load untrusted resources.
var cssExternalURLPattern = regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*(?:https?:)?//[^)]*\)`)

// cssJSURLPattern matches javascript: and vbscript: URIs inside url() values,
// which could execute code in some rendering engines (including headless Chrome
// for PDF). Note: data: URIs are intentionally excluded because they are
// legitimately used for inline images (e.g. data:image/png;base64,...) and
// cannot execute scripts when used as CSS url() values.
var cssJSURLPattern = regexp.MustCompile(`(?i)url\s*\(\s*['"]?\s*(?:javascript|vbscript)\s*:`)

// cssBehaviorPattern matches the legacy IE "behavior" CSS property which can
// load and execute HTC (HTML Component) files.  The pattern requires that
// "behavior" is NOT preceded by a hyphen or letter so that legitimate
// compound properties like "scroll-behavior" are not blocked.
var cssBehaviorPattern = regexp.MustCompile(`(?i)(^|[^a-zA-Z-])behavior\s*:`)

// cssMozBindingPattern matches the legacy Firefox "-moz-binding" CSS property
// which can load XBL bindings to execute JavaScript.
var cssMozBindingPattern = regexp.MustCompile(`(?i)-moz-binding\s*:`)

// SanitizeCSS removes sequences from CSS content that could break out of a
// <style> block or perform injection attacks. This prevents:
// - </style> tag breakout
// - @import-based data exfiltration
// - expression()-based script execution (legacy IE)
// - url() references to external HTTP(S) origins
// - javascript:/vbscript: URIs inside url() values
//
// Note: @font-face rules are allowed because external url() references are
// already blocked by cssExternalURLPattern. This permits legitimate local
// font declarations (e.g. src: local("...") or src: url("fonts/my.woff"))
// while still preventing data exfiltration via external URLs.
func SanitizeCSS(css string) string {
	css = styleClosePattern.ReplaceAllString(css, `<\/style`)
	css = cssImportPattern.ReplaceAllString(css, "/* blocked import */")
	css = cssExpressionPattern.ReplaceAllString(css, "/* blocked expression */(")
	css = cssExternalURLPattern.ReplaceAllString(css, "/* blocked external url */")
	css = cssJSURLPattern.ReplaceAllString(css, "/* blocked uri scheme */(")
	css = cssBehaviorPattern.ReplaceAllString(css, "${1}/* blocked behavior */")
	css = cssMozBindingPattern.ReplaceAllString(css, "/* blocked moz-binding */")
	return css
}
