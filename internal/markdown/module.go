// Package markdown provides Markdown parsing and HTML conversion.
// Built on the goldmark library, it supports GFM extensions, syntax highlighting, footnotes, and more.
//
// Core types:
//   - Parser: Markdown parser; call Parse() to get HTML and a heading list
//   - HeadingInfo: Heading metadata (level, text, ID), used for TOC generation
//
// Usage example:
//
//	p := markdown.NewParser(markdown.WithCodeTheme("monokai"))
//	html, headings, err := p.Parse(source)
package markdown
