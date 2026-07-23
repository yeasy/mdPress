package config

import (
	"strings"
	"testing"
)

func TestFindUnknownKeys(t *testing.T) {
	tests := []struct {
		name           string
		yaml           string
		wantPath       string
		wantSuggestion string
	}{
		{
			name:           "typo in a top-level key",
			yaml:           "book:\n  title: \"T\"\nchapter:\n  - title: \"A\"\n",
			wantPath:       "chapter",
			wantSuggestion: "chapters",
		},
		{
			name:           "typo in a nested key",
			yaml:           "style:\n  them: \"technical\"\n",
			wantPath:       "style.them",
			wantSuggestion: "theme",
		},
		{
			name:           "typo inside a chapter entry",
			yaml:           "chapters:\n  - titel: \"A\"\n    file: \"a.md\"\n",
			wantPath:       "chapters.titel",
			wantSuggestion: "title",
		},
		{
			name:           "unrelated key gets no suggestion",
			yaml:           "book:\n  title: \"T\"\nzzzzzzzzzz: 1\n",
			wantPath:       "zzzzzzzzzz",
			wantSuggestion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := FindUnknownKeys([]byte(tt.yaml))
			suggestion, reported := "", false
			for _, key := range found {
				if key.Path == tt.wantPath {
					suggestion, reported = key.Suggestion, true
					break
				}
			}
			if !reported {
				t.Fatalf("unknown key %q not reported; got %+v", tt.wantPath, found)
			}
			if suggestion != tt.wantSuggestion {
				t.Errorf("suggestion for %q = %q, want %q", tt.wantPath, suggestion, tt.wantSuggestion)
			}
		})
	}
}

func TestFindUnknownKeys_ValidConfigIsSilent(t *testing.T) {
	valid := `book:
  title: "T"
  author: "A"
  cover:
    background: "#111"
chapters:
  - title: "One"
    file: "one.md"
    sections:
      - title: "Sub"
        file: "sub.md"
style:
  theme: "technical"
  margin:
    top: 20
output:
  toc: true
  formats: ["pdf"]
plugins:
  - name: p
    path: ./p
    config:
      any_key_here: 1
`
	if found := FindUnknownKeys([]byte(valid)); len(found) != 0 {
		t.Errorf("valid config reported unknown keys: %+v", found)
	}
}

func TestUnknownKeyHint(t *testing.T) {
	if h := (UnknownKey{Path: "x", Suggestion: "y"}).Hint(); !strings.Contains(h, `"y"`) {
		t.Errorf("hint should name the suggestion, got %q", h)
	}
	if h := (UnknownKey{Path: "x"}).Hint(); strings.Contains(h, "did you mean") {
		t.Errorf("hint should not suggest when there is no candidate, got %q", h)
	}
}
