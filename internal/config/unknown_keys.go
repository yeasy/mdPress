// unknown_keys.go reports book.yaml keys that mdpress does not recognize.
//
// The loader is deliberately non-strict — an unknown key must not break a
// build that used to work, and forward compatibility matters for a config file
// people copy between versions. But silently discarding a key is the worst
// possible outcome for the user: they edit book.yaml, rebuild, see no change,
// and have nothing to debug with. So unknown keys are surfaced as warnings,
// with a suggestion when a known sibling is close enough to be the intent.
package config

import (
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnknownKey is a config key with no corresponding field.
type UnknownKey struct {
	// Path is the dotted location in the document, e.g. "style.them".
	Path string
	// Suggestion is the closest known sibling key, or "" if none is close.
	Suggestion string
}

// Hint renders the "did you mean" text for a warning, or a generic note.
func (u UnknownKey) Hint() string {
	if u.Suggestion == "" {
		return "not a recognized mdpress setting"
	}
	return `did you mean "` + u.Suggestion + `"?`
}

// FindUnknownKeys parses data as a generic document and walks it against the
// BookConfig type, collecting keys with no matching field.
func FindUnknownKeys(data []byte) []UnknownKey {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil // a parse error is reported by the caller's own unmarshal
	}
	var found []UnknownKey
	walkUnknownKeys(doc, reflect.TypeOf(BookConfig{}), "", map[reflect.Type]bool{}, &found)
	return found
}

// walkUnknownKeys compares one mapping level against a struct type and
// recurses into nested mappings.
func walkUnknownKeys(node map[string]any, t reflect.Type, prefix string, seen map[reflect.Type]bool, out *[]UnknownKey) {
	// Self-referential types (ChapterDef.Sections) would recurse forever, and
	// re-checking the same shape adds nothing.
	if seen[t] {
		return
	}
	seen[t] = true
	defer delete(seen, t)

	known := yamlFields(t)
	for key, value := range node {
		field, ok := known[key]
		if !ok {
			*out = append(*out, UnknownKey{
				Path:       prefix + key,
				Suggestion: closestKey(key, known),
			})
			continue
		}
		descend(value, field.Type, prefix+key+".", seen, out)
	}
}

// descend follows a value into nested mappings and sequences.
func descend(value any, ft reflect.Type, prefix string, seen map[reflect.Type]bool, out *[]UnknownKey) {
	for ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	switch typed := value.(type) {
	case map[string]any:
		if ft.Kind() == reflect.Struct {
			walkUnknownKeys(typed, ft, prefix, seen, out)
		}
	case []any:
		if ft.Kind() != reflect.Slice {
			return
		}
		elem := ft.Elem()
		for ft.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
		if elem.Kind() != reflect.Struct {
			return
		}
		for _, item := range typed {
			if m, ok := item.(map[string]any); ok {
				walkUnknownKeys(m, elem, prefix, seen, out)
			}
		}
	}
}

// yamlFields maps a struct's yaml key names to their fields, skipping fields
// excluded from serialization.
func yamlFields(t reflect.Type) map[string]reflect.StructField {
	fields := make(map[string]reflect.StructField, t.NumField())
	if t.Kind() != reflect.Struct {
		return fields
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		fields[name] = f
	}
	return fields
}

// maxSuggestionDistance caps how far a typo may be from a known key before the
// suggestion becomes noise rather than help.
const maxSuggestionDistance = 3

// closestKey returns the known key nearest to name, or "" when nothing is
// close enough to suggest confidently.
func closestKey(name string, known map[string]reflect.StructField) string {
	best, bestDist := "", maxSuggestionDistance+1
	for candidate := range known {
		d := editDistance(name, candidate)
		// Prefer the shorter candidate on ties so suggestions are stable
		// regardless of map iteration order.
		if d < bestDist || (d == bestDist && candidate < best) {
			best, bestDist = candidate, d
		}
	}
	if bestDist > maxSuggestionDistance {
		return ""
	}
	return best
}

// editDistance is the Levenshtein distance between two short strings.
func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
