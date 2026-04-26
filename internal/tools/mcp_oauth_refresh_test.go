package tools

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeMCPRefresher struct {
	calls        int
	refreshToken string
	nextToken    MCPOAuthToken
	err          error
}

func (f *fakeMCPRefresher) Refresh(ctx context.Context, refreshToken string) (MCPOAuthToken, error) {
	f.calls++
	f.refreshToken = refreshToken
	return f.nextToken, f.err
}

func TestRefreshMCPOAuth_StillValidNoOp(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore()
	if err := store.Set("srv", MCPOAuthToken{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeMCPRefresher{}

	got, err := RefreshMCPOAuth(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("RefreshMCPOAuth returned error: %v", err)
	}
	if got.Outcome != "still_valid" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "still_valid")
	}
	if refresher.calls != 0 {
		t.Fatalf("refresher calls = %d, want 0", refresher.calls)
	}
}

func TestRefreshMCPOAuth_RefreshesWhenExpired(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore()
	if err := store.Set("srv", MCPOAuthToken{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Scope:        "old-scope",
		Issuer:       "https://old.example.test",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	wantToken := MCPOAuthToken{
		AccessToken:  "access-2",
		RefreshToken: "refresh-2",
		Scope:        "new-scope",
		Issuer:       "https://new.example.test",
		ExpiresAt:    now.Add(time.Hour),
	}
	refresher := &fakeMCPRefresher{nextToken: wantToken}

	got, err := RefreshMCPOAuth(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("RefreshMCPOAuth returned error: %v", err)
	}
	if got.Outcome != "refreshed" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "refreshed")
	}
	if refresher.calls != 1 {
		t.Fatalf("refresher calls = %d, want 1", refresher.calls)
	}
	if refresher.refreshToken != "refresh-1" {
		t.Fatalf("refresher refreshToken = %q, want %q", refresher.refreshToken, "refresh-1")
	}
	gotToken, ok := store.Get("srv")
	if !ok {
		t.Fatalf("Get(srv) ok=false after refresh")
	}
	if gotToken != wantToken {
		t.Fatalf("Get(srv) = %+v, want %+v", gotToken, wantToken)
	}
}

func TestRefreshMCPOAuth_ClearsOnRefresherError(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore()
	if err := store.Set("srv", MCPOAuthToken{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeMCPRefresher{err: ErrMCPOAuthSessionExpired}

	got, err := RefreshMCPOAuth(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("RefreshMCPOAuth returned error: %v", err)
	}
	if got.Outcome != "token_cleared" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "token_cleared")
	}
	if _, ok := store.Get("srv"); ok {
		t.Fatalf("Get(srv) ok=true after session-expired refresh error")
	}
}

func TestRefreshMCPOAuth_NoninteractiveWhenNoRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore().WithNoninteractive(true)
	if err := store.Set("srv", MCPOAuthToken{
		AccessToken: "access-1",
		ExpiresAt:   now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &fakeMCPRefresher{}

	got, err := RefreshMCPOAuth(context.Background(), store, "srv", refresher, now)

	if !errors.Is(err, ErrMCPOAuthNoninteractiveRequired) {
		t.Fatalf("RefreshMCPOAuth error = %v, want errors.Is ErrMCPOAuthNoninteractiveRequired", err)
	}
	if got.Outcome != "noninteractive_required" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "noninteractive_required")
	}
	if refresher.calls != 0 {
		t.Fatalf("refresher calls = %d, want 0", refresher.calls)
	}
}

func TestRefreshMCPOAuth_RefresherUnavailable(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewMCPOAuthStore()
	wantToken := MCPOAuthToken{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Scope:        "read",
		Issuer:       "https://issuer.example.test",
		ExpiresAt:    now.Add(-time.Minute),
	}
	if err := store.Set("srv", wantToken); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := RefreshMCPOAuth(context.Background(), store, "srv", nil, now)

	if err != nil {
		t.Fatalf("RefreshMCPOAuth returned error: %v", err)
	}
	if got.Outcome != "refresher_unavailable" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "refresher_unavailable")
	}
	gotToken, ok := store.Get("srv")
	if !ok {
		t.Fatalf("Get(srv) ok=false after nil refresher")
	}
	if gotToken != wantToken {
		t.Fatalf("Get(srv) = %+v, want original %+v", gotToken, wantToken)
	}
}
