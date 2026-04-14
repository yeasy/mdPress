package utils

import (
	"testing"
)

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "script tag",
			input:    "<script>",
			expected: "&lt;script&gt;",
		},
		{
			name:     "double quotes",
			input:    `"quotes"`,
			expected: "&quot;quotes&quot;",
		},
		{
			name:     "single quotes",
			input:    "'single'",
			expected: "&#39;single&#39;",
		},
		{
			name:     "ampersand",
			input:    "&amp;",
			expected: "&amp;amp;",
		},
		{
			name:     "unicode café",
			input:    "café",
			expected: "café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "script tag",
			input:    "<script>",
			expected: "&lt;script&gt;",
		},
		{
			name:     "double quotes",
			input:    `"quotes"`,
			expected: "&quot;quotes&quot;",
		},
		{
			name:     "single quotes",
			input:    "'single'",
			expected: "&apos;single&apos;",
		},
		{
			name:     "ampersand",
			input:    "&amp;",
			expected: "&amp;amp;",
		},
		{
			name:     "unicode café",
			input:    "café",
			expected: "café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeAttr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "script tag",
			input:    "<script>",
			expected: "&lt;script&gt;",
		},
		{
			name:     "double quotes",
			input:    `"quotes"`,
			expected: "&quot;quotes&quot;",
		},
		{
			name:     "single quotes",
			input:    "'single'",
			expected: "&#39;single&#39;",
		},
		{
			name:     "ampersand",
			input:    "&amp;",
			expected: "&amp;amp;",
		},
		{
			name:     "unicode café",
			input:    "café",
			expected: "café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeAttr(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeAttr(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEscapeAttrDelegates verifies EscapeAttr produces the same output as EscapeHTML.
func TestEscapeAttrDelegates(t *testing.T) {
	inputs := []string{"", "hello", "<b>", `"x"`, "'y'", "&z", "café"}
	for _, input := range inputs {
		if EscapeAttr(input) != EscapeHTML(input) {
			t.Errorf("EscapeAttr(%q) != EscapeHTML(%q)", input, input)
		}
	}
}

func TestSanitizeCSS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain CSS",
			input:    "body { color: red; }",
			expected: "body { color: red; }",
		},
		{
			name:     "style tag breakout",
			input:    "body{}</style><script>alert(1)</script>",
			expected: `body{}<\/style><script>alert(1)</script>`,
		},
		{
			name:     "case insensitive breakout",
			input:    "body{}</STYLE><script>alert(1)</script>",
			expected: `body{}<\/style><script>alert(1)</script>`,
		},
		{
			name:     "mixed case breakout",
			input:    "body{}</sTyLe><script>alert(1)</script>",
			expected: `body{}<\/style><script>alert(1)</script>`,
		},
		{
			name:     "multiple occurrences",
			input:    "</style>foo</style>",
			expected: `<\/style>foo<\/style>`,
		},
		{
			name:     "partial match not modified",
			input:    "</styl body{}",
			expected: "</styl body{}",
		},
		{
			name:     "style in comment",
			input:    "/* </style> */",
			expected: `/* <\/style> */`,
		},
		{
			name:     "blocks @import",
			input:    `@import url("https://evil.com/steal.css");`,
			expected: `/* blocked import */ /* blocked external url */;`,
		},
		{
			name:     "blocks @import case insensitive",
			input:    `@Import url("https://evil.com");`,
			expected: `/* blocked import */ /* blocked external url */;`,
		},
		{
			name:     "blocks expression()",
			input:    `width: expression(document.body.clientWidth);`,
			expected: `width: /* blocked expression */(document.body.clientWidth);`,
		},
		{
			name:     "blocks expression case insensitive",
			input:    `width: Expression (alert(1));`,
			expected: `width: /* blocked expression */(alert(1));`,
		},
		{
			name:     "preserves legitimate CSS with import-like text",
			input:    `/* important note */ .important { color: red; }`,
			expected: `/* important note */ .important { color: red; }`,
		},
		{
			name:     "blocks external url()",
			input:    `body { background: url("https://attacker.com/track"); }`,
			expected: `body { background: /* blocked external url */; }`,
		},
		{
			name:     "blocks external url case insensitive",
			input:    `body { background: URL( 'HTTP://evil.com/img.png' ); }`,
			expected: `body { background: /* blocked external url */; }`,
		},
		{
			name:     "preserves local url()",
			input:    `body { background: url("images/bg.png"); }`,
			expected: `body { background: url("images/bg.png"); }`,
		},
		{
			name:     "preserves data: url()",
			input:    `body { background: url(data:image/png;base64,abc); }`,
			expected: `body { background: url(data:image/png;base64,abc); }`,
		},
		{
			name:     "allows @font-face with local src",
			input:    `@font-face { font-family: x; src: local("MyFont"), url("fonts/my.woff"); }`,
			expected: `@font-face { font-family: x; src: local("MyFont"), url("fonts/my.woff"); }`,
		},
		{
			name:     "blocks external url inside @font-face",
			input:    `@font-face { font-family: x; src: url("https://evil.com/font.woff"); }`,
			expected: `@font-face { font-family: x; src: /* blocked external url */; }`,
		},
		{
			name:     "blocks protocol-relative url",
			input:    `body { background: url("//evil.com/track.png"); }`,
			expected: `body { background: /* blocked external url */; }`,
		},
		{
			name:     "blocks protocol-relative url in font-face",
			input:    `@font-face { src: url('//evil.com/font.woff'); }`,
			expected: `@font-face { src: /* blocked external url */; }`,
		},
		{
			name:     "blocks javascript url",
			input:    `body { background: url("javascript:alert(1)"); }`,
			expected: `body { background: /* blocked uri scheme */(alert(1)"); }`,
		},
		{
			name:     "blocks javascript url case insensitive",
			input:    `body { background: URL( 'JavaScript:void(0)' ); }`,
			expected: `body { background: /* blocked uri scheme */(void(0)' ); }`,
		},
		{
			name:     "blocks vbscript url in css",
			input:    `body { background: url("vbscript:MsgBox(1)"); }`,
			expected: `body { background: /* blocked uri scheme */(MsgBox(1)"); }`,
		},
		{
			name:     "blocks behavior property",
			input:    `div { behavior: url(malicious.htc); }`,
			expected: `div { /* blocked behavior */ url(malicious.htc); }`,
		},
		{
			name:     "blocks behavior case insensitive",
			input:    `div { Behavior: url(evil.htc); }`,
			expected: `div { /* blocked behavior */ url(evil.htc); }`,
		},
		{
			name:     "allows scroll-behavior property",
			input:    `html { scroll-behavior: smooth; }`,
			expected: `html { scroll-behavior: smooth; }`,
		},
		{
			name:     "blocks -moz-binding",
			input:    `div { -moz-binding: url("xbl.xml#exploit"); }`,
			expected: `div { /* blocked moz-binding */ url("xbl.xml#exploit"); }`,
		},
		{
			name:     "blocks -moz-binding case insensitive",
			input:    `div { -MOZ-BINDING: url("evil.xml"); }`,
			expected: `div { /* blocked moz-binding */ url("evil.xml"); }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCSS(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeCSS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
