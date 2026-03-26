package markdown_test

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/markdown"
)

func TestCJKHeadingIDs(t *testing.T) {
	p := markdown.NewParser()

	md := []byte("## 4.1 归纳法与机器学习\n\n一些内容\n\n### 中文标题\n\n更多内容\n")

	html, headings, err := p.Parse(md)
	if err != nil {
		t.Fatal("Parse error:", err)
	}

	for _, h := range headings {
		if h.ID == "heading" || h.ID == "" {
			t.Errorf("heading %q has generic ID %q", h.Text, h.ID)
		}
	}

	if strings.Contains(html, "id=\"heading\"") {
		t.Error("HTML contains generic 'heading' ID")
	}

}
