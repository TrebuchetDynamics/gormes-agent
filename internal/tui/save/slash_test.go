package save

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

func TestStatuses(t *testing.T) {
	if got := SuccessStatus("/tmp/session.md"); got != "save: wrote /tmp/session.md" {
		t.Fatalf("SuccessStatus = %q", got)
	}
	if got := FailureStatus(transcript.ErrSessionNotFound); got != "save: store unavailable" {
		t.Fatalf("FailureStatus not found = %q", got)
	}
	if got := FailureStatus(errors.New("disk full")); got != "save: write failed: disk full" {
		t.Fatalf("FailureStatus generic = %q", got)
	}
}

func TestHandleSlashValidationAndCleanup(t *testing.T) {
	if got := HandleSlash(false, "sess", nil, nil); got != "save: no conversation" {
		t.Fatalf("HandleSlash(no conversation) = %q", got)
	}
	if got := HandleSlash(true, " ", nil, nil); got != "save: no active session" {
		t.Fatalf("HandleSlash(no session) = %q", got)
	}
	if got := HandleSlash(true, "sess", nil, nil); got != "save: store unavailable" {
		t.Fatalf("HandleSlash(no export) = %q", got)
	}

	if got := HandleSlash(true, " sess ", func(ctx context.Context, sessionID string) (string, error) {
		if ctx == nil {
			t.Fatal("export context is nil")
		}
		if sessionID != "sess" {
			t.Fatalf("export sessionID = %q, want trimmed sess", sessionID)
		}
		return "/tmp/session.md", nil
	}, nil); got != "save: wrote /tmp/session.md" {
		t.Fatalf("HandleSlash(success) = %q", got)
	}

	var removed string
	got := HandleSlash(true, "sess", func(context.Context, string) (string, error) {
		return "/tmp/partial.md", errors.New("disk full")
	}, func(path string) error {
		removed = path
		return nil
	})
	if got != "save: write failed: disk full" || removed != "/tmp/partial.md" {
		t.Fatalf("HandleSlash(error) = (%q, removed %q)", got, removed)
	}
}
