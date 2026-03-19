package cmd

import (
	"testing"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
)

func TestValidateChapterTitleSequenceMismatch(t *testing.T) {
	diag := validateChapterTitleSequence("2. 安装", []markdown.HeadingInfo{
		{Text: "1. 安装", Line: 3, Column: 1},
	})
	if diag == nil {
		t.Fatal("expected diagnostic")
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
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1.md", Heading: markdown.HeadingInfo{Text: "1. 简介", Line: 1, Column: 1}},
		{File: "ch2.md", Heading: markdown.HeadingInfo{Text: "第二章 安装", Line: 1, Column: 1}},
		{File: "ch3.md", Heading: markdown.HeadingInfo{Text: "部署", Line: 1, Column: 1}},
	})
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidateBookTitleConsistencyDuplicateTitles(t *testing.T) {
	warnings := validateBookTitleConsistency([]chapterHeadingRecord{
		{File: "ch1.md", Heading: markdown.HeadingInfo{Text: "1. 简介", Line: 1, Column: 1}},
		{File: "ch2.md", Heading: markdown.HeadingInfo{Text: "2. 简介", Line: 1, Column: 1}},
	})
	found := false
	for _, warning := range warnings {
		if warning.Diagnostic.Rule == "book-title-duplicate" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected duplicate title warning, got %+v", warnings)
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
