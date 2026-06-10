package matrix

import (
	"reflect"
	"testing"
)

func TestEnvConfigUsesSharedBootstrapResolverDefaults(t *testing.T) {
	t.Setenv("MATRIX_HOMESERVER", " https://matrix.example.org/ ")
	t.Setenv("MATRIX_ACCESS_TOKEN", " syt_token ")
	t.Setenv("MATRIX_AUTO_THREAD", "")
	t.Setenv("MATRIX_REQUIRE_MENTION", "false")
	t.Setenv("MATRIX_FREE_RESPONSE_ROOMS", " !free:example.org, !ops:example.org ")

	cfg := EnvConfig()
	if cfg.Homeserver != "https://matrix.example.org" || cfg.AccessToken != "syt_token" {
		t.Fatalf("EnvConfig auth = %+v", cfg)
	}
	if !cfg.AutoThread {
		t.Fatalf("EnvConfig AutoThread = false, want shared default true")
	}
	if cfg.RequireMention {
		t.Fatalf("EnvConfig RequireMention = true, want env override false")
	}
	if !reflect.DeepEqual(cfg.FreeResponseRooms, []string{"!free:example.org", "!ops:example.org"}) {
		t.Fatalf("EnvConfig FreeResponseRooms = %#v", cfg.FreeResponseRooms)
	}
}

func TestConfig_IsAvailable(t *testing.T) {
	c := Config{Homeserver: "https://matrix.org", UserID: "@user:matrix.org", Password: "secret"}
	if !c.IsAvailable() {
		t.Fatal("should be available with all fields")
	}
}

func TestConfig_IsAvailableMissing(t *testing.T) {
	tests := []Config{
		{UserID: "@u", Password: "p"},
		{Homeserver: "h", Password: "p"},
		{Homeserver: "h", UserID: "@u"},
	}
	for _, c := range tests {
		if c.IsAvailable() {
			t.Fatalf("should not be available: %+v", c)
		}
	}
}

func TestChannel_Status(t *testing.T) {
	ch := New(Config{Homeserver: "h", UserID: "u", Password: "p"})
	s := ch.Status()
	if s["platform"] != "matrix" {
		t.Fatalf("platform = %v", s["platform"])
	}
	if s["available"] != true {
		t.Fatal("should be available")
	}
}

func TestChannel_StatusUnavailable(t *testing.T) {
	ch := New(Config{})
	s := ch.Status()
	if s["available"] != false {
		t.Fatal("should be unavailable")
	}
	if s["evidence"] == nil {
		t.Fatal("should have evidence")
	}
}
