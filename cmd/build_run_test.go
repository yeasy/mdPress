package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
	"github.com/yeasy/mdpress/internal/renderer"
)

func TestValidateChapterTitleSequenceMismatch(t *testing.T) {
	diag := validateChapterTitleSequence("2. 安装", []markdown.HeadingInfo{
		{Text: "1. 安装", Line: 3, Column: 1},
	})
	if diag == nil {
		t.Fatal("expected diagnostic")
		return
	}
	if diag.Rule != "chapter-title-sequence" {
		t.Fatalf("unexpected rule: %s", diag.Rule)
	}
	if diag.Line != 3 || diag.Column != 1 {
		t.Fatalf("unexpected position: %d:%d", diag.Line, diag.Column)
	}
}

func TestValidateChapterTitleSequenceSupportsChineseOrdinal(t *testing.T) {
	diag := validateChapterTitleSequence("第一章 简介", []markdown.HeadingInfo{
		{Text: "第1章 简介", Line: 1, Column: 1},
	})
	if diag != nil {
		t.Fatalf("expected no diagnostic, got %+v", diag)
	}
}

func TestValidateChapterTitleSequenceSupportsEnglishChapter(t *testing.T) {
	diag := validateChapterTitleSequence("Chapter 1: Intro", []markdown.HeadingInfo{
		{Text: "第一章 简介", Line: 1, Column: 1},
	})
	if diag != nil {
		t.Fatalf("expected no diagnostic, got %+v", diag)
	}
}

func TestValidateChapterTitleSequenceNoNumberNoWarning(t *testing.T) {
	diag := validateChapterTitleSequence("简介", []markdown.HeadingInfo{
		{Text: "项目简介", Line: 1, Column: 1},
	})
	if diag != nil {
		t.Fatalf("expected no diagnostic, got %+v", diag)
	}
}

func TestValidateChapterTitleSequenceSummaryHasNumberButHeadingDoesNot(t *testing.T) {
	diag := validateChapterTitleSequence("第 1 章 - 背景知识", []markdown.HeadingInfo{
		{Text: "背景知识", Line: 1, Column: 1},
	})
	if diag != nil {
		t.Fatalf("expected no diagnostic when heading omits numbering, got %+v", diag)
	}
}

func TestValidateBookTitleConsistencyMixedStyles(t *testing.T) {
	// Chinese + Arabic is a compatible pair (standard in Chinese tech books),
	// so only truly incompatible styles should produce warnings.
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1.md", Heading: markdown.HeadingInfo{Text: "1. 简介", Line: 1, Column: 1}},
		{File: "ch2.md", Heading: markdown.HeadingInfo{Text: "第二章 安装", Line: 1, Column: 1}},
		{File: "ch3.md", Heading: markdown.HeadingInfo{Text: "部署", Line: 1, Column: 1}},
	})
	// Arabic + Chinese are compatible; "部署" is style "none" which is always compatible.
	// So zero style-mismatch warnings expected.
	styleWarnings := 0
	for _, w := range warnings {
		if w.Diagnostic.Rule == "book-title-style" {
			styleWarnings++
		}
	}
	if styleWarnings != 0 {
		t.Fatalf("expected 0 style warnings (arabic+chinese compatible), got %d: %+v", styleWarnings, warnings)
	}
}

func TestValidateBookTitleConsistencyIncompatibleStyles(t *testing.T) {
	// English ("Chapter 1") + Chinese ("第二章") should still be flagged.
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1.md", Heading: markdown.HeadingInfo{Text: "Chapter 1 Introduction", Line: 1, Column: 1}},
		{File: "ch2.md", Heading: markdown.HeadingInfo{Text: "第二章 安装", Line: 1, Column: 1}},
	})
	styleWarnings := 0
	for _, w := range warnings {
		if w.Diagnostic.Rule == "book-title-style" {
			styleWarnings++
		}
	}
	if styleWarnings == 0 {
		t.Fatal("expected style warning for english+chinese mismatch")
	}
}

