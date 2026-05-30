package ttsconfig

import "testing"

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

func TestNilStoreGetReturnsDefaultAndSetNoops(t *testing.T) {
	var store *Store
	if got := store.Get("telegram:42"); got != DefaultConfig {
		t.Fatalf("nil store Get = %+v, want default %+v", got, DefaultConfig)
	}
	store.Set("telegram:42", Config{Enabled: false})
}
