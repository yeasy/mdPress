// output.go defines the shared output interfaces and registry.
// New output formats can be added through OutputFormat without changing the core build flow.
package output

import (
	"context"
	"fmt"
	"sync"
)

// OutputFormat is the shared interface implemented by each output backend.
type OutputFormat interface {
	// Name returns the format name, for example "pdf", "html", "epub", or "site".
	Name() string

	// Generate writes output using the provided render request.
	Generate(ctx context.Context, req *RenderRequest, outputPath string) error

	// Description returns a short human-readable description.
	Description() string
}

// RenderRequest contains all data needed to render any output format.
type RenderRequest struct {
	// FullHTML is the assembled full HTML document for PDF and standalone HTML output.
	FullHTML string

	// Chapters contains per-chapter content for ePub and site output.
	Chapters []ChapterContent

	// CSS is the merged theme CSS and custom CSS.
	CSS string

	// Meta contains document metadata.
	Meta DocumentMeta
}

// ChapterContent stores rendered content for a single chapter.
type ChapterContent struct {
	// Title is the chapter title.
	Title string
	// ID is the unique chapter identifier.
	ID string
	// HTML is the chapter HTML without the outer document shell.
	HTML string
	// Filename is the suggested output filename, for example "ch_001.xhtml".
	Filename string
}

// DocumentMeta stores document metadata.
type DocumentMeta struct {
	Title    string
	Author   string
	Language string
	Version  string
}

// Registry stores registered output formats.
type Registry struct {
	mu      sync.RWMutex
	formats map[string]OutputFormat
}

// NewRegistry creates an empty output format registry.
func NewRegistry() *Registry {
	return &Registry{
		formats: make(map[string]OutputFormat),
	}
}

// Register adds or replaces an output format implementation.
func (r *Registry) Register(f OutputFormat) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.formats[f.Name()] = f
}

// Get returns an output format by name.
func (r *Registry) Get(name string) (OutputFormat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.formats[name]
	if !ok {
		return nil, fmt.Errorf("unsupported output format: %s (available: %v)", name, r.List())
	}
	return f, nil
}

// List returns the names of all registered formats.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.formats))
	for name := range r.formats {
		names = append(names, name)
	}
	return names
}

// Has reports whether a format name is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.formats[name]
	return ok
}
