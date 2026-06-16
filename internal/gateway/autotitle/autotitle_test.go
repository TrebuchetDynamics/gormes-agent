package autotitle

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestTitleModelToGeneratorAllowsNilContext(t *testing.T) {
	got, err := TitleModelToGenerator(nil, func(ctx context.Context, req llm.TitleModelRequest) (string, error) {
		if ctx == nil {
			panic("nil context")
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "hello" {
			t.Fatalf("request messages = %+v, want single user hello", req.Messages)
		}
		return "Title", nil
	}, []session.TitleTurn{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("TitleModelToGenerator error = %v, want nil", err)
	}
	if got != "Title" {
		t.Fatalf("TitleModelToGenerator = %q, want Title", got)
	}
}

func TestAuxiliarySinkPanicLogged(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(oldLogger)
	})

	store := session.NewMetadataTitleStore(ctx, session.NewMemMap())
	Run(ctx, store, nil, "sess-autotitle-sink-panic", "hello", "hi there", func(context.Context, session.AutoTitleEvidence) {
		panic("sink boom")
	})

	got := logs.String()
	if !strings.Contains(got, "auto_title_sink_panic") {
		t.Fatalf("auto-title sink panic log = %q, want typed panic evidence", got)
	}
}
