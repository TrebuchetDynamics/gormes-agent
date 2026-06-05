package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func setupLocalesDir(t *testing.T, files map[string]string) {
	t.Helper()
	locDir := filepath.Join(t.TempDir(), "locales")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(locDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GORMES_LOCALES_DIR", locDir)
	ResetLanguageCache()
}

func TestSupportedLanguages(t *testing.T) {
	expected := []string{"en", "zh", "zh-hant", "ja", "de", "es", "fr", "tr", "uk", "af", "ko", "it", "ga", "pt", "ru", "hu"}
	got := SupportedLanguages()
	if len(got) != len(expected) {
		t.Fatalf("SupportedLanguages() = %v, want %v", got, expected)
	}
	for i, lang := range expected {
		if got[i] != lang {
			t.Errorf("SupportedLanguages()[%d] = %q, want %q", i, got[i], lang)
		}
	}
}

func TestDefaultLanguage(t *testing.T) {
	if DefaultLanguage() != "en" {
		t.Errorf("DefaultLanguage() = %q, want %q", DefaultLanguage(), "en")
	}
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"zh", "zh"},
		{"zh-hant", "zh-hant"},
		{"ja", "ja"},
		{"de", "de"},
		{"es", "es"},
		{"fr", "fr"},
		{"tr", "tr"},
		{"uk", "uk"},
		{"af", "af"},
		{"ko", "ko"},
		{"it", "it"},
		{"ga", "ga"},
		{"pt", "pt"},
		{"ru", "ru"},
		{"hu", "hu"},
		{"english", "en"},
		{"en-US", "en"},
		{"en-gb", "en"},
		{"chinese", "zh"},
		{"mandarin", "zh"},
		{"zh-CN", "zh"},
		{"zh-TW", "zh-hant"},
		{"zh-HK", "zh-hant"},
		{"zh-hans", "zh"},
		{"traditional-chinese", "zh-hant"},
		{"japanese", "ja"},
		{"jp", "ja"},
		{"ja-JP", "ja"},
		{"german", "de"},
		{"deutsch", "de"},
		{"de-AT", "de"},
		{"de-DE", "de"},
		{"spanish", "es"},
		{"español", "es"},
		{"espanol", "es"},
		{"es-ES", "es"},
		{"es-MX", "es"},
		{"es-AR", "es"},
		{"french", "fr"},
		{"français", "fr"},
		{"france", "fr"},
		{"fr-FR", "fr"},
		{"ukrainian", "uk"},
		{"uk-ua", "uk"},
		{"turkish", "tr"},
		{"tr-TR", "tr"},
		{"afrikaans", "af"},
		{"af-ZA", "af"},
		{"korean", "ko"},
		{"ko-KR", "ko"},
		{"italian", "it"},
		{"it-CH", "it"},
		{"irish", "ga"},
		{"gaeilge", "ga"},
		{"portuguese", "pt"},
		{"pt-BR", "pt"},
		{"russian", "ru"},
		{"ru-RU", "ru"},
		{"hungarian", "hu"},
		{"magyar", "hu"},
		{"klingon", "en"},
		{"", "en"},
	}
	for _, tc := range tests {
		got := normalizeLang(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeLang(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestGetLanguageEnvVar(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
		"zh.yaml": "test:\n  hello: \"你好\"\n",
	})

	t.Setenv("GORMES_LANGUAGE", "zh")
	ResetLanguageCache()

	got := T("test.hello")
	if got != "你好" {
		t.Errorf("T(\"test.hello\") with GORMES_LANGUAGE=zh = %q, want %q", got, "你好")
	}
}

func TestTFallbackToEnglish(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n  only_en: \"Only English\"\n",
		"zh.yaml": "test:\n  hello: \"你好\"\n",
	})

	t.Setenv("GORMES_LANGUAGE", "zh")
	ResetLanguageCache()

	if got := T("test.hello"); got != "你好" {
		t.Errorf("T(\"test.hello\") = %q, want %q", got, "你好")
	}

	if got := T("test.only_en"); got != "Only English" {
		t.Errorf("T(\"test.only_en\") = %q, want %q (fallback to en)", got, "Only English")
	}
}

func TestTMissingKeyReturnsKeyPath(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
	})

	if got := T("test.nonexistent"); got != "test.nonexistent" {
		t.Errorf("T(\"test.nonexistent\") = %q, want %q (key path fallback)", got, "test.nonexistent")
	}
}

