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

func TestRefreshRefreshesMissingAccessTokenEvenBeforeExpiry(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeRefresher{nextToken: Token{
		AccessToken: "access-2",
		ExpiresAt:   now.Add(2 * time.Hour),
	}}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if got.Outcome != RefreshOutcomeRefreshed {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeRefreshed)
	}
	if refresher.refreshToken != "refresh-1" {
		t.Fatalf("refresher refreshToken = %q, want refresh-1", refresher.refreshToken)
	}
	gotToken, ok := store.Get("srv")
	if !ok {
		t.Fatalf("Get(srv) ok=false after refresh")
	}
	if gotToken.AccessToken != "access-2" {
		t.Fatalf("AccessToken = %q, want refreshed access token", gotToken.AccessToken)
	}
	if gotToken.RefreshToken != "refresh-1" {
		t.Fatalf("RefreshToken = %q, want old refresh token preserved", gotToken.RefreshToken)
	}
}

func TestRefreshWithWhitespaceRefreshTokenRequiresNoninteractiveAuth(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: " \t\n ",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeRefresher{nextToken: Token{AccessToken: "access-2"}}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != ErrNoninteractiveRequired {
		t.Fatalf("Refresh error = %v, want %v", err, ErrNoninteractiveRequired)
	}
	if got.Outcome != RefreshOutcomeNoninteractiveRequired {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeNoninteractiveRequired)
	}
	if refresher.refreshToken != "" {
		t.Fatalf("refresher was called with %q, want no refresh attempt", refresher.refreshToken)
	}
}

func TestRefreshRejectsResponseWithoutAccessToken(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	oldToken := Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Scope:        "old-scope",
		Issuer:       "https://old.example.test",
		ExpiresAt:    now.Add(-time.Minute),
	}
	if err := store.Set("srv", oldToken); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeRefresher{nextToken: Token{
		RefreshToken: "refresh-2",
		ExpiresAt:    now.Add(time.Hour),
	}}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err == nil {
		t.Fatalf("Refresh succeeded, want invalid-token error")
	}
	if got.Outcome != RefreshOutcomeRefresherUnavailable {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeRefresherUnavailable)
	}
	gotToken, ok := store.Get("srv")
	if !ok {
		t.Fatalf("Get(srv) ok=false after invalid refresh")
	}
	if gotToken.AccessToken != oldToken.AccessToken || gotToken.RefreshToken != oldToken.RefreshToken || gotToken.ExpiresAt != oldToken.ExpiresAt {
		t.Fatalf("stored token changed after invalid refresh: got %+v want %+v", gotToken, oldToken)
	}
}

func TestRefreshPreservesOldRefreshTokenWhenResponseHasWhitespaceOne(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeRefresher{nextToken: Token{
		AccessToken:  "access-2",
		RefreshToken: " \t\n ",
		ExpiresAt:    now.Add(time.Hour),
	}}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if got.Outcome != RefreshOutcomeRefreshed {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, RefreshOutcomeRefreshed)
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
