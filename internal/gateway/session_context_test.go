package gateway

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

type errorSessionMap struct {
	getErr error
}

func (m errorSessionMap) Get(context.Context, string) (string, error) { return "", m.getErr }
func (m errorSessionMap) Put(context.Context, string, string) error   { return nil }
func (m errorSessionMap) Close() error                                { return nil }

func TestResolveSessionID_StoredValueWins(t *testing.T) {
	smap := session.NewMemMap()
	if err := smap.Put(context.Background(), "telegram:42", "sess-stored"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := resolveSessionID(context.Background(), smap, "telegram:42")
	if err != nil {
		t.Fatalf("resolveSessionID error = %v, want nil", err)
	}
	if got != "sess-stored" {
		t.Fatalf("resolveSessionID = %q, want %q", got, "sess-stored")
	}
}

func TestResolveSessionID_FallsBackToChatKeyWhenMissing(t *testing.T) {
	got, err := resolveSessionID(context.Background(), session.NewMemMap(), "telegram:42")
	if err != nil {
		t.Fatalf("resolveSessionID error = %v, want nil", err)
	}
	if got != "telegram:42" {
		t.Fatalf("resolveSessionID = %q, want %q", got, "telegram:42")
	}
}

func TestResolveSessionID_FallsBackToChatKeyOnError(t *testing.T) {
	boom := errors.New("boom")

	got, err := resolveSessionID(context.Background(), errorSessionMap{getErr: boom}, "telegram:42")
	if !errors.Is(err, boom) {
		t.Fatalf("resolveSessionID error = %v, want %v", err, boom)
	}
	if got != "telegram:42" {
		t.Fatalf("resolveSessionID = %q, want %q", got, "telegram:42")
	}
}

func TestBuildSessionContextPrompt(t *testing.T) {
	got := BuildSessionContextPrompt(SessionContext{
		Source: SessionSource{
			Platform: "telegram",
			ChatID:   "42",
			UserID:   "7",
		},
		SessionKey:         "telegram:42",
		SessionID:          "sess-stored",
		ConnectedPlatforms: []string{"discord", "telegram"},
	})

	assertContainsAll(t, got,
		"## Current Session Context",
		"**Source:** telegram chat `42`",
		"**User ID:** `7`",
		"**Session Key:** `telegram:42`",
		"**Session ID:** `sess-stored`",
		"`origin`",
		"`local`",
		"`discord`",
		"`telegram`",
	)
}

func TestBuildSessionContextPrompt_BlueBubblesGuidanceIncludesShortBubbleHint(t *testing.T) {
	got := BuildSessionContextPrompt(SessionContext{
		Source: SessionSource{
			Platform: "bluebubbles",
			ChatID:   "iMessage;-;+15555550100",
		},
		SessionKey: "bluebubbles:+15555550100",
		SessionID:  "sess-bb",
	})
	assertContainsAll(t, got,
		"**Platform notes:**",
		"iMessage",
		"short",
		"blank line",
		"1-3 sentences",
	)
}

func TestBuildSessionContextPrompt_NonBlueBubblesOmitsPlatformNotes(t *testing.T) {
	got := BuildSessionContextPrompt(SessionContext{
		Source: SessionSource{Platform: "telegram", ChatID: "42"},
	})
	if strings.Contains(got, "**Platform notes:**") {
		t.Fatalf("BuildSessionContextPrompt leaked Platform notes section into telegram prompt:\n%s", got)
	}
}
