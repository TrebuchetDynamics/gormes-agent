package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gonchotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	e2ePeer       = "telegram:6586915095"
	e2eChatKey    = "telegram:6586915095"
	e2eSessionKey = "sess-e2e"
	e2eUserInput  = "Use Goncho memory, inspect my context, and answer with evidence."
	e2eFinalText  = "Goncho confirms the user prefers exact evidence-first reports."
)

func TestPythonFreeNormalAgentTurn(t *testing.T) {
	t.Run("provider_tool_goncho_memory_final_response_and_audit", func(t *testing.T) {
		h := newNormalTurnHarness(t, true)

		final := h.runTurn(t, e2eUserInput)

		if final.Phase != kernel.PhaseIdle {
			t.Fatalf("final phase = %v, want Idle", final.Phase)
		}
		if final.History[len(final.History)-1].Content != e2eFinalText {
			t.Fatalf("final assistant content = %q, want %q", final.History[len(final.History)-1].Content, e2eFinalText)
		}
		if h.evidence().PythonBridgeUsed {
			t.Fatal("python_bridge_used = true, want false for a Go-native turn")
		}
		if h.evidence().HostedHonchoUsed {
			t.Fatal("hosted_honcho_used = true, want false for local Goncho")
		}
		if !h.evidence().GonchoRecall {
			t.Fatal("goncho recall evidence was not recorded")
		}
		if h.evidence().GonchoMissing != "" {
			t.Fatalf("unexpected missing Goncho evidence: %q", h.evidence().GonchoMissing)
		}

		requests := h.provider.Requests()
		if len(requests) != 2 {
			t.Fatalf("provider request count = %d, want 2 (initial tool call + continuation)", len(requests))
		}
		assertInitialRequest(t, requests[0])
		assertContinuationRequest(t, requests[1])

		assertPersistedTurn(t, h.memory.DB(), "user", e2eUserInput)
		assertPersistedTurn(t, h.memory.DB(), "assistant", e2eFinalText)
		assertToolAudit(t, h.auditPath, "honcho_context", "completed")
	})

	t.Run("degraded_missing_goncho_reports_explicit_evidence", func(t *testing.T) {
		h := newNormalTurnHarness(t, false)

		final := h.runTurn(t, "answer without a configured Goncho service")

		if final.Phase != kernel.PhaseIdle {
			t.Fatalf("final phase = %v, want Idle", final.Phase)
		}
		if final.History[len(final.History)-1].Content != e2eFinalText {
			t.Fatalf("final assistant content = %q, want %q", final.History[len(final.History)-1].Content, e2eFinalText)
		}
		if h.evidence().GonchoMissing != "goncho_step_missing" {
			t.Fatalf("goncho missing evidence = %q, want goncho_step_missing", h.evidence().GonchoMissing)
		}
		if h.evidence().GonchoRecall {
			t.Fatal("goncho recall evidence recorded even though Goncho was disabled")
		}
	})
}

type normalTurnHarness struct {
	memory    *memory.SqliteStore
	provider  *hermes.MockClient
	auditPath string
	e         *turnEvidence
	cancel    context.CancelFunc
	done      chan error
	kernel    *kernel.Kernel
}

type turnEvidence struct {
	mu               sync.Mutex
	PythonBridgeUsed bool
	HostedHonchoUsed bool
	GonchoRecall     bool
	GonchoMissing    string
}

func (e *turnEvidence) markGonchoRecall() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.GonchoRecall = true
}

func (e *turnEvidence) markMissingGoncho(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.GonchoMissing = reason
}

func (e *turnEvidence) snapshot() turnEvidence {
	e.mu.Lock()
	defer e.mu.Unlock()
	return turnEvidence{
		PythonBridgeUsed: e.PythonBridgeUsed,
		HostedHonchoUsed: e.HostedHonchoUsed,
		GonchoRecall:     e.GonchoRecall,
		GonchoMissing:    e.GonchoMissing,
	}
}

