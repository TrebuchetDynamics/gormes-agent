package gateway

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// titleModelCallCount is a thread-safe counter used by tests to assert the
// number of times a TitleModel function was invoked.
type titleModelCallCount struct {
	mu    sync.Mutex
	count int
}

func (c *titleModelCallCount) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func (c *titleModelCallCount) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// fakeAuxSink records AutoTitleEvidence passed through the auxiliary failure
// sink for test assertions.
type fakeAuxSink struct {
	mu       sync.Mutex
	received []session.AutoTitleEvidence
}

func (s *fakeAuxSink) record(_ context.Context, ev session.AutoTitleEvidence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.received = append(s.received, ev)
}

func (s *fakeAuxSink) snapshot() []session.AutoTitleEvidence {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]session.AutoTitleEvidence, len(s.received))
	copy(out, s.received)
	return out
}

// buildAutoTitleManager constructs a gateway Manager wired with a hermetic
// session.MemMap, a pinned turn, and the caller-supplied TitleModel and
// AuxiliaryFailureSink. It registers a plain fakeChannel so dispatchFrame can
// deliver the PhaseIdle frame without panicking.
func buildAutoTitleManager(
	t *testing.T,
	ctx context.Context,
	sessionID string,
	titleModel llm.TitleModelFunc,
	sink AutoTitleAuxiliarySink,
) (*Manager, *session.MemMap, *fakeChannel) {
	t.Helper()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", sessionID); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	titleStore := session.NewMetadataTitleStore(ctx, smap)

	fk := &fakeKernel{}
	cfg := ManagerConfig{
		AllowedChats:         map[string]string{"telegram": "42"},
		SessionMap:           smap,
		TitleModel:           titleModel,
		TitleStore:           titleStore,
		AuxiliaryFailureSink: sink,
	}
	m := NewManagerWithSubmitter(cfg, fk, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	return m, smap, ch
}

// dispatchIdleFrame synthesizes a PhaseIdle RenderFrame whose History field
// carries one user+assistant exchange and directly calls dispatchFrame on the
// manager with the turn already pinned. It simulates the path that
// runOutbound would follow after a real provider completes.
func dispatchIdleFrame(t *testing.T, m *Manager, sessionID, userText, assistantText string) {
	t.Helper()
	m.pinTurn("telegram", "42", "msg-1")
	m.turnMu.Lock()
	m.turnLastUserText = userText
	m.turnSessionID = sessionID
	m.turnMu.Unlock()

	var co *coalescer
	var coCancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		SessionID: sessionID,
		History: []llm.Message{
			{Role: "user", Content: userText},
			{Role: "assistant", Content: assistantText},
		},
	}, &co, &coCancel)
}

// TestAutoTitleWiring_FirstUserAssistantPairTriggersGeneration verifies that
// an untitled session receives an auto-generated title after the first
// PhaseIdle frame is dispatched with a hermetic TitleModelFunc.
func TestAutoTitleWiring_FirstUserAssistantPairTriggersGeneration(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-001"

	calls := &titleModelCallCount{}
	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		calls.inc()
		return "Friendly Test Title", nil
	})

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, nil)

	// Seed empty metadata so the session exists but has no title.
	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: sessionID}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	// Verify TitleModelFunc was called once.
	if got := calls.get(); got != 1 {
		t.Errorf("TitleModelFunc calls = %d, want 1", got)
	}

	// Verify title was persisted by reading metadata directly from the MemMap.
	meta, ok, err := smap.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("GetMetadata: session not found")
	}
	if meta.Title != "Friendly Test Title" {
		t.Errorf("Title = %q, want %q", meta.Title, "Friendly Test Title")
	}
	if meta.TitleManuallySet {
		t.Error("TitleManuallySet = true; auto-generated titles must be non-manual")
	}
}

// TestAutoTitleWiring_ManualTitledSessionShortCircuits verifies that a session
// with TitleManuallySet=true never invokes the TitleModelFunc.
func TestAutoTitleWiring_ManualTitledSessionShortCircuits(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-manual"

	calls := &titleModelCallCount{}
	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		calls.inc()
		return "Should Not Be Called", nil
	})

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, nil)

	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID:        sessionID,
		Title:            "Manual Title By Operator",
		TitleManuallySet: true,
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	if got := calls.get(); got != 0 {
		t.Errorf("TitleModelFunc calls = %d, want 0 for manual-titled session", got)
	}

	// Title must remain unchanged.
	meta, _, err := smap.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if meta.Title != "Manual Title By Operator" {
		t.Errorf("Title = %q, want unchanged %q", meta.Title, "Manual Title By Operator")
	}
	if !meta.TitleManuallySet {
		t.Error("TitleManuallySet = false after short-circuit; want true")
	}
}

