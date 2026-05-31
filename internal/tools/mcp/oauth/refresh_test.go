package oauth

import (
	"context"
	"errors"
	"strings"
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

type failingRefresher struct {
	err error
}

func (f failingRefresher) Refresh(context.Context, string) (Token, error) {
	return Token{}, f.err
}

func TestRefreshWithNilStoreRequiresNoninteractiveAuth(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	got, err := Refresh(context.Background(), nil, "srv", nil, now)

	if err != ErrNoninteractiveRequired {
		t.Fatalf("Refresh error = %v, want %v", err, ErrNoninteractiveRequired)
	}
	if got.Server != "srv" {
		t.Fatalf("Server = %q, want srv", got.Server)
	}
	if got.Outcome != RefreshOutcomeNoninteractiveRequired {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeNoninteractiveRequired)
	}
}

func TestRefreshRedactsRefresherErrors(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-secret-1",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	providerErr := errors.New("provider failed with token=refresh-secret-1")
	got, err := Refresh(context.Background(), store, "srv", failingRefresher{
		err: providerErr,
	}, now)

	if err == nil {
		t.Fatalf("Refresh succeeded, want redacted error")
	}
	if got.Outcome != RefreshOutcomeRefresherUnavailable {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeRefresherUnavailable)
	}
	if strings.Contains(err.Error(), "refresh-secret-1") || strings.Contains(err.Error(), "token=") {
		t.Fatalf("refresh error leaked token material: %q", err.Error())
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("Refresh error no longer unwraps provider error: %v", err)
	}
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
