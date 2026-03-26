// quickstart.go implements the quickstart subcommand.
// It creates a complete sample book project so users can see results quickly.
// The command generates book.yaml, README.md, sample chapters, image placeholders,
// and next-step instructions.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/pkg/utils"
)

// quickstartCmd creates a sample book project.
var quickstartCmd = &cobra.Command{
	Use:   "quickstart [directory]",
	Short: "Create a complete sample book project",
	Long: `Create a ready-to-build sample book project with config files,
multiple chapters, and example content.
You can build and preview it immediately.

Examples:
  mdpress quickstart my-book
  mdpress quickstart`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "my-book"
		if len(args) > 0 {
			dir = args[0]
		}
		return executeQuickstart(dir)
	},
}

// executeQuickstart creates the sample project.
func executeQuickstart(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Refuse to write into a non-empty directory.
	// Use os.ReadDir instead of filepath.Glob because Glob("*") does not
	// match hidden files/directories (e.g. .git, .DS_Store), which could
	// lead to silently overwriting an existing Git repository.
	if utils.FileExists(absDir) {
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory %s already exists and is not empty; choose a new directory name", dir)
		}
	}

	projectName := filepath.Base(absDir)

	utils.Header("mdpress Quickstart")
	utils.Info("Creating sample project: %s", projectName)
	fmt.Println()

	// 1. Create book.yaml.
	bookYAML := generateQuickstartBookYAML(projectName)
	if err := utils.WriteFile(filepath.Join(absDir, "book.yaml"), []byte(bookYAML)); err != nil {
		return fmt.Errorf("failed to create book.yaml: %w", err)
	}
	utils.Success("book.yaml - book configuration")

	// 2. Create README.md.
	readmeContent := generateQuickstartREADME(projectName)
	if err := utils.WriteFile(filepath.Join(absDir, "README.md"), []byte(readmeContent)); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}
	utils.Success("README.md - project overview")

	// 3. Create the preface.
	prefaceContent := generateQuickstartPreface()
	if err := utils.WriteFile(filepath.Join(absDir, "preface.md"), []byte(prefaceContent)); err != nil {
		return fmt.Errorf("failed to create preface.md: %w", err)
	}
	utils.Success("preface.md - preface")

	// 4. Create chapter 1.
	ch01Content := generateQuickstartChapter01()
	if err := utils.WriteFile(filepath.Join(absDir, "chapter01", "README.md"), []byte(ch01Content)); err != nil {
		return fmt.Errorf("failed to create chapter01: %w", err)
	}
	utils.Success("chapter01/README.md - Chapter 1: Getting Started")

	// 5. Create chapter 2.
	ch02Content := generateQuickstartChapter02()
	if err := utils.WriteFile(filepath.Join(absDir, "chapter02", "README.md"), []byte(ch02Content)); err != nil {
		return fmt.Errorf("failed to create chapter02: %w", err)
	}
	utils.Success("chapter02/README.md - Chapter 2: Advanced Usage")

	// 6. Create chapter 3.
	ch03Content := generateQuickstartChapter03()
	if err := utils.WriteFile(filepath.Join(absDir, "chapter03", "README.md"), []byte(ch03Content)); err != nil {
		return fmt.Errorf("failed to create chapter03: %w", err)
	}
	utils.Success("chapter03/README.md - Chapter 3: Best Practices")

	// 7. Create the image directory and placeholder notes.
	imgPlaceholder := generateImagePlaceholder()
	if err := utils.WriteFile(filepath.Join(absDir, "images", "README.md"), []byte(imgPlaceholder)); err != nil {
		return fmt.Errorf("failed to create images directory notes: %w", err)
	}
	utils.Success("images/ - image assets directory")

	// 8. Create the placeholder SVG cover.
	coverSVG := generatePlaceholderCoverSVG(projectName)
	if err := utils.WriteFile(filepath.Join(absDir, "images", "cover.svg"), []byte(coverSVG)); err != nil {
		return fmt.Errorf("failed to create placeholder cover: %w", err)
	}
	utils.Success("images/cover.svg - placeholder cover")

	// Print completion details and next steps.
	fmt.Println()
	utils.Header("Project Ready")
	fmt.Println()
	fmt.Println("  Run the following commands to preview the sample project:")
	fmt.Println()
	if colorEnabled := utils.IsColorEnabled(); colorEnabled {
		fmt.Printf("    %s\n", utils.Cyan("cd "+dir))
		fmt.Printf("    %s\n", utils.Cyan("mdpress build --format html"))
		fmt.Printf("    %s\n", utils.Cyan("mdpress serve"))
	} else {
		fmt.Printf("    cd %s\n", dir)
		fmt.Println("    mdpress build --format html")
		fmt.Println("    mdpress serve")
	}
	fmt.Println()
	fmt.Printf("  %s Edit book.yaml to update metadata and edit the .md files to add content\n", utils.Dim("Tip:"))
	fmt.Printf("  %s Run mdpress validate to verify the project configuration\n", utils.Dim("Tip:"))
	fmt.Println()

	return nil
}

