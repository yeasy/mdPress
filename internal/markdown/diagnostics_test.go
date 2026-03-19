package markdown

import "testing"

func TestParseWithDiagnosticsOrderedListGap(t *testing.T) {
	parser := NewParser()
	md := "1. 第一项\n3. 第三项\n"

	_, _, diagnostics, err := parser.ParseWithDiagnostics([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}
	if diagnostics[0].Rule != "ordered-list-sequence" {
		t.Fatalf("unexpected rule: %s", diagnostics[0].Rule)
	}
	if diagnostics[0].Line != 2 || diagnostics[0].Column != 1 {
		t.Fatalf("unexpected position: %d:%d", diagnostics[0].Line, diagnostics[0].Column)
	}
}

func TestParseWithDiagnosticsOrderedListAllowsAllOnes(t *testing.T) {
	parser := NewParser()
	md := "1. 第一项\n1. 第二项\n1. 第三项\n"

	_, _, diagnostics, err := parser.ParseWithDiagnostics([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for _, diag := range diagnostics {
		if diag.Rule == "ordered-list-sequence" {
			t.Fatalf("did not expect ordered-list warning: %+v", diag)
		}
	}
}

func TestParseWithDiagnosticsMermaidUnknownDiagram(t *testing.T) {
	parser := NewParser()
	md := "```mermaid\nnot-a-diagram\n```\n"

	_, _, diagnostics, err := parser.ParseWithDiagnostics([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}
	if diagnostics[0].Rule != "mermaid-unknown-diagram" {
		t.Fatalf("unexpected rule: %s", diagnostics[0].Rule)
	}
	if diagnostics[0].Line != 2 || diagnostics[0].Column != 1 {
		t.Fatalf("unexpected position: %d:%d", diagnostics[0].Line, diagnostics[0].Column)
	}
}

func TestParseWithDiagnosticsMermaidBracketMismatch(t *testing.T) {
	parser := NewParser()
	md := "```mermaid\ngraph TD\n    A[Start --> B\n```\n"

	_, _, diagnostics, err := parser.ParseWithDiagnostics([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	found := false
	for _, diag := range diagnostics {
		if diag.Rule == "mermaid-bracket-mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mermaid bracket diagnostic, got %+v", diagnostics)
	}
}

func TestParseWithDiagnosticsMermaidUnclosedFence(t *testing.T) {
	parser := NewParser()
	md := "```mermaid\ngraph TD\n    A-->B\n"

	_, _, diagnostics, err := parser.ParseWithDiagnostics([]byte(md))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}
	if diagnostics[0].Rule != "mermaid-unclosed-fence" {
		t.Fatalf("unexpected rule: %s", diagnostics[0].Rule)
	}
	if diagnostics[0].Line != 1 || diagnostics[0].Column != 1 {
		t.Fatalf("unexpected position: %d:%d", diagnostics[0].Line, diagnostics[0].Column)
	}
}
