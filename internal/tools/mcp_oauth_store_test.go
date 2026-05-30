package tools

import (
	"testing"
	"time"
)

func TestMCPOAuthStoreFacade_StatusFor(t *testing.T) {
	store := NewMCPOAuthStore()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	if err := store.Set("srv", MCPOAuthToken{AccessToken: "access-1", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got := store.StatusFor("srv", now)
	if got.State != MCPOAuthStateValid || got.Evidence != "ok" {
		t.Fatalf("StatusFor = %+v, want valid/ok", got)
	}
}
