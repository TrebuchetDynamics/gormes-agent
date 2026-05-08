package i18n

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const defaultLanguage = "en"

var supportedLanguages = []string{"en", "zh", "ja", "de", "es", "fr", "tr", "uk"}

var languageAliases = map[string]string{
	"english": "en", "en-us": "en", "en-gb": "en",
	"chinese": "zh", "mandarin": "zh", "zh-cn": "zh", "zh-tw": "zh", "zh-hans": "zh", "zh-hant": "zh",
	"japanese": "ja", "jp": "ja", "ja-jp": "ja",
	"german": "de", "deutsch": "de", "de-de": "de",
	"spanish": "es", "español": "es", "espanol": "es", "es-es": "es", "es-mx": "es",
	"french": "fr", "français": "fr", "france": "fr", "fr-fr": "fr", "fr-be": "fr", "fr-ca": "fr", "fr-ch": "fr",
	"ukrainian": "uk", "ukrainisch": "uk", "українська": "uk", "uk-ua": "uk", "ua": "uk",
	"turkish": "tr", "türkçe": "tr", "tr-tr": "tr",
}

var (
	catalogMu      sync.RWMutex
	catalogCache   = map[string]map[string]string{}
	localesDir     string
	localesOnce    sync.Once
	localesMu      sync.Mutex
	configLanguage string
)

func initLocalesDir() {
	if dir := os.Getenv("GORMES_LOCALES_DIR"); dir != "" {
		localesDir = dir
		return
	}
	exec, err := os.Executable()
	if err == nil {
		localesDir = filepath.Join(filepath.Dir(exec), "locales")
		if fi, statErr := os.Stat(localesDir); statErr == nil && fi.IsDir() {
			return
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "locales")
		if fi, statErr := os.Stat(candidate); statErr == nil && fi.IsDir() {
			localesDir = candidate
			return
		}
	}
	localesDir = "locales"
}

func localesPath() string {
	localesOnce.Do(initLocalesDir)
	return localesDir
}

func SupportedLanguages() []string {
	result := make([]string, len(supportedLanguages))
	copy(result, supportedLanguages)
	return result
}

func DefaultLanguage() string {
	return defaultLanguage
}

func normalizeLang(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	if key == "" {
		return defaultLanguage
	}
	for _, lang := range supportedLanguages {
		if key == lang {
			return lang
		}
	}
	if mapped, ok := languageAliases[key]; ok {
		return mapped
	}
	if idx := strings.IndexByte(key, '-'); idx > 0 {
		base := key[:idx]
		for _, lang := range supportedLanguages {
			if base == lang {
				return lang
			}
		}
	}
	return defaultLanguage
}

func loadCatalog(lang string) map[string]string {
	catalogMu.RLock()
	if cached, ok := catalogCache[lang]; ok {
		catalogMu.RUnlock()
		return cached
	}
	catalogMu.RUnlock()

	path := filepath.Join(localesPath(), lang+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		catalogMu.Lock()
		catalogCache[lang] = map[string]string{}
		catalogMu.Unlock()
		return map[string]string{}
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		catalogMu.Lock()
		catalogCache[lang] = map[string]string{}
		catalogMu.Unlock()
		return map[string]string{}
	}

	flat := map[string]string{}
	flatten(raw, "", flat)

	catalogMu.Lock()
	catalogCache[lang] = flat
	catalogMu.Unlock()
	return flat
}

func flatten(node interface{}, prefix string, out map[string]string) {
	switch v := node.(type) {
	case map[string]interface{}:
		for key, val := range v {
			childKey := key
			if prefix != "" {
				childKey = prefix + "." + key
			}
			flatten(val, childKey, out)
		}
	case string:
		out[prefix] = v
	}
}

func SetConfigLanguage(lang string) {
	configLanguage = normalizeLang(lang)
}

func getLanguage() string {
	if envLang := os.Getenv("GORMES_LANGUAGE"); envLang != "" {
		return normalizeLang(envLang)
	}
	if configLanguage != "" {
		return configLanguage
	}
	return defaultLanguage
}

func ResetLanguageCache() {
	catalogMu.Lock()
	catalogCache = make(map[string]map[string]string)
	catalogMu.Unlock()
	localesMu.Lock()
	localesOnce = sync.Once{}
	localesMu.Unlock()
}

func T(key string, kv ...string) string {
	lang := ""
	formatArgs := map[string]string{}

	for i := 0; i < len(kv); i += 2 {
		k := kv[i]
		v := ""
		if i+1 < len(kv) {
			v = kv[i+1]
		}
		if k == "lang" {
			lang = v
		} else {
			formatArgs[k] = v
		}
	}

	target := defaultLanguage
	if lang != "" {
		target = normalizeLang(lang)
	} else {
		target = getLanguage()
	}

	catalog := loadCatalog(target)
	value, ok := catalog[key]

	if !ok && target != defaultLanguage {
		enCatalog := loadCatalog(defaultLanguage)
		value, ok = enCatalog[key]
	}

	if !ok {
		value = key
	}

	if len(formatArgs) > 0 {
		result := value
		for k, v := range formatArgs {
			result = strings.ReplaceAll(result, "{"+k+"}", v)
		}
		return result
	}
	return value
}
