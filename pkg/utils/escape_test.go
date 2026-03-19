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
