package locale

import "strings"

// primaryDialects maps a language to its primary dialect. Transcribed from
// locales.json in github.com/citation-style-language/locales (commit
// 8bef426, 2026-10-20).
var primaryDialects = map[string]string{
	"af": "af-ZA", "ar": "ar", "bal": "bal-PK", "bg": "bg-BG", "brh": "brh-PK",
	"ca": "ca-AD", "cs": "cs-CZ", "cy": "cy-GB", "da": "da-DK", "de": "de-DE",
	"el": "el-GR", "en": "en-US", "es": "es-ES", "et": "et-EE", "eu": "eu",
	"fa": "fa-IR", "fi": "fi-FI", "fr": "fr-FR", "gl": "gl-ES", "he": "he-IL",
	"hi": "hi-IN", "hr": "hr-HR", "hu": "hu-HU", "hy": "hy-AM", "id": "id-ID",
	"is": "is-IS", "it": "it-IT", "ja": "ja-JP", "km": "km-KH", "ko": "ko-KR",
	"la": "la", "lij": "lij-IT", "lt": "lt-LT", "lv": "lv-LV", "mn": "mn-MN",
	"ms": "ms-MY", "nb": "nb-NO", "nl": "nl-NL", "nn": "nn-NO", "pa": "pa-PK",
	"pl": "pl-PL", "pt": "pt-PT", "ro": "ro-RO", "ru": "ru-RU", "sk": "sk-SK",
	"sl": "sl-SI", "sr": "sr-Latn-RS", "sv": "sv-SE", "th": "th-TH", "tl": "tl-PH",
	"tr": "tr-TR", "uk": "uk-UA", "vi": "vi-VN", "zh": "zh-CN",
}

// BaseLang returns the language subtag of a locale tag ("de" for "de-AT").
func BaseLang(tag string) string {
	base, _, _ := strings.Cut(tag, "-")
	return strings.ToLower(base)
}

// PrimaryDialect returns the primary dialect of tag's language ("de-DE" for
// "de-AT" or "de"), or "" if the language is unknown.
func PrimaryDialect(tag string) string {
	return primaryDialects[BaseLang(tag)]
}

// IsEnglish reports whether tag is an English locale. Title and sentence
// case only apply to English text.
func IsEnglish(tag string) bool { return BaseLang(tag) == "en" }

// sameTag compares locale tags case-insensitively.
func sameTag(a, b string) bool { return strings.EqualFold(a, b) }
