package markdown_test

import (
	"fmt"

	"github.com/yeasy/mdpress/internal/markdown"
)

func ExampleNewParser() {
	parser := markdown.NewParser()
	html, headings, err := parser.Parse([]byte("# Hello\n\nWorld"))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("HTML length:", len(html) > 0)
	fmt.Println("Headings:", len(headings))
	// Output:
	// HTML length: true
	// Headings: 1
}