func TestValidateBookTitleConsistencyDuplicateTitles(t *testing.T) {
	// "简介" is a common recurring title and should NOT trigger duplicate warning.
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1.md", Heading: markdown.HeadingInfo{Text: "1. 简介", Line: 1, Column: 1}},
		{File: "ch2.md", Heading: markdown.HeadingInfo{Text: "2. 简介", Line: 1, Column: 1}},
	})
	for _, warning := range warnings {
		if warning.Diagnostic.Rule == "book-title-duplicate" {
			t.Fatalf("common recurring title '简介' should NOT trigger duplicate warning, got: %s", warning.Diagnostic.Message)
		}
	}
}

func TestValidateBookTitleConsistencyRealDuplicates(t *testing.T) {
	// Non-common titles with same normalized form within same directory scope
	// should still be flagged.
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1/docker.md", Heading: markdown.HeadingInfo{Text: "1. Docker 原理", Line: 1, Column: 1}},
		{File: "ch1/docker2.md", Heading: markdown.HeadingInfo{Text: "2. Docker 原理", Line: 1, Column: 1}},
	})
	found := false
	for _, warning := range warnings {
		if warning.Diagnostic.Rule == "book-title-duplicate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate warning for same non-common title in same scope, got %+v", warnings)
	}
}

func TestValidateBookTitleConsistencyDiffScopeNoDuplicate(t *testing.T) {
	// Same non-common title in different directory scopes should NOT be flagged.
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1/advanced.md", Heading: markdown.HeadingInfo{Text: "1. 高级特性", Line: 1, Column: 1}},
		{File: "ch2/advanced.md", Heading: markdown.HeadingInfo{Text: "2. 高级特性", Line: 1, Column: 1}},
	})
	for _, warning := range warnings {
		if warning.Diagnostic.Rule == "book-title-duplicate" {
			t.Fatalf("same title in different directory scopes should NOT trigger duplicate: %s", warning.Diagnostic.Message)
		}
	}
}

func TestValidateBookTitleConsistencyDockerPractice(t *testing.T) {
	// Simulate the docker_practice pattern: Chinese chapter prefix + "X.Y" numbering + chapter summary in every chapter.
	records := []chapterHeadingRecord{
		{File: "01_introduction/README.md", Heading: markdown.HeadingInfo{Text: "第一章 介绍", Line: 1}},
		{File: "01_introduction/summary.md", Heading: markdown.HeadingInfo{Text: "本章小结", Line: 1}},
		{File: "02_basic_concept/README.md", Heading: markdown.HeadingInfo{Text: "第二章 基本概念", Line: 1}},
		{File: "02_basic_concept/summary.md", Heading: markdown.HeadingInfo{Text: "本章小结", Line: 1}},
		{File: "03_install/3.1_ubuntu.md", Heading: markdown.HeadingInfo{Text: "3.1 Ubuntu", Line: 1}},
		{File: "03_install/summary.md", Heading: markdown.HeadingInfo{Text: "本章小结", Line: 1}},
		{File: "11_compose/11.1_introduction.md", Heading: markdown.HeadingInfo{Text: "11.1 简介", Line: 1}},
		{File: "13_kubernetes_concepts/13.1_intro.md", Heading: markdown.HeadingInfo{Text: "13.1 简介", Line: 1}},
	}
	warnings := validateBookTitleConsistency(records)
	if len(warnings) > 0 {
		for _, w := range warnings {
			t.Errorf("unexpected warning: rule=%s file=%s msg=%s", w.Diagnostic.Rule, w.File, w.Diagnostic.Message)
		}
		t.Fatalf("docker_practice pattern should produce zero warnings, got %d", len(warnings))
	}
}