// TestAutoTitleWiring_AlreadyTitledNonManualSessionShortCircuits verifies that
// a session with an existing non-manual title produces AutoTitleCodeSkippedTitled
// evidence and does not call the TitleModelFunc.
func TestAutoTitleWiring_AlreadyTitledNonManualSessionShortCircuits(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-skipped"

	calls := &titleModelCallCount{}
	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		calls.inc()
		return "Should Not Be Called", nil
	})
	sink := &fakeAuxSink{}

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, sink.record)

	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID:        sessionID,
		Title:            "Old Auto Title",
		TitleManuallySet: false,
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	if got := calls.get(); got != 0 {
		t.Errorf("TitleModelFunc calls = %d, want 0 for already-titled session", got)
	}

	// Verify AutoTitleCodeSkippedTitled evidence was routed through the sink.
	evidences := sink.snapshot()
	if len(evidences) != 1 {
		t.Fatalf("sink received %d evidences, want 1", len(evidences))
	}
	if got := evidences[0].Code; got != session.AutoTitleCodeSkippedTitled {
		t.Errorf("evidence code = %q, want %q", got, session.AutoTitleCodeSkippedTitled)
	}
}

// TestAutoTitleWiring_ProviderFailureRoutesAuxiliaryEvidence verifies that a
// TitleModelFunc returning an error routes title_provider_failed evidence
// through the auxiliary-failure sink.
func TestAutoTitleWiring_ProviderFailureRoutesAuxiliaryEvidence(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-provfail"

	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		return "", errors.New("boom")
	})
	sink := &fakeAuxSink{}

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, sink.record)

	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: sessionID}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	evidences := sink.snapshot()
	if len(evidences) != 1 {
		t.Fatalf("sink received %d evidences, want 1", len(evidences))
	}
	if got := evidences[0].Code; got != session.AutoTitleCodeProviderFailed {
		t.Errorf("evidence code = %q, want %q", got, session.AutoTitleCodeProviderFailed)
	}
}

// TestAutoTitleWiring_BlankResultRoutesEvidence verifies that a TitleModelFunc
// returning "" routes auto_title_blank_result evidence through the sink and
// does not persist a title.
func TestAutoTitleWiring_BlankResultRoutesEvidence(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-blank"

	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		return "", nil
	})
	sink := &fakeAuxSink{}

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, sink.record)

	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: sessionID}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	evidences := sink.snapshot()
	if len(evidences) != 1 {
		t.Fatalf("sink received %d evidences, want 1", len(evidences))
	}
	if got := evidences[0].Code; got != session.AutoTitleCodeBlankResult {
		t.Errorf("evidence code = %q, want %q", got, session.AutoTitleCodeBlankResult)
	}

	// No title should have been written.
	meta, _, err := smap.GetMetadata(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if meta.Title != "" {
		t.Errorf("Title = %q after blank result; want empty", meta.Title)
	}
}

// TestAutoTitleWiring_OneCallPerTurnNoDoubleFire dispatches two PhaseIdle
// frames for the same session. The first turn writes a title; the second turn
// hits AutoTitleCodeSkippedTitled so the generator is called exactly once.
func TestAutoTitleWiring_OneCallPerTurnNoDoubleFire(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-once"

	calls := &titleModelCallCount{}
	titleModel := llm.TitleModelFunc(func(_ context.Context, _ llm.TitleModelRequest) (string, error) {
		calls.inc()
		return "First Turn Title", nil
	})

	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, titleModel, nil)

	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: sessionID}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	// First dispatch — should generate a title.
	dispatchIdleFrame(t, m, sessionID, "first message", "first reply")
	if got := calls.get(); got != 1 {
		t.Errorf("after first dispatch: TitleModelFunc calls = %d, want 1", got)
	}

	// Second dispatch — session already has a title; should short-circuit.
	dispatchIdleFrame(t, m, sessionID, "second message", "second reply")
	if got := calls.get(); got != 1 {
		t.Errorf("after second dispatch: TitleModelFunc calls = %d, want still 1 (no double fire)", got)
	}
}

// TestAutoTitleWiring_NilTitleModelFuncRecordsEvidence verifies that a nil
// ManagerConfig.TitleModel does not panic and routes AutoTitleCodeProviderFailed
// evidence through the sink (because PerformAutoTitle treats nil gen as
// AutoTitleCodeProviderFailed).
func TestAutoTitleWiring_NilTitleModelFuncRecordsEvidence(t *testing.T) {
	ctx := context.Background()
	const sessionID = "sess-autotitle-nilmodel"
	sink := &fakeAuxSink{}

	// TitleModel is intentionally nil.
	m, smap, _ := buildAutoTitleManager(t, ctx, sessionID, nil, sink.record)

	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: sessionID}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}

	// Must not panic.
	dispatchIdleFrame(t, m, sessionID, "hello", "hi there")

	evidences := sink.snapshot()
	if len(evidences) != 1 {
		t.Fatalf("sink received %d evidences, want 1", len(evidences))
	}
	// nil gen → AutoTitleCodeProviderFailed per auto_title.go:113
	if got := evidences[0].Code; got != session.AutoTitleCodeProviderFailed {
		t.Errorf("evidence code = %q, want %q", got, session.AutoTitleCodeProviderFailed)
	}
}

func TestAutoTitleWiring_AuxiliarySinkPanicLogged(t *testing.T) {
	ctx := context.Background()
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(oldLogger)
	})

	store := session.NewMetadataTitleStore(ctx, session.NewMemMap())
	runAutoTitle(ctx, store, nil, "sess-autotitle-sink-panic", "hello", "hi there", func(context.Context, session.AutoTitleEvidence) {
		panic("sink boom")
	})

	got := logs.String()
	if !strings.Contains(got, "auto_title_sink_panic") {
		t.Fatalf("auto-title sink panic log = %q, want typed panic evidence", got)
	}
}
