package memory

import (
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestCrossChatRecallEvidenceAllowedReportsUserScopeSourcesAndWidenedSessions(t *testing.T) {
	got := ExplainCrossChatRecall([]session.Metadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
		{SessionID: "sess-slack", Source: "slack", ChatID: "C123", UserID: "user-juan"},
		{SessionID: "sess-other", Source: "discord", ChatID: "chan-10", UserID: "user-maria"},
	}, SearchFilter{
		UserID:         "user-juan",
		Sources:        []string{"telegram", "discord"},
		CurrentChatKey: "discord:chan-9",
	})

	if got.Decision != CrossChatDecisionAllowed {
		t.Fatalf("Decision = %q, want %q: %+v", got.Decision, CrossChatDecisionAllowed, got)
	}
	if got.UserID != "user-juan" {
		t.Fatalf("UserID = %q, want user-juan", got.UserID)
	}
	if !slices.Equal(got.SourceAllowlist, []string{"telegram", "discord"}) {
		t.Fatalf("SourceAllowlist = %v, want telegram/discord in request order", got.SourceAllowlist)
	}
	if got.SessionsConsidered != 2 {
		t.Fatalf("SessionsConsidered = %d, want 2", got.SessionsConsidered)
	}
	if got.WidenedSessionsConsidered != 1 {
		t.Fatalf("WidenedSessionsConsidered = %d, want 1", got.WidenedSessionsConsidered)
	}
	if got.CurrentBinding == nil {
		t.Fatalf("CurrentBinding = nil, want current session evidence: %+v", got)
	}
	if got.CurrentBinding.SessionID != "sess-current" || got.CurrentBinding.Source != "discord" || got.CurrentBinding.ChatID != "chan-9" {
		t.Fatalf("CurrentBinding = %+v, want sess-current discord/chan-9", got.CurrentBinding)
	}
	if !hasSessionEvidence(got.Sessions, "sess-telegram", "telegram", "42", false) {
		t.Fatalf("Sessions = %+v, missing widened telegram evidence", got.Sessions)
	}
	if !hasSessionEvidence(got.Sessions, "sess-current", "discord", "chan-9", true) {
		t.Fatalf("Sessions = %+v, missing current discord evidence", got.Sessions)
	}
}

func TestCrossChatRecallEvidenceDeniedReportsReasonAndSameChatFallback(t *testing.T) {
	tests := []struct {
		name       string
		metas      []session.Metadata
		filter     SearchFilter
		wantReason string
	}{
		{
			name: "unknown current binding",
			metas: []session.Metadata{
				{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
			},
			filter: SearchFilter{
				UserID:         "user-juan",
				CurrentChatKey: "discord:chan-9",
			},
			wantReason: "unknown current binding",
		},
		{
			name: "conflicting current binding",
			metas: []session.Metadata{
				{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-maria"},
				{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
			},
			filter: SearchFilter{
				UserID:         "user-juan",
				CurrentChatKey: "discord:chan-9",
			},
			wantReason: "conflicting current binding",
		},
		{
			name: "unresolved user",
			metas: []session.Metadata{
				{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: ""},
			},
			filter: SearchFilter{
				CurrentChatKey: "discord:chan-9",
			},
			wantReason: "unresolved user_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExplainCrossChatRecall(tt.metas, tt.filter)
			if got.Decision != CrossChatDecisionDenied {
				t.Fatalf("Decision = %q, want %q: %+v", got.Decision, CrossChatDecisionDenied, got)
			}
			if got.FallbackScope != CrossChatFallbackSameChat {
				t.Fatalf("FallbackScope = %q, want %q", got.FallbackScope, CrossChatFallbackSameChat)
			}
			if !strings.Contains(got.Reason, tt.wantReason) {
				t.Fatalf("Reason = %q, want substring %q", got.Reason, tt.wantReason)
			}
			if got.SessionsConsidered != 0 {
				t.Fatalf("SessionsConsidered = %d, want 0 for denied widening", got.SessionsConsidered)
			}
		})
	}
}

func hasSessionEvidence(items []CrossChatSessionEvidence, sessionID, source, chatID string, current bool) bool {
	for _, item := range items {
		if item.SessionID == sessionID && item.Source == source && item.ChatID == chatID && item.Current == current {
			return true
		}
	}
	return false
}