// generateQuickstartBookYAML returns the quickstart book.yaml content.
func generateQuickstartBookYAML(projectName string) string {
	return fmt.Sprintf(`# mdpress book configuration
# Docs: https://github.com/yeasy/mdpress

book:
  title: "%s"
  subtitle: "A sample book created with mdpress"
  author: "Your Name"
  version: "1.0.0"
  language: "en-US"
  description: "A sample book generated by mdpress quickstart"
  cover:
    image: "images/cover.svg"

chapters:
  - title: "Preface"
    file: "preface.md"
  - title: "Chapter 1: Getting Started"
    file: "chapter01/README.md"
  - title: "Chapter 2: Advanced Usage"
    file: "chapter02/README.md"
  - title: "Chapter 3: Best Practices"
    file: "chapter03/README.md"

style:
  theme: "technical"
  page_size: "A4"
  font_family: "-apple-system, BlinkMacSystemFont, PingFang SC, Hiragino Sans GB, Microsoft YaHei, Noto Sans CJK SC, Noto Sans SC, Source Han Sans SC, Segoe UI, Helvetica Neue, Arial, sans-serif"
  font_size: "12pt"
  code_theme: "monokai"
  line_height: 1.6
  margin:
    top: 25
    bottom: 25
    left: 20
    right: 20
  header:
    left: "{{.Book.Title}}"
    right: "{{.Chapter.Title}}"
  footer:
    center: "{{.PageNum}}"

output:
  filename: "%s.pdf"
  toc: true
  cover: true
  header: true
  footer: true
`, projectName, projectName)
}

// generateQuickstartREADME returns the sample project README.
func generateQuickstartREADME(projectName string) string {
	return fmt.Sprintf(`# %s

This is a sample book project created with [mdpress](https://github.com/yeasy/mdpress).

## Quick Start

Build and preview the HTML version:

`+"```bash"+`
mdpress build --format html
mdpress serve
`+"```"+`

Build the PDF version:

`+"```bash"+`
mdpress build --format pdf
`+"```"+`

## Project Structure

`+"```"+`
%s/
├── book.yaml          # Book configuration
├── README.md          # Project overview
├── preface.md         # Preface
├── chapter01/         # Chapter 1
│   └── README.md
├── chapter02/         # Chapter 2
│   └── README.md
├── chapter03/         # Chapter 3
│   └── README.md
└── images/            # Image assets
    ├── README.md
    └── cover.svg
`+"```"+`

## Common Commands

- `+"`mdpress build`"+` - Build the book (PDF by default)
- `+"`mdpress build --format html`"+` - Build the HTML version
- `+"`mdpress serve`"+` - Start the local preview server
- `+"`mdpress validate`"+` - Validate the project configuration
`, projectName, projectName)
}

// generateQuickstartPreface returns the sample preface content.
func generateQuickstartPreface() string {
	return `# Preface

Welcome to this sample book created with mdpress.

mdpress is a Markdown publishing tool that turns Markdown source files into high-quality PDF and HTML output.

## About This Book

This quickstart template demonstrates the core capabilities of mdpress:

- **Markdown authoring**: Write content with a simple, readable format
- **Multiple output formats**: Generate PDF, HTML, and ePub
- **Code highlighting**: Built-in syntax highlighting for code blocks
- **Automatic TOC**: Generate navigation from your headings
- **Theme customization**: Choose built-in themes or add custom styles

> Replace these sample files with your own content, then run ` + "`mdpress build`" + ` to generate your book.
`
}

// generateQuickstartChapter01 returns chapter 1 sample content.
func generateQuickstartChapter01() string {
	return `# Getting Started

This chapter introduces the basic mdpress workflow.

## Install mdpress

Install the latest version:

` + "```bash" + `
go install github.com/yeasy/mdpress@latest
` + "```" + `

## Create a Project

Create a new project with the quickstart command:

` + "```bash" + `
mdpress quickstart my-awesome-book
cd my-awesome-book
` + "```" + `

## Write Content

Each chapter is backed by a Markdown file. You can use standard Markdown syntax:

### Text Formatting

- **Bold text** uses double asterisks
- *Italic text* uses single asterisks
- ` + "`Inline code`" + ` uses backticks

### Code Blocks

Code fences support syntax highlighting for many languages:

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, mdpress!")
}
` + "```" + `

` + "```python" + `
def hello():
    print("Hello, mdpress!")

if __name__ == "__main__":
    hello()
` + "```" + `

## Build Output

After writing content, run the build commands:

` + "```bash" + `
# Build PDF
mdpress build

# Build HTML
mdpress build --format html

# Build multiple formats
mdpress build --format pdf,html,epub
` + "```" + `
`
}

