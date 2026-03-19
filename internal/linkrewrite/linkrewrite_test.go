package linkrewrite

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"chapter01/README.md", "chapter01/README.md"},
		{"./chapter01/README.md", "chapter01/README.md"},
		{"chapter01/../chapter02/README.md", "chapter02/README.md"},
		{".", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := NormalizePath(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRewriteLinks_SingleMode(t *testing.T) {
	targets := map[string]Target{
		"chapter01/README.md": {ChapterID: "ch01"},
		"chapter02/README.md": {ChapterID: "ch02"},
		"appendix.md":         {ChapterID: "appendix"},
	}

	tests := []struct {
		name        string
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "rewrite relative .md link",
			html:        `<a href="chapter01/README.md">Chapter 1</a>`,
			currentFile: "README.md",
			want:        `<a href="#ch01">Chapter 1</a>`,
		},
		{
			name:        "rewrite .md link with fragment",
			html:        `<a href="chapter02/README.md#section">Section</a>`,
			currentFile: "README.md",
			want:        `<a href="#section">Section</a>`,
		},
		{
			name:        "leave http links unchanged",
			html:        `<a href="https://example.com">Example</a>`,
			currentFile: "README.md",
			want:        `<a href="https://example.com">Example</a>`,
		},
		{
			name:        "leave anchor links unchanged",
			html:        `<a href="#section">Section</a>`,
			currentFile: "README.md",
			want:        `<a href="#section">Section</a>`,
		},
		{
			name:        "leave non-md links unchanged",
			html:        `<a href="image.png">Image</a>`,
			currentFile: "README.md",
			want:        `<a href="image.png">Image</a>`,
		},
		{
			name:        "mark unresolved .md links",
			html:        `<a href="missing.md">Missing</a>`,
			currentFile: "README.md",
			want:        `<a href="missing.md" data-mdpress-link="unresolved-markdown" title="Markdown link target is outside the current build graph">Missing</a>`,
		},
		{
			name:        "rewrite sibling link from subdirectory",
			html:        `<a href="../appendix.md">Appendix</a>`,
			currentFile: "chapter01/README.md",
			want:        `<a href="#appendix">Appendix</a>`,
		},
		{
			name:        "leave mailto links unchanged",
			html:        `<a href="mailto:test@example.com">Email</a>`,
			currentFile: "README.md",
			want:        `<a href="mailto:test@example.com">Email</a>`,
		},
		{
			name:        "empty html returns empty",
			html:        "",
			currentFile: "README.md",
			want:        "",
		},
		{
			name:        "single quote href",
			html:        `<a href='chapter01/README.md'>Ch1</a>`,
			currentFile: "README.md",
			want:        `<a href='#ch01'>Ch1</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, targets, ModeSingle)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinks_SiteMode(t *testing.T) {
	targets := map[string]Target{
		"chapter01/README.md": {ChapterID: "ch01", PageFilename: "ch_000.html"},
		"chapter02/README.md": {ChapterID: "ch02", PageFilename: "ch_001.html"},
	}

	tests := []struct {
		name        string
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "rewrite to page filename",
			html:        `<a href="chapter01/README.md">Chapter 1</a>`,
			currentFile: "README.md",
			want:        `<a href="ch_000.html">Chapter 1</a>`,
		},
		{
			name:        "rewrite with fragment to page filename",
			html:        `<a href="chapter02/README.md#intro">Intro</a>`,
			currentFile: "README.md",
			want:        `<a href="ch_001.html#intro">Intro</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, targets, ModeSite)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinks_EdgeCases(t *testing.T) {
	targets := map[string]Target{
		"docs/chapter.md": {ChapterID: "ch01"},
	}

	tests := []struct {
		name        string
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "empty targets map returns unchanged",
			html:        `<a href="chapter.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="chapter.md">Link</a>`,
		},
		{
			name:        "empty currentFile returns unchanged",
			html:        `<a href="chapter.md">Link</a>`,
			currentFile: "",
			want:        `<a href="chapter.md">Link</a>`,
		},
		{
			name:        "multiple links in content",
			html:        `<a href="docs/chapter.md">Ch1</a> and <a href="https://example.com">Example</a>`,
			currentFile: "README.md",
			want:        `<a href="#ch01">Ch1</a> and <a href="https://example.com">Example</a>`,
		},
		{
			name:        "mixed quote styles",
			html:        `<a href="docs/chapter.md">A</a><b href='docs/chapter.md'>B</b>`,
			currentFile: "README.md",
			want:        `<a href="#ch01">A</a><b href='#ch01'>B</b>`,
		},
		{
			name:        "link with leading and trailing whitespace",
			html:        `<a href=" docs/chapter.md ">Link</a>`,
			currentFile: "README.md",
			want:        `<a href=" docs/chapter.md ">Link</a>`,
		},
		{
			name:        "uppercase .MD extension",
			html:        `<a href="docs/chapter.MD">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="#ch01">Link</a>`,
		},
		{
			name:        "protocol-relative URL unchanged",
			html:        `<a href="//example.com/path">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="//example.com/path">Link</a>`,
		},
		{
			name:        "javascript: protocol unchanged",
			html:        `<a href="javascript:alert('test')">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="javascript:alert('test')">Link</a>`,
		},
		{
			name:        "data: protocol unchanged",
			html:        `<a href="data:text/html,<script>alert('xss')</script>">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="data:text/html,<script>alert('xss')</script>">Link</a>`,
		},
		{
			name:        "absolute path unchanged",
			html:        `<a href="/absolute/path.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="/absolute/path.md">Link</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, targets, ModeSingle)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinks_DeepPaths(t *testing.T) {
	targets := map[string]Target{
		"part1/chapter1/section1.md":     {ChapterID: "s1"},
		"part1/chapter1/section2.md":     {ChapterID: "s2"},
		"part1/chapter2/section1.md":     {ChapterID: "s3"},
		"part2/chapter1/subsection.md":   {ChapterID: "s4"},
		"shared/resource.md":             {ChapterID: "res"},
	}

	tests := []struct {
		name        string
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "same directory relative link",
			html:        `<a href="section2.md">S2</a>`,
			currentFile: "part1/chapter1/section1.md",
			want:        `<a href="#s2">S2</a>`,
		},
		{
			name:        "parent directory relative link",
			html:        `<a href="../chapter2/section1.md">S3</a>`,
			currentFile: "part1/chapter1/section1.md",
			want:        `<a href="#s3">S3</a>`,
		},
		{
			name:        "multi-level parent directory link",
			html:        `<a href="../../part2/chapter1/subsection.md">S4</a>`,
			currentFile: "part1/chapter1/section1.md",
			want:        `<a href="#s4">S4</a>`,
		},
		{
			name:        "shared resource from deep directory",
			html:        `<a href="../../shared/resource.md">Resource</a>`,
			currentFile: "part1/chapter1/section1.md",
			want:        `<a href="#res">Resource</a>`,
		},
		{
			name:        "fragment preservation in deep paths",
			html:        `<a href="../chapter2/section1.md#intro">Intro</a>`,
			currentFile: "part1/chapter1/section1.md",
			want:        `<a href="#intro">Intro</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, targets, ModeSingle)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinks_EmptyTargets(t *testing.T) {
	tests := []struct {
		name        string
		targets     map[string]Target
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "nil targets",
			targets:     nil,
			html:        `<a href="test.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="test.md">Link</a>`,
		},
		{
			name:        "empty targets",
			targets:     map[string]Target{},
			html:        `<a href="test.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="test.md">Link</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, tt.targets, ModeSingle)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinks_ModeVariations(t *testing.T) {
	targets := map[string]Target{
		"chapter.md": {ChapterID: "ch1", PageFilename: "page.html"},
	}

	tests := []struct {
		name        string
		mode        Mode
		html        string
		currentFile string
		want        string
	}{
		{
			name:        "invalid mode defaults to single",
			mode:        Mode("invalid"),
			html:        `<a href="chapter.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="#ch1">Link</a>`,
		},
		{
			name:        "site mode without PageFilename",
			mode:        ModeSite,
			html:        `<a href="chapter.md">Link</a>`,
			currentFile: "README.md",
			want:        `<a href="page.html">Link</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteLinks(tt.html, tt.currentFile, targets, tt.mode)
			if got != tt.want {
				t.Errorf("RewriteLinks() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}
