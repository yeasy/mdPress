// set_keys.go records which settings the project actually wrote down.
//
// Almost every setting mdpress resolves from more than one source has to
// answer the question "did the user configure this?", and comparing the loaded
// value against DefaultConfig() cannot answer it: Load unmarshals *over*
// DefaultConfig, so a field holding the default value is indistinguishable
// from a field the user typed by hand. That confusion has shipped repeatedly —
// a book.json pinned to `"version": "1.0.0"` was replaced by a git tag, a
// theme's page_size never reached any renderer because style.page_size was
// pre-filled with "A4". So key presence is recorded while parsing, and every
// "is this configured?" test asks IsSet instead of guessing from the value.
package config

import "gopkg.in/yaml.v3"

// IsSet reports whether path — a dotted YAML location such as "book.version"
// or "style.margin.top" — was present in the configuration the project
// supplied. It is false for anything that came from DefaultConfig, from
// zero-config discovery, or from any other inference, which is exactly the
// distinction a value comparison cannot make.
func (c *BookConfig) IsSet(path string) bool {
	return c.setKeys[path]
}

// markSet records paths as explicitly configured. Loaders that do not parse
// YAML (book.json) use it to report the same information Load derives from the
// document itself.
func (c *BookConfig) markSet(paths ...string) {
	if c.setKeys == nil {
		c.setKeys = make(map[string]bool, len(paths))
	}
	for _, p := range paths {
		c.setKeys[p] = true
	}
}

// collectSetKeys parses data as a generic document and returns every dotted
// mapping path it contains. Sequence elements are not descended into: the only
// sequences in book.yaml are chapters and plugins, which are read as whole
// values and never merged key by key.
func collectSetKeys(data []byte) map[string]bool {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil // a parse error is reported by the caller's own unmarshal
	}
	keys := make(map[string]bool)
	walkSetKeys(doc, "", keys)
	return keys
}

// walkSetKeys adds one mapping level to out and recurses into nested mappings.
func walkSetKeys(node map[string]any, prefix string, out map[string]bool) {
	for key, value := range node {
		path := prefix + key
		out[path] = true
		if child, ok := value.(map[string]any); ok {
			walkSetKeys(child, path+".", out)
		}
	}
}
