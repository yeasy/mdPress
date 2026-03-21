// bookjson.go provides GitBook book.json compatibility for mdPress.
// It parses a GitBook-style book.json file and converts it to a BookConfig.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// bookJSON is the raw structure of a GitBook book.json file.
// Only fields relevant to mdPress are decoded; unknown fields are silently ignored.
type bookJSON struct {
	Title       string                            `json:"title"`
	Author      jsonStringOrSlice                 `json:"author"`
	Description string                            `json:"description"`
	Language    string                            `json:"language"`
	GitBook     string                            `json:"gitbook"`
	Plugins     []string                          `json:"plugins"`
	PluginsCfg  map[string]map[string]interface{} `json:"pluginsConfig"`
	Structure   bookJSONStructure                 `json:"structure"`
}

// bookJSONStructure holds optional overrides for well-known GitBook file paths.
type bookJSONStructure struct {
	Readme   string `json:"readme"`
	Summary  string `json:"summary"`
	Glossary string `json:"glossary"`
	Langs    string `json:"languages"`
}

// jsonStringOrSlice decodes a JSON field that may be either a single string
// or an array of strings (GitBook allows both for the "author" field).
type jsonStringOrSlice []string

func (v *jsonStringOrSlice) UnmarshalJSON(data []byte) error {
	// Try a plain string first.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = []string{s}
		return nil
	}
	// Fall back to a string array.
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	*v = ss
	return nil
}

// LoadBookJSON reads a GitBook book.json file and returns an equivalent BookConfig.
//
// Chapter definitions are loaded from the SUMMARY.md referenced in
// book.json's structure.summary field (defaults to SUMMARY.md in the same
// directory). If no SUMMARY.md is present, chapters are left empty and the
// caller is expected to populate them via auto-discovery.
func LoadBookJSON(path string) (*BookConfig, error) {
	const maxSize = 10 * 1024 * 1024 // 10 MB
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read book.json: %w", err)
	}
	if fi.Size() > maxSize {
		return nil, fmt.Errorf("book.json is too large (%d bytes; max %d bytes)", fi.Size(), maxSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read book.json: %w", err)
	}

	var raw bookJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse book.json: %w (check JSON syntax in %s)", err, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve book.json path: %w", err)
	}
	dir := filepath.Dir(absPath)

	cfg := DefaultConfig()
	cfg.baseDir = dir

	// Map basic metadata fields.
	if raw.Title != "" {
		cfg.Book.Title = raw.Title
	}
	if len(raw.Author) > 0 {
		cfg.Book.Author = strings.Join(raw.Author, ", ")
	}
	if raw.Description != "" {
		cfg.Book.Description = raw.Description
	}
	if raw.Language != "" {
		cfg.Book.Language = normalizeLanguage(raw.Language)
	}

	// Convert plugins: skip entries prefixed with "-" (GitBook disables them).
	cfg.Plugins = convertBookJSONPlugins(raw.Plugins, raw.PluginsCfg)

	// Resolve the SUMMARY.md path (may be overridden in book.json structure).
	summaryName := "SUMMARY.md"
	if raw.Structure.Summary != "" {
		summaryName = raw.Structure.Summary
	}
	summaryPath := filepath.Join(dir, summaryName)
	if _, err := os.Stat(summaryPath); err == nil {
		chapters, err := ParseSummary(summaryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s referenced from book.json: %w", summaryName, err)
		}
		cfg.Chapters = chapters
	}

	// Enrich metadata from README.md for fields not present in book.json.
	readmeName := "README.md"
	if raw.Structure.Readme != "" {
		readmeName = raw.Structure.Readme
	}
	readmePath := filepath.Join(dir, readmeName)
	meta := ExtractReadmeMetadata(readmePath)
	if cfg.Book.Version == DefaultConfig().Book.Version && meta.Version != "" {
		cfg.Book.Version = meta.Version
	}
	if cfg.Book.Author == "" && meta.Author != "" {
		cfg.Book.Author = meta.Author
	}

	// Auto-detect GLOSSARY.md (honor structure.glossary override).
	glossaryName := "GLOSSARY.md"
	if raw.Structure.Glossary != "" {
		glossaryName = raw.Structure.Glossary
	}
	glossaryPath := filepath.Join(dir, glossaryName)
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	// Auto-detect LANGS.md (honor structure.languages override).
	langsName := "LANGS.md"
	if raw.Structure.Langs != "" {
		langsName = raw.Structure.Langs
	}
	langsPath := filepath.Join(dir, langsName)
	if _, err := os.Stat(langsPath); err == nil {
		cfg.LangsFile = langsPath
	}

	return cfg, nil
}

// normalizeLanguage converts a GitBook short language code to a BCP 47 tag
// understood by mdPress (e.g. "en" → "en-US", "zh-hans" → "zh-CN").
func normalizeLanguage(lang string) string {
	switch strings.ToLower(lang) {
	case "en", "en-us":
		return "en-US"
	case "en-gb":
		return "en-GB"
	case "zh", "zh-cn", "zh-hans", "zh-hans-cn":
		return "zh-CN"
	case "zh-tw", "zh-hant", "zh-hant-tw":
		return "zh-TW"
	case "ja":
		return "ja-JP"
	case "ko":
		return "ko-KR"
	case "fr":
		return "fr-FR"
	case "de":
		return "de-DE"
	case "es":
		return "es-ES"
	case "pt", "pt-br":
		return "pt-BR"
	case "ru":
		return "ru-RU"
	default:
		// Return as-is for unknown codes so users aren't silently misled.
		return lang
	}
}

// convertBookJSONPlugins maps GitBook plugin entries to PluginConfig values.
// Entries prefixed with "-" are disabled in GitBook and are skipped here.
func convertBookJSONPlugins(names []string, cfgs map[string]map[string]interface{}) []PluginConfig {
	if len(names) == 0 {
		return nil
	}
	var plugins []PluginConfig
	for _, name := range names {
		// GitBook uses a "-name" prefix to disable a plugin.
		if strings.HasPrefix(name, "-") {
			continue
		}
		p := PluginConfig{Name: name}
		if cfgs != nil {
			if extra, ok := cfgs[name]; ok {
				p.Config = make(map[string]interface{}, len(extra))
				for k, v := range extra {
					p.Config[k] = v
				}
			}
		}
		plugins = append(plugins, p)
	}
	return plugins
}
