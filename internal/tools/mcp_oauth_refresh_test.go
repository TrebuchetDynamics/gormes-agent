package tools

import (
	"context"
	"testing"
	"time"
)

type facadeMCPRefresher struct {
	nextToken MCPOAuthToken
}

func (f facadeMCPRefresher) Refresh(context.Context, string) (MCPOAuthToken, error) {
	return f.nextToken, nil
}

func TestRefreshMCPOAuthFacade_RefreshesExpiredToken(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore()
	if err := store.Set("srv", MCPOAuthToken{AccessToken: "old", RefreshToken: "refresh", ExpiresAt: now.Add(-time.Minute)}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	wantToken := MCPOAuthToken{AccessToken: "new", RefreshToken: "refresh-2", ExpiresAt: now.Add(time.Hour)}
	got, err := RefreshMCPOAuth(context.Background(), store, "srv", facadeMCPRefresher{nextToken: wantToken}, now)
	if err != nil {
		t.Fatalf("RefreshMCPOAuth returned error: %v", err)
	}
	if got.Outcome != MCPOAuthRefreshOutcomeRefreshed {
		t.Fatalf("Outcome = %q, want refreshed", got.Outcome)
	}
	tok, ok := store.Get("srv")
	if !ok || tok != wantToken {
		t.Fatalf("stored token = %+v ok=%v, want %+v", tok, ok, wantToken)
	}
}
