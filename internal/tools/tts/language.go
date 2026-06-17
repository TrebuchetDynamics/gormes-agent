//go:build !slim

package tts

import "strings"

func normalizeTTSLanguage(raw string) string {
	lang := strings.ToLower(strings.TrimSpace(raw))
	lang = strings.ReplaceAll(lang, "_", "-")
	return lang
}

func edgeTTSVoiceForLanguage(language, text string) string {
	lang := normalizeTTSLanguage(language)
	if lang == "" || lang == "auto" {
		lang = detectTTSLanguage(text)
	}
	if lang == "" {
		return ""
	}
	if voice, ok := edgeTTSLanguageVoices[lang]; ok {
		return voice
	}
	base, _, _ := strings.Cut(lang, "-")
	return edgeTTSLanguageVoices[base]
}

var edgeTTSLanguageVoices = map[string]string{
	"en":    "en-US-AriaNeural",
	"en-us": "en-US-AriaNeural",
	"es":    "es-ES-ElviraNeural",
	"es-es": "es-ES-ElviraNeural",
	"pt":    "pt-BR-FranciscaNeural",
	"pt-br": "pt-BR-FranciscaNeural",
	"fr":    "fr-FR-DeniseNeural",
	"fr-fr": "fr-FR-DeniseNeural",
	"de":    "de-DE-KatjaNeural",
	"de-de": "de-DE-KatjaNeural",
	"it":    "it-IT-ElsaNeural",
	"it-it": "it-IT-ElsaNeural",
	"ja":    "ja-JP-NanamiNeural",
	"ja-jp": "ja-JP-NanamiNeural",
	"ko":    "ko-KR-SunHiNeural",
	"ko-kr": "ko-KR-SunHiNeural",
	"zh":    "zh-CN-XiaoxiaoNeural",
	"zh-cn": "zh-CN-XiaoxiaoNeural",
}

func detectTTSLanguage(text string) string {
	lower := strings.ToLower(" " + strings.TrimSpace(text) + " ")
	if weightedTTSLanguageScore(lower, spanishTTSLanguageWeights) > weightedTTSLanguageScore(lower, englishTTSLanguageWeights) {
		return "es"
	}
	if weightedTTSLanguageScore(lower, englishTTSLanguageWeights) > 0 {
		return "en"
	}
	if strings.ContainsAny(lower, "ãõç") || containsAnyWord(lower, []string{" você ", " não ", " obrigado ", " olá "}) {
		return "pt"
	}
	if strings.ContainsAny(lower, "àâçèéêëîïôùûüÿœ") || containsAnyWord(lower, []string{" bonjour ", " merci ", " avec ", " pour "}) {
		return "fr"
	}
	if strings.ContainsAny(lower, "äöüß") || containsAnyWord(lower, []string{" danke ", " bitte ", " nicht ", " und "}) {
		return "de"
	}
	if strings.ContainsAny(lower, "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん") {
		return "ja"
	}
	if strings.ContainsAny(lower, "가나다라마바사아자차카타파하") {
		return "ko"
	}
	if strings.ContainsAny(lower, "的一是在不了有和人这中大为上个国我以要他") {
		return "zh"
	}
	return ""
}

var englishTTSLanguageWeights = map[string]int{
	" the ": 1, " and ": 1, " you ": 1, " for ": 1, " with ": 1,
	" hello ": 2, " thanks ": 2, " can ": 1, " help ": 2, " task ": 1,
}

var spanishTTSLanguageWeights = map[string]int{
	"¿": 4, "¡": 4, "ñ": 3, "á": 2, "é": 2, "í": 2, "ó": 2, "ú": 2, "ü": 2,
	" el ": 1, " la ": 1, " los ": 1, " las ": 1, " que ": 1, " para ": 1, " porque ": 2,
	" gracias ": 3, " hola ": 3, " puedo ": 2, " ayudarte ": 3, " tarea ": 2,
}

func weightedTTSLanguageScore(text string, weights map[string]int) int {
	score := 0
	for marker, weight := range weights {
		if strings.Contains(text, marker) {
			score += weight
		}
	}
	return score
}

func containsAnyWord(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}
