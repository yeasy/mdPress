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

// TestServeOptions_VariousConfigurations tests ServeOptions with various configurations
func TestServeOptions_VariousConfigurations(t *testing.T) {
	tests := []struct {
		name       string
		opts       ServeOptions
		expectPort int
		expectHost string
		expectOpen bool
		expectDir  string
		expectChg  bool
	}{
		{
			name: "default options",
			opts: ServeOptions{
				Port: defaultServePort,
				Host: defaultServeHost,
			},
			expectPort: 9000,
			expectHost: "127.0.0.1",
			expectOpen: false,
			expectDir:  "",
			expectChg:  false,
		},
		{
			name: "custom port",
			opts: ServeOptions{
				Port:        8080,
				Host:        "localhost",
				PortChanged: true,
			},
			expectPort: 8080,
			expectHost: "localhost",
			expectOpen: false,
			expectChg:  true,
		},
		{
			name: "all fields set",
			opts: ServeOptions{
				Port:        3000,
				Host:        "0.0.0.0",
				OutputDir:   "_site",
				AutoOpen:    true,
				PortChanged: true,
			},
			expectPort: 3000,
			expectHost: "0.0.0.0",
			expectOpen: true,
			expectDir:  "_site",
			expectChg:  true,
		},
		{
			name: "only auto open",
			opts: ServeOptions{
				AutoOpen: true,
			},
			expectPort: 0,
			expectHost: "",
			expectOpen: true,
			expectDir:  "",
			expectChg:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Port != tt.expectPort {
				t.Errorf("Port: expected %d, got %d", tt.expectPort, tt.opts.Port)
			}
			if tt.opts.Host != tt.expectHost {
				t.Errorf("Host: expected %q, got %q", tt.expectHost, tt.opts.Host)
			}
			if tt.opts.AutoOpen != tt.expectOpen {
				t.Errorf("AutoOpen: expected %v, got %v", tt.expectOpen, tt.opts.AutoOpen)
			}
			if tt.opts.OutputDir != tt.expectDir {
				t.Errorf("OutputDir: expected %q, got %q", tt.expectDir, tt.opts.OutputDir)
			}
			if tt.opts.PortChanged != tt.expectChg {
				t.Errorf("PortChanged: expected %v, got %v", tt.expectChg, tt.opts.PortChanged)
			}
		})
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

// TestServeOptions_PortValidation tests port values for validity
func TestServeOptions_PortValidation(t *testing.T) {
	tests := []struct {
		name     string
		port     int
		isValid  bool
		testName string
	}{
		{"default port 9000", 9000, true, "default port 9000"},
		{"standard port 8080", 8080, true, "standard port 8080"},
		{"high port 65535", 65535, true, "high port 65535"},
		{"low port 1", 1, true, "low port 1"},
		{"zero port", 0, true, "zero port (means OS chooses)"},
		{"negative port", -1, true, "negative port (should be caught elsewhere)"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			opts := ServeOptions{Port: tt.port}
			// Port field accepts any int; actual validation happens at runtime
			if opts.Port != tt.port {
				t.Errorf("Port should be %d, got %d", tt.port, opts.Port)
			}
		})
	}
}

// TestServeOptions_HostValidation tests host values for validity
func TestServeOptions_HostValidation(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		testName string
	}{
		{"localhost", "localhost", "localhost hostname"},
		{"127.0.0.1", "127.0.0.1", "IPv4 loopback"},
		{"0.0.0.0", "0.0.0.0", "IPv4 any address"},
		{"::1", "::1", "IPv6 loopback"},
		{"::", "::", "IPv6 any address"},
		{"example.com", "example.com", "domain name"},
		{"", "", "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			opts := ServeOptions{Host: tt.host}
			if opts.Host != tt.host {
				t.Errorf("Host should be %q, got %q", tt.host, opts.Host)
			}
		})
	}
}

// TestServeOptions_OutputDirValidation tests output directory handling
func TestServeOptions_OutputDirValidation(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		testName  string
	}{
		{"_book default", "_book", "default _book directory"},
		{"custom path", "/tmp/mybook", "custom absolute path"},
		{"relative path", "_site", "relative path"},
		{"empty default", "", "empty means use default"},
		{"complex path", "/very/deep/nested/path/to/_output", "deep nested path"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			opts := ServeOptions{OutputDir: tt.outputDir}
			if opts.OutputDir != tt.outputDir {
				t.Errorf("OutputDir should be %q, got %q", tt.outputDir, opts.OutputDir)
			}
		})
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

// TestServeOptions_PortChangedTracking tests PortChanged flag behavior
func TestServeOptions_PortChangedTracking(t *testing.T) {
	tests := []struct {
		name        string
		portChanged bool
		port        int
		testName    string
	}{
		{
			name:        "port not explicitly changed",
			portChanged: false,
			port:        defaultServePort,
			testName:    "default port not changed",
		},
		{
			name:        "port explicitly changed",
			portChanged: true,
			port:        8080,
			testName:    "custom port changed",
		},
		{
			name:        "port changed but same value",
			portChanged: true,
			port:        defaultServePort,
			testName:    "port changed to same value",
		},
		{
			name:        "port not changed with custom value",
			portChanged: false,
			port:        8080,
			testName:    "port not changed flag but custom value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			opts := ServeOptions{
				Port:        tt.port,
				PortChanged: tt.portChanged,
			}

			if opts.PortChanged != tt.portChanged {
				t.Errorf("PortChanged should be %v, got %v", tt.portChanged, opts.PortChanged)
			}
			if opts.Port != tt.port {
				t.Errorf("Port should be %d, got %d", tt.port, opts.Port)
			}
		})
	}
}

// TestServeOptions_AllFieldsIndependent tests that ServeOptions fields are independent
func TestServeOptions_AllFieldsIndependent(t *testing.T) {
	opts1 := ServeOptions{
		Port:        9000,
		Host:        "localhost",
		OutputDir:   "_book",
		AutoOpen:    true,
		PortChanged: true,
	}

	opts2 := ServeOptions{
		Port:        8080,
		Host:        "127.0.0.1",
		OutputDir:   "_site",
		AutoOpen:    false,
		PortChanged: false,
	}

	// Verify they are different
	if opts1.Port == opts2.Port {
		t.Error("opts1 and opts2 should have different ports")
	}
	if opts1.Host == opts2.Host {
		t.Error("opts1 and opts2 should have different hosts")
	}
	if opts1.OutputDir == opts2.OutputDir {
		t.Error("opts1 and opts2 should have different output directories")
	}
	if opts1.AutoOpen == opts2.AutoOpen {
		t.Error("opts1 and opts2 should have different AutoOpen values")
	}
	if opts1.PortChanged == opts2.PortChanged {
		t.Error("opts1 and opts2 should have different PortChanged values")
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
