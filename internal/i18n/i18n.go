package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

// Language represents a supported language.
type Language struct {
	Code string `json:"code"` // ISO 639-3 code
	Name string `json:"name"` // Native display name
}

// SupportedLanguages lists all supported languages in display order.
var SupportedLanguages = []Language{
	{Code: "eng", Name: "English"},
	{Code: "cmn", Name: "官话"},
	{Code: "hin", Name: "हिन्दी"},
	{Code: "spa", Name: "Español"},
	{Code: "arb", Name: "العربية"},
	{Code: "fra", Name: "Français"},
	{Code: "ben", Name: "বাংলা"},
	{Code: "por", Name: "Português"},
	{Code: "ind", Name: "Bahasa Indonesia"},
	{Code: "urd", Name: "اردو"},
	{Code: "rus", Name: "Русский"},
	{Code: "swh", Name: "Kiswahili"},
	{Code: "deu", Name: "Deutsch"},
	{Code: "fas", Name: "فارسی"},
	{Code: "jpn", Name: "日本語"},
	{Code: "vie", Name: "Tiếng Việt"},
	{Code: "hau", Name: "Harshen Hausa"},
	{Code: "tur", Name: "Türkçe"},
	{Code: "kor", Name: "한국어"},
	{Code: "ita", Name: "Italiano"},
	{Code: "amh", Name: "አማርኛ"},
	{Code: "yor", Name: "Èdè Yorùbá"},
	{Code: "pus", Name: "پښتو"},
	{Code: "pol", Name: "Polski"},
	{Code: "ukr", Name: "Українська"},
	{Code: "nld", Name: "Nederlands"},
	{Code: "ron", Name: "Română"},
	{Code: "ell", Name: "Ελληνικά"},
	{Code: "hun", Name: "Magyar"},
	{Code: "ces", Name: "Čeština"},
	{Code: "swe", Name: "Svenska"},
	{Code: "heb", Name: "עברית"},
	{Code: "bul", Name: "Български"},
	{Code: "sqi", Name: "Shqip"},
	{Code: "tat", Name: "Tatarça"},
	{Code: "srp", Name: "Српски"},
	{Code: "afr", Name: "Afrikaans"},
	{Code: "hye", Name: "Հայերեն"},
	{Code: "bel", Name: "Беларуская"},
	{Code: "dan", Name: "Dansk"},
	{Code: "hrv", Name: "Hrvatski"},
	{Code: "fin", Name: "Suomi"},
	{Code: "slk", Name: "Slovenčina"},
	{Code: "nor", Name: "Norsk"},
	{Code: "kat", Name: "ქართული"},
	{Code: "mkd", Name: "Македонски"},
	{Code: "lit", Name: "Lietuvių"},
	{Code: "bos", Name: "Bosanski"},
	{Code: "slv", Name: "Slovenščina"},
	{Code: "gle", Name: "Gaeilge"},
	{Code: "lav", Name: "Latviešu"},
	{Code: "est", Name: "Eesti"},
	{Code: "mlt", Name: "Malti"},
	{Code: "crh", Name: "Qırımtatarca"},
	{Code: "ltz", Name: "Lëtzebuergesch"},
	{Code: "isl", Name: "Íslenska"},
	{Code: "cnr", Name: "Crnogorski"},
}

// RTLLanguages contains codes of right-to-left languages.
var RTLLanguages = map[string]bool{
	"arb": true,
	"heb": true,
	"urd": true,
	"fas": true,
	"pus": true,
}

var (
	cache    = make(map[string]map[string]string)
	cacheMu  sync.RWMutex
	fallback map[string]string
)

// LoadTranslations loads translations for the given language code.
func LoadTranslations(code string) (map[string]string, error) {
	cacheMu.RLock()
	if t, ok := cache[code]; ok {
		cacheMu.RUnlock()
		return t, nil
	}
	cacheMu.RUnlock()

	data, err := localesFS.ReadFile("locales/" + code + ".json")
	if err != nil {
		// Try English fallback
		if code != "eng" {
			return LoadTranslations("eng")
		}
		return nil, fmt.Errorf("load locale %s: %w", code, err)
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return nil, fmt.Errorf("parse locale %s: %w", code, err)
	}

	cacheMu.Lock()
	cache[code] = translations
	if code == "eng" {
		fallback = translations
	}
	cacheMu.Unlock()

	return translations, nil
}

// Translate returns the translated string for the given key and language.
// Falls back to English if the key is missing in the requested language.
func Translate(code, key string) string {
	translations, err := LoadTranslations(code)
	if err != nil {
		if fallback != nil {
			if v, ok := fallback[key]; ok {
				return v
			}
		}
		return key
	}

	if v, ok := translations[key]; ok {
		return v
	}

	// Fallback to English
	if fallback != nil {
		if v, ok := fallback[key]; ok {
			return v
		}
	}

	return key
}

// IsRTL checks if the given language code is right-to-left.
func IsRTL(code string) bool {
	return RTLLanguages[code]
}

// GetLanguageName returns the native name for the given code.
func GetLanguageName(code string) string {
	for _, lang := range SupportedLanguages {
		if lang.Code == code {
			return lang.Name
		}
	}
	return code
}

// IsSupported checks if a language code is supported.
func IsSupported(code string) bool {
	for _, lang := range SupportedLanguages {
		if lang.Code == code {
			return true
		}
	}
	return false
}
