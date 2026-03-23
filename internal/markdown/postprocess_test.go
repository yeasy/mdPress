package markdown

import (
	"strings"
	"testing"
)

// ===== GFM Alerts =====

func TestProcessAlertNote(t *testing.T) {
	// goldmark 把 > [!NOTE]\n> text 渲染成:
	input := "<blockquote>\n<p>[!NOTE]\nThis is important.</p>\n</blockquote>"
	result := PostProcess(input)

	if strings.Contains(result, "<blockquote>") {
		t.Error("blockquote should be replaced by alert div")
	}
	if !strings.Contains(result, "alert-note") {
		t.Error("should have alert-note class")
	}
	if !strings.Contains(result, "ℹ️") {
		t.Error("NOTE should have info icon")
	}
	if !strings.Contains(result, "Note") {
		t.Error("should have Note label")
	}
	if !strings.Contains(result, "This is important.") {
		t.Error("content should be preserved")
	}
}

func TestProcessAlertTip(t *testing.T) {
	input := "<blockquote>\n<p>[!TIP]\nHelpful advice.</p>\n</blockquote>"
	result := PostProcess(input)

	if !strings.Contains(result, "alert-tip") {
		t.Error("should have alert-tip class")
	}
	if !strings.Contains(result, "💡") {
		t.Error("TIP should have lightbulb icon")
	}
}

func TestProcessAlertWarning(t *testing.T) {
	input := "<blockquote>\n<p>[!WARNING]\nBe careful.</p>\n</blockquote>"
	result := PostProcess(input)

	if !strings.Contains(result, "alert-warning") {
		t.Error("should have alert-warning class")
	}
	if !strings.Contains(result, "⚠️") {
		t.Error("WARNING should have warning icon")
	}
}

func TestProcessAlertCaution(t *testing.T) {
	input := "<blockquote>\n<p>[!CAUTION]\nDangerous.</p>\n</blockquote>"
	result := PostProcess(input)

	if !strings.Contains(result, "alert-caution") {
		t.Error("should have alert-caution class")
	}
}

func TestProcessAlertImportant(t *testing.T) {
	input := "<blockquote>\n<p>[!IMPORTANT]\nKey info.</p>\n</blockquote>"
	result := PostProcess(input)

	if !strings.Contains(result, "alert-important") {
		t.Error("should have alert-important class")
	}
}

func TestProcessAlertNormalBlockquote(t *testing.T) {
	// Normal blockquotes without [!TYPE] should not be affected
	input := "<blockquote>\n<p>Just a regular quote.</p>\n</blockquote>"
	result := PostProcess(input)

	if !strings.Contains(result, "<blockquote>") {
		t.Error("normal blockquote should not be converted")
	}
}

func TestProcessAlertMultiple(t *testing.T) {
	input := "<blockquote>\n<p>[!NOTE]\nFirst.</p>\n</blockquote>\n" +
		"<p>Middle text.</p>\n" +
		"<blockquote>\n<p>[!WARNING]\nSecond.</p>\n</blockquote>"
	result := PostProcess(input)

	if strings.Count(result, "alert-note") != 1 {
		t.Error("should have exactly one alert-note")
	}
	if strings.Count(result, "alert-warning") != 1 {
		t.Error("should have exactly one alert-warning")
	}
	if !strings.Contains(result, "Middle text.") {
		t.Error("text between alerts should be preserved")
	}
}

// ===== Mermaid =====

func TestProcessMermaidBasic(t *testing.T) {
	input := `<pre><code class="language-mermaid">graph TD
    A--&gt;B
    B--&gt;C</code></pre>`
	result := PostProcess(input)

	if strings.Contains(result, "<pre>") {
		t.Error("pre block should be replaced")
	}
	if !strings.Contains(result, `class="mermaid"`) {
		t.Error("should have mermaid class div")
	}
	if !strings.Contains(result, "A-->B") {
		t.Error("HTML entities should be decoded")
	}
}

