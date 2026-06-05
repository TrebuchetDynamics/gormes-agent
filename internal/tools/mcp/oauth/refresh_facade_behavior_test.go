package oauth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recordingRefresher struct {
	calls        int
	refreshToken string
	nextToken    Token
	err          error
}

func (f *recordingRefresher) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	f.calls++
	f.refreshToken = refreshToken
	return f.nextToken, f.err
}

func TestRefreshStillValidNoOp(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &recordingRefresher{}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if got.Outcome != "still_valid" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "still_valid")
	}
	if refresher.calls != 0 {
		t.Fatalf("refresher calls = %d, want 0", refresher.calls)
	}
}

func TestRefreshRefreshesWhenExpired(t *testing.T) {
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
	wantToken := Token{
		AccessToken:  "access-2",
		RefreshToken: "refresh-2",
		Scope:        "new-scope",
		Issuer:       "https://new.example.test",
		ExpiresAt:    now.Add(time.Hour),
	}
	refresher := &recordingRefresher{nextToken: wantToken}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
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

func TestRefreshClearsOnRefresherError(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	if err := store.Set("srv", Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &recordingRefresher{err: ErrSessionExpired}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if got.Outcome != "token_cleared" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "token_cleared")
	}
	if _, ok := store.Get("srv"); ok {
		t.Fatalf("Get(srv) ok=true after session-expired refresh error")
	}
}

func TestRefreshNoninteractiveWhenNoRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore().WithNoninteractive(true)
	if err := store.Set("srv", Token{
		AccessToken: "access-1",
		ExpiresAt:   now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	refresher := &recordingRefresher{}

	got, err := Refresh(context.Background(), store, "srv", refresher, now)

	if !errors.Is(err, ErrNoninteractiveRequired) {
		t.Fatalf("Refresh error = %v, want errors.Is ErrNoninteractiveRequired", err)
	}
	if got.Outcome != "noninteractive_required" {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, "noninteractive_required")
	}
	if refresher.calls != 0 {
		t.Fatalf("refresher calls = %d, want 0", refresher.calls)
	}
}

func TestRefreshRefresherUnavailable(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	wantToken := Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		Scope:        "read",
		Issuer:       "https://issuer.example.test",
		ExpiresAt:    now.Add(-time.Minute),
	}
	if err := store.Set("srv", wantToken); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	got, err := Refresh(context.Background(), store, "srv", nil, now)

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
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
