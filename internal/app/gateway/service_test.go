package gateway

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/channelmemory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

type fakeGatewayExtractorLLM struct{}

func (fakeGatewayExtractorLLM) Health(context.Context) error { return nil }
func (fakeGatewayExtractorLLM) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}
func (fakeGatewayExtractorLLM) OpenStream(context.Context, llm.ChatRequest) (llm.Stream, error) {
	return &fakeGatewayExtractorStream{}, nil
}

type fakeGatewayExtractorStream struct {
	emitted bool
}

func (s *fakeGatewayExtractorStream) SessionID() string { return "" }
func (s *fakeGatewayExtractorStream) Close() error      { return nil }
func (s *fakeGatewayExtractorStream) Recv(context.Context) (llm.Event, error) {
	if s.emitted {
		return llm.Event{Kind: llm.EventDone}, io.EOF
	}
	s.emitted = true
	return llm.Event{Kind: llm.EventToken, Token: `{"entities":[],"relationships":[]}`}, nil
}

func TestStartGatewayExtractorProcessesReadyTurns(t *testing.T) {
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())
	if _, err := store.DB().Exec(`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, memory_sync_status) VALUES ('sess', 'assistant', 'hello', 1, 'telegram:42', 'ready')`); err != nil {
		t.Fatalf("seed turn: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ext := startGatewayExtractor(ctx, store, fakeGatewayExtractorLLM{}, config.Config{}, channelmemory.Settings{
		ExtractorBatchSize:    1,
		ExtractorPollInterval: time.Millisecond,
	}, nil)
	defer func() {
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
		defer closeCancel()
		if err := ext.Close(closeCtx); err != nil {
			t.Fatalf("extractor close: %v", err)
		}
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var extracted int
		if err := store.DB().QueryRow(`SELECT extracted FROM turns WHERE session_id = 'sess'`).Scan(&extracted); err != nil {
			t.Fatalf("query extracted: %v", err)
		}
		if extracted == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	var extracted, attempts int
	var syncStatus string
	_ = store.DB().QueryRow(`SELECT extracted, extraction_attempts, memory_sync_status FROM turns WHERE session_id = 'sess'`).Scan(&extracted, &attempts, &syncStatus)
	t.Fatalf("gateway extractor did not process ready turn: extracted=%d attempts=%d sync=%s", extracted, attempts, syncStatus)
}
