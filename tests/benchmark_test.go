//go:build benchmark

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
)

// BenchmarkMarkdownParsing benchmarks parsing Markdown content of various sizes.
func BenchmarkMarkdownParsing(b *testing.B) {
	parser := markdown.NewParser(markdown.WithCodeTheme("monokai"))

	sizes := []struct {
		name  string
		lines int
	}{
		{"small-10lines", 10},
		{"medium-100lines", 100},
		{"large-1000lines", 1000},
	}

	for _, size := range sizes {
		content := generateMarkdown(size.lines)
		b.Run(size.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = parser.Parse(content)
			}
		})
	}
}

// BenchmarkConfigDiscovery benchmarks zero-config auto-discovery.
func BenchmarkConfigDiscovery(b *testing.B) {
	// Create a temporary directory with sample Markdown files.
	tmpDir, err := os.MkdirTemp("", "mdpress-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create 50 sample Markdown files.
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("chapter%02d.md", i+1)
		content := fmt.Sprintf("# Chapter %d\n\nThis is chapter %d content.\n", i+1, i+1)
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			b.Fatalf("write benchmark fixture failed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = config.Discover(context.Background(), tmpDir)
	}
}

// generateMarkdown creates sample Markdown content with the given number of lines.
func generateMarkdown(lines int) []byte {
	var content []byte
	content = append(content, []byte("# Sample Document\n\n")...)
	for i := 0; i < lines; i++ {
		switch i % 5 {
		case 0:
			content = append(content, []byte(fmt.Sprintf("## Section %d\n\n", i/5+1))...)
		case 1:
			content = append(content, []byte("This is a paragraph with **bold** and *italic* text.\n\n")...)
		case 2:
			content = append(content, []byte("```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\n")...)
		case 3:
			content = append(content, []byte("| Column A | Column B |\n| --- | --- |\n| Cell 1 | Cell 2 |\n\n")...)
		case 4:
			content = append(content, []byte("- Item 1\n- Item 2\n- Item 3\n\n")...)
		}
	}
	return content
}
