// bookjson.go provides GitBook book.json compatibility for mdPress.
// It parses a GitBook-style book.json file and converts it to a BookConfig.
package config

import (
	"context"
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
		return fmt.Errorf("failed to unmarshal string array: %w", err)
	}
	*v = ss
	return nil
}

// LoadBookJSON reads a GitBook book.json file and returns an equivalent BookConfig.
//
// Metadata fields (title, author, description, language, plugins) are loaded from book.json.
// Chapter definitions are NOT loaded here; instead, Discover() handles chapters via SUMMARY.md
// or auto-discovery, which allows proper priority orchestration of configuration sources.
// The context is used for potentially long-running operations like git commands.
func LoadBookJSON(ctx context.Context, path string) (*BookConfig, error) {
	const maxSize = 10 * 1024 * 1024 // 10 MB
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat book.json: %w", err)
	}
	if info.Size() > maxSize {
		return nil, fmt.Errorf("book.json is too large (%d bytes; max %d bytes)", info.Size(), maxSize)
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

	// Note: Chapter definitions are NOT loaded here. The Discover() function handles
	// loading chapters from SUMMARY.md (or other sources), which allows proper
	// orchestration of configuration sources and avoids redundant parsing.

	// Enrich metadata from README.md for fields not present in book.json.
	readmeName := "README.md"
	if raw.Structure.Readme != "" {
		readmeName = raw.Structure.Readme
	}
	readmePath, err := safeJoin(dir, readmeName)
	if err != nil {
		return nil, fmt.Errorf("invalid structure.readme: %w", err)
	}
	meta := ExtractReadmeMetadata(ctx, readmePath)
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
	glossaryPath, err := safeJoin(dir, glossaryName)
	if err != nil {
		return nil, fmt.Errorf("invalid structure.glossary: %w", err)
	}
	if _, err := os.Stat(glossaryPath); err == nil {
		cfg.GlossaryFile = glossaryPath
	}

	// Auto-detect LANGS.md (honor structure.languages override).
	langsName := "LANGS.md"
	if raw.Structure.Langs != "" {
		langsName = raw.Structure.Langs
	}
	langsPath, err := safeJoin(dir, langsName)
	if err != nil {
		return nil, fmt.Errorf("invalid structure.languages: %w", err)
	}
	if _, err := os.Stat(langsPath); err == nil {
		cfg.LangsFile = langsPath
	}

	return cfg, nil
}

// safeJoin joins a base directory and a relative name, rejecting paths that
// escape the base via ".." traversal or absolute paths.
func safeJoin(base, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("absolute path not allowed: %q", name)
	}
	joined := filepath.Join(base, name)
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absJoined, absBase+string(filepath.Separator)) && absJoined != absBase {
		return "", fmt.Errorf("path escapes project directory: %q", name)
	}
	return joined, nil
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
