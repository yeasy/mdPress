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
