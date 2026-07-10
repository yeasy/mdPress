// Package markdown provides Markdown parsing and HTML conversion.
// Built on the goldmark library, it supports GFM extensions, syntax highlighting, footnotes, and more.
//
// Core types:
//   - Parser: Markdown parser; call Parse() to get HTML and a heading list
//   - HeadingInfo: Heading metadata (level, text, ID), used for TOC generation
//
// Code blocks are highlighted with CSS classes (chroma), so renderers must
// embed the stylesheets from HighlightCSSLight/HighlightCSSDark for token
// colors to appear; the dark stylesheet only applies under DarkModeSelectors.
//
// Usage example:
//
//	p := markdown.NewParser(markdown.WithCodeTheme("monokai"))
//	html, headings, err := p.Parse(source)
package markdown