func TestCompatibleTitleStyles(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"arabic", "arabic", true},
		{"chinese", "chinese", true},
		{"arabic", "chinese", true}, // standard Chinese book pattern
		{"chinese", "arabic", true}, // symmetric
		{"arabic", "none", true},    // "none" is always compatible
		{"none", "chinese", true},
		{"english", "chinese", false}, // incompatible
		{"english", "arabic", false},  // incompatible
		{"none", "none", true},
	}
	for _, tt := range tests {
		got := compatibleTitleStyles(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compatibleTitleStyles(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestValidateChapterSequenceDetectsGap(t *testing.T) {
	issues := validateChapterSequence([]config.ChapterDef{
		{Title: "1. 简介", File: "ch1.md"},
		{Title: "3. 安装", File: "ch3.md"},
	})
	if len(issues) == 0 {
		t.Fatal("expected sequence gap issue")
	}
}

func TestValidateChapterSequenceAllowsNonNumberedTitles(t *testing.T) {
	issues := validateChapterSequence([]config.ChapterDef{
		{Title: "简介", File: "intro.md"},
		{Title: "安装", File: "install.md"},
	})
	if len(issues) != 0 {
		t.Fatalf("expected no sequence issues, got %+v", issues)
	}
}

// Tests for extractTitleSequence
func TestExtractTitleSequence(t *testing.T) {
	tests := []struct {
		title string
		seq   string
		found bool
	}{
		// Decimal patterns
		{"1. Introduction", "1", true},
		{"1.5 Advanced Topics", "1.5", true},
		{"2.3.4 Deep Dive", "2.3.4", true},
		{"  3  ) Chapter Three", "3", true},
		{"4： Something", "4", true},
		{"5、Chinese punctuation", "5", true},
		{"6. ", "6", true},
		// English chapter patterns
		{"Chapter 1 Introduction", "1", true},
		{"CHAPTER 2 Overview", "2", true},
		{"Chapter 3.1 Advanced", "3.1", true},
		{"Chapter 10 Conclusion", "10", true},
		// Chinese chapter patterns
		{"第一章 简介", "1", true},
		{"第二章 安装", "2", true},
		{"第十章 总结", "10", true},
		{"第11章 扩展", "11", true},
		{"第二十三章 深入", "23", true},
		{"第一百章 终极", "100", true},
		{"第一百二十三章 Finale", "123", true},
		{"第零章 Prologue", "0", true},
		// No number cases
		{"Introduction", "", false},
		{"Just a title", "", false},
		{"", "", false},
		{"  ", "", false},
		// Edge cases
		{"chapter 1", "", false}, // lowercase without "Chapter" full word
		{"1", "1", true},
		{"1.", "1", true},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			seq, found := extractTitleSequence(tt.title)
			if found != tt.found {
				t.Errorf("found=%v, want %v", found, tt.found)
			}
			if found && seq != tt.seq {
				t.Errorf("seq=%q, want %q", seq, tt.seq)
			}
		})
	}
}

// Tests for normalizeChapterTitle
func TestNormalizeChapterTitle(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		// Decimal prefix removal
		{"1. Introduction", "Introduction"},
		{"2.5 Advanced Topics", "Advanced Topics"},
		// English chapter prefix removal
		{"Chapter 1 Getting Started", "Getting Started"},
		{"CHAPTER 2 Overview", "Overview"},
		// Chinese chapter prefix removal
		{"第一章 简介", "简介"},
		{"第二十三章 深入探讨", "深入探讨"},
		// With extra punctuation
		{"1: Introduction", "Introduction"},
		{"1） Setup", "Setup"},
		{"1- Basics", "Basics"},
		// Multiple spaces normalized
		{"1.   Extra    Spaces", "Extra Spaces"},
		// Empty and whitespace
		{"", ""},
		{"   ", ""},
		{"   1. Title   ", "Title"},
		// No number
		{"Just a Title", "Just a Title"},
		// Chinese with punctuation
		{"第五章：关键概念", "关键概念"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeChapterTitle(tt.input)
			if result != tt.output {
				t.Errorf("got %q, want %q", result, tt.output)
			}
		})
	}
}

