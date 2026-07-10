// quickstart.go implements the quickstart subcommand.
// It creates a complete sample book project so users can see results quickly.
// The command generates book.yaml, README.md, sample chapters, an images
// directory, a .gitignore for build artifacts, and next-step instructions.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yeasy/mdpress/pkg/utils"
)

// scaffoldGitignore lists the build artifacts that default mdpress builds can
// drop into a project (site output in _book/, per-language *_site/ dirs, and
// <name>.pdf/.html/.epub files) so they are not committed by accident.
// Shared by quickstart and init scaffolding.
const scaffoldGitignore = `# mdpress build artifacts
_book/
*_site/
*.pdf
*.html
*.epub
`

// quickstartForce permits scaffolding into a non-empty directory.  Individual
// existing files are still never overwritten.
var quickstartForce bool

// quickstartCmd creates a sample book project.
var quickstartCmd = &cobra.Command{
	Use:   "quickstart [directory]",
	Short: "Create a complete sample book project",
	Long: `Create a ready-to-build sample book project with config files,
multiple chapters, and example content.
You can build and preview it immediately.

Examples:
  mdpress quickstart my-book
  mdpress quickstart my-book --force
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

func init() {
	quickstartCmd.Flags().BoolVar(&quickstartForce, "force", false, "Allow scaffolding into a non-empty directory (never overwrites existing files)")
}

// executeQuickstart creates the sample project.
func executeQuickstart(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}

	// Refuse to write into a non-empty directory unless --force is set.
	// Stat the path first so an existing regular file yields a friendly error
	// rather than a raw "readdir ...: not a directory" from os.ReadDir.
	// Use os.ReadDir instead of filepath.Glob because Glob("*") does not
	// match hidden files/directories (e.g. .git, .DS_Store), which could
	// lead to silently overwriting an existing Git repository.
	if info, statErr := os.Stat(absDir); statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("target %q exists and is a file; choose a different name", dir)
		}
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}
		if len(entries) > 0 && !quickstartForce {
			return fmt.Errorf("directory %s already exists and is not empty; choose a new directory name or pass --force", dir)
		}
	}

	projectName := filepath.Base(absDir)

	// quickstartWriteFile writes a scaffolded file but never overwrites an
	// existing one, so --force can populate a non-empty directory without
	// clobbering user files.
	quickstartWriteFile := func(path string, data []byte) error {
		if utils.FileExists(path) {
			return fmt.Errorf("refusing to overwrite existing file %q", path)
		}
		return utils.WriteFile(path, data)
	}

	utils.Header("mdpress Quickstart")
	utils.Info("Creating sample project: %s", projectName)
	fmt.Println()

	// 1. Create book.yaml.
	bookYAML := generateQuickstartBookYAML(projectName)
	if err := quickstartWriteFile(filepath.Join(absDir, "book.yaml"), []byte(bookYAML)); err != nil {
		return fmt.Errorf("failed to create book.yaml: %w", err)
	}
	utils.Success("book.yaml - book configuration")

	// 2. Create README.md.
	readmeContent := generateQuickstartREADME(projectName)
	if err := quickstartWriteFile(filepath.Join(absDir, "README.md"), []byte(readmeContent)); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}
	utils.Success("README.md - project overview")

	// 3. Create the preface.
	prefaceContent := generateQuickstartPreface()
	if err := quickstartWriteFile(filepath.Join(absDir, "preface.md"), []byte(prefaceContent)); err != nil {
		return fmt.Errorf("failed to create preface.md: %w", err)
	}
	utils.Success("preface.md - preface")

	// 4. Create chapter 1.
	ch01Content := generateQuickstartChapter01()
	if err := quickstartWriteFile(filepath.Join(absDir, "chapter01", "README.md"), []byte(ch01Content)); err != nil {
		return fmt.Errorf("failed to create chapter01: %w", err)
	}
	utils.Success("chapter01/README.md - Chapter 1: Getting Started")

	// 5. Create chapter 2.
	ch02Content := generateQuickstartChapter02()
	if err := quickstartWriteFile(filepath.Join(absDir, "chapter02", "README.md"), []byte(ch02Content)); err != nil {
		return fmt.Errorf("failed to create chapter02: %w", err)
	}
	utils.Success("chapter02/README.md - Chapter 2: Advanced Usage")

	// 6. Create chapter 3.
	ch03Content := generateQuickstartChapter03()
	if err := quickstartWriteFile(filepath.Join(absDir, "chapter03", "README.md"), []byte(ch03Content)); err != nil {
		return fmt.Errorf("failed to create chapter03: %w", err)
	}
	utils.Success("chapter03/README.md - Chapter 3: Best Practices")

	// 7. Create the image directory and placeholder notes.
	imgPlaceholder := generateImagePlaceholder()
	if err := quickstartWriteFile(filepath.Join(absDir, "images", "README.md"), []byte(imgPlaceholder)); err != nil {
		return fmt.Errorf("failed to create images directory notes: %w", err)
	}
	utils.Success("images/ - image assets directory")

	// 8. Create .gitignore so default build artifacts are not committed.
	// Skip silently when one already exists (e.g. --force into a Git repo).
	gitignorePath := filepath.Join(absDir, ".gitignore")
	if !utils.FileExists(gitignorePath) {
		if err := utils.WriteFile(gitignorePath, []byte(scaffoldGitignore)); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
		utils.Success(".gitignore - ignores build artifacts")
	}

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
  # A styled cover is generated from the metadata above by default.
  # cover:
  #   image: "images/cover.png"  # use a custom cover image
  #   background: "#102a43"      # override the default cover background color

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
  theme: "technical"   # technical | elegant | minimal
  page_size: "A4"
  # custom_css: "custom.css"

output:
  filename: "%s.pdf"
  toc: true
  cover: true
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
├── .gitignore         # Ignores build artifacts
├── README.md          # Project overview
├── preface.md         # Preface
├── chapter01/         # Chapter 1
│   └── README.md
├── chapter02/         # Chapter 2
│   └── README.md
├── chapter03/         # Chapter 3
│   └── README.md
└── images/            # Image assets
    └── README.md
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

mdpress is a Markdown publishing tool that turns Markdown source files into a polished website, PDF, standalone HTML, and EPUB output.

## About This Book

This quickstart template demonstrates the core capabilities of mdpress:

- **Markdown authoring**: Write content with a simple, readable format
- **Multiple output formats**: Generate a website, PDF, HTML, and EPUB
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

> Tip: the generated ` + "`.gitignore`" + ` already excludes build artifacts (` + "`_book/`" + `, ` + "`*_site/`" + `, ` + "`*.pdf`" + `, ` + "`*.html`" + `, ` + "`*.epub`" + `) so they are not committed.
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

mdpress generates a styled cover from your book metadata by default. To use a custom image instead, point ` + "`book.cover.image`" + ` in book.yaml at a file in this directory:

` + "```yaml" + `
book:
  cover:
    image: "images/cover.png"
` + "```" + `
`
}