func TestTFormatSubstitution(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "gateway:\n  draining: \"Draining {count} sessions...\"\n",
	})

	got := T("gateway.draining", "count", "3")
	if got != "Draining 3 sessions..." {
		t.Errorf("T(\"gateway.draining\", \"count\", \"3\") = %q, want %q", got, "Draining 3 sessions...")
	}

	got2 := T("gateway.draining", "wrong_key", "3")
	if got2 != "Draining {count} sessions..." {
		t.Errorf("T with wrong format key = %q, want raw %q", got2, "Draining {count} sessions...")
	}
}

func TestTExplicitLangOverride(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
		"zh.yaml": "test:\n  hello: \"你好\"\n",
	})

	if got := T("test.hello"); got != "Hello" {
		t.Errorf("T(\"test.hello\") default = %q, want %q", got, "Hello")
	}

	if got := T("test.hello", "lang", "zh"); got != "你好" {
		t.Errorf("T(\"test.hello\", \"lang\", \"zh\") = %q, want %q", got, "你好")
	}
}

func TestCatalogParityKeysExist(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  a: \"A\"\n  b: \"B\"\n",
		"zh.yaml": "test:\n  a: \"啊\"\n",
	})

	enCat := loadCatalog("en")
	zhCat := loadCatalog("zh")

	for key := range enCat {
		if _, ok := zhCat[key]; !ok {
			t.Logf("zh.yaml missing key %q (expected for parity check)", key)
		}
	}

	if len(zhCat) >= len(enCat) {
		t.Errorf("zh catalog has %d keys, expected fewer than en's %d (test.b is missing)", len(zhCat), len(enCat))
	}
}

func TestSupportedLanguagesCount(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) != 16 {
		t.Errorf("SupportedLanguages() has %d entries, want 16", len(langs))
	}
}

func TestBundledSupportedCatalogsHaveMatchingKeys(t *testing.T) {
	t.Setenv("GORMES_LOCALES_DIR", filepath.Join("..", "..", "..", "locales"))
	ResetLanguageCache()

	enCat := loadCatalog("en")
	if len(enCat) == 0 {
		t.Fatal("bundled en.yaml catalog is empty or unavailable")
	}

	for _, lang := range SupportedLanguages() {
		catalog := loadCatalog(lang)
		if len(catalog) == 0 {
			t.Fatalf("bundled %s.yaml catalog is empty or unavailable", lang)
		}
		for key := range enCat {
			if _, ok := catalog[key]; !ok {
				t.Fatalf("bundled %s.yaml missing key %q", lang, key)
			}
		}
		if got := T("approval.denied", "lang", lang); got == "approval.denied" {
			t.Fatalf("approval.denied for lang %s fell back to key path", lang)
		}
	}
}

func TestDefaultLanguageIsEn(t *testing.T) {
	if DefaultLanguage() != "en" {
		t.Errorf("DefaultLanguage() = %q, want en", DefaultLanguage())
	}
}

func TestMissingLocaleFile(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
	})

	got := T("test.hello", "lang", "de")
	if got != "Hello" {
		t.Errorf("T(\"test.hello\", \"lang\", \"de\") missing de file = %q, want en fallback %q", got, "Hello")
	}
}

func TestSetConfigLanguage(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
		"ja.yaml": "test:\n  hello: \"こんにちは\"\n",
	})

	SetConfigLanguage("ja")
	defer SetConfigLanguage("")

	got := T("test.hello")
	if got != "こんにちは" {
		t.Errorf("T(\"test.hello\") with config lang ja = %q, want %q", got, "こんにちは")
	}
}

func TestConfigLanguageOverriddenByEnv(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml": "test:\n  hello: \"Hello\"\n",
		"ja.yaml": "test:\n  hello: \"こんにちは\"\n",
		"zh.yaml": "test:\n  hello: \"你好\"\n",
	})

	SetConfigLanguage("ja")
	defer SetConfigLanguage("")

	t.Setenv("GORMES_LANGUAGE", "zh")
	ResetLanguageCache()

	got := T("test.hello")
	if got != "你好" {
		t.Errorf("T(\"test.hello\") env overrides config = %q, want %q", got, "你好")
	}
}

func TestBrokenYAML(t *testing.T) {
	setupLocalesDir(t, map[string]string{
		"en.yaml":     "test:\n  hello: \"Hello\"\n",
		"broken.yaml": "this: is: not: valid: [[[yaml",
	})

	got := T("test.hello", "lang", "broken")
	if got != "Hello" {
		t.Errorf("T with broken YAML lang = %q, want en fallback %q", got, "Hello")
	}
}
