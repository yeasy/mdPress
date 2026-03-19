// Package cover generates and renders book cover pages.
// It builds a styled HTML cover from book metadata such as title, author, and version.
package cover

import (
	"fmt"
	"strings"
	"time"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/pkg/utils"
)

// CoverGenerator builds the HTML cover page.
type CoverGenerator struct {
	meta config.BookMeta
}

// NewCoverGenerator creates a new cover generator from book metadata.
func NewCoverGenerator(meta config.BookMeta) *CoverGenerator {
	return &CoverGenerator{
		meta: meta,
	}
}

// RenderHTML returns a self-contained HTML cover page.
func (cg *CoverGenerator) RenderHTML() string {
	var buf strings.Builder

	// Write the HTML document head.
	buf.WriteString(`<!DOCTYPE html>` + "\n")
	buf.WriteString(`<html lang="en">` + "\n")
	buf.WriteString(`<head>` + "\n")
	buf.WriteString(`  <meta charset="UTF-8">` + "\n")
	buf.WriteString(`  <meta name="viewport" content="width=device-width, initial-scale=1.0">` + "\n")
	fmt.Fprintf(&buf, `  <title>%s</title>`+"\n", utils.EscapeHTML(cg.meta.Title))
	buf.WriteString(cg.renderStyles())
	buf.WriteString(`</head>` + "\n")
	buf.WriteString(`<body>` + "\n")
	buf.WriteString(cg.renderCoverContent())
	buf.WriteString(`</body>` + "\n")
	buf.WriteString(`</html>` + "\n")

	return buf.String()
}

