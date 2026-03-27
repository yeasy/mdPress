// color.go provides ANSI color output helpers.
// It exposes Success / Warning / Error / Info helpers and
// automatically falls back to plain text when color is unavailable.
package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
)

// ANSI color constants.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// colorEnabled caches terminal color support detection (atomic for thread safety).
var colorEnabled atomic.Bool

func init() {
	colorEnabled.Store(detectColorSupport())
}

// detectColorSupport reports whether the current terminal supports ANSI colors.
func detectColorSupport() bool {
	// Honor NO_COLOR when it is set.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}

	// Respect dumb terminals.
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Windows typically needs an ANSI-capable terminal.
	if runtime.GOOS == "windows" {
		// Windows Terminal and ConEmu support ANSI colors.
		if os.Getenv("WT_SESSION") != "" || os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		return false
	}

	// On Unix-like systems, color output requires a TTY stdout.
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// SetColorEnabled overrides color output for tests or forced modes.
func SetColorEnabled(enabled bool) {
	colorEnabled.Store(enabled)
}

// IsColorEnabled reports whether color output is enabled.
func IsColorEnabled() bool {
	return colorEnabled.Load()
}

// colorize wraps text with ANSI color codes when supported.
func colorize(color, text string) string {
	if !colorEnabled.Load() {
		return text
	}
	return color + text + colorReset
}

// Success prints a green success message with a checkmark prefix.
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(colorize(colorGreen, "  ✓ "+msg))
}

// Warning prints a yellow warning message with a warning prefix.
func Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(colorize(colorYellow, "  ⚠ "+msg))
}

// Error prints a red error message with an error prefix.
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(colorRed, "  ✗ "+msg))
}

// Info prints a blue informational message with an info prefix.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(colorize(colorBlue, "  ℹ "+msg))
}

// Bold returns bold text.
func Bold(text string) string {
	return colorize(colorBold, text)
}

// Dim returns dimmed text.
func Dim(text string) string {
	return colorize(colorDim, text)
}

// Green returns green text.
func Green(text string) string {
	return colorize(colorGreen, text)
}

// Red returns red text.
func Red(text string) string {
	return colorize(colorRed, text)
}

// Yellow returns yellow text.
func Yellow(text string) string {
	return colorize(colorYellow, text)
}

// Blue returns blue text.
func Blue(text string) string {
	return colorize(colorBlue, text)
}

// Cyan returns cyan text.
func Cyan(text string) string {
	return colorize(colorCyan, text)
}

// Header prints a titled separator.
func Header(title string) {
	width := 50
	line := strings.Repeat("─", width)
	fmt.Println()
	if colorEnabled.Load() {
		fmt.Println(colorBold + colorCyan + "  " + title + colorReset)
		fmt.Println(colorDim + "  " + line + colorReset)
	} else {
		fmt.Println("  " + title)
		fmt.Println("  " + line)
	}
}