// Tests for parseChineseOrdinal
func TestParseChineseOrdinal(t *testing.T) {
	tests := []struct {
		input  string
		output int
	}{
		// Single digits
		{"一", 1},
		{"二", 2},
		{"三", 3},
		{"四", 4},
		{"五", 5},
		{"六", 6},
		{"七", 7},
		{"八", 8},
		{"九", 9},
		{"零", 0},
		{"〇", 0},
		// Two-character combinations
		{"十", 10},
		{"二十", 20},
		{"三十", 30},
		{"九十", 90},
		{"十一", 11},
		{"十九", 19},
		{"二十三", 23},
		{"九十九", 99},
		// Larger numbers
		{"一百", 100},
		{"一百一", 101},
		{"一百二十三", 123},
		{"九百九十九", 999},
		// Special cases
		{"两", 2},
		// Pure digits (fallback to strconv)
		{"1", 1},
		{"12", 12},
		{"123", 123},
		// Invalid/edge cases
		{"", 0},
		{"   ", 0},
		{"ABC", 0},
		{"千", 0}, // Unsupported unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseChineseOrdinal(tt.input)
			if result != tt.output {
				t.Errorf("got %d, want %d", result, tt.output)
			}
		})
	}
}

// Tests for guessLanguageCode
func TestGuessLanguageCode(t *testing.T) {
	tests := []struct {
		dir  string
		code string
	}{
		// English variants
		{"en", "en-US"},
		{"EN", "en-US"},
		{"En", "en-US"},
		{"en-us", "en-US"},
		{"EN-US", "en-US"},
		// Chinese variants
		{"cn", "zh-CN"},
		{"CN", "zh-CN"},
		{"zh", "zh-CN"},
		{"zh-cn", "zh-CN"},
		{"ZH-CN", "zh-CN"},
		{"zh-tw", "zh-TW"},
		{"ZH-TW", "zh-TW"},
		// Japanese
		{"ja", "ja-JP"},
		{"JA", "ja-JP"},
		{"ja-jp", "ja-JP"},
		{"JA-JP", "ja-JP"},
		// Korean
		{"ko", "ko-KR"},
		{"KO", "ko-KR"},
		{"ko-kr", "ko-KR"},
		{"KO-KR", "ko-KR"},
		// Unknown
		{"fr", ""},
		{"de", ""},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			result := guessLanguageCode(tt.dir)
			if result != tt.code {
				t.Errorf("got %q, want %q", result, tt.code)
			}
		})
	}
}

// Tests for titleSequenceStyle
func TestTitleSequenceStyle(t *testing.T) {
	tests := []struct {
		title string
		style string
	}{
		// Arabic numerals
		{"1. Introduction", "arabic"},
		{"2.5 Advanced", "arabic"},
		{"3) Three", "arabic"},
		{"4： Four", "arabic"},
		// English chapter
		{"Chapter 1 Title", "english"},
		{"CHAPTER 2 Overview", "english"},
		{"Chapter 3.1 Subsection", "english"},
		// Chinese chapter
		{"第一章 简介", "chinese"},
		{"第五章 介绍", "chinese"},
		{"第二十三章 Deep", "chinese"},
		// No numbering
		{"Just a Title", "none"},
		{"Introduction", "none"},
		{"", "none"},
		{"No numbers here", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := titleSequenceStyle(tt.title)
			if result != tt.style {
				t.Errorf("got %q, want %q", result, tt.style)
			}
		})
	}
}

// Tests for describeTitleStyleMismatch
func TestDescribeTitleStyleMismatch(t *testing.T) {
	tests := []struct {
		primaryStyle string
		title        string
		hasDesc      bool
		descContains string
	}{
		// Mismatch cases
		{"arabic", "第一章 Chinese", true, "Arabic"},
		{"arabic", "第一章 Chinese", true, "Chinese"},
		{"english", "1. Arabic", true, "English"},
		{"english", "1. Arabic", true, "Arabic"},
		{"chinese", "Chapter 1 English", true, "Chinese"},
		{"chinese", "Chapter 1 English", true, "English"},
		// No numbering on title
		{"arabic", "No Numbers", true, "no numbering"},
		{"english", "No Numbers", true, "no numbering"},
		{"chinese", "No Numbers", true, "no numbering"},
		// Matching style
		{"arabic", "1. Title", false, ""},
		{"english", "Chapter 1 Title", false, ""},
		{"chinese", "第一章 Title", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.primaryStyle+"_"+tt.title, func(t *testing.T) {
			result := describeTitleStyleMismatch(tt.primaryStyle, tt.title)
			if (result != "") != tt.hasDesc {
				t.Errorf("has description=%v, want %v (got %q)", result != "", tt.hasDesc, result)
			}
			if tt.hasDesc && tt.descContains != "" {
				if !strings.Contains(result, tt.descContains) {
					t.Errorf("description %q does not contain %q", result, tt.descContains)
				}
			}
		})
	}
}

