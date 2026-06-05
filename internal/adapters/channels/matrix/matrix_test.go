package matrix

import (
	"testing"
)

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
