package utils

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// fcListTimeout is the maximum time to wait for the fc-list command.
const fcListTimeout = 10 * time.Second

// ContainsCJK reports whether the text contains any CJK (Chinese, Japanese, Korean) characters.
func ContainsCJK(text string) bool {
	for _, r := range text {
		if IsCJKRune(r) {
			return true
		}
	}
	return false
}

// IsCJKRune reports whether the rune is a CJK character.
func IsCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) || // CJK Unified Ideographs (Chinese)
		unicode.Is(unicode.Hangul, r) || // Korean
		unicode.Is(unicode.Hiragana, r) || // Japanese Hiragana
		unicode.Is(unicode.Katakana, r) // Japanese Katakana
}

// CJKFontStatus describes the availability of CJK fonts on the system.
type CJKFontStatus struct {
	Available bool     // Whether any CJK font is installed.
	Fonts     []string // Names of detected CJK fonts (up to 5).
}

func newCJKFontStatus(available bool, fonts []string) CJKFontStatus {
	if fonts == nil {
		fonts = []string{}
	}
	return CJKFontStatus{
		Available: available,
		Fonts:     fonts,
	}
}

// CheckCJKFonts checks whether CJK fonts are installed on the system.
// It uses platform-specific methods: fc-list on Linux, system_profiler on macOS.
func CheckCJKFonts() CJKFontStatus {
	switch runtime.GOOS {
	case "linux":
		return checkCJKFontsLinux()
	case "darwin":
		return checkCJKFontsMacOS()
	default:
		// On Windows and other platforms, assume fonts are available
		// since most Windows installations include CJK fonts.
		return newCJKFontStatus(true, []string{"(system default)"})
	}
}

func checkCJKFontsLinux() CJKFontStatus {
	// Use fc-list to query for CJK fonts.
	ctx, cancel := context.WithTimeout(context.Background(), fcListTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "fc-list", ":lang=zh").Output()
	if err != nil {
		// fc-list not available; try alternative check.
		return checkCJKFontsFallback()
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var fonts []string
	seen := make(map[string]bool)
	for _, line := range lines {
		if line == "" {
			continue
		}
		// fc-list output format: /path/to/font.ttf: Font Name:style=Regular
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[1])
			if name != "" && !seen[name] {
				seen[name] = true
				fonts = append(fonts, name)
				if len(fonts) >= 5 {
					break
				}
			}
		}
	}
	return CJKFontStatus{
		Available: len(fonts) > 0,
		Fonts:     append([]string{}, fonts...),
	}
}

func checkCJKFontsMacOS() CJKFontStatus {
	// macOS ships with CJK fonts (PingFang SC, Hiragino Sans GB, etc.)
	// by default, so they're almost always available.
	// Do a quick check with fc-list if available, otherwise assume present.
	ctx, cancel := context.WithTimeout(context.Background(), fcListTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "fc-list", ":lang=zh").Output()
	if err != nil {
		// fc-list not installed on macOS is common; assume fonts are available
		// since macOS bundles PingFang SC etc.
		return newCJKFontStatus(true, []string{"PingFang SC (system)"})
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return newCJKFontStatus(true, []string{"(system CJK fonts detected)"})
	}
	return newCJKFontStatus(false, nil)
}

func checkCJKFontsFallback() CJKFontStatus {
	// Try common CJK font paths on Linux.
	commonPaths := []string{
		"/usr/share/fonts/noto-cjk",
		"/usr/share/fonts/truetype/noto",
		"/usr/share/fonts/opentype/noto",
		"/usr/share/fonts/google-noto-cjk",
		"/usr/share/fonts/google-noto-cjk-fonts",
		"/usr/share/fonts/wqy-zenhei",
		"/usr/share/fonts/wqy-microhei",
	}
	for _, p := range commonPaths {
		if FileExists(p) {
			return newCJKFontStatus(true, []string{p})
		}
	}
	return newCJKFontStatus(false, nil)
}

// CJKFontInstallHint returns platform-specific instructions for installing CJK fonts.
func CJKFontInstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "Install CJK fonts for proper PDF rendering:\n" +
			"  Ubuntu/Debian: sudo apt install fonts-noto-cjk\n" +
			"  Fedora/RHEL:   sudo dnf install google-noto-sans-cjk-fonts\n" +
			"  Arch Linux:    sudo pacman -S noto-fonts-cjk\n" +
			"  Alpine:        apk add font-noto-cjk\n" +
			"  Or use Docker:  docker run --rm -v .:/book ghcr.io/yeasy/mdpress:full build --format pdf"
	case "darwin":
		return "macOS includes CJK fonts by default (PingFang SC, Hiragino Sans GB).\n" +
			"If fonts are missing, install: brew install font-noto-sans-cjk"
	default:
		return "Install Noto Sans CJK or similar CJK fonts for proper PDF rendering."
	}
}