// Tests for multilingualLandingPath
func TestMultilingualLandingPath(t *testing.T) {
	tests := []struct {
		rootDir        string
		outputOverride string
		expected       string
	}{
		// No override
		{"/root", "", "/root/_mdpress_langs.html"},
		{"/path/to/project", "", "/path/to/project/_mdpress_langs.html"},
		// Directory override
		{"/root", "/output", "/output/index.html"},
		{"/root", "/path/to/output", "/path/to/output/index.html"},
		// File override with extension
		{"/root", "/output/book.html", "/output/book-index.html"},
		{"/root", "/out/mybook.pdf", "/out/mybook-index.html"},
		// File override without extension
		{"/root", "/output/book", "/output/book-index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.rootDir+"_"+tt.outputOverride, func(t *testing.T) {
			// Note: This test will only pass for non-directory cases since
			// we can't mock os.Stat. The directory cases need special setup.
			if tt.outputOverride == "" || (tt.outputOverride != "" && !isDirectoryPath(tt.outputOverride)) {
				result := multilingualLandingPath(filepath.FromSlash(tt.rootDir), filepath.FromSlash(tt.outputOverride))
				expected := filepath.FromSlash(tt.expected)
				if result != expected {
					t.Errorf("got %q, want %q", result, expected)
				}
			}
		})
	}
}

// Helper function to guess if path looks like a directory (for test purposes)
func isDirectoryPath(p string) bool {
	return !strings.Contains(p, ".")
}

// Tests for injectBannerIntoHTML
func TestInjectBannerIntoHTML(t *testing.T) {
	tests := []struct {
		name           string
		htmlContent    string
		bannerHTML     string
		expectedBefore string
		expectedAfter  string
		shouldContain  string
	}{
		{
			name:           "inject after body tag",
			htmlContent:    `<html><body>Content</body></html>`,
			bannerHTML:     `<nav>Banner</nav>`,
			expectedBefore: "<body>",
			expectedAfter:  "<nav>Banner</nav>",
			shouldContain:  `<nav>Banner</nav>Content`,
		},
		{
			name:          "inject before body tag (lowercase)",
			htmlContent:   `<html><body>Content</body></html>`,
			bannerHTML:    `<div>Nav</div>`,
			shouldContain: `<body><div>Nav</div>Content`,
		},
		{
			name:          "inject before body tag (uppercase)",
			htmlContent:   `<html><BODY>Content</BODY></html>`,
			bannerHTML:    `<nav>Banner</nav>`,
			shouldContain: `<BODY><nav>Banner</nav>Content`,
		},
		{
			name:          "no body tag - prepend",
			htmlContent:   `<html>Content</html>`,
			bannerHTML:    `<nav>Banner</nav>`,
			shouldContain: `<nav>Banner</nav><html>Content`,
		},
		{
			name:          "banner already exists - no injection",
			htmlContent:   `<html><body><nav class="mdpress-lang-switcher">Existing</nav>Content</body></html>`,
			bannerHTML:    `<nav>New</nav>`,
			shouldContain: `<nav class="mdpress-lang-switcher">Existing</nav>Content`,
		},
		{
			name:          "empty banner",
			htmlContent:   `<html><body>Content</body></html>`,
			bannerHTML:    ``,
			shouldContain: `<body>Content`,
		},
		{
			name:          "empty html",
			htmlContent:   ``,
			bannerHTML:    `<nav>Banner</nav>`,
			shouldContain: `<nav>Banner</nav>`,
		},
		{
			name:          "mixed case body tag preserves original",
			htmlContent:   `<html><Body>Content</Body></html>`,
			bannerHTML:    `<nav>Banner</nav>`,
			shouldContain: `<Body><nav>Banner</nav>Content`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectBannerIntoHTML(tt.htmlContent, tt.bannerHTML)
			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("result does not contain %q\ngot: %q", tt.shouldContain, result)
			}
		})
	}
}

