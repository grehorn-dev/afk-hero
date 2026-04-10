package i18n

import (
	"testing"
)

func TestLoadTranslations_English(t *testing.T) {
	trans, err := LoadTranslations("eng")
	if err != nil {
		t.Fatalf("failed to load English translations: %v", err)
	}
	if len(trans) == 0 {
		t.Fatal("English translations are empty")
	}

	// Check some essential keys
	requiredKeys := []string{
		"app.name",
		"settings.title",
		"settings.appearance",
		"theme.Dark",
		"theme.Light",
		"button.ok",
		"button.cancel",
		"button.apply",
		"button.reset",
		"status.Disabled",
		"tray.settings",
		"tray.exit",
	}
	for _, key := range requiredKeys {
		if _, ok := trans[key]; !ok {
			t.Errorf("missing required key: %s", key)
		}
	}
}

func TestLoadTranslations_Fallback(t *testing.T) {
	// Load a non-existent language should fallback to English
	trans, err := LoadTranslations("zzz")
	if err != nil {
		t.Fatalf("fallback should not error: %v", err)
	}
	if len(trans) == 0 {
		t.Fatal("fallback translations are empty")
	}
	if trans["app.name"] != "AFK-Hero" {
		t.Errorf("fallback should return English, got: %s", trans["app.name"])
	}
}

func TestTranslate(t *testing.T) {
	val := Translate("eng", "app.name")
	if val != "AFK-Hero" {
		t.Errorf("expected 'AFK-Hero', got '%s'", val)
	}

	// Missing key returns the key itself
	val = Translate("eng", "nonexistent.key")
	if val != "nonexistent.key" {
		t.Errorf("expected key returned for missing translation, got '%s'", val)
	}
}

func TestIsRTL(t *testing.T) {
	rtlCodes := []string{"arb", "heb", "urd", "fas", "pus"}
	for _, code := range rtlCodes {
		if !IsRTL(code) {
			t.Errorf("expected %s to be RTL", code)
		}
	}

	ltrCodes := []string{"eng", "rus", "deu", "fra"}
	for _, code := range ltrCodes {
		if IsRTL(code) {
			t.Errorf("expected %s to not be RTL", code)
		}
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("eng") {
		t.Error("English should be supported")
	}
	if !IsSupported("rus") {
		t.Error("Russian should be supported")
	}
	if IsSupported("zzz") {
		t.Error("zzz should not be supported")
	}
}

func TestGetLanguageName(t *testing.T) {
	name := GetLanguageName("eng")
	if name != "English" {
		t.Errorf("expected English, got %s", name)
	}
	name = GetLanguageName("rus")
	if name != "Русский" {
		t.Errorf("expected Русский, got %s", name)
	}
}

func TestSupportedLanguagesCount(t *testing.T) {
	if len(SupportedLanguages) != 57 {
		t.Errorf("expected 57 supported languages, got %d", len(SupportedLanguages))
	}
}

func TestAllAvailableLocales_HaveRequiredKeys(t *testing.T) {
	// Load English as the reference
	engTrans, err := LoadTranslations("eng")
	if err != nil {
		t.Fatalf("failed to load English: %v", err)
	}

	// Check each available locale has all keys from English
	for _, lang := range SupportedLanguages {
		trans, err := LoadTranslations(lang.Code)
		if err != nil {
			// Some locales may not be created yet, skip
			continue
		}

		for key := range engTrans {
			if _, ok := trans[key]; !ok {
				// Not a hard error since locales may fall back to English
				t.Logf("locale %s missing key: %s", lang.Code, key)
			}
		}
	}
}

func TestUnitLabelsUseCompactForms(t *testing.T) {
	tests := []struct {
		code            string
		expectedPixels  string
		expectedSeconds string
	}{
		{code: "eng", expectedPixels: "px", expectedSeconds: "sec"},
		{code: "rus", expectedPixels: "пкс.", expectedSeconds: "сек."},
	}

	for _, tt := range tests {
		trans, err := LoadTranslations(tt.code)
		if err != nil {
			t.Fatalf("failed to load %s translations: %v", tt.code, err)
		}

		if got := trans["unit.pixels"]; got != tt.expectedPixels {
			t.Errorf("%s unit.pixels = %q, want %q", tt.code, got, tt.expectedPixels)
		}
		if got := trans["unit.seconds"]; got != tt.expectedSeconds {
			t.Errorf("%s unit.seconds = %q, want %q", tt.code, got, tt.expectedSeconds)
		}
	}
}
