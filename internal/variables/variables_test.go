package variables

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

// expandString is the string wrapper for Expand, used only in tests.
func expandString(source string, cfg *config.BookConfig) string {
	return string(Expand([]byte(source), cfg))
}

func newTestConfig() *config.BookConfig {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "测试书名"
	cfg.Book.Author = "张三"
	cfg.Book.Version = "2.0.0"
	cfg.Book.Language = "zh-CN"
	cfg.Book.Subtitle = "副标题"
	cfg.Book.Description = "一本测试书"
	cfg.Style.Theme = "elegant"
	cfg.Output.Filename = "test.pdf"
	return cfg
}

func TestExpandBookTitle(t *testing.T) {
	cfg := newTestConfig()
	result := Expand([]byte("书名是 {{ book.title }}"), cfg)
	if !strings.Contains(string(result), "测试书名") {
		t.Errorf("should replace book.title: got %q", result)
	}
}

func TestExpandNoSpaces(t *testing.T) {
	cfg := newTestConfig()
	result := Expand([]byte("{{book.author}}"), cfg)
	if string(result) != "张三" {
		t.Errorf("should handle no-space syntax: got %q", result)
	}
}

func TestExpandMultipleVars(t *testing.T) {
	cfg := newTestConfig()
	input := "{{ book.title }} by {{ book.author }} v{{ book.version }}"
	result := expandString(input, cfg)
	if result != "测试书名 by 张三 v2.0.0" {
		t.Errorf("got %q", result)
	}
}

func TestExpandAllVars(t *testing.T) {
	cfg := newTestConfig()
	vars := map[string]string{
		"book.title":       "测试书名",
		"book.subtitle":    "副标题",
		"book.author":      "张三",
		"book.version":     "2.0.0",
		"book.language":    "zh-CN",
		"book.description": "一本测试书",
		"style.theme":      "elegant",
		"output.filename":  "test.pdf",
	}
	for key, expected := range vars {
		result := expandString("{{ "+key+" }}", cfg)
		if result != expected {
			t.Errorf("%s: got %q, want %q", key, result, expected)
		}
	}
}

func TestExpandUnknownVar(t *testing.T) {
	cfg := newTestConfig()
	input := "{{ unknown.var }}"
	result := expandString(input, cfg)
	if result != input {
		t.Errorf("unknown var should stay: got %q", result)
	}
}

func TestExpandNilConfig(t *testing.T) {
	input := []byte("{{ book.title }}")
	result := Expand(input, nil)
	if string(result) != string(input) {
		t.Error("nil config should not modify input")
	}
}

func TestExpandNoVars(t *testing.T) {
	cfg := newTestConfig()
	input := "No variables here."
	result := expandString(input, cfg)
	if result != input {
		t.Error("no vars should not modify input")
	}
}

func TestExpandInMarkdown(t *testing.T) {
	cfg := newTestConfig()
	input := "# {{ book.title }}\n\n作者: {{ book.author }}\n\n版本 {{ book.version }}"
	result := expandString(input, cfg)
	if !strings.Contains(result, "# 测试书名") {
		t.Error("should expand in heading")
	}
	if !strings.Contains(result, "作者: 张三") {
		t.Error("should expand in text")
	}
}

func TestExpandMixedContent(t *testing.T) {
	cfg := newTestConfig()
	// {{ ref:fig1 }} is not a valid variable name (contains colon), should not be processed
	input := "{{ book.title }} and {{ref:fig1}}"
	result := expandString(input, cfg)
	if !strings.Contains(result, "测试书名") {
		t.Error("should expand book.title")
	}
	if !strings.Contains(result, "{{ref:fig1}}") {
		t.Error("should not touch cross-ref syntax")
	}
}

// TestExpandSkipsCode covers the case that makes this tool's own manual build
// correctly: a book documenting a templating syntax shows that syntax inside
// code, and substituting there corrupts the example the page exists to show.
func TestExpandSkipsCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "MyBook"

	source := "# A\n\n" +
		"Prose: {{ book.title }}\n\n" +
		"Inline: `{{ book.title }}`\n\n" +
		"```yaml\ntitle: {{ book.title }}\n```\n\n" +
		"~~~\ntilde: {{ book.title }}\n~~~\n"

	got := string(Expand([]byte(source), cfg))

	if !strings.Contains(got, "Prose: MyBook") {
		t.Error("prose variable was not substituted")
	}
	if !strings.Contains(got, "Inline: `{{ book.title }}`") {
		t.Error("variable inside an inline code span was substituted")
	}
	if !strings.Contains(got, "title: {{ book.title }}") {
		t.Error("variable inside a backtick fence was substituted")
	}
	if !strings.Contains(got, "tilde: {{ book.title }}") {
		t.Error("variable inside a tilde fence was substituted")
	}
	if n := strings.Count(got, "MyBook"); n != 1 {
		t.Errorf("expected exactly one substitution, got %d", n)
	}
}

func TestExpandWithUnknownReportsTypos(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Book.Author = "Ann"

	_, unknown := ExpandWithUnknown([]byte("{{ book.author }} and {{ book.autor }} and {{ book.autor }}"), cfg)
	if len(unknown) != 1 || unknown[0] != "book.autor" {
		t.Errorf("unknown = %v, want exactly [book.autor] (deduplicated)", unknown)
	}
}

func TestExpandCustomVariables(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Book.Title = "Built-in"
	cfg.Variables = map[string]string{"product": "Widget Pro", "book.title": "Overridden"}

	got := string(Expand([]byte("{{ product }} / {{ book.title }}"), cfg))
	if got != "Widget Pro / Overridden" {
		t.Errorf("got %q; user-defined variables should work and should win over built-ins", got)
	}
}
