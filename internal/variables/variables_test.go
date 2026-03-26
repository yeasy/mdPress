package variables

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

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
	result := ExpandString(input, cfg)
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
		result := ExpandString("{{ "+key+" }}", cfg)
		if result != expected {
			t.Errorf("%s: got %q, want %q", key, result, expected)
		}
	}
}

func TestExpandUnknownVar(t *testing.T) {
	cfg := newTestConfig()
	input := "{{ unknown.var }}"
	result := ExpandString(input, cfg)
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
	result := ExpandString(input, cfg)
	if result != input {
		t.Error("no vars should not modify input")
	}
}

func TestExpandInMarkdown(t *testing.T) {
	cfg := newTestConfig()
	input := "# {{ book.title }}\n\n作者: {{ book.author }}\n\n版本 {{ book.version }}"
	result := ExpandString(input, cfg)
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
	result := ExpandString(input, cfg)
	if !strings.Contains(result, "测试书名") {
		t.Error("should expand book.title")
	}
	if !strings.Contains(result, "{{ref:fig1}}") {
		t.Error("should not touch cross-ref syntax")
	}
}