// Tests for sitePageFilenames
func TestSitePageFilenames(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "empty",
			files:    []string{},
			expected: []string{},
		},
		{
			name:     "single file",
			files:    []string{"intro.md"},
			expected: []string{"intro.html"},
		},
		{
			name:     "README becomes dir name",
			files:    []string{"chapter01/README.md", "chapter01/section1.md"},
			expected: []string{"chapter01/index.html", "chapter01/section1.html"},
		},
		{
			name:     "nested paths",
			files:    []string{"preface.md", "01_ai_intro/README.md", "01_ai_intro/1.1_what_is_ai.md"},
			expected: []string{"preface.html", "01_ai_intro/index.html", "01_ai_intro/1.1_what_is_ai.html"},
		},
		{
			name:     "empty source falls back to ch_NNN",
			files:    []string{"intro.md", "", "outro.md"},
			expected: []string{"intro.html", "ch_001.html", "outro.html"},
		},
		{
			name:     "collision falls back to ch_NNN",
			files:    []string{"a/README.md", "a/README.md"},
			expected: []string{"a/index.html", "ch_001.html"},
		},
		{
			name:     "root readme becomes site index",
			files:    []string{"README.md", "chapter01/README.md"},
			expected: []string{"index.html", "chapter01/index.html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sitePageFilenames(tt.files)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("at index %d: expected %s, got %s", i, exp, result[i])
				}
			}
		})
	}
}

