package cmd

import (
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

// TestServeCommand_PortFlagDefaults tests that the port flag has correct defaults
func TestServeCommand_PortFlagDefaults(t *testing.T) {
	flag := serveCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatal("port flag should exist")
	}

	if flag.DefValue != "9000" {
		t.Errorf("port default value should be '9000', got %q", flag.DefValue)
	}
}

// TestServeCommand_HostFlagDefaults tests that the host flag has correct defaults
func TestServeCommand_HostFlagDefaults(t *testing.T) {
	flag := serveCmd.Flags().Lookup("host")
	if flag == nil {
		t.Fatal("host flag should exist")
	}

	if flag.DefValue != "127.0.0.1" {
		t.Errorf("host default value should be '127.0.0.1', got %q", flag.DefValue)
	}
}

// TestServeCommand_OutputFlagDefaults tests that the output flag has correct defaults
func TestServeCommand_OutputFlagDefaults(t *testing.T) {
	flag := serveCmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("output flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("output default value should be empty, got %q", flag.DefValue)
	}
}

// TestServeCommand_OpenFlagDefaults tests that the open flag has correct defaults
func TestServeCommand_OpenFlagDefaults(t *testing.T) {
	flag := serveCmd.Flags().Lookup("open")
	if flag == nil {
		t.Fatal("open flag should exist")
	}

	if flag.DefValue != "false" {
		t.Errorf("open default value should be 'false', got %q", flag.DefValue)
	}
}

// TestServeCommand_SummaryFlagDefaults tests that the summary flag has correct defaults
func TestServeCommand_SummaryFlagDefaults(t *testing.T) {
	flag := serveCmd.Flags().Lookup("summary")
	if flag == nil {
		t.Fatal("summary flag should exist")
	}

	if flag.DefValue != "" {
		t.Errorf("summary default value should be empty, got %q", flag.DefValue)
	}
}

// TestServeOptions_Structure tests the ServeOptions struct fields
func TestServeOptions_Structure(t *testing.T) {
	opts := ServeOptions{
		Port:        9000,
		Host:        "127.0.0.1",
		OutputDir:   "/tmp/output",
		AutoOpen:    true,
		PortChanged: false,
	}

	if opts.Port != 9000 {
		t.Errorf("Port should be 9000, got %d", opts.Port)
	}

	if opts.Host != "127.0.0.1" {
		t.Errorf("Host should be 127.0.0.1, got %q", opts.Host)
	}

	if opts.OutputDir != "/tmp/output" {
		t.Errorf("OutputDir should be /tmp/output, got %q", opts.OutputDir)
	}

	if opts.AutoOpen != true {
		t.Error("AutoOpen should be true")
	}

	if opts.PortChanged != false {
		t.Error("PortChanged should be false")
	}
}

// TestDefaultServeConstants tests that serve command default constants are properly set
func TestDefaultServeConstants(t *testing.T) {
	if defaultServePort != 9000 {
		t.Errorf("defaultServePort should be 9000, got %d", defaultServePort)
	}

	if defaultServeHost != "127.0.0.1" {
		t.Errorf("defaultServeHost should be 127.0.0.1, got %q", defaultServeHost)
	}
}

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
		if !contains(serveCmd.Long, phrase) {
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

// TestServeOptions_ZeroValue tests ServeOptions with zero values
func TestServeOptions_ZeroValue(t *testing.T) {
	opts := ServeOptions{}

	if opts.Port != 0 {
		t.Errorf("Port zero value should be 0, got %d", opts.Port)
	}

	if opts.Host != "" {
		t.Errorf("Host zero value should be empty, got %q", opts.Host)
	}

	if opts.OutputDir != "" {
		t.Errorf("OutputDir zero value should be empty, got %q", opts.OutputDir)
	}

	if opts.AutoOpen != false {
		t.Error("AutoOpen zero value should be false")
	}

	if opts.PortChanged != false {
		t.Error("PortChanged zero value should be false")
	}
}

// Note: contains() helper is defined in build_manifest_test.go
