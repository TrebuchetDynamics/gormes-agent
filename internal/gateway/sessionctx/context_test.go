package sessionctx

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

	got, err := ResolveSessionID(context.Background(), smap, "telegram:42")
	if err != nil {
		t.Fatalf("ResolveSessionID error = %v, want nil", err)
	}
	if got != "sess-stored" {
		t.Fatalf("ResolveSessionID = %q, want %q", got, "sess-stored")
	}
}

func TestResolveSessionID_FallsBackToChatKeyWhenMissing(t *testing.T) {
	got, err := ResolveSessionID(context.Background(), session.NewMemMap(), "telegram:42")
	if err != nil {
		t.Fatalf("ResolveSessionID error = %v, want nil", err)
	}
	if got != "telegram:42" {
		t.Fatalf("ResolveSessionID = %q, want %q", got, "telegram:42")
	}
}

func TestResolveSessionID_FallsBackToChatKeyOnError(t *testing.T) {
	boom := errors.New("boom")

	got, err := ResolveSessionID(context.Background(), errorSessionMap{getErr: boom}, "telegram:42")
	if !errors.Is(err, boom) {
		t.Fatalf("ResolveSessionID error = %v, want %v", err, boom)
	}
	if got != "telegram:42" {
		t.Fatalf("ResolveSessionID = %q, want %q", got, "telegram:42")
	}
}

func TestBuildPrompt(t *testing.T) {
	got := BuildPrompt(Context{
		Source: Source{
			Platform: "telegram",
			ChatID:   "42",
			UserID:   "7",
		},
		SessionKey:         "telegram:42",
		SessionID:          "sess-stored",
		ConnectedPlatforms: []string{"discord", "telegram"},
	})

	for _, want := range []string{
		"## Current Session Context",
		"**Source:** telegram chat `42`",
		"**User ID:** `7`",
		"**Session Key:** `telegram:42`",
		"**Session ID:** `sess-stored`",
		"`origin`",
		"`local`",
		"`discord`",
		"`telegram`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildPrompt_BlueBubblesGuidanceIncludesShortBubbleHint(t *testing.T) {
	got := BuildPrompt(Context{
		Source: Source{
			Platform: "bluebubbles",
			ChatID:   "iMessage;-;+15555550100",
		},
		SessionKey: "bluebubbles:+15555550100",
		SessionID:  "sess-bb",
	})
	for _, want := range []string{
		"**Platform notes:**",
		"iMessage",
		"short",
		"blank line",
		"1-3 sentences",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildPrompt missing %q for bluebubbles platform:\n%s", want, got)
		}
	}
}

func TestBuildPrompt_NonBlueBubblesOmitsPlatformNotes(t *testing.T) {
	got := BuildPrompt(Context{
		Source: Source{Platform: "telegram", ChatID: "42"},
	})
	if strings.Contains(got, "**Platform notes:**") {
		t.Fatalf("BuildPrompt leaked Platform notes section into telegram prompt:\n%s", got)
	}
}
