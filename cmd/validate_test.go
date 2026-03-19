package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// ========== extractImagePaths Tests ==========

func TestExtractImagePaths(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
		wantErr  bool
	}{
		{
			name: "markdown image syntax",
			content: `# Title
![alt text](path/to/image.png)
Some content`,
			expected: []string{"path/to/image.png"},
		},
		{
			name: "multiple markdown images",
			content: `![first](img1.png)
![second](img2.jpg)
![third](./images/img3.gif)`,
			expected: []string{"img1.png", "img2.jpg", "./images/img3.gif"},
		},
		{
			name:     "markdown image with title",
			content:  `![alt](image.png "image title")`,
			expected: []string{"image.png"},
		},
		{
			name:     "html img tag with src",
			content:  `<img src="path/to/image.png">`,
			expected: []string{"path/to/image.png"},
		},
		{
			name:     "html img tag with single quotes",
			content:  `<img src='image.png'>`,
			expected: []string{"image.png"},
		},
		{
			name: "multiple html img tags",
			content: `<img src="img1.png">
<img src="img2.jpg">
<img alt="test" src="img3.gif">`,
			expected: []string{"img1.png", "img2.jpg", "img3.gif"},
		},
		{
			name: "mixed markdown and html images",
			content: `![md](markdown.png)
<img src="html.jpg">
![another](test.gif)`,
			expected: []string{"markdown.png", "html.jpg", "test.gif"},
		},
		{
			name:     "image with whitespace in path",
			content:  `![alt](  path/to/image.png  )`,
			expected: []string{"path/to/image.png"},
		},
		{
			name: "no images",
			content: `# Just text
No images here`,
			expected: []string{},
		},
		{
			name:     "html img with multiple attributes",
			content:  `<img alt="test" width="100" src="image.png" height="200">`,
			expected: []string{"image.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := extractImagePaths(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractImagePaths() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !stringSlicesEqual(got, tt.expected) {
				t.Errorf("extractImagePaths() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractImagePaths_FileNotFound(t *testing.T) {
	_, err := extractImagePaths("/nonexistent/file.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ========== normalizeMarkdownLinkTarget Tests ==========

func TestNormalizeMarkdownLinkTarget(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple markdown file",
			input:    "chapter.md",
			expected: "chapter.md",
		},
		{
			name:     "markdown with path",
			input:    "chapters/chapter1.md",
			expected: "chapters/chapter1.md",
		},
		{
			name:     "with fragment identifier",
			input:    "chapter.md#section1",
			expected: "chapter.md",
		},
		{
			name:     "whitespace trimming",
			input:    "  chapter.md  ",
			expected: "chapter.md",
		},
		{
			name:     "whitespace with fragment",
			input:    "  chapter.md#section  ",
			expected: "chapter.md",
		},
		{
			name:     "http url",
			input:    "http://example.com/file.md",
			expected: "",
		},
		{
			name:     "https url",
			input:    "https://example.com/file.md",
			expected: "",
		},
		{
			name:     "mailto link",
			input:    "mailto:test@example.com",
			expected: "",
		},
		{
			name:     "tel link",
			input:    "tel:+1234567890",
			expected: "",
		},
		{
			name:     "javascript link",
			input:    "javascript:alert('test')",
			expected: "",
		},
		{
			name:     "data uri",
			input:    "data:text/plain,hello",
			expected: "",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path.md",
			expected: "",
		},
		{
			name:     "protocol relative url",
			input:    "//example.com/file.md",
			expected: "",
		},
		{
			name:     "non-markdown file",
			input:    "document.txt",
			expected: "",
		},
		{
			name:     "html file",
			input:    "page.html",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only fragment",
			input:    "#section",
			expected: "",
		},
		{
			name:     "case insensitive markdown",
			input:    "Chapter.MD",
			expected: "Chapter.MD",
		},
		{
			name:     "relative with dots",
			input:    "../chapters/intro.md",
			expected: "../chapters/intro.md",
		},
		{
			name:     "url in fragment only",
			input:    "file.md#https://example.com",
			expected: "file.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeMarkdownLinkTarget(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeMarkdownLinkTarget(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ========== extractMarkdownLinks Tests ==========

func TestExtractMarkdownLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
		wantErr  bool
	}{
		{
			name: "single markdown link",
			content: `[Chapter 1](ch1.md)
Some text`,
			expected: []string{"ch1.md"},
		},
		{
			name: "multiple markdown links",
			content: `[Link 1](chapter1.md)
[Link 2](chapter2.md)
[Link 3](intro.md)`,
			expected: []string{"chapter1.md", "chapter2.md", "intro.md"},
		},
		{
			name:     "markdown link with fragment",
			content:  `[See section](chapter.md#section1)`,
			expected: []string{"chapter.md"},
		},
		{
			name:     "markdown link with path",
			content:  `[Chapter](chapters/main.md)`,
			expected: []string{"chapters/main.md"},
		},
		{
			name:     "html anchor link",
			content:  `<a href="chapter.md">Chapter 1</a>`,
			expected: []string{"chapter.md"},
		},
		{
			name:     "html anchor with fragment",
			content:  `<a href="chapter.md#section">Link</a>`,
			expected: []string{"chapter.md"},
		},
		{
			name: "mixed markdown and html links",
			content: `[Markdown](md.md)
<a href="html.md">HTML</a>
[Another](another.md)`,
			expected: []string{"md.md", "html.md", "another.md"},
		},
		{
			name: "image link ignored",
			content: `![Image](image.png)
[Real link](chapter.md)`,
			expected: []string{"chapter.md"},
		},
		{
			name:     "http url filtered",
			content:  `[External](https://example.com/doc.md)`,
			expected: []string{},
		},
		{
			name: "non-markdown link filtered",
			content: `[PDF](document.pdf)
[Markdown](chapter.md)`,
			expected: []string{"chapter.md"},
		},
		{
			name:     "no links",
			content:  `Just plain text`,
			expected: []string{},
		},
		{
			name:     "relative path with dots",
			content:  `[Parent](../intro.md)`,
			expected: []string{"../intro.md"},
		},
		{
			name:     "empty markdown link",
			content:  `[Empty]()`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.md")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := extractMarkdownLinks(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractMarkdownLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !stringSlicesEqual(got, tt.expected) {
				t.Errorf("extractMarkdownLinks() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractMarkdownLinks_FileNotFound(t *testing.T) {
	_, err := extractMarkdownLinks("/nonexistent/file.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// ========== validateChapterSequence Tests ==========

func TestValidateChapterSequence(t *testing.T) {
	tests := []struct {
		name       string
		chapters   []config.ChapterDef
		wantIssues bool
	}{
		{
			name: "valid decimal sequence",
			chapters: []config.ChapterDef{
				{Title: "1. First", File: "ch1.md"},
				{Title: "2. Second", File: "ch2.md"},
				{Title: "3. Third", File: "ch3.md"},
			},
			wantIssues: false,
		},
		{
			name: "valid english chapter sequence",
			chapters: []config.ChapterDef{
				{Title: "Chapter 1: Intro", File: "ch1.md"},
				{Title: "Chapter 2: Middle", File: "ch2.md"},
			},
			wantIssues: false,
		},
		{
			name: "sequence gap",
			chapters: []config.ChapterDef{
				{Title: "1. First", File: "ch1.md"},
				{Title: "3. Skip", File: "ch3.md"},
			},
			wantIssues: true,
		},
		{
			name: "no numbering",
			chapters: []config.ChapterDef{
				{Title: "Introduction", File: "intro.md"},
				{Title: "Main Content", File: "main.md"},
			},
			wantIssues: false,
		},
		{
			name: "nested chapters valid",
			chapters: []config.ChapterDef{
				{
					Title: "1. Part One",
					File:  "part1.md",
					Sections: []config.ChapterDef{
						{Title: "1.1. Section A", File: "sec1a.md"},
						{Title: "1.2. Section B", File: "sec1b.md"},
					},
				},
				{
					Title: "2. Part Two",
					File:  "part2.md",
					Sections: []config.ChapterDef{
						{Title: "2.1. Section C", File: "sec2c.md"},
					},
				},
			},
			wantIssues: false,
		},
		{
			name: "nested chapters with gap",
			chapters: []config.ChapterDef{
				{
					Title: "1. Part One",
					File:  "part1.md",
					Sections: []config.ChapterDef{
						{Title: "1.1. Section A", File: "sec1a.md"},
						{Title: "1.3. Skip B", File: "sec1b.md"},
					},
				},
			},
			wantIssues: true,
		},
		{
			name: "mixed numbering styles",
			chapters: []config.ChapterDef{
				{Title: "1. First", File: "ch1.md"},
				{Title: "Chapter 2", File: "ch2.md"},
			},
			wantIssues: false,
		},
		{
			name:       "empty chapters",
			chapters:   []config.ChapterDef{},
			wantIssues: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateChapterSequence(tt.chapters)
			hasIssues := len(issues) > 0
			if hasIssues != tt.wantIssues {
				t.Errorf("validateChapterSequence() hasIssues = %v, want %v. Issues: %v", hasIssues, tt.wantIssues, issues)
			}
		})
	}
}

// ========== parseSequenceParts Tests ==========

func TestParseSequenceParts(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		wantParts []int
		wantOk    bool
	}{
		{
			name:      "simple decimal",
			title:     "1. Title",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "decimal with colon",
			title:     "1: Title",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "decimal with dot separator",
			title:     "1. Title",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "nested decimal",
			title:     "1.2.3 Title",
			wantParts: []int{1, 2, 3},
			wantOk:    true,
		},
		{
			name:      "nested decimal with colon",
			title:     "1.2.3: Title",
			wantParts: []int{1, 2, 3},
			wantOk:    true,
		},
		{
			name:      "chapter prefix",
			title:     "Chapter 1: Intro",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "chapter case insensitive",
			title:     "CHAPTER 2 Something",
			wantParts: []int{2},
			wantOk:    true,
		},
		{
			name:      "chapter with nested numbers",
			title:     "Chapter 2.1: Sub",
			wantParts: []int{2, 1},
			wantOk:    true,
		},
		{
			name:      "no numbering",
			title:     "Plain Title",
			wantParts: nil,
			wantOk:    false,
		},
		{
			name:      "empty title",
			title:     "",
			wantParts: nil,
			wantOk:    false,
		},
		{
			name:      "leading whitespace",
			title:     "  1. Title",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "chinese ordinal",
			title:     "第一章 Introduction",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "chinese ordinal two",
			title:     "第二 Chapter",
			wantParts: []int{2},
			wantOk:    true,
		},
		{
			name:      "chinese ordinal digit",
			title:     "第1章 Title",
			wantParts: []int{1},
			wantOk:    true,
		},
		{
			name:      "chinese without section type",
			title:     "第三",
			wantParts: []int{3},
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, ok := parseSequenceParts(tt.title)
			if ok != tt.wantOk {
				t.Errorf("parseSequenceParts(%q): ok = %v, want %v", tt.title, ok, tt.wantOk)
				return
			}
			if !intSlicesEqual(parts, tt.wantParts) {
				t.Errorf("parseSequenceParts(%q): parts = %v, want %v", tt.title, parts, tt.wantParts)
			}
		})
	}
}

// ========== equalIntSlices Tests ==========

func TestEqualIntSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []int
		b        []int
		expected bool
	}{
		{
			name:     "both empty",
			a:        []int{},
			b:        []int{},
			expected: true,
		},
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "nil and empty",
			a:        nil,
			b:        []int{},
			expected: false,
		},
		{
			name:     "single element same",
			a:        []int{1},
			b:        []int{1},
			expected: true,
		},
		{
			name:     "single element different",
			a:        []int{1},
			b:        []int{2},
			expected: false,
		},
		{
			name:     "multiple elements same",
			a:        []int{1, 2, 3},
			b:        []int{1, 2, 3},
			expected: true,
		},
		{
			name:     "multiple elements different",
			a:        []int{1, 2, 3},
			b:        []int{1, 2, 4},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []int{1, 2},
			b:        []int{1, 2, 3},
			expected: false,
		},
		{
			name:     "different lengths reversed",
			a:        []int{1, 2, 3},
			b:        []int{1, 2},
			expected: false,
		},
		{
			name:     "large slices",
			a:        []int{1, 2, 3, 4, 5, 10, 20, 30},
			b:        []int{1, 2, 3, 4, 5, 10, 20, 30},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalIntSlices(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("equalIntSlices(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// ========== formatSequenceParts Tests ==========

func TestFormatSequenceParts(t *testing.T) {
	tests := []struct {
		name     string
		parts    []int
		expected string
	}{
		{
			name:     "single number",
			parts:    []int{1},
			expected: "1",
		},
		{
			name:     "two numbers",
			parts:    []int{1, 2},
			expected: "1.2",
		},
		{
			name:     "three numbers",
			parts:    []int{1, 2, 3},
			expected: "1.2.3",
		},
		{
			name:     "large numbers",
			parts:    []int{10, 20, 30},
			expected: "10.20.30",
		},
		{
			name:     "empty slice",
			parts:    []int{},
			expected: "",
		},
		{
			name:     "nil slice",
			parts:    nil,
			expected: "",
		},
		{
			name:     "many parts",
			parts:    []int{1, 2, 3, 4, 5},
			expected: "1.2.3.4.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSequenceParts(tt.parts)
			if got != tt.expected {
				t.Errorf("formatSequenceParts(%v) = %q, want %q", tt.parts, got, tt.expected)
			}
		})
	}
}

// ========== summarizeValidationResults Tests ==========

func TestSummarizeValidationResults(t *testing.T) {
	tests := []struct {
		name       string
		results    []validateResult
		wantPassed int
		wantFailed int
	}{
		{
			name:       "all passed",
			results:    []validateResult{{ok: true}, {ok: true}, {ok: true}},
			wantPassed: 3,
			wantFailed: 0,
		},
		{
			name:       "all failed",
			results:    []validateResult{{ok: false}, {ok: false}},
			wantPassed: 0,
			wantFailed: 2,
		},
		{
			name:       "mixed",
			results:    []validateResult{{ok: true}, {ok: false}, {ok: true}, {ok: false}},
			wantPassed: 2,
			wantFailed: 2,
		},
		{
			name:       "empty",
			results:    []validateResult{},
			wantPassed: 0,
			wantFailed: 0,
		},
		{
			name:       "single passed",
			results:    []validateResult{{ok: true}},
			wantPassed: 1,
			wantFailed: 0,
		},
		{
			name:       "single failed",
			results:    []validateResult{{ok: false}},
			wantPassed: 0,
			wantFailed: 1,
		},
		{
			name: "many results",
			results: []validateResult{
				{ok: true}, {ok: true}, {ok: false}, {ok: true}, {ok: false},
				{ok: true}, {ok: false}, {ok: true}, {ok: true}, {ok: false},
			},
			wantPassed: 6,
			wantFailed: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, failed := summarizeValidationResults(tt.results)
			if passed != tt.wantPassed || failed != tt.wantFailed {
				t.Errorf("summarizeValidationResults() = (%d, %d), want (%d, %d)",
					passed, failed, tt.wantPassed, tt.wantFailed)
			}
		})
	}
}

// ========== validationStatus Tests ==========

func TestValidationStatus(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		expected string
	}{
		{
			name:     "no error",
			hasError: false,
			expected: "passed",
		},
		{
			name:     "has error",
			hasError: true,
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validationStatus(tt.hasError)
			if got != tt.expected {
				t.Errorf("validationStatus(%v) = %q, want %q", tt.hasError, got, tt.expected)
			}
		})
	}
}

// ========== renderValidationMarkdown Tests ==========

func TestRenderValidationMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		report   validationReport
		mustHave []string
	}{
		{
			name: "all passed",
			report: validationReport{
				Status:      "passed",
				TotalChecks: 5,
				Passed:      5,
				Failed:      0,
				Results: []validateResult{
					{ok: true, message: "Check 1 passed"},
					{ok: true, message: "Check 2 passed"},
				},
			},
			mustHave: []string{
				"# mdpress Validation Report",
				"- Status: passed",
				"- Total checks: 5",
				"- Passed: 5",
				"- Failed: 0",
				"## Results",
				"[PASS] Check 1 passed",
				"[PASS] Check 2 passed",
			},
		},
		{
			name: "with failures",
			report: validationReport{
				Status:      "failed",
				TotalChecks: 3,
				Passed:      2,
				Failed:      1,
				Results: []validateResult{
					{ok: true, message: "Config found"},
					{ok: false, message: "File not found"},
					{ok: true, message: "Valid"},
				},
			},
			mustHave: []string{
				"# mdpress Validation Report",
				"- Status: failed",
				"- Total checks: 3",
				"- Passed: 2",
				"- Failed: 1",
				"[PASS] Config found",
				"[FAIL] File not found",
				"[PASS] Valid",
			},
		},
		{
			name: "no results",
			report: validationReport{
				Status:      "passed",
				TotalChecks: 0,
				Passed:      0,
				Failed:      0,
				Results:     []validateResult{},
			},
			mustHave: []string{
				"# mdpress Validation Report",
				"- Status: passed",
				"- Total checks: 0",
				"## Results",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderValidationMarkdown(tt.report)
			for _, mustHave := range tt.mustHave {
				if !strings.Contains(got, mustHave) {
					t.Errorf("renderValidationMarkdown() missing %q in output:\n%s", mustHave, got)
				}
			}
		})
	}
}

// ========== flattenChapterDefs Tests ==========

func TestFlattenChapterDefs(t *testing.T) {
	tests := []struct {
		name          string
		chapters      []config.ChapterDef
		expectedCount int
		expectedFiles []string
	}{
		{
			name: "single chapter",
			chapters: []config.ChapterDef{
				{Title: "Chapter 1", File: "ch1.md"},
			},
			expectedCount: 1,
			expectedFiles: []string{"ch1.md"},
		},
		{
			name: "multiple chapters",
			chapters: []config.ChapterDef{
				{Title: "Chapter 1", File: "ch1.md"},
				{Title: "Chapter 2", File: "ch2.md"},
				{Title: "Chapter 3", File: "ch3.md"},
			},
			expectedCount: 3,
			expectedFiles: []string{"ch1.md", "ch2.md", "ch3.md"},
		},
		{
			name: "single level nesting",
			chapters: []config.ChapterDef{
				{
					Title: "Part 1",
					File:  "part1.md",
					Sections: []config.ChapterDef{
						{Title: "Section 1.1", File: "sec1a.md"},
						{Title: "Section 1.2", File: "sec1b.md"},
					},
				},
				{
					Title: "Part 2",
					File:  "part2.md",
				},
			},
			expectedCount: 4,
			expectedFiles: []string{"part1.md", "sec1a.md", "sec1b.md", "part2.md"},
		},
		{
			name: "multi-level nesting",
			chapters: []config.ChapterDef{
				{
					Title: "Part 1",
					File:  "part1.md",
					Sections: []config.ChapterDef{
						{
							Title: "Section 1.1",
							File:  "sec1a.md",
							Sections: []config.ChapterDef{
								{Title: "Subsection 1.1.1", File: "subsec1a1.md"},
							},
						},
					},
				},
			},
			expectedCount: 3,
			expectedFiles: []string{"part1.md", "sec1a.md", "subsec1a1.md"},
		},
		{
			name:          "empty",
			chapters:      []config.ChapterDef{},
			expectedCount: 0,
			expectedFiles: []string{},
		},
		{
			name: "sections only no file",
			chapters: []config.ChapterDef{
				{
					Title: "Part 1",
					File:  "part1.md",
					Sections: []config.ChapterDef{
						{Title: "Section A", File: "sec_a.md"},
						{Title: "Section B", File: "sec_b.md"},
						{Title: "Section C", File: "sec_c.md"},
					},
				},
			},
			expectedCount: 4,
			expectedFiles: []string{"part1.md", "sec_a.md", "sec_b.md", "sec_c.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flattenChapterDefs(tt.chapters)
			if len(got) != tt.expectedCount {
				t.Errorf("flattenChapterDefs(): got %d chapters, want %d", len(got), tt.expectedCount)
			}

			for i, ch := range got {
				if i >= len(tt.expectedFiles) {
					t.Errorf("flattenChapterDefs(): got more chapters than expected")
					break
				}
				if ch.File != tt.expectedFiles[i] {
					t.Errorf("flattenChapterDefs() chapter %d: got file %q, want %q",
						i, ch.File, tt.expectedFiles[i])
				}
			}
		})
	}
}

// ========== Helper Functions ==========

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
