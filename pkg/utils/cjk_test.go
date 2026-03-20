package utils

import "testing"

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
		{"CJK range start", 0x4E00, true},   // First CJK unified ideograph
		{"CJK range end", 0x9FFF, true},     // Last CJK unified ideograph
		{"Korean range", 0xAC00, true},      // First Korean hangul
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
		{"CJK fullwidth forms", "ＡＢＣ", false}, // These are not CJK ideographs
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
