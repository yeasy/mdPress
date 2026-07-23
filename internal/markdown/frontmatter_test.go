package markdown

import "testing"

// bom is the UTF-8 byte order mark, written as an escape so this source file
// does not itself start with one.
const bom = "\xef\xbb\xbf"

func TestStripLeadingMetadata(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantBody string
		wantFM   FrontMatter
	}{
		{
			name:     "plain document is untouched",
			source:   "# Title\n\nbody\n",
			wantBody: "# Title\n\nbody\n",
		},
		{
			name:     "BOM is removed so the first heading parses",
			source:   bom + "# Title\n\nbody\n",
			wantBody: "# Title\n\nbody\n",
		},
		{
			name:     "front matter is removed and read",
			source:   "---\ntitle: Real Title\ndescription: A description\n---\n\n# Heading\n",
			wantBody: "\n# Heading\n",
			wantFM:   FrontMatter{Title: "Real Title", Description: "A description"},
		},
		{
			name:     "quoted values are unwrapped",
			source:   "---\ntitle: \"Quoted\"\n---\nbody\n",
			wantBody: "body\n",
			wantFM:   FrontMatter{Title: "Quoted"},
		},
		{
			name:     "unknown keys are ignored, not rejected",
			source:   "---\nlayout: post\ntags: [a, b]\ntitle: T\n---\nbody\n",
			wantBody: "body\n",
			wantFM:   FrontMatter{Title: "T"},
		},
		{
			name:     "BOM followed by front matter",
			source:   bom + "---\ntitle: T\n---\nbody\n",
			wantBody: "body\n",
			wantFM:   FrontMatter{Title: "T"},
		},
		{
			// Swallowing the document would be far worse than rendering a
			// stray rule.
			name:     "unterminated block is treated as content",
			source:   "---\ntitle: T\n\n# Heading\n",
			wantBody: "---\ntitle: T\n\n# Heading\n",
		},
		{
			name:     "a rule later in the document is not front matter",
			source:   "# Title\n\n---\n\nmore\n",
			wantBody: "# Title\n\n---\n\nmore\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, fm := StripLeadingMetadata([]byte(tt.source))
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if fm != tt.wantFM {
				t.Errorf("front matter = %+v, want %+v", fm, tt.wantFM)
			}
		})
	}
}