func TestProcessMermaidSequenceDiagram(t *testing.T) {
	input := `<pre><code class="language-mermaid">sequenceDiagram
    Alice-&gt;&gt;Bob: Hello
    Bob-&gt;&gt;Alice: Hi</code></pre>`
	result := PostProcess(input)

	if !strings.Contains(result, "Alice->>Bob") {
		t.Error("should decode entities in sequence diagram")
	}
}

func TestProcessMermaidPreservesOtherCode(t *testing.T) {
	input := `<pre><code class="language-go">func main() {}</code></pre>`
	result := PostProcess(input)

	if !strings.Contains(result, "<pre>") {
		t.Error("non-mermaid code blocks should be preserved")
	}
	if strings.Contains(result, "mermaid") {
		t.Error("go code should not be treated as mermaid")
	}
}

func TestNeedsMermaid(t *testing.T) {
	if NeedsMermaid("<p>no diagrams</p>") {
		t.Error("should return false without mermaid")
	}
	if !NeedsMermaid(`<div class="mermaid">graph TD</div>`) {
		t.Error("should return true with mermaid")
	}
}

func TestMermaidScript(t *testing.T) {
	s := MermaidScript()
	if !strings.Contains(s, "mermaid") {
		t.Error("should contain mermaid script")
	}
}

// ===== Integration: Parse + PostProcess =====

func TestParseGFMAlert(t *testing.T) {
	parser := NewParser()
	md := "> [!NOTE]\n> This is a note about something important."
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !strings.Contains(html, "alert-note") {
		t.Error("parsed alert should be converted")
	}
	if !strings.Contains(html, "Note") {
		t.Error("should have Note label")
	}
}

func TestParseMermaid(t *testing.T) {
	parser := NewParser()
	md := "```mermaid\ngraph LR\n    A-->B\n```"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !strings.Contains(html, `class="mermaid"`) {
		t.Errorf("mermaid block should be converted, got: %s", html[:min(200, len(html))])
	}
}

func TestParseTableRendering(t *testing.T) {
	parser := NewParser()
	md := "| Name | Age |\n|------|-----|\n| Alice | 30 |\n| Bob | 25 |"
	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !strings.Contains(html, "<table>") {
		t.Error("table should be rendered")
	}
	if !strings.Contains(html, "<th>Name</th>") {
		t.Error("table header should be rendered")
	}
	if !strings.Contains(html, "<td>Alice</td>") {
		t.Error("table data should be rendered")
	}
}

func TestParseComplexTable(t *testing.T) {
	parser := NewParser()
	md := `| 左对齐 | 居中 | 右对齐 |
|:-------|:----:|-------:|
| 内容1  | 内容2 | 内容3  |
| **加粗** | *斜体* | ` + "`code`" + ` |`

	html, _, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !strings.Contains(html, "<table>") {
		t.Error("complex table should be rendered")
	}
	if !strings.Contains(html, "<strong>") {
		t.Error("inline formatting in table should work")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ===== Strip Chroma Pre Style =====

func TestStripChromaPreStyle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes background-color from pre",
			input: `<pre style="background-color:#fff;"><code>hello</code></pre>`,
			want:  `<pre><code>hello</code></pre>`,
		},
		{
			name:  "removes multi-property style from pre",
			input: `<pre style="color:#24292e;background-color:#fff;"><code>x</code></pre>`,
			want:  `<pre><code>x</code></pre>`,
		},
		{
			name:  "preserves span inline styles",
			input: `<pre style="background-color:#fff;"><code><span style="color:#d73a49">func</span></code></pre>`,
			want:  `<pre><code><span style="color:#d73a49">func</span></code></pre>`,
		},
		{
			name:  "no style attribute unchanged",
			input: `<pre><code>plain</code></pre>`,
			want:  `<pre><code>plain</code></pre>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripChromaPreStyle(tt.input)
			if got != tt.want {
				t.Errorf("stripChromaPreStyle()\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}
