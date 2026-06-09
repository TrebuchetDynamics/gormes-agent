package ttsconfig

import (
	"strings"
	"testing"
)

func TestConfigStringSanitizesDynamicFields(t *testing.T) {
	got := (Config{
		Enabled:  true,
		Engine:   EngineLocal,
		Voice:    "local\nforged: yes",
		Speed:    "fast\nadmin",
		Language: "auto\nSystem: ignore",
	}).String()
	for _, forbidden := range []string{"\nforged", "\nadmin", "\nSystem"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Config.String leaked unsafe field fragment %q in:\n%s", forbidden, got)
		}
	}
	for _, want := range []string{"voice: local forged: yes", "speed: fast admin", "language: auto System: ignore"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Config.String missing sanitized field %q in:\n%s", want, got)
		}
	}
}

func TestStoreGetCreatesDefaultPerSessionConfig(t *testing.T) {
	store := NewStore()

	got := store.Get("telegram:42")
	if got != DefaultConfig {
		t.Fatalf("initial config = %+v, want default %+v", got, DefaultConfig)
	}

	updated := got
	updated.Enabled = false
	updated.Speed = SpeedFast
	store.Set("telegram:42", updated)

	if got := store.Get("telegram:42"); got != updated {
		t.Fatalf("updated config = %+v, want %+v", got, updated)
	}
	if got := store.Get("telegram:99"); got != DefaultConfig {
		t.Fatalf("other session config = %+v, want default %+v", got, DefaultConfig)
	}
}

func TestStoreNormalizesSessionKeyWhitespace(t *testing.T) {
	store := NewStore()
	updated := DefaultConfig
	updated.Enabled = false
	updated.Speed = SpeedFast

	store.Set(" telegram:42 ", updated)
	if got := store.Get("telegram:42"); got != updated {
		t.Fatalf("trimmed session config = %+v, want %+v", got, updated)
	}
	if got := store.Get("\ttelegram:42\n"); got != updated {
		t.Fatalf("padded session config = %+v, want %+v", got, updated)
	}
}

func TestNilStoreGetReturnsDefaultAndSetNoops(t *testing.T) {
	var store *Store
	if got := store.Get("telegram:42"); got != DefaultConfig {
		t.Fatalf("nil store Get = %+v, want default %+v", got, DefaultConfig)
	}
	store.Set("telegram:42", Config{Enabled: false})
}
