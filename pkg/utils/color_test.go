package utils

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestSetColorEnabled verifies that color output can be enabled/disabled.
func TestSetColorEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enable colors",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disable colors",
			enabled:  false,
			expected: false,
		},
	}

	// Save original state
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.enabled)
			if IsColorEnabled() != tt.expected {
				t.Errorf("IsColorEnabled() = %v, want %v", IsColorEnabled(), tt.expected)
			}
		})
	}
}

// TestIsColorEnabled verifies the color detection state can be read.
func TestIsColorEnabled(t *testing.T) {
	// Save original state
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	SetColorEnabled(true)
	if !IsColorEnabled() {
		t.Error("IsColorEnabled() returned false when color is enabled")
	}

	SetColorEnabled(false)
	if IsColorEnabled() {
		t.Error("IsColorEnabled() returned true when color is disabled")
	}
}

// TestBold returns bold text when colors are enabled, plain text when disabled.
func TestBold(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name           string
		input          string
		colorEnabled   bool
		shouldContain  string
		shouldNotEqual string
	}{
		{
			name:          "bold with colors",
			input:         "test",
			colorEnabled:  true,
			shouldContain: "test",
		},
		{
			name:           "bold without colors",
			input:          "test",
			colorEnabled:   false,
			shouldNotEqual: "\033[1m",
		},
		{
			name:          "bold empty string",
			input:         "",
			colorEnabled:  true,
			shouldContain: "",
		},
		{
			name:          "bold unicode",
			input:         "café",
			colorEnabled:  true,
			shouldContain: "café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Bold(tt.input)

			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Bold(%q) doesn't contain %q, got %q", tt.input, tt.shouldContain, result)
			}

			if tt.colorEnabled && !strings.Contains(result, colorBold) {
				t.Errorf("Bold(%q) should contain color code when enabled, got %q", tt.input, result)
			}

			if !tt.colorEnabled && strings.Contains(result, colorBold) {
				t.Errorf("Bold(%q) should not contain color code when disabled, got %q", tt.input, result)
			}
		})
	}
}

// TestDim returns dimmed text when colors are enabled, plain text when disabled.
func TestDim(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "dim with colors",
			input:        "test",
			colorEnabled: true,
		},
		{
			name:         "dim without colors",
			input:        "test",
			colorEnabled: false,
		},
		{
			name:         "dim empty string",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "dim unicode",
			input:        "test 你好",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Dim(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Dim(%q) doesn't contain input, got %q", tt.input, result)
			}

			if tt.colorEnabled && !strings.Contains(result, colorDim) {
				t.Errorf("Dim(%q) should contain dim code when enabled, got %q", tt.input, result)
			}

			if !tt.colorEnabled && strings.Contains(result, colorDim) {
				t.Errorf("Dim(%q) should not contain dim code when disabled, got %q", tt.input, result)
			}
		})
	}
}

// TestGreen returns green text when colors are enabled, plain text when disabled.
func TestGreen(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "green with colors",
			input:        "success",
			colorEnabled: true,
		},
		{
			name:         "green without colors",
			input:        "success",
			colorEnabled: false,
		},
		{
			name:         "green empty",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "green with special chars",
			input:        "✓ done",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Green(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Green(%q) doesn't contain input, got %q", tt.input, result)
			}

			if tt.colorEnabled && !strings.Contains(result, colorGreen) {
				t.Errorf("Green(%q) should contain green code when enabled", tt.input)
			}

			if !tt.colorEnabled && strings.Contains(result, colorGreen) {
				t.Errorf("Green(%q) should not contain green code when disabled", tt.input)
			}
		})
	}
}

// TestRed returns red text when colors are enabled, plain text when disabled.
func TestRed(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "red with colors",
			input:        "error",
			colorEnabled: true,
		},
		{
			name:         "red without colors",
			input:        "error",
			colorEnabled: false,
		},
		{
			name:         "red empty",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "red with emojis",
			input:        "✗ failed",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Red(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Red(%q) doesn't contain input, got %q", tt.input, result)
			}

			if tt.colorEnabled && !strings.Contains(result, colorRed) {
				t.Errorf("Red(%q) should contain red code when enabled", tt.input)
			}

			if !tt.colorEnabled && strings.Contains(result, colorRed) {
				t.Errorf("Red(%q) should not contain red code when disabled", tt.input)
			}
		})
	}
}

