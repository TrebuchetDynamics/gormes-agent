package gateway

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/session"
)

func TestManager_PersistSessionAutoTitlesFirstExchange(t *testing.T) {
	smap := session.NewMemMap()
	m := NewManagerWithSubmitter(ManagerConfig{
		SessionMap: smap,
	}, &fakeKernel{}, slog.Default())
	m.pinTurn("telegram", "42", "m1")

	m.persistSession(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: "sess-title",
		History: []hermes.Message{
			{Role: "user", Content: "Can you help me debug the gateway restart loop after deploy?"},
			{Role: "assistant", Content: "Let's inspect the logs and the service unit first."},
		},
	})

	gotSessionID, err := smap.Get(context.Background(), "telegram:42")
	if err != nil {
		t.Fatalf("Get(session map): %v", err)
	}
	if gotSessionID != "sess-title" {
		t.Fatalf("Get(session map) = %q, want sess-title", gotSessionID)
	}

	meta, ok, err := smap.GetMetadata(context.Background(), "sess-title")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("GetMetadata() ok = false, want true")
	}

	titleField := reflect.ValueOf(meta).FieldByName("Title")
	if !titleField.IsValid() {
		t.Fatalf("Metadata missing Title field: %+v", meta)
	}
	if got := titleField.String(); got != "Debug Gateway Restart Loop" {
		t.Fatalf("Title = %q, want %q", got, "Debug Gateway Restart Loop")
	}
}
