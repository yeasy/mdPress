// Package output implements non-PDF output generators such as HTML and ePub.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// HTMLGenerator writes rendered HTML into a static site directory.
type HTMLGenerator struct{}

// NewHTMLGenerator creates an HTML generator.
func NewHTMLGenerator() *HTMLGenerator {
	return &HTMLGenerator{}
}

// Generate writes the full HTML and optional chapter pages into a static site directory.
func (g *HTMLGenerator) Generate(fullHTML string, outputDir string, chapterHTMLs map[string]string) error {
	if err := utils.EnsureDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write the main entry page.
	indexPath := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("failed to write index.html: %w", err)
	}

	// Write chapter pages when provided.
	for name, html := range chapterHTMLs {
		pageName := slugify(name) + ".html"
		if err := validateFilename(outputDir, pageName); err != nil {
			return fmt.Errorf("invalid chapter name %q: %w", name, err)
		}
		pagePath := filepath.Join(outputDir, pageName)
		if err := os.WriteFile(pagePath, []byte(html), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", pageName, err)
		}
	}

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r > 127 { // Preserve Unicode characters.
			return r
		}
		return -1
	}, s)
	return s
}
