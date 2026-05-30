package autotitle

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

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
