package mirror

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
)

func TestSelectDeliveryMirrorSession_PrefersExactUserAndThread(t *testing.T) {
	candidates := []session.Metadata{
		{SessionID: "sess-old", Source: "telegram", ChatID: "-100", UserID: "u1", UpdatedAt: 10},
		{SessionID: "sess-wrong-thread", Source: "telegram", ChatID: "-100", UserID: "u2", UpdatedAt: 20},
		{SessionID: "sess-topic-user", Source: "telegram", ChatID: "-100", UserID: "u2", UpdatedAt: 30, LineageKind: session.LineageKindPrimary},
	}
	target := DeliveryMirrorTarget{
		Platform: "Telegram",
		ChatID:   "-100",
		ThreadID: "10",
		UserID:   "u2",
	}
	candidates[2].ChatID = "-100:10"

	got, ok := SelectDeliveryMirrorSession(candidates, target)
	if !ok {
		t.Fatal("SelectDeliveryMirrorSession ok = false, want exact user/thread match")
	}
	if got.SessionID != "sess-topic-user" {
		t.Fatalf("selected session = %q, want sess-topic-user", got.SessionID)
	}
}

func TestSelectDeliveryMirrorSessionNormalizesReturnedMetadata(t *testing.T) {
	candidates := []session.Metadata{
		{SessionID: " sess-valid ", Source: " Slack ", ChatID: " C123 ", UserID: " u1 ", UpdatedAt: 10},
	}

	got, ok := SelectDeliveryMirrorSession(candidates, DeliveryMirrorTarget{Platform: "slack", ChatID: "C123", UserID: "u1"})
	if !ok {
		t.Fatal("SelectDeliveryMirrorSession ok = false, want match")
	}
	if got.SessionID != "sess-valid" || got.Source != "slack" || got.ChatID != "C123" || got.UserID != "u1" {
		t.Fatalf("selected metadata = %+v, want normalized identifiers", got)
	}
}

func TestSelectDeliveryMirrorSessionIgnoresEmptySessionIDCandidates(t *testing.T) {
	candidates := []session.Metadata{
		{SessionID: "", Source: "slack", ChatID: "C123", UpdatedAt: 99},
		{SessionID: "sess-valid", Source: "slack", ChatID: "C123", UpdatedAt: 10},
	}

	got, ok := SelectDeliveryMirrorSession(candidates, DeliveryMirrorTarget{Platform: "slack", ChatID: "C123"})
	if !ok || got.SessionID != "sess-valid" {
		t.Fatalf("SelectDeliveryMirrorSession = %+v, %v; want sess-valid ignoring empty session id", got, ok)
	}
}

func TestSelectDeliveryMirrorSession_AmbiguousGroupWithoutUser(t *testing.T) {
	tests := []struct {
		name       string
		candidates []session.Metadata
	}{
		{
			name: "distinct users",
			candidates: []session.Metadata{
				{SessionID: "sess-a", Source: "telegram", ChatID: "-100", UserID: "u1", UpdatedAt: 10},
				{SessionID: "sess-b", Source: "telegram", ChatID: "-100", UserID: "u2", UpdatedAt: 20},
			},
		},
		{
			name: "known and unknown user provenance",
			candidates: []session.Metadata{
				{SessionID: "sess-known", Source: "telegram", ChatID: "-100", UserID: "u1", UpdatedAt: 10},
				{SessionID: "sess-unknown", Source: "telegram", ChatID: "-100", UpdatedAt: 20},
			},
		},
		{
			name: "multiple unknown user provenance",
			candidates: []session.Metadata{
				{SessionID: "sess-unknown-a", Source: "telegram", ChatID: "-100", UpdatedAt: 10},
				{SessionID: "sess-unknown-b", Source: "telegram", ChatID: "-100", UpdatedAt: 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := SelectDeliveryMirrorSession(tt.candidates, DeliveryMirrorTarget{Platform: "telegram", ChatID: "-100"}); ok {
				t.Fatalf("SelectDeliveryMirrorSession = %+v, want no guess for ambiguous users", got)
			}
		})
	}
}

func TestSelectDeliveryMirrorSession_ExplicitUserMustMatchSingleCandidate(t *testing.T) {
	candidates := []session.Metadata{
		{SessionID: "sess-wrong-user", Source: "telegram", ChatID: "-100", UserID: "u1", UpdatedAt: 10},
	}

	if got, ok := SelectDeliveryMirrorSession(candidates, DeliveryMirrorTarget{Platform: "telegram", ChatID: "-100", UserID: "u2"}); ok {
		t.Fatalf("SelectDeliveryMirrorSession = %+v, want no mirror to different explicit user", got)
	}
}

func TestMirrorDeliveryToSessionNormalizesSelectedMetadataPayload(t *testing.T) {
	rec := store.NewRecording()
	now := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)
	candidates := []session.Metadata{
		{SessionID: " sess-target ", Source: " slack ", ChatID: " C123 ", UpdatedAt: 42},
	}

	result, err := MirrorDeliveryToSession(context.Background(), rec, candidates, DeliveryMirrorTarget{
		Platform:    "slack",
		ChatID:      "C123",
		MessageText: "hello",
	}, now)
	if err != nil {
		t.Fatalf("MirrorDeliveryToSession error = %v", err)
	}
	if !result.Mirrored || result.SessionID != "sess-target" {
		t.Fatalf("result = %+v, want normalized mirrored session", result)
	}
	var payload struct {
		SessionID string `json:"session_id"`
		ChatID    string `json:"chat_id"`
	}
	cmds := rec.Commands()
	if len(cmds) != 1 {
		t.Fatalf("commands = %+v, want one mirror command", cmds)
	}
	if err := json.Unmarshal(cmds[0].Payload, &payload); err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	if payload.SessionID != "sess-target" || payload.ChatID != "C123" {
		t.Fatalf("payload = %+v, want normalized session/chat metadata", payload)
	}
}