// renderStyles generates the cover page CSS.
func (cg *CoverGenerator) renderStyles() string {
	var buf strings.Builder

	buf.WriteString(`  <style>` + "\n")

	// Reset styles and page layout.
	buf.WriteString(`    * {` + "\n")
	buf.WriteString(`      margin: 0;` + "\n")
	buf.WriteString(`      padding: 0;` + "\n")
	buf.WriteString(`      box-sizing: border-box;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Base html/body styling.
	buf.WriteString(`    html, body {` + "\n")
	buf.WriteString(`      width: 100%;` + "\n")
	buf.WriteString(`      height: 100%;` + "\n")
	buf.WriteString(`      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif;` + "\n")
	buf.WriteString(`      background-color: #f5f5f5;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Cover container styles.
	buf.WriteString(`    .cover-page {` + "\n")
	buf.WriteString(`      display: flex;` + "\n")
	buf.WriteString(`      align-items: center;` + "\n")
	buf.WriteString(`      justify-content: center;` + "\n")
	buf.WriteString(`      width: 100%;` + "\n")
	buf.WriteString(`      height: 100%;` + "\n")
	buf.WriteString(`      padding: 60px 40px;` + "\n")

	// Prefer a configured background color or image.
	if cg.meta.Cover.Background != "" {
		fmt.Fprintf(&buf, `      background-color: %s;`+"\n", cg.meta.Cover.Background)
		buf.WriteString(`      background-size: cover;` + "\n")
		buf.WriteString(`      background-position: center;` + "\n")
		buf.WriteString(`      background-attachment: fixed;` + "\n")
	} else if cg.meta.Cover.Image != "" {
		fmt.Fprintf(&buf, `      background-image: url('%s');`+"\n", escapeURL(cg.meta.Cover.Image))
		buf.WriteString(`      background-size: cover;` + "\n")
		buf.WriteString(`      background-position: center;` + "\n")
		buf.WriteString(`      background-attachment: fixed;` + "\n")
	} else {
		// Fall back to a default gradient background.
		buf.WriteString(`      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);` + "\n")
	}

	buf.WriteString(`    }` + "\n\n")

	// Cover content layout.
	buf.WriteString(`    .cover-content {` + "\n")
	buf.WriteString(`      text-align: center;` + "\n")
	buf.WriteString(`      color: white;` + "\n")
	buf.WriteString(`      max-width: 800px;` + "\n")
	buf.WriteString(`      text-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);` + "\n")
	buf.WriteString(`      animation: fadeIn 0.8s ease-in-out;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Fade-in animation.
	buf.WriteString(`    @keyframes fadeIn {` + "\n")
	buf.WriteString(`      from {` + "\n")
	buf.WriteString(`        opacity: 0;` + "\n")
	buf.WriteString(`        transform: translateY(-20px);` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`      to {` + "\n")
	buf.WriteString(`        opacity: 1;` + "\n")
	buf.WriteString(`        transform: translateY(0);` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Title styles.
	buf.WriteString(`    .cover-title {` + "\n")
	buf.WriteString(`      font-size: 64px;` + "\n")
	buf.WriteString(`      font-weight: 700;` + "\n")
	buf.WriteString(`      margin-bottom: 20px;` + "\n")
	buf.WriteString(`      letter-spacing: 2px;` + "\n")
	buf.WriteString(`      line-height: 1.2;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Subtitle styles.
	buf.WriteString(`    .cover-subtitle {` + "\n")
	buf.WriteString(`      font-size: 24px;` + "\n")
	buf.WriteString(`      font-weight: 300;` + "\n")
	buf.WriteString(`      margin-bottom: 60px;` + "\n")
	buf.WriteString(`      opacity: 0.9;` + "\n")
	buf.WriteString(`      letter-spacing: 1px;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Divider.
	buf.WriteString(`    .cover-divider {` + "\n")
	buf.WriteString(`      width: 100px;` + "\n")
	buf.WriteString(`      height: 2px;` + "\n")
	buf.WriteString(`      background-color: rgba(255, 255, 255, 0.5);` + "\n")
	buf.WriteString(`      margin: 30px auto;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Metadata container.
	buf.WriteString(`    .cover-metadata {` + "\n")
	buf.WriteString(`      margin-top: 50px;` + "\n")
	buf.WriteString(`      font-size: 18px;` + "\n")
	buf.WriteString(`      opacity: 0.85;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Metadata row.
	buf.WriteString(`    .cover-meta-item {` + "\n")
	buf.WriteString(`      margin: 10px 0;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Metadata label.
	buf.WriteString(`    .cover-meta-label {` + "\n")
	buf.WriteString(`      display: inline-block;` + "\n")
	buf.WriteString(`      font-weight: 600;` + "\n")
	buf.WriteString(`      margin-right: 10px;` + "\n")
	buf.WriteString(`      min-width: 80px;` + "\n")
	buf.WriteString(`    }` + "\n\n")

	// Print-specific rules.
	buf.WriteString(`    @media print {` + "\n")
	buf.WriteString(`      html, body {` + "\n")
	buf.WriteString(`        width: 100%;` + "\n")
	buf.WriteString(`        height: 100%;` + "\n")
	buf.WriteString(`        margin: 0;` + "\n")
	buf.WriteString(`        padding: 0;` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`      .cover-page {` + "\n")
	buf.WriteString(`        page-break-after: always;` + "\n")
	buf.WriteString(`      }` + "\n")
	buf.WriteString(`    }` + "\n\n")

	buf.WriteString(`  </style>` + "\n")

	return buf.String()
}

// renderCoverContent builds the cover page HTML structure.
func (cg *CoverGenerator) renderCoverContent() string {
	var buf strings.Builder

	buf.WriteString(`  <div class="cover-page">` + "\n")
	buf.WriteString(`    <div class="cover-content">` + "\n")

	// Title
	if cg.meta.Title != "" {
		fmt.Fprintf(&buf, `      <h1 class="cover-title">%s</h1>`+"\n", utils.EscapeHTML(cg.meta.Title))
	}

	// Subtitle
	if cg.meta.Subtitle != "" {
		fmt.Fprintf(&buf, `      <h2 class="cover-subtitle">%s</h2>`+"\n", utils.EscapeHTML(cg.meta.Subtitle))
	}

	// Divider
	buf.WriteString(`      <div class="cover-divider"></div>` + "\n")

	// Metadata
	buf.WriteString(`      <div class="cover-metadata">` + "\n")

	// Author
	if cg.meta.Author != "" {
		buf.WriteString(`        <div class="cover-meta-item">` + "\n")
		buf.WriteString(`          <span class="cover-meta-label">Author</span>` + "\n")
		fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", utils.EscapeHTML(cg.meta.Author))
		buf.WriteString(`        </div>` + "\n")
	}

	// Version
	if cg.meta.Version != "" {
		buf.WriteString(`        <div class="cover-meta-item">` + "\n")
		buf.WriteString(`          <span class="cover-meta-label">Version</span>` + "\n")
		fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", utils.EscapeHTML(cg.meta.Version))
		buf.WriteString(`        </div>` + "\n")
	}

	// Date
	currentDate := time.Now().Format("2006-01-02")
	buf.WriteString(`        <div class="cover-meta-item">` + "\n")
	buf.WriteString(`          <span class="cover-meta-label">Date</span>` + "\n")
	fmt.Fprintf(&buf, `          <span>%s</span>`+"\n", currentDate)
	buf.WriteString(`        </div>` + "\n")

	buf.WriteString(`      </div>` + "\n")

	buf.WriteString(`    </div>` + "\n")
	buf.WriteString(`  </div>` + "\n")

	return buf.String()
}

// escapeURL escapes URL-sensitive characters.
func escapeURL(u string) string {
	// Apply minimal escaping to avoid injection.
	replacer := strings.NewReplacer(
		`'`, "\\'",
		`"`, `\"`,
		"\n", "\\n",
		"\r", "\\r",
	)
	return replacer.Replace(u)
}