func newNormalTurnHarness(t *testing.T, enableGoncho bool) *normalTurnHarness {
	t.Helper()

	dir := t.TempDir()
	mem, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mem.Close(ctx); err != nil {
			t.Fatalf("close memory store: %v", err)
		}
	})

	var svc *goncho.Service
	if enableGoncho {
		svc = goncho.NewService(mem.DB(), goncho.Config{
			WorkspaceID:    "default",
			ObserverPeerID: "gormes",
			RecentMessages: 4,
		}, nil)
		seedGonchoState(t, mem.DB(), svc)
	}

	reg := tools.NewRegistry()
	if enableGoncho {
		gonchotools.RegisterHonchoTools(reg, svc)
	} else {
		reg.MustRegister(staticContextTool{})
	}

	mock := hermes.NewMockClient()
	mock.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{{
			ID:        "call_goncho_context",
			Name:      "honcho_context",
			Arguments: json.RawMessage(`{"peer":"telegram:6586915095","query":"evidence-first","session_key":"sess-e2e","max_tokens":400}`),
		}},
		TokensIn:  42,
		TokensOut: 1,
	}}, "provider-session-1")
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: e2eFinalText, TokensOut: 12},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 96, TokensOut: 12},
	}, "provider-session-1")

	evidence := &turnEvidence{}
	recall := &gonchoRecall{
		svc:      svc,
		peer:     e2ePeer,
		evidence: evidence,
	}
	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:             "e2e-model",
		Endpoint:          "mock://provider",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 2,
		MaxToolDuration:   2 * time.Second,
		InitialSessionID:  e2eSessionKey,
		ChatKey:           e2eChatKey,
		Recall:            recall,
		RecallDeadline:    time.Second,
		ToolAudit:         audit.NewJSONLWriter(auditPath),
	}, mock, mem, telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- k.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("kernel run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("kernel did not stop after cleanup cancel")
		}
	})

	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle
	}, time.Second)

	return &normalTurnHarness{
		memory:    mem,
		provider:  mock,
		auditPath: auditPath,
		e:         evidence,
		cancel:    cancel,
		done:      done,
		kernel:    k,
	}
}

func (h *normalTurnHarness) evidence() turnEvidence {
	return h.e.snapshot()
}

func (h *normalTurnHarness) runTurn(t *testing.T, text string) kernel.RenderFrame {
	t.Helper()
	if err := h.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: text}); err != nil {
		t.Fatalf("submit turn: %v", err)
	}
	return waitForFrame(t, h.kernel.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle &&
			len(f.History) >= 2 &&
			f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
}

type gonchoRecall struct {
	svc      *goncho.Service
	peer     string
	evidence *turnEvidence
}

func (r *gonchoRecall) GetContext(ctx context.Context, params kernel.RecallParams) string {
	if r.svc == nil {
		r.evidence.markMissingGoncho("goncho_step_missing")
		return ""
	}
	got, err := r.svc.Context(ctx, goncho.ContextParams{
		Peer:       r.peer,
		Query:      params.UserMessage,
		SessionKey: params.SessionID,
		MaxTokens:  400,
	})
	if err != nil {
		r.evidence.markMissingGoncho("goncho_recall_failed")
		return ""
	}
	r.evidence.markGonchoRecall()
	return "Goncho memory context:\n" + got.Representation
}

type staticContextTool struct{}

func (staticContextTool) Name() string        { return "honcho_context" }
func (staticContextTool) Description() string { return "disabled Goncho context fixture" }
func (staticContextTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"peer":{"type":"string"},"query":{"type":"string"}}}`)
}
func (staticContextTool) Timeout() time.Duration { return time.Second }
func (staticContextTool) Execute(context.Context, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"unavailable":[{"field":"goncho","reason":"goncho_step_missing"}]}`), nil
}