// generateQuickstartChapter02 returns chapter 2 sample content.
func generateQuickstartChapter02() string {
	return `# Advanced Usage

This chapter covers a few advanced mdpress features.

## Understand the Config File

book.yaml is the central project config file and controls how your book is built.

### Book Metadata

` + "```yaml" + `
book:
  title: "Book Title"
  subtitle: "Subtitle"
  author: "Author"
  version: "1.0.0"
  language: "en-US"
` + "```" + `

### Theme Settings

mdpress includes several built-in themes:

| Theme | Style | Best for |
|------|------|----------|
| technical | Clean and professional | Technical writing and developer books |
| elegant | Refined and expressive | Essays and literary work |
| minimal | Sparse and direct | Notes and manuals |

### Custom Styles

Point ` + "`custom_css`" + ` to a CSS file for additional styling:

` + "```yaml" + `
style:
  custom_css: "custom.css"
` + "```" + `

## Live Preview

Start the preview server with:

` + "```bash" + `
mdpress serve
` + "```" + `

When Markdown files change, the browser refreshes automatically.

## Build From GitHub

mdpress can build directly from a GitHub repository:

` + "```bash" + `
mdpress build https://github.com/yeasy/agentic_ai_guide
mdpress build github.com/yeasy/agentic_ai_guide --branch main
` + "```" + `
`
}

// generateQuickstartChapter03 returns chapter 3 sample content.
func generateQuickstartChapter03() string {
	return `# Best Practices

This chapter summarizes a few practical guidelines for mdpress projects.

## Project Organization

Recommended project layout:

` + "```" + `
my-book/
├── book.yaml          # Required config file
├── README.md          # Project overview
├── preface.md         # Preface
├── chapter01/         # One directory per chapter
│   ├── README.md      # Chapter entry file
│   └── section01.md   # Chapter subsection
├── chapter02/
│   └── README.md
├── images/            # Centralized image assets
│   ├── diagram01.png
│   └── screenshot.jpg
├── GLOSSARY.md        # Optional glossary
└── SUMMARY.md         # Optional TOC definition (GitBook compatible)
` + "```" + `

## Writing Tips

1. **Use one directory per chapter** to keep large books organized.
2. **Centralize images** under ` + "`images/`" + ` so asset paths stay predictable.
3. **Use meaningful file names** such as ` + "`setup.md`" + ` and ` + "`deployment.md`" + `.
4. **Outline first, fill in later** by drafting headings before writing details.

## Validation

Before building, validate the project:

` + "```bash" + `
mdpress validate
` + "```" + `

The validator checks:

- Config syntax
- Referenced Markdown files
- Image paths
- Chapter definitions

## Version Control

Use Git to track your book project:

` + "```bash" + `
git init
git add .
git commit -m "Initialize book project"
` + "```" + `

> Tip: add ` + "`_book/`" + ` and ` + "`*.pdf`" + ` to ` + "`.gitignore`" + ` so build artifacts are not committed.
`
}

// generateImagePlaceholder returns the images directory README.
func generateImagePlaceholder() string {
	return `# Image Assets

Store book images in this directory.

## Supported Formats

- PNG (.png)
- JPEG (.jpg, .jpeg)
- SVG (.svg)
- GIF (.gif)

## Usage

Reference images in Markdown like this:

` + "```markdown" + `
![Example image](images/example.png)
` + "```" + `

## Cover Image

Name the cover image ` + "`cover.png`" + `, ` + "`cover.jpg`" + `, or ` + "`cover.svg`" + `. mdpress will detect and use it automatically.
`
}

// generatePlaceholderCoverSVG returns an SVG placeholder cover.
func generatePlaceholderCoverSVG(title string) string {
	// Keep the title short enough to fit the cover.
	displayTitle := title
	runes := []rune(displayTitle)
	if len(runes) > 20 {
		displayTitle = string(runes[:20]) + "..."
	}
	// Escape XML special characters.
	displayTitle = strings.ReplaceAll(displayTitle, "&", "&amp;")
	displayTitle = strings.ReplaceAll(displayTitle, "<", "&lt;")
	displayTitle = strings.ReplaceAll(displayTitle, ">", "&gt;")
	displayTitle = strings.ReplaceAll(displayTitle, "\"", "&quot;")
	displayTitle = strings.ReplaceAll(displayTitle, "'", "&apos;")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="600" height="800" viewBox="0 0 600 800">
  <defs>
    <linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" style="stop-color:#1a1a2e;stop-opacity:1" />
      <stop offset="100%%" style="stop-color:#16213e;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="600" height="800" fill="url(#bg)"/>
  <rect x="50" y="50" width="500" height="700" rx="8" fill="none" stroke="#e94560" stroke-width="2" opacity="0.5"/>
  <text x="300" y="350" text-anchor="middle" font-family="sans-serif" font-size="36" font-weight="bold" fill="#ffffff">%s</text>
  <text x="300" y="420" text-anchor="middle" font-family="sans-serif" font-size="18" fill="#a0a0a0">Built with mdpress</text>
  <line x1="200" y1="380" x2="400" y2="380" stroke="#e94560" stroke-width="2"/>
</svg>`, displayTitle)
}