func TestMirrorDeliveryToSessionRedactsSecretLikeSourceLabel(t *testing.T) {
	rec := store.NewRecording()
	candidates := []session.Metadata{{SessionID: "sess-target", Source: "slack", ChatID: "C123", UpdatedAt: 42}}

	_, err := MirrorDeliveryToSession(context.Background(), rec, candidates, DeliveryMirrorTarget{
		Platform:    "slack",
		ChatID:      "C123",
		MessageText: "hello",
		SourceLabel: "tool api_key=plain-secret-token",
	}, time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("MirrorDeliveryToSession error = %v", err)
	}
	cmds := rec.Commands()
	if len(cmds) != 1 {
		t.Fatalf("commands = %+v, want one mirror command", cmds)
	}
	var payload struct {
		MetaJSON string `json:"meta_json"`
	}
	if err := json.Unmarshal(cmds[0].Payload, &payload); err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(payload.MetaJSON, forbidden) {
			t.Fatalf("meta_json leaked secret-like source label %q: %s", forbidden, payload.MetaJSON)
		}
	}
	if !strings.Contains(payload.MetaJSON, "[redacted]") {
		t.Fatalf("meta_json missing redaction marker: %s", payload.MetaJSON)
	}
}

func TestMirrorDeliveryToSession_WritesAssistantMirrorPayload(t *testing.T) {
	rec := store.NewRecording()
	now := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)
	candidates := []session.Metadata{
		{SessionID: "sess-target", Source: "slack", ChatID: "C123", UpdatedAt: 42},
	}

	result, err := MirrorDeliveryToSession(context.Background(), rec, candidates, DeliveryMirrorTarget{
		Platform:    "slack",
		ChatID:      "C123",
		MessageText: "hello from send_message",
		SourceLabel: "cli",
	}, now)
	if err != nil {
		t.Fatalf("MirrorDeliveryToSession error = %v", err)
	}
	if !result.Mirrored || result.SessionID != "sess-target" {
		t.Fatalf("result = %+v, want mirrored sess-target", result)
	}

	cmds := rec.Commands()
	if len(cmds) != 1 || cmds[0].Kind != store.FinalizeAssistantTurn {
		t.Fatalf("commands = %+v, want one FinalizeAssistantTurn", cmds)
	}
	var payload struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		TsUnix    int64  `json:"ts_unix"`
		MetaJSON  string `json:"meta_json"`
	}
	if err := json.Unmarshal(cmds[0].Payload, &payload); err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	if payload.SessionID != "sess-target" || payload.Content != "hello from send_message" || payload.TsUnix != now.Unix() {
		t.Fatalf("payload = %+v, want target/content/timestamp", payload)
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(payload.MetaJSON), &meta); err != nil {
		t.Fatalf("meta_json decode: %v", err)
	}
	if meta["mirror"] != true || meta["mirror_source"] != "cli" {
		t.Fatalf("meta_json = %#v, want mirror=true and mirror_source=cli", meta)
	}
}