func seedGonchoState(t *testing.T, db *sql.DB, svc *goncho.Service) {
	t.Helper()
	ctx := context.Background()
	if err := svc.SetProfile(ctx, e2ePeer, []string{"Prefers exact evidence-first reports"}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{
		Peer:       e2ePeer,
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: e2eSessionKey,
	}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, memory_sync_status)
		 VALUES (?, 'user', 'Please keep reports exact and evidence-first.', ?, ?, 'ready')`,
		e2eSessionKey, time.Now().Unix(), e2eChatKey,
	); err != nil {
		t.Fatalf("seed recent turn: %v", err)
	}
}

func assertInitialRequest(t *testing.T, req hermes.ChatRequest) {
	t.Helper()
	if !req.Stream {
		t.Fatal("initial request Stream = false, want true")
	}
	if req.Model != "e2e-model" {
		t.Fatalf("initial request Model = %q, want e2e-model", req.Model)
	}
	if req.SessionID != e2eSessionKey {
		t.Fatalf("initial request SessionID = %q, want %q", req.SessionID, e2eSessionKey)
	}
	if len(req.Tools) == 0 {
		t.Fatal("initial request has no tool descriptors")
	}
	if !hasToolDescriptor(req.Tools, "honcho_context") {
		t.Fatalf("initial request tools = %+v, want honcho_context", req.Tools)
	}
	if len(req.Messages) < 2 || req.Messages[0].Role != "system" {
		t.Fatalf("initial request messages = %+v, want system recall before user", req.Messages)
	}
	if !strings.Contains(req.Messages[0].Content, "Goncho memory context") ||
		!strings.Contains(req.Messages[0].Content, "evidence-first") {
		t.Fatalf("initial system recall = %q, want Goncho evidence-first context", req.Messages[0].Content)
	}
	if got := req.Messages[len(req.Messages)-1]; got.Role != "user" || got.Content != e2eUserInput {
		t.Fatalf("initial user message = %+v, want e2e user input", got)
	}
}

func assertContinuationRequest(t *testing.T, req hermes.ChatRequest) {
	t.Helper()
	if len(req.Messages) < 4 {
		t.Fatalf("continuation messages len = %d, want at least 4", len(req.Messages))
	}
	var assistantTool bool
	var toolReply *hermes.Message
	for i := range req.Messages {
		msg := req.Messages[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) == 1 && msg.ToolCalls[0].Name == "honcho_context" {
			assistantTool = true
		}
		if msg.Role == "tool" && msg.ToolCallID == "call_goncho_context" {
			toolReply = &msg
		}
	}
	if !assistantTool {
		t.Fatalf("continuation messages = %+v, want assistant honcho_context tool call", req.Messages)
	}
	if toolReply == nil {
		t.Fatalf("continuation messages = %+v, want honcho_context tool reply", req.Messages)
	}
	if !strings.Contains(toolReply.Content, "evidence-first") {
		t.Fatalf("tool reply content = %q, want Goncho evidence", toolReply.Content)
	}
}

func assertPersistedTurn(t *testing.T, db *sql.DB, role, content string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var count int
		err := db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM turns WHERE role = ? AND content = ? AND chat_id = ?`,
			role, content, e2eChatKey,
		).Scan(&count)
		if err == nil && count > 0 {
			return
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("query persisted %s turn: %v", role, err)
			}
			t.Fatalf("persisted %s turn %q not found", role, content)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertToolAudit(t *testing.T, path, toolName, status string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		records := readAuditRecords(t, path)
		for _, rec := range records {
			if rec.Tool == toolName && rec.Status == status {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("audit record for %s/%s not found in %+v", toolName, status, records)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readAuditRecords(t *testing.T, path string) []audit.Record {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read audit records: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	records := make([]audit.Record, 0, len(lines))
	for _, line := range lines {
		var rec audit.Record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("decode audit record %q: %v", line, err)
		}
		records = append(records, rec)
	}
	return records
}

func hasToolDescriptor(descs []hermes.ToolDescriptor, name string) bool {
	for _, desc := range descs {
		if desc.Name == name {
			return true
		}
	}
	return false
}

func waitForFrame(t *testing.T, ch <-chan kernel.RenderFrame, pred func(kernel.RenderFrame) bool, timeout time.Duration) kernel.RenderFrame {
	t.Helper()
	deadline := time.After(timeout)
	var last kernel.RenderFrame
	for {
		select {
		case f, ok := <-ch:
			if !ok {
				t.Fatalf("render channel closed before expected frame; last=%+v", last)
			}
			last = f
			if pred(f) {
				return f
			}
		case <-deadline:
			t.Fatalf("timeout waiting for expected frame; last=%+v", last)
		}
	}
}
