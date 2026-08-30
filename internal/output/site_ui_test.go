package output

import "testing"

// TestSiteUIKoreanAndFrench: ko and fr books used to ship an English site UI
// with no signal; both are translated now, and anything still uncovered warns
// at build time (see uiLocalized).
func TestSiteUIKoreanAndFrench(t *testing.T) {
	if got := uiString("ko-KR", "search_button"); got != "검색" {
		t.Errorf("ko search_button = %q", got)
	}
	if got := uiString("fr-FR", "next"); got != "Suivant" {
		t.Errorf("fr next = %q", got)
	}
	// Every key in the English table must exist in ko and fr — a missing key
	// silently mixes English into an otherwise localized UI.
	for key := range uiStrings["en"] {
		for _, lang := range []string{"ko", "fr"} {
			if _, ok := uiStrings[lang][key]; !ok {
				t.Errorf("%s is missing key %q", lang, key)
			}
		}
	}
	for lang, want := range map[string]bool{"ko-KR": true, "fr": true, "zh-Hans": true, "de-DE": false, "": true, "en-US": true} {
		if got := uiLocalized(lang); got != want {
			t.Errorf("uiLocalized(%q) = %v, want %v", lang, got, want)
		}
	}
}
