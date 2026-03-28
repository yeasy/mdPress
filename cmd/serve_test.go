package cmd

import (
	"strings"
	"testing"
)

// TestServeCommand_Creation tests that the serve command is properly created
func TestServeCommand_Creation(t *testing.T) {
	if serveCmd == nil {
		t.Fatal("serveCmd should not be nil")
	}

	if serveCmd.Use != "serve [source]" {
		t.Errorf("serveCmd.Use should be 'serve [source]', got %q", serveCmd.Use)
	}

	if serveCmd.Short != "Start the live preview server" {
		t.Errorf("serveCmd.Short should be 'Start the live preview server', got %q", serveCmd.Short)
	}

	if serveCmd.SilenceUsage != true {
		t.Error("serveCmd.SilenceUsage should be true")
	}

	if serveCmd.SilenceErrors != true {
		t.Error("serveCmd.SilenceErrors should be true")
	}
}

// TestServeCommand_FlagRegistration tests that all required flags are registered
func TestServeCommand_FlagRegistration(t *testing.T) {
	flags := []struct {
		name    string
		isValid bool
	}{
		{"port", true},
		{"host", true},
		{"output", true},
		{"open", true},
		{"summary", true},
	}

	for _, f := range flags {
		flag := serveCmd.Flags().Lookup(f.name)
		if f.isValid && flag == nil {
			t.Errorf("serve command should have --%s flag", f.name)
		}
	}
}

// TestServeCommand_FlagDefaults tests all flag defaults with table-driven test
func TestServeCommand_FlagDefaults(t *testing.T) {
	tests := []struct {
		name           string
		flagName       string
		expectedDefVal string
	}{
		{
			name:           "port flag default",
			flagName:       "port",
			expectedDefVal: "9000",
		},
		{
			name:           "host flag default",
			flagName:       "host",
			expectedDefVal: "127.0.0.1",
		},
		{
			name:           "output flag default",
			flagName:       "output",
			expectedDefVal: "",
		},
		{
			name:           "open flag default",
			flagName:       "open",
			expectedDefVal: "false",
		},
		{
			name:           "summary flag default",
			flagName:       "summary",
			expectedDefVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := serveCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag --%s should exist", tt.flagName)
			}

			if flag.DefValue != tt.expectedDefVal {
				t.Errorf("flag --%s default value should be %q, got %q", tt.flagName, tt.expectedDefVal, flag.DefValue)
			}
		})
	}
}

// TestDefaultServeConstants removed: testing compile-time constants provides
// no behavioral safety net (the test always passes unless the constant is
// intentionally changed, at which point the test must also change).

// TestServeCommand_LongDescription tests that serve command has comprehensive documentation
func TestServeCommand_LongDescription(t *testing.T) {
	if serveCmd.Long == "" {
		t.Error("serveCmd.Long should not be empty")
	}

	requiredPhrases := []string{
		"Build an HTML site",
		"HTTP server",
		"live reload",
	}

	for _, phrase := range requiredPhrases {
		if !strings.Contains(serveCmd.Long, phrase) {
			t.Errorf("serveCmd.Long should contain %q", phrase)
		}
	}
}

// TestServeCommand_HasRunE tests that serve command has a RunE function
func TestServeCommand_HasRunE(t *testing.T) {
	if serveCmd.RunE == nil {
		t.Fatal("serveCmd should have a RunE function")
	}
}

// TestServeCommand_ExamplesInLongDescription tests that serve command has usage examples
func TestServeCommand_ExamplesInLongDescription(t *testing.T) {
	if !strings.Contains(serveCmd.Long, "Examples:") {
		t.Error("serveCmd.Long should contain 'Examples:' section")
	}

	exampleKeywords := []string{
		"mdpress serve",
		"--port",
		"--host",
		"--open",
	}

	for _, keyword := range exampleKeywords {
		if !strings.Contains(serveCmd.Long, keyword) {
			t.Errorf("serveCmd.Long should contain example using %q", keyword)
		}
	}
}

// TestServeCommand_NoGlobalFlags tests that serve command respects silence flags
func TestServeCommand_NoGlobalFlags(t *testing.T) {
	if !serveCmd.SilenceUsage {
		t.Error("serveCmd.SilenceUsage should be true to hide usage on errors")
	}
	if !serveCmd.SilenceErrors {
		t.Error("serveCmd.SilenceErrors should be true to hide error messages")
	}
}

// TestServeCommand_FlagTypes tests that flags have correct types
func TestServeCommand_FlagTypes(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		expectedType string
	}{
		{"port is int", "port", "int"},
		{"host is string", "host", "string"},
		{"output is string", "output", "string"},
		{"open is bool", "open", "bool"},
		{"summary is string", "summary", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := serveCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("flag --%s should exist", tt.flagName)
			}

			flagType := flag.Value.Type()
			if flagType != tt.expectedType {
				t.Errorf("flag --%s should be type %s, got %s", tt.flagName, tt.expectedType, flagType)
			}
		})
	}
}
