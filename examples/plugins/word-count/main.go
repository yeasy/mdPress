// word-count is an example mdpress plugin that counts words in each chapter.
//
// Protocol:
//   - Reads a JSON request from stdin on each hook invocation.
//   - Writes a JSON response to stdout with optional content modifications.
//   - Logs statistics to stderr; mdpress captures stderr and emits it as debug logs.
//
// Supported flags:
//
//	--mdpress-info   Print plugin metadata as JSON and exit.
//	--mdpress-hooks  Print the list of hook phases this plugin handles and exit.
//
// Example book.yaml entry:
//
//	plugins:
//	  - name: word-count
//	    path: ./plugins/word-count
//	    config:
//	      warn_threshold: 500
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Request mirrors plugin.ExternalPluginRequest.
type Request struct {
	Phase        string         `json:"phase"`
	Content      string         `json:"content"`
	ChapterIndex int            `json:"chapter_index"`
	ChapterFile  string         `json:"chapter_file"`
	Config       map[string]any `json:"config"`
	Metadata     map[string]any `json:"metadata"`
}

// Response mirrors plugin.ExternalPluginResponse.
type Response struct {
	Content string `json:"content"`
	Stop    bool   `json:"stop"`
}

// tagRE strips HTML tags for a rough word count on rendered HTML content.
var tagRE = regexp.MustCompile(`<[^>]+>`)

func countWords(content string) int {
	// Remove HTML tags before counting.
	plain := tagRE.ReplaceAllString(content, " ")
	return len(strings.Fields(plain))
}

func main() {
	// Handle metadata/hooks query flags so mdpress can discover plugin capabilities.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--mdpress-info":
			info := map[string]string{
				"version":     "1.0.0",
				"description": "Counts words in each chapter and warns when below a threshold.",
			}
			if err := json.NewEncoder(os.Stdout).Encode(info); err != nil {
				os.Exit(1)
			}
			return
		case "--mdpress-hooks":
			// Only interested in the AfterParse phase.
			hooks := []string{"after_parse"}
			if err := json.NewEncoder(os.Stdout).Encode(hooks); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	// Decode the JSON request from stdin.
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintf(os.Stderr, "[word-count] failed to decode request: %v\n", err)
		os.Exit(1)
	}

	wordCount := countWords(req.Content)

	// Read optional configuration.
	warnThreshold := 0
	if req.Config != nil {
		if v, ok := req.Config["warn_threshold"]; ok {
			switch t := v.(type) {
			case float64:
				warnThreshold = int(t)
			case int:
				warnThreshold = t
			}
		}
	}

	chapterLabel := req.ChapterFile
	if chapterLabel == "" {
		chapterLabel = fmt.Sprintf("chapter #%d", req.ChapterIndex+1)
	}

	fmt.Fprintf(os.Stderr, "[word-count] %s: %d words\n", chapterLabel, wordCount)

	if warnThreshold > 0 && wordCount < warnThreshold {
		fmt.Fprintf(os.Stderr,
			"[word-count] WARNING: %s has only %d words (threshold: %d)\n",
			chapterLabel, wordCount, warnThreshold,
		)
	}

	// Return the content unchanged; this plugin is read-only.
	resp := Response{Content: req.Content}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "[word-count] failed to encode response: %v\n", err)
		os.Exit(1)
	}
}
