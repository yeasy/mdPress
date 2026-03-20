package utils

import (
	"strings"
	"testing"
	"unicode"
)

func TestContainsCJK(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"empty string", "", false},
		{"english only", "Hello World", false},
		{"numbers only", "12345", false},
		{"chinese characters", "你好世界", true},
		{"mixed english and chinese", "Hello 你好", true},
		{"japanese hiragana", "こんにちは", true},
		{"japanese katakana", "カタカナ", true},
		{"korean hangul", "안녕하세요", true},
		{"chinese in markdown", "# 第一章 简介", true},
		{"chinese punctuation only", "，。！", false}, // Punctuation is not CJK ideograph
		{"html with chinese", "<p>这是一段中文</p>", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCJK(tt.text)
			if got != tt.want {
				t.Errorf("ContainsCJK(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestContainsChinese(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"empty string", "", false},
		{"english only", "Hello World", false},
		{"chinese characters", "你好", true},
		{"japanese hiragana only", "こんにちは", false},
		{"korean hangul only", "안녕하세요", false},
		{"kanji (shared with chinese)", "漢字", true}, // Kanji uses Han unified ideographs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsChinese(tt.text)
			if got != tt.want {
				t.Errorf("ContainsChinese(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestCheckCJKFonts(t *testing.T) {
	// This is an environment-dependent test; just verify it doesn't panic.
	status := CheckCJKFonts()
	t.Logf("CJK fonts available: %v, fonts: %v", status.Available, status.Fonts)
}

func TestCJKFontInstallHint(t *testing.T) {
	hint := CJKFontInstallHint()
	if hint == "" {
		t.Error("CJKFontInstallHint() returned empty string")
	}
}

// TestCJKFontStatus tests CJKFontStatus structure
func TestCJKFontStatus(t *testing.T) {
	status := CheckCJKFonts()

	// Status should not be nil
	if len(status.Fonts) > 5 {
		t.Errorf("CheckCJKFonts() returned too many fonts: %d (max 5)", len(status.Fonts))
	}

	// If available is true, Fonts should have items (usually)
	if status.Available && len(status.Fonts) == 0 {
		t.Log("Available is true but no fonts listed (may be fallback behavior on some systems)")
	}

	// Windows should always return Available=true
	// (we can't test this directly without mocking, but the behavior is documented)
}

// TestIsCJKRune tests individual rune detection
func TestIsCJKRune(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{"Chinese ideograph", '中', true},
		{"Japanese hiragana", 'あ', true},
		{"Japanese katakana", 'ア', true},
		{"Korean hangul", '가', true},
		{"English letter", 'A', false},
		{"Arabic digit", '٥', false},
		{"Latin extended", 'é', false},
		{"Greek letter", 'α', false},
		{"CJK range start", 0x4E00, true},  // First CJK unified ideograph
		{"CJK range end", 0x9FFF, true},    // Last CJK unified ideograph
		{"Korean range", 0xAC00, true},     // First Korean hangul
		{"Japanese hiragana あ", 'あ', true}, // Actual hiragana character
		{"Japanese katakana ア", 'ア', true}, // Actual katakana character
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCJKRune(tt.r)
			if got != tt.want {
				t.Errorf("isCJKRune(%U) = %v, want %v", tt.r, got, tt.want)
			}
		})
	}
}

// TestContainsCJKEdgeCases tests edge cases for CJK detection
func TestContainsCJKEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"Single Chinese character", "中", true},
		{"Chinese at start", "中文hello", true},
		{"Chinese in middle", "hello中文world", true},
		{"Chinese at end", "hello中文", true},
		{"Multiple CJK types mixed", "中あア가", true},
		{"Only whitespace", "   \n\t  ", false},
		{"Mixed with numbers", "123 中 456", true},
		{"Mixed with punctuation", "中...world", true},
		{"CJK fullwidth forms", "ＡＢＣ", false},        // These are not CJK ideographs
		{"CJK compatibility ideographs", "㐀㐁", true}, // CJK compatibility ideographs
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCJK(tt.text)
			if got != tt.want {
				t.Errorf("ContainsCJK(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// TestContainsChineseEdgeCases tests edge cases for Chinese detection
func TestContainsChineseEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"Simplified Chinese", "简体中文", true},
		{"Traditional Chinese", "繁體中文", true},
		{"Mixed with English", "English 中文", true},
		{"Pure English", "English only", false},
		{"Pure Japanese", "ひらがなカタカナ", false},
		{"Pure Korean", "한글만", false},
		{"Kanji (Han ideographs)", "漢字", true}, // These are Han ideographs used in Japanese
		{"Rare Han characters", "𠀋𠀌", true},    // Rare CJK characters
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsChinese(tt.text)
			if got != tt.want {
				t.Errorf("ContainsChinese(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// TestCJKCharacterRanges tests CJK detection across different Unicode ranges
func TestCJKCharacterRanges(t *testing.T) {
	tests := []struct {
		name      string
		rune      rune
		isCJK     bool
		isChinese bool
	}{
		// CJK Unified Ideographs (Chinese primary range)
		{"CJK U+4E00 (一)", 0x4E00, true, true},
		{"CJK U+6C49 (浩)", 0x6C49, true, true},
		{"CJK U+9FFF (max unified)", 0x9FFF, true, true},

		// CJK Unified Ideographs Extension A
		{"CJK Ext-A U+3400", 0x3400, true, true},
		{"CJK Ext-A U+4DB5", 0x4DB5, true, true},

		// Japanese Hiragana
		{"Hiragana U+3042 (あ)", 0x3042, true, false},
		{"Hiragana U+309F", 0x309F, true, false},

		// Japanese Katakana
		{"Katakana U+30A2 (ア)", 0x30A2, true, false},
		{"Katakana U+30FF", 0x30FF, true, false},

		// Korean Hangul
		{"Hangul U+AC00 (가)", 0xAC00, true, false},
		{"Hangul U+D7A3", 0xD7A3, true, false},

		// Non-CJK
		{"Latin A", 'A', false, false},
		{"Cyrillic А", 'А', false, false},
		{"Arabic ع", 'ع', false, false},
		{"Greek α", 'α', false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCJKRune(tt.rune)
			if got != tt.isCJK {
				t.Errorf("isCJKRune(U+%04X) = %v, want %v", tt.rune, got, tt.isCJK)
			}

			got2 := unicode.Is(unicode.Han, tt.rune)
			if got2 != tt.isChinese {
				t.Errorf("unicode.Is(Han, U+%04X) = %v, want %v", tt.rune, got2, tt.isChinese)
			}
		})
	}
}

// TestContainsCJKMixedText tests CJK detection with mixed language text
func TestContainsCJKMixedText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"Chinese + English", "Hello 你好 World", true},
		{"Japanese + English", "Hello こんにちは World", true},
		{"Korean + English", "Hello 안녕 World", true},
		{"All three CJK types", "中文 ひらがな 한글", true},
		{"CJK with numbers", "2024年第一章", true},
		{"CJK with punctuation", "「这是一个例子」", true},
		{"English with CJK punctuation", "Hello、World", false}, // Only punctuation, no ideographs
		{"Emoji and CJK", "😀 你好", true},
		{"URL with CJK", "https://example.com/中文/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCJK(tt.text)
			if got != tt.want {
				t.Errorf("ContainsCJK(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// TestContainsCJKLargeStrings tests CJK detection on large text
func TestContainsCJKLargeStrings(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"CJK at start", "中文" + strings.Repeat("English text ", 1000), true},
		{"CJK at end", strings.Repeat("English text ", 1000) + "中文", true},
		{"CJK in middle", strings.Repeat("English text ", 500) + "中文" + strings.Repeat("English text ", 500), true},
		{"Large text no CJK", strings.Repeat("English text ", 2000), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCJK(tt.text)
			if got != tt.want {
				t.Errorf("ContainsCJK(large string) = %v, want %v", got, tt.want)
			}
		})
	}
}