// TestYellow returns yellow text when colors are enabled, plain text when disabled.
func TestYellow(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "yellow with colors",
			input:        "warning",
			colorEnabled: true,
		},
		{
			name:         "yellow without colors",
			input:        "warning",
			colorEnabled: false,
		},
		{
			name:         "yellow empty",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "yellow long text",
			input:        "this is a warning message with multiple words",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Yellow(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Yellow(%q) doesn't contain input", tt.input)
			}

			if tt.colorEnabled && !strings.Contains(result, colorYellow) {
				t.Errorf("Yellow(%q) should contain yellow code when enabled", tt.input)
			}

			if !tt.colorEnabled && strings.Contains(result, colorYellow) {
				t.Errorf("Yellow(%q) should not contain yellow code when disabled", tt.input)
			}
		})
	}
}

// TestBlue returns blue text when colors are enabled, plain text when disabled.
func TestBlue(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "blue with colors",
			input:        "info",
			colorEnabled: true,
		},
		{
			name:         "blue without colors",
			input:        "info",
			colorEnabled: false,
		},
		{
			name:         "blue empty",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "blue with numbers",
			input:        "version 1.2.3",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Blue(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Blue(%q) doesn't contain input", tt.input)
			}

			if tt.colorEnabled && !strings.Contains(result, colorBlue) {
				t.Errorf("Blue(%q) should contain blue code when enabled", tt.input)
			}

			if !tt.colorEnabled && strings.Contains(result, colorBlue) {
				t.Errorf("Blue(%q) should not contain blue code when disabled", tt.input)
			}
		})
	}
}

// TestCyan returns cyan text when colors are enabled, plain text when disabled.
func TestCyan(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		input        string
		colorEnabled bool
	}{
		{
			name:         "cyan with colors",
			input:        "highlight",
			colorEnabled: true,
		},
		{
			name:         "cyan without colors",
			input:        "highlight",
			colorEnabled: false,
		},
		{
			name:         "cyan empty",
			input:        "",
			colorEnabled: true,
		},
		{
			name:         "cyan unicode",
			input:        "日本語",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorEnabled(tt.colorEnabled)
			result := Cyan(tt.input)

			if !strings.Contains(result, tt.input) {
				t.Errorf("Cyan(%q) doesn't contain input", tt.input)
			}

			if tt.colorEnabled && !strings.Contains(result, colorCyan) {
				t.Errorf("Cyan(%q) should contain cyan code when enabled", tt.input)
			}

			if !tt.colorEnabled && strings.Contains(result, colorCyan) {
				t.Errorf("Cyan(%q) should not contain cyan code when disabled", tt.input)
			}
		})
	}
}

// TestSuccess writes a green success message to stdout.
func TestSuccess(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	// Capture stdout
	oldStdout := os.Stdout

	defer func() { os.Stdout = oldStdout }()

	tests := []struct {
		name         string
		format       string
		args         []any
		colorEnabled bool
		expectPrefix string
	}{
		{
			name:         "success basic",
			format:       "test passed",
			args:         []any{},
			colorEnabled: true,
			expectPrefix: "✓",
		},
		{
			name:         "success with args",
			format:       "test %d passed",
			args:         []any{5},
			colorEnabled: true,
			expectPrefix: "✓",
		},
		{
			name:         "success no color",
			format:       "test passed",
			args:         []any{},
			colorEnabled: false,
			expectPrefix: "✓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need to recapture stdout for each test
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w

			SetColorEnabled(tt.colorEnabled)
			Success(tt.format, tt.args...)

			w.Close()
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stdout = oldStdout

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectPrefix) {
				t.Errorf("Success output missing prefix %q, got: %q", tt.expectPrefix, outputStr)
			}
		})
	}
}

// TestWarning writes a yellow warning message to stdout.
func TestWarning(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	// Capture stdout
	oldStdout := os.Stdout

	defer func() { os.Stdout = oldStdout }()

	tests := []struct {
		name         string
		format       string
		args         []any
		colorEnabled bool
		expectPrefix string
	}{
		{
			name:         "warning basic",
			format:       "be careful",
			args:         []any{},
			colorEnabled: true,
			expectPrefix: "⚠",
		},
		{
			name:         "warning with args",
			format:       "deprecated: %s",
			args:         []any{"feature"},
			colorEnabled: true,
			expectPrefix: "⚠",
		},
		{
			name:         "warning no color",
			format:       "be careful",
			args:         []any{},
			colorEnabled: false,
			expectPrefix: "⚠",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w

			SetColorEnabled(tt.colorEnabled)
			Warning(tt.format, tt.args...)

			w.Close()
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stdout = oldStdout

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectPrefix) {
				t.Errorf("Warning output missing prefix %q, got: %q", tt.expectPrefix, outputStr)
			}
		})
	}
}