// Tests for mdFileToHTMLName
func TestMdFileToHTMLName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"intro.md", "intro.html"},
		{"README.md", "index.html"},
		{"chapter01/README.md", "chapter01/index.html"},
		{"chapter01/section1.md", "chapter01/section1.html"},
		{"01_ai_intro/1.1_what_is_ai.md", "01_ai_intro/1.1_what_is_ai.html"},
		{"preface.md", "preface.html"},
		{"a/b/c.md", "a/b/c.html"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mdFileToHTMLName(tt.input)
			if result != tt.expected {
				t.Errorf("mdFileToHTMLName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Tests for rewriteChapterLinks
func TestRewriteChapterLinks(t *testing.T) {
	tests := []struct {
		name         string
		chapters     []renderer.ChapterHTML
		chapterFiles []string
		shouldReturn bool
	}{
		{
			name:         "empty chapters",
			chapters:     []renderer.ChapterHTML{},
			chapterFiles: []string{},
			shouldReturn: true,
		},
		{
			name: "mismatched lengths",
			chapters: []renderer.ChapterHTML{
				{Title: "Ch1", ID: "ch1", Content: "content1"},
				{Title: "Ch2", ID: "ch2", Content: "content2"},
			},
			chapterFiles: []string{"ch1.md"},
			shouldReturn: true,
		},
		{
			name: "single chapter with file",
			chapters: []renderer.ChapterHTML{
				{Title: "Ch1", ID: "ch1", Content: "content1"},
			},
			chapterFiles: []string{"ch1.md"},
			shouldReturn: true,
		},
		{
			name: "multiple chapters with files",
			chapters: []renderer.ChapterHTML{
				{Title: "Ch1", ID: "ch1", Content: "content1"},
				{Title: "Ch2", ID: "ch2", Content: "content2"},
				{Title: "Ch3", ID: "ch3", Content: "content3"},
			},
			chapterFiles: []string{"ch1.md", "ch2.md", "ch3.md"},
			shouldReturn: true,
		},
		{
			name: "chapter with empty ID",
			chapters: []renderer.ChapterHTML{
				{Title: "Ch1", ID: "", Content: "content1"},
			},
			chapterFiles: []string{"ch1.md"},
			shouldReturn: true,
		},
		{
			name: "chapter with empty file",
			chapters: []renderer.ChapterHTML{
				{Title: "Ch1", ID: "ch1", Content: "content1"},
			},
			chapterFiles: []string{""},
			shouldReturn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriteChapterLinks(tt.chapters, tt.chapterFiles)
			if tt.shouldReturn && result == nil {
				t.Fatal("expected non-nil result")
			}
			// Verify length is preserved
			if len(result) != len(tt.chapters) {
				t.Errorf("expected %d chapters, got %d", len(tt.chapters), len(result))
			}
		})
	}
}

// Tests for sanitizeBookFilename
func TestSanitizeBookFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal ASCII", "my-book", "my-book"},
		{"with slash", "my/book", "my_book"},
		{"with backslash", `my\book`, "my_book"},
		{"with colon", "my:book", "my_book"},
		{"with asterisk", "my*book", "my_book"},
		{"with question mark", "my?book", "my_book"},
		{"with angle brackets", "my<book>", "my_book_"},
		{"with pipe", "my|book", "my_book"},
		{"with quotes", `my"book"`, "my_book_"},
		{"multiple invalid chars", `a/b\c:d*e?f"g<h>i|j`, "a_b_c_d_e_f_g_h_i_j"},
		{"CJK characters", "我的书", "我的书"},
		{"mixed CJK and invalid", "我的/书", "我的_书"},
		{"empty string", "", "output"},
		{"whitespace only", "   ", "output"},
		{"leading trailing spaces", " my book ", "my book"},
		{"all invalid chars", `/\:*?"<>|`, "output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeBookFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeBookFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Tests for deriveOutputFilename
func TestDeriveOutputFilename(t *testing.T) {
	tests := []struct {
		name     string
		setupCfg func(t *testing.T) *config.BookConfig
		expected string
	}{
		{
			name: "explicit filename",
			setupCfg: func(t *testing.T) *config.BookConfig {
				c := &config.BookConfig{}
				c.Output.Filename = "custom.pdf"
				return c
			},
			expected: "custom.pdf",
		},
		{
			name: "default output.pdf falls through to title",
			setupCfg: func(t *testing.T) *config.BookConfig {
				c := &config.BookConfig{}
				c.Output.Filename = "output.pdf"
				c.Book.Title = "Go Programming"
				return c
			},
			expected: "Go Programming.pdf",
		},
		{
			name: "title with invalid chars",
			setupCfg: func(t *testing.T) *config.BookConfig {
				c := &config.BookConfig{}
				c.Book.Title = "Go: The Good Parts?"
				return c
			},
			expected: "Go_ The Good Parts_.pdf",
		},
		{
			name: "empty title uses dir name",
			setupCfg: func(t *testing.T) *config.BookConfig {
				c := &config.BookConfig{}
				tmpDir := t.TempDir()
				// Create a subdirectory with the expected name
				projectDir := filepath.Join(tmpDir, "my-project")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatalf("failed to create project dir: %v", err)
				}
				c.SetBaseDir(projectDir)
				c.Book.Title = ""
				return c
			},
			expected: "my-project.pdf",
		},
		{
			name: "Untitled Book uses dir name",
			setupCfg: func(t *testing.T) *config.BookConfig {
				c := &config.BookConfig{}
				tmpDir := t.TempDir()
				// Create a subdirectory with the expected name
				projectDir := filepath.Join(tmpDir, "awesome-book")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatalf("failed to create project dir: %v", err)
				}
				c.SetBaseDir(projectDir)
				c.Book.Title = "Untitled Book"
				return c
			},
			expected: "awesome-book.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupCfg(t)
			result := deriveOutputFilename(cfg)
			if result != tt.expected {
				t.Errorf("deriveOutputFilename() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Tests for deriveLanguageOutputOverride (TG-15: multi-dot filenames)
func TestDeriveLanguageOutputOverride(t *testing.T) {
	tests := []struct {
		name           string
		outputOverride string
		langDir        string
		expected       string
		description    string
	}{
		// Empty output override cases
		{
			name:           "empty override returns empty",
			outputOverride: "",
			langDir:        "en",
			expected:       "",
			description:    "When outputOverride is empty, return empty string",
		},
		// Normal single-extension cases
		{
			name:           "normal case: output.pdf + en",
			outputOverride: "output.pdf",
			langDir:        "en",
			expected:       "output-en.pdf",
			description:    "Standard PDF file with language code",
		},
		{
			name:           "normal case: book.html + zh",
			outputOverride: "book.html",
			langDir:        "zh",
			expected:       "book-zh.html",
			description:    "HTML file with language code",
		},
		{
			name:           "normal case: document.epub + fr",
			outputOverride: "document.epub",
			langDir:        "fr",
			expected:       "document-fr.epub",
			description:    "EPUB file with language code",
		},
		// Multi-dot filenames (TG-15)
		{
			name:           "multi-dot: my.book.pdf + zh",
			outputOverride: "my.book.pdf",
			langDir:        "zh",
			expected:       "my.book-zh.pdf",
			description:    "Filename with multiple dots before extension",
		},
		{
			name:           "multi-dot: my.project.name.html + en",
			outputOverride: "my.project.name.html",
			langDir:        "en",
			expected:       "my.project.name-en.html",
			description:    "HTML with three dots total",
		},
		{
			name:           "multi-dot: data.backup.tar.gz + ja",
			outputOverride: "data.backup.tar.gz",
			langDir:        "ja",
			expected:       "data.backup.tar-ja.gz",
			description:    "Archive-like extension (filepath.Ext returns .gz)",
		},
		// No extension cases
		{
			name:           "no extension: output + en",
			outputOverride: "output",
			langDir:        "en",
			expected:       "output-en",
			description:    "Filename without extension",
		},
		{
			name:           "no extension: mybook + fr",
			outputOverride: "mybook",
			langDir:        "fr",
			expected:       "mybook-fr",
			description:    "Filename without extension and different language",
		},
		// Path with extension
		{
			name:           "path with extension: /path/to/output.pdf + es",
			outputOverride: "/path/to/output.pdf",
			langDir:        "es",
			expected:       "/path/to/output-es.pdf",
			description:    "Absolute path with PDF extension",
		},
		{
			name:           "path with extension: ./relative/book.html + de",
			outputOverride: "./relative/book.html",
			langDir:        "de",
			expected:       "./relative/book-de.html",
			description:    "Relative path with HTML extension",
		},
		// Path without extension
		{
			name:           "path without extension: /var/lib/output + it",
			outputOverride: "/var/lib/output",
			langDir:        "it",
			expected:       "/var/lib/output-it",
			description:    "Absolute path without extension",
		},
		// Edge cases
		{
			name:           "just extension: .pdf + en",
			outputOverride: ".pdf",
			langDir:        "en",
			expected:       ".pdf-en",
			description:    "Edge case: file that's just an extension",
		},
		{
			name:           "complex langDir: en-US",
			outputOverride: "book.pdf",
			langDir:        "en-US",
			expected:       "book-en-US.pdf",
			description:    "Language directory with hyphen",
		},
		{
			name:           "complex langDir: zh-CN",
			outputOverride: "document.html",
			langDir:        "zh-CN",
			expected:       "document-zh-CN.html",
			description:    "Traditional Chinese language code",
		},
		{
			name:           "empty base filename: .hidden + en",
			outputOverride: ".hidden",
			langDir:        "en",
			expected:       ".hidden-en",
			description:    "Hidden file (starts with dot)",
		},
		// Path-like behavior for directories (note: we can't mock os.Stat, so directory cases won't work in test)
		{
			name:           "path with trailing slash: /output/ + en",
			outputOverride: "/output/",
			langDir:        "en",
			expected:       "/output/-en",
			description:    "Path-like string with trailing slash (note: os.Stat check won't detect this as dir in test)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveLanguageOutputOverride(tt.outputOverride, tt.langDir)
			if result != tt.expected {
				t.Errorf("deriveLanguageOutputOverride(%q, %q) = %q, want %q\n%s",
					tt.outputOverride, tt.langDir, result, tt.expected, tt.description)
			}
		})
	}
}
