package markdown

// ProcessingResult holds the result of processing a single Markdown document.
type ProcessingResult struct {
	FilePath string
	HTML     string
	Headings []HeadingInfo
	Error    error
}