// TestError writes a red error message to stderr.
func TestError(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		format       string
		args         []any
		colorEnabled bool
		expectPrefix string
	}{
		{
			name:         "error basic",
			format:       "something went wrong",
			args:         []any{},
			colorEnabled: true,
			expectPrefix: "✗",
		},
		{
			name:         "error with args",
			format:       "error code: %d",
			args:         []any{500},
			colorEnabled: true,
			expectPrefix: "✗",
		},
		{
			name:         "error no color",
			format:       "something went wrong",
			args:         []any{},
			colorEnabled: false,
			expectPrefix: "✗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stderr = w

			SetColorEnabled(tt.colorEnabled)
			Error(tt.format, tt.args...)

			w.Close()
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stderr = oldStderr

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectPrefix) {
				t.Errorf("Error output missing prefix %q, got: %q", tt.expectPrefix, outputStr)
			}
		})
	}
}

// TestInfo writes a blue info message to stdout.
func TestInfo(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	// Capture stdout
	oldStdout := os.Stdout

	defer func() { os.Stdout = oldStdout }()

	tests := []struct {
		name         string
		format       string
		args         []any
		colorEnabled bool
		expectPrefix string
	}{
		{
			name:         "info basic",
			format:       "here's some info",
			args:         []any{},
			colorEnabled: true,
			expectPrefix: "ℹ",
		},
		{
			name:         "info with args",
			format:       "processing %d items",
			args:         []any{42},
			colorEnabled: true,
			expectPrefix: "ℹ",
		},
		{
			name:         "info no color",
			format:       "here's some info",
			args:         []any{},
			colorEnabled: false,
			expectPrefix: "ℹ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w

			SetColorEnabled(tt.colorEnabled)
			Info(tt.format, tt.args...)

			w.Close()
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stdout = oldStdout

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectPrefix) {
				t.Errorf("Info output missing prefix %q, got: %q", tt.expectPrefix, outputStr)
			}
		})
	}
}

// TestHeader prints a titled separator with proper formatting.
func TestHeader(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	tests := []struct {
		name         string
		title        string
		colorEnabled bool
	}{
		{
			name:         "header with colors",
			title:        "Test Section",
			colorEnabled: true,
		},
		{
			name:         "header without colors",
			title:        "Test Section",
			colorEnabled: false,
		},
		{
			name:         "header empty title",
			title:        "",
			colorEnabled: true,
		},
		{
			name:         "header long title",
			title:        "This is a very long header title that should still work correctly",
			colorEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w

			SetColorEnabled(tt.colorEnabled)
			Header(tt.title)

			w.Close()
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read pipe: %v", err)
			}
			os.Stdout = oldStdout

			outputStr := string(output)

			// Should contain title
			if !strings.Contains(outputStr, tt.title) {
				t.Errorf("Header output missing title %q, got: %q", tt.title, outputStr)
			}

			// Should contain separator line
			if !strings.Contains(outputStr, "─") {
				t.Errorf("Header output missing separator line, got: %q", outputStr)
			}

			// Verify color codes are present/absent based on colorEnabled
			if tt.colorEnabled {
				if !strings.Contains(outputStr, colorBold) && !strings.Contains(outputStr, colorCyan) {
					t.Errorf("Header should contain color codes when enabled, got: %q", outputStr)
				}
			}
		})
	}
}

// TestColorizeWithColorEnabled verifies colorize adds color codes when enabled.
func TestColorizeWithColorEnabled(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	SetColorEnabled(true)
	result := colorize(colorGreen, "test")

	if !strings.Contains(result, colorGreen) {
		t.Errorf("colorize should contain color code when enabled, got: %q", result)
	}

	if !strings.Contains(result, colorReset) {
		t.Errorf("colorize should contain reset code when enabled, got: %q", result)
	}

	if !strings.Contains(result, "test") {
		t.Errorf("colorize should contain original text, got: %q", result)
	}
}

// TestColorizeWithoutColorEnabled verifies colorize returns plain text when disabled.
func TestColorizeWithoutColorEnabled(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	SetColorEnabled(false)
	result := colorize(colorGreen, "test")

	if result != "test" {
		t.Errorf("colorize should return plain text when disabled, got: %q", result)
	}

	if strings.Contains(result, colorGreen) {
		t.Errorf("colorize should not contain color code when disabled, got: %q", result)
	}
}

