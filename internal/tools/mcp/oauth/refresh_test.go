package oauth

import (
	"context"
	"testing"
	"time"
)

type fakeRefresher struct {
	refreshToken string
	nextToken    Token
}

func (f *fakeRefresher) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	f.refreshToken = refreshToken
	return f.nextToken, nil
}

func TestRefreshPreservesOldRefreshTokenWhenResponseOmitsOne(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Scope:        "old-scope",
		Issuer:       "https://old.example.test",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeRefresher{nextToken: Token{
		AccessToken: "access-2",
		Scope:       "new-scope",
		Issuer:      "https://new.example.test",
		ExpiresAt:   now.Add(time.Hour),
	}}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if got.Outcome != RefreshOutcomeRefreshed {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeRefreshed)
	}
	if refresher.refreshToken != "refresh-1" {
		t.Fatalf("refresher refreshToken = %q, want old refresh token", refresher.refreshToken)
	}
	gotToken, ok := store.Get("srv")
	if !ok {
		t.Fatalf("Get(srv) ok=false after refresh")
	}
	if gotToken.RefreshToken != "refresh-1" {
		t.Fatalf("RefreshToken = %q, want old refresh token preserved", gotToken.RefreshToken)
	}
	if gotToken.AccessToken != "access-2" {
		t.Fatalf("AccessToken = %q, want refreshed access token", gotToken.AccessToken)
	}
}
