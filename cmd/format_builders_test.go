package cmd

import (
	"testing"
)

func TestFormatBuilderRegistryRegistersDefaults(t *testing.T) {
	registry := newFormatBuilderRegistry()

	expectedFormats := []string{"pdf", "html", "site", "epub", "typst"}

	for _, format := range expectedFormats {
		if builder := registry.Get(format); builder == nil {
			t.Errorf("newFormatBuilderRegistry() missing default format: %s", format)
		}
	}
}

func TestFormatBuilderRegistryGet(t *testing.T) {
	registry := newFormatBuilderRegistry()

	builder := registry.Get("unknown_format")
	if builder != nil {
		t.Errorf("Get(\"unknown_format\") = %v, want nil", builder)
	}
}

func TestFormatBuilderNames(t *testing.T) {
	registry := newFormatBuilderRegistry()

	tests := []struct {
		format       string
		expectedName string
	}{
		{"pdf", "pdf"},
		{"html", "html"},
		{"site", "site"},
		{"epub", "epub"},
		{"typst", "typst"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			builder := registry.Get(tt.format)
			if builder == nil {
				t.Fatalf("Get(%q) returned nil", tt.format)
			}

			name := builder.Name()
			if name != tt.expectedName {
				t.Errorf("builder.Name() = %q, want %q", name, tt.expectedName)
			}
		})
	}
}

func TestFormatBuilderRegistryCustomBuilder(t *testing.T) {
	registry := newFormatBuilderRegistry()

	customBuilder := &mockFormatBuilder{name: "custom"}
	registry.Register(customBuilder)

	builder := registry.Get("custom")
	if builder == nil {
		t.Fatalf("Get(\"custom\") returned nil after registration")
	}

	if builder.Name() != "custom" {
		t.Errorf("custom builder Name() = %q, want %q", builder.Name(), "custom")
	}
}

// mockFormatBuilder is a mock implementation of formatBuilder for testing.
type mockFormatBuilder struct {
	name string
}

func (m *mockFormatBuilder) Name() string {
	return m.name
}

func (m *mockFormatBuilder) Build(ctx *buildContext, baseName string) error {
	return nil
}