// TestColorizeEmptyString verifies colorize handles empty strings.
func TestColorizeEmptyString(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	SetColorEnabled(true)
	result := colorize(colorRed, "")

	// Should still have color codes even for empty text
	if !strings.Contains(result, colorRed) {
		t.Errorf("colorize should wrap empty string with color, got: %q", result)
	}
}

// TestMultipleColorApplications verifies that colors can be nested/combined.
func TestMultipleColorApplications(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	SetColorEnabled(true)

	// Apply green, then bold
	green := Green("text")
	bold := Bold(green)

	if !strings.Contains(bold, "text") {
		t.Error("Combined colors should preserve original text")
	}

	// Both should have their respective color codes when enabled
	if !strings.Contains(bold, colorBold) {
		t.Error("Combined colors should contain bold code")
	}
}

// TestColorFunctionConsistency verifies all color functions follow same pattern.
func TestColorFunctionConsistency(t *testing.T) {
	originalState := colorEnabled.Load()
	defer func() { colorEnabled.Store(originalState) }()

	colorFuncs := []struct {
		name string
		fn   func(string) string
	}{
		{"Green", Green},
		{"Red", Red},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Cyan", Cyan},
		{"Bold", Bold},
		{"Dim", Dim},
	}

	testText := "consistency test"

	for _, cf := range colorFuncs {
		t.Run(cf.name, func(t *testing.T) {
			// With colors
			SetColorEnabled(true)
			resultWithColor := cf.fn(testText)
			if !strings.Contains(resultWithColor, testText) {
				t.Errorf("%s should preserve text", cf.name)
			}

			// Without colors
			SetColorEnabled(false)
			resultWithoutColor := cf.fn(testText)
			if resultWithoutColor != testText {
				t.Errorf("%s should return plain text when colors disabled", cf.name)
			}
		})
	}
}

// TestColorCodesFormat verifies ANSI color code format.
func TestColorCodesFormat(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"reset", colorReset, "\033[0m"},
		{"red", colorRed, "\033[31m"},
		{"green", colorGreen, "\033[32m"},
		{"yellow", colorYellow, "\033[33m"},
		{"blue", colorBlue, "\033[34m"},
		{"cyan", colorCyan, "\033[36m"},
		{"bold", colorBold, "\033[1m"},
		{"dim", colorDim, "\033[2m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("Color code %s = %q, want %q", tt.name, tt.code, tt.expected)
			}
		})
	}
}

// TestDetectColorSupportNoColorEnv tests NO_COLOR environment variable.
func TestDetectColorSupportNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	result := detectColorSupport()
	if result {
		t.Error("detectColorSupport should return false when NO_COLOR is set")
	}
}

// TestDetectColorSupportDumbTerminal tests TERM=dumb.
func TestDetectColorSupportDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	result := detectColorSupport()
	if result {
		t.Error("detectColorSupport should return false for dumb terminal")
	}
}

// TestSuccessWithMultipleArgs tests Success with multiple format args.
func TestSuccessWithMultipleArgs(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	originalState := colorEnabled.Load()
	SetColorEnabled(true)
	defer func() { colorEnabled.Store(originalState); os.Stdout = oldStdout }()

	Success("test %s with %d args", "string", 42)

	w.Close()
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	outputStr := string(output)

	if !strings.Contains(outputStr, "test string with 42 args") {
		t.Errorf("Success formatting failed, got: %q", outputStr)
	}
}

// TestErrorWithMultipleArgs tests Error with multiple format args.
func TestErrorWithMultipleArgs(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	originalState := colorEnabled.Load()
	SetColorEnabled(true)
	defer func() { colorEnabled.Store(originalState); os.Stderr = oldStderr }()

	Error("error %d: %s", 500, "server error")

	w.Close()
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	outputStr := string(output)

	if !strings.Contains(outputStr, "error 500: server error") {
		t.Errorf("Error formatting failed, got: %q", outputStr)
	}
}

// BenchmarkGreen benchmarks the Green function.
func BenchmarkGreen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Green("benchmark text")
	}
}

// BenchmarkRed benchmarks the Red function.
func BenchmarkRed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Red("benchmark text")
	}
}

// BenchmarkColorize benchmarks the colorize function.
func BenchmarkColorize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		colorize(colorGreen, "benchmark text")
	}
}

// BenchmarkBold benchmarks the Bold function.
func BenchmarkBold(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Bold("benchmark text")
	}
}
